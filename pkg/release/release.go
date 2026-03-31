// Package release provides release automation with changelog generation and publishing.
// It orchestrates the build system, changelog generation, and publishing to targets
// like GitHub Releases.
package release

import (
	"context"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/build/pkg/build/builders"
	"dappco.re/go/core/build/pkg/release/publishers"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

// Release represents a release with its version, artifacts, and changelog.
//
// rel, err := release.Publish(ctx, cfg, false)
type Release struct {
	// Version is the semantic version string (e.g., "v1.2.3").
	Version string
	// Artifacts are the built release artifacts (archives with checksums).
	Artifacts []build.Artifact
	// Changelog is the generated markdown changelog.
	Changelog string
	// ProjectDir is the root directory of the project.
	ProjectDir string
	// FS is the medium for file operations.
	FS io.Medium
}

// Publish publishes pre-built artifacts from dist/ to configured targets.
// Use this after `core build` to separate build and publish concerns.
//
// rel, err := release.Publish(ctx, cfg, false) // dryRun=true to preview
func Publish(ctx context.Context, cfg *Config, dryRun bool) (*Release, error) {
	if cfg == nil {
		return nil, coreerr.E("release.Publish", "config is nil", nil)
	}

	m := io.Local

	projectDir := cfg.projectDir
	if projectDir == "" {
		projectDir = "."
	}

	// Resolve to absolute path
	absProjectDir, err := ax.Abs(projectDir)
	if err != nil {
		return nil, coreerr.E("release.Publish", "failed to resolve project directory", err)
	}

	// Step 1: Determine version
	version := cfg.version
	if version == "" {
		version, err = DetermineVersionWithContext(ctx, absProjectDir)
		if err != nil {
			return nil, coreerr.E("release.Publish", "failed to determine version", err)
		}
	}

	// Step 2: Find pre-built artifacts in dist/
	distDir := ax.Join(absProjectDir, "dist")
	artifacts, err := findArtifacts(m, distDir)
	if err != nil {
		return nil, coreerr.E("release.Publish", "failed to find artifacts", err)
	}

	if len(artifacts) == 0 {
		return nil, coreerr.E("release.Publish", "no artifacts found in dist/\nRun 'core build' first to create artifacts", nil)
	}

	// Step 3: Generate changelog
	changelog, err := GenerateWithContext(ctx, absProjectDir, "", version)
	if err != nil {
		if ctx.Err() != nil {
			return nil, coreerr.E("release.Publish", "changelog generation cancelled", ctx.Err())
		}
		// Non-fatal: continue with empty changelog
		changelog = core.Sprintf("Release %s", version)
	}

	release := &Release{
		Version:    version,
		Artifacts:  artifacts,
		Changelog:  changelog,
		ProjectDir: absProjectDir,
		FS:         m,
	}

	// Step 4: Publish to configured targets
	if len(cfg.Publishers) > 0 {
		pubRelease := publishers.NewRelease(release.Version, release.Artifacts, release.Changelog, release.ProjectDir, release.FS)

		for _, pubCfg := range cfg.Publishers {
			publisher, err := getPublisher(pubCfg.Type)
			if err != nil {
				return release, coreerr.E("release.Publish", "unsupported publisher", err)
			}

			extendedCfg := buildExtendedConfig(pubCfg)
			publisherCfg := publishers.NewPublisherConfig(pubCfg.Type, pubCfg.Prerelease, pubCfg.Draft, extendedCfg)
			if err := publisher.Publish(ctx, pubRelease, publisherCfg, cfg, dryRun); err != nil {
				return release, coreerr.E("release.Publish", "publish to "+pubCfg.Type+" failed", err)
			}
		}
	}

	return release, nil
}

// findArtifacts discovers pre-built artifacts in the dist directory.
func findArtifacts(m io.Medium, distDir string) ([]build.Artifact, error) {
	if !m.IsDir(distDir) {
		return nil, coreerr.E("release.findArtifacts", "dist/ directory not found", nil)
	}

	var artifacts []build.Artifact

	entries, err := m.List(distDir)
	if err != nil {
		return nil, coreerr.E("release.findArtifacts", "failed to read dist/", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		path := ax.Join(distDir, name)

		// Include archives and checksums
		if core.HasSuffix(name, ".tar.gz") ||
			core.HasSuffix(name, ".zip") ||
			core.HasSuffix(name, ".txt") ||
			core.HasSuffix(name, ".sig") {
			artifacts = append(artifacts, build.Artifact{Path: path})
		}
	}

	return artifacts, nil
}

// Run executes the full release process: determine version, build artifacts,
// generate changelog, and publish to configured targets.
// For separated concerns, prefer `core build` then `core ci` (Publish).
//
// rel, err := release.Run(ctx, cfg, false) // dryRun=true to preview
func Run(ctx context.Context, cfg *Config, dryRun bool) (*Release, error) {
	if cfg == nil {
		return nil, coreerr.E("release.Run", "config is nil", nil)
	}

	m := io.Local

	projectDir := cfg.projectDir
	if projectDir == "" {
		projectDir = "."
	}

	// Resolve to absolute path
	absProjectDir, err := ax.Abs(projectDir)
	if err != nil {
		return nil, coreerr.E("release.Run", "failed to resolve project directory", err)
	}

	// Step 1: Determine version
	version := cfg.version
	if version == "" {
		version, err = DetermineVersionWithContext(ctx, absProjectDir)
		if err != nil {
			return nil, coreerr.E("release.Run", "failed to determine version", err)
		}
	}

	// Step 2: Generate changelog
	changelog, err := GenerateWithContext(ctx, absProjectDir, "", version)
	if err != nil {
		if ctx.Err() != nil {
			return nil, coreerr.E("release.Run", "changelog generation cancelled", ctx.Err())
		}
		// Non-fatal: continue with empty changelog
		changelog = core.Sprintf("Release %s", version)
	}

	// Step 3: Build artifacts
	artifacts, err := buildArtifacts(ctx, m, cfg, absProjectDir, version)
	if err != nil {
		return nil, coreerr.E("release.Run", "build failed", err)
	}

	release := &Release{
		Version:    version,
		Artifacts:  artifacts,
		Changelog:  changelog,
		ProjectDir: absProjectDir,
		FS:         m,
	}

	// Step 4: Publish to configured targets
	if len(cfg.Publishers) > 0 {
		// Convert to publisher types
		pubRelease := publishers.NewRelease(release.Version, release.Artifacts, release.Changelog, release.ProjectDir, release.FS)

		for _, pubCfg := range cfg.Publishers {
			publisher, err := getPublisher(pubCfg.Type)
			if err != nil {
				return release, coreerr.E("release.Run", "unsupported publisher", err)
			}

			// Build extended config for publisher-specific settings
			extendedCfg := buildExtendedConfig(pubCfg)
			publisherCfg := publishers.NewPublisherConfig(pubCfg.Type, pubCfg.Prerelease, pubCfg.Draft, extendedCfg)
			if err := publisher.Publish(ctx, pubRelease, publisherCfg, cfg, dryRun); err != nil {
				return release, coreerr.E("release.Run", "publish to "+pubCfg.Type+" failed", err)
			}
		}
	}

	return release, nil
}

// buildArtifacts builds all artifacts for the release.
func buildArtifacts(ctx context.Context, fs io.Medium, cfg *Config, projectDir, version string) ([]build.Artifact, error) {
	// Load build configuration
	buildCfg, err := build.LoadConfig(fs, projectDir)
	if err != nil {
		return nil, coreerr.E("release.buildArtifacts", "failed to load build config", err)
	}

	// Determine targets
	var targets []build.Target
	if len(cfg.Build.Targets) > 0 {
		for _, t := range cfg.Build.Targets {
			targets = append(targets, build.Target{OS: t.OS, Arch: t.Arch})
		}
	} else if len(buildCfg.Targets) > 0 {
		targets = buildCfg.ToTargets()
	} else {
		// Default targets
		targets = []build.Target{
			{OS: "linux", Arch: "amd64"},
			{OS: "linux", Arch: "arm64"},
			{OS: "darwin", Arch: "arm64"},
			{OS: "windows", Arch: "amd64"},
		}
	}

	// Determine binary name
	binaryName := cfg.Project.Name
	if binaryName == "" {
		binaryName = buildCfg.Project.Binary
	}
	if binaryName == "" {
		binaryName = buildCfg.Project.Name
	}
	if binaryName == "" {
		binaryName = ax.Base(projectDir)
	}

	// Determine output directory
	outputDir := ax.Join(projectDir, "dist")

	// Get builder (detect project type)
	projectType, err := build.PrimaryType(fs, projectDir)
	if err != nil {
		return nil, coreerr.E("release.buildArtifacts", "failed to detect project type", err)
	}

	builder, err := getBuilder(projectType)
	if err != nil {
		return nil, err
	}

	// Build configuration
	buildConfig := &build.Config{
		FS:         fs,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       binaryName,
		Version:    version,
		LDFlags:    buildCfg.Build.LDFlags,
	}

	// Build
	artifacts, err := builder.Build(ctx, buildConfig, targets)
	if err != nil {
		return nil, coreerr.E("release.buildArtifacts", "build failed", err)
	}

	// Archive artifacts
	archivedArtifacts, err := build.ArchiveAll(fs, artifacts)
	if err != nil {
		return nil, coreerr.E("release.buildArtifacts", "archive failed", err)
	}

	// Compute checksums
	checksummedArtifacts, err := build.ChecksumAll(fs, archivedArtifacts)
	if err != nil {
		return nil, coreerr.E("release.buildArtifacts", "checksum failed", err)
	}

	// Write CHECKSUMS.txt
	checksumPath := ax.Join(outputDir, "CHECKSUMS.txt")
	if err := build.WriteChecksumFile(fs, checksummedArtifacts, checksumPath); err != nil {
		return nil, coreerr.E("release.buildArtifacts", "failed to write checksums file", err)
	}

	// Add CHECKSUMS.txt as an artifact
	checksumArtifact := build.Artifact{
		Path: checksumPath,
	}
	checksummedArtifacts = append(checksummedArtifacts, checksumArtifact)

	return checksummedArtifacts, nil
}

// getBuilder returns the appropriate builder for the project type.
func getBuilder(projectType build.ProjectType) (build.Builder, error) {
	switch projectType {
	case build.ProjectTypeWails:
		return builders.NewWailsBuilder(), nil
	case build.ProjectTypeGo:
		return builders.NewGoBuilder(), nil
	case build.ProjectTypeNode:
		return nil, coreerr.E("release.getBuilder", "node.js builder not yet implemented", nil)
	case build.ProjectTypePHP:
		return nil, coreerr.E("release.getBuilder", "PHP builder not yet implemented", nil)
	default:
		return nil, coreerr.E("release.getBuilder", "unsupported project type: "+string(projectType), nil)
	}
}

// getPublisher returns the publisher for the given type.
func getPublisher(pubType string) (publishers.Publisher, error) {
	switch pubType {
	case "github":
		return publishers.NewGitHubPublisher(), nil
	case "linuxkit":
		return publishers.NewLinuxKitPublisher(), nil
	case "docker":
		return publishers.NewDockerPublisher(), nil
	case "npm":
		return publishers.NewNpmPublisher(), nil
	case "homebrew":
		return publishers.NewHomebrewPublisher(), nil
	case "scoop":
		return publishers.NewScoopPublisher(), nil
	case "aur":
		return publishers.NewAURPublisher(), nil
	case "chocolatey":
		return publishers.NewChocolateyPublisher(), nil
	default:
		return nil, coreerr.E("release.getPublisher", "unsupported publisher type: "+pubType, nil)
	}
}

// buildExtendedConfig builds a map of extended configuration for a publisher.
func buildExtendedConfig(pubCfg PublisherConfig) map[string]any {
	ext := make(map[string]any)

	// LinuxKit-specific config
	if pubCfg.Config != "" {
		ext["config"] = pubCfg.Config
	}
	if len(pubCfg.Formats) > 0 {
		ext["formats"] = toAnySlice(pubCfg.Formats)
	}
	if len(pubCfg.Platforms) > 0 {
		ext["platforms"] = toAnySlice(pubCfg.Platforms)
	}

	// Docker-specific config
	if pubCfg.Registry != "" {
		ext["registry"] = pubCfg.Registry
	}
	if pubCfg.Image != "" {
		ext["image"] = pubCfg.Image
	}
	if pubCfg.Dockerfile != "" {
		ext["dockerfile"] = pubCfg.Dockerfile
	}
	if len(pubCfg.Tags) > 0 {
		ext["tags"] = toAnySlice(pubCfg.Tags)
	}
	if len(pubCfg.BuildArgs) > 0 {
		args := make(map[string]any)
		for k, v := range pubCfg.BuildArgs {
			args[k] = v
		}
		ext["build_args"] = args
	}

	// npm-specific config
	if pubCfg.Package != "" {
		ext["package"] = pubCfg.Package
	}
	if pubCfg.Access != "" {
		ext["access"] = pubCfg.Access
	}

	// Homebrew-specific config
	if pubCfg.Tap != "" {
		ext["tap"] = pubCfg.Tap
	}
	if pubCfg.Formula != "" {
		ext["formula"] = pubCfg.Formula
	}

	// Scoop-specific config
	if pubCfg.Bucket != "" {
		ext["bucket"] = pubCfg.Bucket
	}

	// AUR-specific config
	if pubCfg.Maintainer != "" {
		ext["maintainer"] = pubCfg.Maintainer
	}

	// Chocolatey-specific configuration
	if pubCfg.Push {
		ext["push"] = pubCfg.Push
	}

	// Official repo config (shared by multiple publishers)
	if pubCfg.Official != nil {
		official := make(map[string]any)
		official["enabled"] = pubCfg.Official.Enabled
		if pubCfg.Official.Output != "" {
			official["output"] = pubCfg.Official.Output
		}
		ext["official"] = official
	}

	return ext
}

// toAnySlice converts a string slice to an any slice.
func toAnySlice(s []string) []any {
	result := make([]any, len(s))
	for i, v := range s {
		result[i] = v
	}
	return result
}
