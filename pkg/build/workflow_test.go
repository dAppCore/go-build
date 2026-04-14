package build

import (
	"strings"
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
		assert.Contains(t, content, "build:")
		assert.Contains(t, content, "build-name:")
		assert.Contains(t, content, "build-platform:")
		assert.Contains(t, content, "go-version:")
		assert.Contains(t, content, "node-version:")
		assert.Contains(t, content, "wails-version:")
		assert.Contains(t, content, "build-tags:")
		assert.Contains(t, content, "build-obfuscate:")
		assert.Contains(t, content, "sign:")
		assert.Equal(t, 2, strings.Count(content, "sign:\n        description: Enable platform signing after build.\n        required: false\n        type: boolean\n        default: false"))
		assert.Contains(t, content, "package:")
		assert.Contains(t, content, "wails-build-webview2:")
		assert.Contains(t, content, "Setup Go")
		assert.Contains(t, content, "actions/setup-go@v5")
		assert.Contains(t, content, "Install Garble")
		assert.Contains(t, content, "mvdan.cc/garble@latest")
		assert.Contains(t, content, "Setup Node")
		assert.Contains(t, content, "actions/setup-node@v4")
		assert.Contains(t, content, "Enable Corepack")
		assert.Contains(t, content, "corepack enable")
		assert.Contains(t, content, "Install frontend dependencies")
		assert.Contains(t, content, "pnpm install --frozen-lockfile")
		assert.Contains(t, content, "yarn install --immutable")
		assert.Contains(t, content, "bun install --frozen-lockfile")
		assert.Contains(t, content, "curl -fsSL https://bun.sh/install | bash")
		assert.Contains(t, content, "npm ci")
		assert.Contains(t, content, "find . -maxdepth 3 -name package.json")
		assert.Contains(t, content, "Install Wails CLI")
		assert.Contains(t, content, "github.com/wailsapp/wails/v3/cmd/wails3")
		assert.Contains(t, content, "github.com/wailsapp/wails/v2/cmd/wails")
		assert.Contains(t, content, "Setup Python for Conan and MkDocs")
		assert.Contains(t, content, "Install Conan")
		assert.Contains(t, content, "Install MkDocs")
		assert.Contains(t, content, "actions/setup-python@v5")
		assert.Contains(t, content, "python -m pip install conan")
		assert.Contains(t, content, "python -m pip install mkdocs")
		assert.Contains(t, content, "Setup Deno")
		assert.Contains(t, content, "denoland/setup-deno@v2")
		assert.Contains(t, content, "env.DENO_ENABLE == 'true'")
		assert.Contains(t, content, "inputs.deno-build != ''")
		assert.Contains(t, content, "build-cache:")
		assert.Contains(t, content, "Restore build cache")
		assert.Contains(t, content, "actions/cache@v4")
		assert.Contains(t, content, "${{ inputs.working-directory }}/.core/cache")
		assert.Contains(t, content, "core-build-${{ runner.os }}-${{ matrix.target }}-")
		assert.Contains(t, content, "Install Linux Wails dependencies")
		assert.Contains(t, content, "libwebkit2gtk-4.0-dev")
		assert.Contains(t, content, "libwebkit2gtk-4.1-dev")
		assert.Contains(t, content, "dpkg --compare-versions")
		assert.Contains(t, content, "core build --targets")
		assert.Contains(t, content, "inputs.build-platform == '' || inputs.build-platform == matrix.target")
		assert.Contains(t, content, "--build-name")
		assert.Contains(t, content, "--build-tags")
		assert.Contains(t, content, "--build-obfuscate")
		assert.Contains(t, content, "--sign=false")
		assert.Contains(t, content, "--package=false")
		assert.Contains(t, content, "--build-cache=false")
		assert.Contains(t, content, "--wails-build-webview2")
		assert.Contains(t, content, "--archive-format")
		assert.Contains(t, content, "Compute artifact upload name")
		assert.Contains(t, content, "build_name=\"${{ inputs.build-name }}\"")
		assert.Contains(t, content, "build_name=\"${GITHUB_REPOSITORY##*/}\"")
		assert.Contains(t, content, "suffix=\"${GITHUB_SHA::7}\"")
		assert.Contains(t, content, "GITHUB_REF#refs/tags/")
		assert.Contains(t, content, "name: ${{ steps.artifact-name.outputs.value }}")
		assert.Contains(t, content, "echo \"value=${artifact_name}\" >> \"${GITHUB_OUTPUT}\"")
		assert.NotContains(t, content, "release-${artifact_name}")
		assert.Contains(t, content, "if: ${{ inputs.package }}")
		assert.Contains(t, content, "if: ${{ inputs.build && inputs.package && startsWith(github.ref, 'refs/tags/') }}")
		assert.Contains(t, content, "actions/download-artifact@v4")
		assert.NotContains(t, content, "pattern: release-*")
		assert.Contains(t, content, "command: ci")
		assert.Contains(t, content, "we-are-go-for-launch: true")
		assert.Contains(t, content, "uses: dAppCore/build@v3")
	})

	t.Run("writes to a custom path", func(t *testing.T) {
		fs := io.NewMockMedium()

		err := WriteReleaseWorkflow(fs, "custom/workflow.yml")
		require.NoError(t, err)

		content, err := fs.Read("custom/workflow.yml")
		require.NoError(t, err)
		assert.NotEmpty(t, content)
	})

	t.Run("trims surrounding whitespace from the output path", func(t *testing.T) {
		fs := io.NewMockMedium()

		err := WriteReleaseWorkflow(fs, "  ci  ")
		require.NoError(t, err)

		content, err := fs.Read("ci/release.yml")
		require.NoError(t, err)
		assert.NotEmpty(t, content)
	})

	t.Run("writes release.yml for a bare directory-style path", func(t *testing.T) {
		fs := io.NewMockMedium()

		err := WriteReleaseWorkflow(fs, "ci")
		require.NoError(t, err)

		content, err := fs.Read("ci/release.yml")
		require.NoError(t, err)
		assert.NotEmpty(t, content)
	})

	t.Run("writes release.yml inside an existing directory", func(t *testing.T) {
		projectDir := t.TempDir()
		outputDir := ax.Join(projectDir, "ci")
		require.NoError(t, ax.MkdirAll(outputDir, 0o755))

		err := WriteReleaseWorkflow(io.Local, outputDir)
		require.NoError(t, err)

		content, err := io.Local.Read(ax.Join(outputDir, DefaultReleaseWorkflowFileName))
		require.NoError(t, err)

		template, err := releaseWorkflowTemplate.ReadFile("templates/release.yml")
		require.NoError(t, err)

		assert.Equal(t, string(template), content)
	})

	t.Run("writes release.yml for directory-style output paths", func(t *testing.T) {
		fs := io.NewMockMedium()

		err := WriteReleaseWorkflow(fs, "ci/")
		require.NoError(t, err)

		content, err := fs.Read("ci/release.yml")
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

func TestWorkflow_WriteReleaseWorkflow_Bad(t *testing.T) {
	t.Run("rejects a nil filesystem medium", func(t *testing.T) {
		err := WriteReleaseWorkflow(nil, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "filesystem medium is required")
	})
}

func TestWorkflow_ReleaseWorkflowPath_Good(t *testing.T) {
	assert.Equal(t, "/tmp/project/.github/workflows/release.yml", ReleaseWorkflowPath("/tmp/project"))
}

func TestWorkflow_ResolveReleaseWorkflowOutputPathWithMedium_Good(t *testing.T) {
	t.Run("treats an existing directory as a workflow directory", func(t *testing.T) {
		fs := io.NewMockMedium()
		require.NoError(t, fs.EnsureDir("/tmp/project/ci"))

		path := ResolveReleaseWorkflowOutputPathWithMedium(fs, "/tmp/project", "ci")
		assert.Equal(t, "/tmp/project/ci/release.yml", path)
	})

	t.Run("keeps explicit file paths unchanged", func(t *testing.T) {
		fs := io.NewMockMedium()

		path := ResolveReleaseWorkflowOutputPathWithMedium(fs, "/tmp/project", "ci/release.yml")
		assert.Equal(t, "/tmp/project/ci/release.yml", path)
	})
}

func TestWorkflow_ResolveReleaseWorkflowPath_Good(t *testing.T) {
	t.Run("uses the conventional path when empty", func(t *testing.T) {
		assert.Equal(t, "/tmp/project/.github/workflows/release.yml", ResolveReleaseWorkflowPath("/tmp/project", ""))
	})

	t.Run("joins relative paths to the project directory", func(t *testing.T) {
		assert.Equal(t, "/tmp/project/ci/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "ci/release.yml"))
	})

	t.Run("treats bare relative directory names as directories", func(t *testing.T) {
		assert.Equal(t, "/tmp/project/ci/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "ci"))
	})

	t.Run("treats current-directory-prefixed directory names as directories", func(t *testing.T) {
		assert.Equal(t, "/tmp/project/ci/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "./ci"))
	})

	t.Run("treats the conventional workflows directory as a directory", func(t *testing.T) {
		assert.Equal(t, "/tmp/project/.github/workflows/release.yml", ResolveReleaseWorkflowPath("/tmp/project", ".github/workflows"))
	})

	t.Run("treats current-directory-prefixed workflows directories as directories", func(t *testing.T) {
		assert.Equal(t, "/tmp/project/.github/workflows/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "./.github/workflows"))
	})

	t.Run("keeps nested extensionless paths as files", func(t *testing.T) {
		assert.Equal(t, "/tmp/project/ci/release", ResolveReleaseWorkflowPath("/tmp/project", "ci/release"))
	})

	t.Run("treats the current directory as a workflow directory", func(t *testing.T) {
		assert.Equal(t, "/tmp/project/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "."))
	})

	t.Run("treats trailing-slash relative paths as directories", func(t *testing.T) {
		assert.Equal(t, "/tmp/project/ci/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "ci/"))
	})

	t.Run("keeps absolute paths unchanged", func(t *testing.T) {
		assert.Equal(t, "/tmp/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "/tmp/release.yml"))
	})

	t.Run("treats trailing-slash absolute paths as directories", func(t *testing.T) {
		assert.Equal(t, "/tmp/workflows/release.yml", ResolveReleaseWorkflowPath("/tmp/project", "/tmp/workflows/"))
	})
}

func TestWorkflow_ResolveReleaseWorkflowInputPath_Good(t *testing.T) {
	t.Run("uses the conventional path when both inputs are empty", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "", "")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/project/.github/workflows/release.yml", path)
	})

	t.Run("accepts path as the primary input", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "ci/release.yml", "")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/project/ci/release.yml", path)
	})

	t.Run("accepts bare directory-style path as the primary input", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "ci", "")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/project/ci/release.yml", path)
	})

	t.Run("accepts current-directory-prefixed directory-style path as the primary input", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "./ci", "")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/project/ci/release.yml", path)
	})

	t.Run("accepts the conventional workflows directory as the primary input", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", ".github/workflows", "")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/project/.github/workflows/release.yml", path)
	})

	t.Run("accepts current-directory-prefixed workflows directories as the primary input", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "./.github/workflows", "")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/project/.github/workflows/release.yml", path)
	})

	t.Run("keeps nested extensionless paths as files", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "ci/release", "")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/project/ci/release", path)
	})

	t.Run("accepts the current directory as the primary input", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", ".", "")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/project/release.yml", path)
	})

	t.Run("accepts output as an alias for path", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "", "ci/release.yml")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/project/ci/release.yml", path)
	})

	t.Run("trims surrounding whitespace from inputs", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "  ci  ", "  ")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/project/ci/release.yml", path)
	})

	t.Run("accepts matching path and output values", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "ci/release.yml", "ci/release.yml")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/project/ci/release.yml", path)
	})

	t.Run("accepts matching directory-style path and output values", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "ci/", "ci/")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/project/ci/release.yml", path)
	})
}

