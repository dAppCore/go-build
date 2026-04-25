package builders

import (
	"archive/zip"
	"context"
	"os"
	"runtime"
	"strings"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/io"
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
	env | sort >> "$log_file"
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
	if err := ax.WriteFile(ax.Join(binDir, "composer"), []byte(script), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}

func setupPHPTestProject(t *testing.T, withBuildScript bool) string {
	t.Helper()

	dir := t.TempDir()

	composerJSON := `{"name":"test/php-app"}`
	if withBuildScript {
		composerJSON = `{"name":"test/php-app","scripts":{"build":"php build.php"}}`
	}
	if err := ax.WriteFile(ax.Join(dir, "composer.json"), []byte(composerJSON), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(dir, "index.php"), []byte("<?php echo 'hello';"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if withBuildScript {
		if err := ax.WriteFile(ax.Join(dir, "build.php"), []byte("<?php echo 'build';"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

	}

	return dir
}

func TestPHP_PHPBuilderName_Good(t *testing.T) {
	builder := NewPHPBuilder()
	if !stdlibAssertEqual("php", builder.Name()) {
		t.Fatalf("want %v, got %v", "php", builder.Name())
	}

}

func TestPHP_PHPBuilderDetect_Good(t *testing.T) {
	fs := io.Local

	t.Run("detects composer.json projects", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "composer.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewPHPBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		builder := NewPHPBuilder()
		detected, err := builder.Detect(fs, t.TempDir())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detected {
			t.Fatal("expected false")
		}

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
	if len(lines) < 6 {
		t.Fatalf("expected %v to be greater than or equal to %v", len(lines), 6)
	}
	if !stdlibAssertEqual("composer", lines[0]) {
		t.Fatalf("want %v, got %v", "composer", lines[0])
	}
	if !stdlibAssertEqual("install", lines[1]) {
		t.Fatalf("want %v, got %v", "install", lines[1])
	}
	if !stdlibAssertContains(lines, "GOOS=linux") {
		t.Fatalf("expected %v to contain %v", lines, "GOOS=linux")
	}
	if !stdlibAssertContains(lines, "GOARCH=amd64") {
		t.Fatalf("expected %v to contain %v", lines, "GOARCH=amd64")
	}
	if !stdlibAssertContains(lines, "OUTPUT_DIR="+outputDir) {
		t.Fatalf("expected %v to contain %v", lines, "OUTPUT_DIR="+outputDir)
	}
	if !stdlibAssertContains(lines, "TARGET_DIR="+ax.Join(outputDir, "linux_amd64")) {
		t.Fatalf("expected %v to contain %v", lines, "TARGET_DIR="+ax.Join(outputDir, "linux_amd64"))
	}
	if !stdlibAssertContains(string(content), "FOO=bar") {
		t.Fatalf("expected %v to contain %v", string(content), "FOO=bar")
	}

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
		Env:        []string{"FOO=bar"},
	}

	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}
	if _, err := os.Stat(artifacts[0].Path); err != nil {
		t.Fatalf("expected file to exist: %v", artifacts[0].Path)
	}
	if !stdlibAssertEqual(".zip", ax.Ext(artifacts[0].Path)) {
		t.Fatalf("want %v, got %v", ".zip", ax.Ext(artifacts[0].Path))
	}

	reader, err := zip.OpenReader(artifacts[0].Path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defer func() { _ = reader.Close() }()

	var foundComposer bool
	for _, file := range reader.File {
		if !(file.Modified.Equal(deterministicZipTime)) {
			t.Fatal("expected true")
		}

		if file.Name == "composer.json" {
			foundComposer = true
			break
		}
	}
	if !(foundComposer) {
		t.Fatal("expected true")
	}

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
		Env:        []string{"FOO=bar"},
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

func TestPHP_PHPBuilderInterface_Good(t *testing.T) {
	var _ build.Builder = (*PHPBuilder)(nil)
	var _ build.Builder = NewPHPBuilder()
}
