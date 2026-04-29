package sdk

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
)

const validOpenAPISpec = `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /health:
    get:
      operationId: getHealth
      responses:
        "200":
          description: OK
`

func TestValidateSpec_Good(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := ax.Join(tmpDir, "openapi.yaml")
	if err := ax.WriteFile(specPath, []byte(validOpenAPISpec), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sdk := New(tmpDir, nil)
	got, err := sdk.ValidateSpec(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(specPath, got) {
		t.Fatalf("want %v, got %v", specPath, got)
	}

}

func TestValidateSpec_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := ax.Join(tmpDir, "openapi.yaml")
	if err := ax.WriteFile(specPath, []byte("openapi: 3.0.0\ninfo: [\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sdk := New(tmpDir, nil)
	_, err := sdk.ValidateSpec(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "failed to load OpenAPI spec") {
		t.Fatalf("expected %v to contain %v", err.Error(), "failed to load OpenAPI spec")
	}

}

func TestValidateSpec_InvalidDocumentBad(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := ax.Join(tmpDir, "openapi.yaml")
	if err := ax.WriteFile(specPath, []byte(`openapi: "3.0.0"
info:
  title: Test API
paths: {}
`), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sdk := New(tmpDir, nil)
	_, err := sdk.ValidateSpec(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "invalid OpenAPI spec") {
		t.Fatalf("expected %v to contain %v", err.Error(), "invalid OpenAPI spec")
	}

}

// --- v0.9.0 generated compliance triplets ---
func TestValidate_SDK_ValidateSpec_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &SDK{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = subject.ValidateSpec(ctx)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestValidate_SDK_ValidateSpec_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &SDK{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = subject.ValidateSpec(ctx)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestValidate_SDK_ValidateSpec_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &SDK{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = subject.ValidateSpec(ctx)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
