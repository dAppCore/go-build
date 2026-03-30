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

// LinuxKitBuilder builds LinuxKit images.
// Usage example: declare a value of type builders.LinuxKitBuilder in integrating code.
type LinuxKitBuilder struct{}

// NewLinuxKitBuilder creates a new LinuxKit builder.
// Usage example: call builders.NewLinuxKitBuilder(...) from integrating code.
func NewLinuxKitBuilder() *LinuxKitBuilder {
	return &LinuxKitBuilder{}
}

// Name returns the builder's identifier.
// Usage example: call value.Name(...) from integrating code.
func (b *LinuxKitBuilder) Name() string {
	return "linuxkit"
}

// Detect checks if a linuxkit.yml or .yml config exists in the directory.
// Usage example: call value.Detect(...) from integrating code.
func (b *LinuxKitBuilder) Detect(fs io.Medium, dir string) (bool, error) {
	// Check for linuxkit.yml
	if fs.IsFile(ax.Join(dir, "linuxkit.yml")) {
		return true, nil
	}
	// Check for .core/linuxkit/
	lkDir := ax.Join(dir, ".core", "linuxkit")
	if fs.IsDir(lkDir) {
		entries, err := fs.List(lkDir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && core.HasSuffix(entry.Name(), ".yml") {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

// Build builds LinuxKit images for the specified targets.
// Usage example: call value.Build(...) from integrating code.
func (b *LinuxKitBuilder) Build(ctx context.Context, cfg *build.Config, targets []build.Target) ([]build.Artifact, error) {
	// Validate linuxkit CLI is available
	if err := b.validateLinuxKitCli(); err != nil {
		return nil, err
	}

	// Determine config file path
	configPath := cfg.LinuxKitConfig
	if configPath == "" {
		// Auto-detect
		if cfg.FS.IsFile(ax.Join(cfg.ProjectDir, "linuxkit.yml")) {
			configPath = ax.Join(cfg.ProjectDir, "linuxkit.yml")
		} else {
			// Look in .core/linuxkit/
			lkDir := ax.Join(cfg.ProjectDir, ".core", "linuxkit")
			if cfg.FS.IsDir(lkDir) {
				entries, err := cfg.FS.List(lkDir)
				if err == nil {
					for _, entry := range entries {
						if !entry.IsDir() && core.HasSuffix(entry.Name(), ".yml") {
							configPath = ax.Join(lkDir, entry.Name())
							break
						}
					}
				}
			}
		}
	}

	if configPath == "" {
		return nil, coreerr.E("LinuxKitBuilder.Build", "no LinuxKit config file found. Specify with --config or create linuxkit.yml", nil)
	}

	// Validate config file exists
	if !cfg.FS.IsFile(configPath) {
		return nil, coreerr.E("LinuxKitBuilder.Build", "config file not found: "+configPath, nil)
	}

	// Determine output formats
	formats := cfg.Formats
	if len(formats) == 0 {
		formats = []string{"qcow2-bios"} // Default to QEMU-compatible format
	}

	// Create output directory
	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = ax.Join(cfg.ProjectDir, "dist")
	}
	if err := cfg.FS.EnsureDir(outputDir); err != nil {
		return nil, coreerr.E("LinuxKitBuilder.Build", "failed to create output directory", err)
	}

	// Determine base name from config file or project name
	baseName := cfg.Name
	if baseName == "" {
		baseName = core.TrimSuffix(ax.Base(configPath), ".yml")
	}

	// If no targets, default to linux/amd64
	if len(targets) == 0 {
		targets = []build.Target{{OS: "linux", Arch: "amd64"}}
	}

	var artifacts []build.Artifact

	// Build for each target and format
	for _, target := range targets {
		// LinuxKit only supports Linux
		if target.OS != "linux" {
			core.Print(nil, "Skipping %s/%s (LinuxKit only supports Linux)", target.OS, target.Arch)
			continue
		}

		for _, format := range formats {
			outputName := core.Sprintf("%s-%s", baseName, target.Arch)

			args := b.buildLinuxKitArgs(configPath, format, outputName, outputDir, target.Arch)

			core.Print(nil, "Building LinuxKit image: %s (%s, %s)", outputName, format, target.Arch)
			if err := ax.ExecDir(ctx, cfg.ProjectDir, "linuxkit", args...); err != nil {
				return nil, coreerr.E("LinuxKitBuilder.Build", "build failed for "+target.Arch+"/"+format, err)
			}

			// Determine the actual output file path
			artifactPath := b.getArtifactPath(outputDir, outputName, format)

			// Verify the artifact was created
			if !cfg.FS.Exists(artifactPath) {
				// Try alternate naming conventions
				artifactPath = b.findArtifact(cfg.FS, outputDir, outputName, format)
				if artifactPath == "" {
					return nil, coreerr.E("LinuxKitBuilder.Build", "artifact not found after build: expected "+b.getArtifactPath(outputDir, outputName, format), nil)
				}
			}

			artifacts = append(artifacts, build.Artifact{
				Path: artifactPath,
				OS:   target.OS,
				Arch: target.Arch,
			})
		}
	}

	return artifacts, nil
}

// buildLinuxKitArgs builds the arguments for linuxkit build command.
func (b *LinuxKitBuilder) buildLinuxKitArgs(configPath, format, outputName, outputDir, arch string) []string {
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
func (b *LinuxKitBuilder) getArtifactPath(outputDir, outputName, format string) string {
	ext := b.getFormatExtension(format)
	return ax.Join(outputDir, outputName+ext)
}

// findArtifact searches for the built artifact with various naming conventions.
func (b *LinuxKitBuilder) findArtifact(fs io.Medium, outputDir, outputName, format string) string {
	// LinuxKit can create files with different suffixes
	extensions := []string{
		b.getFormatExtension(format),
		"-bios" + b.getFormatExtension(format),
		"-efi" + b.getFormatExtension(format),
	}

	for _, ext := range extensions {
		path := ax.Join(outputDir, outputName+ext)
		if fs.Exists(path) {
			return path
		}
	}

	// Try to find any file matching the output name
	entries, err := fs.List(outputDir)
	if err == nil {
		for _, entry := range entries {
			if core.HasPrefix(entry.Name(), outputName) {
				match := ax.Join(outputDir, entry.Name())
				// Return first match that looks like an image
				ext := ax.Ext(match)
				if ext == ".iso" || ext == ".qcow2" || ext == ".raw" || ext == ".vmdk" || ext == ".vhd" {
					return match
				}
			}
		}
	}

	return ""
}

// getFormatExtension returns the file extension for a LinuxKit output format.
func (b *LinuxKitBuilder) getFormatExtension(format string) string {
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
	default:
		return "." + core.TrimSuffix(format, "-bios")
	}
}

// validateLinuxKitCli checks if the linuxkit CLI is available.
func (b *LinuxKitBuilder) validateLinuxKitCli() error {
	// Check PATH first
	if _, err := ax.LookPath("linuxkit"); err == nil {
		return nil
	}

	// Check common locations
	paths := []string{
		"/usr/local/bin/linuxkit",
		"/opt/homebrew/bin/linuxkit",
	}

	for _, p := range paths {
		if ax.IsFile(p) {
			return nil
		}
	}

	return coreerr.E("LinuxKitBuilder.validateLinuxKitCli", "linuxkit CLI not found. Install with: brew install linuxkit (macOS) or see https://github.com/linuxkit/linuxkit", nil)
}
