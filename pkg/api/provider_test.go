// SPDX-Licence-Identifier: EUPL-1.2

package api

import (
	"bytes"
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	coreapi "dappco.re/go/core/api"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_BuildProviderIdentity_Good(t *testing.T) {
	p := NewProvider(".", nil)

	assert.Equal(t, "build", p.Name())
	assert.Equal(t, "/api/v1/build", p.BasePath())
}

func TestProvider_BuildProviderElement_Good(t *testing.T) {
	p := NewProvider(".", nil)
	el := p.Element()

	assert.Equal(t, "core-build-panel", el.Tag)
	assert.Equal(t, "/assets/core-build.js", el.Source)
}

func TestProvider_BuildProviderChannels_Good(t *testing.T) {
	p := NewProvider(".", nil)
	channels := p.Channels()

	assert.Contains(t, channels, "build.started")
	assert.Contains(t, channels, "build.complete")
	assert.Contains(t, channels, "build.failed")
	assert.Contains(t, channels, "release.started")
	assert.Contains(t, channels, "release.complete")
	assert.Contains(t, channels, "workflow.generated")
	assert.Contains(t, channels, "sdk.generated")
	assert.Len(t, channels, 7)
}

func TestProvider_BuildProviderDescribe_Good(t *testing.T) {
	p := NewProvider(".", nil)
	routes := p.Describe()

	// Should have 10 endpoint descriptions
	assert.Len(t, routes, 10)

	// Verify key routes exist
	paths := make(map[string]string)
	for _, r := range routes {
		paths[r.Path] = r.Method
	}

	assert.Equal(t, "GET", paths["/config"])
	assert.Equal(t, "GET", paths["/discover"])
	assert.Equal(t, "POST", paths["/build"])
	assert.Equal(t, "GET", paths["/artifacts"])
	assert.Equal(t, "GET", paths["/release/version"])
	assert.Equal(t, "GET", paths["/release/changelog"])
	assert.Equal(t, "POST", paths["/release"])
	assert.Equal(t, "POST", paths["/release/workflow"])
	assert.Equal(t, "GET", paths["/sdk/diff"])
	assert.Equal(t, "POST", paths["/sdk/generate"])

	var workflowRoute *coreapi.RouteDescription
	for i := range routes {
		if routes[i].Path == "/release/workflow" {
			workflowRoute = &routes[i]
			break
		}
	}

	require.NotNil(t, workflowRoute)
	require.NotNil(t, workflowRoute.RequestBody)

	properties, ok := workflowRoute.RequestBody["properties"].(map[string]any)
	require.True(t, ok)

	pathSchema, ok := properties["path"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", pathSchema["type"])
	assert.Equal(t, "Preferred workflow path input, relative to the project directory or absolute.", pathSchema["description"])

	workflowPathSchema, ok := properties["workflowPath"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", workflowPathSchema["type"])
	assert.Equal(t, "Predictable alias for path, relative to the project directory or absolute.", workflowPathSchema["description"])

	workflowPathSnakeSchema, ok := properties["workflow_path"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", workflowPathSnakeSchema["type"])
	assert.Equal(t, "Snake_case alias for workflowPath.", workflowPathSnakeSchema["description"])

	workflowPathHyphenSchema, ok := properties["workflow-path"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", workflowPathHyphenSchema["type"])
	assert.Equal(t, "Hyphenated alias for workflowPath.", workflowPathHyphenSchema["description"])

	outputSchema, ok := properties["output"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", outputSchema["type"])
	assert.Equal(t, "Legacy alias for outputPath.", outputSchema["description"])

	outputPathSchema, ok := properties["outputPath"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", outputPathSchema["type"])
	assert.Equal(t, "Preferred explicit workflow output path, relative to the project directory or absolute.", outputPathSchema["description"])

	outputPathHyphenSchema, ok := properties["output-path"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", outputPathHyphenSchema["type"])
	assert.Equal(t, "Hyphenated alias for outputPath.", outputPathHyphenSchema["description"])

	workflowOutputPathSchema, ok := properties["workflowOutputPath"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", workflowOutputPathSchema["type"])
	assert.Equal(t, "Predictable alias for outputPath, relative to the project directory or absolute.", workflowOutputPathSchema["description"])

	workflowOutputSnakeSchema, ok := properties["workflow_output"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", workflowOutputSnakeSchema["type"])
	assert.Equal(t, "Snake_case alias for workflowOutputPath.", workflowOutputSnakeSchema["description"])

	workflowOutputHyphenSchema, ok := properties["workflow-output"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", workflowOutputHyphenSchema["type"])
	assert.Equal(t, "Hyphenated alias for workflowOutputPath.", workflowOutputHyphenSchema["description"])

	workflowOutputPathSnakeSchema, ok := properties["workflow_output_path"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", workflowOutputPathSnakeSchema["type"])
	assert.Equal(t, "Snake_case alias for workflowOutputPath.", workflowOutputPathSnakeSchema["description"])

	workflowOutputPathHyphenSchema, ok := properties["workflow-output-path"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", workflowOutputPathHyphenSchema["type"])
	assert.Equal(t, "Hyphenated alias for workflowOutputPath.", workflowOutputPathHyphenSchema["description"])

	outputPathSnakeSchema, ok := properties["output_path"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", outputPathSnakeSchema["type"])
	assert.Equal(t, "Snake_case alias for outputPath.", outputPathSnakeSchema["description"])
}

func TestProvider_ReleaseWorkflowRequestResolvedOutputPath_Good(t *testing.T) {
	projectDir := t.TempDir()
	absoluteDir := ax.Join(projectDir, "ops")
	require.NoError(t, io.Local.EnsureDir(absoluteDir))

	req := ReleaseWorkflowRequest{
		WorkflowOutputPath: absoluteDir,
	}

	path, err := req.resolveOutputPath(projectDir, io.Local)
	require.NoError(t, err)
	assert.Equal(t, ax.Join(absoluteDir, "release.yml"), path)
}

func TestProvider_ReleaseWorkflowRequestResolvedOutputPathAliases_Good(t *testing.T) {
	projectDir := t.TempDir()

	req := ReleaseWorkflowRequest{
		WorkflowOutputSnake:  "ci/workflow-output.yml",
		WorkflowOutputHyphen: "ci/workflow-output.yml",
	}

	path, err := req.resolveOutputPath(projectDir, io.Local)
	require.NoError(t, err)
	assert.Equal(t, ax.Join(projectDir, "ci", "workflow-output.yml"), path)
}

func TestProvider_BuildProviderDefaultProjectDir_Good(t *testing.T) {
	p := NewProvider("", nil)
	assert.Equal(t, ".", p.projectDir)
}

func TestProvider_BuildProviderCustomProjectDir_Good(t *testing.T) {
	p := NewProvider("/tmp/myproject", nil)
	assert.Equal(t, "/tmp/myproject", p.projectDir)
}

func TestProvider_BuildProviderNilHub_Good(t *testing.T) {
	p := NewProvider(".", nil)
	// emitEvent should not panic with nil hub
	p.emitEvent("build.started", map[string]any{"test": true})
}

func TestProvider_GetBuilderSupportedTypes_Good(t *testing.T) {
	cases := []struct {
		projectType build.ProjectType
		name        string
	}{
		{build.ProjectTypeGo, "go"},
		{build.ProjectTypeWails, "wails"},
		{build.ProjectTypeNode, "node"},
		{build.ProjectTypePHP, "php"},
		{build.ProjectTypePython, "python"},
		{build.ProjectTypeRust, "rust"},
		{build.ProjectTypeDocs, "docs"},
		{build.ProjectTypeCPP, "cpp"},
		{build.ProjectTypeDocker, "docker"},
		{build.ProjectTypeLinuxKit, "linuxkit"},
		{build.ProjectTypeTaskfile, "taskfile"},
	}

	for _, tc := range cases {
		t.Run(string(tc.projectType), func(t *testing.T) {
			b, err := getBuilder(tc.projectType)
			require.NoError(t, err)
			assert.Equal(t, tc.name, b.Name())
		})
	}
}

func TestProvider_GetBuilderUnsupportedType_Bad(t *testing.T) {
	_, err := getBuilder(build.ProjectType("unknown"))
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestProvider_BuildProviderResolveDir_Good(t *testing.T) {
	p := NewProvider("/tmp", nil)
	dir, err := p.resolveDir()
	require.NoError(t, err)
	assert.Equal(t, "/tmp", dir)
}

func TestProvider_BuildProviderResolveDirRelative_Good(t *testing.T) {
	p := NewProvider(".", nil)
	dir, err := p.resolveDir()
	require.NoError(t, err)
	// Should return an absolute path
	assert.True(t, len(dir) > 1 && dir[0] == '/')
}

func TestProvider_BuildProviderMediumSet_Good(t *testing.T) {
	p := NewProvider(".", nil)
	assert.NotNil(t, p.medium, "medium should be set to io.Local")
}

func TestProvider_GetConfig_UsesSnakeCaseJSONKeys_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	require.NoError(t, io.Local.EnsureDir(ax.Join(projectDir, ".core")))
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, ".core", "build.yaml"), []byte(`
version: 1
project:
  name: Demo
  binary: demo
build:
  type: go
  cgo: true
  cache:
    enabled: true
    dir: cache-meta
    key_prefix: demo
    paths:
      - cache/go-build
apple:
  bundle_id: ai.lthn.demo
  xcode_cloud:
    workflow: Release
sign:
  enabled: true
  macos:
    identity: "Developer ID Application: Demo"
`), 0o644))

	p := NewProvider(projectDir, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/config", nil)

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.getConfig(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)

	body := recorder.Body.String()
	assert.Contains(t, body, `"config":`)
	assert.Contains(t, body, `"version":1`)
	assert.Contains(t, body, `"project":{"name":"Demo"`)
	assert.Contains(t, body, `"build":{"type":"go","cgo":true`)
	assert.Contains(t, body, `"cache":{"enabled":true,"dir":"cache-meta","key_prefix":"demo","paths":["`)
	assert.Contains(t, body, `"apple":{"bundle_id":"ai.lthn.demo"`)
	assert.Contains(t, body, `"xcode_cloud":{"workflow":"Release"`)
	assert.Contains(t, body, `"sign":{"enabled":true`)
	assert.Contains(t, body, `"macos":{"identity":"Developer ID Application: Demo"`)
	assert.NotContains(t, body, `"Version":`)
	assert.NotContains(t, body, `"Project":`)
	assert.NotContains(t, body, `"XcodeCloud":`)
	assert.NotContains(t, body, `"MacOS":`)
}

func TestProvider_ResolveProjectType_Good(t *testing.T) {
	t.Run("honours explicit build type override", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example"), 0o644))

		projectType, err := resolveProjectType(io.Local, dir, "docker")
		require.NoError(t, err)
		assert.Equal(t, build.ProjectTypeDocker, projectType)
	})

	t.Run("falls back to detection when build type is empty", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example"), 0o644))

		projectType, err := resolveProjectType(io.Local, dir, "")
		require.NoError(t, err)
		assert.Equal(t, build.ProjectTypeGo, projectType)
	})
}

func TestProvider_GenerateReleaseWorkflow_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	p := NewProvider(projectDir, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release/workflow", bytes.NewBufferString(`{}`))
	request.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.generateReleaseWorkflow(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)

	path := build.ReleaseWorkflowPath(projectDir)
	content, err := io.Local.Read(path)
	require.NoError(t, err)
	assert.Contains(t, content, "workflow_call:")
	assert.Contains(t, content, "workflow_dispatch:")
}

func TestProvider_GenerateReleaseWorkflow_CustomPath_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	p := NewProvider(projectDir, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release/workflow", bytes.NewBufferString(`{"path":"ci/release.yml"}`))
	request.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.generateReleaseWorkflow(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)

	path := ax.Join(projectDir, "ci", "release.yml")
	content, err := io.Local.Read(path)
	require.NoError(t, err)
	assert.Contains(t, content, "workflow_call:")
	assert.Contains(t, content, "workflow_dispatch:")
}

func TestProvider_GenerateReleaseWorkflow_WorkflowPath_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	p := NewProvider(projectDir, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release/workflow", bytes.NewBufferString(`{"workflowPath":"ci/workflow-path.yml"}`))
	request.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.generateReleaseWorkflow(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)

	path := ax.Join(projectDir, "ci", "workflow-path.yml")
	content, err := io.Local.Read(path)
	require.NoError(t, err)
	assert.Contains(t, content, "workflow_call:")
	assert.Contains(t, content, "workflow_dispatch:")
}

func TestProvider_GenerateReleaseWorkflow_WorkflowPathSnake_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	p := NewProvider(projectDir, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release/workflow", bytes.NewBufferString(`{"workflow_path":"ci/workflow-path.yml"}`))
	request.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.generateReleaseWorkflow(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)

	path := ax.Join(projectDir, "ci", "workflow-path.yml")
	content, err := io.Local.Read(path)
	require.NoError(t, err)
	assert.Contains(t, content, "workflow_call:")
	assert.Contains(t, content, "workflow_dispatch:")
}

func TestProvider_GenerateReleaseWorkflow_WorkflowPathHyphen_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	p := NewProvider(projectDir, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release/workflow", bytes.NewBufferString(`{"workflow-path":"ci/workflow-path.yml"}`))
	request.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.generateReleaseWorkflow(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)

	path := ax.Join(projectDir, "ci", "workflow-path.yml")
	content, err := io.Local.Read(path)
	require.NoError(t, err)
	assert.Contains(t, content, "workflow_call:")
	assert.Contains(t, content, "workflow_dispatch:")
}

func TestProvider_GenerateReleaseWorkflow_ConflictingWorkflowPathAliases_Bad(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	p := NewProvider(projectDir, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release/workflow", bytes.NewBufferString(`{"path":"ci/workflow-path.yml","workflowPath":"ops/workflow-path.yml"}`))
	request.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.generateReleaseWorkflow(ctx)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	path := build.ReleaseWorkflowPath(projectDir)
	_, err := io.Local.Read(path)
	assert.Error(t, err)
}

func TestProvider_GenerateReleaseWorkflow_OutputAlias_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	p := NewProvider(projectDir, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release/workflow", bytes.NewBufferString(`{"output":"ci/release.yml"}`))
	request.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.generateReleaseWorkflow(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)

	path := ax.Join(projectDir, "ci", "release.yml")
	content, err := io.Local.Read(path)
	require.NoError(t, err)
	assert.Contains(t, content, "workflow_call:")
	assert.Contains(t, content, "workflow_dispatch:")
}

func TestProvider_GenerateReleaseWorkflow_OutputPath_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	p := NewProvider(projectDir, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release/workflow", bytes.NewBufferString(`{"outputPath":"ci/output-path.yml"}`))
	request.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.generateReleaseWorkflow(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)

	path := ax.Join(projectDir, "ci", "output-path.yml")
	content, err := io.Local.Read(path)
	require.NoError(t, err)
	assert.Contains(t, content, "workflow_call:")
	assert.Contains(t, content, "workflow_dispatch:")
}

func TestProvider_GenerateReleaseWorkflow_OutputPathHyphen_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	p := NewProvider(projectDir, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release/workflow", bytes.NewBufferString(`{"output-path":"ci/output-path.yml"}`))
	request.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.generateReleaseWorkflow(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)

	path := ax.Join(projectDir, "ci", "output-path.yml")
	content, err := io.Local.Read(path)
	require.NoError(t, err)
	assert.Contains(t, content, "workflow_call:")
	assert.Contains(t, content, "workflow_dispatch:")
}

func TestProvider_GenerateReleaseWorkflow_OutputPathSnake_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	p := NewProvider(projectDir, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release/workflow", bytes.NewBufferString(`{"output_path":"ci/output-path.yml"}`))
	request.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.generateReleaseWorkflow(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)

	path := ax.Join(projectDir, "ci", "output-path.yml")
	content, err := io.Local.Read(path)
	require.NoError(t, err)
	assert.Contains(t, content, "workflow_call:")
	assert.Contains(t, content, "workflow_dispatch:")
}

func TestProvider_GenerateReleaseWorkflow_WorkflowOutputPath_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	p := NewProvider(projectDir, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release/workflow", bytes.NewBufferString(`{"workflowOutputPath":"ci/workflow-output-path.yml"}`))
	request.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.generateReleaseWorkflow(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)

	path := ax.Join(projectDir, "ci", "workflow-output-path.yml")
	content, err := io.Local.Read(path)
	require.NoError(t, err)
	assert.Contains(t, content, "workflow_call:")
	assert.Contains(t, content, "workflow_dispatch:")
}

func TestProvider_GenerateReleaseWorkflow_WorkflowOutputSnake_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	p := NewProvider(projectDir, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release/workflow", bytes.NewBufferString(`{"workflow_output":"ci/workflow-output.yml"}`))
	request.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.generateReleaseWorkflow(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)

	path := ax.Join(projectDir, "ci", "workflow-output.yml")
	content, err := io.Local.Read(path)
	require.NoError(t, err)
	assert.Contains(t, content, "workflow_call:")
	assert.Contains(t, content, "workflow_dispatch:")
}

func TestProvider_GenerateReleaseWorkflow_WorkflowOutputPathSnake_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	p := NewProvider(projectDir, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release/workflow", bytes.NewBufferString(`{"workflow_output_path":"ci/workflow-output-path.yml"}`))
	request.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.generateReleaseWorkflow(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)

	path := ax.Join(projectDir, "ci", "workflow-output-path.yml")
	content, err := io.Local.Read(path)
	require.NoError(t, err)
	assert.Contains(t, content, "workflow_call:")
	assert.Contains(t, content, "workflow_dispatch:")
}

func TestProvider_GenerateReleaseWorkflow_WorkflowOutputPathAbsoluteEquivalent_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	p := NewProvider(projectDir, nil)

	absolutePath := ax.Join(projectDir, "ci", "workflow-output-path.yml")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release/workflow", bytes.NewBufferString(`{"outputPath":"ci/workflow-output-path.yml","workflowOutputPath":"`+absolutePath+`"}`))
	request.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.generateReleaseWorkflow(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)

	path := ax.Join(projectDir, "ci", "workflow-output-path.yml")
	content, err := io.Local.Read(path)
	require.NoError(t, err)
	assert.Contains(t, content, "workflow_call:")
	assert.Contains(t, content, "workflow_dispatch:")
}

func TestProvider_GenerateReleaseWorkflow_WorkflowOutputPathHyphen_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	p := NewProvider(projectDir, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release/workflow", bytes.NewBufferString(`{"workflow-output-path":"ci/workflow-output-path.yml"}`))
	request.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.generateReleaseWorkflow(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)

	path := ax.Join(projectDir, "ci", "workflow-output-path.yml")
	content, err := io.Local.Read(path)
	require.NoError(t, err)
	assert.Contains(t, content, "workflow_call:")
	assert.Contains(t, content, "workflow_dispatch:")
}

func TestProvider_GenerateReleaseWorkflow_WorkflowOutputHyphen_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	p := NewProvider(projectDir, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release/workflow", bytes.NewBufferString(`{"workflow-output":"ci/workflow-output.yml"}`))
	request.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.generateReleaseWorkflow(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)

	path := ax.Join(projectDir, "ci", "workflow-output.yml")
	content, err := io.Local.Read(path)
	require.NoError(t, err)
	assert.Contains(t, content, "workflow_call:")
	assert.Contains(t, content, "workflow_dispatch:")
}

func TestProvider_GenerateReleaseWorkflow_ConflictingWorkflowOutputAliases_Bad(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	p := NewProvider(projectDir, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release/workflow", bytes.NewBufferString(`{"outputPath":"ci/output-path.yml","workflowOutputPath":"ops/output-path.yml"}`))
	request.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.generateReleaseWorkflow(ctx)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	path := build.ReleaseWorkflowPath(projectDir)
	_, err := io.Local.Read(path)
	assert.Error(t, err)
}

func TestProvider_GenerateReleaseWorkflow_ConflictingOutputAliases_Bad(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	p := NewProvider(projectDir, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release/workflow", bytes.NewBufferString(`{"outputPath":"ci/output-path.yml","output_path":"ops/output-path.yml"}`))
	request.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.generateReleaseWorkflow(ctx)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	path := build.ReleaseWorkflowPath(projectDir)
	_, err := io.Local.Read(path)
	assert.Error(t, err)
}

func TestProvider_GenerateReleaseWorkflow_ConflictingOutputPathHyphenAliases_Bad(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	p := NewProvider(projectDir, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release/workflow", bytes.NewBufferString(`{"outputPath":"ci/output-path.yml","output-path":"ops/output-path.yml"}`))
	request.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.generateReleaseWorkflow(ctx)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	path := build.ReleaseWorkflowPath(projectDir)
	_, err := io.Local.Read(path)
	assert.Error(t, err)
}

func TestProvider_GenerateReleaseWorkflow_BareDirectoryPath_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	p := NewProvider(projectDir, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release/workflow", bytes.NewBufferString(`{"path":"ci"}`))
	request.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.generateReleaseWorkflow(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)

	path := ax.Join(projectDir, "ci", "release.yml")
	content, err := io.Local.Read(path)
	require.NoError(t, err)
	assert.Contains(t, content, "workflow_call:")
	assert.Contains(t, content, "workflow_dispatch:")
}

func TestProvider_GenerateReleaseWorkflow_CurrentDirectoryPrefixedPath_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	p := NewProvider(projectDir, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release/workflow", bytes.NewBufferString(`{"path":"./ci"}`))
	request.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.generateReleaseWorkflow(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)

	path := ax.Join(projectDir, "ci", "release.yml")
	content, err := io.Local.Read(path)
	require.NoError(t, err)
	assert.Contains(t, content, "workflow_call:")
	assert.Contains(t, content, "workflow_dispatch:")
}

func TestProvider_GenerateReleaseWorkflow_WorkflowsDirectory_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	p := NewProvider(projectDir, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release/workflow", bytes.NewBufferString(`{"path":".github/workflows"}`))
	request.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.generateReleaseWorkflow(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)

	path := ax.Join(projectDir, ".github", "workflows", "release.yml")
	content, err := io.Local.Read(path)
	require.NoError(t, err)
	assert.Contains(t, content, "workflow_call:")
	assert.Contains(t, content, "workflow_dispatch:")
}

func TestProvider_GenerateReleaseWorkflow_ExistingDirectoryPath_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	require.NoError(t, ax.MkdirAll(ax.Join(projectDir, "ci"), 0o755))
	p := NewProvider(projectDir, nil)
	p.medium = io.Local

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release/workflow", bytes.NewBufferString(`{"path":"ci"}`))
	request.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.generateReleaseWorkflow(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)

	path := ax.Join(projectDir, "ci", "release.yml")
	content, err := io.Local.Read(path)
	require.NoError(t, err)
	assert.Contains(t, content, "workflow_call:")
	assert.Contains(t, content, "workflow_dispatch:")
}

func TestProvider_GenerateReleaseWorkflow_ConflictingPathAndOutput_Bad(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	p := NewProvider(projectDir, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release/workflow", bytes.NewBufferString(`{"path":"ci/release.yml","output":"ops/release.yml"}`))
	request.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.generateReleaseWorkflow(ctx)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	path := build.ReleaseWorkflowPath(projectDir)
	_, err := io.Local.Read(path)
	assert.Error(t, err)
}

func TestProvider_GenerateReleaseWorkflow_InvalidJSON_Bad(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	p := NewProvider(projectDir, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release/workflow", bytes.NewBufferString(`{"path":`))
	request.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.generateReleaseWorkflow(ctx)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	path := build.ReleaseWorkflowPath(projectDir)
	_, err := io.Local.Read(path)
	assert.Error(t, err)
}

func TestProvider_GenerateReleaseWorkflow_EmptyBody_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	p := NewProvider(projectDir, nil)
	p.medium = io.Local

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release/workflow", nil)
	request.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.generateReleaseWorkflow(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)

	path := build.ReleaseWorkflowPath(projectDir)
	content, err := io.Local.Read(path)
	require.NoError(t, err)
	assert.Contains(t, content, "workflow_call:")
	assert.Contains(t, content, "workflow_dispatch:")
}

func TestProvider_DiscoverProject_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("GITHUB_SHA", "0123456789abcdef")
	t.Setenv("GITHUB_REF", "refs/heads/main")
	t.Setenv("GITHUB_REPOSITORY", "dappcore/core")

	projectDir := t.TempDir()
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "go.mod"), []byte("module example"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644))
	require.NoError(t, ax.MkdirAll(ax.Join(projectDir, "frontend"), 0o755))
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "frontend", "package.json"), []byte("{}"), 0o644))
	require.NoError(t, ax.MkdirAll(ax.Join(projectDir, ".core"), 0o755))
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, ".core", "build.yaml"), []byte(`
build:
  obfuscate: true
  nsis: true
  webview2: embed
  build_tags:
    - release
  ldflags:
    - -s
    - -w
`), 0o644))

	p := NewProvider(projectDir, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/discover", nil)

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.discoverProject(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	body := recorder.Body.String()
	assert.Contains(t, body, `"types":["wails","go","node"]`)
	assert.Contains(t, body, `"os":"`)
	assert.Contains(t, body, `"arch":"`)
	assert.Contains(t, body, `"primary":"wails"`)
	assert.Contains(t, body, `"primary_stack":"wails"`)
	assert.Contains(t, body, `"suggested_stack":"wails2"`)
	assert.Contains(t, body, `"has_frontend":true`)
	assert.Contains(t, body, `"has_root_package_json":false`)
	assert.Contains(t, body, `"has_frontend_package_json":true`)
	assert.Contains(t, body, `"has_root_go_mod":true`)
	assert.Contains(t, body, `"has_root_main_go":true`)
	assert.Contains(t, body, `"has_root_cmakelists":false`)
	assert.Contains(t, body, `"has_subtree_npm":false`)
	assert.Contains(t, body, `"linux_packages":`)
	assert.Contains(t, body, `"ref":"refs/heads/main"`)
	assert.Contains(t, body, `"branch":"main"`)
	assert.Contains(t, body, `"is_tag":false`)
	assert.Contains(t, body, `"sha":"0123456789abcdef"`)
	assert.Contains(t, body, `"short_sha":"0123456"`)
	assert.Contains(t, body, `"repo":"dappcore/core"`)
	assert.Contains(t, body, `"owner":"dappcore"`)
	assert.Contains(t, body, `"build_options":"`)
	assert.Contains(t, body, `"-obfuscated`)
	assert.Contains(t, body, `"options":{"ldflags":["-s","-w"],"nsis":true,"obfuscate":true`)
	assert.Contains(t, body, `"go.mod":true`)
	assert.Contains(t, body, `"main.go":true`)
	assert.Contains(t, body, `"frontend/package.json":true`)
}

func TestProvider_TriggerBuild_UsesFullBuildRuntimeConfig_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	require.NoError(t, io.Local.EnsureDir(ax.Join(projectDir, ".core")))
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, ".core", "build.yaml"), []byte(`
project:
  name: API Build
  main: ./cmd/api
  binary: api-build
build:
  type: go
  cgo: true
  obfuscate: true
  archive_format: xz
  flags:
    - -mod=readonly
  ldflags:
    - -s
  build_tags:
    - integration
  env:
    - FOO=bar
  cache:
    enabled: true
    paths:
      - cache/go-build
      - cache/go-mod
targets:
  - os: linux
    arch: amd64
`), 0o644))

	oldGetBuilder := providerGetBuilder
	oldDetermineVersion := providerDetermineVersion
	t.Cleanup(func() {
		providerGetBuilder = oldGetBuilder
		providerDetermineVersion = oldDetermineVersion
	})

	var capturedCfg *build.Config
	var capturedTargets []build.Target
	providerGetBuilder = func(projectType build.ProjectType) (build.Builder, error) {
		return &capturingBuilder{
			name: "go",
			buildFn: func(ctx context.Context, cfg *build.Config, targets []build.Target) ([]build.Artifact, error) {
				capturedCfg = cfg
				capturedTargets = append([]build.Target{}, targets...)

				artifactDir := ax.Join(cfg.OutputDir, "linux_amd64")
				require.NoError(t, cfg.FS.EnsureDir(artifactDir))
				artifactPath := ax.Join(artifactDir, cfg.Name)
				require.NoError(t, cfg.FS.WriteMode(artifactPath, "binary", 0o755))

				return []build.Artifact{{
					Path: artifactPath,
					OS:   "linux",
					Arch: "amd64",
				}}, nil
			},
		}, nil
	}
	providerDetermineVersion = func(ctx context.Context, dir string) (string, error) {
		return "v1.2.3", nil
	}

	p := NewProvider(projectDir, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/build", nil)

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.triggerBuild(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	require.NotNil(t, capturedCfg)
	assert.Equal(t, build.Project{
		Name:   "API Build",
		Main:   "./cmd/api",
		Binary: "api-build",
	}, capturedCfg.Project)
	assert.Equal(t, "api-build", capturedCfg.Name)
	assert.Equal(t, "v1.2.3", capturedCfg.Version)
	assert.Equal(t, []string{"-mod=readonly"}, capturedCfg.Flags)
	assert.Equal(t, []string{"-s"}, capturedCfg.LDFlags)
	assert.Equal(t, []string{"integration"}, capturedCfg.BuildTags)
	assert.Equal(t, []string{"FOO=bar"}, capturedCfg.Env)
	assert.True(t, capturedCfg.CGO)
	assert.True(t, capturedCfg.Obfuscate)
	assert.True(t, capturedCfg.Cache.Enabled)
	assert.Equal(t, []string{
		ax.Join(projectDir, "cache", "go-build"),
		ax.Join(projectDir, "cache", "go-mod"),
	}, capturedCfg.Cache.Paths)
	assert.True(t, capturedCfg.FS.Exists(ax.Join(projectDir, ".core", "cache")))
	assert.True(t, capturedCfg.FS.Exists(ax.Join(projectDir, "cache", "go-build")))
	assert.True(t, capturedCfg.FS.Exists(ax.Join(projectDir, "cache", "go-mod")))
	assert.Equal(t, []build.Target{{OS: "linux", Arch: "amd64"}}, capturedTargets)
	assert.Contains(t, recorder.Body.String(), `"archive_format":"xz"`)
	assert.Contains(t, recorder.Body.String(), `.tar.xz`)
	assert.True(t, io.Local.Exists(ax.Join(projectDir, "dist", "api-build_linux_amd64.tar.xz")))
	assert.True(t, io.Local.Exists(ax.Join(projectDir, "dist", "CHECKSUMS.txt")))

	checksums, err := io.Local.Read(ax.Join(projectDir, "dist", "CHECKSUMS.txt"))
	require.NoError(t, err)
	assert.Contains(t, checksums, "api-build_linux_amd64.tar.xz")
}

func TestProvider_ListArtifacts_RecursesIntoPlatformDirectories_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	distDir := ax.Join(projectDir, "dist")
	require.NoError(t, io.Local.EnsureDir(ax.Join(distDir, "linux_amd64")))
	require.NoError(t, io.Local.Write(ax.Join(distDir, "CHECKSUMS.txt"), "checksums"))
	require.NoError(t, io.Local.Write(ax.Join(distDir, "linux_amd64", "demo.tar.xz"), "archive"))

	p := NewProvider(projectDir, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/artifacts", nil)

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.listArtifacts(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	body := recorder.Body.String()
	assert.Contains(t, body, `"exists":true`)
	assert.Contains(t, body, `"name":"CHECKSUMS.txt"`)
	assert.Contains(t, body, `"name":"linux_amd64/demo.tar.xz"`)
	assert.Contains(t, body, ax.Join(distDir, "linux_amd64", "demo.tar.xz"))
}

type capturingBuilder struct {
	name    string
	buildFn func(ctx context.Context, cfg *build.Config, targets []build.Target) ([]build.Artifact, error)
}

func (b *capturingBuilder) Name() string {
	return b.name
}

func (b *capturingBuilder) Detect(fs io.Medium, dir string) (bool, error) {
	return true, nil
}

func (b *capturingBuilder) Build(ctx context.Context, cfg *build.Config, targets []build.Target) ([]build.Artifact, error) {
	return b.buildFn(ctx, cfg, targets)
}
