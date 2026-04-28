package builders

import (
	"context"
	"os"
	"runtime"
	"strings"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/internal/testassert"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/io"
)

// setupGoTestProject creates a minimal Go project for testing.
func setupGoTestProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create a minimal go.mod
	goMod := `module testproject

go 1.21
`
	err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte(goMod), 0644)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Create a minimal main.go
	mainGo := `package main

func main() {
	println("hello")
}
`
	err = ax.WriteFile(ax.Join(dir, "main.go"), []byte(mainGo), 0644)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return dir
}

func setupFakeBuildToolchain(t *testing.T, binDir string) {
	t.Helper()

	goScript := `#!/bin/sh
set -eu

log_file="${GO_BUILD_LOG_FILE:-}"
if [ -n "$log_file" ]; then
	printf '%s\n' "$@" > "$log_file"
fi

env_log_file="${GO_BUILD_ENV_LOG_FILE:-}"
if [ -n "$env_log_file" ]; then
	env | sort > "$env_log_file"
fi

if [ "${GOARCH:-}" = "invalid_arch" ]; then
	exit 1
fi

if [ -f main.go ] && grep -q "not valid go code" main.go; then
	exit 1
fi

output=""
previous=""
for argument in "$@"; do
	if [ "$previous" = "-o" ]; then
		output="$argument"
		break
	fi
	previous="$argument"
done

if [ -n "$output" ]; then
	mkdir -p "$(dirname "$output")"
	printf 'fake binary\n' > "$output"
	chmod +x "$output"
fi
`

	err := ax.WriteFile(ax.Join(binDir, "go"), []byte(goScript), 0o755)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	garbleScript := `#!/bin/sh
set -eu

log_file="${GARBLE_LOG_FILE:-}"
if [ -n "$log_file" ]; then
	printf '%s\n' "$@" > "$log_file"
fi

exec go "$@"
`

	err = ax.WriteFile(ax.Join(binDir, "garble"), []byte(garbleScript), 0o755)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}

func setupFakeGoBinary(t *testing.T, binDir string) {
	t.Helper()

	goScript := `#!/bin/sh
set -eu

log_file="${GO_BUILD_LOG_FILE:-}"
if [ -n "$log_file" ]; then
	printf '%s\n' "$@" > "$log_file"
fi

env_log_file="${GO_BUILD_ENV_LOG_FILE:-}"
if [ -n "$env_log_file" ]; then
	env | sort > "$env_log_file"
fi

if [ "${GOARCH:-}" = "invalid_arch" ]; then
	exit 1
fi

if [ -f main.go ] && grep -q "not valid go code" main.go; then
	exit 1
fi

output=""
previous=""
for argument in "$@"; do
	if [ "$previous" = "-o" ]; then
		output="$argument"
		break
	fi
	previous="$argument"
done

if [ -n "$output" ]; then
	mkdir -p "$(dirname "$output")"
	printf 'fake binary\n' > "$output"
	chmod +x "$output"
fi
`

	err := ax.WriteFile(ax.Join(binDir, "go"), []byte(goScript), 0o755)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}

func setupFakeGarbleBinary(t *testing.T, binDir string) {
	t.Helper()

	garbleScript := `#!/bin/sh
set -eu

log_file="${GARBLE_LOG_FILE:-}"
if [ -n "$log_file" ]; then
	printf '%s\n' "$@" > "$log_file"
fi

exec go "$@"
`

	err := ax.WriteFile(ax.Join(binDir, "garble"), []byte(garbleScript), 0o755)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}

func TestGo_GoBuilderName_Good(t *testing.T) {
	builder := NewGoBuilder()
	if !stdlibAssertEqual("go", builder.Name()) {
		t.Fatalf("want %v, got %v", "go", builder.Name())
	}

}

func TestGo_GoBuilderDetect_Good(t *testing.T) {
	fs := io.Local
	t.Run("detects Go project with go.mod", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module test"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewGoBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("detects Wails project", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "wails.json"), []byte("{}"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewGoBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("returns false for non-Go project", func(t *testing.T) {
		dir := t.TempDir()
		// Create a Node.js project instead
		err := ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewGoBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detected {
			t.Fatal("expected false")
		}

	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		dir := t.TempDir()

		builder := NewGoBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detected {
			t.Fatal("expected false")
		}

	})
}

