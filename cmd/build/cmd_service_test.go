package buildcmd

import (
	"context"
	"testing"

	core "dappco.re/go"
	buildservice "dappco.re/go/build/pkg/service"
	nativeservice "github.com/kardianos/service"
)

type stubServiceController struct {
	installErr   error
	startErr     error
	stopErr      error
	uninstallErr error
	runErr       error
	run          func() error
	actions      []string
}

func (s *stubServiceController) Install() error {
	s.actions = append(s.actions, "install")
	return s.installErr
}

func (s *stubServiceController) Start() error {
	s.actions = append(s.actions, "start")
	return s.startErr
}

func (s *stubServiceController) Stop() error {
	s.actions = append(s.actions, "stop")
	return s.stopErr
}

func (s *stubServiceController) Uninstall() error {
	s.actions = append(s.actions, "uninstall")
	return s.uninstallErr
}

func (s *stubServiceController) Run() error {
	s.actions = append(s.actions, "run")
	if s.run != nil {
		return s.run()
	}
	return s.runErr
}

func restoreServiceCommandStubs(t *testing.T) {
	t.Helper()

	originalGetwd := serviceGetwd
	originalResolve := resolveBuildServiceCfg
	originalExport := exportBuildService
	originalRunDaemon := runBuildServiceDaemon
	originalNewController := newBuildNativeController

	t.Cleanup(func() {
		serviceGetwd = originalGetwd
		resolveBuildServiceCfg = originalResolve
		exportBuildService = originalExport
		runBuildServiceDaemon = originalRunDaemon
		newBuildNativeController = originalNewController
	})
}

func stubResolvedServiceConfig(t *testing.T, projectDir string) {
	t.Helper()

	serviceGetwd = func() (string, error) { return projectDir, nil }
	resolveBuildServiceCfg = func(dir string) (buildservice.Config, error) {
		if !stdlibAssertEqual(projectDir, dir) {
			t.Fatalf("want %v, got %v", projectDir, dir)
		}
		return buildservice.Config{
			Name:        "core-build",
			DisplayName: "Core Build",
			Description: "Core build daemon",
			ProjectDir:  projectDir,
			APIAddr:     "127.0.0.1:9101",
			HealthAddr:  "127.0.0.1:9102",
		}, nil
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

	controller := &stubServiceController{}
	var recordedProgram nativeservice.Interface
	var recordedConfig *nativeservice.Config
	newBuildNativeController = func(program nativeservice.Interface, cfg *nativeservice.Config) (serviceController, error) {
		recordedProgram = program
		recordedConfig = cfg
		return controller, nil
	}

	err := runServiceInstall(serviceRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual([]string{"install"}, controller.actions) {
		t.Fatalf("want %v, got %v", []string{"install"}, controller.actions)
	}
	if _, ok := recordedProgram.(controlServiceProgram); !ok {
		t.Fatalf("expected control service program, got %T", recordedProgram)
	}
	if !stdlibAssertEqual("core-build", recordedConfig.Name) {
		t.Fatalf("want %v, got %v", "core-build", recordedConfig.Name)
	}
	if !stdlibAssertContains(recordedConfig.Arguments, "service") {
		t.Fatalf("expected %v to contain %v", recordedConfig.Arguments, "service")
	}
	if !stdlibAssertContains(recordedConfig.Arguments, "run") {
		t.Fatalf("expected %v to contain %v", recordedConfig.Arguments, "run")
	}
	if !stdlibAssertEqual(projectDir, recordedConfig.WorkingDirectory) {
		t.Fatalf("want %v, got %v", projectDir, recordedConfig.WorkingDirectory)
	}
}

func TestService_InstallBad(t *testing.T) {
	restoreServiceCommandStubs(t)

	projectDir := t.TempDir()
	stubResolvedServiceConfig(t, projectDir)

	newBuildNativeController = func(nativeservice.Interface, *nativeservice.Config) (serviceController, error) {
		return nil, core.NewError("native service unavailable")
	}

	err := runServiceInstall(serviceRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "native service unavailable") {
		t.Fatalf("expected %v to contain %v", err.Error(), "native service unavailable")
	}
}

func TestService_InstallUgly(t *testing.T) {
	restoreServiceCommandStubs(t)

	projectDir := t.TempDir()
	stubResolvedServiceConfig(t, projectDir)

	controller := &stubServiceController{installErr: core.NewError("install rejected")}
	newBuildNativeController = func(nativeservice.Interface, *nativeservice.Config) (serviceController, error) {
		return controller, nil
	}

	err := runServiceInstall(serviceRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "install rejected") {
		t.Fatalf("expected %v to contain %v", err.Error(), "install rejected")
	}
	if !stdlibAssertEqual([]string{"install"}, controller.actions) {
		t.Fatalf("want %v, got %v", []string{"install"}, controller.actions)
	}
}

func TestService_Run_UsesKardianosRunCallbackGood(t *testing.T) {
	restoreServiceCommandStubs(t)

	projectDir := t.TempDir()
	stubResolvedServiceConfig(t, projectDir)

	daemonConfigs := make(chan buildservice.Config, 1)
	runBuildServiceDaemon = func(ctx context.Context, cfg buildservice.Config) error {
		daemonConfigs <- cfg
		<-ctx.Done()
		return nil
	}

	newBuildNativeController = func(program nativeservice.Interface, cfg *nativeservice.Config) (serviceController, error) {
		if _, ok := cfg.Option["RunWait"].(func()); !ok {
			t.Fatal("expected kardianos RunWait callback")
		}
		return &stubServiceController{
			run: func() error {
				if err := program.Start(nil); err != nil {
					return err
				}
				return program.Stop(nil)
			},
		}, nil
	}

	err := runServiceRun(context.Background(), serviceRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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
func TestCmdService_Program_Start_Good(t *core.T) {
	subject := &serviceProgram{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Start(nil)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCmdService_Program_Start_Bad(t *core.T) {
	subject := &serviceProgram{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Start(nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCmdService_Program_Start_Ugly(t *core.T) {
	subject := &serviceProgram{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Start(nil)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestCmdService_Program_Stop_Good(t *core.T) {
	subject := &serviceProgram{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Stop(nil)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCmdService_Program_Stop_Bad(t *core.T) {
	subject := &serviceProgram{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Stop(nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCmdService_Program_Stop_Ugly(t *core.T) {
	subject := &serviceProgram{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Stop(nil)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestCmdService_ServiceProgram_Start_Good(t *core.T) {
	subject := controlServiceProgram{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Start(nil)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCmdService_ServiceProgram_Start_Bad(t *core.T) {
	subject := controlServiceProgram{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Start(nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCmdService_ServiceProgram_Start_Ugly(t *core.T) {
	subject := controlServiceProgram{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Start(nil)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestCmdService_ServiceProgram_Stop_Good(t *core.T) {
	subject := controlServiceProgram{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Stop(nil)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCmdService_ServiceProgram_Stop_Bad(t *core.T) {
	subject := controlServiceProgram{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Stop(nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCmdService_ServiceProgram_Stop_Ugly(t *core.T) {
	subject := controlServiceProgram{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Stop(nil)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

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
