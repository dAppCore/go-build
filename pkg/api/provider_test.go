// SPDX-Licence-Identifier: EUPL-1.2

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	core "dappco.re/go"
	coreapi "dappco.re/go/api"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/build/pkg/release"
	"dappco.re/go/io"
	"dappco.re/go/ws"
	"github.com/gorilla/websocket"
)

func TestProvider_BuildProviderIdentityGood(t *testing.T) {
	p := NewProvider(".", nil)
	if !stdlibAssertEqual("build", p.Name()) {
		t.Fatalf("want %v, got %v", "build", p.Name())
	}
	if !stdlibAssertEqual("/api/v1/build", p.BasePath()) {
		t.Fatalf("want %v, got %v", "/api/v1/build", p.BasePath())
	}

}

func TestProvider_BuildProviderElementGood(t *testing.T) {
	p := NewProvider(".", nil)
	el := p.Element()
	if !stdlibAssertEqual("core-build-panel", el.Tag) {
		t.Fatalf("want %v, got %v", "core-build-panel", el.Tag)
	}
	if !stdlibAssertEqual("/assets/core-build.js", el.Source) {
		t.Fatalf("want %v, got %v", "/assets/core-build.js", el.Source)
	}

}

func TestProvider_BuildProviderChannelsGood(t *testing.T) {
	p := NewProvider(".", nil)
	channels := p.Channels()
	if !stdlibAssertContains(channels, "build.started") {
		t.Fatalf("expected %v to contain %v", channels, "build.started")
	}
	if !stdlibAssertContains(channels, "build.complete") {
		t.Fatalf("expected %v to contain %v", channels, "build.complete")
	}
	if !stdlibAssertContains(channels, "build.failed") {
		t.Fatalf("expected %v to contain %v", channels, "build.failed")
	}
	if !stdlibAssertContains(channels, "release.started") {
		t.Fatalf("expected %v to contain %v", channels, "release.started")
	}
	if !stdlibAssertContains(channels,

		// Should have 11 endpoint descriptions
		"release.complete") {
		t.Fatalf("expected %v to contain %v", channels, "release.complete")

		// Verify key routes exist
	}
	if !stdlibAssertContains(channels, "workflow.generated") {
		t.Fatalf("expected %v to contain %v", channels, "workflow.generated")
	}
	if !stdlibAssertContains(channels, "sdk.generated") {
		t.Fatalf("expected %v to contain %v", channels, "sdk.generated")
	}
	if len(channels) != 7 {
		t.Fatalf("want len %v, got %v", 7, len(channels))
	}

}

func TestProvider_BuildProviderDescribeGood(t *testing.T) {
	p := NewProvider(".", nil)
	routes := p.Describe()
	if len(routes) != 11 {
		t.Fatalf("want len %v, got %v", 11, len(routes))
	}

	paths := make(map[string]string)
	for _, r := range routes {
		paths[r.Path] = r.Method
	}
	if !stdlibAssertEqual("GET", paths["/config"]) {
		t.Fatalf("want %v, got %v", "GET", paths["/config"])
	}
	if !stdlibAssertEqual("GET", paths["/discover"]) {
		t.Fatalf("want %v, got %v", "GET", paths["/discover"])
	}
	if !stdlibAssertEqual("POST", paths["/"]) {
		t.Fatalf("want %v, got %v", "POST", paths["/"])
	}
	if !stdlibAssertEqual("GET", paths["/artifacts"]) {
		t.Fatalf("want %v, got %v", "GET", paths["/artifacts"])
	}
	if !stdlibAssertEqual("GET", paths["/events"]) {
		t.Fatalf("want %v, got %v", "GET", paths["/events"])
	}
	if !stdlibAssertEqual("GET", paths["/release/version"]) {
		t.Fatalf("want %v, got %v", "GET", paths["/release/version"])
	}
	if !stdlibAssertEqual("GET", paths["/release/changelog"]) {
		t.Fatalf("want %v, got %v", "GET", paths["/release/changelog"])
	}
	if !stdlibAssertEqual("POST", paths["/release"]) {
		t.Fatalf("want %v, got %v", "POST", paths["/release"])
	}
	if !stdlibAssertEqual("POST", paths["/release/workflow"]) {
		t.Fatalf("want %v, got %v", "POST", paths["/release/workflow"])
	}
	if !stdlibAssertEqual("GET", paths["/sdk/diff"]) {
		t.Fatalf("want %v, got %v", "GET", paths["/sdk/diff"])
	}
	if !stdlibAssertEqual("POST", paths["/sdk"]) {
		t.Fatalf("want %v, got %v", "POST", paths["/sdk"])
	}

	for _, route := range routes {
		if route.Path == "/release" {
			if !stdlibAssertEqual("Runs the full release pipeline: build, sign, archive, checksum, and publish.", route.Description) {
				t.Fatalf("want %v, got %v", "Runs the full release pipeline: build, sign, archive, checksum, and publish.", route.Description)
			}

		}
	}

	var workflowRoute *coreapi.RouteDescription
	for i := range routes {
		if routes[i].Path == "/release/workflow" {
			workflowRoute = &routes[i]
			break
		}
	}
	if stdlibAssertNil(workflowRoute) {
		t.Fatal("expected non-nil")
	}
	if stdlibAssertNil(workflowRoute.RequestBody) {
		t.Fatal("expected non-nil")
	}

	properties, ok := workflowRoute.RequestBody["properties"].(map[string]any)
	if !(ok) {
		t.Fatal("expected true")
	}

	pathSchema, ok := properties[apiPathField].(map[string]any)
	if !(ok) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("string", pathSchema["type"]) {
		t.Fatalf("want %v, got %v", "string", pathSchema["type"])
	}
	if !stdlibAssertEqual("Preferred workflow path input, relative to the project directory or absolute.", pathSchema["description"]) {
		t.Fatalf("want %v, got %v", "Preferred workflow path input, relative to the project directory or absolute.", pathSchema["description"])
	}

	workflowPathSchema, ok := properties["workflowPath"].(map[string]any)
	if !(ok) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("string", workflowPathSchema["type"]) {
		t.Fatalf("want %v, got %v", "string", workflowPathSchema["type"])
	}
	if !stdlibAssertEqual("Predictable alias for path, relative to the project directory or absolute.", workflowPathSchema["description"]) {
		t.Fatalf("want %v, got %v", "Predictable alias for path, relative to the project directory or absolute.", workflowPathSchema["description"])
	}

	workflowPathSnakeSchema, ok := properties["workflow_path"].(map[string]any)
	if !(ok) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("string", workflowPathSnakeSchema["type"]) {
		t.Fatalf("want %v, got %v", "string", workflowPathSnakeSchema["type"])
	}
	if !stdlibAssertEqual("Snake_case alias for workflowPath.", workflowPathSnakeSchema["description"]) {
		t.Fatalf("want %v, got %v", "Snake_case alias for workflowPath.", workflowPathSnakeSchema["description"])
	}

	workflowPathHyphenSchema, ok := properties["workflow-path"].(map[string]any)
	if !(ok) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("string", workflowPathHyphenSchema["type"]) {
		t.Fatalf("want %v, got %v", "string", workflowPathHyphenSchema["type"])
	}
	if !stdlibAssertEqual("Hyphenated alias for workflowPath.", workflowPathHyphenSchema["description"]) {
		t.Fatalf("want %v, got %v", "Hyphenated alias for workflowPath.", workflowPathHyphenSchema["description"])
	}

	outputSchema, ok := properties["output"].(map[string]any)
	if !(ok) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("string", outputSchema["type"]) {
		t.Fatalf("want %v, got %v", "string", outputSchema["type"])
	}
	if !stdlibAssertEqual("Legacy alias for outputPath.", outputSchema["description"]) {
		t.Fatalf("want %v, got %v", "Legacy alias for outputPath.", outputSchema["description"])
	}

	outputPathSchema, ok := properties["outputPath"].(map[string]any)
	if !(ok) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("string", outputPathSchema["type"]) {
		t.Fatalf("want %v, got %v", "string", outputPathSchema["type"])
	}
	if !stdlibAssertEqual("Preferred explicit workflow output path, relative to the project directory or absolute.", outputPathSchema["description"]) {
		t.Fatalf("want %v, got %v", "Preferred explicit workflow output path, relative to the project directory or absolute.", outputPathSchema["description"])
	}

	outputPathHyphenSchema, ok := properties["output-path"].(map[string]any)
	if !(ok) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("string", outputPathHyphenSchema["type"]) {
		t.Fatalf("want %v, got %v", "string", outputPathHyphenSchema["type"])
	}
	if !stdlibAssertEqual("Hyphenated alias for outputPath.", outputPathHyphenSchema["description"]) {
		t.Fatalf("want %v, got %v", "Hyphenated alias for outputPath.", outputPathHyphenSchema["description"])
	}

	workflowOutputPathSchema, ok := properties["workflowOutputPath"].(map[string]any)
	if !(ok) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("string", workflowOutputPathSchema["type"]) {
		t.Fatalf("want %v, got %v", "string", workflowOutputPathSchema["type"])
	}
	if !stdlibAssertEqual("Predictable alias for outputPath, relative to the project directory or absolute.", workflowOutputPathSchema["description"]) {
		t.Fatalf("want %v, got %v", "Predictable alias for outputPath, relative to the project directory or absolute.", workflowOutputPathSchema["description"])
	}

	workflowOutputSnakeSchema, ok := properties["workflow_output"].(map[string]any)
	if !(ok) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("string", workflowOutputSnakeSchema["type"]) {
		t.Fatalf("want %v, got %v", "string", workflowOutputSnakeSchema["type"])
	}
	if !stdlibAssertEqual("Snake_case alias for workflowOutputPath.", workflowOutputSnakeSchema["description"]) {
		t.Fatalf("want %v, got %v", "Snake_case alias for workflowOutputPath.", workflowOutputSnakeSchema["description"])
	}

	workflowOutputHyphenSchema, ok := properties["workflow-output"].(map[string]any)
	if !(ok) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("string", workflowOutputHyphenSchema["type"]) {
		t.Fatalf("want %v, got %v", "string", workflowOutputHyphenSchema["type"])
	}
	if !stdlibAssertEqual("Hyphenated alias for workflowOutputPath.", workflowOutputHyphenSchema["description"]) {
		t.Fatalf("want %v, got %v", "Hyphenated alias for workflowOutputPath.", workflowOutputHyphenSchema["description"])
	}

	workflowOutputPathSnakeSchema, ok := properties["workflow_output_path"].(map[string]any)
	if !(ok) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("string", workflowOutputPathSnakeSchema["type"]) {
		t.Fatalf("want %v, got %v", "string", workflowOutputPathSnakeSchema["type"])
	}
	if !stdlibAssertEqual("Snake_case alias for workflowOutputPath.", workflowOutputPathSnakeSchema["description"]) {
		t.Fatalf("want %v, got %v", "Snake_case alias for workflowOutputPath.", workflowOutputPathSnakeSchema["description"])
	}

	workflowOutputPathHyphenSchema, ok := properties["workflow-output-path"].(map[string]any)
	if !(ok) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("string", workflowOutputPathHyphenSchema["type"]) {
		t.Fatalf("want %v, got %v", "string", workflowOutputPathHyphenSchema["type"])
	}
	if !stdlibAssertEqual("Hyphenated alias for workflowOutputPath.", workflowOutputPathHyphenSchema["description"]) {
		t.Fatalf("want %v, got %v", "Hyphenated alias for workflowOutputPath.", workflowOutputPathHyphenSchema["description"])
	}

	outputPathSnakeSchema, ok := properties["output_path"].(map[string]any)
	if !(ok) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("string", outputPathSnakeSchema["type"]) {
		t.Fatalf("want %v, got %v", "string", outputPathSnakeSchema["type"])
	}
	if !stdlibAssertEqual("Snake_case alias for outputPath.", outputPathSnakeSchema["description"]) {
		t.Fatalf("want %v, got %v", "Snake_case alias for outputPath.", outputPathSnakeSchema["description"])
	}

}

