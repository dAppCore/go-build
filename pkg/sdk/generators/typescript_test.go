package generators

import (
	"context"
	"testing"
	"time"

	"dappco.re/go/core"
	"dappco.re/go/build/internal/ax"
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
  mkdir -p "$output_dir"
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
