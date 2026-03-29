package generators

import (
	"context"
	"testing"
	"time"

	"dappco.re/go/core/build/internal/ax"
)

func TestPython_PythonGeneratorAvailable_Good(t *testing.T) {
	g := NewPythonGenerator()

	// These should not panic
	lang := g.Language()
	if lang != "python" {
		t.Errorf("expected language 'python', got '%s'", lang)
	}

	_ = g.Available()

	install := g.Install()
	if install == "" {
		t.Error("expected non-empty install instructions")
	}
}

func TestPython_PythonGeneratorGenerate_Good(t *testing.T) {
	g := NewPythonGenerator()
	if !g.Available() && !dockerAvailable() {
		t.Skip("no Python generator available (neither native nor docker)")
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