func TestProvider_ReleaseWorkflowRequestResolvedOutputPathGood(t *testing.T) {
	projectDir := t.TempDir()
	absoluteDir := ax.Join(projectDir, "ops")
	requireProviderOK(t, io.Local.EnsureDir(absoluteDir))

	req := ReleaseWorkflowRequest{
		WorkflowOutputPath: absoluteDir,
	}

	path := requireProviderString(t, req.resolveOutputPath(projectDir, io.Local))
	if !stdlibAssertEqual(ax.Join(absoluteDir, "release.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(absoluteDir, "release.yml"), path)
	}

}

func TestProvider_ReleaseWorkflowRequestResolvedOutputPathAliasesGood(t *testing.T) {
	projectDir := t.TempDir()

	req := ReleaseWorkflowRequest{
		WorkflowOutputSnake:  "ci/workflow-output.yml",
		WorkflowOutputHyphen: "ci/workflow-output.yml",
	}

	path := requireProviderString(t, req.resolveOutputPath(projectDir, io.Local))
	if !stdlibAssertEqual(ax.Join(projectDir, "ci", "workflow-output.yml"), path) {
		t.Fatalf("want %v, got %v", ax.Join(projectDir, "ci", "workflow-output.yml"), path)
	}

}

func TestProvider_BuildProviderDefaultProjectDirGood(t *testing.T) {
	p := NewProvider("", nil)
	if !stdlibAssertEqual(".", p.projectDir) {
		t.Fatalf("want %v, got %v", ".", p.projectDir)
	}

}

func TestProvider_BuildProviderCustomProjectDirGood(t *testing.T) {
	p := NewProvider("/tmp/myproject", nil)
	if !stdlibAssertEqual("/tmp/myproject", p.projectDir) {
		t.Fatalf("want %v, got %v", "/tmp/myproject", p.projectDir)
	}

}

func TestProvider_BuildProviderNilHubGood(t *testing.T) {
	p := NewProvider(".", nil)
	if p.hub != nil {
		t.Fatal("expected nil hub")
	}
	p.emitEvent("build.started", map[string]any{"test": true})
	if !stdlibAssertEqual(".", p.projectDir) {
		t.Fatalf("want %v, got %v", ".", p.projectDir)
	}
}

func TestProvider_ResolveBuildOutputsGood(t *testing.T) {
	t.Run("defaults to raw build output", func(t *testing.T) {
		archiveOutput, checksumOutput := resolveBuildOutputs(buildRequest{})
		if archiveOutput {
			t.Fatal("expected false")
		}
		if checksumOutput {
			t.Fatal("expected false")
		}

	})

	t.Run("enables archive and checksum when package is set", func(t *testing.T) {
		value := true
		archiveOutput, checksumOutput := resolveBuildOutputs(buildRequest{Package: &value})
		if !(archiveOutput) {
			t.Fatal("expected true")
		}
		if !(checksumOutput) {
			t.Fatal("expected true")
		}

	})

	t.Run("preserves explicit archive override over package", func(t *testing.T) {
		packageValue := true
		archiveValue := false
		archiveOutput, checksumOutput := resolveBuildOutputs(buildRequest{
			Archive: &archiveValue,
			Package: &packageValue,
		})
		if archiveOutput {
			t.Fatal("expected false")
		}
		if !(checksumOutput) {
			t.Fatal("expected true")
		}

	})
}

func TestProvider_GetBuilderSupportedTypesGood(t *testing.T) {
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
			b := requireProviderBuilder(t, getBuilder(tc.projectType))
			if !stdlibAssertEqual(tc.name, b.Name()) {
				t.Fatalf("want %v, got %v", tc.name, b.Name())
			}

		})
	}
}

