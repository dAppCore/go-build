package build

import (
	"testing"

	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/sdk"

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
		t.Setenv("DENO_BUILD", "deno task bundle")
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
  deno_build: ${DENO_BUILD}
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
		assert.Equal(t, "deno task bundle", cfg.Build.DenoBuild)
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

	t.Run("supports top-level cache block from the RFC", func(t *testing.T) {
		content := `
version: 1
cache:
  enabled: true
  dir: .core/cache
  paths:
    - ~/.cache/go-build
    - ~/go/pkg/mod
  restore_keys:
    - go-
`
		dir := setupConfigTestDir(t, content)

		cfg, err := LoadConfig(fs, dir)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		assert.True(t, cfg.Build.Cache.Enabled)
		assert.Equal(t, ".core/cache", cfg.Build.Cache.Directory)
		assert.Equal(t, []string{"~/.cache/go-build", "~/go/pkg/mod"}, cfg.Build.Cache.Paths)
		assert.Equal(t, []string{"go-"}, cfg.Build.Cache.RestoreKeys)
	})

	t.Run("loads apple pipeline config with env expansion", func(t *testing.T) {
		t.Setenv("APPLE_TEAM_ID", "ABC123DEF4")
		t.Setenv("APPLE_BUNDLE_ID", "ai.lthn.core")
		t.Setenv("APPLE_CERT_ID", "Developer ID Application: Lethean CIC (ABC123DEF4)")
		t.Setenv("APPLE_KEY_PATH", "/tmp/AuthKey_TEST.p8")
		t.Setenv("APPLE_METADATA_PATH", ".core/apple/appstore")
		t.Setenv("APPLE_PRIVACY_URL", "https://lthn.ai/privacy")
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
  metadata_path: ${APPLE_METADATA_PATH}
  privacy_policy_url: ${APPLE_PRIVACY_URL}
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
		assert.Equal(t, ".core/apple/appstore", cfg.Apple.MetadataPath)
		assert.Equal(t, "https://lthn.ai/privacy", cfg.Apple.PrivacyPolicyURL)
		assert.Equal(t, "/tmp/AuthKey_TEST.p8", cfg.Apple.APIKeyPath)
		assert.Equal(t, "assets/dmg-background.png", cfg.Apple.DMGBackground)
		assert.Equal(t, "CoreGUI Release", cfg.Apple.XcodeCloud.Workflow)
		require.Len(t, cfg.Apple.XcodeCloud.Triggers, 2)
		assert.Equal(t, "main", cfg.Apple.XcodeCloud.Triggers[0].Branch)
		assert.Equal(t, "testflight", cfg.Apple.XcodeCloud.Triggers[0].Action)
		assert.Equal(t, "v*", cfg.Apple.XcodeCloud.Triggers[1].Tag)
		assert.Equal(t, "appstore", cfg.Apple.XcodeCloud.Triggers[1].Action)
	})

	t.Run("loads immutable LinuxKit image config with env expansion", func(t *testing.T) {
		t.Setenv("CORE_IMAGE_BASE", "core-ml")
		t.Setenv("CORE_IMAGE_PACKAGE", "gh")
		t.Setenv("CORE_IMAGE_MOUNT", "/workspace")
		t.Setenv("CORE_IMAGE_FORMAT", "oci")
		t.Setenv("CORE_IMAGE_REGISTRY", "ghcr.io/dappcore")

		content := `
version: 1
linuxkit:
  base: ${CORE_IMAGE_BASE}
  packages:
    - ${CORE_IMAGE_PACKAGE}
  mounts:
    - ${CORE_IMAGE_MOUNT}
  gpu: true
  formats:
    - ${CORE_IMAGE_FORMAT}
    - apple
  registry: ${CORE_IMAGE_REGISTRY}
`
		dir := setupConfigTestDir(t, content)

		cfg, err := LoadConfig(fs, dir)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		assert.Equal(t, "core-ml", cfg.LinuxKit.Base)
		assert.Equal(t, []string{"gh"}, cfg.LinuxKit.Packages)
		assert.Equal(t, []string{"/workspace"}, cfg.LinuxKit.Mounts)
		assert.True(t, cfg.LinuxKit.GPU)
		assert.Equal(t, []string{"oci", "apple"}, cfg.LinuxKit.Formats)
		assert.Equal(t, "ghcr.io/dappcore", cfg.LinuxKit.Registry)
	})

	t.Run("loads sdk config from build yaml with shorthand diff and defaults", func(t *testing.T) {
		t.Setenv("SDK_SPEC", "docs/openapi.yaml")
		t.Setenv("SDK_LANG", "typescript")

		content := `
version: 1
sdk:
  spec: ${SDK_SPEC}
  languages:
    - ${SDK_LANG}
  diff: true
`
		dir := setupConfigTestDir(t, content)

		cfg, err := LoadConfig(fs, dir)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		require.NotNil(t, cfg.SDK)

		assert.Equal(t, "docs/openapi.yaml", cfg.SDK.Spec)
		assert.Equal(t, []string{"typescript"}, cfg.SDK.Languages)
		assert.Equal(t, "sdk", cfg.SDK.Output)
		assert.True(t, cfg.SDK.Diff.Enabled)
		assert.False(t, cfg.SDK.Diff.FailOnBreaking)
	})

	t.Run("honours explicit windows signtool disablement", func(t *testing.T) {
		content := `
version: 1
sign:
  windows:
    signtool: false
    certificate: C:/certs/core.pfx
`
		dir := setupConfigTestDir(t, content)

		cfg, err := LoadConfig(fs, dir)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		assert.False(t, cfg.Sign.Windows.Signtool)
		assert.Equal(t, "C:/certs/core.pfx", cfg.Sign.Windows.Certificate)
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
		assert.Equal(t, "core-dev", cfg.LinuxKit.Base)
		assert.Equal(t, []string{"/workspace"}, cfg.LinuxKit.Mounts)
		assert.Equal(t, []string{"oci", "apple"}, cfg.LinuxKit.Formats)

		// Default targets cover common platforms
		assert.Len(t, cfg.Targets, 5)
		hasLinuxAmd64 := false
		hasDarwinAmd64 := false
		hasDarwinArm64 := false
		hasWindowsAmd64 := false
		for _, t := range cfg.Targets {
			if t.OS == "linux" && t.Arch == "amd64" {
				hasLinuxAmd64 = true
			}
			if t.OS == "darwin" && t.Arch == "amd64" {
				hasDarwinAmd64 = true
			}
			if t.OS == "darwin" && t.Arch == "arm64" {
				hasDarwinArm64 = true
			}
			if t.OS == "windows" && t.Arch == "amd64" {
				hasWindowsAmd64 = true
			}
		}
		assert.True(t, hasLinuxAmd64)
		assert.True(t, hasDarwinAmd64)
		assert.True(t, hasDarwinArm64)
		assert.True(t, hasWindowsAmd64)
	})
}

