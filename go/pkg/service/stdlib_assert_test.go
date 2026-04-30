package service

import (
	"time"

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

type serviceFatal interface {
	Helper()
	Fatalf(format string, args ...any)
}

func requireServiceOK(t serviceFatal, result core.Result) {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
}

func requireServiceConfig(t serviceFatal, result core.Result) Config {
	t.Helper()
	requireServiceOK(t, result)
	value, ok := result.Value.(Config)
	if !ok {
		t.Fatalf("expected Config result, got %T", result.Value)
	}
	return value
}

func requireServiceNativeFormat(t serviceFatal, result core.Result) NativeFormat {
	t.Helper()
	requireServiceOK(t, result)
	value, ok := result.Value.(NativeFormat)
	if !ok {
		t.Fatalf("expected NativeFormat result, got %T", result.Value)
	}
	return value
}

func requireServiceExportedConfig(t serviceFatal, result core.Result) ExportedConfig {
	t.Helper()
	requireServiceOK(t, result)
	value, ok := result.Value.(ExportedConfig)
	if !ok {
		t.Fatalf("expected ExportedConfig result, got %T", result.Value)
	}
	return value
}

func requireServiceSnapshot(t serviceFatal, result core.Result) map[string]time.Time {
	t.Helper()
	requireServiceOK(t, result)
	value, ok := result.Value.(map[string]time.Time)
	if !ok {
		t.Fatalf("expected snapshot result, got %T", result.Value)
	}
	return value
}

func requireServiceDirEntries(t serviceFatal, result core.Result) []core.FsDirEntry {
	t.Helper()
	requireServiceOK(t, result)
	value, ok := result.Value.([]core.FsDirEntry)
	if !ok {
		t.Fatalf("expected directory entries result, got %T", result.Value)
	}
	return value
}
