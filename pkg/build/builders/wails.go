// Package builders provides build implementations for different project types.
package builders

import (
	"context"
	stdfs "io/fs"
	"runtime"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/io"
	coreerr "dappco.re/go/log"
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
func (b *WailsBuilder) Detect(fs io.Medium, dir string) core.Result {
	return core.Ok(build.IsWailsProject(fs, dir))
}

// Build compiles the Wails project for the specified targets.
// Wails v3: delegates to Taskfile; Wails v2: uses 'wails build'.
//
// artifacts, err := b.Build(ctx, cfg, []build.Target{{OS: "darwin", Arch: "arm64"}})
func (b *WailsBuilder) Build(ctx context.Context, cfg *build.Config, targets []build.Target) core.Result {
	if cfg == nil {
		return core.Fail(coreerr.E("WailsBuilder.Build", "config is nil", nil))
	}
	filesystem := ensureBuildFilesystem(cfg)

	if len(targets) == 0 {
		return core.Fail(coreerr.E("WailsBuilder.Build", "no targets specified", nil))
	}

	if versionFlag := build.VersionLinkerFlag(cfg.Version); !versionFlag.OK {
		return versionFlag
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
		if detected := taskBuilder.Detect(filesystem, cfg.ProjectDir); detected.OK && detected.Value.(bool) {
			return taskBuilder.Build(ctx, b.buildV3Config(cfg), targets)
		}

		prebuilt := b.PreBuild(ctx, cfg)
		if !prebuilt.OK {
			return prebuilt
		}

		var artifacts []build.Artifact
		for _, target := range targets {
			artifactResult := b.buildV3Target(ctx, cfg, target)
			if !artifactResult.OK {
				return core.Fail(coreerr.E("WailsBuilder.Build", "failed to build "+target.String(), core.NewError(artifactResult.Error())))
			}
			artifacts = append(artifacts, artifactResult.Value.(build.Artifact))
		}

		return core.Ok(artifacts)
	}

	// Wails v2 strategy: Use 'wails build'
	prebuilt := b.PreBuild(ctx, cfg)
	if !prebuilt.OK {
		return prebuilt
	}

	// Ensure output directory exists
	created := filesystem.EnsureDir(cfg.OutputDir)
	if !created.OK {
		return core.Fail(coreerr.E("WailsBuilder.Build", "failed to create output directory", core.NewError(created.Error())))
	}

	// Note: Wails v2 handles frontend installation/building automatically via wails.json config

	var artifacts []build.Artifact

	for _, target := range targets {
		artifactResult := b.buildV2Target(ctx, cfg, target)
		if !artifactResult.OK {
			return core.Fail(coreerr.E("WailsBuilder.Build", "failed to build "+target.String(), core.NewError(artifactResult.Error())))
		}
		artifacts = append(artifacts, artifactResult.Value.(build.Artifact))
	}

	return core.Ok(artifacts)
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
func (b *WailsBuilder) buildV3Target(ctx context.Context, cfg *build.Config, target build.Target) core.Result {
	filesystem := ensureBuildFilesystem(cfg)

	wailsCommandResult := b.resolveWails3Cli()
	if !wailsCommandResult.OK {
		return wailsCommandResult
	}
	wailsCommand := wailsCommandResult.Value.(string)

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
	taskVarsResult := buildV3TaskVars(cfg, target)
	if !taskVarsResult.OK {
		return taskVarsResult
	}
	taskVars := taskVarsResult.Value.([]string)
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
	goflagsResult := buildV3GoFlags(cfg)
	if !goflagsResult.OK {
		return goflagsResult
	}
	if goflags := goflagsResult.Value.(string); goflags != "" {
		env = append(env, "GOFLAGS="+goflags)
	}
	if cfg.CGO {
		env = append(env, "CGO_ENABLED=1")
	}
	cleanup := func() {}
	if cfg.Obfuscate {
		obfuscationResult := b.prepareV3Obfuscation(env)
		if !obfuscationResult.OK {
			return obfuscationResult
		}
		obfuscation := obfuscationResult.Value.(obfuscationEnv)
		env = obfuscation.env
		cleanup = obfuscation.cleanup
		defer cleanup()
	}

	output := ax.CombinedOutput(ctx, cfg.ProjectDir, env, wailsCommand, args...)
	if !output.OK {
		return core.Fail(coreerr.E("WailsBuilder.buildV3Target", "wails3 "+verb+" failed: "+output.Error(), core.NewError(output.Error())))
	}

	sourcePathResult := b.findV3Artifact(filesystem, cfg.ProjectDir, binaryName, target, verb == "package")
	if !sourcePathResult.OK {
		return sourcePathResult
	}
	sourcePath := sourcePathResult.Value.(string)

	platformDir := ax.Join(cfg.OutputDir, core.Sprintf("%s_%s", target.OS, target.Arch))
	created := filesystem.EnsureDir(platformDir)
	if !created.OK {
		return core.Fail(coreerr.E("WailsBuilder.buildV3Target", "failed to create output dir", core.NewError(created.Error())))
	}

	destPath := ax.Join(platformDir, ax.Base(sourcePath))
	copied := copyBuildArtifact(filesystem, sourcePath, destPath)
	if !copied.OK {
		return core.Fail(coreerr.E("WailsBuilder.buildV3Target", "failed to copy artifact "+sourcePath, core.NewError(copied.Error())))
	}

	return core.Ok(build.Artifact{
		Path: destPath,
		OS:   target.OS,
		Arch: target.Arch,
	})
}

// PreBuild runs the frontend build step before Wails compiles the desktop app.
//
// err := b.PreBuild(ctx, cfg) // runs `deno task build` or `npm run build`
func (b *WailsBuilder) PreBuild(ctx context.Context, cfg *build.Config) core.Result {
	if cfg == nil {
		return core.Fail(coreerr.E("WailsBuilder.PreBuild", "config is nil", nil))
	}

	frontendResult := b.resolveFrontendBuild(cfg)
	if !frontendResult.OK {
		return frontendResult
	}
	frontend := frontendResult.Value.(frontendBuild)
	frontendDir := frontend.dir
	command := frontend.command
	args := frontend.args
	if command == "" {
		return core.Ok(nil)
	}

	output := ax.CombinedOutput(ctx, frontendDir, build.BuildEnvironment(cfg), command, args...)
	if !output.OK {
		return core.Fail(coreerr.E("WailsBuilder.PreBuild", command+" build failed: "+output.Error(), core.NewError(output.Error())))
	}

	return core.Ok(nil)
}

// isWailsV3 checks if the project uses Wails v3 by inspecting go.mod.
func (b *WailsBuilder) isWailsV3(fs io.Medium, dir string) bool {
	goModPath := ax.Join(dir, "go.mod")
	content := fs.Read(goModPath)
	if !content.OK {
		return false
	}
	return core.Contains(content.Value.(string), "github.com/wailsapp/wails/v3")
}

// resolveFrontendBuild selects the frontend directory and build command.
//
// dir, command, args, err := b.resolveFrontendBuild(cfg)
type frontendBuild struct {
	dir     string
	command string
	args    []string
}

func (b *WailsBuilder) resolveFrontendBuild(cfg *build.Config) core.Result {
	if cfg == nil {
		return core.Fail(coreerr.E("WailsBuilder.resolveFrontendBuild", "config is nil", nil))
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
			return core.Ok(frontendBuild{})
		}
	}

	if b.hasDenoConfig(fs, frontendDir) || build.DenoRequested(cfg.DenoBuild) {
		resolved := resolveDenoBuildCommand(cfg, b.resolveDenoCli)
		if !resolved.OK {
			return resolved
		}
		spec := resolved.Value.(commandSpec)
		return core.Ok(frontendBuild{dir: frontendDir, command: spec.command, args: spec.args})
	}

	if build.NpmRequested(cfg.NpmBuild) {
		resolved := resolveNpmBuildCommand(cfg, b.resolveNpmCli)
		if !resolved.OK {
			return resolved
		}
		spec := resolved.Value.(commandSpec)
		return core.Ok(frontendBuild{dir: frontendDir, command: spec.command, args: spec.args})
	}

	if fs.IsFile(ax.Join(frontendDir, "package.json")) {
		packageManager := detectPackageManager(fs, frontendDir)
		return b.resolvePackageManagerBuild(frontendDir, packageManager)
	}

	return core.Ok(frontendBuild{})
}

// resolvePackageManagerBuild returns the frontend build command for a detected package manager.
func (b *WailsBuilder) resolvePackageManagerBuild(frontendDir, packageManager string) core.Result {
	switch packageManager {
	case "bun":
		command := b.resolveBunCli()
		if !command.OK {
			return command
		}
		return core.Ok(frontendBuild{dir: frontendDir, command: command.Value.(string), args: []string{"run", "build"}})
	case "pnpm":
		command := b.resolvePnpmCli()
		if !command.OK {
			return command
		}
		return core.Ok(frontendBuild{dir: frontendDir, command: command.Value.(string), args: []string{"run", "build"}})
	case "yarn":
		command := b.resolveYarnCli()
		if !command.OK {
			return command
		}
		return core.Ok(frontendBuild{dir: frontendDir, command: command.Value.(string), args: []string{"build"}})
	default:
		command := b.resolveNpmCli()
		if !command.OK {
			return command
		}
		return core.Ok(frontendBuild{dir: frontendDir, command: command.Value.(string), args: []string{"run", "build"}})
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
func (b *WailsBuilder) buildV2Target(ctx context.Context, cfg *build.Config, target build.Target) core.Result {
	filesystem := ensureBuildFilesystem(cfg)

	if cfg.WebView2 != "" && target.OS == "windows" {
		valid := validateWebView2Mode(cfg.WebView2)
		if !valid.OK {
			return valid
		}
	}

	wailsCommandResult := b.resolveWailsCli()
	if !wailsCommandResult.OK {
		return wailsCommandResult
	}
	wailsCommand := wailsCommandResult.Value.(string)

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
		versionFlag := build.VersionLinkerFlag(cfg.Version)
		if !versionFlag.OK {
			return versionFlag
		}
		ldflags = append(ldflags, versionFlag.Value.(string))
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
	output := ax.CombinedOutput(ctx, cfg.ProjectDir, build.BuildEnvironment(cfg), wailsCommand, args...)
	if !output.OK {
		return core.Fail(coreerr.E("WailsBuilder.buildV2Target", "wails build failed: "+output.Error(), core.NewError(output.Error())))
	}

	// Wails v2 typically outputs to build/bin
	// We need to move/copy it to our desired output dir

	// Construct the source path where Wails v2 puts the binary
	wailsOutputDir := ax.Join(cfg.ProjectDir, "build", "bin")

	// Find the artifact in Wails output dir
	sourcePathResult := b.findArtifact(filesystem, wailsOutputDir, binaryName, target)
	if !sourcePathResult.OK {
		return core.Fail(coreerr.E("WailsBuilder.buildV2Target", "failed to find Wails v2 build artifact", core.NewError(sourcePathResult.Error())))
	}
	sourcePath := sourcePathResult.Value.(string)

	// Move/Copy to our output dir
	// Create platform specific dir in our output
	platformDir := ax.Join(cfg.OutputDir, core.Sprintf("%s_%s", target.OS, target.Arch))
	created := filesystem.EnsureDir(platformDir)
	if !created.OK {
		return core.Fail(coreerr.E("WailsBuilder.buildV2Target", "failed to create output dir", core.NewError(created.Error())))
	}

	destPath := ax.Join(platformDir, ax.Base(sourcePath))

	// Copy the selected artifact, preserving directory bundles such as .app packages.
	copied := copyBuildArtifact(filesystem, sourcePath, destPath)
	if !copied.OK {
		return core.Fail(coreerr.E("WailsBuilder.buildV2Target", "failed to copy artifact "+sourcePath, core.NewError(copied.Error())))
	}

	return core.Ok(build.Artifact{
		Path: destPath,
		OS:   target.OS,
		Arch: target.Arch,
	})
}

// findArtifact locates the built artifact based on the target platform.
func (b *WailsBuilder) findArtifact(fs io.Medium, platformDir, binaryName string, target build.Target) core.Result {
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
			return core.Ok(candidate)
		}
	}

	// If no specific candidate found, try to find any executable or package in the directory
	entriesResult := fs.List(platformDir)
	if !entriesResult.OK {
		return core.Fail(coreerr.E("WailsBuilder.findArtifact", "failed to read platform directory", core.NewError(entriesResult.Error())))
	}
	entries := entriesResult.Value.([]stdfs.DirEntry)

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
				return core.Ok(path)
			}
		} else if info.Mode()&0111 != 0 || entry.IsDir() {
			// Executable file or directory (.app bundle)
			return core.Ok(path)
		}
	}

	return core.Fail(coreerr.E("WailsBuilder.findArtifact", "no artifact found in "+platformDir, nil))
}

