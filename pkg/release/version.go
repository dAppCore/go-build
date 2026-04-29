// Package release provides release automation with changelog generation and publishing.
package release

import (
	"context"
	"regexp"
	"strconv"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	coreerr "dappco.re/go/log"
)

// semverRegex matches semantic version strings with or without 'v' prefix.
var semverRegex = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-([a-zA-Z0-9.-]+))?(?:\+([a-zA-Z0-9.-]+))?$`)

// DetermineVersion determines the version for a release.
// It checks in order:
//  1. Git tag on HEAD
//  2. Most recent tag + increment patch
//  3. Default to v0.0.1 if no tags exist
//
// result := release.DetermineVersion(".") // → "v1.2.4"
func DetermineVersion(dir string) core.Result {
	return DetermineVersionWithContext(context.Background(), dir)
}

// DetermineVersionWithContext determines the version while honouring caller cancellation.
// It checks in order:
//  1. Git tag on HEAD
//  2. Most recent tag + increment patch
//  3. Default to v0.0.1 if no tags exist
//
// result := release.DetermineVersionWithContext(ctx, ".") // → "v1.2.4"
func DetermineVersionWithContext(ctx context.Context, dir string) core.Result {
	if git := build.DetectGitHubMetadata(); git != nil && git.IsTag {
		tag := normalizeVersion(core.Trim(git.Tag))
		if !ValidateVersion(tag) {
			return core.Fail(coreerr.E("release.DetermineVersionWithContext", "unsafe release tag detected: "+tag, nil))
		}
		return core.Ok(tag)
	}

	// Check if HEAD has a tag
	headTagResult := getTagOnHeadWithContext(ctx, dir)
	if headTagResult.OK && headTagResult.Value.(string) != "" {
		headTag := headTagResult.Value.(string)
		headTag = normalizeVersion(headTag)
		if !ValidateVersion(headTag) {
			return core.Fail(coreerr.E("release.DetermineVersionWithContext", "unsafe release tag detected: "+headTag, nil))
		}
		return core.Ok(headTag)
	}
	if !headTagResult.OK && ctx.Err() != nil {
		return core.Fail(coreerr.E("release.DetermineVersionWithContext", "version lookup cancelled", ctx.Err()))
	}

	// Get most recent tag
	latestTagResult := getLatestTagWithContext(ctx, dir)
	if !latestTagResult.OK && ctx.Err() != nil {
		return core.Fail(coreerr.E("release.DetermineVersionWithContext", "version lookup cancelled", ctx.Err()))
	}
	if !latestTagResult.OK || latestTagResult.Value.(string) == "" {
		// No tags exist, return default
		return core.Ok("v0.0.1")
	}
	latestTag := latestTagResult.Value.(string)
	if !ValidateVersion(latestTag) {
		return core.Fail(coreerr.E("release.DetermineVersionWithContext", "unsafe release tag detected: "+latestTag, nil))
	}

	// Increment patch version
	return core.Ok(IncrementVersion(latestTag))
}

// IncrementVersion increments the patch version of a semver string.
//   - "v1.2.3"       → "v1.2.4"
//   - "1.2.3"        → "v1.2.4"
//   - "v1.2.3-alpha" → "v1.2.4" (strips prerelease)
//
// next := release.IncrementVersion("v1.2.3") // → "v1.2.4"
func IncrementVersion(current string) string {
	matches := semverRegex.FindStringSubmatch(current)
	if matches == nil {
		// Not a valid semver, return as-is with increment suffix
		return current + ".1"
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	// Increment patch
	patch++

	return core.Sprintf("v%d.%d.%d", major, minor, patch)
}

// IncrementMinor increments the minor version of a semver string.
//   - "v1.2.3" → "v1.3.0"
//   - "1.2.3"  → "v1.3.0"
//
// next := release.IncrementMinor("v1.2.3") // → "v1.3.0"
func IncrementMinor(current string) string {
	matches := semverRegex.FindStringSubmatch(current)
	if matches == nil {
		return current + ".1"
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])

	// Increment minor, reset patch
	minor++

	return core.Sprintf("v%d.%d.0", major, minor)
}

// IncrementMajor increments the major version of a semver string.
//   - "v1.2.3" → "v2.0.0"
//   - "1.2.3"  → "v2.0.0"
//
// next := release.IncrementMajor("v1.2.3") // → "v2.0.0"
func IncrementMajor(current string) string {
	matches := semverRegex.FindStringSubmatch(current)
	if matches == nil {
		return current + ".1"
	}

	major, _ := strconv.Atoi(matches[1])

	// Increment major, reset minor and patch
	major++

	return core.Sprintf("v%d.0.0", major)
}

// ParsedVersion holds the components of a semantic version string.
type ParsedVersion struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
	Build      string
}

// ParseVersion parses a semver string into its components.
//
// result := release.ParseVersion("v1.2.3-alpha+001")
func ParseVersion(version string) core.Result {
	parsed, ok := parseVersionParts(version)
	if !ok {
		return core.Fail(coreerr.E("release.ParseVersion", "invalid semver: "+version, nil))
	}
	return core.Ok(parsed)
}

func parseVersionParts(version string) (ParsedVersion, bool) {
	matches := semverRegex.FindStringSubmatch(version)
	if matches == nil {
		return ParsedVersion{}, false
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])
	prerelease := matches[4]
	build := matches[5]

	return ParsedVersion{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Prerelease: prerelease,
		Build:      build,
	}, true
}

// ValidateVersion checks if a string is a valid semver.
//
// if release.ValidateVersion("v1.2.3") { ... }
func ValidateVersion(version string) bool {
	return semverRegex.MatchString(version)
}

// ValidateVersionIdentifier reports whether a version override is safe to
// interpolate into release metadata and command arguments.
//
// This is intentionally looser than semver validation so release automation can
// accept safe non-semver labels such as "dev" when needed.
func ValidateVersionIdentifier(version string) core.Result {
	if version == "" {
		return core.Ok(nil)
	}
	validated := build.ValidateVersionString(version)
	if !validated.OK {
		return core.Fail(coreerr.E("release.ValidateVersionIdentifier", "version contains unsupported characters", core.NewError(validated.Error())))
	}

	return core.Ok(nil)
}

// normalizeVersion ensures the version starts with 'v'.
func normalizeVersion(version string) string {
	if !core.HasPrefix(version, "v") {
		return "v" + version
	}
	return version
}

func getTagOnHeadWithContext(ctx context.Context, dir string) core.Result {
	output := ax.RunDir(ctx, dir, "git", "describe", "--tags", "--exact-match", "HEAD")
	if !output.OK {
		return output
	}
	return core.Ok(core.Trim(output.Value.(string)))
}

func getLatestTagWithContext(ctx context.Context, dir string) core.Result {
	output := ax.RunDir(ctx, dir, "git", "describe", "--tags", "--abbrev=0")
	if !output.OK {
		return output
	}
	return core.Ok(core.Trim(output.Value.(string)))
}

// CompareVersions compares two semver strings.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
//
// result := release.CompareVersions("v1.2.3", "v1.2.4") // → -1
func CompareVersions(a, b string) int {
	aVersion, okA := parseVersionParts(a)
	bVersion, okB := parseVersionParts(b)

	// Invalid versions are considered less than valid ones
	if !okA && !okB {
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		default:
			return 0
		}
	}
	if !okA {
		return -1
	}
	if !okB {
		return 1
	}

	// Compare major
	if aVersion.Major != bVersion.Major {
		if aVersion.Major < bVersion.Major {
			return -1
		}
		return 1
	}

	// Compare minor
	if aVersion.Minor != bVersion.Minor {
		if aVersion.Minor < bVersion.Minor {
			return -1
		}
		return 1
	}

	// Compare patch
	if aVersion.Patch != bVersion.Patch {
		if aVersion.Patch < bVersion.Patch {
			return -1
		}
		return 1
	}

	return comparePrereleaseVersions(aVersion.Prerelease, bVersion.Prerelease)
}

func comparePrereleaseVersions(a, b string) int {
	switch {
	case a == "" && b == "":
		return 0
	case a == "":
		return 1
	case b == "":
		return -1
	}

	aParts := core.Split(a, ".")
	bParts := core.Split(b, ".")
	limit := len(aParts)
	if len(bParts) < limit {
		limit = len(bParts)
	}

	for i := 0; i < limit; i++ {
		if aParts[i] == bParts[i] {
			continue
		}

		aNumeric, aIsNumeric := parsePrereleaseNumber(aParts[i])
		bNumeric, bIsNumeric := parsePrereleaseNumber(bParts[i])
		switch {
		case aIsNumeric && bIsNumeric:
			if aNumeric < bNumeric {
				return -1
			}
			return 1
		case aIsNumeric:
			return -1
		case bIsNumeric:
			return 1
		case aParts[i] < bParts[i]:
			return -1
		default:
			return 1
		}
	}

	switch {
	case len(aParts) < len(bParts):
		return -1
	case len(aParts) > len(bParts):
		return 1
	default:
		return 0
	}
}

func parsePrereleaseNumber(value string) (int, bool) {
	if value == "" {
		return 0, false
	}

	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, false
		}
	}

	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}

	return n, true
}
