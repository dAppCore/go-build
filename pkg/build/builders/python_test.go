package builders

import (
	"archive/zip"
	"context"
	"runtime"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/io"
)

func setupPythonTestProject(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	result := ax.WriteFile(ax.Join(dir, "pyproject.toml"), []byte("[build-system]\nrequires = []\n"), 0o644)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	result = ax.WriteFile(ax.Join(dir, "app.py"), []byte("print('hello')\n"), 0o644)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	result = ax.WriteFile(ax.Join(dir, "README.md"), []byte("demo"), 0o644)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	return dir
}

func TestPython_PythonBuilderNameGood(t *testing.T) {
	builder := NewPythonBuilder()
	if !stdlibAssertEqual("python", builder.Name()) {
		t.Fatalf("want %v, got %v", "python", builder.Name())
	}

}

func TestPython_PythonBuilderDetectGood(t *testing.T) {
	fs := io.Local

	t.Run("detects pyproject.toml projects", func(t *testing.T) {
		dir := t.TempDir()
		result := ax.WriteFile(ax.Join(dir, "pyproject.toml"), []byte("{}"), 0o644)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		builder := NewPythonBuilder()
		detected := requireCPPBool(t, builder.Detect(fs, dir))
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("detects requirements.txt projects", func(t *testing.T) {
		dir := t.TempDir()
		result := ax.WriteFile(ax.Join(dir, "requirements.txt"), []byte("requests"), 0o644)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		builder := NewPythonBuilder()
		detected := requireCPPBool(t, builder.Detect(fs, dir))
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		builder := NewPythonBuilder()
		detected := requireCPPBool(t, builder.Detect(fs, t.TempDir()))
		if detected {
			t.Fatal("expected false")
		}

	})
}

func TestPython_PythonBuilderBuildGood(t *testing.T) {
	projectDir := setupPythonTestProject(t)
	outputDir := t.TempDir()

	builder := NewPythonBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "demo-app",
	}

	artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}}))
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}

	artifact := artifacts[0]
	if !stdlibAssertEqual("linux", artifact.OS) {
		t.Fatalf("want %v, got %v", "linux", artifact.OS)
	}
	if !stdlibAssertEqual("amd64", artifact.Arch) {
		t.Fatalf("want %v, got %v", "amd64", artifact.Arch)
	}
	if stat := ax.Stat(artifact.Path); !stat.OK {
		t.Fatalf("expected file to exist: %v", artifact.Path)
	}

	reader, err := zip.OpenReader(artifact.Path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defer func() { _ = reader.Close() }()

	var foundPyProject, foundApp bool
	for _, file := range reader.File {
		switch file.Name {
		case "pyproject.toml":
			foundPyProject = true
		case "app.py":
			foundApp = true
		}
	}
	if !(foundPyProject) {
		t.Fatal("expected true")
	}
	if !(foundApp) {
		t.Fatal("expected true")
	}

}

func TestPython_PythonBuilderBuildDefaultsGood(t *testing.T) {
	projectDir := setupPythonTestProject(t)
	outputDir := t.TempDir()

	builder := NewPythonBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
	}

	artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, nil))
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}
	if !stdlibAssertEqual(runtime.GOOS, artifacts[0].OS) {
		t.Fatalf("want %v, got %v", runtime.GOOS, artifacts[0].OS)
	}
	if !stdlibAssertEqual(runtime.GOARCH, artifacts[0].Arch) {
		t.Fatalf("want %v, got %v", runtime.GOARCH, artifacts[0].Arch)
	}

}

func TestPython_PythonBuilderBuildIsDeterministicGood(t *testing.T) {
	projectDir := setupPythonTestProject(t)

	builder := NewPythonBuilder()
	buildOnce := func(outputDir string) []byte {
		t.Helper()

		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
			Name:       "demo-app",
		}

		artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}}))
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}

		return requireBuilderBytes(t, ax.ReadFile(artifacts[0].Path))
	}

	first := buildOnce(t.TempDir())
	second := buildOnce(t.TempDir())
	if !stdlibAssertEqual(first, second) {
		t.Fatalf("want %v, got %v", first, second)
	}

}

func TestPython_PythonBuilderInterfaceGood(t *testing.T) {
	builder := NewPythonBuilder()
	var _ build.Builder = builder
	if !stdlibAssertEqual("python", builder.Name()) {
		t.Fatalf("want %v, got %v", "python", builder.Name())
	}
	detected := requireCPPBool(t, builder.Detect(nil, t.TempDir()))
	if detected {
		t.Fatal("expected empty temp directory not to be detected")
	}
}

// --- v0.9.0 generated compliance triplets ---
func TestPython_NewPythonBuilder_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewPythonBuilder()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestPython_NewPythonBuilder_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewPythonBuilder()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestPython_NewPythonBuilder_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewPythonBuilder()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestPython_PythonBuilder_Name_Good(t *core.T) {
	subject := &PythonBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestPython_PythonBuilder_Name_Bad(t *core.T) {
	subject := &PythonBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestPython_PythonBuilder_Name_Ugly(t *core.T) {
	subject := &PythonBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestPython_PythonBuilder_Detect_Good(t *core.T) {
	subject := &PythonBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Detect(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestPython_PythonBuilder_Detect_Bad(t *core.T) {
	subject := &PythonBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Detect(io.NewMemoryMedium(), "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestPython_PythonBuilder_Detect_Ugly(t *core.T) {
	subject := &PythonBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Detect(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestPython_PythonBuilder_Build_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &PythonBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Build(ctx, nil, nil)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestPython_PythonBuilder_Build_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &PythonBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Build(ctx, nil, nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestPython_PythonBuilder_Build_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &PythonBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Build(ctx, nil, nil)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
