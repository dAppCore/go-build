package builders

import (
	"archive/zip"
	"context"
	"runtime"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/io"
	"os"
)

func setupPythonTestProject(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	if err := ax.WriteFile(ax.Join(dir, "pyproject.toml"), []byte("[build-system]\nrequires = []\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(dir, "app.py"), []byte("print('hello')\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(dir, "README.md"), []byte("demo"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return dir
}

func TestPython_PythonBuilderName_Good(t *testing.T) {
	builder := NewPythonBuilder()
	if !stdlibAssertEqual("python", builder.Name()) {
		t.Fatalf("want %v, got %v", "python", builder.Name())
	}

}

func TestPython_PythonBuilderDetect_Good(t *testing.T) {
	fs := io.Local

	t.Run("detects pyproject.toml projects", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "pyproject.toml"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewPythonBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("detects requirements.txt projects", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "requirements.txt"), []byte("requests"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewPythonBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		builder := NewPythonBuilder()
		detected, err := builder.Detect(fs, t.TempDir())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detected {
			t.Fatal("expected false")
		}

	})
}

func TestPython_PythonBuilderBuild_Good(t *testing.T) {
	projectDir := setupPythonTestProject(t)
	outputDir := t.TempDir()

	builder := NewPythonBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "demo-app",
	}

	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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
	if _, err := os.Stat(artifact.Path); err != nil {
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

func TestPython_PythonBuilderBuildDefaults_Good(t *testing.T) {
	projectDir := setupPythonTestProject(t)
	outputDir := t.TempDir()

	builder := NewPythonBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
	}

	artifacts, err := builder.Build(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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

func TestPython_PythonBuilderBuildIsDeterministic_Good(t *testing.T) {
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

		artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}

		content, err := ax.ReadFile(artifacts[0].Path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		return content
	}

	first := buildOnce(t.TempDir())
	second := buildOnce(t.TempDir())
	if !stdlibAssertEqual(first, second) {
		t.Fatalf("want %v, got %v", first, second)
	}

}

func TestPython_PythonBuilderInterface_Good(t *testing.T) {
	var _ build.Builder = (*PythonBuilder)(nil)
	var _ build.Builder = NewPythonBuilder()
}
