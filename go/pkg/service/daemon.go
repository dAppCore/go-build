package service

import (
	"context"
	"net/http"
	"sort"
	"time"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	coreapi "dappco.re/go/build/pkg/api"
	providerpkg "dappco.re/go/build/pkg/api/provider"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/build/pkg/build/builders"
	"dappco.re/go/build/pkg/events"
	"dappco.re/go/build/pkg/release"
	storage "dappco.re/go/build/pkg/storage"
)

type apiEngine interface {
	Register(group coreapi.RouteGroup)
	Serve(ctx context.Context) core.Result
}

type processDaemon interface {
	Start() core.Result
	Stop() core.Result
	SetReady(ready bool)
}

	var (
		newHub           = events.NewHub
		newBuildProvider = func(projectDir string, hub *events.Hub) providerpkg.Provider {
			return coreapi.NewProvider(projectDir, hub)
		}
	newProviderRegistry    = providerpkg.NewRegistry
	newAPIEngine           = func(opts ...coreapi.Option) core.Result { return coreapi.New(opts...) }
	newMCPServer           = defaultNewMCPServer
	newAgenticOrchestrator = defaultNewAgenticOrchestrator
	runWatchedBuild        = defaultRunWatchedBuild
	discoverProject        = func(projectDir string) core.Result {
		return build.DiscoverFull(storage.Local, projectDir)
	}
	newProcessDaemon = func(opts daemonOptions) processDaemon { return newManagedDaemon(opts) }
)

// Run starts the background daemon for cfg until ctx is cancelled.
func Run(ctx context.Context, cfg Config) core.Result {
	if ctx == nil {
		return core.Fail(core.E("service.Run", "daemon context is required", nil))
	}

	cfg = cfg.Normalized()
	created := ax.MkdirAll(core.PathDir(cfg.PIDFile), 0o755)
	if !created.OK {
		return core.Fail(core.E("service.Run", "failed to create pid directory", core.NewError(created.Error())))
	}

	daemon := newProcessDaemon(daemonOptions{
		PIDFile:         cfg.PIDFile,
		HealthAddr:      cfg.HealthAddr,
		ShutdownTimeout: 30 * time.Second,
	})
	started := daemon.Start()
	if !started.OK {
		return started
	}

	hub := newHub()
	go hub.Run(ctx)

	registry := newProviderRegistry()
	if registry == nil {
		registry = providerpkg.NewRegistry()
	}

	buildProvider := newBuildProvider(cfg.ProjectDir, hub)
	if buildProvider != nil {
		registry.Add(buildProvider)
	}

	mcpServer := newMCPServer(cfg, registry, hub)
	agentic := newAgenticOrchestrator(cfg, registry, hub)
	if agentic != nil {
		go agentic.Run(ctx)
	}

	engineResult := newAPIEngine(
		coreapi.WithAddr(cfg.APIAddr),
		coreapi.WithWSPath("/api/v1/build/events"),
		coreapi.WithWSHandler(hub.Handler()),
	)
	if !engineResult.OK {
		stopped := daemon.Stop()
		if !stopped.OK {
			return core.Fail(core.E("service.Run", engineResult.Error()+": "+stopped.Error(), nil))
		}
		return engineResult
	}
	engine := engineResult.Value.(apiEngine)
	if buildProvider != nil {
		engine.Register(buildProvider)
	}
	if mcpServer != nil {
		engine.Register(mcpServer)
	}

	emitter := daemonEventEmitter{
		hub:     hub,
		agentic: agentic,
	}
	if mcpServer != nil {
		emitter.Emit("service.mcp.ready", map[string]any{
			"projectDir": cfg.ProjectDir,
			"basePath":   mcpServer.BasePath(),
			"tools":      mcpToolNames(mcpServer),
		})
	}

	serverErrCh := make(chan core.Result, 1)
	go func() {
		serverErrCh <- engine.Serve(ctx)
	}()

	if cfg.AutoRebuild {
		go watchLoop(ctx, cfg, emitter)
	}
	go schedulerLoop(ctx, cfg, emitter)

	daemon.SetReady(true)

	select {
	case served := <-serverErrCh:
		stopped := daemon.Stop()
		if served.OK || core.Contains(served.Error(), context.Canceled.Error()) || core.Contains(served.Error(), http.ErrServerClosed.Error()) {
			return stopped
		}
		if !stopped.OK {
			return core.Fail(core.E("service.Run", served.Error()+": "+stopped.Error(), nil))
		}
		return served
	case <-ctx.Done():
		return daemon.Stop()
	}
}

func defaultRunWatchedBuild(ctx context.Context, projectDir string) core.Result {
	filesystem := storage.Local

	var buildConfig *build.BuildConfig
	if build.ConfigExists(filesystem, projectDir) {
		loaded := build.LoadConfig(filesystem, projectDir)
		if !loaded.OK {
			return core.Fail(core.E("service.defaultRunWatchedBuild", "failed to load build config", core.NewError(loaded.Error())))
		}
		buildConfig = loaded.Value.(*build.BuildConfig)
	}

	pipeline := &build.Pipeline{
		FS:             filesystem,
		ResolveBuilder: builders.ResolveBuilder,
		ResolveVersion: release.DetermineVersionWithContext,
	}

	plan := pipeline.Plan(ctx, build.PipelineRequest{
		ProjectDir:  projectDir,
		BuildConfig: buildConfig,
	})
	if !plan.OK {
		return plan
	}

	return pipeline.Run(ctx, plan.Value.(*build.PipelinePlan))
}

