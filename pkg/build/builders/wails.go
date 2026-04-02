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

// WailsBuilder implements the Builder interface for Wails v3 projects.
//
// b := builders.NewWailsBuilder()
type WailsBuilder struct{}

// NewWailsBuilder creates a new WailsBuilder instance.
//
// b := builders.NewWailsBuilder()
func NewWailsBuilder() *WailsBuilder {
	return &WailsBuilder{}
}

// Name returns the builder's identifier.
//
// name := b.Name() // → "wails"
func (b *WailsBuilder) Name() string {
	return "wails"
}

// Detect checks if this builder can handle the project (checks for wails.json).
//
// ok, err := b.Detect(io.Local, ".")
func (b *WailsBuilder) Detect(fs io.Medium, dir string) (bool, error) {
	return build.IsWailsProject(fs, dir), nil
}

// Build compiles the Wails project for the specified targets.
// Wails v3: delegates to Taskfile; Wails v2: uses 'wails build'.
//
// artifacts, err := b.Build(ctx, cfg, []build.Target{{OS: "darwin", Arch: "arm64"}})
func (b *WailsBuilder) Build(ctx context.Context, cfg *build.Config, targets []build.Target) ([]build.Artifact, error) {
	if cfg == nil {
		return nil, coreerr.E("WailsBuilder.Build", "config is nil", nil)
	}

	if len(targets) == 0 {
		return nil, coreerr.E("WailsBuilder.Build", "no targets specified", nil)
	}

	// Detect Wails version
	isV3 := b.isWailsV3(cfg.FS, cfg.ProjectDir)

	if isV3 {
		// Wails v3 strategy: Delegate to Taskfile if present, otherwise use Go builder with CGO
		taskBuilder := NewTaskfileBuilder()
		if detected, _ := taskBuilder.Detect(cfg.FS, cfg.ProjectDir); detected {
			return taskBuilder.Build(ctx, cfg, targets)
		}
		// Fall back to Go builder — Wails v3 is just a Go project that needs CGO
		v3Config := b.buildV3Config(cfg)
		goBuilder := NewGoBuilder()
		return goBuilder.Build(ctx, v3Config, targets)
	}

	// Wails v2 strategy: Use 'wails build'
	if err := b.PreBuild(ctx, cfg); err != nil {
		return nil, err
	}

	// Ensure output directory exists
	if err := cfg.FS.EnsureDir(cfg.OutputDir); err != nil {
		return nil, coreerr.E("WailsBuilder.Build", "failed to create output directory", err)
	}

	// Note: Wails v2 handles frontend installation/building automatically via wails.json config

	var artifacts []build.Artifact

	for _, target := range targets {
		artifact, err := b.buildV2Target(ctx, cfg, target)
		if err != nil {
			return artifacts, coreerr.E("WailsBuilder.Build", "failed to build "+target.String(), err)
		}
		artifacts = append(artifacts, artifact)
	}

	return artifacts, nil
}

// buildV3Config returns a copy of the build config with Wails v3 requirements applied.
func (b *WailsBuilder) buildV3Config(cfg *build.Config) *build.Config {
	if cfg == nil {
		return nil
	}

	v3Config := *cfg
	v3Config.CGO = true
	return &v3Config
}

// PreBuild runs the frontend build step before Wails compiles the desktop app.
//
// err := b.PreBuild(ctx, cfg) // runs `deno task build` or `npm run build`
func (b *WailsBuilder) PreBuild(ctx context.Context, cfg *build.Config) error {
	if cfg == nil {
		return coreerr.E("WailsBuilder.PreBuild", "config is nil", nil)
	}

	frontendDir, command, args, err := b.resolveFrontendBuild(cfg.FS, cfg.ProjectDir)
	if err != nil {
		return err
	}
	if command == "" {
		return nil
	}

	output, err := ax.CombinedOutput(ctx, frontendDir, cfg.Env, command, args...)
	if err != nil {
		return coreerr.E("WailsBuilder.PreBuild", command+" build failed: "+output, err)
	}

	return nil
}

// isWailsV3 checks if the project uses Wails v3 by inspecting go.mod.
func (b *WailsBuilder) isWailsV3(fs io.Medium, dir string) bool {
	goModPath := ax.Join(dir, "go.mod")
	content, err := fs.Read(goModPath)
	if err != nil {
		return false
	}
	return core.Contains(content, "github.com/wailsapp/wails/v3")
}

