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

// setupWailsTestProject creates a minimal Wails project structure for testing.
func setupWailsTestProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create wails.json
	wailsJSON := `{
  "name": "testapp",
  "outputfilename": "testapp"
}`
	err := ax.WriteFile(ax.Join(dir, "wails.json"), []byte(wailsJSON), 0o644)
	require.NoError(t, err)

	// Create a minimal go.mod
	goMod := `module testapp

go 1.21

require github.com/wailsapp/wails/v3 v3.0.0
`
	err = ax.WriteFile(ax.Join(dir, "go.mod"), []byte(goMod), 0o644)
	require.NoError(t, err)

	// Create a minimal main.go
	mainGo := `package main

func main() {
	println("hello wails")
}
`
	err = ax.WriteFile(ax.Join(dir, "main.go"), []byte(mainGo), 0o644)
	require.NoError(t, err)

	// Create a minimal Taskfile.yml
	taskfile := `version: '3'
tasks:
  build:
    cmds:
      - mkdir -p {{.OUTPUT_DIR}}/{{.GOOS}}_{{.GOARCH}}
      - touch {{.OUTPUT_DIR}}/{{.GOOS}}_{{.GOARCH}}/testapp
`
	err = ax.WriteFile(ax.Join(dir, "Taskfile.yml"), []byte(taskfile), 0o644)
	require.NoError(t, err)

	return dir
}

// setupWailsV2TestProject creates a Wails v2 project structure.
func setupWailsV2TestProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// wails.json
	err := ax.WriteFile(ax.Join(dir, "wails.json"), []byte("{}"), 0o644)
	require.NoError(t, err)

	// go.mod with v2
	goMod := `module testapp
go 1.21
require github.com/wailsapp/wails/v2 v2.8.0
`
	err = ax.WriteFile(ax.Join(dir, "go.mod"), []byte(goMod), 0o644)
	require.NoError(t, err)

	return dir
}

func setupFakeWailsToolchain(t *testing.T, binDir string) {
	t.Helper()

	wailsScript := `#!/bin/sh
set -eu

log_file="${WAILS_BUILD_LOG_FILE:-}"
if [ -n "$log_file" ]; then
	printf '%s\n' "$@" > "$log_file"
	if [ -n "${GOCACHE:-}" ]; then
		printf '%s\n' "GOCACHE=${GOCACHE}" >> "$log_file"
	fi
	if [ -n "${GOMODCACHE:-}" ]; then
		printf '%s\n' "GOMODCACHE=${GOMODCACHE}" >> "$log_file"
	fi
fi

sequence_file="${BUILD_SEQUENCE_FILE:-}"
if [ -n "$sequence_file" ]; then
	printf '%s\n' "wails" >> "$sequence_file"
	printf '%s\n' "$@" >> "$sequence_file"
	if [ -n "${CUSTOM_ENV:-}" ]; then
		printf '%s\n' "CUSTOM_ENV=${CUSTOM_ENV}" >> "$sequence_file"
	fi
fi

output_dir="build/bin"
binary_name="testapp"
mkdir -p "$output_dir"
platform=""
use_nsis=0

while [ "$#" -gt 0 ]; do
	case "$1" in
		-platform)
			shift
			platform="${1:-}"
			;;
		-nsis)
			use_nsis=1
			;;
	esac
	shift || true
done

target_os="${platform%%/*}"

case "$target_os" in
	windows)
		if [ "$use_nsis" -eq 1 ]; then
			printf 'fake wails installer\n' > "$output_dir/${binary_name}-installer.exe"
			chmod +x "$output_dir/${binary_name}-installer.exe"
		else
			printf 'fake wails binary\n' > "$output_dir/${binary_name}.exe"
			chmod +x "$output_dir/${binary_name}.exe"
		fi
		;;
	darwin)
		mkdir -p "$output_dir/${binary_name}.app/Contents/MacOS"
		printf 'fake wails binary\n' > "$output_dir/${binary_name}.app/Contents/MacOS/${binary_name}"
		chmod +x "$output_dir/${binary_name}.app/Contents/MacOS/${binary_name}"
		;;
	*)
		printf 'fake wails binary\n' > "$output_dir/$binary_name"
		chmod +x "$output_dir/$binary_name"
		;;
esac
`

	err := ax.WriteFile(ax.Join(binDir, "wails"), []byte(wailsScript), 0o755)
	require.NoError(t, err)
}

func setupFakeWails3Toolchain(t *testing.T, binDir string) {
	t.Helper()

	wails3Script := `#!/bin/sh
set -eu

log_file="${WAILS_BUILD_LOG_FILE:-}"
if [ -n "$log_file" ]; then
	printf '%s\n' "$@" > "$log_file"
	printf '%s\n' "GOFLAGS=${GOFLAGS:-}" >> "$log_file"
	if [ -n "${GOCACHE:-}" ]; then
		printf '%s\n' "GOCACHE=${GOCACHE}" >> "$log_file"
	fi
	if [ -n "${GOMODCACHE:-}" ]; then
		printf '%s\n' "GOMODCACHE=${GOMODCACHE}" >> "$log_file"
	fi
fi

sequence_file="${BUILD_SEQUENCE_FILE:-}"
if [ -n "$sequence_file" ]; then
	printf '%s\n' "wails3" >> "$sequence_file"
	printf '%s\n' "$@" >> "$sequence_file"
	if [ -n "${GOFLAGS:-}" ]; then
		printf '%s\n' "GOFLAGS=${GOFLAGS}" >> "$sequence_file"
	fi
fi

verb="${1:-build}"
shift || true

goos=""
goarch=""
for arg in "$@"; do
	case "$arg" in
		GOOS=*) goos="${arg#GOOS=}" ;;
		GOARCH=*) goarch="${arg#GOARCH=}" ;;
	esac
done

	name="${NAME:-testapp}"
	if [ "$verb" = "package" ] && [ "$goos" = "windows" ]; then
		mkdir -p "build/windows/nsis"
		printf 'fake wails3 installer\n' > "build/windows/nsis/${name}-installer.exe"
		chmod +x "build/windows/nsis/${name}-installer.exe"
		exit 0
	fi

	mkdir -p "bin"
	if [ "$goos" = "windows" ]; then
		name="${name}.exe"
	fi
	printf 'fake wails3 binary\n' > "bin/${name}"
	chmod +x "bin/${name}"
`

	require.NoError(t, ax.WriteFile(ax.Join(binDir, "wails3"), []byte(wails3Script), 0o755))
}

