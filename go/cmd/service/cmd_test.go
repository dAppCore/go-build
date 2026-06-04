package servicecmd

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/internal/cli"
	buildservice "dappco.re/go/build/pkg/service"
)

type stubManager struct {
	install func(buildservice.Config) core.Result
	start   func(buildservice.Config) core.Result
	stop    func(buildservice.Config) core.Result
	remove  func(buildservice.Config) core.Result
}

func (s stubManager) Install(cfg buildservice.Config) core.Result {
	if s.install != nil {
		return s.install(cfg)
	}
	return core.Ok(nil)
}

func (s stubManager) Start(cfg buildservice.Config) core.Result {
	if s.start != nil {
		return s.start(cfg)
	}
	return core.Ok(nil)
}

func (s stubManager) Stop(cfg buildservice.Config) core.Result {
	if s.stop != nil {
		return s.stop(cfg)
	}
	return core.Ok(nil)
}

func (s stubManager) Uninstall(cfg buildservice.Config) core.Result {
	if s.remove != nil {
		return s.remove(cfg)
	}
	return core.Ok(nil)
}

func TestAddServiceCommands_RegistersSubcommandsGood(t *testing.T) {
	c := core.New()

	AddServiceCommands(c)
	if !(c.Command("service").OK) {
		t.Fatal("expected true")
	}
	if !(c.Command("service/install").OK) {
		t.Fatal("expected true")
	}
	if !(c.Command("service/start").OK) {
		t.Fatal("expected true")
	}
	if !(c.Command("service/stop").OK) {
		t.Fatal("expected true")
	}
	if !(c.Command("service/uninstall").OK) {
		t.Fatal("expected true")
	}
	if !(c.Command("service/export").OK) {
		t.Fatal("expected true")
	}
	if !(c.Command("service/run").OK) {
		t.Fatal("expected true")
	}

}

func TestRunServiceInstall_UsesManagerGood(t *testing.T) {
	projectDir := t.TempDir()

	originalGetwd := serviceGetwd
	originalResolve := resolveServiceCfg
	originalManager := serviceManager
	t.Cleanup(func() {
		serviceGetwd = originalGetwd
		resolveServiceCfg = originalResolve
		serviceManager = originalManager
	})

	serviceGetwd = func() core.Result { return core.Ok(projectDir) }
	resolveServiceCfg = func(projectDir string) core.Result {
		return core.Ok(buildservice.Config{
			Name:       "core-build",
			ProjectDir: projectDir,
			APIAddr:    "127.0.0.1:9101",
			HealthAddr: "127.0.0.1:9102",
		})
	}

	called := false
	serviceManager = stubManager{
		install: func(cfg buildservice.Config) core.Result {
			called = true
			if !stdlibAssertEqual(projectDir, cfg.ProjectDir) {
				t.Fatalf("want %v, got %v", projectDir, cfg.ProjectDir)
			}
			if !stdlibAssertEqual("core-build", cfg.Name) {
				t.Fatalf("want %v, got %v", "core-build", cfg.Name)
			}

			return core.Ok(nil)
		},
	}

	requireServiceCmdOK(t, runServiceInstall(serviceRequest{}))
	if !(called) {
		t.Fatal("expected true")
	}

}

func TestRunServiceExport_WritesFileGood(t *testing.T) {
	projectDir := t.TempDir()

	originalGetwd := serviceGetwd
	originalResolve := resolveServiceCfg
	originalExport := exportService
	t.Cleanup(func() {
		serviceGetwd = originalGetwd
		resolveServiceCfg = originalResolve
		exportService = originalExport
	})

	serviceGetwd = func() core.Result { return core.Ok(projectDir) }
	resolveServiceCfg = func(projectDir string) core.Result {
		return core.Ok(buildservice.Config{Name: "core-build", ProjectDir: projectDir})
	}
	exportService = func(cfg buildservice.Config, format string) core.Result {
		return core.Ok(buildservice.ExportedConfig{
			Format:   buildservice.NativeFormatSystemd,
			Filename: "core-build.service",
			Content:  "[Unit]\nDescription=Core Build\n",
		})
	}

	outputPath := core.PathJoin("dist", "core-build.service")
	requireServiceCmdOK(t, runServiceExport(serviceRequest{Output: outputPath}))

	content := requireServiceCmdBytes(t, ax.ReadFile(core.PathJoin(projectDir, outputPath)))
	if !stdlibAssertEqual("[Unit]\nDescription=Core Build\n", string(content)) {
		t.Fatalf("want %v, got %v", "[Unit]\nDescription=Core Build\n", string(content))
	}

}

