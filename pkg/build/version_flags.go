package build

import (
	"dappco.re/go"
)

// VersionLinkerFlag returns a safe -X linker flag for injecting the build version.
// Only ASCII version strings without whitespace or shell metacharacters are accepted
// so the resulting ldflags string cannot be split into extra linker options.
//
//	flag, err := build.VersionLinkerFlag("v1.2.3")
func VersionLinkerFlag(version string) core.Result {
	if version == "" {
		return core.Ok("")
	}
	valid := ValidateVersionString(version)
	if !valid.OK {
		return core.Fail(core.E("build.VersionLinkerFlag", "version contains unsupported characters for linker flags", core.NewError(valid.Error())))
	}

	return core.Ok(core.Sprintf("-X main.version=%s", version))
}