func TestWorkflow_ResolveReleaseWorkflowInputPath_Bad(t *testing.T) {
	t.Run("rejects conflicting path and output values", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowInputPath("/tmp/project", "ci/release.yml", "ops/release.yml")
		assert.Error(t, err)
		assert.Empty(t, path)
		assert.Contains(t, err.Error(), "path and output specify different locations")
	})
}

func TestWorkflow_ResolveReleaseWorkflowInputPathWithMedium_Good(t *testing.T) {
	t.Run("treats an existing directory as a workflow directory", func(t *testing.T) {
		fs := io.NewMockMedium()
		require.NoError(t, fs.EnsureDir("/tmp/project/ci"))

		path, err := ResolveReleaseWorkflowInputPathWithMedium(fs, "/tmp/project", "ci", "")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/project/ci/release.yml", path)
	})

	t.Run("treats a bare directory-style path as a workflow directory", func(t *testing.T) {
		fs := io.NewMockMedium()

		path, err := ResolveReleaseWorkflowInputPathWithMedium(fs, "/tmp/project", "ci", "")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/project/ci/release.yml", path)
	})

	t.Run("treats a current-directory-prefixed directory-style path as a workflow directory", func(t *testing.T) {
		fs := io.NewMockMedium()

		path, err := ResolveReleaseWorkflowInputPathWithMedium(fs, "/tmp/project", "./ci", "")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/project/ci/release.yml", path)
	})

	t.Run("treats the conventional workflows directory as a workflow directory", func(t *testing.T) {
		fs := io.NewMockMedium()

		path, err := ResolveReleaseWorkflowInputPathWithMedium(fs, "/tmp/project", ".github/workflows", "")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/project/.github/workflows/release.yml", path)
	})

	t.Run("treats current-directory-prefixed workflows directories as workflow directories", func(t *testing.T) {
		fs := io.NewMockMedium()

		path, err := ResolveReleaseWorkflowInputPathWithMedium(fs, "/tmp/project", "./.github/workflows", "")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/project/.github/workflows/release.yml", path)
	})

	t.Run("keeps a file path unchanged when the target is not a directory", func(t *testing.T) {
		fs := io.NewMockMedium()

		path, err := ResolveReleaseWorkflowInputPathWithMedium(fs, "/tmp/project", "ci/release.yml", "")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/project/ci/release.yml", path)
	})

	t.Run("normalizes matching directory aliases", func(t *testing.T) {
		fs := io.NewMockMedium()
		require.NoError(t, fs.EnsureDir("/tmp/project/ci"))

		path, err := ResolveReleaseWorkflowInputPathWithMedium(fs, "/tmp/project", "ci", "ci/")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/project/ci/release.yml", path)
	})

	t.Run("trims surrounding whitespace before resolving", func(t *testing.T) {
		fs := io.NewMockMedium()
		require.NoError(t, fs.EnsureDir("/tmp/project/ci"))

		path, err := ResolveReleaseWorkflowInputPathWithMedium(fs, "/tmp/project", "  ci  ", "  ")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/project/ci/release.yml", path)
	})
}