func TestRunServiceRun_InvokesDaemonGood(t *testing.T) {
	projectDir := t.TempDir()

	originalGetwd := serviceGetwd
	originalResolve := resolveServiceCfg
	originalRun := runDaemon
	t.Cleanup(func() {
		serviceGetwd = originalGetwd
		resolveServiceCfg = originalResolve
		runDaemon = originalRun
	})

	serviceGetwd = func() core.Result { return core.Ok(projectDir) }
	resolveServiceCfg = func(projectDir string) core.Result {
		return core.Ok(buildservice.Config{Name: "core-build", ProjectDir: projectDir})
	}

	called := false
	runDaemon = func(ctx context.Context, cfg buildservice.Config) core.Result {
		called = true
		if !stdlibAssertEqual(projectDir, cfg.ProjectDir) {
			t.Fatalf("want %v, got %v", projectDir, cfg.ProjectDir)
		}

		return core.Ok(nil)
	}

	requireServiceCmdOK(t, runServiceRun(context.Background(), serviceRequest{}))
	if !(called) {
		t.Fatal("expected true")
	}

}

func TestApplyServiceOverrides_BadDuration(t *testing.T) {
	cfg := buildservice.Config{ProjectDir: t.TempDir()}

	message := requireServiceCmdError(t, applyServiceOverrides(&cfg, serviceRequest{WatchInterval: "not-a-duration"}))
	if !stdlibAssertContains(message, "not-a-duration") {
		t.Fatalf("expected %v to contain %v", message, "not-a-duration")
	}

}

func TestRunServiceInstall_BubblesManagerErrorBad(t *testing.T) {
	projectDir := t.TempDir()

	originalGetwd := serviceGetwd
	originalResolve := resolveServiceCfg
	originalManager := serviceManager
	t.Cleanup(func() {
		serviceGetwd = originalGetwd
		resolveServiceCfg = originalResolve
		serviceManager = originalManager
	})

	serviceGetwd = func() core.Result { return core.Ok(projectDir) }
	resolveServiceCfg = func(projectDir string) core.Result {
		return core.Ok(buildservice.Config{Name: "core-build", ProjectDir: projectDir})
	}
	serviceManager = stubManager{
		install: func(buildservice.Config) core.Result {
			return core.Fail(core.NewError("boom"))
		},
	}

	message := requireServiceCmdError(t, runServiceInstall(serviceRequest{}))
	if !stdlibAssertContains(message, "boom") {
		t.Fatalf("expected %v to contain %v", message, "boom")
	}

}

// noopServiceAction is a placeholder executable action used to pre-occupy
// command paths so AddServiceCommands' partial-failure branches can be observed.
func noopServiceAction(core.Options) core.Result { return core.Ok(nil) }

// stubServiceConfig wires the package-level seams to deterministic stand-ins for
// the duration of the test: a fixed working directory, a fixed resolved config,
// and the provided manager. The originals are restored on cleanup.
func stubServiceConfig(t *core.T, projectDir string, mgr buildservice.Manager) {
	t.Helper()
	originalGetwd := serviceGetwd
	originalResolve := resolveServiceCfg
	originalManager := serviceManager
	t.Cleanup(func() {
		serviceGetwd = originalGetwd
		resolveServiceCfg = originalResolve
		serviceManager = originalManager
	})
	serviceGetwd = func() core.Result { return core.Ok(projectDir) }
	resolveServiceCfg = func(dir string) core.Result {
		return core.Ok(buildservice.Config{Name: "core-build", ProjectDir: dir})
	}
	if mgr != nil {
		serviceManager = mgr
	}
}

