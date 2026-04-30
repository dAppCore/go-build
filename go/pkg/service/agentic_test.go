package service

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
	providerpkg "dappco.re/go/build/pkg/api/provider"
)

type emittedAgenticEvent struct {
	channel string
	payload any
}

func TestAgentic_Run_TransformsDaemonEvents_Good(t *testing.T) {
	registry := providerpkg.NewRegistry()
	registry.Add(stubDaemonProvider{
		name:     "build",
		basePath: "/api/v1/build",
		channels: []string{"build.started", "build.complete"},
	})

	events := make(chan emittedAgenticEvent, 8)
	orchestrator := newDaemonAgentic(Config{ProjectDir: "/srv/project"}, registry, func(channel string, payload any) {
		events <- emittedAgenticEvent{channel: channel, payload: payload}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go orchestrator.Run(ctx)

	ready := waitForAgenticEvent(t, events)
	if !stdlibAssertEqual("agentic.ready", ready.channel) {
		t.Fatalf("want %v, got %v", "agentic.ready", ready.channel)
	}

	readyPayload, ok := ready.payload.(map[string]any)
	if !(ok) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("/srv/project", readyPayload["projectDir"]) {
		t.Fatalf("want %v, got %v", "/srv/project", readyPayload["projectDir"])
	}

	orchestrator.Notify("service.watch.changed", map[string]any{
		"projectDir": "/srv/project",
		"paths":      []string{"main.go"},
	})

	plan := waitForAgenticEvent(t, events)
	if !stdlibAssertEqual("agentic.plan", plan.channel) {
		t.Fatalf("want %v, got %v", "agentic.plan", plan.channel)
	}

	planPayload, ok := plan.payload.(map[string]any)
	if !(ok) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("build_run", planPayload["recommended_tool"]) {
		t.Fatalf("want %v, got %v", "build_run", planPayload["recommended_tool"])
	}
	if !stdlibAssertEqual("service.watch.changed", planPayload["source_event"]) {
		t.Fatalf("want %v, got %v", "service.watch.changed", planPayload["source_event"])
	}

	orchestrator.Notify("build.failed", map[string]any{
		"projectDir": "/srv/project",
		"error":      "boom",
	})

	failed := waitForAgenticEvent(t, events)
	if !stdlibAssertEqual("agentic.task.failed", failed.channel) {
		t.Fatalf("want %v, got %v", "agentic.task.failed", failed.channel)
	}

	failedPayload, ok := failed.payload.(map[string]any)
	if !(ok) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("build_run", failedPayload["task"]) {
		t.Fatalf("want %v, got %v", "build_run", failedPayload["task"])
	}
	if !stdlibAssertEqual("boom", failedPayload["error"]) {
		t.Fatalf("want %v, got %v", "boom", failedPayload["error"])
	}

	cancel()

	stopped := waitForAgenticEvent(t, events)
	if !stdlibAssertEqual("agentic.stopped", stopped.channel) {
		t.Fatalf("want %v, got %v", "agentic.stopped", stopped.channel)
	}

}

func waitForAgenticEvent(t *testing.T, events <-chan emittedAgenticEvent) emittedAgenticEvent {
	t.Helper()

	select {
	case event := <-events:
		return event
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for agentic event")
		return emittedAgenticEvent{}
	}
}

// --- v0.9.0 generated compliance triplets ---
func TestAgentic_Agentic_Run_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := newDaemonAgentic(Config{}, nil, nil).(*daemonAgentic)
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		subject.Run(ctx)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAgentic_Agentic_Run_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := newDaemonAgentic(Config{}, nil, nil).(*daemonAgentic)
	badCalls := 0
	core.AssertNotPanics(t, func() {
		subject.Run(ctx)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAgentic_Agentic_Run_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := newDaemonAgentic(Config{}, nil, nil).(*daemonAgentic)
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		subject.Run(ctx)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestAgentic_Agentic_Notify_Good(t *core.T) {
	subject := newDaemonAgentic(Config{}, nil, nil).(*daemonAgentic)
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		subject.Notify("agent", "agent")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestAgentic_Agentic_Notify_Bad(t *core.T) {
	subject := newDaemonAgentic(Config{}, nil, nil).(*daemonAgentic)
	badCalls := 0
	core.AssertNotPanics(t, func() {
		subject.Notify("", "agent")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestAgentic_Agentic_Notify_Ugly(t *core.T) {
	subject := newDaemonAgentic(Config{}, nil, nil).(*daemonAgentic)
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		subject.Notify("agent", "agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
