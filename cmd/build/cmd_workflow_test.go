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

func TestBuildCmd_RunReleaseWorkflow_Good(t *testing.T) {
	projectDir := t.TempDir()

	t.Run("writes to the conventional workflow path by default", func(t *testing.T) {
		err := runReleaseWorkflowInDir(projectDir, "")
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

		assert.NotNil(t, releaseWorkflowCmd.Flags().Lookup("path"))
		assert.NotNil(t, releaseWorkflowCmd.Flags().Lookup("output"))
	})

	t.Run("writes to a custom relative path", func(t *testing.T) {
		customPath := "ci/release.yml"
		err := runReleaseWorkflowInDir(projectDir, customPath)
		require.NoError(t, err)

		content, err := io.Local.Read(ax.Join(projectDir, customPath))
		require.NoError(t, err)
		assert.Contains(t, content, "workflow_call:")
		assert.Contains(t, content, "workflow_dispatch:")
		assert.Contains(t, content, "--archive-format")
		assert.Contains(t, content, "actions/download-artifact@v4")
		assert.Contains(t, content, "command: ci")
	})
}
