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
	releaseWorkflowPathFlag string
)

var releaseWorkflowCmd = &cli.Command{
	Use: "workflow",
	RunE: func(cmd *cli.Command, args []string) error {
		return runReleaseWorkflow(cmd.Context(), releaseWorkflowPathFlag)
	},
}

func setWorkflowI18n() {
	releaseWorkflowCmd.Short = i18n.T("cmd.build.workflow.short")
	releaseWorkflowCmd.Long = i18n.T("cmd.build.workflow.long")
}

func initWorkflowFlags() {
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowPathFlag, "path", "", i18n.T("cmd.build.workflow.flag.path"))
}

// AddWorkflowCommand registers the release workflow generation subcommand.
func AddWorkflowCommand(buildCmd *cli.Command) {
	setWorkflowI18n()
	initWorkflowFlags()
	buildCmd.AddCommand(releaseWorkflowCmd)
}

// runReleaseWorkflow writes the embedded release workflow into the project.
//
// buildcmd.AddWorkflowCommand(buildCmd)
// runReleaseWorkflow(ctx, "")                  // writes to .github/workflows/release.yml
// runReleaseWorkflow(ctx, "ci/release.yml")   // writes to ./ci/release.yml under the project root
func runReleaseWorkflow(ctx context.Context, path string) error {
	_ = ctx

	projectDir, err := ax.Getwd()
	if err != nil {
		return coreerr.E("build.runReleaseWorkflow", "failed to get working directory", err)
	}

	return runReleaseWorkflowInDir(projectDir, path)
}

// runReleaseWorkflowInDir writes the embedded release workflow into projectDir.
//
// runReleaseWorkflowInDir("/tmp/project", "")               // /tmp/project/.github/workflows/release.yml
// runReleaseWorkflowInDir("/tmp/project", "ci/release.yml") // /tmp/project/ci/release.yml
func runReleaseWorkflowInDir(projectDir, path string) error {
	path = build.ResolveReleaseWorkflowPath(projectDir, path)

	if err := io.Local.EnsureDir(ax.Dir(path)); err != nil {
		return coreerr.E("build.runReleaseWorkflowInDir", "failed to create release workflow directory", err)
	}

	return build.WriteReleaseWorkflow(io.Local, path)
}
