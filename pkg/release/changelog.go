// Package release provides release automation with changelog generation and publishing.
package release

import (
	"bufio"
	"bytes"
	"context"
	"regexp"
	"slices"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	coreerr "dappco.re/go/core/log"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// ConventionalCommit represents a parsed conventional commit.
//
// commit := release.ConventionalCommit{Type: "feat", Scope: "build", Description: "add linuxkit support"}
type ConventionalCommit struct {
	Type        string // feat, fix, etc.
	Scope       string // optional scope in parentheses
	Description string // commit description
	Hash        string // short commit hash
	Breaking    bool   // has breaking change indicator
}

// commitTypeLabels maps commit types to human-readable labels for the changelog.
var commitTypeLabels = map[string]string{
	"feat":     "Features",
	"fix":      "Bug Fixes",
	"perf":     "Performance Improvements",
	"refactor": "Code Refactoring",
	"docs":     "Documentation",
	"style":    "Styles",
	"test":     "Tests",
	"build":    "Build System",
	"ci":       "Continuous Integration",
	"chore":    "Chores",
	"revert":   "Reverts",
}

// commitTypeOrder defines the order of sections in the changelog.
var commitTypeOrder = []string{
	"feat",
	"fix",
	"perf",
	"refactor",
	"docs",
	"style",
	"test",
	"build",
	"ci",
	"chore",
	"revert",
}

// conventionalCommitRegex matches conventional commit format.
// Examples: "feat: add feature", "fix(scope): fix bug", "feat!: breaking change"
var conventionalCommitRegex = regexp.MustCompile(`^(\w+)(?:\(([^)]+)\))?(!)?:\s*(.+)$`)

// Generate generates a markdown changelog from git commits between two refs.
// If fromRef is empty, it uses the previous tag or initial commit.
// If toRef is empty, it uses HEAD.
//
// md, err := release.Generate(".", "v1.2.3", "HEAD")
func Generate(dir, fromRef, toRef string) (string, error) {
	return GenerateWithContext(context.Background(), dir, fromRef, toRef)
}

// GenerateWithContext generates a markdown changelog while honouring caller cancellation.
// If fromRef is empty, it uses the previous tag or initial commit.
// If toRef is empty, it uses HEAD.
//
// md, err := release.GenerateWithContext(ctx, ".", "v1.2.3", "HEAD")
func GenerateWithContext(ctx context.Context, dir, fromRef, toRef string) (string, error) {
	if toRef == "" {
		toRef = "HEAD"
	}

	// If fromRef is empty, try to find previous tag
	if fromRef == "" {
		prevTag, err := getPreviousTagWithContext(ctx, dir, toRef)
		if err != nil {
			if ctx.Err() != nil {
				return "", coreerr.E("changelog.Generate", "generation cancelled", ctx.Err())
			}
			// No previous tag, use initial commit
			fromRef = ""
		} else {
			fromRef = prevTag
		}
	}

	// Get commits between refs
	commits, err := getCommitsWithContext(ctx, dir, fromRef, toRef)
	if err != nil {
		return "", coreerr.E("changelog.Generate", "failed to get commits", err)
	}

	// Parse conventional commits
	var parsedCommits []ConventionalCommit
	for _, commit := range commits {
		parsed := parseConventionalCommit(commit)
		if parsed != nil {
			parsedCommits = append(parsedCommits, *parsed)
		}
	}

	// Generate markdown
	return formatChangelog(parsedCommits, toRef), nil
}

// GenerateWithConfig generates a changelog with filtering based on config.
//
// md, err := release.GenerateWithConfig(".", "v1.2.3", "HEAD", &cfg.Changelog)
func GenerateWithConfig(dir, fromRef, toRef string, cfg *ChangelogConfig) (string, error) {
	return GenerateWithConfigWithContext(context.Background(), dir, fromRef, toRef, cfg)
}

// GenerateWithConfigWithContext generates a filtered changelog while honouring caller cancellation.
//
// md, err := release.GenerateWithConfigWithContext(ctx, ".", "v1.2.3", "HEAD", &cfg.Changelog)
func GenerateWithConfigWithContext(ctx context.Context, dir, fromRef, toRef string, cfg *ChangelogConfig) (string, error) {
	if toRef == "" {
		toRef = "HEAD"
	}

	// If fromRef is empty, try to find previous tag
	if fromRef == "" {
		prevTag, err := getPreviousTagWithContext(ctx, dir, toRef)
		if err != nil {
			if ctx.Err() != nil {
				return "", coreerr.E("changelog.GenerateWithConfig", "generation cancelled", ctx.Err())
			}
			fromRef = ""
		} else {
			fromRef = prevTag
		}
	}

	// Get commits between refs
	commits, err := getCommitsWithContext(ctx, dir, fromRef, toRef)
	if err != nil {
		return "", coreerr.E("changelog.GenerateWithConfig", "failed to get commits", err)
	}

	// Build include/exclude sets
	includeSet := make(map[string]bool)
	excludeSet := make(map[string]bool)
	for _, t := range cfg.Include {
		includeSet[t] = true
	}
	for _, t := range cfg.Exclude {
		excludeSet[t] = true
	}

	// Parse and filter conventional commits
	var parsedCommits []ConventionalCommit
	for _, commit := range commits {
		parsed := parseConventionalCommit(commit)
		if parsed == nil {
			continue
		}

		// Apply filters
		if len(includeSet) > 0 && !includeSet[parsed.Type] {
			continue
		}
		if excludeSet[parsed.Type] {
			continue
		}

		parsedCommits = append(parsedCommits, *parsed)
	}

	return formatChangelog(parsedCommits, toRef), nil
}

func getPreviousTagWithContext(ctx context.Context, dir, ref string) (string, error) {
	output, err := ax.RunDir(ctx, dir, "git", "describe", "--tags", "--abbrev=0", ref+"^")
	if err != nil {
		return "", err
	}
	return core.Trim(output), nil
}

func getCommitsWithContext(ctx context.Context, dir, fromRef, toRef string) ([]string, error) {
	var args []string
	if fromRef == "" {
		// All commits up to toRef
		args = []string{"log", "--oneline", "--no-merges", toRef}
	} else {
		// Commits between refs
		args = []string{"log", "--oneline", "--no-merges", fromRef + ".." + toRef}
	}

	output, err := ax.RunDir(ctx, dir, "git", args...)
	if err != nil {
		return nil, err
	}

	var commits []string
	scanner := bufio.NewScanner(bytes.NewReader([]byte(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			commits = append(commits, line)
		}
	}

	return commits, scanner.Err()
}

// parseConventionalCommit parses a git log --oneline output into a ConventionalCommit.
// Returns nil if the commit doesn't follow conventional commit format.
func parseConventionalCommit(commitLine string) *ConventionalCommit {
	// Split hash and subject
	parts := core.SplitN(commitLine, " ", 2)
	if len(parts) != 2 {
		return nil
	}

	hash := parts[0]
	subject := parts[1]

	// Match conventional commit format
	matches := conventionalCommitRegex.FindStringSubmatch(subject)
	if matches == nil {
		return nil
	}

	return &ConventionalCommit{
		Type:        core.Lower(matches[1]),
		Scope:       matches[2],
		Breaking:    matches[3] == "!",
		Description: matches[4],
		Hash:        hash,
	}
}

// formatChangelog formats parsed commits into markdown.
func formatChangelog(commits []ConventionalCommit, version string) string {
	if len(commits) == 0 {
		return core.Sprintf("## %s\n\nNo notable changes.", version)
	}

	// Group commits by type
	grouped := make(map[string][]ConventionalCommit)
	var breaking []ConventionalCommit

	for _, commit := range commits {
		if commit.Breaking {
			breaking = append(breaking, commit)
		}
		grouped[commit.Type] = append(grouped[commit.Type], commit)
	}

	buf := core.NewBuilder()
	buf.WriteString(core.Sprintf("## %s\n\n", version))

	// Breaking changes first
	if len(breaking) > 0 {
		buf.WriteString("### BREAKING CHANGES\n\n")
		for _, commit := range breaking {
			buf.WriteString(formatCommitLine(commit))
		}
		buf.WriteString("\n")
	}

	// Other sections in order
	for _, commitType := range commitTypeOrder {
		commits, ok := grouped[commitType]
		if !ok || len(commits) == 0 {
			continue
		}

		label, ok := commitTypeLabels[commitType]
		if !ok {
			label = cases.Title(language.English).String(commitType)
		}

		buf.WriteString(core.Sprintf("### %s\n\n", label))
		for _, commit := range commits {
			buf.WriteString(formatCommitLine(commit))
		}
		buf.WriteString("\n")
	}

	// Any remaining types not in the order list
	var remainingTypes []string
	for commitType := range grouped {
		if !containsCommitType(commitTypeOrder, commitType) {
			remainingTypes = append(remainingTypes, commitType)
		}
	}
	slices.Sort(remainingTypes)

	for _, commitType := range remainingTypes {
		commits := grouped[commitType]
		label := cases.Title(language.English).String(commitType)
		buf.WriteString(core.Sprintf("### %s\n\n", label))
		for _, commit := range commits {
			buf.WriteString(formatCommitLine(commit))
		}
		buf.WriteString("\n")
	}

	return core.TrimSuffix(buf.String(), "\n")
}

// formatCommitLine formats a single commit as a changelog line.
func formatCommitLine(commit ConventionalCommit) string {
	buf := core.NewBuilder()
	buf.WriteString("- ")

	if commit.Scope != "" {
		buf.WriteString(core.Sprintf("**%s**: ", commit.Scope))
	}

	buf.WriteString(commit.Description)
	buf.WriteString(core.Sprintf(" (%s)\n", commit.Hash))

	return buf.String()
}

// ParseCommitType extracts the type from a conventional commit subject.
// Returns empty string if not a conventional commit.
//
// t := release.ParseCommitType("feat(build): add linuxkit support") // → "feat"
func ParseCommitType(subject string) string {
	matches := conventionalCommitRegex.FindStringSubmatch(subject)
	if matches == nil {
		return ""
	}
	return core.Lower(matches[1])
}

func containsCommitType(types []string, target string) bool {
	for _, item := range types {
		if item == target {
			return true
		}
	}
	return false
}
