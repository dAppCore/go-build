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

// restoreReleaseStubs snapshots and restores the release command seams.
func restoreReleaseStubs(t *core.T) {
	t.Helper()
	g, ce, lc, fr, sr := getReleaseWorkingDir, releaseConfigExistsFn, loadReleaseConfigFn, runFullReleaseFn, runSDKReleaseFn
	t.Cleanup(func() {
		getReleaseWorkingDir = g
		releaseConfigExistsFn = ce
		loadReleaseConfigFn = lc
		runFullReleaseFn = fr
		runSDKReleaseFn = sr
	})
}

// --- AddReleaseCommand (meaningful) ---

func TestCmdRelease_AddReleaseCommand_Good(t *core.T) {
	c := core.New()
	result := AddReleaseCommand(c)
	core.AssertTrue(t, result.OK)
	core.AssertTrue(t, c.Command("build/release").OK)
	core.AssertTrue(t, c.Command("release").OK)
	core.AssertNotNil(t, c.Command("release").Value.(*core.Command).Action)
}

func TestCmdRelease_AddReleaseCommand_Bad(t *core.T) {
	// The top-level `release` alias is pre-occupied -> registration aborts at
	// the second step after `build/release` registers.
	c := core.New()
	core.AssertTrue(t, c.Command("release", core.Command{
		Action: func(core.Options) core.Result { return core.Ok(nil) },
	}).OK)
	result := AddReleaseCommand(c)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "already registered")
}

func TestCmdRelease_AddReleaseCommand_Ugly(t *core.T) {
	// Edge case: `build/release` pre-occupied -> the very first registration
	// step fails and the `release` alias is never reached.
	c := core.New()
	core.AssertTrue(t, c.Command("build/release", core.Command{
		Action: func(core.Options) core.Result { return core.Ok(nil) },
	}).OK)
	result := AddReleaseCommand(c)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "build/release")
	core.AssertFalse(t, c.Command("release").OK)
}

// TestCmdRelease_registerReleaseCommand_ActionWired drives the registered
// release action (and thus runRelease) via the command surface. The test
// working directory has no release config, so it fails fast with a config error.
func TestCmdRelease_registerReleaseCommand_ActionWired(t *core.T) {
	c := core.New()
	core.AssertTrue(t, AddReleaseCommand(c).OK)
	captureBuildStdout(t)

	result := c.Command("release").Value.(*core.Command).Run(core.NewOptions(
		core.Option{Key: "target", Value: "bogus-target"},
	))
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "config not found")
}

// --- runRelease: remaining branches ---

func TestCmdRelease_runRelease_FullReleaseGood(t *core.T) {
	restoreReleaseStubs(t)
	projectDir := t.TempDir()
	getReleaseWorkingDir = func() core.Result { return core.Ok(projectDir) }
	releaseConfigExistsFn = func(string) bool { return true }
	loadReleaseConfigFn = func(dir string) core.Result {
		cfg := release.DefaultConfig()
		cfg.SetProjectDir(dir)
		return core.Ok(cfg)
	}
	ran := false
	runFullReleaseFn = func(ctx context.Context, cfg *release.Config, dryRun bool) core.Result {
		ran = true
		core.AssertTrue(t, dryRun)
		return core.Ok(&release.Release{Version: "v2.0.0", Artifacts: nil})
	}
	buf := captureBuildStdout(t)

	result := runRelease(context.Background(), true, false, "release", "", false, false, "")
	core.AssertTrue(t, result.OK)
	core.AssertTrue(t, ran)
	out := buf.String()
	core.AssertContains(t, out, "Release completed")
	core.AssertContains(t, out, "v2.0.0")
}

func TestCmdRelease_runRelease_UnsupportedTargetBad(t *core.T) {
	restoreReleaseStubs(t)
	projectDir := t.TempDir()
	getReleaseWorkingDir = func() core.Result { return core.Ok(projectDir) }
	releaseConfigExistsFn = func(string) bool { return true }
	loadReleaseConfigFn = func(dir string) core.Result {
		cfg := release.DefaultConfig()
		cfg.SetProjectDir(dir)
		return core.Ok(cfg)
	}
	captureBuildStdout(t)

	result := runRelease(context.Background(), true, false, "bogus-target", "", false, false, "")
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "unsupported release target: bogus-target")
}

func TestCmdRelease_runRelease_LoadConfigErrorUgly(t *core.T) {
	// Edge case: config exists but fails to load -> wrapped load error.
	restoreReleaseStubs(t)
	getReleaseWorkingDir = func() core.Result { return core.Ok(t.TempDir()) }
	releaseConfigExistsFn = func(string) bool { return true }
	loadReleaseConfigFn = func(string) core.Result { return core.Fail(core.NewError("corrupt-config")) }
	captureBuildStdout(t)

	result := runRelease(context.Background(), true, false, "release", "", false, false, "")
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "corrupt-config")
}

// TestCmdRelease_runRelease_GetwdError covers the working-directory failure
// branch before any config work.
func TestCmdRelease_runRelease_GetwdError(t *core.T) {
	restoreReleaseStubs(t)
	getReleaseWorkingDir = func() core.Result { return core.Fail(core.NewError("no-cwd")) }
	resolveCalled := false
	releaseConfigExistsFn = func(string) bool { resolveCalled = true; return true }

	result := runRelease(context.Background(), true, false, "release", "", false, false, "")
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "get working directory")
	core.AssertFalse(t, resolveCalled)
}

// TestCmdRelease_runRelease_FullReleaseError surfaces a failing release run.
func TestCmdRelease_runRelease_FullReleaseError(t *core.T) {
	restoreReleaseStubs(t)
	getReleaseWorkingDir = func() core.Result { return core.Ok(t.TempDir()) }
	releaseConfigExistsFn = func(string) bool { return true }
	loadReleaseConfigFn = func(dir string) core.Result {
		cfg := release.DefaultConfig()
		cfg.SetProjectDir(dir)
		return core.Ok(cfg)
	}
	runFullReleaseFn = func(context.Context, *release.Config, bool) core.Result {
		return core.Fail(core.NewError("publish-failed"))
	}
	captureBuildStdout(t)

	result := runRelease(context.Background(), false, false, "release", "", false, false, "")
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "publish-failed")
}
