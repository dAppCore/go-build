// Package build provides project type detection and cross-compilation for the Core build system.
// This file exposes the release workflow generator and its path-resolution helpers.
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

// WriteReleaseWorkflow writes the embedded release workflow template to outputPath.
//
// build.WriteReleaseWorkflow(io.Local, "")                                        // writes .github/workflows/release.yml
// build.WriteReleaseWorkflow(io.Local, "ci/release.yml")                          // writes ./ci/release.yml under the project root
// build.WriteReleaseWorkflow(io.Local, "/tmp/repo/.github/workflows/release.yml") // writes the absolute path unchanged
func WriteReleaseWorkflow(medium io_interface.Medium, outputPath string) error {
	if outputPath == "" {
		outputPath = DefaultReleaseWorkflowPath
	}

	content, err := releaseWorkflowTemplate.ReadFile("templates/release.yml")
	if err != nil {
		return coreerr.E("build.WriteReleaseWorkflow", "failed to read embedded workflow template", err)
	}

	if err := medium.EnsureDir(ax.Dir(outputPath)); err != nil {
		return coreerr.E("build.WriteReleaseWorkflow", "failed to create release workflow directory", err)
	}

	if err := medium.Write(outputPath, string(content)); err != nil {
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
func ResolveReleaseWorkflowPath(projectDir, outputPath string) string {
	if outputPath == "" {
		return ReleaseWorkflowPath(projectDir)
	}
	if isDirectoryLikePath(outputPath) {
		if ax.IsAbs(outputPath) {
			return ax.Join(outputPath, DefaultReleaseWorkflowFileName)
		}
		return ax.Join(projectDir, outputPath, DefaultReleaseWorkflowFileName)
	}
	if !ax.IsAbs(outputPath) {
		return ax.Join(projectDir, outputPath)
	}
	return outputPath
}

// ResolveReleaseWorkflowInputPath resolves the workflow path from the CLI/API
// `path` field and its `output` alias.
//
// build.ResolveReleaseWorkflowInputPath("/tmp/project", "", "")                      // /tmp/project/.github/workflows/release.yml
// build.ResolveReleaseWorkflowInputPath("/tmp/project", "ci/release.yml", "")        // /tmp/project/ci/release.yml
// build.ResolveReleaseWorkflowInputPath("/tmp/project", "", "ci/release.yml")        // /tmp/project/ci/release.yml
// build.ResolveReleaseWorkflowInputPath("/tmp/project", "ci/release.yml", "ci.yml")  // error
func ResolveReleaseWorkflowInputPath(projectDir, path, outputPath string) (string, error) {
	if path != "" && outputPath != "" {
		resolvedPath := ResolveReleaseWorkflowPath(projectDir, path)
		resolvedOutput := ResolveReleaseWorkflowPath(projectDir, outputPath)
		if resolvedPath != resolvedOutput {
			return "", coreerr.E("build.ResolveReleaseWorkflowInputPath", "path and output specify different locations", nil)
		}
		return resolvedPath, nil
	}

	if path != "" {
		return ResolveReleaseWorkflowPath(projectDir, path), nil
	}

	if outputPath != "" {
		return ResolveReleaseWorkflowPath(projectDir, outputPath), nil
	}

	return ReleaseWorkflowPath(projectDir), nil
}

// ResolveReleaseWorkflowInputPathWithMedium resolves the workflow path and
// treats an existing directory as a directory even when the caller omits a
// trailing slash.
//
// build.ResolveReleaseWorkflowInputPathWithMedium(io.Local, "/tmp/project", "ci", "") // /tmp/project/ci/release.yml when /tmp/project/ci exists
func ResolveReleaseWorkflowInputPathWithMedium(medium io_interface.Medium, projectDir, path, outputPath string) (string, error) {
	resolve := func(input string) string {
		resolved := ResolveReleaseWorkflowPath(projectDir, input)
		if medium != nil && medium.IsDir(resolved) {
			return ax.Join(resolved, DefaultReleaseWorkflowFileName)
		}
		return resolved
	}

	if path != "" && outputPath != "" {
		resolvedPath := resolve(path)
		resolvedOutput := resolve(outputPath)
		if resolvedPath != resolvedOutput {
			return "", coreerr.E("build.ResolveReleaseWorkflowInputPath", "path and output specify different locations", nil)
		}
		return resolvedPath, nil
	}

	if path != "" {
		return resolve(path), nil
	}

	if outputPath != "" {
		return resolve(outputPath), nil
	}

	return resolve(""), nil
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
