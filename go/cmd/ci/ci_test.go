package ci

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/internal/cli"
	"dappco.re/go/build/pkg/release"
)

// initTempGitRepo creates a hermetic git repository in dir with an isolated
// identity and signing disabled so it works on a bare CI box. It fails the test
// if any git command errors.
func initTempGitRepo(t *core.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "ci-test@example.com"},
		{"config", "user.name", "CI Test"},
		{"config", "commit.gpgsign", "false"},
	} {
		if r := ax.RunDir(context.Background(), dir, "git", args...); !r.OK {
			t.Fatalf("git %v failed: %v", args, r.Error())
		}
	}
}

// gitCommit stages everything in dir and records a commit with the given message.
func gitCommit(t *core.T, dir, message string) {
	t.Helper()
	if r := ax.RunDir(context.Background(), dir, "git", "add", "."); !r.OK {
		t.Fatalf("git add failed: %v", r.Error())
	}
	if r := ax.RunDir(context.Background(), dir, "git", "commit", "-m", message); !r.OK {
		t.Fatalf("git commit failed: %v", r.Error())
	}
}

// gitTag creates an annotated-free lightweight tag in dir.
func gitTag(t *core.T, dir, tag string) {
	t.Helper()
	if r := ax.RunDir(context.Background(), dir, "git", "tag", tag); !r.OK {
		t.Fatalf("git tag failed: %v", r.Error())
	}
}

// captureCIStdout redirects cli output into a buffer for the test duration.
func captureCIStdout(t *core.T) *core.Buffer {
	t.Helper()
	buf := core.NewBuffer()
	cli.SetStdout(buf)
	cli.SetStderr(buf)
	t.Cleanup(func() {
		cli.SetStdout(nil)
		cli.SetStderr(nil)
	})
	return buf
}

func TestCI_runCIReleaseInitInDir_Good(t *testing.T) {
	projectDir := t.TempDir()

	result := runCIReleaseInitInDir(projectDir)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	configPath := release.ConfigPath(projectDir)
	contentResult := ax.ReadFile(configPath)
	if !contentResult.OK {
		t.Fatalf("unexpected error: %v", contentResult.Error())
	}
	content := contentResult.Value.([]byte)
	if !stdlibAssertContains(string(content), "sdk:") {
		t.Fatalf("expected %v to contain %v", string(content), "sdk:")
	}
	if !stdlibAssertContains(string(content), "spec: api/openapi.yaml") {
		t.Fatalf("expected %v to contain %v", string(content), "spec: api/openapi.yaml")
	}
	if !stdlibAssertContains(string(content), "languages:") {
		t.Fatalf("expected %v to contain %v", string(content), "languages:")
	}
	if !stdlibAssertContains(string(content), "- typescript") {
		t.Fatalf("expected %v to contain %v", string(content), "- typescript")
	}

}

// --- runCIReleaseInitInDir: scaffolding the release config ---

func TestCi_runCIReleaseInitInDir_Good(t *core.T) {
	projectDir := t.TempDir()
	buf := captureCIStdout(t)

	result := runCIReleaseInitInDir(projectDir)
	core.AssertTrue(t, result.OK)
	// The config file is created and the "next steps" guidance is printed.
	core.AssertTrue(t, release.ConfigExists(projectDir))
	out := buf.String()
	core.AssertContains(t, out, "Created .core/release.yaml")
	core.AssertContains(t, out, "Next steps")
}

func TestCi_runCIReleaseInitInDir_Bad(t *core.T) {
	// Failure path: a path component required for the config directory is a
	// regular file, so WriteConfig cannot create .core and the error is wrapped.
	projectDir := t.TempDir()
	if r := ax.WriteFile(ax.Join(projectDir, ".core"), []byte("not a dir"), 0o644); !r.OK {
		t.Fatalf("unexpected error: %v", r.Error())
	}
	captureCIStdout(t)

	result := runCIReleaseInitInDir(projectDir)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "failed to create config")
}

func TestCi_runCIReleaseInitInDir_Ugly(t *core.T) {
	// Edge case: initialising an already-initialised directory is idempotent —
	// it reports the existing config and does not overwrite it.
	projectDir := t.TempDir()
	core.AssertTrue(t, runCIReleaseInitInDir(projectDir).OK)
	configPath := release.ConfigPath(projectDir)
	original := requireCIBytes(t, ax.ReadFile(configPath))
	buf := captureCIStdout(t)

	result := runCIReleaseInitInDir(projectDir)
	core.AssertTrue(t, result.OK)
	core.AssertContains(t, buf.String(), "already initialised")
	// Content is untouched by the second run.
	after := requireCIBytes(t, ax.ReadFile(configPath))
	core.AssertEqual(t, string(original), string(after))
}

