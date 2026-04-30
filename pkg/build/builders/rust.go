// Package builders provides build implementations for different project types.
package builders

import (
	"context"
	stdfs "io/fs"
	"runtime"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	storage "dappco.re/go/build/pkg/storage"
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
// ok, err := b.Detect(storage.Local, ".")
func (b *RustBuilder) Detect(filesystem storage.Medium, dir string) core.Result {
	return core.Ok(build.IsRustProject(filesystem, dir))
}

// Build compiles the Rust project for the specified targets.
//
// artifacts, err := b.Build(ctx, cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
func (b *RustBuilder) Build(ctx context.Context, cfg *build.Config, targets []build.Target) core.Result {
	if cfg == nil {
		return core.Fail(core.E("RustBuilder.Build", "config is nil", nil))
	}
	filesystem := ensureBuildFilesystem(cfg)

	cargoCommandResult := b.resolveCargoCli()
	if !cargoCommandResult.OK {
		return cargoCommandResult
	}
	cargoCommand := cargoCommandResult.Value.(string)

	targets = defaultRuntimeTargets(targets, runtime.GOOS, runtime.GOARCH)

	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = defaultOutputDir(cfg)
	}
	created := ensureOutputDir(filesystem, outputDir, "RustBuilder.Build")
	if !created.OK {
		return created
	}

	var artifacts []build.Artifact
	for _, target := range targets {
		targetTripleResult := rustTargetTriple(target)
		if !targetTripleResult.OK {
			return targetTripleResult
		}
		targetTriple := targetTripleResult.Value.(string)

		platformDirResult := ensurePlatformDir(filesystem, outputDir, target, "RustBuilder.Build")
		if !platformDirResult.OK {
			return platformDirResult
		}
		platformDir := platformDirResult.Value.(string)

		env := configuredTargetEnv(cfg, target,
			core.Sprintf("CARGO_TARGET_DIR=%s", platformDir),
			core.Sprintf("TARGET_OS=%s", target.OS),
			core.Sprintf("TARGET_ARCH=%s", target.Arch),
		)

		args := []string{"build", "--release", "--target", targetTriple}
		output := ax.CombinedOutput(ctx, cfg.ProjectDir, env, cargoCommand, args...)
		if !output.OK {
			return core.Fail(core.E("RustBuilder.Build", "cargo build failed: "+output.Error(), core.NewError(output.Error())))
		}

		found := b.findArtifactsForTarget(filesystem, platformDir, targetTriple, target)
		if len(found) == 0 {
			return core.Fail(core.E("RustBuilder.Build", "no build artifacts found for "+target.String(), nil))
		}

		artifacts = append(artifacts, found...)
	}

	return core.Ok(artifacts)
}

// resolveCargoCli returns the executable path for cargo.
//
// command, err := b.resolveCargoCli()
func (b *RustBuilder) resolveCargoCli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/cargo",
			"/opt/homebrew/bin/cargo",
		}
	}

	command := ax.ResolveCommand("cargo", paths...)
	if !command.OK {
		return core.Fail(core.E("RustBuilder.resolveCargoCli", "cargo CLI not found. Install Rust from https://www.rust-lang.org/tools/install", core.NewError(command.Error())))
	}

	return command
}

// findArtifactsForTarget looks for compiled binaries in the cargo target directory.
func (b *RustBuilder) findArtifactsForTarget(fs storage.Medium, targetDir, targetTriple string, target build.Target) []build.Artifact {
	releaseDir := ax.Join(targetDir, targetTriple, "release")
	if !fs.IsDir(releaseDir) {
		return nil
	}

	entriesResult := fs.List(releaseDir)
	if !entriesResult.OK {
		return nil
	}
	entries := entriesResult.Value.([]stdfs.DirEntry)

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
			info := fs.Stat(fullPath)
			if !info.OK || info.Value.(stdfs.FileInfo).Mode()&0o111 == 0 {
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
func rustTargetTriple(target build.Target) core.Result {
	switch target.OS + "/" + target.Arch {
	case "linux/amd64":
		return core.Ok("x86_64-unknown-linux-gnu")
	case "linux/arm64":
		return core.Ok("aarch64-unknown-linux-gnu")
	case "darwin/amd64":
		return core.Ok("x86_64-apple-darwin")
	case "darwin/arm64":
		return core.Ok("aarch64-apple-darwin")
	case "windows/amd64":
		return core.Ok("x86_64-pc-windows-msvc")
	case "windows/arm64":
		return core.Ok("aarch64-pc-windows-msvc")
	default:
		return core.Fail(core.E("RustBuilder.rustTargetTriple", "unsupported Rust target: "+target.String(), nil))
	}
}

// Ensure RustBuilder implements the Builder interface.
var _ build.Builder = (*RustBuilder)(nil)