func TestWorkflow_ResolveReleaseWorkflowInputPathAliases_Good(t *testing.T) {
	t.Run("accepts the preferred path input", func(t *testing.T) {
		fs := io.NewMockMedium()

		path, err := ResolveReleaseWorkflowInputPathAliases(fs, "/tmp/project", "ci", "", "", "")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/project/ci/release.yml", path)
	})

	t.Run("accepts the workflowPath alias", func(t *testing.T) {
		fs := io.NewMockMedium()

		path, err := ResolveReleaseWorkflowInputPathAliases(fs, "/tmp/project", "", "ci", "", "")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/project/ci/release.yml", path)
	})

	t.Run("accepts the workflow_path alias", func(t *testing.T) {
		fs := io.NewMockMedium()

		path, err := ResolveReleaseWorkflowInputPathAliases(fs, "/tmp/project", "", "", "ci", "")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/project/ci/release.yml", path)
	})

	t.Run("accepts the workflow-path alias", func(t *testing.T) {
		fs := io.NewMockMedium()

		path, err := ResolveReleaseWorkflowInputPathAliases(fs, "/tmp/project", "", "", "", "ci")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/project/ci/release.yml", path)
	})

	t.Run("normalises matching aliases", func(t *testing.T) {
		fs := io.NewMockMedium()
		require.NoError(t, fs.EnsureDir("/tmp/project/ci"))

		path, err := ResolveReleaseWorkflowInputPathAliases(fs, "/tmp/project", "ci/", "./ci", "ci", "")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/project/ci/release.yml", path)
	})
}

