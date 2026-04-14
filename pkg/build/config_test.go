package build

import (
	"testing"

	"dappco.re/go/core/build/internal/ax"

	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupConfigTestDir creates a temp directory with optional .core/build.yaml content.
func setupConfigTestDir(t *testing.T, configContent string) string {
	t.Helper()
	dir := t.TempDir()

	if configContent != "" {
		coreDir := ax.Join(dir, ConfigDir)
		err := ax.MkdirAll(coreDir, 0755)
		require.NoError(t, err)

		configPath := ax.Join(coreDir, ConfigFileName)
		err = ax.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)
	}

	return dir
}

func TestConfig_LoadConfig_Good(t *testing.T) {
	fs := io.Local
	t.Run("loads valid config", func(t *testing.T) {
		content := `
version: 1
project:
  name: myapp
  description: A test application
  main: ./cmd/myapp
  binary: myapp
build:
  cgo: true
  flags:
    - -trimpath
    - -race
  ldflags:
    - -s
    - -w
  build_tags:
    - integration
    - webkit2_41
  archive_format: xz
  env:
    - FOO=bar
  load: true
targets:
  - os: linux
    arch: amd64
  - os: darwin
    arch: arm64
`
		dir := setupConfigTestDir(t, content)

		cfg, err := LoadConfig(fs, dir)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		assert.Equal(t, 1, cfg.Version)
		assert.Equal(t, "myapp", cfg.Project.Name)
		assert.Equal(t, "A test application", cfg.Project.Description)
		assert.Equal(t, "./cmd/myapp", cfg.Project.Main)
		assert.Equal(t, "myapp", cfg.Project.Binary)
		assert.True(t, cfg.Build.CGO)
		assert.Equal(t, []string{"-trimpath", "-race"}, cfg.Build.Flags)
		assert.Equal(t, []string{"-s", "-w"}, cfg.Build.LDFlags)
		assert.Equal(t, []string{"integration", "webkit2_41"}, cfg.Build.BuildTags)
		assert.Equal(t, "xz", cfg.Build.ArchiveFormat)
		assert.Equal(t, []string{"FOO=bar"}, cfg.Build.Env)
		assert.True(t, cfg.Build.Load)
		assert.Len(t, cfg.Targets, 2)
		assert.Equal(t, "linux", cfg.Targets[0].OS)
		assert.Equal(t, "amd64", cfg.Targets[0].Arch)
		assert.Equal(t, "darwin", cfg.Targets[1].OS)
		assert.Equal(t, "arm64", cfg.Targets[1].Arch)
	})

	t.Run("expands environment variables in target config", func(t *testing.T) {
		t.Setenv("TARGET_OS", "linux")
		t.Setenv("TARGET_ARCH", "arm64")

		content := `
version: 1
targets:
  - os: ${TARGET_OS}
    arch: ${TARGET_ARCH}
`
		dir := setupConfigTestDir(t, content)

		cfg, err := LoadConfig(fs, dir)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		require.Len(t, cfg.Targets, 1)
		assert.Equal(t, "linux", cfg.Targets[0].OS)
		assert.Equal(t, "arm64", cfg.Targets[0].Arch)
	})

	t.Run("expands environment variables in build and signing config", func(t *testing.T) {
		t.Setenv("APP_NAME", "demo-app")
		t.Setenv("APP_ROOT", "./cmd/demo")
		t.Setenv("APP_BINARY", "demo-bin")
		t.Setenv("BUILD_TYPE", "wails")
		t.Setenv("WEBVIEW2", "embed")
		t.Setenv("ARCHIVE_FORMAT", "xz")
		t.Setenv("APP_VERSION", "v1.2.3")
		t.Setenv("APP_TAG", "integration")
		t.Setenv("CACHE_DIR", ".core/cache/demo-app")
		t.Setenv("DOCKERFILE", "Dockerfile.release")
		t.Setenv("IMAGE_NAME", "owner/demo-app")
		t.Setenv("GPG_KEY_ID", "ABCD1234")

		content := `
version: 1
project:
  name: ${APP_NAME}
  main: ${APP_ROOT}
  binary: ${APP_BINARY}
build:
  type: ${BUILD_TYPE}
  webview2: ${WEBVIEW2}
  archive_format: ${ARCHIVE_FORMAT}
  flags:
    - -trimpath
    - -X
    - main.version=${APP_VERSION}
  ldflags:
    - -s
    - -w
  build_tags:
    - ${APP_TAG}
  env:
    - VERSION=${APP_VERSION}
  cache:
    enabled: true
    dir: ${CACHE_DIR}
    paths:
      - ${CACHE_DIR}/go-build
  dockerfile: ${DOCKERFILE}
  image: ${IMAGE_NAME}
  tags:
    - latest
    - ${APP_VERSION}
  build_args:
    VERSION: ${APP_VERSION}
sign:
  gpg:
    key: ${GPG_KEY_ID}
`
		dir := setupConfigTestDir(t, content)

		cfg, err := LoadConfig(fs, dir)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		assert.Equal(t, "demo-app", cfg.Project.Name)
		assert.Equal(t, "./cmd/demo", cfg.Project.Main)
		assert.Equal(t, "demo-bin", cfg.Project.Binary)
		assert.Equal(t, "wails", cfg.Build.Type)
		assert.Equal(t, "embed", cfg.Build.WebView2)
		assert.Equal(t, "xz", cfg.Build.ArchiveFormat)
		assert.Equal(t, []string{"-trimpath", "-X", "main.version=v1.2.3"}, cfg.Build.Flags)
		assert.Equal(t, []string{"-s", "-w"}, cfg.Build.LDFlags)
		assert.Equal(t, []string{"integration"}, cfg.Build.BuildTags)
		assert.Equal(t, []string{"VERSION=v1.2.3"}, cfg.Build.Env)
		assert.Equal(t, ".core/cache/demo-app", cfg.Build.Cache.Directory)
		assert.Equal(t, []string{".core/cache/demo-app/go-build"}, cfg.Build.Cache.Paths)
		assert.Equal(t, "Dockerfile.release", cfg.Build.Dockerfile)
		assert.Equal(t, "owner/demo-app", cfg.Build.Image)
		assert.Equal(t, []string{"latest", "v1.2.3"}, cfg.Build.Tags)
		assert.Equal(t, map[string]string{"VERSION": "v1.2.3"}, cfg.Build.BuildArgs)
		assert.Equal(t, "ABCD1234", cfg.Sign.GPG.Key)
	})

	t.Run("loads apple pipeline config with env expansion", func(t *testing.T) {
		t.Setenv("APPLE_TEAM_ID", "ABC123DEF4")
		t.Setenv("APPLE_BUNDLE_ID", "ai.lthn.core")
		t.Setenv("APPLE_CERT_ID", "Developer ID Application: Lethean CIC (ABC123DEF4)")
		t.Setenv("APPLE_KEY_PATH", "/tmp/AuthKey_TEST.p8")
		t.Setenv("APPLE_BG", "assets/dmg-background.png")
		t.Setenv("XCLOUD_WORKFLOW", "CoreGUI Release")
		t.Setenv("XCLOUD_BRANCH", "main")

		content := `
version: 1
apple:
  team_id: ${APPLE_TEAM_ID}
  bundle_id: ${APPLE_BUNDLE_ID}
  arch: universal
  cert_identity: ${APPLE_CERT_ID}
  sign: false
  notarise: true
  dmg: true
  api_key_path: ${APPLE_KEY_PATH}
  dmg_background: ${APPLE_BG}
  xcode_cloud:
    workflow: ${XCLOUD_WORKFLOW}
    triggers:
      - branch: ${XCLOUD_BRANCH}
        action: testflight
      - tag: v*
        action: appstore
`
		dir := setupConfigTestDir(t, content)

		cfg, err := LoadConfig(fs, dir)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		assert.Equal(t, "ABC123DEF4", cfg.Apple.TeamID)
		assert.Equal(t, "ai.lthn.core", cfg.Apple.BundleID)
		assert.Equal(t, "universal", cfg.Apple.Arch)
		assert.Equal(t, "Developer ID Application: Lethean CIC (ABC123DEF4)", cfg.Apple.CertIdentity)
		require.NotNil(t, cfg.Apple.Sign)
		assert.False(t, *cfg.Apple.Sign)
		require.NotNil(t, cfg.Apple.Notarise)
		assert.True(t, *cfg.Apple.Notarise)
		require.NotNil(t, cfg.Apple.DMG)
		assert.True(t, *cfg.Apple.DMG)
		assert.Equal(t, "/tmp/AuthKey_TEST.p8", cfg.Apple.APIKeyPath)
		assert.Equal(t, "assets/dmg-background.png", cfg.Apple.DMGBackground)
		assert.Equal(t, "CoreGUI Release", cfg.Apple.XcodeCloud.Workflow)
		require.Len(t, cfg.Apple.XcodeCloud.Triggers, 2)
		assert.Equal(t, "main", cfg.Apple.XcodeCloud.Triggers[0].Branch)
		assert.Equal(t, "testflight", cfg.Apple.XcodeCloud.Triggers[0].Action)
		assert.Equal(t, "v*", cfg.Apple.XcodeCloud.Triggers[1].Tag)
		assert.Equal(t, "appstore", cfg.Apple.XcodeCloud.Triggers[1].Action)
	})

	t.Run("returns defaults when config file missing", func(t *testing.T) {
		dir := t.TempDir()

		cfg, err := LoadConfig(fs, dir)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		defaults := DefaultConfig()
		assert.Equal(t, defaults.Version, cfg.Version)
		assert.Equal(t, defaults.Project.Main, cfg.Project.Main)
		assert.Equal(t, defaults.Build.CGO, cfg.Build.CGO)
		assert.Equal(t, defaults.Build.Flags, cfg.Build.Flags)
		assert.Equal(t, defaults.Build.LDFlags, cfg.Build.LDFlags)
		assert.False(t, cfg.Build.Load)
		assert.Empty(t, cfg.Build.BuildTags)
		assert.Equal(t, defaults.Targets, cfg.Targets)
	})

	t.Run("applies defaults for missing fields", func(t *testing.T) {
		content := `
version: 2
project:
  name: partial
`
		dir := setupConfigTestDir(t, content)

		cfg, err := LoadConfig(fs, dir)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Explicit values preserved
		assert.Equal(t, 2, cfg.Version)
		assert.Equal(t, "partial", cfg.Project.Name)

		// Defaults applied
		defaults := DefaultConfig()
		assert.Equal(t, defaults.Project.Main, cfg.Project.Main)
		assert.Equal(t, defaults.Build.Flags, cfg.Build.Flags)
		assert.Equal(t, defaults.Build.LDFlags, cfg.Build.LDFlags)
		assert.Equal(t, defaults.Targets, cfg.Targets)
		assert.True(t, cfg.Sign.Enabled)
	})

	t.Run("preserves explicit signing disablement", func(t *testing.T) {
		content := `
version: 1
sign:
  enabled: false
`
		dir := setupConfigTestDir(t, content)

		cfg, err := LoadConfig(fs, dir)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		assert.False(t, cfg.Sign.Enabled)
	})

	t.Run("preserves empty arrays when explicitly set", func(t *testing.T) {
		content := `
version: 1
project:
  name: noflags
build:
  flags: []
  ldflags: []
  build_tags: []
targets:
  - os: linux
    arch: amd64
`
		dir := setupConfigTestDir(t, content)

		cfg, err := LoadConfig(fs, dir)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Empty arrays are preserved (not replaced with defaults)
		assert.Empty(t, cfg.Build.Flags)
		assert.Empty(t, cfg.Build.LDFlags)
		assert.Empty(t, cfg.Build.BuildTags)
		// Targets explicitly set
		assert.Len(t, cfg.Targets, 1)
	})
}