func setupFakeWails3GoBuildToolchain(t *testing.T, binDir string) {
	t.Helper()

	wails3Script := `#!/bin/sh
set -eu

name="${NAME:-testapp}"
mkdir -p "bin"
go build -o "bin/${name}" .
`
	require.NoError(t, ax.WriteFile(ax.Join(binDir, "wails3"), []byte(wails3Script), 0o755))

	garbleScript := `#!/bin/sh
set -eu

log_file="${GARBLE_LOG_FILE:-}"
if [ -n "$log_file" ]; then
	printf '%s\n' "$@" > "$log_file"
fi

output=""
while [ "$#" -gt 0 ]; do
	case "$1" in
		-o)
			shift
			output="${1:-}"
			;;
	esac
	shift || true
done

if [ -z "$output" ]; then
	echo "missing -o output path" >&2
	exit 1
fi

mkdir -p "$(dirname "$output")"
printf 'fake garbled binary\n' > "$output"
chmod +x "$output"
`
	require.NoError(t, ax.WriteFile(ax.Join(binDir, "garble"), []byte(garbleScript), 0o755))
}

func setupFakeFrontendCommand(t *testing.T, binDir, name string) {
	t.Helper()

	script := strings.ReplaceAll(`#!/bin/sh
set -eu

sequence_file="${BUILD_SEQUENCE_FILE:-}"
if [ -n "$sequence_file" ]; then
	printf '%s\n' "__NAME__" >> "$sequence_file"
	printf '%s\n' "$@" >> "$sequence_file"
	if [ -n "${CUSTOM_ENV:-}" ]; then
		printf '%s\n' "CUSTOM_ENV=${CUSTOM_ENV}" >> "$sequence_file"
	fi
fi
`, "__NAME__", name)

	require.NoError(t, ax.WriteFile(ax.Join(binDir, name), []byte(script), 0o755))
}

func TestWails_WailsBuilderBuildTaskfile_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Check if task is available
	if _, err := ax.LookPath("task"); err != nil {
		t.Skip("task not installed, skipping test")
	}

	t.Run("delegates to Taskfile if present", func(t *testing.T) {
		fs := io.Local
		projectDir := setupWailsTestProject(t)
		outputDir := t.TempDir()

		// Create a Taskfile that just touches a file
		taskfile := `version: '3'
tasks:
  build:
    cmds:
      - mkdir -p {{.OUTPUT_DIR}}/{{.GOOS}}_{{.GOARCH}}
      - touch {{.OUTPUT_DIR}}/{{.GOOS}}_{{.GOARCH}}/testapp
`
		err := ax.WriteFile(ax.Join(projectDir, "Taskfile.yml"), []byte(taskfile), 0o644)
		require.NoError(t, err)

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         fs,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
			Name:       "testapp",
		}
		targets := []build.Target{
			{OS: runtime.GOOS, Arch: runtime.GOARCH},
		}

		artifacts, err := builder.Build(context.Background(), cfg, targets)
		require.NoError(t, err)
		assert.NotEmpty(t, artifacts)
	})

	t.Run("passes Wails v3 build vars through Taskfile builds", func(t *testing.T) {
		projectDir := setupWailsTestProject(t)
		outputDir := t.TempDir()
		binDir := t.TempDir()
		logPath := ax.Join(t.TempDir(), "task.env")
		taskPath := ax.Join(binDir, "task")

		script := `#!/bin/sh
set -eu

env | sort > "${TASK_BUILD_LOG_FILE}"

name="${NAME:-testapp}"
if [ "${GOOS:-}" = "windows" ]; then
	name="${name}.exe"
fi

mkdir -p "${OUTPUT_DIR}/${GOOS}_${GOARCH}"
printf 'taskfile build\n' > "${OUTPUT_DIR}/${GOOS}_${GOARCH}/${name}"
chmod +x "${OUTPUT_DIR}/${GOOS}_${GOARCH}/${name}"
`
		require.NoError(t, ax.WriteFile(taskPath, []byte(script), 0o755))

		t.Setenv("TASK_BUILD_LOG_FILE", logPath)
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
			Name:       "testapp",
			Version:    "v1.2.3",
			BuildTags:  []string{"integration"},
			LDFlags:    []string{"-s", "-w"},
			WebView2:   "download",
		}

		artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "windows", Arch: "amd64"}})
		require.NoError(t, err)
		require.Len(t, artifacts, 1)
		assert.FileExists(t, artifacts[0].Path)

		content, err := ax.ReadFile(logPath)
		require.NoError(t, err)

		assert.Contains(t, string(content), "GOOS=windows")
		assert.Contains(t, string(content), "GOARCH=amd64")
		assert.Contains(t, string(content), "CGO_ENABLED=1")
		assert.Contains(t, string(content), "GOFLAGS=-trimpath -tags=integration -ldflags=-s -w -X main.version=v1.2.3")
		assert.Contains(t, string(content), "EXTRA_TAGS=integration")
		assert.Contains(t, string(content), "WEBVIEW2_MODE=download")
		assert.Contains(t, string(content), `BUILD_FLAGS=-tags production,integration -trimpath -buildvcs=false -ldflags="-s -w -H windowsgui -X main.version=v1.2.3"`)
	})

	t.Run("uses the garble shim for Wails v3 Taskfile builds", func(t *testing.T) {
		projectDir := setupWailsTestProject(t)
		binDir := t.TempDir()
		logPath := ax.Join(t.TempDir(), "garble.log")
		taskPath := ax.Join(binDir, "task")

		script := `#!/bin/sh
set -eu

name="${NAME:-testapp}"
mkdir -p "${OUTPUT_DIR}/${GOOS}_${GOARCH}"
go build -o "${OUTPUT_DIR}/${GOOS}_${GOARCH}/${name}" .
`
		require.NoError(t, ax.WriteFile(taskPath, []byte(script), 0o755))

		setupFakeWails3GoBuildToolchain(t, binDir)
		t.Setenv("GARBLE_LOG_FILE", logPath)
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  t.TempDir(),
			Name:       "testapp",
			Obfuscate:  true,
		}

		artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
		require.NoError(t, err)
		require.Len(t, artifacts, 1)
		assert.FileExists(t, artifacts[0].Path)

		content, err := ax.ReadFile(logPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "build")
		assert.Contains(t, string(content), "-o")
	})
}

