package builders

import (
	"context"
	"os"
	"runtime"
	"strings"
	"testing"

	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupFakeNodeToolchain(t *testing.T, binDir string) {
	t.Helper()

	script := `#!/bin/sh
set -eu

log_file="${NODE_BUILD_LOG_FILE:-}"
if [ -n "$log_file" ]; then
	printf '%s\n' "$(basename "$0")" >> "$log_file"
	printf '%s\n' "$@" >> "$log_file"
	printf '%s\n' "GOOS=${GOOS:-}" >> "$log_file"
	printf '%s\n' "GOARCH=${GOARCH:-}" >> "$log_file"
	printf '%s\n' "OUTPUT_DIR=${OUTPUT_DIR:-}" >> "$log_file"
	printf '%s\n' "TARGET_DIR=${TARGET_DIR:-}" >> "$log_file"
	env | sort >> "$log_file"
fi

output_dir="${OUTPUT_DIR:-dist}"
platform_dir="${TARGET_DIR:-$output_dir/${GOOS:-}_${GOARCH:-}}"
mkdir -p "$platform_dir"

name="${NAME:-nodeapp}"
printf 'fake node artifact\n' > "$platform_dir/$name"
chmod +x "$platform_dir/$name"
`

	for _, name := range []string{"npm", "pnpm", "yarn", "bun", "deno"} {
		require.NoError(t, ax.WriteFile(ax.Join(binDir, name), []byte(script), 0o755))
	}
}

func setupFakeNodeCommand(t *testing.T, binDir, name string) {
	t.Helper()

	script := `#!/bin/sh
set -eu

log_file="${NODE_BUILD_LOG_FILE:-}"
if [ -n "$log_file" ]; then
	printf '%s\n' "$(basename "$0")" >> "$log_file"
	printf '%s\n' "$@" >> "$log_file"
fi

output_dir="${OUTPUT_DIR:-dist}"
platform_dir="${TARGET_DIR:-$output_dir/${GOOS:-}_${GOARCH:-}}"
mkdir -p "$platform_dir"
printf 'fake node artifact\n' > "$platform_dir/${NAME:-nodeapp}"
chmod +x "$platform_dir/${NAME:-nodeapp}"
`

	require.NoError(t, ax.WriteFile(ax.Join(binDir, name), []byte(script), 0o755))
}

func setupNodeTestProject(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	require.NoError(t, ax.WriteFile(ax.Join(dir, "package.json"), []byte(`{"name":"testapp","scripts":{"build":"node build.js"}}`), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(dir, "build.js"), []byte(`console.log("build")`), 0o644))

	return dir
}

func TestNode_NodeBuilderName_Good(t *testing.T) {
	builder := NewNodeBuilder()
	assert.Equal(t, "node", builder.Name())
}

func TestNode_NodeBuilderDetect_Good(t *testing.T) {
	fs := io.Local

	t.Run("detects package.json projects", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0o644))

		builder := NewNodeBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		builder := NewNodeBuilder()
		detected, err := builder.Detect(fs, t.TempDir())
		assert.NoError(t, err)
		assert.False(t, detected)
	})

	t.Run("detects nested package.json projects", func(t *testing.T) {
		dir := t.TempDir()
		nested := ax.Join(dir, "apps", "web")
		require.NoError(t, ax.MkdirAll(nested, 0o755))
		require.NoError(t, ax.WriteFile(ax.Join(nested, "package.json"), []byte("{}"), 0o644))

		builder := NewNodeBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("detects root deno projects", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "deno.json"), []byte(`{"tasks":{"build":"deno eval ''"}}`), 0o644))

		builder := NewNodeBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})
}

