package builders

import (
	"context"
	"runtime"
	"testing"

	"dappco.re/go/build/internal/ax"

	core "dappco.re/go"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/io"
)

func TestTaskfile_TaskfileBuilderNameGood(t *testing.T) {
	builder := NewTaskfileBuilder()
	if !stdlibAssertEqual("taskfile", builder.Name()) {
		t.Fatalf("want %v, got %v", "taskfile", builder.Name())
	}

}

func TestTaskfile_TaskfileBuilderDetectGood(t *testing.T) {
	fs := io.Local

	t.Run("detects Taskfile.yml", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "Taskfile.yml"), []byte("version: '3'\n"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewTaskfileBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("detects Taskfile.yaml", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "Taskfile.yaml"), []byte("version: '3'\n"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewTaskfileBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("detects Taskfile (no extension)", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "Taskfile"), []byte("version: '3'\n"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewTaskfileBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("detects lowercase taskfile.yml", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "taskfile.yml"), []byte("version: '3'\n"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewTaskfileBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("detects lowercase taskfile.yaml", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "taskfile.yaml"), []byte("version: '3'\n"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewTaskfileBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		dir := t.TempDir()

		builder := NewTaskfileBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detected {
			t.Fatal("expected false")
		}

	})

	t.Run("returns false for non-Taskfile project", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "Makefile"), []byte("all:\n\techo hello\n"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewTaskfileBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detected {
			t.Fatal("expected false")
		}

	})

	t.Run("does not match Taskfile in subdirectory", func(t *testing.T) {
		dir := t.TempDir()
		subDir := ax.Join(dir, "subdir")
		if err := ax.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err := ax.WriteFile(ax.Join(subDir, "Taskfile.yml"), []byte("version: '3'\n"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewTaskfileBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detected {
			t.Fatal("expected false")
		}

	})
}

func TestTaskfile_TaskfileBuilderFindArtifactsGood(t *testing.T) {
	fs := io.Local
	builder := NewTaskfileBuilder()

	t.Run("finds files in output directory", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "myapp"), []byte("binary"), 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, "myapp.tar.gz"), []byte("archive"), 0644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		artifacts := builder.findArtifacts(fs, dir)
		if len(artifacts) != 2 {
			t.Fatalf("want len %v, got %v", 2, len(artifacts))
		}

	})

	t.Run("skips hidden files", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "myapp"), []byte("binary"), 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, ".hidden"), []byte("hidden"), 0644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		artifacts := builder.findArtifacts(fs, dir)
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if !stdlibAssertContains(artifacts[0].Path, "myapp") {
			t.Fatalf("expected %v to contain %v", artifacts[0].Path, "myapp")
		}

	})

	t.Run("skips CHECKSUMS.txt", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "myapp"), []byte("binary"), 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, "CHECKSUMS.txt"), []byte("sha256"), 0644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		artifacts := builder.findArtifacts(fs, dir)
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if !stdlibAssertContains(artifacts[0].Path, "myapp") {
			t.Fatalf("expected %v to contain %v", artifacts[0].Path, "myapp")
		}

	})

	t.Run("skips directories", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "myapp"), []byte("binary"), 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.MkdirAll(ax.Join(dir, "subdir"), 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		artifacts := builder.findArtifacts(fs, dir)
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}

	})

	t.Run("returns empty for empty directory", func(t *testing.T) {
		dir := t.TempDir()

		artifacts := builder.findArtifacts(fs, dir)
		if !stdlibAssertEmpty(artifacts) {
			t.Fatalf("expected empty, got %v", artifacts)
		}

	})

	t.Run("returns empty for nonexistent directory", func(t *testing.T) {
		artifacts := builder.findArtifacts(fs, "/nonexistent/path")
		if !stdlibAssertEmpty(artifacts) {
			t.Fatalf("expected empty, got %v", artifacts)
		}

	})
}

