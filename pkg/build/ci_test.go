package build

import (
	"context"
	"encoding/json"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/core/io"
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
	if err := ax.ExecDir(ctx, dir, "git", "init", "-b", "main"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.ExecDir(ctx, dir, "git", "config", "user.email", "codex@example.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.ExecDir(ctx, dir, "git", "config", "user.name", "Codex"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(dir, "README.md"), []byte("# demo\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.ExecDir(ctx, dir, "git", "add", "README.md"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.ExecDir(ctx, dir, "git", "commit", "-m", "init"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.ExecDir(ctx, dir, "git", "remote", "add", "origin", "git@github.com:dappcore/core.git"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sha, err := ax.RunDir(ctx, dir, "git", "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return dir, sha
}

func TestCi_FormatGitHubAnnotation_Good(t *testing.T) {
	t.Run("formats error annotation correctly", func(t *testing.T) {
		s := FormatGitHubAnnotation("error", "main.go", 42, "undefined: foo")
		if !stdlibAssertEqual("::error file=main.go,line=42::undefined: foo", s) {
			t.Fatalf("want %v, got %v", "::error file=main.go,line=42::undefined: foo", s)
		}

	})

	t.Run("formats warning annotation correctly", func(t *testing.T) {
		s := FormatGitHubAnnotation("warning", "pkg/build/ci.go", 10, "unused import")
		if !stdlibAssertEqual("::warning file=pkg/build/ci.go,line=10::unused import", s) {
			t.Fatalf("want %v, got %v", "::warning file=pkg/build/ci.go,line=10::unused import", s)
		}

	})

	t.Run("formats notice annotation correctly", func(t *testing.T) {
		s := FormatGitHubAnnotation("notice", "cmd/main.go", 1, "build started")
		if !stdlibAssertEqual("::notice file=cmd/main.go,line=1::build started", s) {
			t.Fatalf("want %v, got %v", "::notice file=cmd/main.go,line=1::build started", s)
		}

	})

	t.Run("uses correct line numbers", func(t *testing.T) {
		s := FormatGitHubAnnotation("error", "file.go", 99, "msg")
		if !stdlibAssertContains(s, "line=99") {
			t.Fatalf("expected %v to contain %v", s, "line=99")
		}

	})
}

func TestCi_FormatGitHubAnnotation_Bad(t *testing.T) {
	t.Run("empty file produces empty file field", func(t *testing.T) {
		s := FormatGitHubAnnotation("error", "", 1, "message")
		if !stdlibAssertEqual("::error file=,line=1::message", s) {
			t.Fatalf("want %v, got %v", "::error file=,line=1::message", s)
		}

	})

	t.Run("empty level still produces annotation format", func(t *testing.T) {
		s := FormatGitHubAnnotation("", "main.go", 1, "message")
		if !stdlibAssertEqual(":: file=main.go,line=1::message", s) {
			t.Fatalf("want %v, got %v", ":: file=main.go,line=1::message", s)
		}

	})

	t.Run("empty message produces empty message section", func(t *testing.T) {
		s := FormatGitHubAnnotation("error", "main.go", 1, "")
		if !stdlibAssertEqual("::error file=main.go,line=1::", s) {
			t.Fatalf("want %v, got %v", "::error file=main.go,line=1::", s)
		}

	})

	t.Run("line zero is valid", func(t *testing.T) {
		s := FormatGitHubAnnotation("error", "main.go", 0, "msg")
		if !stdlibAssertContains(s, "line=0") {
			t.Fatalf("expected %v to contain %v", s, "line=0")
		}

	})
}

func TestCi_FormatGitHubAnnotation_Ugly(t *testing.T) {
	t.Run("message with newline is escaped", func(t *testing.T) {
		s := FormatGitHubAnnotation("error", "main.go", 1, "line one\nline two")
		if !stdlibAssertContains(s, "line one%0Aline two") {
			t.Fatalf("expected %v to contain %v", s, "line one%0Aline two")
		}

	})

	t.Run("message with colons does not break format", func(t *testing.T) {
		s := FormatGitHubAnnotation("error", "main.go", 1, "error: something::bad")
		if !stdlibAssertContains(
			// The leading ::level file=... part should still be present
			s, "::error file=main.go,line=1::") {
			t.Fatalf("expected %v to contain %v", s, "::error file=main.go,line=1::")
		}
		if !stdlibAssertContains(s, "error: something::bad") {
			t.Fatalf("expected %v to contain %v", s, "error: something::bad")
		}

	})

	t.Run("file path with spaces is included as-is", func(t *testing.T) {
		s := FormatGitHubAnnotation("warning", "my file.go", 5, "msg")
		if !stdlibAssertContains(s, "file=my file.go") {
			t.Fatalf("expected %v to contain %v", s, "file=my file.go")
		}

	})

	t.Run("unicode message is preserved", func(t *testing.T) {
		s := FormatGitHubAnnotation("error", "main.go", 1, "résumé: 日本語")
		if !stdlibAssertContains(s, "résumé: 日本語") {
			t.Fatalf("expected %v to contain %v", s, "résumé: 日本語")
		}

	})

	t.Run("percent characters are escaped for GitHub annotations", func(t *testing.T) {
		s := FormatGitHubAnnotation("error", "main.go", 1, "100% done")
		if !stdlibAssertContains(s, "100%25 done") {
			t.Fatalf("expected %v to contain %v", s, "100%25 done")
		}

	})
}

func TestCi_DetectCI_Good(t *testing.T) {
	t.Run("detects tag ref", func(t *testing.T) {
		setenvCI(t, "abc1234def5678901234567890123456789012345", "refs/tags/v1.2.3", "dappcore/core")

		ci := DetectCI()
		if stdlibAssertNil(ci) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("refs/tags/v1.2.3", ci.Ref) {
			t.Fatalf("want %v, got %v", "refs/tags/v1.2.3", ci.Ref)
		}
		if !stdlibAssertEqual("abc1234def5678901234567890123456789012345", ci.SHA) {
			t.Fatalf("want %v, got %v", "abc1234def5678901234567890123456789012345", ci.SHA)
		}
		if !stdlibAssertEqual("abc1234", ci.ShortSHA) {
			t.Fatalf("want %v, got %v", "abc1234", ci.ShortSHA)
		}
		if !stdlibAssertEqual("v1.2.3", ci.Tag) {
			t.Fatalf("want %v, got %v", "v1.2.3", ci.Tag)
		}
		if !(ci.IsTag) {
			t.Fatal("expected true")
		}
		if !stdlibAssertEqual("", ci.Branch) {
			t.Fatalf("want %v, got %v", "", ci.Branch)
		}
		if !stdlibAssertEqual("dappcore/core", ci.Repo) {
			t.Fatalf("want %v, got %v", "dappcore/core", ci.Repo)
		}
		if !stdlibAssertEqual("dappcore", ci.Owner) {
			t.Fatalf("want %v, got %v", "dappcore", ci.Owner)
		}

	})

	t.Run("detects branch ref", func(t *testing.T) {
		setenvCI(t, "deadbeef1234567890123456789012345678abcd", "refs/heads/main", "org/repo")

		ci := DetectCI()
		if stdlibAssertNil(ci) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("main", ci.Branch) {
			t.Fatalf("want %v, got %v", "main", ci.Branch)
		}
		if ci.IsTag {
			t.Fatal("expected false")
		}
		if !stdlibAssertEqual("", ci.Tag) {
			t.Fatalf("want %v, got %v", "", ci.Tag)
		}
		if !stdlibAssertEqual("deadbee", ci.ShortSHA) {
			t.Fatalf("want %v, got %v", "deadbee", ci.ShortSHA)
		}

	})

	t.Run("owner is derived from repo", func(t *testing.T) {
		setenvCI(t, "aaaaaaaaaaaaaaaa", "refs/heads/dev", "myorg/myrepo")

		ci := DetectCI()
		if stdlibAssertNil(ci) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("myorg", ci.Owner) {
			t.Fatalf("want %v, got %v", "myorg", ci.Owner)
		}
		if !stdlibAssertEqual("myorg/myrepo", ci.Repo) {
			t.Fatalf("want %v, got %v", "myorg/myrepo", ci.Repo)
		}

	})
}

func TestCi_DetectCI_Bad(t *testing.T) {
	t.Run("returns nil when GITHUB_ACTIONS is not set", func(t *testing.T) {
		t.Setenv("GITHUB_ACTIONS", "")
		t.Setenv("GITHUB_SHA", "abc1234def5678901234567890123456789012345")
		t.Setenv("GITHUB_REF", "refs/heads/main")
		t.Setenv("GITHUB_REPOSITORY", "org/repo")

		ci := DetectCI()
		if !stdlibAssertNil(ci) {
			t.Fatalf("expected nil, got %v", ci)
		}

	})

	t.Run("returns nil when GITHUB_SHA is not set", func(t *testing.T) {
		t.Setenv("GITHUB_ACTIONS", "true")
		t.Setenv("GITHUB_SHA", "")
		t.Setenv("GITHUB_REF", "")
		t.Setenv("GITHUB_REPOSITORY", "")

		ci := DetectCI()
		if !stdlibAssertNil(ci) {
			t.Fatalf("expected nil, got %v", ci)
		}

	})
}

func TestCi_DetectGitHubMetadata_Good(t *testing.T) {
	t.Run("detects GitHub metadata without GITHUB_ACTIONS", func(t *testing.T) {
		t.Setenv("GITHUB_ACTIONS", "")
		t.Setenv("GITHUB_SHA", "abc1234def5678901234567890123456789012345")
		t.Setenv("GITHUB_REF", "refs/heads/main")
		t.Setenv("GITHUB_REPOSITORY", "org/repo")

		ci := DetectGitHubMetadata()
		if stdlibAssertNil(ci) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("abc1234", ci.ShortSHA) {
			t.Fatalf("want %v, got %v", "abc1234", ci.ShortSHA)
		}
		if !stdlibAssertEqual("main", ci.Branch) {
			t.Fatalf("want %v, got %v", "main", ci.Branch)
		}
		if !stdlibAssertEqual("org/repo", ci.Repo) {
			t.Fatalf("want %v, got %v", "org/repo", ci.Repo)
		}
		if !stdlibAssertEqual("org", ci.Owner) {
			t.Fatalf("want %v, got %v", "org", ci.Owner)
		}

	})
}

func TestCi_detectLocalGitMetadata_Good(t *testing.T) {
	t.Run("detects branch metadata from local git repository", func(t *testing.T) {
		dir, sha := initGitMetadataRepo(t)

		ci := detectLocalGitMetadata(dir)
		if stdlibAssertNil(ci) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual(sha, ci.SHA) {
			t.Fatalf("want %v, got %v", sha, ci.SHA)
		}
		if !stdlibAssertEqual(sha[:7], ci.ShortSHA) {
			t.Fatalf("want %v, got %v", sha[:7], ci.ShortSHA)
		}
		if !stdlibAssertEqual("refs/heads/main", ci.Ref) {
			t.Fatalf("want %v, got %v", "refs/heads/main", ci.Ref)
		}
		if !stdlibAssertEqual("main", ci.Branch) {
			t.Fatalf("want %v, got %v", "main", ci.Branch)
		}
		if ci.IsTag {
			t.Fatal("expected false")
		}
		if !stdlibAssertEqual("", ci.Tag) {
			t.Fatalf("want %v, got %v", "", ci.Tag)
		}
		if !stdlibAssertEqual("dappcore/core", ci.Repo) {
			t.Fatalf("want %v, got %v", "dappcore/core", ci.Repo)
		}
		if !stdlibAssertEqual("dappcore", ci.Owner) {
			t.Fatalf("want %v, got %v", "dappcore", ci.Owner)
		}

	})

	t.Run("prefers exact tag metadata when HEAD is tagged", func(t *testing.T) {
		dir, sha := initGitMetadataRepo(t)
		if err := ax.ExecDir(context.Background(), dir, "git", "tag", "v1.2.3"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		ci := detectLocalGitMetadata(dir)
		if stdlibAssertNil(ci) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual(sha, ci.SHA) {
			t.Fatalf("want %v, got %v", sha, ci.SHA)
		}
		if !stdlibAssertEqual("refs/tags/v1.2.3", ci.Ref) {
			t.Fatalf("want %v, got %v", "refs/tags/v1.2.3", ci.Ref)
		}
		if !(ci.IsTag) {
			t.Fatal("expected true")
		}
		if !stdlibAssertEqual("v1.2.3", ci.Tag) {
			t.Fatalf("want %v, got %v", "v1.2.3", ci.Tag)
		}
		if !stdlibAssertEqual("", ci.Branch) {
			t.Fatalf("want %v, got %v", "", ci.Branch)
		}

	})
}

func TestCi_detectLocalGitMetadata_Bad(t *testing.T) {
	t.Run("returns nil outside a git repository", func(t *testing.T) {
		if !stdlibAssertNil(detectLocalGitMetadata(t.TempDir())) {
			t.Fatalf("expected nil, got %v", detectLocalGitMetadata(t.TempDir()))
		}

	})
}

func TestCi_DetectCI_Ugly(t *testing.T) {
	t.Run("SHA shorter than 7 chars still works", func(t *testing.T) {
		setenvCI(t, "abc", "refs/heads/main", "org/repo")

		ci := DetectCI()
		if stdlibAssertNil(ci) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("abc", ci.ShortSHA) {
			t.Fatalf("want %v, got %v", "abc", ci.ShortSHA)
		}

	})

	t.Run("ref with unknown prefix leaves tag and branch empty", func(t *testing.T) {
		setenvCI(t, "abc1234def5678", "refs/pull/42/merge", "org/repo")

		ci := DetectCI()
		if stdlibAssertNil(ci) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("", ci.Tag) {
			t.Fatalf("want %v, got %v", "", ci.Tag)
		}
		if !stdlibAssertEqual("", ci.Branch) {
			t.Fatalf("want %v, got %v", "", ci.Branch)
		}
		if ci.IsTag {
			t.Fatal("expected false")
		}

	})

	t.Run("repo without slash leaves owner empty", func(t *testing.T) {
		setenvCI(t, "abc1234def5678", "refs/heads/main", "noslashrepo")

		ci := DetectCI()
		if stdlibAssertNil(ci) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("", ci.Owner) {
			t.Fatalf("want %v, got %v", "", ci.Owner)
		}
		if !stdlibAssertEqual("noslashrepo", ci.Repo) {
			t.Fatalf("want %v, got %v", "noslashrepo", ci.Repo)
		}

	})

	t.Run("empty repo is tolerated", func(t *testing.T) {
		setenvCI(t, "abc1234def5678", "refs/heads/main", "")

		ci := DetectCI()
		if stdlibAssertNil(ci) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("", ci.Owner) {
			t.Fatalf("want %v, got %v", "", ci.Owner)
		}
		if !stdlibAssertEqual("", ci.Repo) {
			t.Fatalf("want %v, got %v", "", ci.Repo)
		}

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
		if !stdlibAssertEqual("core_linux_amd64_v1.2.3", name) {
			t.Fatalf("want %v, got %v", "core_linux_amd64_v1.2.3", name)
		}

	})

	t.Run("uses ShortSHA when not a tag", func(t *testing.T) {
		ci := &CIContext{
			IsTag:    false,
			ShortSHA: "abc1234",
		}
		name := ArtifactName("myapp", ci, Target{OS: "darwin", Arch: "arm64"})
		if !stdlibAssertEqual("myapp_darwin_arm64_abc1234", name) {
			t.Fatalf("want %v, got %v", "myapp_darwin_arm64_abc1234", name)
		}

	})

	t.Run("produces correct format for windows", func(t *testing.T) {
		ci := &CIContext{IsTag: true, Tag: "v2.0.0", ShortSHA: "ff00ff0"}
		name := ArtifactName("core", ci, Target{OS: "windows", Arch: "amd64"})
		if !stdlibAssertEqual("core_windows_amd64_v2.0.0", name) {
			t.Fatalf("want %v, got %v", "core_windows_amd64_v2.0.0", name)
		}

	})
}

func TestCi_ArtifactName_Bad(t *testing.T) {
	t.Run("nil ci returns name_os_arch only", func(t *testing.T) {
		name := ArtifactName("core", nil, Target{OS: "linux", Arch: "amd64"})
		if !stdlibAssertEqual("core_linux_amd64", name) {
			t.Fatalf("want %v, got %v", "core_linux_amd64", name)
		}

	})

	t.Run("ci with no tag and no SHA returns name_os_arch only", func(t *testing.T) {
		ci := &CIContext{IsTag: false, ShortSHA: "", Tag: ""}
		name := ArtifactName("core", ci, Target{OS: "linux", Arch: "amd64"})
		if !stdlibAssertEqual("core_linux_amd64", name) {
			t.Fatalf("want %v, got %v", "core_linux_amd64", name)
		}

	})
}

func TestCi_ArtifactName_Ugly(t *testing.T) {
	t.Run("empty build name produces leading underscore segments", func(t *testing.T) {
		ci := &CIContext{IsTag: true, Tag: "v1.0.0", ShortSHA: "abc1234"}
		name := ArtifactName("", ci, Target{OS: "linux", Arch: "amd64"})
		if !stdlibAssertContains(
			// Empty name results in "_linux_amd64_v1.0.0"
			name, "linux_amd64_v1.0.0") {
			t.Fatalf("expected %v to contain %v", name, "linux_amd64_v1.0.0")
		}

	})

	t.Run("IsTag true but empty tag falls back to ShortSHA", func(t *testing.T) {
		ci := &CIContext{IsTag: true, Tag: "", ShortSHA: "abc1234"}
		name := ArtifactName("core", ci, Target{OS: "linux", Arch: "amd64"})
		if !stdlibAssertEqual("core_linux_amd64_abc1234", name) {
			t.Fatalf("want %v, got %v", "core_linux_amd64_abc1234", name)
		}

	})

	t.Run("special chars in build name are preserved", func(t *testing.T) {
		ci := &CIContext{IsTag: true, Tag: "v1.0.0"}
		name := ArtifactName("core-build", ci, Target{OS: "linux", Arch: "amd64"})
		if !stdlibAssertEqual("core-build_linux_amd64_v1.0.0", name) {
			t.Fatalf("want %v, got %v", "core-build_linux_amd64_v1.0.0", name)
		}

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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, readErr := ax.ReadFile(path)
		if readErr != nil {
			t.Fatalf("unexpected error: %v", readErr)
		}

		var meta map[string]any
		if err := json.Unmarshal(content, &meta); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("core", meta["name"]) {
			t.Fatalf("want %v, got %v", "core", meta["name"])
		}
		if !stdlibAssertEqual("linux", meta["os"]) {
			t.Fatalf("want %v, got %v", "linux", meta["os"])
		}
		if !stdlibAssertEqual("amd64", meta["arch"]) {
			t.Fatalf("want %v, got %v", "amd64", meta["arch"])
		}
		if !stdlibAssertEqual("v1.2.3", meta["tag"]) {
			t.Fatalf("want %v, got %v", "v1.2.3", meta["tag"])
		}
		if !stdlibAssertEqual(true, meta["is_tag"]) {
			t.Fatalf("want %v, got %v", true, meta["is_tag"])
		}
		if !stdlibAssertEqual("dappcore/core", meta["repo"]) {
			t.Fatalf("want %v, got %v", "dappcore/core", meta["repo"])
		}
		if !stdlibAssertEqual("refs/tags/v1.2.3", meta["ref"]) {
			t.Fatalf("want %v, got %v", "refs/tags/v1.2.3", meta["ref"])
		}

	})

	t.Run("writes valid JSON without CI context", func(t *testing.T) {
		dir := t.TempDir()
		path := ax.Join(dir, "artifact_meta.json")

		err := WriteArtifactMeta(fs, path, "myapp", Target{OS: "darwin", Arch: "arm64"}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, readErr := ax.ReadFile(path)
		if readErr != nil {
			t.Fatalf("unexpected error: %v", readErr)
		}

		var meta map[string]any
		if err := json.Unmarshal(content, &meta); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("myapp", meta["name"]) {
			t.Fatalf("want %v, got %v", "myapp", meta["name"])
		}
		if !stdlibAssertEqual("darwin", meta["os"]) {
			t.Fatalf("want %v, got %v", "darwin", meta["os"])
		}
		if !stdlibAssertEqual("arm64", meta["arch"]) {
			t.Fatalf("want %v, got %v", "arm64", meta["arch"])
		}
		if !stdlibAssertEqual(false, meta["is_tag"]) {
			t.Fatalf("want %v, got %v", false, meta["is_tag"])
		}

	})

	t.Run("output is pretty-printed JSON", func(t *testing.T) {
		dir := t.TempDir()
		path := ax.Join(dir, "artifact_meta.json")

		err := WriteArtifactMeta(fs, path, "core", Target{OS: "windows", Arch: "amd64"}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, readErr := ax.ReadFile(path)
		if readErr != nil {
			t.Fatalf("unexpected error: %v",

				// Pretty-printed JSON contains indentation
				readErr)
		}
		if !stdlibAssertContains(string(content), "\n") {
			t.Fatalf("expected %v to contain %v", string(content), "\n")
		}
		if !stdlibAssertContains(string(content), "  ") {
			t.Fatalf("expected %v to contain %v", string(content), "  ")
		}

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
		if !stdlibAssertEqual("/tmp/dist/linux_amd64/core_linux_amd64_v1.2.3.tar.gz", path) {
			t.Fatalf("want %v, got %v", "/tmp/dist/linux_amd64/core_linux_amd64_v1.2.3.tar.gz", path)
		}

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
		if !stdlibAssertEqual("/tmp/dist/darwin_arm64/core_darwin_arm64_abc1234.app", path) {
			t.Fatalf("want %v, got %v", "/tmp/dist/darwin_arm64/core_darwin_arm64_abc1234.app", path)
		}

	})

	t.Run("returns the original path when CI metadata is unavailable", func(t *testing.T) {
		artifact := Artifact{
			Path: "/tmp/dist/linux_amd64/core",
			OS:   "linux",
			Arch: "amd64",
		}
		if !stdlibAssertEqual(artifact.Path, CIArtifactPath("core", nil, artifact)) {
			t.Fatalf("want %v, got %v", artifact.Path, CIArtifactPath("core", nil, artifact))
		}

	})
}
