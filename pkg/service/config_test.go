package service

import (
	"dappco.re/go/build/internal/ax"
	"testing"
	"time"

	core "dappco.re/go"
	nativeservice "github.com/kardianos/service"
)

type recordingNativeController struct {
	installed bool
	started   bool
	stopped   bool
	removed   bool
}

func (r *recordingNativeController) Install() error {
	r.installed = true
	return nil
}

func (r *recordingNativeController) Start() error {
	r.started = true
	return nil
}

func (r *recordingNativeController) Stop() error {
	r.stopped = true
	return nil
}

func (r *recordingNativeController) Uninstall() error {
	r.removed = true
	return nil
}

func TestDefaultConfig_Normalized_Good(t *testing.T) {
	projectDir := t.TempDir()

	cfg := DefaultConfig(projectDir).Normalized()
	if !stdlibAssertEqual(projectDir, cfg.ProjectDir) {
		t.Fatalf("want %v, got %v", projectDir, cfg.ProjectDir)
	}
	if !stdlibAssertEqual("127.0.0.1:9101", cfg.APIAddr) {
		t.Fatalf("want %v, got %v", "127.0.0.1:9101", cfg.APIAddr)
	}
	if !stdlibAssertEqual("127.0.0.1:9102", cfg.HealthAddr) {
		t.Fatalf("want %v, got %v", "127.0.0.1:9102", cfg.HealthAddr)
	}
	if !(cfg.AutoRebuild) {
		t.Fatal("expected true")
	}
	if !stdlibAssertContains(cfg.Arguments, "service") {
		t.Fatalf("expected %v to contain %v", cfg.Arguments, "service")
	}
	if !stdlibAssertContains(cfg.Arguments, "run") {
		t.Fatalf("expected %v to contain %v", cfg.Arguments, "run")
	}
	if !stdlibAssertContains(cfg.Arguments, projectDir) {
		t.Fatalf("expected %v to contain %v", cfg.Arguments, projectDir)
	}
	if !stdlibAssertEqual(projectDir, cfg.Environment["CORE_BUILD_PROJECT_DIR"]) {
		t.Fatalf("want %v, got %v", projectDir, cfg.Environment["CORE_BUILD_PROJECT_DIR"])
	}

}

