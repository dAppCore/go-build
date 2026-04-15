package buildcmd

import (
	"context"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/build/pkg/build/signing"
	"dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCmd_resolveAppleCommandOptions_Good(t *testing.T) {
	cfg := &build.BuildConfig{
		Apple: build.AppleConfig{
			BundleID: "ai.lthn.core",
			Arch:     "arm64",
			Sign:     boolPtr(false),
		},
		Sign: signing.SignConfig{
			MacOS: signing.MacOSConfig{
				Identity:    "Developer ID Application: Lethean CIC (ABC123DEF4)",
				TeamID:      "ABC123DEF4",
				AppleID:     "dev@example.com",
				AppPassword: "secret",
			},
		},
	}

	options := resolveAppleCommandOptions(cfg, appleCLIOptions{})
	assert.Equal(t, "ai.lthn.core", options.BundleID)
	assert.Equal(t, "arm64", options.Arch)
	assert.False(t, options.Sign)
	assert.Equal(t, "Developer ID Application: Lethean CIC (ABC123DEF4)", options.CertIdentity)
	assert.Equal(t, "ABC123DEF4", options.TeamID)
	assert.Equal(t, "dev@example.com", options.AppleID)
	assert.Equal(t, "secret", options.Password)

	options = resolveAppleCommandOptions(cfg, appleCLIOptions{
		Arch:              "universal",
		ArchChanged:       true,
		Sign:              true,
		SignChanged:       true,
		BundleID:          "ai.lthn.core.preview",
		BundleIDChanged:   true,
		TeamID:            "ZZZ9876543",
		TeamIDChanged:     true,
		TestFlight:        true,
		TestFlightChanged: true,
	})
	assert.Equal(t, "universal", options.Arch)
	assert.True(t, options.Sign)
	assert.Equal(t, "ai.lthn.core.preview", options.BundleID)
	assert.Equal(t, "ZZZ9876543", options.TeamID)
	assert.True(t, options.TestFlight)
}

func TestBuildCmd_resolveAppleBuildNumber_Good(t *testing.T) {
	t.Run("prefers github run number when valid", func(t *testing.T) {
		t.Setenv("GITHUB_RUN_NUMBER", "77")
		value, err := resolveAppleBuildNumber(context.Background(), t.TempDir())
		require.NoError(t, err)
		assert.Equal(t, "77", value)
	})

	t.Run("falls back to git commit count", func(t *testing.T) {
		dir := t.TempDir()
		runGit(t, dir, "init")
		runGit(t, dir, "config", "user.email", "test@example.com")
		runGit(t, dir, "config", "user.name", "Test User")

		require.NoError(t, ax.WriteFile(ax.Join(dir, "README.md"), []byte("hello\n"), 0o644))
		runGit(t, dir, "add", ".")
		runGit(t, dir, "commit", "-m", "feat: initial commit")

		t.Setenv("GITHUB_RUN_NUMBER", "")
		value, err := resolveAppleBuildNumber(context.Background(), dir)
		require.NoError(t, err)
		assert.Equal(t, "1", value)
	})
}

func TestBuildCmd_AddAppleCommand_Good(t *testing.T) {
	c := core.New()
	AddAppleCommand(c)

	result := c.Command("build/apple")
	require.True(t, result.OK)

	command, ok := result.Value.(*core.Command)
	require.True(t, ok)
	assert.Equal(t, "build/apple", command.Path)
	assert.Equal(t, "cmd.build.apple.long", command.Description)
}

func TestBuildCmd_runAppleBuildInDir_Good(t *testing.T) {
	projectDir := t.TempDir()
	coreDir := ax.Join(projectDir, ".core")
	require.NoError(t, ax.MkdirAll(coreDir, 0o755))
	require.NoError(t, ax.WriteFile(ax.Join(coreDir, "build.yaml"), []byte(`
project:
  name: Core
  binary: Core
apple:
  bundle_id: ai.lthn.core
  sign: false
sign:
  macos:
    identity: "Developer ID Application: Lethean CIC (ABC123DEF4)"
    team_id: ABC123DEF4
    apple_id: dev@example.com
    app_password: secret
`), 0o644))

	oldBuildApple := buildAppleFn
	t.Cleanup(func() {
		buildAppleFn = oldBuildApple
	})

	var called bool
	buildAppleFn = func(ctx context.Context, cfg *build.Config, options build.AppleOptions, buildNumber string) (*build.AppleBuildResult, error) {
		called = true
		assert.Equal(t, ax.Join(projectDir, "out"), cfg.OutputDir)
		assert.Equal(t, "Core", cfg.Name)
		assert.Equal(t, "v1.2.3", cfg.Version)
		assert.Equal(t, "42", buildNumber)
		assert.Equal(t, "ai.lthn.core", options.BundleID)
		assert.True(t, options.Sign)
		return &build.AppleBuildResult{
			BundlePath:  ax.Join(cfg.OutputDir, "Core.app"),
			Version:     "1.2.3",
			BuildNumber: buildNumber,
		}, nil
	}

	err := runAppleBuildInDir(context.Background(), projectDir, appleCLIOptions{
		Sign:        true,
		SignChanged: true,
		Version:     "v1.2.3",
		BuildNumber: "42",
		OutputDir:   "out",
	})
	require.NoError(t, err)
	assert.True(t, called)
}