func TestWails_WailsBuilderName_Good(t *testing.T) {
	builder := NewWailsBuilder()
	assert.Equal(t, "wails", builder.Name())
}

func TestWails_WailsBuilderBuildV3Config_Good(t *testing.T) {
	builder := NewWailsBuilder()
	cfg := &build.Config{
		CGO:   false,
		Name:  "testapp",
		Flags: []string{"-trimpath"},
		LDFlags: []string{
			"-s",
			"-w",
		},
	}

	v3Config := builder.buildV3Config(cfg)

	require.NotNil(t, v3Config)
	assert.False(t, cfg.CGO)
	assert.True(t, v3Config.CGO)
	assert.Equal(t, cfg.Name, v3Config.Name)
	assert.Equal(t, cfg.Flags, v3Config.Flags)
	assert.Equal(t, cfg.LDFlags, v3Config.LDFlags)
}

func TestWails_WailsBuilderResolveFrontendDir_Good(t *testing.T) {
	builder := NewWailsBuilder()
	fs := io.Local

	t.Run("finds nested package.json frontends", func(t *testing.T) {
		projectDir := t.TempDir()
		frontendDir := ax.Join(projectDir, "apps", "web")
		require.NoError(t, ax.MkdirAll(frontendDir, 0o755))
		require.NoError(t, ax.WriteFile(ax.Join(frontendDir, "package.json"), []byte("{}"), 0o644))

		got := builder.resolveFrontendDir(fs, projectDir)
		assert.Equal(t, frontendDir, got)
	})

	t.Run("finds nested deno.json frontends", func(t *testing.T) {
		projectDir := t.TempDir()
		frontendDir := ax.Join(projectDir, "packages", "site")
		require.NoError(t, ax.MkdirAll(frontendDir, 0o755))
		require.NoError(t, ax.WriteFile(ax.Join(frontendDir, "deno.json"), []byte("{}"), 0o644))

		got := builder.resolveFrontendDir(fs, projectDir)
		assert.Equal(t, frontendDir, got)
	})

	t.Run("ignores frontends deeper than depth 2", func(t *testing.T) {
		projectDir := t.TempDir()
		frontendDir := ax.Join(projectDir, "apps", "marketing", "web")
		require.NoError(t, ax.MkdirAll(frontendDir, 0o755))
		require.NoError(t, ax.WriteFile(ax.Join(frontendDir, "package.json"), []byte("{}"), 0o644))

		got := builder.resolveFrontendDir(fs, projectDir)
		assert.Empty(t, got)
	})

	t.Run("falls back to frontend directory when DENO_ENABLE is set", func(t *testing.T) {
		t.Setenv("DENO_ENABLE", "true")

		projectDir := t.TempDir()
		frontendDir := ax.Join(projectDir, "frontend")
		require.NoError(t, ax.MkdirAll(frontendDir, 0o755))

		got := builder.resolveFrontendDir(fs, projectDir)
		assert.Equal(t, frontendDir, got)
	})
}

func TestWails_WailsBuilderBuildV2_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeWailsToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	builder := NewWailsBuilder()

	t.Run("builds v2 project", func(t *testing.T) {
		fs := io.Local
		projectDir := setupWailsV2TestProject(t)
		outputDir := t.TempDir()

		cfg := &build.Config{
			FS:         fs,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
			Name:       "testapp",
		}
		targets := []build.Target{
			{OS: runtime.GOOS, Arch: runtime.GOARCH},
		}

		artifacts, err := builder.Build(context.Background(), cfg, targets)
		require.NoError(t, err)
		require.Len(t, artifacts, 1)
		assert.True(t, io.Local.Exists(artifacts[0].Path))
	})
}

func TestWails_copyBuildArtifact_PreservesMode_Good(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable mode bits are not portable on Windows")
	}

	sourceDir := t.TempDir()
	sourcePath := ax.Join(sourceDir, "testapp")
	require.NoError(t, ax.WriteFile(sourcePath, []byte("fake wails binary\n"), 0o755))

	destDir := t.TempDir()
	destPath := ax.Join(destDir, "testapp")

	require.NoError(t, copyBuildArtifact(io.Local, sourcePath, destPath))

	info, err := ax.Stat(destPath)
	require.NoError(t, err)
	assert.NotZero(t, info.Mode()&0o111)
}

func TestWails_WailsBuilderBuildV2Flags_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeWailsToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectDir := setupWailsV2TestProject(t)
	outputDir := t.TempDir()
	logDir := t.TempDir()
	logPath := ax.Join(logDir, "wails.log")
	t.Setenv("WAILS_BUILD_LOG_FILE", logPath)

	goCacheDir := ax.Join(outputDir, "cache", "go-build")
	goModCacheDir := ax.Join(outputDir, "cache", "go-mod")

	builder := NewWailsBuilder()
	t.Run("includes Windows-only packaging flags for Windows targets", func(t *testing.T) {
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
			Name:       "testapp",
			Version:    "v1.2.3",
			BuildTags:  []string{"integration", "webkit2_41"},
			LDFlags:    []string{"-s", "-w"},
			Obfuscate:  true,
			NSIS:       true,
			WebView2:   "embed",
			Cache: build.CacheConfig{
				Enabled: true,
				Paths: []string{
					goCacheDir,
					goModCacheDir,
				},
			},
		}

		artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "windows", Arch: "amd64"}})
		require.NoError(t, err)
		require.Len(t, artifacts, 1)

		content, err := ax.ReadFile(logPath)
		require.NoError(t, err)

		args := strings.Split(strings.TrimSpace(string(content)), "\n")
		require.NotEmpty(t, args)
		assert.Equal(t, "build", args[0])
		assert.Contains(t, args, "-tags")
		assert.Contains(t, args, "integration,webkit2_41")
		assert.Contains(t, args, "-ldflags")
		assert.Contains(t, args, "-s -w -X main.version=v1.2.3")
		assert.Contains(t, args, "-obfuscated")
		assert.Contains(t, args, "-nsis")
		assert.Contains(t, args, "-webview2")
		assert.Contains(t, args, "embed")
		assert.Contains(t, args, "GOCACHE="+goCacheDir)
		assert.Contains(t, args, "GOMODCACHE="+goModCacheDir)
	})

	t.Run("omits Windows-only packaging flags for non-Windows targets", func(t *testing.T) {
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
			Name:       "testapp",
			Version:    "v1.2.3",
			BuildTags:  []string{"integration", "webkit2_41"},
			LDFlags:    []string{"-s", "-w"},
			Obfuscate:  true,
			NSIS:       true,
			WebView2:   "embed",
		}

		artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
		require.NoError(t, err)
		require.Len(t, artifacts, 1)

		content, err := ax.ReadFile(logPath)
		require.NoError(t, err)

		args := strings.Split(strings.TrimSpace(string(content)), "\n")
		require.NotEmpty(t, args)
		assert.NotContains(t, args, "-nsis")
		assert.NotContains(t, args, "-webview2")
		assert.NotContains(t, args, "embed")
	})
}