// captureServiceStdout redirects cli output to a buffer for the test duration.
func captureServiceStdout(t *core.T) *core.Buffer {
	t.Helper()
	buf := core.NewBuffer()
	cli.SetStdout(buf)
	cli.SetStderr(buf)
	t.Cleanup(func() {
		cli.SetStdout(nil)
		cli.SetStderr(nil)
	})
	return buf
}

// --- AddServiceCommands: registers the command surface ---

func TestCmd_AddServiceCommands_Good(t *core.T) {
	c := core.New()

	result := AddServiceCommands(c)
	core.AssertTrue(t, result.OK)
	for _, path := range []string{
		"service", "service/install", "service/start",
		"service/stop", "service/uninstall", "service/export", "service/run",
	} {
		core.AssertTrue(t, c.Command(path).OK, "expected command "+path+" registered")
	}
	// `service/run` is a hidden internal command.
	runCmd := c.Command("service/run").Value.(*core.Command)
	core.AssertTrue(t, runCmd.Hidden)
}

func TestCmd_AddServiceCommands_Bad(t *core.T) {
	// Failure at the very first step: `service` is already an executable command,
	// so AddServiceCommands returns immediately and registers nothing further.
	c := core.New()
	core.AssertTrue(t, c.Command("service", core.Command{Action: noopServiceAction}).OK)

	result := AddServiceCommands(c)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "service")
	core.AssertContains(t, result.Error(), "already registered")
	core.AssertFalse(t, c.Command("service/install").OK)
}

func TestCmd_AddServiceCommands_Ugly(t *core.T) {
	// Edge case: a later subcommand path is pre-occupied. The `service` root and
	// the steps before the clash register, but the clashing step aborts the rest.
	c := core.New()
	core.AssertTrue(t, c.Command("service/stop", core.Command{Action: noopServiceAction}).OK)

	result := AddServiceCommands(c)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "service/stop")
	// install/start were registered before the stop clash; uninstall was not.
	core.AssertTrue(t, c.Command("service/install").OK)
	core.AssertTrue(t, c.Command("service/start").OK)
	core.AssertFalse(t, c.Command("service/uninstall").OK)
}

// TestCmd_AddServiceCommands_EveryStepCanFail asserts that a clash on any single
// registration step aborts AddServiceCommands, exercising each early-return
// branch in turn.
func TestCmd_AddServiceCommands_EveryStepCanFail(t *core.T) {
	for _, path := range []string{
		"service", "service/install", "service/start",
		"service/stop", "service/uninstall", "service/export", "service/run",
	} {
		c := core.New()
		core.AssertTrue(t, c.Command(path, core.Command{Action: noopServiceAction}).OK)
		result := AddServiceCommands(c)
		core.AssertFalse(t, result.OK, "clash on "+path+" should abort registration")
		core.AssertContains(t, result.Error(), path)
	}
}

// TestCmd_AddServiceCommands_ActionsWired registers the commands and invokes
// each command's Action so the action closures (which dispatch to the run*
// helpers) are exercised. The package seams are stubbed so no real OS service
// manager or daemon is touched.
func TestCmd_AddServiceCommands_ActionsWired(t *core.T) {
	projectDir := t.TempDir()
	stubServiceConfig(t, projectDir, stubManager{})
	originalExport := exportService
	originalRun := runDaemon
	t.Cleanup(func() {
		exportService = originalExport
		runDaemon = originalRun
	})
	exportService = func(buildservice.Config, string) core.Result {
		return core.Ok(buildservice.ExportedConfig{Content: "rendered\n"})
	}
	daemonRan := false
	runDaemon = func(context.Context, buildservice.Config) core.Result {
		daemonRan = true
		return core.Ok(nil)
	}
	captureServiceStdout(t)

	c := core.New()
	core.AssertTrue(t, AddServiceCommands(c).OK)

	// The bare `service` action is a usage error directing to a subcommand.
	rootResult := c.Command("service").Value.(*core.Command).Run(core.NewOptions())
	core.AssertFalse(t, rootResult.OK)
	core.AssertContains(t, rootResult.Error(), "subcommand")

	// Each managed subcommand action resolves config and dispatches successfully.
	for _, path := range []string{
		"service/install", "service/start", "service/stop",
		"service/uninstall", "service/export", "service/run",
	} {
		cmd := c.Command(path).Value.(*core.Command)
		result := cmd.Run(core.NewOptions())
		core.AssertTrue(t, result.OK, "action "+path+" should succeed")
	}
	core.AssertTrue(t, daemonRan)
}

