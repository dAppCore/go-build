package generators

import (
	"context"
	"testing"
	"time"

	"dappco.re/go/build/internal/ax"
)

func TestPHP_PHPGeneratorAvailable_Good(t *testing.T) {
	g := NewPHPGenerator()

	// These should not panic
	lang := g.Language()
	if lang != "php" {
		t.Errorf("expected language 'php', got '%s'", lang)
	}

	_ = g.Available()

	install := g.Install()
	if install == "" {
		t.Error("expected non-empty install instructions")
	}
}

func TestPHP_PHPGeneratorGenerate_Good(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("no PHP generator available (docker not installed)")
	}

	g := NewPHPGenerator()

	// Create temp directories
	tmpDir := t.TempDir()
	specPath := createTestSpec(t, tmpDir)
	outputDir := ax.Join(tmpDir, "output")

	opts := Options{
		SpecPath:    specPath,
		OutputDir:   outputDir,
		PackageName: "TestClient",
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
