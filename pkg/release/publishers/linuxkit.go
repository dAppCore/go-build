// Package publishers provides release publishing implementations.
package publishers

import (
	"context"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
)

// LinuxKitConfig holds configuration for the LinuxKit publisher.
//
// cfg := publishers.LinuxKitConfig{Config: ".core/node.yaml", Formats: []string{"iso", "qcow2"}}
type LinuxKitConfig struct {
	// Config is the path to the LinuxKit YAML configuration file.
	Config string `yaml:"config"`
	// Formats are the output formats to build.
	// Supported: iso, iso-bios, iso-efi, raw, raw-bios, raw-efi,
	//            qcow2, qcow2-bios, qcow2-efi, vmdk, vhd, gcp, aws,
	//            docker (tarball for `docker load`), tar, kernel+initrd
	Formats []string `yaml:"formats"`
	// Platforms are the target platforms (linux/amd64, linux/arm64).
	Platforms []string `yaml:"platforms"`
	// Targets describe optional cloud upload targets for cloud image formats.
	Targets []LinuxKitTarget `yaml:"targets"`
}

// LinuxKitTarget describes a cloud upload target for LinuxKit images.
type LinuxKitTarget struct {
	// Name is an optional target name.
	Name string `json:"name" yaml:"name"`
	// Type is the target type (aws, gcp, s3, gcs).
	Type string `json:"type" yaml:"type"`
	// Provider is the cloud provider (aws or gcp).
	Provider string `json:"provider" yaml:"provider"`
	// Bucket is the destination bucket name.
	Bucket string `json:"bucket" yaml:"bucket"`
	// Prefix is the object key prefix inside the bucket.
	Prefix string `json:"prefix" yaml:"prefix"`
	// Region is the AWS region for S3 uploads.
	Region string `json:"region" yaml:"region"`
}

// LinuxKitPublisher builds and publishes LinuxKit images.
//
// pub := publishers.NewLinuxKitPublisher()
type LinuxKitPublisher struct{}

// NewLinuxKitPublisher creates a new LinuxKit publisher.
//
// pub := publishers.NewLinuxKitPublisher()
func NewLinuxKitPublisher() *LinuxKitPublisher {
	return &LinuxKitPublisher{}
}

// Name returns the publisher's identifier.
//
// name := pub.Name() // → "linuxkit"
func (p *LinuxKitPublisher) Name() string {
	return "linuxkit"
}

// Validate checks the LinuxKit publisher configuration before publishing.
func (p *LinuxKitPublisher) Validate(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig) core.Result {
	_ = ctx
	_ = relCfg
	validated := validatePublisherRelease(p.Name(), release)
	if !validated.OK {
		return validated
	}

	lkCfg := p.parseConfig(pubCfg, release.ProjectDir)
	if !release.FS.Exists(lkCfg.Config) {
		return core.Fail(core.E("linuxkit.Validate", "config file not found: "+lkCfg.Config, nil))
	}
	if len(lkCfg.Formats) == 0 {
		return core.Fail(core.E("linuxkit.Validate", "at least one LinuxKit format is required", nil))
	}

	return core.Ok(nil)
}

// Supports reports whether the publisher handles the requested target.
func (p *LinuxKitPublisher) Supports(target string) bool {
	return supportsPublisherTarget(p.Name(), target)
}

// Publish builds LinuxKit images and routes them by output format.
//
// result := pub.Publish(ctx, rel, pubCfg, relCfg, false)
func (p *LinuxKitPublisher) Publish(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig, dryRun bool) core.Result {
	validated := validatePublisherRelease(p.Name(), release)
	if !validated.OK {
		return validated
	}

	linuxkitCommandResult := resolveLinuxKitCli()
	if !linuxkitCommandResult.OK {
		return linuxkitCommandResult
	}
	linuxkitCommand := linuxkitCommandResult.Value.(string)

	// Parse LinuxKit-specific config from publisher config
	lkCfg := p.parseConfig(pubCfg, release.ProjectDir)

	// Validate config file exists
	if release.FS == nil {
		return core.Fail(core.E("linuxkit.Publish", "release filesystem (FS) is nil", nil))
	}
	if !release.FS.Exists(lkCfg.Config) {
		return core.Fail(core.E("linuxkit.Publish", "config file not found: "+lkCfg.Config, nil))
	}

	// Determine repository for dry-run display.
	repo := ""
	if relCfg != nil {
		repo = relCfg.GetRepository()
	}
	if repo == "" && dryRun {
		detectedRepoResult := detectRepository(ctx, release.ProjectDir)
		if !detectedRepoResult.OK {
			return core.Fail(core.E("linuxkit.Publish", "could not determine repository", core.NewError(detectedRepoResult.Error())))
		}
		repo = detectedRepoResult.Value.(string)
	}

	if dryRun {
		return p.dryRunPublish(release, lkCfg, repo)
	}

	return p.executePublish(ctx, release, lkCfg, linuxkitCommand)
}