func TestConfig_LoadConfigAtPath_Good(t *testing.T) {
	fs := io.Local

	t.Run("loads config from explicit file path", func(t *testing.T) {
		dir := t.TempDir()
		configPath := ax.Join(dir, "custom-build.yaml")
		content := `
version: 3
project:
  name: custom-app
  binary: custom-app
build:
  cgo: true
targets:
  - os: linux
    arch: amd64
`
		err := ax.WriteFile(configPath, []byte(content), 0644)
		require.NoError(t, err)

		cfg, err := LoadConfigAtPath(fs, configPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		assert.Equal(t, 3, cfg.Version)
		assert.Equal(t, "custom-app", cfg.Project.Name)
		assert.Equal(t, "custom-app", cfg.Project.Binary)
		assert.True(t, cfg.Build.CGO)
		assert.Empty(t, cfg.Build.BuildTags)
		assert.Len(t, cfg.Targets, 1)
		assert.Equal(t, "linux", cfg.Targets[0].OS)
		assert.Equal(t, "amd64", cfg.Targets[0].Arch)
	})
}

func TestConfig_LoadConfig_Bad(t *testing.T) {
	fs := io.Local
	t.Run("returns error for invalid YAML", func(t *testing.T) {
		content := `
version: 1
project:
  name: [invalid yaml
`
		dir := setupConfigTestDir(t, content)

		cfg, err := LoadConfig(fs, dir)
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "failed to parse config file")
	})

	t.Run("returns error for unreadable file", func(t *testing.T) {
		dir := t.TempDir()
		coreDir := ax.Join(dir, ConfigDir)
		err := ax.MkdirAll(coreDir, 0755)
		require.NoError(t, err)

		// Create config as a directory instead of file
		configPath := ax.Join(coreDir, ConfigFileName)
		err = ax.Mkdir(configPath, 0755)
		require.NoError(t, err)

		cfg, err := LoadConfig(fs, dir)
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "failed to read config file")
	})
}