func TestWails_WailsBuilderBuildV2Flags_Bad(t *testing.T) {
	err := validateWebView2Mode("invalid")
	require.Error(t, err)
	assert.ErrorContains(t, err, "webview2 must be one of")
}

func TestWails_WailsBuilderPreBuild_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("uses deno when deno manifest exists", func(t *testing.T) {
		binDir := t.TempDir()
		setupFakeFrontendCommand(t, binDir, "deno")
		setupFakeFrontendCommand(t, binDir, "npm")
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		projectDir := setupWailsTestProject(t)
		frontendDir := ax.Join(projectDir, "frontend")
		require.NoError(t, ax.MkdirAll(frontendDir, 0o755))
		require.NoError(t, ax.WriteFile(ax.Join(frontendDir, "deno.json"), []byte(`{}`), 0o644))
		require.NoError(t, ax.WriteFile(ax.Join(frontendDir, "package.json"), []byte(`{}`), 0o644))

		logPath := ax.Join(t.TempDir(), "frontend.log")
		t.Setenv("BUILD_SEQUENCE_FILE", logPath)

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
		}

		require.NoError(t, builder.PreBuild(context.Background(), cfg))

		content, err := ax.ReadFile(logPath)
		require.NoError(t, err)

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		require.Len(t, lines, 3)
		assert.Equal(t, "deno", lines[0])
		assert.Equal(t, "task", lines[1])
		assert.Equal(t, "build", lines[2])
	})

	t.Run("uses configured deno build command when provided", func(t *testing.T) {
		binDir := t.TempDir()
		setupFakeFrontendCommand(t, binDir, "deno")
		setupFakeFrontendCommand(t, binDir, "deno-build")
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		projectDir := setupWailsTestProject(t)
		frontendDir := ax.Join(projectDir, "frontend")
		require.NoError(t, ax.MkdirAll(frontendDir, 0o755))
		require.NoError(t, ax.WriteFile(ax.Join(frontendDir, "deno.json"), []byte(`{}`), 0o644))

		logPath := ax.Join(t.TempDir(), "frontend-custom.log")
		t.Setenv("BUILD_SEQUENCE_FILE", logPath)

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			DenoBuild:  "deno-build --target release",
		}

		require.NoError(t, builder.PreBuild(context.Background(), cfg))

		content, err := ax.ReadFile(logPath)
		require.NoError(t, err)

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		require.Len(t, lines, 3)
		assert.Equal(t, "deno-build", lines[0])
		assert.Equal(t, "--target", lines[1])
		assert.Equal(t, "release", lines[2])
	})

	t.Run("DENO_BUILD env override wins over config", func(t *testing.T) {
		binDir := t.TempDir()
		setupFakeFrontendCommand(t, binDir, "deno")
		setupFakeFrontendCommand(t, binDir, "deno-build")
		setupFakeFrontendCommand(t, binDir, "env-deno-build")
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		t.Setenv("DENO_BUILD", "env-deno-build --env")

		projectDir := setupWailsTestProject(t)
		frontendDir := ax.Join(projectDir, "frontend")
		require.NoError(t, ax.MkdirAll(frontendDir, 0o755))
		require.NoError(t, ax.WriteFile(ax.Join(frontendDir, "deno.json"), []byte(`{}`), 0o644))

		logPath := ax.Join(t.TempDir(), "frontend-env.log")
		t.Setenv("BUILD_SEQUENCE_FILE", logPath)

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			DenoBuild:  "deno-build --config",
		}

		require.NoError(t, builder.PreBuild(context.Background(), cfg))

		content, err := ax.ReadFile(logPath)
		require.NoError(t, err)

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		require.Len(t, lines, 2)
		assert.Equal(t, "env-deno-build", lines[0])
		assert.Equal(t, "--env", lines[1])
	})

	t.Run("falls back to npm when only package.json exists", func(t *testing.T) {
		binDir := t.TempDir()
		setupFakeFrontendCommand(t, binDir, "deno")
		setupFakeFrontendCommand(t, binDir, "npm")
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		projectDir := setupWailsTestProject(t)
		require.NoError(t, ax.WriteFile(ax.Join(projectDir, "package.json"), []byte(`{}`), 0o644))

		logPath := ax.Join(t.TempDir(), "frontend.log")
		t.Setenv("BUILD_SEQUENCE_FILE", logPath)

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
		}

		require.NoError(t, builder.PreBuild(context.Background(), cfg))

		content, err := ax.ReadFile(logPath)
		require.NoError(t, err)

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		require.Len(t, lines, 3)
		assert.Equal(t, "npm", lines[0])
		assert.Equal(t, "run", lines[1])
		assert.Equal(t, "build", lines[2])
	})

	t.Run("prefers deno when DENO_ENABLE is set without a deno manifest", func(t *testing.T) {
		binDir := t.TempDir()
		setupFakeFrontendCommand(t, binDir, "deno")
		setupFakeFrontendCommand(t, binDir, "npm")
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		t.Setenv("DENO_ENABLE", "true")

		projectDir := setupWailsTestProject(t)
		require.NoError(t, ax.WriteFile(ax.Join(projectDir, "package.json"), []byte(`{}`), 0o644))

		logPath := ax.Join(t.TempDir(), "frontend-deno-enable.log")
		t.Setenv("BUILD_SEQUENCE_FILE", logPath)

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
		}

		require.NoError(t, builder.PreBuild(context.Background(), cfg))

		content, err := ax.ReadFile(logPath)
		require.NoError(t, err)

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		require.Len(t, lines, 3)
		assert.Equal(t, "deno", lines[0])
		assert.Equal(t, "task", lines[1])
		assert.Equal(t, "build", lines[2])
	})

	t.Run("uses configured deno build command without a deno manifest", func(t *testing.T) {
		binDir := t.TempDir()
		setupFakeFrontendCommand(t, binDir, "deno-build")
		setupFakeFrontendCommand(t, binDir, "npm")
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		projectDir := setupWailsTestProject(t)
		require.NoError(t, ax.WriteFile(ax.Join(projectDir, "package.json"), []byte(`{}`), 0o644))

		logPath := ax.Join(t.TempDir(), "frontend-config-deno.log")
		t.Setenv("BUILD_SEQUENCE_FILE", logPath)

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			DenoBuild:  "deno-build --target release",
		}

		require.NoError(t, builder.PreBuild(context.Background(), cfg))

		content, err := ax.ReadFile(logPath)
		require.NoError(t, err)

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		require.Len(t, lines, 3)
		assert.Equal(t, "deno-build", lines[0])
		assert.Equal(t, "--target", lines[1])
		assert.Equal(t, "release", lines[2])
	})

	t.Run("discovers nested package.json in a monorepo", func(t *testing.T) {
		binDir := t.TempDir()
		setupFakeFrontendCommand(t, binDir, "npm")
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		projectDir := setupWailsTestProject(t)
		frontendDir := ax.Join(projectDir, "apps", "web")
		require.NoError(t, ax.MkdirAll(frontendDir, 0o755))
		require.NoError(t, ax.WriteFile(ax.Join(frontendDir, "package.json"), []byte(`{}`), 0o644))

		logPath := ax.Join(t.TempDir(), "frontend.log")
		t.Setenv("BUILD_SEQUENCE_FILE", logPath)

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
		}

		require.NoError(t, builder.PreBuild(context.Background(), cfg))

		content, err := ax.ReadFile(logPath)
		require.NoError(t, err)

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		require.Len(t, lines, 3)
		assert.Equal(t, "npm", lines[0])
		assert.Equal(t, "run", lines[1])
		assert.Equal(t, "build", lines[2])
	})

	t.Run("uses bun when bun.lockb exists", func(t *testing.T) {
		binDir := t.TempDir()
		setupFakeFrontendCommand(t, binDir, "bun")
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		projectDir := setupWailsTestProject(t)
		frontendDir := ax.Join(projectDir, "frontend")
		require.NoError(t, ax.MkdirAll(frontendDir, 0o755))
		require.NoError(t, ax.WriteFile(ax.Join(frontendDir, "package.json"), []byte(`{}`), 0o644))
		require.NoError(t, ax.WriteFile(ax.Join(frontendDir, "bun.lockb"), []byte(""), 0o644))

		logPath := ax.Join(t.TempDir(), "frontend.log")
		t.Setenv("BUILD_SEQUENCE_FILE", logPath)

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
		}

		require.NoError(t, builder.PreBuild(context.Background(), cfg))

		content, err := ax.ReadFile(logPath)
		require.NoError(t, err)

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		require.Len(t, lines, 3)
		assert.Equal(t, "bun", lines[0])
		assert.Equal(t, "run", lines[1])
		assert.Equal(t, "build", lines[2])
	})

	t.Run("uses pnpm when pnpm-lock.yaml exists", func(t *testing.T) {
		binDir := t.TempDir()
		setupFakeFrontendCommand(t, binDir, "pnpm")
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		projectDir := setupWailsTestProject(t)
		frontendDir := ax.Join(projectDir, "frontend")
		require.NoError(t, ax.MkdirAll(frontendDir, 0o755))
		require.NoError(t, ax.WriteFile(ax.Join(frontendDir, "package.json"), []byte(`{}`), 0o644))
		require.NoError(t, ax.WriteFile(ax.Join(frontendDir, "pnpm-lock.yaml"), []byte(""), 0o644))

		logPath := ax.Join(t.TempDir(), "frontend.log")
		t.Setenv("BUILD_SEQUENCE_FILE", logPath)

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
		}

		require.NoError(t, builder.PreBuild(context.Background(), cfg))

		content, err := ax.ReadFile(logPath)
		require.NoError(t, err)

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		require.Len(t, lines, 3)
		assert.Equal(t, "pnpm", lines[0])
		assert.Equal(t, "run", lines[1])
		assert.Equal(t, "build", lines[2])
	})

	t.Run("uses yarn when yarn.lock exists", func(t *testing.T) {
		binDir := t.TempDir()
		setupFakeFrontendCommand(t, binDir, "yarn")
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		projectDir := setupWailsTestProject(t)
		frontendDir := ax.Join(projectDir, "frontend")
		require.NoError(t, ax.MkdirAll(frontendDir, 0o755))
		require.NoError(t, ax.WriteFile(ax.Join(frontendDir, "package.json"), []byte(`{}`), 0o644))
		require.NoError(t, ax.WriteFile(ax.Join(frontendDir, "yarn.lock"), []byte(""), 0o644))

		logPath := ax.Join(t.TempDir(), "frontend.log")
		t.Setenv("BUILD_SEQUENCE_FILE", logPath)

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
		}

		require.NoError(t, builder.PreBuild(context.Background(), cfg))

		content, err := ax.ReadFile(logPath)
		require.NoError(t, err)

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		require.Len(t, lines, 2)
		assert.Equal(t, "yarn", lines[0])
		assert.Equal(t, "build", lines[1])
	})
}

