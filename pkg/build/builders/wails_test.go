package builders

import (
	"context"
	"os"
	"runtime"
	"strings"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/io"
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
	if err != nil {
		t.Fatalf("unexpected error: %v",

			// Create a minimal go.mod
			err)
	}

	goMod := `module testapp

go 1.21

require github.com/wailsapp/wails/v3 v3.0.0
`
	err = ax.WriteFile(ax.Join(dir, "go.mod"), []byte(goMod), 0o644)
	if err != nil {
		t.Fatalf("unexpected error: %v",

			// Create a minimal main.go
			err)
	}

	mainGo := `package main

func main() {
	println("hello wails")
}
`
	err = ax.WriteFile(ax.Join(dir, "main.go"), []byte(mainGo), 0o644)
	if err != nil {
		t.Fatalf("unexpected error: %v",

			// Create a minimal Taskfile.yml
			err)
	}

	taskfile := `version: '3'
tasks:
  build:
    cmds:
      - mkdir -p {{.OUTPUT_DIR}}/{{.GOOS}}_{{.GOARCH}}
      - touch {{.OUTPUT_DIR}}/{{.GOOS}}_{{.GOARCH}}/testapp
`
	err = ax.WriteFile(ax.Join(dir, "Taskfile.yml"), []byte(taskfile), 0o644)
	if err != nil {
		t.Fatalf("unexpected error: %v",

			// setupWailsV2TestProject creates a Wails v2 project structure.
			err)
	}

	return dir
}

func setupWailsV2TestProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// wails.json
	err := ax.WriteFile(ax.Join(dir, "wails.json"), []byte("{}"), 0o644)
	if err != nil {
		t.Fatalf("unexpected error: %v",

			// go.mod with v2
			err)
	}

	goMod := `module testapp
go 1.21
require github.com/wailsapp/wails/v2 v2.8.0
`
	err = ax.WriteFile(ax.Join(dir, "go.mod"), []byte(goMod), 0o644)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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
		-o)
			shift
			binary_name="${1:-}"
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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
	if err := ax.WriteFile(ax.Join(binDir, "wails3"), []byte(wails3Script), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}

