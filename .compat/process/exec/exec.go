package exec

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	osexec "os/exec"
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

func (c *Cmd) build() (*osexec.Cmd, error) {
	if c.ctx == nil {
		return nil, errors.New("command context is required")
	}
	cmd := osexec.CommandContext(c.ctx, c.name, c.args...)
	cmd.Dir = c.dir
	if len(c.env) > 0 {
		cmd.Env = append(os.Environ(), c.env...)
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
	return cmd.Run()
}

func (c *Cmd) CombinedOutput() ([]byte, error) {
	cmd, err := c.build()
	if err != nil {
		return nil, err
	}
	if c.stdout != nil || c.stderr != nil {
		var buf bytes.Buffer
		if c.stdout == nil {
			cmd.Stdout = &buf
		}
		if c.stderr == nil {
			cmd.Stderr = &buf
		}
		err := cmd.Run()
		return buf.Bytes(), err
	}
	return cmd.CombinedOutput()
}