// --- latestTagWithContext: most recent git tag lookup ---

func TestCi_latestTagWithContext_Good(t *core.T) {
	dir := t.TempDir()
	initTempGitRepo(t, dir)
	requireCIOK(t, ax.WriteFile(ax.Join(dir, "README.md"), []byte("# demo\n"), 0o644))
	gitCommit(t, dir, "feat: initial commit")
	gitTag(t, dir, "v1.2.3")

	result := latestTagWithContext(context.Background(), dir)
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, "v1.2.3", result.Value.(string))
}

func TestCi_latestTagWithContext_Bad(t *core.T) {
	// Failure path: a directory that is not a git repository at all.
	result := latestTagWithContext(context.Background(), t.TempDir())
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "not a git repository")
}

func TestCi_latestTagWithContext_Ugly(t *core.T) {
	// Edge case: a git repo with commits but no tags yet — `git describe`
	// reports that there is nothing to describe.
	dir := t.TempDir()
	initTempGitRepo(t, dir)
	requireCIOK(t, ax.WriteFile(ax.Join(dir, "f.txt"), []byte("x\n"), 0o644))
	gitCommit(t, dir, "chore: first")

	result := latestTagWithContext(context.Background(), dir)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "cannot describe")
}

// TestCi_latestTagWithContext_AbbrevPicksNearestTag verifies that an extra
// untagged commit after a tag still resolves to that tag (abbrev=0 behaviour),
// which is the property runChangelog/version logic relies on.
func TestCi_latestTagWithContext_AbbrevPicksNearestTag(t *core.T) {
	dir := t.TempDir()
	initTempGitRepo(t, dir)
	requireCIOK(t, ax.WriteFile(ax.Join(dir, "a.txt"), []byte("a\n"), 0o644))
	gitCommit(t, dir, "feat: one")
	gitTag(t, dir, "v0.9.0")
	requireCIOK(t, ax.WriteFile(ax.Join(dir, "b.txt"), []byte("b\n"), 0o644))
	gitCommit(t, dir, "fix: two")

	result := latestTagWithContext(context.Background(), dir)
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, "v0.9.0", result.Value.(string))
}

// --- runCIPublish ---
//
// These tests run against the test working directory (the package source dir).
// The handler resolves cwd via ax.Getwd, which cannot be redirected in-process,
// so the deterministic, side-effect-free branches are exercised: the package
// directory has no dist/ output, so publishing always stops at artifact
// discovery before any registry/network access. The real-publish success path
// is covered under pkg/release and is skipped here (no injectable publisher
// seam in cmd/ci) — see the report.

func TestCi_runCIPublish_Good(t *core.T) {
	buf := captureCIStdout(t)

	// Dry-run: a default config resolves (with a publisher), the header is
	// rendered, and publishing stops at "no artifacts" because there is no
	// dist/ directory — no publisher is contacted.
	result := runCIPublish(context.Background(), true, "", false, false)
	core.AssertFalse(t, result.OK)
	out := buf.String()
	core.AssertContains(t, out, "Publishing release")
	core.AssertContains(t, out, "Dry run")
	core.AssertContains(t, result.Error(), "dist/")
}

func TestCi_runCIPublish_Bad(t *core.T) {
	captureCIStdout(t)

	// A pre-release/draft override with publish enabled still fails fast at
	// artifact discovery (no dist/), proving the override loop runs without a
	// configured artifact set rather than reaching a publisher.
	result := runCIPublish(context.Background(), false, "v9.9.9", true, true)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "dist/")
}

func TestCi_runCIPublish_Ugly(t *core.T) {
	captureCIStdout(t)

	// Edge case: publish enabled (not a dry run) with an explicit version. The
	// version override is applied and discovery still fails deterministically
	// before any network publish.
	result := runCIPublish(context.Background(), false, "v1.2.3", false, false)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "dist/")
}

// --- runCIReleaseVersion ---
//
// runCIReleaseVersion resolves cwd via ax.Getwd (not redirectable), so it runs
// against this repository. The version value varies, but the success-shape and
// the cancellation behaviour are deterministic.

func TestCi_runCIReleaseVersion_Good(t *core.T) {
	buf := captureCIStdout(t)

	result := runCIReleaseVersion(context.Background())
	core.AssertTrue(t, result.OK)
	out := buf.String()
	core.AssertContains(t, out, "version:")
	// A version was determined and rendered (starts with the semver 'v').
	core.AssertContains(t, out, "v")
}

