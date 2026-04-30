// cmd_workflow.go implements the release workflow generation command.

package buildcmd

import (
	"context"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/internal/cmdutil"
	"dappco.re/go/build/pkg/build"
	storage "dappco.re/go/build/pkg/storage"
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
// inputs := releaseWorkflowRequestInputs{pathInput: "ci/release.yml", outputPathInput: "ci/release.yml"}
// path, err := inputs.resolveReleaseWorkflowTargetPath("/tmp/project", storage.Local)
func (inputs releaseWorkflowRequestInputs) resolveReleaseWorkflowTargetPath(projectDir string, medium storage.Medium) core.Result {
	resolvedWorkflowPath := resolveReleaseWorkflowInputPathAliases(
		projectDir,
		inputs.pathInput,
		inputs.workflowPathInput,
		inputs.workflowPathSnakeInput,
		inputs.workflowPathHyphenInput,
	)
	if !resolvedWorkflowPath.OK {
		return resolvedWorkflowPath
	}

	resolvedWorkflowOutputPath := resolveReleaseWorkflowOutputPathAliases(
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
	if !resolvedWorkflowOutputPath.OK {
		return resolvedWorkflowOutputPath
	}

	return build.ResolveReleaseWorkflowInputPathWithMedium(medium, projectDir, resolvedWorkflowPath.Value.(string), resolvedWorkflowOutputPath.Value.(string))
}

// AddWorkflowCommand registers the build/workflow subcommand.
func AddWorkflowCommand(c *core.Core) {
	c.Command("build/workflow", core.Command{
		Description: "cmd.build.workflow.long",
		Action: func(opts core.Options) core.Result {
			return runReleaseWorkflow(cmdutil.ContextOrBackground(), releaseWorkflowRequestInputs{
				pathInput:                     cmdutil.OptionString(opts, buildPathOptionKey),
				workflowPathInput:             cmdutil.OptionString(opts, "workflowPath"),
				workflowPathSnakeInput:        cmdutil.OptionString(opts, "workflow_path"),
				workflowPathHyphenInput:       cmdutil.OptionString(opts, "workflow-path"),
				outputPathInput:               cmdutil.OptionString(opts, "outputPath"),
				outputPathHyphenInput:         cmdutil.OptionString(opts, "output-path"),
				outputPathSnakeInput:          cmdutil.OptionString(opts, "output_path"),
				legacyOutputInput:             cmdutil.OptionString(opts, "output"),
				workflowOutputPathInput:       cmdutil.OptionString(opts, "workflowOutputPath"),
				workflowOutputSnakeInput:      cmdutil.OptionString(opts, "workflow_output"),
				workflowOutputHyphenInput:     cmdutil.OptionString(opts, "workflow-output"),
				workflowOutputPathHyphenInput: cmdutil.OptionString(opts, "workflow-output-path"),
				workflowOutputPathSnakeInput:  cmdutil.OptionString(opts, "workflow_output_path"),
			})
		},
	})
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
func runReleaseWorkflow(_ context.Context, inputs releaseWorkflowRequestInputs) core.Result {
	projectDirResult := ax.Getwd()
	if !projectDirResult.OK {
		return core.Fail(core.E("build.runReleaseWorkflow", "failed to get working directory", core.NewError(projectDirResult.Error())))
	}
	projectDir := projectDirResult.Value.(string)

	resolvedWorkflowPath := inputs.resolveReleaseWorkflowTargetPath(projectDir, storage.Local)
	if !resolvedWorkflowPath.OK {
		return resolvedWorkflowPath
	}

	return build.WriteReleaseWorkflow(storage.Local, resolvedWorkflowPath.Value.(string))
}

// resolveReleaseWorkflowInputPathAliases("/tmp/project", "ci/release.yml", "", "", "") // "/tmp/project/ci/release.yml"
// resolveReleaseWorkflowInputPathAliases("/tmp/project", "", "ci/release.yml", "", "") // "/tmp/project/ci/release.yml"
func resolveReleaseWorkflowInputPathAliases(projectDir, pathInput, workflowPathInput, workflowPathSnakeInput, workflowPathHyphenInput string) core.Result {
	resolvedWorkflowPath := build.ResolveReleaseWorkflowInputPathAliases(
		storage.Local,
		projectDir,
		pathInput,
		workflowPathInput,
		workflowPathSnakeInput,
		workflowPathHyphenInput,
	)
	if !resolvedWorkflowPath.OK {
		return core.Fail(core.E("build.runReleaseWorkflow", "workflow path aliases specify different locations", nil))
	}

	return resolvedWorkflowPath
}

// resolveReleaseWorkflowOutputPathAliases("/tmp/project", "ci/release.yml", "", "", "", "", "", "", "", "") // "/tmp/project/ci/release.yml"
// resolveReleaseWorkflowOutputPathAliases("/tmp/project", "", "", "", "", "ci/release.yml", "", "", "", "") // "/tmp/project/ci/release.yml"
func resolveReleaseWorkflowOutputPathAliases(projectDir, outputPathInput, outputPathHyphenInput, outputPathSnakeInput, legacyOutputInput, workflowOutputPathInput, workflowOutputSnakeInput, workflowOutputHyphenInput, workflowOutputPathSnakeInput, workflowOutputPathHyphenInput string) core.Result {
	resolvedWorkflowOutputPath := build.ResolveReleaseWorkflowOutputPathAliasesInProjectWithMedium(
		storage.Local,
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
	if !resolvedWorkflowOutputPath.OK {
		return core.Fail(core.E("build.runReleaseWorkflow", "workflow output aliases specify different locations", nil))
	}

	return resolvedWorkflowOutputPath
}

// runReleaseWorkflowInDir writes the embedded release workflow into projectDir.
//
// runReleaseWorkflowInDir("/tmp/project", "", "")                // /tmp/project/.github/workflows/release.yml
// runReleaseWorkflowInDir("/tmp/project", "ci/release.yml", "")  // /tmp/project/ci/release.yml
// runReleaseWorkflowInDir("/tmp/project", ".github/workflows", "") // /tmp/project/.github/workflows/release.yml
func runReleaseWorkflowInDir(projectDir, workflowPathInput, workflowOutputPathInput string) core.Result {
	resolvedPath := build.ResolveReleaseWorkflowInputPathWithMedium(storage.Local, projectDir, workflowPathInput, workflowOutputPathInput)
	if !resolvedPath.OK {
		return resolvedPath
	}

	return build.WriteReleaseWorkflow(storage.Local, resolvedPath.Value.(string))
}