func TestGo_GoBuilderBuild_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeBuildToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	t.Run("builds for current platform", func(t *testing.T) {
		projectDir := setupGoTestProject(t)
		outputDir := t.TempDir()

		builder := NewGoBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
			Name:       "testbinary",
		}
		targets := []build.Target{
			{OS: runtime.GOOS, Arch: runtime.GOARCH},
		}

		artifacts, err := builder.Build(context.Background(), cfg, targets)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(artifacts) !=

			// Verify artifact properties
			1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}

		artifact := artifacts[0]
		if !stdlibAssertEqual(runtime.GOOS, artifact.OS) {
			t.Fatalf("want %v, got %v", runtime.GOOS, artifact.OS)

			// Verify binary was created
		}
		if !stdlibAssertEqual(runtime.GOARCH, artifact.Arch) {
			t.Fatalf("want %v, got %v",

				// Verify the path is in the expected location
				runtime.GOARCH, artifact.Arch)
		}
		if _, err := os.Stat(artifact.Path); err != nil {
			t.Fatalf("expected file to exist: %v", artifact.Path)
		}

		expectedName := "testbinary"
		if runtime.GOOS == "windows" {
			expectedName += ".exe"
		}
		if !stdlibAssertContains(artifact.Path, expectedName) {
			t.Fatalf("expected %v to contain %v", artifact.Path, expectedName)
		}

	})

	t.Run("defaults to current platform when targets are empty", func(t *testing.T) {
		projectDir := setupGoTestProject(t)
		outputDir := t.TempDir()

		builder := NewGoBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
			Name:       "fallback",
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
		if _, err := os.Stat(artifacts[0].Path); err != nil {
			t.Fatalf("expected file to exist: %v", artifacts[0].Path)
		}

	})

	t.Run("does not mutate the caller output directory when using defaults", func(t *testing.T) {
		projectDir := setupGoTestProject(t)

		builder := NewGoBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			Name:       "mutability",
		}

		artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: runtime.GOOS, Arch: runtime.GOARCH}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if !stdlibAssertEmpty(cfg.OutputDir) {
			t.Fatalf("expected empty, got %v", cfg.OutputDir)
		}
		if !stdlibAssertEqual(ax.Join(projectDir, "dist"), ax.Dir(ax.Dir(artifacts[0].Path))) {
			t.Fatalf("want %v, got %v", ax.Join(projectDir, "dist"), ax.Dir(ax.Dir(artifacts[0].Path)))
		}

	})

	t.Run("builds multiple targets", func(t *testing.T) {
		projectDir := setupGoTestProject(t)
		outputDir := t.TempDir()

		builder := NewGoBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
			Name:       "multitest",
		}
		targets := []build.Target{
			{OS: "linux", Arch: "amd64"},
			{OS: "linux", Arch: "arm64"},
		}

		artifacts, err := builder.Build(context.Background(), cfg, targets)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(artifacts) !=

			// Verify both artifacts were created
			2 {
			t.Fatalf("want len %v, got %v", 2, len(artifacts))
		}

		for i, artifact := range artifacts {
			if !stdlibAssertEqual(targets[i].OS, artifact.OS) {
				t.Fatalf("want %v, got %v", targets[i].OS, artifact.OS)
			}
			if !stdlibAssertEqual(targets[i].Arch, artifact.Arch) {
				t.Fatalf("want %v, got %v", targets[i].Arch, artifact.Arch)
			}
			if _, err := os.Stat(artifact.Path); err != nil {
				t.Fatalf("expected file to exist: %v", artifact.Path)
			}

		}
	})

	t.Run("adds .exe extension for Windows", func(t *testing.T) {
		projectDir := setupGoTestProject(t)
		outputDir := t.TempDir()

		builder := NewGoBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
			Name:       "wintest",
		}
		targets := []build.Target{
			{OS: "windows", Arch: "amd64"},
		}

		artifacts, err := builder.Build(context.Background(), cfg, targets)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(artifacts) !=

			// Verify .exe extension
			1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if !(ax.Ext(artifacts[0].Path) == ".exe") {
			t.Fatal("expected true")
		}
		if _, err := os.Stat(artifacts[0].Path); err != nil {
			t.Fatalf("expected file to exist: %v", artifacts[0].Path)
		}

	})

	t.Run("uses directory name when Name not specified", func(t *testing.T) {
		projectDir := setupGoTestProject(t)
		outputDir := t.TempDir()

		builder := NewGoBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
			Name:       "", // Empty name
		}
		targets := []build.Target{
			{OS: runtime.GOOS, Arch: runtime.GOARCH},
		}

		artifacts, err := builder.Build(context.Background(), cfg, targets)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(artifacts) !=

			// Binary should use the project directory base name
			1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}

		baseName := ax.Base(projectDir)
		if runtime.GOOS == "windows" {
			baseName += ".exe"
		}
		if !stdlibAssertContains(artifacts[0].Path, baseName) {
			t.Fatalf("expected %v to contain %v", artifacts[0].Path, baseName)
		}

	})

	t.Run("uses configured project binary when Name not specified", func(t *testing.T) {
		projectDir := setupGoTestProject(t)
		outputDir := t.TempDir()

		builder := NewGoBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
		}
		cfg.Project.Binary = "example-binary"
		targets := []build.Target{
			{OS: runtime.GOOS, Arch: runtime.GOARCH},
		}

		artifacts, err := builder.Build(context.Background(), cfg, targets)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}

		expectedName := "example-binary"
		if runtime.GOOS == "windows" {
			expectedName += ".exe"
		}
		if !stdlibAssertContains(artifacts[0].Path, expectedName) {
			t.Fatalf("expected %v to contain %v", artifacts[0].Path, expectedName)
		}

	})

	t.Run("uses configured project name when Binary not specified", func(t *testing.T) {
		projectDir := setupGoTestProject(t)
		outputDir := t.TempDir()

		builder := NewGoBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
		}
		cfg.Project.Name = "example-name"
		targets := []build.Target{
			{OS: runtime.GOOS, Arch: runtime.GOARCH},
		}

		artifacts, err := builder.Build(context.Background(), cfg, targets)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}

		expectedName := "example-name"
		if runtime.GOOS == "windows" {
			expectedName += ".exe"
		}
		if !stdlibAssertContains(artifacts[0].Path, expectedName) {
			t.Fatalf("expected %v to contain %v", artifacts[0].Path, expectedName)
		}

	})

	t.Run("applies ldflags", func(t *testing.T) {
		projectDir := setupGoTestProject(t)
		outputDir := t.TempDir()

		builder := NewGoBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
			Name:       "ldflagstest",
			LDFlags:    []string{"-s", "-w"}, // Strip debug info
		}
		targets := []build.Target{
			{OS: runtime.GOOS, Arch: runtime.GOARCH},
		}

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

	})

	t.Run("applies config flags and env", func(t *testing.T) {
		projectDir := setupGoTestProject(t)
		outputDir := t.TempDir()
		logDir := t.TempDir()
		argsLogPath := ax.Join(logDir, "go-args.log")
		envLogPath := ax.Join(logDir, "go-env.log")

		t.Setenv("GO_BUILD_LOG_FILE", argsLogPath)
		t.Setenv("GO_BUILD_ENV_LOG_FILE", envLogPath)

		builder := NewGoBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
			Name:       "envflags",
			Version:    "v1.2.3",
			Flags:      []string{"-race"},
			Env:        []string{"FOO=bar", "BAR=baz"},
		}
		targets := []build.Target{
			{OS: runtime.GOOS, Arch: runtime.GOARCH},
		}

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

		argsContent, err := ax.ReadFile(argsLogPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		args := strings.Split(strings.TrimSpace(string(argsContent)), "\n")
		if stdlibAssertEmpty(args) {
			t.Fatal("expected non-empty")
		}
		if !stdlibAssertEqual("build", args[0]) {
			t.Fatalf("want %v, got %v", "build", args[0])
		}
		if !stdlibAssertContains(args, "-trimpath") {
			t.Fatalf("expected %v to contain %v", args, "-trimpath")
		}
		if !stdlibAssertContains(args, "-race") {
			t.Fatalf("expected %v to contain %v", args, "-race")
		}

		envContent, err := ax.ReadFile(envLogPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		envLines := strings.Split(strings.TrimSpace(string(envContent)), "\n")
		if !stdlibAssertContains(envLines, "BAR=baz") {
			t.Fatalf("expected %v to contain %v", envLines, "BAR=baz")
		}
		if !stdlibAssertContains(envLines, "FOO=bar") {
			t.Fatalf("expected %v to contain %v", envLines, "FOO=bar")
		}
		if !stdlibAssertContains(envLines, "TARGET_OS="+runtime.GOOS) {
			t.Fatalf("expected %v to contain %v", envLines, "TARGET_OS="+runtime.GOOS)
		}
		if !stdlibAssertContains(envLines, "TARGET_ARCH="+runtime.GOARCH) {
			t.Fatalf("expected %v to contain %v", envLines, "TARGET_ARCH="+runtime.GOARCH)
		}
		if !stdlibAssertContains(envLines, "OUTPUT_DIR="+outputDir) {
			t.Fatalf("expected %v to contain %v", envLines, "OUTPUT_DIR="+outputDir)
		}
		if !stdlibAssertContains(envLines, "TARGET_DIR="+ax.Join(outputDir, runtime.GOOS+"_"+runtime.GOARCH)) {
			t.Fatalf("expected %v to contain %v", envLines, "TARGET_DIR="+ax.Join(outputDir, runtime.GOOS+"_"+runtime.GOARCH))
		}
		if !stdlibAssertContains(envLines, "GOOS="+runtime.GOOS) {
			t.Fatalf("expected %v to contain %v", envLines, "GOOS="+runtime.GOOS)
		}
		if !stdlibAssertContains(envLines, "GOARCH="+runtime.GOARCH) {
			t.Fatalf("expected %v to contain %v", envLines, "GOARCH="+runtime.GOARCH)
		}
		if !stdlibAssertContains(envLines, "NAME=envflags") {
			t.Fatalf("expected %v to contain %v", envLines, "NAME=envflags")
		}
		if !stdlibAssertContains(envLines, "VERSION=v1.2.3") {
			t.Fatalf("expected %v to contain %v", envLines, "VERSION=v1.2.3")
		}
		if !stdlibAssertContains(envLines, "CGO_ENABLED=0") {
			t.Fatalf("expected %v to contain %v", envLines, "CGO_ENABLED=0")
		}

	})

	t.Run("applies configured cache paths to go cache env vars", func(t *testing.T) {
		projectDir := setupGoTestProject(t)
		outputDir := t.TempDir()
		logDir := t.TempDir()
		envLogPath := ax.Join(logDir, "go-cache-env.log")

		t.Setenv("GO_BUILD_ENV_LOG_FILE", envLogPath)

		builder := NewGoBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
			Name:       "cachetest",
			Cache: build.CacheConfig{
				Enabled: true,
				Paths: []string{
					ax.Join(outputDir, "cache", "go-build"),
					ax.Join(outputDir, "cache", "go-mod"),
				},
			},
		}
		targets := []build.Target{{OS: runtime.GOOS, Arch: runtime.GOARCH}}

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

		envContent, err := ax.ReadFile(envLogPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		envLines := strings.Split(strings.TrimSpace(string(envContent)), "\n")
		if !stdlibAssertContains(envLines, "GOCACHE="+ax.Join(outputDir, "cache", "go-build")) {
			t.Fatalf("expected %v to contain %v", envLines, "GOCACHE="+ax.Join(outputDir, "cache", "go-build"))
		}
		if !stdlibAssertContains(envLines, "GOMODCACHE="+ax.Join(outputDir, "cache", "go-mod")) {
			t.Fatalf("expected %v to contain %v", envLines, "GOMODCACHE="+ax.Join(outputDir, "cache", "go-mod"))
		}

	})

	t.Run("passes build tags through to go build", func(t *testing.T) {
		projectDir := setupGoTestProject(t)
		outputDir := t.TempDir()
		logPath := ax.Join(t.TempDir(), "go-tags.log")
		t.Setenv("GO_BUILD_LOG_FILE", logPath)

		builder := NewGoBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
			Name:       "tagged",
			BuildTags:  []string{"webkit2_41", "integration"},
		}
		targets := []build.Target{{OS: runtime.GOOS, Arch: runtime.GOARCH}}

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

		content, err := ax.ReadFile(logPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		args := strings.Split(strings.TrimSpace(string(content)), "\n")
		if stdlibAssertEmpty(args) {
			t.Fatal("expected non-empty")
		}
		if !stdlibAssertEqual("build", args[0]) {
			t.Fatalf("want %v, got %v", "build", args[0])
		}
		if !stdlibAssertContains(args, "-tags") {
			t.Fatalf("expected %v to contain %v", args, "-tags")
		}
		if !stdlibAssertContains(args, "webkit2_41,integration") {
			t.Fatalf("expected %v to contain %v", args, "webkit2_41,integration")
		}

	})

	t.Run("injects version into ldflags and environment", func(t *testing.T) {
		projectDir := setupGoTestProject(t)
		outputDir := t.TempDir()
		argsLogPath := ax.Join(t.TempDir(), "go-version-args.log")
		envLogPath := ax.Join(t.TempDir(), "go-version-env.log")

		t.Setenv("GO_BUILD_LOG_FILE", argsLogPath)
		t.Setenv("GO_BUILD_ENV_LOG_FILE", envLogPath)

		builder := NewGoBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
			Name:       "versioned",
			Version:    "v1.2.3",
		}
		targets := []build.Target{{OS: runtime.GOOS, Arch: runtime.GOARCH}}

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

		argsContent, err := ax.ReadFile(argsLogPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		args := strings.Split(strings.TrimSpace(string(argsContent)), "\n")
		if stdlibAssertEmpty(args) {
			t.Fatal("expected non-empty")
		}
		if !stdlibAssertContains(args, "-ldflags") {
			t.Fatalf("expected %v to contain %v", args, "-ldflags")
		}
		if !stdlibAssertContains(args, "-X main.version=v1.2.3") {
			t.Fatalf("expected %v to contain %v", args, "-X main.version=v1.2.3")
		}

		envContent, err := ax.ReadFile(envLogPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		envLines := strings.Split(strings.TrimSpace(string(envContent)), "\n")
		if !stdlibAssertContains(envLines, "VERSION=v1.2.3") {
			t.Fatalf("expected %v to contain %v", envLines, "VERSION=v1.2.3")
		}

	})

	t.Run("uses garble when obfuscation is enabled", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("garble test helper uses a shell script")
		}

		projectDir := setupGoTestProject(t)
		outputDir := t.TempDir()
		logDir := t.TempDir()
		logPath := ax.Join(logDir, "garble.log")

		t.Setenv("GARBLE_LOG_FILE", logPath)

		builder := NewGoBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
			Name:       "obfuscated",
			Obfuscate:  true,
		}
		targets := []build.Target{
			{OS: runtime.GOOS, Arch: runtime.GOARCH},
		}

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

		content, err := ax.ReadFile(logPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		args := strings.Split(strings.TrimSpace(string(content)), "\n")
		if stdlibAssertEmpty(args) {
			t.Fatal("expected non-empty")
		}
		if !stdlibAssertEqual("build", args[0]) {
			t.Fatalf("want %v, got %v", "build", args[0])
		}
		if !stdlibAssertContains(args, "-trimpath") {
			t.Fatalf("expected %v to contain %v", args, "-trimpath")
		}
		if !stdlibAssertContains(args, "-o") {
			t.Fatalf("expected %v to contain %v", args, "-o")
		}
		if !stdlibAssertContains(args, ".") {
			t.Fatalf("expected %v to contain %v", args, ".")
		}

	})

	t.Run("finds garble in GOBIN when it is not on PATH", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("garble test helper uses a shell script")
		}

		goDir := t.TempDir()
		setupFakeGoBinary(t, goDir)
		t.Setenv("PATH", goDir+string(os.PathListSeparator)+"/usr/bin"+string(os.PathListSeparator)+"/bin")

		garbleDir := t.TempDir()
		setupFakeGarbleBinary(t, garbleDir)
		t.Setenv("GOBIN", garbleDir)

		projectDir := setupGoTestProject(t)
		outputDir := t.TempDir()
		logDir := t.TempDir()
		logPath := ax.Join(logDir, "garble-gobin.log")

		t.Setenv("GARBLE_LOG_FILE", logPath)

		builder := NewGoBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
			Name:       "obfuscated-gobin",
			Obfuscate:  true,
		}
		targets := []build.Target{{OS: runtime.GOOS, Arch: runtime.GOARCH}}

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

		content, err := ax.ReadFile(logPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		args := strings.Split(strings.TrimSpace(string(content)), "\n")
		if stdlibAssertEmpty(args) {
			t.Fatal("expected non-empty")
		}
		if !stdlibAssertEqual("build", args[0]) {
			t.Fatalf("want %v, got %v", "build", args[0])
		}
		if !stdlibAssertContains(args, "-trimpath") {
			t.Fatalf("expected %v to contain %v", args, "-trimpath")
		}

	})

	t.Run("builds the configured main package path", func(t *testing.T) {
		projectDir := setupGoTestProject(t)
		err := ax.MkdirAll(ax.Join(projectDir, "cmd", "myapp"), 0755)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = ax.WriteFile(ax.Join(projectDir, "cmd", "myapp", "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		outputDir := t.TempDir()
		logPath := ax.Join(t.TempDir(), "go-build-args.log")
		t.Setenv("GO_BUILD_LOG_FILE", logPath)

		builder := NewGoBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
			Name:       "mainpackage",
		}
		cfg.Project.Main = "./cmd/myapp"
		targets := []build.Target{
			{OS: runtime.GOOS, Arch: runtime.GOARCH},
		}

		artifacts, err := builder.Build(context.Background(), cfg, targets)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}

		content, err := ax.ReadFile(logPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		args := strings.Split(strings.TrimSpace(string(content)), "\n")
		if stdlibAssertEmpty(args) {
			t.Fatal("expected non-empty")
		}
		if !stdlibAssertContains(args, "./cmd/myapp") {
			t.Fatalf("expected %v to contain %v", args, "./cmd/myapp")
		}
		if stdlibAssertContains(args, ".") {
			t.Fatalf("expected %v not to contain %v", args, ".")
		}

	})

	t.Run("creates output directory if missing", func(t *testing.T) {
		projectDir := setupGoTestProject(t)
		outputDir := ax.Join(t.TempDir(), "nested", "output")

		builder := NewGoBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
			Name:       "nestedtest",
		}
		targets := []build.Target{
			{OS: runtime.GOOS, Arch: runtime.GOARCH},
		}

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
		if info, err := os.Stat(outputDir); err != nil {
			t.Fatalf("expected directory to exist: %v", outputDir)
		} else if !info.IsDir() {
			t.Fatalf("expected directory to exist: %v", outputDir)
		}

	})

	t.Run("defaults output directory to project dist when not specified", func(t *testing.T) {
		projectDir := setupGoTestProject(t)

		builder := NewGoBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			Name:       "defaultoutput",
		}
		targets := []build.Target{
			{OS: runtime.GOOS, Arch: runtime.GOARCH},
		}

		artifacts, err := builder.Build(context.Background(), cfg, targets)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}

		expectedDir := ax.Join(projectDir, "dist")
		if info, err := os.Stat(expectedDir); err != nil {
			t.Fatalf("expected directory to exist: %v", expectedDir)
		} else if !info.IsDir() {
			t.Fatalf("expected directory to exist: %v", expectedDir)
		}
		if !stdlibAssertContains(artifacts[0].Path, expectedDir) {
			t.Fatalf("expected %v to contain %v", artifacts[0].Path, expectedDir)
		}
		if _, err := os.Stat(artifacts[0].Path); err != nil {
			t.Fatalf("expected file to exist: %v", artifacts[0].Path)
		}

	})
}

