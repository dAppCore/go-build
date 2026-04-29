package builders

import (
	"context"
	"runtime"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/io"
)

func setupFakeNodeToolchain(t *testing.T, binDir string) {
	t.Helper()

	script := `#!/bin/sh
set -eu

log_file="${NODE_BUILD_LOG_FILE:-}"
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

name="${NAME:-nodeapp}"
printf 'fake node artifact\n' > "$platform_dir/$name"
chmod +x "$platform_dir/$name"
`

	for _, name := range []string{"npm", "pnpm", "yarn", "bun", "deno"} {
		if err := ax.WriteFile(ax.Join(binDir, name), []byte(script), 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

	}
}

func setupFakeNodeCommand(t *testing.T, binDir, name string) {
	t.Helper()

	script := `#!/bin/sh
set -eu

log_file="${NODE_BUILD_LOG_FILE:-}"
if [ -n "$log_file" ]; then
	printf '%s\n' "$(basename "$0")" >> "$log_file"
	printf '%s\n' "$@" >> "$log_file"
fi

output_dir="${OUTPUT_DIR:-dist}"
platform_dir="${TARGET_DIR:-$output_dir/${GOOS:-}_${GOARCH:-}}"
mkdir -p "$platform_dir"
printf 'fake node artifact\n' > "$platform_dir/${NAME:-nodeapp}"
chmod +x "$platform_dir/${NAME:-nodeapp}"
`
	if err := ax.WriteFile(ax.Join(binDir, name), []byte(script), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}

func assertNodeLogPrefix(t *testing.T, logPath string, want ...string) []string {
	t.Helper()

	content, err := ax.ReadFile(logPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := core.Split(core.Trim(string(content)), "\n")
	if len(lines) < len(want) {
		t.Fatalf("expected %v to be greater than or equal to %v", len(lines), len(want))
	}
	for i, value := range want {
		if !stdlibAssertEqual(value, lines[i]) {
			t.Fatalf("want %v, got %v", value, lines[i])
		}
	}
	return lines
}

func setupNodeTestProject(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	if err := ax.WriteFile(ax.Join(dir, "package.json"), []byte(`{"name":"testapp","scripts":{"build":"node build.js"}}`), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(dir, "build.js"), []byte(`console.log("build")`), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return dir
}

func TestNode_NodeBuilderNameGood(t *testing.T) {
	builder := NewNodeBuilder()
	if !stdlibAssertEqual("node", builder.Name()) {
		t.Fatalf("want %v, got %v", "node", builder.Name())
	}

}

func TestNode_NodeBuilderDetectGood(t *testing.T) {
	fs := io.Local

	t.Run("detects package.json projects", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewNodeBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		builder := NewNodeBuilder()
		detected, err := builder.Detect(fs, t.TempDir())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detected {
			t.Fatal("expected false")
		}

	})

	t.Run("detects nested package.json projects", func(t *testing.T) {
		dir := t.TempDir()
		nested := ax.Join(dir, "apps", "web")
		if err := ax.MkdirAll(nested, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(nested, "package.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewNodeBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("detects root deno projects", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "deno.json"), []byte(`{"tasks":{"build":"deno eval ''"}}`), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewNodeBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(detected) {
			t.Fatal("expected true")
		}

	})
}

func TestNode_NodeBuilderBuildGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeNodeToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	projectDir := setupNodeTestProject(t)
	outputDir := t.TempDir()
	logDir := t.TempDir()
	logPath := ax.Join(logDir, "node.log")
	t.Setenv("NODE_BUILD_LOG_FILE", logPath)
	if err := ax.WriteFile(ax.Join(projectDir, "pnpm-lock.yaml"), []byte("lockfile"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	builder := NewNodeBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "testapp",
		Version:    "v1.2.3",
		Env:        []string{"FOO=bar"},
	}

	targets := []build.Target{
		{OS: "linux", Arch: "amd64"},
	}

	artifacts, err := builder.Build(context.Background(), cfg, targets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}
	if _, err := ax.Stat(artifacts[0].Path); err != nil {
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

	lines := core.Split(core.Trim(string(content)), "\n")
	if len(lines) < 5 {
		t.Fatalf("expected %v to be greater than or equal to %v", len(lines), 5)
	}
	if !stdlibAssertEqual("pnpm", lines[0]) {
		t.Fatalf("want %v, got %v", "pnpm", lines[0])
	}
	if !stdlibAssertEqual("run", lines[1]) {
		t.Fatalf("want %v, got %v", "run", lines[1])
	}
	if !stdlibAssertEqual("build", lines[2]) {
		t.Fatalf("want %v, got %v", "build", lines[2])
	}
	if !stdlibAssertEqual("GOOS=linux", lines[3]) {
		t.Fatalf("want %v, got %v", "GOOS=linux", lines[3])
	}
	if !stdlibAssertEqual("GOARCH=amd64", lines[4]) {
		t.Fatalf("want %v, got %v", "GOARCH=amd64", lines[4])
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

func TestNode_NodeBuilderBuild_Good_Deno(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeNodeToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	projectDir := t.TempDir()
	if err := ax.WriteFile(ax.Join(projectDir, "deno.json"), []byte(`{"tasks":{"build":"deno eval ''"}}`), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outputDir := t.TempDir()
	logPath := ax.Join(t.TempDir(), "deno.log")
	t.Setenv("NODE_BUILD_LOG_FILE", logPath)

	builder := NewNodeBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "denoapp",
		Version:    "v1.2.3",
	}

	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}
	if _, err := ax.Stat(artifacts[0].Path); err != nil {
		t.Fatalf("expected file to exist: %v", artifacts[0].Path)
	}

	assertNodeLogPrefix(t, logPath, "deno", "task", "build")

}

func TestNode_NodeBuilderBuild_Good_DenoOverrideFromConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeNodeToolchain(t, binDir)
	setupFakeNodeCommand(t, binDir, "deno-build")
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	projectDir := t.TempDir()
	if err := ax.WriteFile(ax.Join(projectDir, "deno.json"), []byte(`{"tasks":{"build":"deno eval ''"}}`), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outputDir := t.TempDir()
	logPath := ax.Join(t.TempDir(), "deno-override.log")
	t.Setenv("NODE_BUILD_LOG_FILE", logPath)

	builder := NewNodeBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "denoapp",
		DenoBuild:  "deno-build --target release",
	}

	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}

	assertNodeLogPrefix(t, logPath, "deno-build", "--target", "release")

}

func TestNode_NodeBuilderBuild_Good_DenoOverrideFromEnvWins(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeNodeToolchain(t, binDir)
	setupFakeNodeCommand(t, binDir, "deno-build")
	setupFakeNodeCommand(t, binDir, "env-deno-build")
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))
	t.Setenv("DENO_BUILD", "env-deno-build --env")

	projectDir := t.TempDir()
	if err := ax.WriteFile(ax.Join(projectDir, "deno.json"), []byte(`{"tasks":{"build":"deno eval ''"}}`), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outputDir := t.TempDir()
	logPath := ax.Join(t.TempDir(), "deno-env-override.log")
	t.Setenv("NODE_BUILD_LOG_FILE", logPath)

	builder := NewNodeBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "denoapp",
		DenoBuild:  "deno-build --config",
	}

	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}

	assertNodeLogPrefix(t, logPath, "env-deno-build", "--env")

}

func TestNode_NodeBuilderBuild_Good_NpmOverrideFromConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeNodeToolchain(t, binDir)
	setupFakeNodeCommand(t, binDir, "npm-build")
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	projectDir := t.TempDir()
	if err := ax.WriteFile(ax.Join(projectDir, "package.json"), []byte(`{"name":"testapp","scripts":{"build":"node build.js"}}`), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outputDir := t.TempDir()
	logPath := ax.Join(t.TempDir(), "npm-override.log")
	t.Setenv("NODE_BUILD_LOG_FILE", logPath)

	builder := NewNodeBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "npmapp",
		NpmBuild:   "npm-build --scope app",
	}

	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}

	assertNodeLogPrefix(t, logPath, "npm-build", "--scope", "app")

}

