package build

import (
	"testing"

	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflow_WriteReleaseWorkflow_Good(t *testing.T) {
	t.Run("writes the embedded template to the default path", func(t *testing.T) {
		fs := io.NewMockMedium()

		err := WriteReleaseWorkflow(fs, "")
		require.NoError(t, err)

		content, err := fs.Read(DefaultReleaseWorkflowPath)
		require.NoError(t, err)

		template, err := releaseWorkflowTemplate.ReadFile("templates/release.yml")
		require.NoError(t, err)

		assert.Equal(t, string(template), content)
		assert.Contains(t, content, "workflow_call:")
		assert.Contains(t, content, "workflow_dispatch:")
		assert.Contains(t, content, "core build --targets")
		assert.Contains(t, content, "--archive-format")
		assert.Contains(t, content, "actions/download-artifact@v4")
		assert.Contains(t, content, "command: ci")
		assert.Contains(t, content, "we-are-go-for-launch: true")
	})

	t.Run("writes to a custom path", func(t *testing.T) {
		fs := io.NewMockMedium()

		err := WriteReleaseWorkflow(fs, "custom/workflow.yml")
		require.NoError(t, err)

		content, err := fs.Read("custom/workflow.yml")
		require.NoError(t, err)
		assert.NotEmpty(t, content)
	})

	t.Run("creates parent directories on a real filesystem", func(t *testing.T) {
		projectDir := t.TempDir()
		path := ax.Join(projectDir, ".github", "workflows", "release.yml")

		err := WriteReleaseWorkflow(io.Local, path)
		require.NoError(t, err)

		content, err := io.Local.Read(path)
		require.NoError(t, err)

		template, err := releaseWorkflowTemplate.ReadFile("templates/release.yml")
		require.NoError(t, err)

		assert.Equal(t, string(template), content)
	})
}

func TestWorkflow_ReleaseWorkflowPath_Good(t *testing.T) {
	assert.Equal(t, "/tmp/project/.github/workflows/release.yml", ReleaseWorkflowPath("/tmp/project"))
}

func TestWorkflow_ResolveReleaseWorkflowPath_Good(t *testing.T) {
	t.Run("uses the conventional path when empty", func(t *testing.T) {
		assert.Equal(t, "/tmp/project/.github/workflows/release.yml", ResolveReleaseWorkflowPath("/tmp/project", ""))
	})

	t.Run("joins relative paths to the project directory", func(t *testing.T) {
		assert.Equal(t, "/tmp/project/ci/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "ci/release.yml"))
	})

	t.Run("keeps absolute paths unchanged", func(t *testing.T) {
		assert.Equal(t, "/tmp/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "/tmp/release.yml"))
	})
}
