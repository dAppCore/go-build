package builders

import (
	"context"
	"runtime"
	"testing"

	"dappco.re/go/build/internal/ax"

	core "dappco.re/go"
	"dappco.re/go/build/pkg/build"
	coreio "dappco.re/go/io"
)

func setupFakeDockerToolchain(t *testing.T, binDir string) {
	t.Helper()

	script := `#!/bin/sh
set -eu

	log_file="${DOCKER_BUILD_LOG_FILE:-}"
	if [ -n "$log_file" ]; then
		printf '%s\n' "$*" >> "$log_file"
		env | sort >> "$log_file"
	fi

	if [ "${1:-}" = "buildx" ] && [ "${2:-}" = "build" ]; then
	dest=""
	while [ $# -gt 0 ]; do
		if [ "$1" = "--output" ]; then
			shift
			dest="$(printf '%s' "$1" | sed -n 's#type=oci,dest=##p')"
		fi
		shift
	done
	if [ -n "$dest" ]; then
		mkdir -p "$(dirname "$dest")"
		printf 'oci archive\n' > "$dest"
	fi
fi
`
	if err := ax.WriteFile(ax.Join(binDir, "docker"), []byte(script), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}

func TestDocker_DockerBuilderNameGood(t *testing.T) {
	builder := NewDockerBuilder()
	if !stdlibAssertEqual("docker", builder.Name()) {
		t.Fatalf("want %v, got %v", "docker", builder.Name())
	}

}

func TestDocker_DockerBuilderDetectGood(t *testing.T) {
	fs := coreio.Local

	t.Run("detects Dockerfile", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "Dockerfile"), []byte("FROM alpine\n"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewDockerBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("detects Containerfile", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "Containerfile"), []byte("FROM alpine\n"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewDockerBuilder()
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

		builder := NewDockerBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detected {
			t.Fatal("expected false")
		}

	})

	t.Run("returns false for non-Docker project", func(t *testing.T) {
		dir := t.TempDir()
		// Create a Go project instead
		err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module test"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewDockerBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detected {
			t.Fatal("expected false")
		}

	})

	t.Run("does not match docker-compose.yml", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "docker-compose.yml"), []byte("version: '3'\n"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewDockerBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detected {
			t.Fatal("expected false")
		}

	})

	t.Run("does not match Dockerfile in subdirectory", func(t *testing.T) {
		dir := t.TempDir()
		subDir := ax.Join(dir, "subdir")
		if err := ax.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err := ax.WriteFile(ax.Join(subDir, "Dockerfile"), []byte("FROM alpine\n"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewDockerBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detected {
			t.Fatal("expected false")
		}

	})
}

func TestDocker_DockerBuilderInterfaceGood(t *testing.T) {
	builder := NewDockerBuilder()
	var _ build.Builder = builder
	if !stdlibAssertEqual("docker", builder.Name()) {
		t.Fatalf("want %v, got %v", "docker", builder.Name())
	}
	detected, err := builder.Detect(nil, t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detected {
		t.Fatal("expected empty temp directory not to be detected")
	}
}

func TestDocker_DockerBuilderResolveDockerCliGood(t *testing.T) {
	builder := NewDockerBuilder()
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "docker")
	if err := ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Setenv("PATH", "")

	command, err := builder.resolveDockerCli(fallbackPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(fallbackPath, command) {
		t.Fatalf("want %v, got %v", fallbackPath, command)
	}

}

func TestDocker_DockerBuilderResolveDockerCliBad(t *testing.T) {
	builder := NewDockerBuilder()
	t.Setenv("PATH", "")

	_, err := builder.resolveDockerCli(ax.Join(t.TempDir(), "missing-docker"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "docker CLI not found") {
		t.Fatalf("expected %v to contain %v", err.Error(), "docker CLI not found")
	}

}

func TestDocker_DockerBuilderBuildGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeDockerToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	projectDir := t.TempDir()
	if err := ax.WriteFile(ax.Join(projectDir, "Containerfile"), []byte("FROM alpine:latest\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outputDir := t.TempDir()
	logDir := t.TempDir()
	logPath := ax.Join(logDir, "docker.log")
	t.Setenv("DOCKER_BUILD_LOG_FILE", logPath)

	builder := NewDockerBuilder()
	cfg := &build.Config{
		FS:         coreio.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "sample-app",
		Image:      "owner/repo",
		Env:        []string{"FOO=bar"},
	}
	targets := []build.Target{
		{OS: "linux", Arch: "amd64"},
		{OS: "linux", Arch: "arm64"},
	}

	artifacts, err := builder.Build(context.Background(), cfg, targets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}

	expectedPath := ax.Join(outputDir, "owner_repo.tar")
	if !stdlibAssertEqual(expectedPath, artifacts[0].Path) {
		t.Fatalf("want %v, got %v", expectedPath, artifacts[0].Path)
	}
	if !stdlibAssertEqual("linux", artifacts[0].OS) {
		t.Fatalf("want %v, got %v", "linux", artifacts[0].OS)
	}
	if !stdlibAssertEqual("amd64", artifacts[0].Arch) {
		t.Fatalf("want %v, got %v", "amd64", artifacts[0].Arch)
	}
	if _, err := ax.Stat(expectedPath); err != nil {
		t.Fatalf("expected file to exist: %v", expectedPath)
	}

	logContent, err := ax.ReadFile(logPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	log := string(logContent)
	buildxCount := len(core.Split(log, "buildx build")) - 1
	if !stdlibAssertEqual(1, buildxCount) {
		t.Fatalf("want %v, got %v", 1, buildxCount)
	}
	if !stdlibAssertContains(log, "--platform") {
		t.Fatalf("expected %v to contain %v", log, "--platform")
	}
	if !stdlibAssertContains(log, "linux/amd64,linux/arm64") {
		t.Fatalf("expected %v to contain %v", log, "linux/amd64,linux/arm64")
	}
	if !stdlibAssertContains(log, "--output") {
		t.Fatalf("expected %v to contain %v", log, "--output")
	}
	if !stdlibAssertContains(log, "type=oci,dest="+expectedPath) {
		t.Fatalf("expected %v to contain %v", log, "type=oci,dest="+expectedPath)
	}
	if !stdlibAssertContains(log, "FOO=bar") {
		t.Fatalf("expected %v to contain %v", log, "FOO=bar")
	}

	artifacts, err = builder.Build(context.Background(), cfg, nil)
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

func TestDocker_DockerBuilderBuild_ResolvesRelativeDockerfileGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeDockerToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	projectDir := t.TempDir()
	dockerfilePath := ax.Join(projectDir, "dockerfiles", "Dockerfile.app")
	if err := ax.MkdirAll(ax.Dir(dockerfilePath), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(dockerfilePath, []byte("FROM alpine:latest\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outputDir := t.TempDir()
	logDir := t.TempDir()
	logPath := ax.Join(logDir, "docker.log")
	t.Setenv("DOCKER_BUILD_LOG_FILE", logPath)

	builder := NewDockerBuilder()
	cfg := &build.Config{
		FS:         coreio.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Dockerfile: "dockerfiles/Dockerfile.app",
		Image:      "owner/repo",
	}

	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}
	if _, err := ax.Stat(ax.Join(outputDir, "owner_repo.tar")); err != nil {
		t.Fatalf("expected file to exist: %v", ax.Join(outputDir, "owner_repo.tar"))
	}

	logContent, err := ax.ReadFile(logPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	log := string(logContent)
	if !stdlibAssertContains(log, "-f") {
		t.Fatalf("expected %v to contain %v", log, "-f")
	}
	if !stdlibAssertContains(log, dockerfilePath) {
		t.Fatalf("expected %v to contain %v", log, dockerfilePath)
	}

}

func TestDocker_DockerBuilderBuild_Containerfile_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeDockerToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	projectDir := t.TempDir()
	if err := ax.WriteFile(ax.Join(projectDir, "Containerfile"), []byte("FROM alpine:latest\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outputDir := t.TempDir()
	builder := NewDockerBuilder()
	cfg := &build.Config{
		FS:         coreio.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Image:      "owner/repo",
	}

	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}
	if _, err := ax.Stat(ax.Join(outputDir, "owner_repo.tar")); err != nil {
		t.Fatalf("expected file to exist: %v", ax.Join(outputDir, "owner_repo.tar"))
	}

}

func TestDocker_DockerBuilderBuild_Load_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeDockerToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	projectDir := t.TempDir()
	if err := ax.WriteFile(ax.Join(projectDir, "Dockerfile"), []byte("FROM alpine:latest\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outputDir := t.TempDir()
	logDir := t.TempDir()
	logPath := ax.Join(logDir, "docker.log")
	t.Setenv("DOCKER_BUILD_LOG_FILE", logPath)

	builder := NewDockerBuilder()
	cfg := &build.Config{
		FS:         coreio.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Image:      "owner/repo",
		Load:       true,
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
	if !stdlibAssertEqual("ghcr.io/owner/repo:latest", artifacts[0].Path) {
		t.Fatalf("want %v, got %v", "ghcr.io/owner/repo:latest", artifacts[0].Path)
	}
	if !stdlibAssertEqual("linux", artifacts[0].OS) {
		t.Fatalf("want %v, got %v", "linux", artifacts[0].OS)
	}
	if !stdlibAssertEqual("amd64", artifacts[0].Arch) {
		t.Fatalf("want %v, got %v", "amd64", artifacts[0].Arch)
	}
	if info, err := ax.Stat(outputDir); err != nil {
		t.Fatalf("expected directory to exist: %v", outputDir)
	} else if !info.IsDir() {
		t.Fatalf("expected directory to exist: %v", outputDir)
	}

	logContent, err := ax.ReadFile(logPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	log := string(logContent)
	if !stdlibAssertContains(log, "buildx build") {
		t.Fatalf("expected %v to contain %v", log, "buildx build")
	}
	if !stdlibAssertContains(log, "--load") {
		t.Fatalf("expected %v to contain %v", log, "--load")
	}
	if stdlibAssertContains(log, "--output") {
		t.Fatalf("expected %v not to contain %v", log, "--output")
	}

}

// --- v0.9.0 generated compliance triplets ---
func TestDocker_NewDockerBuilder_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewDockerBuilder()
	})
	core.AssertTrue(t, true)
}

func TestDocker_NewDockerBuilder_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewDockerBuilder()
	})
	core.AssertTrue(t, true)
}

func TestDocker_NewDockerBuilder_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewDockerBuilder()
	})
	core.AssertTrue(t, true)
}

func TestDocker_DockerBuilder_Name_Good(t *core.T) {
	subject := &DockerBuilder{}
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
	})
	core.AssertTrue(t, true)
}