func TestWorkflow_ResolveReleaseWorkflowInputPathAliases_Bad(t *testing.T) {
	fs := io.NewMockMedium()

	path, err := ResolveReleaseWorkflowInputPathAliases(fs, "/tmp/project", "ci/release.yml", "ops/release.yml", "", "")
	assert.Error(t, err)
	assert.Empty(t, path)
	assert.Contains(t, err.Error(), "path aliases specify different locations")
}

func TestWorkflow_ResolveReleaseWorkflowOutputPath_Good(t *testing.T) {
	t.Run("accepts the preferred output path", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPath("ci/release.yml", "", "")
		require.NoError(t, err)
		assert.Equal(t, "ci/release.yml", path)
	})

	t.Run("accepts the snake_case output path alias", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPath("", "ci/release.yml", "")
		require.NoError(t, err)
		assert.Equal(t, "ci/release.yml", path)
	})

	t.Run("accepts the hyphenated output path alias", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPathAliases("", "ci/release.yml", "", "", "", "", "", "", "")
		require.NoError(t, err)
		assert.Equal(t, "ci/release.yml", path)
	})

	t.Run("accepts the legacy output alias", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPath("", "", "ci/release.yml")
		require.NoError(t, err)
		assert.Equal(t, "ci/release.yml", path)
	})

	t.Run("trims surrounding whitespace from aliases", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPath("  ci/release.yml  ", "  ", "  ")
		require.NoError(t, err)
		assert.Equal(t, "ci/release.yml", path)
	})

	t.Run("accepts matching aliases", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPath("ci/release.yml", "ci/release.yml", "ci/release.yml")
		require.NoError(t, err)
		assert.Equal(t, "ci/release.yml", path)
	})

	t.Run("normalises equivalent path aliases", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPath("ci/release.yml", "./ci/release.yml", "ci/release.yml")
		require.NoError(t, err)
		assert.Equal(t, "ci/release.yml", path)
	})
}