func TestConfig_DefaultConfig_Good(t *testing.T) {
	t.Run("returns sensible defaults", func(t *testing.T) {
		cfg := DefaultConfig()

		assert.Equal(t, 1, cfg.Version)
		assert.Equal(t, ".", cfg.Project.Main)
		assert.Empty(t, cfg.Project.Name)
		assert.Empty(t, cfg.Project.Binary)
		assert.False(t, cfg.Build.CGO)
		assert.Contains(t, cfg.Build.Flags, "-trimpath")
		assert.Contains(t, cfg.Build.LDFlags, "-s")
		assert.Contains(t, cfg.Build.LDFlags, "-w")
		assert.Empty(t, cfg.Build.Env)

		// Default targets cover common platforms
		assert.Len(t, cfg.Targets, 4)
		hasLinuxAmd64 := false
		hasDarwinArm64 := false
		hasWindowsAmd64 := false
		for _, t := range cfg.Targets {
			if t.OS == "linux" && t.Arch == "amd64" {
				hasLinuxAmd64 = true
			}
			if t.OS == "darwin" && t.Arch == "arm64" {
				hasDarwinArm64 = true
			}
			if t.OS == "windows" && t.Arch == "amd64" {
				hasWindowsAmd64 = true
			}
		}
		assert.True(t, hasLinuxAmd64)
		assert.True(t, hasDarwinArm64)
		assert.True(t, hasWindowsAmd64)
	})
}

