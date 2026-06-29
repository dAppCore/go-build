package buildcmd

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	buildservice "dappco.re/go/build/pkg/service"
)

type stubBuildServiceManager struct {
	install func(buildservice.Config) core.Result
	start   func(buildservice.Config) core.Result
	stop    func(buildservice.Config) core.Result
	remove  func(buildservice.Config) core.Result
}

func (s stubBuildServiceManager) Install(cfg buildservice.Config) core.Result {
	if s.install != nil {
		return s.install(cfg)
	}
	return core.Ok(nil)
}

func (s stubBuildServiceManager) Start(cfg buildservice.Config) core.Result {
	if s.start != nil {
		return s.start(cfg)
	}
	return core.Ok(nil)
}

func (s stubBuildServiceManager) Stop(cfg buildservice.Config) core.Result {
	if s.stop != nil {
		return s.stop(cfg)
	}
	return core.Ok(nil)
}

func (s stubBuildServiceManager) Uninstall(cfg buildservice.Config) core.Result {
	if s.remove != nil {
		return s.remove(cfg)
	}
	return core.Ok(nil)
}

func restoreServiceCommandStubs(t *testing.T) {
	t.Helper()

	originalGetwd := serviceGetwd
	originalResolve := resolveBuildServiceCfg
	originalExport := exportBuildService
	originalRunDaemon := runBuildServiceDaemon
	originalManager := buildServiceManager

	t.Cleanup(func() {
		serviceGetwd = originalGetwd
		resolveBuildServiceCfg = originalResolve
		exportBuildService = originalExport
		runBuildServiceDaemon = originalRunDaemon
		buildServiceManager = originalManager
	})
}

func stubResolvedServiceConfig(t *testing.T, projectDir string) {
	t.Helper()

	serviceGetwd = func() core.Result { return core.Ok(projectDir) }
	resolveBuildServiceCfg = func(dir string) core.Result {
		if !stdlibAssertEqual(projectDir, dir) {
			t.Fatalf("want %v, got %v", projectDir, dir)
		}
		return core.Ok(buildservice.Config{
			Name:        "core-build",
			DisplayName: "Core Build",
			Description: "Core build daemon",
			ProjectDir:  projectDir,
			APIAddr:     "127.0.0.1:9101",
			HealthAddr:  "127.0.0.1:9102",
		})
	}
}

func TestService_AddServiceCommands_RegistersSubcommandsGood(t *testing.T) {
	c := core.New()

	AddBuildCommands(c)
	for _, path := range []string{
		"service",
		"service/install",
		"service/start",
		"service/stop",
		"service/uninstall",
		"service/export",
	} {
		if !(c.Command(path).OK) {
			t.Fatalf("expected command to be registered: %s", path)
		}
	}

	command := c.Command("service/install").Value.(*core.Command)
	if !stdlibAssertEqual("cmd.service.install.short", command.Description) {
		t.Fatalf("want %v, got %v", "cmd.service.install.short", command.Description)
	}
}

