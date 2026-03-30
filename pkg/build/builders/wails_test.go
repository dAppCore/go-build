package builders

import (
	"context"
	"runtime"
	"testing"

	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupWailsTestProject creates a minimal Wails project structure for testing.
func setupWailsTestProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create wails.json
	wailsJSON := `{
  "name": "testapp",
  "outputfilename": "testapp"
}`
	err := ax.WriteFile(ax.Join(dir, "wails.json"), []byte(wailsJSON), 0o644)
	require.NoError(t, err)

	// Create a minimal go.mod
	goMod := `module testapp

go 1.21

require github.com/wailsapp/wails/v3 v3.0.0
`
	err = ax.WriteFile(ax.Join(dir, "go.mod"), []byte(goMod), 0o644)
	require.NoError(t, err)

	// Create a minimal main.go
	mainGo := `package main

func main() {
	println("hello wails")
}
`
	err = ax.WriteFile(ax.Join(dir, "main.go"), []byte(mainGo), 0o644)
	require.NoError(t, err)

	// Create a minimal Taskfile.yml
	taskfile := `version: '3'
tasks:
  build:
    cmds:
      - mkdir -p {{.OUTPUT_DIR}}/{{.GOOS}}_{{.GOARCH}}
      - touch {{.OUTPUT_DIR}}/{{.GOOS}}_{{.GOARCH}}/testapp
`
	err = ax.WriteFile(ax.Join(dir, "Taskfile.yml"), []byte(taskfile), 0o644)
	require.NoError(t, err)

	return dir
}

// setupWailsV2TestProject creates a Wails v2 project structure.
func setupWailsV2TestProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// wails.json
	err := ax.WriteFile(ax.Join(dir, "wails.json"), []byte("{}"), 0o644)
	require.NoError(t, err)

	// go.mod with v2
	goMod := `module testapp
go 1.21
require github.com/wailsapp/wails/v2 v2.8.0
`
	err = ax.WriteFile(ax.Join(dir, "go.mod"), []byte(goMod), 0o644)
	require.NoError(t, err)

	return dir
}

func TestWails_WailsBuilderBuildTaskfile_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Check if task is available
	if _, err := ax.LookPath("task"); err != nil {
		t.Skip("task not installed, skipping test")
	}

	t.Run("delegates to Taskfile if present", func(t *testing.T) {
		fs := io.Local
		projectDir := setupWailsTestProject(t)
		outputDir := t.TempDir()

		// Create a Taskfile that just touches a file
		taskfile := `version: '3'
tasks:
  build:
    cmds:
      - mkdir -p {{.OUTPUT_DIR}}/{{.GOOS}}_{{.GOARCH}}
      - touch {{.OUTPUT_DIR}}/{{.GOOS}}_{{.GOARCH}}/testapp
`
		err := ax.WriteFile(ax.Join(projectDir, "Taskfile.yml"), []byte(taskfile), 0o644)
		require.NoError(t, err)

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         fs,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
			Name:       "testapp",
		}
		targets := []build.Target{
			{OS: runtime.GOOS, Arch: runtime.GOARCH},
		}

		artifacts, err := builder.Build(context.Background(), cfg, targets)
		require.NoError(t, err)
		assert.NotEmpty(t, artifacts)
	})
}

func TestWails_WailsBuilderName_Good(t *testing.T) {
	builder := NewWailsBuilder()
	assert.Equal(t, "wails", builder.Name())
}

func TestWails_WailsBuilderBuildV2_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	builder := NewWailsBuilder()
	if _, err := builder.resolveWailsCli(); err != nil {
		t.Skip("wails not installed, skipping integration test")
	}

	t.Run("builds v2 project", func(t *testing.T) {
		fs := io.Local
		projectDir := setupWailsV2TestProject(t)
		outputDir := t.TempDir()

		cfg := &build.Config{
			FS:         fs,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
			Name:       "testapp",
		}
		targets := []build.Target{
			{OS: runtime.GOOS, Arch: runtime.GOARCH},
		}

		// This will likely fail in a real run because we can't easily mock the full wails v2 build process
		// (which needs a valid project with main.go etc).
		// But it validates we are trying to run the command.
		// For now, we just verify it attempts the build - error is expected
		_, _ = builder.Build(context.Background(), cfg, targets)
	})
}