func (b *WailsBuilder) findV3Artifact(fs io.Medium, projectDir, binaryName string, target build.Target, packaged bool) core.Result {
	if packaged && target.OS == "windows" {
		for _, candidate := range []string{
			ax.Join(projectDir, "build", "windows", "nsis", binaryName+"-installer.exe"),
			ax.Join(projectDir, "bin", binaryName+"-installer.exe"),
		} {
			if fs.Exists(candidate) {
				return core.Ok(candidate)
			}
		}
	}

	for _, platformDir := range []string{
		ax.Join(projectDir, "build", "bin"),
		ax.Join(projectDir, "bin"),
	} {
		path := b.findArtifact(fs, platformDir, binaryName, target)
		if path.OK {
			return path
		}
	}

	return core.Fail(coreerr.E("WailsBuilder.findV3Artifact", "no artifact found for "+target.String(), nil))
}

// copyBuildArtifact copies a file or directory artifact into the build output tree.
//
// err := copyBuildArtifact(io.Local, "/tmp/source.app", "/tmp/dist/source.app")
func copyBuildArtifact(fs io.Medium, sourcePath, destPath string) core.Result {
	if fs.IsDir(sourcePath) {
		created := fs.EnsureDir(destPath)
		if !created.OK {
			return created
		}

		entriesResult := fs.List(sourcePath)
		if !entriesResult.OK {
			return entriesResult
		}
		entries := entriesResult.Value.([]stdfs.DirEntry)

		for _, entry := range entries {
			childSource := ax.Join(sourcePath, entry.Name())
			childDest := ax.Join(destPath, entry.Name())
			copied := copyBuildArtifact(fs, childSource, childDest)
			if !copied.OK {
				return copied
			}
		}

		return core.Ok(nil)
	}

	infoResult := fs.Stat(sourcePath)
	if !infoResult.OK {
		return infoResult
	}
	info := infoResult.Value.(stdfs.FileInfo)

	content := fs.Read(sourcePath)
	if !content.OK {
		return content
	}

	written := fs.WriteMode(destPath, content.Value.(string), info.Mode().Perm())
	if !written.OK {
		return written
	}

	return core.Ok(nil)
}

