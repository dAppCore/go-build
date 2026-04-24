package service

import (
	"context"

	providerpkg "dappco.re/go/api/pkg/provider"
	"dappco.re/go/ws"
)

type agenticOrchestrator interface {
	Run(ctx context.Context)
	Notify(channel string, payload any)
}

type daemonAgentic struct {
	cfg      Config
	registry *providerpkg.Registry
	events   chan daemonAgenticEvent
	emit     func(channel string, payload any)
}

type daemonAgenticEvent struct {
	channel string
	payload any
}

func defaultNewAgenticOrchestrator(cfg Config, registry *providerpkg.Registry, hub *ws.Hub) agenticOrchestrator {
	return newDaemonAgentic(cfg, registry, func(channel string, payload any) {
		sendEvent(hub, channel, payload)
	})
}

func newDaemonAgentic(cfg Config, registry *providerpkg.Registry, emit func(channel string, payload any)) agenticOrchestrator {
	if emit == nil {
		emit = func(string, any) {}
	}
	if registry == nil {
		registry = providerpkg.NewRegistry()
	}

	return &daemonAgentic{
		cfg:      cfg,
		registry: registry,
		events:   make(chan daemonAgenticEvent, 32),
		emit:     emit,
	}
}

func (o *daemonAgentic) Run(ctx context.Context) {
	if o == nil {
		return
	}

	o.emit("agentic.ready", map[string]any{
		"projectDir": cfgProjectDir(o.cfg),
		"providers":  o.registry.Info(),
	})

	for {
		select {
		case <-ctx.Done():
			o.emit("agentic.stopped", map[string]any{
				"projectDir": cfgProjectDir(o.cfg),
			})
			return
		case event := <-o.events:
			o.handleEvent(event)
		}
	}
}

func (o *daemonAgentic) Notify(channel string, payload any) {
	if o == nil {
		return
	}

	select {
	case o.events <- daemonAgenticEvent{channel: channel, payload: payload}:
	default:
	}
}

func (o *daemonAgentic) handleEvent(event daemonAgenticEvent) {
	payload := cloneAgenticPayload(event.payload)
	payload["source_event"] = event.channel

	switch event.channel {
	case "service.discovery":
		payload["providers"] = o.registry.Info()
		payload["recommended_tool"] = "project_discover"
		o.emit("agentic.context", payload)
	case "service.watch.changed":
		payload["recommended_tool"] = "build_run"
		payload["reason"] = "source_change_detected"
		o.emit("agentic.plan", payload)
	case "build.started":
		payload["task"] = "build_run"
		o.emit("agentic.task.started", payload)
	case "build.complete":
		payload["task"] = "build_run"
		o.emit("agentic.task.complete", payload)
	case "build.failed":
		payload["task"] = "build_run"
		o.emit("agentic.task.failed", payload)
	}
}

func cloneAgenticPayload(payload any) map[string]any {
	typed, ok := payload.(map[string]any)
	if !ok || typed == nil {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(typed))
	for key, value := range typed {
		cloned[key] = value
	}
	return cloned
}

func cfgProjectDir(cfg Config) string {
	if cfg.ProjectDir == "" {
		return "."
	}
	return cfg.ProjectDir
}
