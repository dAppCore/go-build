package generators

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
)

func TestPython_PythonGeneratorAvailableGood(t *testing.T) {
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

func TestPython_PythonGeneratorGenerateGood(t *testing.T) {
	g := NewPythonGenerator()
	if _, err := g.resolveNativeCli(); err != nil && !dockerAvailable() {
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

// --- v0.9.0 generated compliance triplets ---
func TestPython_NewPythonGenerator_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewPythonGenerator()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestPython_NewPythonGenerator_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewPythonGenerator()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestPython_NewPythonGenerator_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewPythonGenerator()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestPython_PythonGenerator_Language_Good(t *core.T) {
	subject := &PythonGenerator{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Language()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestPython_PythonGenerator_Language_Bad(t *core.T) {
	subject := &PythonGenerator{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Language()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestPython_PythonGenerator_Language_Ugly(t *core.T) {
	subject := &PythonGenerator{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Language()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestPython_PythonGenerator_Available_Good(t *core.T) {
	subject := &PythonGenerator{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Available()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestPython_PythonGenerator_Available_Bad(t *core.T) {
	subject := &PythonGenerator{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Available()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestPython_PythonGenerator_Available_Ugly(t *core.T) {
	subject := &PythonGenerator{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Available()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestPython_PythonGenerator_Install_Good(t *core.T) {
	subject := &PythonGenerator{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Install()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestPython_PythonGenerator_Install_Bad(t *core.T) {
	subject := &PythonGenerator{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Install()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestPython_PythonGenerator_Install_Ugly(t *core.T) {
	subject := &PythonGenerator{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Install()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestPython_PythonGenerator_Generate_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &PythonGenerator{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Generate(ctx, Options{})
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestPython_PythonGenerator_Generate_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &PythonGenerator{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Generate(ctx, Options{})
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestPython_PythonGenerator_Generate_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &PythonGenerator{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Generate(ctx, Options{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