func TestConfig_ConfigPath_Good(t *testing.T) {
	t.Run("returns correct path", func(t *testing.T) {
		path := ConfigPath("/project/root")
		assert.Equal(t, "/project/root/.core/build.yaml", path)
	})
}

func TestConfig_ConfigExists_Good(t *testing.T) {
	fs := io.Local
	t.Run("returns true when config exists", func(t *testing.T) {
		dir := setupConfigTestDir(t, "version: 1")
		assert.True(t, ConfigExists(fs, dir))
	})

	t.Run("returns false when config missing", func(t *testing.T) {
		dir := t.TempDir()
		assert.False(t, ConfigExists(fs, dir))
	})

	t.Run("returns false when .core dir missing", func(t *testing.T) {
		dir := t.TempDir()
		assert.False(t, ConfigExists(fs, dir))
	})
}

func TestConfig_LoadConfigSignConfig_Good(t *testing.T) {
	tmpDir := t.TempDir()
	coreDir := ax.Join(tmpDir, ".core")
	_ = ax.MkdirAll(coreDir, 0755)

	configContent := `version: 1
sign:
  enabled: true
  gpg:
    key: "ABCD1234"
  macos:
    identity: "Developer ID Application: Test"
    notarize: true
`
	_ = ax.WriteFile(ax.Join(coreDir, "build.yaml"), []byte(configContent), 0644)

	cfg, err := LoadConfig(io.Local, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cfg.Sign.Enabled {
		t.Error("expected Sign.Enabled to be true")
	}
	if cfg.Sign.GPG.Key != "ABCD1234" {
		t.Errorf("expected GPG.Key 'ABCD1234', got %q", cfg.Sign.GPG.Key)
	}
	if cfg.Sign.MacOS.Identity != "Developer ID Application: Test" {
		t.Errorf("expected MacOS.Identity, got %q", cfg.Sign.MacOS.Identity)
	}
	if !cfg.Sign.MacOS.Notarize {
		t.Error("expected MacOS.Notarize to be true")
	}
}

func TestConfig_BuildConfigToTargets_Good(t *testing.T) {
	t.Run("converts TargetConfig to Target", func(t *testing.T) {
		cfg := &BuildConfig{
			Targets: []TargetConfig{
				{OS: "linux", Arch: "amd64"},
				{OS: "darwin", Arch: "arm64"},
				{OS: "windows", Arch: "386"},
			},
		}

		targets := cfg.ToTargets()
		require.Len(t, targets, 3)

		assert.Equal(t, Target{OS: "linux", Arch: "amd64"}, targets[0])
		assert.Equal(t, Target{OS: "darwin", Arch: "arm64"}, targets[1])
		assert.Equal(t, Target{OS: "windows", Arch: "386"}, targets[2])
	})

	t.Run("returns empty slice for no targets", func(t *testing.T) {
		cfg := &BuildConfig{
			Targets: []TargetConfig{},
		}

		targets := cfg.ToTargets()
		assert.Empty(t, targets)
	})
}

// TestLoadConfig_Testdata tests loading from the testdata fixture.
func TestConfig_LoadConfigTestdata_Good(t *testing.T) {
	fs := io.Local
	abs, err := ax.Abs("testdata/config-project")
	require.NoError(t, err)

	t.Run("loads config-project fixture", func(t *testing.T) {
		cfg, err := LoadConfig(fs, abs)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		assert.Equal(t, 1, cfg.Version)
		assert.Equal(t, "example-cli", cfg.Project.Name)
		assert.Equal(t, "An example CLI application", cfg.Project.Description)
		assert.Equal(t, "./cmd/example", cfg.Project.Main)
		assert.Equal(t, "example", cfg.Project.Binary)
		assert.False(t, cfg.Build.CGO)
		assert.Equal(t, []string{"-trimpath"}, cfg.Build.Flags)
		assert.Equal(t, []string{"-s", "-w"}, cfg.Build.LDFlags)
		assert.Len(t, cfg.Targets, 3)
	})
}
