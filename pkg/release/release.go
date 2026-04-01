// Package release provides release automation with changelog generation and publishing.
// It orchestrates the build system, changelog generation, and publishing to targets
// like GitHub Releases.
package release

import (
	"context"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/internal/projectdetect"
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

	filesystem := io.Local

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
	artifacts, err := findArtifacts(filesystem, distDir)
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
		FS:         filesystem,
	}

	// Step 4: Publish to configured targets
	if len(cfg.Publishers) > 0 {
		pubRelease := publishers.NewRelease(release.Version, release.Artifacts, release.Changelog, release.ProjectDir, release.FS)

		for _, publisherConfig := range cfg.Publishers {
			publisher, err := getPublisher(publisherConfig.Type)
			if err != nil {
				return release, coreerr.E("release.Publish", "unsupported publisher", err)
			}

			extendedConfig := buildExtendedConfig(publisherConfig)
			publisherRuntimeConfig := publishers.NewPublisherConfig(publisherConfig.Type, publisherConfig.Prerelease, publisherConfig.Draft, extendedConfig)
			if err := publisher.Publish(ctx, pubRelease, publisherRuntimeConfig, cfg, dryRun); err != nil {
				return release, coreerr.E("release.Publish", "publish to "+publisherConfig.Type+" failed", err)
			}
		}
	}

	return release, nil
}

// findArtifacts discovers pre-built artifacts in the dist directory.
func findArtifacts(filesystem io.Medium, distDir string) ([]build.Artifact, error) {
	if !filesystem.IsDir(distDir) {
		return nil, coreerr.E("release.findArtifacts", "dist/ directory not found", nil)
	}

	var artifacts []build.Artifact

	entries, err := filesystem.List(distDir)
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

	filesystem := io.Local

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
	artifacts, err := buildArtifacts(ctx, filesystem, cfg, absProjectDir, version)
	if err != nil {
		return nil, coreerr.E("release.Run", "build failed", err)
	}

	release := &Release{
		Version:    version,
		Artifacts:  artifacts,
		Changelog:  changelog,
		ProjectDir: absProjectDir,
		FS:         filesystem,
	}

	// Step 4: Publish to configured targets
	if len(cfg.Publishers) > 0 {
		// Convert to publisher types
		pubRelease := publishers.NewRelease(release.Version, release.Artifacts, release.Changelog, release.ProjectDir, release.FS)

		for _, publisherConfig := range cfg.Publishers {
			publisher, err := getPublisher(publisherConfig.Type)
			if err != nil {
				return release, coreerr.E("release.Run", "unsupported publisher", err)
			}

			// Build extended config for publisher-specific settings
			extendedConfig := buildExtendedConfig(publisherConfig)
			publisherRuntimeConfig := publishers.NewPublisherConfig(publisherConfig.Type, publisherConfig.Prerelease, publisherConfig.Draft, extendedConfig)
			if err := publisher.Publish(ctx, pubRelease, publisherRuntimeConfig, cfg, dryRun); err != nil {
				return release, coreerr.E("release.Run", "publish to "+publisherConfig.Type+" failed", err)
			}
		}
	}

	return release, nil
}