// resolveFrontendBuild selects the frontend directory and build command.
//
// dir, command, args, err := b.resolveFrontendBuild(io.Local, ".")
func (b *WailsBuilder) resolveFrontendBuild(fs io.Medium, projectDir string) (string, string, []string, error) {
	frontendDir := b.resolveFrontendDir(fs, projectDir)
	if frontendDir == "" {
		return "", "", nil, nil
	}

	if b.hasDenoConfig(fs, frontendDir) {
		command, err := b.resolveDenoCli()
		if err != nil {
			return "", "", nil, err
		}
		return frontendDir, command, []string{"task", "build"}, nil
	}

	if fs.IsFile(ax.Join(frontendDir, "package.json")) {
		packageManager := detectPackageManager(fs, frontendDir)
		return b.resolvePackageManagerBuild(frontendDir, packageManager)
	}

	return "", "", nil, nil
}

// resolvePackageManagerBuild returns the frontend build command for a detected package manager.
func (b *WailsBuilder) resolvePackageManagerBuild(frontendDir, packageManager string) (string, string, []string, error) {
	switch packageManager {
	case "bun":
		command, err := b.resolveBunCli()
		if err != nil {
			return "", "", nil, err
		}
		return frontendDir, command, []string{"run", "build"}, nil
	case "pnpm":
		command, err := b.resolvePnpmCli()
		if err != nil {
			return "", "", nil, err
		}
		return frontendDir, command, []string{"run", "build"}, nil
	case "yarn":
		command, err := b.resolveYarnCli()
		if err != nil {
			return "", "", nil, err
		}
		return frontendDir, command, []string{"build"}, nil
	default:
		command, err := b.resolveNpmCli()
		if err != nil {
			return "", "", nil, err
		}
		return frontendDir, command, []string{"run", "build"}, nil
	}
}

// resolveFrontendDir returns the directory that contains the frontend build manifest.
func (b *WailsBuilder) resolveFrontendDir(fs io.Medium, projectDir string) string {
	frontendDir := ax.Join(projectDir, "frontend")
	if fs.IsDir(frontendDir) && (b.hasDenoConfig(fs, frontendDir) || fs.IsFile(ax.Join(frontendDir, "package.json"))) {
		return frontendDir
	}

	if b.hasDenoConfig(fs, projectDir) || fs.IsFile(ax.Join(projectDir, "package.json")) {
		return projectDir
	}

	if nestedFrontendDir := b.resolveSubtreeFrontendDir(fs, projectDir); nestedFrontendDir != "" {
		return nestedFrontendDir
	}

	return ""
}

// hasDenoConfig reports whether the frontend directory contains a Deno manifest.
func (b *WailsBuilder) hasDenoConfig(fs io.Medium, dir string) bool {
	return fs.IsFile(ax.Join(dir, "deno.json")) || fs.IsFile(ax.Join(dir, "deno.jsonc"))
}

// resolveSubtreeFrontendDir finds a nested frontend manifest within the project tree.
// This supports monorepo layouts such as apps/web/package.json or apps/web/deno.json
// when frontend/ is absent.
func (b *WailsBuilder) resolveSubtreeFrontendDir(fs io.Medium, projectDir string) string {
	return b.findFrontendDir(fs, projectDir)
}

// findFrontendDir walks nested directories until it finds a frontend manifest.
func (b *WailsBuilder) findFrontendDir(fs io.Medium, dir string) string {
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
		if b.hasDenoConfig(fs, candidateDir) || fs.IsFile(ax.Join(candidateDir, "package.json")) {
			return candidateDir
		}

		if nested := b.findFrontendDir(fs, candidateDir); nested != "" {
			return nested
		}
	}

	return ""
}