// parseConfig extracts LinuxKit-specific configuration.
func (p *LinuxKitPublisher) parseConfig(pubCfg PublisherConfig, projectDir string) LinuxKitConfig {
	cfg := LinuxKitConfig{
		Config:    ax.Join(projectDir, ".core", "linuxkit", "server.yml"),
		Formats:   []string{"iso"},
		Platforms: []string{"linux/amd64"},
	}

	// Override from extended config if present
	if ext, ok := pubCfg.Extended.(map[string]any); ok {
		if configPath, ok := ext["config"].(string); ok && configPath != "" {
			if ax.IsAbs(configPath) {
				cfg.Config = configPath
			} else {
				cfg.Config = ax.Join(projectDir, configPath)
			}
		}
		if formats, ok := ext["formats"].([]any); ok && len(formats) > 0 {
			cfg.Formats = make([]string, 0, len(formats))
			for _, f := range formats {
				if s, ok := f.(string); ok {
					cfg.Formats = append(cfg.Formats, s)
				}
			}
		}
		if platforms, ok := ext["platforms"].([]any); ok && len(platforms) > 0 {
			cfg.Platforms = make([]string, 0, len(platforms))
			for _, p := range platforms {
				if s, ok := p.(string); ok {
					cfg.Platforms = append(cfg.Platforms, s)
				}
			}
		}
		if targets, ok := ext["targets"]; ok {
			cfg.Targets = append(cfg.Targets, parseLinuxKitTargets(targets)...)
		}
		if target, ok := ext["target"]; ok {
			cfg.Targets = append(cfg.Targets, parseLinuxKitTargets(target)...)
		}
		appendLinuxKitTargetValue(&cfg, "aws", ext["aws"])
		appendLinuxKitTargetValue(&cfg, "gcp", ext["gcp"])
		appendLinuxKitBucketTargets(&cfg, ext)
	}

	return cfg
}

// dryRunPublish shows what would be done without actually building.
func (p *LinuxKitPublisher) dryRunPublish(release *Release, cfg LinuxKitConfig, repo string) core.Result {
	publisherPrintln()
	publisherPrintln("=== DRY RUN: LinuxKit Build & Publish ===")
	publisherPrintln()
	publisherPrint("Repository:    %s", repo)
	publisherPrint("Version:       %s", release.Version)
	publisherPrint("Config:        %s", cfg.Config)
	publisherPrint("Formats:       %s", core.Join(", ", cfg.Formats...))
	publisherPrint("Platforms:     %s", core.Join(", ", cfg.Platforms...))
	publisherPrintln()

	outputDir := ax.Join(release.ProjectDir, "dist", "linuxkit")
	baseName := p.buildBaseName(release.Version)

	publisherPrintln("Would execute commands:")
	for _, platform := range cfg.Platforms {
		parts := core.Split(platform, "/")
		arch := "amd64"
		if len(parts) == 2 {
			arch = parts[1]
		}

		for _, format := range cfg.Formats {
			outputName := core.Sprintf("%s-%s", baseName, arch)
			args := p.buildLinuxKitArgs(cfg.Config, format, outputName, outputDir, arch)
			publisherPrint("  linuxkit %s", core.Join(" ", args...))
		}
	}
	publisherPrintln()

	publisherPrintln("Would produce/upload artifacts:")
	for _, platform := range cfg.Platforms {
		parts := core.Split(platform, "/")
		arch := "amd64"
		if len(parts) == 2 {
			arch = parts[1]
		}

		for _, format := range cfg.Formats {
			outputName := core.Sprintf("%s-%s", baseName, arch)
			artifactPath := p.getArtifactPath(outputDir, outputName, format)
			publisherPrint("  - %s", ax.Base(artifactPath))
			if format == "docker" {
				publisherPrint("    Usage: docker load < %s", ax.Base(artifactPath))
			}
		}
	}

	publisherPrintln()
	publisherPrintln("=== END DRY RUN ===")

	return core.Ok(nil)
}

