// Package builders provides build implementations for different project types.
package builders

import (
	"context"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	storage "dappco.re/go/build/pkg/storage"
)

// GoBuilder implements the Builder interface for Go projects.
//
// b := builders.NewGoBuilder()
type GoBuilder struct{}

// NewGoBuilder creates a new GoBuilder instance.
//
// b := builders.NewGoBuilder()
func NewGoBuilder() *GoBuilder {
	return &GoBuilder{}
}

// Name returns the builder's identifier.
//
// name := b.Name() // → "go"
func (b *GoBuilder) Name() string {
	return "go"
}

// Detect checks if this builder can handle the project in the given directory.
// Uses IsGoProject from the build package which checks for go.mod, go.work, or wails.json.
//
// result := b.Detect(storage.Local, ".")
func (b *GoBuilder) Detect(fs storage.Medium, dir string) core.Result {
	return core.Ok(build.IsGoProject(fs, dir))
}

// Build compiles the Go project for the specified targets.
// If targets is empty, it falls back to the current host platform.
// It sets GOOS, GOARCH, and CGO_ENABLED, applies config-defined build flags
// and ldflags, and uses garble when obfuscation is enabled.
//
// result := b.Build(ctx, cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
func (b *GoBuilder) Build(ctx context.Context, cfg *build.Config, targets []build.Target) core.Result {
	if cfg == nil {
		return core.Fail(core.E("GoBuilder.Build", "config is nil", nil))
	}
	ensureBuildFilesystem(cfg)
	artifactFilesystem := build.ResolveOutputMedium(cfg)

	targets = defaultHostTargets(targets)

	outputDir := cfg.OutputDir
	if outputDir == "" && build.MediumIsLocal(artifactFilesystem) {
		outputDir = defaultOutputDir(cfg)
	}

	created := ensureOutputDir(artifactFilesystem, outputDir, "GoBuilder.Build")
	if !created.OK {
		return created
	}

	var artifacts []build.Artifact

	for _, target := range targets {
		artifactResult := b.buildTarget(ctx, cfg, artifactFilesystem, outputDir, target)
		if !artifactResult.OK {
			return core.Fail(core.E("GoBuilder.Build", "failed to build "+target.String(), core.NewError(artifactResult.Error())))
		}
		artifacts = append(artifacts, artifactResult.Value.(build.Artifact))
	}

	return core.Ok(artifacts)
}

