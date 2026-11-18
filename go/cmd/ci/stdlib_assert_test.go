package ci

import (
	core "github.com/dappcore/go"
	"github.com/dappcore/go-build/go/internal/testassert"
)

var (
	stdlibAssertContains = testassert.Contains
)

type ciCmdFatal interface {
	Helper()
	Fatalf(format string, args ...any)
}

func requireCIOK(t ciCmdFatal, result core.Result) {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
}

func requireCIBytes(t ciCmdFatal, result core.Result) []byte {
	t.Helper()
	requireCIOK(t, result)
	value, ok := result.Value.([]byte)
	if !ok {
		t.Fatalf("expected []byte result, got %T", result.Value)
	}
	return value
}
