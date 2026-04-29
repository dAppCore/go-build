package service

import (
	"context"
	"sync"
	"testing"
	"time"

	coreapi "dappco.re/go/api"
	providerpkg "dappco.re/go/api/pkg/provider"
	"dappco.re/go/process"
	"dappco.re/go/ws"
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
		if stdlibAssertNil(registry.Get("build")) {
			t.Fatal("expected non-nil")
		}

		return stubRouteGroup{name: "mcp", basePath: "/api/v1/mcp"}
	}
	newAgenticOrchestrator = func(cfg Config, registry *providerpkg.Registry, hub *ws.Hub) agenticOrchestrator {
		if stdlibAssertNil(registry.Get("build")) {
			t.Fatal("expected non-nil")
		}

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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
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