// buildTarget compiles for a single target platform.
func (b *GoBuilder) buildTarget(ctx context.Context, cfg *build.Config, artifactFilesystem storage.Medium, outputDir string, target build.Target) core.Result {
	// Determine output binary name
	binaryName := cfg.Name
	if binaryName == "" {
		binaryName = cfg.Project.Binary
	}
	if binaryName == "" {
		binaryName = cfg.Project.Name
	}
	if binaryName == "" {
		binaryName = ax.Base(cfg.ProjectDir)
	}

	// Add .exe extension for Windows
	if target.OS == "windows" && !core.HasSuffix(binaryName, ".exe") {
		binaryName += ".exe"
	}

	platformID := platformName(target)
	platformDirResult := ensurePlatformDir(artifactFilesystem, outputDir, target, "GoBuilder.buildTarget")
	if !platformDirResult.OK {
		return platformDirResult
	}
	platformDir := platformDirResult.Value.(string)

	outputPath := ax.Join(platformDir, binaryName)
	commandOutputPath := outputPath
	stageResult := prepareStagedOutput(outputDir, artifactFilesystem, "core-build-go-*", "GoBuilder.buildTarget")
	if !stageResult.OK {
		return stageResult
	}
	stage := stageResult.Value.(stagedOutput)
	defer stage.cleanup()
	if !build.MediumIsLocal(artifactFilesystem) {
		stagePlatformDir := ax.Join(stage.commandOutputDir, platformID)
		created := stage.commandFS.EnsureDir(stagePlatformDir)
		if !created.OK {
			return core.Fail(core.E("GoBuilder.buildTarget", "failed to create local platform staging directory", core.NewError(created.Error())))
		}
		commandOutputPath = ax.Join(stagePlatformDir, binaryName)
	}

	// Build the go/garble arguments.
	args := []string{"build"}
	if !containsString(cfg.Flags, "-trimpath") {
		args = append(args, "-trimpath")
	}
	if len(cfg.Flags) > 0 {
		args = append(args, cfg.Flags...)
	}

	if len(cfg.BuildTags) > 0 {
		args = append(args, "-tags", core.Join(",", cfg.BuildTags...))
	}

	// Add ldflags if specified, and inject the build version when needed.
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

	// Add output path
	args = append(args, "-o", commandOutputPath)

	// Build the configured main package path, defaulting to the project root.
	mainPackage := cfg.Project.Main
	if mainPackage == "" {
		mainPackage = "."
	}
	args = append(args, mainPackage)

	// Set up environment.
	env := appendConfiguredEnv(cfg, standardTargetValues(outputDir, platformDir, target)...)
	if binaryName != "" {
		env = append(env, core.Sprintf("NAME=%s", binaryName))
	}
	if cfg.Version != "" {
		env = append(env, core.Sprintf("VERSION=%s", cfg.Version))
	}
	env = append(env, cgoEnvValue(cfg.CGO))

	command := "go"
	if cfg.Obfuscate {
		resolved := b.resolveGarbleCli()
		if !resolved.OK {
			return resolved
		}
		command = resolved.Value.(string)
	}

	// Capture output for error messages
	output := ax.CombinedOutput(ctx, cfg.ProjectDir, env, command, args...)
	if !output.OK {
		return core.Fail(core.E("GoBuilder.buildTarget", command+" build failed: "+output.Error(), core.NewError(output.Error())))
	}

	if commandOutputPath != outputPath {
		copied := build.CopyMediumPath(storage.Local, commandOutputPath, artifactFilesystem, outputPath)
		if !copied.OK {
			return copied
		}
	}

	return core.Ok(build.Artifact{
		Path: outputPath,
		OS:   target.OS,
		Arch: target.Arch,
	})
}

// resolveGarbleCli returns the executable path for the garble CLI.
//
// command, err := b.resolveGarbleCli()
func (b *GoBuilder) resolveGarbleCli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/garble",
			"/opt/homebrew/bin/garble",
		}

		paths = append(paths, garbleInstallPaths()...)

		if home := core.Env("HOME"); home != "" {
			paths = append(paths, ax.Join(home, "go", "bin", "garble"))
		}
	}

	command := ax.ResolveCommand("garble", paths...)
	if !command.OK {
		return core.Fail(core.E("GoBuilder.resolveGarbleCli", "garble CLI not found. Install it with: go install mvdan.cc/garble@latest", core.NewError(command.Error())))
	}

	return command
}

// garbleInstallPaths returns the standard Go install locations for garble.
func garbleInstallPaths() []string {
	var paths []string

	if gobin := core.Env("GOBIN"); gobin != "" {
		paths = append(paths, ax.Join(gobin, "garble"))
	}

	if gopath := core.Env("GOPATH"); gopath != "" {
		sep := ":"
		if core.Env("GOOS") == "windows" {
			sep = ";"
		}
		for _, root := range core.Split(gopath, sep) {
			root = core.Trim(root)
			if root == "" {
				continue
			}
			paths = append(paths, ax.Join(root, "bin", "garble"))
		}
	}

	return paths
}

// hasVersionLDFlag reports whether a version linker flag is already present.
func hasVersionLDFlag(ldflags []string) bool {
	for _, flag := range ldflags {
		if core.Contains(flag, "main.version=") || core.Contains(flag, "main.Version=") {
			return true
		}
	}
	return false
}

// containsString reports whether a slice contains the given string.
func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

// Ensure GoBuilder implements the Builder interface.
var _ build.Builder = (*GoBuilder)(nil)
