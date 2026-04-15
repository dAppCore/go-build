package generators

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	require.NoError(t, ax.WriteFile(commandPath, []byte(script), 0o755))
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
	require.NoError(t, ax.WriteFile(npxPath, []byte("#!/bin/sh\nwhile :; do :; done\n"), 0o755))
	t.Setenv("PATH", npxDir)

	started := time.Now()
	assert.False(t, NewTypeScriptGenerator().npxAvailable())
	assert.Less(t, time.Since(started), 500*time.Millisecond)
}

func TestTypeScript_TypeScriptGeneratorGenerate_Good(t *testing.T) {
	commandDir := t.TempDir()
	writeFakeTypeScriptGenerator(t, commandDir)
	t.Setenv("PATH", commandDir+core.Env("PS")+core.Env("PATH"))

	g := NewTypeScriptGenerator()
	require.True(t, g.Available())

	// Create temp directories
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
	assert.FileExists(t, ax.Join(outputDir, "src", "index.ts"))
	assert.FileExists(t, ax.Join(outputDir, "src", "core", "client.ts"))
	assert.NoFileExists(t, ax.Join(outputDir, "index.ts"))

	content, err := ax.ReadFile(ax.Join(outputDir, "package.json"))
	require.NoError(t, err)

	manifest := map[string]any{}
	require.NoError(t, json.Unmarshal(content, &manifest))
	assert.Equal(t, "testclient", manifest["name"])
	assert.Equal(t, "1.0.0", manifest["version"])
	assert.Equal(t, []any{"src"}, manifest["files"])
	assert.Equal(t, "./src/index.ts", manifest["types"])
}

func TestTypeScript_finalizeTypeScriptOutputNormalizesRootLayout_Good(t *testing.T) {
	stagingDir := t.TempDir()
	require.NoError(t, ax.MkdirAll(ax.Join(stagingDir, "apis"), 0o755))
	require.NoError(t, ax.MkdirAll(ax.Join(stagingDir, "models"), 0o755))
	require.NoError(t, ax.WriteFile(ax.Join(stagingDir, "index.ts"), []byte("export * from './apis';\n"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(stagingDir, "runtime.ts"), []byte("export const runtime = true;\n"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(stagingDir, "apis", "default.ts"), []byte("export const api = true;\n"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(stagingDir, "models", "widget.ts"), []byte("export type Widget = {};\n"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(stagingDir, "README.md"), []byte("# SDK\n"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(stagingDir, "package.json"), []byte("{\"scripts\":{\"build\":\"tsc\"}}\n"), 0o644))

	outputDir := ax.Join(t.TempDir(), "typescript")
	require.NoError(t, finalizeTypeScriptOutput(stagingDir, Options{
		OutputDir:   outputDir,
		PackageName: "@example/sdk",
		Version:     "2.3.4",
	}))

	assert.FileExists(t, ax.Join(outputDir, "src", "index.ts"))
	assert.FileExists(t, ax.Join(outputDir, "src", "runtime.ts"))
	assert.FileExists(t, ax.Join(outputDir, "src", "apis", "default.ts"))
	assert.FileExists(t, ax.Join(outputDir, "src", "models", "widget.ts"))
	assert.FileExists(t, ax.Join(outputDir, "README.md"))
	assert.NoFileExists(t, ax.Join(outputDir, "index.ts"))
	assert.NoFileExists(t, ax.Join(outputDir, "runtime.ts"))

	content, err := ax.ReadFile(ax.Join(outputDir, "package.json"))
	require.NoError(t, err)

	manifest := map[string]any{}
	require.NoError(t, json.Unmarshal(content, &manifest))
	assert.Equal(t, "@example/sdk", manifest["name"])
	assert.Equal(t, "2.3.4", manifest["version"])
	assert.Equal(t, "module", manifest["type"])
	assert.Equal(t, "./src/index.ts", manifest["types"])

	scripts, ok := manifest["scripts"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "tsc", scripts["build"])
}

func TestTypeScript_TypeScriptGeneratorGenerate_Bad(t *testing.T) {
	resetDockerRuntimeState()
	t.Cleanup(resetDockerRuntimeState)

	dockerDir := t.TempDir()
	dockerPath := ax.Join(dockerDir, "docker")
	require.NoError(t, ax.WriteFile(dockerPath, []byte("#!/bin/sh\nexit 0\n"), 0o755))
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

	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}
