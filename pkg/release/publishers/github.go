// Package publishers provides release publishing implementations.
package publishers

import (
	"bufio"    // Note: AX-6 — scans git remote output without ad hoc shell parsing.
	"context"  // Note: AX-6 — carries cancellation through GitHub publishing commands.
	stdio "io" // Note: AX-6 — reads artifact streams from Core Medium implementations.
	"io/fs"    // Note: AX-6 — preserves staged artifact file modes.
	"net/url"  // Note: AX-6 — parses GitHub remote URLs using the structured URL parser.
	"sort"     // Note: AX-6 — keeps remote selection deterministic with origin first.

	"dappco.re/go"                          // Note: AX-6 — approved string helpers and Core error joining.
	"dappco.re/go/build/internal/ax"        // Note: AX-6 — Core-backed command, path, JSON, and temp helpers.
	coreio "dappco.re/go/build/pkg/storage" // Note: AX-6 — Core Medium abstraction for artifact filesystem access.
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
func DetectGitHubRepository(ctx context.Context, dir string) core.Result {
	return detectRepository(ctx, dir)
}

// Name returns the publisher's identifier.
//
// name := pub.Name() // → "github"
func (p *GitHubPublisher) Name() string {
	return "github"
}

// Validate checks that the GitHub publisher has a release to publish.
func (p *GitHubPublisher) Validate(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig) core.Result {
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
// result := pub.Publish(ctx, rel, pubCfg, relCfg, false) // dryRun=true to preview
func (p *GitHubPublisher) Publish(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig, dryRun bool) core.Result {
	validated := validatePublisherRelease(p.Name(), release)
	if !validated.OK {
		return validated
	}

	// Determine repository
	repo := ""
	if relCfg != nil {
		repo = relCfg.GetRepository()
	}
	if repo == "" {
		// Try to detect from git remote
		detectedRepoResult := detectRepository(ctx, release.ProjectDir)
		if !detectedRepoResult.OK {
			return core.Fail(core.E("github.Publish", "could not determine repository", core.NewError(detectedRepoResult.Error())))
		}
		repo = detectedRepoResult.Value.(string)
	}

	if dryRun {
		return p.dryRunPublish(release, pubCfg, repo)
	}

	ghCommandResult := resolveGhCli()
	if !ghCommandResult.OK {
		return ghCommandResult
	}
	ghCommand := ghCommandResult.Value.(string)

	// Validate gh CLI is available and authenticated for actual publish
	authenticated := validateGhAuth(ctx, ghCommand)
	if !authenticated.OK {
		return authenticated
	}

	return p.executePublish(ctx, release, pubCfg, repo, ghCommand)
}

// dryRunPublish shows what would be done without actually publishing.
func (p *GitHubPublisher) dryRunPublish(release *Release, pubCfg PublisherConfig, repo string) core.Result {
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

	return core.Ok(nil)
}

// executePublish actually creates the release and uploads artifacts.
func (p *GitHubPublisher) executePublish(ctx context.Context, release *Release, pubCfg PublisherConfig, repo, ghCommand string) core.Result {
	// Build the release create command
	args := p.buildCreateArgs(release, pubCfg, repo)

	materializedResult := p.materializeArtifacts(release)
	if !materializedResult.OK {
		return materializedResult
	}
	materialized := materializedResult.Value.(githubArtifactMaterialization)
	defer materialized.cleanup()

	args = append(args, materialized.paths...)

	// Execute gh release create
	created := publisherRun(ctx, release.ProjectDir, nil, ghCommand, args...)
	if !created.OK {
		return core.Fail(core.E("github.Publish", "gh release create failed", core.NewError(created.Error())))
	}

	return core.Ok(nil)
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

type githubArtifactMaterialization struct {
	paths   []string
	cleanup func()
}

func (p *GitHubPublisher) materializeArtifacts(release *Release) core.Result {
	artifactFS := releaseArtifactFS(release)
	if artifactFS == nil {
		return core.Fail(core.E("github.Publish", "artifact filesystem is nil", nil))
	}

	paths := make([]string, 0, len(release.Artifacts))
	if mediumEquals(artifactFS, coreio.Local) {
		for _, artifact := range release.Artifacts {
			paths = append(paths, artifact.Path)
		}
		return core.Ok(githubArtifactMaterialization{paths: paths, cleanup: func() {}})
	}

	tempDirResult := ax.TempDir("github-release-artifacts-*")
	if !tempDirResult.OK {
		return core.Fail(core.E("github.Publish", "failed to create artifact staging directory", core.NewError(tempDirResult.Error())))
	}
	tempDir := tempDirResult.Value.(string)

	for i, artifact := range release.Artifacts {
		localPath := ax.Join(tempDir, core.Sprintf("%03d", i), ax.Base(artifact.Path))
		copied := copyArtifactPathToLocal(artifactFS, artifact.Path, localPath)
		if !copied.OK {
			cleaned := ax.RemoveAll(tempDir)
			if !cleaned.OK {
				return core.Fail(core.E("github.Publish", "failed to clean up artifact staging directory", core.NewError(cleaned.Error())))
			}
			return core.Fail(core.E("github.Publish", "failed to stage artifact "+artifact.Path, core.NewError(copied.Error())))
		}
		paths = append(paths, localPath)
	}

	return core.Ok(githubArtifactMaterialization{paths: paths, cleanup: func() { ax.RemoveAll(tempDir) }})
}

func copyArtifactPathToLocal(artifactFS coreio.Medium, sourcePath, destinationPath string) core.Result {
	if artifactFS.IsDir(sourcePath) {
		return copyArtifactDirToLocal(artifactFS, sourcePath, destinationPath)
	}

	return copyArtifactFileToLocal(artifactFS, sourcePath, destinationPath)
}

func copyArtifactDirToLocal(artifactFS coreio.Medium, sourcePath, destinationPath string) core.Result {
	created := coreio.Local.EnsureDir(destinationPath)
	if !created.OK {
		return core.Fail(core.E("github.copyArtifactDirToLocal", "failed to create destination directory", core.NewError(created.Error())))
	}

	entriesResult := artifactFS.List(sourcePath)
	if !entriesResult.OK {
		return core.Fail(core.E("github.copyArtifactDirToLocal", "failed to list artifact directory", core.NewError(entriesResult.Error())))
	}
	entries := entriesResult.Value.([]fs.DirEntry)

	for _, entry := range entries {
		childSource := ax.Join(sourcePath, entry.Name())
		childDestination := ax.Join(destinationPath, entry.Name())
		copied := copyArtifactPathToLocal(artifactFS, childSource, childDestination)
		if !copied.OK {
			return copied
		}
	}

	return core.Ok(nil)
}

func copyArtifactFileToLocal(artifactFS coreio.Medium, sourcePath, destinationPath string) core.Result {
	fileResult := artifactFS.Open(sourcePath)
	if !fileResult.OK {
		return core.Fail(core.E("github.copyArtifactFileToLocal", "failed to open artifact", core.NewError(fileResult.Error())))
	}
	file := fileResult.Value.(core.FsFile)
	defer file.Close()

	content, readFailure := stdio.ReadAll(file)
	if readFailure != nil {
		return core.Fail(core.E("github.copyArtifactFileToLocal", "failed to read artifact", readFailure))
	}

	mode := fs.FileMode(0o644)
	infoResult := artifactFS.Stat(sourcePath)
	if infoResult.OK {
		mode = infoResult.Value.(fs.FileInfo).Mode()
	}

	written := coreio.Local.WriteMode(destinationPath, string(content), mode)
	if !written.OK {
		return core.Fail(core.E("github.copyArtifactFileToLocal", "failed to write staged artifact", core.NewError(written.Error())))
	}

	return core.Ok(nil)
}

func resolveGhCli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/gh",
			"/opt/homebrew/bin/gh",
		}
	}

	command := ax.ResolveCommand("gh", paths...)
	if !command.OK {
		return core.Fail(core.E("github.resolveGhCli", "gh CLI not found. Install it from https://cli.github.com", core.NewError(command.Error())))
	}

	return command
}

