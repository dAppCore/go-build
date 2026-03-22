// Package release provides release automation with changelog generation and publishing.
// It orchestrates the build system, changelog generation, and publishing to targets
// like GitHub Releases.
package release

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"forge.lthn.ai/core/go-build/pkg/build"
	"forge.lthn.ai/core/go-build/pkg/build/builders"
	"forge.lthn.ai/core/go-build/pkg/release/publishers"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

// Release represents a release with its version, artifacts, and changelog.
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

// resolveProjectDir validates cfg is non-nil and returns the absolute project directory.
func resolveProjectDir(cfg *Config, caller string) (string, error) {
	if cfg == nil {
		return "", coreerr.E(caller, "config is nil", nil)
	}
	projectDir := cfg.projectDir
	if projectDir == "" {
		projectDir = "."
	}
	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return "", coreerr.E(caller, "failed to resolve project directory", err)
	}
	return absProjectDir, nil
}

// resolveVersion returns cfg.version if set, otherwise determines it from git tags.
func resolveVersion(cfg *Config, absProjectDir, caller string) (string, error) {
	if cfg.version != "" {
		return cfg.version, nil
	}
	version, err := DetermineVersion(absProjectDir)
	if err != nil {
		return "", coreerr.E(caller, "failed to determine version", err)
	}
	return version, nil
}

// newRelease constructs a Release with the changelog generated (non-fatal on failure).
func newRelease(version string, artifacts []build.Artifact, absProjectDir string) *Release {
	changelog, err := Generate(absProjectDir, "", version)
	if err != nil {
		changelog = fmt.Sprintf("Release %s", version)
	}
	return &Release{
		Version:    version,
		Artifacts:  artifacts,
		Changelog:  changelog,
		ProjectDir: absProjectDir,
		FS:         io.Local,
	}
}

// publishAll dispatches the release to all configured publishers.
func publishAll(ctx context.Context, cfg *Config, release *Release, dryRun bool, caller string) error {
	if len(cfg.Publishers) == 0 {
		return nil
	}
	pubRelease := publishers.NewRelease(release.Version, release.Artifacts, release.Changelog, release.ProjectDir, release.FS)
	for _, pubCfg := range cfg.Publishers {
		publisher, err := getPublisher(pubCfg.Type)
		if err != nil {
			return coreerr.E(caller, "unsupported publisher", err)
		}
		publisherCfg := publishers.NewPublisherConfig(pubCfg.Type, pubCfg.Prerelease, pubCfg.Draft, buildExtendedConfig(pubCfg))
		if err := publisher.Publish(ctx, pubRelease, publisherCfg, cfg, dryRun); err != nil {
			return coreerr.E(caller, "publish to "+pubCfg.Type+" failed", err)
		}
	}
	return nil
}

// Publish publishes pre-built artifacts from dist/ to configured targets.
// Use this after `core build` to separate build and publish concerns.
// If dryRun is true, it will show what would be done without actually publishing.
func Publish(ctx context.Context, cfg *Config, dryRun bool) (*Release, error) {
	absProjectDir, err := resolveProjectDir(cfg, "release.Publish")
	if err != nil {
		return nil, err
	}

	version, err := resolveVersion(cfg, absProjectDir, "release.Publish")
	if err != nil {
		return nil, err
	}

	distDir := filepath.Join(absProjectDir, "dist")
	artifacts, err := findArtifacts(io.Local, distDir)
	if err != nil {
		return nil, coreerr.E("release.Publish", "failed to find artifacts", err)
	}
	if len(artifacts) == 0 {
		return nil, coreerr.E("release.Publish", "no artifacts found in dist/\nRun 'core build' first to create artifacts", nil)
	}

	release := newRelease(version, artifacts, absProjectDir)
	return release, publishAll(ctx, cfg, release, dryRun, "release.Publish")
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
		path := filepath.Join(distDir, name)

		// Include archives and checksums
		if strings.HasSuffix(name, ".tar.gz") ||
			strings.HasSuffix(name, ".zip") ||
			strings.HasSuffix(name, ".txt") ||
			strings.HasSuffix(name, ".sig") {
			artifacts = append(artifacts, build.Artifact{Path: path})
		}
	}

	return artifacts, nil
}

// Run executes the full release process: determine version, build artifacts,
// generate changelog, and publish to configured targets.
// For separated concerns, prefer using `core build` then `core ci` (Publish).
// If dryRun is true, it will show what would be done without actually publishing.
func Run(ctx context.Context, cfg *Config, dryRun bool) (*Release, error) {
	absProjectDir, err := resolveProjectDir(cfg, "release.Run")
	if err != nil {
		return nil, err
	}

	version, err := resolveVersion(cfg, absProjectDir, "release.Run")
	if err != nil {
		return nil, err
	}

	artifacts, err := buildArtifacts(ctx, io.Local, cfg, absProjectDir, version)
	if err != nil {
		return nil, coreerr.E("release.Run", "build failed", err)
	}

	release := newRelease(version, artifacts, absProjectDir)
	return release, publishAll(ctx, cfg, release, dryRun, "release.Run")
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
		binaryName = filepath.Base(projectDir)
	}

	// Determine output directory
	outputDir := filepath.Join(projectDir, "dist")

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
	checksumPath := filepath.Join(outputDir, "CHECKSUMS.txt")
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
