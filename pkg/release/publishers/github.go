// Package publishers provides release publishing implementations.
package publishers

import (
	"context"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	coreerr "dappco.re/go/core/log"
)

// GitHubPublisher publishes releases to GitHub using the gh CLI.
//
// pub := publishers.NewGitHubPublisher()
type GitHubPublisher struct{}

// NewGitHubPublisher creates a new GitHub publisher.
//
// pub := publishers.NewGitHubPublisher()
func NewGitHubPublisher() *GitHubPublisher {
	return &GitHubPublisher{}
}

// Name returns the publisher's identifier.
//
// name := pub.Name() // → "github"
func (p *GitHubPublisher) Name() string {
	return "github"
}

// Validate checks that the GitHub publisher has a release to publish.
func (p *GitHubPublisher) Validate(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig) error {
	_ = ctx
	_ = pubCfg
	_ = relCfg
	return validatePublisherRelease(p.Name(), release)
}

// Supports reports whether the publisher handles the requested target.
func (p *GitHubPublisher) Supports(target string) bool {
	return supportsPublisherTarget(p.Name(), target)
}

// Publish publishes the release to GitHub using the gh CLI.
//
// err := pub.Publish(ctx, rel, pubCfg, relCfg, false) // dryRun=true to preview
func (p *GitHubPublisher) Publish(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig, dryRun bool) error {
	// Determine repository
	repo := ""
	if relCfg != nil {
		repo = relCfg.GetRepository()
	}
	if repo == "" {
		// Try to detect from git remote
		detectedRepo, err := detectRepository(ctx, release.ProjectDir)
		if err != nil {
			return coreerr.E("github.Publish", "could not determine repository", err)
		}
		repo = detectedRepo
	}

	if dryRun {
		return p.dryRunPublish(release, pubCfg, repo)
	}

	ghCommand, err := resolveGhCli()
	if err != nil {
		return err
	}

	// Validate gh CLI is available and authenticated for actual publish
	if err := validateGhAuth(ctx, ghCommand); err != nil {
		return err
	}

	return p.executePublish(ctx, release, pubCfg, repo, ghCommand)
}

// dryRunPublish shows what would be done without actually publishing.
func (p *GitHubPublisher) dryRunPublish(release *Release, pubCfg PublisherConfig, repo string) error {
	publisherPrintln()
	publisherPrintln("=== DRY RUN: GitHub Release ===")
	publisherPrintln()
	publisherPrint("Repository: %s", repo)
	publisherPrint("Version:    %s", release.Version)
	publisherPrint("Draft:      %t", pubCfg.Draft)
	publisherPrint("Prerelease: %t", pubCfg.Prerelease)
	publisherPrintln()

	publisherPrintln("Would create release with command:")
	args := p.buildCreateArgs(release, pubCfg, repo)
	publisherPrint("  gh %s", core.Join(" ", args...))
	publisherPrintln()

	if len(release.Artifacts) > 0 {
		publisherPrintln("Would upload artifacts:")
		for _, artifact := range release.Artifacts {
			publisherPrint("  - %s", ax.Base(artifact.Path))
		}
	}

	publisherPrintln()
	publisherPrintln("Changelog:")
	publisherPrintln("---")
	publisherPrintln(release.Changelog)
	publisherPrintln("---")
	publisherPrintln()
	publisherPrintln("=== END DRY RUN ===")

	return nil
}

// executePublish actually creates the release and uploads artifacts.
func (p *GitHubPublisher) executePublish(ctx context.Context, release *Release, pubCfg PublisherConfig, repo, ghCommand string) error {
	// Build the release create command
	args := p.buildCreateArgs(release, pubCfg, repo)

	// Add artifact paths to the command
	for _, artifact := range release.Artifacts {
		args = append(args, artifact.Path)
	}

	// Execute gh release create
	if err := publisherRun(ctx, release.ProjectDir, nil, ghCommand, args...); err != nil {
		return coreerr.E("github.Publish", "gh release create failed", err)
	}

	return nil
}