// --- requestFromOptions: decode CLI options into a service request ---

func TestCmd_requestFromOptions_Good(t *core.T) {
	req := requestFromOptions(core.NewOptions(
		core.Option{Key: "name", Value: "myapp"},
		core.Option{Key: "display-name", Value: "My App"},
		core.Option{Key: "description", Value: "a service"},
		core.Option{Key: "project-dir", Value: "/srv/app"},
		core.Option{Key: "addr", Value: ":7300"},
		core.Option{Key: "health-addr", Value: ":7301"},
		core.Option{Key: "pid-file", Value: "run/app.pid"},
		core.Option{Key: "watch-paths", Value: "src,docs"},
		core.Option{Key: "watch-interval", Value: "5s"},
		core.Option{Key: "auto-rebuild", Value: false},
	))

	core.AssertEqual(t, "myapp", req.Name)
	core.AssertEqual(t, "My App", req.DisplayName)
	core.AssertEqual(t, "a service", req.Description)
	core.AssertEqual(t, "/srv/app", req.ProjectDir)
	core.AssertEqual(t, ":7300", req.APIAddr)
	core.AssertEqual(t, ":7301", req.HealthAddr)
	core.AssertEqual(t, "run/app.pid", req.PIDFile)
	core.AssertEqual(t, "src,docs", req.WatchPaths)
	core.AssertEqual(t, "5s", req.WatchInterval)
	// An explicit auto-rebuild=false must be captured and marked as set.
	core.AssertFalse(t, req.AutoRebuild)
	core.AssertTrue(t, req.AutoRebuildSet)
}

func TestCmd_requestFromOptions_Bad(t *core.T) {
	// Empty options: every string field is blank and auto-rebuild is unset.
	// The default value surfaces as true, but AutoRebuildSet stays false so the
	// override layer knows not to apply it.
	req := requestFromOptions(core.NewOptions())

	core.AssertEqual(t, "", req.Name)
	core.AssertEqual(t, "", req.ProjectDir)
	core.AssertEqual(t, "", req.APIAddr)
	core.AssertEqual(t, "", req.WatchPaths)
	core.AssertTrue(t, req.AutoRebuild)
	core.AssertFalse(t, req.AutoRebuildSet)
}

func TestCmd_requestFromOptions_Ugly(t *core.T) {
	// Edge case: the snake_case aliases and an explicit auto-rebuild=true must
	// resolve identically to their hyphenated forms.
	req := requestFromOptions(core.NewOptions(
		core.Option{Key: "api_addr", Value: ":9000"},
		core.Option{Key: "health_addr", Value: ":9001"},
		core.Option{Key: "pid_file", Value: "/var/run/app.pid"},
		core.Option{Key: "watch_paths", Value: "internal"},
		core.Option{Key: "schedule_interval", Value: "30s"},
		core.Option{Key: "auto_rebuild", Value: true},
	))

	core.AssertEqual(t, ":9000", req.APIAddr)
	core.AssertEqual(t, ":9001", req.HealthAddr)
	core.AssertEqual(t, "/var/run/app.pid", req.PIDFile)
	core.AssertEqual(t, "internal", req.WatchPaths)
	core.AssertEqual(t, "30s", req.ScheduleInterval)
	core.AssertTrue(t, req.AutoRebuild)
	core.AssertTrue(t, req.AutoRebuildSet)
}

// --- runServiceStart ---

