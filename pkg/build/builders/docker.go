// Package builders provides build implementations for different project types.
package builders

import (
	"context"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

// DockerBuilder builds Docker images.
// Usage example: declare a value of type builders.DockerBuilder in integrating code.
type DockerBuilder struct{}

// NewDockerBuilder creates a new Docker builder.
// Usage example: call builders.NewDockerBuilder(...) from integrating code.
func NewDockerBuilder() *DockerBuilder {
	return &DockerBuilder{}
}

// Name returns the builder's identifier.
// Usage example: call value.Name(...) from integrating code.
func (b *DockerBuilder) Name() string {
	return "docker"
}

// Detect checks if a Dockerfile exists in the directory.
// Usage example: call value.Detect(...) from integrating code.
func (b *DockerBuilder) Detect(fs io.Medium, dir string) (bool, error) {
	dockerfilePath := ax.Join(dir, "Dockerfile")
	if fs.IsFile(dockerfilePath) {
		return true, nil
	}
	return false, nil
}

// Build builds Docker images for the specified targets.
// Usage example: call value.Build(...) from integrating code.
func (b *DockerBuilder) Build(ctx context.Context, cfg *build.Config, targets []build.Target) ([]build.Artifact, error) {
	dockerCommand, err := b.resolveDockerCli()
	if err != nil {
		return nil, err
	}

	// Ensure buildx is available
	if err := b.ensureBuildx(ctx, dockerCommand); err != nil {
		return nil, err
	}

	// Determine Dockerfile path
	dockerfile := cfg.Dockerfile
	if dockerfile == "" {
		dockerfile = ax.Join(cfg.ProjectDir, "Dockerfile")
	}

	// Validate Dockerfile exists
	if !cfg.FS.IsFile(dockerfile) {
		return nil, coreerr.E("DockerBuilder.Build", "Dockerfile not found: "+dockerfile, nil)
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
	var platforms []string
	for _, t := range targets {
		platforms = append(platforms, core.Sprintf("%s/%s", t.OS, t.Arch))
	}

	// If no targets specified, use current platform
	if len(platforms) == 0 {
		platforms = []string{"linux/amd64"}
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
		// Expand version template
		expandedTag := core.Replace(tag, "{{.Version}}", cfg.Version)
		expandedTag = core.Replace(expandedTag, "{{Version}}", cfg.Version)

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
		expandedValue := core.Replace(v, "{{.Version}}", cfg.Version)
		expandedValue = core.Replace(expandedValue, "{{Version}}", cfg.Version)
		args = append(args, "--build-arg", core.Sprintf("%s=%s", k, expandedValue))
	}

	// Always add VERSION build arg if version is set
	if cfg.Version != "" {
		args = append(args, "--build-arg", core.Sprintf("VERSION=%s", cfg.Version))
	}

	// Output to local docker images or push
	if cfg.Push {
		args = append(args, "--push")
	} else {
		// For multi-platform builds without push, we need to load or output somewhere
		if len(platforms) == 1 {
			args = append(args, "--load")
		} else {
			// Multi-platform builds can't use --load, output to tarball
			outputPath := ax.Join(cfg.OutputDir, core.Sprintf("%s.tar", imageName))
			args = append(args, "--output", core.Sprintf("type=oci,dest=%s", outputPath))
		}
	}

	// Build context (project directory)
	args = append(args, cfg.ProjectDir)

	// Create output directory
	if err := cfg.FS.EnsureDir(cfg.OutputDir); err != nil {
		return nil, coreerr.E("DockerBuilder.Build", "failed to create output directory", err)
	}

	core.Print(nil, "Building Docker image: %s", imageName)
	core.Print(nil, "  Platforms: %s", core.Join(", ", platforms...))
	core.Print(nil, "  Tags: %s", core.Join(", ", imageRefs...))

	if err := ax.ExecDir(ctx, cfg.ProjectDir, dockerCommand, args...); err != nil {
		return nil, coreerr.E("DockerBuilder.Build", "buildx build failed", err)
	}

	// Create artifacts for each platform
	var artifacts []build.Artifact
	for _, t := range targets {
		artifacts = append(artifacts, build.Artifact{
			Path: imageRefs[0], // Primary image reference
			OS:   t.OS,
			Arch: t.Arch,
		})
	}

	return artifacts, nil
}

// resolveDockerCli returns the executable path for the docker CLI.
func (b *DockerBuilder) resolveDockerCli(paths ...string) (string, error) {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/docker",
			"/opt/homebrew/bin/docker",
			"/Applications/Docker.app/Contents/Resources/bin/docker",
		}
	}

	command, err := ax.ResolveCommand("docker", paths...)
	if err != nil {
		return "", coreerr.E("DockerBuilder.resolveDockerCli", "docker CLI not found. Install it from https://docs.docker.com/get-docker/", err)
	}

	return command, nil
}

// ensureBuildx ensures docker buildx is available and has a builder.
func (b *DockerBuilder) ensureBuildx(ctx context.Context, dockerCommand string) error {
	// Check if buildx is available
	if err := ax.Exec(ctx, dockerCommand, "buildx", "version"); err != nil {
		return coreerr.E("DockerBuilder.ensureBuildx", "buildx is not available. Install it from https://docs.docker.com/buildx/working-with-buildx/", err)
	}

	// Check if we have a builder, create one if not
	if err := ax.Exec(ctx, dockerCommand, "buildx", "inspect", "--bootstrap"); err != nil {
		// Try to create a builder
		if err := ax.Exec(ctx, dockerCommand, "buildx", "create", "--use", "--bootstrap"); err != nil {
			return coreerr.E("DockerBuilder.ensureBuildx", "failed to create buildx builder", err)
		}
	}

	return nil
}
