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
func (b *DocsBuilder) Detect(fs io.Medium, dir string) (bool, error) {
	return build.IsMkDocsProject(fs, dir), nil
}

// Build runs mkdocs build and packages the generated site into a zip archive.
//
// artifacts, err := b.Build(ctx, cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
func (b *DocsBuilder) Build(ctx context.Context, cfg *build.Config, targets []build.Target) ([]build.Artifact, error) {
	if cfg == nil {
		return nil, coreerr.E("DocsBuilder.Build", "config is nil", nil)
	}

	if len(targets) == 0 {
		targets = []build.Target{{OS: runtime.GOOS, Arch: runtime.GOARCH}}
	}

	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = ax.Join(cfg.ProjectDir, "dist")
	}
	if err := cfg.FS.EnsureDir(outputDir); err != nil {
		return nil, coreerr.E("DocsBuilder.Build", "failed to create output directory", err)
	}

	configPath := b.resolveMkDocsConfigPath(cfg.FS, cfg.ProjectDir)
	if configPath == "" {
		return nil, coreerr.E("DocsBuilder.Build", "mkdocs.yml or mkdocs.yaml not found", nil)
	}

	mkdocsCommand, err := b.resolveMkDocsCli()
	if err != nil {
		return nil, err
	}

	var artifacts []build.Artifact
	for _, target := range targets {
		platformDir := ax.Join(outputDir, core.Sprintf("%s_%s", target.OS, target.Arch))
		if err := cfg.FS.EnsureDir(platformDir); err != nil {
			return artifacts, coreerr.E("DocsBuilder.Build", "failed to create platform directory", err)
		}

		siteDir := ax.Join(platformDir, "site")
		if err := cfg.FS.EnsureDir(siteDir); err != nil {
			return artifacts, coreerr.E("DocsBuilder.Build", "failed to create site directory", err)
		}

		args := []string{"build", "--clean", "--site-dir", siteDir, "--config-file", configPath}
		output, err := ax.CombinedOutput(ctx, cfg.ProjectDir, cfg.Env, mkdocsCommand, args...)
		if err != nil {
			return artifacts, coreerr.E("DocsBuilder.Build", "mkdocs build failed: "+output, err)
		}

		bundlePath := ax.Join(platformDir, b.bundleName(cfg)+".zip")
		if err := b.bundleSite(cfg.FS, siteDir, bundlePath); err != nil {
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

// resolveMkDocsConfigPath returns the MkDocs config file path if present.
func (b *DocsBuilder) resolveMkDocsConfigPath(fs io.Medium, projectDir string) string {
	return build.ResolveMkDocsConfigPath(fs, projectDir)
}

// resolveMkDocsCli returns the executable path for the mkdocs CLI.
func (b *DocsBuilder) resolveMkDocsCli(paths ...string) (string, error) {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/mkdocs",
			"/opt/homebrew/bin/mkdocs",
		}
	}

	command, err := ax.ResolveCommand("mkdocs", paths...)
	if err != nil {
		return "", coreerr.E("DocsBuilder.resolveMkDocsCli", "mkdocs CLI not found. Install it with: pip install mkdocs", err)
	}

	return command, nil
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
func (b *DocsBuilder) bundleSite(fs io.Medium, siteDir, bundlePath string) error {
	if err := fs.EnsureDir(ax.Dir(bundlePath)); err != nil {
		return coreerr.E("DocsBuilder.bundleSite", "failed to create bundle directory", err)
	}

	file, err := fs.Create(bundlePath)
	if err != nil {
		return coreerr.E("DocsBuilder.bundleSite", "failed to create bundle file", err)
	}
	defer func() { _ = file.Close() }()

	writer := zip.NewWriter(file)
	defer func() { _ = writer.Close() }()

	return b.writeZipTree(fs, writer, siteDir, siteDir)
}

// writeZipTree walks a directory and writes files into the zip bundle.
func (b *DocsBuilder) writeZipTree(fs io.Medium, writer *zip.Writer, rootDir, currentDir string) error {
	entries, err := fs.List(currentDir)
	if err != nil {
		return coreerr.E("DocsBuilder.writeZipTree", "failed to list directory", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		entryPath := ax.Join(currentDir, entry.Name())

		if entry.IsDir() {
			if err := b.writeZipTree(fs, writer, rootDir, entryPath); err != nil {
				return err
			}
			continue
		}

		relPath, err := ax.Rel(rootDir, entryPath)
		if err != nil {
			return coreerr.E("DocsBuilder.writeZipTree", "failed to relativise bundle path", err)
		}

		info, err := fs.Stat(entryPath)
		if err != nil {
			return coreerr.E("DocsBuilder.writeZipTree", "failed to stat bundle entry", err)
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return coreerr.E("DocsBuilder.writeZipTree", "failed to create zip header", err)
		}
		header.Name = strings.ReplaceAll(relPath, ax.DS(), "/")
		header.Method = zip.Deflate
		header.SetModTime(deterministicZipTime)

		zipEntry, err := writer.CreateHeader(header)
		if err != nil {
			return coreerr.E("DocsBuilder.writeZipTree", "failed to create zip entry", err)
		}

		source, err := fs.Open(entryPath)
		if err != nil {
			return coreerr.E("DocsBuilder.writeZipTree", "failed to open bundle entry", err)
		}

		if _, err := stdio.Copy(zipEntry, source); err != nil {
			_ = source.Close()
			return coreerr.E("DocsBuilder.writeZipTree", "failed to write bundle entry", err)
		}
		if err := source.Close(); err != nil {
			return coreerr.E("DocsBuilder.writeZipTree", "failed to close bundle entry", err)
		}
	}

	return nil
}

// Ensure DocsBuilder implements the Builder interface.
var _ build.Builder = (*DocsBuilder)(nil)