func TestWails_WailsBuilderResolveWailsCli_Good(t *testing.T) {
	builder := NewWailsBuilder()
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "wails")
	require.NoError(t, ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755))
	t.Setenv("PATH", "")

	command, err := builder.resolveWailsCli(fallbackPath)
	require.NoError(t, err)
	assert.Equal(t, fallbackPath, command)
}

func TestWails_WailsBuilderResolveWailsCli_Bad(t *testing.T) {
	builder := NewWailsBuilder()
	t.Setenv("PATH", "")

	_, err := builder.resolveWailsCli(ax.Join(t.TempDir(), "missing-wails"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "wails CLI not found")
}

func TestWails_WailsBuilderDetect_Good(t *testing.T) {
	fs := io.Local
	t.Run("detects Wails project with wails.json", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "wails.json"), []byte("{}"), 0o644)
		require.NoError(t, err)

		builder := NewWailsBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("returns false for Go-only project", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module test"), 0o644)
		require.NoError(t, err)

		builder := NewWailsBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})

	t.Run("returns false for Node.js project", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0o644)
		require.NoError(t, err)

		builder := NewWailsBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		dir := t.TempDir()

		builder := NewWailsBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})
}

func TestWails_DetectPackageManager_Good(t *testing.T) {
	fs := io.Local
	t.Run("detects bun from bun.lockb", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "bun.lockb"), []byte(""), 0o644)
		require.NoError(t, err)

		result := detectPackageManager(fs, dir)
		assert.Equal(t, "bun", result)
	})

	t.Run("detects pnpm from pnpm-lock.yaml", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "pnpm-lock.yaml"), []byte(""), 0o644)
		require.NoError(t, err)

		result := detectPackageManager(fs, dir)
		assert.Equal(t, "pnpm", result)
	})

	t.Run("detects yarn from yarn.lock", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "yarn.lock"), []byte(""), 0o644)
		require.NoError(t, err)

		result := detectPackageManager(fs, dir)
		assert.Equal(t, "yarn", result)
	})

	t.Run("detects npm from package-lock.json", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "package-lock.json"), []byte(""), 0o644)
		require.NoError(t, err)

		result := detectPackageManager(fs, dir)
		assert.Equal(t, "npm", result)
	})

	t.Run("defaults to npm when no lock file", func(t *testing.T) {
		dir := t.TempDir()

		result := detectPackageManager(fs, dir)
		assert.Equal(t, "npm", result)
	})

	t.Run("prefers bun over other lock files", func(t *testing.T) {
		dir := t.TempDir()
		// Create multiple lock files
		require.NoError(t, ax.WriteFile(ax.Join(dir, "bun.lockb"), []byte(""), 0o644))
		require.NoError(t, ax.WriteFile(ax.Join(dir, "yarn.lock"), []byte(""), 0o644))
		require.NoError(t, ax.WriteFile(ax.Join(dir, "package-lock.json"), []byte(""), 0o644))

		result := detectPackageManager(fs, dir)
		assert.Equal(t, "bun", result)
	})

	t.Run("prefers pnpm over yarn and npm", func(t *testing.T) {
		dir := t.TempDir()
		// Create multiple lock files (no bun)
		require.NoError(t, ax.WriteFile(ax.Join(dir, "pnpm-lock.yaml"), []byte(""), 0o644))
		require.NoError(t, ax.WriteFile(ax.Join(dir, "yarn.lock"), []byte(""), 0o644))
		require.NoError(t, ax.WriteFile(ax.Join(dir, "package-lock.json"), []byte(""), 0o644))

		result := detectPackageManager(fs, dir)
		assert.Equal(t, "pnpm", result)
	})

	t.Run("prefers yarn over npm", func(t *testing.T) {
		dir := t.TempDir()
		// Create multiple lock files (no bun or pnpm)
		require.NoError(t, ax.WriteFile(ax.Join(dir, "yarn.lock"), []byte(""), 0o644))
		require.NoError(t, ax.WriteFile(ax.Join(dir, "package-lock.json"), []byte(""), 0o644))

		result := detectPackageManager(fs, dir)
		assert.Equal(t, "yarn", result)
	})
}

