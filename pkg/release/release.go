// Package release provides release automation with changelog generation and publishing.
// It orchestrates the build system, changelog generation, and publishing to targets
// like GitHub Releases.
package release

import (
	"context" // Note: AX-6 — carries cancellation through release build and publish workflows.
	stdfs "io/fs"
	"slices" // Note: AX-6 — sorts discovered release artifacts deterministically.

	"dappco.re/go"                              // Note: AX-6 — provides approved string and formatting helpers.
	"dappco.re/go/build/internal/ax"            // Note: AX-6 — Core-backed path and filesystem helpers replace banned stdlib calls.
	"dappco.re/go/build/pkg/build"              // Note: AX-6 — release pipeline depends on build config, artifacts, and checksum helpers.
	"dappco.re/go/build/pkg/build/builders"     // Note: AX-6 — resolves project builders for release artifact generation.
	"dappco.re/go/build/pkg/build/signing"      // Note: AX-6 — wires release signing and notarization hooks.
	"dappco.re/go/build/pkg/release/publishers" // Note: AX-6 — publishes completed release artifacts to configured targets.
	"dappco.re/go/io"                           // Note: AX-6 — Medium abstraction for release filesystem access.
	coreerr "dappco.re/go/log"                  // Note: AX-6 — wraps release errors with Core logging semantics.
)

// release signing hooks allow tests to observe the release pipeline without
// shelling out to platform-specific signing tools.
var (
	signReleaseBinaries        = signing.SignBinaries
	notarizeReleaseBinaries    = signing.NotarizeBinaries
	signReleaseChecksums       = signing.SignChecksums
	generateReleaseChangelogFn = generateReleaseChangelog
)

const defaultChecksumFileName = "CHECKSUMS.txt"

// Release represents a release with its version, artifacts, and changelog.
//
// result := release.Publish(ctx, cfg, false)
type Release struct {
	// Version is the semantic version string (e.g., "v1.2.3").
	Version string
	// Artifacts are the built release artifacts (archives with checksums).
	Artifacts []build.Artifact
	// Changelog is the generated markdown changelog.
	Changelog string
	// ProjectDir is the root directory of the project.
	ProjectDir string
	// FS is the project filesystem used for local project file access.
	FS io.Medium
	// ArtifactFS is the medium backing the release artifact paths.
	ArtifactFS io.Medium
}

