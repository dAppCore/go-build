package build

import (
	"regexp"

	"dappco.re/go"
)

var safeVersionString = regexp.MustCompile(`^[A-Za-z0-9._+-]+$`)

// ValidateVersionString reports whether a version string is safe to embed in
// linker flags, generated installers, and release metadata.
//
// Safe identifiers are non-empty ASCII strings limited to characters that
// cannot split a linker flag or shell token.
func ValidateVersionString(version string) core.Result {
	if !safeVersionString.MatchString(version) {
		return core.Fail(core.E("build.ValidateVersionString", "version must be a non-empty safe release identifier", nil))
	}

	return core.Ok(nil)
}

// ValidateVersionIdentifier reports whether a version override is safe when a
// caller also permits the absence of a version.
func ValidateVersionIdentifier(version string) core.Result {
	if version == "" {
		return core.Ok(nil)
	}
	valid := ValidateVersionString(version)
	if !valid.OK {
		return core.Fail(core.E("build.ValidateVersionIdentifier", "version contains unsupported characters", core.NewError(valid.Error())))
	}

	return core.Ok(nil)
}