func TestWails_WailsBuilderBuildV2PreBuild_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeFrontendCommand(t, binDir, "deno")
	setupFakeFrontendCommand(t, binDir, "npm")
	setupFakeWailsToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectDir := setupWailsV2TestProject(t)
	frontendDir := ax.Join(projectDir, "frontend")
	require.NoError(t, ax.MkdirAll(frontendDir, 0o755))
	require.NoError(t, ax.WriteFile(ax.Join(frontendDir, "deno.json"), []byte(`{}`), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(frontendDir, "package.json"), []byte(`{}`), 0o644))

	outputDir := t.TempDir()
	sequencePath := ax.Join(t.TempDir(), "build-sequence.log")
	wailsLogPath := ax.Join(t.TempDir(), "wails.log")
	t.Setenv("BUILD_SEQUENCE_FILE", sequencePath)
	t.Setenv("WAILS_BUILD_LOG_FILE", wailsLogPath)

	builder := NewWailsBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "testapp",
	}
	targets := []build.Target{
		{OS: runtime.GOOS, Arch: runtime.GOARCH},
	}

	artifacts, err := builder.Build(context.Background(), cfg, targets)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	content, err := ax.ReadFile(sequencePath)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	require.GreaterOrEqual(t, len(lines), 4)
	assert.Equal(t, "deno", lines[0])
	assert.Equal(t, "task", lines[1])
	assert.Equal(t, "build", lines[2])
	assert.Equal(t, "wails", lines[3])
}