func setupFakeWails3GoBuildToolchain(t *testing.T, binDir string) {
	t.Helper()

	wails3Script := `#!/bin/sh
set -eu

name="${NAME:-testapp}"
mkdir -p "bin"
go build -o "bin/${name}" .
`
	if err := ax.WriteFile(ax.Join(binDir, "wails3"), []byte(wails3Script), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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
	if err := ax.WriteFile(ax.Join(binDir, "garble"), []byte(garbleScript), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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
	if err := ax.WriteFile(ax.Join(binDir, name), []byte(script), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}

func assertWailsLogLines(t *testing.T, logPath string, want ...string) []string {
	t.Helper()

	content, err := ax.ReadFile(logPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if !stdlibAssertEqual(want, lines) {
		t.Fatalf("want %v, got %v", want, lines)
	}
	return lines
}

func assertWailsPreBuildLog(t *testing.T, cfg *build.Config, logName string, want ...string) {
	t.Helper()

	logPath := ax.Join(t.TempDir(), logName)
	t.Setenv("BUILD_SEQUENCE_FILE", logPath)
	if err := NewWailsBuilder().PreBuild(context.Background(), cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertWailsLogLines(t, logPath, want...)
}

func assertWailsPackagePreBuildLog(t *testing.T, commands []string, configure func(*build.Config), logName string, want ...string) {
	t.Helper()

	binDir := t.TempDir()
	for _, command := range commands {
		setupFakeFrontendCommand(t, binDir, command)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectDir := setupWailsTestProject(t)
	if err := ax.WriteFile(ax.Join(projectDir, "package.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg := &build.Config{FS: io.Local, ProjectDir: projectDir}
	if configure != nil {
		configure(cfg)
	}
	assertWailsPreBuildLog(t, cfg, logName, want...)
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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertEmpty(artifacts) {
			t.Fatal("expected non-empty")
		}

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
		if err := ax.WriteFile(taskPath, []byte(script), 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

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
		if !stdlibAssertContains(string(content), "GOOS=windows") {
			t.Fatalf("expected %v to contain %v", string(content), "GOOS=windows")
		}
		if !stdlibAssertContains(string(content), "GOARCH=amd64") {
			t.Fatalf("expected %v to contain %v", string(content), "GOARCH=amd64")
		}
		if !stdlibAssertContains(string(content), "CGO_ENABLED=1") {
			t.Fatalf("expected %v to contain %v", string(content), "CGO_ENABLED=1")
		}
		if !stdlibAssertContains(string(content), "GOFLAGS=-trimpath -tags=integration -ldflags=-s -w -X main.version=v1.2.3") {
			t.Fatalf("expected %v to contain %v", string(content), "GOFLAGS=-trimpath -tags=integration -ldflags=-s -w -X main.version=v1.2.3")
		}
		if !stdlibAssertContains(string(content), "EXTRA_TAGS=integration") {
			t.Fatalf("expected %v to contain %v", string(content), "EXTRA_TAGS=integration")
		}
		if !stdlibAssertContains(string(content), "WEBVIEW2_MODE=download") {
			t.Fatalf("expected %v to contain %v", string(content), "WEBVIEW2_MODE=download")
		}
		if !stdlibAssertContains(string(content), `BUILD_FLAGS=-tags production,integration -trimpath -buildvcs=false -ldflags="-s -w -H windowsgui -X main.version=v1.2.3"`) {
			t.Fatalf("expected %v to contain %v", string(content), `BUILD_FLAGS=-tags production,integration -trimpath -buildvcs=false -ldflags="-s -w -H windowsgui -X main.version=v1.2.3"`)
		}

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
		if err := ax.WriteFile(taskPath, []byte(script), 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

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
		if !stdlibAssertContains(string(content), "build") {
			t.Fatalf("expected %v to contain %v", string(content), "build")
		}
		if !stdlibAssertContains(string(content), "-o") {
			t.Fatalf("expected %v to contain %v", string(content), "-o")
		}

	})
}

func TestWails_WailsBuilderName_Good(t *testing.T) {
	builder := NewWailsBuilder()
	if !stdlibAssertEqual("wails", builder.Name()) {
		t.Fatalf("want %v, got %v", "wails", builder.Name())
	}

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
	if stdlibAssertNil(v3Config) {
		t.Fatal("expected non-nil")
	}
	if cfg.CGO {
		t.Fatal("expected false")
	}
	if !(v3Config.CGO) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual(cfg.Name, v3Config.Name) {
		t.Fatalf("want %v, got %v", cfg.Name, v3Config.Name)
	}
	if !stdlibAssertEqual(cfg.Flags, v3Config.Flags) {
		t.Fatalf("want %v, got %v", cfg.Flags, v3Config.Flags)
	}
	if !stdlibAssertEqual(cfg.LDFlags, v3Config.LDFlags) {
		t.Fatalf("want %v, got %v", cfg.LDFlags, v3Config.LDFlags)
	}

}

func TestWails_WailsBuilderResolveFrontendDir_Good(t *testing.T) {
	builder := NewWailsBuilder()
	fs := io.Local

	for _, tc := range []struct {
		name       string
		frontend   []string
		marker     string
		denoEnable bool
		wantEmpty  bool
	}{
		{name: "finds nested package.json frontends", frontend: []string{"apps", "web"}, marker: "package.json"},
		{name: "finds nested deno.json frontends", frontend: []string{"packages", "site"}, marker: "deno.json"},
		{name: "ignores frontends deeper than depth 2", frontend: []string{"apps", "marketing", "web"}, marker: "package.json", wantEmpty: true},
		{name: "falls back to frontend directory when DENO_ENABLE is set", frontend: []string{"frontend"}, denoEnable: true},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.denoEnable {
				t.Setenv("DENO_ENABLE", "true")
			}

			projectDir := t.TempDir()
			frontendDir := ax.Join(append([]string{projectDir}, tc.frontend...)...)
			if err := ax.MkdirAll(frontendDir, 0o755); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.marker != "" {
				if err := ax.WriteFile(ax.Join(frontendDir, tc.marker), []byte("{}"), 0o644); err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			got := builder.resolveFrontendDir(fs, projectDir)
			if tc.wantEmpty {
				if !stdlibAssertEmpty(got) {
					t.Fatalf("expected empty, got %v", got)
				}
				return
			}
			if !stdlibAssertEqual(frontendDir, got) {
				t.Fatalf("want %v, got %v", frontendDir, got)
			}
		})
	}
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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if !(io.Local.Exists(artifacts[0].Path)) {
			t.Fatal("expected true")
		}

	})
}

func TestWails_copyBuildArtifact_PreservesMode_Good(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable mode bits are not portable on Windows")
	}

	sourceDir := t.TempDir()
	sourcePath := ax.Join(sourceDir, "testapp")
	if err := ax.WriteFile(sourcePath, []byte("fake wails binary\n"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	destDir := t.TempDir()
	destPath := ax.Join(destDir, "testapp")
	if err := copyBuildArtifact(io.Local, sourcePath, destPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, err := ax.Stat(destPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdlibAssertZero(info.Mode() & 0o111) {
		t.Fatal("expected non-zero")
	}

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
		if !stdlibAssertEqual("build", args[0]) {
			t.Fatalf("want %v, got %v", "build", args[0])
		}
		if !stdlibAssertContains(args, "-o") {
			t.Fatalf("expected %v to contain %v", args, "-o")
		}
		if !stdlibAssertContains(args, "testapp") {
			t.Fatalf("expected %v to contain %v", args, "testapp")
		}
		if !stdlibAssertContains(args, "-tags") {
			t.Fatalf("expected %v to contain %v", args, "-tags")
		}
		if !stdlibAssertContains(args, "integration,webkit2_41") {
			t.Fatalf("expected %v to contain %v", args, "integration,webkit2_41")
		}
		if !stdlibAssertContains(args, "-ldflags") {
			t.Fatalf("expected %v to contain %v", args, "-ldflags")
		}
		if !stdlibAssertContains(args, "-s -w -X main.version=v1.2.3") {
			t.Fatalf("expected %v to contain %v", args, "-s -w -X main.version=v1.2.3")
		}
		if !stdlibAssertContains(args, "-obfuscated") {
			t.Fatalf("expected %v to contain %v", args, "-obfuscated")
		}
		if !stdlibAssertContains(args, "-nsis") {
			t.Fatalf("expected %v to contain %v", args, "-nsis")
		}
		if !stdlibAssertContains(args, "-webview2") {
			t.Fatalf("expected %v to contain %v", args, "-webview2")
		}
		if !stdlibAssertContains(args, "embed") {
			t.Fatalf("expected %v to contain %v", args, "embed")
		}
		if !stdlibAssertContains(args, "GOCACHE="+goCacheDir) {
			t.Fatalf("expected %v to contain %v", args, "GOCACHE="+goCacheDir)
		}
		if !stdlibAssertContains(args, "GOMODCACHE="+goModCacheDir) {
			t.Fatalf("expected %v to contain %v", args, "GOMODCACHE="+goModCacheDir)
		}

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
		if !stdlibAssertContains(args, "-o") {
			t.Fatalf("expected %v to contain %v", args, "-o")
		}
		if !stdlibAssertContains(args, "testapp") {
			t.Fatalf("expected %v to contain %v", args, "testapp")
		}
		if stdlibAssertContains(args, "-nsis") {
			t.Fatalf("expected %v not to contain %v", args, "-nsis")
		}
		if stdlibAssertContains(args, "-webview2") {
			t.Fatalf("expected %v not to contain %v", args, "-webview2")
		}
		if stdlibAssertContains(args, "embed") {
			t.Fatalf("expected %v not to contain %v", args, "embed")
		}

	})
}

func TestWails_WailsBuilderBuildV2_RespectsConfiguredOutputName_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cases := []struct {
		name         string
		target       build.Target
		nsis         bool
		expectedBase string
	}{
		{
			name:         "linux binary",
			target:       build.Target{OS: "linux", Arch: "amd64"},
			expectedBase: "customapp",
		},
		{
			name:         "darwin app bundle",
			target:       build.Target{OS: "darwin", Arch: "arm64"},
			expectedBase: "customapp.app",
		},
		{
			name:         "windows executable",
			target:       build.Target{OS: "windows", Arch: "amd64"},
			expectedBase: "customapp.exe",
		},
		{
			name:         "windows nsis installer",
			target:       build.Target{OS: "windows", Arch: "amd64"},
			nsis:         true,
			expectedBase: "customapp-installer.exe",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			binDir := t.TempDir()
			setupFakeWailsToolchain(t, binDir)
			t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

			projectDir := setupWailsV2TestProject(t)
			outputDir := t.TempDir()
			logPath := ax.Join(t.TempDir(), "wails.log")
			t.Setenv("WAILS_BUILD_LOG_FILE", logPath)

			builder := NewWailsBuilder()
			cfg := &build.Config{
				FS:         io.Local,
				ProjectDir: projectDir,
				OutputDir:  outputDir,
				Name:       "customapp",
				NSIS:       tc.nsis,
			}

			artifacts, err := builder.Build(context.Background(), cfg, []build.Target{tc.target})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(artifacts) != 1 {
				t.Fatalf("want len %v, got %v", 1, len(artifacts))
			}
			if !stdlibAssertEqual(tc.expectedBase, ax.Base(artifacts[0].Path)) {
				t.Fatalf("want %v, got %v", tc.expectedBase, ax.Base(artifacts[0].Path))
			}

			content, err := ax.ReadFile(logPath)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			args := strings.Split(strings.TrimSpace(string(content)), "\n")
			if !stdlibAssertContains(args, "-o") {
				t.Fatalf("expected %v to contain %v", args, "-o")
			}
			if !stdlibAssertContains(args, "customapp") {
				t.Fatalf("expected %v to contain %v", args, "customapp")
			}

		})
	}
}

func TestWails_WailsBuilderBuildV2Flags_Bad(t *testing.T) {
	err := validateWebView2Mode("invalid")
	if err == nil {
		t.Fatal("expected error")
	}
	if err == nil {
		t.Fatal("expected error")
	} else if !stdlibAssertContains(err.Error(), "webview2 must be one of") {
		t.Fatalf("expected error %v to contain %v", err, "webview2 must be one of")
	}

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
		if err := ax.MkdirAll(frontendDir, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(frontendDir, "deno.json"), []byte(`{}`), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(frontendDir, "package.json"), []byte(`{}`), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		logPath := ax.Join(t.TempDir(), "frontend.log")
		t.Setenv("BUILD_SEQUENCE_FILE", logPath)

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
		}
		if err := builder.PreBuild(context.Background(), cfg); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertWailsLogLines(t, logPath, "deno", "task", "build")

	})

	t.Run("uses configured deno build command when provided", func(t *testing.T) {
		binDir := t.TempDir()
		setupFakeFrontendCommand(t, binDir, "deno")
		setupFakeFrontendCommand(t, binDir, "deno-build")
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		projectDir := setupWailsTestProject(t)
		frontendDir := ax.Join(projectDir, "frontend")
		if err := ax.MkdirAll(frontendDir, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(frontendDir, "deno.json"), []byte(`{}`), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		logPath := ax.Join(t.TempDir(), "frontend-custom.log")
		t.Setenv("BUILD_SEQUENCE_FILE", logPath)

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			DenoBuild:  "deno-build --target release",
		}
		if err := builder.PreBuild(context.Background(), cfg); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertWailsLogLines(t, logPath, "deno-build", "--target", "release")

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
		if err := ax.MkdirAll(frontendDir, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(frontendDir, "deno.json"), []byte(`{}`), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		logPath := ax.Join(t.TempDir(), "frontend-env.log")
		t.Setenv("BUILD_SEQUENCE_FILE", logPath)

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			DenoBuild:  "deno-build --config",
		}
		if err := builder.PreBuild(context.Background(), cfg); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertWailsLogLines(t, logPath, "env-deno-build", "--env")

	})

	t.Run("falls back to npm when only package.json exists", func(t *testing.T) {
		assertWailsPackagePreBuildLog(t, []string{"deno", "npm"}, nil, "frontend.log", "npm", "run", "build")
	})

	t.Run("uses configured npm build command when provided", func(t *testing.T) {
		assertWailsPackagePreBuildLog(t, []string{"npm", "npm-build"}, func(cfg *build.Config) {
			cfg.NpmBuild = "npm-build --scope app"
		}, "frontend-npm-custom.log", "npm-build", "--scope", "app")
	})

	t.Run("prefers deno when DENO_ENABLE is set without a deno manifest", func(t *testing.T) {
		t.Setenv("DENO_ENABLE", "true")

		assertWailsPackagePreBuildLog(t, []string{"deno", "npm"}, nil, "frontend-deno-enable.log", "deno", "task", "build")
	})

	t.Run("uses configured deno build command without a deno manifest", func(t *testing.T) {
		assertWailsPackagePreBuildLog(t, []string{"deno-build", "npm"}, func(cfg *build.Config) {
			cfg.DenoBuild = "deno-build --target release"
		}, "frontend-config-deno.log", "deno-build", "--target", "release")
	})

	t.Run("discovers nested package.json in a monorepo", func(t *testing.T) {
		binDir := t.TempDir()
		setupFakeFrontendCommand(t, binDir, "npm")
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		projectDir := setupWailsTestProject(t)
		frontendDir := ax.Join(projectDir, "apps", "web")
		if err := ax.MkdirAll(frontendDir, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(frontendDir, "package.json"), []byte(`{}`), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		logPath := ax.Join(t.TempDir(), "frontend.log")
		t.Setenv("BUILD_SEQUENCE_FILE", logPath)

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
		}
		if err := builder.PreBuild(context.Background(), cfg); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertWailsLogLines(t, logPath, "npm", "run", "build")

	})

	for _, tc := range []struct {
		name    string
		command string
		lock    string
	}{
		{name: "uses bun when bun.lockb exists", command: "bun", lock: "bun.lockb"},
		{name: "uses pnpm when pnpm-lock.yaml exists", command: "pnpm", lock: "pnpm-lock.yaml"},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			binDir := t.TempDir()
			setupFakeFrontendCommand(t, binDir, tc.command)
			t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

			projectDir := setupWailsTestProject(t)
			frontendDir := ax.Join(projectDir, "frontend")
			if err := ax.MkdirAll(frontendDir, 0o755); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if err := ax.WriteFile(ax.Join(frontendDir, "package.json"), []byte(`{}`), 0o644); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if err := ax.WriteFile(ax.Join(frontendDir, tc.lock), []byte(""), 0o644); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			assertWailsPreBuildLog(t, &build.Config{
				FS:         io.Local,
				ProjectDir: projectDir,
			}, "frontend.log", tc.command, "run", "build")
		})
	}

	t.Run("uses yarn when yarn.lock exists", func(t *testing.T) {
		binDir := t.TempDir()
		setupFakeFrontendCommand(t, binDir, "yarn")
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		projectDir := setupWailsTestProject(t)
		frontendDir := ax.Join(projectDir, "frontend")
		if err := ax.MkdirAll(frontendDir, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(frontendDir, "package.json"), []byte(`{}`), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(frontendDir, "yarn.lock"), []byte(""), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		logPath := ax.Join(t.TempDir(), "frontend.log")
		t.Setenv("BUILD_SEQUENCE_FILE", logPath)

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
		}
		if err := builder.PreBuild(context.Background(), cfg); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertWailsLogLines(t, logPath, "yarn", "build")

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
	if err := ax.MkdirAll(frontendDir, 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(frontendDir, "deno.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(frontendDir, "package.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}

	content, err := ax.ReadFile(sequencePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) < 4 {
		t.Fatalf("expected %v to be greater than or equal to %v", len(lines), 4)
	}
	if !stdlibAssertEqual("deno", lines[0]) {
		t.Fatalf("want %v, got %v", "deno", lines[0])
	}
	if !stdlibAssertEqual("task", lines[1]) {
		t.Fatalf("want %v, got %v", "task", lines[1])
	}
	if !stdlibAssertEqual("build", lines[2]) {
		t.Fatalf("want %v, got %v", "build", lines[2])
	}
	if !stdlibAssertEqual("wails", lines[3]) {
		t.Fatalf("want %v, got %v", "wails", lines[3])
	}

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
	if err := ax.MkdirAll(frontendDir, 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(frontendDir, "deno.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(frontendDir, "package.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}

	content, err := ax.ReadFile(sequencePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if !stdlibAssertContains(lines, "CUSTOM_ENV=expected-value") {
		t.Fatalf("expected %v to contain %v", lines, "CUSTOM_ENV=expected-value")
	}

}

func TestWails_WailsBuilderResolveWailsCli_Good(t *testing.T) {
	builder := NewWailsBuilder()
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "wails")
	if err := ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Setenv("PATH", "")

	command, err := builder.resolveWailsCli(fallbackPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(fallbackPath, command) {
		t.Fatalf("want %v, got %v", fallbackPath, command)
	}

}

func TestWails_WailsBuilderResolveWailsCli_Bad(t *testing.T) {
	builder := NewWailsBuilder()
	t.Setenv("PATH", "")

	_, err := builder.resolveWailsCli(ax.Join(t.TempDir(), "missing-wails"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "wails CLI not found") {
		t.Fatalf("expected %v to contain %v", err.Error(), "wails CLI not found")
	}

}

func TestWails_WailsBuilderDetect_Good(t *testing.T) {
	fs := io.Local
	t.Run("detects Wails project with wails.json", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "wails.json"), []byte("{}"), 0o644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewWailsBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("returns false for Go-only project", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module test"), 0o644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewWailsBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detected {
			t.Fatal("expected false")
		}

	})

	t.Run("detects Go project with root frontend package.json", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module test"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewWailsBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("detects Go project with nested frontend deno manifest", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "go.work"), []byte("go 1.26\nuse ."), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		frontendDir := ax.Join(dir, "apps", "web")
		if err := ax.MkdirAll(frontendDir, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(frontendDir, "deno.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewWailsBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("returns false for Node.js project", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0o644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewWailsBuilder()
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

		builder := NewWailsBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detected {
			t.Fatal("expected false")
		}

	})
}

func TestWails_DetectPackageManager_Good(t *testing.T) {
	fs := io.Local
	for _, tc := range []struct {
		name  string
		files map[string]string
		want  string
	}{
		{
			name: "detects declared packageManager value",
			files: map[string]string{
				"package.json":   `{"packageManager":"yarn@4.5.1"}`,
				"pnpm-lock.yaml": "",
			},
			want: "yarn",
		},
		{name: "detects bun from bun.lockb", files: map[string]string{"bun.lockb": ""}, want: "bun"},
		{name: "detects bun from bun.lock", files: map[string]string{"bun.lock": ""}, want: "bun"},
		{name: "detects pnpm from pnpm-lock.yaml", files: map[string]string{"pnpm-lock.yaml": ""}, want: "pnpm"},
		{name: "detects yarn from yarn.lock", files: map[string]string{"yarn.lock": ""}, want: "yarn"},
		{name: "detects npm from package-lock.json", files: map[string]string{"package-lock.json": ""}, want: "npm"},
		{name: "defaults to npm when no lock file", want: "npm"},
		{
			name: "prefers bun over other lock files",
			files: map[string]string{
				"bun.lockb":         "",
				"yarn.lock":         "",
				"package-lock.json": "",
			},
			want: "bun",
		},
		{
			name: "prefers pnpm over yarn and npm",
			files: map[string]string{
				"pnpm-lock.yaml":    "",
				"yarn.lock":         "",
				"package-lock.json": "",
			},
			want: "pnpm",
		},
		{
			name: "prefers yarn over npm",
			files: map[string]string{
				"yarn.lock":         "",
				"package-lock.json": "",
			},
			want: "yarn",
		},
		{name: "normalises package manager version pins", files: map[string]string{"package.json": `{"packageManager":"npm@10.8.2"}`}, want: "npm"},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for path, content := range tc.files {
				if err := ax.WriteFile(ax.Join(dir, path), []byte(content), 0o644); err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			result := detectPackageManager(fs, dir)
			if !stdlibAssertEqual(tc.want, result) {
				t.Fatalf("want %v, got %v", tc.want, result)
			}
		})
	}
}

func TestWails_CopyBuildArtifact_Good(t *testing.T) {
	fs := io.Local

	t.Run("copies files", func(t *testing.T) {
		dir := t.TempDir()
		sourcePath := ax.Join(dir, "build", "bin", "testapp")
		destPath := ax.Join(dir, "dist", "linux_amd64", "testapp")
		if err := ax.MkdirAll(ax.Dir(sourcePath), 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := fs.Write(sourcePath, "binary-data"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := copyBuildArtifact(fs, sourcePath, destPath); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := fs.Read(destPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("binary-data", got) {
			t.Fatalf("want %v, got %v", "binary-data", got)
		}

	})

	t.Run("copies app bundles recursively", func(t *testing.T) {
		dir := t.TempDir()
		sourcePath := ax.Join(dir, "build", "bin", "testapp.app")
		binaryPath := ax.Join(sourcePath, "Contents", "MacOS", "testapp")
		destPath := ax.Join(dir, "dist", "darwin_arm64", "testapp.app")
		if err := ax.MkdirAll(ax.Dir(binaryPath), 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := fs.Write(binaryPath, "bundle-binary"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := copyBuildArtifact(fs, sourcePath, destPath); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := fs.Read(ax.Join(destPath, "Contents", "MacOS", "testapp"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("bundle-binary", got) {
			t.Fatalf("want %v, got %v", "bundle-binary", got)
		}

	})
}

func TestWails_WailsBuilderBuildUnsafeVersion_Bad(t *testing.T) {
	t.Run("returns error for nil config", func(t *testing.T) {
		builder := NewWailsBuilder()

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
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertNil(artifacts) {
			t.Fatalf("expected nil, got %v", artifacts)
		}
		if !stdlibAssertContains(err.Error(), "no targets specified") {
			t.Fatalf("expected %v to contain %v", err.Error(), "no targets specified")
		}

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
		}
		if !stdlibAssertEqual(runtime.GOARCH, artifact.Arch) {
			t.Fatalf("want %v, got %v", runtime.GOARCH, artifact.Arch)
		}

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
	if err := ax.RemoveAll(ax.Join(projectDir, "Taskfile.yml")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}
	if _, err := os.Stat(artifacts[0].Path); err != nil {
		t.Fatalf("expected file to exist: %v", artifacts[0].Path)
	}
	if !stdlibAssertEqual("testapp", ax.Base(artifacts[0].Path)) {
		t.Fatalf("want %v, got %v", "testapp", ax.Base(artifacts[0].Path))
	}

	content, err := ax.ReadFile(logPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) < 4 {
		t.Fatalf("expected %v to be greater than or equal to %v", len(lines), 4)
	}
	if !stdlibAssertEqual("build", lines[0]) {
		t.Fatalf("want %v, got %v", "build", lines[0])
	}
	if !stdlibAssertContains(lines, "GOOS=linux") {
		t.Fatalf("expected %v to contain %v", lines, "GOOS=linux")
	}
	if !stdlibAssertContains(lines, "GOARCH=amd64") {
		t.Fatalf("expected %v to contain %v", lines, "GOARCH=amd64")
	}
	if !stdlibAssertContains(lines, "EXTRA_TAGS=integration") {
		t.Fatalf("expected %v to contain %v", lines, "EXTRA_TAGS=integration")
	}
	if !stdlibAssertContains(strings.Join(lines, "\n"), `BUILD_FLAGS=-tags production,integration -trimpath -buildvcs=false -ldflags="-s -w -X main.version=v1.2.3"`) {
		t.Fatalf("expected %v to contain %v", strings.Join(lines, "\n"), `BUILD_FLAGS=-tags production,integration -trimpath -buildvcs=false -ldflags="-s -w -X main.version=v1.2.3"`)
	}
	if !stdlibAssertContains(strings.Join(lines, "\n"), "GOFLAGS=-trimpath -tags=integration -ldflags=-s -w -X main.version=v1.2.3") {
		t.Fatalf("expected %v to contain %v", strings.Join(lines, "\n"), "GOFLAGS=-trimpath -tags=integration -ldflags=-s -w -X main.version=v1.2.3")
	}
	if !stdlibAssertContains(lines, "GOCACHE="+goCacheDir) {
		t.Fatalf("expected %v to contain %v", lines, "GOCACHE="+goCacheDir)
	}
	if !stdlibAssertContains(lines, "GOMODCACHE="+goModCacheDir) {
		t.Fatalf("expected %v to contain %v", lines, "GOMODCACHE="+goModCacheDir)
	}

}

func TestWails_WailsBuilderBuildV3Fallback_Obfuscate_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeWails3GoBuildToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectDir := setupWailsTestProject(t)
	if err := ax.RemoveAll(ax.Join(projectDir, "Taskfile.yml")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) < 1 {
		t.Fatalf("expected %v to be greater than or equal to %v", len(lines), 1)
	}
	if !stdlibAssertEqual("build", lines[0]) {
		t.Fatalf("want %v, got %v", "build", lines[0])
	}
	if !stdlibAssertContains(strings.Join(lines, "\n"), "-o") {
		t.Fatalf("expected %v to contain %v", strings.Join(lines, "\n"), "-o")
	}

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
	if err := ax.RemoveAll(ax.Join(projectDir, "Taskfile.yml")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	frontendDir := ax.Join(projectDir, "frontend")
	if err := ax.MkdirAll(frontendDir, 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(frontendDir, "deno.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) < 7 {
		t.Fatalf("expected %v to be greater than or equal to %v", len(lines), 7)
	}
	if !stdlibAssertEqual("deno", lines[0]) {
		t.Fatalf("want %v, got %v", "deno", lines[0])
	}
	if !stdlibAssertEqual("task", lines[1]) {
		t.Fatalf("want %v, got %v", "task", lines[1])
	}
	if !stdlibAssertEqual("build", lines[2]) {
		t.Fatalf("want %v, got %v", "build", lines[2])
	}
	if !stdlibAssertEqual("wails3", lines[3]) {
		t.Fatalf("want %v, got %v", "wails3", lines[3])
	}
	if !stdlibAssertEqual("build", lines[4]) {
		t.Fatalf("want %v, got %v", "build", lines[4])
	}
	if !stdlibAssertContains(lines, "GOOS=linux") {
		t.Fatalf("expected %v to contain %v", lines, "GOOS=linux")
	}
	if !stdlibAssertContains(lines, "GOARCH=amd64") {
		t.Fatalf("expected %v to contain %v", lines, "GOARCH=amd64")
	}

}

func TestWails_WailsBuilderBuildV3NSIS_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeWails3Toolchain(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectDir := setupWailsTestProject(t)
	if err := ax.RemoveAll(ax.Join(projectDir, "Taskfile.yml")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}
	if _, err := os.Stat(artifacts[0].Path); err != nil {
		t.Fatalf("expected file to exist: %v", artifacts[0].Path)
	}
	if !stdlibAssertEqual("testapp-installer.exe", ax.Base(artifacts[0].Path)) {
		t.Fatalf("want %v, got %v", "testapp-installer.exe", ax.Base(artifacts[0].Path))
	}

	content, err := ax.ReadFile(logPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected %v to be greater than or equal to %v", len(lines), 3)
	}
	if !stdlibAssertEqual("package", lines[0]) {
		t.Fatalf("want %v, got %v", "package", lines[0])
	}
	if !stdlibAssertContains(lines, "GOOS=windows") {
		t.Fatalf("expected %v to contain %v", lines, "GOOS=windows")
	}
	if !stdlibAssertContains(lines, "GOARCH=amd64") {
		t.Fatalf("expected %v to contain %v", lines, "GOARCH=amd64")
	}

}

func TestWails_WailsBuilderBuildV3NSISWebView2Download_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	assertWailsBuilderBuildV3NSISWebView2(t, "download")
}

func assertWailsBuilderBuildV3NSISWebView2(t *testing.T, mode string) {
	t.Helper()

	binDir := t.TempDir()
	setupFakeWails3Toolchain(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectDir := setupWailsTestProject(t)
	if err := ax.RemoveAll(ax.Join(projectDir, "Taskfile.yml")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logPath := ax.Join(t.TempDir(), "wails3-package-webview2.log")
	t.Setenv("WAILS_BUILD_LOG_FILE", logPath)

	builder := NewWailsBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  t.TempDir(),
		Name:       "testapp",
		NSIS:       true,
		WebView2:   mode,
	}

	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "windows", Arch: "amd64"}})
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
	if !stdlibAssertContains(string(content), "WEBVIEW2_MODE="+mode) {
		t.Fatalf("expected %v to contain %v", string(content), "WEBVIEW2_MODE="+mode)
	}
}

func TestWails_buildV3TaskVars_WebView2Modes_Good(t *testing.T) {
	modes := []string{"download", "embed", "browser", "error"}
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			taskVars, err := buildV3TaskVars(&build.Config{WebView2: mode}, build.Target{OS: "windows", Arch: "amd64"})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !stdlibAssertContains(taskVars, "WEBVIEW2_MODE="+mode) {
				t.Fatalf("expected %v to contain %v", taskVars, "WEBVIEW2_MODE="+mode)
			}

		})
	}
}

func TestWails_WailsBuilderBuildV3NSISWebView2Embed_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	assertWailsBuilderBuildV3NSISWebView2(t, "embed")
}

func TestWails_WailsBuilderBuild_Bad(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	projectDir := setupWailsTestProject(t)
	if err := ax.RemoveAll(ax.Join(projectDir, "Taskfile.yml")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	builder := NewWailsBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  t.TempDir(),
		Name:       "unsafe-version",
		Version:    "v1.2.3 && echo unsafe",
	}

	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertEmpty(artifacts) {
		t.Fatalf("expected empty, got %v", artifacts)
	}
	if !stdlibAssertContains(err.Error(), "unsupported characters") {

		// Verify WailsBuilder implements Builder interface
		t.Fatalf("expected %v to contain %v", err.Error(), "unsupported characters")
	}

}

func TestWails_WailsBuilderInterface_Good(t *testing.T) {

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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

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
			if stdlibAssertContains(err.Error(), "frontend dependencies") {
				t.Fatalf("expected %v not to contain %v", err.Error(), "frontend dependencies")
			}

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
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertEmpty(artifacts) {
			t.Fatalf("expected empty, got %v", artifacts)
		}

	})
}