// buildArtifacts builds all artifacts for the release.
func buildArtifacts(ctx context.Context, filesystem io.Medium, cfg *Config, projectDir, version string) ([]build.Artifact, error) {
	// Load build configuration
	buildConfig, err := build.LoadConfig(filesystem, projectDir)
	if err != nil {
		return nil, coreerr.E("release.buildArtifacts", "failed to load build config", err)
	}

	// Determine targets
	var targets []build.Target
	if len(cfg.Build.Targets) > 0 {
		for _, targetConfig := range cfg.Build.Targets {
			targets = append(targets, build.Target{OS: targetConfig.OS, Arch: targetConfig.Arch})
		}
	} else if len(buildConfig.Targets) > 0 {
		targets = buildConfig.ToTargets()
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
		binaryName = buildConfig.Project.Binary
	}
	if binaryName == "" {
		binaryName = buildConfig.Project.Name
	}
	if binaryName == "" {
		binaryName = ax.Base(projectDir)
	}

	// Determine output directory
	outputDir := ax.Join(projectDir, "dist")

	// Get builder (detect project type)
	projectType, err := projectdetect.DetectProjectType(filesystem, projectDir)
	if err != nil {
		return nil, coreerr.E("release.buildArtifacts", "failed to detect project type", err)
	}

	builder, err := getBuilder(projectType)
	if err != nil {
		return nil, err
	}

	// Build configuration
	builderConfig := &build.Config{
		FS:         filesystem,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       binaryName,
		Version:    version,
		LDFlags:    buildConfig.Build.LDFlags,
	}

	// Build
	artifacts, err := builder.Build(ctx, builderConfig, targets)
	if err != nil {
		return nil, coreerr.E("release.buildArtifacts", "build failed", err)
	}

	// Archive artifacts
	archivedArtifacts, err := build.ArchiveAll(filesystem, artifacts)
	if err != nil {
		return nil, coreerr.E("release.buildArtifacts", "archive failed", err)
	}

	// Compute checksums
	checksummedArtifacts, err := build.ChecksumAll(filesystem, archivedArtifacts)
	if err != nil {
		return nil, coreerr.E("release.buildArtifacts", "checksum failed", err)
	}

	// Write CHECKSUMS.txt
	checksumPath := ax.Join(outputDir, "CHECKSUMS.txt")
	if err := build.WriteChecksumFile(filesystem, checksummedArtifacts, checksumPath); err != nil {
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
		return builders.NewNodeBuilder(), nil
	case build.ProjectTypePHP:
		return builders.NewPHPBuilder(), nil
	case build.ProjectTypeDocs:
		return builders.NewDocsBuilder(), nil
	default:
		return nil, coreerr.E("release.getBuilder", "unsupported project type: "+string(projectType), nil)
	}
}

// getPublisher returns the publisher for the given type.
func getPublisher(publisherType string) (publishers.Publisher, error) {
	switch publisherType {
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
		return nil, coreerr.E("release.getPublisher", "unsupported publisher type: "+publisherType, nil)
	}
}

// buildExtendedConfig builds a map of extended configuration for a publisher.
func buildExtendedConfig(publisherConfig PublisherConfig) map[string]any {
	extendedConfig := make(map[string]any)

	// LinuxKit-specific config
	if publisherConfig.Config != "" {
		extendedConfig["config"] = publisherConfig.Config
	}
	if len(publisherConfig.Formats) > 0 {
		extendedConfig["formats"] = toAnySlice(publisherConfig.Formats)
	}
	if len(publisherConfig.Platforms) > 0 {
		extendedConfig["platforms"] = toAnySlice(publisherConfig.Platforms)
	}

	// Docker-specific config
	if publisherConfig.Registry != "" {
		extendedConfig["registry"] = publisherConfig.Registry
	}
	if publisherConfig.Image != "" {
		extendedConfig["image"] = publisherConfig.Image
	}
	if publisherConfig.Dockerfile != "" {
		extendedConfig["dockerfile"] = publisherConfig.Dockerfile
	}
	if len(publisherConfig.Tags) > 0 {
		extendedConfig["tags"] = toAnySlice(publisherConfig.Tags)
	}
	if len(publisherConfig.BuildArgs) > 0 {
		args := make(map[string]any)
		for k, v := range publisherConfig.BuildArgs {
			args[k] = v
		}
		extendedConfig["build_args"] = args
	}

	// npm-specific config
	if publisherConfig.Package != "" {
		extendedConfig["package"] = publisherConfig.Package
	}
	if publisherConfig.Access != "" {
		extendedConfig["access"] = publisherConfig.Access
	}

	// Homebrew-specific config
	if publisherConfig.Tap != "" {
		extendedConfig["tap"] = publisherConfig.Tap
	}
	if publisherConfig.Formula != "" {
		extendedConfig["formula"] = publisherConfig.Formula
	}

	// Scoop-specific config
	if publisherConfig.Bucket != "" {
		extendedConfig["bucket"] = publisherConfig.Bucket
	}

	// AUR-specific config
	if publisherConfig.Maintainer != "" {
		extendedConfig["maintainer"] = publisherConfig.Maintainer
	}

	// Chocolatey-specific configuration
	if publisherConfig.Push {
		extendedConfig["push"] = publisherConfig.Push
	}

	// Official repo config (shared by multiple publishers)
	if publisherConfig.Official != nil {
		official := make(map[string]any)
		official["enabled"] = publisherConfig.Official.Enabled
		if publisherConfig.Official.Output != "" {
			official["output"] = publisherConfig.Official.Output
		}
		extendedConfig["official"] = official
	}

	return extendedConfig
}

// toAnySlice converts a string slice to an any slice.
func toAnySlice(s []string) []any {
	result := make([]any, len(s))
	for i, v := range s {
		result[i] = v
	}
	return result
}
