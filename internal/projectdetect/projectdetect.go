package projectdetect

import (
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
func DetectProjectType(fs io.Medium, dir string) (build.ProjectType, error) {
	projectType, err := build.PrimaryType(fs, dir)
	if err != nil {
		return "", err
	}
	if projectType != "" {
		return projectType, nil
	}

	for _, fallback := range fallbackDetectors {
		detected, err := fallback.builder.Detect(fs, dir)
		if err != nil {
			return "", err
		}
		if detected {
			return fallback.projectType, nil
		}
	}

	return "", nil
}
