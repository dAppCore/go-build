package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	coreapi "dappco.re/go/core/api"
	providerpkg "dappco.re/go/core/api/pkg/provider"
	"dappco.re/go/build/internal/ax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCP_DefaultNewMCPServer_ExposesDaemonTools_Good(t *testing.T) {
	projectDir := t.TempDir()
	registry := providerpkg.NewRegistry()
	registry.Add(stubDaemonProvider{
		name:     "build",
		basePath: "/api/v1/build",
		channels: []string{"build.started"},
	})

	group := defaultNewMCPServer(DefaultConfig(projectDir).Normalized(), registry, nil)

	assert.Equal(t, "/api/v1/mcp", group.BasePath())
	assert.Equal(t, []string{
		"build_run",
		"daemon_status",
		"project_discover",
		"providers_list",
	}, mcpToolNames(group))
}

func TestMCP_BuildRunAndDiscover_Good(t *testing.T) {
	projectDir := t.TempDir()
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, "go.mod"), []byte("module example.com/demo\n"), 0o644))

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
	runWatchedBuild = func(ctx context.Context, dir string) error {
		called = true
		assert.Equal(t, projectDir, dir)
		return nil
	}

	group := defaultNewMCPServer(DefaultConfig(projectDir).Normalized(), registry, nil)

	engine, err := coreapi.New()
	require.NoError(t, err)
	engine.Register(group)

	server := httptest.NewServer(engine.Handler())
	defer server.Close()

	buildResponse := postTool(t, server.URL+"/api/v1/mcp/build_run")
	assert.Contains(t, buildResponse, `"success":true`)
	assert.True(t, called)

	discoverResponse := postTool(t, server.URL+"/api/v1/mcp/project_discover")
	assert.Contains(t, discoverResponse, `"success":true`)
	assert.Contains(t, discoverResponse, `"primary_stack":"go"`)
}

func postTool(t *testing.T, url string) string {
	t.Helper()

	response, err := http.Post(url, "application/json", bytes.NewBufferString(`{}`))
	require.NoError(t, err)
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode, string(body))

	return string(body)
}
