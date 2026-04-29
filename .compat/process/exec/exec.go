package exec

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

func (c *Cmd) build() (*core.Cmd, error) {
	if c.ctx == nil {
		return nil, core.NewError("command context is required")
	}
	resolved, err := resolveExecutable(c.name)
	if err != nil {
		return nil, err
	}
	cmd := &core.Cmd{Path: resolved, Args: append([]string{resolved}, c.args...)}
	cmd.Dir = c.dir
	if len(c.env) > 0 {
		cmd.Env = append(core.Environ(), c.env...)
	}
	cmd.Stdin = c.stdin
	cmd.Stdout = c.stdout
	cmd.Stderr = c.stderr
	return cmd, nil
}

func (c *Cmd) Run() error {
	cmd, err := c.build()
	if err != nil {
		return err
	}
	return runCommand(c.ctx, cmd)
}

func (c *Cmd) CombinedOutput() ([]byte, error) {
	cmd, err := c.build()
	if err != nil {
		return nil, err
	}
	buf := core.NewBuffer()
	if c.stdout != nil || c.stderr != nil {
		if c.stdout == nil {
			cmd.Stdout = buf
		}
		if c.stderr == nil {
			cmd.Stderr = buf
		}
		err := runCommand(c.ctx, cmd)
		return buf.Bytes(), err
	}
	cmd.Stdout = buf
	cmd.Stderr = buf
	err = runCommand(c.ctx, cmd)
	return buf.Bytes(), err
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

func resultError(result core.Result) error {
	if err, ok := result.Value.(error); ok {
		return err
	}
	return core.NewError(result.Error())
}
