package publishers

import (
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/testassert"
)

var (
	stdlibAssertEqual         = testassert.Equal
	stdlibAssertNil           = testassert.Nil
	stdlibAssertEmpty         = testassert.Empty
	stdlibAssertContains      = testassert.Contains
)

func requirePublisherOK(t *testing.T, result core.Result) {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
}

func requirePublisherString(t *testing.T, result core.Result) string {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	return result.Value.(string)
}

func requirePublisherBytes(t *testing.T, result core.Result) []byte {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	return result.Value.([]byte)
}

func requirePublisherDirEntries(t *testing.T, result core.Result) []core.FsDirEntry {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	return result.Value.([]core.FsDirEntry)
}

func requirePublisherError(t *testing.T, result core.Result) string {
	t.Helper()
	if result.OK {
		t.Fatal("expected error")
	}
	return result.Error()
}