func TestNode_NodeBuilderBuild_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeNodeToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectDir := setupNodeTestProject(t)
	outputDir := t.TempDir()
	logDir := t.TempDir()
	logPath := ax.Join(logDir, "node.log")
	t.Setenv("NODE_BUILD_LOG_FILE", logPath)

	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "pnpm-lock.yaml"), []byte("lockfile"), 0o644))

	builder := NewNodeBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "testapp",
		Version:    "v1.2.3",
		Env:        []string{"FOO=bar"},
	}

	targets := []build.Target{
		{OS: "linux", Arch: "amd64"},
	}

	artifacts, err := builder.Build(context.Background(), cfg, targets)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	assert.FileExists(t, artifacts[0].Path)
	assert.Equal(t, "linux", artifacts[0].OS)
	assert.Equal(t, "amd64", artifacts[0].Arch)

	content, err := ax.ReadFile(logPath)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	require.GreaterOrEqual(t, len(lines), 5)
	assert.Equal(t, "pnpm", lines[0])
	assert.Equal(t, "run", lines[1])
	assert.Equal(t, "build", lines[2])
	assert.Equal(t, "GOOS=linux", lines[3])
	assert.Equal(t, "GOARCH=amd64", lines[4])
	assert.Contains(t, lines, "OUTPUT_DIR="+outputDir)
	assert.Contains(t, lines, "TARGET_DIR="+ax.Join(outputDir, "linux_amd64"))
	assert.Contains(t, string(content), "FOO=bar")
}

func TestNode_NodeBuilderBuild_Good_Deno(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeNodeToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectDir := t.TempDir()
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "deno.json"), []byte(`{"tasks":{"build":"deno eval ''"}}`), 0o644))

	outputDir := t.TempDir()
	logPath := ax.Join(t.TempDir(), "deno.log")
	t.Setenv("NODE_BUILD_LOG_FILE", logPath)

	builder := NewNodeBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "denoapp",
		Version:    "v1.2.3",
	}

	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	assert.FileExists(t, artifacts[0].Path)

	content, err := ax.ReadFile(logPath)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	require.GreaterOrEqual(t, len(lines), 3)
	assert.Equal(t, "deno", lines[0])
	assert.Equal(t, "task", lines[1])
	assert.Equal(t, "build", lines[2])
}

func TestNode_NodeBuilderBuild_Good_DenoOverrideFromConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeNodeToolchain(t, binDir)
	setupFakeNodeCommand(t, binDir, "deno-build")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectDir := t.TempDir()
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "deno.json"), []byte(`{"tasks":{"build":"deno eval ''"}}`), 0o644))

	outputDir := t.TempDir()
	logPath := ax.Join(t.TempDir(), "deno-override.log")
	t.Setenv("NODE_BUILD_LOG_FILE", logPath)

	builder := NewNodeBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "denoapp",
		DenoBuild:  "deno-build --target release",
	}

	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	content, err := ax.ReadFile(logPath)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	require.GreaterOrEqual(t, len(lines), 3)
	assert.Equal(t, "deno-build", lines[0])
	assert.Equal(t, "--target", lines[1])
	assert.Equal(t, "release", lines[2])
}

func TestNode_NodeBuilderBuild_Good_DenoOverrideFromEnvWins(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeNodeToolchain(t, binDir)
	setupFakeNodeCommand(t, binDir, "deno-build")
	setupFakeNodeCommand(t, binDir, "env-deno-build")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("DENO_BUILD", "env-deno-build --env")

	projectDir := t.TempDir()
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "deno.json"), []byte(`{"tasks":{"build":"deno eval ''"}}`), 0o644))

	outputDir := t.TempDir()
	logPath := ax.Join(t.TempDir(), "deno-env-override.log")
	t.Setenv("NODE_BUILD_LOG_FILE", logPath)

	builder := NewNodeBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "denoapp",
		DenoBuild:  "deno-build --config",
	}

	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	content, err := ax.ReadFile(logPath)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	require.GreaterOrEqual(t, len(lines), 2)
	assert.Equal(t, "env-deno-build", lines[0])
	assert.Equal(t, "--env", lines[1])
}

