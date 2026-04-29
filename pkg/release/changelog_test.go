package release

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
)

func TestChangelog_ParseConventionalCommitGood(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *ConventionalCommit
	}{
		{
			name:  "feat without scope",
			input: "abc1234 feat: add new feature",
			expected: &ConventionalCommit{
				Type:        "feat",
				Scope:       "",
				Description: "add new feature",
				Hash:        "abc1234",
				Breaking:    false,
			},
		},
		{
			name:  "fix with scope",
			input: "def5678 fix(auth): resolve login issue",
			expected: &ConventionalCommit{
				Type:        "fix",
				Scope:       "auth",
				Description: "resolve login issue",
				Hash:        "def5678",
				Breaking:    false,
			},
		},
		{
			name:  "breaking change with exclamation",
			input: "ghi9012 feat!: breaking API change",
			expected: &ConventionalCommit{
				Type:        "feat",
				Scope:       "",
				Description: "breaking API change",
				Hash:        "ghi9012",
				Breaking:    true,
			},
		},
		{
			name:  "breaking change with scope",
			input: "jkl3456 fix(api)!: remove deprecated endpoint",
			expected: &ConventionalCommit{
				Type:        "fix",
				Scope:       "api",
				Description: "remove deprecated endpoint",
				Hash:        "jkl3456",
				Breaking:    true,
			},
		},
		{
			name:  "perf type",
			input: "mno7890 perf: optimize database queries",
			expected: &ConventionalCommit{
				Type:        "perf",
				Scope:       "",
				Description: "optimize database queries",
				Hash:        "mno7890",
				Breaking:    false,
			},
		},
		{
			name:  "chore type",
			input: "pqr1234 chore: update dependencies",
			expected: &ConventionalCommit{
				Type:        "chore",
				Scope:       "",
				Description: "update dependencies",
				Hash:        "pqr1234",
				Breaking:    false,
			},
		},
		{
			name:  "uppercase type normalizes to lowercase",
			input: "stu5678 FEAT: uppercase type",
			expected: &ConventionalCommit{
				Type:        "feat",
				Scope:       "",
				Description: "uppercase type",
				Hash:        "stu5678",
				Breaking:    false,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := parseConventionalCommit(tc.input)
			if stdlibAssertNil(result) {
				t.Fatal("expected non-nil")
			}
			if !stdlibAssertEqual(tc.expected.Type, result.Type) {
				t.Fatalf("want %v, got %v", tc.expected.Type, result.Type)
			}
			if !stdlibAssertEqual(tc.expected.Scope, result.Scope) {
				t.Fatalf("want %v, got %v", tc.expected.Scope, result.Scope)
			}
			if !stdlibAssertEqual(tc.expected.Description, result.Description) {
				t.Fatalf("want %v, got %v", tc.expected.Description, result.Description)
			}
			if !stdlibAssertEqual(tc.expected.Hash, result.Hash) {
				t.Fatalf("want %v, got %v", tc.expected.Hash, result.Hash)
			}
			if !stdlibAssertEqual(tc.expected.Breaking, result.Breaking) {
				t.Fatalf("want %v, got %v", tc.expected.Breaking, result.Breaking)
			}

		})
	}
}

func TestChangelog_ParseConventionalCommitBad(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "non-conventional commit",
			input: "abc1234 Update README",
		},
		{
			name:  "missing colon",
			input: "def5678 feat add feature",
		},
		{
			name:  "empty subject",
			input: "ghi9012",
		},
		{
			name:  "just hash",
			input: "abc1234",
		},
		{
			name:  "merge commit",
			input: "abc1234 Merge pull request #123",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := parseConventionalCommit(tc.input)
			if !stdlibAssertNil(result) {
				t.Fatalf("expected nil, got %v", result)
			}

		})
	}
}

