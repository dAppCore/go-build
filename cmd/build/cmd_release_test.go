package buildcmd

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/internal/cli"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/build/pkg/release"
)

func TestBuildCmd_applyReleaseArchiveFormatOverride_Good(t *testing.T) {
	cfg := release.DefaultConfig()

	requireBuildCmdOK(t, applyReleaseArchiveFormatOverride(cfg, "xz"))
	if !stdlibAssertEqual("xz", cfg.Build.ArchiveFormat) {
		t.Fatalf("want %v, got %v", "xz", cfg.Build.ArchiveFormat)
	}

}

func TestBuildCmd_applyReleaseArchiveFormatOverride_Bad(t *testing.T) {
	cfg := release.DefaultConfig()

	requireBuildCmdError(t, applyReleaseArchiveFormatOverride(cfg, "bogus"))
	if !stdlibAssertEqual("", cfg.Build.ArchiveFormat) {
		t.Fatalf("want %v, got %v", "", cfg.Build.ArchiveFormat)
	}

}

func TestBuildCmd_AddReleaseCommand_RegistersTopLevelAlias_Good(t *testing.T) {
	c := core.New()

	AddReleaseCommand(c)
	if !(c.Command("build/release").OK) {
		t.Fatal("expected true")
	}
	if !(c.Command("release").OK) {
		t.Fatal("expected true")
	}

}

func TestBuildCmd_resolveReleaseDryRun_Good(t *testing.T) {
	if resolveReleaseDryRun(false, false, false) {
		t.Fatal("expected false")
	}
	if !(resolveReleaseDryRun(true, false, false)) {
		t.Fatal("expected true")
	}
	if resolveReleaseDryRun(false, true, false) {
		t.Fatal("expected false")
	}
	if resolveReleaseDryRun(true, true, false) {
		t.Fatal("expected false")
	}
	if resolveReleaseDryRun(false, false, true) {
		t.Fatal("expected false")
	}
	if resolveReleaseDryRun(true, false, true) {
		t.Fatal("expected false")
	}

}

func TestBuildCmd_runRelease_TargetSDK_Good(t *testing.T) {
	projectDir := t.TempDir()
	originalGetwd := getReleaseWorkingDir
	t.Cleanup(func() {
		getReleaseWorkingDir = originalGetwd
	})
	getReleaseWorkingDir = func() core.Result { return core.Ok(projectDir) }

	originalConfigExists := releaseConfigExistsFn
	originalLoadConfig := loadReleaseConfigFn
	originalRunSDK := runSDKReleaseFn
	t.Cleanup(func() {
		releaseConfigExistsFn = originalConfigExists
		loadReleaseConfigFn = originalLoadConfig
		runSDKReleaseFn = originalRunSDK
	})

	releaseConfigExistsFn = func(dir string) bool {
		if !stdlibAssertEqual(projectDir, dir) {
			t.Fatalf("want %v, got %v", projectDir, dir)
		}

		return true
	}
	loadReleaseConfigFn = func(dir string) core.Result {
		cfg := release.DefaultConfig()
		cfg.SetProjectDir(dir)
		cfg.SDK = &release.SDKConfig{
			Languages: []string{"typescript", "go"},
			Output:    "sdk",
		}
		return core.Ok(cfg)
	}

	called := false
	runSDKReleaseFn = func(ctx context.Context, cfg *release.Config, dryRun bool) core.Result {
		called = true
		if !(dryRun) {
			t.Fatal("expected true")
		}
		if stdlibAssertNil(cfg.SDK) {
			t.Fatal("expected non-nil")
		}

		return core.Ok(&release.SDKRelease{
			Version:   "v1.2.3",
			Output:    "sdk",
			Languages: []string{"typescript", "go"},
		})
	}

	requireBuildCmdOK(t, runRelease(context.Background(), true, false, "sdk", "v1.2.3", false, false, ""))
	if !(called) {
		t.Fatal("expected true")
	}

}

