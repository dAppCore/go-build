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
	releaseWorkflowPathInput                     string
	releaseWorkflowWorkflowPathHyphenInput       string
	releaseWorkflowWorkflowPathSnakeInput        string
	releaseWorkflowOutputPathHyphenInput         string
	releaseWorkflowOutputPathSnakeInput          string
	releaseWorkflowOutputLegacyInput             string
	releaseWorkflowWorkflowOutputPathHyphenInput string
	releaseWorkflowWorkflowOutputPathSnakeInput  string
)

var releaseWorkflowCmd = &cli.Command{
	Use: "workflow",
	RunE: func(cmd *cli.Command, args []string) error {
		return runReleaseWorkflow(
			cmd.Context(),
			releaseWorkflowPathInput,
			releaseWorkflowWorkflowPathHyphenInput,
			releaseWorkflowWorkflowPathSnakeInput,
			releaseWorkflowOutputPathHyphenInput,
			releaseWorkflowOutputPathSnakeInput,
			releaseWorkflowOutputLegacyInput,
			releaseWorkflowWorkflowOutputPathHyphenInput,
			releaseWorkflowWorkflowOutputPathSnakeInput,
		)
	},
}

func setWorkflowI18n() {
	releaseWorkflowCmd.Short = i18n.T("cmd.build.workflow.short")
	releaseWorkflowCmd.Long = i18n.T("cmd.build.workflow.long")
}

func initWorkflowFlags() {
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowPathInput, "path", "", i18n.T("cmd.build.workflow.flag.path"))
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowWorkflowPathHyphenInput, "workflow-path", "", i18n.T("cmd.build.workflow.flag.path"))
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowWorkflowPathSnakeInput, "workflow_path", "", i18n.T("cmd.build.workflow.flag.path"))
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowOutputPathHyphenInput, "output-path", "", i18n.T("cmd.build.workflow.flag.output_path"))
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowOutputPathSnakeInput, "output_path", "", i18n.T("cmd.build.workflow.flag.output_path"))
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowOutputLegacyInput, "output", "", i18n.T("cmd.build.workflow.flag.output"))
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowWorkflowOutputPathHyphenInput, "workflow-output-path", "", i18n.T("cmd.build.workflow.flag.workflow_output_path"))
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowWorkflowOutputPathSnakeInput, "workflow_output_path", "", i18n.T("cmd.build.workflow.flag.workflow_output_path"))
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
// runReleaseWorkflow(ctx, "", "", "", "", "", "", "", "")                 // writes to .github/workflows/release.yml
// runReleaseWorkflow(ctx, "ci/release.yml", "", "", "", "", "", "", "")   // writes to ./ci/release.yml under the project root
// runReleaseWorkflow(ctx, "", "ci/release.yml", "", "", "", "", "", "")   // uses the workflow-path alias
// runReleaseWorkflow(ctx, "", "", "ci/release.yml", "", "", "", "", "")   // uses the workflow_path alias
func runReleaseWorkflow(_ context.Context, pathInput, workflowPathHyphenInput, workflowPathSnakeInput, outputPathHyphenInput, outputPathSnakeInput, outputLegacyInput, workflowOutputPathHyphenInput, workflowOutputPathSnakeInput string) error {
	projectDir, err := ax.Getwd()
	if err != nil {
		return coreerr.E("build.runReleaseWorkflow", "failed to get working directory", err)
	}

	resolvedWorkflowPath, err := resolveReleaseWorkflowInputPathAliases(
		projectDir,
		pathInput,
		workflowPathHyphenInput,
		workflowPathSnakeInput,
	)
	if err != nil {
		return err
	}

	resolvedWorkflowOutputPath, err := resolveReleaseWorkflowOutputPathAliases(
		projectDir,
		outputPathHyphenInput,
		outputPathSnakeInput,
		outputLegacyInput,
		workflowOutputPathHyphenInput,
		workflowOutputPathSnakeInput,
	)
	if err != nil {
		return err
	}

	return runReleaseWorkflowInDir(projectDir, resolvedWorkflowPath, resolvedWorkflowOutputPath)
}

// resolveReleaseWorkflowInputPathAliases keeps the CLI error wording stable while
// delegating the conflict detection to the shared build helper.
func resolveReleaseWorkflowInputPathAliases(projectDir, pathInput, workflowPathHyphenInput, workflowPathSnakeInput string) (string, error) {
	resolvedWorkflowPath, err := build.ResolveReleaseWorkflowInputPathAliases(
		io.Local,
		projectDir,
		pathInput,
		"",
		workflowPathSnakeInput,
		workflowPathHyphenInput,
	)
	if err != nil {
		return "", coreerr.E("build.runReleaseWorkflow", "workflow path aliases specify different locations", nil)
	}

	return resolvedWorkflowPath, nil
}

// resolveReleaseWorkflowOutputPathAliases keeps the CLI error wording stable while
// delegating the conflict detection to the shared build helper.
func resolveReleaseWorkflowOutputPathAliases(projectDir, outputPathHyphenInput, outputPathSnakeInput, outputLegacyInput, workflowOutputPathHyphenInput, workflowOutputPathSnakeInput string) (string, error) {
	resolvedWorkflowOutputPath, err := build.ResolveReleaseWorkflowOutputPathAliasesInProject(
		projectDir,
		outputPathHyphenInput,
		outputPathSnakeInput,
		outputLegacyInput,
		"",
		workflowOutputPathSnakeInput,
		workflowOutputPathHyphenInput,
	)
	if err != nil {
		return "", coreerr.E("build.runReleaseWorkflow", "workflow output aliases specify different locations", nil)
	}

	return resolvedWorkflowOutputPath, nil
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