func TestWails_WailsBuilderPropagatesEnvToExternalCommands_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeFrontendCommand(t, binDir, "deno")
	setupFakeWailsToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectDir := setupWailsV2TestProject(t)
	frontendDir := ax.Join(projectDir, "frontend")
	require.NoError(t, ax.MkdirAll(frontendDir, 0o755))
	require.NoError(t, ax.WriteFile(ax.Join(frontendDir, "deno.json"), []byte(`{}`), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(frontendDir, "package.json"), []byte(`{}`), 0o644))

	sequencePath := ax.Join(t.TempDir(), "build-sequence.log")
	t.Setenv("BUILD_SEQUENCE_FILE", sequencePath)
	t.Setenv("CUSTOM_ENV", "expected-value")

	builder := NewWailsBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  t.TempDir(),
		Name:       "testapp",
		Env:        []string{"CUSTOM_ENV=expected-value"},
	}
	targets := []build.Target{
		{OS: runtime.GOOS, Arch: runtime.GOARCH},
	}

	artifacts, err := builder.Build(context.Background(), cfg, targets)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	content, err := ax.ReadFile(sequencePath)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	require.Contains(t, lines, "CUSTOM_ENV=expected-value")
}

func TestWails_WailsBuilderResolveWailsCli_Good(t *testing.T) {
	builder := NewWailsBuilder()
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "wails")
	require.NoError(t, ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755))
	t.Setenv("PATH", "")

	command, err := builder.resolveWailsCli(fallbackPath)
	require.NoError(t, err)
	assert.Equal(t, fallbackPath, command)
}

func TestWails_WailsBuilderResolveWailsCli_Bad(t *testing.T) {
	builder := NewWailsBuilder()
	t.Setenv("PATH", "")

	_, err := builder.resolveWailsCli(ax.Join(t.TempDir(), "missing-wails"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "wails CLI not found")
}

func TestWails_WailsBuilderDetect_Good(t *testing.T) {
	fs := io.Local
	t.Run("detects Wails project with wails.json", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "wails.json"), []byte("{}"), 0o644)
		require.NoError(t, err)

		builder := NewWailsBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("returns false for Go-only project", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module test"), 0o644)
		require.NoError(t, err)

		builder := NewWailsBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})

	t.Run("detects Go project with root frontend package.json", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module test"), 0o644))
		require.NoError(t, ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0o644))

		builder := NewWailsBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("detects Go project with nested frontend deno manifest", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "go.work"), []byte("go 1.26\nuse ."), 0o644))
		frontendDir := ax.Join(dir, "apps", "web")
		require.NoError(t, ax.MkdirAll(frontendDir, 0o755))
		require.NoError(t, ax.WriteFile(ax.Join(frontendDir, "deno.json"), []byte("{}"), 0o644))

		builder := NewWailsBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("returns false for Node.js project", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0o644)
		require.NoError(t, err)

		builder := NewWailsBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		dir := t.TempDir()

		builder := NewWailsBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})
}

func TestWails_DetectPackageManager_Good(t *testing.T) {
	fs := io.Local
	t.Run("detects declared packageManager value", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "package.json"), []byte(`{"packageManager":"yarn@4.5.1"}`), 0o644))
		require.NoError(t, ax.WriteFile(ax.Join(dir, "pnpm-lock.yaml"), []byte(""), 0o644))

		result := detectPackageManager(fs, dir)
		assert.Equal(t, "yarn", result)
	})

	t.Run("detects bun from bun.lockb", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "bun.lockb"), []byte(""), 0o644)
		require.NoError(t, err)

		result := detectPackageManager(fs, dir)
		assert.Equal(t, "bun", result)
	})

	t.Run("detects bun from bun.lock", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "bun.lock"), []byte(""), 0o644)
		require.NoError(t, err)

		result := detectPackageManager(fs, dir)
		assert.Equal(t, "bun", result)
	})

	t.Run("detects pnpm from pnpm-lock.yaml", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "pnpm-lock.yaml"), []byte(""), 0o644)
		require.NoError(t, err)

		result := detectPackageManager(fs, dir)
		assert.Equal(t, "pnpm", result)
	})

	t.Run("detects yarn from yarn.lock", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "yarn.lock"), []byte(""), 0o644)
		require.NoError(t, err)

		result := detectPackageManager(fs, dir)
		assert.Equal(t, "yarn", result)
	})

	t.Run("detects npm from package-lock.json", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "package-lock.json"), []byte(""), 0o644)
		require.NoError(t, err)

		result := detectPackageManager(fs, dir)
		assert.Equal(t, "npm", result)
	})

	t.Run("defaults to npm when no lock file", func(t *testing.T) {
		dir := t.TempDir()

		result := detectPackageManager(fs, dir)
		assert.Equal(t, "npm", result)
	})

	t.Run("prefers bun over other lock files", func(t *testing.T) {
		dir := t.TempDir()
		// Create multiple lock files
		require.NoError(t, ax.WriteFile(ax.Join(dir, "bun.lockb"), []byte(""), 0o644))
		require.NoError(t, ax.WriteFile(ax.Join(dir, "yarn.lock"), []byte(""), 0o644))
		require.NoError(t, ax.WriteFile(ax.Join(dir, "package-lock.json"), []byte(""), 0o644))

		result := detectPackageManager(fs, dir)
		assert.Equal(t, "bun", result)
	})

	t.Run("prefers pnpm over yarn and npm", func(t *testing.T) {
		dir := t.TempDir()
		// Create multiple lock files (no bun)
		require.NoError(t, ax.WriteFile(ax.Join(dir, "pnpm-lock.yaml"), []byte(""), 0o644))
		require.NoError(t, ax.WriteFile(ax.Join(dir, "yarn.lock"), []byte(""), 0o644))
		require.NoError(t, ax.WriteFile(ax.Join(dir, "package-lock.json"), []byte(""), 0o644))

		result := detectPackageManager(fs, dir)
		assert.Equal(t, "pnpm", result)
	})

	t.Run("prefers yarn over npm", func(t *testing.T) {
		dir := t.TempDir()
		// Create multiple lock files (no bun or pnpm)
		require.NoError(t, ax.WriteFile(ax.Join(dir, "yarn.lock"), []byte(""), 0o644))
		require.NoError(t, ax.WriteFile(ax.Join(dir, "package-lock.json"), []byte(""), 0o644))

		result := detectPackageManager(fs, dir)
		assert.Equal(t, "yarn", result)
	})

	t.Run("normalises package manager version pins", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "package.json"), []byte(`{"packageManager":"npm@10.8.2"}`), 0o644))

		result := detectPackageManager(fs, dir)
		assert.Equal(t, "npm", result)
	})
}

