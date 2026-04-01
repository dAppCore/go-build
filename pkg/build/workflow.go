// Package build provides project type detection and cross-compilation for the Core build system.
// This file handles generation of the release GitHub Actions workflow from the embedded template.
package build

import (
	"embed"

	"dappco.re/go/core/build/internal/ax"
	io_interface "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

//go:embed templates/release.yml
var releaseWorkflowTemplate embed.FS

// DefaultReleaseWorkflowPath is the conventional output path for the release workflow.
//
// path := build.DefaultReleaseWorkflowPath // ".github/workflows/release.yml"
const DefaultReleaseWorkflowPath = ".github/workflows/release.yml"

// DefaultReleaseWorkflowFileName is the workflow filename used when a directory-style
// output path is supplied.
const DefaultReleaseWorkflowFileName = "release.yml"

// WriteReleaseWorkflow writes the embedded release workflow template to path.
//
// build.WriteReleaseWorkflow(io.Local, "")                                   // writes .github/workflows/release.yml
// build.WriteReleaseWorkflow(io.Local, "/tmp/repo/.github/workflows/release.yml")
func WriteReleaseWorkflow(fs io_interface.Medium, path string) error {
	if path == "" {
		path = DefaultReleaseWorkflowPath
	}

	content, err := releaseWorkflowTemplate.ReadFile("templates/release.yml")
	if err != nil {
		return coreerr.E("build.WriteReleaseWorkflow", "failed to read embedded workflow template", err)
	}

	if err := fs.EnsureDir(ax.Dir(path)); err != nil {
		return coreerr.E("build.WriteReleaseWorkflow", "failed to create release workflow directory", err)
	}

	if err := fs.Write(path, string(content)); err != nil {
		return coreerr.E("build.WriteReleaseWorkflow", "failed to write release workflow", err)
	}

	return nil
}

// ReleaseWorkflowPath joins a project directory with the conventional workflow path.
//
// build.ReleaseWorkflowPath("/home/user/project") // /home/user/project/.github/workflows/release.yml
func ReleaseWorkflowPath(projectDir string) string {
	return ax.Join(projectDir, DefaultReleaseWorkflowPath)
}

// ResolveReleaseWorkflowPath resolves the workflow output path relative to the
// project directory when the caller supplies a relative path.
//
// build.ResolveReleaseWorkflowPath("/tmp/project", "")                // /tmp/project/.github/workflows/release.yml
// build.ResolveReleaseWorkflowPath("/tmp/project", "ci/release.yml")   // /tmp/project/ci/release.yml
// build.ResolveReleaseWorkflowPath("/tmp/project", "/tmp/release.yml") // /tmp/release.yml
func ResolveReleaseWorkflowPath(projectDir, path string) string {
	if path == "" {
		return ReleaseWorkflowPath(projectDir)
	}
	if isDirectoryLikePath(path) {
		if ax.IsAbs(path) {
			return ax.Join(path, DefaultReleaseWorkflowFileName)
		}
		return ax.Join(projectDir, path, DefaultReleaseWorkflowFileName)
	}
	if !ax.IsAbs(path) {
		return ax.Join(projectDir, path)
	}
	return path
}

// ResolveReleaseWorkflowInputPath resolves the workflow path from the CLI/API
// `path` field and its `output` alias.
//
// build.ResolveReleaseWorkflowInputPath("/tmp/project", "", "")                      // /tmp/project/.github/workflows/release.yml
// build.ResolveReleaseWorkflowInputPath("/tmp/project", "ci/release.yml", "")        // /tmp/project/ci/release.yml
// build.ResolveReleaseWorkflowInputPath("/tmp/project", "", "ci/release.yml")        // /tmp/project/ci/release.yml
// build.ResolveReleaseWorkflowInputPath("/tmp/project", "ci/release.yml", "ci.yml")  // error
func ResolveReleaseWorkflowInputPath(projectDir, path, output string) (string, error) {
	if path != "" && output != "" {
		resolvedPath := ResolveReleaseWorkflowPath(projectDir, path)
		resolvedOutput := ResolveReleaseWorkflowPath(projectDir, output)
		if resolvedPath != resolvedOutput {
			return "", coreerr.E("build.ResolveReleaseWorkflowInputPath", "path and output specify different locations", nil)
		}
		return resolvedPath, nil
	}

	if path != "" {
		return ResolveReleaseWorkflowPath(projectDir, path), nil
	}

	if output != "" {
		return ResolveReleaseWorkflowPath(projectDir, output), nil
	}

	return ReleaseWorkflowPath(projectDir), nil
}

// isDirectoryLikePath reports whether a path should be treated as a directory
// rather than a file path.
func isDirectoryLikePath(path string) bool {
	if path == "" {
		return false
	}

	last := path[len(path)-1]
	return last == '/' || last == '\\'
}