func TestNode_ResolvePackageManager_Good(t *testing.T) {
	fs := io.Local
	builder := NewNodeBuilder()

	t.Run("prefers packageManager declaration over lockfiles", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "package.json"), []byte(`{"packageManager":"pnpm@9.12.0"}`), 0o644))
		require.NoError(t, ax.WriteFile(ax.Join(dir, "bun.lockb"), []byte(""), 0o644))

		result, err := builder.resolvePackageManager(fs, dir)
		require.NoError(t, err)
		assert.Equal(t, "pnpm", result)
	})

	t.Run("normalises package manager version pins", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "package.json"), []byte(`{"packageManager":"bun@1.1.38"}`), 0o644))

		result, err := builder.resolvePackageManager(fs, dir)
		require.NoError(t, err)
		assert.Equal(t, "bun", result)
	})
}

func TestNode_NodeBuilderFindArtifactsForTarget_Good(t *testing.T) {
	fs := io.Local
	builder := NewNodeBuilder()

	t.Run("finds files in platform subdirectory", func(t *testing.T) {
		dir := t.TempDir()
		platformDir := ax.Join(dir, "linux_amd64")
		require.NoError(t, ax.MkdirAll(platformDir, 0o755))
		artifactPath := ax.Join(platformDir, "testapp")
		require.NoError(t, ax.WriteFile(artifactPath, []byte("binary"), 0o755))

		artifacts := builder.findArtifactsForTarget(fs, dir, build.Target{OS: "linux", Arch: "amd64"})
		require.Len(t, artifacts, 1)
		assert.Equal(t, artifactPath, artifacts[0].Path)
	})

	t.Run("finds darwin app bundles", func(t *testing.T) {
		dir := t.TempDir()
		platformDir := ax.Join(dir, "darwin_arm64")
		appDir := ax.Join(platformDir, "TestApp.app")
		require.NoError(t, ax.MkdirAll(appDir, 0o755))

		artifacts := builder.findArtifactsForTarget(fs, dir, build.Target{OS: "darwin", Arch: "arm64"})
		require.Len(t, artifacts, 1)
		assert.Equal(t, appDir, artifacts[0].Path)
	})

	t.Run("falls back to name patterns in root", func(t *testing.T) {
		dir := t.TempDir()
		artifactPath := ax.Join(dir, "testapp-linux-amd64")
		require.NoError(t, ax.WriteFile(artifactPath, []byte("binary"), 0o755))

		artifacts := builder.findArtifactsForTarget(fs, dir, build.Target{OS: "linux", Arch: "amd64"})
		require.NotEmpty(t, artifacts)
		assert.Equal(t, artifactPath, artifacts[0].Path)
	})
}

func TestNode_NodeBuilderInterface_Good(t *testing.T) {
	var _ build.Builder = (*NodeBuilder)(nil)
	var _ build.Builder = NewNodeBuilder()
}

func TestNode_NodeBuilderBuildDefaults_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeNodeToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectDir := setupNodeTestProject(t)
	outputDir := t.TempDir()

	builder := NewNodeBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Env:        []string{"FOO=bar"},
	}

	artifacts, err := builder.Build(context.Background(), cfg, nil)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	assert.Equal(t, runtime.GOOS, artifacts[0].OS)
	assert.Equal(t, runtime.GOARCH, artifacts[0].Arch)
}

func TestNode_NodeBuilderBuild_Good_NestedProject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeNodeToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectDir := t.TempDir()
	nestedDir := ax.Join(projectDir, "apps", "web")
	require.NoError(t, ax.MkdirAll(nestedDir, 0o755))
	require.NoError(t, ax.WriteFile(ax.Join(nestedDir, "package.json"), []byte(`{"name":"nested-app","scripts":{"build":"node build.js"}}`), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(nestedDir, "build.js"), []byte(`console.log("nested build")`), 0o644))

	outputDir := t.TempDir()
	logDir := t.TempDir()
	logPath := ax.Join(logDir, "node-nested.log")
	t.Setenv("NODE_BUILD_LOG_FILE", logPath)

	builder := NewNodeBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "nested-app",
		Version:    "v1.2.3",
	}

	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	assert.FileExists(t, artifacts[0].Path)

	content, err := ax.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "apps/web")
	assert.Contains(t, string(content), "GOOS=linux")
	assert.Contains(t, string(content), "GOARCH=amd64")
}
