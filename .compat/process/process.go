package process

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unicode"
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
		return errors.New("program name is empty")
	}
	path, err := exec.LookPath(target)
	if err != nil {
		return err
	}
	p.Path = path
	return nil
}

func (p *Program) Run(ctx context.Context, args ...string) (string, error) {
	return p.RunDir(ctx, "", args...)
}

func (p *Program) RunDir(ctx context.Context, dir string, args ...string) (string, error) {
	if ctx == nil {
		return "", errors.New("command context is required")
	}
	binary := p.Path
	if binary == "" {
		binary = p.Name
	}
	if binary == "" {
		return "", errors.New("program name is empty")
	}
	cmd := exec.CommandContext(ctx, binary, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return strings.TrimRightFunc(out.String(), unicode.IsSpace), err
}

func RunWithOptions(ctx context.Context, opts RunOptions) (string, error) {
	if ctx == nil {
		return "", errors.New("command context is required")
	}
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, opts.Command, opts.Args...)
	cmd.Dir = opts.Dir
	if len(opts.Env) > 0 {
		cmd.Env = append(os.Environ(), opts.Env...)
	}
	if opts.Detach {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}
	out, err := cmd.CombinedOutput()
	return strings.TrimRightFunc(string(out), unicode.IsSpace), err
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
	return os.WriteFile(d.options.PIDFile, []byte("0\n"), 0o644)
}

func (d *Daemon) Stop() error {
	if d == nil || d.options.PIDFile == "" {
		return nil
	}
	if err := os.Remove(d.options.PIDFile); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (d *Daemon) SetReady(ready bool) {
	if d != nil {
		d.ready = ready
	}
}
