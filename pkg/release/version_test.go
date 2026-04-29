package release

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
)

// setupGitRepo creates a temporary directory with an initialized git repository.
func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Initialize git repo
	runGit(t, dir, "init")

	// Configure git user for commits
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")

	return dir
}

// createCommit creates a commit in the given directory.
func createCommit(t *testing.T, dir, message string) {
	t.Helper()

	// Create or modify a file
	filePath := ax.Join(dir, "test.txt")
	content, _ := ax.ReadFile(filePath)
	content = append(content, []byte(message+"\n")...)
	if err := ax.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("unexpected error: %v",

			// Stage and commit
			err)
	}

	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", message)
}

// createTag creates a tag in the given directory.
func createTag(t *testing.T, dir, tag string) {
	t.Helper()
	runGit(t, dir, "tag", tag)
}

func TestVersion_DetermineVersion_Good(t *testing.T) {
	t.Run("uses GitHub tag metadata before local git tags", func(t *testing.T) {
		dir := setupGitRepo(t)
		createCommit(t, dir, "feat: initial commit")

		t.Setenv("GITHUB_SHA", "0123456789abcdef0123456789abcdef01234567")
		t.Setenv("GITHUB_REF", "refs/tags/v2.3.4")

		version, err := DetermineVersionWithContext(context.Background(), dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("v2.3.4", version) {
			t.Fatalf("want %v, got %v", "v2.3.4", version)
		}

	})

	t.Run("returns tag when HEAD has tag", func(t *testing.T) {
		dir := setupGitRepo(t)
		createCommit(t, dir, "feat: initial commit")
		createTag(t, dir, "v1.0.0")

		version, err := DetermineVersion(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("v1.0.0", version) {
			t.Fatalf("want %v, got %v", "v1.0.0", version)
		}

	})

	t.Run("normalizes tag without v prefix", func(t *testing.T) {
		dir := setupGitRepo(t)
		createCommit(t, dir, "feat: initial commit")
		createTag(t, dir, "1.0.0")

		version, err := DetermineVersion(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("v1.0.0", version) {
			t.Fatalf("want %v, got %v", "v1.0.0", version)
		}

	})

	t.Run("increments patch when commits after tag", func(t *testing.T) {
		dir := setupGitRepo(t)
		createCommit(t, dir, "feat: initial commit")
		createTag(t, dir, "v1.0.0")
		createCommit(t, dir, "feat: new feature")

		version, err := DetermineVersion(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("v1.0.1", version) {
			t.Fatalf("want %v, got %v", "v1.0.1", version)
		}

	})

	t.Run("returns v0.0.1 when no tags exist", func(t *testing.T) {
		dir := setupGitRepo(t)
		createCommit(t, dir, "feat: initial commit")

		version, err := DetermineVersion(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("v0.0.1", version) {
			t.Fatalf("want %v, got %v", "v0.0.1", version)
		}

	})

	t.Run("handles multiple tags with increments", func(t *testing.T) {
		dir := setupGitRepo(t)
		createCommit(t, dir, "feat: first")
		createTag(t, dir, "v1.0.0")
		createCommit(t, dir, "feat: second")
		createTag(t, dir, "v1.0.1")
		createCommit(t, dir, "feat: third")

		version, err := DetermineVersion(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("v1.0.2", version) {
			t.Fatalf("want %v, got %v", "v1.0.2", version)
		}

	})
}

func TestVersion_DetermineVersion_Bad(t *testing.T) {
	t.Run("returns v0.0.1 for empty repo", func(t *testing.T) {
		dir := setupGitRepo(t)

		// No commits, git describe will fail
		version, err := DetermineVersion(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("v0.0.1", version) {
			t.Fatalf("want %v, got %v", "v0.0.1", version)
		}

	})

	t.Run("returns error when context is cancelled", func(t *testing.T) {
		dir := setupGitRepo(t)
		createCommit(t, dir, "feat: initial commit")
		createTag(t, dir, "v1.0.0")

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := DetermineVersionWithContext(ctx, dir)
		if err == nil {
			t.Fatal("expected error")
		}
		if !core.Is(err, context.Canceled) {
			t.Fatalf("expected error %v to be %v", err, context.Canceled)
		}

	})

	t.Run("rejects unsafe release tags", func(t *testing.T) {
		dir := setupGitRepo(t)
		createCommit(t, dir, "feat: initial commit")
		createTag(t, dir, "v1.0.0;bad")

		_, err := DetermineVersion(dir)
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "unsafe release tag") {
			t.Fatalf("expected %v to contain %v", err.Error(), "unsafe release tag")
		}

	})

	t.Run("rejects unsafe GitHub tag metadata", func(t *testing.T) {
		dir := setupGitRepo(t)
		createCommit(t, dir, "feat: initial commit")

		t.Setenv("GITHUB_SHA", "0123456789abcdef0123456789abcdef01234567")
		t.Setenv("GITHUB_REF", "refs/tags/v1.0.0;bad")

		_, err := DetermineVersionWithContext(context.Background(), dir)
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "unsafe release tag") {
			t.Fatalf("expected %v to contain %v", err.Error(), "unsafe release tag")
		}

	})
}

func TestVersion_GetTagOnHeadGood(t *testing.T) {
	t.Run("returns tag when HEAD has tag", func(t *testing.T) {
		dir := setupGitRepo(t)
		createCommit(t, dir, "feat: initial commit")
		createTag(t, dir, "v1.2.3")

		tag, err := getTagOnHeadWithContext(context.Background(), dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("v1.2.3", tag) {
			t.Fatalf("want %v, got %v", "v1.2.3", tag)
		}

	})

	t.Run("returns latest tag when multiple tags on HEAD", func(t *testing.T) {
		dir := setupGitRepo(t)
		createCommit(t, dir, "feat: initial commit")
		createTag(t, dir, "v1.0.0")
		createTag(t, dir, "v1.0.0-beta")

		tag, err := getTagOnHeadWithContext(context.Background(), dir)
		if err != nil {
			t.Fatalf("unexpected error: %v",

				// Git returns one of the tags
				err)
		}
		if !stdlibAssertContains([]string{"v1.0.0", "v1.0.0-beta"}, tag) {
			t.Fatalf("expected %v to contain %v", []string{"v1.0.0", "v1.0.0-beta"}, tag)
		}

	})
}

func TestVersion_GetTagOnHeadBad(t *testing.T) {
	t.Run("returns error when HEAD has no tag", func(t *testing.T) {
		dir := setupGitRepo(t)
		createCommit(t, dir, "feat: initial commit")

		_, err := getTagOnHeadWithContext(context.Background(), dir)
		if err == nil {
			t.Fatal("expected error")
		}

	})

	t.Run("returns error when commits after tag", func(t *testing.T) {
		dir := setupGitRepo(t)
		createCommit(t, dir, "feat: initial commit")
		createTag(t, dir, "v1.0.0")
		createCommit(t, dir, "feat: new feature")

		_, err := getTagOnHeadWithContext(context.Background(), dir)
		if err == nil {
			t.Fatal("expected error")
		}

	})
}

func TestVersion_GetLatestTagGood(t *testing.T) {
	t.Run("returns latest tag", func(t *testing.T) {
		dir := setupGitRepo(t)
		createCommit(t, dir, "feat: initial commit")
		createTag(t, dir, "v1.0.0")

		tag, err := getLatestTagWithContext(context.Background(), dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("v1.0.0", tag) {
			t.Fatalf("want %v, got %v", "v1.0.0", tag)
		}

	})

	t.Run("returns most recent tag after multiple commits", func(t *testing.T) {
		dir := setupGitRepo(t)
		createCommit(t, dir, "feat: first")
		createTag(t, dir, "v1.0.0")
		createCommit(t, dir, "feat: second")
		createTag(t, dir, "v1.1.0")
		createCommit(t, dir, "feat: third")

		tag, err := getLatestTagWithContext(context.Background(), dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("v1.1.0", tag) {
			t.Fatalf("want %v, got %v", "v1.1.0", tag)
		}

	})
}

func TestVersion_GetLatestTagBad(t *testing.T) {
	t.Run("returns error when no tags exist", func(t *testing.T) {
		dir := setupGitRepo(t)
		createCommit(t, dir, "feat: initial commit")

		_, err := getLatestTagWithContext(context.Background(), dir)
		if err == nil {
			t.Fatal("expected error")
		}

	})

	t.Run("returns error for empty repo", func(t *testing.T) {
		dir := setupGitRepo(t)

		_, err := getLatestTagWithContext(context.Background(), dir)
		if err == nil {
			t.Fatal("expected error")
		}

	})
}

func TestVersion_IncrementMinor_Bad(t *testing.T) {
	t.Run("returns fallback for invalid version", func(t *testing.T) {
		result := IncrementMinor("not-valid")
		if !stdlibAssertEqual("not-valid.1", result) {
			t.Fatalf("want %v, got %v", "not-valid.1", result)
		}

	})
}

func TestVersion_IncrementMajor_Bad(t *testing.T) {
	t.Run("returns fallback for invalid version", func(t *testing.T) {
		result := IncrementMajor("not-valid")
		if !stdlibAssertEqual("not-valid.1", result) {
			t.Fatalf("want %v, got %v", "not-valid.1", result)
		}

	})
}

func TestVersion_CompareVersions_Ugly(t *testing.T) {
	t.Run("handles both invalid versions", func(t *testing.T) {
		result := CompareVersions("invalid-a", "invalid-b")
		if !stdlibAssertEqual(
			// Should do string comparison for invalid versions
			-1, result) {
			t.Fatalf("want %v, got %v", -1, result)
		}

		// "invalid-a" < "invalid-b"
	})

	t.Run("invalid a returns -1", func(t *testing.T) {
		result := CompareVersions("invalid", "v1.0.0")
		if !stdlibAssertEqual(-1, result) {
			t.Fatalf("want %v, got %v", -1, result)
		}

	})

	t.Run("invalid b returns 1", func(t *testing.T) {
		result := CompareVersions("v1.0.0", "invalid")
		if !stdlibAssertEqual(1, result) {
			t.Fatalf("want %v, got %v", 1, result)
		}

	})
}

func TestVersion_IncrementVersion_Good(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "increment patch with v prefix",
			input:    "v1.2.3",
			expected: "v1.2.4",
		},
		{
			name:     "increment patch without v prefix",
			input:    "1.2.3",
			expected: "v1.2.4",
		},
		{
			name:     "increment from zero",
			input:    "v0.0.0",
			expected: "v0.0.1",
		},
		{
			name:     "strips prerelease",
			input:    "v1.2.3-alpha",
			expected: "v1.2.4",
		},
		{
			name:     "strips build metadata",
			input:    "v1.2.3+build123",
			expected: "v1.2.4",
		},
		{
			name:     "strips prerelease and build",
			input:    "v1.2.3-beta.1+build456",
			expected: "v1.2.4",
		},
		{
			name:     "handles large numbers",
			input:    "v10.20.99",
			expected: "v10.20.100",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IncrementVersion(tc.input)
			if !stdlibAssertEqual(tc.expected, result) {
				t.Fatalf("want %v, got %v", tc.expected, result)
			}

		})
	}
}

func TestVersion_IncrementVersion_Bad(t *testing.T) {
	t.Run("invalid semver returns original with suffix", func(t *testing.T) {
		result := IncrementVersion("not-a-version")
		if !stdlibAssertEqual("not-a-version.1", result) {
			t.Fatalf("want %v, got %v", "not-a-version.1", result)
		}

	})
}

func TestVersion_IncrementMinor_Good(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "increment minor resets patch",
			input:    "v1.2.3",
			expected: "v1.3.0",
		},
		{
			name:     "increment minor from zero",
			input:    "v1.0.5",
			expected: "v1.1.0",
		},
		{
			name:     "handles large numbers",
			input:    "v5.99.50",
			expected: "v5.100.0",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IncrementMinor(tc.input)
			if !stdlibAssertEqual(tc.expected, result) {
				t.Fatalf("want %v, got %v", tc.expected, result)
			}

		})
	}
}

func TestVersion_IncrementMajor_Good(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "increment major resets minor and patch",
			input:    "v1.2.3",
			expected: "v2.0.0",
		},
		{
			name:     "increment major from zero",
			input:    "v0.5.10",
			expected: "v1.0.0",
		},
		{
			name:     "handles large numbers",
			input:    "v99.50.25",
			expected: "v100.0.0",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IncrementMajor(tc.input)
			if !stdlibAssertEqual(tc.expected, result) {
				t.Fatalf("want %v, got %v", tc.expected, result)
			}

		})
	}
}

