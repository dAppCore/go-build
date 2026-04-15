package build

import (
	"regexp"

	"dappco.re/go/core"
	coreerr "dappco.re/go/core/log"
)

var safeVersionLinkerValue = regexp.MustCompile(`^[A-Za-z0-9._+-]+$`)

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
	if !safeVersionLinkerValue.MatchString(version) {
		return "", coreerr.E("build.VersionLinkerFlag", "version contains unsupported characters for linker flags", nil)
	}

	return core.Sprintf("-X main.version=%s", version), nil
}