func TestCmd_runServiceStart_Good(t *core.T) {
	projectDir := t.TempDir()
	started := false
	stubServiceConfig(t, projectDir, stubManager{
		start: func(cfg buildservice.Config) core.Result {
			started = true
			core.AssertEqual(t, projectDir, cfg.ProjectDir)
			return core.Ok(nil)
		},
	})
	buf := captureServiceStdout(t)

	result := runServiceStart(serviceRequest{})
	core.AssertTrue(t, result.OK)
	core.AssertTrue(t, started)
	core.AssertContains(t, buf.String(), "Service started")
}

func TestCmd_runServiceStart_Bad(t *core.T) {
	projectDir := t.TempDir()
	stubServiceConfig(t, projectDir, stubManager{
		start: func(buildservice.Config) core.Result {
			return core.Fail(core.NewError("start-failed"))
		},
	})
	captureServiceStdout(t)

	result := runServiceStart(serviceRequest{})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "start-failed")
}

func TestCmd_runServiceStart_Ugly(t *core.T) {
	// Edge case: config resolution fails before the manager is ever consulted.
	projectDir := t.TempDir()
	startCalled := false
	stubServiceConfig(t, projectDir, stubManager{
		start: func(buildservice.Config) core.Result {
			startCalled = true
			return core.Ok(nil)
		},
	})
	resolveServiceCfg = func(string) core.Result { return core.Fail(core.NewError("resolve-failed")) }
	captureServiceStdout(t)

	result := runServiceStart(serviceRequest{})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "resolve-failed")
	core.AssertFalse(t, startCalled)
}

// --- runServiceStop ---

func TestCmd_runServiceStop_Good(t *core.T) {
	projectDir := t.TempDir()
	stopped := false
	stubServiceConfig(t, projectDir, stubManager{
		stop: func(cfg buildservice.Config) core.Result {
			stopped = true
			core.AssertEqual(t, projectDir, cfg.ProjectDir)
			return core.Ok(nil)
		},
	})
	buf := captureServiceStdout(t)

	result := runServiceStop(serviceRequest{})
	core.AssertTrue(t, result.OK)
	core.AssertTrue(t, stopped)
	core.AssertContains(t, buf.String(), "Service stopped")
}

func TestCmd_runServiceStop_Bad(t *core.T) {
	projectDir := t.TempDir()
	stubServiceConfig(t, projectDir, stubManager{
		stop: func(buildservice.Config) core.Result {
			return core.Fail(core.NewError("stop-failed"))
		},
	})
	captureServiceStdout(t)

	result := runServiceStop(serviceRequest{})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "stop-failed")
}

func TestCmd_runServiceStop_Ugly(t *core.T) {
	// Edge case: the working directory cannot be determined, so config loading
	// fails with a wrapped error before the stop call.
	projectDir := t.TempDir()
	stubServiceConfig(t, projectDir, stubManager{})
	serviceGetwd = func() core.Result { return core.Fail(core.NewError("no-cwd")) }
	captureServiceStdout(t)

	result := runServiceStop(serviceRequest{})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "failed to get working directory")
}

// --- runServiceUninstall ---

func TestCmd_runServiceUninstall_Good(t *core.T) {
	projectDir := t.TempDir()
	removed := false
	stubServiceConfig(t, projectDir, stubManager{
		remove: func(cfg buildservice.Config) core.Result {
			removed = true
			core.AssertEqual(t, projectDir, cfg.ProjectDir)
			return core.Ok(nil)
		},
	})
	buf := captureServiceStdout(t)

	result := runServiceUninstall(serviceRequest{})
	core.AssertTrue(t, result.OK)
	core.AssertTrue(t, removed)
	core.AssertContains(t, buf.String(), "Service uninstalled")
}

func TestCmd_runServiceUninstall_Bad(t *core.T) {
	projectDir := t.TempDir()
	stubServiceConfig(t, projectDir, stubManager{
		remove: func(buildservice.Config) core.Result {
			return core.Fail(core.NewError("uninstall-failed"))
		},
	})
	captureServiceStdout(t)

	result := runServiceUninstall(serviceRequest{})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "uninstall-failed")
}

