package builders

import (
	"testing"

	"dappco.re/go/build/pkg/build"
)

func TestResolver_InitRegistersDefaultBuilderResolver_Good(t *testing.T) {
	resolver := build.DefaultBuilderResolver()
	if stdlibAssertNil(resolver) {
		t.Fatal("expected non-nil")
	}

	builder, err := resolver(build.ProjectTypeGo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdlibAssertNil(builder) {
		t.Fatal("expected non-nil")
	}
	if !stdlibAssertEqual("go", builder.Name()) {
		t.Fatalf("want %v, got %v", "go", builder.Name())
	}

}
