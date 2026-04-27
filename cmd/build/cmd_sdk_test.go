package buildcmd

import (
	"context"
	"testing"

	"dappco.re/go/build/internal/ax"
)

const validBuildOpenAPISpec = `openapi: "3.0.0"
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

func TestRunBuildSDKInDir_ValidSpecDryRun_Good(t *testing.T) {
	tmpDir := t.TempDir()
	if err := ax.WriteFile(ax.Join(tmpDir, "openapi.yaml"), []byte(validBuildOpenAPISpec), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := runBuildSDKInDir(context.Background(), tmpDir, "", "go", "", true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}

func TestRunBuildSDKInDir_UsesBuildSDKConfig_Good(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := ax.Join(tmpDir, "docs", "openapi.yaml")
	if err := ax.MkdirAll(ax.Dir(specPath), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(specPath, []byte(validBuildOpenAPISpec), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.MkdirAll(ax.Join(tmpDir, ".core"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(tmpDir, ".core", "build.yaml"), []byte(`version: 1
sdk:
  spec: docs/openapi.yaml
  languages:
    - go
`), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := runBuildSDKInDir(context.Background(), tmpDir, "", "", "", true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}

func TestRunBuildSDKInDir_InvalidDocument_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	if err := ax.WriteFile(ax.Join(tmpDir, "openapi.yaml"), []byte(`openapi: "3.0.0"
info:
  title: Test API
paths: {}
`), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := runBuildSDKInDir(context.Background(), tmpDir, "", "", "", true, false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "invalid OpenAPI spec") {
		t.Fatalf("expected %v to contain %v", err.Error(), "invalid OpenAPI spec")
	}

}
