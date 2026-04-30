// Package publishers provides release publishing implementations.
package publishers

import (
	"context"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	storage "dappco.re/go/build/pkg/storage"
)

// DockerConfig holds configuration for the Docker publisher.
//
// cfg := publishers.DockerConfig{Registry: "ghcr.io", Image: "host-uk/core-build", Platforms: []string{"linux/amd64", "linux/arm64"}}
type DockerConfig struct {
	// Registry is the container registry (default: ghcr.io).
	Registry string `yaml:"registry"`
	// Image is the image name in owner/repo format.
	Image string `yaml:"image"`
	// Dockerfile is the path to the Dockerfile (default: Dockerfile).
	Dockerfile string `yaml:"dockerfile"`
	// Platforms are the target platforms (linux/amd64, linux/arm64).
	Platforms []string `yaml:"platforms"`
	// Tags are additional tags to apply (supports {{.Version}} template).
	Tags []string `yaml:"tags"`
	// BuildArgs are additional build arguments.
	BuildArgs map[string]string `yaml:"build_args"`
}

// DockerPublisher builds and publishes Docker images.
//
// pub := publishers.NewDockerPublisher()
type DockerPublisher struct{}

// NewDockerPublisher creates a new Docker publisher.
//
// pub := publishers.NewDockerPublisher()
func NewDockerPublisher() *DockerPublisher {
	return &DockerPublisher{}
}

// Name returns the publisher's identifier.
//
// name := pub.Name() // → "docker"
func (p *DockerPublisher) Name() string {
	return "docker"
}

// Validate checks the Docker publisher configuration before publishing.
func (p *DockerPublisher) Validate(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig) core.Result {
	_ = ctx
	validated := validatePublisherRelease(p.Name(), release)
	if !validated.OK {
		return validated
	}

	dockerCfg := p.parseConfig(release.FS, pubCfg, relCfg, release.ProjectDir)
	if !release.FS.Exists(dockerCfg.Dockerfile) {
		return core.Fail(core.E("docker.Validate", "Dockerfile not found: "+dockerCfg.Dockerfile, nil))
	}
	if dockerCfg.Image == "" {
		return core.Fail(core.E("docker.Validate", "image name is required", nil))
	}

	return core.Ok(nil)
}

// Supports reports whether the publisher handles the requested target.
func (p *DockerPublisher) Supports(target string) bool {
	return supportsPublisherTarget(p.Name(), target)
}

// Publish builds and pushes Docker images.
//
// result := pub.Publish(ctx, rel, pubCfg, relCfg, false)
func (p *DockerPublisher) Publish(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig, dryRun bool) core.Result {
	validated := validatePublisherRelease(p.Name(), release)
	if !validated.OK {
		return validated
	}

	// Parse Docker-specific config from publisher config
	dockerCfg := p.parseConfig(release.FS, pubCfg, relCfg, release.ProjectDir)

	// Validate Dockerfile exists
	if !release.FS.Exists(dockerCfg.Dockerfile) {
		return core.Fail(core.E("docker.Publish", "Dockerfile not found: "+dockerCfg.Dockerfile, nil))
	}

	// Validate docker CLI is available after local config checks.
	dockerCommandResult := resolveDockerCli()
	if !dockerCommandResult.OK {
		return dockerCommandResult
	}
	dockerCommand := dockerCommandResult.Value.(string)

	if dryRun {
		return p.dryRunPublish(release, dockerCfg)
	}

	return p.executePublish(ctx, release, dockerCfg, dockerCommand)
}

