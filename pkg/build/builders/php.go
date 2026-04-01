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

	if len(targets) == 0 {
		targets = []build.Target{{OS: runtime.GOOS, Arch: runtime.GOARCH}}
	}

	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = ax.Join(cfg.ProjectDir, "dist")
	}
	if err := cfg.FS.EnsureDir(outputDir); err != nil {
		return nil, coreerr.E("PHPBuilder.Build", "failed to create output directory", err)
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
		platformDir := ax.Join(outputDir, core.Sprintf("%s_%s", target.OS, target.Arch))
		if err := cfg.FS.EnsureDir(platformDir); err != nil {
			return artifacts, coreerr.E("PHPBuilder.Build", "failed to create platform directory", err)
		}

		env := appendConfiguredEnv(cfg.Env,
			core.Sprintf("GOOS=%s", target.OS),
			core.Sprintf("GOARCH=%s", target.Arch),
			core.Sprintf("TARGET_OS=%s", target.OS),
			core.Sprintf("TARGET_ARCH=%s", target.Arch),
			core.Sprintf("OUTPUT_DIR=%s", outputDir),
			core.Sprintf("TARGET_DIR=%s", platformDir),
		)
		if cfg.Name != "" {
			env = append(env, core.Sprintf("NAME=%s", cfg.Name))
		}
		if cfg.Version != "" {
			env = append(env, core.Sprintf("VERSION=%s", cfg.Version))
		}

		if hasBuildScript {
			output, err := ax.CombinedOutput(ctx, cfg.ProjectDir, env, composerCommand, "run-script", "build")
			if err != nil {
				return artifacts, coreerr.E("PHPBuilder.Build", "composer build failed: "+output, err)
			}
		}

		found := (&NodeBuilder{}).findArtifactsForTarget(cfg.FS, outputDir, target)
		if len(found) == 0 {
			bundlePath := ax.Join(platformDir, b.bundleName(cfg)+".zip")
			if err := b.bundleProject(cfg.FS, cfg.ProjectDir, outputDir, bundlePath); err != nil {
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
	output, err := ax.CombinedOutput(ctx, cfg.ProjectDir, cfg.Env, composerCommand, args...)
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
	if err := fs.EnsureDir(ax.Dir(bundlePath)); err != nil {
		return coreerr.E("PHPBuilder.bundleProject", "failed to create bundle directory", err)
	}

	file, err := fs.Create(bundlePath)
	if err != nil {
		return coreerr.E("PHPBuilder.bundleProject", "failed to create bundle file", err)
	}
	defer func() { _ = file.Close() }()

	writer := zip.NewWriter(file)
	defer func() { _ = writer.Close() }()

	return b.writeZipTree(fs, writer, projectDir, projectDir, outputDir, bundlePath)
}

// writeZipTree walks the project directory and writes files into the zip bundle.
func (b *PHPBuilder) writeZipTree(fs io.Medium, writer *zip.Writer, rootDir, currentDir, outputDir, bundlePath string) error {
	entries, err := fs.List(currentDir)
	if err != nil {
		return coreerr.E("PHPBuilder.writeZipTree", "failed to list directory", err)
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
			return coreerr.E("PHPBuilder.writeZipTree", "failed to relativise bundle path", err)
		}

		info, err := fs.Stat(entryPath)
		if err != nil {
			return coreerr.E("PHPBuilder.writeZipTree", "failed to stat bundle entry", err)
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return coreerr.E("PHPBuilder.writeZipTree", "failed to create zip header", err)
		}
		header.Name = strings.ReplaceAll(relPath, ax.DS(), "/")
		header.Method = zip.Deflate
		header.SetModTime(deterministicZipTime)

		zipEntry, err := writer.CreateHeader(header)
		if err != nil {
			return coreerr.E("PHPBuilder.writeZipTree", "failed to create zip entry", err)
		}

		source, err := fs.Open(entryPath)
		if err != nil {
			return coreerr.E("PHPBuilder.writeZipTree", "failed to open bundle entry", err)
		}

		if _, err := stdio.Copy(zipEntry, source); err != nil {
			_ = source.Close()
			return coreerr.E("PHPBuilder.writeZipTree", "failed to write bundle entry", err)
		}
		if err := source.Close(); err != nil {
			return coreerr.E("PHPBuilder.writeZipTree", "failed to close bundle entry", err)
		}
	}

	return nil
}

// isExcludedPath reports whether a path should be omitted from the bundle.
func (b *PHPBuilder) isExcludedPath(path, outputDir, bundlePath string) bool {
	cleanPath := ax.Clean(path)
	cleanOutputDir := ax.Clean(outputDir)
	cleanBundlePath := ax.Clean(bundlePath)

	if cleanPath == cleanOutputDir || strings.HasPrefix(cleanPath, cleanOutputDir+ax.DS()) {
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
