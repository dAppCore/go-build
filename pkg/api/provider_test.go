// SPDX-Licence-Identifier: EUPL-1.2

package api

import (
	"bytes"
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

func TestProvider_DiscoverProject_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "go.mod"), []byte("module example"), 0o644))
	require.NoError(t, ax.MkdirAll(ax.Join(projectDir, "frontend"), 0o755))
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "frontend", "package.json"), []byte("{}"), 0o644))

	p := NewProvider(projectDir, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/discover", nil)

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.discoverProject(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	body := recorder.Body.String()
	assert.Contains(t, body, `"types":["go","node"]`)
	assert.Contains(t, body, `"primary":"go"`)
	assert.Contains(t, body, `"primary_stack":"go"`)
	assert.Contains(t, body, `"has_frontend":true`)
	assert.Contains(t, body, `"has_subtree_npm":true`)
	assert.Contains(t, body, `"go.mod":true`)
	assert.Contains(t, body, `"frontend/package.json":true`)
}