// validateGhCli checks if the gh CLI is available and authenticated.
func validateGhCli(ctx context.Context) core.Result {
	ghCommandResult := resolveGhCli()
	if !ghCommandResult.OK {
		return ghCommandResult
	}

	return validateGhAuth(ctx, ghCommandResult.Value.(string))
}

func validateGhAuth(ctx context.Context, ghCommand string) core.Result {
	outputResult := ax.CombinedOutput(ctx, "", nil, ghCommand, "auth", "status")
	if !outputResult.OK {
		return core.Fail(core.E("github.validateGhCli", "not authenticated with gh CLI. Run 'gh auth login' first", core.NewError(outputResult.Error())))
	}
	output := outputResult.Value.(string)

	if !core.Contains(output, "Logged in") {
		return core.Fail(core.E("github.validateGhCli", "not authenticated with gh CLI. Run 'gh auth login' first", nil))
	}

	return core.Ok(nil)
}

// detectRepository detects the GitHub repository from git remote.
func detectRepository(ctx context.Context, dir string) core.Result {
	remotesResult := listGitRemotes(ctx, dir)
	if !remotesResult.OK {
		return core.Fail(core.E("github.detectRepository", "failed to list git remotes", core.NewError(remotesResult.Error())))
	}
	remotes := remotesResult.Value.([]gitRemote)
	if len(remotes) == 0 {
		repoResult := detectRepositoryViaGh(ctx, dir)
		if repoResult.OK {
			return repoResult
		}
		return core.Fail(core.E("github.detectRepository", "no git remotes configured", core.NewError(repoResult.Error())))
	}

	var parseFailure error
	for _, remote := range remotes {
		repoResult := parseGitHubRepo(remote.URL)
		if repoResult.OK {
			return repoResult
		}
		if parseFailure == nil {
			parseFailure = core.NewError(repoResult.Error())
		}
	}

	repoResult := detectRepositoryViaGh(ctx, dir)
	if repoResult.OK {
		return repoResult
	}
	if parseFailure == nil {
		parseFailure = core.NewError(repoResult.Error())
	} else {
		parseFailure = core.ErrorJoin(parseFailure, core.NewError(repoResult.Error()))
	}

	return core.Fail(core.E("github.detectRepository", "no GitHub remote found", parseFailure))
}

