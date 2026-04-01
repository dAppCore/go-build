package builders

import (
	"context"
	"os"
	"runtime"
	"testing"

	"dappco.re/go/core/build/internal/ax"

	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskfile_TaskfileBuilderName_Good(t *testing.T) {
	builder := NewTaskfileBuilder()
	assert.Equal(t, "taskfile", builder.Name())
}

func TestTaskfile_TaskfileBuilderDetect_Good(t *testing.T) {
	fs := io.Local

	t.Run("detects Taskfile.yml", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "Taskfile.yml"), []byte("version: '3'\n"), 0644)
		require.NoError(t, err)

		builder := NewTaskfileBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("detects Taskfile.yaml", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "Taskfile.yaml"), []byte("version: '3'\n"), 0644)
		require.NoError(t, err)

		builder := NewTaskfileBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("detects Taskfile (no extension)", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "Taskfile"), []byte("version: '3'\n"), 0644)
		require.NoError(t, err)

		builder := NewTaskfileBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("detects lowercase taskfile.yml", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "taskfile.yml"), []byte("version: '3'\n"), 0644)
		require.NoError(t, err)

		builder := NewTaskfileBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("detects lowercase taskfile.yaml", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "taskfile.yaml"), []byte("version: '3'\n"), 0644)
		require.NoError(t, err)

		builder := NewTaskfileBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		dir := t.TempDir()

		builder := NewTaskfileBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})

	t.Run("returns false for non-Taskfile project", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "Makefile"), []byte("all:\n\techo hello\n"), 0644)
		require.NoError(t, err)

		builder := NewTaskfileBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})

	t.Run("does not match Taskfile in subdirectory", func(t *testing.T) {
		dir := t.TempDir()
		subDir := ax.Join(dir, "subdir")
		require.NoError(t, ax.MkdirAll(subDir, 0755))
		err := ax.WriteFile(ax.Join(subDir, "Taskfile.yml"), []byte("version: '3'\n"), 0644)
		require.NoError(t, err)

		builder := NewTaskfileBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})
}

func TestTaskfile_TaskfileBuilderFindArtifacts_Good(t *testing.T) {
	fs := io.Local
	builder := NewTaskfileBuilder()

	t.Run("finds files in output directory", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "myapp"), []byte("binary"), 0755))
		require.NoError(t, ax.WriteFile(ax.Join(dir, "myapp.tar.gz"), []byte("archive"), 0644))

		artifacts := builder.findArtifacts(fs, dir)
		assert.Len(t, artifacts, 2)
	})

	t.Run("skips hidden files", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "myapp"), []byte("binary"), 0755))
		require.NoError(t, ax.WriteFile(ax.Join(dir, ".hidden"), []byte("hidden"), 0644))

		artifacts := builder.findArtifacts(fs, dir)
		assert.Len(t, artifacts, 1)
		assert.Contains(t, artifacts[0].Path, "myapp")
	})

	t.Run("skips CHECKSUMS.txt", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "myapp"), []byte("binary"), 0755))
		require.NoError(t, ax.WriteFile(ax.Join(dir, "CHECKSUMS.txt"), []byte("sha256"), 0644))

		artifacts := builder.findArtifacts(fs, dir)
		assert.Len(t, artifacts, 1)
		assert.Contains(t, artifacts[0].Path, "myapp")
	})

	t.Run("skips directories", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "myapp"), []byte("binary"), 0755))
		require.NoError(t, ax.MkdirAll(ax.Join(dir, "subdir"), 0755))

		artifacts := builder.findArtifacts(fs, dir)
		assert.Len(t, artifacts, 1)
	})

	t.Run("returns empty for empty directory", func(t *testing.T) {
		dir := t.TempDir()

		artifacts := builder.findArtifacts(fs, dir)
		assert.Empty(t, artifacts)
	})

	t.Run("returns empty for nonexistent directory", func(t *testing.T) {
		artifacts := builder.findArtifacts(fs, "/nonexistent/path")
		assert.Empty(t, artifacts)
	})
}