// buildCreateArgs builds the arguments for gh release create.
func (p *GitHubPublisher) buildCreateArgs(release *Release, pubCfg PublisherConfig, repo string) []string {
	args := []string{"release", "create", release.Version}

	// Add repository flag
	if repo != "" {
		args = append(args, "--repo", repo)
	}

	// Add title
	args = append(args, "--title", release.Version)

	// Add notes (changelog)
	if release.Changelog != "" {
		args = append(args, "--notes", release.Changelog)
	} else {
		args = append(args, "--generate-notes")
	}

	// Add draft flag
	if pubCfg.Draft {
		args = append(args, "--draft")
	}

	// Add prerelease flag
	if pubCfg.Prerelease {
		args = append(args, "--prerelease")
	}

	return args
}

func resolveGhCli(paths ...string) (string, error) {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/gh",
			"/opt/homebrew/bin/gh",
		}
	}

	command, err := ax.ResolveCommand("gh", paths...)
	if err != nil {
		return "", coreerr.E("github.resolveGhCli", "gh CLI not found. Install it from https://cli.github.com", err)
	}

	return command, nil
}

// validateGhCli checks if the gh CLI is available and authenticated.
func validateGhCli(ctx context.Context) error {
	ghCommand, err := resolveGhCli()
	if err != nil {
		return err
	}

	return validateGhAuth(ctx, ghCommand)
}

func validateGhAuth(ctx context.Context, ghCommand string) error {
	output, err := ax.CombinedOutput(ctx, "", nil, ghCommand, "auth", "status")
	if err != nil {
		return coreerr.E("github.validateGhCli", "not authenticated with gh CLI. Run 'gh auth login' first", err)
	}

	if !core.Contains(output, "Logged in") {
		return coreerr.E("github.validateGhCli", "not authenticated with gh CLI. Run 'gh auth login' first", nil)
	}

	return nil
}

// detectRepository detects the GitHub repository from git remote.
func detectRepository(ctx context.Context, dir string) (string, error) {
	output, err := ax.RunDir(ctx, dir, "git", "remote", "get-url", "origin")
	if err != nil {
		return "", coreerr.E("github.detectRepository", "failed to get git remote", err)
	}

	return parseGitHubRepo(core.Trim(output))
}

// parseGitHubRepo extracts owner/repo from a GitHub URL.
// Supports:
//   - git@github.com:owner/repo.git
//   - https://github.com/owner/repo.git
//   - https://github.com/owner/repo
func parseGitHubRepo(url string) (string, error) {
	// SSH format
	if core.HasPrefix(url, "git@github.com:") {
		repo := core.TrimPrefix(url, "git@github.com:")
		repo = core.TrimSuffix(repo, ".git")
		return repo, nil
	}

	// HTTPS format
	if core.HasPrefix(url, "https://github.com/") {
		repo := core.TrimPrefix(url, "https://github.com/")
		repo = core.TrimSuffix(repo, ".git")
		return repo, nil
	}

	return "", coreerr.E("github.parseGitHubRepo", "not a GitHub URL: "+url, nil)
}

// UploadArtifact uploads a single artifact to an existing release.
// This can be used to add artifacts to a release after creation.
//
// err := publishers.UploadArtifact(ctx, "host-uk/core-build", "v1.2.3", "dist/core-build_v1.2.3_linux_amd64.tar.gz")
func UploadArtifact(ctx context.Context, repo, version, artifactPath string) error {
	ghCommand, err := resolveGhCli()
	if err != nil {
		return err
	}

	if err := publisherRun(ctx, "", nil, ghCommand, "release", "upload", version, artifactPath, "--repo", repo); err != nil {
		return coreerr.E("github.UploadArtifact", "failed to upload "+artifactPath, err)
	}

	return nil
}

// DeleteRelease deletes a release by tag name.
//
// err := publishers.DeleteRelease(ctx, "host-uk/core-build", "v1.2.3")
func DeleteRelease(ctx context.Context, repo, version string) error {
	ghCommand, err := resolveGhCli()
	if err != nil {
		return err
	}

	if err := publisherRun(ctx, "", nil, ghCommand, "release", "delete", version, "--repo", repo, "--yes"); err != nil {
		return coreerr.E("github.DeleteRelease", "failed to delete "+version, err)
	}

	return nil
}

// ReleaseExists checks if a release exists for the given version.
//
// exists := publishers.ReleaseExists(ctx, "host-uk/core-build", "v1.2.3")
func ReleaseExists(ctx context.Context, repo, version string) bool {
	ghCommand, err := resolveGhCli()
	if err != nil {
		return false
	}

	return ax.Exec(ctx, ghCommand, "release", "view", version, "--repo", repo) == nil
}