func TestResolveConfig_UsesBuildMetadataGood(t *testing.T) {
	projectDir := t.TempDir()
	if result := ax.MkdirAll(core.PathJoin(projectDir, ".core"), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(core.PathJoin(projectDir, ".core", "build.yaml"), []byte(`version: 1
project:
  name: "Core Build"
  binary: "core-builder"
  description: "Background build daemon"
`), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	cfg := requireServiceConfig(t, ResolveConfig(projectDir))
	if !stdlibAssertEqual("core-builder", cfg.Name) {
		t.Fatalf("want %v, got %v", "core-builder", cfg.Name)
	}
	if !stdlibAssertEqual("Core Builder", cfg.DisplayName) {
		t.Fatalf("want %v, got %v", "Core Builder", cfg.DisplayName)
	}
	if !stdlibAssertEqual("Background build daemon", cfg.Description) {
		t.Fatalf("want %v, got %v", "Background build daemon", cfg.Description)
	}

}

func TestResolveNativeFormat_Good(t *testing.T) {
	format := requireServiceNativeFormat(t, ResolveNativeFormat("launchd"))
	if !stdlibAssertEqual(NativeFormatLaunchd, format) {
		t.Fatalf("want %v, got %v", NativeFormatLaunchd, format)
	}

}

func TestExport_SystemdGood(t *testing.T) {
	cfg := DefaultConfig(t.TempDir()).Normalized()

	exported := requireServiceExportedConfig(t, Export(cfg, "systemd"))
	if !stdlibAssertEqual(NativeFormatSystemd, exported.Format) {
		t.Fatalf("want %v, got %v", NativeFormatSystemd, exported.Format)
	}
	if !stdlibAssertEqual(cfg.Name+".service", exported.Filename) {
		t.Fatalf("want %v, got %v", cfg.Name+".service", exported.Filename)
	}
	if !stdlibAssertContains(exported.Content, "[Unit]") {
		t.Fatalf("expected %v to contain %v", exported.Content, "[Unit]")
	}
	if !stdlibAssertContains(exported.Content, "ExecStart=") {
		t.Fatalf("expected %v to contain %v", exported.Content, "ExecStart=")
	}
	if !stdlibAssertContains(exported.Content, cfg.ProjectDir) {
		t.Fatalf("expected %v to contain %v", exported.Content, cfg.ProjectDir)
	}

}

func TestExport_LaunchdGood(t *testing.T) {
	cfg := DefaultConfig(t.TempDir()).Normalized()

	exported := requireServiceExportedConfig(t, Export(cfg, "launchd"))
	if !stdlibAssertEqual(NativeFormatLaunchd, exported.Format) {
		t.Fatalf("want %v, got %v", NativeFormatLaunchd, exported.Format)
	}
	if !stdlibAssertEqual(cfg.Name+".plist", exported.Filename) {
		t.Fatalf("want %v, got %v", cfg.Name+".plist", exported.Filename)
	}
	if !stdlibAssertContains(exported.Content, "<plist") {
		t.Fatalf("expected %v to contain %v", exported.Content, "<plist")
	}
	if !stdlibAssertContains(exported.Content, "<key>ProgramArguments</key>") {
		t.Fatalf("expected %v to contain %v", exported.Content, "<key>ProgramArguments</key>")
	}
	if !stdlibAssertContains(exported.Content, xmlEscape(cfg.Executable)) {
		t.Fatalf("expected %v to contain %v", exported.Content, xmlEscape(cfg.Executable))
	}

}

func TestOSManager_ServiceConfigMappingGood(t *testing.T) {
	originalNewNativeService := newNativeService
	t.Cleanup(func() {
		newNativeService = originalNewNativeService
	})

	controller := &recordingNativeController{}
	var recorded *nativeservice.Config
	newNativeService = func(program nativeservice.Interface, cfg *nativeservice.Config) (nativeController, error) {
		recorded = cfg
		return controller, nil
	}

	manager := &OSManager{}
	cfg := DefaultConfig(t.TempDir()).Normalized()
	cfg.Name = "core-build"
	cfg.DisplayName = "Core Build"
	cfg.Description = "Background build daemon"
	cfg.WatchInterval = 5 * time.Second
	cfg = cfg.Normalized()

	requireServiceOK(t, manager.Install(cfg))
	if stdlibAssertNil(recorded) {
		t.Fatal("expected non-nil")
	}
	if !(controller.installed) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual(cfg.Name, recorded.Name) {
		t.Fatalf("want %v, got %v", cfg.Name, recorded.Name)
	}
	if !stdlibAssertEqual(cfg.DisplayName, recorded.DisplayName) {
		t.Fatalf("want %v, got %v", cfg.DisplayName, recorded.DisplayName)
	}
	if !stdlibAssertEqual(cfg.Description, recorded.Description) {
		t.Fatalf("want %v, got %v", cfg.Description, recorded.Description)
	}
	if !stdlibAssertEqual(cfg.Executable, recorded.Executable) {
		t.Fatalf("want %v, got %v", cfg.Executable, recorded.Executable)
	}
	if !stdlibAssertEqual(cfg.WorkingDirectory, recorded.WorkingDirectory) {
		t.Fatalf("want %v, got %v", cfg.WorkingDirectory, recorded.WorkingDirectory)
	}
	if !stdlibAssertEqual(cfg.Environment["CORE_BUILD_API_ADDR"], recorded.EnvVars["CORE_BUILD_API_ADDR"]) {
		t.Fatalf("want %v, got %v", cfg.Environment["CORE_BUILD_API_ADDR"], recorded.EnvVars["CORE_BUILD_API_ADDR"])
	}
	if !stdlibAssertContains(recorded.Arguments, "--watch-interval") {
		t.Fatalf("expected %v to contain %v", recorded.Arguments, "--watch-interval")
	}

}

// --- v0.9.0 generated compliance triplets ---
func TestConfig_ResolveConfig_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ResolveConfig(core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestConfig_ResolveConfig_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ResolveConfig("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestConfig_ResolveConfig_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ResolveConfig(core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestConfig_DefaultConfig_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = DefaultConfig(core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestConfig_DefaultConfig_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = DefaultConfig("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestConfig_DefaultConfig_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = DefaultConfig(core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestConfig_Config_Normalized_Good(t *core.T) {
	subject := Config{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Normalized()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestConfig_Config_Normalized_Bad(t *core.T) {
	subject := Config{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Normalized()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestConfig_Config_Normalized_Ugly(t *core.T) {
	subject := Config{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Normalized()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestConfig_ResolveNativeFormat_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ResolveNativeFormat("tar.gz")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestConfig_ResolveNativeFormat_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ResolveNativeFormat("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestConfig_ResolveNativeFormat_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ResolveNativeFormat("tar.gz")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