func detectRepositoryViaGh(ctx context.Context, dir string) core.Result {
	ghCommandResult := resolveGhCli()
	if !ghCommandResult.OK {
		return core.Fail(core.E("github.detectRepositoryViaGh", "gh CLI not available for repository fallback", core.NewError(ghCommandResult.Error())))
	}
	ghCommand := ghCommandResult.Value.(string)

	outputResult := ax.CombinedOutput(ctx, dir, nil, ghCommand, "repo", "view", "--json", "nameWithOwner")
	if !outputResult.OK {
		return core.Fail(core.E("github.detectRepositoryViaGh", "gh repo view failed", core.NewError(outputResult.Error())))
	}
	output := outputResult.Value.(string)

	var payload struct {
		NameWithOwner string `json:"nameWithOwner"`
	}
	decoded := ax.JSONUnmarshal([]byte(output), &payload)
	if !decoded.OK {
		return core.Fail(core.E("github.detectRepositoryViaGh", "failed to parse gh repo view output", core.NewError(decoded.Error())))
	}

	repo := core.Trim(payload.NameWithOwner)
	if repo == "" {
		return core.Fail(core.E("github.detectRepositoryViaGh", "gh repo view did not report a repository", nil))
	}

	return core.Ok(repo)
}

// parseGitHubRepo extracts owner/repo from a GitHub URL.
// Supports:
//   - git@github.com:owner/repo.git
//   - https://github.com/owner/repo.git
//   - https://github.com/owner/repo
func parseGitHubRepo(remoteURL string) core.Result {
	remoteURL = core.Trim(remoteURL)
	if remoteURL == "" {
		return core.Fail(core.E("github.parseGitHubRepo", "not a GitHub URL: "+remoteURL, nil))
	}

	// SSH format
	if core.HasPrefix(remoteURL, "git@github.com:") {
		repo := core.TrimPrefix(remoteURL, "git@github.com:")
		return normaliseGitHubRepoPath(repo)
	}

	parsed, parseFailure := url.Parse(remoteURL)
	if parseFailure == nil && core.Lower(parsed.Hostname()) == "github.com" {
		return normaliseGitHubRepoPath(parsed.Path)
	}

	return core.Fail(core.E("github.parseGitHubRepo", "not a GitHub URL: "+remoteURL, nil))
}

