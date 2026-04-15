package service

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	coreapi "dappco.re/go/core/api"
	buildapi "dappco.re/go/core/build/pkg/api"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/build/pkg/build/builders"
	"dappco.re/go/core/build/pkg/release"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
	"dappco.re/go/core/process"
	"dappco.re/go/core/ws"
)

type apiEngine interface {
	Register(group coreapi.RouteGroup)
	Serve(ctx context.Context) error
}

var (
	newHub           = ws.NewHub
	newBuildProvider = func(projectDir string, hub *ws.Hub) coreapi.RouteGroup { return buildapi.NewProvider(projectDir, hub) }
	newAPIEngine     = func(opts ...coreapi.Option) (apiEngine, error) { return coreapi.New(opts...) }
	runWatchedBuild  = defaultRunWatchedBuild
	discoverProject  = func(projectDir string) (*build.DiscoveryResult, error) {
		return build.DiscoverFull(io.Local, projectDir)
	}
	newProcessDaemon = process.NewDaemon
)

// Run starts the background daemon for cfg until ctx is cancelled.
func Run(ctx context.Context, cfg Config) error {
	if ctx == nil {
		return coreerr.E("service.Run", "daemon context is required", nil)
	}

	cfg = cfg.Normalized()
	if err := os.MkdirAll(filepath.Dir(cfg.PIDFile), 0o755); err != nil {
		return coreerr.E("service.Run", "failed to create pid directory", err)
	}

	daemon := newProcessDaemon(process.DaemonOptions{
		PIDFile:         cfg.PIDFile,
		HealthAddr:      cfg.HealthAddr,
		ShutdownTimeout: 30 * time.Second,
	})
	if err := daemon.Start(); err != nil {
		return err
	}

	hub := newHub()
	go hub.Run(ctx)

	engine, err := newAPIEngine(
		coreapi.WithAddr(cfg.APIAddr),
		coreapi.WithWSPath("/api/v1/build/events"),
		coreapi.WithWSHandler(hub.Handler()),
	)
	if err != nil {
		stopErr := daemon.Stop()
		return errors.Join(err, stopErr)
	}
	engine.Register(newBuildProvider(cfg.ProjectDir, hub))

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- engine.Serve(ctx)
	}()

	if cfg.AutoRebuild {
		go watchLoop(ctx, cfg, hub)
	}
	go schedulerLoop(ctx, cfg, hub)

	daemon.SetReady(true)

	select {
	case err := <-serverErrCh:
		stopErr := daemon.Stop()
		if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, http.ErrServerClosed) {
			return stopErr
		}
		return errors.Join(err, stopErr)
	case <-ctx.Done():
		return daemon.Stop()
	}
}

func defaultRunWatchedBuild(ctx context.Context, projectDir string) error {
	filesystem := io.Local

	buildConfig, err := build.LoadConfig(filesystem, projectDir)
	if err != nil {
		return coreerr.E("service.defaultRunWatchedBuild", "failed to load build config", err)
	}

	pipeline := &build.Pipeline{
		FS:             filesystem,
		ResolveBuilder: builders.ResolveBuilder,
		ResolveVersion: release.DetermineVersionWithContext,
	}

	plan, err := pipeline.Plan(ctx, build.PipelineRequest{
		ProjectDir:  projectDir,
		BuildConfig: buildConfig,
	})
	if err != nil {
		return err
	}

	_, err = pipeline.Run(ctx, plan)
	return err
}

func schedulerLoop(ctx context.Context, cfg Config, hub *ws.Hub) {
	ticker := time.NewTicker(cfg.ScheduleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			discovery, err := discoverProject(cfg.ProjectDir)
			payload := map[string]any{
				"projectDir": cfg.ProjectDir,
				"timestamp":  time.Now().UTC().Format(time.RFC3339),
			}
			if err != nil {
				payload["error"] = err.Error()
			} else {
				payload["types"] = discoveryTypes(discovery)
			}
			sendEvent(hub, "service.discovery", payload)
		}
	}
}

func watchLoop(ctx context.Context, cfg Config, hub *ws.Hub) {
	ticker := time.NewTicker(cfg.WatchInterval)
	defer ticker.Stop()

	current, err := snapshotFiles(cfg)
	if err != nil {
		sendEvent(hub, "service.watch.error", map[string]any{"error": err.Error()})
		return
	}

	buildQueue := make(chan []string, 1)
	go buildWorker(ctx, cfg, hub, buildQueue)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			next, err := snapshotFiles(cfg)
			if err != nil {
				sendEvent(hub, "service.watch.error", map[string]any{"error": err.Error()})
				continue
			}

			changed := diffSnapshots(current, next)
			current = next
			if len(changed) == 0 {
				continue
			}

			sendEvent(hub, "service.watch.changed", map[string]any{
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

func buildWorker(ctx context.Context, cfg Config, hub *ws.Hub, buildQueue <-chan []string) {
	for {
		select {
		case <-ctx.Done():
			return
		case changed := <-buildQueue:
			sendEvent(hub, "build.started", map[string]any{
				"projectDir": cfg.ProjectDir,
				"paths":      changed,
			})

			err := runWatchedBuild(ctx, cfg.ProjectDir)
			if err != nil {
				sendEvent(hub, "build.failed", map[string]any{
					"projectDir": cfg.ProjectDir,
					"error":      err.Error(),
				})
				continue
			}

			sendEvent(hub, "build.complete", map[string]any{
				"projectDir": cfg.ProjectDir,
				"paths":      changed,
			})
		}
	}
}

func snapshotFiles(cfg Config) (map[string]time.Time, error) {
	snapshot := make(map[string]time.Time)

	for _, root := range cfg.WatchPaths {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if shouldSkipWatchPath(cfg.ProjectDir, path) {
				if entry.IsDir() {
					return filepath.SkipDir
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
			return nil, err
		}
	}

	return snapshot, nil
}

func shouldSkipWatchPath(projectDir, path string) bool {
	projectDir = filepath.Clean(projectDir)
	path = filepath.Clean(path)

	skipRoots := []string{
		filepath.Join(projectDir, ".git"),
		filepath.Join(projectDir, "dist"),
		filepath.Join(projectDir, ".core", "cache"),
		filepath.Join(projectDir, ".core", "build", "app"),
	}
	for _, root := range skipRoots {
		root = filepath.Clean(root)
		if path == root || strings.HasPrefix(path, root+string(filepath.Separator)) {
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

func sendEvent(hub *ws.Hub, channel string, payload any) {
	if hub == nil {
		return
	}
	_ = hub.SendToChannel(channel, ws.Message{
		Type: ws.TypeEvent,
		Data: payload,
	})
}