func TestService_InstallGood(t *testing.T) {
	restoreServiceCommandStubs(t)

	projectDir := t.TempDir()
	stubResolvedServiceConfig(t, projectDir)

	called := false
	buildServiceManager = stubBuildServiceManager{
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

	requireBuildCmdOK(t, runServiceInstall(serviceRequest{}))
	if !called {
		t.Fatal("expected true")
	}
}

func TestService_InstallBad(t *testing.T) {
	restoreServiceCommandStubs(t)

	projectDir := t.TempDir()
	stubResolvedServiceConfig(t, projectDir)

	buildServiceManager = stubBuildServiceManager{
		install: func(buildservice.Config) core.Result {
			return core.Fail(core.NewError("native service unavailable"))
		},
	}

	message := requireBuildCmdError(t, runServiceInstall(serviceRequest{}))
	if !stdlibAssertContains(message, "native service unavailable") {
		t.Fatalf("expected %v to contain %v", message, "native service unavailable")
	}
}

func TestService_InstallUgly(t *testing.T) {
	restoreServiceCommandStubs(t)

	projectDir := t.TempDir()
	stubResolvedServiceConfig(t, projectDir)

	actions := make([]string, 0, 1)
	buildServiceManager = stubBuildServiceManager{
		install: func(buildservice.Config) core.Result {
			actions = append(actions, "install")
			return core.Fail(core.NewError("install rejected"))
		},
	}

	message := requireBuildCmdError(t, runServiceInstall(serviceRequest{}))
	if !stdlibAssertContains(message, "install rejected") {
		t.Fatalf("expected %v to contain %v", message, "install rejected")
	}
	if !stdlibAssertEqual([]string{"install"}, actions) {
		t.Fatalf("want %v, got %v", []string{"install"}, actions)
	}
}

func TestService_Run_InvokesDaemonGood(t *testing.T) {
	restoreServiceCommandStubs(t)

	projectDir := t.TempDir()
	stubResolvedServiceConfig(t, projectDir)

	daemonConfigs := make(chan buildservice.Config, 1)
	runBuildServiceDaemon = func(ctx context.Context, cfg buildservice.Config) core.Result {
		daemonConfigs <- cfg
		return core.Ok(nil)
	}

	requireBuildCmdOK(t, runServiceRun(context.Background(), serviceRequest{}))
	select {
	case cfg := <-daemonConfigs:
		if !stdlibAssertEqual(projectDir, cfg.ProjectDir) {
			t.Fatalf("want %v, got %v", projectDir, cfg.ProjectDir)
		}
	default:
		t.Fatal("expected daemon to be called")
	}
}

// noopBuildServiceAction is a placeholder action used to pre-occupy command
// paths so AddServiceCommands' partial-failure branches can be observed.
func noopBuildServiceAction(core.Options) core.Result { return core.Ok(nil) }

func TestCmdService_AddServiceCommands_Good(t *core.T) {
	c := core.New()

	result := AddServiceCommands(c)
	core.AssertTrue(t, result.OK)
	for _, path := range []string{
		"service", "service/install", "service/start",
		"service/stop", "service/uninstall", "service/export", "service/run",
	} {
		core.AssertTrue(t, c.Command(path).OK, "expected command "+path+" registered")
	}
	core.AssertTrue(t, c.Command("service/run").Value.(*core.Command).Hidden)
}

func TestCmdService_AddServiceCommands_Bad(t *core.T) {
	// First-step clash: `service` already executable -> registration aborts.
	c := core.New()
	core.AssertTrue(t, c.Command("service", core.Command{Action: noopBuildServiceAction}).OK)

	result := AddServiceCommands(c)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "already registered")
	core.AssertFalse(t, c.Command("service/install").OK)
}

func TestCmdService_AddServiceCommands_Ugly(t *core.T) {
	// Edge case: a clash on any single later step aborts the whole registration.
	for _, path := range []string{"service/install", "service/start", "service/export", "service/run"} {
		c := core.New()
		core.AssertTrue(t, c.Command(path, core.Command{Action: noopBuildServiceAction}).OK)
		result := AddServiceCommands(c)
		core.AssertFalse(t, result.OK, "clash on "+path+" should abort")
		core.AssertContains(t, result.Error(), path)
	}
}

// --- serviceRequestFromOptions ---

func TestCmdService_serviceRequestFromOptions_Good(t *core.T) {
	req := serviceRequestFromOptions(core.NewOptions(
		core.Option{Key: "name", Value: "myapp"},
		core.Option{Key: "addr", Value: ":7300"},
		core.Option{Key: "auto-rebuild", Value: false},
	))
	core.AssertEqual(t, "myapp", req.Name)
	core.AssertEqual(t, ":7300", req.APIAddr)
	core.AssertFalse(t, req.AutoRebuild)
	core.AssertTrue(t, req.AutoRebuildSet)
}

func TestCmdService_serviceRequestFromOptions_Bad(t *core.T) {
	// Empty options: blank fields, auto-rebuild defaults true but unset.
	req := serviceRequestFromOptions(core.NewOptions())
	core.AssertEqual(t, "", req.Name)
	core.AssertTrue(t, req.AutoRebuild)
	core.AssertFalse(t, req.AutoRebuildSet)
}

func TestCmdService_serviceRequestFromOptions_Ugly(t *core.T) {
	// Edge case: snake_case aliases resolve identically to hyphenated forms.
	req := serviceRequestFromOptions(core.NewOptions(
		core.Option{Key: "project_dir", Value: "/srv"},
		core.Option{Key: "health_addr", Value: ":9001"},
	))
	core.AssertEqual(t, "/srv", req.ProjectDir)
	core.AssertEqual(t, ":9001", req.HealthAddr)
}