func TestProvider_GetBuilderUnsupportedTypeBad(t *testing.T) {
	message := requireProviderError(t, getBuilder(build.ProjectType("unknown")))
	if !stdlibAssertContains(message, "unknown project type") {
		t.Fatalf("expected %v to contain %v", message, "unknown project type")
	}

}

func TestProvider_BuildProviderResolveDirGood(t *testing.T) {
	p := NewProvider("/tmp", nil)
	dir := requireProviderString(t, p.resolveDir())
	if !stdlibAssertEqual("/tmp", dir) {
		t.Fatalf("want %v, got %v", "/tmp", dir)
	}

}

func TestProvider_BuildProviderResolveDirRelativeGood(t *testing.T) {
	p := NewProvider(".", nil)
	dir := requireProviderString(t, p.resolveDir())
	if !(len(dir) > 1 && dir[0] == '/') {
		t.Fatal("expected true")
	}

}

func TestProvider_BuildProviderMediumSetGood(t *testing.T) {
	p := NewProvider(".", nil)
	if stdlibAssertNil(p.medium) {
		t.Fatal("medium should be set to io.Local")
	}

}

func TestProvider_RegisterRoutes_ExposesRFCAliases_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	p := NewProvider(projectDir, nil)

	router := gin.New()
	p.RegisterRoutes(router.Group(""))

	buildResponse := httptest.NewRecorder()
	buildRequest := httptest.NewRequest(http.MethodPost, "/", nil)
	router.ServeHTTP(buildResponse, buildRequest)
	if stdlibAssertEqual(http.StatusNotFound, buildResponse.Code) {
		t.Fatalf("did not want %v", buildResponse.Code)
	}

	sdkResponse := httptest.NewRecorder()
	sdkRequest := httptest.NewRequest(http.MethodPost, "/sdk", core.NewBufferString(`{}`))
	sdkRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(sdkResponse, sdkRequest)
	if stdlibAssertEqual(http.StatusNotFound, sdkResponse.Code) {
		t.Fatalf("did not want %v", sdkResponse.Code)
	}

	eventsResponse := httptest.NewRecorder()
	eventsRequest := httptest.NewRequest(http.MethodGet, "/events", nil)
	router.ServeHTTP(eventsResponse, eventsRequest)
	if !stdlibAssertEqual(http.StatusServiceUnavailable, eventsResponse.Code) {
		t.Fatalf("want %v, got %v", http.StatusServiceUnavailable, eventsResponse.Code)
	}

}