// resolveWailsCli returns the executable path for the wails CLI.
func (b *WailsBuilder) resolveWailsCli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/wails",
			"/opt/homebrew/bin/wails",
		}

		if home := core.Env("HOME"); home != "" {
			paths = append(paths, ax.Join(home, "go", "bin", "wails"))
		}
	}

	command := ax.ResolveCommand("wails", paths...)
	if !command.OK {
		return core.Fail(coreerr.E("WailsBuilder.resolveWailsCli", "wails CLI not found. Install it with: go install github.com/wailsapp/wails/v2/cmd/wails@latest", core.NewError(command.Error())))
	}

	return command
}

func (b *WailsBuilder) resolveWails3Cli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/wails3",
			"/opt/homebrew/bin/wails3",
		}

		if home := core.Env("HOME"); home != "" {
			paths = append(paths, ax.Join(home, "go", "bin", "wails3"))
		}
	}

	command := ax.ResolveCommand("wails3", paths...)
	if !command.OK {
		return core.Fail(coreerr.E("WailsBuilder.resolveWails3Cli", "wails3 CLI not found. Install Wails v3 or expose it on PATH.", core.NewError(command.Error())))
	}

	return command
}

func buildV3GoFlags(cfg *build.Config) core.Result {
	if cfg == nil {
		return core.Ok("")
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
		versionFlag := build.VersionLinkerFlag(cfg.Version)
		if !versionFlag.OK {
			return versionFlag
		}
		ldflags = append(ldflags, versionFlag.Value.(string))
	}
	if len(ldflags) > 0 {
		flags = append(flags, "-ldflags="+core.Join(" ", ldflags...))
	}

	return core.Ok(core.Join(" ", flags...))
}

