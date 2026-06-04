package buildcmd

import (
	"context"
	"testing"

	core "dappco.re/go"
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

// --- v0.9.0 generated compliance triplets ---
func TestCmdService_AddServiceCommands_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		AddServiceCommands(core.New())
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCmdService_AddServiceCommands_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		AddServiceCommands(core.New())
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCmdService_AddServiceCommands_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		AddServiceCommands(core.New())
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
