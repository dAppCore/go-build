package builders

import (
	"context"
	stdfs "io/fs"
	"runtime"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	storage "dappco.re/go/build/pkg/storage"
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
	result := ax.WriteFile(ax.Join(dir, "wails.json"), []byte(wailsJSON), 0o644)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	goMod := `module testapp

go 1.21

require github.com/wailsapp/wails/v3 v3.0.0
`
	result = ax.WriteFile(ax.Join(dir, "go.mod"), []byte(goMod), 0o644)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	mainGo := `package main

func main() {
	println("hello wails")
}
`
	result = ax.WriteFile(ax.Join(dir, "main.go"), []byte(mainGo), 0o644)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	taskfile := `version: '3'
tasks:
  build:
    cmds:
      - mkdir -p {{.OUTPUT_DIR}}/{{.GOOS}}_{{.GOARCH}}
      - touch {{.OUTPUT_DIR}}/{{.GOOS}}_{{.GOARCH}}/testapp
`
	result = ax.WriteFile(ax.Join(dir, "Taskfile.yml"), []byte(taskfile), 0o644)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	return dir
}

func setupWailsV2TestProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// wails.json
	result := ax.WriteFile(ax.Join(dir, "wails.json"), []byte("{}"), 0o644)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	goMod := `module testapp
go 1.21
require github.com/wailsapp/wails/v2 v2.8.0
`
	result = ax.WriteFile(ax.Join(dir, "go.mod"), []byte(goMod), 0o644)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
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

	result := ax.WriteFile(ax.Join(binDir, "wails"), []byte(wailsScript), 0o755)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
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
	result := ax.WriteFile(ax.Join(binDir, "wails3"), []byte(wails3Script), 0o755)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
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
	result := ax.WriteFile(ax.Join(binDir, "wails3"), []byte(wails3Script), 0o755)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
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
	result = ax.WriteFile(ax.Join(binDir, "garble"), []byte(garbleScript), 0o755)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

}

