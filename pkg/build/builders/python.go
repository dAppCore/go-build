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

// PythonBuilder builds Python projects with pyproject.toml or requirements.txt markers.
//
// b := builders.NewPythonBuilder()
type PythonBuilder struct{}

// NewPythonBuilder creates a new PythonBuilder instance.
//
// b := builders.NewPythonBuilder()
func NewPythonBuilder() *PythonBuilder {
	return &PythonBuilder{}
}

// Name returns the builder's identifier.
//
// name := b.Name() // → "python"
func (b *PythonBuilder) Name() string {
	return "python"
}

// Detect checks if this builder can handle the project in the given directory.
//
// ok, err := b.Detect(io.Local, ".")
func (b *PythonBuilder) Detect(fs io.Medium, dir string) core.Result {
	return core.Ok(build.IsPythonProject(fs, dir))
}

// Build packages the Python project into a deterministic zip bundle per target.
//
// artifacts, err := b.Build(ctx, cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
func (b *PythonBuilder) Build(ctx context.Context, cfg *build.Config, targets []build.Target) core.Result {
	if cfg == nil {
		return core.Fail(coreerr.E("PythonBuilder.Build", "config is nil", nil))
	}
	filesystem := ensureBuildFilesystem(cfg)

	targets = defaultRuntimeTargets(targets, runtime.GOOS, runtime.GOARCH)

	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = defaultOutputDir(cfg)
	}
	created := ensureOutputDir(filesystem, outputDir, "PythonBuilder.Build")
	if !created.OK {
		return created
	}

	var artifacts []build.Artifact
	for _, target := range targets {
		platformDirResult := ensurePlatformDir(filesystem, outputDir, target, "PythonBuilder.Build")
		if !platformDirResult.OK {
			return platformDirResult
		}
		platformDir := platformDirResult.Value.(string)

		bundlePath := ax.Join(platformDir, b.bundleName(cfg)+".zip")
		bundled := b.bundleProject(filesystem, cfg.ProjectDir, outputDir, bundlePath)
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

// bundleName returns the bundle filename stem.
func (b *PythonBuilder) bundleName(cfg *build.Config) string {
	if cfg.Name != "" {
		return cfg.Name
	}
	if cfg.ProjectDir != "" {
		return ax.Base(cfg.ProjectDir)
	}
	return "python-app"
}

// bundleProject creates a zip bundle containing the Python project tree.
func (b *PythonBuilder) bundleProject(fs io.Medium, projectDir, outputDir, bundlePath string) core.Result {
	exclude := func(path string) bool {
		return b.isExcludedPath(path, outputDir, bundlePath)
	}
	return bundleZipTree(fs, projectDir, bundlePath, "PythonBuilder.bundleProject", exclude)
}

// isExcludedPath excludes generated output from the archive.
func (b *PythonBuilder) isExcludedPath(path, outputDir, bundlePath string) bool {
	return path == bundlePath || path == outputDir || core.HasPrefix(path, outputDir+ax.DS())
}

// Ensure PythonBuilder implements the Builder interface.
var _ build.Builder = (*PythonBuilder)(nil)
