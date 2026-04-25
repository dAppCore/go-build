package build

import (
	"regexp"

	coreerr "dappco.re/go/core/log"
)

var safeVersionString = regexp.MustCompile(`^[A-Za-z0-9._+-]+$`)

// ValidateVersionString reports whether a version string is safe to embed in
// linker flags, generated installers, and release metadata.
//
// Safe identifiers are non-empty ASCII strings limited to characters that
// cannot split a linker flag or shell token.
func ValidateVersionString(version string) error {
	if !safeVersionString.MatchString(version) {
		return coreerr.E("build.ValidateVersionString", "version must be a non-empty safe release identifier", nil)
	}

	return nil
}

// ValidateVersionIdentifier reports whether a version override is safe when a
// caller also permits the absence of a version.
func ValidateVersionIdentifier(version string) error {
	if version == "" {
		return nil
	}
	if err := ValidateVersionString(version); err != nil {
		return coreerr.E("build.ValidateVersionIdentifier", "version contains unsupported characters", err)
	}

	return nil
}
