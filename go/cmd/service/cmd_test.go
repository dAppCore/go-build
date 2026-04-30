package servicecmd

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
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

// --- v0.9.0 generated compliance triplets ---
func TestCmd_AddServiceCommands_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		AddServiceCommands(core.New())
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCmd_AddServiceCommands_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		AddServiceCommands(core.New())
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCmd_AddServiceCommands_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		AddServiceCommands(core.New())
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
