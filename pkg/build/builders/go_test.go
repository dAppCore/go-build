package builders

import (
	"context"
	"runtime"
	"testing"

	core "dappco.re/go"
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
	if result := ax.WriteFile(ax.Join(dir, "go.mod"), []byte(goMod), 0644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	// Create a minimal main.go
	mainGo := `package main

func main() {
	println("hello")
}
`
	if result := ax.WriteFile(ax.Join(dir, "main.go"), []byte(mainGo), 0644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
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

	if result := ax.WriteFile(ax.Join(binDir, "go"), []byte(goScript), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	garbleScript := `#!/bin/sh
set -eu

log_file="${GARBLE_LOG_FILE:-}"
if [ -n "$log_file" ]; then
	printf '%s\n' "$@" > "$log_file"
fi

exec go "$@"
`

	if result := ax.WriteFile(ax.Join(binDir, "garble"), []byte(garbleScript), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
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

	if result := ax.WriteFile(ax.Join(binDir, "go"), []byte(goScript), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
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

	if result := ax.WriteFile(ax.Join(binDir, "garble"), []byte(garbleScript), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

}

func TestGo_GoBuilderNameGood(t *testing.T) {
	builder := NewGoBuilder()
	if !stdlibAssertEqual("go", builder.Name()) {
		t.Fatalf("want %v, got %v", "go", builder.Name())
	}

}

func TestGo_GoBuilderDetectGood(t *testing.T) {
	fs := io.Local
	t.Run("detects Go project with go.mod", func(t *testing.T) {
		dir := t.TempDir()
		if result := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module test"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		builder := NewGoBuilder()
		detected := requireCPPBool(t, builder.Detect(fs, dir))
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("detects Wails project", func(t *testing.T) {
		dir := t.TempDir()
		if result := ax.WriteFile(ax.Join(dir, "wails.json"), []byte("{}"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		builder := NewGoBuilder()
		detected := requireCPPBool(t, builder.Detect(fs, dir))
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("returns false for non-Go project", func(t *testing.T) {
		dir := t.TempDir()
		// Create a Node.js project instead
		if result := ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		builder := NewGoBuilder()
		detected := requireCPPBool(t, builder.Detect(fs, dir))
		if detected {
			t.Fatal("expected false")
		}

	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		dir := t.TempDir()

		builder := NewGoBuilder()
		detected := requireCPPBool(t, builder.Detect(fs, dir))
		if detected {
			t.Fatal("expected false")
		}

	})
}

func TestGo_GoBuilderBuildGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeBuildToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

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

		artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, targets))
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
		if result := ax.Stat(artifact.Path); !result.OK {
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
		if result := ax.Stat(artifacts[0].Path); !result.OK {
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

		artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, []build.Target{{OS: runtime.GOOS, Arch: runtime.GOARCH}}))
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

		artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, targets))
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
			if result := ax.Stat(artifact.Path); !result.OK {
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

		artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, targets))
		if len(artifacts) !=

			// Verify .exe extension
			1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if !(ax.Ext(artifacts[0].Path) == ".exe") {
			t.Fatal("expected true")
		}
		if result := ax.Stat(artifacts[0].Path); !result.OK {
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

		artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, targets))
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

		artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, targets))
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

		artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, targets))
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

		artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, targets))
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if result := ax.Stat(artifacts[0].Path); !result.OK {
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

		artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, targets))
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if result := ax.Stat(artifacts[0].Path); !result.OK {
			t.Fatalf("expected file to exist: %v", artifacts[0].Path)
		}

		argsContent := requireBuilderBytes(t, ax.ReadFile(argsLogPath))

		args := core.Split(core.Trim(string(argsContent)), "\n")
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

		envContent := requireBuilderBytes(t, ax.ReadFile(envLogPath))

		envLines := core.Split(core.Trim(string(envContent)), "\n")
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

		artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, targets))
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if result := ax.Stat(artifacts[0].Path); !result.OK {
			t.Fatalf("expected file to exist: %v", artifacts[0].Path)
		}

		envContent := requireBuilderBytes(t, ax.ReadFile(envLogPath))

		envLines := core.Split(core.Trim(string(envContent)), "\n")
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

		artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, targets))
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if result := ax.Stat(artifacts[0].Path); !result.OK {
			t.Fatalf("expected file to exist: %v", artifacts[0].Path)
		}

		content := requireBuilderBytes(t, ax.ReadFile(logPath))

		args := core.Split(core.Trim(string(content)), "\n")
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

		artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, targets))
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if result := ax.Stat(artifacts[0].Path); !result.OK {
			t.Fatalf("expected file to exist: %v", artifacts[0].Path)
		}

		argsContent := requireBuilderBytes(t, ax.ReadFile(argsLogPath))

		args := core.Split(core.Trim(string(argsContent)), "\n")
		if stdlibAssertEmpty(args) {
			t.Fatal("expected non-empty")
		}
		if !stdlibAssertContains(args, "-ldflags") {
			t.Fatalf("expected %v to contain %v", args, "-ldflags")
		}
		if !stdlibAssertContains(args, "-X main.version=v1.2.3") {
			t.Fatalf("expected %v to contain %v", args, "-X main.version=v1.2.3")
		}

		envContent := requireBuilderBytes(t, ax.ReadFile(envLogPath))

		envLines := core.Split(core.Trim(string(envContent)), "\n")
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

		artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, targets))
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if result := ax.Stat(artifacts[0].Path); !result.OK {
			t.Fatalf("expected file to exist: %v", artifacts[0].Path)
		}

		content := requireBuilderBytes(t, ax.ReadFile(logPath))

		args := core.Split(core.Trim(string(content)), "\n")
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
		t.Setenv("PATH", goDir+string(core.PathListSeparator)+"/usr/bin"+string(core.PathListSeparator)+"/bin")

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

		artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, targets))
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if result := ax.Stat(artifacts[0].Path); !result.OK {
			t.Fatalf("expected file to exist: %v", artifacts[0].Path)
		}

		content := requireBuilderBytes(t, ax.ReadFile(logPath))

		args := core.Split(core.Trim(string(content)), "\n")
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
		if result := ax.MkdirAll(ax.Join(projectDir, "cmd", "myapp"), 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		if result := ax.WriteFile(ax.Join(projectDir, "cmd", "myapp", "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
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

		artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, targets))
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}

		content := requireBuilderBytes(t, ax.ReadFile(logPath))

		args := core.Split(core.Trim(string(content)), "\n")
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

		artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, targets))
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if result := ax.Stat(artifacts[0].Path); !result.OK {
			t.Fatalf("expected file to exist: %v", artifacts[0].Path)
		}
		if !io.Local.IsDir(outputDir) {
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

		artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, targets))
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}

		expectedDir := ax.Join(projectDir, "dist")
		if !io.Local.IsDir(expectedDir) {
			t.Fatalf("expected directory to exist: %v", expectedDir)
		}
		if !stdlibAssertContains(artifacts[0].Path, expectedDir) {
			t.Fatalf("expected %v to contain %v", artifacts[0].Path, expectedDir)
		}
		if result := ax.Stat(artifacts[0].Path); !result.OK {
			t.Fatalf("expected file to exist: %v", artifacts[0].Path)
		}

	})
}