func TestBuildCmd_runAppleBuildInDir_RejectsUnsafeVersion_Bad(t *testing.T) {
	projectDir := t.TempDir()
	coreDir := ax.Join(projectDir, ".core")
	require.NoError(t, ax.MkdirAll(coreDir, 0o755))
	require.NoError(t, ax.WriteFile(ax.Join(coreDir, "build.yaml"), []byte(`
project:
  name: Core
  binary: Core
apple:
  bundle_id: ai.lthn.core
  sign: false
`), 0o644))

	oldBuildApple := buildAppleFn
	t.Cleanup(func() {
		buildAppleFn = oldBuildApple
	})

	buildAppleFn = func(ctx context.Context, cfg *build.Config, options build.AppleOptions, buildNumber string) (*build.AppleBuildResult, error) {
		t.Fatal("buildAppleFn must not be called for unsafe versions")
		return nil, nil
	}

	err := runAppleBuildInDir(context.Background(), projectDir, appleCLIOptions{
		Version:     "v1.2.3 --bad",
		BuildNumber: "42",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid build version")
}

func TestBuildCmd_runAppleBuildInDir_SetsUpBuildCache_Good(t *testing.T) {
	projectDir := t.TempDir()
	coreDir := ax.Join(projectDir, ".core")
	require.NoError(t, ax.MkdirAll(coreDir, 0o755))
	require.NoError(t, ax.WriteFile(ax.Join(coreDir, "build.yaml"), []byte(`
project:
  name: Core
  binary: Core
build:
  cache:
    enabled: true
    paths:
      - cache/go-build
      - cache/go-mod
apple:
  bundle_id: ai.lthn.core
  sign: false
`), 0o644))

	oldBuildApple := buildAppleFn
	t.Cleanup(func() {
		buildAppleFn = oldBuildApple
	})

	buildAppleFn = func(ctx context.Context, cfg *build.Config, options build.AppleOptions, buildNumber string) (*build.AppleBuildResult, error) {
		assert.Equal(t, []string{
			ax.Join(projectDir, "cache", "go-build"),
			ax.Join(projectDir, "cache", "go-mod"),
		}, cfg.Cache.Paths)
		assert.True(t, cfg.Cache.Enabled)
		assert.True(t, cfg.FS.Exists(ax.Join(projectDir, ".core", "cache")))
		assert.True(t, cfg.FS.Exists(ax.Join(projectDir, "cache", "go-build")))
		assert.True(t, cfg.FS.Exists(ax.Join(projectDir, "cache", "go-mod")))
		return &build.AppleBuildResult{
			BundlePath:  ax.Join(cfg.OutputDir, "Core.app"),
			Version:     "1.2.3",
			BuildNumber: buildNumber,
		}, nil
	}

	err := runAppleBuildInDir(context.Background(), projectDir, appleCLIOptions{
		Version:     "v1.2.3",
		BuildNumber: "42",
	})
	require.NoError(t, err)
}

func TestBuildCmd_runAppleBuildInDir_WritesXcodeCloudScripts_Good(t *testing.T) {
	projectDir := t.TempDir()
	coreDir := ax.Join(projectDir, ".core")
	require.NoError(t, ax.MkdirAll(coreDir, 0o755))
	require.NoError(t, ax.WriteFile(ax.Join(coreDir, "build.yaml"), []byte(`
project:
  name: Core
  binary: Core
apple:
  bundle_id: ai.lthn.core
  sign: false
  xcode_cloud:
    workflow: CoreGUI Release
`), 0o644))

	oldBuildApple := buildAppleFn
	t.Cleanup(func() {
		buildAppleFn = oldBuildApple
	})

	buildAppleFn = func(ctx context.Context, cfg *build.Config, options build.AppleOptions, buildNumber string) (*build.AppleBuildResult, error) {
		return &build.AppleBuildResult{
			BundlePath:  ax.Join(cfg.OutputDir, "Core.app"),
			Version:     "1.2.3",
			BuildNumber: buildNumber,
		}, nil
	}

	err := runAppleBuildInDir(context.Background(), projectDir, appleCLIOptions{
		Version:     "v1.2.3",
		BuildNumber: "42",
	})
	require.NoError(t, err)

	preScriptPath := ax.Join(projectDir, build.XcodeCloudScriptsDir, build.XcodeCloudPreXcodebuildScriptName)
	preScript, err := ax.ReadFile(preScriptPath)
	require.NoError(t, err)
	assert.Contains(t, string(preScript), `core build apple --arch 'universal' --config '.core/build.yaml'`)
}

func boolPtr(value bool) *bool {
	return &value
}