func schedulerLoop(ctx context.Context, cfg Config, emitter daemonEventEmitter) {
	ticker := time.NewTicker(cfg.ScheduleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			discovery := discoverProject(cfg.ProjectDir)
			payload := map[string]any{
				"projectDir": cfg.ProjectDir,
				"timestamp":  time.Now().UTC().Format(time.RFC3339),
			}
			if !discovery.OK {
				payload["error"] = discovery.Error()
			} else {
				payload["types"] = discoveryTypes(discovery.Value.(*build.DiscoveryResult))
			}
			emitter.Emit("service.discovery", payload)
		}
	}
}

func watchLoop(ctx context.Context, cfg Config, emitter daemonEventEmitter) {
	ticker := time.NewTicker(cfg.WatchInterval)
	defer ticker.Stop()

	currentResult := snapshotFiles(cfg)
	if !currentResult.OK {
		emitter.Emit("service.watch.error", map[string]any{"error": currentResult.Error()})
		return
	}
	current := currentResult.Value.(map[string]time.Time)

	buildQueue := make(chan []string, 1)
	go buildWorker(ctx, cfg, emitter, buildQueue)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			nextResult := snapshotFiles(cfg)
			if !nextResult.OK {
				emitter.Emit("service.watch.error", map[string]any{"error": nextResult.Error()})
				continue
			}
			next := nextResult.Value.(map[string]time.Time)

			changed := diffSnapshots(current, next)
			current = next
			if len(changed) == 0 {
				continue
			}

			emitter.Emit("service.watch.changed", map[string]any{
				"projectDir": cfg.ProjectDir,
				"paths":      changed,
			})

			select {
			case buildQueue <- changed:
			default:
			}
		}
	}
}

func buildWorker(ctx context.Context, cfg Config, emitter daemonEventEmitter, buildQueue <-chan []string) {
	for {
		select {
		case <-ctx.Done():
			return
		case changed := <-buildQueue:
			emitter.Emit("build.started", map[string]any{
				"projectDir": cfg.ProjectDir,
				"paths":      changed,
			})

			built := runWatchedBuild(ctx, cfg.ProjectDir)
			if !built.OK {
				emitter.Emit("build.failed", map[string]any{
					"projectDir": cfg.ProjectDir,
					"error":      built.Error(),
				})
				continue
			}

			emitter.Emit("build.complete", map[string]any{
				"projectDir": cfg.ProjectDir,
				"paths":      changed,
			})
		}
	}
}

func snapshotFiles(cfg Config) core.Result {
	snapshot := make(map[string]time.Time)

	for _, root := range cfg.WatchPaths {
		err := core.PathWalkDir(root, func(path string, entry core.FsDirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if shouldSkipWatchPath(cfg.ProjectDir, path) {
				if entry.IsDir() {
					return core.PathSkipDir
				}
				return nil
			}
			if entry.IsDir() {
				return nil
			}

			info, err := entry.Info()
			if err != nil {
				return err
			}
			snapshot[path] = info.ModTime()
			return nil
		})
		if err != nil {
			return core.Fail(err)
		}
	}

	return core.Ok(snapshot)
}

func shouldSkipWatchPath(projectDir, path string) bool {
	projectDir = core.PathJoin(projectDir)
	path = core.PathJoin(path)

	skipRoots := []string{
		core.PathJoin(projectDir, ".git"),
		core.PathJoin(projectDir, "dist"),
		core.PathJoin(projectDir, ".core", "cache"),
		core.PathJoin(projectDir, ".core", "build", "app"),
	}
	for _, root := range skipRoots {
		root = core.PathJoin(root)
		if path == root || core.HasPrefix(path, root+string(core.PathSeparator)) {
			return true
		}
	}
	return false
}

func diffSnapshots(previous, current map[string]time.Time) []string {
	changed := make([]string, 0)
	seen := make(map[string]struct{})

	for path, modTime := range current {
		if previousModTime, ok := previous[path]; !ok || !previousModTime.Equal(modTime) {
			changed = append(changed, path)
			seen[path] = struct{}{}
		}
	}
	for path := range previous {
		if _, ok := current[path]; !ok {
			if _, exists := seen[path]; exists {
				continue
			}
			changed = append(changed, path)
		}
	}

	sort.Strings(changed)
	return changed
}

func discoveryTypes(discovery *build.DiscoveryResult) []string {
	if discovery == nil {
		return nil
	}

	types := make([]string, 0, len(discovery.Types))
	for _, projectType := range discovery.Types {
		types = append(types, string(projectType))
	}
	sort.Strings(types)
	return types
}

func sendEvent(hub *events.Hub, channel string, payload any) {
	if hub == nil {
		return
	}
	sent := hub.SendToChannel(channel, events.Message{
		Type: events.TypeEvent,
		Data: payload,
	})
	if !sent.OK {
		return
	}
}

type daemonEventEmitter struct {
	hub     *events.Hub
	agentic agenticOrchestrator
}

func (e daemonEventEmitter) Emit(channel string, payload any) {
	sendEvent(e.hub, channel, payload)
	if e.agentic != nil {
		e.agentic.Notify(channel, payload)
	}
}
