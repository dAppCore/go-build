package service

import (
	"context"
	core "dappco.re/go"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	coreapi "dappco.re/go/api"
	providerpkg "dappco.re/go/api/pkg/provider"
	"dappco.re/go/build/internal/ax"
)

func TestMCP_DefaultNewMCPServer_ExposesDaemonToolsGood(t *testing.T) {
	projectDir := t.TempDir()
	registry := providerpkg.NewRegistry()
	registry.Add(stubDaemonProvider{
		name:     "build",
		basePath: "/api/v1/build",
		channels: []string{"build.started"},
	})

	group := defaultNewMCPServer(DefaultConfig(projectDir).Normalized(), registry, nil)
	if !stdlibAssertEqual("/api/v1/mcp", group.BasePath()) {
		t.Fatalf("want %v, got %v", "/api/v1/mcp", group.BasePath())
	}
	if !stdlibAssertEqual([]string{"build_run", "daemon_status", "project_discover", "providers_list"}, mcpToolNames(group)) {
		t.Fatalf("want %v, got %v", []string{"build_run", "daemon_status", "project_discover", "providers_list"}, mcpToolNames(group))
	}

}

func TestMCP_BuildRunAndDiscoverGood(t *testing.T) {
	projectDir := t.TempDir()
	if result := ax.WriteFile(ax.Join(projectDir, "go.mod"), []byte("module example.com/demo\n"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	registry := providerpkg.NewRegistry()
	registry.Add(stubDaemonProvider{
		name:     "build",
		basePath: "/api/v1/build",
		channels: []string{"build.started", "build.complete"},
	})

	originalRun := runWatchedBuild
	t.Cleanup(func() {
		runWatchedBuild = originalRun
	})

	called := false
	runWatchedBuild = func(ctx context.Context, dir string) core.Result {
		called = true
		if !stdlibAssertEqual(projectDir, dir) {
			t.Fatalf("want %v, got %v", projectDir, dir)
		}

		return core.Ok(nil)
	}

	group := defaultNewMCPServer(DefaultConfig(projectDir).Normalized(), registry, nil)

	engineResult := coreapi.New()
	requireServiceOK(t, engineResult)
	engine, ok := engineResult.Value.(*coreapi.Engine)
	if !ok {
		t.Fatalf("expected *coreapi.Engine result, got %T", engineResult.Value)
	}

	engine.Register(group)

	server := httptest.NewServer(engine.Handler())
	defer server.Close()

	buildResponse := postTool(t, server.URL+"/api/v1/mcp/build_run")
	if !stdlibAssertContains(buildResponse, `"success":true`) {
		t.Fatalf("expected %v to contain %v", buildResponse, `"success":true`)
	}
	if !(called) {
		t.Fatal("expected true")
	}

	discoverResponse := postTool(t, server.URL+"/api/v1/mcp/project_discover")
	if !stdlibAssertContains(discoverResponse, `"success":true`) {
		t.Fatalf("expected %v to contain %v", discoverResponse, `"success":true`)
	}
	if !stdlibAssertContains(discoverResponse, `"primary_stack":"go"`) {
		t.Fatalf("expected %v to contain %v", discoverResponse, `"primary_stack":"go"`)
	}

}

func postTool(t *testing.T, url string) string {
	t.Helper()

	response, err := http.Post(url, "application/json", core.NewBufferString(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(http.StatusOK, response.StatusCode) {
		t.Fatal(string(body))
	}

	return string(body)
}
