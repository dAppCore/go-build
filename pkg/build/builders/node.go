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
// ok, err := b.Detect(storage.Local, ".")
func (b *NodeBuilder) Detect(fs storage.Medium, dir string) core.Result {
	return core.Ok(build.IsNodeProject(fs, dir))
}

// Build runs the project build script once per target and collects artifacts
// from the target-specific output directory.
//
// artifacts, err := b.Build(ctx, cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
func (b *NodeBuilder) Build(ctx context.Context, cfg *build.Config, targets []build.Target) core.Result {
	if cfg == nil {
		return core.Fail(core.E("NodeBuilder.Build", "config is nil", nil))
	}
	filesystem := ensureBuildFilesystem(cfg)

	targets = defaultRuntimeTargets(targets, runtime.GOOS, runtime.GOARCH)

	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = defaultOutputDir(cfg)
	}
	created := ensureOutputDir(filesystem, outputDir, "NodeBuilder.Build")
	if !created.OK {
		return created
	}

	projectDir := b.resolveNodeProjectDir(filesystem, cfg.ProjectDir)
	if projectDir == "" {
		projectDir = cfg.ProjectDir
	}

	commandResult := b.resolveBuildCommand(cfg, filesystem, projectDir)
	if !commandResult.OK {
		return commandResult
	}
	spec := commandResult.Value.(commandSpec)
	command := spec.command
	args := spec.args

	var artifacts []build.Artifact
	for _, target := range targets {
		platformDirResult := ensurePlatformDir(filesystem, outputDir, target, "NodeBuilder.Build")
		if !platformDirResult.OK {
			return platformDirResult
		}
		platformDir := platformDirResult.Value.(string)

		env := configuredTargetEnv(cfg, target, standardTargetValues(outputDir, platformDir, target)...)

		output := ax.CombinedOutput(ctx, projectDir, env, command, args...)
		if !output.OK {
			return core.Fail(core.E("NodeBuilder.Build", command+" build failed: "+output.Error(), core.NewError(output.Error())))
		}

		found := b.findArtifactsForTarget(cfg.FS, outputDir, target)
		artifacts = append(artifacts, found...)
	}

	return core.Ok(artifacts)
}

// resolveNodeProjectDir locates the directory containing package.json.
// It prefers the project root, then searches nested directories to depth 2.
func (b *NodeBuilder) resolveNodeProjectDir(fs storage.Medium, projectDir string) string {
	if b.hasNodeManifest(fs, projectDir) {
		return projectDir
	}

	return b.findNodeProjectDir(fs, projectDir, 0)
}

// findNodeProjectDir searches for a package.json within nested directories.
func (b *NodeBuilder) findNodeProjectDir(fs storage.Medium, dir string, depth int) string {
	if depth >= 2 {
		return ""
	}

	entriesResult := fs.List(dir)
	if !entriesResult.OK {
		return ""
	}
	entries := entriesResult.Value.([]stdfs.DirEntry)

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

func (b *NodeBuilder) hasNodeManifest(fs storage.Medium, dir string) bool {
	return fs.IsFile(ax.Join(dir, "package.json")) || b.hasDenoConfig(fs, dir)
}

func (b *NodeBuilder) hasDenoConfig(fs storage.Medium, dir string) bool {
	return fs.IsFile(ax.Join(dir, "deno.json")) || fs.IsFile(ax.Join(dir, "deno.jsonc"))
}

// resolvePackageManager selects the package manager from lockfiles.
//
// packageManager := b.resolvePackageManager(storage.Local, ".")
func (b *NodeBuilder) resolvePackageManager(fs storage.Medium, projectDir string) core.Result {
	if declared := detectDeclaredPackageManager(fs, projectDir); declared != "" {
		return core.Ok(declared)
	}

	switch {
	case fs.IsFile(ax.Join(projectDir, "bun.lockb")) || fs.IsFile(ax.Join(projectDir, "bun.lock")):
		return core.Ok("bun")
	case fs.IsFile(ax.Join(projectDir, "pnpm-lock.yaml")):
		return core.Ok("pnpm")
	case fs.IsFile(ax.Join(projectDir, "yarn.lock")):
		return core.Ok("yarn")
	case fs.IsFile(ax.Join(projectDir, "package-lock.json")):
		return core.Ok("npm")
	default:
		return core.Ok("npm")
	}
}

// resolveBuildCommand returns the executable and arguments for the selected package manager.
//
// command, args, err := b.resolveBuildCommand("npm")
func (b *NodeBuilder) resolveBuildCommand(cfg *build.Config, fs storage.Medium, projectDir string) core.Result {
	configuredDenoBuild := ""
	if cfg != nil {
		configuredDenoBuild = cfg.DenoBuild
	}

	if b.hasDenoConfig(fs, projectDir) || build.DenoRequested(configuredDenoBuild) {
		return resolveDenoBuildCommand(cfg, b.resolveDenoCli)
	}

	if build.NpmRequested(configuredNpmBuild(cfg)) {
		return resolveNpmBuildCommand(cfg, b.resolveNpmCli)
	}

	packageManagerResult := b.resolvePackageManager(fs, projectDir)
	if !packageManagerResult.OK {
		return packageManagerResult
	}
	packageManager := packageManagerResult.Value.(string)

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

	command := ax.ResolveCommand(packageManager, paths...)
	if !command.OK {
		return core.Fail(core.E("NodeBuilder.resolveBuildCommand", packageManager+" CLI not found", core.NewError(command.Error())))
	}

	switch packageManager {
	case "yarn":
		return core.Ok(commandSpec{command: command.Value.(string), args: []string{"build"}})
	default:
		return core.Ok(commandSpec{command: command.Value.(string), args: []string{"run", "build"}})
	}
}

func configuredNpmBuild(cfg *build.Config) string {
	if cfg == nil {
		return ""
	}
	return cfg.NpmBuild
}

func (b *NodeBuilder) resolveDenoCli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/deno",
			"/opt/homebrew/bin/deno",
		}
	}

	command := ax.ResolveCommand("deno", paths...)
	if !command.OK {
		return core.Fail(core.E("NodeBuilder.resolveDenoCli", "deno CLI not found. Install it from https://deno.com/runtime", core.NewError(command.Error())))
	}

	return command
}

func (b *NodeBuilder) resolveNpmCli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/npm",
			"/opt/homebrew/bin/npm",
		}
	}

	command := ax.ResolveCommand("npm", paths...)
	if !command.OK {
		return core.Fail(core.E("NodeBuilder.resolveNpmCli", "npm CLI not found. Install Node.js from https://nodejs.org/", core.NewError(command.Error())))
	}

	return command
}

// findArtifactsForTarget searches for build outputs in the target-specific output directory.
//
// artifacts := b.findArtifactsForTarget(storage.Local, "dist", build.Target{OS: "linux", Arch: "amd64"})
func (b *NodeBuilder) findArtifactsForTarget(fs storage.Medium, outputDir string, target build.Target) []build.Artifact {
	var artifacts []build.Artifact

	platformDir := ax.Join(outputDir, core.Sprintf("%s_%s", target.OS, target.Arch))
	if fs.IsDir(platformDir) {
		entriesResult := fs.List(platformDir)
		if entriesResult.OK {
			entries := entriesResult.Value.([]stdfs.DirEntry)
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
		entriesResult := fs.List(outputDir)
		if !entriesResult.OK {
			continue
		}
		entries := entriesResult.Value.([]stdfs.DirEntry)
		for _, entry := range entries {
			match := entry.Name()
			matched := core.PathMatch(pattern, match)
			if !matched.OK || !matched.Value.(bool) {
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