// --- runServiceStart / Stop / Uninstall ---

func TestCmdService_runServiceStart_Good(t *core.T) {
	restoreServiceCommandStubs(t)
	projectDir := t.TempDir()
	stubResolvedServiceConfig(t, projectDir)
	started := false
	buildServiceManager = stubBuildServiceManager{
		start: func(cfg buildservice.Config) core.Result {
			started = true
			core.AssertEqual(t, projectDir, cfg.ProjectDir)
			return core.Ok(nil)
		},
	}
	buf := captureBuildStdout(t)

	result := runServiceStart(serviceRequest{})
	core.AssertTrue(t, result.OK)
	core.AssertTrue(t, started)
	core.AssertContains(t, buf.String(), "Service started")
}

func TestCmdService_runServiceStart_Bad(t *core.T) {
	restoreServiceCommandStubs(t)
	stubResolvedServiceConfig(t, t.TempDir())
	buildServiceManager = stubBuildServiceManager{
		start: func(buildservice.Config) core.Result { return core.Fail(core.NewError("start-failed")) },
	}
	captureBuildStdout(t)

	result := runServiceStart(serviceRequest{})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "start-failed")
}

func TestCmdService_runServiceStart_Ugly(t *core.T) {
	// Edge case: config load fails before the manager is consulted.
	restoreServiceCommandStubs(t)
	serviceGetwd = func() core.Result { return core.Ok(t.TempDir()) }
	resolveBuildServiceCfg = func(string) core.Result { return core.Fail(core.NewError("resolve-failed")) }
	startCalled := false
	buildServiceManager = stubBuildServiceManager{
		start: func(buildservice.Config) core.Result { startCalled = true; return core.Ok(nil) },
	}
	captureBuildStdout(t)

	result := runServiceStart(serviceRequest{})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "resolve-failed")
	core.AssertFalse(t, startCalled)
}

func TestCmdService_runServiceStop_Good(t *core.T) {
	restoreServiceCommandStubs(t)
	projectDir := t.TempDir()
	stubResolvedServiceConfig(t, projectDir)
	stopped := false
	buildServiceManager = stubBuildServiceManager{
		stop: func(buildservice.Config) core.Result { stopped = true; return core.Ok(nil) },
	}
	buf := captureBuildStdout(t)

	result := runServiceStop(serviceRequest{})
	core.AssertTrue(t, result.OK)
	core.AssertTrue(t, stopped)
	core.AssertContains(t, buf.String(), "Service stopped")
}

func TestCmdService_runServiceStop_Bad(t *core.T) {
	restoreServiceCommandStubs(t)
	stubResolvedServiceConfig(t, t.TempDir())
	buildServiceManager = stubBuildServiceManager{
		stop: func(buildservice.Config) core.Result { return core.Fail(core.NewError("stop-failed")) },
	}
	captureBuildStdout(t)

	result := runServiceStop(serviceRequest{})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "stop-failed")
}

func TestCmdService_runServiceStop_Ugly(t *core.T) {
	// Edge case: getwd failure surfaces a wrapped error before the stop call.
	restoreServiceCommandStubs(t)
	serviceGetwd = func() core.Result { return core.Fail(core.NewError("no-cwd")) }
	captureBuildStdout(t)

	result := runServiceStop(serviceRequest{})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "failed to get working directory")
}

func TestCmdService_runServiceUninstall_Good(t *core.T) {
	restoreServiceCommandStubs(t)
	projectDir := t.TempDir()
	stubResolvedServiceConfig(t, projectDir)
	removed := false
	buildServiceManager = stubBuildServiceManager{
		remove: func(buildservice.Config) core.Result { removed = true; return core.Ok(nil) },
	}
	buf := captureBuildStdout(t)

	result := runServiceUninstall(serviceRequest{})
	core.AssertTrue(t, result.OK)
	core.AssertTrue(t, removed)
	core.AssertContains(t, buf.String(), "Service uninstalled")
}

