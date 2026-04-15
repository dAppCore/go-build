package buildcmd

import (
	"bytes"
	"context"
	"testing"

	"dappco.re/go/build/pkg/release"
	"dappco.re/go/core"
	"dappco.re/go/core/cli/pkg/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCmd_applyReleaseArchiveFormatOverride_Good(t *testing.T) {
	cfg := release.DefaultConfig()

	err := applyReleaseArchiveFormatOverride(cfg, "xz")
	require.NoError(t, err)
	assert.Equal(t, "xz", cfg.Build.ArchiveFormat)
}

func TestBuildCmd_applyReleaseArchiveFormatOverride_Bad(t *testing.T) {
	cfg := release.DefaultConfig()

	err := applyReleaseArchiveFormatOverride(cfg, "bogus")
	require.Error(t, err)
	assert.Equal(t, "", cfg.Build.ArchiveFormat)
}

func TestBuildCmd_AddReleaseCommand_RegistersTopLevelAlias_Good(t *testing.T) {
	c := core.New()

	AddReleaseCommand(c)

	assert.True(t, c.Command("build/release").OK)
	assert.True(t, c.Command("release").OK)
}

func TestBuildCmd_resolveReleaseDryRun_Good(t *testing.T) {
	assert.False(t, resolveReleaseDryRun(false, false, false))
	assert.True(t, resolveReleaseDryRun(true, false, false))
	assert.False(t, resolveReleaseDryRun(false, true, false))
	assert.False(t, resolveReleaseDryRun(true, true, false))
	assert.False(t, resolveReleaseDryRun(false, false, true))
	assert.False(t, resolveReleaseDryRun(true, false, true))
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
		assert.Equal(t, projectDir, dir)
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
		assert.True(t, dryRun)
		require.NotNil(t, cfg.SDK)
		return &release.SDKRelease{
			Version:   "v1.2.3",
			Output:    "sdk",
			Languages: []string{"typescript", "go"},
		}, nil
	}

	err := runRelease(context.Background(), true, false, "sdk", "v1.2.3", false, false, "")
	require.NoError(t, err)
	assert.True(t, called)
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
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid release version override")
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
		assert.Equal(t, projectDir, dir)
		return false
	}

	var stdout bytes.Buffer
	cli.SetStdout(&stdout)
	cli.SetStderr(&stdout)

	err := runRelease(context.Background(), false, true, "release", "", false, false, "")
	require.Error(t, err)
	assert.Contains(t, stdout.String(), emitCIAnnotationForTest(err))
}
