package builders

import (
	"archive/zip"
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

func setupFakePHPToolchain(t *testing.T, binDir string) {
	t.Helper()

	script := `#!/bin/sh
set -eu

log_file="${PHP_BUILD_LOG_FILE:-}"
if [ -n "$log_file" ]; then
	printf '%s\n' "$(basename "$0")" >> "$log_file"
	printf '%s\n' "$@" >> "$log_file"
	printf '%s\n' "GOOS=${GOOS:-}" >> "$log_file"
	printf '%s\n' "GOARCH=${GOARCH:-}" >> "$log_file"
	printf '%s\n' "OUTPUT_DIR=${OUTPUT_DIR:-}" >> "$log_file"
	printf '%s\n' "TARGET_DIR=${TARGET_DIR:-}" >> "$log_file"
fi

output_dir="${OUTPUT_DIR:-dist}"
platform_dir="${TARGET_DIR:-$output_dir/${GOOS:-}_${GOARCH:-}}"
mkdir -p "$platform_dir"

if [ "${1:-}" = "run-script" ] && [ "${2:-}" = "build" ]; then
	artifact="${platform_dir}/${NAME:-phpapp}"
	printf 'fake php artifact\n' > "$artifact"
	chmod +x "$artifact"
fi
`

	require.NoError(t, ax.WriteFile(ax.Join(binDir, "composer"), []byte(script), 0o755))
}

func setupPHPTestProject(t *testing.T, withBuildScript bool) string {
	t.Helper()

	dir := t.TempDir()

	composerJSON := `{"name":"test/php-app"}`
	if withBuildScript {
		composerJSON = `{"name":"test/php-app","scripts":{"build":"php build.php"}}`
	}

	require.NoError(t, ax.WriteFile(ax.Join(dir, "composer.json"), []byte(composerJSON), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(dir, "index.php"), []byte("<?php echo 'hello';"), 0o644))
	if withBuildScript {
		require.NoError(t, ax.WriteFile(ax.Join(dir, "build.php"), []byte("<?php echo 'build';"), 0o644))
	}

	return dir
}

func TestPHP_PHPBuilderName_Good(t *testing.T) {
	builder := NewPHPBuilder()
	assert.Equal(t, "php", builder.Name())
}

func TestPHP_PHPBuilderDetect_Good(t *testing.T) {
	fs := io.Local

	t.Run("detects composer.json projects", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "composer.json"), []byte("{}"), 0o644))

		builder := NewPHPBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		builder := NewPHPBuilder()
		detected, err := builder.Detect(fs, t.TempDir())
		assert.NoError(t, err)
		assert.False(t, detected)
	})
}

func TestPHP_PHPBuilderBuild_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakePHPToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectDir := setupPHPTestProject(t, true)
	outputDir := t.TempDir()
	logDir := t.TempDir()
	logPath := ax.Join(logDir, "php.log")
	t.Setenv("PHP_BUILD_LOG_FILE", logPath)

	builder := NewPHPBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "testapp",
		Version:    "v1.2.3",
	}

	targets := []build.Target{{OS: "linux", Arch: "amd64"}}

	artifacts, err := builder.Build(context.Background(), cfg, targets)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	assert.FileExists(t, artifacts[0].Path)
	assert.Equal(t, "linux", artifacts[0].OS)
	assert.Equal(t, "amd64", artifacts[0].Arch)

	content, err := ax.ReadFile(logPath)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	require.GreaterOrEqual(t, len(lines), 6)
	assert.Equal(t, "composer", lines[0])
	assert.Equal(t, "install", lines[1])
	assert.Contains(t, lines, "GOOS=linux")
	assert.Contains(t, lines, "GOARCH=amd64")
	assert.Contains(t, lines, "OUTPUT_DIR="+outputDir)
	assert.Contains(t, lines, "TARGET_DIR="+ax.Join(outputDir, "linux_amd64"))
}

func TestPHP_PHPBuilderBuildFallbackBundle_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakePHPToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectDir := setupPHPTestProject(t, false)
	outputDir := t.TempDir()

	builder := NewPHPBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "testapp",
	}

	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	assert.FileExists(t, artifacts[0].Path)
	assert.Equal(t, ".zip", ax.Ext(artifacts[0].Path))

	reader, err := zip.OpenReader(artifacts[0].Path)
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	var foundComposer bool
	for _, file := range reader.File {
		if file.Name == "composer.json" {
			foundComposer = true
			break
		}
	}
	assert.True(t, foundComposer)
}

func TestPHP_PHPBuilderBuildDefaults_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakePHPToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectDir := setupPHPTestProject(t, false)
	outputDir := t.TempDir()

	builder := NewPHPBuilder()
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

func TestPHP_PHPBuilderInterface_Good(t *testing.T) {
	var _ build.Builder = (*PHPBuilder)(nil)
	var _ build.Builder = NewPHPBuilder()
}
