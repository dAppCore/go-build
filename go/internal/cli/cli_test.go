package cli

import core "dappco.re/go"

func TestCli_Style_Render_Good(t *core.T) {
	style := Style{}
	rendered := style.Render("value")
	core.AssertEqual(t, "value", rendered)
	core.AssertFalse(t, rendered == "")
}

func TestCli_Style_Render_Bad(t *core.T) {
	style := Style{}
	rendered := style.Render("")
	core.AssertEqual(t, "", rendered)
	core.AssertTrue(t, len(rendered) == 0)
}

func TestCli_Style_Render_Ugly(t *core.T) {
	style := Style{}
	rendered := style.Render("  spaced  ")
	core.AssertEqual(t, "  spaced  ", rendered)
	core.AssertTrue(t, core.Contains(rendered, "spaced"))
}

func TestCli_SetStdout_Good(t *core.T) {
	buffer := core.NewBuffer()
	SetStdout(buffer)
	Print("%s", "stdout")
	core.AssertEqual(t, "stdout", buffer.String())
}

func TestCli_SetStdout_Bad(t *core.T) {
	buffer := core.NewBuffer()
	SetStdout(buffer)
	SetStdout(nil)
	core.AssertEqual(t, "", buffer.String())
}

func TestCli_SetStdout_Ugly(t *core.T) {
	first := core.NewBuffer()
	second := core.NewBuffer()
	SetStdout(first)
	SetStdout(second)
	Print("%s", "swapped")
	core.AssertEqual(t, "swapped", second.String())
}

func TestCli_SetStderr_Good(t *core.T) {
	buffer := core.NewBuffer()
	SetStderr(buffer)
	_ = Err("%s", "stderr")
	core.AssertEqual(t, "", buffer.String())
}

func TestCli_SetStderr_Bad(t *core.T) {
	buffer := core.NewBuffer()
	SetStderr(buffer)
	SetStderr(nil)
	core.AssertEqual(t, "", buffer.String())
}

func TestCli_SetStderr_Ugly(t *core.T) {
	first := core.NewBuffer()
	second := core.NewBuffer()
	SetStderr(first)
	SetStderr(second)
	core.AssertFalse(t, first == second)
}

func TestCli_Print_Good(t *core.T) {
	buffer := core.NewBuffer()
	SetStdout(buffer)
	Print("hello %s", "world")
	core.AssertEqual(t, "hello world", buffer.String())
}

func TestCli_Print_Bad(t *core.T) {
	buffer := core.NewBuffer()
	SetStdout(buffer)
	Print("%s", "")
	core.AssertEqual(t, "", buffer.String())
}

func TestCli_Print_Ugly(t *core.T) {
	buffer := core.NewBuffer()
	SetStdout(buffer)
	Print("%d:%s", 7, "value")
	core.AssertEqual(t, "7:value", buffer.String())
}

func TestCli_Text_Good(t *core.T) {
	buffer := core.NewBuffer()
	SetStdout(buffer)
	Text("line")
	core.AssertEqual(t, "line\n", buffer.String())
}

func TestCli_Text_Bad(t *core.T) {
	buffer := core.NewBuffer()
	SetStdout(buffer)
	Text("")
	core.AssertEqual(t, "\n", buffer.String())
}

func TestCli_Text_Ugly(t *core.T) {
	buffer := core.NewBuffer()
	SetStdout(buffer)
	Text("line\nnext")
	core.AssertTrue(t, core.Contains(buffer.String(), "next"))
}

func TestCli_Blank_Good(t *core.T) {
	buffer := core.NewBuffer()
	SetStdout(buffer)
	Blank()
	core.AssertEqual(t, "\n", buffer.String())
}

func TestCli_Blank_Bad(t *core.T) {
	buffer := core.NewBuffer()
	SetStdout(buffer)
	Blank()
	core.AssertTrue(t, len(buffer.String()) == 1)
}

func TestCli_Blank_Ugly(t *core.T) {
	buffer := core.NewBuffer()
	SetStdout(buffer)
	Blank()
	Blank()
	core.AssertEqual(t, "\n\n", buffer.String())
}

func TestCli_Err_Good(t *core.T) {
	result := Err("%s", "message")
	core.AssertFalse(t, result.OK)
	core.AssertTrue(t, core.Contains(result.Error(), "message"))
}

func TestCli_Err_Bad(t *core.T) {
	result := Err("%s", "")
	core.AssertFalse(t, result.OK)
	core.AssertFalse(t, result.Error() == "message")
}

func TestCli_Err_Ugly(t *core.T) {
	result := Err("code %d", 42)
	core.AssertFalse(t, result.OK)
	core.AssertTrue(t, core.Contains(result.Error(), "42"))
}

func TestCli_Wrap_Good(t *core.T) {
	result := Wrap(core.NewError("cause"), "context")
	core.AssertFalse(t, result.OK)
	core.AssertTrue(t, core.Contains(result.Error(), "context"))
}

func TestCli_Wrap_Bad(t *core.T) {
	result := Wrap(nil, "context")
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, nil, result.Value)
}

func TestCli_Wrap_Ugly(t *core.T) {
	result := Wrap("cause", "")
	core.AssertFalse(t, result.OK)
	core.AssertTrue(t, core.Contains(result.Error(), "cause"))
}

func TestCli_WrapVerb_Good(t *core.T) {
	result := WrapVerb(core.NewError("cause"), "read", "file")
	core.AssertFalse(t, result.OK)
	core.AssertTrue(t, core.Contains(result.Error(), "failed to read file"))
}

func TestCli_WrapVerb_Bad(t *core.T) {
	result := WrapVerb(nil, "read", "file")
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, nil, result.Value)
}

func TestCli_WrapVerb_Ugly(t *core.T) {
	result := WrapVerb("cause", "parse", "")
	core.AssertFalse(t, result.OK)
	core.AssertTrue(t, core.Contains(result.Error(), "parse"))
}

func TestCli_ExitError_Error_Good(t *core.T) {
	exitError := &ExitError{Code: 2, Err: core.NewError("boom")}
	message := exitError.Error()
	core.AssertEqual(t, "boom", message)
}

func TestCli_ExitError_Error_Bad(t *core.T) {
	var exitError *ExitError
	message := exitError.Error()
	core.AssertEqual(t, "", message)
}

func TestCli_ExitError_Error_Ugly(t *core.T) {
	exitError := &ExitError{Code: 7}
	message := exitError.Error()
	core.AssertTrue(t, core.Contains(message, "7"))
}

func TestCli_Exit_Good(t *core.T) {
	result := Exit(2, core.NewError("boom"))
	exitError := result.Value.(*ExitError)
	core.AssertEqual(t, 2, exitError.Code)
}

func TestCli_Exit_Bad(t *core.T) {
	result := Exit(1, nil)
	exitError := result.Value.(*ExitError)
	core.AssertTrue(t, core.Contains(exitError.Error(), "exit"))
}

func TestCli_Exit_Ugly(t *core.T) {
	result := Exit(3, "text")
	exitError := result.Value.(*ExitError)
	core.AssertEqual(t, 3, exitError.Code)
}

func TestCli_Context_Good(t *core.T) {
	ctx := Context()
	core.AssertEqual(t, nil, ctx.Err())
	core.AssertFalse(t, ctx == nil)
}

func TestCli_Context_Bad(t *core.T) {
	ctx := Context()
	done := ctx.Done()
	core.AssertTrue(t, done == nil)
}

func TestCli_Context_Ugly(t *core.T) {
	first := Context()
	second := Context()
	core.AssertEqual(t, first, second)
}
