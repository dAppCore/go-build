package builders

import (
	"archive/zip"
	"context"
	"runtime"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPythonTestProject(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	require.NoError(t, ax.WriteFile(ax.Join(dir, "pyproject.toml"), []byte("[build-system]\nrequires = []\n"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(dir, "app.py"), []byte("print('hello')\n"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(dir, "README.md"), []byte("demo"), 0o644))

	return dir
}

func TestPython_PythonBuilderName_Good(t *testing.T) {
	builder := NewPythonBuilder()
	assert.Equal(t, "python", builder.Name())
}

func TestPython_PythonBuilderDetect_Good(t *testing.T) {
	fs := io.Local

	t.Run("detects pyproject.toml projects", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "pyproject.toml"), []byte("{}"), 0o644))

		builder := NewPythonBuilder()
		detected, err := builder.Detect(fs, dir)
		require.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("detects requirements.txt projects", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "requirements.txt"), []byte("requests"), 0o644))

		builder := NewPythonBuilder()
		detected, err := builder.Detect(fs, dir)
		require.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		builder := NewPythonBuilder()
		detected, err := builder.Detect(fs, t.TempDir())
		require.NoError(t, err)
		assert.False(t, detected)
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
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	artifact := artifacts[0]
	assert.Equal(t, "linux", artifact.OS)
	assert.Equal(t, "amd64", artifact.Arch)
	assert.FileExists(t, artifact.Path)

	reader, err := zip.OpenReader(artifact.Path)
	require.NoError(t, err)
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

	assert.True(t, foundPyProject)
	assert.True(t, foundApp)
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
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	assert.Equal(t, runtime.GOOS, artifacts[0].OS)
	assert.Equal(t, runtime.GOARCH, artifacts[0].Arch)
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
		require.NoError(t, err)
		require.Len(t, artifacts, 1)

		content, err := ax.ReadFile(artifacts[0].Path)
		require.NoError(t, err)
		return content
	}

	first := buildOnce(t.TempDir())
	second := buildOnce(t.TempDir())

	assert.Equal(t, first, second)
}

func TestPython_PythonBuilderInterface_Good(t *testing.T) {
	var _ build.Builder = (*PythonBuilder)(nil)
	var _ build.Builder = NewPythonBuilder()
}
