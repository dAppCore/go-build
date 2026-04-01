// Package builders provides build implementations for different project types.
package builders

import (
	"archive/zip"
	"context"
	stdio "io"
	"runtime"
	"sort"
	"strings"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
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
func (b *PythonBuilder) Detect(fs io.Medium, dir string) (bool, error) {
	return build.IsPythonProject(fs, dir), nil
}

// Build packages the Python project into a deterministic zip bundle per target.
//
// artifacts, err := b.Build(ctx, cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
func (b *PythonBuilder) Build(ctx context.Context, cfg *build.Config, targets []build.Target) ([]build.Artifact, error) {
	if cfg == nil {
		return nil, coreerr.E("PythonBuilder.Build", "config is nil", nil)
	}

	if len(targets) == 0 {
		targets = []build.Target{{OS: runtime.GOOS, Arch: runtime.GOARCH}}
	}

	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = ax.Join(cfg.ProjectDir, "dist")
	}
	if err := cfg.FS.EnsureDir(outputDir); err != nil {
		return nil, coreerr.E("PythonBuilder.Build", "failed to create output directory", err)
	}

	var artifacts []build.Artifact
	for _, target := range targets {
		platformDir := ax.Join(outputDir, core.Sprintf("%s_%s", target.OS, target.Arch))
		if err := cfg.FS.EnsureDir(platformDir); err != nil {
			return artifacts, coreerr.E("PythonBuilder.Build", "failed to create platform directory", err)
		}

		bundlePath := ax.Join(platformDir, b.bundleName(cfg)+".zip")
		if err := b.bundleProject(cfg.FS, cfg.ProjectDir, outputDir, bundlePath); err != nil {
			return artifacts, err
		}

		artifacts = append(artifacts, build.Artifact{
			Path: bundlePath,
			OS:   target.OS,
			Arch: target.Arch,
		})
	}

	return artifacts, nil
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
func (b *PythonBuilder) bundleProject(fs io.Medium, projectDir, outputDir, bundlePath string) error {
	if err := fs.EnsureDir(ax.Dir(bundlePath)); err != nil {
		return coreerr.E("PythonBuilder.bundleProject", "failed to create bundle directory", err)
	}

	file, err := fs.Create(bundlePath)
	if err != nil {
		return coreerr.E("PythonBuilder.bundleProject", "failed to create bundle file", err)
	}
	defer func() { _ = file.Close() }()

	writer := zip.NewWriter(file)
	defer func() { _ = writer.Close() }()

	return b.writeZipTree(fs, writer, projectDir, projectDir, outputDir, bundlePath)
}

// writeZipTree walks the project directory and writes files into the zip bundle.
func (b *PythonBuilder) writeZipTree(fs io.Medium, writer *zip.Writer, rootDir, currentDir, outputDir, bundlePath string) error {
	entries, err := fs.List(currentDir)
	if err != nil {
		return coreerr.E("PythonBuilder.writeZipTree", "failed to list directory", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		entryPath := ax.Join(currentDir, entry.Name())
		if b.isExcludedPath(entryPath, outputDir, bundlePath) {
			continue
		}

		if entry.IsDir() {
			if err := b.writeZipTree(fs, writer, rootDir, entryPath, outputDir, bundlePath); err != nil {
				return err
			}
			continue
		}

		relPath, err := ax.Rel(rootDir, entryPath)
		if err != nil {
			return coreerr.E("PythonBuilder.writeZipTree", "failed to relativise bundle path", err)
		}

		info, err := fs.Stat(entryPath)
		if err != nil {
			return coreerr.E("PythonBuilder.writeZipTree", "failed to stat bundle entry", err)
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return coreerr.E("PythonBuilder.writeZipTree", "failed to create zip header", err)
		}
		header.Name = strings.ReplaceAll(relPath, ax.DS(), "/")
		header.Method = zip.Deflate

		zipEntry, err := writer.CreateHeader(header)
		if err != nil {
			return coreerr.E("PythonBuilder.writeZipTree", "failed to create zip entry", err)
		}

		source, err := fs.Open(entryPath)
		if err != nil {
			return coreerr.E("PythonBuilder.writeZipTree", "failed to open bundle entry", err)
		}

		if _, err := stdio.Copy(zipEntry, source); err != nil {
			_ = source.Close()
			return coreerr.E("PythonBuilder.writeZipTree", "failed to write bundle entry", err)
		}
		if err := source.Close(); err != nil {
			return coreerr.E("PythonBuilder.writeZipTree", "failed to close bundle entry", err)
		}
	}

	return nil
}

// isExcludedPath excludes generated output from the archive.
func (b *PythonBuilder) isExcludedPath(path, outputDir, bundlePath string) bool {
	return path == bundlePath || path == outputDir || core.HasPrefix(path, outputDir+ax.DS())
}

// Ensure PythonBuilder implements the Builder interface.
var _ build.Builder = (*PythonBuilder)(nil)