func buildV3TaskVars(cfg *build.Config, target build.Target) core.Result {
	if cfg == nil {
		return core.Ok([]string(nil))
	}

	var taskVars []string
	buildFlagsResult := buildV3BuildFlags(cfg, target)
	if !buildFlagsResult.OK {
		return buildFlagsResult
	}
	if buildFlags := buildFlagsResult.Value.(string); buildFlags != "" {
		taskVars = append(taskVars, "BUILD_FLAGS="+buildFlags)
	}
	if len(cfg.BuildTags) > 0 {
		taskVars = append(taskVars, "EXTRA_TAGS="+core.Join(",", deduplicateStrings(append([]string{}, cfg.BuildTags...))...))
	}

	if target.OS == "windows" && cfg.WebView2 != "" {
		valid := validateWebView2Mode(cfg.WebView2)
		if !valid.OK {
			return valid
		}
		taskVars = append(taskVars, "WEBVIEW2_MODE="+cfg.WebView2)
	}

	return core.Ok(taskVars)
}

func buildV3BuildFlags(cfg *build.Config, target build.Target) core.Result {
	if cfg == nil {
		return core.Ok("")
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
		versionFlag := build.VersionLinkerFlag(cfg.Version)
		if !versionFlag.OK {
			return versionFlag
		}
		ldflags = append(ldflags, versionFlag.Value.(string))
	}
	if len(ldflags) > 0 {
		flags = append(flags, `-ldflags="`+core.Join(" ", ldflags...)+`"`)
	}

	return core.Ok(core.Join(" ", flags...))
}

