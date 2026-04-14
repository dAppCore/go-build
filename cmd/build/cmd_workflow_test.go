package buildcmd

import (
	"testing"

	"dappco.re/go/core"
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

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_Good(t *testing.T) {
	projectDir := t.TempDir()

	path, err := resolveReleaseWorkflowOutputPathAliases(projectDir, "ci/release.yml", "", "", "", "", "./ci/release.yml", "ci/release.yml", "", "")
	require.NoError(t, err)
	assert.Equal(t, ax.Join(projectDir, "ci", "release.yml"), path)
}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_CamelCaseGood(t *testing.T) {
	projectDir := t.TempDir()

	path, err := resolveReleaseWorkflowOutputPathAliases(projectDir, "ci/release.yml", "", "", "", "", "", "", "", "")
	require.NoError(t, err)
	assert.Equal(t, ax.Join(projectDir, "ci", "release.yml"), path)
}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_WorkflowCamelCaseGood(t *testing.T) {
	projectDir := t.TempDir()

	path, err := resolveReleaseWorkflowOutputPathAliases(projectDir, "", "", "", "", "ci/release.yml", "", "", "", "")
	require.NoError(t, err)
	assert.Equal(t, ax.Join(projectDir, "ci", "release.yml"), path)
}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_WorkflowHyphenGood(t *testing.T) {
	projectDir := t.TempDir()

	path, err := resolveReleaseWorkflowOutputPathAliases(projectDir, "", "", "", "", "", "ci/release.yml", "", "", "")
	require.NoError(t, err)
	assert.Equal(t, ax.Join(projectDir, "ci", "release.yml"), path)
}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_WorkflowSnakeGood(t *testing.T) {
	projectDir := t.TempDir()

	path, err := resolveReleaseWorkflowOutputPathAliases(projectDir, "", "", "", "", "", "", "ci/release.yml", "", "")
	require.NoError(t, err)
	assert.Equal(t, ax.Join(projectDir, "ci", "release.yml"), path)
}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_Bad(t *testing.T) {
	projectDir := t.TempDir()

	_, err := resolveReleaseWorkflowOutputPathAliases(projectDir, "ci/release.yml", "", "", "", "ops/release.yml", "", "", "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workflow output aliases specify different locations")
}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_HyphenatedGood(t *testing.T) {
	projectDir := t.TempDir()

	path, err := resolveReleaseWorkflowOutputPathAliases(projectDir, "", "ci/release.yml", "", "", "", "", "", "", "")
	require.NoError(t, err)
	assert.Equal(t, ax.Join(projectDir, "ci", "release.yml"), path)
}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_AbsoluteEquivalent_Good(t *testing.T) {
	projectDir := t.TempDir()
	absolutePath := ax.Join(projectDir, "ci", "release.yml")

	path, err := resolveReleaseWorkflowOutputPathAliases(projectDir, "ci/release.yml", "", "", "", "", "", "", "", absolutePath)
	require.NoError(t, err)
	assert.Equal(t, absolutePath, path)
}

func TestBuildCmd_resolveReleaseWorkflowOutputPathAliases_AbsoluteDirectory_Good(t *testing.T) {
	projectDir := t.TempDir()
	absoluteDir := ax.Join(projectDir, "ops")
	require.NoError(t, io.Local.EnsureDir(absoluteDir))

	path, err := resolveReleaseWorkflowOutputPathAliases(projectDir, "", "", "", "", absoluteDir, "", "", "", "")
	require.NoError(t, err)
	assert.Equal(t, ax.Join(absoluteDir, "release.yml"), path)
}

func TestBuildCmd_resolveReleaseWorkflowInputPathAliases_Good(t *testing.T) {
	projectDir := t.TempDir()

	path, err := resolveReleaseWorkflowInputPathAliases(projectDir, "ci/release.yml", "", "", "")
	require.NoError(t, err)
	assert.Equal(t, ax.Join(projectDir, "ci", "release.yml"), path)
}

func TestBuildCmd_resolveReleaseWorkflowInputPathAliases_WorkflowPathGood(t *testing.T) {
	projectDir := t.TempDir()

	path, err := resolveReleaseWorkflowInputPathAliases(projectDir, "", "ci/release.yml", "", "")
	require.NoError(t, err)
	assert.Equal(t, ax.Join(projectDir, "ci", "release.yml"), path)
}

func TestBuildCmd_resolveReleaseWorkflowInputPathAliases_Bad(t *testing.T) {
	projectDir := t.TempDir()

	_, err := resolveReleaseWorkflowInputPathAliases(projectDir, "ci/release.yml", "ops/release.yml", "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workflow path aliases specify different locations")
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
		assert.Contains(t, content, "build:")
		assert.Contains(t, content, "build-name:")
		assert.Contains(t, content, "build-platform:")
		assert.Contains(t, content, "go-version:")
		assert.Contains(t, content, "node-version:")
		assert.Contains(t, content, "wails-version:")
		assert.Contains(t, content, "build-tags:")
		assert.Contains(t, content, "build-obfuscate:")
		assert.Contains(t, content, "sign:")
		assert.Contains(t, content, "package:")
		assert.Contains(t, content, "wails-build-webview2:")
		assert.Contains(t, content, "Setup Go")
		assert.Contains(t, content, "actions/setup-go@v5")
		assert.Contains(t, content, "Setup Node")
		assert.Contains(t, content, "actions/setup-node@v4")
		assert.Contains(t, content, "Enable Corepack")
		assert.Contains(t, content, "Install Wails CLI")
		assert.Contains(t, content, "Install Linux Wails dependencies")
		assert.Contains(t, content, "libwebkit2gtk-4.1-dev")
		assert.Contains(t, content, "Install MkDocs")
		assert.Contains(t, content, "Setup Deno")
		assert.Contains(t, content, "build-cache:")
		assert.Contains(t, content, "Restore build cache")
		assert.Contains(t, content, "actions/cache@v4")
		assert.Contains(t, content, "inputs.build-platform == '' || inputs.build-platform == matrix.target")
		assert.Contains(t, content, "--build-name")
		assert.Contains(t, content, "--build-tags")
		assert.Contains(t, content, "--build-obfuscate")
		assert.Contains(t, content, "--sign=false")
		assert.Contains(t, content, "--package=false")
		assert.Contains(t, content, "--build-cache=false")
		assert.Contains(t, content, "--wails-build-webview2")
		assert.Contains(t, content, "--archive-format")
		assert.Contains(t, content, "if: ${{ inputs.package }}")
		assert.Contains(t, content, "if: ${{ inputs.build && inputs.package }}")
		assert.Contains(t, content, "actions/download-artifact@v4")
		assert.Contains(t, content, "command: ci")
	})

	t.Run("registers the build/workflow command", func(t *testing.T) {
		c := core.New()
		AddWorkflowCommand(c)

		result := c.Command("build/workflow")
		require.True(t, result.OK)

		command, ok := result.Value.(*core.Command)
		require.True(t, ok)
		assert.Equal(t, "build/workflow", command.Path)
		assert.Equal(t, "cmd.build.workflow.long", command.Description)
	})

	t.Run("writes to a custom relative path", func(t *testing.T) {
		customPath := "ci/release.yml"
		err := runReleaseWorkflowInDir(projectDir, customPath, "")
		require.NoError(t, err)

		content, err := io.Local.Read(ax.Join(projectDir, customPath))
		require.NoError(t, err)
		assert.Contains(t, content, "workflow_call:")
		assert.Contains(t, content, "workflow_dispatch:")
		assert.Contains(t, content, "build:")
		assert.Contains(t, content, "build-name:")
		assert.Contains(t, content, "build-platform:")
		assert.Contains(t, content, "go-version:")
		assert.Contains(t, content, "node-version:")
		assert.Contains(t, content, "wails-version:")
		assert.Contains(t, content, "build-tags:")
		assert.Contains(t, content, "sign:")
		assert.Contains(t, content, "package:")
		assert.Contains(t, content, "Setup Go")
		assert.Contains(t, content, "Setup Node")
		assert.Contains(t, content, "Enable Corepack")
		assert.Contains(t, content, "Install Wails CLI")
		assert.Contains(t, content, "Install Linux Wails dependencies")
		assert.Contains(t, content, "Install MkDocs")
		assert.Contains(t, content, "Setup Deno")
		assert.Contains(t, content, "build-cache:")
		assert.Contains(t, content, "Restore build cache")
		assert.Contains(t, content, "--sign=false")
		assert.Contains(t, content, "--package=false")
		assert.Contains(t, content, "--build-name")
		assert.Contains(t, content, "--build-tags")
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

	t.Run("writes to the workflow-output alias", func(t *testing.T) {
		customPath := "ci/workflow-output-release.yml"
		err := runReleaseWorkflowInDir(projectDir, "", customPath)
		require.NoError(t, err)

		content, err := io.Local.Read(ax.Join(projectDir, customPath))
		require.NoError(t, err)
		assert.Contains(t, content, "workflow_call:")
		assert.Contains(t, content, "workflow_dispatch:")
	})

	t.Run("writes to the workflow_output alias", func(t *testing.T) {
		customPath := "ci/workflow_output-release.yml"
		err := runReleaseWorkflowInDir(projectDir, "", customPath)
		require.NoError(t, err)

		content, err := io.Local.Read(ax.Join(projectDir, customPath))
		require.NoError(t, err)
		assert.Contains(t, content, "workflow_call:")
		assert.Contains(t, content, "workflow_dispatch:")
	})
}
