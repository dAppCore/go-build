// Package publishers provides release publishing implementations.
package publishers

import (
	"context"
	"reflect"

	"dappco.re/go"
	"dappco.re/go/build/pkg/build"
	storage "dappco.re/go/build/pkg/storage"
)

// Release represents a release to be published.
//
// rel := publishers.NewRelease("v1.2.3", artifacts, changelog, ".", storage.Local)
type Release struct {
	// Version is the semantic version string (e.g., "v1.2.3").
	Version string
	// Artifacts are the built release artifacts.
	Artifacts []build.Artifact
	// Changelog is the generated markdown changelog.
	Changelog string
	// ProjectDir is the root directory of the project.
	ProjectDir string
	// FS is the project filesystem used for local project file access.
	FS storage.Medium
	// ArtifactFS is the medium backing the release artifact paths.
	ArtifactFS storage.Medium
}

// PublisherConfig holds configuration for a publisher.
//
// cfg := publishers.NewPublisherConfig("github", false, false, nil)
type PublisherConfig struct {
	// Type is the publisher type (e.g., "github", "linuxkit", "docker").
	Type string
	// Prerelease marks the release as a prerelease.
	Prerelease bool
	// Draft creates the release as a draft.
	Draft bool
	// Extended holds publisher-specific configuration.
	Extended any
}

// ReleaseConfig holds release configuration needed by publishers.
//
// var relCfg publishers.ReleaseConfig = cfg // *release.Config implements this interface
type ReleaseConfig interface {
	GetRepository() string
	GetProjectName() string
}

// Publisher defines the interface for release publishers.
//
// var pub publishers.Publisher = publishers.NewGitHubPublisher()
type Publisher interface {
	// Name returns the publisher's identifier.
	Name() string
	// Validate checks the runtime release and publisher configuration before publish.
	Validate(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig) core.Result
	// Publish publishes the release to the target.
	// If dryRun is true, it prints what would be done without executing.
	Publish(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig, dryRun bool) core.Result
	// Supports reports whether the publisher handles the named target.
	Supports(target string) bool
}

// NewRelease creates a Release from the release package's Release type.
// This is a helper to convert between packages.
//
// rel := publishers.NewRelease("v1.2.3", artifacts, changelog, ".", storage.Local)
func NewRelease(version string, artifacts []build.Artifact, changelog, projectDir string, fs storage.Medium) *Release {
	return NewReleaseWithArtifactFS(version, artifacts, changelog, projectDir, fs, fs)
}

// NewReleaseWithArtifactFS creates a Release with explicit project and artifact media.
//
// rel := publishers.NewReleaseWithArtifactFS("v1.2.3", artifacts, changelog, ".", storage.Local, storage.NewMemoryMedium())
func NewReleaseWithArtifactFS(version string, artifacts []build.Artifact, changelog, projectDir string, fs storage.Medium, artifactFS storage.Medium) *Release {
	if artifactFS == nil {
		artifactFS = fs
	}

	return &Release{
		Version:    version,
		Artifacts:  artifacts,
		Changelog:  changelog,
		ProjectDir: projectDir,
		FS:         fs,
		ArtifactFS: artifactFS,
	}
}

// NewPublisherConfig creates a PublisherConfig.
//
// cfg := publishers.NewPublisherConfig("github", false, false, nil)
func NewPublisherConfig(pubType string, prerelease, draft bool, extended any) PublisherConfig {
	return PublisherConfig{
		Type:       pubType,
		Prerelease: prerelease,
		Draft:      draft,
		Extended:   extended,
	}
}

func validatePublisherRelease(name string, release *Release) core.Result {
	if release == nil {
		return core.Fail(core.E(name+".Validate", "release is nil", nil))
	}
	if release.FS == nil {
		return core.Fail(core.E(name+".Validate", "release filesystem (FS) is nil", nil))
	}
	validated := build.ValidateVersionIdentifier(release.Version)
	if !validated.OK {
		return core.Fail(core.E(name+".Validate", "release version contains unsupported characters", core.NewError(validated.Error())))
	}
	return core.Ok(nil)
}

func releaseArtifactFS(release *Release) storage.Medium {
	if release == nil {
		return nil
	}
	if release.ArtifactFS != nil {
		return release.ArtifactFS
	}
	return release.FS
}

func mediumEquals(left, right storage.Medium) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	leftType := reflect.TypeOf(left)
	rightType := reflect.TypeOf(right)
	if leftType != rightType || !leftType.Comparable() {
		return false
	}

	return reflect.ValueOf(left).Interface() == reflect.ValueOf(right).Interface()
}

func supportsPublisherTarget(name, target string) bool {
	return core.Lower(core.Trim(target)) == core.Lower(name)
}
