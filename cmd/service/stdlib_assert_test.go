package servicecmd

import (
	core "dappco.re/go"
	"dappco.re/go/build/internal/testassert"
)

var (
	stdlibAssertEqual         = testassert.Equal
	stdlibAssertNil           = testassert.Nil
	stdlibAssertEmpty         = testassert.Empty
	stdlibAssertZero          = testassert.Zero
	stdlibAssertContains      = testassert.Contains
	stdlibAssertElementsMatch = testassert.ElementsMatch
)

type serviceCmdFatal interface {
	Helper()
	Fatalf(format string, args ...any)
}

func requireServiceCmdOK(t serviceCmdFatal, result core.Result) {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
}

func requireServiceCmdBytes(t serviceCmdFatal, result core.Result) []byte {
	t.Helper()
	requireServiceCmdOK(t, result)
	value, ok := result.Value.([]byte)
	if !ok {
		t.Fatalf("expected []byte result, got %T", result.Value)
	}
	return value
}

func requireServiceCmdError(t serviceCmdFatal, result core.Result) string {
	t.Helper()
	if result.OK {
		t.Fatalf("expected error result, got %v", result.Value)
	}
	return result.Error()
}