type obfuscationEnv struct {
	env     []string
	cleanup func()
}

func (b *WailsBuilder) prepareV3Obfuscation(env []string) core.Result {
	garbleCommandResult := (&GoBuilder{}).resolveGarbleCli()
	if !garbleCommandResult.OK {
		return garbleCommandResult
	}
	garbleCommand := garbleCommandResult.Value.(string)
	goCommandResult := resolveGoCli()
	if !goCommandResult.OK {
		return goCommandResult
	}
	goCommand := goCommandResult.Value.(string)

	shimDirResult := ax.TempDir("core-build-wails3-go-*")
	if !shimDirResult.OK {
		return core.Fail(coreerr.E("WailsBuilder.prepareV3Obfuscation", "failed to create garble shim directory", core.NewError(shimDirResult.Error())))
	}
	shimDir := shimDirResult.Value.(string)

	written := writeGoShim(shimDir, goCommand, garbleCommand)
	if !written.OK {
		cleaned := ax.RemoveAll(shimDir)
		if !cleaned.OK {
			return core.Fail(coreerr.E("WailsBuilder.prepareV3Obfuscation", "failed to clean up garble shim directory", core.NewError(cleaned.Error())))
		}
		return written
	}

	return core.Ok(obfuscationEnv{
		env: prependPathEnv(env, shimDir),
		cleanup: func() {
			ax.RemoveAll(shimDir)
		},
	})
}

func resolveGoCli() core.Result {
	paths := []string{
		"/usr/local/go/bin/go",
		"/opt/homebrew/bin/go",
	}

	if goroot := core.Env("GOROOT"); goroot != "" {
		paths = append(paths, ax.Join(goroot, "bin", "go"))
	}

	command := ax.ResolveCommand("go", paths...)
	if !command.OK {
		return core.Fail(coreerr.E("WailsBuilder.resolveGoCli", "go CLI not found. Install Go from https://go.dev/dl/", core.NewError(command.Error())))
	}

	return command
}

