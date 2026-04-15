package build

import (
	"context"
	"runtime"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/core"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

func resolveBuiltinBuilder(projectType ProjectType) (Builder, error) {
	switch projectType {
	case ProjectTypeGo:
		return &builtinGoBuilder{}, nil
	default:
		return nil, coreerr.E(
			"build.resolveBuiltinBuilder",
			"no builder resolver registered; builtin fallback only supports go projects (requested "+string(projectType)+")",
			nil,
		)
	}
}

type builtinGoBuilder struct{}

func (b *builtinGoBuilder) Name() string { return "go" }

func (b *builtinGoBuilder) Detect(fs io.Medium, dir string) (bool, error) {
	return IsGoProject(fs, dir), nil
}

func (b *builtinGoBuilder) Build(ctx context.Context, cfg *Config, targets []Target) ([]Artifact, error) {
	if cfg == nil {
		return nil, coreerr.E("builtinGoBuilder.Build", "config is nil", nil)
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
	if err := filesystem.EnsureDir(outputDir); err != nil {
		return nil, coreerr.E("builtinGoBuilder.Build", "failed to create output directory", err)
	}

	artifacts := make([]Artifact, 0, len(targets))
	for _, target := range targets {
		artifact, err := b.buildTarget(ctx, filesystem, cfg, outputDir, target)
		if err != nil {
			return artifacts, coreerr.E("builtinGoBuilder.Build", "failed to build "+target.String(), err)
		}
		artifacts = append(artifacts, artifact)
	}

	return artifacts, nil
}

func (b *builtinGoBuilder) buildTarget(ctx context.Context, filesystem io.Medium, cfg *Config, outputDir string, target Target) (Artifact, error) {
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
	if err := filesystem.EnsureDir(platformDir); err != nil {
		return Artifact{}, coreerr.E("builtinGoBuilder.buildTarget", "failed to create platform directory", err)
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
		versionFlag, err := VersionLinkerFlag(cfg.Version)
		if err != nil {
			return Artifact{}, err
		}
		ldflags = append(ldflags, versionFlag)
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
	var err error
	if cfg.Obfuscate {
		command, err = resolveBuiltinGarbleCli()
		if err != nil {
			return Artifact{}, err
		}
	}

	output, err := ax.CombinedOutput(ctx, cfg.ProjectDir, env, command, args...)
	if err != nil {
		return Artifact{}, coreerr.E("builtinGoBuilder.buildTarget", command+" build failed: "+output, err)
	}

	return Artifact{
		Path: outputPath,
		OS:   target.OS,
		Arch: target.Arch,
	}, nil
}

func resolveBuiltinGarbleCli(paths ...string) (string, error) {
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

	command, err := ax.ResolveCommand("garble", paths...)
	if err != nil {
		return "", coreerr.E("builtinGoBuilder.resolveGarbleCli", "garble CLI not found. Install it with: go install mvdan.cc/garble@latest", err)
	}

	return command, nil
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
