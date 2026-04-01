// Package builders provides build implementations for different project types.
package builders

import (
	"context"
	"path"
	"runtime"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

// NodeBuilder builds Node.js projects with the detected package manager.
//
// b := builders.NewNodeBuilder()
type NodeBuilder struct{}

// NewNodeBuilder creates a new NodeBuilder instance.
//
// b := builders.NewNodeBuilder()
func NewNodeBuilder() *NodeBuilder {
	return &NodeBuilder{}
}

// Name returns the builder's identifier.
//
// name := b.Name() // → "node"
func (b *NodeBuilder) Name() string {
	return "node"
}

// Detect checks if this builder can handle the project in the given directory.
//
// ok, err := b.Detect(io.Local, ".")
func (b *NodeBuilder) Detect(fs io.Medium, dir string) (bool, error) {
	return build.IsNodeProject(fs, dir), nil
}

// Build runs the project build script once per target and collects artifacts
// from the target-specific output directory.
//
// artifacts, err := b.Build(ctx, cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
func (b *NodeBuilder) Build(ctx context.Context, cfg *build.Config, targets []build.Target) ([]build.Artifact, error) {
	if cfg == nil {
		return nil, coreerr.E("NodeBuilder.Build", "config is nil", nil)
	}

	if len(targets) == 0 {
		targets = []build.Target{{OS: runtime.GOOS, Arch: runtime.GOARCH}}
	}

	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = ax.Join(cfg.ProjectDir, "dist")
	}
	if err := cfg.FS.EnsureDir(outputDir); err != nil {
		return nil, coreerr.E("NodeBuilder.Build", "failed to create output directory", err)
	}

	packageManager, err := b.resolvePackageManager(cfg.FS, cfg.ProjectDir)
	if err != nil {
		return nil, err
	}

	command, args, err := b.resolveBuildCommand(packageManager)
	if err != nil {
		return nil, err
	}

	var artifacts []build.Artifact
	for _, target := range targets {
		platformDir := ax.Join(outputDir, core.Sprintf("%s_%s", target.OS, target.Arch))
		if err := cfg.FS.EnsureDir(platformDir); err != nil {
			return artifacts, coreerr.E("NodeBuilder.Build", "failed to create platform directory", err)
		}

		env := []string{
			core.Sprintf("GOOS=%s", target.OS),
			core.Sprintf("GOARCH=%s", target.Arch),
			core.Sprintf("TARGET_OS=%s", target.OS),
			core.Sprintf("TARGET_ARCH=%s", target.Arch),
			core.Sprintf("OUTPUT_DIR=%s", outputDir),
			core.Sprintf("TARGET_DIR=%s", platformDir),
		}
		if cfg.Name != "" {
			env = append(env, core.Sprintf("NAME=%s", cfg.Name))
		}
		if cfg.Version != "" {
			env = append(env, core.Sprintf("VERSION=%s", cfg.Version))
		}

		output, err := ax.CombinedOutput(ctx, cfg.ProjectDir, env, command, args...)
		if err != nil {
			return artifacts, coreerr.E("NodeBuilder.Build", command+" build failed: "+output, err)
		}

		found := b.findArtifactsForTarget(cfg.FS, outputDir, target)
		artifacts = append(artifacts, found...)
	}

	return artifacts, nil
}

// resolvePackageManager selects the package manager from lockfiles.
//
// packageManager := b.resolvePackageManager(io.Local, ".")
func (b *NodeBuilder) resolvePackageManager(fs io.Medium, projectDir string) (string, error) {
	switch {
	case fs.IsFile(ax.Join(projectDir, "bun.lockb")) || fs.IsFile(ax.Join(projectDir, "bun.lock")):
		return "bun", nil
	case fs.IsFile(ax.Join(projectDir, "pnpm-lock.yaml")):
		return "pnpm", nil
	case fs.IsFile(ax.Join(projectDir, "yarn.lock")):
		return "yarn", nil
	case fs.IsFile(ax.Join(projectDir, "package-lock.json")):
		return "npm", nil
	default:
		return "npm", nil
	}
}

// resolveBuildCommand returns the executable and arguments for the selected package manager.
//
// command, args, err := b.resolveBuildCommand("npm")
func (b *NodeBuilder) resolveBuildCommand(packageManager string) (string, []string, error) {
	var paths []string
	switch packageManager {
	case "bun":
		paths = []string{"/usr/local/bin/bun", "/opt/homebrew/bin/bun"}
	case "pnpm":
		paths = []string{"/usr/local/bin/pnpm", "/opt/homebrew/bin/pnpm"}
	case "yarn":
		paths = []string{"/usr/local/bin/yarn", "/opt/homebrew/bin/yarn"}
	default:
		paths = []string{"/usr/local/bin/npm", "/opt/homebrew/bin/npm"}
		packageManager = "npm"
	}

	command, err := ax.ResolveCommand(packageManager, paths...)
	if err != nil {
		return "", nil, coreerr.E("NodeBuilder.resolveBuildCommand", packageManager+" CLI not found", err)
	}

	switch packageManager {
	case "yarn":
		return command, []string{"build"}, nil
	default:
		return command, []string{"run", "build"}, nil
	}
}

// findArtifactsForTarget searches for build outputs in the target-specific output directory.
//
// artifacts := b.findArtifactsForTarget(io.Local, "dist", build.Target{OS: "linux", Arch: "amd64"})
func (b *NodeBuilder) findArtifactsForTarget(fs io.Medium, outputDir string, target build.Target) []build.Artifact {
	var artifacts []build.Artifact

	platformDir := ax.Join(outputDir, core.Sprintf("%s_%s", target.OS, target.Arch))
	if fs.IsDir(platformDir) {
		entries, err := fs.List(platformDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					if target.OS == "darwin" && core.HasSuffix(entry.Name(), ".app") {
						artifacts = append(artifacts, build.Artifact{
							Path: ax.Join(platformDir, entry.Name()),
							OS:   target.OS,
							Arch: target.Arch,
						})
					}
					continue
				}

				name := entry.Name()
				if core.HasPrefix(name, ".") || name == "CHECKSUMS.txt" {
					continue
				}

				artifacts = append(artifacts, build.Artifact{
					Path: ax.Join(platformDir, name),
					OS:   target.OS,
					Arch: target.Arch,
				})
			}
		}
		if len(artifacts) > 0 {
			return artifacts
		}
	}

	patterns := []string{
		core.Sprintf("*-%s-%s*", target.OS, target.Arch),
		core.Sprintf("*_%s_%s*", target.OS, target.Arch),
		core.Sprintf("*-%s*", target.Arch),
	}

	for _, pattern := range patterns {
		entries, err := fs.List(outputDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			match := entry.Name()
			matched, _ := path.Match(pattern, match)
			if !matched {
				continue
			}
			fullPath := ax.Join(outputDir, match)
			if fs.IsDir(fullPath) {
				continue
			}

			artifacts = append(artifacts, build.Artifact{
				Path: fullPath,
				OS:   target.OS,
				Arch: target.Arch,
			})
		}
		if len(artifacts) > 0 {
			break
		}
	}

	return artifacts
}

// Ensure NodeBuilder implements the Builder interface.
var _ build.Builder = (*NodeBuilder)(nil)
