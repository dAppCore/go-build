package build

import (
	"reflect"
	"strings"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/sdk"
	"dappco.re/go/core/io"

	"gopkg.in/yaml.v3"
)

// setupConfigTestDir creates a temp directory with optional .core/build.yaml content.
func setupConfigTestDir(t *testing.T, configContent string) string {
	t.Helper()
	dir := t.TempDir()

	if configContent != "" {
		coreDir := ax.Join(dir, ConfigDir)
		err := ax.MkdirAll(coreDir, 0755)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		configPath := ax.Join(coreDir, ConfigFileName)
		err = ax.WriteFile(configPath, []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual(1, cfg.Version) {
			t.Fatalf("want %v, got %v", 1, cfg.Version)
		}
		if !stdlibAssertEqual("myapp", cfg.Project.Name) {
			t.Fatalf("want %v, got %v", "myapp", cfg.Project.Name)
		}
		if !stdlibAssertEqual("A test application", cfg.Project.Description) {
			t.Fatalf("want %v, got %v", "A test application", cfg.Project.Description)
		}
		if !stdlibAssertEqual("./cmd/myapp", cfg.Project.Main) {
			t.Fatalf("want %v, got %v", "./cmd/myapp", cfg.Project.Main)
		}
		if !stdlibAssertEqual("myapp", cfg.Project.Binary) {
			t.Fatalf("want %v, got %v", "myapp", cfg.Project.Binary)
		}
		if !(cfg.Build.CGO) {
			t.Fatal("expected true")
		}
		if !stdlibAssertEqual([]string{"-trimpath", "-race"}, cfg.Build.Flags) {
			t.Fatalf("want %v, got %v", []string{"-trimpath", "-race"}, cfg.Build.Flags)
		}
		if !stdlibAssertEqual([]string{"-s", "-w"}, cfg.Build.LDFlags) {
			t.Fatalf("want %v, got %v", []string{"-s", "-w"}, cfg.Build.LDFlags)
		}
		if !stdlibAssertEqual([]string{"integration", "webkit2_41"}, cfg.Build.BuildTags) {
			t.Fatalf("want %v, got %v", []string{"integration", "webkit2_41"}, cfg.Build.BuildTags)
		}
		if !stdlibAssertEqual("xz", cfg.Build.ArchiveFormat) {
			t.Fatalf("want %v, got %v", "xz", cfg.Build.ArchiveFormat)
		}
		if !stdlibAssertEqual([]string{"FOO=bar"}, cfg.Build.Env) {
			t.Fatalf("want %v, got %v", []string{"FOO=bar"}, cfg.Build.Env)
		}
		if !(cfg.Build.Load) {
			t.Fatal("expected true")
		}
		if len(cfg.Targets) != 2 {
			t.Fatalf("want len %v, got %v", 2, len(cfg.Targets))
		}
		if !stdlibAssertEqual("linux", cfg.Targets[0].OS) {
			t.Fatalf("want %v, got %v", "linux", cfg.Targets[0].OS)
		}
		if !stdlibAssertEqual("amd64", cfg.Targets[0].Arch) {
			t.Fatalf("want %v, got %v", "amd64", cfg.Targets[0].Arch)
		}
		if !stdlibAssertEqual("darwin", cfg.Targets[1].OS) {
			t.Fatalf("want %v, got %v", "darwin", cfg.Targets[1].OS)
		}
		if !stdlibAssertEqual("arm64", cfg.Targets[1].Arch) {
			t.Fatalf("want %v, got %v", "arm64", cfg.Targets[1].Arch)
		}

	})

	t.Run("defaults to the local medium when nil is passed", func(t *testing.T) {
		content := `
version: 1
project:
  name: nil-medium
`
		dir := setupConfigTestDir(t, content)

		cfg, err := LoadConfig(nil, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("nil-medium", cfg.Project.Name) {
			t.Fatalf("want %v, got %v", "nil-medium", cfg.Project.Name)
		}

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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}
		if len(cfg.Targets) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(cfg.Targets))
		}
		if !stdlibAssertEqual("linux", cfg.Targets[0].OS) {
			t.Fatalf("want %v, got %v", "linux", cfg.Targets[0].OS)
		}
		if !stdlibAssertEqual("arm64", cfg.Targets[0].Arch) {
			t.Fatalf("want %v, got %v", "arm64", cfg.Targets[0].Arch)
		}

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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("demo-app", cfg.Project.Name) {
			t.Fatalf("want %v, got %v", "demo-app", cfg.Project.Name)
		}
		if !stdlibAssertEqual("./cmd/demo", cfg.Project.Main) {
			t.Fatalf("want %v, got %v", "./cmd/demo", cfg.Project.Main)
		}
		if !stdlibAssertEqual("demo-bin", cfg.Project.Binary) {
			t.Fatalf("want %v, got %v", "demo-bin", cfg.Project.Binary)
		}
		if !stdlibAssertEqual("wails", cfg.Build.Type) {
			t.Fatalf("want %v, got %v", "wails", cfg.Build.Type)
		}
		if !stdlibAssertEqual("deno task bundle", cfg.Build.DenoBuild) {
			t.Fatalf("want %v, got %v", "deno task bundle", cfg.Build.DenoBuild)
		}
		if !stdlibAssertEqual("embed", cfg.Build.WebView2) {
			t.Fatalf("want %v, got %v", "embed", cfg.Build.WebView2)
		}
		if !stdlibAssertEqual("xz", cfg.Build.ArchiveFormat) {
			t.Fatalf("want %v, got %v", "xz", cfg.Build.ArchiveFormat)
		}
		if !stdlibAssertEqual([]string{"-trimpath", "-X", "main.version=v1.2.3"}, cfg.Build.Flags) {
			t.Fatalf("want %v, got %v", []string{"-trimpath", "-X", "main.version=v1.2.3"}, cfg.Build.Flags)
		}
		if !stdlibAssertEqual([]string{"-s", "-w"}, cfg.Build.LDFlags) {
			t.Fatalf("want %v, got %v", []string{"-s", "-w"}, cfg.Build.LDFlags)
		}
		if !stdlibAssertEqual([]string{"integration"}, cfg.Build.BuildTags) {
			t.Fatalf("want %v, got %v", []string{"integration"}, cfg.Build.BuildTags)
		}
		if !stdlibAssertEqual([]string{"VERSION=v1.2.3"}, cfg.Build.Env) {
			t.Fatalf("want %v, got %v", []string{"VERSION=v1.2.3"}, cfg.Build.Env)
		}
		if !stdlibAssertEqual(".core/cache/demo-app", cfg.Build.Cache.Directory) {
			t.Fatalf("want %v, got %v", ".core/cache/demo-app", cfg.Build.Cache.Directory)
		}
		if !stdlibAssertEqual([]string{".core/cache/demo-app/go-build"}, cfg.Build.Cache.Paths) {
			t.Fatalf("want %v, got %v", []string{".core/cache/demo-app/go-build"}, cfg.Build.Cache.Paths)
		}
		if !stdlibAssertEqual("Dockerfile.release", cfg.Build.Dockerfile) {
			t.Fatalf("want %v, got %v", "Dockerfile.release", cfg.Build.Dockerfile)
		}
		if !stdlibAssertEqual("owner/demo-app", cfg.Build.Image) {
			t.Fatalf("want %v, got %v", "owner/demo-app", cfg.Build.Image)
		}
		if !stdlibAssertEqual([]string{"latest", "v1.2.3"}, cfg.Build.Tags) {
			t.Fatalf("want %v, got %v", []string{"latest", "v1.2.3"}, cfg.Build.Tags)
		}
		if !stdlibAssertEqual(map[string]string{"VERSION": "v1.2.3"}, cfg.Build.BuildArgs) {
			t.Fatalf("want %v, got %v", map[string]string{"VERSION": "v1.2.3"}, cfg.Build.BuildArgs)
		}
		if !stdlibAssertEqual("ABCD1234", cfg.Sign.GPG.Key) {
			t.Fatalf("want %v, got %v", "ABCD1234", cfg.Sign.GPG.Key)
		}

	})

	t.Run("loads RFC build flags for obfuscation and NSIS", func(t *testing.T) {
		content := `
version: 1
build:
  obfuscate: true
  nsis: true
  webview2: download
`
		dir := setupConfigTestDir(t, content)

		cfg, err := LoadConfig(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}
		if !(cfg.Build.Obfuscate) {
			t.Fatal("expected true")
		}
		if !(cfg.Build.NSIS) {
			t.Fatal("expected true")
		}
		if !stdlibAssertEqual("download", cfg.Build.WebView2) {
			t.Fatalf("want %v, got %v", "download", cfg.Build.WebView2)
		}

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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}
		if !(cfg.Build.Cache.Enabled) {
			t.Fatal("expected true")
		}
		if !stdlibAssertEqual(".core/cache", cfg.Build.Cache.Directory) {
			t.Fatalf("want %v, got %v", ".core/cache", cfg.Build.Cache.Directory)
		}
		if !stdlibAssertEqual([]string{"~/.cache/go-build", "~/go/pkg/mod"}, cfg.Build.Cache.Paths) {
			t.Fatalf("want %v, got %v", []string{"~/.cache/go-build", "~/go/pkg/mod"}, cfg.Build.Cache.Paths)
		}
		if !stdlibAssertEqual([]string{"go-"}, cfg.Build.Cache.RestoreKeys) {
			t.Fatalf("want %v, got %v", []string{"go-"}, cfg.Build.Cache.RestoreKeys)
		}

	})

	t.Run("supports RFC pre_build block for frontend hooks", func(t *testing.T) {
		t.Setenv("DENO_BUILD", "deno task bundle")
		t.Setenv("NPM_BUILD", "npm run bundle")

		content := `
version: 1
pre_build:
  deno: ${DENO_BUILD}
  npm: ${NPM_BUILD}
`
		dir := setupConfigTestDir(t, content)

		cfg, err := LoadConfig(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("deno task bundle", cfg.Build.DenoBuild) {
			t.Fatalf("want %v, got %v", "deno task bundle", cfg.Build.DenoBuild)
		}
		if !stdlibAssertEqual("npm run bundle", cfg.Build.NpmBuild) {
			t.Fatalf("want %v, got %v", "npm run bundle", cfg.Build.NpmBuild)
		}
		if !stdlibAssertEqual(PreBuild{Deno: "deno task bundle", Npm: "npm run bundle"}, cfg.PreBuild) {
			t.Fatalf("want %v, got %v", PreBuild{Deno: "deno task bundle", Npm: "npm run bundle"}, cfg.PreBuild)
		}

	})

	t.Run("keeps legacy build frontend hooks when both shapes are present", func(t *testing.T) {
		content := `
version: 1
build:
  deno_build: deno task legacy
  npm_build: npm run legacy
pre_build:
  deno: deno task ignored
  npm: npm run ignored
`
		dir := setupConfigTestDir(t, content)

		cfg, err := LoadConfig(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("deno task legacy", cfg.Build.DenoBuild) {
			t.Fatalf("want %v, got %v", "deno task legacy", cfg.Build.DenoBuild)
		}
		if !stdlibAssertEqual("npm run legacy", cfg.Build.NpmBuild) {
			t.Fatalf("want %v, got %v", "npm run legacy", cfg.Build.NpmBuild)
		}
		if !stdlibAssertEqual(PreBuild{Deno: "deno task legacy", Npm: "npm run legacy"}, cfg.PreBuild) {
			t.Fatalf("want %v, got %v", PreBuild{Deno: "deno task legacy", Npm: "npm run legacy"}, cfg.PreBuild)
		}

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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("ABC123DEF4", cfg.Apple.TeamID) {
			t.Fatalf("want %v, got %v", "ABC123DEF4", cfg.Apple.TeamID)
		}
		if !stdlibAssertEqual("ai.lthn.core", cfg.Apple.BundleID) {
			t.Fatalf("want %v, got %v", "ai.lthn.core", cfg.Apple.BundleID)
		}
		if !stdlibAssertEqual("universal", cfg.Apple.Arch) {
			t.Fatalf("want %v, got %v", "universal", cfg.Apple.Arch)
		}
		if !stdlibAssertEqual("Developer ID Application: Lethean CIC (ABC123DEF4)", cfg.Apple.CertIdentity) {
			t.Fatalf("want %v, got %v", "Developer ID Application: Lethean CIC (ABC123DEF4)", cfg.Apple.CertIdentity)
		}
		if stdlibAssertNil(cfg.Apple.Sign) {
			t.Fatal("expected non-nil")
		}
		if *cfg.Apple.Sign {
			t.Fatal("expected false")
		}
		if stdlibAssertNil(cfg.Apple.Notarise) {
			t.Fatal("expected non-nil")
		}
		if !(*cfg.Apple.Notarise) {
			t.Fatal("expected true")
		}
		if stdlibAssertNil(cfg.Apple.DMG) {
			t.Fatal("expected non-nil")
		}
		if !(*cfg.Apple.DMG) {
			t.Fatal("expected true")
		}
		if !stdlibAssertEqual(".core/apple/appstore", cfg.Apple.MetadataPath) {
			t.Fatalf("want %v, got %v", ".core/apple/appstore", cfg.Apple.MetadataPath)
		}
		if !stdlibAssertEqual("https://lthn.ai/privacy", cfg.Apple.PrivacyPolicyURL) {
			t.Fatalf("want %v, got %v", "https://lthn.ai/privacy", cfg.Apple.PrivacyPolicyURL)
		}
		if !stdlibAssertEqual("/tmp/AuthKey_TEST.p8", cfg.Apple.APIKeyPath) {
			t.Fatalf("want %v, got %v", "/tmp/AuthKey_TEST.p8", cfg.Apple.APIKeyPath)
		}
		if !stdlibAssertEqual("assets/dmg-background.png", cfg.Apple.DMGBackground) {
			t.Fatalf("want %v, got %v", "assets/dmg-background.png", cfg.Apple.DMGBackground)
		}
		if !stdlibAssertEqual("CoreGUI Release", cfg.Apple.XcodeCloud.Workflow) {
			t.Fatalf("want %v, got %v", "CoreGUI Release", cfg.Apple.XcodeCloud.Workflow)
		}
		if len(cfg.Apple.XcodeCloud.Triggers) != 2 {
			t.Fatalf("want len %v, got %v", 2, len(cfg.Apple.XcodeCloud.Triggers))
		}
		if !stdlibAssertEqual("main", cfg.Apple.XcodeCloud.Triggers[0].Branch) {
			t.Fatalf("want %v, got %v", "main", cfg.Apple.XcodeCloud.Triggers[0].Branch)
		}
		if !stdlibAssertEqual("testflight", cfg.Apple.XcodeCloud.Triggers[0].Action) {
			t.Fatalf("want %v, got %v", "testflight", cfg.Apple.XcodeCloud.Triggers[0].Action)
		}
		if !stdlibAssertEqual("v*", cfg.Apple.XcodeCloud.Triggers[1].Tag) {
			t.Fatalf("want %v, got %v", "v*", cfg.Apple.XcodeCloud.Triggers[1].Tag)
		}
		if !stdlibAssertEqual("appstore", cfg.Apple.XcodeCloud.Triggers[1].Action) {
			t.Fatalf("want %v, got %v", "appstore", cfg.Apple.XcodeCloud.Triggers[1].Action)
		}

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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("core-ml", cfg.LinuxKit.Base) {
			t.Fatalf("want %v, got %v", "core-ml", cfg.LinuxKit.Base)
		}
		if !stdlibAssertEqual([]string{"gh"}, cfg.LinuxKit.Packages) {
			t.Fatalf("want %v, got %v", []string{"gh"}, cfg.LinuxKit.Packages)
		}
		if !stdlibAssertEqual([]string{"/workspace"}, cfg.LinuxKit.Mounts) {
			t.Fatalf("want %v, got %v", []string{"/workspace"}, cfg.LinuxKit.Mounts)
		}
		if !(cfg.LinuxKit.GPU) {
			t.Fatal("expected true")
		}
		if !stdlibAssertEqual([]string{"oci", "apple"}, cfg.LinuxKit.Formats) {
			t.Fatalf("want %v, got %v", []string{"oci", "apple"}, cfg.LinuxKit.Formats)
		}
		if !stdlibAssertEqual("ghcr.io/dappcore", cfg.LinuxKit.Registry) {
			t.Fatalf("want %v, got %v", "ghcr.io/dappcore", cfg.LinuxKit.Registry)
		}

	})

	t.Run("normalizes LinuxKit list values and formats", func(t *testing.T) {
		content := `
version: 1
build:
  formats:
    - " OCI "
    - apple
    - APPLE
linuxkit:
  base: " core-dev "
  packages:
    - " git "
    - git
    - task
  mounts:
    - " /workspace "
    - /workspace
    - /src
  formats:
    - " OCI "
    - apple
    - APPLE
  registry: " ghcr.io/dappcore "
`
		dir := setupConfigTestDir(t, content)

		cfg, err := LoadConfig(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual([]string{"oci", "apple"}, cfg.Build.Formats) {
			t.Fatalf("want %v, got %v", []string{"oci", "apple"}, cfg.Build.Formats)
		}
		if !stdlibAssertEqual("core-dev", cfg.LinuxKit.Base) {
			t.Fatalf("want %v, got %v", "core-dev", cfg.LinuxKit.Base)
		}
		if !stdlibAssertEqual([]string{"git", "task"}, cfg.LinuxKit.Packages) {
			t.Fatalf("want %v, got %v", []string{"git", "task"}, cfg.LinuxKit.Packages)
		}
		if !stdlibAssertEqual([]string{"/workspace", "/src"}, cfg.LinuxKit.Mounts) {
			t.Fatalf("want %v, got %v", []string{"/workspace", "/src"}, cfg.LinuxKit.Mounts)
		}
		if !stdlibAssertEqual([]string{"oci", "apple"}, cfg.LinuxKit.Formats) {
			t.Fatalf("want %v, got %v", []string{"oci", "apple"}, cfg.LinuxKit.Formats)
		}
		if !stdlibAssertEqual("ghcr.io/dappcore", cfg.LinuxKit.Registry) {
			t.Fatalf("want %v, got %v", "ghcr.io/dappcore", cfg.LinuxKit.Registry)
		}

	})

	t.Run("restores default LinuxKit base mounts and formats when expansion resolves empty", func(t *testing.T) {
		t.Setenv("CORE_IMAGE_BASE", "")
		t.Setenv("CORE_IMAGE_MOUNT", "")
		t.Setenv("CORE_IMAGE_FORMAT", "")

		content := `
version: 1
linuxkit:
  base: ${CORE_IMAGE_BASE}
  mounts:
    - ${CORE_IMAGE_MOUNT}
  formats:
    - ${CORE_IMAGE_FORMAT}
`
		dir := setupConfigTestDir(t, content)

		cfg, err := LoadConfig(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("core-dev", cfg.LinuxKit.Base) {
			t.Fatalf("want %v, got %v", "core-dev", cfg.LinuxKit.Base)
		}
		if !stdlibAssertEqual([]string{"/workspace"}, cfg.LinuxKit.Mounts) {
			t.Fatalf("want %v, got %v", []string{"/workspace"}, cfg.LinuxKit.Mounts)
		}
		if !stdlibAssertEqual([]string{"oci", "apple"}, cfg.LinuxKit.Formats) {
			t.Fatalf("want %v, got %v", []string{"oci", "apple"}, cfg.LinuxKit.Formats)
		}

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
  skip_unavailable: true
  diff: true
`
		dir := setupConfigTestDir(t, content)

		cfg, err := LoadConfig(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}
		if stdlibAssertNil(cfg.SDK) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("docs/openapi.yaml", cfg.SDK.Spec) {
			t.Fatalf("want %v, got %v", "docs/openapi.yaml", cfg.SDK.Spec)
		}
		if !stdlibAssertEqual([]string{"typescript"}, cfg.SDK.Languages) {
			t.Fatalf("want %v, got %v", []string{"typescript"}, cfg.SDK.Languages)
		}
		if !stdlibAssertEqual("sdk", cfg.SDK.Output) {
			t.Fatalf("want %v, got %v", "sdk", cfg.SDK.Output)
		}
		if !(cfg.SDK.SkipUnavailable) {
			t.Fatal("expected true")
		}
		if !(cfg.SDK.Diff.Enabled) {
			t.Fatal("expected true")
		}
		if cfg.SDK.Diff.FailOnBreaking {
			t.Fatal("expected false")
		}

	})

	t.Run("preserves explicit empty sdk languages list", func(t *testing.T) {
		content := `
version: 1
sdk:
  languages: []
`
		dir := setupConfigTestDir(t, content)

		cfg, err := LoadConfig(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}
		if stdlibAssertNil(cfg.SDK) {
			t.Fatal("expected non-nil")
		}
		if stdlibAssertNil(cfg.SDK.Languages) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEmpty(cfg.SDK.Languages) {
			t.Fatalf("expected empty, got %v", cfg.SDK.Languages)
		}

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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}
		if cfg.Sign.Windows.Signtool {
			t.Fatal("expected false")
		}
		if !stdlibAssertEqual("C:/certs/core.pfx", cfg.Sign.Windows.Certificate) {
			t.Fatalf("want %v, got %v", "C:/certs/core.pfx", cfg.Sign.Windows.Certificate)
		}

	})
	t.Run("returns defaults when config file missing", func(t *testing.T) {
		dir := t.TempDir()

		cfg, err := LoadConfig(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}

		defaults := DefaultConfig()
		if !stdlibAssertEqual(defaults.Version, cfg.Version) {
			t.Fatalf("want %v, got %v", defaults.Version, cfg.Version)
		}
		if !stdlibAssertEqual(defaults.Project.Main, cfg.Project.Main) {
			t.Fatalf("want %v, got %v", defaults.Project.Main, cfg.Project.Main)
		}
		if !stdlibAssertEqual(defaults.Build.CGO, cfg.Build.CGO) {
			t.Fatalf("want %v, got %v", defaults.Build.CGO, cfg.Build.CGO)
		}
		if !stdlibAssertEqual(defaults.Build.Flags, cfg.Build.Flags) {
			t.Fatalf("want %v, got %v", defaults.Build.Flags, cfg.Build.Flags)
		}
		if !stdlibAssertEqual(defaults.Build.LDFlags, cfg.Build.LDFlags) {
			t.Fatalf("want %v, got %v", defaults.Build.LDFlags, cfg.Build.LDFlags)
		}
		if cfg.Build.Load {
			t.Fatal("expected false")
		}
		if !stdlibAssertEmpty(

			// Explicit values preserved
			cfg.Build.BuildTags) {
			t.Fatalf("expected empty, got %v", cfg.Build.BuildTags)
		}
		if !stdlibAssertEqual(defaults.

			// Defaults applied
			Targets, cfg.Targets) {
			t.Fatalf("want %v, got %v", defaults.Targets, cfg.Targets)
		}

	})

	t.Run("applies defaults for missing fields", func(t *testing.T) {
		content := `
version: 2
project:
  name: partial
`
		dir := setupConfigTestDir(t, content)

		cfg, err := LoadConfig(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual(2, cfg.Version) {
			t.Fatalf("want %v, got %v", 2, cfg.Version)
		}
		if !stdlibAssertEqual("partial", cfg.Project.Name) {
			t.Fatalf("want %v, got %v", "partial", cfg.Project.Name)
		}

		defaults := DefaultConfig()
		if !stdlibAssertEqual(defaults.Project.Main, cfg.Project.Main) {
			t.Fatalf("want %v, got %v", defaults.Project.Main, cfg.Project.Main)
		}
		if !stdlibAssertEqual(defaults.Build.Flags, cfg.Build.Flags) {
			t.Fatalf("want %v, got %v", defaults.Build.Flags, cfg.Build.Flags)
		}
		if !stdlibAssertEqual(defaults.Build.LDFlags, cfg.Build.LDFlags) {
			t.Fatalf("want %v, got %v", defaults.Build.LDFlags, cfg.Build.LDFlags)
		}
		if !stdlibAssertEqual(defaults.Targets, cfg.Targets) {
			t.Fatalf("want %v, got %v", defaults.Targets, cfg.Targets)
		}
		if !(cfg.Sign.Enabled) {
			t.Fatal("expected true")
		}

	})

	t.Run("preserves explicit signing disablement", func(t *testing.T) {
		content := `
version: 1
sign:
  enabled: false
`
		dir := setupConfigTestDir(t, content)

		cfg, err := LoadConfig(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}
		if cfg.Sign.Enabled {
			t.Fatal("expected false")
		}

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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(

			// Empty arrays are preserved (not replaced with defaults)
			cfg) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEmpty(cfg.Build.Flags) {
			t.Fatalf("expected empty, got %v", cfg.Build.Flags)
		}
		if !stdlibAssertEmpty(cfg.Build.LDFlags) {

			// Targets explicitly set
			t.Fatalf("expected empty, got %v", cfg.Build.LDFlags)
		}
		if !stdlibAssertEmpty(cfg.Build.BuildTags) {
			t.Fatalf("expected empty, got %v", cfg.Build.BuildTags)
		}
		if len(cfg.Targets) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(cfg.Targets))
		}

	})
}

func TestConfig_MarshalYAML_Good(t *testing.T) {
	type marshalledBuildConfig struct {
		Build    map[string]any `yaml:"build"`
		Cache    map[string]any `yaml:"cache"`
		PreBuild map[string]any `yaml:"pre_build"`
	}

	t.Run("emits the RFC top-level cache block", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Project.Name = "demo"
		cfg.Build.Cache = CacheConfig{
			Enabled:     true,
			Directory:   ".core/cache",
			KeyPrefix:   "demo",
			Paths:       []string{"cache/go-build", "cache/go-mod"},
			RestoreKeys: []string{"go-"},
		}

		data, err := yaml.Marshal(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var decoded marshalledBuildConfig
		if err := yaml.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(decoded.Cache) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual(true, decoded.Cache["enabled"]) {
			t.Fatalf("want %v, got %v", true, decoded.Cache["enabled"])
		}
		if !stdlibAssertEqual(".core/cache", decoded.Cache["dir"]) {
			t.Fatalf("want %v, got %v", ".core/cache", decoded.Cache["dir"])
		}
		if !stdlibAssertEqual("demo", decoded.Cache["key_prefix"]) {
			t.Fatalf("want %v, got %v", "demo", decoded.Cache["key_prefix"])
		}

		_, hasNestedCache := decoded.Build["cache"]
		if hasNestedCache {
			t.Fatal("expected false")
		}

	})

	t.Run("omits cache when it is not configured", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Build.Cache = CacheConfig{}

		data, err := yaml.Marshal(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var decoded marshalledBuildConfig
		if err := yaml.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertNil(decoded.Cache) {
			t.Fatalf("expected nil, got %v", decoded.Cache)
		}

		_, hasNestedCache := decoded.Build["cache"]
		if hasNestedCache {
			t.Fatal("expected false")
		}

	})

	t.Run("emits the RFC pre_build block instead of legacy build hooks", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Build.DenoBuild = "deno task build"
		cfg.Build.NpmBuild = "npm run build"
		cfg.PreBuild = PreBuild{
			Deno: "deno task build",
			Npm:  "npm run build",
		}

		data, err := yaml.Marshal(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var decoded marshalledBuildConfig
		if err := yaml.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(decoded.PreBuild) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("deno task build", decoded.PreBuild["deno"]) {
			t.Fatalf("want %v, got %v", "deno task build", decoded.PreBuild["deno"])
		}
		if !stdlibAssertEqual("npm run build", decoded.PreBuild["npm"]) {
			t.Fatalf("want %v, got %v", "npm run build", decoded.PreBuild["npm"])
		}

		_, hasLegacyDeno := decoded.Build["deno_build"]
		_, hasLegacyNpm := decoded.Build["npm_build"]
		if hasLegacyDeno {
			t.Fatal("expected false")
		}
		if hasLegacyNpm {
			t.Fatal("expected false")
		}

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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cfg, err := LoadConfigAtPath(fs, configPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual(3, cfg.Version) {
			t.Fatalf("want %v, got %v", 3, cfg.Version)
		}
		if !stdlibAssertEqual("custom-app", cfg.Project.Name) {
			t.Fatalf("want %v, got %v", "custom-app", cfg.Project.Name)
		}
		if !stdlibAssertEqual("custom-app", cfg.Project.Binary) {
			t.Fatalf("want %v, got %v", "custom-app", cfg.Project.Binary)
		}
		if !(cfg.Build.CGO) {
			t.Fatal("expected true")
		}
		if !stdlibAssertEmpty(cfg.Build.BuildTags) {
			t.Fatalf("expected empty, got %v", cfg.Build.BuildTags)
		}
		if len(cfg.Targets) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(cfg.Targets))
		}
		if !stdlibAssertEqual("linux", cfg.Targets[0].OS) {
			t.Fatalf("want %v, got %v", "linux", cfg.Targets[0].OS)
		}
		if !stdlibAssertEqual("amd64", cfg.Targets[0].Arch) {
			t.Fatalf("want %v, got %v", "amd64", cfg.Targets[0].Arch)
		}

	})

	t.Run("defaults to the local medium when nil is passed", func(t *testing.T) {
		dir := t.TempDir()
		configPath := ax.Join(dir, "custom-build.yaml")
		content := `
version: 1
project:
  name: explicit-nil-medium
`
		err := ax.WriteFile(configPath, []byte(content), 0o644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cfg, err := LoadConfigAtPath(nil, configPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("explicit-nil-medium", cfg.Project.Name) {
			t.Fatalf("want %v, got %v", "explicit-nil-medium", cfg.Project.Name)
		}

	})
}

func TestConfig_ConfigExistsNilMedium_Good(t *testing.T) {
	t.Run("returns false for a nil medium", func(t *testing.T) {
		if ConfigExists(nil, t.TempDir()) {
			t.Fatal("expected false")
		}

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
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertNil(cfg) {
			t.Fatalf("expected nil, got %v", cfg)
		}
		if !stdlibAssertContains(err.Error(), "failed to parse config file") {
			t.Fatalf("expected %v to contain %v", err.Error(), "failed to parse config file")
		}

	})

	t.Run("returns error for unreadable file", func(t *testing.T) {
		dir := t.TempDir()
		coreDir := ax.Join(dir, ConfigDir)
		err := ax.MkdirAll(coreDir, 0755)
		if err != nil {
			t.Fatalf("unexpected error: %v",

				// Create config as a directory instead of file
				err)
		}

		configPath := ax.Join(coreDir, ConfigFileName)
		err = ax.Mkdir(configPath, 0755)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cfg, err := LoadConfig(fs, dir)
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertNil(cfg) {
			t.Fatalf("expected nil, got %v", cfg)
		}
		if !stdlibAssertContains(err.Error(), "failed to read config file") {
			t.Fatalf("expected %v to contain %v", err.Error(), "failed to read config file")
		}

	})
}

func TestConfig_DefaultConfig_Good(t *testing.T) {
	t.Run("returns sensible defaults", func(t *testing.T) {
		cfg := DefaultConfig()
		if !stdlibAssertEqual(1, cfg.Version) {
			t.Fatalf("want %v, got %v", 1, cfg.Version)
		}
		if !stdlibAssertEqual(".", cfg.Project.Main) {
			t.Fatalf("want %v, got %v", ".", cfg.Project.Main)
		}
		if !stdlibAssertEmpty(cfg.Project.Name) {
			t.Fatalf("expected empty, got %v", cfg.Project.Name)
		}
		if !stdlibAssertEmpty(cfg.Project.Binary) {
			t.Fatalf("expected empty, got %v", cfg.Project.Binary)
		}
		if cfg.Build.CGO {
			t.Fatal("expected false")
		}
		if !stdlibAssertContains(cfg.Build.Flags, "-trimpath") {
			t.Fatalf("expected %v to contain %v", cfg.Build.Flags, "-trimpath")
		}
		if !stdlibAssertContains(cfg.

			// Default targets cover common platforms
			Build.LDFlags, "-s") {
			t.Fatalf("expected %v to contain %v", cfg.Build.LDFlags, "-s")
		}
		if !stdlibAssertContains(cfg.Build.LDFlags, "-w") {
			t.Fatalf("expected %v to contain %v", cfg.Build.LDFlags, "-w")
		}
		if !stdlibAssertEmpty(cfg.Build.Env) {
			t.Fatalf("expected empty, got %v", cfg.Build.Env)
		}
		if !stdlibAssertEqual("core-dev", cfg.LinuxKit.Base) {
			t.Fatalf("want %v, got %v", "core-dev", cfg.LinuxKit.Base)
		}
		if !stdlibAssertEqual([]string{"/workspace"}, cfg.LinuxKit.Mounts) {
			t.Fatalf("want %v, got %v", []string{"/workspace"}, cfg.LinuxKit.Mounts)
		}
		if !stdlibAssertEqual([]string{"oci", "apple"}, cfg.LinuxKit.Formats) {
			t.Fatalf("want %v, got %v", []string{"oci", "apple"}, cfg.LinuxKit.Formats)
		}
		if len(cfg.Targets) != 5 {
			t.Fatalf("want len %v, got %v", 5, len(cfg.Targets))
		}

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
		if !(hasLinuxAmd64) {
			t.Fatal("expected true")
		}
		if !(hasDarwinAmd64) {
			t.Fatal("expected true")
		}
		if !(hasDarwinArm64) {
			t.Fatal("expected true")
		}
		if !(hasWindowsAmd64) {
			t.Fatal("expected true")
		}

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
	if stdlibAssertNil(clone) {
		t.Fatal("expected non-nil")
	}

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
	if !stdlibAssertEqual([]string{"-trimpath"}, cfg.Build.Flags) {
		t.Fatalf("want %v, got %v", []string{"-trimpath"}, cfg.Build.Flags)
	}
	if !stdlibAssertEqual([]string{"-s", "-w"}, cfg.Build.LDFlags) {
		t.Fatalf("want %v, got %v", []string{"-s", "-w"}, cfg.Build.LDFlags)
	}
	if !stdlibAssertEqual([]string{"integration"}, cfg.Build.BuildTags) {
		t.Fatalf("want %v, got %v", []string{"integration"}, cfg.Build.BuildTags)
	}
	if !stdlibAssertEqual([]string{"FOO=bar"}, cfg.Build.Env) {
		t.Fatalf("want %v, got %v", []string{"FOO=bar"}, cfg.Build.Env)
	}
	if !stdlibAssertEqual([]string{"cache/go-build"}, cfg.Build.Cache.Paths) {
		t.Fatalf("want %v, got %v", []string{"cache/go-build"}, cfg.Build.Cache.Paths)
	}
	if !stdlibAssertEqual([]string{"main"}, cfg.Build.Cache.RestoreKeys) {
		t.Fatalf("want %v, got %v", []string{"main"}, cfg.Build.Cache.RestoreKeys)
	}
	if !stdlibAssertEqual([]string{"latest"}, cfg.Build.Tags) {
		t.Fatalf("want %v, got %v", []string{"latest"}, cfg.Build.Tags)
	}
	if !stdlibAssertEqual(map[string]string{"VERSION": "v1.2.3"}, cfg.Build.BuildArgs) {
		t.Fatalf("want %v, got %v", map[string]string{"VERSION": "v1.2.3"}, cfg.Build.BuildArgs)
	}
	if !stdlibAssertEqual([]string{"iso"}, cfg.Build.Formats) {
		t.Fatalf("want %v, got %v", []string{"iso"}, cfg.Build.Formats)
	}
	if !stdlibAssertEqual("core-dev", cfg.LinuxKit.Base) {
		t.Fatalf("want %v, got %v", "core-dev", cfg.LinuxKit.Base)
	}
	if !stdlibAssertEqual([]string{"git"}, cfg.LinuxKit.Packages) {
		t.Fatalf("want %v, got %v", []string{"git"}, cfg.LinuxKit.Packages)
	}
	if !stdlibAssertEqual([]string{"/workspace"}, cfg.LinuxKit.Mounts) {
		t.Fatalf("want %v, got %v", []string{"/workspace"}, cfg.LinuxKit.Mounts)
	}
	if !stdlibAssertEqual([]string{"oci", "apple"}, cfg.LinuxKit.Formats) {
		t.Fatalf("want %v, got %v", []string{"oci", "apple"}, cfg.LinuxKit.Formats)
	}
	if !stdlibAssertEqual("ghcr.io/dappcore", cfg.LinuxKit.Registry) {
		t.Fatalf("want %v, got %v", "ghcr.io/dappcore", cfg.LinuxKit.Registry)
	}
	if stdlibAssertNil(cfg.Apple.Sign) {
		t.Fatal("expected non-nil")
	}
	if stdlibAssertNil(cfg.Apple.Notarise) {
		t.Fatal("expected non-nil")
	}
	if stdlibAssertNil(cfg.Apple.DMG) {
		t.Fatal("expected non-nil")
	}
	if !(*cfg.Apple.Sign) {
		t.Fatal("expected true")
	}
	if *cfg.Apple.Notarise {
		t.Fatal("expected false")
	}
	if !(*cfg.Apple.DMG) {
		t.Fatal("expected true")
	}
	if len(cfg.Apple.XcodeCloud.Triggers) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(cfg.Apple.XcodeCloud.Triggers))
	}
	if !stdlibAssertEqual("main", cfg.Apple.XcodeCloud.Triggers[0].Branch) {
		t.Fatalf("want %v, got %v", "main", cfg.Apple.XcodeCloud.Triggers[0].Branch)
	}
	if stdlibAssertNil(cfg.SDK) {
		t.Fatal("expected non-nil")
	}
	if !stdlibAssertEqual([]string{"typescript"}, cfg.SDK.Languages) {
		t.Fatalf("want %v, got %v", []string{"typescript"}, cfg.SDK.Languages)
	}
	if !stdlibAssertEqual("generated/sdk", cfg.SDK.Output) {
		t.Fatalf("want %v, got %v", "generated/sdk", cfg.SDK.Output)
	}
	if !stdlibAssertEqual([]TargetConfig{{OS: "linux", Arch: "amd64"}}, cfg.Targets) {
		t.Fatalf("want %v, got %v", []TargetConfig{{OS: "linux", Arch: "amd64"}}, cfg.Targets)
	}

}

func TestConfig_ConfigPath_Good(t *testing.T) {
	t.Run("returns correct path", func(t *testing.T) {
		path := ConfigPath("/project/root")
		if !stdlibAssertEqual("/project/root/.core/build.yaml", path) {
			t.Fatalf("want %v, got %v", "/project/root/.core/build.yaml", path)
		}

	})
}

func TestConfig_ConfigExists_Good(t *testing.T) {
	fs := io.Local
	t.Run("returns true when config exists", func(t *testing.T) {
		dir := setupConfigTestDir(t, "version: 1")
		if !(ConfigExists(fs, dir)) {
			t.Fatal("expected true")
		}

	})

	t.Run("returns false when config missing", func(t *testing.T) {
		dir := t.TempDir()
		if ConfigExists(fs, dir) {
			t.Fatal("expected false")
		}

	})

	t.Run("returns false when .core dir missing", func(t *testing.T) {
		dir := t.TempDir()
		if ConfigExists(fs, dir) {
			t.Fatal("expected false")
		}

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
		if len(targets) != 3 {
			t.Fatalf("want len %v, got %v", 3, len(targets))
		}
		if !stdlibAssertEqual(Target{OS: "linux", Arch: "amd64"}, targets[0]) {
			t.Fatalf("want %v, got %v", Target{OS: "linux", Arch: "amd64"}, targets[0])
		}
		if !stdlibAssertEqual(Target{OS: "darwin", Arch: "arm64"}, targets[1]) {
			t.Fatalf("want %v, got %v", Target{OS: "darwin", Arch: "arm64"}, targets[1])
		}
		if !stdlibAssertEqual(Target{OS: "windows", Arch: "386"}, targets[2]) {
			t.Fatalf("want %v, got %v",

				// TestLoadConfig_Testdata tests loading from the testdata fixture.
				Target{OS: "windows", Arch: "386"}, targets[2])
		}

	})

	t.Run("returns empty slice for no targets", func(t *testing.T) {
		cfg := &BuildConfig{
			Targets: []TargetConfig{},
		}

		targets := cfg.ToTargets()
		if !stdlibAssertEmpty(targets) {
			t.Fatalf("expected empty, got %v", targets)
		}

	})
}

func TestConfig_LoadConfigTestdata_Good(t *testing.T) {
	fs := io.Local
	abs, err := ax.Abs("testdata/config-project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("loads config-project fixture", func(t *testing.T) {
		cfg, err := LoadConfig(fs, abs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual(1, cfg.Version) {
			t.Fatalf("want %v, got %v", 1, cfg.Version)
		}
		if !stdlibAssertEqual("example-cli", cfg.Project.Name) {
			t.Fatalf("want %v, got %v", "example-cli", cfg.Project.Name)
		}
		if !stdlibAssertEqual("An example CLI application", cfg.Project.Description) {
			t.Fatalf("want %v, got %v", "An example CLI application", cfg.Project.Description)
		}
		if !stdlibAssertEqual("./cmd/example", cfg.Project.Main) {
			t.Fatalf("want %v, got %v", "./cmd/example", cfg.Project.Main)
		}
		if !stdlibAssertEqual("example", cfg.Project.Binary) {
			t.Fatalf("want %v, got %v", "example", cfg.Project.Binary)
		}
		if cfg.Build.CGO {
			t.Fatal("expected false")
		}
		if !stdlibAssertEqual([]string{"-trimpath"}, cfg.Build.Flags) {
			t.Fatalf("want %v, got %v", []string{"-trimpath"}, cfg.Build.Flags)
		}
		if !stdlibAssertEqual([]string{"-s", "-w"}, cfg.Build.LDFlags) {
			t.Fatalf("want %v, got %v", []string{"-s", "-w"}, cfg.Build.LDFlags)
		}
		if len(cfg.Targets) != 3 {
			t.Fatalf("want len %v, got %v", 3, len(cfg.Targets))
		}

	})
}

func stdlibAssertEqual(want, got any) bool {
	return reflect.DeepEqual(want, got)
}

func stdlibAssertNil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

func stdlibAssertEmpty(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	default:
		return v.IsZero()
	}
}

func stdlibAssertZero(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	return !v.IsValid() || v.IsZero()
}

func stdlibAssertContains(container, elem any) bool {
	if s, ok := container.(string); ok {
		sub, ok := elem.(string)
		return ok && strings.Contains(s, sub)
	}

	v := reflect.ValueOf(container)
	if !v.IsValid() {
		return false
	}
	switch v.Kind() {
	case reflect.Map:
		key := reflect.ValueOf(elem)
		if !key.IsValid() {
			return false
		}
		if key.Type().AssignableTo(v.Type().Key()) {
			return v.MapIndex(key).IsValid()
		}
		if key.Type().ConvertibleTo(v.Type().Key()) {
			return v.MapIndex(key.Convert(v.Type().Key())).IsValid()
		}
	case reflect.Array, reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if reflect.DeepEqual(v.Index(i).Interface(), elem) {
				return true
			}
		}
	}
	return false
}

func stdlibAssertElementsMatch(want, got any) bool {
	wantValue := reflect.ValueOf(want)
	gotValue := reflect.ValueOf(got)
	if !wantValue.IsValid() || !gotValue.IsValid() {
		return !wantValue.IsValid() && !gotValue.IsValid()
	}
	if !isListValue(wantValue) || !isListValue(gotValue) {
		return reflect.DeepEqual(want, got)
	}
	if wantValue.Len() != gotValue.Len() {
		return false
	}

	used := make([]bool, gotValue.Len())
	for i := 0; i < wantValue.Len(); i++ {
		found := false
		wantElem := wantValue.Index(i).Interface()
		for j := 0; j < gotValue.Len(); j++ {
			if used[j] {
				continue
			}
			if reflect.DeepEqual(wantElem, gotValue.Index(j).Interface()) {
				used[j] = true
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func isListValue(value reflect.Value) bool {
	return value.Kind() == reflect.Array || value.Kind() == reflect.Slice
}
