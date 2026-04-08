// Package builders provides project builders and reusable construction helpers for
// each supported project stack.
package builders

import (
	"io/fs"

	"dappco.re/go/core/build/pkg/build"
)

// ResolveBuilder returns a concrete builder implementation for the project type.
//
//	builder, err := builders.ResolveBuilder(build.ProjectTypeGo)
func ResolveBuilder(projectType build.ProjectType) (build.Builder, error) {
	switch projectType {
	case build.ProjectTypeWails:
		return NewWailsBuilder(), nil
	case build.ProjectTypeGo:
		return NewGoBuilder(), nil
	case build.ProjectTypeDocker:
		return NewDockerBuilder(), nil
	case build.ProjectTypeLinuxKit:
		return NewLinuxKitBuilder(), nil
	case build.ProjectTypeTaskfile:
		return NewTaskfileBuilder(), nil
	case build.ProjectTypeCPP:
		return NewCPPBuilder(), nil
	case build.ProjectTypeNode:
		return NewNodeBuilder(), nil
	case build.ProjectTypePHP:
		return NewPHPBuilder(), nil
	case build.ProjectTypePython:
		return NewPythonBuilder(), nil
	case build.ProjectTypeRust:
		return NewRustBuilder(), nil
	case build.ProjectTypeDocs:
		return NewDocsBuilder(), nil
	default:
		return nil, fs.ErrNotExist
	}
}