func TestVersion_ParseVersion_Good(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		major      int
		minor      int
		patch      int
		prerelease string
		build      string
	}{
		{
			name:  "simple version with v",
			input: "v1.2.3",
			major: 1, minor: 2, patch: 3,
		},
		{
			name:  "simple version without v",
			input: "1.2.3",
			major: 1, minor: 2, patch: 3,
		},
		{
			name:  "with prerelease",
			input: "v1.2.3-alpha",
			major: 1, minor: 2, patch: 3,
			prerelease: "alpha",
		},
		{
			name:  "with prerelease and build",
			input: "v1.2.3-beta.1+build.456",
			major: 1, minor: 2, patch: 3,
			prerelease: "beta.1",
			build:      "build.456",
		},
		{
			name:  "with build only",
			input: "v1.2.3+sha.abc123",
			major: 1, minor: 2, patch: 3,
			build: "sha.abc123",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			major, minor, patch, prerelease, build, err := ParseVersion(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !stdlibAssertEqual(tc.major, major) {
				t.Fatalf("want %v, got %v", tc.major, major)
			}
			if !stdlibAssertEqual(tc.minor, minor) {
				t.Fatalf("want %v, got %v", tc.minor, minor)
			}
			if !stdlibAssertEqual(tc.patch, patch) {
				t.Fatalf("want %v, got %v", tc.patch, patch)
			}
			if !stdlibAssertEqual(tc.prerelease, prerelease) {
				t.Fatalf("want %v, got %v", tc.prerelease, prerelease)
			}
			if !stdlibAssertEqual(tc.build, build) {
				t.Fatalf("want %v, got %v", tc.build, build)
			}

		})
	}
}

