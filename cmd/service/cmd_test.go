package servicecmd

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	buildservice "dappco.re/go/build/pkg/service"
)

type stubManager struct {
	install func(buildservice.Config) error
	start   func(buildservice.Config) error
	stop    func(buildservice.Config) error
	remove  func(buildservice.Config) error
}

func (s stubManager) Install(cfg buildservice.Config) error {
	if s.install != nil {
		return s.install(cfg)
	}
	return nil
}

func (s stubManager) Start(cfg buildservice.Config) error {
	if s.start != nil {
		return s.start(cfg)
	}
	return nil
}

func (s stubManager) Stop(cfg buildservice.Config) error {
	if s.stop != nil {
		return s.stop(cfg)
	}
	return nil
}

func (s stubManager) Uninstall(cfg buildservice.Config) error {
	if s.remove != nil {
		return s.remove(cfg)
	}
	return nil
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

	serviceGetwd = func() (string, error) { return projectDir, nil }
	resolveServiceCfg = func(projectDir string) (buildservice.Config, error) {
		return buildservice.Config{
			Name:       "core-build",
			ProjectDir: projectDir,
			APIAddr:    "127.0.0.1:9101",
			HealthAddr: "127.0.0.1:9102",
		}, nil
	}

	called := false
	serviceManager = stubManager{
		install: func(cfg buildservice.Config) error {
			called = true
			if !stdlibAssertEqual(projectDir, cfg.ProjectDir) {
				t.Fatalf("want %v, got %v", projectDir, cfg.ProjectDir)
			}
			if !stdlibAssertEqual("core-build", cfg.Name) {
				t.Fatalf("want %v, got %v", "core-build", cfg.Name)
			}

			return nil
		},
	}

	err := runServiceInstall(serviceRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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

	serviceGetwd = func() (string, error) { return projectDir, nil }
	resolveServiceCfg = func(projectDir string) (buildservice.Config, error) {
		return buildservice.Config{Name: "core-build", ProjectDir: projectDir}, nil
	}
	exportService = func(cfg buildservice.Config, format string) (buildservice.ExportedConfig, error) {
		return buildservice.ExportedConfig{
			Format:   buildservice.NativeFormatSystemd,
			Filename: "core-build.service",
			Content:  "[Unit]\nDescription=Core Build\n",
		}, nil
	}

	outputPath := core.PathJoin("dist", "core-build.service")
	err := runServiceExport(serviceRequest{Output: outputPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, readErr := ax.ReadFile(core.PathJoin(projectDir, outputPath))
	if readErr != nil {
		t.Fatalf("unexpected error: %v", readErr)
	}
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

	serviceGetwd = func() (string, error) { return projectDir, nil }
	resolveServiceCfg = func(projectDir string) (buildservice.Config, error) {
		return buildservice.Config{Name: "core-build", ProjectDir: projectDir}, nil
	}

	called := false
	runDaemon = func(ctx context.Context, cfg buildservice.Config) error {
		called = true
		if !stdlibAssertEqual(projectDir, cfg.ProjectDir) {
			t.Fatalf("want %v, got %v", projectDir, cfg.ProjectDir)
		}

		return nil
	}

	err := runServiceRun(context.Background(), serviceRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !(called) {
		t.Fatal("expected true")
	}

}

func TestApplyServiceOverrides_BadDuration(t *testing.T) {
	cfg := buildservice.Config{ProjectDir: t.TempDir()}

	err := applyServiceOverrides(&cfg, serviceRequest{WatchInterval: "not-a-duration"})
	if err == nil {
		t.Fatal("expected error")
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

	serviceGetwd = func() (string, error) { return projectDir, nil }
	resolveServiceCfg = func(projectDir string) (buildservice.Config, error) {
		return buildservice.Config{Name: "core-build", ProjectDir: projectDir}, nil
	}
	serviceManager = stubManager{
		install: func(buildservice.Config) error {
			return core.NewError("boom")
		},
	}

	err := runServiceInstall(serviceRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "boom") {
		t.Fatalf("expected %v to contain %v", err.Error(), "boom")
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
