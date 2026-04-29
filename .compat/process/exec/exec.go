package processexec

import (
	"context"
	"io"

	core "dappco.re/go"
)

type Cmd struct {
	ctx    context.Context
	name   string
	args   []string
	dir    string
	env    []string
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

func Command(ctx context.Context, name string, args ...string) *Cmd {
	return &Cmd{ctx: ctx, name: name, args: args}
}

func (c *Cmd) WithDir(dir string) *Cmd {
	c.dir = dir
	return c
}

func (c *Cmd) WithEnv(env []string) *Cmd {
	c.env = env
	return c
}

func (c *Cmd) WithStdin(r io.Reader) *Cmd {
	c.stdin = r
	return c
}

func (c *Cmd) WithStdout(w io.Writer) *Cmd {
	c.stdout = w
	return c
}

func (c *Cmd) WithStderr(w io.Writer) *Cmd {
	c.stderr = w
	return c
}

func (c *Cmd) build() core.Result {
	if c.ctx == nil {
		return core.Fail(core.NewError("command context is required"))
	}
	resolved := resolveExecutable(c.name)
	if !resolved.OK {
		return resolved
	}
	path := resolved.Value.(string)
	cmd := &core.Cmd{Path: path, Args: append([]string{path}, c.args...)}
	cmd.Dir = c.dir
	if len(c.env) > 0 {
		cmd.Env = append(core.Environ(), c.env...)
	}
	cmd.Stdin = c.stdin
	cmd.Stdout = c.stdout
	cmd.Stderr = c.stderr
	return core.Ok(cmd)
}

func (c *Cmd) Run() core.Result {
	cmd := c.build()
	if !cmd.OK {
		return cmd
	}
	return runCommand(c.ctx, cmd.Value.(*core.Cmd))
}

func (c *Cmd) CombinedOutput() core.Result {
	built := c.build()
	if !built.OK {
		return built
	}
	cmd := built.Value.(*core.Cmd)
	buf := core.NewBuffer()
	if c.stdout != nil || c.stderr != nil {
		if c.stdout == nil {
			cmd.Stdout = buf
		}
		if c.stderr == nil {
			cmd.Stderr = buf
		}
		run := runCommand(c.ctx, cmd)
		if !run.OK {
			return core.Fail(core.E("process.CombinedOutput", core.Trim(buf.String()), core.NewError(run.Error())))
		}
		return core.Ok(buf.Bytes())
	}
	cmd.Stdout = buf
	cmd.Stderr = buf
	run := runCommand(c.ctx, cmd)
	if !run.OK {
		return core.Fail(core.E("process.CombinedOutput", core.Trim(buf.String()), core.NewError(run.Error())))
	}
	return core.Ok(buf.Bytes())
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