func writeGoShim(dir, goCommand, garbleCommand string) core.Result {
	switch runtime.GOOS {
	case "windows":
		content := "@echo off\r\n" +
			"if \"%1\"==\"build\" (\r\n" +
			"  \"" + garbleCommand + "\" %*\r\n" +
			"  exit /b %errorlevel%\r\n" +
			")\r\n" +
			"\"" + goCommand + "\" %*\r\n"
		for _, name := range []string{"go.bat", "go.cmd"} {
			written := ax.WriteFile(ax.Join(dir, name), []byte(content), 0o755)
			if !written.OK {
				return core.Fail(coreerr.E("WailsBuilder.writeGoShim", "failed to write Windows go shim", core.NewError(written.Error())))
			}
		}
	default:
		content := "#!/bin/sh\nset -eu\nif [ \"${1:-}\" = \"build\" ]; then\n  exec \"" + garbleCommand + "\" \"$@\"\nfi\nexec \"" + goCommand + "\" \"$@\"\n"
		written := ax.WriteFile(ax.Join(dir, "go"), []byte(content), 0o755)
		if !written.OK {
			return core.Fail(coreerr.E("WailsBuilder.writeGoShim", "failed to write go shim", core.NewError(written.Error())))
		}
	}

	return core.Ok(nil)
}

func prependPathEnv(env []string, dir string) []string {
	pathSeparator := string(core.PathListSeparator)
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

func validateWebView2Mode(mode string) core.Result {
	switch mode {
	case "", "download", "embed", "browser", "error":
		return core.Ok(nil)
	default:
		return core.Fail(coreerr.E("WailsBuilder.validateWebView2Mode", "webview2 must be one of download, embed, browser, or error", nil))
	}
}

// resolveDenoCli returns the executable path for the deno CLI.
func (b *WailsBuilder) resolveDenoCli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/deno",
			"/opt/homebrew/bin/deno",
		}
	}

	command := ax.ResolveCommand("deno", paths...)
	if !command.OK {
		return core.Fail(coreerr.E("WailsBuilder.resolveDenoCli", "deno CLI not found. Install it from https://deno.com/runtime", core.NewError(command.Error())))
	}

	return command
}

// resolveNpmCli returns the executable path for the npm CLI.
func (b *WailsBuilder) resolveNpmCli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/npm",
			"/opt/homebrew/bin/npm",
		}
	}

	command := ax.ResolveCommand("npm", paths...)
	if !command.OK {
		return core.Fail(coreerr.E("WailsBuilder.resolveNpmCli", "npm CLI not found. Install Node.js from https://nodejs.org/", core.NewError(command.Error())))
	}

	return command
}

// resolveBunCli returns the executable path for the bun CLI.
func (b *WailsBuilder) resolveBunCli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/bun",
			"/opt/homebrew/bin/bun",
		}
	}

	command := ax.ResolveCommand("bun", paths...)
	if !command.OK {
		return core.Fail(coreerr.E("WailsBuilder.resolveBunCli", "bun CLI not found. Install it from https://bun.sh/", core.NewError(command.Error())))
	}

	return command
}

// resolvePnpmCli returns the executable path for the pnpm CLI.
func (b *WailsBuilder) resolvePnpmCli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/pnpm",
			"/opt/homebrew/bin/pnpm",
		}
	}

	command := ax.ResolveCommand("pnpm", paths...)
	if !command.OK {
		return core.Fail(coreerr.E("WailsBuilder.resolvePnpmCli", "pnpm CLI not found. Install it from https://pnpm.io/installation", core.NewError(command.Error())))
	}

	return command
}

// resolveYarnCli returns the executable path for the yarn CLI.
func (b *WailsBuilder) resolveYarnCli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/yarn",
			"/opt/homebrew/bin/yarn",
		}
	}

	command := ax.ResolveCommand("yarn", paths...)
	if !command.OK {
		return core.Fail(coreerr.E("WailsBuilder.resolveYarnCli", "yarn CLI not found. Install it from https://yarnpkg.com/getting-started/install", core.NewError(command.Error())))
	}

	return command
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
