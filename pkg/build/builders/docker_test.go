package builders

import (
	"context"
	"os"
	"runtime"
	"strings"
	"testing"

	"dappco.re/go/core/build/internal/ax"

	"dappco.re/go/core/build/pkg/build"
	coreio "dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupFakeDockerToolchain(t *testing.T, binDir string) {
	t.Helper()

	script := `#!/bin/sh
set -eu

	log_file="${DOCKER_BUILD_LOG_FILE:-}"
	if [ -n "$log_file" ]; then
		printf '%s\n' "$*" >> "$log_file"
		env | sort >> "$log_file"
	fi

	if [ "${1:-}" = "buildx" ] && [ "${2:-}" = "build" ]; then
	dest=""
	while [ $# -gt 0 ]; do
		if [ "$1" = "--output" ]; then
			shift
			dest="$(printf '%s' "$1" | sed -n 's#type=oci,dest=##p')"
		fi
		shift
	done
	if [ -n "$dest" ]; then
		mkdir -p "$(dirname "$dest")"
		printf 'oci archive\n' > "$dest"
	fi
fi
`

	require.NoError(t, ax.WriteFile(ax.Join(binDir, "docker"), []byte(script), 0o755))
}

func TestDocker_DockerBuilderName_Good(t *testing.T) {
	builder := NewDockerBuilder()
	assert.Equal(t, "docker", builder.Name())
}

func TestDocker_DockerBuilderDetect_Good(t *testing.T) {
	fs := coreio.Local

	t.Run("detects Dockerfile", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "Dockerfile"), []byte("FROM alpine\n"), 0644)
		require.NoError(t, err)

		builder := NewDockerBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		dir := t.TempDir()

		builder := NewDockerBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})

	t.Run("returns false for non-Docker project", func(t *testing.T) {
		dir := t.TempDir()
		// Create a Go project instead
		err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module test"), 0644)
		require.NoError(t, err)

		builder := NewDockerBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})

	t.Run("does not match docker-compose.yml", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "docker-compose.yml"), []byte("version: '3'\n"), 0644)
		require.NoError(t, err)

		builder := NewDockerBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})

	t.Run("does not match Dockerfile in subdirectory", func(t *testing.T) {
		dir := t.TempDir()
		subDir := ax.Join(dir, "subdir")
		require.NoError(t, ax.MkdirAll(subDir, 0755))
		err := ax.WriteFile(ax.Join(subDir, "Dockerfile"), []byte("FROM alpine\n"), 0644)
		require.NoError(t, err)

		builder := NewDockerBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})
}

func TestDocker_DockerBuilderInterface_Good(t *testing.T) {
	// Verify DockerBuilder implements Builder interface
	var _ build.Builder = (*DockerBuilder)(nil)
	var _ build.Builder = NewDockerBuilder()
}

func TestDocker_DockerBuilderResolveDockerCli_Good(t *testing.T) {
	builder := NewDockerBuilder()
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "docker")
	require.NoError(t, ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755))
	t.Setenv("PATH", "")

	command, err := builder.resolveDockerCli(fallbackPath)
	require.NoError(t, err)
	assert.Equal(t, fallbackPath, command)
}

func TestDocker_DockerBuilderResolveDockerCli_Bad(t *testing.T) {
	builder := NewDockerBuilder()
	t.Setenv("PATH", "")

	_, err := builder.resolveDockerCli(ax.Join(t.TempDir(), "missing-docker"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "docker CLI not found")
}

func TestDocker_DockerBuilderBuild_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeDockerToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectDir := t.TempDir()
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "Dockerfile"), []byte("FROM alpine:latest\n"), 0o644))

	outputDir := t.TempDir()
	logDir := t.TempDir()
	logPath := ax.Join(logDir, "docker.log")
	t.Setenv("DOCKER_BUILD_LOG_FILE", logPath)

	builder := NewDockerBuilder()
	cfg := &build.Config{
		FS:         coreio.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "sample-app",
		Image:      "owner/repo",
		Env:        []string{"FOO=bar"},
	}
	targets := []build.Target{
		{OS: "linux", Arch: "amd64"},
		{OS: "linux", Arch: "arm64"},
	}

	artifacts, err := builder.Build(context.Background(), cfg, targets)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	expectedPath := ax.Join(outputDir, "owner_repo.tar")
	assert.Equal(t, expectedPath, artifacts[0].Path)
	assert.Equal(t, "linux", artifacts[0].OS)
	assert.Equal(t, "amd64", artifacts[0].Arch)
	assert.FileExists(t, expectedPath)

	logContent, err := ax.ReadFile(logPath)
	require.NoError(t, err)

	log := string(logContent)
	assert.Equal(t, 1, strings.Count(log, "buildx build"))
	assert.Contains(t, log, "--platform")
	assert.Contains(t, log, "linux/amd64,linux/arm64")
	assert.Contains(t, log, "--output")
	assert.Contains(t, log, "type=oci,dest="+expectedPath)

	assert.Contains(t, log, "FOO=bar")

	artifacts, err = builder.Build(context.Background(), cfg, nil)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	assert.Equal(t, runtime.GOOS, artifacts[0].OS)
	assert.Equal(t, runtime.GOARCH, artifacts[0].Arch)
}

func TestDocker_DockerBuilderBuild_Load_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeDockerToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectDir := t.TempDir()
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "Dockerfile"), []byte("FROM alpine:latest\n"), 0o644))

	outputDir := t.TempDir()
	logDir := t.TempDir()
	logPath := ax.Join(logDir, "docker.log")
	t.Setenv("DOCKER_BUILD_LOG_FILE", logPath)

	builder := NewDockerBuilder()
	cfg := &build.Config{
		FS:         coreio.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Image:      "owner/repo",
		Load:       true,
		Env:        []string{"FOO=bar"},
	}
	targets := []build.Target{
		{OS: "linux", Arch: "amd64"},
	}

	artifacts, err := builder.Build(context.Background(), cfg, targets)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	assert.Equal(t, "ghcr.io/owner/repo:latest", artifacts[0].Path)
	assert.Equal(t, "linux", artifacts[0].OS)
	assert.Equal(t, "amd64", artifacts[0].Arch)
	assert.DirExists(t, outputDir)

	logContent, err := ax.ReadFile(logPath)
	require.NoError(t, err)

	log := string(logContent)
	assert.Contains(t, log, "buildx build")
	assert.Contains(t, log, "--load")
	assert.NotContains(t, log, "--output")
}