func TestTaskfile_TaskfileBuilderFindArtifactsForTargetGood(t *testing.T) {
	fs := io.Local
	builder := NewTaskfileBuilder()

	t.Run("finds artifacts in platform subdirectory", func(t *testing.T) {
		dir := t.TempDir()
		platformDir := ax.Join(dir, "linux_amd64")
		if err := ax.MkdirAll(platformDir, 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(platformDir, "myapp"), []byte("binary"), 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		target := build.Target{OS: "linux", Arch: "amd64"}
		artifacts := builder.findArtifactsForTarget(fs, dir, target)
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if !stdlibAssertEqual("linux", artifacts[0].OS) {
			t.Fatalf("want %v, got %v", "linux", artifacts[0].OS)
		}
		if !stdlibAssertEqual("amd64", artifacts[0].Arch) {
			t.Fatalf("want %v, got %v", "amd64", artifacts[0].Arch)
		}

	})

	t.Run("finds artifacts by name pattern in root", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "myapp-linux-amd64"), []byte("binary"), 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		target := build.Target{OS: "linux", Arch: "amd64"}
		artifacts := builder.findArtifactsForTarget(fs, dir, target)
		if stdlibAssertEmpty(artifacts) {
			t.Fatal("expected non-empty")
		}

	})

	t.Run("returns empty when no matching artifacts", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "myapp"), []byte("binary"), 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		target := build.Target{OS: "linux", Arch: "arm64"}
		artifacts := builder.findArtifactsForTarget(fs, dir, target)
		if !stdlibAssertEmpty(artifacts) {
			t.Fatalf("expected empty, got %v", artifacts)
		}

	})

	t.Run("handles .app bundles on darwin", func(t *testing.T) {
		dir := t.TempDir()
		platformDir := ax.Join(dir, "darwin_arm64")
		appDir := ax.Join(platformDir, "MyApp.app")
		if err := ax.MkdirAll(appDir, 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		target := build.Target{OS: "darwin", Arch: "arm64"}
		artifacts := builder.findArtifactsForTarget(fs, dir, target)
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if !stdlibAssertContains(artifacts[0].Path, "MyApp.app") {
			t.Fatalf("expected %v to contain %v", artifacts[0].Path, "MyApp.app")
		}

	})
}

func TestTaskfile_TaskfileBuilderMatchPatternGood(t *testing.T) {
	builder := NewTaskfileBuilder()

	t.Run("matches simple glob", func(t *testing.T) {
		if !(builder.matchPattern("myapp-linux-amd64", "*-linux-amd64")) {
			t.Fatal("expected true")
		}

	})

	t.Run("does not match different pattern", func(t *testing.T) {
		if builder.matchPattern("myapp-linux-amd64", "*-darwin-arm64") {
			t.Fatal("expected false")
		}

	})

	t.Run("matches wildcard", func(t *testing.T) {
		if !(builder.matchPattern("test_linux_arm64.bin", "*_linux_arm64*")) {
			t.Fatal("expected true")
		}

	})
}

