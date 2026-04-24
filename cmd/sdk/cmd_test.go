package sdkcmd

import (
	"context"
	"errors"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/core"
	"dappco.re/go/core/cli/pkg/cli"
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

func TestRunSDKValidate_Good(t *testing.T) {
	tmpDir := t.TempDir()
	if err := ax.WriteFile(ax.Join(tmpDir, "openapi.yaml"), []byte(validOpenAPISpec), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := runSDKValidateInDir(context.Background(), tmpDir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}

func TestAddSDKCommands_RegistersGenerateAlias_Good(t *testing.T) {
	c := core.New()

	AddSDKCommands(c)
	if !(c.Command("sdk").OK) {
		t.Fatal("expected true")
	}
	if !(c.Command("sdk/generate").OK) {
		t.Fatal("expected true")
	}
	if !(c.Command("sdk/diff").OK) {
		t.Fatal("expected true")
	}
	if !(c.Command("sdk/validate").OK) {
		t.Fatal("expected true")
	}

}

func TestRunSDKGenerateInDir_ValidSpecDryRun_Good(t *testing.T) {
	tmpDir := t.TempDir()
	if err := ax.WriteFile(ax.Join(tmpDir, "openapi.yaml"), []byte(validOpenAPISpec), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := runSDKGenerateInDir(context.Background(), tmpDir, "", "go", "", true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}

func TestRunSDKGenerateInDir_UsesBuildSDKConfig_Good(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := ax.Join(tmpDir, "docs", "openapi.yaml")
	if err := ax.MkdirAll(ax.Dir(specPath), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(specPath, []byte(validOpenAPISpec), 0o644); err != nil {
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

	err := runSDKGenerateInDir(context.Background(), tmpDir, "", "", "", true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}

func TestRunSDKGenerateInDir_InvalidDocument_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	if err := ax.WriteFile(ax.Join(tmpDir, "openapi.yaml"), []byte(`openapi: "3.0.0"
info:
  title: Test API
paths: {}
`), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := runSDKGenerateInDir(context.Background(), tmpDir, "", "", "", true, false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "invalid OpenAPI spec") {
		t.Fatalf("expected %v to contain %v", err.Error(), "invalid OpenAPI spec")
	}

}

func TestRunSDKValidate_InvalidDocument_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	if err := ax.WriteFile(ax.Join(tmpDir, "openapi.yaml"), []byte(`openapi: "3.0.0"
info:
  title: Test API
paths: {}
`), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := runSDKValidateInDir(context.Background(), tmpDir, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "invalid OpenAPI spec") {
		t.Fatalf("expected %v to contain %v", err.Error(), "invalid OpenAPI spec")
	}

}

func TestRunSDKValidate_UsesBuildSDKConfig_Good(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := ax.Join(tmpDir, "docs", "openapi.yaml")
	if err := ax.MkdirAll(ax.Dir(specPath), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(specPath, []byte(validOpenAPISpec), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.MkdirAll(ax.Join(tmpDir, ".core"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(tmpDir, ".core", "build.yaml"), []byte(`version: 1
sdk:
  spec: docs/openapi.yaml
`), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := runSDKValidateInDir(context.Background(), tmpDir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}

func TestRunSDKDiffInDir_FailOnWarn_Good(t *testing.T) {
	tmpDir := t.TempDir()

	baseSpec := `openapi: "3.0.0"
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
          content:
            application/json:
              schema:
                type: object
                properties:
                  status:
                    type: string
                  detail:
                    type: string
`
	revSpec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.1.0"
paths:
  /health:
    get:
      operationId: getHealth
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  status:
                    type: string
`
	basePath := ax.Join(tmpDir, "base.yaml")
	specPath := ax.Join(tmpDir, "openapi.yaml")
	if err := ax.WriteFile(basePath, []byte(baseSpec), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(specPath, []byte(revSpec), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := runSDKDiffInDir(tmpDir, basePath, specPath, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = runSDKDiffInDir(tmpDir, basePath, specPath, true)
	if err == nil {
		t.Fatal("expected error")
	}

	var exitErr *cli.ExitError
	if !(errors.As(err, &exitErr)) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual(1, exitErr.Code) {
		t.Fatalf("want %v, got %v", 1, exitErr.Code)
	}

}