func TestGo_GoBuilderBuildBad(t *testing.T) {
	binDir := t.TempDir()
	setupFakeBuildToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	t.Run("returns error for nil config", func(t *testing.T) {
		builder := NewGoBuilder()

		result := builder.Build(context.Background(), nil, []build.Target{{OS: "linux", Arch: "amd64"}})
		if result.OK {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(result.Error(), "config is nil") {
			t.Fatalf("expected %v to contain %v", result.Error(), "config is nil")
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

		artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, []build.Target{}))
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if !stdlibAssertEqual(runtime.GOOS, artifacts[0].OS) {
			t.Fatalf("want %v, got %v", runtime.GOOS, artifacts[0].OS)
		}
		if !stdlibAssertEqual(runtime.GOARCH, artifacts[0].Arch) {
			t.Fatalf("want %v, got %v", runtime.GOARCH, artifacts[0].Arch)
		}
		if result := ax.Stat(artifacts[0].Path); !result.OK {
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

		result := builder.Build(context.Background(), cfg, targets)
		if result.OK {
			t.Fatal("expected error")
		}

	})

	t.Run("returns error for invalid Go code", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping integration test in short mode")
		}

		dir := t.TempDir()

		// Create go.mod
		if result := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module test\n\ngo 1.21"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		if result := ax.WriteFile(ax.Join(dir, "main.go"), []byte("this is not valid go code"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
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

		result := builder.Build(context.Background(), cfg, targets)
		if result.OK {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(result.Error(), "go build failed") {
			t.Fatalf("expected %v to contain %v", result.Error(), "go build failed")
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

		result := builder.Build(context.Background(), cfg, targets)
		if result.OK {
			t.Fatal("expected error")
		}
		if stdlibAssertEmpty(result.Error()) {
			t.Fatal("expected non-empty error")
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

		result := builder.Build(ctx, cfg, targets)
		if result.OK {
			t.Fatal("expected error")
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

		result := builder.Build(context.Background(), cfg, []build.Target{{OS: runtime.GOOS, Arch: runtime.GOARCH}})
		if result.OK {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(result.Error(), "unsupported characters") {
			t.Fatalf("expected %v to contain %v", result.Error(), "unsupported characters")
		}

	})
}

func TestGo_GoBuilderResolveGarbleCliGood(t *testing.T) {
	t.Run("returns an explicit fallback path when it exists", func(t *testing.T) {
		builder := NewGoBuilder()
		garblePath := ax.Join(t.TempDir(), "garble")
		if result := ax.WriteFile(garblePath, []byte("#!/bin/sh\n"), 0o755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		t.Setenv("PATH", t.TempDir())

		command := requireCPPString(t, builder.resolveGarbleCli(garblePath))
		if !stdlibAssertEqual(garblePath, command) {
			t.Fatalf("want %v, got %v", garblePath, command)
		}

	})
}

func TestGo_GoBuilderResolveGarbleCliBad(t *testing.T) {
	t.Run("returns an error when garble cannot be resolved", func(t *testing.T) {
		builder := NewGoBuilder()
		t.Setenv("PATH", t.TempDir())

		result := builder.resolveGarbleCli(ax.Join(t.TempDir(), "missing-garble"))
		if result.OK {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(result.Error(), "garble CLI not found") {
			t.Fatalf("expected %v to contain %v", result.Error(), "garble CLI not found")
		}

	})
}

func TestGo_GarbleInstallPathsUgly(t *testing.T) {
	gobin := ax.Join(t.TempDir(), "gobin")
	gopathOne := ax.Join(t.TempDir(), "gopath-one")
	gopathTwo := ax.Join(t.TempDir(), "gopath-two")

	t.Setenv("GOBIN", gobin)
	t.Setenv("GOPATH", gopathOne+string(core.PathListSeparator)+" "+string(core.PathListSeparator)+gopathTwo)

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

func TestGo_GoBuilderInterfaceGood(t *testing.T) {
	builder := NewGoBuilder()
	var _ build.Builder = builder
	if !stdlibAssertEqual("go", builder.Name()) {
		t.Fatalf("want %v, got %v", "go", builder.Name())
	}
	detected := requireCPPBool(t, builder.Detect(nil, t.TempDir()))
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

// --- v0.9.0 generated compliance triplets ---
func TestGo_NewGoBuilder_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewGoBuilder()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestGo_NewGoBuilder_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewGoBuilder()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestGo_NewGoBuilder_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewGoBuilder()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestGo_GoBuilder_Name_Good(t *core.T) {
	subject := &GoBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestGo_GoBuilder_Name_Bad(t *core.T) {
	subject := &GoBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestGo_GoBuilder_Name_Ugly(t *core.T) {
	subject := &GoBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestGo_GoBuilder_Detect_Good(t *core.T) {
	subject := &GoBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Detect(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestGo_GoBuilder_Detect_Bad(t *core.T) {
	subject := &GoBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Detect(io.NewMemoryMedium(), "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestGo_GoBuilder_Detect_Ugly(t *core.T) {
	subject := &GoBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Detect(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestGo_GoBuilder_Build_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &GoBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Build(ctx, nil, nil)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestGo_GoBuilder_Build_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &GoBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Build(ctx, nil, nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestGo_GoBuilder_Build_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &GoBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Build(ctx, nil, nil)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
