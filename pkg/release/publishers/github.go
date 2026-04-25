// Package publishers provides release publishing implementations.
package publishers

import (
	"bufio"
	"context"
	"errors"
	stdio "io"
	"io/fs"
	"net/url"
	"sort"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/core"
	coreio "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

// GitHubPublisher publishes releases to GitHub using the gh CLI.
//
// pub := publishers.NewGitHubPublisher()
type GitHubPublisher struct{}

type gitRemote struct {
	Name string
	URL  string
}

// NewGitHubPublisher creates a new GitHub publisher.
//
// pub := publishers.NewGitHubPublisher()
func NewGitHubPublisher() *GitHubPublisher {
	return &GitHubPublisher{}
}

// DetectGitHubRepository detects the owner/repo pair from any GitHub remote in
// the repository. Repositories that use Forge or another self-hosted origin can
// still resolve against a secondary `github` remote.
func DetectGitHubRepository(ctx context.Context, dir string) (string, error) {
	return detectRepository(ctx, dir)
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
	prerelease := shouldMarkGitHubPrerelease(release, pubCfg)

	publisherPrintln()
	publisherPrintln("=== DRY RUN: GitHub Release ===")
	publisherPrintln()
	publisherPrint("Repository: %s", repo)
	publisherPrint("Version:    %s", release.Version)
	publisherPrint("Draft:      %t", pubCfg.Draft)
	publisherPrint("Prerelease: %t", prerelease)
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

	artifactPaths, cleanup, err := p.materializeArtifacts(release)
	if err != nil {
		return err
	}
	defer cleanup()

	args = append(args, artifactPaths...)

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
	if shouldMarkGitHubPrerelease(release, pubCfg) {
		args = append(args, "--prerelease")
	}

	return args
}

func shouldMarkGitHubPrerelease(release *Release, pubCfg PublisherConfig) bool {
	if pubCfg.Prerelease {
		return true
	}
	if release == nil {
		return false
	}
	return isSemverPrerelease(release.Version)
}

func isSemverPrerelease(version string) bool {
	version = core.Trim(version)
	version = core.TrimPrefix(version, "v")
	if version == "" {
		return false
	}

	version = core.SplitN(version, "+", 2)[0]

	parts := core.SplitN(version, "-", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}

	return isCoreSemver(parts[0])
}

func isCoreSemver(version string) bool {
	parts := core.Split(version, ".")
	if len(parts) != 3 {
		return false
	}

	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}

	return true
}

func (p *GitHubPublisher) materializeArtifacts(release *Release) ([]string, func(), error) {
	artifactFS := releaseArtifactFS(release)
	if artifactFS == nil {
		return nil, func() {}, coreerr.E("github.Publish", "artifact filesystem is nil", nil)
	}

	paths := make([]string, 0, len(release.Artifacts))
	if mediumEquals(artifactFS, coreio.Local) {
		for _, artifact := range release.Artifacts {
			paths = append(paths, artifact.Path)
		}
		return paths, func() {}, nil
	}

	tempDir, err := ax.TempDir("github-release-artifacts-*")
	if err != nil {
		return nil, func() {}, coreerr.E("github.Publish", "failed to create artifact staging directory", err)
	}

	for i, artifact := range release.Artifacts {
		localPath := ax.Join(tempDir, core.Sprintf("%03d", i), ax.Base(artifact.Path))
		if err := copyArtifactPathToLocal(artifactFS, artifact.Path, localPath); err != nil {
			_ = ax.RemoveAll(tempDir)
			return nil, func() {}, coreerr.E("github.Publish", "failed to stage artifact "+artifact.Path, err)
		}
		paths = append(paths, localPath)
	}

	return paths, func() {
		_ = ax.RemoveAll(tempDir)
	}, nil
}

func copyArtifactPathToLocal(artifactFS coreio.Medium, sourcePath, destinationPath string) error {
	if artifactFS.IsDir(sourcePath) {
		return copyArtifactDirToLocal(artifactFS, sourcePath, destinationPath)
	}

	return copyArtifactFileToLocal(artifactFS, sourcePath, destinationPath)
}

func copyArtifactDirToLocal(artifactFS coreio.Medium, sourcePath, destinationPath string) error {
	if err := coreio.Local.EnsureDir(destinationPath); err != nil {
		return coreerr.E("github.copyArtifactDirToLocal", "failed to create destination directory", err)
	}

	entries, err := artifactFS.List(sourcePath)
	if err != nil {
		return coreerr.E("github.copyArtifactDirToLocal", "failed to list artifact directory", err)
	}

	for _, entry := range entries {
		childSource := ax.Join(sourcePath, entry.Name())
		childDestination := ax.Join(destinationPath, entry.Name())
		if err := copyArtifactPathToLocal(artifactFS, childSource, childDestination); err != nil {
			return err
		}
	}

	return nil
}

