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
	releaseWorkflowPathInput                  string
	releaseWorkflowPathAliasInput             string
	releaseWorkflowPathHyphenAliasInput       string
	releaseWorkflowPathSnakeAliasInput        string
	releaseWorkflowOutputPathHyphenInput      string
	releaseWorkflowOutputPathSnakeInput       string
	releaseWorkflowOutputPathInput            string
	releaseWorkflowOutputLegacyInput          string
	releaseWorkflowOutputPathAliasInput       string
	releaseWorkflowOutputPathHyphenAliasInput string
	releaseWorkflowOutputPathSnakeAliasInput  string
	releaseWorkflowOutputHyphenAliasInput     string
	releaseWorkflowOutputSnakeAliasInput      string
)

// releaseWorkflowRequestInputs keeps the workflow alias inputs grouped by the
// public request fields they represent, rather than by call-site position.
type releaseWorkflowRequestInputs struct {
	pathInput                     string
	workflowPathInput             string
	workflowPathSnakeInput        string
	workflowPathHyphenInput       string
	outputPathInput               string
	outputPathHyphenInput         string
	outputPathSnakeInput          string
	legacyOutputInput             string
	workflowOutputPathInput       string
	workflowOutputSnakeInput      string
	workflowOutputHyphenInput     string
	workflowOutputPathHyphenInput string
	workflowOutputPathSnakeInput  string
}

// resolveReleaseWorkflowTargetPath merges the workflow path aliases and the
// workflow output aliases into one final target path.
//
// inputs := releaseWorkflowRequestInputs{pathInput: "ci/release.yml"}
// path, err := inputs.resolveReleaseWorkflowTargetPath("/tmp/project")
func (inputs releaseWorkflowRequestInputs) resolveReleaseWorkflowTargetPath(projectDir string) (string, error) {
	resolvedWorkflowPath, err := resolveReleaseWorkflowInputPathAliases(
		projectDir,
		inputs.pathInput,
		inputs.workflowPathInput,
		inputs.workflowPathSnakeInput,
		inputs.workflowPathHyphenInput,
	)
	if err != nil {
		return "", err
	}

	resolvedWorkflowOutputPath, err := resolveReleaseWorkflowOutputPathAliases(
		projectDir,
		inputs.outputPathInput,
		inputs.outputPathHyphenInput,
		inputs.outputPathSnakeInput,
		inputs.legacyOutputInput,
		inputs.workflowOutputPathInput,
		inputs.workflowOutputSnakeInput,
		inputs.workflowOutputHyphenInput,
		inputs.workflowOutputPathSnakeInput,
		inputs.workflowOutputPathHyphenInput,
	)
	if err != nil {
		return "", err
	}

	return build.ResolveReleaseWorkflowInputPathWithMedium(io.Local, projectDir, resolvedWorkflowPath, resolvedWorkflowOutputPath)
}

var releaseWorkflowCmd = &cli.Command{
	Use: "workflow",
	RunE: func(cmd *cli.Command, args []string) error {
		return runReleaseWorkflow(cmd.Context(), releaseWorkflowRequestInputs{
			pathInput:                     releaseWorkflowPathInput,
			workflowPathInput:             releaseWorkflowPathAliasInput,
			workflowPathSnakeInput:        releaseWorkflowPathSnakeAliasInput,
			workflowPathHyphenInput:       releaseWorkflowPathHyphenAliasInput,
			outputPathInput:               releaseWorkflowOutputPathInput,
			outputPathHyphenInput:         releaseWorkflowOutputPathHyphenInput,
			outputPathSnakeInput:          releaseWorkflowOutputPathSnakeInput,
			legacyOutputInput:             releaseWorkflowOutputLegacyInput,
			workflowOutputPathInput:       releaseWorkflowOutputPathAliasInput,
			workflowOutputSnakeInput:      releaseWorkflowOutputSnakeAliasInput,
			workflowOutputHyphenInput:     releaseWorkflowOutputHyphenAliasInput,
			workflowOutputPathHyphenInput: releaseWorkflowOutputPathHyphenAliasInput,
			workflowOutputPathSnakeInput:  releaseWorkflowOutputPathSnakeAliasInput,
		})
	},
}

func setWorkflowI18n() {
	releaseWorkflowCmd.Short = i18n.T("cmd.build.workflow.short")
	releaseWorkflowCmd.Long = i18n.T("cmd.build.workflow.long")
}

