package build

import (
	"regexp"

	"dappco.re/go/core"
	coreerr "dappco.re/go/core/log"
)

var safeVersionLinkerValue = regexp.MustCompile(`^[A-Za-z0-9._+-]+$`)

// ValidateVersionIdentifier reports whether a version string is safe to embed
// into build command arguments and generated release metadata.
//
// Safe identifiers are ASCII-only and limited to characters that cannot split a
// linker flag or shell token.
func ValidateVersionIdentifier(version string) error {
	version = core.Trim(version)
	if version == "" {
		return nil
	}
	if !safeVersionLinkerValue.MatchString(version) {
		return coreerr.E("build.ValidateVersionIdentifier", "version contains unsupported characters", nil)
	}

	return nil
}

// VersionLinkerFlag returns a safe -X linker flag for injecting the build version.
// Only ASCII version strings without whitespace or shell metacharacters are accepted
// so the resulting ldflags string cannot be split into extra linker options.
//
//	flag, err := build.VersionLinkerFlag("v1.2.3")
func VersionLinkerFlag(version string) (string, error) {
	version = core.Trim(version)
	if version == "" {
		return "", nil
	}
	if err := ValidateVersionIdentifier(version); err != nil {
		return "", coreerr.E("build.VersionLinkerFlag", "version contains unsupported characters for linker flags", err)
	}

	return core.Sprintf("-X main.version=%s", version), nil
}
