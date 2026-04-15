// Package builders provides build implementations for different project types.
package builders

import (
	"context"

	"dappco.re/go/core"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

// LinuxKitBuilder builds LinuxKit images.
//
// b := builders.NewLinuxKitBuilder()
type LinuxKitBuilder struct{}

// NewLinuxKitBuilder creates a new LinuxKit builder.
//
// b := builders.NewLinuxKitBuilder()
func NewLinuxKitBuilder() *LinuxKitBuilder {
	return &LinuxKitBuilder{}
}

// Name returns the builder's identifier.
//
// name := b.Name() // → "linuxkit"
func (b *LinuxKitBuilder) Name() string {
	return "linuxkit"
}

// Detect checks if a linuxkit.yml, linuxkit.yaml, or nested YAML config exists in the directory.
//
// ok, err := b.Detect(io.Local, ".")
func (b *LinuxKitBuilder) Detect(fs io.Medium, dir string) (bool, error) {
	// Check for linuxkit.yml
	if fs.IsFile(ax.Join(dir, "linuxkit.yml")) || fs.IsFile(ax.Join(dir, "linuxkit.yaml")) {
		return true, nil
	}
	// Check for .core/linuxkit/
	lkDir := ax.Join(dir, ".core", "linuxkit")
	if fs.IsDir(lkDir) {
		entries, err := fs.List(lkDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				name := entry.Name()
				if core.HasSuffix(name, ".yml") || core.HasSuffix(name, ".yaml") {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

// Build builds LinuxKit images for the specified targets.
//
// artifacts, err := b.Build(ctx, cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
func (b *LinuxKitBuilder) Build(ctx context.Context, cfg *build.Config, targets []build.Target) ([]build.Artifact, error) {
	linuxkitCommand, err := b.resolveLinuxKitCli()
	if err != nil {
		return nil, err
	}

	// Determine config file path
	configPath := cfg.LinuxKitConfig
	if configPath == "" {
		// Auto-detect
		if cfg.FS.IsFile(ax.Join(cfg.ProjectDir, "linuxkit.yml")) {
			configPath = ax.Join(cfg.ProjectDir, "linuxkit.yml")
		} else if cfg.FS.IsFile(ax.Join(cfg.ProjectDir, "linuxkit.yaml")) {
			configPath = ax.Join(cfg.ProjectDir, "linuxkit.yaml")
		} else {
			// Look in .core/linuxkit/
			lkDir := ax.Join(cfg.ProjectDir, ".core", "linuxkit")
			if cfg.FS.IsDir(lkDir) {
				entries, err := cfg.FS.List(lkDir)
				if err == nil {
					for _, entry := range entries {
						if entry.IsDir() {
							continue
						}
						name := entry.Name()
						if core.HasSuffix(name, ".yml") || core.HasSuffix(name, ".yaml") {
							configPath = ax.Join(lkDir, entry.Name())
							break
						}
					}
				}
			}
		}
	} else if !ax.IsAbs(configPath) {
		configPath = ax.Join(cfg.ProjectDir, configPath)
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
		baseName = core.TrimSuffix(baseName, ".yaml")
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
			if err := ax.ExecWithEnv(ctx, cfg.ProjectDir, build.BuildEnvironment(cfg), linuxkitCommand, args...); err != nil {
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
				if isLinuxKitArtifact(match) {
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
	case "docker":
		return ".docker.tar"
	case "tar":
		return ".tar"
	case "kernel+initrd":
		return "-initrd.img"
	default:
		return "." + core.TrimSuffix(format, "-bios")
	}
}

// isLinuxKitArtifact reports whether a file path looks like a LinuxKit build output.
func isLinuxKitArtifact(path string) bool {
	switch {
	case core.HasSuffix(path, ".img.tar.gz"):
		return true
	case core.HasSuffix(path, ".docker.tar"):
		return true
	case core.HasSuffix(path, "-initrd.img"):
		return true
	case core.HasSuffix(path, ".tar"):
		return true
	case core.HasSuffix(path, ".iso"):
		return true
	case core.HasSuffix(path, ".qcow2"):
		return true
	case core.HasSuffix(path, ".raw"):
		return true
	case core.HasSuffix(path, ".vmdk"):
		return true
	case core.HasSuffix(path, ".vhd"):
		return true
	default:
		return false
	}
}

// resolveLinuxKitCli returns the executable path for the linuxkit CLI.
func (b *LinuxKitBuilder) resolveLinuxKitCli(paths ...string) (string, error) {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/linuxkit",
			"/opt/homebrew/bin/linuxkit",
		}
	}

	command, err := ax.ResolveCommand("linuxkit", paths...)
	if err != nil {
		return "", coreerr.E("LinuxKitBuilder.resolveLinuxKitCli", "linuxkit CLI not found. Install with: brew install linuxkit (macOS) or see https://github.com/linuxkit/linuxkit", err)
	}

	return command, nil
}