func TestConfig_CloneBuildConfig_Good(t *testing.T) {
	sign := true
	notarise := false
	dmg := true

	cfg := &BuildConfig{
		Build: Build{
			Flags:     []string{"-trimpath"},
			LDFlags:   []string{"-s", "-w"},
			BuildTags: []string{"integration"},
			Env:       []string{"FOO=bar"},
			Cache:     CacheConfig{Enabled: true, Directory: ".core/cache", Paths: []string{"cache/go-build"}, RestoreKeys: []string{"main"}},
			Tags:      []string{"latest"},
			BuildArgs: map[string]string{"VERSION": "v1.2.3"},
			Formats:   []string{"iso"},
		},
		LinuxKit: LinuxKitConfig{
			Base:     "core-dev",
			Packages: []string{"git"},
			Mounts:   []string{"/workspace"},
			GPU:      true,
			Formats:  []string{"oci", "apple"},
			Registry: "ghcr.io/dappcore",
		},
		Apple: AppleConfig{
			Sign:     &sign,
			Notarise: &notarise,
			DMG:      &dmg,
			XcodeCloud: XcodeCloudConfig{
				Workflow: "Release",
				Triggers: []XcodeCloudTrigger{{Branch: "main", Action: "testflight"}},
			},
		},
		SDK: &sdk.Config{
			Spec:      "docs/openapi.yaml",
			Languages: []string{"typescript"},
			Output:    "generated/sdk",
		},
		Targets: []TargetConfig{{OS: "linux", Arch: "amd64"}},
	}

	clone := CloneBuildConfig(cfg)
	require.NotNil(t, clone)

	clone.Build.Flags[0] = "-mod=readonly"
	clone.Build.LDFlags[0] = "-X"
	clone.Build.BuildTags[0] = "release"
	clone.Build.Env[0] = "BAR=baz"
	clone.Build.Cache.Paths[0] = "cache/go-mod"
	clone.Build.Cache.RestoreKeys[0] = "fallback"
	clone.Build.Tags[0] = "stable"
	clone.Build.BuildArgs["VERSION"] = "v2.0.0"
	clone.Build.Formats[0] = "qcow2"
	clone.LinuxKit.Base = "core-minimal"
	clone.LinuxKit.Packages[0] = "task"
	clone.LinuxKit.Mounts[0] = "/src"
	clone.LinuxKit.Formats[0] = "tar"
	clone.LinuxKit.Registry = "registry.example.com/core"
	*clone.Apple.Sign = false
	*clone.Apple.Notarise = true
	*clone.Apple.DMG = false
	clone.Apple.XcodeCloud.Triggers[0].Branch = "dev"
	clone.SDK.Languages[0] = "python"
	clone.SDK.Output = "sdk"
	clone.Targets[0].OS = "darwin"

	assert.Equal(t, []string{"-trimpath"}, cfg.Build.Flags)
	assert.Equal(t, []string{"-s", "-w"}, cfg.Build.LDFlags)
	assert.Equal(t, []string{"integration"}, cfg.Build.BuildTags)
	assert.Equal(t, []string{"FOO=bar"}, cfg.Build.Env)
	assert.Equal(t, []string{"cache/go-build"}, cfg.Build.Cache.Paths)
	assert.Equal(t, []string{"main"}, cfg.Build.Cache.RestoreKeys)
	assert.Equal(t, []string{"latest"}, cfg.Build.Tags)
	assert.Equal(t, map[string]string{"VERSION": "v1.2.3"}, cfg.Build.BuildArgs)
	assert.Equal(t, []string{"iso"}, cfg.Build.Formats)
	assert.Equal(t, "core-dev", cfg.LinuxKit.Base)
	assert.Equal(t, []string{"git"}, cfg.LinuxKit.Packages)
	assert.Equal(t, []string{"/workspace"}, cfg.LinuxKit.Mounts)
	assert.Equal(t, []string{"oci", "apple"}, cfg.LinuxKit.Formats)
	assert.Equal(t, "ghcr.io/dappcore", cfg.LinuxKit.Registry)
	require.NotNil(t, cfg.Apple.Sign)
	require.NotNil(t, cfg.Apple.Notarise)
	require.NotNil(t, cfg.Apple.DMG)
	assert.True(t, *cfg.Apple.Sign)
	assert.False(t, *cfg.Apple.Notarise)
	assert.True(t, *cfg.Apple.DMG)
	require.Len(t, cfg.Apple.XcodeCloud.Triggers, 1)
	assert.Equal(t, "main", cfg.Apple.XcodeCloud.Triggers[0].Branch)
	require.NotNil(t, cfg.SDK)
	assert.Equal(t, []string{"typescript"}, cfg.SDK.Languages)
	assert.Equal(t, "generated/sdk", cfg.SDK.Output)
	assert.Equal(t, []TargetConfig{{OS: "linux", Arch: "amd64"}}, cfg.Targets)
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