func TestGo_GoBuilderBuild_Bad(t *testing.T) {
	binDir := t.TempDir()
	setupFakeBuildToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	t.Run("returns error for nil config", func(t *testing.T) {
		builder := NewGoBuilder()

		artifacts, err := builder.Build(context.Background(), nil, []build.Target{{OS: "linux", Arch: "amd64"}})
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertNil(artifacts) {
			t.Fatalf("expected nil, got %v", artifacts)
		}
		if !stdlibAssertContains(err.Error(), "config is nil") {
			t.Fatalf("expected %v to contain %v", err.Error(), "config is nil")
		}

	})

	t.Run("defaults to current platform when targets are empty", func(t *testing.T) {
		projectDir := setupGoTestProject(t)

		builder := NewGoBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  t.TempDir(),
			Name:       "test",
		}

		artifacts, err := builder.Build(context.Background(), cfg, []build.Target{})
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
		if _, err := os.Stat(artifacts[0].Path); err != nil {
			t.Fatalf("expected file to exist: %v", artifacts[0].Path)
		}

	})

	t.Run("returns error for invalid project directory", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping integration test in short mode")
		}

		builder := NewGoBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: "/nonexistent/path",
			OutputDir:  t.TempDir(),
			Name:       "test",
		}
		targets := []build.Target{
			{OS: runtime.GOOS, Arch: runtime.GOARCH},
		}

		artifacts, err := builder.Build(context.Background(), cfg, targets)
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertEmpty(artifacts) {
			t.Fatalf("expected empty, got %v", artifacts)
		}

	})

	t.Run("returns error for invalid Go code", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping integration test in short mode")
		}

		dir := t.TempDir()

		// Create go.mod
		err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module test\n\ngo 1.21"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v",

				// Create invalid Go code
				err)
		}

		err = ax.WriteFile(ax.Join(dir, "main.go"), []byte("this is not valid go code"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewGoBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: dir,
			OutputDir:  t.TempDir(),
			Name:       "test",
		}
		targets := []build.Target{
			{OS: runtime.GOOS, Arch: runtime.GOARCH},
		}

		artifacts, err := builder.Build(context.Background(), cfg, targets)
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "go build failed") {
			t.Fatalf("expected %v to contain %v", err.Error(), "go build failed")
		}
		if !stdlibAssertEmpty(artifacts) {
			t.Fatalf("expected empty, got %v", artifacts)
		}

	})

	t.Run("returns partial artifacts on partial failure", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping integration test in short mode")
		}

		// Create a project that will fail on one target
		// Using an invalid arch for linux
		projectDir := setupGoTestProject(t)
		outputDir := t.TempDir()

		builder := NewGoBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
			Name:       "partialtest",
		}
		targets := []build.Target{
			{OS: runtime.GOOS, Arch: runtime.GOARCH}, // This should succeed
			{OS: "linux", Arch: "invalid_arch"},      // This should fail
		}

		artifacts, err := builder.Build(context.Background(), cfg, targets)
		if err ==
			// Should return error for the failed build
			nil {
			t.Fatal("expected error")
		}
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v",

				// Should have the successful artifact
				1, len(artifacts))
		}

	})

	t.Run("respects context cancellation", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping integration test in short mode")
		}

		projectDir := setupGoTestProject(t)

		builder := NewGoBuilder()
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
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertEmpty(artifacts) {
			t.Fatalf("expected empty, got %v", artifacts)
		}

	})

	t.Run("rejects unsafe version identifiers before invoking go build", func(t *testing.T) {
		projectDir := setupGoTestProject(t)

		builder := NewGoBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  t.TempDir(),
			Name:       "unsafe-version",
			Version:    "v1.2.3;rm -rf /",
		}

		artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: runtime.GOOS, Arch: runtime.GOARCH}})
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertEmpty(artifacts) {
			t.Fatalf("expected empty, got %v", artifacts)
		}
		if !stdlibAssertContains(err.Error(), "unsupported characters") {
			t.Fatalf("expected %v to contain %v", err.Error(), "unsupported characters")
		}

	})
}

