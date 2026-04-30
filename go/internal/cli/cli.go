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
		return
	}
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

func Err(format string, args ...any) core.Result {
	return core.Fail(core.E("cli.Err", core.Sprintf(format, args...), nil))
}

func Wrap(cause any, message string) core.Result {
	if cause == nil {
		return core.Ok(nil)
	}
	err, ok := cause.(error)
	if !ok {
		err = core.NewError(core.Sprintf("%v", cause))
	}
	if message == "" {
		return core.Fail(err)
	}
	return core.Fail(core.E("cli.Wrap", message, err))
}

func WrapVerb(cause any, verb, subject string) core.Result {
	if cause == nil {
		return core.Ok(nil)
	}
	err, ok := cause.(error)
	if !ok {
		err = core.NewError(core.Sprintf("%v", cause))
	}
	return core.Fail(core.E("cli.WrapVerb", core.Sprintf("failed to %s %s", verb, subject), err))
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

func Exit(code int, err any) core.Result {
	if err == nil {
		err = core.NewError("exit")
	}
	cause, ok := err.(error)
	if !ok {
		cause = core.NewError(core.Sprintf("%v", err))
	}
	return core.Fail(&ExitError{Code: code, Err: cause})
}

func Context() context.Context {
	return context.Background()
}
