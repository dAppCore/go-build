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

func TestRunBuildSDKInDir_ValidSpecDryRunGood(t *testing.T) {
	tmpDir := t.TempDir()
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(tmpDir, "openapi.yaml"), []byte(validBuildOpenAPISpec), 0o644))

	requireBuildCmdOK(t, runBuildSDKInDir(context.Background(), tmpDir, "", "go", "", true, false))

}

func TestRunBuildSDKInDir_UsesBuildSDKConfigGood(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := ax.Join(tmpDir, "docs", "openapi.yaml")
	requireBuildCmdOK(t, ax.MkdirAll(ax.Dir(specPath), 0o755))
	requireBuildCmdOK(t, ax.WriteFile(specPath, []byte(validBuildOpenAPISpec), 0o644))
	requireBuildCmdOK(t, ax.MkdirAll(ax.Join(tmpDir, ".core"), 0o755))
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(tmpDir, ".core", "build.yaml"), []byte(`version: 1
sdk:
  spec: docs/openapi.yaml
  languages:
    - go
`), 0o644))

	requireBuildCmdOK(t, runBuildSDKInDir(context.Background(), tmpDir, "", "", "", true, false))

}

func TestRunBuildSDKInDir_InvalidDocumentBad(t *testing.T) {
	tmpDir := t.TempDir()
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(tmpDir, "openapi.yaml"), []byte(`openapi: "3.0.0"
info:
  title: Test API
paths: {}
`), 0o644))

	message := requireBuildCmdError(t, runBuildSDKInDir(context.Background(), tmpDir, "", "", "", true, false))
	if !stdlibAssertContains(message, "invalid OpenAPI spec") {
		t.Fatalf("expected %v to contain %v", message, "invalid OpenAPI spec")
	}

}
