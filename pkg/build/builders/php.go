// Package builders provides build implementations for different project types.
package builders

import (
	"context"
	"runtime"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/io"
	coreerr "dappco.re/go/log"
)

// PHPBuilder builds PHP projects with composer.json manifests.
//
// b := builders.NewPHPBuilder()
type PHPBuilder struct{}

// NewPHPBuilder creates a new PHP builder instance.
//
// b := builders.NewPHPBuilder()
func NewPHPBuilder() *PHPBuilder {
	return &PHPBuilder{}
}

// Name returns the builder's identifier.
//
// name := b.Name() // → "php"
func (b *PHPBuilder) Name() string {
	return "php"
}

// Detect checks if this builder can handle the project in the given directory.
//
// ok, err := b.Detect(io.Local, ".")
func (b *PHPBuilder) Detect(fs io.Medium, dir string) core.Result {
	return core.Ok(build.IsPHPProject(fs, dir))
}

// Build installs dependencies and produces either composer-generated artifacts
// or a deterministic bundle when the project does not emit build outputs.
//
// artifacts, err := b.Build(ctx, cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
func (b *PHPBuilder) Build(ctx context.Context, cfg *build.Config, targets []build.Target) core.Result {
	if cfg == nil {
		return core.Fail(coreerr.E("PHPBuilder.Build", "config is nil", nil))
	}
	filesystem := ensureBuildFilesystem(cfg)

	targets = defaultRuntimeTargets(targets, runtime.GOOS, runtime.GOARCH)

	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = defaultOutputDir(cfg)
	}
	created := ensureOutputDir(filesystem, outputDir, "PHPBuilder.Build")
	if !created.OK {
		return created
	}

	composerCommandResult := b.resolveComposerCli()
	if !composerCommandResult.OK {
		return composerCommandResult
	}
	composerCommand := composerCommandResult.Value.(string)

	installed := b.installDependencies(ctx, cfg, composerCommand)
	if !installed.OK {
		return installed
	}

	hasBuildScriptResult := b.hasBuildScript(cfg.FS, cfg.ProjectDir)
	if !hasBuildScriptResult.OK {
		return hasBuildScriptResult
	}
	hasBuildScript := hasBuildScriptResult.Value.(bool)

	var artifacts []build.Artifact
	for _, target := range targets {
		platformDirResult := ensurePlatformDir(filesystem, outputDir, target, "PHPBuilder.Build")
		if !platformDirResult.OK {
			return platformDirResult
		}
		platformDir := platformDirResult.Value.(string)

		env := configuredTargetEnv(cfg, target, standardTargetValues(outputDir, platformDir, target)...)

		if hasBuildScript {
			output := ax.CombinedOutput(ctx, cfg.ProjectDir, env, composerCommand, "run-script", "build")
			if !output.OK {
				return core.Fail(coreerr.E("PHPBuilder.Build", "composer build failed: "+output.Error(), core.NewError(output.Error())))
			}
		}

		found := (&NodeBuilder{}).findArtifactsForTarget(filesystem, outputDir, target)
		if len(found) == 0 {
			bundlePath := ax.Join(platformDir, b.bundleName(cfg)+".zip")
			bundled := b.bundleProject(filesystem, cfg.ProjectDir, outputDir, bundlePath)
			if !bundled.OK {
				return bundled
			}

			found = append(found, build.Artifact{
				Path: bundlePath,
				OS:   target.OS,
				Arch: target.Arch,
			})
		}

		artifacts = append(artifacts, found...)
	}

	return core.Ok(artifacts)
}

// installDependencies runs composer install once before the per-target build.
func (b *PHPBuilder) installDependencies(ctx context.Context, cfg *build.Config, composerCommand string) core.Result {
	args := []string{"install", "--no-interaction", "--no-dev", "--prefer-dist", "--optimize-autoloader"}
	output := ax.CombinedOutput(ctx, cfg.ProjectDir, build.BuildEnvironment(cfg), composerCommand, args...)
	if !output.OK {
		return core.Fail(coreerr.E("PHPBuilder.installDependencies", "composer install failed: "+output.Error(), core.NewError(output.Error())))
	}
	return core.Ok(nil)
}

// hasBuildScript reports whether composer.json defines a build script.
func (b *PHPBuilder) hasBuildScript(fs io.Medium, projectDir string) core.Result {
	content := fs.Read(ax.Join(projectDir, "composer.json"))
	if !content.OK {
		return core.Fail(coreerr.E("PHPBuilder.hasBuildScript", "failed to read composer.json", core.NewError(content.Error())))
	}

	var manifest struct {
		Scripts map[string]any `json:"scripts"`
	}
	decoded := ax.JSONUnmarshal([]byte(content.Value.(string)), &manifest)
	if !decoded.OK {
		return core.Fail(coreerr.E("PHPBuilder.hasBuildScript", "failed to parse composer.json", core.NewError(decoded.Error())))
	}

	_, ok := manifest.Scripts["build"]
	return core.Ok(ok)
}

// bundleName returns the bundle filename stem.
func (b *PHPBuilder) bundleName(cfg *build.Config) string {
	if cfg.Name != "" {
		return cfg.Name
	}
	if cfg.ProjectDir != "" {
		return ax.Base(cfg.ProjectDir)
	}
	return "php-app"
}

// bundleProject creates a zip bundle containing the project tree.
func (b *PHPBuilder) bundleProject(fs io.Medium, projectDir, outputDir, bundlePath string) core.Result {
	exclude := func(path string) bool {
		return b.isExcludedPath(path, outputDir, bundlePath)
	}
	return bundleZipTree(fs, projectDir, bundlePath, "PHPBuilder.bundleProject", exclude)
}

// isExcludedPath reports whether a path should be omitted from the bundle.
func (b *PHPBuilder) isExcludedPath(path, outputDir, bundlePath string) bool {
	cleanPath := ax.Clean(path)
	cleanOutputDir := ax.Clean(outputDir)
	cleanBundlePath := ax.Clean(bundlePath)

	if cleanPath == cleanOutputDir || core.HasPrefix(cleanPath, cleanOutputDir+ax.DS()) {
		return true
	}
	if cleanPath == cleanBundlePath {
		return true
	}

	base := ax.Base(cleanPath)
	switch base {
	case ".git", ".core":
		return true
	default:
		return false
	}
}

// resolveComposerCli returns the executable path for the composer CLI.
func (b *PHPBuilder) resolveComposerCli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/composer",
			"/opt/homebrew/bin/composer",
			"/usr/bin/composer",
		}
	}

	command := ax.ResolveCommand("composer", paths...)
	if !command.OK {
		return core.Fail(coreerr.E("PHPBuilder.resolveComposerCli", "composer CLI not found. Install it from https://getcomposer.org/", core.NewError(command.Error())))
	}

	return command
}

// Ensure PHPBuilder implements the Builder interface.
var _ build.Builder = (*PHPBuilder)(nil)
