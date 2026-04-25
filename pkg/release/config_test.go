package release

import (
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/io"
)

// setupConfigTestDir creates a temp directory with optional .core/release.yaml content.
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
	t.Run("loads valid config", func(t *testing.T) {
		content := `
version: 1
project:
  name: myapp
  repository: owner/repo
build:
  targets:
    - os: linux
      arch: amd64
    - os: darwin
      arch: arm64
  archive_format: xz
publishers:
  - type: github
    prerelease: true
    draft: false
changelog:
  include:
    - feat
    - fix
  exclude:
    - chore
`
		dir := setupConfigTestDir(t, content)

		cfg, err := LoadConfig(dir)
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
		if !stdlibAssertEqual("owner/repo", cfg.Project.Repository) {
			t.Fatalf("want %v, got %v", "owner/repo", cfg.Project.Repository)
		}
		if len(cfg.Build.Targets) != 2 {
			t.Fatalf("want len %v, got %v", 2, len(cfg.Build.Targets))
		}
		if !stdlibAssertEqual("xz", cfg.Build.ArchiveFormat) {
			t.Fatalf("want %v, got %v", "xz", cfg.Build.ArchiveFormat)
		}
		if !stdlibAssertEqual("linux", cfg.Build.Targets[0].OS) {
			t.Fatalf("want %v, got %v", "linux", cfg.Build.Targets[0].OS)
		}
		if !stdlibAssertEqual("amd64", cfg.Build.Targets[0].Arch) {
			t.Fatalf("want %v, got %v", "amd64", cfg.Build.Targets[0].Arch)
		}
		if !stdlibAssertEqual("darwin", cfg.Build.Targets[1].OS) {
			t.Fatalf("want %v, got %v", "darwin", cfg.Build.Targets[1].OS)
		}
		if !stdlibAssertEqual("arm64", cfg.Build.Targets[1].Arch) {
			t.Fatalf("want %v, got %v", "arm64", cfg.Build.Targets[1].Arch)
		}
		if len(cfg.Publishers) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(cfg.Publishers))
		}
		if !stdlibAssertEqual("github", cfg.Publishers[0].Type) {
			t.Fatalf("want %v, got %v", "github", cfg.Publishers[0].Type)
		}
		if !(cfg.Publishers[0].Prerelease) {
			t.Fatal("expected true")
		}
		if cfg.Publishers[0].Draft {
			t.Fatal("expected false")
		}
		if !stdlibAssertEqual([]string{"feat", "fix"}, cfg.Changelog.Include) {
			t.Fatalf("want %v, got %v", []string{"feat", "fix"}, cfg.Changelog.Include)
		}
		if !stdlibAssertEqual([]string{"chore"}, cfg.Changelog.Exclude) {

			// Explicit values preserved
			t.Fatalf("want %v, got %v", []string{"chore"}, cfg.Changelog.Exclude)
		}

	})

	t.Run("returns defaults when config file missing", func(t *testing.T) {
		dir := t.TempDir()

		cfg, err := LoadConfig(dir)
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
		if !stdlibAssertEqual(defaults.Build.Targets, cfg.Build.Targets) {
			t.Fatalf("want %v, got %v", defaults.Build.Targets, cfg.Build.Targets)
		}
		if !stdlibAssertEqual(defaults.Publishers, cfg.Publishers) {
			t.Fatalf("want %v, got %v", defaults.Publishers, cfg.Publishers)
		}
		if !stdlibAssertEqual(defaults.Changelog.Include, cfg.Changelog.Include) {
			t.Fatalf("want %v, got %v", defaults.Changelog.Include, cfg.Changelog.Include)
		}
		if !stdlibAssertEqual(defaults.Changelog.Exclude, cfg.Changelog.Exclude) {
			t.Fatalf("want %v, got %v", defaults.Changelog.Exclude, cfg.Changelog.Exclude)
		}

	})

	t.Run("applies defaults for missing fields", func(t *testing.T) {
		content := `
version: 2
project:
  name: partial
`
		dir := setupConfigTestDir(t, content)

		cfg, err := LoadConfig(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual(2, cfg.Version) {
			t.Fatalf("want %v, got %v", 2, cfg.Version)

			// Defaults applied to release-only fields while build targets stay unset so
			// the release pipeline can inherit them from .core/build.yaml.
		}
		if !stdlibAssertEqual("partial", cfg.Project.Name) {
			t.Fatalf("want %v, got %v", "partial", cfg.Project.Name)
		}

		defaults := DefaultConfig()
		if !stdlibAssertEmpty(cfg.Build.Targets) {
			t.Fatalf("expected empty, got %v", cfg.Build.Targets)
		}
		if !stdlibAssertEqual(defaults.Publishers, cfg.Publishers) {
			t.Fatalf("want %v, got %v", defaults.Publishers, cfg.Publishers)
		}

	})

	t.Run("sets project directory on load", func(t *testing.T) {
		dir := setupConfigTestDir(t, "version: 1")

		cfg, err := LoadConfig(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(dir, cfg.projectDir) {
			t.Fatalf("want %v, got %v", dir, cfg.projectDir)
		}

	})

	t.Run("loads sdk config with shorthand diff and defaults", func(t *testing.T) {
		content := `
version: 1
sdk:
  spec: docs/openapi.yaml
  skip_unavailable: true
  diff: true
`
		dir := setupConfigTestDir(t, content)

		cfg, err := LoadConfig(dir)
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
		if !stdlibAssertEqual([]string{"typescript", "python", "go", "php"}, cfg.SDK.Languages) {
			t.Fatalf("want %v, got %v", []string{"typescript", "python", "go", "php"}, cfg.SDK.Languages)
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

		cfg, err := LoadConfig(dir)
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

	t.Run("loads checksum config", func(t *testing.T) {
		content := `
version: 1
checksum:
  algorithm: sha256
  file: checksums.txt
`
		dir := setupConfigTestDir(t, content)

		cfg, err := LoadConfig(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("sha256", cfg.Checksum.Algorithm) {
			t.Fatalf("want %v, got %v", "sha256", cfg.Checksum.Algorithm)
		}
		if !stdlibAssertEqual("checksums.txt", cfg.Checksum.File) {
			t.Fatalf("want %v, got %v", "checksums.txt", cfg.Checksum.File)
		}

	})

	t.Run("loads config from a custom medium", func(t *testing.T) {
		medium := io.NewMemoryMedium()
		dir := "project"
		configPath := ConfigPath(dir)
		if err := medium.EnsureDir(ax.Dir(configPath)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := medium.Write(configPath, `
version: 1
project:
  name: medium-app
  repository: owner/medium-app
sdk:
  spec: docs/openapi.yaml
  languages: [go]
`); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cfg, err := LoadConfigWithMedium(medium, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("medium-app", cfg.Project.Name) {
			t.Fatalf("want %v, got %v", "medium-app", cfg.Project.Name)
		}
		if !stdlibAssertEqual("owner/medium-app", cfg.Project.Repository) {
			t.Fatalf("want %v, got %v", "owner/medium-app", cfg.Project.Repository)
		}
		if stdlibAssertNil(cfg.SDK) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("docs/openapi.yaml", cfg.SDK.Spec) {
			t.Fatalf("want %v, got %v", "docs/openapi.yaml", cfg.SDK.Spec)
		}
		if !stdlibAssertEqual([]string{"go"}, cfg.SDK.Languages) {
			t.Fatalf("want %v, got %v", []string{"go"}, cfg.SDK.Languages)
		}
		if !stdlibAssertEqual(dir, cfg.projectDir) {
			t.Fatalf("want %v, got %v", dir, cfg.projectDir)
		}

	})

	t.Run("returns defaults from a custom medium when config is missing", func(t *testing.T) {
		dir := "virtual-project"

		cfg, err := LoadConfigWithMedium(io.NewMemoryMedium(), dir)
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
		if !stdlibAssertEqual(defaults.Publishers, cfg.Publishers) {
			t.Fatalf("want %v, got %v", defaults.Publishers, cfg.Publishers)
		}
		if !stdlibAssertEqual(dir, cfg.projectDir) {
			t.Fatalf("want %v, got %v", dir, cfg.projectDir)
		}

	})
}

func TestConfig_LoadConfig_ExpandEnv_Good(t *testing.T) {
	t.Setenv("RELEASE_REPO", "owner/release-app")
	t.Setenv("RELEASE_ARCHIVE", "xz")
	t.Setenv("RELEASE_TARGET_OS", "darwin")
	t.Setenv("RELEASE_TARGET_ARCH", "arm64")
	t.Setenv("HOMEBREW_TAP", "owner/homebrew-tap")
	t.Setenv("SDK_SPEC", "docs/openapi.yaml")
	t.Setenv("SDK_OUTPUT", "generated/sdk")
	t.Setenv("SDK_LANGUAGE", "typescript")
	t.Setenv("CHECKSUM_FILE", "dist/checksums.txt")

	content := `
version: 1
project:
  name: release-app
  repository: $RELEASE_REPO
build:
  archive_format: $RELEASE_ARCHIVE
  targets:
    - os: $RELEASE_TARGET_OS
      arch: $RELEASE_TARGET_ARCH
publishers:
  - type: homebrew
    tap: $HOMEBREW_TAP
sdk:
  spec: $SDK_SPEC
  languages:
    - $SDK_LANGUAGE
  output: $SDK_OUTPUT
checksum:
  file: $CHECKSUM_FILE
`
	dir := setupConfigTestDir(t, content)

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdlibAssertNil(cfg) {
		t.Fatal("expected non-nil")
	}
	if !stdlibAssertEqual("owner/release-app", cfg.Project.Repository) {
		t.Fatalf("want %v, got %v", "owner/release-app", cfg.Project.Repository)
	}
	if !stdlibAssertEqual("xz", cfg.Build.ArchiveFormat) {
		t.Fatalf("want %v, got %v", "xz", cfg.Build.ArchiveFormat)
	}
	if len(cfg.Build.Targets) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(cfg.Build.Targets))
	}
	if !stdlibAssertEqual("darwin", cfg.Build.Targets[0].OS) {
		t.Fatalf("want %v, got %v", "darwin", cfg.Build.Targets[0].OS)
	}
	if !stdlibAssertEqual("arm64", cfg.Build.Targets[0].Arch) {
		t.Fatalf("want %v, got %v", "arm64", cfg.Build.Targets[0].Arch)
	}
	if len(cfg.Publishers) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(cfg.Publishers))
	}
	if !stdlibAssertEqual("owner/homebrew-tap", cfg.Publishers[0].Tap) {
		t.Fatalf("want %v, got %v", "owner/homebrew-tap", cfg.Publishers[0].Tap)
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
	if !stdlibAssertEqual("generated/sdk",

		// Create config as a directory instead of file
		cfg.SDK.Output) {
		t.Fatalf("want %v, got %v", "generated/sdk", cfg.SDK.Output)
	}
	if !stdlibAssertEqual("dist/checksums.txt", cfg.Checksum.File) {
		t.Fatalf("want %v, got %v", "dist/checksums.txt", cfg.Checksum.File)
	}

}

func TestConfig_LoadConfig_Bad(t *testing.T) {
	t.Run("returns error for invalid YAML", func(t *testing.T) {
		content := `
version: 1
project:
  name: [invalid yaml
`
		dir := setupConfigTestDir(t, content)

		cfg, err := LoadConfig(dir)
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

	t.Run("returns default config when config path is a directory", func(t *testing.T) {
		dir := t.TempDir()
		coreDir := ax.Join(dir, ConfigDir)
		err := ax.MkdirAll(coreDir, 0755)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		configPath := ax.Join(coreDir, ConfigFileName)
		err = ax.Mkdir(configPath, 0755)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cfg, err := LoadConfig(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual(1, cfg.Version) {
			t.Fatalf("want %v, got %v", 1, cfg.Version)
		}
		if !stdlibAssertEqual(dir, cfg.projectDir) {
			t.Fatalf("want %v, got %v", dir, cfg.projectDir)
		}

	})
}

func TestConfig_DefaultConfig_Good(t *testing.T) {
	t.Run("returns sensible defaults", func(t *testing.T) {
		cfg := DefaultConfig()
		if !stdlibAssertEqual(1, cfg.Version) {
			t.Fatalf("want %v, got %v", 1, cfg.Version)
		}
		if !stdlibAssertEmpty(cfg.Project.Name) {
			t.

				// Default targets
				Fatalf("expected empty, got %v", cfg.Project.Name)
		}
		if !stdlibAssertEmpty(cfg.Project.Repository) {
			t.Fatalf("expected empty, got %v", cfg.Project.Repository)
		}
		if len(cfg.Build.Targets) != 5 {
			t.Fatalf("want len %v, got %v", 5, len(cfg.Build.Targets))
		}

		hasLinuxAmd64 := false
		hasDarwinAmd64 := false
		hasDarwinArm64 := false
		hasWindowsAmd64 := false
		for _, target := range cfg.Build.Targets {
			if target.OS == "linux" && target.Arch == "amd64" {
				hasLinuxAmd64 = true
			}
			if target.OS == "darwin" && target.Arch == "amd64" {
				hasDarwinAmd64 = true
			}
			if target.OS == "darwin" && target.Arch == "arm64" {
				hasDarwinArm64 = true
			}
			if target.OS == "windows" && target.Arch == "amd64" {
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
		if !

		// Default publisher
		(hasWindowsAmd64) {
			t.Fatal("expected true")
		}
		if len(cfg.Publishers) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(cfg.Publishers))
		}
		if !stdlibAssertEqual("github", cfg.Publishers[0].Type) {
			t.Fatalf("want %v, got %v",

				// Default changelog settings
				"github", cfg.Publishers[0].Type)
		}
		if cfg.Publishers[0].Prerelease {
			t.Fatal("expected false")
		}
		if cfg.Publishers[0].Draft {
			t.Fatal("expected false")
		}
		if !stdlibAssertEqual("conventional", cfg.Changelog.Use) {
			t.Fatalf("want %v, got %v", "conventional", cfg.Changelog.Use)
		}
		if !stdlibAssertContains(cfg.Changelog.Include, "feat") {
			t.Fatalf("expected %v to contain %v", cfg.Changelog.Include, "feat")
		}
		if !stdlibAssertContains(cfg.Changelog.Include, "fix") {
			t.Fatalf("expected %v to contain %v", cfg.Changelog.Include, "fix")
		}
		if !stdlibAssertContains(cfg.Changelog.Exclude, "chore") {
			t.Fatalf("expected %v to contain %v", cfg.Changelog.Exclude, "chore")
		}
		if !stdlibAssertContains(cfg.Changelog.Exclude, "docs") {
			t.Fatalf("expected %v to contain %v", cfg.Changelog.Exclude, "docs")
		}
		if !stdlibAssertEqual("sha256", cfg.Checksum.Algorithm) {
			t.Fatalf("want %v, got %v", "sha256", cfg.Checksum.Algorithm)
		}
		if !stdlibAssertEqual("CHECKSUMS.txt", cfg.Checksum.File) {
			t.Fatalf("want %v, got %v", "CHECKSUMS.txt", cfg.Checksum.File)
		}

	})
}

func TestConfig_ScaffoldConfig_Good(t *testing.T) {
	t.Run("returns documented init scaffold", func(t *testing.T) {
		cfg := ScaffoldConfig()
		if stdlibAssertNil(cfg.SDK) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("api/openapi.yaml", cfg.SDK.Spec) {
			t.Fatalf("want %v, got %v", "api/openapi.yaml", cfg.SDK.Spec)
		}
		if !stdlibAssertEqual([]string{"typescript", "python", "go", "php"}, cfg.SDK.Languages) {
			t.Fatalf("want %v, got %v", []string{"typescript", "python", "go", "php"}, cfg.SDK.Languages)
		}
		if !stdlibAssertEqual("sdk", cfg.SDK.Output) {
			t.Fatalf("want %v, got %v", "sdk", cfg.SDK.Output)
		}
		if !(cfg.SDK.Diff.Enabled) {
			t.Fatal("expected true")
		}
		if cfg.SDK.Diff.FailOnBreaking {
			t.Fatal("expected false")
		}

	})
}

func TestConfig_ConfigPath_Good(t *testing.T) {
	t.Run("returns correct path", func(t *testing.T) {
		path := ConfigPath("/project/root")
		if !stdlibAssertEqual("/project/root/.core/release.yaml", path) {
			t.Fatalf("want %v, got %v", "/project/root/.core/release.yaml", path)
		}

	})
}

func TestConfig_ConfigExists_Good(t *testing.T) {
	t.Run("returns true when config exists", func(t *testing.T) {
		dir := setupConfigTestDir(t, "version: 1")
		if !(ConfigExists(dir)) {
			t.Fatal("expected true")
		}

	})

	t.Run("returns false when config missing", func(t *testing.T) {
		dir := t.TempDir()
		if ConfigExists(dir) {
			t.Fatal("expected false")
		}

	})

	t.Run("returns false when .core dir missing", func(t *testing.T) {
		dir := t.TempDir()
		if ConfigExists(dir) {
			t.Fatal("expected false")
		}

	})
}

func TestConfig_WriteConfig_Good(t *testing.T) {
	t.Run("writes config to file", func(t *testing.T) {
		dir := t.TempDir()

		cfg := DefaultConfig()
		cfg.Project.Name = "testapp"
		cfg.Project.Repository = "owner/testapp"

		err := WriteConfig(cfg, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v",

				// Verify file exists
				err)
		}
		if !(ConfigExists(dir)) {
			t.Fatal("expected true")

			// Reload and verify
		}

		loaded, err := LoadConfig(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("testapp", loaded.Project.Name) {
			t.Fatalf("want %v, got %v", "testapp", loaded.Project.Name)
		}
		if !stdlibAssertEqual("owner/testapp", loaded.Project.Repository) {
			t.Fatalf("want %v, got %v", "owner/testapp", loaded.Project.Repository)
		}

	})

	t.Run("creates .core directory if missing", func(t *testing.T) {
		dir := t.TempDir()

		cfg := DefaultConfig()
		err := WriteConfig(cfg, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v",

				// Check directory was created
				err)
		}

		coreDir := ax.Join(dir, ConfigDir)
		info, err := ax.Stat(coreDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(info.IsDir()) {
			t.Fatal("expected true")
		}

	})
}

func TestConfig_GetRepository_Good(t *testing.T) {
	t.Run("returns repository", func(t *testing.T) {
		cfg := &Config{
			Project: ProjectConfig{
				Repository: "owner/repo",
			},
		}
		if !stdlibAssertEqual("owner/repo", cfg.GetRepository()) {
			t.Fatalf("want %v, got %v", "owner/repo", cfg.GetRepository())
		}

	})

	t.Run("returns empty string when not set", func(t *testing.T) {
		cfg := &Config{}
		if !stdlibAssertEmpty(cfg.GetRepository()) {
			t.Fatalf("expected empty, got %v", cfg.GetRepository())
		}

	})

	t.Run("returns empty string for nil config", func(t *testing.T) {
		var cfg *Config
		if !stdlibAssertEmpty(cfg.GetRepository()) {
			t.Fatalf("expected empty, got %v", cfg.GetRepository())
		}

	})
}

func TestConfig_GetProjectName_Good(t *testing.T) {
	t.Run("returns project name", func(t *testing.T) {
		cfg := &Config{
			Project: ProjectConfig{
				Name: "myapp",
			},
		}
		if !stdlibAssertEqual("myapp", cfg.GetProjectName()) {
			t.Fatalf("want %v, got %v", "myapp", cfg.GetProjectName())
		}

	})

	t.Run("returns empty string when not set", func(t *testing.T) {
		cfg := &Config{}
		if !stdlibAssertEmpty(cfg.GetProjectName()) {
			t.Fatalf("expected empty, got %v", cfg.GetProjectName())
		}

	})

	t.Run("returns empty string for nil config", func(t *testing.T) {
		var cfg *Config
		if !stdlibAssertEmpty(cfg.GetProjectName()) {
			t.Fatalf("expected empty, got %v", cfg.GetProjectName())
		}

	})
}

func TestConfig_SetVersion_Good(t *testing.T) {
	t.Run("sets version override", func(t *testing.T) {
		cfg := &Config{}
		cfg.SetVersion("v1.2.3")
		if !stdlibAssertEqual("v1.2.3", cfg.version) {
			t.Fatalf("want %v, got %v", "v1.2.3", cfg.version)
		}

	})

	t.Run("is safe on nil config", func(t *testing.T) {
		var cfg *Config
		cfg.SetVersion("v1.2.3")
	})
}

func TestConfig_SetProjectDir_Good(t *testing.T) {
	t.Run("sets project directory", func(t *testing.T) {
		cfg := &Config{}
		cfg.SetProjectDir("/path/to/project")
		if !stdlibAssertEqual("/path/to/project", cfg.projectDir) {
			t.Fatalf("want %v, got %v", "/path/to/project", cfg.projectDir)
		}

	})

	t.Run("is safe on nil config", func(t *testing.T) {
		var cfg *Config
		cfg.SetProjectDir("/path/to/project")
	})
}

func TestConfig_PublishersIter_NilSafe(t *testing.T) {
	var cfg *Config

	iter := cfg.PublishersIter()
	if stdlibAssertNil(iter) {
		t.Fatal("expected non-nil")
	}

	called := false
	iter(func(p PublisherConfig) bool {
		called = true
		return true
	})
	if called {
		t.Fatal("expected false")
	}

}

func TestConfig_SetOutput_Good(t *testing.T) {
	t.Run("sets output medium and directory", func(t *testing.T) {
		cfg := &Config{}
		medium := io.NewMemoryMedium()

		cfg.SetOutput(medium, "releases")
		if !stdlibAssertEqual(medium, cfg.output) {
			t.Fatalf("want %v, got %v", medium, cfg.output)
		}
		if !stdlibAssertEqual("releases", cfg.outputDir) {
			t.Fatalf("want %v, got %v", "releases", cfg.outputDir)
		}

	})

	t.Run("sets output medium only", func(t *testing.T) {
		cfg := &Config{}
		medium := io.NewMemoryMedium()

		cfg.SetOutputMedium(medium)
		if !stdlibAssertEqual(medium, cfg.output) {
			t.Fatalf("want %v, got %v", medium, cfg.output)
		}

	})

	t.Run("sets output directory only", func(t *testing.T) {
		cfg := &Config{}

		cfg.SetOutputDir("artifacts")
		if !stdlibAssertEqual("artifacts", cfg.outputDir) {
			t.Fatalf("want %v, got %v", "artifacts", cfg.outputDir)
		}

	})
}

func TestConfig_WriteConfig_Bad(t *testing.T) {
	t.Run("returns error for unwritable directory", func(t *testing.T) {
		if ax.Geteuid() == 0 {
			t.Skip("root can write to any directory")
		}
		dir := t.TempDir()

		// Create .core directory and make it unwritable
		coreDir := ax.Join(dir, ConfigDir)
		err := ax.MkdirAll(coreDir, 0755)
		if err != nil {
			t.Fatalf("unexpected error: %v",

				// Make directory read-only
				err)
		}

		err = ax.Chmod(coreDir, 0555)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		defer func() { _ = ax.Chmod(coreDir, 0755) }()

		cfg := DefaultConfig()
		err = WriteConfig(cfg, dir)
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "failed to write config file") {
			t.Fatalf("expected %v to contain %v", err.Error(), "failed to write config file")
		}

	})

	t.Run("returns error when directory creation fails", func(t *testing.T) {
		if ax.Geteuid() == 0 {
			t.Skip("root can create directories anywhere")
		}
		// Use a path that doesn't exist and can't be created
		cfg := DefaultConfig()
		err := WriteConfig(cfg, "/nonexistent/path/that/cannot/be/created")
		if err == nil {
			t.Fatal("expected error")
		}

	})
}

func TestConfig_ApplyDefaults_Good(t *testing.T) {
	t.Run("applies version default when zero", func(t *testing.T) {
		cfg := &Config{Version: 0}
		applyDefaults(cfg)
		if !stdlibAssertEqual(1, cfg.Version) {
			t.Fatalf("want %v, got %v", 1, cfg.Version)
		}

	})

	t.Run("preserves existing version", func(t *testing.T) {
		cfg := &Config{Version: 2}
		applyDefaults(cfg)
		if !stdlibAssertEqual(2, cfg.Version) {
			t.Fatalf("want %v, got %v", 2, cfg.Version)
		}

	})

	t.Run("applies changelog defaults only when both empty", func(t *testing.T) {
		cfg := &Config{
			Changelog: ChangelogConfig{
				Include: []string{"feat"},
			},
		}
		applyDefaults(cfg)
		if !stdlibAssertEqual("conventional", cfg.Changelog.Use) {
			t.

				// Include/Exclude defaults are only applied when both lists are empty.
				Fatalf("want %v, got %v", "conventional", cfg.Changelog.Use)
		}
		if !stdlibAssertEqual([]string{"feat"}, cfg.Changelog.Include) {
			t.Fatalf("want %v, got %v", []string{"feat"}, cfg.Changelog.Include)
		}
		if !stdlibAssertEmpty(cfg.Changelog.Exclude) {
			t.Fatalf("expected empty, got %v", cfg.Changelog.Exclude)
		}

	})
}
