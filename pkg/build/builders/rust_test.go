package builders

import (
	"context"
	"os"
	"strings"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/io"
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
	if err := ax.WriteFile(ax.Join(binDir, "cargo"), []byte(script), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}

func setupRustTestProject(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	if err := ax.WriteFile(ax.Join(dir, "Cargo.toml"), []byte("[package]\nname = \"testapp\"\nversion = \"0.1.0\""), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.MkdirAll(ax.Join(dir, "src"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(dir, "src", "main.rs"), []byte("fn main() {}"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return dir
}

func TestRust_RustBuilderName_Good(t *testing.T) {
	builder := NewRustBuilder()
	if !stdlibAssertEqual("rust", builder.Name()) {
		t.Fatalf("want %v, got %v", "rust", builder.Name())
	}

}

func TestRust_RustBuilderDetect_Good(t *testing.T) {
	fs := io.Local

	t.Run("detects Cargo.toml projects", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "Cargo.toml"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewRustBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		builder := NewRustBuilder()
		detected, err := builder.Detect(fs, t.TempDir())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detected {
			t.Fatal("expected false")
		}

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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}
	if _, err := os.Stat(artifacts[0].Path); err != nil {
		t.Fatalf("expected file to exist: %v", artifacts[0].Path)
	}
	if !stdlibAssertEqual("linux", artifacts[0].OS) {
		t.Fatalf("want %v, got %v", "linux", artifacts[0].OS)
	}
	if !stdlibAssertEqual("amd64", artifacts[0].Arch) {
		t.Fatalf("want %v, got %v", "amd64", artifacts[0].Arch)
	}

	content, err := ax.ReadFile(logPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) < 5 {
		t.Fatalf("expected %v to be greater than or equal to %v", len(lines), 5)
	}
	if !stdlibAssertEqual("cargo", lines[0]) {
		t.Fatalf("want %v, got %v", "cargo", lines[0])
	}
	if !stdlibAssertEqual("build", lines[1]) {
		t.Fatalf("want %v, got %v", "build", lines[1])
	}
	if !stdlibAssertEqual("--release", lines[2]) {
		t.Fatalf("want %v, got %v", "--release", lines[2])
	}
	if !stdlibAssertEqual("--target", lines[3]) {
		t.Fatalf("want %v, got %v", "--target", lines[3])
	}
	if !stdlibAssertEqual("x86_64-unknown-linux-gnu", lines[4]) {
		t.Fatalf("want %v, got %v", "x86_64-unknown-linux-gnu", lines[4])
	}
	if !stdlibAssertContains(lines, "CARGO_TARGET_DIR="+ax.Join(outputDir, "linux_amd64")) {
		t.Fatalf("expected %v to contain %v", lines, "CARGO_TARGET_DIR="+ax.Join(outputDir, "linux_amd64"))
	}
	if !stdlibAssertContains(string(content), "FOO=bar") {
		t.Fatalf("expected %v to contain %v", string(content), "FOO=bar")
	}

}

func TestRust_RustBuilderInterface_Good(t *testing.T) {
	var _ build.Builder = (*RustBuilder)(nil)
	var _ build.Builder = NewRustBuilder()
}
