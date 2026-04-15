package builders

import (
	"context"
	"os"
	"strings"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupFakeRustToolchain(t *testing.T, binDir string) {
	t.Helper()

	script := `#!/bin/sh
set -eu

log_file="${RUST_BUILD_LOG_FILE:-}"
if [ -n "$log_file" ]; then
	printf '%s\n' "$(basename "$0")" >> "$log_file"
	printf '%s\n' "$@" >> "$log_file"
	printf '%s\n' "CARGO_TARGET_DIR=${CARGO_TARGET_DIR:-}" >> "$log_file"
	printf '%s\n' "TARGET_OS=${TARGET_OS:-}" >> "$log_file"
	printf '%s\n' "TARGET_ARCH=${TARGET_ARCH:-}" >> "$log_file"
	env | sort >> "$log_file"
fi

target_triple=""
prev=""
for arg in "$@"; do
	if [ "$prev" = "--target" ]; then
		target_triple="$arg"
		prev=""
		continue
	fi
	if [ "$arg" = "--target" ]; then
		prev="--target"
	fi
done

target_dir="${CARGO_TARGET_DIR:-target}"
release_dir="$target_dir/$target_triple/release"
mkdir -p "$release_dir"

name="${NAME:-rustapp}"
artifact="$release_dir/$name"
case "$target_triple" in
	*-windows-*)
		artifact="$artifact.exe"
		;;
esac

printf 'fake rust artifact\n' > "$artifact"
chmod +x "$artifact" 2>/dev/null || true
`

	require.NoError(t, ax.WriteFile(ax.Join(binDir, "cargo"), []byte(script), 0o755))
}

func setupRustTestProject(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, ax.WriteFile(ax.Join(dir, "Cargo.toml"), []byte("[package]\nname = \"testapp\"\nversion = \"0.1.0\""), 0o644))
	require.NoError(t, ax.MkdirAll(ax.Join(dir, "src"), 0o755))
	require.NoError(t, ax.WriteFile(ax.Join(dir, "src", "main.rs"), []byte("fn main() {}"), 0o644))
	return dir
}

func TestRust_RustBuilderName_Good(t *testing.T) {
	builder := NewRustBuilder()
	assert.Equal(t, "rust", builder.Name())
}

func TestRust_RustBuilderDetect_Good(t *testing.T) {
	fs := io.Local

	t.Run("detects Cargo.toml projects", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "Cargo.toml"), []byte("{}"), 0o644))

		builder := NewRustBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		builder := NewRustBuilder()
		detected, err := builder.Detect(fs, t.TempDir())
		assert.NoError(t, err)
		assert.False(t, detected)
	})
}

func TestRust_RustBuilderBuild_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeRustToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectDir := setupRustTestProject(t)
	outputDir := t.TempDir()
	logDir := t.TempDir()
	logPath := ax.Join(logDir, "rust.log")
	t.Setenv("RUST_BUILD_LOG_FILE", logPath)

	builder := NewRustBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "testapp",
		Version:    "v1.2.3",
		Env:        []string{"FOO=bar"},
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
	require.GreaterOrEqual(t, len(lines), 5)
	assert.Equal(t, "cargo", lines[0])
	assert.Equal(t, "build", lines[1])
	assert.Equal(t, "--release", lines[2])
	assert.Equal(t, "--target", lines[3])
	assert.Equal(t, "x86_64-unknown-linux-gnu", lines[4])
	assert.Contains(t, lines, "CARGO_TARGET_DIR="+ax.Join(outputDir, "linux_amd64"))
	assert.Contains(t, string(content), "FOO=bar")
}

func TestRust_RustBuilderInterface_Good(t *testing.T) {
	var _ build.Builder = (*RustBuilder)(nil)
	var _ build.Builder = NewRustBuilder()
}
