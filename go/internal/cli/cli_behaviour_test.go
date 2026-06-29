package cli

import (
	"io"

	core "dappco.re/go"
)

// failingWriter always reports a write error so the !written.OK return branches
// of Print, Text and Blank can be reached.
type failingWriter struct{}

func (failingWriter) Write(p []byte) (int, error) {
	return 0, io.ErrClosedPipe
}

// recordingWriter captures everything written so the success path can be
// asserted against real output.
type recordingWriter struct {
	data []byte
}

func (w *recordingWriter) Write(p []byte) (int, error) {
	w.data = append(w.data, p...)
	return len(p), nil
}

func TestCli_Print_WriteError_Bad(t *core.T) {
	SetStdout(failingWriter{})
	defer SetStdout(nil)
	// Reaching the !written.OK branch must not panic.
	core.AssertNotPanics(t, func() { Print("hello %s", "world") })
}

func TestCli_Text_WriteError_Bad(t *core.T) {
	SetStdout(failingWriter{})
	defer SetStdout(nil)
	core.AssertNotPanics(t, func() { Text("line") })
}

func TestCli_Blank_WriteError_Bad(t *core.T) {
	SetStdout(failingWriter{})
	defer SetStdout(nil)
	core.AssertNotPanics(t, func() { Blank() })
}

func TestCli_Print_RealOutput_Good(t *core.T) {
	rec := &recordingWriter{}
	SetStdout(rec)
	defer SetStdout(nil)
	Print("count=%d", 7)
	core.AssertEqual(t, "count=7", string(rec.data))
}

func TestCli_Text_RealOutput_Good(t *core.T) {
	rec := &recordingWriter{}
	SetStdout(rec)
	defer SetStdout(nil)
	Text("hephaestus")
	core.AssertEqual(t, "hephaestus\n", string(rec.data))
}

func TestCli_Blank_RealOutput_Good(t *core.T) {
	rec := &recordingWriter{}
	SetStdout(rec)
	defer SetStdout(nil)
	Blank()
	core.AssertEqual(t, "\n", string(rec.data))
}