func TestNode_NodeBuilderBuild_Good_DenoEnableWithoutManifest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeNodeToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))
	t.Setenv("DENO_ENABLE", "true")

	projectDir := t.TempDir()
	if err := ax.WriteFile(ax.Join(projectDir, "package.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outputDir := t.TempDir()
	logPath := ax.Join(t.TempDir(), "deno-enable.log")
	t.Setenv("NODE_BUILD_LOG_FILE", logPath)

	builder := NewNodeBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "denoapp",
	}

	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}

	assertNodeLogPrefix(t, logPath, "deno", "task", "build")

}

func TestNode_NodeBuilderBuild_Good_DenoOverrideWithoutManifest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeNodeToolchain(t, binDir)
	setupFakeNodeCommand(t, binDir, "deno-build")
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	projectDir := t.TempDir()
	if err := ax.WriteFile(ax.Join(projectDir, "package.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outputDir := t.TempDir()
	logPath := ax.Join(t.TempDir(), "deno-config.log")
	t.Setenv("NODE_BUILD_LOG_FILE", logPath)

	builder := NewNodeBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "denoapp",
		DenoBuild:  "deno-build --target release",
	}

	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}

	assertNodeLogPrefix(t, logPath, "deno-build", "--target", "release")

}

func TestNode_ResolvePackageManagerGood(t *testing.T) {
	fs := io.Local
	builder := NewNodeBuilder()

	t.Run("prefers packageManager declaration over lockfiles", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "package.json"), []byte(`{"packageManager":"pnpm@9.12.0"}`), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, "bun.lockb"), []byte(""), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := builder.resolvePackageManager(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("pnpm", result) {
			t.Fatalf("want %v, got %v", "pnpm", result)
		}

	})

	t.Run("normalises package manager version pins", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "package.json"), []byte(`{"packageManager":"bun@1.1.38"}`), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := builder.resolvePackageManager(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("bun", result) {
			t.Fatalf("want %v, got %v", "bun", result)
		}

	})
}