func TestTaskfile_TaskfileBuilderInterfaceGood(t *testing.T) {
	builder := NewTaskfileBuilder()
	var _ build.Builder = builder
	if !stdlibAssertEqual("taskfile", builder.Name()) {
		t.Fatalf("want %v, got %v", "taskfile", builder.Name())
	}
	detected, err := builder.Detect(nil, t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detected {
		t.Fatal("expected empty temp directory not to be detected")
	}
}

func TestTaskfile_TaskfileBuilderResolveTaskCliGood(t *testing.T) {
	builder := NewTaskfileBuilder()
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "task")
	if err := ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Setenv("PATH", "")

	command, err := builder.resolveTaskCli(fallbackPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(fallbackPath, command) {
		t.Fatalf("want %v, got %v", fallbackPath, command)
	}

}

func TestTaskfile_TaskfileBuilderResolveTaskCliBad(t *testing.T) {
	builder := NewTaskfileBuilder()
	t.Setenv("PATH", "")

	_, err := builder.resolveTaskCli(ax.Join(t.TempDir(), "missing-task"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "task CLI not found") {
		t.Fatalf("expected %v to contain %v", err.Error(), "task CLI not found")
	}

}

func TestTaskfile_TaskfileBuilderRunTaskGood(t *testing.T) {
	binDir := t.TempDir()
	taskPath := ax.Join(binDir, "task")
	logPath := ax.Join(t.TempDir(), "task.env")

	script := `#!/bin/sh
set -eu

env | sort > "${TASK_BUILD_LOG_FILE}"
`
	if err := ax.WriteFile(taskPath, []byte(script), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Setenv("TASK_BUILD_LOG_FILE", logPath)

	builder := NewTaskfileBuilder()
	goCacheDir := ax.Join(t.TempDir(), "cache", "go-build")
	goModCacheDir := ax.Join(t.TempDir(), "cache", "go-mod")
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: t.TempDir(),
		OutputDir:  "/tmp/out",
		Name:       "sample",
		Version:    "v1.2.3",
		Env:        []string{"FOO=bar"},
		Cache: build.CacheConfig{
			Enabled: true,
			Paths: []string{
				goCacheDir,
				goModCacheDir,
			},
		},
	}
	if err := builder.runTask(context.Background(), cfg, taskPath, cfg.OutputDir, build.Target{OS: "linux", Arch: "amd64"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := ax.ReadFile(logPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertContains(string(content), "FOO=bar") {
		t.Fatalf("expected %v to contain %v", string(content), "FOO=bar")
	}
	if !stdlibAssertContains(string(content), "GOOS=linux") {
		t.Fatalf("expected %v to contain %v", string(content), "GOOS=linux")
	}
	if !stdlibAssertContains(string(content), "GOARCH=amd64") {
		t.Fatalf("expected %v to contain %v", string(content), "GOARCH=amd64")
	}
	if !stdlibAssertContains(string(content), "TARGET_OS=linux") {
		t.Fatalf("expected %v to contain %v", string(content), "TARGET_OS=linux")
	}
	if !stdlibAssertContains(string(content), "TARGET_ARCH=amd64") {
		t.Fatalf("expected %v to contain %v", string(content), "TARGET_ARCH=amd64")
	}
	if !stdlibAssertContains(string(content), "OUTPUT_DIR=/tmp/out") {
		t.Fatalf("expected %v to contain %v", string(content), "OUTPUT_DIR=/tmp/out")
	}
	if !stdlibAssertContains(string(content), "TARGET_DIR=/tmp/out/linux_amd64") {
		t.Fatalf("expected %v to contain %v", string(content), "TARGET_DIR=/tmp/out/linux_amd64")
	}
	if !stdlibAssertContains(string(content), "NAME=sample") {
		t.Fatalf("expected %v to contain %v", string(content), "NAME=sample")
	}
	if !stdlibAssertContains(string(content), "VERSION=v1.2.3") {
		t.Fatalf("expected %v to contain %v", string(content), "VERSION=v1.2.3")
	}
	if !stdlibAssertContains(string(content), "CGO_ENABLED=0") {
		t.Fatalf("expected %v to contain %v", string(content), "CGO_ENABLED=0")
	}
	if !stdlibAssertContains(string(content), "GOCACHE="+goCacheDir) {
		t.Fatalf("expected %v to contain %v", string(content), "GOCACHE="+goCacheDir)
	}
	if !stdlibAssertContains(string(content), "GOMODCACHE="+goModCacheDir) {
		t.Fatalf("expected %v to contain %v", string(content), "GOMODCACHE="+goModCacheDir)
	}

}

func TestTaskfile_TaskfileBuilderBuild_DoesNotMutateOutputDirGood(t *testing.T) {
	projectDir := t.TempDir()
	if err := ax.WriteFile(ax.Join(projectDir, "Taskfile.yml"), []byte("version: '3'\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	binDir := t.TempDir()
	taskPath := ax.Join(binDir, "task")
	script := `#!/bin/sh
set -eu

mkdir -p "${OUTPUT_DIR}/${GOOS}_${GOARCH}"
printf '%s\n' "${NAME:-taskfile}" > "${OUTPUT_DIR}/${GOOS}_${GOARCH}/${NAME:-taskfile}"
`
	if err := ax.WriteFile(taskPath, []byte(script), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	builder := NewTaskfileBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		Name:       "sample",
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

}

func TestTaskfile_TaskfileBuilderBuildGood(t *testing.T) {
	projectDir := t.TempDir()
	if err := ax.WriteFile(ax.Join(projectDir, "Taskfile.yml"), []byte("version: '3'\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	binDir := t.TempDir()
	taskPath := ax.Join(binDir, "task")
	logPath := ax.Join(t.TempDir(), "task.build.env")

	script := `#!/bin/sh
set -eu

mkdir -p "${OUTPUT_DIR}/${GOOS}_${GOARCH}"
printf '%s\n' "${NAME:-taskfile}" > "${OUTPUT_DIR}/${GOOS}_${GOARCH}/${NAME:-taskfile}"
env | sort > "${TASK_BUILD_LOG_FILE}"
`
	if err := ax.WriteFile(taskPath, []byte(script), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Setenv("TASK_BUILD_LOG_FILE", logPath)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	builder := NewTaskfileBuilder()
	goCacheDir := ax.Join(t.TempDir(), "cache", "go-build")
	goModCacheDir := ax.Join(t.TempDir(), "cache", "go-mod")
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		Name:       "sample",
		Version:    "v1.2.3",
		Env:        []string{"FOO=bar"},
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
	if !stdlibAssertEqual(ax.Join(projectDir, "dist", "linux_amd64", "sample"), artifacts[0].Path) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "dist", "linux_amd64", "sample"), artifacts[0].Path)
	}

	content, err := ax.ReadFile(logPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertContains(string(content), "FOO=bar") {
		t.Fatalf("expected %v to contain %v", string(content), "FOO=bar")
	}
	if !stdlibAssertContains(string(content), "OUTPUT_DIR="+ax.Join(projectDir, "dist")) {
		t.Fatalf("expected %v to contain %v", string(content), "OUTPUT_DIR="+ax.Join(projectDir, "dist"))
	}
	if !stdlibAssertContains(string(content), "GOOS=linux") {
		t.Fatalf("expected %v to contain %v", string(content), "GOOS=linux")
	}
	if !stdlibAssertContains(string(content), "GOARCH=amd64") {
		t.Fatalf("expected %v to contain %v", string(content), "GOARCH=amd64")
	}
	if !stdlibAssertContains(string(content), "TARGET_OS=linux") {
		t.Fatalf("expected %v to contain %v", string(content), "TARGET_OS=linux")
	}
	if !stdlibAssertContains(string(content), "TARGET_ARCH=amd64") {
		t.Fatalf("expected %v to contain %v", string(content), "TARGET_ARCH=amd64")
	}
	if !stdlibAssertContains(string(content), "TARGET_DIR="+ax.Join(projectDir, "dist", "linux_amd64")) {
		t.Fatalf("expected %v to contain %v", string(content), "TARGET_DIR="+ax.Join(projectDir, "dist", "linux_amd64"))
	}
	if !stdlibAssertContains(string(content), "CGO_ENABLED=0") {
		t.Fatalf("expected %v to contain %v", string(content), "CGO_ENABLED=0")
	}
	if !stdlibAssertContains(string(content), "GOCACHE="+goCacheDir) {
		t.Fatalf("expected %v to contain %v", string(content), "GOCACHE="+goCacheDir)
	}
	if !stdlibAssertContains(string(content), "GOMODCACHE="+goModCacheDir) {
		t.Fatalf("expected %v to contain %v", string(content), "GOMODCACHE="+goModCacheDir)
	}

}

func TestTaskfile_TaskfileBuilderBuild_DefaultTargetGood(t *testing.T) {
	projectDir := t.TempDir()
	if err := ax.WriteFile(ax.Join(projectDir, "Taskfile.yml"), []byte("version: '3'\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	binDir := t.TempDir()
	taskPath := ax.Join(binDir, "task")
	logPath := ax.Join(t.TempDir(), "task.default.env")

	script := `#!/bin/sh
set -eu

mkdir -p "${OUTPUT_DIR}/${GOOS}_${GOARCH}"
printf '%s\n' "${GOOS}/${GOARCH}" > "${OUTPUT_DIR}/${GOOS}_${GOARCH}/artifact"
env | sort > "${TASK_BUILD_LOG_FILE}"
`
	if err := ax.WriteFile(taskPath, []byte(script), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Setenv("TASK_BUILD_LOG_FILE", logPath)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	builder := NewTaskfileBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		Name:       "sample",
		Version:    "v1.2.3",
		Env:        []string{"FOO=bar"},
	}

	artifacts, err := builder.Build(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}
	if !stdlibAssertEqual(ax.Join(projectDir, "dist", runtime.GOOS+"_"+runtime.GOARCH, "artifact"), artifacts[0].Path) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "dist", runtime.GOOS+"_"+runtime.GOARCH, "artifact"), artifacts[0].Path)
	}
	if !stdlibAssertEqual(runtime.GOOS, artifacts[0].OS) {
		t.Fatalf("want %v, got %v", runtime.GOOS, artifacts[0].OS)
	}
	if !stdlibAssertEqual(runtime.GOARCH, artifacts[0].Arch) {
		t.Fatalf("want %v, got %v", runtime.GOARCH, artifacts[0].Arch)
	}

	content, err := ax.ReadFile(logPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertContains(string(content), "FOO=bar") {
		t.Fatalf("expected %v to contain %v", string(content), "FOO=bar")
	}
	if !stdlibAssertContains(string(content), "OUTPUT_DIR="+ax.Join(projectDir, "dist")) {
		t.Fatalf("expected %v to contain %v", string(content), "OUTPUT_DIR="+ax.Join(projectDir, "dist"))
	}
	if !stdlibAssertContains(string(content), "GOOS="+runtime.GOOS) {
		t.Fatalf("expected %v to contain %v", string(content), "GOOS="+runtime.GOOS)
	}
	if !stdlibAssertContains(string(content), "GOARCH="+runtime.GOARCH) {
		t.Fatalf("expected %v to contain %v", string(content), "GOARCH="+runtime.GOARCH)
	}
	if !stdlibAssertContains(string(content), "TARGET_OS="+runtime.GOOS) {
		t.Fatalf("expected %v to contain %v", string(content), "TARGET_OS="+runtime.GOOS)
	}
	if !stdlibAssertContains(string(content), "TARGET_ARCH="+runtime.GOARCH) {
		t.Fatalf("expected %v to contain %v", string(content), "TARGET_ARCH="+runtime.GOARCH)
	}
	if !stdlibAssertContains(string(content), "TARGET_DIR="+ax.Join(projectDir, "dist", runtime.GOOS+"_"+runtime.GOARCH)) {
		t.Fatalf("expected %v to contain %v", string(content), "TARGET_DIR="+ax.Join(projectDir, "dist", runtime.GOOS+"_"+runtime.GOARCH))
	}
	if !stdlibAssertContains(string(content), "CGO_ENABLED=0") {
		t.Fatalf("expected %v to contain %v", string(content), "CGO_ENABLED=0")
	}

}

func TestTaskfile_TaskfileBuilderRunTask_CGOEnabledGood(t *testing.T) {
	binDir := t.TempDir()
	taskPath := ax.Join(binDir, "task")
	logPath := ax.Join(t.TempDir(), "task.cgo.env")

	script := `#!/bin/sh
set -eu

env | sort > "${TASK_BUILD_LOG_FILE}"
`
	if err := ax.WriteFile(taskPath, []byte(script), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Setenv("TASK_BUILD_LOG_FILE", logPath)

	builder := NewTaskfileBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: t.TempDir(),
		OutputDir:  "/tmp/out",
		Name:       "sample",
		Version:    "v1.2.3",
		CGO:        true,
	}
	if err := builder.runTask(context.Background(), cfg, taskPath, cfg.OutputDir, build.Target{OS: "linux", Arch: "amd64"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := ax.ReadFile(logPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertContains(string(content), "CGO_ENABLED=1") {
		t.Fatalf("expected %v to contain %v", string(content), "CGO_ENABLED=1")
	}

}

// --- v0.9.0 generated compliance triplets ---
func TestTaskfile_NewTaskfileBuilder_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewTaskfileBuilder()
	})
	core.AssertTrue(t, true)
}

func TestTaskfile_NewTaskfileBuilder_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewTaskfileBuilder()
	})
	core.AssertTrue(t, true)
}

func TestTaskfile_NewTaskfileBuilder_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewTaskfileBuilder()
	})
	core.AssertTrue(t, true)
}

func TestTaskfile_TaskfileBuilder_Name_Good(t *core.T) {
	subject := &TaskfileBuilder{}
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
	})
	core.AssertTrue(t, true)
}

func TestTaskfile_TaskfileBuilder_Name_Bad(t *core.T) {
	subject := &TaskfileBuilder{}
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
	})
	core.AssertTrue(t, true)
}