func TestCmd_runServiceUninstall_Ugly(t *core.T) {
	// Edge case: a request override (custom name) flows through to the resolved
	// config and the uninstall still succeeds — the override layer is applied
	// before the manager call.
	projectDir := t.TempDir()
	var seenName string
	stubServiceConfig(t, projectDir, stubManager{
		remove: func(cfg buildservice.Config) core.Result {
			seenName = cfg.Name
			return core.Ok(nil)
		},
	})
	captureServiceStdout(t)

	result := runServiceUninstall(serviceRequest{Name: "renamed-svc"})
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, "renamed-svc", seenName)
}

// --- runServiceExport (additional branches beyond the existing file test) ---

func TestCmd_runServiceExport_Good(t *core.T) {
	// With no output path the rendered content is written to stdout verbatim.
	projectDir := t.TempDir()
	stubServiceConfig(t, projectDir, nil)
	originalExport := exportService
	t.Cleanup(func() { exportService = originalExport })
	exportService = func(cfg buildservice.Config, format string) core.Result {
		return core.Ok(buildservice.ExportedConfig{
			Format:  buildservice.NativeFormatSystemd,
			Content: "[Unit]\nDescription=Demo\n",
		})
	}
	buf := captureServiceStdout(t)

	result := runServiceExport(serviceRequest{})
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, "[Unit]\nDescription=Demo\n", buf.String())
}

func TestCmd_runServiceExport_Bad(t *core.T) {
	// A failure from the export renderer is bubbled unchanged.
	projectDir := t.TempDir()
	stubServiceConfig(t, projectDir, nil)
	originalExport := exportService
	t.Cleanup(func() { exportService = originalExport })
	exportService = func(buildservice.Config, string) core.Result {
		return core.Fail(core.NewError("unsupported-format"))
	}
	captureServiceStdout(t)

	result := runServiceExport(serviceRequest{Format: "nonsense"})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "unsupported-format")
}

func TestCmd_runServiceExport_Ugly(t *core.T) {
	// Edge case: the output directory cannot be created because a path component
	// is an existing regular file, so MkdirAll fails before any write.
	projectDir := t.TempDir()
	stubServiceConfig(t, projectDir, nil)
	originalExport := exportService
	t.Cleanup(func() { exportService = originalExport })
	exportService = func(buildservice.Config, string) core.Result {
		return core.Ok(buildservice.ExportedConfig{Content: "data\n"})
	}
	// Create a file that blocks directory creation underneath it.
	blocker := ax.Join(projectDir, "blocker")
	requireServiceCmdOK(t, ax.WriteFile(blocker, []byte("file"), 0o644))
	captureServiceStdout(t)

	result := runServiceExport(serviceRequest{Output: core.PathJoin("blocker", "sub", "out.service")})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "MkdirAll")
}

// TestCmd_runServiceExport_AbsoluteOutput covers the absolute-path branch: the
// output path is used as-is (not joined to the project dir) and the file is
// written under a freshly created parent directory.
func TestCmd_runServiceExport_AbsoluteOutput(t *core.T) {
	projectDir := t.TempDir()
	stubServiceConfig(t, projectDir, nil)
	originalExport := exportService
	t.Cleanup(func() { exportService = originalExport })
	exportService = func(buildservice.Config, string) core.Result {
		return core.Ok(buildservice.ExportedConfig{Content: "ABSOLUTE\n"})
	}
	buf := captureServiceStdout(t)

	outputPath := ax.Join(t.TempDir(), "nested", "core-build.service")
	result := runServiceExport(serviceRequest{Output: outputPath})
	core.AssertTrue(t, result.OK)
	content := requireServiceCmdBytes(t, ax.ReadFile(outputPath))
	core.AssertEqual(t, "ABSOLUTE\n", string(content))
	// The success line names the written path.
	core.AssertContains(t, buf.String(), outputPath)
}