// executePublish builds LinuxKit images and routes them by format.
func (p *LinuxKitPublisher) executePublish(ctx context.Context, release *Release, cfg LinuxKitConfig, linuxkitCommand string) core.Result {
	outputDir := ax.Join(release.ProjectDir, "dist", "linuxkit")

	// Create output directory
	created := release.FS.EnsureDir(outputDir)
	if !created.OK {
		return core.Fail(core.E("linuxkit.Publish", "failed to create output directory", core.NewError(created.Error())))
	}

	baseName := p.buildBaseName(release.Version)

	// Build for each platform and format
	for _, platform := range cfg.Platforms {
		parts := core.Split(platform, "/")
		arch := "amd64"
		if len(parts) == 2 {
			arch = parts[1]
		}

		for _, format := range cfg.Formats {
			outputName := core.Sprintf("%s-%s", baseName, arch)

			// Build the image
			args := p.buildLinuxKitArgs(cfg.Config, format, outputName, outputDir, arch)
			publisherPrint("Building LinuxKit image: %s (%s)", outputName, format)
			built := publisherRun(ctx, release.ProjectDir, nil, linuxkitCommand, args...)
			if !built.OK {
				return core.Fail(core.E("linuxkit.Publish", "build failed for "+platform+"/"+format, core.NewError(built.Error())))
			}

			// Track artifact for upload
			artifactPath := p.getArtifactPath(outputDir, outputName, format)
			published := p.publishLinuxKitArtifact(ctx, release, cfg, format, artifactPath)
			if !published.OK {
				return published
			}
		}
	}

	return core.Ok(nil)
}

// buildBaseName creates the base name for output files.
func (p *LinuxKitPublisher) buildBaseName(version string) string {
	// Strip leading 'v' if present for cleaner filenames
	name := core.TrimPrefix(version, "v")
	return core.Sprintf("linuxkit-%s", name)
}

// buildLinuxKitArgs builds the arguments for linuxkit build command.
func (p *LinuxKitPublisher) buildLinuxKitArgs(configPath, format, outputName, outputDir, arch string) []string {
	args := []string{"build"}

	// Output format
	args = append(args, "--format", format)

	// Output name
	args = append(args, "--name", outputName)

	// Output directory
	args = append(args, "--dir", outputDir)

	// Architecture (if not amd64)
	if arch != "amd64" {
		args = append(args, "--arch", arch)
	}

	// Config file
	args = append(args, configPath)

	return args
}

// getArtifactPath returns the expected path of the built artifact.
func (p *LinuxKitPublisher) getArtifactPath(outputDir, outputName, format string) string {
	ext := p.getFormatExtension(format)
	return ax.Join(outputDir, outputName+ext)
}

// getFormatExtension returns the file extension for a LinuxKit output format.
func (p *LinuxKitPublisher) getFormatExtension(format string) string {
	switch format {
	case "iso", "iso-bios", "iso-efi":
		return ".iso"
	case "raw", "raw-bios", "raw-efi":
		return ".raw"
	case "qcow2", "qcow2-bios", "qcow2-efi":
		return ".qcow2"
	case "vmdk":
		return ".vmdk"
	case "vhd":
		return ".vhd"
	case "gcp":
		return ".img.tar.gz"
	case "aws":
		return ".raw"
	case "docker":
		// Docker format outputs a tarball that can be loaded with `docker load`
		return ".docker.tar"
	case "tar":
		return ".tar"
	case "kernel+initrd":
		return "-initrd.img"
	default:
		return "." + format
	}
}

// resolveLinuxKitCli returns the executable path for the linuxkit CLI.
func resolveLinuxKitCli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/linuxkit",
			"/opt/homebrew/bin/linuxkit",
		}
	}

	command := ax.ResolveCommand("linuxkit", paths...)
	if !command.OK {
		return core.Fail(core.E("linuxkit.resolveLinuxKitCli", "linuxkit CLI not found. Install it from https://github.com/linuxkit/linuxkit", core.NewError(command.Error())))
	}

	return command
}

// validateLinuxKitCli checks if the linuxkit CLI is available.
func validateLinuxKitCli() core.Result {
	resolved := resolveLinuxKitCli()
	if !resolved.OK {
		return core.Fail(core.E("linuxkit.validateLinuxKitCli", "linuxkit CLI not found. Install it from https://github.com/linuxkit/linuxkit", core.NewError(resolved.Error())))
	}
	return core.Ok(nil)
}