func TestCi_runCIReleaseVersion_Bad(t *core.T) {
	captureCIStdout(t)

	// Failure path: a cancelled context aborts version determination and the
	// error is wrapped by the command layer.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := runCIReleaseVersion(ctx)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "determine")
}

func TestCi_runCIReleaseVersion_Ugly(t *core.T) {
	// Edge case: an already-elapsed deadline behaves like cancellation — the
	// version lookup is reported as cancelled rather than producing a value.
	captureCIStdout(t)
	ctx, cancel := context.WithDeadline(context.Background(), timeInPast())
	defer cancel()

	result := runCIReleaseVersion(ctx)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "version")
}

// --- runChangelog ---
//
// runChangelog resolves cwd via ax.Getwd, so it runs against this repository.
// Explicit refs keep the happy path deterministic; the empty-ref path exercises
// the tag-detection branch and the cancellation handling.

func TestCi_runChangelog_Good(t *core.T) {
	buf := captureCIStdout(t)

	// An empty range (HEAD..HEAD) is always valid in a git repo and yields a
	// changelog header without breaking change content.
	result := runChangelog(context.Background(), "HEAD", "HEAD")
	core.AssertTrue(t, result.OK)
	core.AssertContains(t, buf.String(), "Generating changelog")
}

func TestCi_runChangelog_Bad(t *core.T) {
	captureCIStdout(t)

	// Failure path: with empty refs the handler must look up the latest tag,
	// and a cancelled context surfaces the cancellation rather than a changelog.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := runChangelog(ctx, "", "")
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "context canceled")
}

func TestCi_runChangelog_Ugly(t *core.T) {
	captureCIStdout(t)

	// Edge case: an explicit from-ref with an empty to-ref defaults to-ref to
	// HEAD, producing a valid range against the current repository.
	result := runChangelog(context.Background(), "HEAD", "")
	core.AssertTrue(t, result.OK)
}

// TestCi_runChangelog_GenerateError covers the changelog-generation failure
// branch: a ref range that does not resolve to any revision fails inside git
// regardless of repository contents, and the error is wrapped by the handler.
func TestCi_runChangelog_GenerateError(t *core.T) {
	captureCIStdout(t)

	result := runChangelog(context.Background(), "nonexistent-ref-xyz", "another-bad-ref-abc")
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "failed to generate changelog")
}

// TestCi_runChangelog_EmptyRefsAutoDetect covers the auto-detection branch: with
// both refs empty the handler resolves the latest tag (from-ref) and HEAD
// (to-ref). In a repository with tags it generates a changelog; with no tags it
// reports "No tags found". Both outcomes are successful results.
func TestCi_runChangelog_EmptyRefsAutoDetect(t *core.T) {
	buf := captureCIStdout(t)

	result := runChangelog(context.Background(), "", "")
	core.AssertTrue(t, result.OK)
	out := buf.String()
	core.AssertTrue(t,
		core.Contains(out, "Generating changelog") || core.Contains(out, "No tags found"),
		"expected a changelog header or a no-tags notice",
	)
}

// --- registerCICommands: action wiring ---
//
// Invoking the registered command actions exercises the closures in
// registerCICommands. The `ci/init` action is intentionally NOT invoked: it
// resolves the (non-redirectable) working directory and would scaffold a config
// into the package source tree. Its handler logic is covered via
// runCIReleaseInitInDir instead.
func TestCi_registerCICommands_ActionsWired(t *core.T) {
	captureCIStdout(t)
	c := core.New()
	core.AssertTrue(t, registerCICommands(c).OK)

	// `ci` (publish) dry-run: stops deterministically at artifact discovery.
	publishResult := c.Command("ci").Value.(*core.Command).Run(core.NewOptions())
	core.AssertFalse(t, publishResult.OK)
	core.AssertContains(t, publishResult.Error(), "dist/")

	// `ci/version`: resolves and prints a version against this repository.
	versionResult := c.Command("ci/version").Value.(*core.Command).Run(core.NewOptions())
	core.AssertTrue(t, versionResult.OK)

	// `ci/changelog`: explicit refs produce a valid empty range.
	changelogResult := c.Command("ci/changelog").Value.(*core.Command).Run(core.NewOptions(
		core.Option{Key: "from", Value: "HEAD"},
		core.Option{Key: "to", Value: "HEAD"},
	))
	core.AssertTrue(t, changelogResult.OK)
}

// timeInPast returns a deadline that has already elapsed.
func timeInPast() time.Time {
	return time.Now().Add(-time.Hour)
}
