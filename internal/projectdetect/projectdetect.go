package projectdetect

import (
	"dappco.re/go"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/build/pkg/build/builders"
	"dappco.re/go/io"
)

type detector struct {
	projectType build.ProjectType
	builder     build.Builder
}

var fallbackDetectors = []detector{
	{projectType: build.ProjectTypeDocker, builder: builders.NewDockerBuilder()},
	{projectType: build.ProjectTypeLinuxKit, builder: builders.NewLinuxKitBuilder()},
	{projectType: build.ProjectTypeCPP, builder: builders.NewCPPBuilder()},
	{projectType: build.ProjectTypeTaskfile, builder: builders.NewTaskfileBuilder()},
}

// DetectProjectType returns the first buildable project type in detection order.
//
// projectType, err := projectdetect.DetectProjectType(io.Local, ".")
func DetectProjectType(fs io.Medium, dir string) core.Result {
	projectType := build.PrimaryType(fs, dir)
	if !projectType.OK {
		return projectType
	}
	if projectType.Value.(build.ProjectType) != "" {
		return projectType
	}

	for _, fallback := range fallbackDetectors {
		detected := fallback.builder.Detect(fs, dir)
		if !detected.OK {
			return detected
		}
		if detected.Value.(bool) {
			return core.Ok(fallback.projectType)
		}
	}

	return core.Ok(build.ProjectType(""))
}