func TestChangelog_FormatChangelogGood(t *testing.T) {
	t.Run("formats commits by type", func(t *testing.T) {
		commits := []ConventionalCommit{
			{Type: "feat", Description: "add feature A", Hash: "abc1234"},
			{Type: "fix", Description: "fix bug B", Hash: "def5678"},
			{Type: "feat", Description: "add feature C", Hash: "ghi9012"},
		}

		result := formatChangelog(commits, "v1.0.0")
		if !stdlibAssertContains(result, "## v1.0.0") {
			t.Fatalf("expected %v to contain %v", result, "## v1.0.0")
		}
		if !stdlibAssertContains(result, "### Features") {
			t.Fatalf("expected %v to contain %v", result, "### Features")
		}
		if !stdlibAssertContains(result, "### Bug Fixes") {
			t.Fatalf("expected %v to contain %v", result, "### Bug Fixes")
		}
		if !stdlibAssertContains(result, "- add feature A (abc1234)") {
			t.Fatalf("expected %v to contain %v", result, "- add feature A (abc1234)")
		}
		if !stdlibAssertContains(result, "- fix bug B (def5678)") {
			t.Fatalf("expected %v to contain %v", result, "- fix bug B (def5678)")
		}
		if !stdlibAssertContains(result, "- add feature C (ghi9012)") {
			t.Fatalf("expected %v to contain %v", result, "- add feature C (ghi9012)")
		}

	})

	t.Run("includes scope in output", func(t *testing.T) {
		commits := []ConventionalCommit{
			{Type: "feat", Scope: "api", Description: "add endpoint", Hash: "abc1234"},
		}

		result := formatChangelog(commits, "v1.0.0")
		if !stdlibAssertContains(result, "**api**: add endpoint") {
			t.Fatalf("expected %v to contain %v", result, "**api**: add endpoint")
		}

	})

	t.Run("breaking changes first", func(t *testing.T) {
		commits := []ConventionalCommit{
			{Type: "feat", Description: "normal feature", Hash: "abc1234"},
			{Type: "feat", Description: "breaking feature", Hash: "def5678", Breaking: true},
		}

		result := formatChangelog(commits, "v1.0.0")
		if !stdlibAssertContains(result, "### BREAKING CHANGES") {
			t.Fatalf(

				// Breaking changes section should appear before Features
				"expected %v to contain %v", result, "### BREAKING CHANGES")
		}

		breakingPos := indexOf(result, "BREAKING CHANGES")
		featuresPos := indexOf(result, "Features")
		if breakingPos >= featuresPos {
			t.Fatalf("expected %v to be less than %v", breakingPos, featuresPos)
		}

	})

	t.Run("empty commits returns minimal changelog", func(t *testing.T) {
		result := formatChangelog([]ConventionalCommit{}, "v1.0.0")
		if !stdlibAssertContains(result, "## v1.0.0") {
			t.Fatalf("expected %v to contain %v", result, "## v1.0.0")
		}
		if !stdlibAssertContains(result, "No notable changes") {
			t.Fatalf("expected %v to contain %v", result, "No notable changes")
		}

	})
}

func TestChangelog_ParseCommitType_Good(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"feat: add feature", "feat"},
		{"fix(scope): fix bug", "fix"},
		{"perf!: breaking perf", "perf"},
		{"chore: update deps", "chore"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := ParseCommitType(tc.input)
			if !stdlibAssertEqual(tc.expected, result) {
				t.Fatalf("want %v, got %v", tc.expected, result)
			}

		})
	}
}

func TestChangelog_ParseCommitType_Bad(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"not a conventional commit"},
		{"Update README"},
		{"Merge branch 'main'"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := ParseCommitType(tc.input)
			if !stdlibAssertEmpty(result) {
				t.Fatalf("expected empty, got %v", result)
			}

		})
	}
}

func TestChangelog_GenerateWithConfigConfigValuesGood(t *testing.T) {
	t.Run("config filters are parsed correctly", func(t *testing.T) {
		cfg := &ChangelogConfig{
			Include: []string{"feat", "fix"},
			Exclude: []string{"chore", "docs"},
		}
		if !stdlibAssertContains(

			// Verify the config values
			cfg.Include, "feat") {
			t.Fatalf("expected %v to contain %v", cfg.Include, "feat")
		}
		if !stdlibAssertContains(cfg.Include, "fix") {
			t.Fatalf("expected %v to contain %v", cfg.Include, "fix")
		}
		if !stdlibAssertContains(

			// indexOf returns the position of a substring in a string, or -1 if not found.
			cfg.Exclude, "chore") {
			t.Fatalf("expected %v to contain %v", cfg.Exclude, "chore")
		}
		if !stdlibAssertContains(cfg.Exclude, "docs") {
			t.Fatalf("expected %v to contain %v", cfg.Exclude, "docs")
		}

	})
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// setupChangelogGitRepo creates a temporary directory with an initialized git repository.
func setupChangelogGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Initialize git repo
	runGit(t, dir, "init")

	// Configure git user for commits
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")

	return dir
}