// Publish publishes pre-built artifacts from dist/ to configured targets.
// Use this after `core build` to separate build and publish concerns.
//
// result := release.Publish(ctx, cfg, false) // dryRun=true to preview
func Publish(ctx context.Context, cfg *Config, dryRun bool) core.Result {
	if cfg == nil {
		return core.Fail(coreerr.E("release.Publish", "config is nil", nil))
	}

	projectFS := io.Local
	artifactFS := resolveReleaseOutputMedium(cfg)

	projectDir := cfg.projectDir
	if projectDir == "" {
		projectDir = "."
	}

	// Resolve to absolute path
	absProjectDirResult := ax.Abs(projectDir)
	if !absProjectDirResult.OK {
		return core.Fail(coreerr.E("release.Publish", "failed to resolve project directory", core.NewError(absProjectDirResult.Error())))
	}
	absProjectDir := absProjectDirResult.Value.(string)

	// Step 1: Determine version
	version := cfg.version
	if version == "" {
		versionResult := DetermineVersionWithContext(ctx, absProjectDir)
		if !versionResult.OK {
			return core.Fail(coreerr.E("release.Publish", "failed to determine version", core.NewError(versionResult.Error())))
		}
		version = versionResult.Value.(string)
	}
	validatedVersion := ValidateVersionIdentifier(version)
	if !validatedVersion.OK {
		return core.Fail(coreerr.E("release.Publish", "invalid release version override", core.NewError(validatedVersion.Error())))
	}

	// Step 2: Find pre-built artifacts in dist/
	distDir := resolveReleaseOutputRoot(absProjectDir, cfg, artifactFS)
	artifactsResult := findArtifacts(artifactFS, distDir)
	if !artifactsResult.OK {
		return core.Fail(coreerr.E("release.Publish", "failed to find artifacts", core.NewError(artifactsResult.Error())))
	}
	artifacts := artifactsResult.Value.([]build.Artifact)
	artifacts = appendConfiguredChecksumArtifacts(artifactFS, distDir, artifacts, cfg)

	if len(artifacts) == 0 {
		return core.Fail(coreerr.E("release.Publish", "no artifacts found in dist/\nRun 'core build' first to create artifacts", nil))
	}

	// Step 3: Generate changelog
	changelogResult := generateReleaseChangelogFn(ctx, absProjectDir, version, cfg)
	changelog := ""
	if !changelogResult.OK {
		if ctx.Err() != nil {
			return core.Fail(coreerr.E("release.Publish", "changelog generation cancelled", ctx.Err()))
		}
		// Non-fatal: continue with empty changelog
		changelog = core.Sprintf("Release %s", version)
	} else {
		changelog = changelogResult.Value.(string)
	}

	release := &Release{
		Version:    version,
		Artifacts:  artifacts,
		Changelog:  changelog,
		ProjectDir: absProjectDir,
		FS:         projectFS,
		ArtifactFS: artifactFS,
	}

	// Step 4: Publish to configured targets
	if len(cfg.Publishers) > 0 {
		pubRelease := publishers.NewReleaseWithArtifactFS(release.Version, release.Artifacts, release.Changelog, release.ProjectDir, release.FS, release.ArtifactFS)

		for _, publisherConfig := range cfg.Publishers {
			publisherResult := getPublisher(publisherConfig.Type)
			if !publisherResult.OK {
				return core.Fail(coreerr.E("release.Publish", "unsupported publisher", core.NewError(publisherResult.Error())))
			}
			publisher := publisherResult.Value.(publishers.Publisher)

			extendedConfig := buildExtendedConfig(publisherConfig)
			publisherRuntimeConfig := publishers.NewPublisherConfig(publisherConfig.Type, publisherConfig.Prerelease, publisherConfig.Draft, extendedConfig)
			if !publisher.Supports(publisherConfig.Type) {
				return core.Fail(coreerr.E("release.Publish", "publisher does not support target "+publisherConfig.Type, nil))
			}
			validated := publisher.Validate(ctx, pubRelease, publisherRuntimeConfig, cfg)
			if !validated.OK {
				return core.Fail(coreerr.E("release.Publish", "validate publisher "+publisherConfig.Type+" failed", core.NewError(validated.Error())))
			}
			published := publisher.Publish(ctx, pubRelease, publisherRuntimeConfig, cfg, dryRun)
			if !published.OK {
				return core.Fail(coreerr.E("release.Publish", "publish to "+publisherConfig.Type+" failed", core.NewError(published.Error())))
			}
		}
	}

	return core.Ok(release)
}

// findArtifacts discovers pre-built artifacts in the dist directory.
func findArtifacts(filesystem io.Medium, distDir string) core.Result {
	if distDir != "" && !filesystem.IsDir(distDir) {
		return core.Fail(coreerr.E("release.findArtifacts", "dist/ directory not found", nil))
	}

	releaseArtifactsResult := findReleaseArtifacts(filesystem, distDir)
	if !releaseArtifactsResult.OK {
		return releaseArtifactsResult
	}
	releaseArtifacts := releaseArtifactsResult.Value.([]build.Artifact)

	platformArtifactsResult := findPlatformArtifacts(filesystem, distDir)
	if !platformArtifactsResult.OK {
		return platformArtifactsResult
	}
	platformArtifacts := platformArtifactsResult.Value.([]build.Artifact)

	switch {
	case len(releaseArtifacts) == 0:
		return core.Ok(platformArtifacts)
	case len(platformArtifacts) == 0 || hasReleaseArchives(releaseArtifacts):
		sortArtifactsByPath(releaseArtifacts)
		return core.Ok(releaseArtifacts)
	default:
		artifacts := append(platformArtifacts, releaseArtifacts...)
		sortArtifactsByPath(artifacts)
		return core.Ok(artifacts)
	}
}

