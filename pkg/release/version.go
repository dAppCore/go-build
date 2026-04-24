// Package release provides release automation with changelog generation and publishing.
package release

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/core"
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
// version, err := release.DetermineVersion(".") // → "v1.2.4"
func DetermineVersion(dir string) (string, error) {
	return DetermineVersionWithContext(context.Background(), dir)
}

// DetermineVersionWithContext determines the version while honouring caller cancellation.
// It checks in order:
//  1. Git tag on HEAD
//  2. Most recent tag + increment patch
//  3. Default to v0.0.1 if no tags exist
//
// version, err := release.DetermineVersionWithContext(ctx, ".") // → "v1.2.4"
func DetermineVersionWithContext(ctx context.Context, dir string) (string, error) {
	if git := build.DetectGitHubMetadata(); git != nil && git.IsTag {
		tag := normalizeVersion(strings.TrimSpace(git.Tag))
		if !ValidateVersion(tag) {
			return "", coreerr.E("release.DetermineVersionWithContext", "unsafe release tag detected: "+tag, nil)
		}
		return tag, nil
	}

	// Check if HEAD has a tag
	headTag, err := getTagOnHeadWithContext(ctx, dir)
	if err == nil && headTag != "" {
		headTag = normalizeVersion(headTag)
		if !ValidateVersion(headTag) {
			return "", coreerr.E("release.DetermineVersionWithContext", "unsafe release tag detected: "+headTag, nil)
		}
		return headTag, nil
	}
	if err != nil && ctx.Err() != nil {
		return "", coreerr.E("release.DetermineVersionWithContext", "version lookup cancelled", ctx.Err())
	}

	// Get most recent tag
	latestTag, err := getLatestTagWithContext(ctx, dir)
	if err != nil && ctx.Err() != nil {
		return "", coreerr.E("release.DetermineVersionWithContext", "version lookup cancelled", ctx.Err())
	}
	if err != nil || latestTag == "" {
		// No tags exist, return default
		return "v0.0.1", nil
	}
	if !ValidateVersion(latestTag) {
		return "", coreerr.E("release.DetermineVersionWithContext", "unsafe release tag detected: "+latestTag, nil)
	}

	// Increment patch version
	return IncrementVersion(latestTag), nil
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

// ParseVersion parses a semver string into its components.
// Returns (major, minor, patch, prerelease, build, error).
//
// major, minor, patch, pre, build, err := release.ParseVersion("v1.2.3-alpha+001")
func ParseVersion(version string) (int, int, int, string, string, error) {
	matches := semverRegex.FindStringSubmatch(version)
	if matches == nil {
		return 0, 0, 0, "", "", coreerr.E("release.ParseVersion", "invalid semver: "+version, nil)
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])
	prerelease := matches[4]
	build := matches[5]

	return major, minor, patch, prerelease, build, nil
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
func ValidateVersionIdentifier(version string) error {
	version = strings.TrimSpace(version)
	if version == "" {
		return nil
	}
	if err := build.ValidateVersionIdentifier(version); err != nil {
		return coreerr.E("release.ValidateVersionIdentifier", "version contains unsupported characters", err)
	}

	return nil
}

// normalizeVersion ensures the version starts with 'v'.
func normalizeVersion(version string) string {
	if !core.HasPrefix(version, "v") {
		return "v" + version
	}
	return version
}

func getTagOnHeadWithContext(ctx context.Context, dir string) (string, error) {
	output, err := ax.RunDir(ctx, dir, "git", "describe", "--tags", "--exact-match", "HEAD")
	if err != nil {
		return "", err
	}
	return core.Trim(output), nil
}

func getLatestTagWithContext(ctx context.Context, dir string) (string, error) {
	output, err := ax.RunDir(ctx, dir, "git", "describe", "--tags", "--abbrev=0")
	if err != nil {
		return "", err
	}
	return core.Trim(output), nil
}

// CompareVersions compares two semver strings.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
//
// result := release.CompareVersions("v1.2.3", "v1.2.4") // → -1
func CompareVersions(a, b string) int {
	aMajor, aMinor, aPatch, aPrerelease, _, errA := ParseVersion(a)
	bMajor, bMinor, bPatch, bPrerelease, _, errB := ParseVersion(b)

	// Invalid versions are considered less than valid ones
	if errA != nil && errB != nil {
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		default:
			return 0
		}
	}
	if errA != nil {
		return -1
	}
	if errB != nil {
		return 1
	}

	// Compare major
	if aMajor != bMajor {
		if aMajor < bMajor {
			return -1
		}
		return 1
	}

	// Compare minor
	if aMinor != bMinor {
		if aMinor < bMinor {
			return -1
		}
		return 1
	}

	// Compare patch
	if aPatch != bPatch {
		if aPatch < bPatch {
			return -1
		}
		return 1
	}

	return comparePrereleaseVersions(aPrerelease, bPrerelease)
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

	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")
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