func TestGo_GoBuilderResolveGarbleCli_Good(t *testing.T) {
	t.Run("returns an explicit fallback path when it exists", func(t *testing.T) {
		builder := NewGoBuilder()
		garblePath := ax.Join(t.TempDir(), "garble")
		if err := ax.WriteFile(garblePath, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		t.Setenv("PATH", t.TempDir())

		command, err := builder.resolveGarbleCli(garblePath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(garblePath, command) {
			t.Fatalf("want %v, got %v", garblePath, command)
		}

	})
}

func TestGo_GoBuilderResolveGarbleCli_Bad(t *testing.T) {
	t.Run("returns an error when garble cannot be resolved", func(t *testing.T) {
		builder := NewGoBuilder()
		t.Setenv("PATH", t.TempDir())

		command, err := builder.resolveGarbleCli(ax.Join(t.TempDir(), "missing-garble"))
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertEmpty(command) {
			t.Fatalf("expected empty, got %v", command)
		}
		if !stdlibAssertContains(err.Error(), "garble CLI not found") {
			t.Fatalf("expected %v to contain %v", err.Error(), "garble CLI not found")
		}

	})
}

func TestGo_GarbleInstallPaths_Ugly(t *testing.T) {
	gobin := ax.Join(t.TempDir(), "gobin")
	gopathOne := ax.Join(t.TempDir(), "gopath-one")
	gopathTwo := ax.Join(t.TempDir(), "gopath-two")

	t.Setenv("GOBIN", gobin)
	t.Setenv("GOPATH", gopathOne+string(os.PathListSeparator)+" "+string(os.PathListSeparator)+gopathTwo)

	paths := garbleInstallPaths()
	if !stdlibAssertEqual([]string{ax.Join(gobin, "garble"), ax.Join(gopathOne, "bin", "garble"), ax.Join(gopathTwo, "bin", "garble")}, paths) {
		t.Fatalf("want %v, got %v", []string{ax.Join(gobin, "garble"), ax.Join(gopathOne, "bin", "garble"), ax.Join(gopathTwo, "bin", "garble")}, paths)
	}

}

func TestGo_hasVersionLDFlag_Good(t *testing.T) {
	if !(hasVersionLDFlag([]string{"-s", "-w", "-X main.version=v1.2.3"})) {
		t.Fatal("expected true")
	}
	if !(hasVersionLDFlag([]string{"-X main.Version=v1.2.3"})) {
		t.Fatal("expected true")
	}

}

func TestGo_hasVersionLDFlag_Bad(t *testing.T) {
	if hasVersionLDFlag([]string{"-s", "-w"}) {
		t.Fatal("expected false")
	}

}

func TestGo_containsString_Ugly(t *testing.T) {
	if !(containsString([]string{"alpha", "beta"}, "beta")) {
		t.Fatal("expected true")
	}
	if containsString([]string{"alpha", "beta"}, "gamma") {
		t.Fatal("expected false")
	}

}

func TestGo_GoBuilderInterface_Good(t *testing.T) {
	builder := NewGoBuilder()
	var _ build.Builder = builder
	if !stdlibAssertEqual("go", builder.Name()) {
		t.Fatalf("want %v, got %v", "go", builder.Name())
	}
	detected, err := builder.Detect(nil, t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detected {
		t.Fatal("expected empty temp directory not to be detected")
	}
}

var (
	stdlibAssertEqual         = testassert.Equal
	stdlibAssertNil           = testassert.Nil
	stdlibAssertEmpty         = testassert.Empty
	stdlibAssertZero          = testassert.Zero
	stdlibAssertContains      = testassert.Contains
	stdlibAssertElementsMatch = testassert.ElementsMatch
)
