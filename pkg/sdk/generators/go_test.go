package generators

import (
	"context"
	"testing"
	"time"

	"dappco.re/go/build/internal/ax"
)

func TestGo_GoGeneratorAvailable_Good(t *testing.T) {
	g := NewGoGenerator()

	// These should not panic
	lang := g.Language()
	if lang != "go" {
		t.Errorf("expected language 'go', got '%s'", lang)
	}

	_ = g.Available()

	install := g.Install()
	if install == "" {
		t.Error("expected non-empty install instructions")
	}
}

func TestGo_GoGeneratorGenerate_Good(t *testing.T) {
	g := NewGoGenerator()
	if _, err := g.resolveNativeCli(); err != nil && !dockerAvailable() {
		t.Skip("no Go generator available (neither native nor docker)")
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