func listGitRemotes(ctx context.Context, dir string) core.Result {
	outputResult := ax.RunDir(ctx, dir, "git", "remote", "-v")
	if !outputResult.OK {
		return outputResult
	}
	output := outputResult.Value.(string)

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
	if scanFailure := scanner.Err(); scanFailure != nil {
		return core.Fail(scanFailure)
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

	return core.Ok(remotes)
}

func normaliseGitHubRepoPath(path string) core.Result {
	path = core.Trim(path)
	path = trimSlashes(path)
	path = core.TrimSuffix(path, ".git")
	path = trimSlashes(path)
	if path == "" {
		return core.Fail(core.E("github.parseGitHubRepo", "not a GitHub URL: "+path, nil))
	}

	parts := core.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return core.Fail(core.E("github.parseGitHubRepo", "not a GitHub URL: "+path, nil))
	}

	return core.Ok(parts[0] + "/" + parts[1])
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

// UploadArtifact uploads a single artifact to an existing release.
// This can be used to add artifacts to a release after creation.
//
// result := publishers.UploadArtifact(ctx, "host-uk/core-build", "v1.2.3", "dist/core-build_v1.2.3_linux_amd64.tar.gz")
func UploadArtifact(ctx context.Context, repo, version, artifactPath string) core.Result {
	ghCommandResult := resolveGhCli()
	if !ghCommandResult.OK {
		return ghCommandResult
	}
	ghCommand := ghCommandResult.Value.(string)

	uploaded := publisherRun(ctx, "", nil, ghCommand, "release", "upload", version, artifactPath, "--repo", repo)
	if !uploaded.OK {
		return core.Fail(core.E("github.UploadArtifact", "failed to upload "+artifactPath, core.NewError(uploaded.Error())))
	}

	return core.Ok(nil)
}

// DeleteRelease deletes a release by tag name.
//
// result := publishers.DeleteRelease(ctx, "host-uk/core-build", "v1.2.3")
func DeleteRelease(ctx context.Context, repo, version string) core.Result {
	ghCommandResult := resolveGhCli()
	if !ghCommandResult.OK {
		return ghCommandResult
	}
	ghCommand := ghCommandResult.Value.(string)

	deleted := publisherRun(ctx, "", nil, ghCommand, "release", "delete", version, "--repo", repo, "--yes")
	if !deleted.OK {
		return core.Fail(core.E("github.DeleteRelease", "failed to delete "+version, core.NewError(deleted.Error())))
	}

	return core.Ok(nil)
}

// ReleaseExists checks if a release exists for the given version.
//
// exists := publishers.ReleaseExists(ctx, "host-uk/core-build", "v1.2.3")
func ReleaseExists(ctx context.Context, repo, version string) bool {
	ghCommandResult := resolveGhCli()
	if !ghCommandResult.OK {
		return false
	}
	ghCommand := ghCommandResult.Value.(string)

	return ax.Exec(ctx, ghCommand, "release", "view", version, "--repo", repo).OK
}