func findReleaseArtifacts(filesystem io.Medium, currentDir string) core.Result {
	entriesResult := filesystem.List(currentDir)
	if !entriesResult.OK {
		return core.Fail(coreerr.E("release.findArtifacts", "failed to read dist/", core.NewError(entriesResult.Error())))
	}
	entries := entriesResult.Value.([]stdfs.DirEntry)

	var artifacts []build.Artifact
	for _, entry := range entries {
		path := joinReleasePath(currentDir, entry.Name())
		if entry.IsDir() {
			nestedArtifactsResult := findReleaseArtifacts(filesystem, path)
			if !nestedArtifactsResult.OK {
				return nestedArtifactsResult
			}
			nestedArtifacts := nestedArtifactsResult.Value.([]build.Artifact)
			artifacts = append(artifacts, nestedArtifacts...)
			continue
		}

		if artifact, ok := releaseArtifactFromName(path, entry.Name()); ok {
			artifacts = append(artifacts, artifact)
		}
	}

	return core.Ok(artifacts)
}

func findPlatformArtifacts(filesystem io.Medium, distDir string) core.Result {
	var artifacts []build.Artifact
	collected := collectPlatformArtifacts(filesystem, distDir, &artifacts)
	if !collected.OK {
		return collected
	}

	sortArtifactsByPath(artifacts)
	return core.Ok(artifacts)
}

func collectPlatformArtifacts(filesystem io.Medium, currentDir string, artifacts *[]build.Artifact) core.Result {
	entriesResult := filesystem.List(currentDir)
	if !entriesResult.OK {
		return core.Fail(coreerr.E("release.findArtifacts", "failed to read dist/", core.NewError(entriesResult.Error())))
	}
	entries := entriesResult.Value.([]stdfs.DirEntry)

	for _, entry := range entries {
		if !entry.IsDir() || shouldSkipRecursivePlatformDir(entry.Name()) {
			continue
		}

		path := joinReleasePath(currentDir, entry.Name())
		osValue, archValue, ok := parsePlatformDir(entry.Name())
		if ok {
			filesResult := filesystem.List(path)
			if !filesResult.OK {
				continue
			}
			files := filesResult.Value.([]stdfs.DirEntry)

			for _, file := range files {
				if file.IsDir() {
					if shouldPublishAppBundle(file.Name()) {
						*artifacts = append(*artifacts, build.Artifact{
							Path: joinReleasePath(path, file.Name()),
							OS:   osValue,
							Arch: archValue,
						})
					}
					continue
				}

				name := file.Name()
				if !shouldPublishRawArtifact(name) {
					continue
				}

				*artifacts = append(*artifacts, build.Artifact{
					Path: joinReleasePath(path, name),
					OS:   osValue,
					Arch: archValue,
				})
			}
			continue
		}

		collected := collectPlatformArtifacts(filesystem, path, artifacts)
		if !collected.OK {
			return collected
		}
	}

	return core.Ok(nil)
}

func releaseArtifactFromName(path, name string) (build.Artifact, bool) {
	if shouldPublishArchive(name) || shouldPublishChecksum(name) || shouldPublishSignature(name) {
		return build.Artifact{Path: path}, true
	}
	return build.Artifact{}, false
}

func sortArtifactsByPath(artifacts []build.Artifact) {
	slices.SortFunc(artifacts, func(a, b build.Artifact) int {
		if a.Path < b.Path {
			return -1
		}
		if a.Path > b.Path {
			return 1
		}
		return 0
	})
}

func hasReleaseArchives(artifacts []build.Artifact) bool {
	for _, artifact := range artifacts {
		if shouldPublishArchive(ax.Base(artifact.Path)) {
			return true
		}
	}
	return false
}