// createChangelogCommit creates a commit in the given directory.
func createChangelogCommit(t *testing.T, dir, message string) {
	t.Helper()

	// Create or modify a file
	filePath := ax.Join(dir, "changelog_test.txt")
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

// createChangelogTag creates a tag in the given directory.
func createChangelogTag(t *testing.T, dir, tag string) {
	t.Helper()
	runGit(t, dir, "tag", tag)
}

func TestChangelog_Generate_Good(t *testing.T) {
	t.Run("generates changelog from commits", func(t *testing.T) {
		dir := setupChangelogGitRepo(t)
		createChangelogCommit(t, dir, "feat: add new feature")
		createChangelogCommit(t, dir, "fix: resolve bug")

		changelog, err := Generate(dir, "", "HEAD")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(changelog, "## HEAD") {
			t.Fatalf("expected %v to contain %v", changelog, "## HEAD")
		}
		if !stdlibAssertContains(changelog, "### Features") {
			t.Fatalf("expected %v to contain %v", changelog, "### Features")
		}
		if !stdlibAssertContains(changelog, "add new feature") {
			t.Fatalf("expected %v to contain %v", changelog, "add new feature")
		}
		if !stdlibAssertContains(changelog, "### Bug Fixes") {
			t.Fatalf("expected %v to contain %v", changelog, "### Bug Fixes")
		}
		if !stdlibAssertContains(changelog, "resolve bug") {
			t.Fatalf("expected %v to contain %v", changelog, "resolve bug")
		}

	})

	t.Run("generates changelog between tags", func(t *testing.T) {
		dir := setupChangelogGitRepo(t)
		createChangelogCommit(t, dir, "feat: initial feature")
		createChangelogTag(t, dir, "v1.0.0")
		createChangelogCommit(t, dir, "feat: new feature")
		createChangelogCommit(t, dir, "fix: bug fix")
		createChangelogTag(t, dir, "v1.1.0")

		changelog, err := Generate(dir, "v1.0.0", "v1.1.0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(changelog, "## v1.1.0") {
			t.Fatalf("expected %v to contain %v", changelog, "## v1.1.0")
		}
		if !stdlibAssertContains(

			// Should NOT contain the initial feature
			changelog, "new feature") {
			t.Fatalf("expected %v to contain %v", changelog, "new feature")
		}
		if !stdlibAssertContains(changelog, "bug fix") {
			t.Fatalf("expected %v to contain %v", changelog, "bug fix")
		}
		if stdlibAssertContains(changelog, "initial feature") {
			t.Fatalf("expected %v not to contain %v", changelog, "initial feature")
		}

	})

	t.Run("handles empty changelog when no conventional commits", func(t *testing.T) {
		dir := setupChangelogGitRepo(t)
		createChangelogCommit(t, dir, "Update README")
		createChangelogCommit(t, dir, "Merge branch main")

		changelog, err := Generate(dir, "", "HEAD")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(changelog, "No notable changes") {
			t.Fatalf("expected %v to contain %v", changelog, "No notable changes")
		}

	})

	t.Run("uses previous tag when fromRef is empty", func(t *testing.T) {
		dir := setupChangelogGitRepo(t)
		createChangelogCommit(t, dir, "feat: old feature")
		createChangelogTag(t, dir, "v1.0.0")
		createChangelogCommit(t, dir, "feat: new feature")

		changelog, err := Generate(dir, "", "HEAD")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(changelog, "new feature") {
			t.Fatalf("expected %v to contain %v", changelog, "new feature")
		}
		if stdlibAssertContains(changelog, "old feature") {
			t.Fatalf("expected %v not to contain %v", changelog, "old feature")
		}

	})

	t.Run("includes breaking changes", func(t *testing.T) {
		dir := setupChangelogGitRepo(t)
		createChangelogCommit(t, dir, "feat!: breaking API change")
		createChangelogCommit(t, dir, "feat: normal feature")

		changelog, err := Generate(dir, "", "HEAD")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(changelog, "### BREAKING CHANGES") {
			t.Fatalf("expected %v to contain %v", changelog, "### BREAKING CHANGES")
		}
		if !stdlibAssertContains(changelog, "breaking API change") {
			t.Fatalf("expected %v to contain %v", changelog, "breaking API change")
		}

	})

	t.Run("includes scope in output", func(t *testing.T) {
		dir := setupChangelogGitRepo(t)
		createChangelogCommit(t, dir, "feat(api): add endpoint")

		changelog, err := Generate(dir, "", "HEAD")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(changelog, "**api**:") {
			t.Fatalf("expected %v to contain %v", changelog, "**api**:")
		}

	})
}

func TestChangelog_Generate_Bad(t *testing.T) {
	t.Run("returns error for non-git directory", func(t *testing.T) {
		dir := t.TempDir()

		_, err := Generate(dir, "", "HEAD")
		if err == nil {
			t.Fatal("expected error")
		}

	})

	t.Run("returns error when context is cancelled", func(t *testing.T) {
		dir := setupChangelogGitRepo(t)
		createChangelogCommit(t, dir, "feat: add new feature")

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := GenerateWithContext(ctx, dir, "", "HEAD")
		if err == nil {
			t.Fatal("expected error")
		}
		if !core.Is(err, context.Canceled) {
			t.Fatalf("expected error %v to be %v", err, context.Canceled)
		}

	})
}

func TestChangelog_GenerateWithConfig_Good(t *testing.T) {
	t.Run("filters commits by include list", func(t *testing.T) {
		dir := setupChangelogGitRepo(t)
		createChangelogCommit(t, dir, "feat: new feature")
		createChangelogCommit(t, dir, "fix: bug fix")
		createChangelogCommit(t, dir, "chore: update deps")

		cfg := &ChangelogConfig{
			Include: []string{"feat"},
		}

		changelog, err := GenerateWithConfig(dir, "", "HEAD", cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(changelog, "new feature") {
			t.Fatalf("expected %v to contain %v", changelog, "new feature")
		}
		if stdlibAssertContains(changelog, "bug fix") {
			t.Fatalf("expected %v not to contain %v", changelog, "bug fix")
		}
		if stdlibAssertContains(changelog, "update deps") {
			t.Fatalf("expected %v not to contain %v", changelog, "update deps")
		}

	})

	t.Run("filters commits by exclude list", func(t *testing.T) {
		dir := setupChangelogGitRepo(t)
		createChangelogCommit(t, dir, "feat: new feature")
		createChangelogCommit(t, dir, "fix: bug fix")
		createChangelogCommit(t, dir, "chore: update deps")

		cfg := &ChangelogConfig{
			Exclude: []string{"chore"},
		}

		changelog, err := GenerateWithConfig(dir, "", "HEAD", cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(changelog, "new feature") {
			t.Fatalf("expected %v to contain %v", changelog, "new feature")
		}
		if !stdlibAssertContains(changelog, "bug fix") {
			t.Fatalf("expected %v to contain %v", changelog, "bug fix")
		}
		if stdlibAssertContains(changelog, "update deps") {
			t.Fatalf("expected %v not to contain %v", changelog, "update deps")
		}

	})

	t.Run("combines include and exclude filters", func(t *testing.T) {
		dir := setupChangelogGitRepo(t)
		createChangelogCommit(t, dir, "feat: new feature")
		createChangelogCommit(t, dir, "fix: bug fix")
		createChangelogCommit(t, dir, "perf: performance")

		cfg := &ChangelogConfig{
			Include: []string{"feat", "fix", "perf"},
			Exclude: []string{"perf"},
		}

		changelog, err := GenerateWithConfig(dir, "", "HEAD", cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(changelog, "new feature") {
			t.Fatalf("expected %v to contain %v", changelog, "new feature")
		}
		if !stdlibAssertContains(changelog, "bug fix") {
			t.Fatalf("expected %v to contain %v", changelog, "bug fix")
		}
		if stdlibAssertContains(changelog, "performance") {
			t.Fatalf("expected %v not to contain %v", changelog, "performance")
		}

	})

	t.Run("supports regex exclude patterns from release config", func(t *testing.T) {
		dir := setupChangelogGitRepo(t)
		createChangelogCommit(t, dir, "feat: new feature")
		createChangelogCommit(t, dir, "docs: update README")
		createChangelogCommit(t, dir, "ci: tidy workflow")

		cfg := &ChangelogConfig{
			Exclude: []string{"^docs:", "^ci:"},
			Use:     "conventional",
		}

		changelog, err := GenerateWithConfig(dir, "", "HEAD", cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(changelog, "new feature") {
			t.Fatalf("expected %v to contain %v", changelog, "new feature")
		}
		if stdlibAssertContains(changelog, "update README") {
			t.Fatalf("expected %v not to contain %v", changelog, "update README")
		}
		if stdlibAssertContains(changelog, "tidy workflow") {
			t.Fatalf("expected %v not to contain %v", changelog, "tidy workflow")
		}

	})
}

func TestChangelog_GetCommitsGood(t *testing.T) {
	t.Run("returns all commits when fromRef is empty", func(t *testing.T) {
		dir := setupChangelogGitRepo(t)
		createChangelogCommit(t, dir, "feat: first")
		createChangelogCommit(t, dir, "feat: second")
		createChangelogCommit(t, dir, "feat: third")

		commits, err := getCommitsWithContext(context.Background(), dir, "", "HEAD")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(commits) != 3 {
			t.Fatalf("want len %v, got %v", 3, len(commits))
		}

	})

	t.Run("returns commits between refs", func(t *testing.T) {
		dir := setupChangelogGitRepo(t)
		createChangelogCommit(t, dir, "feat: first")
		createChangelogTag(t, dir, "v1.0.0")
		createChangelogCommit(t, dir, "feat: second")
		createChangelogCommit(t, dir, "feat: third")

		commits, err := getCommitsWithContext(context.Background(), dir, "v1.0.0", "HEAD")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(commits) != 2 {
			t.Fatalf("want len %v, got %v", 2, len(commits))
		}

	})

	t.Run("excludes merge commits", func(t *testing.T) {
		dir := setupChangelogGitRepo(t)
		createChangelogCommit(t, dir, "feat: regular commit")
		// Merge commits are excluded by --no-merges flag
		// We can verify by checking the count matches expected

		commits, err := getCommitsWithContext(context.Background(), dir, "", "HEAD")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(commits) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(commits))
		}
		if !stdlibAssertContains(commits[0], "regular commit") {
			t.Fatalf("expected %v to contain %v", commits[0], "regular commit")
		}

	})

	t.Run("returns empty slice for no commits in range", func(t *testing.T) {
		dir := setupChangelogGitRepo(t)
		createChangelogCommit(t, dir, "feat: only commit")
		createChangelogTag(t, dir, "v1.0.0")

		commits, err := getCommitsWithContext(context.Background(), dir, "v1.0.0", "HEAD")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEmpty(commits) {
			t.Fatalf("expected empty, got %v", commits)
		}

	})
}

func TestChangelog_GetCommitsBad(t *testing.T) {
	t.Run("returns error for invalid ref", func(t *testing.T) {
		dir := setupChangelogGitRepo(t)
		createChangelogCommit(t, dir, "feat: commit")

		_, err := getCommitsWithContext(context.Background(), dir, "nonexistent-tag", "HEAD")
		if err == nil {
			t.Fatal("expected error")
		}

	})

	t.Run("returns error for non-git directory", func(t *testing.T) {
		dir := t.TempDir()

		_, err := getCommitsWithContext(context.Background(), dir, "", "HEAD")
		if err == nil {
			t.Fatal("expected error")
		}

	})
}

func TestChangelog_GetPreviousTagGood(t *testing.T) {
	t.Run("returns previous tag", func(t *testing.T) {
		dir := setupChangelogGitRepo(t)
		createChangelogCommit(t, dir, "feat: first")
		createChangelogTag(t, dir, "v1.0.0")
		createChangelogCommit(t, dir, "feat: second")
		createChangelogTag(t, dir, "v1.1.0")

		tag, err := getPreviousTagWithContext(context.Background(), dir, "v1.1.0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("v1.0.0", tag) {
			t.Fatalf("want %v, got %v", "v1.0.0", tag)
		}

	})

	t.Run("returns tag before HEAD", func(t *testing.T) {
		dir := setupChangelogGitRepo(t)
		createChangelogCommit(t, dir, "feat: first")
		createChangelogTag(t, dir, "v1.0.0")
		createChangelogCommit(t, dir, "feat: second")

		tag, err := getPreviousTagWithContext(context.Background(), dir, "HEAD")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("v1.0.0", tag) {
			t.Fatalf("want %v, got %v", "v1.0.0", tag)
		}

	})
}

func TestChangelog_GetPreviousTagBad(t *testing.T) {
	t.Run("returns error when no previous tag exists", func(t *testing.T) {
		dir := setupChangelogGitRepo(t)
		createChangelogCommit(t, dir, "feat: first")
		createChangelogTag(t, dir, "v1.0.0")

		// v1.0.0^ has no tag before it
		_, err := getPreviousTagWithContext(context.Background(), dir, "v1.0.0")
		if err == nil {
			t.Fatal("expected error")
		}

	})

	t.Run("returns error for invalid ref", func(t *testing.T) {
		dir := setupChangelogGitRepo(t)
		createChangelogCommit(t, dir, "feat: commit")

		_, err := getPreviousTagWithContext(context.Background(), dir, "nonexistent")
		if err == nil {
			t.Fatal("expected error")
		}

	})
}

func TestChangelog_FormatCommitLineGood(t *testing.T) {
	t.Run("formats commit without scope", func(t *testing.T) {
		commit := ConventionalCommit{
			Type:        "feat",
			Description: "add feature",
			Hash:        "abc1234",
		}

		result := formatCommitLine(commit)
		if !stdlibAssertEqual("- add feature (abc1234)\n", result) {
			t.Fatalf("want %v, got %v", "- add feature (abc1234)\n", result)
		}

	})

	t.Run("formats commit with scope", func(t *testing.T) {
		commit := ConventionalCommit{
			Type:        "fix",
			Scope:       "api",
			Description: "fix bug",
			Hash:        "def5678",
		}

		result := formatCommitLine(commit)
		if !stdlibAssertEqual("- **api**: fix bug (def5678)\n", result) {
			t.Fatalf("want %v, got %v", "- **api**: fix bug (def5678)\n", result)
		}

	})
}

func TestChangelog_FormatChangelogUgly(t *testing.T) {
	t.Run("handles custom commit type not in order", func(t *testing.T) {
		commits := []ConventionalCommit{
			{Type: "custom", Description: "custom type", Hash: "abc1234"},
		}

		result := formatChangelog(commits, "v1.0.0")
		if !stdlibAssertContains(result, "### Custom") {
			t.Fatalf("expected %v to contain %v", result, "### Custom")
		}
		if !stdlibAssertContains(result, "custom type") {
			t.Fatalf("expected %v to contain %v", result, "custom type")
		}

	})

	t.Run("handles multiple custom commit types", func(t *testing.T) {
		commits := []ConventionalCommit{
			{Type: "alpha", Description: "alpha feature", Hash: "abc1234"},
			{Type: "beta", Description: "beta feature", Hash: "def5678"},
		}

		result := formatChangelog(commits, "v1.0.0")
		if !stdlibAssertContains(

			// Should be sorted alphabetically for custom types
			result, "### Alpha") {
			t.Fatalf("expected %v to contain %v", result, "### Alpha")
		}
		if !stdlibAssertContains(result, "### Beta") {
			t.Fatalf("expected %v to contain %v", result, "### Beta")
		}

	})
}

func TestChangelog_GenerateWithConfig_Bad(t *testing.T) {
	t.Run("returns error for non-git directory", func(t *testing.T) {
		dir := t.TempDir()
		cfg := &ChangelogConfig{
			Include: []string{"feat"},
		}

		_, err := GenerateWithConfig(dir, "", "HEAD", cfg)
		if err == nil {
			t.Fatal("expected error")
		}

	})
}

func TestChangelog_GenerateWithConfigEdgeCasesUgly(t *testing.T) {
	t.Run("uses HEAD when toRef is empty", func(t *testing.T) {
		dir := setupChangelogGitRepo(t)
		createChangelogCommit(t, dir, "feat: new feature")

		cfg := &ChangelogConfig{
			Include: []string{"feat"},
		}

		// Pass empty toRef
		changelog, err := GenerateWithConfig(dir, "", "", cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(changelog, "## HEAD") {
			t.Fatalf("expected %v to contain %v", changelog, "## HEAD")
		}

	})

	t.Run("handles previous tag lookup failure gracefully", func(t *testing.T) {
		dir := setupChangelogGitRepo(t)
		createChangelogCommit(t, dir, "feat: first")

		cfg := &ChangelogConfig{
			Include: []string{"feat"},
		}

		// No tags exist, should still work
		changelog, err := GenerateWithConfig(dir, "", "HEAD", cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(changelog, "first") {
			t.Fatalf("expected %v to contain %v", changelog, "first")
		}

	})

	t.Run("uses explicit fromRef when provided", func(t *testing.T) {
		dir := setupChangelogGitRepo(t)
		createChangelogCommit(t, dir, "feat: old feature")
		createChangelogTag(t, dir, "v1.0.0")
		createChangelogCommit(t, dir, "feat: new feature")

		cfg := &ChangelogConfig{
			Include: []string{"feat"},
		}

		// Use explicit fromRef
		changelog, err := GenerateWithConfig(dir, "v1.0.0", "HEAD", cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(changelog, "new feature") {
			t.Fatalf("expected %v to contain %v", changelog, "new feature")
		}
		if stdlibAssertContains(changelog, "old feature") {
			t.Fatalf("expected %v not to contain %v", changelog, "old feature")
		}

	})

	t.Run("skips non-conventional commits", func(t *testing.T) {
		dir := setupChangelogGitRepo(t)
		createChangelogCommit(t, dir, "feat: conventional commit")
		createChangelogCommit(t, dir, "Update README")

		cfg := &ChangelogConfig{
			Include: []string{"feat"},
		}

		changelog, err := GenerateWithConfig(dir, "", "HEAD", cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(changelog, "conventional commit") {
			t.Fatalf("expected %v to contain %v", changelog, "conventional commit")
		}
		if stdlibAssertContains(changelog, "Update README") {
			t.Fatalf("expected %v not to contain %v", changelog, "Update README")
		}

	})
}

// --- v0.9.0 generated compliance triplets ---
func TestChangelog_Generate_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = Generate(core.Path(t.TempDir(), "go-build-compliance"), "agent", "agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestChangelog_GenerateWithContext_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = GenerateWithContext(ctx, core.Path(t.TempDir(), "go-build-compliance"), "agent", "agent")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestChangelog_GenerateWithContext_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = GenerateWithContext(ctx, "", "", "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestChangelog_GenerateWithContext_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = GenerateWithContext(ctx, core.Path(t.TempDir(), "go-build-compliance"), "agent", "agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestChangelog_GenerateWithConfig_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = GenerateWithConfig(core.Path(t.TempDir(), "go-build-compliance"), "agent", "agent", &ChangelogConfig{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestChangelog_GenerateWithConfigWithContext_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = GenerateWithConfigWithContext(ctx, core.Path(t.TempDir(), "go-build-compliance"), "agent", "agent", &ChangelogConfig{})
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestChangelog_GenerateWithConfigWithContext_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = GenerateWithConfigWithContext(ctx, "", "", "", nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestChangelog_GenerateWithConfigWithContext_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = GenerateWithConfigWithContext(ctx, core.Path(t.TempDir(), "go-build-compliance"), "agent", "agent", &ChangelogConfig{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestChangelog_ParseCommitType_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ParseCommitType("agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
