package generators

import (
	"context"
	"testing"
	"time"

	"dappco.re/go/core/build/internal/ax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dockerAvailable checks if docker is available for fallback generation.
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

func TestTypeScript_TypeScriptGeneratorGenerate_Good(t *testing.T) {
	g := NewTypeScriptGenerator()
	if !g.Available() && !dockerAvailable() {
		t.Skip("no TypeScript generator available (neither native nor docker)")
	}

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
