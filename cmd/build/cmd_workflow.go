// cmd_workflow.go implements the release workflow generation command.

package buildcmd

import (
	"context"

	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/i18n"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
	"forge.lthn.ai/core/cli/pkg/cli"
)

var (
	releaseWorkflowPath       string
	releaseWorkflowOutputPath string
)

var releaseWorkflowCmd = &cli.Command{
	Use: "workflow",
	RunE: func(cmd *cli.Command, args []string) error {
		return runReleaseWorkflow(cmd.Context(), releaseWorkflowPath, releaseWorkflowOutputPath)
	},
}

func setWorkflowI18n() {
	releaseWorkflowCmd.Short = i18n.T("cmd.build.workflow.short")
	releaseWorkflowCmd.Long = i18n.T("cmd.build.workflow.long")
}

func initWorkflowFlags() {
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowPath, "path", "", i18n.T("cmd.build.workflow.flag.path"))
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowOutputPath, "output-path", "", i18n.T("cmd.build.workflow.flag.output_path"))
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowOutputPath, "output", "", i18n.T("cmd.build.workflow.flag.output"))
}

// buildCmd := &cli.Command{Use: "build"}
// buildcmd.AddWorkflowCommand(buildCmd)
func AddWorkflowCommand(buildCmd *cli.Command) {
	setWorkflowI18n()
	initWorkflowFlags()
	buildCmd.AddCommand(releaseWorkflowCmd)
}

// runReleaseWorkflow writes the embedded release workflow into the project.
//
// buildcmd.AddWorkflowCommand(buildCmd)
// runReleaseWorkflow(ctx, "", "")                  // writes to .github/workflows/release.yml
// runReleaseWorkflow(ctx, "ci/release.yml", "")    // writes to ./ci/release.yml under the project root
// runReleaseWorkflow(ctx, "", "ci/release.yml")    // output-path and output are aliases for path
func runReleaseWorkflow(_ context.Context, workflowPath, workflowOutputPath string) error {

	projectDir, err := ax.Getwd()
	if err != nil {
		return coreerr.E("build.runReleaseWorkflow", "failed to get working directory", err)
	}

	return runReleaseWorkflowInDir(projectDir, workflowPath, workflowOutputPath)
}

// runReleaseWorkflowInDir writes the embedded release workflow into projectDir.
//
// runReleaseWorkflowInDir("/tmp/project", "", "")               // /tmp/project/.github/workflows/release.yml
// runReleaseWorkflowInDir("/tmp/project", "ci/release.yml", "") // /tmp/project/ci/release.yml
// runReleaseWorkflowInDir("/tmp/project", ".github/workflows", "") // /tmp/project/.github/workflows/release.yml
func runReleaseWorkflowInDir(projectDir, workflowPath, workflowOutputPath string) error {
	resolvedPath, err := build.ResolveReleaseWorkflowInputPathWithMedium(io.Local, projectDir, workflowPath, workflowOutputPath)
	if err != nil {
		return err
	}

	if err := io.Local.EnsureDir(ax.Dir(resolvedPath)); err != nil {
		return coreerr.E("build.runReleaseWorkflowInDir", "failed to create release workflow directory", err)
	}

	return build.WriteReleaseWorkflow(io.Local, resolvedPath)
}