func copyArtifactFileToLocal(artifactFS coreio.Medium, sourcePath, destinationPath string) error {
	file, err := artifactFS.Open(sourcePath)
	if err != nil {
		return coreerr.E("github.copyArtifactFileToLocal", "failed to open artifact", err)
	}
	defer func() { _ = file.Close() }()

	content, err := stdio.ReadAll(file)
	if err != nil {
		return coreerr.E("github.copyArtifactFileToLocal", "failed to read artifact", err)
	}

	mode := fs.FileMode(0o644)
	if info, err := artifactFS.Stat(sourcePath); err == nil {
		mode = info.Mode()
	}

	if err := coreio.Local.WriteMode(destinationPath, string(content), mode); err != nil {
		return coreerr.E("github.copyArtifactFileToLocal", "failed to write staged artifact", err)
	}

	return nil
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
	remotes, err := listGitRemotes(ctx, dir)
	if err != nil {
		return "", coreerr.E("github.detectRepository", "failed to list git remotes", err)
	}
	if len(remotes) == 0 {
		repo, ghErr := detectRepositoryViaGh(ctx, dir)
		if ghErr == nil {
			return repo, nil
		}
		return "", coreerr.E("github.detectRepository", "no git remotes configured", ghErr)
	}

	var parseErr error
	for _, remote := range remotes {
		repo, err := parseGitHubRepo(remote.URL)
		if err == nil {
			return repo, nil
		}
		if parseErr == nil {
			parseErr = err
		}
	}

	repo, ghErr := detectRepositoryViaGh(ctx, dir)
	if ghErr == nil {
		return repo, nil
	}
	if parseErr == nil {
		parseErr = ghErr
	} else if ghErr != nil {
		parseErr = errors.Join(parseErr, ghErr)
	}

	return "", coreerr.E("github.detectRepository", "no GitHub remote found", parseErr)
}

func detectRepositoryViaGh(ctx context.Context, dir string) (string, error) {
	ghCommand, err := resolveGhCli()
	if err != nil {
		return "", coreerr.E("github.detectRepositoryViaGh", "gh CLI not available for repository fallback", err)
	}

	output, err := ax.CombinedOutput(ctx, dir, nil, ghCommand, "repo", "view", "--json", "nameWithOwner")
	if err != nil {
		return "", coreerr.E("github.detectRepositoryViaGh", "gh repo view failed", err)
	}

	var payload struct {
		NameWithOwner string `json:"nameWithOwner"`
	}
	if err := ax.JSONUnmarshal([]byte(output), &payload); err != nil {
		return "", coreerr.E("github.detectRepositoryViaGh", "failed to parse gh repo view output", err)
	}

	repo := core.Trim(payload.NameWithOwner)
	if repo == "" {
		return "", coreerr.E("github.detectRepositoryViaGh", "gh repo view did not report a repository", nil)
	}

	return repo, nil
}

// parseGitHubRepo extracts owner/repo from a GitHub URL.
// Supports:
//   - git@github.com:owner/repo.git
//   - https://github.com/owner/repo.git
//   - https://github.com/owner/repo
func parseGitHubRepo(url string) (string, error) {
	url = core.Trim(url)
	if url == "" {
		return "", coreerr.E("github.parseGitHubRepo", "not a GitHub URL: "+url, nil)
	}

	// SSH format
	if core.HasPrefix(url, "git@github.com:") {
		repo := core.TrimPrefix(url, "git@github.com:")
		return normaliseGitHubRepoPath(repo)
	}

	parsed, err := urlpkgParse(url)
	if err == nil && core.Lower(parsed.Hostname()) == "github.com" {
		return normaliseGitHubRepoPath(parsed.Path)
	}

	return "", coreerr.E("github.parseGitHubRepo", "not a GitHub URL: "+url, nil)
}

func listGitRemotes(ctx context.Context, dir string) ([]gitRemote, error) {
	output, err := ax.RunDir(ctx, dir, "git", "remote", "-v")
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	remotes := make([]gitRemote, 0)
	scanner := bufio.NewScanner(core.NewReader(output))
	for scanner.Scan() {
		fields := splitRemoteFields(scanner.Text())
		if len(fields) < 2 {
			continue
		}

		name := core.Trim(fields[0])
		remoteURL := core.Trim(fields[1])
		if name == "" || remoteURL == "" {
			continue
		}

		key := name + "\n" + remoteURL
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		remotes = append(remotes, gitRemote{Name: name, URL: remoteURL})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	sort.SliceStable(remotes, func(i, j int) bool {
		if remotes[i].Name == remotes[j].Name {
			return remotes[i].URL < remotes[j].URL
		}
		if remotes[i].Name == "origin" {
			return true
		}
		if remotes[j].Name == "origin" {
			return false
		}
		return remotes[i].Name < remotes[j].Name
	})

	return remotes, nil
}

func normaliseGitHubRepoPath(path string) (string, error) {
	path = core.Trim(path)
	path = trimSlashes(path)
	path = core.TrimSuffix(path, ".git")
	path = trimSlashes(path)
	if path == "" {
		return "", coreerr.E("github.parseGitHubRepo", "not a GitHub URL: "+path, nil)
	}

	parts := core.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", coreerr.E("github.parseGitHubRepo", "not a GitHub URL: "+path, nil)
	}

	return parts[0] + "/" + parts[1], nil
}

func splitRemoteFields(line string) []string {
	fields := make([]string, 0, 3)
	start := -1
	for index, r := range line {
		if isRemoteFieldSeparator(r) {
			if start >= 0 {
				fields = append(fields, line[start:index])
				start = -1
			}
			continue
		}
		if start < 0 {
			start = index
		}
	}
	if start >= 0 {
		fields = append(fields, line[start:])
	}
	return fields
}

func isRemoteFieldSeparator(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	default:
		return false
	}
}

func trimSlashes(path string) string {
	for core.HasPrefix(path, "/") {
		path = core.TrimPrefix(path, "/")
	}
	for core.HasSuffix(path, "/") {
		path = core.TrimSuffix(path, "/")
	}
	return path
}

func urlpkgParse(rawURL string) (*url.URL, error) {
	return url.Parse(rawURL)
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
