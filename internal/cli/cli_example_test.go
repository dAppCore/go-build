package cli

import core "dappco.re/go"

func ExampleStyle_Render() {
	style := Style{}
	_ = style.Render("value")
}

func ExampleSetStdout() {
	buffer := core.NewBuffer()
	SetStdout(buffer)
	Print("%s", "value")
}

func ExampleSetStderr() {
	buffer := core.NewBuffer()
	SetStderr(buffer)
	_ = Err("%s", "value")
}

func ExamplePrint() {
	buffer := core.NewBuffer()
	SetStdout(buffer)
	Print("%s", "value")
}

func ExampleText() {
	buffer := core.NewBuffer()
	SetStdout(buffer)
	Text("value")
}

func ExampleBlank() {
	buffer := core.NewBuffer()
	SetStdout(buffer)
	Blank()
}

func ExampleErr() {
	_ = Err("%s", "value")
}

func ExampleWrap() {
	_ = Wrap(core.NewError("cause"), "context")
}

func ExampleWrapVerb() {
	_ = WrapVerb(core.NewError("cause"), "read", "file")
}

func ExampleExitError_Error() {
	err := &ExitError{Code: 2, Err: core.NewError("value")}
	_ = err.Error()
}

func ExampleExit() {
	_ = Exit(2, core.NewError("value"))
}

func ExampleContext() {
	_ = Context()
}