func shouldPublishArchive(name string) bool {
	return core.HasSuffix(name, ".tar.gz") ||
		core.HasSuffix(name, ".tar.xz") ||
		core.HasSuffix(name, ".zip")
}

func shouldPublishChecksum(name string) bool {
	return name == "CHECKSUMS.txt"
}

func shouldPublishSignature(name string) bool {
	return core.HasSuffix(name, ".asc") ||
		core.HasSuffix(name, ".sig")
}

func shouldPublishRawArtifact(name string) bool {
	if name == "" || core.HasPrefix(name, ".") {
		return false
	}
	if name == "artifact_meta.json" || name == "CHECKSUMS.txt" || name == "CHECKSUMS.txt.asc" {
		return false
	}
	return true
}

func shouldPublishAppBundle(name string) bool {
	return core.HasSuffix(name, ".app")
}

func shouldSkipRecursivePlatformDir(name string) bool {
	return shouldPublishAppBundle(name)
}

func parsePlatformDir(name string) (string, string, bool) {
	parts := core.SplitN(name, "_", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}

	return parts[0], parts[1], true
}

// Run executes the full release process: determine version, build artifacts,
// generate changelog, and publish to configured targets.
// For separated concerns, prefer `core build` then `core ci` (Publish).
//
// result := release.Run(ctx, cfg, false) // dryRun=true to preview
func Run(ctx context.Context, cfg *Config, dryRun bool) core.Result {
	if cfg == nil {
		return core.Fail(coreerr.E("release.Run", "config is nil", nil))
	}

	projectFS := io.Local
	artifactFS := resolveReleaseOutputMedium(cfg)

	projectDir := cfg.projectDir
	if projectDir == "" {
		projectDir = "."
	}

	// Resolve to absolute path
	absProjectDirResult := ax.Abs(projectDir)
	if !absProjectDirResult.OK {
		return core.Fail(coreerr.E("release.Run", "failed to resolve project directory", core.NewError(absProjectDirResult.Error())))
	}
	absProjectDir := absProjectDirResult.Value.(string)

	// Step 1: Determine version
	version := cfg.version
	if version == "" {
		versionResult := DetermineVersionWithContext(ctx, absProjectDir)
		if !versionResult.OK {
			return core.Fail(coreerr.E("release.Run", "failed to determine version", core.NewError(versionResult.Error())))
		}
		version = versionResult.Value.(string)
	}
	validatedVersion := ValidateVersionIdentifier(version)
	if !validatedVersion.OK {
		return core.Fail(coreerr.E("release.Run", "invalid release version override", core.NewError(validatedVersion.Error())))
	}

	// Step 2: Generate changelog
	changelogResult := generateReleaseChangelogFn(ctx, absProjectDir, version, cfg)
	changelog := ""
	if !changelogResult.OK {
		if ctx.Err() != nil {
			return core.Fail(coreerr.E("release.Run", "changelog generation cancelled", ctx.Err()))
		}
		// Non-fatal: continue with empty changelog
		changelog = core.Sprintf("Release %s", version)
	} else {
		changelog = changelogResult.Value.(string)
	}

	// Step 3: Build artifacts
	outputDir := resolveReleaseOutputRoot(absProjectDir, cfg, artifactFS)
	artifactsResult := buildArtifacts(ctx, projectFS, cfg, absProjectDir, outputDir, version)
	if !artifactsResult.OK {
		return core.Fail(coreerr.E("release.Run", "build failed", core.NewError(artifactsResult.Error())))
	}
	artifacts := artifactsResult.Value.([]build.Artifact)

	release := &Release{
		Version:    version,
		Artifacts:  artifacts,
		Changelog:  changelog,
		ProjectDir: absProjectDir,
		FS:         projectFS,
		ArtifactFS: artifactFS,
	}

	// Step 4: Publish to configured targets
	if len(cfg.Publishers) > 0 {
		// Convert to publisher types
		pubRelease := publishers.NewReleaseWithArtifactFS(release.Version, release.Artifacts, release.Changelog, release.ProjectDir, release.FS, release.ArtifactFS)

		for _, publisherConfig := range cfg.Publishers {
			publisherResult := getPublisher(publisherConfig.Type)
			if !publisherResult.OK {
				return core.Fail(coreerr.E("release.Run", "unsupported publisher", core.NewError(publisherResult.Error())))
			}
			publisher := publisherResult.Value.(publishers.Publisher)

			// Build extended config for publisher-specific settings
			extendedConfig := buildExtendedConfig(publisherConfig)
			publisherRuntimeConfig := publishers.NewPublisherConfig(publisherConfig.Type, publisherConfig.Prerelease, publisherConfig.Draft, extendedConfig)
			if !publisher.Supports(publisherConfig.Type) {
				return core.Fail(coreerr.E("release.Run", "publisher does not support target "+publisherConfig.Type, nil))
			}
			validated := publisher.Validate(ctx, pubRelease, publisherRuntimeConfig, cfg)
			if !validated.OK {
				return core.Fail(coreerr.E("release.Run", "validate publisher "+publisherConfig.Type+" failed", core.NewError(validated.Error())))
			}
			published := publisher.Publish(ctx, pubRelease, publisherRuntimeConfig, cfg, dryRun)
			if !published.OK {
				return core.Fail(coreerr.E("release.Run", "publish to "+publisherConfig.Type+" failed", core.NewError(published.Error())))
			}
		}
	}

	return core.Ok(release)
}