func TestWails_CopyBuildArtifact_Good(t *testing.T) {
	fs := io.Local

	t.Run("copies files", func(t *testing.T) {
		dir := t.TempDir()
		sourcePath := ax.Join(dir, "build", "bin", "testapp")
		destPath := ax.Join(dir, "dist", "linux_amd64", "testapp")

		require.NoError(t, ax.MkdirAll(ax.Dir(sourcePath), 0o755))
		require.NoError(t, fs.Write(sourcePath, "binary-data"))

		require.NoError(t, copyBuildArtifact(fs, sourcePath, destPath))

		got, err := fs.Read(destPath)
		require.NoError(t, err)
		assert.Equal(t, "binary-data", got)
	})

	t.Run("copies app bundles recursively", func(t *testing.T) {
		dir := t.TempDir()
		sourcePath := ax.Join(dir, "build", "bin", "testapp.app")
		binaryPath := ax.Join(sourcePath, "Contents", "MacOS", "testapp")
		destPath := ax.Join(dir, "dist", "darwin_arm64", "testapp.app")

		require.NoError(t, ax.MkdirAll(ax.Dir(binaryPath), 0o755))
		require.NoError(t, fs.Write(binaryPath, "bundle-binary"))

		require.NoError(t, copyBuildArtifact(fs, sourcePath, destPath))

		got, err := fs.Read(ax.Join(destPath, "Contents", "MacOS", "testapp"))
		require.NoError(t, err)
		assert.Equal(t, "bundle-binary", got)
	})
}

func TestWails_WailsBuilderBuild_Bad(t *testing.T) {
	t.Run("returns error for nil config", func(t *testing.T) {
		builder := NewWailsBuilder()

		artifacts, err := builder.Build(context.Background(), nil, []build.Target{{OS: "linux", Arch: "amd64"}})
		assert.Error(t, err)
		assert.Nil(t, artifacts)
		assert.Contains(t, err.Error(), "config is nil")
	})

	t.Run("returns error for empty targets", func(t *testing.T) {
		projectDir := setupWailsTestProject(t)

		builder := NewWailsBuilder()
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
}

func TestWails_WailsBuilderBuild_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Check if wails3 is available in PATH
	if _, err := ax.LookPath("wails3"); err != nil {
		t.Skip("wails3 not installed, skipping integration test")
	}

	t.Run("builds for current platform", func(t *testing.T) {
		projectDir := setupWailsTestProject(t)
		outputDir := t.TempDir()

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
			Name:       "testapp",
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
	})
}

func TestWails_WailsBuilderBuildV3Fallback_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeWails3Toolchain(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectDir := setupWailsTestProject(t)
	require.NoError(t, ax.RemoveAll(ax.Join(projectDir, "Taskfile.yml")))

	logPath := ax.Join(t.TempDir(), "wails3.log")
	t.Setenv("WAILS_BUILD_LOG_FILE", logPath)

	builder := NewWailsBuilder()
	goCacheDir := ax.Join(t.TempDir(), "cache", "go-build")
	goModCacheDir := ax.Join(t.TempDir(), "cache", "go-mod")
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  t.TempDir(),
		Name:       "testapp",
		Version:    "v1.2.3",
		BuildTags:  []string{"integration"},
		LDFlags:    []string{"-s", "-w"},
		Cache: build.CacheConfig{
			Enabled: true,
			Paths: []string{
				goCacheDir,
				goModCacheDir,
			},
		},
	}

	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	assert.FileExists(t, artifacts[0].Path)
	assert.Equal(t, "testapp", ax.Base(artifacts[0].Path))

	content, err := ax.ReadFile(logPath)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	require.GreaterOrEqual(t, len(lines), 4)
	assert.Equal(t, "build", lines[0])
	assert.Contains(t, lines, "GOOS=linux")
	assert.Contains(t, lines, "GOARCH=amd64")
	assert.Contains(t, lines, "EXTRA_TAGS=integration")
	assert.Contains(t, strings.Join(lines, "\n"), `BUILD_FLAGS=-tags production,integration -trimpath -buildvcs=false -ldflags="-s -w -X main.version=v1.2.3"`)
	assert.Contains(t, strings.Join(lines, "\n"), "GOFLAGS=-trimpath -tags=integration -ldflags=-s -w -X main.version=v1.2.3")
	assert.Contains(t, lines, "GOCACHE="+goCacheDir)
	assert.Contains(t, lines, "GOMODCACHE="+goModCacheDir)
}

