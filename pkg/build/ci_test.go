package build

import (
	"context"
	"encoding/json"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setenvCI sets the GitHub Actions environment variables for a test and cleans up afterwards.
func setenvCI(t *testing.T, sha, ref, repo string) {
	t.Helper()
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("GITHUB_SHA", sha)
	t.Setenv("GITHUB_REF", ref)
	t.Setenv("GITHUB_REPOSITORY", repo)
}

func initGitMetadataRepo(t *testing.T) (string, string) {
	t.Helper()

	dir := t.TempDir()
	ctx := context.Background()

	require.NoError(t, ax.ExecDir(ctx, dir, "git", "init", "-b", "main"))
	require.NoError(t, ax.ExecDir(ctx, dir, "git", "config", "user.email", "codex@example.com"))
	require.NoError(t, ax.ExecDir(ctx, dir, "git", "config", "user.name", "Codex"))
	require.NoError(t, ax.WriteFile(ax.Join(dir, "README.md"), []byte("# demo\n"), 0o644))
	require.NoError(t, ax.ExecDir(ctx, dir, "git", "add", "README.md"))
	require.NoError(t, ax.ExecDir(ctx, dir, "git", "commit", "-m", "init"))
	require.NoError(t, ax.ExecDir(ctx, dir, "git", "remote", "add", "origin", "git@github.com:dappcore/core.git"))

	sha, err := ax.RunDir(ctx, dir, "git", "rev-parse", "HEAD")
	require.NoError(t, err)

	return dir, sha
}

func TestCi_FormatGitHubAnnotation_Good(t *testing.T) {
	t.Run("formats error annotation correctly", func(t *testing.T) {
		s := FormatGitHubAnnotation("error", "main.go", 42, "undefined: foo")
		assert.Equal(t, "::error file=main.go,line=42::undefined: foo", s)
	})

	t.Run("formats warning annotation correctly", func(t *testing.T) {
		s := FormatGitHubAnnotation("warning", "pkg/build/ci.go", 10, "unused import")
		assert.Equal(t, "::warning file=pkg/build/ci.go,line=10::unused import", s)
	})

	t.Run("formats notice annotation correctly", func(t *testing.T) {
		s := FormatGitHubAnnotation("notice", "cmd/main.go", 1, "build started")
		assert.Equal(t, "::notice file=cmd/main.go,line=1::build started", s)
	})

	t.Run("uses correct line numbers", func(t *testing.T) {
		s := FormatGitHubAnnotation("error", "file.go", 99, "msg")
		assert.Contains(t, s, "line=99")
	})
}

func TestCi_FormatGitHubAnnotation_Bad(t *testing.T) {
	t.Run("empty file produces empty file field", func(t *testing.T) {
		s := FormatGitHubAnnotation("error", "", 1, "message")
		assert.Equal(t, "::error file=,line=1::message", s)
	})

	t.Run("empty level still produces annotation format", func(t *testing.T) {
		s := FormatGitHubAnnotation("", "main.go", 1, "message")
		assert.Equal(t, ":: file=main.go,line=1::message", s)
	})

	t.Run("empty message produces empty message section", func(t *testing.T) {
		s := FormatGitHubAnnotation("error", "main.go", 1, "")
		assert.Equal(t, "::error file=main.go,line=1::", s)
	})

	t.Run("line zero is valid", func(t *testing.T) {
		s := FormatGitHubAnnotation("error", "main.go", 0, "msg")
		assert.Contains(t, s, "line=0")
	})
}

func TestCi_FormatGitHubAnnotation_Ugly(t *testing.T) {
	t.Run("message with newline is escaped", func(t *testing.T) {
		s := FormatGitHubAnnotation("error", "main.go", 1, "line one\nline two")
		assert.Contains(t, s, "line one%0Aline two")
	})

	t.Run("message with colons does not break format", func(t *testing.T) {
		s := FormatGitHubAnnotation("error", "main.go", 1, "error: something::bad")
		// The leading ::level file=... part should still be present
		assert.Contains(t, s, "::error file=main.go,line=1::")
		assert.Contains(t, s, "error: something::bad")
	})

	t.Run("file path with spaces is included as-is", func(t *testing.T) {
		s := FormatGitHubAnnotation("warning", "my file.go", 5, "msg")
		assert.Contains(t, s, "file=my file.go")
	})

	t.Run("unicode message is preserved", func(t *testing.T) {
		s := FormatGitHubAnnotation("error", "main.go", 1, "résumé: 日本語")
		assert.Contains(t, s, "résumé: 日本語")
	})

	t.Run("percent characters are escaped for GitHub annotations", func(t *testing.T) {
		s := FormatGitHubAnnotation("error", "main.go", 1, "100% done")
		assert.Contains(t, s, "100%25 done")
	})
}

func TestCi_DetectCI_Good(t *testing.T) {
	t.Run("detects tag ref", func(t *testing.T) {
		setenvCI(t, "abc1234def5678901234567890123456789012345", "refs/tags/v1.2.3", "dappcore/core")

		ci := DetectCI()
		require.NotNil(t, ci)
		assert.Equal(t, "refs/tags/v1.2.3", ci.Ref)
		assert.Equal(t, "abc1234def5678901234567890123456789012345", ci.SHA)
		assert.Equal(t, "abc1234", ci.ShortSHA)
		assert.Equal(t, "v1.2.3", ci.Tag)
		assert.True(t, ci.IsTag)
		assert.Equal(t, "", ci.Branch)
		assert.Equal(t, "dappcore/core", ci.Repo)
		assert.Equal(t, "dappcore", ci.Owner)
	})

	t.Run("detects branch ref", func(t *testing.T) {
		setenvCI(t, "deadbeef1234567890123456789012345678abcd", "refs/heads/main", "org/repo")

		ci := DetectCI()
		require.NotNil(t, ci)
		assert.Equal(t, "main", ci.Branch)
		assert.False(t, ci.IsTag)
		assert.Equal(t, "", ci.Tag)
		assert.Equal(t, "deadbee", ci.ShortSHA)
	})

	t.Run("owner is derived from repo", func(t *testing.T) {
		setenvCI(t, "aaaaaaaaaaaaaaaa", "refs/heads/dev", "myorg/myrepo")

		ci := DetectCI()
		require.NotNil(t, ci)
		assert.Equal(t, "myorg", ci.Owner)
		assert.Equal(t, "myorg/myrepo", ci.Repo)
	})
}

func TestCi_DetectCI_Bad(t *testing.T) {
	t.Run("returns nil when GITHUB_ACTIONS is not set", func(t *testing.T) {
		t.Setenv("GITHUB_ACTIONS", "")
		t.Setenv("GITHUB_SHA", "abc1234def5678901234567890123456789012345")
		t.Setenv("GITHUB_REF", "refs/heads/main")
		t.Setenv("GITHUB_REPOSITORY", "org/repo")

		ci := DetectCI()
		assert.Nil(t, ci)
	})

	t.Run("returns nil when GITHUB_SHA is not set", func(t *testing.T) {
		t.Setenv("GITHUB_ACTIONS", "true")
		t.Setenv("GITHUB_SHA", "")
		t.Setenv("GITHUB_REF", "")
		t.Setenv("GITHUB_REPOSITORY", "")

		ci := DetectCI()
		assert.Nil(t, ci)
	})
}

func TestCi_DetectGitHubMetadata_Good(t *testing.T) {
	t.Run("detects GitHub metadata without GITHUB_ACTIONS", func(t *testing.T) {
		t.Setenv("GITHUB_ACTIONS", "")
		t.Setenv("GITHUB_SHA", "abc1234def5678901234567890123456789012345")
		t.Setenv("GITHUB_REF", "refs/heads/main")
		t.Setenv("GITHUB_REPOSITORY", "org/repo")

		ci := DetectGitHubMetadata()
		require.NotNil(t, ci)
		assert.Equal(t, "abc1234", ci.ShortSHA)
		assert.Equal(t, "main", ci.Branch)
		assert.Equal(t, "org/repo", ci.Repo)
		assert.Equal(t, "org", ci.Owner)
	})
}

func TestCi_detectLocalGitMetadata_Good(t *testing.T) {
	t.Run("detects branch metadata from local git repository", func(t *testing.T) {
		dir, sha := initGitMetadataRepo(t)

		ci := detectLocalGitMetadata(dir)
		require.NotNil(t, ci)
		assert.Equal(t, sha, ci.SHA)
		assert.Equal(t, sha[:7], ci.ShortSHA)
		assert.Equal(t, "refs/heads/main", ci.Ref)
		assert.Equal(t, "main", ci.Branch)
		assert.False(t, ci.IsTag)
		assert.Equal(t, "", ci.Tag)
		assert.Equal(t, "dappcore/core", ci.Repo)
		assert.Equal(t, "dappcore", ci.Owner)
	})

	t.Run("prefers exact tag metadata when HEAD is tagged", func(t *testing.T) {
		dir, sha := initGitMetadataRepo(t)
		require.NoError(t, ax.ExecDir(context.Background(), dir, "git", "tag", "v1.2.3"))

		ci := detectLocalGitMetadata(dir)
		require.NotNil(t, ci)
		assert.Equal(t, sha, ci.SHA)
		assert.Equal(t, "refs/tags/v1.2.3", ci.Ref)
		assert.True(t, ci.IsTag)
		assert.Equal(t, "v1.2.3", ci.Tag)
		assert.Equal(t, "", ci.Branch)
	})
}

func TestCi_detectLocalGitMetadata_Bad(t *testing.T) {
	t.Run("returns nil outside a git repository", func(t *testing.T) {
		assert.Nil(t, detectLocalGitMetadata(t.TempDir()))
	})
}

func TestCi_DetectCI_Ugly(t *testing.T) {
	t.Run("SHA shorter than 7 chars still works", func(t *testing.T) {
		setenvCI(t, "abc", "refs/heads/main", "org/repo")

		ci := DetectCI()
		require.NotNil(t, ci)
		assert.Equal(t, "abc", ci.ShortSHA)
	})

	t.Run("ref with unknown prefix leaves tag and branch empty", func(t *testing.T) {
		setenvCI(t, "abc1234def5678", "refs/pull/42/merge", "org/repo")

		ci := DetectCI()
		require.NotNil(t, ci)
		assert.Equal(t, "", ci.Tag)
		assert.Equal(t, "", ci.Branch)
		assert.False(t, ci.IsTag)
	})

	t.Run("repo without slash leaves owner empty", func(t *testing.T) {
		setenvCI(t, "abc1234def5678", "refs/heads/main", "noslashrepo")

		ci := DetectCI()
		require.NotNil(t, ci)
		assert.Equal(t, "", ci.Owner)
		assert.Equal(t, "noslashrepo", ci.Repo)
	})

	t.Run("empty repo is tolerated", func(t *testing.T) {
		setenvCI(t, "abc1234def5678", "refs/heads/main", "")

		ci := DetectCI()
		require.NotNil(t, ci)
		assert.Equal(t, "", ci.Owner)
		assert.Equal(t, "", ci.Repo)
	})
}

func TestCi_ArtifactName_Good(t *testing.T) {
	t.Run("uses tag when IsTag is true", func(t *testing.T) {
		ci := &CIContext{
			IsTag:    true,
			Tag:      "v1.2.3",
			ShortSHA: "abc1234",
		}
		name := ArtifactName("core", ci, Target{OS: "linux", Arch: "amd64"})
		assert.Equal(t, "core_linux_amd64_v1.2.3", name)
	})

	t.Run("uses ShortSHA when not a tag", func(t *testing.T) {
		ci := &CIContext{
			IsTag:    false,
			ShortSHA: "abc1234",
		}
		name := ArtifactName("myapp", ci, Target{OS: "darwin", Arch: "arm64"})
		assert.Equal(t, "myapp_darwin_arm64_abc1234", name)
	})

	t.Run("produces correct format for windows", func(t *testing.T) {
		ci := &CIContext{IsTag: true, Tag: "v2.0.0", ShortSHA: "ff00ff0"}
		name := ArtifactName("core", ci, Target{OS: "windows", Arch: "amd64"})
		assert.Equal(t, "core_windows_amd64_v2.0.0", name)
	})
}

func TestCi_ArtifactName_Bad(t *testing.T) {
	t.Run("nil ci returns name_os_arch only", func(t *testing.T) {
		name := ArtifactName("core", nil, Target{OS: "linux", Arch: "amd64"})
		assert.Equal(t, "core_linux_amd64", name)
	})

	t.Run("ci with no tag and no SHA returns name_os_arch only", func(t *testing.T) {
		ci := &CIContext{IsTag: false, ShortSHA: "", Tag: ""}
		name := ArtifactName("core", ci, Target{OS: "linux", Arch: "amd64"})
		assert.Equal(t, "core_linux_amd64", name)
	})
}

func TestCi_ArtifactName_Ugly(t *testing.T) {
	t.Run("empty build name produces leading underscore segments", func(t *testing.T) {
		ci := &CIContext{IsTag: true, Tag: "v1.0.0", ShortSHA: "abc1234"}
		name := ArtifactName("", ci, Target{OS: "linux", Arch: "amd64"})
		// Empty name results in "_linux_amd64_v1.0.0"
		assert.Contains(t, name, "linux_amd64_v1.0.0")
	})

	t.Run("IsTag true but empty tag falls back to ShortSHA", func(t *testing.T) {
		ci := &CIContext{IsTag: true, Tag: "", ShortSHA: "abc1234"}
		name := ArtifactName("core", ci, Target{OS: "linux", Arch: "amd64"})
		assert.Equal(t, "core_linux_amd64_abc1234", name)
	})

	t.Run("special chars in build name are preserved", func(t *testing.T) {
		ci := &CIContext{IsTag: true, Tag: "v1.0.0"}
		name := ArtifactName("core-build", ci, Target{OS: "linux", Arch: "amd64"})
		assert.Equal(t, "core-build_linux_amd64_v1.0.0", name)
	})
}

func TestCi_WriteArtifactMeta_Good(t *testing.T) {
	fs := io.Local

	t.Run("writes valid JSON with CI context", func(t *testing.T) {
		dir := t.TempDir()
		path := ax.Join(dir, "artifact_meta.json")

		ci := &CIContext{
			Ref:      "refs/tags/v1.2.3",
			SHA:      "abc1234def5678",
			ShortSHA: "abc1234",
			Tag:      "v1.2.3",
			IsTag:    true,
			Repo:     "dappcore/core",
			Owner:    "dappcore",
		}

		err := WriteArtifactMeta(fs, path, "core", Target{OS: "linux", Arch: "amd64"}, ci)
		require.NoError(t, err)

		content, readErr := ax.ReadFile(path)
		require.NoError(t, readErr)

		var meta map[string]any
		require.NoError(t, json.Unmarshal(content, &meta))

		assert.Equal(t, "core", meta["name"])
		assert.Equal(t, "linux", meta["os"])
		assert.Equal(t, "amd64", meta["arch"])
		assert.Equal(t, "v1.2.3", meta["tag"])
		assert.Equal(t, true, meta["is_tag"])
		assert.Equal(t, "dappcore/core", meta["repo"])
		assert.Equal(t, "refs/tags/v1.2.3", meta["ref"])
	})

	t.Run("writes valid JSON without CI context", func(t *testing.T) {
		dir := t.TempDir()
		path := ax.Join(dir, "artifact_meta.json")

		err := WriteArtifactMeta(fs, path, "myapp", Target{OS: "darwin", Arch: "arm64"}, nil)
		require.NoError(t, err)

		content, readErr := ax.ReadFile(path)
		require.NoError(t, readErr)

		var meta map[string]any
		require.NoError(t, json.Unmarshal(content, &meta))

		assert.Equal(t, "myapp", meta["name"])
		assert.Equal(t, "darwin", meta["os"])
		assert.Equal(t, "arm64", meta["arch"])
		assert.Equal(t, false, meta["is_tag"])
	})

	t.Run("output is pretty-printed JSON", func(t *testing.T) {
		dir := t.TempDir()
		path := ax.Join(dir, "artifact_meta.json")

		err := WriteArtifactMeta(fs, path, "core", Target{OS: "windows", Arch: "amd64"}, nil)
		require.NoError(t, err)

		content, readErr := ax.ReadFile(path)
		require.NoError(t, readErr)

		// Pretty-printed JSON contains indentation
		assert.Contains(t, string(content), "\n")
		assert.Contains(t, string(content), "  ")
	})
}

func TestCi_CIArtifactPath_Good(t *testing.T) {
	t.Run("stamps tar.gz artifacts with tag names", func(t *testing.T) {
		ci := &CIContext{
			IsTag:    true,
			Tag:      "v1.2.3",
			ShortSHA: "abc1234",
		}

		path := CIArtifactPath("core", ci, Artifact{
			Path: "/tmp/dist/linux_amd64/core.tar.gz",
			OS:   "linux",
			Arch: "amd64",
		})

		assert.Equal(t, "/tmp/dist/linux_amd64/core_linux_amd64_v1.2.3.tar.gz", path)
	})

	t.Run("stamps app bundles without losing the bundle suffix", func(t *testing.T) {
		ci := &CIContext{
			IsTag:    false,
			ShortSHA: "abc1234",
		}

		path := CIArtifactPath("core", ci, Artifact{
			Path: "/tmp/dist/darwin_arm64/Core.app",
			OS:   "darwin",
			Arch: "arm64",
		})

		assert.Equal(t, "/tmp/dist/darwin_arm64/core_darwin_arm64_abc1234.app", path)
	})

	t.Run("returns the original path when CI metadata is unavailable", func(t *testing.T) {
		artifact := Artifact{
			Path: "/tmp/dist/linux_amd64/core",
			OS:   "linux",
			Arch: "amd64",
		}

		assert.Equal(t, artifact.Path, CIArtifactPath("core", nil, artifact))
	})
}