// buildV2Target compiles for a single target platform using wails (v2).
func (b *WailsBuilder) buildV2Target(ctx context.Context, cfg *build.Config, target build.Target) (build.Artifact, error) {
	wailsCommand, err := b.resolveWailsCli()
	if err != nil {
		return build.Artifact{}, err
	}

	// Determine output binary name
	binaryName := cfg.Name
	if binaryName == "" {
		binaryName = ax.Base(cfg.ProjectDir)
	}

	// Build the wails build arguments
	args := []string{"build"}

	if cfg.NSIS {
		args = append(args, "-nsis")
	}

	if cfg.WebView2 != "" {
		args = append(args, "-webview2", cfg.WebView2)
	}

	// Platform
	args = append(args, "-platform", core.Sprintf("%s/%s", target.OS, target.Arch))

	// Output (Wails v2 uses -o for the binary name, relative to build/bin usually, but we want to control it)
	// Actually, Wails v2 is opinionated about output dir (build/bin).
	// We might need to copy artifacts after build if we want them in cfg.OutputDir.
	// For now, let's try to let Wails do its thing and find the artifact.

	// Capture output for error messages
	output, err := ax.CombinedOutput(ctx, cfg.ProjectDir, cfg.Env, wailsCommand, args...)
	if err != nil {
		return build.Artifact{}, coreerr.E("WailsBuilder.buildV2Target", "wails build failed: "+output, err)
	}

	// Wails v2 typically outputs to build/bin
	// We need to move/copy it to our desired output dir

	// Construct the source path where Wails v2 puts the binary
	wailsOutputDir := ax.Join(cfg.ProjectDir, "build", "bin")

	// Find the artifact in Wails output dir
	sourcePath, err := b.findArtifact(cfg.FS, wailsOutputDir, binaryName, target)
	if err != nil {
		return build.Artifact{}, coreerr.E("WailsBuilder.buildV2Target", "failed to find Wails v2 build artifact", err)
	}

	// Move/Copy to our output dir
	// Create platform specific dir in our output
	platformDir := ax.Join(cfg.OutputDir, core.Sprintf("%s_%s", target.OS, target.Arch))
	if err := cfg.FS.EnsureDir(platformDir); err != nil {
		return build.Artifact{}, coreerr.E("WailsBuilder.buildV2Target", "failed to create output dir", err)
	}

	destPath := ax.Join(platformDir, ax.Base(sourcePath))

	// Copy the selected artifact, preserving directory bundles such as .app packages.
	if err := copyBuildArtifact(cfg.FS, sourcePath, destPath); err != nil {
		return build.Artifact{}, coreerr.E("WailsBuilder.buildV2Target", "failed to copy artifact "+sourcePath, err)
	}

	return build.Artifact{
		Path: destPath,
		OS:   target.OS,
		Arch: target.Arch,
	}, nil
}

// findArtifact locates the built artifact based on the target platform.
func (b *WailsBuilder) findArtifact(fs io.Medium, platformDir, binaryName string, target build.Target) (string, error) {
	var candidates []string

	switch target.OS {
	case "windows":
		// Look for NSIS installer first, then plain exe
		candidates = []string{
			ax.Join(platformDir, binaryName+"-installer.exe"),
			ax.Join(platformDir, binaryName+".exe"),
			ax.Join(platformDir, binaryName+"-amd64-installer.exe"),
		}
	case "darwin":
		// Look for .dmg, then .app bundle, then plain binary
		candidates = []string{
			ax.Join(platformDir, binaryName+".dmg"),
			ax.Join(platformDir, binaryName+".app"),
			ax.Join(platformDir, binaryName),
		}
	default:
		// Linux and others: look for plain binary
		candidates = []string{
			ax.Join(platformDir, binaryName),
		}
	}

	// Try each candidate
	for _, candidate := range candidates {
		if fs.Exists(candidate) {
			return candidate, nil
		}
	}

	// If no specific candidate found, try to find any executable or package in the directory
	entries, err := fs.List(platformDir)
	if err != nil {
		return "", coreerr.E("WailsBuilder.findArtifact", "failed to read platform directory", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		// Skip common non-artifact files
		if core.HasSuffix(name, ".go") || core.HasSuffix(name, ".json") {
			continue
		}

		path := ax.Join(platformDir, name)
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// On Unix, check if it's executable; on Windows, check for .exe
		if target.OS == "windows" {
			if core.HasSuffix(name, ".exe") {
				return path, nil
			}
		} else if info.Mode()&0111 != 0 || entry.IsDir() {
			// Executable file or directory (.app bundle)
			return path, nil
		}
	}

	return "", coreerr.E("WailsBuilder.findArtifact", "no artifact found in "+platformDir, nil)
}

// copyBuildArtifact copies a file or directory artifact into the build output tree.
//
// err := copyBuildArtifact(io.Local, "/tmp/source.app", "/tmp/dist/source.app")
func copyBuildArtifact(fs io.Medium, sourcePath, destPath string) error {
	if fs.IsDir(sourcePath) {
		if err := fs.EnsureDir(destPath); err != nil {
			return err
		}

		entries, err := fs.List(sourcePath)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			childSource := ax.Join(sourcePath, entry.Name())
			childDest := ax.Join(destPath, entry.Name())
			if err := copyBuildArtifact(fs, childSource, childDest); err != nil {
				return err
			}
		}

		return nil
	}

	info, err := fs.Stat(sourcePath)
	if err != nil {
		return err
	}

	content, err := fs.Read(sourcePath)
	if err != nil {
		return err
	}

	if err := fs.WriteMode(destPath, content, info.Mode().Perm()); err != nil {
		return err
	}

	return nil
}

