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

func (p *Program) Find() error {
	target := p.Path
	if target == "" {
		target = p.Name
	}
	if target == "" {
		return core.NewError("program name is empty")
	}
	found := core.App{}.Find(target, target)
	if !found.OK {
		return resultError(found)
	}
	p.Path = found.Value.(*core.App).Path
	return nil
}

func (p *Program) Run(ctx context.Context, args ...string) (string, error) {
	return p.RunDir(ctx, "", args...)
}

func (p *Program) RunDir(ctx context.Context, dir string, args ...string) (string, error) {
	if ctx == nil {
		return "", core.NewError("command context is required")
	}
	binary := p.Path
	if binary == "" {
		binary = p.Name
	}
	if binary == "" {
		return "", core.NewError("program name is empty")
	}
	resolved, err := resolveExecutable(binary)
	if err != nil {
		return "", err
	}
	out := core.NewBuffer()
	cmd := &core.Cmd{Path: resolved, Args: append([]string{resolved}, args...), Dir: dir, Stdout: out, Stderr: out}
	err = runCommand(ctx, cmd)
	return trimRightSpace(out.String()), err
}

func RunWithOptions(ctx context.Context, opts RunOptions) (string, error) {
	if ctx == nil {
		return "", core.NewError("command context is required")
	}
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}
	resolved, err := resolveExecutable(opts.Command)
	if err != nil {
		return "", err
	}
	cmd := &core.Cmd{Path: resolved, Args: append([]string{resolved}, opts.Args...)}
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
	err = runCommand(ctx, cmd)
	return trimRightSpace(out.String()), err
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

func (d *Daemon) Start() error {
	if d == nil || d.options.PIDFile == "" {
		return nil
	}
	written := core.WriteFile(d.options.PIDFile, []byte("0\n"), 0o644)
	if !written.OK {
		return resultError(written)
	}
	return nil
}

func (d *Daemon) Stop() error {
	if d == nil || d.options.PIDFile == "" {
		return nil
	}
	removed := core.Remove(d.options.PIDFile)
	if !removed.OK {
		err := resultError(removed)
		if !core.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (d *Daemon) SetReady(ready bool) {
	if d != nil {
		d.ready = ready
	}
}

func resolveExecutable(name string) (string, error) {
	if name == "" {
		return "", core.NewError("program name is empty")
	}
	if core.Contains(name, "/") || core.Contains(name, "\\") {
		return name, nil
	}
	found := core.App{}.Find(name, name)
	if !found.OK {
		return "", resultError(found)
	}
	return found.Value.(*core.App).Path, nil
}

func runCommand(ctx context.Context, cmd *core.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		if cmd.Process != nil {
			killErr := cmd.Process.Kill()
			<-done
			if killErr != nil {
				return killErr
			}
			return ctx.Err()
		}
		return ctx.Err()
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

func resultError(result core.Result) error {
	if err, ok := result.Value.(error); ok {
		return err
	}
	return core.NewError(result.Error())
}
