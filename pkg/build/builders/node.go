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
	return build.IsNodeProject(fs, dir) || b.resolveNodeProjectDir(fs, dir) != "", nil
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

	projectDir := b.resolveNodeProjectDir(cfg.FS, cfg.ProjectDir)
	if projectDir == "" {
		projectDir = cfg.ProjectDir
	}

	command, args, err := b.resolveBuildCommand(cfg, cfg.FS, projectDir)
	if err != nil {
		return nil, err
	}

	var artifacts []build.Artifact
	for _, target := range targets {
		platformDir := ax.Join(outputDir, core.Sprintf("%s_%s", target.OS, target.Arch))
		if err := cfg.FS.EnsureDir(platformDir); err != nil {
			return artifacts, coreerr.E("NodeBuilder.Build", "failed to create platform directory", err)
		}

		env := appendConfiguredEnv(cfg,
			core.Sprintf("GOOS=%s", target.OS),
			core.Sprintf("GOARCH=%s", target.Arch),
			core.Sprintf("TARGET_OS=%s", target.OS),
			core.Sprintf("TARGET_ARCH=%s", target.Arch),
			core.Sprintf("OUTPUT_DIR=%s", outputDir),
			core.Sprintf("TARGET_DIR=%s", platformDir),
		)
		if cfg.Name != "" {
			env = append(env, core.Sprintf("NAME=%s", cfg.Name))
		}
		if cfg.Version != "" {
			env = append(env, core.Sprintf("VERSION=%s", cfg.Version))
		}

		output, err := ax.CombinedOutput(ctx, projectDir, env, command, args...)
		if err != nil {
			return artifacts, coreerr.E("NodeBuilder.Build", command+" build failed: "+output, err)
		}

		found := b.findArtifactsForTarget(cfg.FS, outputDir, target)
		artifacts = append(artifacts, found...)
	}

	return artifacts, nil
}

// resolveNodeProjectDir locates the directory containing package.json.
// It prefers the project root, then searches nested directories to depth 2.
func (b *NodeBuilder) resolveNodeProjectDir(fs io.Medium, projectDir string) string {
	if b.hasNodeManifest(fs, projectDir) {
		return projectDir
	}

	return b.findNodeProjectDir(fs, projectDir, 0)
}

// findNodeProjectDir searches for a package.json within nested directories.
func (b *NodeBuilder) findNodeProjectDir(fs io.Medium, dir string, depth int) string {
	if depth >= 2 {
		return ""
	}

	entries, err := fs.List(dir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		if name == "node_modules" || core.HasPrefix(name, ".") {
			continue
		}

		candidateDir := ax.Join(dir, name)
		if b.hasNodeManifest(fs, candidateDir) {
			return candidateDir
		}

		if nested := b.findNodeProjectDir(fs, candidateDir, depth+1); nested != "" {
			return nested
		}
	}

	return ""
}

func (b *NodeBuilder) hasNodeManifest(fs io.Medium, dir string) bool {
	return fs.IsFile(ax.Join(dir, "package.json")) || b.hasDenoConfig(fs, dir)
}

func (b *NodeBuilder) hasDenoConfig(fs io.Medium, dir string) bool {
	return fs.IsFile(ax.Join(dir, "deno.json")) || fs.IsFile(ax.Join(dir, "deno.jsonc"))
}

// resolvePackageManager selects the package manager from lockfiles.
//
// packageManager := b.resolvePackageManager(io.Local, ".")
func (b *NodeBuilder) resolvePackageManager(fs io.Medium, projectDir string) (string, error) {
	if declared := detectDeclaredPackageManager(fs, projectDir); declared != "" {
		return declared, nil
	}

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
func (b *NodeBuilder) resolveBuildCommand(cfg *build.Config, fs io.Medium, projectDir string) (string, []string, error) {
	configuredDenoBuild := ""
	if cfg != nil {
		configuredDenoBuild = cfg.DenoBuild
	}

	if b.hasDenoConfig(fs, projectDir) || build.DenoRequested(configuredDenoBuild) {
		return resolveDenoBuildCommand(cfg, b.resolveDenoCli)
	}

	packageManager, err := b.resolvePackageManager(fs, projectDir)
	if err != nil {
		return "", nil, err
	}

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

func (b *NodeBuilder) resolveDenoCli(paths ...string) (string, error) {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/deno",
			"/opt/homebrew/bin/deno",
		}
	}

	command, err := ax.ResolveCommand("deno", paths...)
	if err != nil {
		return "", coreerr.E("NodeBuilder.resolveDenoCli", "deno CLI not found. Install it from https://deno.com/runtime", err)
	}

	return command, nil
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