// parseConfig extracts Docker-specific configuration.
func (p *DockerPublisher) parseConfig(fs storage.Medium, pubCfg PublisherConfig, relCfg ReleaseConfig, projectDir string) DockerConfig {
	cfg := DockerConfig{
		Registry:  "ghcr.io",
		Image:     "",
		Platforms: []string{"linux/amd64", "linux/arm64"},
		Tags:      []string{"latest", "{{.Version}}"},
		BuildArgs: make(map[string]string),
	}

	if dockerfile := build.ResolveDockerfilePath(fs, projectDir); dockerfile != "" {
		cfg.Dockerfile = dockerfile
	} else {
		cfg.Dockerfile = ax.Join(projectDir, "Dockerfile")
	}

	// Try to get image from repository config
	if relCfg != nil && relCfg.GetRepository() != "" {
		cfg.Image = relCfg.GetRepository()
	}

	// Override from extended config if present
	if ext, ok := pubCfg.Extended.(map[string]any); ok {
		if registry, ok := ext["registry"].(string); ok && registry != "" {
			cfg.Registry = registry
		}
		if image, ok := ext["image"].(string); ok && image != "" {
			cfg.Image = image
		}
		if dockerfile, ok := ext["dockerfile"].(string); ok && dockerfile != "" {
			if ax.IsAbs(dockerfile) {
				cfg.Dockerfile = dockerfile
			} else {
				cfg.Dockerfile = ax.Join(projectDir, dockerfile)
			}
		}
		if platforms, ok := ext["platforms"].([]any); ok && len(platforms) > 0 {
			cfg.Platforms = make([]string, 0, len(platforms))
			for _, plat := range platforms {
				if s, ok := plat.(string); ok {
					cfg.Platforms = append(cfg.Platforms, s)
				}
			}
		}
		if tags, ok := ext["tags"].([]any); ok && len(tags) > 0 {
			cfg.Tags = make([]string, 0, len(tags))
			for _, tag := range tags {
				if s, ok := tag.(string); ok {
					cfg.Tags = append(cfg.Tags, s)
				}
			}
		}
		if buildArgs, ok := ext["build_args"].(map[string]any); ok {
			for k, v := range buildArgs {
				if s, ok := v.(string); ok {
					cfg.BuildArgs[k] = s
				}
			}
		}
	}

	return cfg
}

// dryRunPublish shows what would be done without actually building.
func (p *DockerPublisher) dryRunPublish(release *Release, cfg DockerConfig) core.Result {
	publisherPrintln()
	publisherPrintln("=== DRY RUN: Docker Build & Push ===")
	publisherPrintln()
	publisherPrint("Version:       %s", release.Version)
	publisherPrint("Registry:      %s", cfg.Registry)
	publisherPrint("Image:         %s", cfg.Image)
	publisherPrint("Dockerfile:    %s", cfg.Dockerfile)
	publisherPrint("Platforms:     %s", core.Join(", ", cfg.Platforms...))
	publisherPrintln()

	// Resolve tags
	tags := p.resolveTags(cfg.Tags, release.Version)
	publisherPrintln("Tags to be applied:")
	for _, tag := range tags {
		fullTag := p.buildFullTag(cfg.Registry, cfg.Image, tag)
		publisherPrint("  - %s", fullTag)
	}
	publisherPrintln()

	publisherPrintln("Would execute command:")
	args := p.buildBuildxArgs(cfg, tags, release.Version)
	publisherPrint("  docker %s", core.Join(" ", args...))

	if len(cfg.BuildArgs) > 0 {
		publisherPrintln()
		publisherPrintln("Build arguments:")
		for k, v := range cfg.BuildArgs {
			publisherPrint("  %s=%s", k, v)
		}
	}

	publisherPrintln()
	publisherPrintln("=== END DRY RUN ===")

	return core.Ok(nil)
}

// executePublish builds and pushes Docker images.
func (p *DockerPublisher) executePublish(ctx context.Context, release *Release, cfg DockerConfig, dockerCommand string) core.Result {
	// Ensure buildx is available and builder is set up
	ensured := p.ensureBuildx(ctx, dockerCommand)
	if !ensured.OK {
		return ensured
	}

	// Resolve tags
	tags := p.resolveTags(cfg.Tags, release.Version)

	// Build the docker buildx command
	args := p.buildBuildxArgs(cfg, tags, release.Version)

	publisherPrint("Building and pushing Docker image: %s", cfg.Image)
	built := publisherRun(ctx, release.ProjectDir, nil, dockerCommand, args...)
	if !built.OK {
		return core.Fail(core.E("docker.Publish", "buildx build failed", core.NewError(built.Error())))
	}

	return core.Ok(nil)
}