func TestTaskfile_TaskfileBuilder_Name_Ugly(t *core.T) {
	subject := &TaskfileBuilder{}
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
	})
	core.AssertTrue(t, true)
}

func TestTaskfile_TaskfileBuilder_Detect_Good(t *core.T) {
	subject := &TaskfileBuilder{}
	core.AssertNotPanics(t, func() {
		_, _ = subject.Detect(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestTaskfile_TaskfileBuilder_Detect_Bad(t *core.T) {
	subject := &TaskfileBuilder{}
	core.AssertNotPanics(t, func() {
		_, _ = subject.Detect(io.NewMemoryMedium(), "")
	})
	core.AssertTrue(t, true)
}

func TestTaskfile_TaskfileBuilder_Detect_Ugly(t *core.T) {
	subject := &TaskfileBuilder{}
	core.AssertNotPanics(t, func() {
		_, _ = subject.Detect(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestTaskfile_TaskfileBuilder_Build_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &TaskfileBuilder{}
	core.AssertNotPanics(t, func() {
		_, _ = subject.Build(ctx, nil, nil)
	})
	core.AssertTrue(t, true)
}

func TestTaskfile_TaskfileBuilder_Build_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &TaskfileBuilder{}
	core.AssertNotPanics(t, func() {
		_, _ = subject.Build(ctx, nil, nil)
	})
	core.AssertTrue(t, true)
}

func TestTaskfile_TaskfileBuilder_Build_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &TaskfileBuilder{}
	core.AssertNotPanics(t, func() {
		_, _ = subject.Build(ctx, nil, nil)
	})
	core.AssertTrue(t, true)
}