// TestCmd_runServiceInstall_ConfigLoadErrorBad covers the config-load failure
// branch of install: when resolution fails the manager is never invoked.
func TestCmd_runServiceInstall_ConfigLoadErrorBad(t *core.T) {
	projectDir := t.TempDir()
	installCalled := false
	stubServiceConfig(t, projectDir, stubManager{
		install: func(buildservice.Config) core.Result {
			installCalled = true
			return core.Ok(nil)
		},
	})
	resolveServiceCfg = func(string) core.Result { return core.Fail(core.NewError("no-build-config")) }
	captureServiceStdout(t)

	result := runServiceInstall(serviceRequest{})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "no-build-config")
	core.AssertFalse(t, installCalled)
}

// TestCmd_runServiceUninstall_ConfigLoadErrorBad covers the config-load failure
// branch of uninstall.
func TestCmd_runServiceUninstall_ConfigLoadErrorBad(t *core.T) {
	projectDir := t.TempDir()
	removeCalled := false
	stubServiceConfig(t, projectDir, stubManager{
		remove: func(buildservice.Config) core.Result {
			removeCalled = true
			return core.Ok(nil)
		},
	})
	resolveServiceCfg = func(string) core.Result { return core.Fail(core.NewError("no-build-config")) }
	captureServiceStdout(t)

	result := runServiceUninstall(serviceRequest{})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "no-build-config")
	core.AssertFalse(t, removeCalled)
}

// TestCmd_runServiceExport_ConfigLoadErrorBad covers the config-load failure
// branch of export: the renderer is never invoked.
func TestCmd_runServiceExport_ConfigLoadErrorBad(t *core.T) {
	projectDir := t.TempDir()
	stubServiceConfig(t, projectDir, nil)
	resolveServiceCfg = func(string) core.Result { return core.Fail(core.NewError("no-build-config")) }
	exportCalled := false
	originalExport := exportService
	t.Cleanup(func() { exportService = originalExport })
	exportService = func(buildservice.Config, string) core.Result {
		exportCalled = true
		return core.Ok(buildservice.ExportedConfig{})
	}
	captureServiceStdout(t)

	result := runServiceExport(serviceRequest{})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "no-build-config")
	core.AssertFalse(t, exportCalled)
}

// --- runServiceRun ---

func TestCmd_runServiceRun_Good(t *core.T) {
	projectDir := t.TempDir()
	stubServiceConfig(t, projectDir, nil)
	originalRun := runDaemon
	t.Cleanup(func() { runDaemon = originalRun })
	daemonCfg := buildservice.Config{}
	runDaemon = func(ctx context.Context, cfg buildservice.Config) core.Result {
		daemonCfg = cfg
		core.AssertNotNil(t, ctx)
		return core.Ok(nil)
	}

	result := runServiceRun(context.Background(), serviceRequest{})
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, projectDir, daemonCfg.ProjectDir)
}

func TestCmd_runServiceRun_Bad(t *core.T) {
	// The daemon's failure is returned to the caller unchanged.
	projectDir := t.TempDir()
	stubServiceConfig(t, projectDir, nil)
	originalRun := runDaemon
	t.Cleanup(func() { runDaemon = originalRun })
	runDaemon = func(context.Context, buildservice.Config) core.Result {
		return core.Fail(core.NewError("daemon-crashed"))
	}

	result := runServiceRun(context.Background(), serviceRequest{})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "daemon-crashed")
}

func TestCmd_runServiceRun_Ugly(t *core.T) {
	// Edge case: config loading fails, so the daemon is never started.
	projectDir := t.TempDir()
	stubServiceConfig(t, projectDir, nil)
	resolveServiceCfg = func(string) core.Result { return core.Fail(core.NewError("resolve-failed")) }
	daemonStarted := false
	originalRun := runDaemon
	t.Cleanup(func() { runDaemon = originalRun })
	runDaemon = func(context.Context, buildservice.Config) core.Result {
		daemonStarted = true
		return core.Ok(nil)
	}

	result := runServiceRun(context.Background(), serviceRequest{})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "resolve-failed")
	core.AssertFalse(t, daemonStarted)
}