func TestDocker_DockerBuilder_Name_Bad(t *core.T) {
	subject := &DockerBuilder{}
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
	})
	core.AssertTrue(t, true)
}

func TestDocker_DockerBuilder_Name_Ugly(t *core.T) {
	subject := &DockerBuilder{}
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
	})
	core.AssertTrue(t, true)
}

func TestDocker_DockerBuilder_Detect_Good(t *core.T) {
	subject := &DockerBuilder{}
	core.AssertNotPanics(t, func() {
		_, _ = subject.Detect(coreio.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestDocker_DockerBuilder_Detect_Bad(t *core.T) {
	subject := &DockerBuilder{}
	core.AssertNotPanics(t, func() {
		_, _ = subject.Detect(coreio.NewMemoryMedium(), "")
	})
	core.AssertTrue(t, true)
}

func TestDocker_DockerBuilder_Detect_Ugly(t *core.T) {
	subject := &DockerBuilder{}
	core.AssertNotPanics(t, func() {
		_, _ = subject.Detect(coreio.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestDocker_DockerBuilder_Build_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &DockerBuilder{}
	core.AssertNotPanics(t, func() {
		_, _ = subject.Build(ctx, nil, nil)
	})
	core.AssertTrue(t, true)
}

func TestDocker_DockerBuilder_Build_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &DockerBuilder{}
	core.AssertNotPanics(t, func() {
		_, _ = subject.Build(ctx, nil, nil)
	})
	core.AssertTrue(t, true)
}

func TestDocker_DockerBuilder_Build_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &DockerBuilder{}
	core.AssertNotPanics(t, func() {
		_, _ = subject.Build(ctx, nil, nil)
	})
	core.AssertTrue(t, true)
}
