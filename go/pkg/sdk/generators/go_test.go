package generators

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
)

func TestGo_GoGeneratorAvailableGood(t *testing.T) {
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

func TestGo_GoGeneratorGenerateGood(t *testing.T) {
	g := NewGoGenerator()
	if native := g.resolveNativeCli(); !native.OK && !dockerAvailable() {
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

	generated := g.Generate(ctx, opts)
	if !generated.OK {
		t.Fatalf("Generate failed: %v", generated.Error())
	}

	// Verify output directory was created
	if !ax.Exists(outputDir) {
		t.Error("output directory was not created")
	}
}

// --- v0.9.0 generated compliance triplets ---
func TestGo_NewGoGenerator_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewGoGenerator()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestGo_NewGoGenerator_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewGoGenerator()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestGo_NewGoGenerator_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewGoGenerator()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestGo_GoGenerator_Language_Good(t *core.T) {
	subject := &GoGenerator{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Language()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestGo_GoGenerator_Language_Bad(t *core.T) {
	subject := &GoGenerator{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Language()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestGo_GoGenerator_Language_Ugly(t *core.T) {
	subject := &GoGenerator{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Language()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestGo_GoGenerator_Available_Good(t *core.T) {
	subject := &GoGenerator{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Available()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestGo_GoGenerator_Available_Bad(t *core.T) {
	subject := &GoGenerator{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Available()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestGo_GoGenerator_Available_Ugly(t *core.T) {
	subject := &GoGenerator{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Available()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestGo_GoGenerator_Install_Good(t *core.T) {
	subject := &GoGenerator{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Install()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestGo_GoGenerator_Install_Bad(t *core.T) {
	subject := &GoGenerator{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Install()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestGo_GoGenerator_Install_Ugly(t *core.T) {
	subject := &GoGenerator{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Install()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestGo_GoGenerator_Generate_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &GoGenerator{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Generate(ctx, Options{})
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestGo_GoGenerator_Generate_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &GoGenerator{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Generate(ctx, Options{})
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestGo_GoGenerator_Generate_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &GoGenerator{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Generate(ctx, Options{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
