package api

import (
	core "dappco.re/go"
	"dappco.re/go/build/internal/testassert"
	"dappco.re/go/build/pkg/build"
)

var (
	stdlibAssertEqual         = testassert.Equal
	stdlibAssertNil           = testassert.Nil
	stdlibAssertEmpty         = testassert.Empty
	stdlibAssertZero          = testassert.Zero
	stdlibAssertContains      = testassert.Contains
	stdlibAssertElementsMatch = testassert.ElementsMatch
)

type providerFatal interface {
	Helper()
	Fatalf(format string, args ...any)
}

func requireProviderOK(t providerFatal, result core.Result) {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
}

func requireProviderString(t providerFatal, result core.Result) string {
	t.Helper()
	requireProviderOK(t, result)
	value, ok := result.Value.(string)
	if !ok {
		t.Fatalf("expected string result, got %T", result.Value)
	}
	return value
}

func requireProviderBuilder(t providerFatal, result core.Result) build.Builder {
	t.Helper()
	requireProviderOK(t, result)
	value, ok := result.Value.(build.Builder)
	if !ok {
		t.Fatalf("expected build.Builder result, got %T", result.Value)
	}
	return value
}

func requireProviderProjectType(t providerFatal, result core.Result) build.ProjectType {
	t.Helper()
	requireProviderOK(t, result)
	value, ok := result.Value.(build.ProjectType)
	if !ok {
		t.Fatalf("expected build.ProjectType result, got %T", result.Value)
	}
	return value
}

func requireProviderError(t providerFatal, result core.Result) string {
	t.Helper()
	if result.OK {
		t.Fatalf("expected error result, got %v", result.Value)
	}
	return result.Error()
}
