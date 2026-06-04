// Package builders provides project builders and reusable construction helpers for
// each supported project stack.
package builders

import (
	"dappco.re/go"
	"dappco.re/go/build/pkg/build"
)

func init() {
	build.RegisterDefaultBuilderResolver(ResolveBuilder)
}

// ResolveBuilder returns a concrete builder implementation for the project type.
//
//	result := builders.ResolveBuilder(build.ProjectTypeGo)
func ResolveBuilder(projectType build.ProjectType) core.Result {
	switch projectType {
	case build.ProjectTypeWails:
		return core.Ok(NewWailsBuilder())
	case build.ProjectTypeGo:
		return core.Ok(NewGoBuilder())
	case build.ProjectTypeDocker:
		return core.Ok(NewDockerBuilder())
	case build.ProjectTypeLinuxKit:
		return core.Ok(NewLinuxKitBuilder())
	case build.ProjectTypeTaskfile:
		return core.Ok(NewTaskfileBuilder())
	case build.ProjectTypeCPP:
		return core.Ok(NewCPPBuilder())
	case build.ProjectTypeNode:
		return core.Ok(NewNodeBuilder())
	case build.ProjectTypePHP:
		return core.Ok(NewPHPBuilder())
	case build.ProjectTypePython:
		return core.Ok(NewPythonBuilder())
	case build.ProjectTypeRust:
		return core.Ok(NewRustBuilder())
	case build.ProjectTypeDocs:
		return core.Ok(NewDocsBuilder())
	default:
		return core.Fail(core.E("builders.ResolveBuilder", "unknown project type: "+string(projectType), nil))
	}
}
