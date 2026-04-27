// Package builders provides build implementations for different project types.
package builders

import (
	"context"
	"runtime"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/core"
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
func (b *PHPBuilder) Detect(fs io.Medium, dir string) (bool, error) {
	return build.IsPHPProject(fs, dir), nil
}

// Build installs dependencies and produces either composer-generated artifacts
// or a deterministic bundle when the project does not emit build outputs.
//
// artifacts, err := b.Build(ctx, cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
func (b *PHPBuilder) Build(ctx context.Context, cfg *build.Config, targets []build.Target) ([]build.Artifact, error) {
	if cfg == nil {
		return nil, coreerr.E("PHPBuilder.Build", "config is nil", nil)
	}
	filesystem := ensureBuildFilesystem(cfg)

	targets = defaultRuntimeTargets(targets, runtime.GOOS, runtime.GOARCH)

	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = defaultOutputDir(cfg)
	}
	if err := ensureOutputDir(filesystem, outputDir, "PHPBuilder.Build"); err != nil {
		return nil, err
	}

	composerCommand, err := b.resolveComposerCli()
	if err != nil {
		return nil, err
	}

	if err := b.installDependencies(ctx, cfg, composerCommand); err != nil {
		return nil, err
	}

	hasBuildScript, err := b.hasBuildScript(cfg.FS, cfg.ProjectDir)
	if err != nil {
		return nil, err
	}

	var artifacts []build.Artifact
	for _, target := range targets {
		platformDir, err := ensurePlatformDir(filesystem, outputDir, target, "PHPBuilder.Build")
		if err != nil {
			return artifacts, err
		}

		env := configuredTargetEnv(cfg, target, standardTargetValues(outputDir, platformDir, target)...)

		if hasBuildScript {
			output, err := ax.CombinedOutput(ctx, cfg.ProjectDir, env, composerCommand, "run-script", "build")
			if err != nil {
				return artifacts, coreerr.E("PHPBuilder.Build", "composer build failed: "+output, err)
			}
		}

		found := (&NodeBuilder{}).findArtifactsForTarget(filesystem, outputDir, target)
		if len(found) == 0 {
			bundlePath := ax.Join(platformDir, b.bundleName(cfg)+".zip")
			if err := b.bundleProject(filesystem, cfg.ProjectDir, outputDir, bundlePath); err != nil {
				return artifacts, err
			}

			found = append(found, build.Artifact{
				Path: bundlePath,
				OS:   target.OS,
				Arch: target.Arch,
			})
		}

		artifacts = append(artifacts, found...)
	}

	return artifacts, nil
}

// installDependencies runs composer install once before the per-target build.
func (b *PHPBuilder) installDependencies(ctx context.Context, cfg *build.Config, composerCommand string) error {
	args := []string{"install", "--no-interaction", "--no-dev", "--prefer-dist", "--optimize-autoloader"}
	output, err := ax.CombinedOutput(ctx, cfg.ProjectDir, build.BuildEnvironment(cfg), composerCommand, args...)
	if err != nil {
		return coreerr.E("PHPBuilder.installDependencies", "composer install failed: "+output, err)
	}
	return nil
}

// hasBuildScript reports whether composer.json defines a build script.
func (b *PHPBuilder) hasBuildScript(fs io.Medium, projectDir string) (bool, error) {
	content, err := fs.Read(ax.Join(projectDir, "composer.json"))
	if err != nil {
		return false, coreerr.E("PHPBuilder.hasBuildScript", "failed to read composer.json", err)
	}

	var manifest struct {
		Scripts map[string]any `json:"scripts"`
	}
	if err := ax.JSONUnmarshal([]byte(content), &manifest); err != nil {
		return false, coreerr.E("PHPBuilder.hasBuildScript", "failed to parse composer.json", err)
	}

	_, ok := manifest.Scripts["build"]
	return ok, nil
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
func (b *PHPBuilder) bundleProject(fs io.Medium, projectDir, outputDir, bundlePath string) error {
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
func (b *PHPBuilder) resolveComposerCli(paths ...string) (string, error) {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/composer",
			"/opt/homebrew/bin/composer",
			"/usr/bin/composer",
		}
	}

	command, err := ax.ResolveCommand("composer", paths...)
	if err != nil {
		return "", coreerr.E("PHPBuilder.resolveComposerCli", "composer CLI not found. Install it from https://getcomposer.org/", err)
	}

	return command, nil
}

// Ensure PHPBuilder implements the Builder interface.
var _ build.Builder = (*PHPBuilder)(nil)
