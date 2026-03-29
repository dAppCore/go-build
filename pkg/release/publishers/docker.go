// Package publishers provides release publishing implementations.
package publishers

import (
	"context"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	coreerr "dappco.re/go/core/log"
)

// DockerConfig holds configuration for the Docker publisher.
// Usage example: declare a value of type publishers.DockerConfig in integrating code.
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
// Usage example: declare a value of type publishers.DockerPublisher in integrating code.
type DockerPublisher struct{}

// NewDockerPublisher creates a new Docker publisher.
// Usage example: call publishers.NewDockerPublisher(...) from integrating code.
func NewDockerPublisher() *DockerPublisher {
	return &DockerPublisher{}
}

// Name returns the publisher's identifier.
// Usage example: call value.Name(...) from integrating code.
func (p *DockerPublisher) Name() string {
	return "docker"
}

// Publish builds and pushes Docker images.
// Usage example: call value.Publish(...) from integrating code.
func (p *DockerPublisher) Publish(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig, dryRun bool) error {
	// Validate docker CLI is available
	if err := validateDockerCli(); err != nil {
		return err
	}

	// Parse Docker-specific config from publisher config
	dockerCfg := p.parseConfig(pubCfg, relCfg, release.ProjectDir)

	// Validate Dockerfile exists
	if !release.FS.Exists(dockerCfg.Dockerfile) {
		return coreerr.E("docker.Publish", "Dockerfile not found: "+dockerCfg.Dockerfile, nil)
	}

	if dryRun {
		return p.dryRunPublish(release, dockerCfg)
	}

	return p.executePublish(ctx, release, dockerCfg)
}

// parseConfig extracts Docker-specific configuration.
func (p *DockerPublisher) parseConfig(pubCfg PublisherConfig, relCfg ReleaseConfig, projectDir string) DockerConfig {
	cfg := DockerConfig{
		Registry:   "ghcr.io",
		Image:      "",
		Dockerfile: ax.Join(projectDir, "Dockerfile"),
		Platforms:  []string{"linux/amd64", "linux/arm64"},
		Tags:       []string{"latest", "{{.Version}}"},
		BuildArgs:  make(map[string]string),
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
func (p *DockerPublisher) dryRunPublish(release *Release, cfg DockerConfig) error {
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

	return nil
}

// executePublish builds and pushes Docker images.
func (p *DockerPublisher) executePublish(ctx context.Context, release *Release, cfg DockerConfig) error {
	// Ensure buildx is available and builder is set up
	if err := p.ensureBuildx(ctx); err != nil {
		return err
	}

	// Resolve tags
	tags := p.resolveTags(cfg.Tags, release.Version)

	// Build the docker buildx command
	args := p.buildBuildxArgs(cfg, tags, release.Version)

	publisherPrint("Building and pushing Docker image: %s", cfg.Image)
	if err := publisherRun(ctx, release.ProjectDir, nil, "docker", args...); err != nil {
		return coreerr.E("docker.Publish", "buildx build failed", err)
	}

	return nil
}

// resolveTags expands template variables in tags.
func (p *DockerPublisher) resolveTags(tags []string, version string) []string {
	resolved := make([]string, 0, len(tags))
	for _, tag := range tags {
		// Replace {{.Version}} with actual version
		resolvedTag := core.Replace(tag, "{{.Version}}", version)
		// Also support simpler {{Version}} syntax
		resolvedTag = core.Replace(resolvedTag, "{{Version}}", version)
		resolved = append(resolved, resolvedTag)
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
		// Expand version in build args
		expandedValue := core.Replace(v, "{{.Version}}", version)
		expandedValue = core.Replace(expandedValue, "{{Version}}", version)
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
func (p *DockerPublisher) ensureBuildx(ctx context.Context) error {
	// Check if buildx is available
	if err := ax.Exec(ctx, "docker", "buildx", "version"); err != nil {
		return coreerr.E("docker.ensureBuildx", "buildx is not available. Install it from https://docs.docker.com/buildx/working-with-buildx/", nil)
	}

	// Check if we have a builder, create one if not
	if err := ax.Exec(ctx, "docker", "buildx", "inspect", "--bootstrap"); err != nil {
		// Try to create a builder
		if err := publisherRun(ctx, "", nil, "docker", "buildx", "create", "--use", "--bootstrap"); err != nil {
			return coreerr.E("docker.ensureBuildx", "failed to create buildx builder", err)
		}
	}

	return nil
}

// validateDockerCli checks if the docker CLI is available.
func validateDockerCli() error {
	if _, err := ax.LookPath("docker"); err != nil {
		return coreerr.E("docker.validateDockerCli", "docker CLI not found. Install it from https://docs.docker.com/get-docker/", nil)
	}
	return nil
}
