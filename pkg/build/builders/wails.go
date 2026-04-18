// Package builders provides build implementations for different project types.
package builders

import (
	"context"
	"os"
	"runtime"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/core"
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
	filesystem := ensureBuildFilesystem(cfg)

	if len(targets) == 0 {
		return nil, coreerr.E("WailsBuilder.Build", "no targets specified", nil)
	}

	if _, err := build.VersionLinkerFlag(cfg.Version); err != nil {
		return nil, err
	}

	if cfg.OutputDir == "" {
		cfg.OutputDir = ax.Join(cfg.ProjectDir, "dist")
	}

	// Detect Wails version
	isV3 := b.isWailsV3(filesystem, cfg.ProjectDir)

	if isV3 {
		// Wails v3 projects already ship Taskfiles. Prefer them when present because
		// they capture project-specific packaging logic. Fall back to the CLI when a
		// project is Wails-backed but does not expose Task targets.
		taskBuilder := NewTaskfileBuilder()
		if detected, _ := taskBuilder.Detect(filesystem, cfg.ProjectDir); detected {
			return taskBuilder.Build(ctx, b.buildV3Config(cfg), targets)
		}

		if err := b.PreBuild(ctx, cfg); err != nil {
			return nil, err
		}

		var artifacts []build.Artifact
		for _, target := range targets {
			artifact, err := b.buildV3Target(ctx, cfg, target)
			if err != nil {
				return artifacts, coreerr.E("WailsBuilder.Build", "failed to build "+target.String(), err)
			}
			artifacts = append(artifacts, artifact)
		}

		return artifacts, nil
	}

	// Wails v2 strategy: Use 'wails build'
	if err := b.PreBuild(ctx, cfg); err != nil {
		return nil, err
	}

	// Ensure output directory exists
	if err := filesystem.EnsureDir(cfg.OutputDir); err != nil {
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

// buildV3Target builds a Wails v3 project for a single target using the wails3 CLI.
func (b *WailsBuilder) buildV3Target(ctx context.Context, cfg *build.Config, target build.Target) (build.Artifact, error) {
	filesystem := ensureBuildFilesystem(cfg)

	wailsCommand, err := b.resolveWails3Cli()
	if err != nil {
		return build.Artifact{}, err
	}

	binaryName := cfg.Name
	if binaryName == "" {
		binaryName = ax.Base(cfg.ProjectDir)
	}

	verb := "build"
	args := []string{verb, "GOOS=" + target.OS, "GOARCH=" + target.Arch}
	if cfg.NSIS && target.OS == "windows" {
		verb = "package"
		args[0] = verb
	}
	taskVars, err := buildV3TaskVars(cfg, target)
	if err != nil {
		return build.Artifact{}, err
	}
	args = append(args, taskVars...)

	env := appendConfiguredEnv(cfg,
		core.Sprintf("GOOS=%s", target.OS),
		core.Sprintf("GOARCH=%s", target.Arch),
		core.Sprintf("TARGET_OS=%s", target.OS),
		core.Sprintf("TARGET_ARCH=%s", target.Arch),
		core.Sprintf("OUTPUT_DIR=%s", cfg.OutputDir),
	)
	if cfg.Version != "" {
		env = append(env, core.Sprintf("VERSION=%s", cfg.Version))
	}
	if binaryName != "" {
		env = append(env, core.Sprintf("NAME=%s", binaryName))
	}
	if goflags, err := buildV3GoFlags(cfg); err != nil {
		return build.Artifact{}, err
	} else if goflags != "" {
		env = append(env, "GOFLAGS="+goflags)
	}
	if cfg.CGO {
		env = append(env, "CGO_ENABLED=1")
	}
	cleanup := func() {}
	if cfg.Obfuscate {
		env, cleanup, err = b.prepareV3Obfuscation(env)
		if err != nil {
			return build.Artifact{}, err
		}
		defer cleanup()
	}

	output, err := ax.CombinedOutput(ctx, cfg.ProjectDir, env, wailsCommand, args...)
	if err != nil {
		return build.Artifact{}, coreerr.E("WailsBuilder.buildV3Target", "wails3 "+verb+" failed: "+output, err)
	}

	sourcePath, err := b.findV3Artifact(filesystem, cfg.ProjectDir, binaryName, target, verb == "package")
	if err != nil {
		return build.Artifact{}, err
	}

	platformDir := ax.Join(cfg.OutputDir, core.Sprintf("%s_%s", target.OS, target.Arch))
	if err := filesystem.EnsureDir(platformDir); err != nil {
		return build.Artifact{}, coreerr.E("WailsBuilder.buildV3Target", "failed to create output dir", err)
	}

	destPath := ax.Join(platformDir, ax.Base(sourcePath))
	if err := copyBuildArtifact(filesystem, sourcePath, destPath); err != nil {
		return build.Artifact{}, coreerr.E("WailsBuilder.buildV3Target", "failed to copy artifact "+sourcePath, err)
	}

	return build.Artifact{
		Path: destPath,
		OS:   target.OS,
		Arch: target.Arch,
	}, nil
}

// PreBuild runs the frontend build step before Wails compiles the desktop app.
//
// err := b.PreBuild(ctx, cfg) // runs `deno task build` or `npm run build`
func (b *WailsBuilder) PreBuild(ctx context.Context, cfg *build.Config) error {
	if cfg == nil {
		return coreerr.E("WailsBuilder.PreBuild", "config is nil", nil)
	}

	frontendDir, command, args, err := b.resolveFrontendBuild(cfg)
	if err != nil {
		return err
	}
	if command == "" {
		return nil
	}

	output, err := ax.CombinedOutput(ctx, frontendDir, build.BuildEnvironment(cfg), command, args...)
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
// dir, command, args, err := b.resolveFrontendBuild(cfg)
func (b *WailsBuilder) resolveFrontendBuild(cfg *build.Config) (string, string, []string, error) {
	if cfg == nil {
		return "", "", nil, coreerr.E("WailsBuilder.resolveFrontendBuild", "config is nil", nil)
	}

	fs := cfg.FS
	if fs == nil {
		fs = io.Local
	}
	projectDir := cfg.ProjectDir
	frontendDir := b.resolveFrontendDir(fs, projectDir)
	if frontendDir == "" {
		if build.DenoRequested(cfg.DenoBuild) {
			if fs.IsDir(ax.Join(projectDir, "frontend")) {
				frontendDir = ax.Join(projectDir, "frontend")
			} else {
				frontendDir = projectDir
			}
		} else {
			return "", "", nil, nil
		}
	}

	if b.hasDenoConfig(fs, frontendDir) || build.DenoRequested(cfg.DenoBuild) {
		command, args, err := resolveDenoBuildCommand(cfg, b.resolveDenoCli)
		if err != nil {
			return "", "", nil, err
		}
		return frontendDir, command, args, nil
	}

	if build.NpmRequested(cfg.NpmBuild) {
		command, args, err := resolveNpmBuildCommand(cfg, b.resolveNpmCli)
		if err != nil {
			return "", "", nil, err
		}
		return frontendDir, command, args, nil
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

	if build.DenoRequested("") {
		if fs.IsDir(frontendDir) {
			return frontendDir
		}
		return projectDir
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
	return b.findFrontendDir(fs, projectDir, 0)
}

// findFrontendDir walks nested directories until it finds a frontend manifest.
// The v3 discovery contract only scans to depth 2 for monorepo frontends.
func (b *WailsBuilder) findFrontendDir(fs io.Medium, dir string, depth int) string {
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
		if b.hasDenoConfig(fs, candidateDir) || fs.IsFile(ax.Join(candidateDir, "package.json")) {
			return candidateDir
		}

		if nested := b.findFrontendDir(fs, candidateDir, depth+1); nested != "" {
			return nested
		}
	}

	return ""
}

// buildV2Target compiles for a single target platform using wails (v2).
func (b *WailsBuilder) buildV2Target(ctx context.Context, cfg *build.Config, target build.Target) (build.Artifact, error) {
	filesystem := ensureBuildFilesystem(cfg)

	if cfg.WebView2 != "" && target.OS == "windows" {
		if err := validateWebView2Mode(cfg.WebView2); err != nil {
			return build.Artifact{}, err
		}
	}

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

	// Honour the action/CLI build-name override by forwarding it to Wails v2.
	if binaryName != "" {
		args = append(args, "-o", binaryName)
	}

	if len(cfg.BuildTags) > 0 {
		args = append(args, "-tags", core.Join(",", cfg.BuildTags...))
	}

	ldflags := append([]string{}, cfg.LDFlags...)
	if cfg.Version != "" && !hasVersionLDFlag(ldflags) {
		versionFlag, err := build.VersionLinkerFlag(cfg.Version)
		if err != nil {
			return build.Artifact{}, err
		}
		ldflags = append(ldflags, versionFlag)
	}
	if len(ldflags) > 0 {
		args = append(args, "-ldflags", core.Join(" ", ldflags...))
	}

	if cfg.Obfuscate {
		args = append(args, "-obfuscated")
	}

	if cfg.NSIS && target.OS == "windows" {
		args = append(args, "-nsis")
	}

	if cfg.WebView2 != "" && target.OS == "windows" {
		args = append(args, "-webview2", cfg.WebView2)
	}

	// Platform
	args = append(args, "-platform", core.Sprintf("%s/%s", target.OS, target.Arch))

	// Output (Wails v2 uses -o for the binary name, relative to build/bin usually, but we want to control it)
	// Actually, Wails v2 is opinionated about output dir (build/bin).
	// We might need to copy artifacts after build if we want them in cfg.OutputDir.
	// For now, let's try to let Wails do its thing and find the artifact.

	// Capture output for error messages
	output, err := ax.CombinedOutput(ctx, cfg.ProjectDir, build.BuildEnvironment(cfg), wailsCommand, args...)
	if err != nil {
		return build.Artifact{}, coreerr.E("WailsBuilder.buildV2Target", "wails build failed: "+output, err)
	}

	// Wails v2 typically outputs to build/bin
	// We need to move/copy it to our desired output dir

	// Construct the source path where Wails v2 puts the binary
	wailsOutputDir := ax.Join(cfg.ProjectDir, "build", "bin")

	// Find the artifact in Wails output dir
	sourcePath, err := b.findArtifact(filesystem, wailsOutputDir, binaryName, target)
	if err != nil {
		return build.Artifact{}, coreerr.E("WailsBuilder.buildV2Target", "failed to find Wails v2 build artifact", err)
	}

	// Move/Copy to our output dir
	// Create platform specific dir in our output
	platformDir := ax.Join(cfg.OutputDir, core.Sprintf("%s_%s", target.OS, target.Arch))
	if err := filesystem.EnsureDir(platformDir); err != nil {
		return build.Artifact{}, coreerr.E("WailsBuilder.buildV2Target", "failed to create output dir", err)
	}

	destPath := ax.Join(platformDir, ax.Base(sourcePath))

	// Copy the selected artifact, preserving directory bundles such as .app packages.
	if err := copyBuildArtifact(filesystem, sourcePath, destPath); err != nil {
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

func (b *WailsBuilder) findV3Artifact(fs io.Medium, projectDir, binaryName string, target build.Target, packaged bool) (string, error) {
	if packaged && target.OS == "windows" {
		for _, candidate := range []string{
			ax.Join(projectDir, "build", "windows", "nsis", binaryName+"-installer.exe"),
			ax.Join(projectDir, "bin", binaryName+"-installer.exe"),
		} {
			if fs.Exists(candidate) {
				return candidate, nil
			}
		}
	}

	for _, platformDir := range []string{
		ax.Join(projectDir, "build", "bin"),
		ax.Join(projectDir, "bin"),
	} {
		path, err := b.findArtifact(fs, platformDir, binaryName, target)
		if err == nil {
			return path, nil
		}
	}

	return "", coreerr.E("WailsBuilder.findV3Artifact", "no artifact found for "+target.String(), nil)
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

func (b *WailsBuilder) resolveWails3Cli(paths ...string) (string, error) {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/wails3",
			"/opt/homebrew/bin/wails3",
		}

		if home := core.Env("HOME"); home != "" {
			paths = append(paths, ax.Join(home, "go", "bin", "wails3"))
		}
	}

	command, err := ax.ResolveCommand("wails3", paths...)
	if err != nil {
		return "", coreerr.E("WailsBuilder.resolveWails3Cli", "wails3 CLI not found. Install Wails v3 or expose it on PATH.", err)
	}

	return command, nil
}

func buildV3GoFlags(cfg *build.Config) (string, error) {
	if cfg == nil {
		return "", nil
	}

	var flags []string
	if !containsString(cfg.Flags, "-trimpath") {
		flags = append(flags, "-trimpath")
	}
	flags = append(flags, cfg.Flags...)

	if len(cfg.BuildTags) > 0 {
		flags = append(flags, "-tags="+core.Join(",", cfg.BuildTags...))
	}

	ldflags := append([]string{}, cfg.LDFlags...)
	if cfg.Version != "" && !hasVersionLDFlag(ldflags) {
		versionFlag, err := build.VersionLinkerFlag(cfg.Version)
		if err != nil {
			return "", err
		}
		ldflags = append(ldflags, versionFlag)
	}
	if len(ldflags) > 0 {
		flags = append(flags, "-ldflags="+core.Join(" ", ldflags...))
	}

	return core.Join(" ", flags...), nil
}

func buildV3TaskVars(cfg *build.Config, target build.Target) ([]string, error) {
	if cfg == nil {
		return nil, nil
	}

	var taskVars []string
	if buildFlags, err := buildV3BuildFlags(cfg, target); err != nil {
		return nil, err
	} else if buildFlags != "" {
		taskVars = append(taskVars, "BUILD_FLAGS="+buildFlags)
	}
	if len(cfg.BuildTags) > 0 {
		taskVars = append(taskVars, "EXTRA_TAGS="+core.Join(",", deduplicateStrings(append([]string{}, cfg.BuildTags...))...))
	}

	if target.OS == "windows" && cfg.WebView2 != "" {
		if err := validateWebView2Mode(cfg.WebView2); err != nil {
			return nil, err
		}
		taskVars = append(taskVars, "WEBVIEW2_MODE="+cfg.WebView2)
	}

	return taskVars, nil
}

func buildV3BuildFlags(cfg *build.Config, target build.Target) (string, error) {
	if cfg == nil {
		return "", nil
	}

	var flags []string

	tags := deduplicateStrings(append([]string{"production"}, cfg.BuildTags...))
	if len(tags) > 0 {
		flags = append(flags, "-tags", core.Join(",", tags...))
	}

	if !containsString(cfg.Flags, "-trimpath") {
		flags = append(flags, "-trimpath")
	}
	flags = append(flags, cfg.Flags...)
	if !hasFlagPrefix(cfg.Flags, "-buildvcs") {
		flags = append(flags, "-buildvcs=false")
	}

	ldflags := append([]string{}, cfg.LDFlags...)
	if target.OS == "windows" && !hasWindowsGUIFlag(ldflags) {
		ldflags = append(ldflags, "-H windowsgui")
	}
	if cfg.Version != "" && !hasVersionLDFlag(ldflags) {
		versionFlag, err := build.VersionLinkerFlag(cfg.Version)
		if err != nil {
			return "", err
		}
		ldflags = append(ldflags, versionFlag)
	}
	if len(ldflags) > 0 {
		flags = append(flags, `-ldflags="`+core.Join(" ", ldflags...)+`"`)
	}

	return core.Join(" ", flags...), nil
}

func (b *WailsBuilder) prepareV3Obfuscation(env []string) ([]string, func(), error) {
	garbleCommand, err := (&GoBuilder{}).resolveGarbleCli()
	if err != nil {
		return nil, nil, err
	}
	goCommand, err := resolveGoCli()
	if err != nil {
		return nil, nil, err
	}

	shimDir, err := ax.TempDir("core-build-wails3-go-*")
	if err != nil {
		return nil, nil, coreerr.E("WailsBuilder.prepareV3Obfuscation", "failed to create garble shim directory", err)
	}

	if err := writeGoShim(shimDir, goCommand, garbleCommand); err != nil {
		_ = ax.RemoveAll(shimDir)
		return nil, nil, err
	}

	return prependPathEnv(env, shimDir), func() {
		_ = ax.RemoveAll(shimDir)
	}, nil
}

func resolveGoCli() (string, error) {
	paths := []string{
		"/usr/local/go/bin/go",
		"/opt/homebrew/bin/go",
	}

	if goroot := core.Env("GOROOT"); goroot != "" {
		paths = append(paths, ax.Join(goroot, "bin", "go"))
	}

	command, err := ax.ResolveCommand("go", paths...)
	if err != nil {
		return "", coreerr.E("WailsBuilder.resolveGoCli", "go CLI not found. Install Go from https://go.dev/dl/", err)
	}

	return command, nil
}

func writeGoShim(dir, goCommand, garbleCommand string) error {
	switch runtime.GOOS {
	case "windows":
		content := "@echo off\r\n" +
			"if \"%1\"==\"build\" (\r\n" +
			"  \"" + garbleCommand + "\" %*\r\n" +
			"  exit /b %errorlevel%\r\n" +
			")\r\n" +
			"\"" + goCommand + "\" %*\r\n"
		for _, name := range []string{"go.bat", "go.cmd"} {
			if err := ax.WriteFile(ax.Join(dir, name), []byte(content), 0o755); err != nil {
				return coreerr.E("WailsBuilder.writeGoShim", "failed to write Windows go shim", err)
			}
		}
	default:
		content := "#!/bin/sh\nset -eu\nif [ \"${1:-}\" = \"build\" ]; then\n  exec \"" + garbleCommand + "\" \"$@\"\nfi\nexec \"" + goCommand + "\" \"$@\"\n"
		if err := ax.WriteFile(ax.Join(dir, "go"), []byte(content), 0o755); err != nil {
			return coreerr.E("WailsBuilder.writeGoShim", "failed to write go shim", err)
		}
	}

	return nil
}

func prependPathEnv(env []string, dir string) []string {
	pathSeparator := string(os.PathListSeparator)
	for i, entry := range env {
		if core.HasPrefix(entry, "PATH=") {
			current := core.TrimPrefix(entry, "PATH=")
			if current == "" {
				env[i] = "PATH=" + dir
			} else {
				env[i] = "PATH=" + dir + pathSeparator + current
			}
			return env
		}
	}

	currentPath := core.Env("PATH")
	if currentPath == "" {
		return append(env, "PATH="+dir)
	}

	return append(env, "PATH="+dir+pathSeparator+currentPath)
}

func hasFlagPrefix(flags []string, prefix string) bool {
	for _, flag := range flags {
		if core.HasPrefix(flag, prefix) {
			return true
		}
	}
	return false
}

func hasWindowsGUIFlag(ldflags []string) bool {
	for _, flag := range ldflags {
		if core.Contains(flag, "-H windowsgui") || core.Contains(flag, "-H=windowsgui") {
			return true
		}
	}
	return false
}

func deduplicateStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func validateWebView2Mode(mode string) error {
	switch mode {
	case "", "download", "embed", "browser", "error":
		return nil
	default:
		return coreerr.E("WailsBuilder.validateWebView2Mode", "webview2 must be one of download, embed, browser, or error", nil)
	}
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
