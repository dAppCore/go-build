// Package builders provides build implementations for different project types.
package builders

import (
	"context"
	"runtime"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	storage "dappco.re/go/build/pkg/storage"
)

// DockerBuilder builds Docker images.
//
// b := builders.NewDockerBuilder()
type DockerBuilder struct{}

// NewDockerBuilder creates a new Docker builder.
//
// b := builders.NewDockerBuilder()
func NewDockerBuilder() *DockerBuilder {
	return &DockerBuilder{}
}

// Name returns the builder's identifier.
//
// name := b.Name() // → "docker"
func (b *DockerBuilder) Name() string {
	return "docker"
}

// Detect checks if a Dockerfile or Containerfile exists in the directory.
//
// ok, err := b.Detect(storage.Local, ".")
func (b *DockerBuilder) Detect(fs storage.Medium, dir string) core.Result {
	if build.ResolveDockerfilePath(fs, dir) != "" {
		return core.Ok(true)
	}
	return core.Ok(false)
}

// Build builds Docker images for the specified targets.
//
// artifacts, err := b.Build(ctx, cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
func (b *DockerBuilder) Build(ctx context.Context, cfg *build.Config, targets []build.Target) core.Result {
	if cfg == nil {
		return core.Fail(core.E("DockerBuilder.Build", "config is nil", nil))
	}
	filesystem := ensureBuildFilesystem(cfg)

	dockerCommandResult := b.resolveDockerCli()
	if !dockerCommandResult.OK {
		return dockerCommandResult
	}
	dockerCommand := dockerCommandResult.Value.(string)

	// Ensure buildx is available
	ensured := b.ensureBuildx(ctx, dockerCommand)
	if !ensured.OK {
		return ensured
	}

	// Determine Docker manifest path
	dockerfile := cfg.Dockerfile
	if dockerfile == "" {
		dockerfile = build.ResolveDockerfilePath(filesystem, cfg.ProjectDir)
	} else if !ax.IsAbs(dockerfile) {
		dockerfile = ax.Join(cfg.ProjectDir, dockerfile)
	}

	// Validate Dockerfile exists
	if dockerfile == "" || !filesystem.IsFile(dockerfile) {
		return core.Fail(core.E("DockerBuilder.Build", "Dockerfile or Containerfile not found", nil))
	}

	// Determine image name
	imageName := cfg.Image
	if imageName == "" {
		imageName = cfg.Name
	}
	if imageName == "" {
		imageName = ax.Base(cfg.ProjectDir)
	}

	// Build platform string from targets
	buildTargets := targets
	if len(buildTargets) == 0 {
		buildTargets = []build.Target{{OS: runtime.GOOS, Arch: runtime.GOARCH}}
	}

	var platforms []string
	for _, t := range buildTargets {
		platforms = append(platforms, core.Sprintf("%s/%s", t.OS, t.Arch))
	}

	// Determine registry
	registry := cfg.Registry
	if registry == "" {
		registry = "ghcr.io"
	}

	// Determine tags
	tags := cfg.Tags
	if len(tags) == 0 {
		tags = []string{"latest"}
		if cfg.Version != "" {
			tags = append(tags, cfg.Version)
		}
	}

	// Build full image references
	var imageRefs []string
	for _, tag := range tags {
		expandedTag := build.ExpandVersionTemplate(tag, cfg.Version)

		if registry != "" {
			imageRefs = append(imageRefs, core.Sprintf("%s/%s:%s", registry, imageName, expandedTag))
		} else {
			imageRefs = append(imageRefs, core.Sprintf("%s:%s", imageName, expandedTag))
		}
	}

	// Build the docker buildx command
	args := []string{"buildx", "build"}

	// Multi-platform support
	args = append(args, "--platform", core.Join(",", platforms...))

	// Add all tags
	for _, ref := range imageRefs {
		args = append(args, "-t", ref)
	}

	// Dockerfile path
	args = append(args, "-f", dockerfile)

	// Build arguments
	for k, v := range cfg.BuildArgs {
		expandedValue := build.ExpandVersionTemplate(v, cfg.Version)
		args = append(args, "--build-arg", core.Sprintf("%s=%s", k, expandedValue))
	}

	// Always add VERSION build arg if version is set
	if cfg.Version != "" {
		args = append(args, "--build-arg", core.Sprintf("VERSION=%s", cfg.Version))
	}

	safeImageName := core.Replace(imageName, "/", "_")

	// Output to local docker images or push.
	// `--load` only works for a single target, so multi-platform local builds
	// fall back to an OCI archive on disk.
	useLoad := cfg.Load && !cfg.Push && len(buildTargets) == 1
	if cfg.Push {
		args = append(args, "--push")
	} else if useLoad {
		args = append(args, "--load")
	} else {
		// Local Docker builds emit an OCI archive so the build output is a file.
		outputPath := ax.Join(cfg.OutputDir, core.Sprintf("%s.tar", safeImageName))
		args = append(args, "--output", core.Sprintf("type=oci,dest=%s", outputPath))
	}

	// Build context (project directory)
	args = append(args, cfg.ProjectDir)

	// Create output directory
	created := filesystem.EnsureDir(cfg.OutputDir)
	if !created.OK {
		return core.Fail(core.E("DockerBuilder.Build", "failed to create output directory", core.NewError(created.Error())))
	}

	core.Print(nil, "Building Docker image: %s", imageName)
	core.Print(nil, "  Platforms: %s", core.Join(", ", platforms...))
	core.Print(nil, "  Tags: %s", core.Join(", ", imageRefs...))

	// Build once for the full platform set. Docker buildx produces a single
	// multi-arch image or OCI archive from the combined platform list.
	executed := ax.ExecWithEnv(ctx, cfg.ProjectDir, build.BuildEnvironment(cfg), dockerCommand, args...)
	if !executed.OK {
		return core.Fail(core.E("DockerBuilder.Build", "buildx build failed", core.NewError(executed.Error())))
	}

	artifactPath := imageRefs[0]
	if !cfg.Push && !useLoad {
		artifactPath = ax.Join(cfg.OutputDir, core.Sprintf("%s.tar", safeImageName))
	}

	primaryTarget := buildTargets[0]
	return core.Ok([]build.Artifact{{
		Path: artifactPath,
		OS:   primaryTarget.OS,
		Arch: primaryTarget.Arch,
	}})
}

