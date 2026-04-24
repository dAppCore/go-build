package generators

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/core"
	"errors"
	"os"
)

// dockerAvailable checks whether Docker can run fallback generation.
func dockerAvailable() bool {
	return ax.Exec(context.Background(), "docker", "info") == nil
}

// createTestSpec creates a minimal OpenAPI spec for testing.
func createTestSpec(t *testing.T, dir string) string {
	t.Helper()
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /health:
    get:
      summary: Health check
      responses:
        "200":
          description: OK
`
	specPath := ax.Join(dir, "openapi.yaml")
	if err := ax.WriteFile(specPath, []byte(spec), 0o644); err != nil {
		t.Fatalf("failed to write test spec: %v", err)
	}
	return specPath
}

func writeFakeTypeScriptGenerator(t *testing.T, dir string) string {
	t.Helper()

	commandPath := ax.Join(dir, "openapi-typescript-codegen")
	script := `#!/bin/sh
set -eu
output_dir=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --output)
      shift
      output_dir="$1"
      ;;
    --output=*)
      output_dir="${1#--output=}"
      ;;
  esac
  shift
done
if [ -n "$output_dir" ]; then
  mkdir -p "$output_dir/core"
  printf 'export * from "./core/client";\n' > "$output_dir/index.ts"
  printf 'export const client = true;\n' > "$output_dir/core/client.ts"
