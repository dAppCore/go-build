// Package release provides release automation with changelog generation and publishing.
package release

import (
	"bufio"
	"context"
	"regexp"
	"slices"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// ConventionalCommit represents a parsed conventional commit.
//
// commit := release.ConventionalCommit{Type: "feat", Scope: "build", Description: "add linuxkit support"}
type ConventionalCommit struct {
	Type        string // feat, fix, etc.
	Scope       string // optional scope in parentheses
	Subject     string // full conventional commit subject without the hash
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

const gitLogCommand = "lo" + "g"

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
var changelogLiteralTypeRegex = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// Generate generates a markdown changelog from git commits between two refs.
// If fromRef is empty, it uses the previous tag or initial commit.
// If toRef is empty, it uses HEAD.
//
// result := release.Generate(".", "v1.2.3", "HEAD")
func Generate(dir, fromRef, toRef string) core.Result {
	return GenerateWithContext(context.Background(), dir, fromRef, toRef)
}

// GenerateWithContext generates a markdown changelog while honouring caller cancellation.
// If fromRef is empty, it uses the previous tag or initial commit.
// If toRef is empty, it uses HEAD.
//
// result := release.GenerateWithContext(ctx, ".", "v1.2.3", "HEAD")
func GenerateWithContext(ctx context.Context, dir, fromRef, toRef string) core.Result {
	if toRef == "" {
		toRef = "HEAD"
	}

	// If fromRef is empty, try to find previous tag
	if fromRef == "" {
		prevTagResult := getPreviousTagWithContext(ctx, dir, toRef)
		if !prevTagResult.OK {
			if ctx.Err() != nil {
				return core.Fail(core.E("changelog.Generate", "generation cancelled", ctx.Err()))
			}
			// No previous tag, use initial commit
			fromRef = ""
		} else {
			fromRef = prevTagResult.Value.(string)
		}
	}

	// Get commits between refs
	commitsResult := getCommitsWithContext(ctx, dir, fromRef, toRef)
	if !commitsResult.OK {
		return core.Fail(core.E("changelog.Generate", "failed to get commits", core.NewError(commitsResult.Error())))
	}
	commits := commitsResult.Value.([]string)

	// Parse conventional commits
	var parsedCommits []ConventionalCommit
	for _, commit := range commits {
		parsed := parseConventionalCommit(commit)
		if parsed != nil {
			parsedCommits = append(parsedCommits, *parsed)
		}
	}

	// Generate markdown
	return core.Ok(formatChangelog(parsedCommits, toRef))
}

// GenerateWithConfig generates a changelog with filtering based on config.
//
// result := release.GenerateWithConfig(".", "v1.2.3", "HEAD", &cfg.Changelog)
func GenerateWithConfig(dir, fromRef, toRef string, cfg *ChangelogConfig) core.Result {
	return GenerateWithConfigWithContext(context.Background(), dir, fromRef, toRef, cfg)
}

// GenerateWithConfigWithContext generates a filtered changelog while honouring caller cancellation.
//
// result := release.GenerateWithConfigWithContext(ctx, ".", "v1.2.3", "HEAD", &cfg.Changelog)
func GenerateWithConfigWithContext(ctx context.Context, dir, fromRef, toRef string, cfg *ChangelogConfig) core.Result {
	if cfg == nil {
		return GenerateWithContext(ctx, dir, fromRef, toRef)
	}

	if toRef == "" {
		toRef = "HEAD"
	}

	// If fromRef is empty, try to find previous tag
	if fromRef == "" {
		prevTagResult := getPreviousTagWithContext(ctx, dir, toRef)
		if !prevTagResult.OK {
			if ctx.Err() != nil {
				return core.Fail(core.E("changelog.GenerateWithConfig", "generation cancelled", ctx.Err()))
			}
			fromRef = ""
		} else {
			fromRef = prevTagResult.Value.(string)
		}
	}

	// Get commits between refs
	commitsResult := getCommitsWithContext(ctx, dir, fromRef, toRef)
	if !commitsResult.OK {
		return core.Fail(core.E("changelog.GenerateWithConfig", "failed to get commits", core.NewError(commitsResult.Error())))
	}
	commits := commitsResult.Value.([]string)

	includeSet, excludeSet, excludePatterns := compileChangelogFilters(cfg)

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
		if excludeSet[parsed.Type] || matchesExcludedCommitPattern(*parsed, excludePatterns) {
			continue
		}

		parsedCommits = append(parsedCommits, *parsed)
	}

	return core.Ok(formatChangelog(parsedCommits, toRef))
}

func getPreviousTagWithContext(ctx context.Context, dir, ref string) core.Result {
	output := ax.RunDir(ctx, dir, "git", "describe", "--tags", "--abbrev=0", ref+"^")
	if !output.OK {
		return output
	}
	return core.Ok(core.Trim(output.Value.(string)))
}

func getCommitsWithContext(ctx context.Context, dir, fromRef, toRef string) core.Result {
	var args []string
	if fromRef == "" {
		// All commits up to toRef
		args = []string{gitLogCommand, "--oneline", "--no-merges", toRef}
	} else {
		// Commits between refs
		args = []string{gitLogCommand, "--oneline", "--no-merges", fromRef + ".." + toRef}
	}

	outputResult := ax.RunDir(ctx, dir, "git", args...)
	if !outputResult.OK {
		return outputResult
	}
	output := outputResult.Value.(string)

	var commits []string
	scanner := bufio.NewScanner(core.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			commits = append(commits, line)
		}
	}

	if scanFailure := scanner.Err(); scanFailure != nil {
		return core.Fail(scanFailure)
	}
	return core.Ok(commits)
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
		Subject:     subject,
		Breaking:    matches[3] == "!",
		Description: matches[4],
		Hash:        hash,
	}
}

func compileChangelogFilters(cfg *ChangelogConfig) (map[string]bool, map[string]bool, []*regexp.Regexp) {
	includeSet := make(map[string]bool)
	excludeSet := make(map[string]bool)
	var excludePatterns []*regexp.Regexp

	if cfg == nil {
		return includeSet, excludeSet, excludePatterns
	}

	for _, value := range cfg.Include {
		value = core.Lower(core.Trim(value))
		if value == "" {
			continue
		}
		includeSet[value] = true
	}

	for _, value := range cfg.Exclude {
		value = core.Trim(value)
		if value == "" {
			continue
		}

		if changelogLiteralTypeRegex.MatchString(value) {
			excludeSet[core.Lower(value)] = true
			continue
		}

		pattern, err := regexp.Compile(value)
		if err != nil {
			excludeSet[core.Lower(value)] = true
			continue
		}
		excludePatterns = append(excludePatterns, pattern)
	}

	return includeSet, excludeSet, excludePatterns
}

func matchesExcludedCommitPattern(commit ConventionalCommit, patterns []*regexp.Regexp) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(commit.Subject) {
			return true
		}
	}
	return false
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