func TestTaskfile_TaskfileBuilderFindArtifactsForTarget_Good(t *testing.T) {
	fs := io.Local
	builder := NewTaskfileBuilder()

	t.Run("finds artifacts in platform subdirectory", func(t *testing.T) {
		dir := t.TempDir()
		platformDir := ax.Join(dir, "linux_amd64")
		require.NoError(t, ax.MkdirAll(platformDir, 0755))
		require.NoError(t, ax.WriteFile(ax.Join(platformDir, "myapp"), []byte("binary"), 0755))

		target := build.Target{OS: "linux", Arch: "amd64"}
		artifacts := builder.findArtifactsForTarget(fs, dir, target)
		assert.Len(t, artifacts, 1)
		assert.Equal(t, "linux", artifacts[0].OS)
		assert.Equal(t, "amd64", artifacts[0].Arch)
	})

	t.Run("finds artifacts by name pattern in root", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "myapp-linux-amd64"), []byte("binary"), 0755))

		target := build.Target{OS: "linux", Arch: "amd64"}
		artifacts := builder.findArtifactsForTarget(fs, dir, target)
		assert.NotEmpty(t, artifacts)
	})

	t.Run("returns empty when no matching artifacts", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "myapp"), []byte("binary"), 0755))

		target := build.Target{OS: "linux", Arch: "arm64"}
		artifacts := builder.findArtifactsForTarget(fs, dir, target)
		assert.Empty(t, artifacts)
	})

	t.Run("handles .app bundles on darwin", func(t *testing.T) {
		dir := t.TempDir()
		platformDir := ax.Join(dir, "darwin_arm64")
		appDir := ax.Join(platformDir, "MyApp.app")
		require.NoError(t, ax.MkdirAll(appDir, 0755))

		target := build.Target{OS: "darwin", Arch: "arm64"}
		artifacts := builder.findArtifactsForTarget(fs, dir, target)
		assert.Len(t, artifacts, 1)
		assert.Contains(t, artifacts[0].Path, "MyApp.app")
	})
}

func TestTaskfile_TaskfileBuilderMatchPattern_Good(t *testing.T) {
	builder := NewTaskfileBuilder()

	t.Run("matches simple glob", func(t *testing.T) {
		assert.True(t, builder.matchPattern("myapp-linux-amd64", "*-linux-amd64"))
	})

	t.Run("does not match different pattern", func(t *testing.T) {
		assert.False(t, builder.matchPattern("myapp-linux-amd64", "*-darwin-arm64"))
	})

	t.Run("matches wildcard", func(t *testing.T) {
		assert.True(t, builder.matchPattern("test_linux_arm64.bin", "*_linux_arm64*"))
	})
}

func TestTaskfile_TaskfileBuilderInterface_Good(t *testing.T) {
	// Verify TaskfileBuilder implements Builder interface
	var _ build.Builder = (*TaskfileBuilder)(nil)
	var _ build.Builder = NewTaskfileBuilder()
}

func TestTaskfile_TaskfileBuilderResolveTaskCli_Good(t *testing.T) {
	builder := NewTaskfileBuilder()
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "task")
	require.NoError(t, ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755))
	t.Setenv("PATH", "")

	command, err := builder.resolveTaskCli(fallbackPath)
	require.NoError(t, err)
	assert.Equal(t, fallbackPath, command)
}