func TestCmdService_runServiceUninstall_Bad(t *core.T) {
	restoreServiceCommandStubs(t)
	stubResolvedServiceConfig(t, t.TempDir())
	buildServiceManager = stubBuildServiceManager{
		remove: func(buildservice.Config) core.Result { return core.Fail(core.NewError("uninstall-failed")) },
	}
	captureBuildStdout(t)

	result := runServiceUninstall(serviceRequest{})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "uninstall-failed")
}

func TestCmdService_runServiceUninstall_Ugly(t *core.T) {
	// Edge case: config-load failure short-circuits before the manager call.
	restoreServiceCommandStubs(t)
	serviceGetwd = func() core.Result { return core.Ok(t.TempDir()) }
	resolveBuildServiceCfg = func(string) core.Result { return core.Fail(core.NewError("no-config")) }
	removeCalled := false
	buildServiceManager = stubBuildServiceManager{
		remove: func(buildservice.Config) core.Result { removeCalled = true; return core.Ok(nil) },
	}
	captureBuildStdout(t)

	result := runServiceUninstall(serviceRequest{})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "no-config")
	core.AssertFalse(t, removeCalled)
}

// --- runServiceExport ---

func TestCmdService_runServiceExport_Good(t *core.T) {
	// No output path -> rendered content is written to stdout verbatim.
	restoreServiceCommandStubs(t)
	stubResolvedServiceConfig(t, t.TempDir())
	exportBuildService = func(buildservice.Config, string) core.Result {
		return core.Ok(buildservice.ExportedConfig{Content: "[Unit]\nDescription=Demo\n"})
	}
	buf := captureBuildStdout(t)

	result := runServiceExport(serviceRequest{})
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, "[Unit]\nDescription=Demo\n", buf.String())
}

func TestCmdService_runServiceExport_Bad(t *core.T) {
	// Export renderer failure is bubbled.
	restoreServiceCommandStubs(t)
	stubResolvedServiceConfig(t, t.TempDir())
	exportBuildService = func(buildservice.Config, string) core.Result {
		return core.Fail(core.NewError("unsupported-format"))
	}
	captureBuildStdout(t)

	result := runServiceExport(serviceRequest{Format: "nonsense"})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "unsupported-format")
}

func TestCmdService_runServiceExport_Ugly(t *core.T) {
	// Edge case: an output dir that cannot be created (a path component is a
	// regular file) fails at MkdirAll before any write.
	restoreServiceCommandStubs(t)
	projectDir := t.TempDir()
	stubResolvedServiceConfig(t, projectDir)
	exportBuildService = func(buildservice.Config, string) core.Result {
		return core.Ok(buildservice.ExportedConfig{Content: "data\n"})
	}
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(projectDir, "blocker"), []byte("file"), 0o644))
	captureBuildStdout(t)

	result := runServiceExport(serviceRequest{Output: core.PathJoin("blocker", "sub", "svc.service")})
	core.AssertFalse(t, result.OK)
}

// TestCmdService_runServiceExport_WritesFile covers the file-output success path.
func TestCmdService_runServiceExport_WritesFile(t *core.T) {
	restoreServiceCommandStubs(t)
	projectDir := t.TempDir()
	stubResolvedServiceConfig(t, projectDir)
	exportBuildService = func(buildservice.Config, string) core.Result {
		return core.Ok(buildservice.ExportedConfig{Content: "WRITTEN\n"})
	}
	buf := captureBuildStdout(t)

	outputPath := ax.Join(t.TempDir(), "nested", "svc.service")
	result := runServiceExport(serviceRequest{Output: outputPath})
	core.AssertTrue(t, result.OK)
	content := requireBuildCmdBytes(t, ax.ReadFile(outputPath))
	core.AssertEqual(t, "WRITTEN\n", string(content))
	core.AssertContains(t, buf.String(), outputPath)
}

// --- runServiceRun error path ---

func TestCmdService_runServiceRun_Bad(t *core.T) {
	// Daemon failure is returned to the caller.
	restoreServiceCommandStubs(t)
	stubResolvedServiceConfig(t, t.TempDir())
	runBuildServiceDaemon = func(context.Context, buildservice.Config) core.Result {
		return core.Fail(core.NewError("daemon-crashed"))
	}

	result := runServiceRun(context.Background(), serviceRequest{})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "daemon-crashed")
}
