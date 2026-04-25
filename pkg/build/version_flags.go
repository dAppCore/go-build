package build

import (
	"dappco.re/go/core"
	coreerr "dappco.re/go/log"
)

// VersionLinkerFlag returns a safe -X linker flag for injecting the build version.
// Only ASCII version strings without whitespace or shell metacharacters are accepted
// so the resulting ldflags string cannot be split into extra linker options.
//
//	flag, err := build.VersionLinkerFlag("v1.2.3")
func VersionLinkerFlag(version string) (string, error) {
	if version == "" {
		return "", nil
	}
	if err := ValidateVersionString(version); err != nil {
		return "", coreerr.E("build.VersionLinkerFlag", "version contains unsupported characters for linker flags", err)
	}

	return core.Sprintf("-X main.version=%s", version), nil
}