func TestTaskfile_TaskfileBuilderResolveTaskCli_Bad(t *testing.T) {
	builder := NewTaskfileBuilder()
	t.Setenv("PATH", "")

	_, err := builder.resolveTaskCli(ax.Join(t.TempDir(), "missing-task"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task CLI not found")
}

func TestTaskfile_TaskfileBuilderRunTask_Good(t *testing.T) {
	binDir := t.TempDir()
	taskPath := ax.Join(binDir, "task")
	logPath := ax.Join(t.TempDir(), "task.env")

	script := `#!/bin/sh
set -eu

env | sort > "${TASK_BUILD_LOG_FILE}"
`
	require.NoError(t, ax.WriteFile(taskPath, []byte(script), 0o755))

	t.Setenv("TASK_BUILD_LOG_FILE", logPath)

	builder := NewTaskfileBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: t.TempDir(),
		OutputDir:  "/tmp/out",
		Name:       "sample",
		Version:    "v1.2.3",
		Env:        []string{"FOO=bar"},
	}

	require.NoError(t, builder.runTask(context.Background(), cfg, taskPath, "linux", "amd64"))

	content, err := ax.ReadFile(logPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), "FOO=bar")
	assert.Contains(t, string(content), "GOOS=linux")
	assert.Contains(t, string(content), "GOARCH=amd64")
	assert.Contains(t, string(content), "OUTPUT_DIR=/tmp/out")
	assert.Contains(t, string(content), "NAME=sample")
	assert.Contains(t, string(content), "VERSION=v1.2.3")
}

func TestTaskfile_TaskfileBuilderBuild_Good(t *testing.T) {
	projectDir := t.TempDir()
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "Taskfile.yml"), []byte("version: '3'\n"), 0o644))

	binDir := t.TempDir()
	taskPath := ax.Join(binDir, "task")
	logPath := ax.Join(t.TempDir(), "task.build.env")

	script := `#!/bin/sh
set -eu

mkdir -p "${OUTPUT_DIR}/${GOOS}_${GOARCH}"
printf '%s\n' "${NAME:-taskfile}" > "${OUTPUT_DIR}/${GOOS}_${GOARCH}/${NAME:-taskfile}"
env | sort > "${TASK_BUILD_LOG_FILE}"
`
	require.NoError(t, ax.WriteFile(taskPath, []byte(script), 0o755))

	t.Setenv("TASK_BUILD_LOG_FILE", logPath)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	builder := NewTaskfileBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		Name:       "sample",
		Version:    "v1.2.3",
		Env:        []string{"FOO=bar"},
	}

	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	assert.Equal(t, ax.Join(projectDir, "dist", "linux_amd64", "sample"), artifacts[0].Path)

	content, err := ax.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "FOO=bar")
	assert.Contains(t, string(content), "OUTPUT_DIR="+ax.Join(projectDir, "dist"))
	assert.Contains(t, string(content), "GOOS=linux")
	assert.Contains(t, string(content), "GOARCH=amd64")
}

func TestTaskfile_TaskfileBuilderBuild_DefaultTarget_Good(t *testing.T) {
	projectDir := t.TempDir()
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "Taskfile.yml"), []byte("version: '3'\n"), 0o644))

	binDir := t.TempDir()
	taskPath := ax.Join(binDir, "task")
	logPath := ax.Join(t.TempDir(), "task.default.env")

	script := `#!/bin/sh
set -eu

mkdir -p "${OUTPUT_DIR}/${GOOS}_${GOARCH}"
printf '%s\n' "${GOOS}/${GOARCH}" > "${OUTPUT_DIR}/${GOOS}_${GOARCH}/artifact"
env | sort > "${TASK_BUILD_LOG_FILE}"
`
	require.NoError(t, ax.WriteFile(taskPath, []byte(script), 0o755))

	t.Setenv("TASK_BUILD_LOG_FILE", logPath)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	builder := NewTaskfileBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		Name:       "sample",
		Version:    "v1.2.3",
		Env:        []string{"FOO=bar"},
	}

	artifacts, err := builder.Build(context.Background(), cfg, nil)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	assert.Equal(t, ax.Join(projectDir, "dist", runtime.GOOS+"_"+runtime.GOARCH, "artifact"), artifacts[0].Path)
	assert.Equal(t, runtime.GOOS, artifacts[0].OS)
	assert.Equal(t, runtime.GOARCH, artifacts[0].Arch)

	content, err := ax.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "FOO=bar")
	assert.Contains(t, string(content), "OUTPUT_DIR="+ax.Join(projectDir, "dist"))
	assert.Contains(t, string(content), "GOOS="+runtime.GOOS)
	assert.Contains(t, string(content), "GOARCH="+runtime.GOARCH)
}