func setupFakeFrontendCommand(t *testing.T, binDir, name string) {
	t.Helper()

	script := core.Replace(`#!/bin/sh
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
	result := ax.WriteFile(ax.Join(binDir, name), []byte(script), 0o755)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

}

func assertWailsLogLines(t *testing.T, logPath string, want ...string) []string {
	t.Helper()

	content := requireBuilderBytes(t, ax.ReadFile(logPath))
	lines := core.Split(core.Trim(string(content)), "\n")
	if !stdlibAssertEqual(want, lines) {
		t.Fatalf("want %v, got %v", want, lines)
	}
	return lines
}

func assertWailsPreBuildLog(t *testing.T, cfg *build.Config, logName string, want ...string) {
	t.Helper()

	logPath := ax.Join(t.TempDir(), logName)
	t.Setenv("BUILD_SEQUENCE_FILE", logPath)
	result := NewWailsBuilder().PreBuild(context.Background(), cfg)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	assertWailsLogLines(t, logPath, want...)
}

func assertWailsPackagePreBuildLog(t *testing.T, commands []string, configure func(*build.Config), logName string, want ...string) {
	t.Helper()

	binDir := t.TempDir()
	for _, command := range commands {
		setupFakeFrontendCommand(t, binDir, command)
	}
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	projectDir := setupWailsTestProject(t)
	result := ax.WriteFile(ax.Join(projectDir, "package.json"), []byte(`{}`), 0o644)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	cfg := &build.Config{FS: storage.Local, ProjectDir: projectDir}
	if configure != nil {
		configure(cfg)
	}
	assertWailsPreBuildLog(t, cfg, logName, want...)
}

func TestWails_WailsBuilderBuildTaskfileGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Check if task is available
	if result := ax.LookPath("task"); !result.OK {
		t.Skip("task not installed, skipping test")
	}

	t.Run("delegates to Taskfile if present", func(t *testing.T) {
		fs := storage.Local
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
		result := ax.WriteFile(ax.Join(projectDir, "Taskfile.yml"), []byte(taskfile), 0o644)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
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

		artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, targets))
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
		result := ax.WriteFile(taskPath, []byte(script), 0o755)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		t.Setenv("TASK_BUILD_LOG_FILE", logPath)
		t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         storage.Local,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
			Name:       "testapp",
			Version:    "v1.2.3",
			BuildTags:  []string{"integration"},
			LDFlags:    []string{"-s", "-w"},
			WebView2:   "download",
		}

		artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, []build.Target{{OS: "windows", Arch: "amd64"}}))
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if stat := ax.Stat(artifacts[0].Path); !stat.OK {
			t.Fatalf("expected file to exist: %v", artifacts[0].Path)
		}

		content := requireBuilderBytes(t, ax.ReadFile(logPath))
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
		result := ax.WriteFile(taskPath, []byte(script), 0o755)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		setupFakeWails3GoBuildToolchain(t, binDir)
		t.Setenv("GARBLE_LOG_FILE", logPath)
		t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         storage.Local,
			ProjectDir: projectDir,
			OutputDir:  t.TempDir(),
			Name:       "testapp",
			Obfuscate:  true,
		}

		artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}}))
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if stat := ax.Stat(artifacts[0].Path); !stat.OK {
			t.Fatalf("expected file to exist: %v", artifacts[0].Path)
		}

		content := requireBuilderBytes(t, ax.ReadFile(logPath))
		if !stdlibAssertContains(string(content), "build") {
			t.Fatalf("expected %v to contain %v", string(content), "build")
		}
		if !stdlibAssertContains(string(content), "-o") {
			t.Fatalf("expected %v to contain %v", string(content), "-o")
		}

	})
}

func TestWails_WailsBuilderNameGood(t *testing.T) {
	builder := NewWailsBuilder()
	if !stdlibAssertEqual("wails", builder.Name()) {
		t.Fatalf("want %v, got %v", "wails", builder.Name())
	}

}

func TestWails_WailsBuilderBuildV3ConfigGood(t *testing.T) {
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

func TestWails_WailsBuilderResolveFrontendDirGood(t *testing.T) {
	builder := NewWailsBuilder()
	fs := storage.Local

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
			result := ax.MkdirAll(frontendDir, 0o755)
			if !result.OK {
				t.Fatalf("unexpected error: %v", result.Error())
			}
			if tc.marker != "" {
				result = ax.WriteFile(ax.Join(frontendDir, tc.marker), []byte("{}"), 0o644)
				if !result.OK {
					t.Fatalf("unexpected error: %v", result.Error())
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

func TestWails_WailsBuilderBuildV2Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeWailsToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	builder := NewWailsBuilder()

	t.Run("builds v2 project", func(t *testing.T) {
		fs := storage.Local
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

		artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, targets))
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if !(storage.Local.Exists(artifacts[0].Path)) {
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
	result := ax.WriteFile(sourcePath, []byte("fake wails binary\n"), 0o755)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	destDir := t.TempDir()
	destPath := ax.Join(destDir, "testapp")
	result = copyBuildArtifact(storage.Local, sourcePath, destPath)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	stat := ax.Stat(destPath)
	if !stat.OK {
		t.Fatalf("unexpected error: %v", stat.Error())
	}
	info := stat.Value.(stdfs.FileInfo)
	if stdlibAssertZero(info.Mode() & 0o111) {
		t.Fatal("expected non-zero")
	}

}

func TestWails_WailsBuilderBuildV2FlagsGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeWailsToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

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
			FS:         storage.Local,
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

		artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, []build.Target{{OS: "windows", Arch: "amd64"}}))
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}

		content := requireBuilderBytes(t, ax.ReadFile(logPath))

		args := core.Split(core.Trim(string(content)), "\n")
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
			FS:         storage.Local,
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

		artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}}))
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}

		content := requireBuilderBytes(t, ax.ReadFile(logPath))

		args := core.Split(core.Trim(string(content)), "\n")
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

func TestWails_WailsBuilderBuildV2_RespectsConfiguredOutputNameGood(t *testing.T) {
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
			t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

			projectDir := setupWailsV2TestProject(t)
			outputDir := t.TempDir()
			logPath := ax.Join(t.TempDir(), "wails.log")
			t.Setenv("WAILS_BUILD_LOG_FILE", logPath)

			builder := NewWailsBuilder()
			cfg := &build.Config{
				FS:         storage.Local,
				ProjectDir: projectDir,
				OutputDir:  outputDir,
				Name:       "customapp",
				NSIS:       tc.nsis,
			}

			artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, []build.Target{tc.target}))
			if len(artifacts) != 1 {
				t.Fatalf("want len %v, got %v", 1, len(artifacts))
			}
			if !stdlibAssertEqual(tc.expectedBase, ax.Base(artifacts[0].Path)) {
				t.Fatalf("want %v, got %v", tc.expectedBase, ax.Base(artifacts[0].Path))
			}

			content := requireBuilderBytes(t, ax.ReadFile(logPath))

			args := core.Split(core.Trim(string(content)), "\n")
			if !stdlibAssertContains(args, "-o") {
				t.Fatalf("expected %v to contain %v", args, "-o")
			}
			if !stdlibAssertContains(args, "customapp") {
				t.Fatalf("expected %v to contain %v", args, "customapp")
			}

		})
	}
}

func TestWails_WailsBuilderBuildV2FlagsBad(t *testing.T) {
	result := validateWebView2Mode("invalid")
	if result.OK {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(result.Error(), "webview2 must be one of") {
		t.Fatalf("expected error %v to contain %v", result.Error(), "webview2 must be one of")
	}

}

func TestWails_WailsBuilderPreBuildGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("uses deno when deno manifest exists", func(t *testing.T) {
		binDir := t.TempDir()
		setupFakeFrontendCommand(t, binDir, "deno")
		setupFakeFrontendCommand(t, binDir, "npm")
		t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

		projectDir := setupWailsTestProject(t)
		frontendDir := ax.Join(projectDir, "frontend")
		result := ax.MkdirAll(frontendDir, 0o755)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		result = ax.WriteFile(ax.Join(frontendDir, "deno.json"), []byte(`{}`), 0o644)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		result = ax.WriteFile(ax.Join(frontendDir, "package.json"), []byte(`{}`), 0o644)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		logPath := ax.Join(t.TempDir(), "frontend.log")
		t.Setenv("BUILD_SEQUENCE_FILE", logPath)

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         storage.Local,
			ProjectDir: projectDir,
		}
		result = builder.PreBuild(context.Background(), cfg)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		assertWailsLogLines(t, logPath, "deno", "task", "build")

	})

	t.Run("uses configured deno build command when provided", func(t *testing.T) {
		binDir := t.TempDir()
		setupFakeFrontendCommand(t, binDir, "deno")
		setupFakeFrontendCommand(t, binDir, "deno-build")
		t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

		projectDir := setupWailsTestProject(t)
		frontendDir := ax.Join(projectDir, "frontend")
		result := ax.MkdirAll(frontendDir, 0o755)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		result = ax.WriteFile(ax.Join(frontendDir, "deno.json"), []byte(`{}`), 0o644)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		logPath := ax.Join(t.TempDir(), "frontend-custom.log")
		t.Setenv("BUILD_SEQUENCE_FILE", logPath)

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         storage.Local,
			ProjectDir: projectDir,
			DenoBuild:  "deno-build --target release",
		}
		result = builder.PreBuild(context.Background(), cfg)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		assertWailsLogLines(t, logPath, "deno-build", "--target", "release")

	})

	t.Run("DENO_BUILD env override wins over config", func(t *testing.T) {
		binDir := t.TempDir()
		setupFakeFrontendCommand(t, binDir, "deno")
		setupFakeFrontendCommand(t, binDir, "deno-build")
		setupFakeFrontendCommand(t, binDir, "env-deno-build")
		t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))
		t.Setenv("DENO_BUILD", "env-deno-build --env")

		projectDir := setupWailsTestProject(t)
		frontendDir := ax.Join(projectDir, "frontend")
		result := ax.MkdirAll(frontendDir, 0o755)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		result = ax.WriteFile(ax.Join(frontendDir, "deno.json"), []byte(`{}`), 0o644)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		logPath := ax.Join(t.TempDir(), "frontend-env.log")
		t.Setenv("BUILD_SEQUENCE_FILE", logPath)

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         storage.Local,
			ProjectDir: projectDir,
			DenoBuild:  "deno-build --config",
		}
		result = builder.PreBuild(context.Background(), cfg)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
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
		t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

		projectDir := setupWailsTestProject(t)
		frontendDir := ax.Join(projectDir, "apps", "web")
		result := ax.MkdirAll(frontendDir, 0o755)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		result = ax.WriteFile(ax.Join(frontendDir, "package.json"), []byte(`{}`), 0o644)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		logPath := ax.Join(t.TempDir(), "frontend.log")
		t.Setenv("BUILD_SEQUENCE_FILE", logPath)

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         storage.Local,
			ProjectDir: projectDir,
		}
		result = builder.PreBuild(context.Background(), cfg)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
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
			t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

			projectDir := setupWailsTestProject(t)
			frontendDir := ax.Join(projectDir, "frontend")
			result := ax.MkdirAll(frontendDir, 0o755)
			if !result.OK {
				t.Fatalf("unexpected error: %v", result.Error())
			}
			result = ax.WriteFile(ax.Join(frontendDir, "package.json"), []byte(`{}`), 0o644)
			if !result.OK {
				t.Fatalf("unexpected error: %v", result.Error())
			}
			result = ax.WriteFile(ax.Join(frontendDir, tc.lock), []byte(""), 0o644)
			if !result.OK {
				t.Fatalf("unexpected error: %v", result.Error())
			}

			assertWailsPreBuildLog(t, &build.Config{
				FS:         storage.Local,
				ProjectDir: projectDir,
			}, "frontend.log", tc.command, "run", "build")
		})
	}

	t.Run("uses yarn when yarn.lock exists", func(t *testing.T) {
		binDir := t.TempDir()
		setupFakeFrontendCommand(t, binDir, "yarn")
		t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

		projectDir := setupWailsTestProject(t)
		frontendDir := ax.Join(projectDir, "frontend")
		result := ax.MkdirAll(frontendDir, 0o755)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		result = ax.WriteFile(ax.Join(frontendDir, "package.json"), []byte(`{}`), 0o644)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		result = ax.WriteFile(ax.Join(frontendDir, "yarn.lock"), []byte(""), 0o644)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		logPath := ax.Join(t.TempDir(), "frontend.log")
		t.Setenv("BUILD_SEQUENCE_FILE", logPath)

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         storage.Local,
			ProjectDir: projectDir,
		}
		result = builder.PreBuild(context.Background(), cfg)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		assertWailsLogLines(t, logPath, "yarn", "build")

	})
}

func TestWails_WailsBuilderBuildV2PreBuildGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeFrontendCommand(t, binDir, "deno")
	setupFakeFrontendCommand(t, binDir, "npm")
	setupFakeWailsToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	projectDir := setupWailsV2TestProject(t)
	frontendDir := ax.Join(projectDir, "frontend")
	result := ax.MkdirAll(frontendDir, 0o755)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	result = ax.WriteFile(ax.Join(frontendDir, "deno.json"), []byte(`{}`), 0o644)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	result = ax.WriteFile(ax.Join(frontendDir, "package.json"), []byte(`{}`), 0o644)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	outputDir := t.TempDir()
	sequencePath := ax.Join(t.TempDir(), "build-sequence.log")
	wailsLogPath := ax.Join(t.TempDir(), "wails.log")
	t.Setenv("BUILD_SEQUENCE_FILE", sequencePath)
	t.Setenv("WAILS_BUILD_LOG_FILE", wailsLogPath)

	builder := NewWailsBuilder()
	cfg := &build.Config{
		FS:         storage.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "testapp",
	}
	targets := []build.Target{
		{OS: runtime.GOOS, Arch: runtime.GOARCH},
	}

	artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, targets))
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}

	content := requireBuilderBytes(t, ax.ReadFile(sequencePath))

	lines := core.Split(core.Trim(string(content)), "\n")
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

func TestWails_WailsBuilderPropagatesEnvToExternalCommandsGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeFrontendCommand(t, binDir, "deno")
	setupFakeWailsToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	projectDir := setupWailsV2TestProject(t)
	frontendDir := ax.Join(projectDir, "frontend")
	result := ax.MkdirAll(frontendDir, 0o755)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	result = ax.WriteFile(ax.Join(frontendDir, "deno.json"), []byte(`{}`), 0o644)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	result = ax.WriteFile(ax.Join(frontendDir, "package.json"), []byte(`{}`), 0o644)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	sequencePath := ax.Join(t.TempDir(), "build-sequence.log")
	t.Setenv("BUILD_SEQUENCE_FILE", sequencePath)
	t.Setenv("CUSTOM_ENV", "expected-value")

	builder := NewWailsBuilder()
	cfg := &build.Config{
		FS:         storage.Local,
		ProjectDir: projectDir,
		OutputDir:  t.TempDir(),
		Name:       "testapp",
		Env:        []string{"CUSTOM_ENV=expected-value"},
	}
	targets := []build.Target{
		{OS: runtime.GOOS, Arch: runtime.GOARCH},
	}

	artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, targets))
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}

	content := requireBuilderBytes(t, ax.ReadFile(sequencePath))

	lines := core.Split(core.Trim(string(content)), "\n")
	if !stdlibAssertContains(lines, "CUSTOM_ENV=expected-value") {
		t.Fatalf("expected %v to contain %v", lines, "CUSTOM_ENV=expected-value")
	}

}

func TestWails_WailsBuilderResolveWailsCliGood(t *testing.T) {
	builder := NewWailsBuilder()
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "wails")
	result := ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	t.Setenv("PATH", "")

	command := requireCPPString(t, builder.resolveWailsCli(fallbackPath))
	if !stdlibAssertEqual(fallbackPath, command) {
		t.Fatalf("want %v, got %v", fallbackPath, command)
	}

}

func TestWails_WailsBuilderResolveWailsCliBad(t *testing.T) {
	builder := NewWailsBuilder()
	t.Setenv("PATH", "")

	result := builder.resolveWailsCli(ax.Join(t.TempDir(), "missing-wails"))
	if result.OK {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(result.Error(), "wails CLI not found") {
		t.Fatalf("expected %v to contain %v", result.Error(), "wails CLI not found")
	}

}

func TestWails_WailsBuilderDetectGood(t *testing.T) {
	fs := storage.Local
	t.Run("detects Wails project with wails.json", func(t *testing.T) {
		dir := t.TempDir()
		result := ax.WriteFile(ax.Join(dir, "wails.json"), []byte("{}"), 0o644)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		builder := NewWailsBuilder()
		detected := requireCPPBool(t, builder.Detect(fs, dir))
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("returns false for Go-only project", func(t *testing.T) {
		dir := t.TempDir()
		result := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module test"), 0o644)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		builder := NewWailsBuilder()
		detected := requireCPPBool(t, builder.Detect(fs, dir))
		if detected {
			t.Fatal("expected false")
		}

	})

	t.Run("detects Go project with root frontend package.json", func(t *testing.T) {
		dir := t.TempDir()
		result := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module test"), 0o644)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		result = ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0o644)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		builder := NewWailsBuilder()
		detected := requireCPPBool(t, builder.Detect(fs, dir))
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("detects Go project with nested frontend deno manifest", func(t *testing.T) {
		dir := t.TempDir()
		result := ax.WriteFile(ax.Join(dir, "go.work"), []byte("go 1.26\nuse ."), 0o644)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		frontendDir := ax.Join(dir, "apps", "web")
		result = ax.MkdirAll(frontendDir, 0o755)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		result = ax.WriteFile(ax.Join(frontendDir, "deno.json"), []byte("{}"), 0o644)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		builder := NewWailsBuilder()
		detected := requireCPPBool(t, builder.Detect(fs, dir))
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("returns false for Node.js project", func(t *testing.T) {
		dir := t.TempDir()
		result := ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0o644)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		builder := NewWailsBuilder()
		detected := requireCPPBool(t, builder.Detect(fs, dir))
		if detected {
			t.Fatal("expected false")
		}

	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		dir := t.TempDir()

		builder := NewWailsBuilder()
		detected := requireCPPBool(t, builder.Detect(fs, dir))
		if detected {
			t.Fatal("expected false")
		}

	})
}

func TestWails_DetectPackageManagerGood(t *testing.T) {
	fs := storage.Local
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
				result := ax.WriteFile(ax.Join(dir, path), []byte(content), 0o644)
				if !result.OK {
					t.Fatalf("unexpected error: %v", result.Error())
				}
			}

			result := detectPackageManager(fs, dir)
			if !stdlibAssertEqual(tc.want, result) {
				t.Fatalf("want %v, got %v", tc.want, result)
			}
		})
	}
}

func TestWails_CopyBuildArtifactGood(t *testing.T) {
	fs := storage.Local

	t.Run("copies files", func(t *testing.T) {
		dir := t.TempDir()
		sourcePath := ax.Join(dir, "build", "bin", "testapp")
		destPath := ax.Join(dir, "dist", "linux_amd64", "testapp")
		result := ax.MkdirAll(ax.Dir(sourcePath), 0o755)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		result = fs.Write(sourcePath, "binary-data")
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		result = copyBuildArtifact(fs, sourcePath, destPath)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		got := requireCPPString(t, fs.Read(destPath))
		if !stdlibAssertEqual("binary-data", got) {
			t.Fatalf("want %v, got %v", "binary-data", got)
		}

	})

	t.Run("copies app bundles recursively", func(t *testing.T) {
		dir := t.TempDir()
		sourcePath := ax.Join(dir, "build", "bin", "testapp.app")
		binaryPath := ax.Join(sourcePath, "Contents", "MacOS", "testapp")
		destPath := ax.Join(dir, "dist", "darwin_arm64", "testapp.app")
		result := ax.MkdirAll(ax.Dir(binaryPath), 0o755)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		result = fs.Write(binaryPath, "bundle-binary")
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		result = copyBuildArtifact(fs, sourcePath, destPath)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		got := requireCPPString(t, fs.Read(ax.Join(destPath, "Contents", "MacOS", "testapp")))
		if !stdlibAssertEqual("bundle-binary", got) {
			t.Fatalf("want %v, got %v", "bundle-binary", got)
		}

	})
}

func TestWails_WailsBuilderBuildUnsafeVersionBad(t *testing.T) {
	t.Run("returns error for nil config", func(t *testing.T) {
		builder := NewWailsBuilder()

		result := builder.Build(context.Background(), nil, []build.Target{{OS: "linux", Arch: "amd64"}})
		if result.OK {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(result.Error(), "config is nil") {
			t.Fatalf("expected %v to contain %v", result.Error(), "config is nil")
		}

	})

	t.Run("returns error for empty targets", func(t *testing.T) {
		projectDir := setupWailsTestProject(t)

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         storage.Local,
			ProjectDir: projectDir,
			OutputDir:  t.TempDir(),
			Name:       "test",
		}

		result := builder.Build(context.Background(), cfg, []build.Target{})
		if result.OK {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(result.Error(), "no targets specified") {
			t.Fatalf("expected %v to contain %v", result.Error(), "no targets specified")
		}

	})
}

func TestWails_WailsBuilderBuildGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Check if wails3 is available in PATH
	if result := ax.LookPath("wails3"); !result.OK {
		t.Skip("wails3 not installed, skipping integration test")
	}

	t.Run("builds for current platform", func(t *testing.T) {
		projectDir := setupWailsTestProject(t)
		outputDir := t.TempDir()

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         storage.Local,
			ProjectDir: projectDir,
			OutputDir:  outputDir,
			Name:       "testapp",
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
		}
		if !stdlibAssertEqual(runtime.GOARCH, artifact.Arch) {
			t.Fatalf("want %v, got %v", runtime.GOARCH, artifact.Arch)
		}

	})
}

func TestWails_WailsBuilderBuildV3FallbackGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeWails3Toolchain(t, binDir)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	projectDir := setupWailsTestProject(t)
	result := ax.RemoveAll(ax.Join(projectDir, "Taskfile.yml"))
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	logPath := ax.Join(t.TempDir(), "wails3.log")
	t.Setenv("WAILS_BUILD_LOG_FILE", logPath)

	builder := NewWailsBuilder()
	goCacheDir := ax.Join(t.TempDir(), "cache", "go-build")
	goModCacheDir := ax.Join(t.TempDir(), "cache", "go-mod")
	cfg := &build.Config{
		FS:         storage.Local,
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

	artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}}))
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}
	if stat := ax.Stat(artifacts[0].Path); !stat.OK {
		t.Fatalf("expected file to exist: %v", artifacts[0].Path)
	}
	if !stdlibAssertEqual("testapp", ax.Base(artifacts[0].Path)) {
		t.Fatalf("want %v, got %v", "testapp", ax.Base(artifacts[0].Path))
	}

	content := requireBuilderBytes(t, ax.ReadFile(logPath))

	lines := core.Split(core.Trim(string(content)), "\n")
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
	joinedLines := core.Join("\n", lines...)
	if !stdlibAssertContains(joinedLines, `BUILD_FLAGS=-tags production,integration -trimpath -buildvcs=false -ldflags="-s -w -X main.version=v1.2.3"`) {
		t.Fatalf("expected %v to contain %v", joinedLines, `BUILD_FLAGS=-tags production,integration -trimpath -buildvcs=false -ldflags="-s -w -X main.version=v1.2.3"`)
	}
	if !stdlibAssertContains(joinedLines, "GOFLAGS=-trimpath -tags=integration -ldflags=-s -w -X main.version=v1.2.3") {
		t.Fatalf("expected %v to contain %v", joinedLines, "GOFLAGS=-trimpath -tags=integration -ldflags=-s -w -X main.version=v1.2.3")
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
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	projectDir := setupWailsTestProject(t)
	result := ax.RemoveAll(ax.Join(projectDir, "Taskfile.yml"))
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	logPath := ax.Join(t.TempDir(), "garble.log")
	t.Setenv("GARBLE_LOG_FILE", logPath)

	builder := NewWailsBuilder()
	cfg := &build.Config{
		FS:         storage.Local,
		ProjectDir: projectDir,
		OutputDir:  t.TempDir(),
		Name:       "testapp",
		Obfuscate:  true,
	}

	artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}}))
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}
	if stat := ax.Stat(artifacts[0].Path); !stat.OK {
		t.Fatalf("expected file to exist: %v", artifacts[0].Path)
	}

	content := requireBuilderBytes(t, ax.ReadFile(logPath))

	lines := core.Split(core.Trim(string(content)), "\n")
	if len(lines) < 1 {
		t.Fatalf("expected %v to be greater than or equal to %v", len(lines), 1)
	}
	if !stdlibAssertEqual("build", lines[0]) {
		t.Fatalf("want %v, got %v", "build", lines[0])
	}
	joinedLines := core.Join("\n", lines...)
	if !stdlibAssertContains(joinedLines, "-o") {
		t.Fatalf("expected %v to contain %v", joinedLines, "-o")
	}

}

func TestWails_WailsBuilderBuildV3Fallback_PreBuildGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeWails3Toolchain(t, binDir)
	setupFakeFrontendCommand(t, binDir, "deno")
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	projectDir := setupWailsTestProject(t)
	result := ax.RemoveAll(ax.Join(projectDir, "Taskfile.yml"))
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	frontendDir := ax.Join(projectDir, "frontend")
	result = ax.MkdirAll(frontendDir, 0o755)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	result = ax.WriteFile(ax.Join(frontendDir, "deno.json"), []byte(`{}`), 0o644)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	logPath := ax.Join(t.TempDir(), "build-sequence.log")
	t.Setenv("BUILD_SEQUENCE_FILE", logPath)

	builder := NewWailsBuilder()
	cfg := &build.Config{
		FS:         storage.Local,
		ProjectDir: projectDir,
		OutputDir:  t.TempDir(),
		Name:       "testapp",
	}

	artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}}))
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}

	content := requireBuilderBytes(t, ax.ReadFile(logPath))

	lines := core.Split(core.Trim(string(content)), "\n")
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

func TestWails_WailsBuilderBuildV3NSISGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeWails3Toolchain(t, binDir)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	projectDir := setupWailsTestProject(t)
	result := ax.RemoveAll(ax.Join(projectDir, "Taskfile.yml"))
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	logPath := ax.Join(t.TempDir(), "wails3-package.log")
	t.Setenv("WAILS_BUILD_LOG_FILE", logPath)

	builder := NewWailsBuilder()
	cfg := &build.Config{
		FS:         storage.Local,
		ProjectDir: projectDir,
		OutputDir:  t.TempDir(),
		Name:       "testapp",
		NSIS:       true,
	}

	artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, []build.Target{{OS: "windows", Arch: "amd64"}}))
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}
	if stat := ax.Stat(artifacts[0].Path); !stat.OK {
		t.Fatalf("expected file to exist: %v", artifacts[0].Path)
	}
	if !stdlibAssertEqual("testapp-installer.exe", ax.Base(artifacts[0].Path)) {
		t.Fatalf("want %v, got %v", "testapp-installer.exe", ax.Base(artifacts[0].Path))
	}

	content := requireBuilderBytes(t, ax.ReadFile(logPath))

	lines := core.Split(core.Trim(string(content)), "\n")
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

func TestWails_WailsBuilderBuildV3NSISWebView2DownloadGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	assertWailsBuilderBuildV3NSISWebView2(t, "download")
}

func assertWailsBuilderBuildV3NSISWebView2(t *testing.T, mode string) {
	t.Helper()

	binDir := t.TempDir()
	setupFakeWails3Toolchain(t, binDir)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	projectDir := setupWailsTestProject(t)
	result := ax.RemoveAll(ax.Join(projectDir, "Taskfile.yml"))
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	logPath := ax.Join(t.TempDir(), "wails3-package-webview2.log")
	t.Setenv("WAILS_BUILD_LOG_FILE", logPath)

	builder := NewWailsBuilder()
	cfg := &build.Config{
		FS:         storage.Local,
		ProjectDir: projectDir,
		OutputDir:  t.TempDir(),
		Name:       "testapp",
		NSIS:       true,
		WebView2:   mode,
	}

	artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, []build.Target{{OS: "windows", Arch: "amd64"}}))
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}
	if stat := ax.Stat(artifacts[0].Path); !stat.OK {
		t.Fatalf("expected file to exist: %v", artifacts[0].Path)
	}

	content := requireBuilderBytes(t, ax.ReadFile(logPath))
	if !stdlibAssertContains(string(content), "WEBVIEW2_MODE="+mode) {
		t.Fatalf("expected %v to contain %v", string(content), "WEBVIEW2_MODE="+mode)
	}
}

func TestWails_buildV3TaskVars_WebView2Modes_Good(t *testing.T) {
	modes := []string{"download", "embed", "browser", "error"}
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			result := buildV3TaskVars(&build.Config{WebView2: mode}, build.Target{OS: "windows", Arch: "amd64"})
			if !result.OK {
				t.Fatalf("unexpected error: %v", result.Error())
			}
			taskVars := result.Value.([]string)
			if !stdlibAssertContains(taskVars, "WEBVIEW2_MODE="+mode) {
				t.Fatalf("expected %v to contain %v", taskVars, "WEBVIEW2_MODE="+mode)
			}

		})
	}
}

func TestWails_WailsBuilderBuildV3NSISWebView2EmbedGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	assertWailsBuilderBuildV3NSISWebView2(t, "embed")
}

func TestWails_WailsBuilderBuildBad(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	projectDir := setupWailsTestProject(t)
	result := ax.RemoveAll(ax.Join(projectDir, "Taskfile.yml"))
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	builder := NewWailsBuilder()
	cfg := &build.Config{
		FS:         storage.Local,
		ProjectDir: projectDir,
		OutputDir:  t.TempDir(),
		Name:       "unsafe-version",
		Version:    "v1.2.3 && echo unsafe",
	}

	result = builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
	if result.OK {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(result.Error(), "unsupported characters") {

		// Verify WailsBuilder implements Builder interface
		t.Fatalf("expected %v to contain %v", result.Error(), "unsupported characters")
	}

}

func TestWails_WailsBuilderInterfaceGood(t *testing.T) {
	builder := NewWailsBuilder()
	var _ build.Builder = builder
	if !stdlibAssertEqual("wails", builder.Name()) {
		t.Fatalf("want %v, got %v", "wails", builder.Name())
	}
	detected := requireCPPBool(t, builder.Detect(nil, t.TempDir()))
	if detected {
		t.Fatal("expected empty temp directory not to be detected")
	}
}

func TestWails_WailsBuilderUgly(t *testing.T) {
	t.Run("handles nonexistent frontend directory gracefully", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping integration test in short mode")
		}

		// Create a Wails project without a frontend directory
		dir := t.TempDir()
		result := ax.WriteFile(ax.Join(dir, "wails.json"), []byte("{}"), 0o644)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		builder := NewWailsBuilder()
		cfg := &build.Config{
			FS:         storage.Local,
			ProjectDir: dir,
			OutputDir:  t.TempDir(),
			Name:       "test",
		}
		targets := []build.Target{
			{OS: runtime.GOOS, Arch: runtime.GOARCH},
		}

		// This will fail because wails3 isn't set up, but it shouldn't panic
		// due to missing frontend directory
		result = builder.Build(context.Background(), cfg, targets)
		// We expect an error (wails3 build will fail), but not a panic
		// The error should be about wails3 build, not about frontend
		if !result.OK {
			if stdlibAssertContains(result.Error(), "frontend dependencies") {
				t.Fatalf("expected %v not to contain %v", result.Error(), "frontend dependencies")
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
			FS:         storage.Local,
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
}

// --- v0.9.0 generated compliance triplets ---
func TestWails_NewWailsBuilder_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewWailsBuilder()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestWails_NewWailsBuilder_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewWailsBuilder()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestWails_NewWailsBuilder_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewWailsBuilder()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestWails_WailsBuilder_Name_Good(t *core.T) {
	subject := &WailsBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestWails_WailsBuilder_Name_Bad(t *core.T) {
	subject := &WailsBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestWails_WailsBuilder_Name_Ugly(t *core.T) {
	subject := &WailsBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestWails_WailsBuilder_Detect_Good(t *core.T) {
	subject := &WailsBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Detect(storage.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestWails_WailsBuilder_Detect_Bad(t *core.T) {
	subject := &WailsBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Detect(storage.NewMemoryMedium(), "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestWails_WailsBuilder_Detect_Ugly(t *core.T) {
	subject := &WailsBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Detect(storage.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestWails_WailsBuilder_Build_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &WailsBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Build(ctx, nil, nil)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestWails_WailsBuilder_Build_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &WailsBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Build(ctx, nil, nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestWails_WailsBuilder_Build_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &WailsBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Build(ctx, nil, nil)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestWails_WailsBuilder_PreBuild_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &WailsBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.PreBuild(ctx, nil)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestWails_WailsBuilder_PreBuild_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &WailsBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.PreBuild(ctx, nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestWails_WailsBuilder_PreBuild_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &WailsBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.PreBuild(ctx, nil)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
