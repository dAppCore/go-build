package process

import (
	"context"
	"syscall"
	"time"
	"unicode"
	"unicode/utf8"

	core "dappco.re/go"
)

type RunOptions struct {
	Command        string
	Args           []string
	Dir            string
	Env            []string
	DisableCapture bool
	Detach         bool
	Timeout        time.Duration
	GracePeriod    time.Duration
	KillGroup      bool
}

type Program struct {
	Name string
	Path string
}

func (p *Program) Find() core.Result {
	target := p.Path
	if target == "" {
		target = p.Name
	}
	if target == "" {
		return core.Fail(core.NewError("program name is empty"))
	}
	found := core.App{}.Find(target, target)
	if !found.OK {
		return found
	}
	p.Path = found.Value.(*core.App).Path
	return core.Ok(nil)
}

func (p *Program) Run(ctx context.Context, args ...string) core.Result {
	return p.RunDir(ctx, "", args...)
}

func (p *Program) RunDir(ctx context.Context, dir string, args ...string) core.Result {
	if ctx == nil {
		return core.Fail(core.NewError("command context is required"))
	}
	binary := p.Path
	if binary == "" {
		binary = p.Name
	}
	if binary == "" {
		return core.Fail(core.NewError("program name is empty"))
	}
	resolved := resolveExecutable(binary)
	if !resolved.OK {
		return resolved
	}
	path := resolved.Value.(string)
	out := core.NewBuffer()
	cmd := &core.Cmd{Path: path, Args: append([]string{path}, args...), Dir: dir, Stdout: out, Stderr: out}
	run := runCommand(ctx, cmd)
	if !run.OK {
		return core.Fail(core.E("process.RunDir", trimRightSpace(out.String()), core.NewError(run.Error())))
	}
	return core.Ok(trimRightSpace(out.String()))
}

func RunWithOptions(ctx context.Context, opts RunOptions) core.Result {
	if ctx == nil {
		return core.Fail(core.NewError("command context is required"))
	}
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}
	resolved := resolveExecutable(opts.Command)
	if !resolved.OK {
		return resolved
	}
	path := resolved.Value.(string)
	cmd := &core.Cmd{Path: path, Args: append([]string{path}, opts.Args...)}
	cmd.Dir = opts.Dir
	if len(opts.Env) > 0 {
		cmd.Env = append(core.Environ(), opts.Env...)
	}
	if opts.Detach {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}
	out := core.NewBuffer()
	if !opts.DisableCapture {
		cmd.Stdout = out
		cmd.Stderr = out
	}
	run := runCommand(ctx, cmd)
	if !run.OK {
		return core.Fail(core.E("process.RunWithOptions", trimRightSpace(out.String()), core.NewError(run.Error())))
	}
	return core.Ok(trimRightSpace(out.String()))
}

type DaemonOptions struct {
	PIDFile         string
	HealthAddr      string
	ShutdownTimeout time.Duration
}

type Daemon struct {
	options DaemonOptions
	ready   bool
}

func NewDaemon(opts DaemonOptions) *Daemon {
	return &Daemon{options: opts}
}

func (d *Daemon) Start() core.Result {
	if d == nil || d.options.PIDFile == "" {
		return core.Ok(nil)
	}
	written := core.WriteFile(d.options.PIDFile, []byte("0\n"), 0o644)
	if !written.OK {
		return written
	}
	return core.Ok(nil)
}

func (d *Daemon) Stop() core.Result {
	if d == nil || d.options.PIDFile == "" {
		return core.Ok(nil)
	}
	removed := core.Remove(d.options.PIDFile)
	if !removed.OK {
		if err, ok := removed.Value.(error); ok && core.IsNotExist(err) {
			return core.Ok(nil)
		}
		return removed
	}
	return core.Ok(nil)
}

func (d *Daemon) SetReady(ready bool) {
	if d != nil {
		d.ready = ready
	}
}

func resolveExecutable(name string) core.Result {
	if name == "" {
		return core.Fail(core.NewError("program name is empty"))
	}
	if core.Contains(name, "/") || core.Contains(name, "\\") {
		return core.Ok(name)
	}
	found := core.App{}.Find(name, name)
	if !found.OK {
		return found
	}
	return core.Ok(found.Value.(*core.App).Path)
}

func runCommand(ctx context.Context, cmd *core.Cmd) core.Result {
	if err := cmd.Start(); err != nil {
		return core.Fail(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case err := <-done:
		return core.ResultOf(nil, err)
	case <-ctx.Done():
		if cmd.Process != nil {
			killErr := cmd.Process.Kill()
			<-done
			if killErr != nil {
				return core.Fail(killErr)
			}
			return core.Fail(ctx.Err())
		}
		return core.Fail(ctx.Err())
	}
}

func trimRightSpace(value string) string {
	for len(value) > 0 {
		r, size := utf8.DecodeLastRuneInString(value)
		if !unicode.IsSpace(r) {
			return value
		}
		value = value[:len(value)-size]
	}
	return value
}
