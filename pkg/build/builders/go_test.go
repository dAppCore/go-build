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

// setupGoTestProject creates a minimal Go project for testing.
func setupGoTestProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create a minimal go.mod
	goMod := `module testproject

go 1.21
`
	err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	// Create a minimal main.go
	mainGo := `package main

func main() {
	println("hello")
}
`
	err = ax.WriteFile(ax.Join(dir, "main.go"), []byte(mainGo), 0644)
	require.NoError(t, err)

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
	require.NoError(t, err)

	garbleScript := `#!/bin/sh
set -eu

log_file="${GARBLE_LOG_FILE:-}"
if [ -n "$log_file" ]; then
	printf '%s\n' "$@" > "$log_file"
fi

exec go "$@"
`

	err = ax.WriteFile(ax.Join(binDir, "garble"), []byte(garbleScript), 0o755)
	require.NoError(t, err)
}

func TestGo_GoBuilderName_Good(t *testing.T) {
	builder := NewGoBuilder()
	assert.Equal(t, "go", builder.Name())
}

func TestGo_GoBuilderDetect_Good(t *testing.T) {
	fs := io.Local
	t.Run("detects Go project with go.mod", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module test"), 0644)
		require.NoError(t, err)

		builder := NewGoBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("detects Wails project", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "wails.json"), []byte("{}"), 0644)
		require.NoError(t, err)

		builder := NewGoBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("returns false for non-Go project", func(t *testing.T) {
		dir := t.TempDir()
		// Create a Node.js project instead
		err := ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0644)
		require.NoError(t, err)

		builder := NewGoBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		dir := t.TempDir()

		builder := NewGoBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
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
		require.NoError(t, err)
		require.Len(t, artifacts, 1)

		// Verify artifact properties
		artifact := artifacts[0]
		assert.Equal(t, runtime.GOOS, artifact.OS)
		assert.Equal(t, runtime.GOARCH, artifact.Arch)

		// Verify binary was created
		assert.FileExists(t, artifact.Path)

		// Verify the path is in the expected location
		expectedName := "testbinary"
		if runtime.GOOS == "windows" {
			expectedName += ".exe"
		}
		assert.Contains(t, artifact.Path, expectedName)
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
		require.NoError(t, err)
		require.Len(t, artifacts, 2)

		// Verify both artifacts were created
		for i, artifact := range artifacts {
			assert.Equal(t, targets[i].OS, artifact.OS)
			assert.Equal(t, targets[i].Arch, artifact.Arch)
			assert.FileExists(t, artifact.Path)
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
		require.NoError(t, err)
		require.Len(t, artifacts, 1)

		// Verify .exe extension
		assert.True(t, ax.Ext(artifacts[0].Path) == ".exe")
		assert.FileExists(t, artifacts[0].Path)
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
		require.NoError(t, err)
		require.Len(t, artifacts, 1)

		// Binary should use the project directory base name
		baseName := ax.Base(projectDir)
		if runtime.GOOS == "windows" {
			baseName += ".exe"
		}
		assert.Contains(t, artifacts[0].Path, baseName)
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
		require.NoError(t, err)
		require.Len(t, artifacts, 1)

		expectedName := "example-binary"
		if runtime.GOOS == "windows" {
			expectedName += ".exe"
		}
		assert.Contains(t, artifacts[0].Path, expectedName)
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
		require.NoError(t, err)
		require.Len(t, artifacts, 1)
		assert.FileExists(t, artifacts[0].Path)
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
		require.NoError(t, err)
		require.Len(t, artifacts, 1)
		assert.FileExists(t, artifacts[0].Path)

		argsContent, err := ax.ReadFile(argsLogPath)
		require.NoError(t, err)
		args := strings.Split(strings.TrimSpace(string(argsContent)), "\n")
		require.NotEmpty(t, args)
		assert.Equal(t, "build", args[0])
		assert.Contains(t, args, "-race")
		assert.NotContains(t, args, "-trimpath")

		envContent, err := ax.ReadFile(envLogPath)
		require.NoError(t, err)
		envLines := strings.Split(strings.TrimSpace(string(envContent)), "\n")
		assert.Contains(t, envLines, "BAR=baz")
		assert.Contains(t, envLines, "FOO=bar")
		assert.Contains(t, envLines, "TARGET_OS="+runtime.GOOS)
		assert.Contains(t, envLines, "TARGET_ARCH="+runtime.GOARCH)
		assert.Contains(t, envLines, "OUTPUT_DIR="+outputDir)
		assert.Contains(t, envLines, "TARGET_DIR="+ax.Join(outputDir, runtime.GOOS+"_"+runtime.GOARCH))
		assert.Contains(t, envLines, "GOOS="+runtime.GOOS)
		assert.Contains(t, envLines, "GOARCH="+runtime.GOARCH)
		assert.Contains(t, envLines, "NAME=envflags")
		assert.Contains(t, envLines, "VERSION=v1.2.3")
		assert.Contains(t, envLines, "CGO_ENABLED=0")
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
		require.NoError(t, err)
		require.Len(t, artifacts, 1)
		assert.FileExists(t, artifacts[0].Path)

		envContent, err := ax.ReadFile(envLogPath)
		require.NoError(t, err)

		envLines := strings.Split(strings.TrimSpace(string(envContent)), "\n")
		assert.Contains(t, envLines, "GOCACHE="+ax.Join(outputDir, "cache", "go-build"))
		assert.Contains(t, envLines, "GOMODCACHE="+ax.Join(outputDir, "cache", "go-mod"))
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
		require.NoError(t, err)
		require.Len(t, artifacts, 1)
		assert.FileExists(t, artifacts[0].Path)

		content, err := ax.ReadFile(logPath)
		require.NoError(t, err)

		args := strings.Split(strings.TrimSpace(string(content)), "\n")
		require.NotEmpty(t, args)
		assert.Equal(t, "build", args[0])
		assert.Contains(t, args, "-tags")
		assert.Contains(t, args, "webkit2_41,integration")
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
		require.NoError(t, err)
		require.Len(t, artifacts, 1)
		assert.FileExists(t, artifacts[0].Path)

		argsContent, err := ax.ReadFile(argsLogPath)
		require.NoError(t, err)

		args := strings.Split(strings.TrimSpace(string(argsContent)), "\n")
		require.NotEmpty(t, args)
		assert.Contains(t, args, "-ldflags")
		assert.Contains(t, args, "-X main.version=v1.2.3")

		envContent, err := ax.ReadFile(envLogPath)
		require.NoError(t, err)

		envLines := strings.Split(strings.TrimSpace(string(envContent)), "\n")
		assert.Contains(t, envLines, "VERSION=v1.2.3")
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
		require.NoError(t, err)
		require.Len(t, artifacts, 1)
		assert.FileExists(t, artifacts[0].Path)

		content, err := ax.ReadFile(logPath)
		require.NoError(t, err)

		args := strings.Split(strings.TrimSpace(string(content)), "\n")
		require.NotEmpty(t, args)
		assert.Equal(t, "build", args[0])
		assert.Contains(t, args, "-trimpath")
		assert.Contains(t, args, "-o")
		assert.Contains(t, args, ".")
	})

	t.Run("builds the configured main package path", func(t *testing.T) {
		projectDir := setupGoTestProject(t)
		err := ax.MkdirAll(ax.Join(projectDir, "cmd", "myapp"), 0755)
		require.NoError(t, err)
		err = ax.WriteFile(ax.Join(projectDir, "cmd", "myapp", "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644)
		require.NoError(t, err)

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
		require.NoError(t, err)
		require.Len(t, artifacts, 1)

		content, err := ax.ReadFile(logPath)
		require.NoError(t, err)

		args := strings.Split(strings.TrimSpace(string(content)), "\n")
		require.NotEmpty(t, args)
		assert.Contains(t, args, "./cmd/myapp")
		assert.NotContains(t, args, ".")
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
		require.NoError(t, err)
		require.Len(t, artifacts, 1)
		assert.FileExists(t, artifacts[0].Path)
		assert.DirExists(t, outputDir)
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
		require.NoError(t, err)
		require.Len(t, artifacts, 1)

		expectedDir := ax.Join(projectDir, "dist")
		assert.DirExists(t, expectedDir)
		assert.Contains(t, artifacts[0].Path, expectedDir)
		assert.FileExists(t, artifacts[0].Path)
	})
}

func TestGo_GoBuilderBuild_Bad(t *testing.T) {
	binDir := t.TempDir()
	setupFakeBuildToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	t.Run("returns error for nil config", func(t *testing.T) {
		builder := NewGoBuilder()

		artifacts, err := builder.Build(context.Background(), nil, []build.Target{{OS: "linux", Arch: "amd64"}})
		assert.Error(t, err)
		assert.Nil(t, artifacts)
		assert.Contains(t, err.Error(), "config is nil")
	})

	t.Run("returns error for empty targets", func(t *testing.T) {
		projectDir := setupGoTestProject(t)

		builder := NewGoBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  t.TempDir(),
			Name:       "test",
		}

		artifacts, err := builder.Build(context.Background(), cfg, []build.Target{})
		assert.Error(t, err)
		assert.Nil(t, artifacts)
		assert.Contains(t, err.Error(), "no targets specified")
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
		assert.Error(t, err)
		assert.Empty(t, artifacts)
	})

	t.Run("returns error for invalid Go code", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping integration test in short mode")
		}

		dir := t.TempDir()

		// Create go.mod
		err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module test\n\ngo 1.21"), 0644)
		require.NoError(t, err)

		// Create invalid Go code
		err = ax.WriteFile(ax.Join(dir, "main.go"), []byte("this is not valid go code"), 0644)
		require.NoError(t, err)

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
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "go build failed")
		assert.Empty(t, artifacts)
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
		// Should return error for the failed build
		assert.Error(t, err)
		// Should have the successful artifact
		assert.Len(t, artifacts, 1)
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
		assert.Error(t, err)
		assert.Empty(t, artifacts)
	})
}

func TestGo_GoBuilderInterface_Good(t *testing.T) {
	// Verify GoBuilder implements Builder interface
	var _ build.Builder = (*GoBuilder)(nil)
	var _ build.Builder = NewGoBuilder()
}
