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

// WriteReleaseWorkflow writes the embedded release workflow template to path.
// When path is empty, it writes to .github/workflows/release.yml.
//
// err := build.WriteReleaseWorkflow(io.Local, "")
// err := build.WriteReleaseWorkflow(io.Local, "/tmp/repo/.github/workflows/release.yml")
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
// path := build.ReleaseWorkflowPath("/home/user/project")
func ReleaseWorkflowPath(projectDir string) string {
	return ax.Join(projectDir, DefaultReleaseWorkflowPath)
}