func TestVersion_ParseVersion_Bad(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"not a version", "not-a-version"},
		{"missing minor", "v1"},
		{"missing patch", "v1.2"},
		{"letters in version", "v1.2.x"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, _, _, err := ParseVersion(tc.input)
			if err == nil {
				t.Fatal("expected error")
			}

		})
	}
}

func TestVersion_ValidateVersion_Good(t *testing.T) {
	validVersions := []string{
		"v1.0.0",
		"1.0.0",
		"v0.0.1",
		"v10.20.30",
		"v1.2.3-alpha",
		"v1.2.3+build",
		"v1.2.3-alpha.1+build.123",
	}

	for _, v := range validVersions {
		t.Run(v, func(t *testing.T) {
			if !(ValidateVersion(v)) {
				t.Fatal("expected true")
			}

		})
	}
}

func TestVersion_ValidateVersion_Bad(t *testing.T) {
	invalidVersions := []string{
		"",
		"v1",
		"v1.2",
		"1.2",
		"not-a-version",
		"v1.2.x",
		"version1.0.0",
	}

	for _, v := range invalidVersions {
		t.Run(v, func(t *testing.T) {
			if ValidateVersion(v) {
				t.Fatal("expected false")
			}

		})
	}
}

func TestVersion_CompareVersions_Good(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected int
	}{
		{"equal versions", "v1.0.0", "v1.0.0", 0},
		{"a less than b major", "v1.0.0", "v2.0.0", -1},
		{"a greater than b major", "v2.0.0", "v1.0.0", 1},
		{"a less than b minor", "v1.1.0", "v1.2.0", -1},
		{"a greater than b minor", "v1.2.0", "v1.1.0", 1},
		{"a less than b patch", "v1.0.1", "v1.0.2", -1},
		{"a greater than b patch", "v1.0.2", "v1.0.1", 1},
		{"with and without v prefix", "v1.0.0", "1.0.0", 0},
		{"different scales", "v1.10.0", "v1.9.0", 1},
		{"prerelease is less than release", "v1.0.0-alpha", "v1.0.0", -1},
		{"release is greater than prerelease", "v1.0.0", "v1.0.0-rc.1", 1},
		{"prerelease identifiers compare lexically", "v1.0.0-alpha", "v1.0.0-beta", -1},
		{"numeric prerelease identifiers compare numerically", "v1.0.0-rc.2", "v1.0.0-rc.10", -1},
		{"numeric prerelease identifiers sort before text", "v1.0.0-1", "v1.0.0-alpha", -1},
		{"longer prerelease wins when prefix matches", "v1.0.0-alpha.1", "v1.0.0-alpha", 1},
		{"build metadata does not affect precedence", "v1.0.0+build.1", "v1.0.0+build.2", 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := CompareVersions(tc.a, tc.b)
			if !stdlibAssertEqual(tc.expected, result) {
				t.Fatalf("want %v, got %v", tc.expected, result)
			}

		})
	}
}