// resolveDockerCli returns the executable path for the docker CLI.
func (b *DockerBuilder) resolveDockerCli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/docker",
			"/opt/homebrew/bin/docker",
			"/Applications/Docker.app/Contents/Resources/bin/docker",
		}
	}

	command := ax.ResolveCommand("docker", paths...)
	if !command.OK {
		return core.Fail(core.E("DockerBuilder.resolveDockerCli", "docker CLI not found. Install it from https://docs.docker.com/get-docker/", core.NewError(command.Error())))
	}

	return command
}

// ensureBuildx ensures docker buildx is available and has a builder.
func (b *DockerBuilder) ensureBuildx(ctx context.Context, dockerCommand string) core.Result {
	// Check if buildx is available
	version := ax.Exec(ctx, dockerCommand, "buildx", "version")
	if !version.OK {
		return core.Fail(core.E("DockerBuilder.ensureBuildx", "buildx is not available. Install it from https://docs.docker.com/buildx/working-with-buildx/", core.NewError(version.Error())))
	}

	// Check if we have a builder, create one if not
	inspected := ax.Exec(ctx, dockerCommand, "buildx", "inspect", "--bootstrap")
	if !inspected.OK {
		// Try to create a builder
		created := ax.Exec(ctx, dockerCommand, "buildx", "create", "--use", "--bootstrap")
		if !created.OK {
			return core.Fail(core.E("DockerBuilder.ensureBuildx", "failed to create buildx builder", core.NewError(created.Error())))
		}
	}

	return core.Ok(nil)
}
