package build

import (
	"context"
	"runtime"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/io"
)

func resolveBuiltinBuilder(projectType ProjectType) core.Result {
	switch projectType {
	case ProjectTypeGo:
		return core.Ok(&builtinGoBuilder{})
	default:
		return core.Fail(core.E(
			"build.resolveBuiltinBuilder",
			"no builder resolver registered; builtin fallback only supports go projects (requested "+string(projectType)+")",
			nil,
		))
	}
}

type builtinGoBuilder struct{}

func (b *builtinGoBuilder) Name() string { return "go" }

func (b *builtinGoBuilder) Detect(fs io.Medium, dir string) core.Result {
	return core.Ok(IsGoProject(fs, dir))
}

func (b *builtinGoBuilder) Build(ctx context.Context, cfg *Config, targets []Target) core.Result {
	if cfg == nil {
		return core.Fail(core.E("builtinGoBuilder.Build", "config is nil", nil))
	}

	if len(targets) == 0 {
		targets = []Target{{OS: runtime.GOOS, Arch: runtime.GOARCH}}
	}

	filesystem := cfg.FS
	if filesystem == nil {
		filesystem = io.Local
	}

	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = ax.Join(cfg.ProjectDir, "dist")
	}
	created := filesystem.EnsureDir(outputDir)
	if !created.OK {
		return core.Fail(core.E("builtinGoBuilder.Build", "failed to create output directory", core.NewError(created.Error())))
	}

	artifacts := make([]Artifact, 0, len(targets))
	for _, target := range targets {
		artifactResult := b.buildTarget(ctx, filesystem, cfg, outputDir, target)
		if !artifactResult.OK {
			return core.Fail(core.E("builtinGoBuilder.Build", "failed to build "+target.String(), core.NewError(artifactResult.Error())))
		}
		artifacts = append(artifacts, artifactResult.Value.(Artifact))
	}

	return core.Ok(artifacts)
}

func (b *builtinGoBuilder) buildTarget(ctx context.Context, filesystem io.Medium, cfg *Config, outputDir string, target Target) core.Result {
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

	if target.OS == "windows" && !core.HasSuffix(binaryName, ".exe") {
		binaryName += ".exe"
	}

	platformDir := ax.Join(outputDir, core.Sprintf("%s_%s", target.OS, target.Arch))
	created := filesystem.EnsureDir(platformDir)
	if !created.OK {
		return core.Fail(core.E("builtinGoBuilder.buildTarget", "failed to create platform directory", core.NewError(created.Error())))
	}

	outputPath := ax.Join(platformDir, binaryName)

	args := []string{"build"}
	if !builtinContainsString(cfg.Flags, "-trimpath") {
		args = append(args, "-trimpath")
	}
	if len(cfg.Flags) > 0 {
		args = append(args, cfg.Flags...)
	}
	if len(cfg.BuildTags) > 0 {
		args = append(args, "-tags", core.Join(",", cfg.BuildTags...))
	}

	ldflags := append([]string{}, cfg.LDFlags...)
	if cfg.Version != "" && !builtinHasVersionLDFlag(ldflags) {
		versionFlag := VersionLinkerFlag(cfg.Version)
		if !versionFlag.OK {
			return versionFlag
		}
		ldflags = append(ldflags, versionFlag.Value.(string))
	}
	if len(ldflags) > 0 {
		args = append(args, "-ldflags", core.Join(" ", ldflags...))
	}

	args = append(args, "-o", outputPath)

	mainPackage := cfg.Project.Main
	if mainPackage == "" {
		mainPackage = "."
	}
	args = append(args, mainPackage)

	env := append([]string{}, cfg.Env...)
	env = append(env, CacheEnvironment(&cfg.Cache)...)
	env = append(env,
		core.Sprintf("TARGET_OS=%s", target.OS),
		core.Sprintf("TARGET_ARCH=%s", target.Arch),
		core.Sprintf("OUTPUT_DIR=%s", outputDir),
		core.Sprintf("TARGET_DIR=%s", platformDir),
		core.Sprintf("GOOS=%s", target.OS),
		core.Sprintf("GOARCH=%s", target.Arch),
	)
	if binaryName != "" {
		env = append(env, core.Sprintf("NAME=%s", binaryName))
	}
	if cfg.Version != "" {
		env = append(env, core.Sprintf("VERSION=%s", cfg.Version))
	}
	if cfg.CGO {
		env = append(env, "CGO_ENABLED=1")
	} else {
		env = append(env, "CGO_ENABLED=0")
	}

	command := "go"
	if cfg.Obfuscate {
		resolved := resolveBuiltinGarbleCli()
		if !resolved.OK {
			return resolved
		}
		command = resolved.Value.(string)
	}

	output := ax.CombinedOutput(ctx, cfg.ProjectDir, env, command, args...)
	if !output.OK {
		return core.Fail(core.E("builtinGoBuilder.buildTarget", command+" build failed: "+output.Error(), core.NewError(output.Error())))
	}

	return core.Ok(Artifact{
		Path: outputPath,
		OS:   target.OS,
		Arch: target.Arch,
	})
}

func resolveBuiltinGarbleCli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/garble",
			"/opt/homebrew/bin/garble",
		}

		paths = append(paths, builtinGarbleInstallPaths()...)

		if home := core.Env("HOME"); home != "" {
			paths = append(paths, ax.Join(home, "go", "bin", "garble"))
		}
	}

	command := ax.ResolveCommand("garble", paths...)
	if !command.OK {
		return core.Fail(core.E("builtinGoBuilder.resolveGarbleCli", "garble CLI not found. Install it with: go install mvdan.cc/garble@latest", core.NewError(command.Error())))
	}

	return command
}

func builtinGarbleInstallPaths() []string {
	var paths []string

	if gobin := core.Env("GOBIN"); gobin != "" {
		paths = append(paths, ax.Join(gobin, "garble"))
	}

	if gopath := core.Env("GOPATH"); gopath != "" {
		sep := ":"
		if runtime.GOOS == "windows" {
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

func builtinHasVersionLDFlag(ldflags []string) bool {
	for _, flag := range ldflags {
		if core.Contains(flag, "main.version=") || core.Contains(flag, "main.Version=") {
			return true
		}
	}
	return false
}

func builtinContainsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
