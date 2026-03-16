// Package publishers provides release publishing implementations.
package publishers

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	coreerr "forge.lthn.ai/core/go-log"
)

// GitHubPublisher publishes releases to GitHub using the gh CLI.
type GitHubPublisher struct{}

// NewGitHubPublisher creates a new GitHub publisher.
func NewGitHubPublisher() *GitHubPublisher {
	return &GitHubPublisher{}
}

// Name returns the publisher's identifier.
func (p *GitHubPublisher) Name() string {
	return "github"
}

// Publish publishes the release to GitHub.
// Uses the gh CLI for creating releases and uploading assets.
func (p *GitHubPublisher) Publish(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig, dryRun bool) error {
	// Determine repository
	repo := ""
	if relCfg != nil {
		repo = relCfg.GetRepository()
	}
	if repo == "" {
		// Try to detect from git remote
		detectedRepo, err := detectRepository(release.ProjectDir)
		if err != nil {
			return coreerr.E("github.Publish", "could not determine repository", err)
		}
		repo = detectedRepo
	}

	if dryRun {
		return p.dryRunPublish(release, pubCfg, repo)
	}

	// Validate gh CLI is available and authenticated for actual publish
	if err := validateGhCli(); err != nil {
		return err
	}

	return p.executePublish(ctx, release, pubCfg, repo)
}

// dryRunPublish shows what would be done without actually publishing.
func (p *GitHubPublisher) dryRunPublish(release *Release, pubCfg PublisherConfig, repo string) error {
	fmt.Println()
	fmt.Println("=== DRY RUN: GitHub Release ===")
	fmt.Println()
	fmt.Printf("Repository: %s\n", repo)
	fmt.Printf("Version:    %s\n", release.Version)
	fmt.Printf("Draft:      %t\n", pubCfg.Draft)
	fmt.Printf("Prerelease: %t\n", pubCfg.Prerelease)
	fmt.Println()

	fmt.Println("Would create release with command:")
	args := p.buildCreateArgs(release, pubCfg, repo)
	fmt.Printf("  gh %s\n", strings.Join(args, " "))
	fmt.Println()

	if len(release.Artifacts) > 0 {
		fmt.Println("Would upload artifacts:")
		for _, artifact := range release.Artifacts {
			fmt.Printf("  - %s\n", filepath.Base(artifact.Path))
		}
	}

	fmt.Println()
	fmt.Println("Changelog:")
	fmt.Println("---")
	fmt.Println(release.Changelog)
	fmt.Println("---")
	fmt.Println()
	fmt.Println("=== END DRY RUN ===")

	return nil
}

// executePublish actually creates the release and uploads artifacts.
func (p *GitHubPublisher) executePublish(ctx context.Context, release *Release, pubCfg PublisherConfig, repo string) error {
	// Build the release create command
	args := p.buildCreateArgs(release, pubCfg, repo)

	// Add artifact paths to the command
	for _, artifact := range release.Artifacts {
		args = append(args, artifact.Path)
	}

	// Execute gh release create
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Dir = release.ProjectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
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

// validateGhCli checks if the gh CLI is available and authenticated.
func validateGhCli() error {
	// Check if gh is installed
	cmd := exec.Command("gh", "--version")
	if err := cmd.Run(); err != nil {
		return coreerr.E("github.validateGhCli", "gh CLI not found. Install it from https://cli.github.com", nil)
	}

	// Check if authenticated
	cmd = exec.Command("gh", "auth", "status")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return coreerr.E("github.validateGhCli", "not authenticated with gh CLI. Run 'gh auth login' first", nil)
	}

	if !strings.Contains(string(output), "Logged in") {
		return coreerr.E("github.validateGhCli", "not authenticated with gh CLI. Run 'gh auth login' first", nil)
	}

	return nil
}

// detectRepository detects the GitHub repository from git remote.
func detectRepository(dir string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", coreerr.E("github.detectRepository", "failed to get git remote", err)
	}

	url := strings.TrimSpace(string(output))
	return parseGitHubRepo(url)
}

// parseGitHubRepo extracts owner/repo from a GitHub URL.
// Supports:
//   - git@github.com:owner/repo.git
//   - https://github.com/owner/repo.git
//   - https://github.com/owner/repo
func parseGitHubRepo(url string) (string, error) {
	// SSH format
	if strings.HasPrefix(url, "git@github.com:") {
		repo := strings.TrimPrefix(url, "git@github.com:")
		repo = strings.TrimSuffix(repo, ".git")
		return repo, nil
	}

	// HTTPS format
	if strings.HasPrefix(url, "https://github.com/") {
		repo := strings.TrimPrefix(url, "https://github.com/")
		repo = strings.TrimSuffix(repo, ".git")
		return repo, nil
	}

	return "", coreerr.E("github.parseGitHubRepo", "not a GitHub URL: "+url, nil)
}

// UploadArtifact uploads a single artifact to an existing release.
// This can be used to add artifacts to a release after creation.
func UploadArtifact(ctx context.Context, repo, version, artifactPath string) error {
	cmd := exec.CommandContext(ctx, "gh", "release", "upload", version, artifactPath, "--repo", repo)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return coreerr.E("github.UploadArtifact", "failed to upload "+artifactPath, err)
	}

	return nil
}

// DeleteRelease deletes a release by tag name.
func DeleteRelease(ctx context.Context, repo, version string) error {
	cmd := exec.CommandContext(ctx, "gh", "release", "delete", version, "--repo", repo, "--yes")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return coreerr.E("github.DeleteRelease", "failed to delete "+version, err)
	}

	return nil
}

// ReleaseExists checks if a release exists for the given version.
func ReleaseExists(ctx context.Context, repo, version string) bool {
	cmd := exec.CommandContext(ctx, "gh", "release", "view", version, "--repo", repo)
	return cmd.Run() == nil
}
