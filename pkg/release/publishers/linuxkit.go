// Package publishers provides release publishing implementations.
package publishers

import (
	"context"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	coreerr "dappco.re/go/core/log"
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

// Publish builds LinuxKit images and uploads them to the GitHub release.
//
// err := pub.Publish(ctx, rel, pubCfg, relCfg, false)
func (p *LinuxKitPublisher) Publish(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig, dryRun bool) error {
	linuxkitCommand, err := resolveLinuxKitCli()
	if err != nil {
		return err
	}

	// Parse LinuxKit-specific config from publisher config
	lkCfg := p.parseConfig(pubCfg, release.ProjectDir)

	// Validate config file exists
	if release.FS == nil {
		return coreerr.E("linuxkit.Publish", "release filesystem (FS) is nil", nil)
	}
	if !release.FS.Exists(lkCfg.Config) {
		return coreerr.E("linuxkit.Publish", "config file not found: "+lkCfg.Config, nil)
	}

	// Determine repository for artifact upload
	repo := ""
	if relCfg != nil {
		repo = relCfg.GetRepository()
	}
	if repo == "" {
		detectedRepo, err := detectRepository(ctx, release.ProjectDir)
		if err != nil {
			return coreerr.E("linuxkit.Publish", "could not determine repository", err)
		}
		repo = detectedRepo
	}

	if dryRun {
		return p.dryRunPublish(release, lkCfg, repo)
	}

	return p.executePublish(ctx, release, lkCfg, repo, linuxkitCommand)
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
	}

	return cfg
}

// dryRunPublish shows what would be done without actually building.
func (p *LinuxKitPublisher) dryRunPublish(release *Release, cfg LinuxKitConfig, repo string) error {
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

	publisherPrintln("Would upload artifacts to release:")
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

	return nil
}

// executePublish builds LinuxKit images and uploads them.
func (p *LinuxKitPublisher) executePublish(ctx context.Context, release *Release, cfg LinuxKitConfig, repo, linuxkitCommand string) error {
	outputDir := ax.Join(release.ProjectDir, "dist", "linuxkit")

	// Create output directory
	if err := release.FS.EnsureDir(outputDir); err != nil {
		return coreerr.E("linuxkit.Publish", "failed to create output directory", err)
	}

	baseName := p.buildBaseName(release.Version)
	var artifacts []string

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
			if err := publisherRun(ctx, release.ProjectDir, nil, linuxkitCommand, args...); err != nil {
				return coreerr.E("linuxkit.Publish", "build failed for "+platform+"/"+format, err)
			}

			// Track artifact for upload
			artifactPath := p.getArtifactPath(outputDir, outputName, format)
			artifacts = append(artifacts, artifactPath)
		}
	}

	// Upload artifacts to GitHub release
	for _, artifactPath := range artifacts {
		if !release.FS.Exists(artifactPath) {
			return coreerr.E("linuxkit.Publish", "artifact not found after build: "+artifactPath, nil)
		}

		if err := UploadArtifact(ctx, repo, release.Version, artifactPath); err != nil {
			return coreerr.E("linuxkit.Publish", "failed to upload "+ax.Base(artifactPath), err)
		}

		// Print helpful usage info for docker format
		if core.HasSuffix(artifactPath, ".docker.tar") {
			publisherPrint("  Load with: docker load < %s", ax.Base(artifactPath))
		}
	}

	return nil
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
func resolveLinuxKitCli(paths ...string) (string, error) {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/linuxkit",
			"/opt/homebrew/bin/linuxkit",
		}
	}

	command, err := ax.ResolveCommand("linuxkit", paths...)
	if err != nil {
		return "", coreerr.E("linuxkit.resolveLinuxKitCli", "linuxkit CLI not found. Install it from https://github.com/linuxkit/linuxkit", err)
	}

	return command, nil
}

// validateLinuxKitCli checks if the linuxkit CLI is available.
func validateLinuxKitCli() error {
	if _, err := resolveLinuxKitCli(); err != nil {
		return coreerr.E("linuxkit.validateLinuxKitCli", "linuxkit CLI not found. Install it from https://github.com/linuxkit/linuxkit", err)
	}
	return nil
}
