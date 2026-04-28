package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
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
	stdout io.Writer = os.Stdout
	stderr io.Writer = os.Stderr
)

func SetStdout(w io.Writer) {
	if w == nil {
		stdout = os.Stdout
		return
	}
	stdout = w
}

func SetStderr(w io.Writer) {
	if w == nil {
		stderr = os.Stderr
		return
	}
	stderr = w
}

func Print(format string, args ...any) {
	_, _ = fmt.Fprintf(stdout, format, args...)
}

func Text(text string) {
	_, _ = fmt.Fprintln(stdout, text)
}

func Blank() {
	_, _ = fmt.Fprintln(stdout)
}

func Err(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}

func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	if message == "" {
		return err
	}
	return fmt.Errorf("%s: %w", message, err)
}

func WrapVerb(err error, verb, subject string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("failed to %s %s: %w", verb, subject, err)
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
	return fmt.Sprintf("exit %d", e.Code)
}

func (e *ExitError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func Exit(code int, err error) error {
	if err == nil {
		err = errors.New("exit")
	}
	return &ExitError{Code: code, Err: err}
}

func Context() context.Context {
	return context.Background()
}