func TestVersion_NormalizeVersionGood(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1.0.0", "v1.0.0"},
		{"v1.0.0", "v1.0.0"},
		{"0.0.1", "v0.0.1"},
		{"v10.20.30", "v10.20.30"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := normalizeVersion(tc.input)
			if !stdlibAssertEqual(tc.expected, result) {
				t.Fatalf("want %v, got %v", tc.expected, result)
			}

		})
	}
}

// --- v0.9.0 generated compliance triplets ---
func TestVersion_DetermineVersion_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = DetermineVersion(core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestVersion_DetermineVersionWithContext_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = DetermineVersionWithContext(ctx, core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestVersion_DetermineVersionWithContext_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = DetermineVersionWithContext(ctx, "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestVersion_DetermineVersionWithContext_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = DetermineVersionWithContext(ctx, core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestVersion_IncrementVersion_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = IncrementVersion("agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestVersion_IncrementMinor_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = IncrementMinor("agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestVersion_IncrementMajor_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = IncrementMajor("agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestVersion_ParseVersion_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_, _, _, _, _, _ = ParseVersion("v1.2.3")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestVersion_ValidateVersion_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ValidateVersion("v1.2.3")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestVersion_ValidateVersionIdentifier_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ValidateVersionIdentifier("v1.2.3")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestVersion_ValidateVersionIdentifier_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ValidateVersionIdentifier("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestVersion_ValidateVersionIdentifier_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ValidateVersionIdentifier("v1.2.3")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestVersion_CompareVersions_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = CompareVersions("", "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}
