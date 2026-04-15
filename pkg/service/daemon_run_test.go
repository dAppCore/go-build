package service

import (
	"context"
	"sync"
	"testing"
	"time"

	coreapi "dappco.re/go/core/api"
	providerpkg "dappco.re/go/core/api/pkg/provider"
	"dappco.re/go/core/process"
	"dappco.re/go/core/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubAPIEngine struct {
	mu           sync.Mutex
	groups       []string
	serveStarted chan struct{}
	once         sync.Once
}

func (e *stubAPIEngine) Register(group coreapi.RouteGroup) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.groups = append(e.groups, group.Name())
}

func (e *stubAPIEngine) Serve(ctx context.Context) error {
	e.once.Do(func() {
		close(e.serveStarted)
	})
	<-ctx.Done()
	return context.Canceled
}

type stubProcessDaemon struct {
	started bool
	stopped bool
	ready   []bool
}

func (d *stubProcessDaemon) Start() error {
	d.started = true
	return nil
}

func (d *stubProcessDaemon) Stop() error {
	d.stopped = true
	return nil
}

func (d *stubProcessDaemon) SetReady(ready bool) {
	d.ready = append(d.ready, ready)
}

type stubAgenticOrchestrator struct {
	runStarted    chan struct{}
	mu            sync.Mutex
	notifications []string
}

func (o *stubAgenticOrchestrator) Run(ctx context.Context) {
	close(o.runStarted)
	<-ctx.Done()
}

func (o *stubAgenticOrchestrator) Notify(channel string, payload any) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.notifications = append(o.notifications, channel)
}

func TestRun_WiresMCPAndAgentic_Good(t *testing.T) {
	originalHub := newHub
	originalBuildProvider := newBuildProvider
	originalRegistry := newProviderRegistry
	originalEngine := newAPIEngine
	originalMCP := newMCPServer
	originalAgentic := newAgenticOrchestrator
	originalDaemon := newProcessDaemon
	t.Cleanup(func() {
		newHub = originalHub
		newBuildProvider = originalBuildProvider
		newProviderRegistry = originalRegistry
		newAPIEngine = originalEngine
		newMCPServer = originalMCP
		newAgenticOrchestrator = originalAgentic
		newProcessDaemon = originalDaemon
	})

	projectDir := t.TempDir()
	stubDaemon := &stubProcessDaemon{}
	engine := &stubAPIEngine{serveStarted: make(chan struct{})}
	agentic := &stubAgenticOrchestrator{runStarted: make(chan struct{})}

	newHub = ws.NewHub
	newBuildProvider = func(projectDir string, hub *ws.Hub) providerpkg.Provider {
		return stubDaemonProvider{
			name:     "build",
			basePath: "/api/v1/build",
			channels: []string{"build.started"},
		}
	}
	newProviderRegistry = providerpkg.NewRegistry
	newAPIEngine = func(opts ...coreapi.Option) (apiEngine, error) {
		return engine, nil
	}
	newMCPServer = func(cfg Config, registry *providerpkg.Registry, hub *ws.Hub) coreapi.RouteGroup {
		require.NotNil(t, registry.Get("build"))
		return stubRouteGroup{name: "mcp", basePath: "/api/v1/mcp"}
	}
	newAgenticOrchestrator = func(cfg Config, registry *providerpkg.Registry, hub *ws.Hub) agenticOrchestrator {
		require.NotNil(t, registry.Get("build"))
		return agentic
	}
	newProcessDaemon = func(opts process.DaemonOptions) processDaemon {
		return stubDaemon
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, Config{
			ProjectDir:       projectDir,
			AutoRebuild:      false,
			ScheduleInterval: time.Hour,
		})
	}()

	select {
	case <-engine.serveStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for API engine")
	}

	select {
	case <-agentic.runStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for agentic orchestrator")
	}

	cancel()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for daemon shutdown")
	}

	assert.True(t, stubDaemon.started)
	assert.True(t, stubDaemon.stopped)
	assert.Equal(t, []bool{true}, stubDaemon.ready)

	engine.mu.Lock()
	assert.Contains(t, engine.groups, "build")
	assert.Contains(t, engine.groups, "mcp")
	engine.mu.Unlock()

	agentic.mu.Lock()
	assert.Contains(t, agentic.notifications, "service.mcp.ready")
	agentic.mu.Unlock()
}