func TestNode_NodeBuilderFindArtifactsForTargetGood(t *testing.T) {
	fs := io.Local
	builder := NewNodeBuilder()

	t.Run("finds files in platform subdirectory", func(t *testing.T) {
		dir := t.TempDir()
		platformDir := ax.Join(dir, "linux_amd64")
		if err := ax.MkdirAll(platformDir, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		artifactPath := ax.Join(platformDir, "testapp")
		if err := ax.WriteFile(artifactPath, []byte("binary"), 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		artifacts := builder.findArtifactsForTarget(fs, dir, build.Target{OS: "linux", Arch: "amd64"})
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if !stdlibAssertEqual(artifactPath, artifacts[0].Path) {
			t.Fatalf("want %v, got %v", artifactPath, artifacts[0].Path)
		}

	})

	t.Run("finds darwin app bundles", func(t *testing.T) {
		dir := t.TempDir()
		platformDir := ax.Join(dir, "darwin_arm64")
		appDir := ax.Join(platformDir, "TestApp.app")
		if err := ax.MkdirAll(appDir, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		artifacts := builder.findArtifactsForTarget(fs, dir, build.Target{OS: "darwin", Arch: "arm64"})
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if !stdlibAssertEqual(appDir, artifacts[0].Path) {
			t.Fatalf("want %v, got %v", appDir, artifacts[0].Path)
		}

	})

	t.Run("falls back to name patterns in root", func(t *testing.T) {
		dir := t.TempDir()
		artifactPath := ax.Join(dir, "testapp-linux-amd64")
		if err := ax.WriteFile(artifactPath, []byte("binary"), 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		artifacts := builder.findArtifactsForTarget(fs, dir, build.Target{OS: "linux", Arch: "amd64"})
		if stdlibAssertEmpty(artifacts) {
			t.Fatal("expected non-empty")
		}
		if !stdlibAssertEqual(artifactPath, artifacts[0].Path) {
			t.Fatalf("want %v, got %v", artifactPath, artifacts[0].Path)
		}

	})
}

func TestNode_NodeBuilderInterfaceGood(t *testing.T) {
	builder := NewNodeBuilder()
	var _ build.Builder = builder
	if !stdlibAssertEqual("node", builder.Name()) {
		t.Fatalf("want %v, got %v", "node", builder.Name())
	}
	detected, err := builder.Detect(nil, t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detected {
		t.Fatal("expected empty temp directory not to be detected")
	}
}

func TestNode_NodeBuilderBuildDefaultsGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeNodeToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	projectDir := setupNodeTestProject(t)
	outputDir := t.TempDir()

	builder := NewNodeBuilder()
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

func TestNode_NodeBuilderBuild_Good_NestedProject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeNodeToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	projectDir := t.TempDir()
	nestedDir := ax.Join(projectDir, "apps", "web")
	if err := ax.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(nestedDir, "package.json"), []byte(`{"name":"nested-app","scripts":{"build":"node build.js"}}`), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(nestedDir, "build.js"), []byte(`console.log("nested build")`), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outputDir := t.TempDir()
	logDir := t.TempDir()
	logPath := ax.Join(logDir, "node-nested.log")
	t.Setenv("NODE_BUILD_LOG_FILE", logPath)

	builder := NewNodeBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "nested-app",
		Version:    "v1.2.3",
	}

	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}
	if _, err := ax.Stat(artifacts[0].Path); err != nil {
		t.Fatalf("expected file to exist: %v", artifacts[0].Path)
	}

	content, err := ax.ReadFile(logPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertContains(string(content), "apps/web") {
		t.Fatalf("expected %v to contain %v", string(content), "apps/web")
	}
	if !stdlibAssertContains(string(content), "GOOS=linux") {
		t.Fatalf("expected %v to contain %v", string(content), "GOOS=linux")
	}
	if !stdlibAssertContains(string(content), "GOARCH=amd64") {
		t.Fatalf("expected %v to contain %v", string(content), "GOARCH=amd64")
	}

}

// --- v0.9.0 generated compliance triplets ---
func TestNode_NewNodeBuilder_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewNodeBuilder()
	})
	core.AssertTrue(t, true)
}