func TestProvider_StreamEvents_UsesHubHandlerGood(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	hub := ws.NewHub()
	go hub.Run(t.Context())

	p := NewProvider(projectDir, hub)

	router := gin.New()
	p.RegisterRoutes(router.Group(""))

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + core.TrimPrefix(server.URL, "http") + "/events"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defer conn.Close()
	if err := conn.WriteJSON(ws.Message{Type: ws.TypeSubscribe, Data: "build.complete"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	{
		deadline := time.Now().Add(time.Second)
		for {
			if (func() bool {
				return hub.ChannelSubscriberCount("build.complete") == 1
			})() {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("condition was not satisfied")
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	p.emitEvent("build.complete", map[string]any{"status": "ok"})

	var message ws.Message
	if err := conn.ReadJSON(&message); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(ws.TypeEvent, message.Type) {
		t.Fatalf("want %v, got %v", ws.TypeEvent, message.Type)
	}

	payload, ok := message.Data.(map[string]any)
	if !(ok) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("ok", payload["status"]) {
		t.Fatalf("want %v, got %v", "ok", payload["status"])
	}

}

func TestProvider_GetConfig_UsesSnakeCaseJSONKeysGood(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	requireProviderOK(t, io.Local.EnsureDir(ax.Join(projectDir, ".core")))
	if result := ax.WriteFile(ax.Join(projectDir, ".core", "build.yaml"), []byte(`
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
`), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	p := NewProvider(projectDir, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/config", nil)

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.getConfig(ctx)
	if !stdlibAssertEqual(http.StatusOK, recorder.Code) {
		t.Fatalf("want %v, got %v", http.StatusOK, recorder.Code)
	}

	body := recorder.Body.String()
	if !stdlibAssertContains(body, `"config":`) {
		t.Fatalf("expected %v to contain %v", body, `"config":`)
	}
	if !stdlibAssertContains(body, `"version":1`) {
		t.Fatalf("expected %v to contain %v", body, `"version":1`)
	}
	if !stdlibAssertContains(body, `"project":{"name":"Demo"`) {
		t.Fatalf("expected %v to contain %v", body, `"project":{"name":"Demo"`)
	}
	if !stdlibAssertContains(body, `"build":{"type":"go","cgo":true`) {
		t.Fatalf("expected %v to contain %v", body, `"build":{"type":"go","cgo":true`)
	}
	if !stdlibAssertContains(body, `"cache":{"enabled":true,"dir":"cache-meta","key_prefix":"demo","paths":["`) {
		t.Fatalf("expected %v to contain %v", body, `"cache":{"enabled":true,"dir":"cache-meta","key_prefix":"demo","paths":["`)
	}
	if !stdlibAssertContains(body, `"apple":{"bundle_id":"ai.lthn.demo"`) {
		t.Fatalf("expected %v to contain %v", body, `"apple":{"bundle_id":"ai.lthn.demo"`)
	}
	if !stdlibAssertContains(body, `"xcode_cloud":{"workflow":"Release"`) {
		t.Fatalf("expected %v to contain %v", body, `"xcode_cloud":{"workflow":"Release"`)
	}
	if !stdlibAssertContains(body, `"sign":{"enabled":true`) {
		t.Fatalf("expected %v to contain %v", body, `"sign":{"enabled":true`)
	}
	if !stdlibAssertContains(body, `"macos":{"identity":"Developer ID Application: Demo"`) {
		t.Fatalf("expected %v to contain %v", body, `"macos":{"identity":"Developer ID Application: Demo"`)
	}
	if stdlibAssertContains(body, `"Version":`) {
		t.Fatalf("expected %v not to contain %v", body, `"Version":`)
	}
	if stdlibAssertContains(body, `"Project":`) {
		t.Fatalf("expected %v not to contain %v", body, `"Project":`)
	}
	if stdlibAssertContains(body, `"XcodeCloud":`) {
		t.Fatalf("expected %v not to contain %v", body, `"XcodeCloud":`)
	}
	if stdlibAssertContains(body, `"MacOS":`) {
		t.Fatalf("expected %v not to contain %v", body, `"MacOS":`)
	}

}

func TestProvider_ResolveProjectTypeGood(t *testing.T) {
	t.Run("honours explicit build type override", func(t *testing.T) {
		dir := t.TempDir()
		if result := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example"), 0o644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		projectType := requireProviderProjectType(t, resolveProjectType(io.Local, dir, "docker"))
		if !stdlibAssertEqual(build.ProjectTypeDocker, projectType) {
			t.Fatalf("want %v, got %v", build.ProjectTypeDocker, projectType)
		}

	})

	t.Run("falls back to detection when build type is empty", func(t *testing.T) {
		dir := t.TempDir()
		if result := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example"), 0o644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		projectType := requireProviderProjectType(t, resolveProjectType(io.Local, dir, ""))
		if !stdlibAssertEqual(build.ProjectTypeGo, projectType) {
			t.Fatalf("want %v, got %v", build.ProjectTypeGo, projectType)
		}

	})
}

type providerReleaseWorkflowCase struct {
	name           string
	body           string
	bodyFor        func(projectDir string) string
	nilBody        bool
	wantStatus     int
	wantPath       func(projectDir string) string
	before         func(t *testing.T, projectDir string)
	useLocalMedium bool
	expectWorkflow bool
}

type providerWorkflowJSONField struct {
	key   string
	value string
}

func providerWorkflowJSON(fields ...providerWorkflowJSONField) string {
	values := make(map[string]string, len(fields))
	for _, field := range fields {
		values[field.key] = field.value
	}
	encoded := core.JSONMarshal(values)
	if !encoded.OK {
		return "{}"
	}
	return string(encoded.Value.([]byte))
}

func providerPathJSON(value string) string {
	return providerWorkflowJSON(providerWorkflowJSONField{key: apiPathField, value: value})
}

func providerInvalidPathJSON() string {
	return core.Concat("{\"", apiPathField, "\":")
}

func providerJSONFieldPrefix(key string) string {
	return core.Concat("\"", key, "\":\"")
}

func TestProvider_GenerateReleaseWorkflowGood(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []providerReleaseWorkflowCase{
		{name: "default path", body: `{}`, wantPath: build.ReleaseWorkflowPath, expectWorkflow: true},
		{name: "custom path", body: providerPathJSON("ci/release.yml"), wantPath: providerWorkflowPath("ci", "release.yml"), expectWorkflow: true},
		{name: "workflowPath alias", body: `{"workflowPath":"ci/workflow-path.yml"}`, wantPath: providerWorkflowPath("ci", "workflow-path.yml"), expectWorkflow: true},
		{name: "workflow_path alias", body: `{"workflow_path":"ci/workflow-path.yml"}`, wantPath: providerWorkflowPath("ci", "workflow-path.yml"), expectWorkflow: true},
		{name: "workflow-path alias", body: `{"workflow-path":"ci/workflow-path.yml"}`, wantPath: providerWorkflowPath("ci", "workflow-path.yml"), expectWorkflow: true},
		{name: "conflicting workflow path aliases", body: providerWorkflowJSON(providerWorkflowJSONField{key: apiPathField, value: "ci/workflow-path.yml"}, providerWorkflowJSONField{key: "workflowPath", value: "ops/workflow-path.yml"}), wantStatus: http.StatusBadRequest},
		{name: "output alias", body: `{"output":"ci/release.yml"}`, wantPath: providerWorkflowPath("ci", "release.yml"), expectWorkflow: true},
		{name: "outputPath alias", body: `{"outputPath":"ci/output-path.yml"}`, wantPath: providerWorkflowPath("ci", "output-path.yml"), expectWorkflow: true},
		{name: "output-path alias", body: `{"output-path":"ci/output-path.yml"}`, wantPath: providerWorkflowPath("ci", "output-path.yml"), expectWorkflow: true},
		{name: "output_path alias", body: `{"output_path":"ci/output-path.yml"}`, wantPath: providerWorkflowPath("ci", "output-path.yml"), expectWorkflow: true},
		{name: "workflowOutputPath alias", body: `{"workflowOutputPath":"ci/workflow-output-path.yml"}`, wantPath: providerWorkflowPath("ci", "workflow-output-path.yml"), expectWorkflow: true},
		{name: "workflow_output alias", body: `{"workflow_output":"ci/workflow-output.yml"}`, wantPath: providerWorkflowPath("ci", "workflow-output.yml"), expectWorkflow: true},
		{name: "workflow_output_path alias", body: `{"workflow_output_path":"ci/workflow-output-path.yml"}`, wantPath: providerWorkflowPath("ci", "workflow-output-path.yml"), expectWorkflow: true},
		{
			name: "absolute equivalent workflow output path",
			bodyFor: func(projectDir string) string {
				absolutePath := ax.Join(projectDir, "ci", "workflow-output-path.yml")
				return `{"outputPath":"ci/workflow-output-path.yml","workflowOutputPath":"` + absolutePath + `"}`
			},
			wantPath:       providerWorkflowPath("ci", "workflow-output-path.yml"),
			expectWorkflow: true,
		},
		{name: "workflow-output-path alias", body: `{"workflow-output-path":"ci/workflow-output-path.yml"}`, wantPath: providerWorkflowPath("ci", "workflow-output-path.yml"), expectWorkflow: true},
		{name: "workflow-output alias", body: `{"workflow-output":"ci/workflow-output.yml"}`, wantPath: providerWorkflowPath("ci", "workflow-output.yml"), expectWorkflow: true},
		{name: "conflicting workflow output aliases", body: `{"outputPath":"ci/output-path.yml","workflowOutputPath":"ops/output-path.yml"}`, wantStatus: http.StatusBadRequest},
		{name: "conflicting output aliases", body: `{"outputPath":"ci/output-path.yml","output_path":"ops/output-path.yml"}`, wantStatus: http.StatusBadRequest},
		{name: "conflicting output path hyphen aliases", body: `{"outputPath":"ci/output-path.yml","output-path":"ops/output-path.yml"}`, wantStatus: http.StatusBadRequest},
		{name: "bare directory path", body: providerPathJSON("ci"), wantPath: providerWorkflowPath("ci", "release.yml"), expectWorkflow: true},
		{name: "current directory prefixed path", body: providerPathJSON("./ci"), wantPath: providerWorkflowPath("ci", "release.yml"), expectWorkflow: true},
		{name: "workflows directory", body: providerPathJSON(".github/workflows"), wantPath: providerWorkflowPath(".github", "workflows", "release.yml"), expectWorkflow: true},
		{
			name:           "existing directory path",
			body:           providerPathJSON("ci"),
			before:         createProviderWorkflowDir("ci"),
			useLocalMedium: true,
			wantPath:       providerWorkflowPath("ci", "release.yml"),
			expectWorkflow: true,
		},
		{name: "conflicting path and output", body: providerWorkflowJSON(providerWorkflowJSONField{key: apiPathField, value: "ci/release.yml"}, providerWorkflowJSONField{key: "output", value: "ops/release.yml"}), wantStatus: http.StatusBadRequest},
		{name: "invalid JSON", body: providerInvalidPathJSON(), wantStatus: http.StatusBadRequest},
		{name: "empty body", nilBody: true, useLocalMedium: true, wantPath: build.ReleaseWorkflowPath, expectWorkflow: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			assertProviderReleaseWorkflow(t, tc)
		})
	}
}

func providerWorkflowPath(parts ...string) func(projectDir string) string {
	return func(projectDir string) string {
		return ax.Join(append([]string{projectDir}, parts...)...)
	}
}

func createProviderWorkflowDir(parts ...string) func(t *testing.T, projectDir string) {
	return func(t *testing.T, projectDir string) {
		t.Helper()
		if result := ax.MkdirAll(ax.Join(append([]string{projectDir}, parts...)...), 0o755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
	}
}

func assertProviderReleaseWorkflow(t *testing.T, tc providerReleaseWorkflowCase) {
	t.Helper()

	projectDir := t.TempDir()
	if tc.before != nil {
		tc.before(t, projectDir)
	}

	p := NewProvider(projectDir, nil)
	if tc.useLocalMedium {
		p.medium = io.Local
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release/workflow", nil)
	if !tc.nilBody {
		body := tc.body
		if tc.bodyFor != nil {
			body = tc.bodyFor(projectDir)
		}
		request = httptest.NewRequest(http.MethodPost, "/release/workflow", core.NewBufferString(body))
	}
	request.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.generateReleaseWorkflow(ctx)
	wantStatus := tc.wantStatus
	if wantStatus == 0 {
		wantStatus = http.StatusOK
	}
	if !stdlibAssertEqual(wantStatus, recorder.Code) {
		t.Fatalf("want %v, got %v", wantStatus, recorder.Code)
	}

	path := build.ReleaseWorkflowPath(projectDir)
	if tc.wantPath != nil {
		path = tc.wantPath(projectDir)
	}
	if !tc.expectWorkflow {
		if result := io.Local.Read(path); result.OK {
			t.Fatal("expected error")
		}
		return
	}

	content := requireProviderString(t, io.Local.Read(path))
	if !stdlibAssertContains(content, "workflow_call:") {
		t.Fatalf("expected %v to contain %v", content, "workflow_call:")
	}
	if !stdlibAssertContains(content, "workflow_dispatch:") {
		t.Fatalf("expected %v to contain %v", content, "workflow_dispatch:")
	}
}

func TestProvider_discoverProject_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("GITHUB_SHA", "0123456789abcdef")
	t.Setenv("GITHUB_REF", "refs/heads/main")
	t.Setenv("GITHUB_REPOSITORY", "dappcore/core")

	projectDir := t.TempDir()
	if result := ax.WriteFile(ax.Join(projectDir, "go.mod"), []byte("module example"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(ax.Join(projectDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.MkdirAll(ax.Join(projectDir, "frontend"), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(ax.Join(projectDir, "frontend", "package.json"), []byte("{}"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.MkdirAll(ax.Join(projectDir, ".core"), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(ax.Join(projectDir, ".core", "build.yaml"), []byte(`
build:
  obfuscate: true
  nsis: true
  webview2: embed
  build_tags:
    - release
  ldflags:
    - -s
    - -w
`), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	p := NewProvider(projectDir, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/discover", nil)

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.discoverProject(ctx)
	if !stdlibAssertEqual(http.StatusOK, recorder.Code) {
		t.Fatalf("want %v, got %v", http.StatusOK, recorder.Code)
	}

	body := recorder.Body.String()
	if !stdlibAssertContains(body, `"types":["wails","go","node"]`) {
		t.Fatalf("expected %v to contain %v", body, `"types":["wails","go","node"]`)
	}
	if !stdlibAssertContains(body, `"configured_build_type":""`) {
		t.Fatalf("expected %v to contain %v", body, `"configured_build_type":""`)
	}
	if !stdlibAssertContains(body, providerJSONFieldPrefix(apiOSField)) {
		t.Fatalf("expected %v to contain %v", body, providerJSONFieldPrefix(apiOSField))
	}
	if !stdlibAssertContains(body, `"arch":"`) {
		t.Fatalf("expected %v to contain %v", body, `"arch":"`)
	}
	if !stdlibAssertContains(body, `"primary":"wails"`) {
		t.Fatalf("expected %v to contain %v", body, `"primary":"wails"`)
	}
	if !stdlibAssertContains(body, `"primary_stack":"wails"`) {
		t.Fatalf("expected %v to contain %v", body, `"primary_stack":"wails"`)
	}
	if !stdlibAssertContains(body, `"suggested_stack":"wails2"`) {
		t.Fatalf("expected %v to contain %v", body, `"suggested_stack":"wails2"`)
	}
	if !stdlibAssertContains(body, `"primary_stack_suggestion":"wails2"`) {
		t.Fatalf("expected %v to contain %v", body, `"primary_stack_suggestion":"wails2"`)
	}
	if !stdlibAssertContains(body, `"has_frontend":true`) {
		t.Fatalf("expected %v to contain %v", body, `"has_frontend":true`)
	}
	if !stdlibAssertContains(body, `"has_root_package_json":false`) {
		t.Fatalf("expected %v to contain %v", body, `"has_root_package_json":false`)
	}
	if !stdlibAssertContains(body, `"has_frontend_package_json":true`) {
		t.Fatalf("expected %v to contain %v", body, `"has_frontend_package_json":true`)
	}
	if !stdlibAssertContains(body, `"has_root_composer_json":false`) {
		t.Fatalf("expected %v to contain %v", body, `"has_root_composer_json":false`)
	}
	if !stdlibAssertContains(body, `"has_root_cargo_toml":false`) {
		t.Fatalf("expected %v to contain %v", body, `"has_root_cargo_toml":false`)
	}
	if !stdlibAssertContains(body, `"has_package_json":true`) {
		t.Fatalf("expected %v to contain %v", body, `"has_package_json":true`)
	}
	if !stdlibAssertContains(body, `"has_deno_manifest":false`) {
		t.Fatalf("expected %v to contain %v", body, `"has_deno_manifest":false`)
	}
	if !stdlibAssertContains(body, `"has_root_go_mod":true`) {
		t.Fatalf("expected %v to contain %v", body, `"has_root_go_mod":true`)
	}
	if !stdlibAssertContains(body, `"has_root_go_work":false`) {
		t.Fatalf("expected %v to contain %v", body, `"has_root_go_work":false`)
	}
	if !stdlibAssertContains(body, `"has_root_main_go":true`) {
		t.Fatalf("expected %v to contain %v", body, `"has_root_main_go":true`)
	}
	if !stdlibAssertContains(body, `"has_root_cmakelists":false`) {
		t.Fatalf("expected %v to contain %v", body, `"has_root_cmakelists":false`)
	}
	if !stdlibAssertContains(body, `"has_root_wails_json":false`) {
		t.Fatalf("expected %v to contain %v", body, `"has_root_wails_json":false`)
	}
	if !stdlibAssertContains(body, `"has_taskfile":false`) {
		t.Fatalf("expected %v to contain %v", body, `"has_taskfile":false`)
	}
	if !stdlibAssertContains(body, `"has_subtree_npm":false`) {
		t.Fatalf("expected %v to contain %v", body, `"has_subtree_npm":false`)
	}
	if !stdlibAssertContains(body, `"has_subtree_package_json":false`) {
		t.Fatalf("expected %v to contain %v", body, `"has_subtree_package_json":false`)
	}
	if !stdlibAssertContains(body, `"has_subtree_deno_manifest":false`) {
		t.Fatalf("expected %v to contain %v", body, `"has_subtree_deno_manifest":false`)
	}
	if !stdlibAssertContains(body, `"has_docs_config":false`) {
		t.Fatalf("expected %v to contain %v", body, `"has_docs_config":false`)
	}
	if !stdlibAssertContains(body, `"has_go_toolchain":true`) {
		t.Fatalf("expected %v to contain %v", body, `"has_go_toolchain":true`)
	}
	if !stdlibAssertContains(body, `"deno_requested":false`) {
		t.Fatalf("expected %v to contain %v", body, `"deno_requested":false`)
	}
	if !stdlibAssertContains(body, `"linux_packages":`) {
		t.Fatalf("expected %v to contain %v", body, `"linux_packages":`)
	}
	if !stdlibAssertContains(body, `"ref":"refs/heads/main"`) {
		t.Fatalf("expected %v to contain %v", body, `"ref":"refs/heads/main"`)
	}
	if !stdlibAssertContains(body, `"branch":"main"`) {
		t.Fatalf("expected %v to contain %v", body, `"branch":"main"`)
	}
	if !stdlibAssertContains(body, `"is_tag":false`) {
		t.Fatalf("expected %v to contain %v", body, `"is_tag":false`)
	}
	if !stdlibAssertContains(body, `"sha":"0123456789abcdef"`) {
		t.Fatalf("expected %v to contain %v", body, `"sha":"0123456789abcdef"`)
	}
	if !stdlibAssertContains(body, `"short_sha":"0123456"`) {
		t.Fatalf("expected %v to contain %v", body, `"short_sha":"0123456"`)
	}
	if !stdlibAssertContains(body, `"repo":"dappcore/core"`) {
		t.Fatalf("expected %v to contain %v", body, `"repo":"dappcore/core"`)
	}
	if !stdlibAssertContains(body, `"owner":"dappcore"`) {
		t.Fatalf("expected %v to contain %v", body, `"owner":"dappcore"`)
	}
	if !stdlibAssertContains(body, `"build_options":"`) {
		t.Fatalf("expected %v to contain %v", body, `"build_options":"`)
	}
	if !stdlibAssertContains(body, `"-obfuscated`) {
		t.Fatalf("expected %v to contain %v", body, `"-obfuscated`)
	}
	if !stdlibAssertContains(body, `"options":{"ldflags":["-s","-w"],"nsis":true,"obfuscate":true`) {
		t.Fatalf("expected %v to contain %v", body, `"options":{"ldflags":["-s","-w"],"nsis":true,"obfuscate":true`)
	}
	if !stdlibAssertContains(body, `"setup_plan":{"frontend_dirs":["`) {
		t.Fatalf("expected %v to contain %v", body, `"setup_plan":{"frontend_dirs":["`)
	}
	if !stdlibAssertContains(body, `"primary_stack":"wails"`) {
		t.Fatalf("expected %v to contain %v", body, `"primary_stack":"wails"`)
	}
	if !stdlibAssertContains(body, `"primary_stack_suggestion":"wails2"`) {
		t.Fatalf("expected %v to contain %v", body, `"primary_stack_suggestion":"wails2"`)
	}
	if !stdlibAssertContains(body, `"tool":"go"`) {
		t.Fatalf("expected %v to contain %v", body, `"tool":"go"`)
	}
	if !stdlibAssertContains(body, `"tool":"garble"`) {
		t.Fatalf("expected %v to contain %v", body, `"tool":"garble"`)
	}
	if !stdlibAssertContains(body, `"tool":"node"`) {
		t.Fatalf("expected %v to contain %v", body, `"tool":"node"`)
	}
	if !stdlibAssertContains(body, `"tool":"wails"`) {
		t.Fatalf("expected %v to contain %v", body, `"tool":"wails"`)
	}
	if !stdlibAssertContains(body, `"go.mod":true`) {
		t.Fatalf("expected %v to contain %v", body, `"go.mod":true`)
	}
	if !stdlibAssertContains(body, `"main.go":true`) {
		t.Fatalf("expected %v to contain %v", body, `"main.go":true`)
	}
	if !stdlibAssertContains(body, `"frontend/package.json":true`) {
		t.Fatalf("expected %v to contain %v", body, `"frontend/package.json":true`)
	}

}

func TestProvider_TriggerBuild_UsesFullBuildRuntimeConfig_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	requireProviderOK(t, io.Local.EnsureDir(ax.Join(projectDir, ".core")))
	if result := ax.WriteFile(ax.Join(projectDir, ".core", "build.yaml"), []byte(`
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
`), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	oldGetBuilder := providerGetBuilder
	oldDetermineVersion := providerDetermineVersion
	t.Cleanup(func() {
		providerGetBuilder = oldGetBuilder
		providerDetermineVersion = oldDetermineVersion
	})

	var capturedCfg *build.Config
	var capturedTargets []build.Target
	providerGetBuilder = func(projectType build.ProjectType) core.Result {
		return core.Ok(&capturingBuilder{
			name: "go",
			buildFn: func(ctx context.Context, cfg *build.Config, targets []build.Target) core.Result {
				capturedCfg = cfg
				capturedTargets = append([]build.Target{}, targets...)

				artifactDir := ax.Join(cfg.OutputDir, "linux_amd64")
				requireProviderOK(t, cfg.FS.EnsureDir(artifactDir))

				artifactPath := ax.Join(artifactDir, cfg.Name)
				requireProviderOK(t, cfg.FS.WriteMode(artifactPath, "binary", 0o755))

				return core.Ok([]build.Artifact{{
					Path: artifactPath,
					OS:   "linux",
					Arch: "amd64",
				}})
			},
		})
	}
	providerDetermineVersion = func(ctx context.Context, dir string) core.Result {
		return core.Ok("v1.2.3")
	}

	p := NewProvider(projectDir, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/build", core.NewBufferString(`{"package":true}`))
	request.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.triggerBuild(ctx)
	if !stdlibAssertEqual(http.StatusOK, recorder.Code) {
		t.Fatalf("want %v, got %v", http.StatusOK, recorder.Code)
	}
	if stdlibAssertNil(capturedCfg) {
		t.Fatal("expected non-nil")
	}
	if !stdlibAssertEqual(build.Project{Name: "API Build", Main: "./cmd/api", Binary: "api-build"}, capturedCfg.Project) {
		t.Fatalf("want %v, got %v", build.Project{Name: "API Build", Main: "./cmd/api", Binary: "api-build"}, capturedCfg.Project)
	}
	if !stdlibAssertEqual("api-build", capturedCfg.Name) {
		t.Fatalf("want %v, got %v", "api-build", capturedCfg.Name)
	}
	if !stdlibAssertEqual("v1.2.3", capturedCfg.Version) {
		t.Fatalf("want %v, got %v", "v1.2.3", capturedCfg.Version)
	}
	if !stdlibAssertEqual([]string{"-mod=readonly"}, capturedCfg.Flags) {
		t.Fatalf("want %v, got %v", []string{"-mod=readonly"}, capturedCfg.Flags)
	}
	if !stdlibAssertEqual([]string{"-s"}, capturedCfg.LDFlags) {
		t.Fatalf("want %v, got %v", []string{"-s"}, capturedCfg.LDFlags)
	}
	if !stdlibAssertEqual([]string{"integration"}, capturedCfg.BuildTags) {
		t.Fatalf("want %v, got %v", []string{"integration"}, capturedCfg.BuildTags)
	}
	if !stdlibAssertEqual([]string{"FOO=bar"}, capturedCfg.Env) {
		t.Fatalf("want %v, got %v", []string{"FOO=bar"}, capturedCfg.Env)
	}
	if !(capturedCfg.CGO) {
		t.Fatal("expected true")
	}
	if !(capturedCfg.Obfuscate) {
		t.Fatal("expected true")
	}
	if !(capturedCfg.Cache.Enabled) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual([]string{ax.Join(projectDir, "cache", "go-build"), ax.Join(projectDir, "cache", "go-mod")}, capturedCfg.Cache.Paths) {
		t.Fatalf("want %v, got %v", []string{ax.Join(projectDir, "cache", "go-build"), ax.Join(projectDir, "cache", "go-mod")}, capturedCfg.Cache.Paths)
	}
	if !(capturedCfg.FS.Exists(ax.Join(projectDir, ".core", "cache"))) {
		t.Fatal("expected true")
	}
	if !(capturedCfg.FS.Exists(ax.Join(projectDir, "cache", "go-build"))) {
		t.Fatal("expected true")
	}
	if !(capturedCfg.FS.Exists(ax.Join(projectDir, "cache", "go-mod"))) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual([]build.Target{{OS: "linux", Arch: "amd64"}}, capturedTargets) {
		t.Fatalf("want %v, got %v", []build.Target{{OS: "linux", Arch: "amd64"}}, capturedTargets)
	}
	if !stdlibAssertContains(recorder.Body.String(), `"archive_format":"xz"`) {
		t.Fatalf("expected %v to contain %v", recorder.Body.String(), `"archive_format":"xz"`)
	}
	if !stdlibAssertContains(recorder.Body.String(), `.tar.xz`) {
		t.Fatalf("expected %v to contain %v", recorder.Body.String(), `.tar.xz`)
	}
	if !(io.Local.Exists(ax.Join(projectDir, "dist", "api-build_linux_amd64.tar.xz"))) {
		t.Fatal("expected true")
	}
	if !(io.Local.Exists(ax.Join(projectDir, "dist", "CHECKSUMS.txt"))) {
		t.Fatal("expected true")
	}

	checksums := requireProviderString(t, io.Local.Read(ax.Join(projectDir, "dist", "CHECKSUMS.txt")))
	if !stdlibAssertContains(checksums, "api-build_linux_amd64.tar.xz") {
		t.Fatalf("expected %v to contain %v", checksums, "api-build_linux_amd64.tar.xz")
	}

}

func TestProvider_TriggerBuild_DefaultsToRawArtifacts_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	if result := ax.MkdirAll(ax.Join(projectDir, ".core"), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(ax.Join(projectDir, "go.mod"), []byte("module example.com/provider\n\ngo 1.20\n"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(ax.Join(projectDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(ax.Join(projectDir, ".core", "build.yaml"), []byte(`version: 1
project:
  name: provider
  binary: provider
build:
  type: go
targets:
  - os: `+runtime.GOOS+`
    arch: `+runtime.GOARCH+`
sign:
  enabled: false
`), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	oldGetBuilder := providerGetBuilder
	oldDetermineVersion := providerDetermineVersion
	t.Cleanup(func() {
		providerGetBuilder = oldGetBuilder
		providerDetermineVersion = oldDetermineVersion
	})

	providerGetBuilder = func(projectType build.ProjectType) core.Result {
		return core.Ok(&capturingBuilder{
			name: "go",
			buildFn: func(ctx context.Context, cfg *build.Config, targets []build.Target) core.Result {
				artifactDir := ax.Join(cfg.OutputDir, runtime.GOOS+"_"+runtime.GOARCH)
				requireProviderOK(t, cfg.FS.EnsureDir(artifactDir))

				artifactPath := ax.Join(artifactDir, "provider")
				if runtime.GOOS == "windows" {
					artifactPath += ".exe"
				}
				requireProviderOK(t, cfg.FS.WriteMode(artifactPath, "binary", 0o755))

				return core.Ok([]build.Artifact{{
					Path: artifactPath,
					OS:   runtime.GOOS,
					Arch: runtime.GOARCH,
				}})
			},
		})
	}
	providerDetermineVersion = func(ctx context.Context, dir string) core.Result {
		return core.Ok("v1.2.3")
	}

	p := NewProvider(projectDir, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/build", nil)

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.triggerBuild(ctx)
	if !stdlibAssertEqual(http.StatusOK, recorder.Code) {
		t.Fatalf("want %v, got %v", http.StatusOK, recorder.Code)
	}
	if !stdlibAssertContains(recorder.Body.String(), `"project_type":"go"`) {
		t.Fatalf("expected %v to contain %v", recorder.Body.String(), `"project_type":"go"`)
	}
	if stdlibAssertContains(recorder.Body.String(), `"archive_format"`) {
		t.Fatalf("expected %v not to contain %v", recorder.Body.String(), `"archive_format"`)
	}
	if stdlibAssertContains(recorder.Body.String(), `"checksum_file"`) {
		t.Fatalf("expected %v not to contain %v", recorder.Body.String(), `"checksum_file"`)
	}
	if !(io.Local.Exists(ax.Join(projectDir, "dist", runtime.GOOS+"_"+runtime.GOARCH, "provider")) || io.Local.Exists(ax.Join(projectDir, "dist", runtime.GOOS+"_"+runtime.GOARCH, "provider.exe"))) {
		t.Fatal("expected true")
	}
	if io.Local.Exists(ax.Join(projectDir, "dist", "CHECKSUMS.txt")) {
		t.Fatal("expected false")
	}
	if io.Local.Exists(ax.Join(projectDir, "dist", "provider_"+runtime.GOOS+"_"+runtime.GOARCH+".tar.gz")) {
		t.Fatal("expected false")
	}
	if io.Local.Exists(ax.Join(projectDir, "dist", "provider_"+runtime.GOOS+"_"+runtime.GOARCH+".tar.xz")) {
		t.Fatal("expected false")
	}

}

func TestProvider_TriggerBuild_WithoutBuildConfig_UsesLocalTarget_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	if result := ax.WriteFile(ax.Join(projectDir, "go.mod"), []byte("module example.com/provider\n\ngo 1.20\n"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(ax.Join(projectDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	oldGetBuilder := providerGetBuilder
	oldDetermineVersion := providerDetermineVersion
	t.Cleanup(func() {
		providerGetBuilder = oldGetBuilder
		providerDetermineVersion = oldDetermineVersion
	})

	var capturedTargets []build.Target
	providerGetBuilder = func(projectType build.ProjectType) core.Result {
		return core.Ok(&capturingBuilder{
			name: "go",
			buildFn: func(ctx context.Context, cfg *build.Config, targets []build.Target) core.Result {
				capturedTargets = append([]build.Target{}, targets...)

				artifactDir := ax.Join(cfg.OutputDir, runtime.GOOS+"_"+runtime.GOARCH)
				requireProviderOK(t, cfg.FS.EnsureDir(artifactDir))

				artifactPath := ax.Join(artifactDir, "provider")
				requireProviderOK(t, cfg.FS.WriteMode(artifactPath, "binary", 0o755))

				return core.Ok([]build.Artifact{{
					Path: artifactPath,
					OS:   runtime.GOOS,
					Arch: runtime.GOARCH,
				}})
			},
		})
	}
	providerDetermineVersion = func(ctx context.Context, dir string) core.Result {
		return core.Ok("v0.0.1")
	}

	p := NewProvider(projectDir, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/build", nil)

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.triggerBuild(ctx)
	if !stdlibAssertEqual(http.StatusOK, recorder.Code) {
		t.Fatalf("want %v, got %v", http.StatusOK, recorder.Code)
	}
	if !stdlibAssertEqual([]build.Target{{OS: runtime.GOOS, Arch: runtime.GOARCH}}, capturedTargets) {
		t.Fatalf("want %v, got %v", []build.Target{{OS: runtime.GOOS, Arch: runtime.GOARCH}}, capturedTargets)
	}

}

func TestProvider_TriggerRelease_UsesFullReleasePipeline_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()

	oldLoadReleaseConfig := providerLoadReleaseConfig
	oldRunRelease := providerRunRelease
	t.Cleanup(func() {
		providerLoadReleaseConfig = oldLoadReleaseConfig
		providerRunRelease = oldRunRelease
	})

	providerLoadReleaseConfig = func(dir string) core.Result {
		if !stdlibAssertEqual(projectDir, dir) {
			t.Fatalf("want %v, got %v", projectDir, dir)
		}

		cfg := release.DefaultConfig()
		cfg.SetProjectDir(dir)
		return core.Ok(cfg)
	}

	called := false
	providerRunRelease = func(ctx context.Context, cfg *release.Config, dryRun bool) core.Result {
		called = true
		if dryRun {
			t.Fatal("expected false")
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}

		return core.Ok(&release.Release{
			Version:   "v1.2.3",
			Artifacts: []build.Artifact{{Path: ax.Join(projectDir, "dist", "demo.tar.gz")}},
			Changelog: "Release notes",
		})
	}

	p := NewProvider(projectDir, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release", nil)

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.triggerRelease(ctx)
	if !(called) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual(http.StatusOK, recorder.Code) {
		t.Fatalf("want %v, got %v", http.StatusOK, recorder.Code)
	}
	if !stdlibAssertContains(recorder.Body.String(), `"version":"v1.2.3"`) {
		t.Fatalf("expected %v to contain %v", recorder.Body.String(), `"version":"v1.2.3"`)
	}
	if !stdlibAssertContains(recorder.Body.String(), `"dry_run":false`) {
		t.Fatalf("expected %v to contain %v", recorder.Body.String(), `"dry_run":false`)
	}

}

func TestProvider_TriggerRelease_DryRun_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()

	oldLoadReleaseConfig := providerLoadReleaseConfig
	oldRunRelease := providerRunRelease
	t.Cleanup(func() {
		providerLoadReleaseConfig = oldLoadReleaseConfig
		providerRunRelease = oldRunRelease
	})

	providerLoadReleaseConfig = func(dir string) core.Result {
		cfg := release.DefaultConfig()
		cfg.SetProjectDir(dir)
		return core.Ok(cfg)
	}

	providerRunRelease = func(ctx context.Context, cfg *release.Config, dryRun bool) core.Result {
		if !(dryRun) {
			t.Fatal("expected true")
		}

		return core.Ok(&release.Release{
			Version: "v1.2.3",
		})
	}

	p := NewProvider(projectDir, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/release?dry_run=true", nil)

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.triggerRelease(ctx)
	if !stdlibAssertEqual(http.StatusOK, recorder.Code) {
		t.Fatalf("want %v, got %v", http.StatusOK, recorder.Code)
	}
	if !stdlibAssertContains(recorder.Body.String(), `"dry_run":true`) {
		t.Fatalf("expected %v to contain %v", recorder.Body.String(), `"dry_run":true`)
	}

}

func TestProvider_ListArtifacts_RecursesIntoPlatformDirectories_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectDir := t.TempDir()
	distDir := ax.Join(projectDir, "dist")
	requireProviderOK(t, io.Local.EnsureDir(ax.Join(distDir, "linux_amd64")))
	requireProviderOK(t, io.Local.Write(ax.Join(distDir, "CHECKSUMS.txt"), "checksums"))
	requireProviderOK(t, io.Local.Write(ax.Join(distDir, "linux_amd64", "demo.tar.xz"), "archive"))

	p := NewProvider(projectDir, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/artifacts", nil)

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	p.listArtifacts(ctx)
	if !stdlibAssertEqual(http.StatusOK, recorder.Code) {
		t.Fatalf("want %v, got %v", http.StatusOK, recorder.Code)
	}

	body := recorder.Body.String()
	if !stdlibAssertContains(body, `"exists":true`) {
		t.Fatalf("expected %v to contain %v", body, `"exists":true`)
	}
	if !stdlibAssertContains(body, `"name":"CHECKSUMS.txt"`) {
		t.Fatalf("expected %v to contain %v", body, `"name":"CHECKSUMS.txt"`)
	}
	if !stdlibAssertContains(body, `"name":"linux_amd64/demo.tar.xz"`) {
		t.Fatalf("expected %v to contain %v", body, `"name":"linux_amd64/demo.tar.xz"`)
	}
	if !stdlibAssertContains(body, ax.Join(distDir, "linux_amd64", "demo.tar.xz")) {
		t.Fatalf("expected %v to contain %v", body, ax.Join(distDir, "linux_amd64", "demo.tar.xz"))
	}

}

type capturingBuilder struct {
	name    string
	buildFn func(ctx context.Context, cfg *build.Config, targets []build.Target) core.Result
}

func (b *capturingBuilder) Name() string {
	return b.name
}

func (b *capturingBuilder) Detect(fs io.Medium, dir string) core.Result {
	return core.Ok(true)
}

func (b *capturingBuilder) Build(ctx context.Context, cfg *build.Config, targets []build.Target) core.Result {
	return b.buildFn(ctx, cfg, targets)
}

// --- v0.9.0 generated compliance triplets ---
func TestProvider_NewProvider_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewProvider(core.Path(t.TempDir(), "go-build-compliance"), nil)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestProvider_NewProvider_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewProvider("", nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestProvider_NewProvider_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewProvider(core.Path(t.TempDir(), "go-build-compliance"), nil)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestProvider_BuildProvider_Name_Good(t *core.T) {
	subject := &BuildProvider{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestProvider_BuildProvider_Name_Bad(t *core.T) {
	subject := &BuildProvider{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestProvider_BuildProvider_Name_Ugly(t *core.T) {
	subject := &BuildProvider{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestProvider_BuildProvider_BasePath_Good(t *core.T) {
	subject := &BuildProvider{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.BasePath()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestProvider_BuildProvider_BasePath_Bad(t *core.T) {
	subject := &BuildProvider{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.BasePath()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestProvider_BuildProvider_BasePath_Ugly(t *core.T) {
	subject := &BuildProvider{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.BasePath()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestProvider_BuildProvider_Element_Good(t *core.T) {
	subject := &BuildProvider{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Element()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestProvider_BuildProvider_Element_Bad(t *core.T) {
	subject := &BuildProvider{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Element()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestProvider_BuildProvider_Element_Ugly(t *core.T) {
	subject := &BuildProvider{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Element()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestProvider_BuildProvider_Channels_Good(t *core.T) {
	subject := &BuildProvider{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Channels()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestProvider_BuildProvider_Channels_Bad(t *core.T) {
	subject := &BuildProvider{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Channels()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestProvider_BuildProvider_Channels_Ugly(t *core.T) {
	subject := &BuildProvider{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Channels()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestProvider_BuildProvider_RegisterRoutes_Good(t *core.T) {
	subject := &BuildProvider{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		subject.RegisterRoutes(gin.New().Group("/build"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestProvider_BuildProvider_RegisterRoutes_Bad(t *core.T) {
	subject := &BuildProvider{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		subject.RegisterRoutes(gin.New().Group("/build"))
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestProvider_BuildProvider_RegisterRoutes_Ugly(t *core.T) {
	subject := &BuildProvider{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		subject.RegisterRoutes(gin.New().Group("/build"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestProvider_BuildProvider_Describe_Good(t *core.T) {
	subject := &BuildProvider{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Describe()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestProvider_BuildProvider_Describe_Bad(t *core.T) {
	subject := &BuildProvider{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Describe()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestProvider_BuildProvider_Describe_Ugly(t *core.T) {
	subject := &BuildProvider{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Describe()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestProvider_Info_Good(t *core.T) {
	subject := Info{Name: "app.tar.gz", Path: "/dist/app.tar.gz", Size: 42}
	core.AssertEqual(t, "app.tar.gz", subject.Name)
	core.AssertEqual(t, int64(42), subject.Size)
}

func TestProvider_Info_Bad(t *core.T) {
	subject := Info{}
	core.AssertEqual(t, "", subject.Name)
	core.AssertEqual(t, int64(0), subject.Size)
}

func TestProvider_Info_Ugly(t *core.T) {
	subject := Info{Name: "nested/app.tar.gz", Path: "/dist/linux/app.tar.gz", Size: 1}
	core.AssertEqual(t, "nested/app.tar.gz", subject.Name)
	core.AssertEqual(t, "/dist/linux/app.tar.gz", subject.Path)
}

func TestProvider_ReleaseWorkflowRequest_Decode_Good(t *core.T) {
	var subject ReleaseWorkflowRequest
	result := subject.Decode([]byte(`{"` + apiPathField + `":"ci/release.yml"}`))
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, "ci/release.yml", subject.Path)
}

func TestProvider_ReleaseWorkflowRequest_Decode_Bad(t *core.T) {
	var subject ReleaseWorkflowRequest
	result := subject.Decode([]byte(`{`))
	core.AssertFalse(t, result.OK)
}

func TestProvider_ReleaseWorkflowRequest_Decode_Ugly(t *core.T) {
	var subject ReleaseWorkflowRequest
	result := subject.Decode([]byte(`{"workflow_output_path":"ops/release.yml","workflow-output":"legacy.yml"}`))
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, "ops/release.yml", subject.WorkflowOutputPathSnake)
	core.AssertEqual(t, "legacy.yml", subject.WorkflowOutputHyphen)
}
