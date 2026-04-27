// Package builders provides build implementations for different project types.
package builders

import (
	"context"
	"runtime"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/core"
	"dappco.re/go/io"
	coreerr "dappco.re/go/log"
)

// RustBuilder implements the Builder interface for Rust projects.
//
// b := builders.NewRustBuilder()
type RustBuilder struct{}

// NewRustBuilder creates a new RustBuilder instance.
//
// b := builders.NewRustBuilder()
func NewRustBuilder() *RustBuilder {
	return &RustBuilder{}
}

// Name returns the builder's identifier.
//
// name := b.Name() // → "rust"
func (b *RustBuilder) Name() string {
	return "rust"
}

// Detect checks if this builder can handle the project in the given directory.
//
// ok, err := b.Detect(io.Local, ".")
func (b *RustBuilder) Detect(fs io.Medium, dir string) (bool, error) {
	return build.IsRustProject(fs, dir), nil
}

// Build compiles the Rust project for the specified targets.
//
// artifacts, err := b.Build(ctx, cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
func (b *RustBuilder) Build(ctx context.Context, cfg *build.Config, targets []build.Target) ([]build.Artifact, error) {
	if cfg == nil {
		return nil, coreerr.E("RustBuilder.Build", "config is nil", nil)
	}
	filesystem := ensureBuildFilesystem(cfg)

	cargoCommand, err := b.resolveCargoCli()
	if err != nil {
		return nil, err
	}

	targets = defaultRuntimeTargets(targets, runtime.GOOS, runtime.GOARCH)

	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = defaultOutputDir(cfg)
	}
	if err := ensureOutputDir(filesystem, outputDir, "RustBuilder.Build"); err != nil {
		return nil, err
	}

	var artifacts []build.Artifact
	for _, target := range targets {
		targetTriple, err := rustTargetTriple(target)
		if err != nil {
			return artifacts, err
		}

		platformDir, err := ensurePlatformDir(filesystem, outputDir, target, "RustBuilder.Build")
		if err != nil {
			return artifacts, err
		}

		env := configuredTargetEnv(cfg, target,
			core.Sprintf("CARGO_TARGET_DIR=%s", platformDir),
			core.Sprintf("TARGET_OS=%s", target.OS),
			core.Sprintf("TARGET_ARCH=%s", target.Arch),
		)

		args := []string{"build", "--release", "--target", targetTriple}
		output, err := ax.CombinedOutput(ctx, cfg.ProjectDir, env, cargoCommand, args...)
		if err != nil {
			return artifacts, coreerr.E("RustBuilder.Build", "cargo build failed: "+output, err)
		}

		found := b.findArtifactsForTarget(filesystem, platformDir, targetTriple, target)
		if len(found) == 0 {
			return artifacts, coreerr.E("RustBuilder.Build", "no build artifacts found for "+target.String(), nil)
		}

		artifacts = append(artifacts, found...)
	}

	return artifacts, nil
}

// resolveCargoCli returns the executable path for cargo.
//
// command, err := b.resolveCargoCli()
func (b *RustBuilder) resolveCargoCli(paths ...string) (string, error) {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/cargo",
			"/opt/homebrew/bin/cargo",
		}
	}

	command, err := ax.ResolveCommand("cargo", paths...)
	if err != nil {
		return "", coreerr.E("RustBuilder.resolveCargoCli", "cargo CLI not found. Install Rust from https://www.rust-lang.org/tools/install", err)
	}

	return command, nil
}

// findArtifactsForTarget looks for compiled binaries in the cargo target directory.
func (b *RustBuilder) findArtifactsForTarget(fs io.Medium, targetDir, targetTriple string, target build.Target) []build.Artifact {
	releaseDir := ax.Join(targetDir, targetTriple, "release")
	if !fs.IsDir(releaseDir) {
		return nil
	}

	entries, err := fs.List(releaseDir)
	if err != nil {
		return nil
	}

	var artifacts []build.Artifact
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if core.HasPrefix(name, ".") ||
			core.HasSuffix(name, ".d") ||
			core.HasSuffix(name, ".rlib") ||
			core.HasSuffix(name, ".rmeta") ||
			core.HasSuffix(name, ".a") ||
			core.HasSuffix(name, ".lib") ||
			core.HasSuffix(name, ".pdb") {
			continue
		}

		fullPath := ax.Join(releaseDir, name)
		if target.OS != "windows" {
			info, statErr := fs.Stat(fullPath)
			if statErr != nil || info.Mode()&0o111 == 0 {
				continue
			}
		}

		artifacts = append(artifacts, build.Artifact{
			Path: fullPath,
			OS:   target.OS,
			Arch: target.Arch,
		})
	}

	return artifacts
}

// rustTargetTriple maps a build target to a Rust target triple.
func rustTargetTriple(target build.Target) (string, error) {
	switch target.OS + "/" + target.Arch {
	case "linux/amd64":
		return "x86_64-unknown-linux-gnu", nil
	case "linux/arm64":
		return "aarch64-unknown-linux-gnu", nil
	case "darwin/amd64":
		return "x86_64-apple-darwin", nil
	case "darwin/arm64":
		return "aarch64-apple-darwin", nil
	case "windows/amd64":
		return "x86_64-pc-windows-msvc", nil
	case "windows/arm64":
		return "aarch64-pc-windows-msvc", nil
	default:
		return "", coreerr.E("RustBuilder.rustTargetTriple", "unsupported Rust target: "+target.String(), nil)
	}
}

// Ensure RustBuilder implements the Builder interface.
var _ build.Builder = (*RustBuilder)(nil)