// resolveWailsCli returns the executable path for the wails CLI.
func (b *WailsBuilder) resolveWailsCli(paths ...string) (string, error) {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/wails",
			"/opt/homebrew/bin/wails",
		}

		if home := core.Env("HOME"); home != "" {
			paths = append(paths, ax.Join(home, "go", "bin", "wails"))
		}
	}

	command, err := ax.ResolveCommand("wails", paths...)
	if err != nil {
		return "", coreerr.E("WailsBuilder.resolveWailsCli", "wails CLI not found. Install it with: go install github.com/wailsapp/wails/v2/cmd/wails@latest", err)
	}

	return command, nil
}

// resolveDenoCli returns the executable path for the deno CLI.
func (b *WailsBuilder) resolveDenoCli(paths ...string) (string, error) {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/deno",
			"/opt/homebrew/bin/deno",
		}
	}

	command, err := ax.ResolveCommand("deno", paths...)
	if err != nil {
		return "", coreerr.E("WailsBuilder.resolveDenoCli", "deno CLI not found. Install it from https://deno.com/runtime", err)
	}

	return command, nil
}

// resolveNpmCli returns the executable path for the npm CLI.
func (b *WailsBuilder) resolveNpmCli(paths ...string) (string, error) {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/npm",
			"/opt/homebrew/bin/npm",
		}
	}

	command, err := ax.ResolveCommand("npm", paths...)
	if err != nil {
		return "", coreerr.E("WailsBuilder.resolveNpmCli", "npm CLI not found. Install Node.js from https://nodejs.org/", err)
	}

	return command, nil
}

// resolveBunCli returns the executable path for the bun CLI.
func (b *WailsBuilder) resolveBunCli(paths ...string) (string, error) {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/bun",
			"/opt/homebrew/bin/bun",
		}
	}

	command, err := ax.ResolveCommand("bun", paths...)
	if err != nil {
		return "", coreerr.E("WailsBuilder.resolveBunCli", "bun CLI not found. Install it from https://bun.sh/", err)
	}

	return command, nil
}

// resolvePnpmCli returns the executable path for the pnpm CLI.
func (b *WailsBuilder) resolvePnpmCli(paths ...string) (string, error) {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/pnpm",
			"/opt/homebrew/bin/pnpm",
		}
	}

	command, err := ax.ResolveCommand("pnpm", paths...)
	if err != nil {
		return "", coreerr.E("WailsBuilder.resolvePnpmCli", "pnpm CLI not found. Install it from https://pnpm.io/installation", err)
	}

	return command, nil
}

// resolveYarnCli returns the executable path for the yarn CLI.
func (b *WailsBuilder) resolveYarnCli(paths ...string) (string, error) {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/yarn",
			"/opt/homebrew/bin/yarn",
		}
	}

	command, err := ax.ResolveCommand("yarn", paths...)
	if err != nil {
		return "", coreerr.E("WailsBuilder.resolveYarnCli", "yarn CLI not found. Install it from https://yarnpkg.com/getting-started/install", err)
	}

	return command, nil
}

// detectPackageManager detects the frontend package manager based on lock files.
// Returns "bun", "pnpm", "yarn", or "npm" (default).
func detectPackageManager(fs io.Medium, dir string) string {
	if declared := detectDeclaredPackageManager(fs, dir); declared != "" {
		return declared
	}

	// Check in priority order: bun, pnpm, yarn, npm
	lockFiles := []struct {
		file    string
		manager string
	}{
		{"bun.lock", "bun"},
		{"bun.lockb", "bun"},
		{"pnpm-lock.yaml", "pnpm"},
		{"yarn.lock", "yarn"},
		{"package-lock.json", "npm"},
	}

	for _, lf := range lockFiles {
		if fs.IsFile(ax.Join(dir, lf.file)) {
			return lf.manager
		}
	}

	// Default to npm if no lock file found
	return "npm"
}

// Ensure WailsBuilder implements the Builder interface.
var _ build.Builder = (*WailsBuilder)(nil)
