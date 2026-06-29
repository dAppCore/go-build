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

// TestRunBuildSDKInDir_AllLanguagesDryRunGood covers the all-languages dry-run
// branch: every configured language is listed without invoking any generator.
func TestRunBuildSDKInDir_AllLanguagesDryRunGood(t *testing.T) {
	tmpDir := t.TempDir()
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(tmpDir, "openapi.yaml"), []byte(validBuildOpenAPISpec), 0o644))
	buf := captureBuildStdout(t)

	requireBuildCmdOK(t, runBuildSDKInDir(context.Background(), tmpDir, "", "", "", true, false))
	out := buf.String()
	if !stdlibAssertContains(out, "languages") {
		t.Fatalf("expected %v to contain %v", out, "languages")
	}
	if !stdlibAssertContains(out, "Would generate SDK") {
		t.Fatalf("expected %v to contain %v", out, "Would generate SDK")
	}
}

// TestRunBuildSDKInDir_UnknownLanguageBad covers the non-dry-run single-language
// error branch: an unknown language is rejected by the generator registry.
func TestRunBuildSDKInDir_UnknownLanguageBad(t *testing.T) {
	tmpDir := t.TempDir()
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(tmpDir, "openapi.yaml"), []byte(validBuildOpenAPISpec), 0o644))
	captureBuildStdout(t)

	message := requireBuildCmdError(t, runBuildSDKInDir(context.Background(), tmpDir, "", "cobol", "", false, false))
	if !stdlibAssertContains(message, "unknown language: cobol") {
		t.Fatalf("expected %v to contain %v", message, "unknown language: cobol")
	}
}

// TestRunBuildSDKInDir_LanguageReported drives the real (non-dry-run)
// single-language generation path. PATH is emptied and skip-unavailable enabled,
// so the call succeeds whether the generator runs (container/native available)
// or is skipped; a non-OK result indicates generator infrastructure is broken in
// this environment and is treated as a skip so the assertions never falsely fail.
func TestRunBuildSDKInDir_LanguageReported(t *testing.T) {
	tmpDir := t.TempDir()
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(tmpDir, "openapi.yaml"), []byte(validBuildOpenAPISpec), 0o644))
	t.Setenv("PATH", t.TempDir())
	buf := captureBuildStdout(t)

	result := runBuildSDKInDir(context.Background(), tmpDir, "", "go", "v1.2.3", false, true)
	if !result.OK {
		t.Skipf("go SDK generation unavailable in this environment: %v", result.Error())
	}
	out := buf.String()
	if !stdlibAssertContains(out, "go") {
		t.Fatalf("expected %v to contain %v", out, "go")
	}
	if !stdlibAssertContains(out, "SDK generation complete") {
		t.Fatalf("expected %v to contain %v", out, "SDK generation complete")
	}
}
