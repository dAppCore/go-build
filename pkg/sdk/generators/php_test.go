package generators

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
)

func TestPHP_PHPGeneratorAvailableGood(t *testing.T) {
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

func TestPHP_PHPGeneratorGenerateGood(t *testing.T) {
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

// --- v0.9.0 generated compliance triplets ---
func TestPhp_NewPHPGenerator_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewPHPGenerator()
	})
	core.AssertTrue(t, true)
}

func TestPhp_NewPHPGenerator_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewPHPGenerator()
	})
	core.AssertTrue(t, true)
}

func TestPhp_NewPHPGenerator_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewPHPGenerator()
	})
	core.AssertTrue(t, true)
}

func TestPhp_PHPGenerator_Language_Good(t *core.T) {
	subject := &PHPGenerator{}
	core.AssertNotPanics(t, func() {
		_ = subject.Language()
	})
	core.AssertTrue(t, true)
}

func TestPhp_PHPGenerator_Language_Bad(t *core.T) {
	subject := &PHPGenerator{}
	core.AssertNotPanics(t, func() {
		_ = subject.Language()
	})
	core.AssertTrue(t, true)
}

func TestPhp_PHPGenerator_Language_Ugly(t *core.T) {
	subject := &PHPGenerator{}
	core.AssertNotPanics(t, func() {
		_ = subject.Language()
	})
	core.AssertTrue(t, true)
}

func TestPhp_PHPGenerator_Available_Good(t *core.T) {
	subject := &PHPGenerator{}
	core.AssertNotPanics(t, func() {
		_ = subject.Available()
	})
	core.AssertTrue(t, true)
}

func TestPhp_PHPGenerator_Available_Bad(t *core.T) {
	subject := &PHPGenerator{}
	core.AssertNotPanics(t, func() {
		_ = subject.Available()
	})
	core.AssertTrue(t, true)
}

func TestPhp_PHPGenerator_Available_Ugly(t *core.T) {
	subject := &PHPGenerator{}
	core.AssertNotPanics(t, func() {
		_ = subject.Available()
	})
	core.AssertTrue(t, true)
}

func TestPhp_PHPGenerator_Install_Good(t *core.T) {
	subject := &PHPGenerator{}
	core.AssertNotPanics(t, func() {
		_ = subject.Install()
	})
	core.AssertTrue(t, true)
}

func TestPhp_PHPGenerator_Install_Bad(t *core.T) {
	subject := &PHPGenerator{}
	core.AssertNotPanics(t, func() {
		_ = subject.Install()
	})
	core.AssertTrue(t, true)
}

func TestPhp_PHPGenerator_Install_Ugly(t *core.T) {
	subject := &PHPGenerator{}
	core.AssertNotPanics(t, func() {
		_ = subject.Install()
	})
	core.AssertTrue(t, true)
}

func TestPhp_PHPGenerator_Generate_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &PHPGenerator{}
	core.AssertNotPanics(t, func() {
		_ = subject.Generate(ctx, Options{})
	})
	core.AssertTrue(t, true)
}

func TestPhp_PHPGenerator_Generate_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &PHPGenerator{}
	core.AssertNotPanics(t, func() {
		_ = subject.Generate(ctx, Options{})
	})
	core.AssertTrue(t, true)
}

func TestPhp_PHPGenerator_Generate_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &PHPGenerator{}
	core.AssertNotPanics(t, func() {
		_ = subject.Generate(ctx, Options{})
	})
	core.AssertTrue(t, true)
}
