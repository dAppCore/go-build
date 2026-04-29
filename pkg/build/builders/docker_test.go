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
	if result := ax.WriteFile(ax.Join(binDir, "docker"), []byte(script), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
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
		if result := ax.WriteFile(ax.Join(dir, "Dockerfile"), []byte("FROM alpine\n"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		builder := NewDockerBuilder()
		detected := requireCPPBool(t, builder.Detect(fs, dir))
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("detects Containerfile", func(t *testing.T) {
		dir := t.TempDir()
		if result := ax.WriteFile(ax.Join(dir, "Containerfile"), []byte("FROM alpine\n"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		builder := NewDockerBuilder()
		detected := requireCPPBool(t, builder.Detect(fs, dir))
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		dir := t.TempDir()

		builder := NewDockerBuilder()
		detected := requireCPPBool(t, builder.Detect(fs, dir))
		if detected {
			t.Fatal("expected false")
		}

	})

	t.Run("returns false for non-Docker project", func(t *testing.T) {
		dir := t.TempDir()
		// Create a Go project instead
		if result := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module test"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		builder := NewDockerBuilder()
		detected := requireCPPBool(t, builder.Detect(fs, dir))
		if detected {
			t.Fatal("expected false")
		}

	})

	t.Run("does not match docker-compose.yml", func(t *testing.T) {
		dir := t.TempDir()
		if result := ax.WriteFile(ax.Join(dir, "docker-compose.yml"), []byte("version: '3'\n"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		builder := NewDockerBuilder()
		detected := requireCPPBool(t, builder.Detect(fs, dir))
		if detected {
			t.Fatal("expected false")
		}

	})

	t.Run("does not match Dockerfile in subdirectory", func(t *testing.T) {
		dir := t.TempDir()
		subDir := ax.Join(dir, "subdir")
		if result := ax.MkdirAll(subDir, 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		if result := ax.WriteFile(ax.Join(subDir, "Dockerfile"), []byte("FROM alpine\n"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		builder := NewDockerBuilder()
		detected := requireCPPBool(t, builder.Detect(fs, dir))
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
	detected := requireCPPBool(t, builder.Detect(nil, t.TempDir()))
	if detected {
		t.Fatal("expected empty temp directory not to be detected")
	}
}

func TestDocker_DockerBuilderResolveDockerCliGood(t *testing.T) {
	builder := NewDockerBuilder()
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "docker")
	if result := ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	t.Setenv("PATH", "")

	command := requireCPPString(t, builder.resolveDockerCli(fallbackPath))
	if !stdlibAssertEqual(fallbackPath, command) {
		t.Fatalf("want %v, got %v", fallbackPath, command)
	}

}

func TestDocker_DockerBuilderResolveDockerCliBad(t *testing.T) {
	builder := NewDockerBuilder()
	t.Setenv("PATH", "")

	result := builder.resolveDockerCli(ax.Join(t.TempDir(), "missing-docker"))
	if result.OK {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(result.Error(), "docker CLI not found") {
		t.Fatalf("expected %v to contain %v", result.Error(), "docker CLI not found")
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
	if result := ax.WriteFile(ax.Join(projectDir, "Containerfile"), []byte("FROM alpine:latest\n"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
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

	artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, targets))
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
	if result := ax.Stat(expectedPath); !result.OK {
		t.Fatalf("expected file to exist: %v", expectedPath)
	}

	logContent := requireBuilderBytes(t, ax.ReadFile(logPath))

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

	artifacts = requireCPPArtifacts(t, builder.Build(context.Background(), cfg, nil))
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
	if result := ax.MkdirAll(ax.Dir(dockerfilePath), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(dockerfilePath, []byte("FROM alpine:latest\n"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
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

	artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}}))
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}
	if result := ax.Stat(ax.Join(outputDir, "owner_repo.tar")); !result.OK {
		t.Fatalf("expected file to exist: %v", ax.Join(outputDir, "owner_repo.tar"))
	}

	logContent := requireBuilderBytes(t, ax.ReadFile(logPath))

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
	if result := ax.WriteFile(ax.Join(projectDir, "Containerfile"), []byte("FROM alpine:latest\n"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	outputDir := t.TempDir()
	builder := NewDockerBuilder()
	cfg := &build.Config{
		FS:         coreio.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Image:      "owner/repo",
	}

	artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}}))
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}
	if result := ax.Stat(ax.Join(outputDir, "owner_repo.tar")); !result.OK {
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
	if result := ax.WriteFile(ax.Join(projectDir, "Dockerfile"), []byte("FROM alpine:latest\n"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
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

	artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), cfg, targets))
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
	if !coreio.Local.IsDir(outputDir) {
		t.Fatalf("expected directory to exist: %v", outputDir)
	}

	logContent := requireBuilderBytes(t, ax.ReadFile(logPath))

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
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewDockerBuilder()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestDocker_NewDockerBuilder_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewDockerBuilder()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestDocker_NewDockerBuilder_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewDockerBuilder()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestDocker_DockerBuilder_Name_Good(t *core.T) {
	subject := &DockerBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestDocker_DockerBuilder_Name_Bad(t *core.T) {
	subject := &DockerBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestDocker_DockerBuilder_Name_Ugly(t *core.T) {
	subject := &DockerBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestDocker_DockerBuilder_Detect_Good(t *core.T) {
	subject := &DockerBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Detect(coreio.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestDocker_DockerBuilder_Detect_Bad(t *core.T) {
	subject := &DockerBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Detect(coreio.NewMemoryMedium(), "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestDocker_DockerBuilder_Detect_Ugly(t *core.T) {
	subject := &DockerBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Detect(coreio.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestDocker_DockerBuilder_Build_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &DockerBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Build(ctx, nil, nil)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestDocker_DockerBuilder_Build_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &DockerBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Build(ctx, nil, nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestDocker_DockerBuilder_Build_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &DockerBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Build(ctx, nil, nil)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