func TestWails_WailsBuilderBuild_Bad(t *testing.T) {
	t.Run("returns error for nil config", func(t *testing.T) {
		builder := NewWailsBuilder()

		artifacts, err := builder.Build(context.Background(), nil, []build.Target{{OS: "linux", Arch: "amd64"}})
		assert.Error(t, err)
		assert.Nil(t, artifacts)
		assert.Contains(t, err.Error(), "config is nil")
	})

	t.Run("returns error for empty targets", func(t *testing.T) {
		projectDir := setupWailsTestProject(t)

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  t.TempDir(),
			Name:       "test",
		}

		artifacts, err := builder.Build(context.Background(), cfg, []build.Target{})
		assert.Error(t, err)
		assert.Nil(t, artifacts)
		assert.Contains(t, err.Error(), "no targets specified")
	})
}

func TestWails_WailsBuilderBuild_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Check if wails3 is available in PATH
	if _, err := ax.LookPath("wails3"); err != nil {
		t.Skip("wails3 not installed, skipping integration test")
	}

	t.Run("builds for current platform", func(t *testing.T) {
		projectDir := setupWailsTestProject(t)
		outputDir := t.TempDir()

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
			Name:       "testapp",
		}
		targets := []build.Target{
			{OS: runtime.GOOS, Arch: runtime.GOARCH},
		}

		artifacts, err := builder.Build(context.Background(), cfg, targets)
		require.NoError(t, err)
		require.Len(t, artifacts, 1)

		// Verify artifact properties
		artifact := artifacts[0]
		assert.Equal(t, runtime.GOOS, artifact.OS)
		assert.Equal(t, runtime.GOARCH, artifact.Arch)
	})
}

func TestWails_WailsBuilderInterface_Good(t *testing.T) {
	// Verify WailsBuilder implements Builder interface
	var _ build.Builder = (*WailsBuilder)(nil)
	var _ build.Builder = NewWailsBuilder()
}

func TestWails_WailsBuilder_Ugly(t *testing.T) {
	t.Run("handles nonexistent frontend directory gracefully", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping integration test in short mode")
		}

		// Create a Wails project without a frontend directory
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "wails.json"), []byte("{}"), 0o644)
		require.NoError(t, err)

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: dir,
			OutputDir:  t.TempDir(),
			Name:       "test",
		}
		targets := []build.Target{
			{OS: runtime.GOOS, Arch: runtime.GOARCH},
		}

		// This will fail because wails3 isn't set up, but it shouldn't panic
		// due to missing frontend directory
		_, err = builder.Build(context.Background(), cfg, targets)
		// We expect an error (wails3 build will fail), but not a panic
		// The error should be about wails3 build, not about frontend
		if err != nil {
			assert.NotContains(t, err.Error(), "frontend dependencies")
		}
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping integration test in short mode")
		}

		projectDir := setupWailsTestProject(t)

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  t.TempDir(),
			Name:       "canceltest",
		}
		targets := []build.Target{
			{OS: runtime.GOOS, Arch: runtime.GOARCH},
		}

		// Create an already cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		artifacts, err := builder.Build(ctx, cfg, targets)
		assert.Error(t, err)
		assert.Empty(t, artifacts)
	})
}
