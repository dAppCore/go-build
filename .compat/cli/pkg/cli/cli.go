package cli

import (
	"context"
	"io"

	core "dappco.re/go"
)

type Style struct{}

func (Style) Render(text string) string { return text }

var (
	TitleStyle   Style
	ValueStyle   Style
	SuccessStyle Style
	ErrorStyle   Style
	DimStyle     Style
	RepoStyle    Style
)

var (
	stdout io.Writer = core.Stdout()
	stderr io.Writer = core.Stderr()
)

func SetStdout(w io.Writer) {
	if w == nil {
		stdout = core.Stdout()
		return
	}
	stdout = w
}

func SetStderr(w io.Writer) {
	if w == nil {
		stderr = core.Stderr()
		return
	}
	stderr = w
}

func Print(format string, args ...any) {
	if written := core.WriteString(stdout, core.Sprintf(format, args...)); !written.OK {
		return
	}
}

func Text(text string) {
	if written := core.WriteString(stdout, text+"\n"); !written.OK {
		return
	}
}

func Blank() {
	if written := core.WriteString(stdout, "\n"); !written.OK {
		return
	}
}

func Err(format string, args ...any) error {
	return core.Errorf(format, args...)
}

func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	if message == "" {
		return err
	}
	return core.Errorf("%s: %w", message, err)
}

func WrapVerb(err error, verb, subject string) error {
	if err == nil {
		return nil
	}
	return core.Errorf("failed to %s %s: %w", verb, subject, err)
}

type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return core.Sprintf("exit %d", e.Code)
}

func (e *ExitError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func Exit(code int, err error) error {
	if err == nil {
		err = core.NewError("exit")
	}
	return &ExitError{Code: code, Err: err}
}

func Context() context.Context {
	return context.Background()
}
