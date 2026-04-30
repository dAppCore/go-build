package service

import (
	"context"
	"sync"
	"testing"
	"time"

	core "dappco.re/go"
	coreapi "dappco.re/go/build/pkg/api"
	providerpkg "dappco.re/go/build/pkg/api/provider"
	events "dappco.re/go/build/pkg/events"
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

func (e *stubAPIEngine) Serve(ctx context.Context) core.Result {
	e.once.Do(func() {
		close(e.serveStarted)
	})
	<-ctx.Done()
	return core.Fail(context.Canceled)
}

type stubProcessDaemon struct {
	started bool
	stopped bool
	ready   []bool
}

func (d *stubProcessDaemon) Start() core.Result {
	d.started = true
	return core.Ok(nil)
}

func (d *stubProcessDaemon) Stop() core.Result {
	d.stopped = true
	return core.Ok(nil)
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

func TestRun_WiresMCPAndAgenticGood(t *testing.T) {
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

	newHub = events.NewHub
	newBuildProvider = func(projectDir string, hub *events.Hub) providerpkg.Provider {
		return stubDaemonProvider{
			name:     "build",
			basePath: "/api/v1/build",
			channels: []string{"build.started"},
		}
	}
	newProviderRegistry = providerpkg.NewRegistry
	newAPIEngine = func(opts ...coreapi.Option) core.Result {
		return core.Ok(engine)
	}
	newMCPServer = func(cfg Config, registry *providerpkg.Registry, hub *events.Hub) coreapi.RouteGroup {
		if stdlibAssertNil(registry.Get("build")) {
			t.Fatal("expected non-nil")
		}

		return stubRouteGroup{name: "mcp", basePath: "/api/v1/mcp"}
	}
	newAgenticOrchestrator = func(cfg Config, registry *providerpkg.Registry, hub *events.Hub) agenticOrchestrator {
		if stdlibAssertNil(registry.Get("build")) {
			t.Fatal("expected non-nil")
		}

		return agentic
	}
	newProcessDaemon = func(opts daemonOptions) processDaemon {
		return stubDaemon
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan core.Result, 1)
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
	case result := <-done:
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for daemon shutdown")
	}
	if !(stubDaemon.started) {
		t.Fatal("expected true")
	}
	if !(stubDaemon.stopped) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual([]bool{true}, stubDaemon.ready) {
		t.Fatalf("want %v, got %v", []bool{true}, stubDaemon.ready)
	}

	engine.mu.Lock()
	if !stdlibAssertContains(engine.groups, "build") {
		t.Fatalf("expected %v to contain %v", engine.groups, "build")
	}
	if !stdlibAssertContains(engine.groups, "mcp") {
		t.Fatalf("expected %v to contain %v", engine.groups, "mcp")
	}

	engine.mu.Unlock()

	agentic.mu.Lock()
	if !stdlibAssertContains(agentic.notifications, "service.mcp.ready") {
		t.Fatalf("expected %v to contain %v", agentic.notifications, "service.mcp.ready")
	}

	agentic.mu.Unlock()
}
