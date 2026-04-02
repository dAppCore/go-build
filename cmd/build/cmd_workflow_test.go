package buildcmd

import (
	"testing"

	"forge.lthn.ai/core/cli/pkg/cli"

	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCmd_resolveReleaseWorkflowOutputPathInput_Good(t *testing.T) {
	t.Run("accepts the preferred output path", func(t *testing.T) {
		path, err := build.ResolveReleaseWorkflowOutputPath("ci/release.yml", "", "")
		require.NoError(t, err)
		assert.Equal(t, "ci/release.yml", path)
	})

	t.Run("accepts the snake_case output path alias", func(t *testing.T) {
		path, err := build.ResolveReleaseWorkflowOutputPath("", "ci/release.yml", "")
		require.NoError(t, err)
		assert.Equal(t, "ci/release.yml", path)
	})

	t.Run("accepts the legacy output alias", func(t *testing.T) {
		path, err := build.ResolveReleaseWorkflowOutputPath("", "", "ci/release.yml")
		require.NoError(t, err)
		assert.Equal(t, "ci/release.yml", path)
	})

	t.Run("accepts matching output aliases", func(t *testing.T) {
		path, err := build.ResolveReleaseWorkflowOutputPath("ci/release.yml", "ci/release.yml", "ci/release.yml")
		require.NoError(t, err)
		assert.Equal(t, "ci/release.yml", path)
	})
}

func TestBuildCmd_resolveReleaseWorkflowOutputPathInput_Bad(t *testing.T) {
	_, err := build.ResolveReleaseWorkflowOutputPath("ci/release.yml", "ops/release.yml", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "output aliases specify different locations")
}

func TestBuildCmd_RunReleaseWorkflow_Good(t *testing.T) {
	projectDir := t.TempDir()

	t.Run("writes to the conventional workflow path by default", func(t *testing.T) {
		err := runReleaseWorkflowInDir(projectDir, "", "")
		require.NoError(t, err)

		path := build.ReleaseWorkflowPath(projectDir)
		content, err := io.Local.Read(path)
		require.NoError(t, err)
		assert.Contains(t, content, "workflow_call:")
		assert.Contains(t, content, "workflow_dispatch:")
		assert.Contains(t, content, "--archive-format")
		assert.Contains(t, content, "actions/download-artifact@v4")
		assert.Contains(t, content, "command: ci")
	})

	t.Run("registers both path and output flags", func(t *testing.T) {
		buildCmd := &cli.Command{Use: "build"}
		AddWorkflowCommand(buildCmd)

		pathFlag := releaseWorkflowCmd.Flags().Lookup("path")
		outputPathFlag := releaseWorkflowCmd.Flags().Lookup("output-path")
		outputPathSnakeFlag := releaseWorkflowCmd.Flags().Lookup("output_path")
		outputFlag := releaseWorkflowCmd.Flags().Lookup("output")
		workflowOutputPathFlag := releaseWorkflowCmd.Flags().Lookup("workflow-output-path")
		workflowOutputPathSnakeFlag := releaseWorkflowCmd.Flags().Lookup("workflow_output_path")

		assert.NotNil(t, pathFlag)
		assert.NotNil(t, outputPathFlag)
		assert.NotNil(t, outputPathSnakeFlag)
		assert.NotNil(t, outputFlag)
		assert.NotNil(t, workflowOutputPathFlag)
		assert.NotNil(t, workflowOutputPathSnakeFlag)
		assert.NotEmpty(t, pathFlag.Usage)
		assert.NotEmpty(t, outputPathFlag.Usage)
		assert.NotEmpty(t, outputPathSnakeFlag.Usage)
		assert.NotEmpty(t, outputFlag.Usage)
		assert.NotEmpty(t, workflowOutputPathFlag.Usage)
		assert.NotEmpty(t, workflowOutputPathSnakeFlag.Usage)
		assert.NotEqual(t, pathFlag.Usage, outputFlag.Usage)
		assert.NotEqual(t, outputPathFlag.Usage, outputFlag.Usage)
		assert.Equal(t, outputPathFlag.Usage, outputPathSnakeFlag.Usage)
		assert.Equal(t, workflowOutputPathFlag.Usage, workflowOutputPathSnakeFlag.Usage)
	})

	t.Run("writes to a custom relative path", func(t *testing.T) {
		customPath := "ci/release.yml"
		err := runReleaseWorkflowInDir(projectDir, customPath, "")
		require.NoError(t, err)

		content, err := io.Local.Read(ax.Join(projectDir, customPath))
		require.NoError(t, err)
		assert.Contains(t, content, "workflow_call:")
		assert.Contains(t, content, "workflow_dispatch:")
		assert.Contains(t, content, "--archive-format")
		assert.Contains(t, content, "actions/download-artifact@v4")
		assert.Contains(t, content, "command: ci")
	})

	t.Run("writes release.yml inside a directory-style relative path", func(t *testing.T) {
		customPath := "ci/"
		err := runReleaseWorkflowInDir(projectDir, customPath, "")
		require.NoError(t, err)

		content, err := io.Local.Read(ax.Join(projectDir, "ci", "release.yml"))
		require.NoError(t, err)
		assert.Contains(t, content, "workflow_call:")
		assert.Contains(t, content, "workflow_dispatch:")
	})

	t.Run("writes release.yml inside an existing directory without a trailing slash", func(t *testing.T) {
		require.NoError(t, io.Local.EnsureDir(ax.Join(projectDir, "ops")))

		err := runReleaseWorkflowInDir(projectDir, "ops", "")
		require.NoError(t, err)

		content, err := io.Local.Read(ax.Join(projectDir, "ops", "release.yml"))
		require.NoError(t, err)
		assert.Contains(t, content, "workflow_call:")
		assert.Contains(t, content, "workflow_dispatch:")
	})

	t.Run("writes release.yml inside a bare directory-style path", func(t *testing.T) {
		err := runReleaseWorkflowInDir(projectDir, "ci", "")
		require.NoError(t, err)

		content, err := io.Local.Read(ax.Join(projectDir, "ci", "release.yml"))
		require.NoError(t, err)
		assert.Contains(t, content, "workflow_call:")
		assert.Contains(t, content, "workflow_dispatch:")
	})

	t.Run("writes release.yml inside a current-directory-prefixed directory-style path", func(t *testing.T) {
		err := runReleaseWorkflowInDir(projectDir, "./ci", "")
		require.NoError(t, err)

		content, err := io.Local.Read(ax.Join(projectDir, "ci", "release.yml"))
		require.NoError(t, err)
		assert.Contains(t, content, "workflow_call:")
		assert.Contains(t, content, "workflow_dispatch:")
	})

	t.Run("writes release.yml inside the conventional workflows directory", func(t *testing.T) {
		err := runReleaseWorkflowInDir(projectDir, ".github/workflows", "")
		require.NoError(t, err)

		content, err := io.Local.Read(ax.Join(projectDir, ".github", "workflows", "release.yml"))
		require.NoError(t, err)
		assert.Contains(t, content, "workflow_call:")
		assert.Contains(t, content, "workflow_dispatch:")
	})

	t.Run("writes release.yml inside a current-directory-prefixed workflows directory", func(t *testing.T) {
		err := runReleaseWorkflowInDir(projectDir, "./.github/workflows", "")
		require.NoError(t, err)

		content, err := io.Local.Read(ax.Join(projectDir, ".github", "workflows", "release.yml"))
		require.NoError(t, err)
		assert.Contains(t, content, "workflow_call:")
		assert.Contains(t, content, "workflow_dispatch:")
	})

	t.Run("writes to the output alias", func(t *testing.T) {
		customPath := "ci/alias-release.yml"
		err := runReleaseWorkflowInDir(projectDir, "", customPath)
		require.NoError(t, err)

		content, err := io.Local.Read(ax.Join(projectDir, customPath))
		require.NoError(t, err)
		assert.Contains(t, content, "workflow_call:")
		assert.Contains(t, content, "workflow_dispatch:")
	})

	t.Run("writes to the output-path alias", func(t *testing.T) {
		customPath := "ci/output-path-release.yml"
		err := runReleaseWorkflowInDir(projectDir, "", customPath)
		require.NoError(t, err)

		content, err := io.Local.Read(ax.Join(projectDir, customPath))
		require.NoError(t, err)
		assert.Contains(t, content, "workflow_call:")
		assert.Contains(t, content, "workflow_dispatch:")
	})

	t.Run("writes to the output_path alias", func(t *testing.T) {
		customPath := "ci/output_path-release.yml"
		err := runReleaseWorkflowInDir(projectDir, "", customPath)
		require.NoError(t, err)

		content, err := io.Local.Read(ax.Join(projectDir, customPath))
		require.NoError(t, err)
		assert.Contains(t, content, "workflow_call:")
		assert.Contains(t, content, "workflow_dispatch:")
	})
}
