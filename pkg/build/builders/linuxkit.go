// Package builders provides build implementations for different project types.
package builders

import (
	"context"
	stdfs "io/fs"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/io"
	coreerr "dappco.re/go/log"
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
// result := b.Detect(io.Local, ".")
func (b *LinuxKitBuilder) Detect(fs io.Medium, dir string) core.Result {
	return core.Ok(build.IsLinuxKitProject(fs, dir))
}

// Build builds LinuxKit images for the specified targets.
//
// result := b.Build(ctx, cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
func (b *LinuxKitBuilder) Build(ctx context.Context, cfg *build.Config, targets []build.Target) core.Result {
	if cfg == nil {
		return core.Fail(coreerr.E("LinuxKitBuilder.Build", "config is nil", nil))
	}
	filesystem := ensureBuildFilesystem(cfg)
	artifactFilesystem := build.ResolveOutputMedium(cfg)

	linuxkitCommandResult := b.resolveLinuxKitCli()
	if !linuxkitCommandResult.OK {
		return linuxkitCommandResult
	}
	linuxkitCommand := linuxkitCommandResult.Value.(string)

	// Determine config file path
	configPath := cfg.LinuxKitConfig
	if configPath == "" {
		// Auto-detect
		if filesystem.IsFile(ax.Join(cfg.ProjectDir, "linuxkit.yml")) {
			configPath = ax.Join(cfg.ProjectDir, "linuxkit.yml")
		} else if filesystem.IsFile(ax.Join(cfg.ProjectDir, "linuxkit.yaml")) {
			configPath = ax.Join(cfg.ProjectDir, "linuxkit.yaml")
		} else {
			// Look in .core/linuxkit/
			lkDir := ax.Join(cfg.ProjectDir, ".core", "linuxkit")
			if filesystem.IsDir(lkDir) {
				entriesResult := filesystem.List(lkDir)
				if entriesResult.OK {
					entries := entriesResult.Value.([]stdfs.DirEntry)
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
		return core.Fail(coreerr.E("LinuxKitBuilder.Build", "no LinuxKit config file found. Specify with --config or create linuxkit.yml", nil))
	}

	// Validate config file exists
	if !filesystem.IsFile(configPath) {
		return core.Fail(coreerr.E("LinuxKitBuilder.Build", "config file not found: "+configPath, nil))
	}

	// Determine output formats
	formats := cfg.Formats
	if len(formats) == 0 {
		formats = []string{"qcow2-bios"} // Default to QEMU-compatible format
	}

	// Create output directory
	outputDir := cfg.OutputDir
	if outputDir == "" && build.MediumIsLocal(artifactFilesystem) {
		outputDir = defaultOutputDir(cfg)
	}
	created := ensureOutputDir(artifactFilesystem, outputDir, "LinuxKitBuilder.Build")
	if !created.OK {
		return created
	}

	stageResult := prepareStagedOutput(outputDir, artifactFilesystem, "core-build-linuxkit-*", "LinuxKitBuilder.Build")
	if !stageResult.OK {
		return stageResult
	}
	stage := stageResult.Value.(stagedOutput)
	defer stage.cleanup()

	// Determine base name from config file or project name
	baseName := cfg.Name
	if baseName == "" {
		baseName = core.TrimSuffix(ax.Base(configPath), ".yml")
		baseName = core.TrimSuffix(baseName, ".yaml")
	}

	// If no targets, default to linux/amd64
	targets = defaultLinuxTargets(targets)

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

			args := b.buildLinuxKitArgs(configPath, format, outputName, stage.commandOutputDir, target.Arch)

			core.Print(nil, "Building LinuxKit image: %s (%s, %s)", outputName, format, target.Arch)
			executed := ax.ExecWithEnv(ctx, cfg.ProjectDir, build.BuildEnvironment(cfg), linuxkitCommand, args...)
			if !executed.OK {
				return core.Fail(coreerr.E("LinuxKitBuilder.Build", "build failed for "+target.Arch+"/"+format, core.NewError(executed.Error())))
			}

			// Determine the actual output file path
			artifactPath := b.getArtifactPath(stage.commandOutputDir, outputName, format)

			// Verify the artifact was created
			if !stage.commandFS.Exists(artifactPath) {
				// Try alternate naming conventions
				artifactPath = b.findArtifact(stage.commandFS, stage.commandOutputDir, outputName, format)
				if artifactPath == "" {
					return core.Fail(coreerr.E("LinuxKitBuilder.Build", "artifact not found after build: expected "+b.getArtifactPath(stage.commandOutputDir, outputName, format), nil))
				}
			}

			finalArtifactPath := b.getArtifactPath(outputDir, outputName, format)
			if artifactPath != finalArtifactPath {
				copied := build.CopyMediumPath(stage.commandFS, artifactPath, artifactFilesystem, finalArtifactPath)
				if !copied.OK {
					return copied
				}
			}

			artifacts = append(artifacts, build.Artifact{
				Path: finalArtifactPath,
				OS:   target.OS,
				Arch: target.Arch,
			})
		}
	}

	return core.Ok(artifacts)
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
	if outputDir == "" {
		return outputName + ext
	}
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
		path := outputName + ext
		if outputDir != "" {
			path = ax.Join(outputDir, outputName+ext)
		}
		if fs.Exists(path) {
			return path
		}
	}

	// Try to find any file matching the output name
	entriesResult := fs.List(outputDir)
	if entriesResult.OK {
		entries := entriesResult.Value.([]stdfs.DirEntry)
		for _, entry := range entries {
			if core.HasPrefix(entry.Name(), outputName) {
				match := entry.Name()
				if outputDir != "" {
					match = ax.Join(outputDir, entry.Name())
				}
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
func (b *LinuxKitBuilder) resolveLinuxKitCli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/linuxkit",
			"/opt/homebrew/bin/linuxkit",
		}
	}

	command := ax.ResolveCommand("linuxkit", paths...)
	if !command.OK {
		return core.Fail(coreerr.E("LinuxKitBuilder.resolveLinuxKitCli", "linuxkit CLI not found. Install with: brew install linuxkit (macOS) or see https://github.com/linuxkit/linuxkit", core.NewError(command.Error())))
	}

	return command
}