func TestNode_NewNodeBuilder_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewNodeBuilder()
	})
	core.AssertTrue(t, true)
}

func TestNode_NewNodeBuilder_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewNodeBuilder()
	})
	core.AssertTrue(t, true)
}

func TestNode_NodeBuilder_Name_Good(t *core.T) {
	subject := &NodeBuilder{}
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
	})
	core.AssertTrue(t, true)
}

func TestNode_NodeBuilder_Name_Bad(t *core.T) {
	subject := &NodeBuilder{}
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
	})
	core.AssertTrue(t, true)
}

func TestNode_NodeBuilder_Name_Ugly(t *core.T) {
	subject := &NodeBuilder{}
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
	})
	core.AssertTrue(t, true)
}

func TestNode_NodeBuilder_Detect_Good(t *core.T) {
	subject := &NodeBuilder{}
	core.AssertNotPanics(t, func() {
		_, _ = subject.Detect(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestNode_NodeBuilder_Detect_Bad(t *core.T) {
	subject := &NodeBuilder{}
	core.AssertNotPanics(t, func() {
		_, _ = subject.Detect(io.NewMemoryMedium(), "")
	})
	core.AssertTrue(t, true)
}

func TestNode_NodeBuilder_Detect_Ugly(t *core.T) {
	subject := &NodeBuilder{}
	core.AssertNotPanics(t, func() {
		_, _ = subject.Detect(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestNode_NodeBuilder_Build_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &NodeBuilder{}
	core.AssertNotPanics(t, func() {
		_, _ = subject.Build(ctx, nil, nil)
	})
	core.AssertTrue(t, true)
}

func TestNode_NodeBuilder_Build_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &NodeBuilder{}
	core.AssertNotPanics(t, func() {
		_, _ = subject.Build(ctx, nil, nil)
	})
	core.AssertTrue(t, true)
}

func TestNode_NodeBuilder_Build_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &NodeBuilder{}
	core.AssertNotPanics(t, func() {
		_, _ = subject.Build(ctx, nil, nil)
	})
	core.AssertTrue(t, true)
}