// buildArtifacts builds all artifacts for the release.
func buildArtifacts(ctx context.Context, filesystem io.Medium, cfg *Config, projectDir, outputDir, version string) core.Result {
	artifactFS := resolveReleaseOutputMedium(cfg)

	// Load build configuration
	buildConfigResult := build.LoadConfig(filesystem, projectDir)
	if !buildConfigResult.OK {
		return core.Fail(coreerr.E("release.buildArtifacts", "failed to load build config", core.NewError(buildConfigResult.Error())))
	}
	buildConfig := buildConfigResult.Value.(*build.BuildConfig)

	cacheSetup := build.SetupBuildCache(filesystem, projectDir, buildConfig)
	if !cacheSetup.OK {
		return core.Fail(coreerr.E("release.buildArtifacts", "failed to set up build cache", core.NewError(cacheSetup.Error())))
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
			{OS: "darwin", Arch: "amd64"},
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

	pipeline := &build.Pipeline{
		FS:             filesystem,
		ResolveBuilder: getBuilder,
	}
	planResult := pipeline.Plan(ctx, build.PipelineRequest{
		ProjectDir:  projectDir,
		BuildConfig: buildConfig,
		OutputDir:   outputDir,
		BuildName:   binaryName,
		Targets:     targets,
		Version:     version,
	})
	if !planResult.OK {
		return core.Fail(coreerr.E("release.buildArtifacts", "failed to plan build", core.NewError(planResult.Error())))
	}
	plan := planResult.Value.(*build.PipelinePlan)
	plan.OutputDir = outputDir
	plan.RuntimeConfig.OutputDir = outputDir
	plan.RuntimeConfig.OutputMedium = artifactFS

	pipelineRunResult := pipeline.Run(ctx, plan)
	if !pipelineRunResult.OK {
		return core.Fail(coreerr.E("release.buildArtifacts", "build failed", core.NewError(pipelineRunResult.Error())))
	}
	pipelineResult := pipelineRunResult.Value.(*build.PipelineResult)
	artifacts := pipelineResult.Artifacts

	metadataWritten := writeArtifactMetadata(artifactFS, binaryName, artifacts)
	if !metadataWritten.OK {
		return core.Fail(coreerr.E("release.buildArtifacts", "failed to write artifact metadata", core.NewError(metadataWritten.Error())))
	}

	signingArtifacts := make([]signing.Artifact, len(artifacts))
	for i, artifact := range artifacts {
		signingArtifacts[i] = signing.Artifact{
			Path: artifact.Path,
			OS:   artifact.OS,
			Arch: artifact.Arch,
		}
	}

	if buildConfig.Sign.Enabled {
		signed := signReleaseBinaries(ctx, artifactFS, buildConfig.Sign, signingArtifacts)
		if !signed.OK {
			return core.Fail(coreerr.E("release.buildArtifacts", "failed to sign binaries", core.NewError(signed.Error())))
		}

		notarized := notarizeReleaseBinaries(ctx, artifactFS, buildConfig.Sign, signingArtifacts)
		if !notarized.OK {
			return core.Fail(coreerr.E("release.buildArtifacts", "failed to notarise binaries", core.NewError(notarized.Error())))
		}
	}

	// Archive artifacts
	archiveFormatValue := cfg.Build.ArchiveFormat
	if archiveFormatValue == "" {
		archiveFormatValue = buildConfig.Build.ArchiveFormat
	}

	archiveFormatResult := build.ParseArchiveFormat(archiveFormatValue)
	if !archiveFormatResult.OK {
		return core.Fail(coreerr.E("release.buildArtifacts", "invalid archive format", core.NewError(archiveFormatResult.Error())))
	}
	archiveFormat := archiveFormatResult.Value.(build.ArchiveFormat)

	archivedArtifactsResult := build.ArchiveAllWithFormat(artifactFS, artifacts, archiveFormat)
	if !archivedArtifactsResult.OK {
		return core.Fail(coreerr.E("release.buildArtifacts", "archive failed", core.NewError(archivedArtifactsResult.Error())))
	}
	archivedArtifacts := archivedArtifactsResult.Value.([]build.Artifact)

	// Compute checksums
	checksummedArtifactsResult := build.ChecksumAll(artifactFS, archivedArtifacts)
	if !checksummedArtifactsResult.OK {
		return core.Fail(coreerr.E("release.buildArtifacts", "checksum failed", core.NewError(checksummedArtifactsResult.Error())))
	}
	checksummedArtifacts := checksummedArtifactsResult.Value.([]build.Artifact)

	validatedChecksum := validateChecksumAlgorithm(cfg)
	if !validatedChecksum.OK {
		return validatedChecksum
	}

	checksumPath := resolveChecksumPath(outputDir, cfg)
	wroteChecksums := build.WriteChecksumFile(artifactFS, checksummedArtifacts, checksumPath)
	if !wroteChecksums.OK {
		return core.Fail(coreerr.E("release.buildArtifacts", "failed to write checksums file", core.NewError(wroteChecksums.Error())))
	}

	// Sign CHECKSUMS.txt when signing is configured.
	signedChecksums := signReleaseChecksums(ctx, artifactFS, buildConfig.Sign, checksumPath)
	if !signedChecksums.OK {
		return core.Fail(coreerr.E("release.buildArtifacts", "failed to sign checksums file", core.NewError(signedChecksums.Error())))
	}

	// Add CHECKSUMS.txt as an artifact
	checksumArtifact := build.Artifact{
		Path: checksumPath,
	}
	checksummedArtifacts = append(checksummedArtifacts, checksumArtifact)

	// Add the detached signature when one was created.
	signaturePath := checksumPath + ".asc"
	if artifactFS.Exists(signaturePath) {
		checksummedArtifacts = append(checksummedArtifacts, build.Artifact{
			Path: signaturePath,
		})
	}

	return core.Ok(checksummedArtifacts)
}

func validateChecksumAlgorithm(cfg *Config) core.Result {
	algorithm := core.Lower(core.Trim(resolveChecksumAlgorithm(cfg)))
	switch algorithm {
	case "", "sha256":
		return core.Ok(nil)
	default:
		return core.Fail(coreerr.E("release.buildArtifacts", "unsupported checksum algorithm: "+algorithm, nil))
	}
}

func resolveChecksumAlgorithm(cfg *Config) string {
	if cfg == nil {
		return "sha256"
	}
	if cfg.Checksum.Algorithm != "" {
		return cfg.Checksum.Algorithm
	}
	return "sha256"
}

func resolveChecksumPath(outputDir string, cfg *Config) string {
	fileName := defaultChecksumFileName
	if cfg != nil && cfg.Checksum.File != "" {
		fileName = cfg.Checksum.File
	}
	if ax.IsAbs(fileName) {
		return ax.Clean(fileName)
	}
	return joinReleasePath(outputDir, fileName)
}

func appendConfiguredChecksumArtifacts(filesystem io.Medium, distDir string, artifacts []build.Artifact, cfg *Config) []build.Artifact {
	checksumPath := resolveChecksumPath(distDir, cfg)
	if !filesystem.Exists(checksumPath) || containsArtifactPath(artifacts, checksumPath) {
		return artifacts
	}

	artifacts = append(artifacts, build.Artifact{Path: checksumPath})
	signaturePath := checksumPath + ".asc"
	if filesystem.Exists(signaturePath) && !containsArtifactPath(artifacts, signaturePath) {
		artifacts = append(artifacts, build.Artifact{Path: signaturePath})
	}

	return artifacts
}

func containsArtifactPath(artifacts []build.Artifact, path string) bool {
	for _, artifact := range artifacts {
		if artifact.Path == path {
			return true
		}
	}
	return false
}

func generateReleaseChangelog(ctx context.Context, projectDir, version string, cfg *Config) core.Result {
	if cfg == nil {
		return GenerateWithContext(ctx, projectDir, "", version)
	}
	return GenerateWithConfigWithContext(ctx, projectDir, "", version, &cfg.Changelog)
}

// writeArtifactMetadata writes artifact_meta.json files next to built artifacts
// when GitHub metadata is available.
func writeArtifactMetadata(filesystem io.Medium, buildName string, artifacts []build.Artifact) core.Result {
	ci := build.DetectCI()
	if ci == nil {
		ci = build.DetectGitHubMetadata()
	}
	if ci == nil {
		return core.Ok(nil)
	}

	for _, artifact := range artifacts {
		if artifact.OS == "" || artifact.Arch == "" {
			continue
		}
		metaPath := ax.Join(ax.Dir(artifact.Path), "artifact_meta.json")
		written := build.WriteArtifactMeta(filesystem, metaPath, buildName, build.Target{OS: artifact.OS, Arch: artifact.Arch}, ci)
		if !written.OK {
			return written
		}
	}

	return core.Ok(nil)
}

// getBuilder returns the appropriate builder for the project type.
func getBuilder(projectType build.ProjectType) core.Result {
	builder := builders.ResolveBuilder(projectType)
	if !builder.OK {
		return core.Fail(coreerr.E("release.getBuilder", "unsupported project type: "+string(projectType), core.NewError(builder.Error())))
	}
	return builder
}

// resolveProjectType determines which builder type to use for release builds.
// An explicit build type in .core/build.yaml takes precedence over marker-based detection.
func resolveProjectType(filesystem io.Medium, projectDir, buildType string) core.Result {
	if buildType != "" {
		return core.Ok(build.ProjectType(buildType))
	}

	return build.PrimaryType(filesystem, projectDir)
}

// getPublisher returns the publisher for the given type.
func getPublisher(publisherType string) core.Result {
	switch publisherType {
	case "github":
		return core.Ok(publishers.NewGitHubPublisher())
	case "linuxkit":
		return core.Ok(publishers.NewLinuxKitPublisher())
	case "docker":
		return core.Ok(publishers.NewDockerPublisher())
	case "npm":
		return core.Ok(publishers.NewNpmPublisher())
	case "homebrew":
		return core.Ok(publishers.NewHomebrewPublisher())
	case "scoop":
		return core.Ok(publishers.NewScoopPublisher())
	case "aur":
		return core.Ok(publishers.NewAURPublisher())
	case "chocolatey":
		return core.Ok(publishers.NewChocolateyPublisher())
	default:
		return core.Fail(coreerr.E("release.getPublisher", "unsupported publisher type: "+publisherType, nil))
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
