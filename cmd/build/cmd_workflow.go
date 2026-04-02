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
	releaseWorkflowPathInput                 string
	releaseWorkflowOutputPathInput           string
	releaseWorkflowOutputPathSnakeInput      string
	releaseWorkflowOutputLegacyInput         string
	releaseWorkflowOutputPathAliasInput      string
	releaseWorkflowOutputPathAliasSnakeInput string
)

var releaseWorkflowCmd = &cli.Command{
	Use: "workflow",
	RunE: func(cmd *cli.Command, args []string) error {
		return runReleaseWorkflow(
			cmd.Context(),
			releaseWorkflowPathInput,
			releaseWorkflowOutputPathInput,
			releaseWorkflowOutputPathSnakeInput,
			releaseWorkflowOutputLegacyInput,
			releaseWorkflowOutputPathAliasInput,
			releaseWorkflowOutputPathAliasSnakeInput,
		)
	},
}

func setWorkflowI18n() {
	releaseWorkflowCmd.Short = i18n.T("cmd.build.workflow.short")
	releaseWorkflowCmd.Long = i18n.T("cmd.build.workflow.long")
}

func initWorkflowFlags() {
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowPathInput, "path", "", i18n.T("cmd.build.workflow.flag.path"))
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowOutputPathInput, "output-path", "", i18n.T("cmd.build.workflow.flag.output_path"))
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowOutputPathSnakeInput, "output_path", "", i18n.T("cmd.build.workflow.flag.output_path"))
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowOutputLegacyInput, "output", "", i18n.T("cmd.build.workflow.flag.output"))
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowOutputPathAliasInput, "workflow-output-path", "", i18n.T("cmd.build.workflow.flag.workflow_output_path"))
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowOutputPathAliasSnakeInput, "workflow_output_path", "", i18n.T("cmd.build.workflow.flag.workflow_output_path"))
}

// buildCmd := &cli.Command{Use: "build"}
// buildcmd.AddWorkflowCommand(buildCmd)
func AddWorkflowCommand(buildCmd *cli.Command) {
	setWorkflowI18n()
	initWorkflowFlags()
	buildCmd.AddCommand(releaseWorkflowCmd)
}

// runReleaseWorkflow writes the embedded release workflow into the current project directory.
//
// buildcmd.AddWorkflowCommand(buildCmd)
// runReleaseWorkflow(ctx, "", "", "", "", "", "")               // writes to .github/workflows/release.yml
// runReleaseWorkflow(ctx, "ci/release.yml", "", "", "", "", "") // writes to ./ci/release.yml under the project root
// runReleaseWorkflow(ctx, "", "ci/release.yml", "", "", "", "") // uses the preferred explicit output path
// runReleaseWorkflow(ctx, "", "", "ci/release.yml", "", "", "") // uses the snake_case alias
// runReleaseWorkflow(ctx, "", "", "", "ci/release.yml", "", "") // uses the legacy output alias
// runReleaseWorkflow(ctx, "", "", "", "", "ci/release.yml", "") // uses the workflow-output-path alias
// runReleaseWorkflow(ctx, "", "", "", "", "", "ci/release.yml") // uses the workflow_output_path alias
func runReleaseWorkflow(_ context.Context, workflowPathInput, workflowOutputPathInput, workflowOutputPathSnakeInput, workflowOutputLegacyInput, workflowOutputPathAliasInput, workflowOutputPathAliasSnakeInput string) error {
	resolvedOutputPathInput, err := resolveWorkflowOutputPathAliases(
		workflowOutputPathInput,
		workflowOutputPathSnakeInput,
		workflowOutputLegacyInput,
		workflowOutputPathAliasInput,
		workflowOutputPathAliasSnakeInput,
	)
	if err != nil {
		return err
	}

	projectDir, err := ax.Getwd()
	if err != nil {
		return coreerr.E("build.runReleaseWorkflow", "failed to get working directory", err)
	}

	return runReleaseWorkflowInDir(projectDir, workflowPathInput, resolvedOutputPathInput)
}

// resolveWorkflowOutputPathAliases keeps the CLI error wording stable while
// delegating the conflict detection to the shared build helper.
func resolveWorkflowOutputPathAliases(outputPathInput, outputPathSnakeInput, outputLegacyInput, workflowOutputPathInput, workflowOutputPathSnakeInput string) (string, error) {
	resolvedOutputPath, err := build.ResolveReleaseWorkflowOutputPathAliases(
		outputPathInput,
		outputPathSnakeInput,
		outputLegacyInput,
		"",
		workflowOutputPathSnakeInput,
		workflowOutputPathInput,
	)
	if err != nil {
		return "", coreerr.E("build.runReleaseWorkflow", "workflow output aliases specify different locations", nil)
	}

	return resolvedOutputPath, nil
}

// runReleaseWorkflowInDir writes the embedded release workflow into projectDir.
//
// runReleaseWorkflowInDir("/tmp/project", "", "")                // /tmp/project/.github/workflows/release.yml
// runReleaseWorkflowInDir("/tmp/project", "ci/release.yml", "")  // /tmp/project/ci/release.yml
// runReleaseWorkflowInDir("/tmp/project", ".github/workflows", "") // /tmp/project/.github/workflows/release.yml
func runReleaseWorkflowInDir(projectDir, workflowPathInput, workflowOutputPathInput string) error {
	resolvedPath, err := build.ResolveReleaseWorkflowInputPathWithMedium(io.Local, projectDir, workflowPathInput, workflowOutputPathInput)
	if err != nil {
		return err
	}

	if err := io.Local.EnsureDir(ax.Dir(resolvedPath)); err != nil {
		return coreerr.E("build.runReleaseWorkflowInDir", "failed to create release workflow directory", err)
	}

	return build.WriteReleaseWorkflow(io.Local, resolvedPath)
}