fi
`
	if err := ax.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return commandPath
}

func TestTypeScript_TypeScriptGeneratorAvailable_Good(t *testing.T) {
	g := NewTypeScriptGenerator()

	// These should not panic
	lang := g.Language()
	if lang != "typescript" {
		t.Errorf("expected language 'typescript', got '%s'", lang)
	}

	_ = g.Available()

	install := g.Install()
	if install == "" {
		t.Error("expected non-empty install instructions")
	}
}

func TestTypeScript_TypeScriptGeneratorNpxAvailabilityUsesProbeTimeout_Bad(t *testing.T) {
	setAvailabilityProbeTimeout(t, 20*time.Millisecond)

	npxDir := t.TempDir()
	npxPath := ax.Join(npxDir, "npx")
	if err := ax.WriteFile(npxPath, []byte("#!/bin/sh\nwhile :; do :; done\n"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Setenv("PATH", npxDir)

	started := time.Now()
	if NewTypeScriptGenerator().npxAvailable() {
		t.Fatal("expected false")
	}
	if time.Since(started) >= 500*time.Millisecond {
		t.Fatalf("expected %v to be less than %v", time.Since(started), 500*time.Millisecond)
	}

}

func TestTypeScript_TypeScriptGeneratorGenerate_Good(t *testing.T) {
	commandDir := t.TempDir()
	writeFakeTypeScriptGenerator(t, commandDir)
	t.Setenv("PATH", commandDir+core.Env("PS")+core.Env("PATH"))

	g := NewTypeScriptGenerator()
	if !(g.Available()) {
		t.Fatal("expected true")

		// Create temp directories
	}

	tmpDir := t.TempDir()
	specPath := createTestSpec(t, tmpDir)
	outputDir := ax.Join(tmpDir, "output")

	opts := Options{
		SpecPath:    specPath,
		OutputDir:   outputDir,
		PackageName: "testclient",
		Version:     "1.0.0",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	err := g.Generate(ctx, opts)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify output directory was created
	if !ax.Exists(outputDir) {
		t.Error("output directory was not created")
	}
	if _, err := os.Stat(ax.Join(outputDir, "src", "index.ts")); err != nil {
		t.Fatalf("expected file to exist: %v", ax.Join(outputDir, "src", "index.ts"))
	}
	if _, err := os.Stat(ax.Join(outputDir, "src", "core", "client.ts")); err != nil {
		t.Fatalf("expected file to exist: %v", ax.Join(outputDir, "src", "core", "client.ts"))
	}
	if _, err := os.Stat(ax.Join(outputDir, "index.ts")); err == nil {
		t.Fatalf("expected file not to exist: %v", ax.Join(outputDir, "index.ts"))
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}

	content, err := ax.ReadFile(ax.Join(outputDir, "package.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	manifest := map[string]any{}
	if err := json.Unmarshal(content, &manifest); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual("testclient", manifest["name"]) {
		t.Fatalf("want %v, got %v", "testclient", manifest["name"])
	}
	if !stdlibAssertEqual("1.0.0", manifest["version"]) {
		t.Fatalf("want %v, got %v", "1.0.0", manifest["version"])
	}
	if !stdlibAssertEqual([]any{"src"}, manifest["files"]) {
		t.Fatalf("want %v, got %v", []any{"src"}, manifest["files"])
	}
	if !stdlibAssertEqual("./src/index.ts", manifest["types"]) {
		t.Fatalf("want %v, got %v", "./src/index.ts", manifest["types"])
	}

}

func TestTypeScript_finalizeTypeScriptOutputNormalizesRootLayout_Good(t *testing.T) {
	stagingDir := t.TempDir()
	if err := ax.MkdirAll(ax.Join(stagingDir, "apis"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.MkdirAll(ax.Join(stagingDir, "models"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(stagingDir, "index.ts"), []byte("export * from './apis';\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(stagingDir, "runtime.ts"), []byte("export const runtime = true;\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(stagingDir, "apis", "default.ts"), []byte("export const api = true;\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(stagingDir, "models", "widget.ts"), []byte("export type Widget = {};\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(stagingDir, "README.md"), []byte("# SDK\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(stagingDir, "package.json"), []byte("{\"scripts\":{\"build\":\"tsc\"}}\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outputDir := ax.Join(t.TempDir(), "typescript")
	if err := finalizeTypeScriptOutput(stagingDir, Options{OutputDir: outputDir, PackageName: "@example/sdk", Version: "2.3.4"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(ax.Join(outputDir, "src", "index.ts")); err != nil {
		t.Fatalf("expected file to exist: %v", ax.Join(outputDir, "src", "index.ts"))
	}
	if _, err := os.Stat(ax.Join(outputDir, "src", "runtime.ts")); err != nil {
		t.Fatalf("expected file to exist: %v", ax.Join(outputDir, "src", "runtime.ts"))
	}
	if _, err := os.Stat(ax.Join(outputDir, "src", "apis", "default.ts")); err != nil {
		t.Fatalf("expected file to exist: %v", ax.Join(outputDir, "src", "apis", "default.ts"))
	}
	if _, err := os.Stat(ax.Join(outputDir, "src", "models", "widget.ts")); err != nil {
		t.Fatalf("expected file to exist: %v", ax.Join(outputDir, "src", "models", "widget.ts"))
	}
	if _, err := os.Stat(ax.Join(outputDir, "README.md")); err != nil {
		t.Fatalf("expected file to exist: %v", ax.Join(outputDir, "README.md"))
	}
	if _, err := os.Stat(ax.Join(outputDir, "index.ts")); err == nil {
		t.Fatalf("expected file not to exist: %v", ax.Join(outputDir, "index.ts"))
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	if _, err := os.Stat(ax.Join(outputDir, "runtime.ts")); err == nil {
		t.Fatalf("expected file not to exist: %v", ax.Join(outputDir, "runtime.ts"))
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}

	content, err := ax.ReadFile(ax.Join(outputDir, "package.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	manifest := map[string]any{}
	if err := json.Unmarshal(content, &manifest); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual("@example/sdk", manifest["name"]) {
		t.Fatalf("want %v, got %v", "@example/sdk", manifest["name"])
	}
	if !stdlibAssertEqual("2.3.4", manifest["version"]) {
		t.Fatalf("want %v, got %v", "2.3.4", manifest["version"])
	}
	if !stdlibAssertEqual("module", manifest["type"]) {
		t.Fatalf("want %v, got %v", "module", manifest["type"])
	}
	if !stdlibAssertEqual("./src/index.ts", manifest["types"]) {
		t.Fatalf("want %v, got %v", "./src/index.ts", manifest["types"])
	}

	scripts, ok := manifest["scripts"].(map[string]any)
	if !(ok) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("tsc", scripts["build"]) {
		t.Fatalf("want %v, got %v", "tsc", scripts["build"])
	}

}

func TestTypeScript_TypeScriptGeneratorGenerate_Bad(t *testing.T) {
	resetDockerRuntimeState()
	t.Cleanup(resetDockerRuntimeState)

	dockerDir := t.TempDir()
	dockerPath := ax.Join(dockerDir, "docker")
	if err := ax.WriteFile(dockerPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Setenv("PATH", dockerDir)

	tmpDir := t.TempDir()
	specPath := createTestSpec(t, tmpDir)
	outputDir := ax.Join(tmpDir, "output")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := NewTypeScriptGenerator().Generate(ctx, Options{
		SpecPath:    specPath,
		OutputDir:   outputDir,
		PackageName: "testclient",
		Version:     "1.0.0",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "context canceled") {
		t.Fatalf("expected %v to contain %v", err.Error(), "context canceled")
	}

}
