package buildcmd

import (
	"bytes"
	"context"
	"testing"

	"dappco.re/go/build/pkg/release"
	"dappco.re/go/core"
	"dappco.re/go/core/cli/pkg/cli"
)

func TestBuildCmd_applyReleaseArchiveFormatOverride_Good(t *testing.T) {
	cfg := release.DefaultConfig()

	err := applyReleaseArchiveFormatOverride(cfg, "xz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual("xz", cfg.Build.ArchiveFormat) {
		t.Fatalf("want %v, got %v", "xz", cfg.Build.ArchiveFormat)
	}

}

func TestBuildCmd_applyReleaseArchiveFormatOverride_Bad(t *testing.T) {
	cfg := release.DefaultConfig()

	err := applyReleaseArchiveFormatOverride(cfg, "bogus")
	if err == nil {
		t.Fatal("expected error")
	}
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
	getReleaseWorkingDir = func() (string, error) { return projectDir, nil }

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
	loadReleaseConfigFn = func(dir string) (*release.Config, error) {
		cfg := release.DefaultConfig()
		cfg.SetProjectDir(dir)
		cfg.SDK = &release.SDKConfig{
			Languages: []string{"typescript", "go"},
			Output:    "sdk",
		}
		return cfg, nil
	}

	called := false
	runSDKReleaseFn = func(ctx context.Context, cfg *release.Config, dryRun bool) (*release.SDKRelease, error) {
		called = true
		if !(dryRun) {
			t.Fatal("expected true")
		}
		if stdlibAssertNil(cfg.SDK) {
			t.Fatal("expected non-nil")
		}

		return &release.SDKRelease{
			Version:   "v1.2.3",
			Output:    "sdk",
			Languages: []string{"typescript", "go"},
		}, nil
	}

	err := runRelease(context.Background(), true, false, "sdk", "v1.2.3", false, false, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !(called) {
		t.Fatal("expected true")
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

	getReleaseWorkingDir = func() (string, error) { return projectDir, nil }
	releaseConfigExistsFn = func(dir string) bool { return true }

	err := runRelease(context.Background(), true, false, "release", "v1.2.3 --bad", false, false, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "invalid release version override") {
		t.Fatalf("expected %v to contain %v", err.Error(), "invalid release version override")
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

	getReleaseWorkingDir = func() (string, error) { return projectDir, nil }
	releaseConfigExistsFn = func(dir string) bool {
		if !stdlibAssertEqual(projectDir, dir) {
			t.Fatalf("want %v, got %v", projectDir, dir)
		}

		return false
	}

	var stdout bytes.Buffer
	cli.SetStdout(&stdout)
	cli.SetStderr(&stdout)

	err := runRelease(context.Background(), false, true, "release", "", false, false, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(stdout.String(), emitCIAnnotationForTest(err)) {
		t.Fatalf("expected %v to contain %v", stdout.String(), emitCIAnnotationForTest(err))
	}

}