func TestWorkflow_ResolveReleaseWorkflowOutputPath_Bad(t *testing.T) {
	path, err := ResolveReleaseWorkflowOutputPath("ci/release.yml", "ops/release.yml", "")
	assert.Error(t, err)
	assert.Empty(t, path)
	assert.Contains(t, err.Error(), "output aliases specify different locations")
}

func TestWorkflow_ResolveReleaseWorkflowOutputPathAliases_Good(t *testing.T) {
	t.Run("accepts workflowOutputPath aliases", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPathAliases("", "", "", "", "ci/release.yml", "", "", "", "")
		require.NoError(t, err)
		assert.Equal(t, "ci/release.yml", path)
	})

	t.Run("accepts the hyphenated workflowOutputPath alias", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPathAliases("", "", "", "", "", "", "ci/release.yml", "", "")
		require.NoError(t, err)
		assert.Equal(t, "ci/release.yml", path)
	})

	t.Run("accepts the workflow_output alias", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPathAliases("", "", "", "", "", "ci/release.yml", "", "", "")
		require.NoError(t, err)
		assert.Equal(t, "ci/release.yml", path)
	})

	t.Run("accepts the workflow-output alias", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPathAliases("", "", "", "", "", "", "ci/release.yml", "", "")
		require.NoError(t, err)
		assert.Equal(t, "ci/release.yml", path)
	})

	t.Run("normalises matching workflow output aliases", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPathAliases("ci/release.yml", "", "", "./ci/release.yml", "ci/release.yml", "", "", "", "")
		require.NoError(t, err)
		assert.Equal(t, "ci/release.yml", path)
	})
}

func TestWorkflow_ResolveReleaseWorkflowOutputPathAliasesInProject_Good(t *testing.T) {
	projectDir := t.TempDir()
	absolutePath := ax.Join(projectDir, "ci", "release.yml")
	absoluteDirectory := ax.Join(projectDir, "ops")
	require.NoError(t, ax.MkdirAll(absoluteDirectory, 0o755))

	t.Run("accepts the preferred output path", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPathAliasesInProject(projectDir, "ci/release.yml", "", "", "", "", "", "", "", "")
		require.NoError(t, err)
		assert.Equal(t, absolutePath, path)
	})

	t.Run("accepts an absolute workflow output alias equivalent to the project path", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPathAliasesInProject(projectDir, "", "", "", "", absolutePath, "", "", "", "")
		require.NoError(t, err)
		assert.Equal(t, absolutePath, path)
	})

	t.Run("accepts matching relative and absolute aliases", func(t *testing.T) {
		path, err := ResolveReleaseWorkflowOutputPathAliasesInProject(projectDir, "ci/release.yml", "", "", "", "", "", "", "", absolutePath)
		require.NoError(t, err)
		assert.Equal(t, absolutePath, path)
	})

	t.Run("treats an existing absolute directory as a workflow directory", func(t *testing.T) {
		fs := io.NewMockMedium()
		require.NoError(t, fs.EnsureDir(absoluteDirectory))

		path, err := ResolveReleaseWorkflowOutputPathAliasesInProjectWithMedium(fs, projectDir, "", "", "", "", absoluteDirectory, "", "", "", "")
		require.NoError(t, err)
		assert.Equal(t, ax.Join(absoluteDirectory, DefaultReleaseWorkflowFileName), path)
	})
}

func TestWorkflow_ResolveReleaseWorkflowOutputPathAliasesInProject_Bad(t *testing.T) {
	projectDir := t.TempDir()

	path, err := ResolveReleaseWorkflowOutputPathAliasesInProject(projectDir, "ci/release.yml", "", "", "", "", "", "", "", ax.Join(projectDir, "ops", "release.yml"))
	assert.Error(t, err)
	assert.Empty(t, path)
	assert.Contains(t, err.Error(), "output aliases specify different locations")
}

func TestWorkflow_ResolveReleaseWorkflowOutputPathAliases_Bad(t *testing.T) {
	path, err := ResolveReleaseWorkflowOutputPathAliases("ci/release.yml", "", "", "", "ops/release.yml", "", "", "", "")
	assert.Error(t, err)
	assert.Empty(t, path)
	assert.Contains(t, err.Error(), "output aliases specify different locations")
}
