package builders

import (
	"testing"

	"dappco.re/go/build/pkg/build"
)

func TestResolver_InitRegistersDefaultBuilderResolverGood(t *testing.T) {
	resolver := build.DefaultBuilderResolver()
	if stdlibAssertNil(resolver) {
		t.Fatal("expected non-nil")
	}

	result := resolver(build.ProjectTypeGo)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	builder := result.Value.(build.Builder)
	if stdlibAssertNil(builder) {
		t.Fatal("expected non-nil")
	}
	if !stdlibAssertEqual("go", builder.Name()) {
		t.Fatalf("want %v, got %v", "go", builder.Name())
	}

}