func TestWails_WailsBuilderBuildV3Fallback_Obfuscate_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeWails3GoBuildToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectDir := setupWailsTestProject(t)
	require.NoError(t, ax.RemoveAll(ax.Join(projectDir, "Taskfile.yml")))

	logPath := ax.Join(t.TempDir(), "garble.log")
	t.Setenv("GARBLE_LOG_FILE", logPath)

	builder := NewWailsBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  t.TempDir(),
		Name:       "testapp",
		Obfuscate:  true,
	}

	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	assert.FileExists(t, artifacts[0].Path)

	content, err := ax.ReadFile(logPath)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	require.GreaterOrEqual(t, len(lines), 1)
	assert.Equal(t, "build", lines[0])
	assert.Contains(t, strings.Join(lines, "\n"), "-o")
}

func TestWails_WailsBuilderBuildV3Fallback_PreBuild_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeWails3Toolchain(t, binDir)
	setupFakeFrontendCommand(t, binDir, "deno")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectDir := setupWailsTestProject(t)
	require.NoError(t, ax.RemoveAll(ax.Join(projectDir, "Taskfile.yml")))

	frontendDir := ax.Join(projectDir, "frontend")
	require.NoError(t, ax.MkdirAll(frontendDir, 0o755))
	require.NoError(t, ax.WriteFile(ax.Join(frontendDir, "deno.json"), []byte(`{}`), 0o644))

	logPath := ax.Join(t.TempDir(), "build-sequence.log")
	t.Setenv("BUILD_SEQUENCE_FILE", logPath)

	builder := NewWailsBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  t.TempDir(),
		Name:       "testapp",
	}

	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	content, err := ax.ReadFile(logPath)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	require.GreaterOrEqual(t, len(lines), 7)
	assert.Equal(t, "deno", lines[0])
	assert.Equal(t, "task", lines[1])
	assert.Equal(t, "build", lines[2])
	assert.Equal(t, "wails3", lines[3])
	assert.Equal(t, "build", lines[4])
	assert.Contains(t, lines, "GOOS=linux")
	assert.Contains(t, lines, "GOARCH=amd64")
}

func TestWails_WailsBuilderBuildV3NSIS_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeWails3Toolchain(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectDir := setupWailsTestProject(t)
	require.NoError(t, ax.RemoveAll(ax.Join(projectDir, "Taskfile.yml")))

	logPath := ax.Join(t.TempDir(), "wails3-package.log")
	t.Setenv("WAILS_BUILD_LOG_FILE", logPath)

	builder := NewWailsBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  t.TempDir(),
		Name:       "testapp",
		NSIS:       true,
	}

	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "windows", Arch: "amd64"}})
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	assert.FileExists(t, artifacts[0].Path)
	assert.Equal(t, "testapp-installer.exe", ax.Base(artifacts[0].Path))

	content, err := ax.ReadFile(logPath)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	require.GreaterOrEqual(t, len(lines), 3)
	assert.Equal(t, "package", lines[0])
	assert.Contains(t, lines, "GOOS=windows")
	assert.Contains(t, lines, "GOARCH=amd64")
}

func TestWails_WailsBuilderBuildV3NSISWebView2Download_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeWails3Toolchain(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectDir := setupWailsTestProject(t)
	require.NoError(t, ax.RemoveAll(ax.Join(projectDir, "Taskfile.yml")))

	logPath := ax.Join(t.TempDir(), "wails3-package-webview2.log")
	t.Setenv("WAILS_BUILD_LOG_FILE", logPath)

	builder := NewWailsBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  t.TempDir(),
		Name:       "testapp",
		NSIS:       true,
		WebView2:   "download",
	}

	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "windows", Arch: "amd64"}})
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	assert.FileExists(t, artifacts[0].Path)

	content, err := ax.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "WEBVIEW2_MODE=download")
}

func TestWails_buildV3TaskVars_WebView2Modes_Good(t *testing.T) {
	modes := []string{"download", "embed", "browser", "error"}
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			taskVars, err := buildV3TaskVars(&build.Config{WebView2: mode}, build.Target{OS: "windows", Arch: "amd64"})
			require.NoError(t, err)
			assert.Contains(t, taskVars, "WEBVIEW2_MODE="+mode)
		})
	}
}

func TestWails_WailsBuilderBuildV3NSISWebView2Embed_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeWails3Toolchain(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectDir := setupWailsTestProject(t)
	require.NoError(t, ax.RemoveAll(ax.Join(projectDir, "Taskfile.yml")))
	logPath := ax.Join(t.TempDir(), "wails3-package-webview2-embed.log")
	t.Setenv("WAILS_BUILD_LOG_FILE", logPath)

	builder := NewWailsBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  t.TempDir(),
		Name:       "testapp",
		NSIS:       true,
		WebView2:   "embed",
	}

	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "windows", Arch: "amd64"}})
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	assert.FileExists(t, artifacts[0].Path)

	content, err := ax.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "WEBVIEW2_MODE=embed")
}

func TestWails_WailsBuilderInterface_Good(t *testing.T) {
	// Verify WailsBuilder implements Builder interface
	var _ build.Builder = (*WailsBuilder)(nil)
	var _ build.Builder = NewWailsBuilder()
}

func TestWails_WailsBuilder_Ugly(t *testing.T) {
	t.Run("handles nonexistent frontend directory gracefully", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping integration test in short mode")
		}

		// Create a Wails project without a frontend directory
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "wails.json"), []byte("{}"), 0o644)
		require.NoError(t, err)

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: dir,
			OutputDir:  t.TempDir(),
			Name:       "test",
		}
		targets := []build.Target{
			{OS: runtime.GOOS, Arch: runtime.GOARCH},
		}

		// This will fail because wails3 isn't set up, but it shouldn't panic
		// due to missing frontend directory
		_, err = builder.Build(context.Background(), cfg, targets)
		// We expect an error (wails3 build will fail), but not a panic
		// The error should be about wails3 build, not about frontend
		if err != nil {
			assert.NotContains(t, err.Error(), "frontend dependencies")
		}
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping integration test in short mode")
		}

		projectDir := setupWailsTestProject(t)

		builder := NewWailsBuilder()
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
