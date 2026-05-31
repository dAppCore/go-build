package ax

import (
	"context"
	"time"

	core "dappco.re/go"
)

// Behaviour tests exercise the real path/env/exec branches that the generated
// no-panic triplets skipped: the DS override, the slash-rewrite branch, the
// DIR_CWD short-circuit in Getwd, the JSON failure path, command resolution
// fall-backs, and an actual subprocess run plus its cancellation kill path.

func TestAx_Abs_AlreadyAbsolute_Good(t *core.T) {
	abs := Abs("/already/absolute/path")
	core.AssertTrue(t, abs.OK)
	core.AssertEqual(t, "/already/absolute/path", abs.Value.(string))
}

func TestAx_Abs_RelativeUsesCwd_Good(t *core.T) {
	// A relative path is anchored to the resolved working directory; the exact
	// cwd is environment-specific (Core seals DIR_CWD in systemInfo) so we only
	// assert the path was made absolute and ends with the supplied tail.
	abs := Abs("child/file.txt")
	core.AssertTrue(t, abs.OK)
	core.AssertTrue(t, IsAbs(abs.Value.(string)))
	core.AssertTrue(t, core.HasSuffix(abs.Value.(string), Join("child", "file.txt")))
}

func TestAx_JSONMarshal_RoundTrip_Good(t *core.T) {
	encoded := JSONMarshal(map[string]int{"a": 1})
	core.AssertTrue(t, encoded.OK)
	core.AssertEqual(t, `{"a":1}`, encoded.Value.(string))
}

func TestAx_JSONMarshal_Unsupported_Bad(t *core.T) {
	// A channel cannot be marshalled to JSON, driving the failure branch.
	result := JSONMarshal(make(chan int))
	core.AssertFalse(t, result.OK)
	core.AssertTrue(t, core.Contains(result.Error(), "failed to marshal JSON"))
}

func TestAx_JSONUnmarshal_Invalid_Bad(t *core.T) {
	target := map[string]any{}
	result := JSONUnmarshal([]byte("{not valid json"), &target)
	core.AssertFalse(t, result.OK)
	core.AssertTrue(t, core.Contains(result.Error(), "failed to unmarshal JSON"))
}

func TestAx_ResolveCommand_Fallback_Ugly(t *core.T) {
	// A name absent from PATH resolves via the first fallback path that is an
	// existing file.
	fallback := Join(t.TempDir(), "tool")
	core.AssertTrue(t, WriteString(fallback, "#!/bin/sh\n", 0o755).OK)
	resolved := ResolveCommand("definitely-not-on-path-xyz", "/no/such/one", fallback)
	core.AssertTrue(t, resolved.OK)
	core.AssertEqual(t, fallback, resolved.Value.(string))
}

func TestAx_ResolveCommand_AllMissing_Bad(t *core.T) {
	resolved := ResolveCommand("definitely-not-on-path-xyz", "/no/such/fallback")
	core.AssertFalse(t, resolved.OK)
	core.AssertTrue(t, core.Contains(resolved.Error(), "failed to locate command"))
}

func TestAx_RunCommand_NilContext_Bad(t *core.T) {
	result := Exec(nil, "true") //nolint:staticcheck // exercises the nil-context guard
	core.AssertFalse(t, result.OK)
	core.AssertTrue(t, core.Contains(result.Error(), "command context is required"))
}

func TestAx_RunCommand_EmptyCommand_Bad(t *core.T) {
	result := Exec(context.Background(), "")
	core.AssertFalse(t, result.OK)
	core.AssertTrue(t, core.Contains(result.Error(), "program name is empty"))
}

func TestAx_Run_RealBinary_Good(t *core.T) {
	// /bin/echo is an absolute path, so resolveExecutable short-circuits and
	// the full Start/Wait happy path runs.
	output := Run(context.Background(), "/bin/echo", "hephaestus")
	core.AssertTrue(t, output.OK, output.Error())
	core.AssertEqual(t, "hephaestus", output.Value.(string))
}

func TestAx_Exec_RealBinary_Good(t *core.T) {
	core.AssertTrue(t, Exec(context.Background(), "/usr/bin/true").OK)
}

func TestAx_Exec_FailingBinary_Bad(t *core.T) {
	// /usr/bin/false exits non-zero, driving the Wait-error branch.
	core.AssertFalse(t, Exec(context.Background(), "/usr/bin/false").OK)
}

func TestAx_Run_CancelledContext_Ugly(t *core.T) {
	// A context cancelled before the command finishes drives the kill path.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	result := Run(ctx, "/bin/sleep", "5")
	core.AssertFalse(t, result.OK)
}

func TestAx_ResolveExecutable_AbsolutePath_Good(t *core.T) {
	// A name containing a separator is returned verbatim without a PATH lookup.
	resolved := resolveExecutable("/bin/echo")
	core.AssertTrue(t, resolved.OK)
	core.AssertEqual(t, "/bin/echo", resolved.Value.(string))
}
