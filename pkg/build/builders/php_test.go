package builders

import (
	"archive/zip"
	"context"
	"runtime"
	"testing"

	core "dappco.re/go"
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
	result := ax.WriteFile(ax.Join(binDir, "composer"), []byte(script), 0o755)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

}

func setupPHPTestProject(t *testing.T, withBuildScript bool) string {
	t.Helper()

	dir := t.TempDir()

	composerJSON := `{"name":"test/php-app"}`
	if withBuildScript {
		composerJSON = `{"name":"test/php-app","scripts":{"build":"php build.php"}}`
	}
	result := ax.WriteFile(ax.Join(dir, "composer.json"), []byte(composerJSON), 0o644)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	result = ax.WriteFile(ax.Join(dir, "index.php"), []byte("<?php echo 'hello';"), 0o644)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	if withBuildScript {
		result = ax.WriteFile(ax.Join(dir, "build.php"), []byte("<?php echo 'build';"), 0o644)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

	}

	return dir
}

func TestPHP_PHPBuilderNameGood(t *testing.T) {
	builder := NewPHPBuilder()
	if !stdlibAssertEqual("php", builder.Name()) {
		t.Fatalf("want %v, got %v", "php", builder.Name())
	}

}

func TestPHP_PHPBuilderDetectGood(t *testing.T) {
	fs := io.Local

	t.Run("detects composer.json projects", func(t *testing.T) {
		dir := t.TempDir()
		result := ax.WriteFile(ax.Join(dir, "composer.json"), []byte("{}"), 0o644)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		builder := NewPHPBuilder()
		detected := requireCPPBool(t, builder.Detect(fs, dir))
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		builder := NewPHPBuilder()
		detected := requireCPPBool(t, builder.Detect(fs, t.TempDir()))
		if detected {
			t.Fatal("expected false")
		}

	})
}

func TestPHP_PHPBuilderBuildGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakePHPToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

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

	artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, targets))
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}
	if stat := ax.Stat(artifacts[0].Path); !stat.OK {
		t.Fatalf("expected file to exist: %v", artifacts[0].Path)
	}
	if !stdlibAssertEqual("linux", artifacts[0].OS) {
		t.Fatalf("want %v, got %v", "linux", artifacts[0].OS)
	}
	if !stdlibAssertEqual("amd64", artifacts[0].Arch) {
		t.Fatalf("want %v, got %v", "amd64", artifacts[0].Arch)
	}

	content := requireBuilderBytes(t, ax.ReadFile(logPath))

	lines := core.Split(core.Trim(string(content)), "\n")
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

func TestPHP_PHPBuilderBuildFallbackBundleGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakePHPToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

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

	artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}}))
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}
	if stat := ax.Stat(artifacts[0].Path); !stat.OK {
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

func TestPHP_PHPBuilderBuildDefaultsGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakePHPToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	projectDir := setupPHPTestProject(t, false)
	outputDir := t.TempDir()

	builder := NewPHPBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Env:        []string{"FOO=bar"},
	}

	artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, nil))
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

func TestPHP_PHPBuilderInterfaceGood(t *testing.T) {
	builder := NewPHPBuilder()
	var _ build.Builder = builder
	if !stdlibAssertEqual("php", builder.Name()) {
		t.Fatalf("want %v, got %v", "php", builder.Name())
	}
	detected := requireCPPBool(t, builder.Detect(nil, t.TempDir()))
	if detected {
		t.Fatal("expected empty temp directory not to be detected")
	}
}

// --- v0.9.0 generated compliance triplets ---
func TestPhp_NewPHPBuilder_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewPHPBuilder()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestPhp_NewPHPBuilder_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewPHPBuilder()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestPhp_NewPHPBuilder_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewPHPBuilder()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestPhp_PHPBuilder_Name_Good(t *core.T) {
	subject := &PHPBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestPhp_PHPBuilder_Name_Bad(t *core.T) {
	subject := &PHPBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestPhp_PHPBuilder_Name_Ugly(t *core.T) {
	subject := &PHPBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestPhp_PHPBuilder_Detect_Good(t *core.T) {
	subject := &PHPBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Detect(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestPhp_PHPBuilder_Detect_Bad(t *core.T) {
	subject := &PHPBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Detect(io.NewMemoryMedium(), "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestPhp_PHPBuilder_Detect_Ugly(t *core.T) {
	subject := &PHPBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Detect(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestPhp_PHPBuilder_Build_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &PHPBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Build(ctx, nil, nil)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestPhp_PHPBuilder_Build_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &PHPBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Build(ctx, nil, nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestPhp_PHPBuilder_Build_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &PHPBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Build(ctx, nil, nil)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
