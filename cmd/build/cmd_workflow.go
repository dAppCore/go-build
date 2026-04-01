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
	workflowPath string
)

var workflowCmd = &cli.Command{
	Use: "workflow",
	RunE: func(cmd *cli.Command, args []string) error {
		return runReleaseWorkflow(cmd.Context(), workflowPath)
	},
}

func setWorkflowI18n() {
	workflowCmd.Short = i18n.T("cmd.build.workflow.short")
	workflowCmd.Long = i18n.T("cmd.build.workflow.long")
}

func initWorkflowFlags() {
	workflowCmd.Flags().StringVar(&workflowPath, "path", "", i18n.T("cmd.build.workflow.flag.path"))
}

// AddWorkflowCommand registers the release workflow generation subcommand.
func AddWorkflowCommand(buildCmd *cli.Command) {
	setWorkflowI18n()
	initWorkflowFlags()
	buildCmd.AddCommand(workflowCmd)
}

// runReleaseWorkflow writes the embedded release workflow into the project.
func runReleaseWorkflow(ctx context.Context, path string) error {
	_ = ctx

	projectDir, err := ax.Getwd()
	if err != nil {
		return coreerr.E("build.runReleaseWorkflow", "failed to get working directory", err)
	}

	return runReleaseWorkflowInDir(projectDir, path)
}

// runReleaseWorkflowInDir writes the embedded release workflow into projectDir.
func runReleaseWorkflowInDir(projectDir, path string) error {
	if path == "" {
		path = build.ReleaseWorkflowPath(projectDir)
	} else if !ax.IsAbs(path) {
		path = ax.Join(projectDir, path)
	}

	if err := io.Local.EnsureDir(ax.Dir(path)); err != nil {
		return coreerr.E("build.runReleaseWorkflowInDir", "failed to create release workflow directory", err)
	}

	return build.WriteReleaseWorkflow(io.Local, path)
}