// resolveTags expands template variables in tags.
func (p *DockerPublisher) resolveTags(tags []string, version string) []string {
	resolved := make([]string, 0, len(tags))
	for _, tag := range tags {
		resolved = append(resolved, build.ExpandVersionTemplate(tag, version))
	}
	return resolved
}

// buildFullTag builds the full image tag including registry.
func (p *DockerPublisher) buildFullTag(registry, image, tag string) string {
	if registry != "" {
		return core.Sprintf("%s/%s:%s", registry, image, tag)
	}
	return core.Sprintf("%s:%s", image, tag)
}

// buildBuildxArgs builds the arguments for docker buildx build command.
func (p *DockerPublisher) buildBuildxArgs(cfg DockerConfig, tags []string, version string) []string {
	args := []string{"buildx", "build"}

	// Multi-platform support
	if len(cfg.Platforms) > 0 {
		args = append(args, "--platform", core.Join(",", cfg.Platforms...))
	}

	// Add all tags
	for _, tag := range tags {
		fullTag := p.buildFullTag(cfg.Registry, cfg.Image, tag)
		args = append(args, "-t", fullTag)
	}

	// Dockerfile path
	dockerfilePath := cfg.Dockerfile
	args = append(args, "-f", dockerfilePath)

	// Build arguments
	for k, v := range cfg.BuildArgs {
		expandedValue := build.ExpandVersionTemplate(v, version)
		args = append(args, "--build-arg", core.Sprintf("%s=%s", k, expandedValue))
	}

	// Always add VERSION build arg
	args = append(args, "--build-arg", core.Sprintf("VERSION=%s", version))

	// Push the image
	args = append(args, "--push")

	// Build context (current directory)
	args = append(args, ".")

	return args
}

// ensureBuildx ensures docker buildx is available and has a builder.
func (p *DockerPublisher) ensureBuildx(ctx context.Context, dockerCommand string) core.Result {
	// Check if buildx is available
	available := ax.Exec(ctx, dockerCommand, "buildx", "version")
	if !available.OK {
		return core.Fail(core.E("docker.ensureBuildx", "buildx is not available. Install it from https://docs.docker.com/buildx/working-with-buildx/", nil))
	}

	// Check if we have a builder, create one if not
	inspected := ax.Exec(ctx, dockerCommand, "buildx", "inspect", "--bootstrap")
	if !inspected.OK {
		// Try to create a builder
		created := publisherRun(ctx, "", nil, dockerCommand, "buildx", "create", "--use", "--bootstrap")
		if !created.OK {
			return core.Fail(core.E("docker.ensureBuildx", "failed to create buildx builder", core.NewError(created.Error())))
		}
	}

	return core.Ok(nil)
}

// resolveDockerCli returns the executable path for the docker CLI.
func resolveDockerCli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/docker",
			"/opt/homebrew/bin/docker",
			"/Applications/Docker.app/Contents/Resources/bin/docker",
		}
	}

	command := ax.ResolveCommand("docker", paths...)
	if !command.OK {
		return core.Fail(core.E("docker.resolveDockerCli", "docker CLI not found. Install it from https://docs.docker.com/get-docker/", core.NewError(command.Error())))
	}

	return command
}

// validateDockerCli checks if the docker CLI is available.
func validateDockerCli() core.Result {
	resolved := resolveDockerCli()
	if !resolved.OK {
		return core.Fail(core.E("docker.validateDockerCli", "docker CLI not found. Install it from https://docs.docker.com/get-docker/", core.NewError(resolved.Error())))
	}
	return core.Ok(nil)
}
