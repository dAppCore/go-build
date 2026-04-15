package service

import (
	"context"
	"testing"
	"time"

	providerpkg "dappco.re/go/core/api/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.Equal(t, "agentic.ready", ready.channel)

	readyPayload, ok := ready.payload.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "/srv/project", readyPayload["projectDir"])

	orchestrator.Notify("service.watch.changed", map[string]any{
		"projectDir": "/srv/project",
		"paths":      []string{"main.go"},
	})

	plan := waitForAgenticEvent(t, events)
	require.Equal(t, "agentic.plan", plan.channel)

	planPayload, ok := plan.payload.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "build_run", planPayload["recommended_tool"])
	assert.Equal(t, "service.watch.changed", planPayload["source_event"])

	orchestrator.Notify("build.failed", map[string]any{
		"projectDir": "/srv/project",
		"error":      "boom",
	})

	failed := waitForAgenticEvent(t, events)
	require.Equal(t, "agentic.task.failed", failed.channel)

	failedPayload, ok := failed.payload.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "build_run", failedPayload["task"])
	assert.Equal(t, "boom", failedPayload["error"])

	cancel()

	stopped := waitForAgenticEvent(t, events)
	require.Equal(t, "agentic.stopped", stopped.channel)
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