func initWorkflowFlags() {
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowPathInput, "path", "", i18n.T("cmd.build.workflow.flag.path"))
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowPathAliasInput, "workflowPath", "", i18n.T("cmd.build.workflow.flag.path"))
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowPathHyphenAliasInput, "workflow-path", "", i18n.T("cmd.build.workflow.flag.path"))
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowPathSnakeAliasInput, "workflow_path", "", i18n.T("cmd.build.workflow.flag.path"))
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowOutputPathInput, "outputPath", "", i18n.T("cmd.build.workflow.flag.output_path"))
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowOutputPathHyphenInput, "output-path", "", i18n.T("cmd.build.workflow.flag.output_path"))
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowOutputPathSnakeInput, "output_path", "", i18n.T("cmd.build.workflow.flag.output_path"))
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowOutputLegacyInput, "output", "", i18n.T("cmd.build.workflow.flag.output"))
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowOutputPathAliasInput, "workflowOutputPath", "", i18n.T("cmd.build.workflow.flag.workflow_output_path"))
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowOutputPathHyphenAliasInput, "workflow-output-path", "", i18n.T("cmd.build.workflow.flag.workflow_output_path"))
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowOutputPathSnakeAliasInput, "workflow_output_path", "", i18n.T("cmd.build.workflow.flag.workflow_output_path"))
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowOutputHyphenAliasInput, "workflow-output", "", i18n.T("cmd.build.workflow.flag.workflow_output_path"))
	releaseWorkflowCmd.Flags().StringVar(&releaseWorkflowOutputSnakeAliasInput, "workflow_output", "", i18n.T("cmd.build.workflow.flag.workflow_output_path"))
}

// buildCmd := &cli.Command{Use: "build"}
// buildcmd.AddWorkflowCommand(buildCmd)
func AddWorkflowCommand(buildCmd *cli.Command) {
	setWorkflowI18n()
	initWorkflowFlags()
	buildCmd.AddCommand(releaseWorkflowCmd)
}

// runReleaseWorkflow writes the embedded release workflow into the current
// project directory.
//
// runReleaseWorkflow(ctx, releaseWorkflowRequestInputs{})                                              // writes .github/workflows/release.yml
// runReleaseWorkflow(ctx, releaseWorkflowRequestInputs{pathInput: "ci/release.yml"})                  // writes ./ci/release.yml under the project root
// runReleaseWorkflow(ctx, releaseWorkflowRequestInputs{workflowPathInput: "ci/release.yml"})          // uses the workflowPath alias
// runReleaseWorkflow(ctx, releaseWorkflowRequestInputs{workflowPathSnakeInput: "ci/release.yml"})     // uses the workflow_path alias
// runReleaseWorkflow(ctx, releaseWorkflowRequestInputs{workflowPathHyphenInput: "ci/release.yml"})    // uses the workflow-path alias
// runReleaseWorkflow(ctx, releaseWorkflowRequestInputs{outputPathInput: "ci/release.yml"})            // uses the outputPath alias
// runReleaseWorkflow(ctx, releaseWorkflowRequestInputs{legacyOutputInput: "ci/release.yml"})          // uses the legacy output alias
// runReleaseWorkflow(ctx, releaseWorkflowRequestInputs{workflowOutputPathInput: "ci/release.yml"})    // uses the workflowOutputPath alias
// runReleaseWorkflow(ctx, releaseWorkflowRequestInputs{workflowOutputHyphenInput: "ci/release.yml"})  // uses the workflow-output alias
// runReleaseWorkflow(ctx, releaseWorkflowRequestInputs{workflowOutputSnakeInput: "ci/release.yml"})   // uses the workflow_output alias
// runReleaseWorkflow(ctx, releaseWorkflowRequestInputs{workflowOutputPathSnakeInput: "ci/release.yml"}) // uses the workflow_output_path alias
// runReleaseWorkflow(ctx, releaseWorkflowRequestInputs{workflowOutputPathHyphenInput: "ci/release.yml"}) // uses the workflow-output-path alias
func runReleaseWorkflow(_ context.Context, inputs releaseWorkflowRequestInputs) error {
	projectDir, err := ax.Getwd()
	if err != nil {
		return coreerr.E("build.runReleaseWorkflow", "failed to get working directory", err)
	}

	resolvedWorkflowPath, err := inputs.resolveReleaseWorkflowTargetPath(projectDir)
	if err != nil {
		return err
	}

	return build.WriteReleaseWorkflow(io.Local, resolvedWorkflowPath)
}

// resolveReleaseWorkflowInputPathAliases keeps the CLI error wording stable while
// delegating the conflict detection to the shared build helper.
func resolveReleaseWorkflowInputPathAliases(projectDir, pathInput, workflowPathInput, workflowPathSnakeInput, workflowPathHyphenInput string) (string, error) {
	resolvedWorkflowPath, err := build.ResolveReleaseWorkflowInputPathAliases(
		io.Local,
		projectDir,
		pathInput,
		workflowPathInput,
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
func resolveReleaseWorkflowOutputPathAliases(projectDir, outputPathInput, outputPathHyphenInput, outputPathSnakeInput, legacyOutputInput, workflowOutputPathInput, workflowOutputSnakeInput, workflowOutputHyphenInput, workflowOutputPathSnakeInput, workflowOutputPathHyphenInput string) (string, error) {
	resolvedWorkflowOutputPath, err := build.ResolveReleaseWorkflowOutputPathAliasesInProjectWithMedium(
		io.Local,
		projectDir,
		outputPathInput,
		outputPathHyphenInput,
		outputPathSnakeInput,
		legacyOutputInput,
		workflowOutputPathInput,
		workflowOutputSnakeInput,
		workflowOutputHyphenInput,
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

	return build.WriteReleaseWorkflow(io.Local, resolvedPath)
}