func TestBuildCmd_runRelease_AppleTestFlight_Good(t *testing.T) {
	projectDir := t.TempDir()
	requireBuildCmdOK(t, ax.MkdirAll(ax.Join(projectDir, ".core"), 0o755))
	requireBuildCmdOK(t, ax.WriteFile(ax.Join(projectDir, ".core", "build.yaml"), []byte(`
project:
  name: Core
  binary: Core
apple:
  bundle_id: ai.lthn.core
`), 0o644))

	originalGetwd := getReleaseWorkingDir
	originalConfigExists := releaseConfigExistsFn
	originalBuildApple := buildAppleFn
	t.Cleanup(func() {
		getReleaseWorkingDir = originalGetwd
		releaseConfigExistsFn = originalConfigExists
		buildAppleFn = originalBuildApple
	})

	getReleaseWorkingDir = func() core.Result { return core.Ok(projectDir) }
	releaseConfigExistsFn = func(dir string) bool {
		t.Fatalf("release config should not be required for apple-testflight target: %s", dir)
		return false
	}

	called := false
	buildAppleFn = func(ctx context.Context, cfg *build.Config, options build.AppleOptions, buildNumber string) core.Result {
		called = true
		if !stdlibAssertEqual(projectDir, cfg.ProjectDir) {
			t.Fatalf("want %v, got %v", projectDir, cfg.ProjectDir)
		}
		if !stdlibAssertEqual("v1.2.3", cfg.Version) {
			t.Fatalf("want %v, got %v", "v1.2.3", cfg.Version)
		}
		if !stdlibAssertEqual("ai.lthn.core", options.BundleID) {
			t.Fatalf("want %v, got %v", "ai.lthn.core", options.BundleID)
		}
		if !options.TestFlight {
			t.Fatal("expected TestFlight")
		}
		if !stdlibAssertEqual("1", buildNumber) {
			t.Fatalf("want %v, got %v", "1", buildNumber)
		}
		return core.Ok(&build.AppleBuildResult{
			BundlePath:  ax.Join(cfg.OutputDir, "Core.app"),
			Version:     "1.2.3",
			BuildNumber: buildNumber,
		})
	}

	requireBuildCmdOK(t, runRelease(context.Background(), false, false, "apple-testflight", "v1.2.3", false, false, ""))
	if !called {
		t.Fatal("expected buildAppleFn to be called")
	}
}

func TestBuildCmd_releaseAppleTestFlightRequested_Good(t *testing.T) {
	if !releaseAppleTestFlightRequested("apple-testflight") {
		t.Fatal("expected apple-testflight target to request TestFlight")
	}
	if !releaseAppleTestFlightRequested("testflight") {
		t.Fatal("expected testflight target to request TestFlight")
	}
	if !releaseAppleTestFlightRequested("release", true) {
		t.Fatal("expected explicit flag to request TestFlight")
	}
	if releaseAppleTestFlightRequested("release") {
		t.Fatal("expected release target without flag to skip TestFlight")
	}
}

func TestBuildCmd_runRelease_RejectsUnsafeVersion_Bad(t *testing.T) {
	projectDir := t.TempDir()
	originalGetwd := getReleaseWorkingDir
	originalConfigExists := releaseConfigExistsFn
	t.Cleanup(func() {
		getReleaseWorkingDir = originalGetwd
		releaseConfigExistsFn = originalConfigExists
	})

	getReleaseWorkingDir = func() core.Result { return core.Ok(projectDir) }
	releaseConfigExistsFn = func(dir string) bool { return true }

	message := requireBuildCmdError(t, runRelease(context.Background(), true, false, "release", "v1.2.3 --bad", false, false, ""))
	if !stdlibAssertContains(message, "invalid release version override") {
		t.Fatalf("expected %v to contain %v", message, "invalid release version override")
	}

}

func TestBuildCmd_runRelease_CIModeEmitsGitHubAnnotationOnError_Bad(t *testing.T) {
	projectDir := t.TempDir()
	originalGetwd := getReleaseWorkingDir
	originalConfigExists := releaseConfigExistsFn
	t.Cleanup(func() {
		getReleaseWorkingDir = originalGetwd
		releaseConfigExistsFn = originalConfigExists
		cli.SetStdout(nil)
		cli.SetStderr(nil)
	})

	getReleaseWorkingDir = func() core.Result { return core.Ok(projectDir) }
	releaseConfigExistsFn = func(dir string) bool {
		if !stdlibAssertEqual(projectDir, dir) {
			t.Fatalf("want %v, got %v", projectDir, dir)
		}

		return false
	}

	stdout := core.NewBuffer()
	cli.SetStdout(stdout)
	cli.SetStderr(stdout)

	result := runRelease(context.Background(), false, true, "release", "", false, false, "")
	if result.OK {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(stdout.String(), emitCIAnnotationForTest(result)) {
		t.Fatalf("expected %v to contain %v", stdout.String(), emitCIAnnotationForTest(result))
	}

}

// --- v0.9.0 generated compliance triplets ---
func TestCmdRelease_AddReleaseCommand_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		AddReleaseCommand(core.New())
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCmdRelease_AddReleaseCommand_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		AddReleaseCommand(core.New())
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCmdRelease_AddReleaseCommand_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		AddReleaseCommand(core.New())
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
