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

// DocsBuilder builds MkDocs projects.
//
// b := builders.NewDocsBuilder()
type DocsBuilder struct{}

// NewDocsBuilder creates a new DocsBuilder instance.
//
// b := builders.NewDocsBuilder()
func NewDocsBuilder() *DocsBuilder {
	return &DocsBuilder{}
}

// Name returns the builder's identifier.
//
// name := b.Name() // → "docs"
func (b *DocsBuilder) Name() string {
	return "docs"
}

// Detect checks if this builder can handle the project in the given directory.
//
// ok, err := b.Detect(io.Local, ".")
func (b *DocsBuilder) Detect(fs io.Medium, dir string) core.Result {
	return core.Ok(build.IsDocsProject(fs, dir))
}

// Build runs mkdocs build and packages the generated site into a zip archive.
//
// artifacts, err := b.Build(ctx, cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
func (b *DocsBuilder) Build(ctx context.Context, cfg *build.Config, targets []build.Target) core.Result {
	if cfg == nil {
		return core.Fail(coreerr.E("DocsBuilder.Build", "config is nil", nil))
	}
	filesystem := ensureBuildFilesystem(cfg)

	targets = defaultRuntimeTargets(targets, runtime.GOOS, runtime.GOARCH)

	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = defaultOutputDir(cfg)
	}
	created := ensureOutputDir(filesystem, outputDir, "DocsBuilder.Build")
	if !created.OK {
		return created
	}

	configPath := b.resolveMkDocsConfigPath(cfg.FS, cfg.ProjectDir)
	if configPath == "" {
		return core.Fail(coreerr.E("DocsBuilder.Build", "mkdocs.yml or mkdocs.yaml not found", nil))
	}

	mkdocsCommandResult := b.resolveMkDocsCli()
	if !mkdocsCommandResult.OK {
		return mkdocsCommandResult
	}
	mkdocsCommand := mkdocsCommandResult.Value.(string)

	var artifacts []build.Artifact
	for _, target := range targets {
		platformDirResult := ensurePlatformDir(filesystem, outputDir, target, "DocsBuilder.Build")
		if !platformDirResult.OK {
			return platformDirResult
		}
		platformDir := platformDirResult.Value.(string)

		siteDir := ax.Join(platformDir, "site")
		createdSite := filesystem.EnsureDir(siteDir)
		if !createdSite.OK {
			return core.Fail(coreerr.E("DocsBuilder.Build", "failed to create site directory", core.NewError(createdSite.Error())))
		}

		env := configuredTargetEnv(cfg, target, standardTargetValues(outputDir, platformDir, target)...)

		args := []string{"build", "--clean", "--site-dir", siteDir, "--config-file", configPath}
		output := ax.CombinedOutput(ctx, cfg.ProjectDir, env, mkdocsCommand, args...)
		if !output.OK {
			return core.Fail(coreerr.E("DocsBuilder.Build", "mkdocs build failed: "+output.Error(), core.NewError(output.Error())))
		}

		bundlePath := ax.Join(platformDir, b.bundleName(cfg)+".zip")
		bundled := b.bundleSite(filesystem, siteDir, bundlePath)
		if !bundled.OK {
			return bundled
		}

		artifacts = append(artifacts, build.Artifact{
			Path: bundlePath,
			OS:   target.OS,
			Arch: target.Arch,
		})
	}

	return core.Ok(artifacts)
}

// resolveMkDocsConfigPath returns the MkDocs config file path if present.
func (b *DocsBuilder) resolveMkDocsConfigPath(fs io.Medium, projectDir string) string {
	return build.ResolveMkDocsConfigPath(fs, projectDir)
}

// resolveMkDocsCli returns the executable path for the mkdocs CLI.
func (b *DocsBuilder) resolveMkDocsCli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/mkdocs",
			"/opt/homebrew/bin/mkdocs",
		}
	}

	command := ax.ResolveCommand("mkdocs", paths...)
	if !command.OK {
		return core.Fail(coreerr.E("DocsBuilder.resolveMkDocsCli", "mkdocs CLI not found. Install it with: pip install mkdocs", core.NewError(command.Error())))
	}

	return command
}

// bundleName returns the bundle filename stem.
func (b *DocsBuilder) bundleName(cfg *build.Config) string {
	if cfg.Name != "" {
		return cfg.Name
	}
	if cfg.ProjectDir != "" {
		return ax.Base(cfg.ProjectDir)
	}
	return "docs-site"
}

// bundleSite creates a zip bundle containing the generated MkDocs site.
func (b *DocsBuilder) bundleSite(fs io.Medium, siteDir, bundlePath string) core.Result {
	return bundleZipTree(fs, siteDir, bundlePath, "DocsBuilder.bundleSite", nil)
}

// Ensure DocsBuilder implements the Builder interface.
var _ build.Builder = (*DocsBuilder)(nil)
