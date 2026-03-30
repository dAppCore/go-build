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
// Usage example: declare a value of type builders.WailsBuilder in integrating code.
type WailsBuilder struct{}

// NewWailsBuilder creates a new WailsBuilder instance.
// Usage example: call builders.NewWailsBuilder(...) from integrating code.
func NewWailsBuilder() *WailsBuilder {
	return &WailsBuilder{}
}

// Name returns the builder's identifier.
// Usage example: call value.Name(...) from integrating code.
func (b *WailsBuilder) Name() string {
	return "wails"
}

// Detect checks if this builder can handle the project in the given directory.
// Uses IsWailsProject from the build package which checks for wails.json.
// Usage example: call value.Detect(...) from integrating code.
func (b *WailsBuilder) Detect(fs io.Medium, dir string) (bool, error) {
	return build.IsWailsProject(fs, dir), nil
}

// Build compiles the Wails project for the specified targets.
// It detects the Wails version and chooses the appropriate build strategy:
// - Wails v3: Delegates to Taskfile (error if missing)
// - Wails v2: Uses 'wails build' command
// Usage example: call value.Build(...) from integrating code.
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
		cfg.CGO = true
		goBuilder := NewGoBuilder()
		return goBuilder.Build(ctx, cfg, targets)
	}

	// Wails v2 strategy: Use 'wails build'
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

// isWailsV3 checks if the project uses Wails v3 by inspecting go.mod.
func (b *WailsBuilder) isWailsV3(fs io.Medium, dir string) bool {
	goModPath := ax.Join(dir, "go.mod")
	content, err := fs.Read(goModPath)
	if err != nil {
		return false
	}
	return core.Contains(content, "github.com/wailsapp/wails/v3")
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

	// Platform
	args = append(args, "-platform", core.Sprintf("%s/%s", target.OS, target.Arch))

	// Output (Wails v2 uses -o for the binary name, relative to build/bin usually, but we want to control it)
	// Actually, Wails v2 is opinionated about output dir (build/bin).
	// We might need to copy artifacts after build if we want them in cfg.OutputDir.
	// For now, let's try to let Wails do its thing and find the artifact.

	// Capture output for error messages
	output, err := ax.CombinedOutput(ctx, cfg.ProjectDir, nil, wailsCommand, args...)
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

	// Simple copy using the medium
	content, err := cfg.FS.Read(sourcePath)
	if err != nil {
		return build.Artifact{}, coreerr.E("WailsBuilder.buildV2Target", "failed to read artifact "+sourcePath, err)
	}
	if err := cfg.FS.Write(destPath, content); err != nil {
		return build.Artifact{}, coreerr.E("WailsBuilder.buildV2Target", "failed to write artifact "+destPath, err)
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

// detectPackageManager detects the frontend package manager based on lock files.
// Returns "bun", "pnpm", "yarn", or "npm" (default).
func detectPackageManager(fs io.Medium, dir string) string {
	// Check in priority order: bun, pnpm, yarn, npm
	lockFiles := []struct {
		file    string
		manager string
	}{
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
