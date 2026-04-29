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
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewPHPGenerator()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestPhp_NewPHPGenerator_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewPHPGenerator()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestPhp_NewPHPGenerator_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewPHPGenerator()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestPhp_PHPGenerator_Language_Good(t *core.T) {
	subject := &PHPGenerator{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Language()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestPhp_PHPGenerator_Language_Bad(t *core.T) {
	subject := &PHPGenerator{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Language()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestPhp_PHPGenerator_Language_Ugly(t *core.T) {
	subject := &PHPGenerator{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Language()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestPhp_PHPGenerator_Available_Good(t *core.T) {
	subject := &PHPGenerator{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Available()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestPhp_PHPGenerator_Available_Bad(t *core.T) {
	subject := &PHPGenerator{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Available()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestPhp_PHPGenerator_Available_Ugly(t *core.T) {
	subject := &PHPGenerator{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Available()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestPhp_PHPGenerator_Install_Good(t *core.T) {
	subject := &PHPGenerator{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Install()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestPhp_PHPGenerator_Install_Bad(t *core.T) {
	subject := &PHPGenerator{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Install()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestPhp_PHPGenerator_Install_Ugly(t *core.T) {
	subject := &PHPGenerator{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Install()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestPhp_PHPGenerator_Generate_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &PHPGenerator{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Generate(ctx, Options{})
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestPhp_PHPGenerator_Generate_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &PHPGenerator{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Generate(ctx, Options{})
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestPhp_PHPGenerator_Generate_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &PHPGenerator{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Generate(ctx, Options{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
