package release

import (
	"testing"

	"dappco.re/go/core/build/internal/ax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupConfigTestDir creates a temp directory with optional .core/release.yaml content.
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
		require.NoError(t, err)
		require.NotNil(t, cfg)

		assert.Equal(t, 1, cfg.Version)
		assert.Equal(t, "myapp", cfg.Project.Name)
		assert.Equal(t, "owner/repo", cfg.Project.Repository)
		assert.Len(t, cfg.Build.Targets, 2)
		assert.Equal(t, "xz", cfg.Build.ArchiveFormat)
		assert.Equal(t, "linux", cfg.Build.Targets[0].OS)
		assert.Equal(t, "amd64", cfg.Build.Targets[0].Arch)
		assert.Equal(t, "darwin", cfg.Build.Targets[1].OS)
		assert.Equal(t, "arm64", cfg.Build.Targets[1].Arch)
		assert.Len(t, cfg.Publishers, 1)
		assert.Equal(t, "github", cfg.Publishers[0].Type)
		assert.True(t, cfg.Publishers[0].Prerelease)
		assert.False(t, cfg.Publishers[0].Draft)
		assert.Equal(t, []string{"feat", "fix"}, cfg.Changelog.Include)
		assert.Equal(t, []string{"chore"}, cfg.Changelog.Exclude)
	})

	t.Run("returns defaults when config file missing", func(t *testing.T) {
		dir := t.TempDir()

		cfg, err := LoadConfig(dir)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		defaults := DefaultConfig()
		assert.Equal(t, defaults.Version, cfg.Version)
		assert.Equal(t, defaults.Build.Targets, cfg.Build.Targets)
		assert.Equal(t, defaults.Publishers, cfg.Publishers)
		assert.Equal(t, defaults.Changelog.Include, cfg.Changelog.Include)
		assert.Equal(t, defaults.Changelog.Exclude, cfg.Changelog.Exclude)
	})

	t.Run("applies defaults for missing fields", func(t *testing.T) {
		content := `
version: 2
project:
  name: partial
`
		dir := setupConfigTestDir(t, content)

		cfg, err := LoadConfig(dir)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Explicit values preserved
		assert.Equal(t, 2, cfg.Version)
		assert.Equal(t, "partial", cfg.Project.Name)

		// Defaults applied
		defaults := DefaultConfig()
		assert.Equal(t, defaults.Build.Targets, cfg.Build.Targets)
		assert.Equal(t, defaults.Publishers, cfg.Publishers)
	})

	t.Run("sets project directory on load", func(t *testing.T) {
		dir := setupConfigTestDir(t, "version: 1")

		cfg, err := LoadConfig(dir)
		require.NoError(t, err)
		assert.Equal(t, dir, cfg.projectDir)
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
  output: $SDK_OUTPUT
`
	dir := setupConfigTestDir(t, content)

	cfg, err := LoadConfig(dir)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "owner/release-app", cfg.Project.Repository)
	assert.Equal(t, "xz", cfg.Build.ArchiveFormat)
	require.Len(t, cfg.Build.Targets, 1)
	assert.Equal(t, "darwin", cfg.Build.Targets[0].OS)
	assert.Equal(t, "arm64", cfg.Build.Targets[0].Arch)
	require.Len(t, cfg.Publishers, 1)
	assert.Equal(t, "owner/homebrew-tap", cfg.Publishers[0].Tap)
	require.NotNil(t, cfg.SDK)
	assert.Equal(t, "docs/openapi.yaml", cfg.SDK.Spec)
	assert.Equal(t, "generated/sdk", cfg.SDK.Output)
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
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "failed to parse config file")
	})

	t.Run("returns default config when config path is a directory", func(t *testing.T) {
		dir := t.TempDir()
		coreDir := ax.Join(dir, ConfigDir)
		err := ax.MkdirAll(coreDir, 0755)
		require.NoError(t, err)

		// Create config as a directory instead of file
		configPath := ax.Join(coreDir, ConfigFileName)
		err = ax.Mkdir(configPath, 0755)
		require.NoError(t, err)

		cfg, err := LoadConfig(dir)
		assert.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, 1, cfg.Version)
		assert.Equal(t, dir, cfg.projectDir)
	})
}

func TestConfig_DefaultConfig_Good(t *testing.T) {
	t.Run("returns sensible defaults", func(t *testing.T) {
		cfg := DefaultConfig()

		assert.Equal(t, 1, cfg.Version)
		assert.Empty(t, cfg.Project.Name)
		assert.Empty(t, cfg.Project.Repository)

		// Default targets
		assert.Len(t, cfg.Build.Targets, 4)
		hasLinuxAmd64 := false
		hasDarwinArm64 := false
		hasWindowsAmd64 := false
		for _, target := range cfg.Build.Targets {
			if target.OS == "linux" && target.Arch == "amd64" {
				hasLinuxAmd64 = true
			}
			if target.OS == "darwin" && target.Arch == "arm64" {
				hasDarwinArm64 = true
			}
			if target.OS == "windows" && target.Arch == "amd64" {
				hasWindowsAmd64 = true
			}
		}
		assert.True(t, hasLinuxAmd64)
		assert.True(t, hasDarwinArm64)
		assert.True(t, hasWindowsAmd64)

		// Default publisher
		assert.Len(t, cfg.Publishers, 1)
		assert.Equal(t, "github", cfg.Publishers[0].Type)
		assert.False(t, cfg.Publishers[0].Prerelease)
		assert.False(t, cfg.Publishers[0].Draft)

		// Default changelog settings
		assert.Contains(t, cfg.Changelog.Include, "feat")
		assert.Contains(t, cfg.Changelog.Include, "fix")
		assert.Contains(t, cfg.Changelog.Exclude, "chore")
		assert.Contains(t, cfg.Changelog.Exclude, "docs")
	})
}

func TestConfig_ScaffoldConfig_Good(t *testing.T) {
	t.Run("returns documented init scaffold", func(t *testing.T) {
		cfg := ScaffoldConfig()

		require.NotNil(t, cfg.SDK)
		assert.Equal(t, "api/openapi.yaml", cfg.SDK.Spec)
		assert.Equal(t, []string{"typescript", "python", "go", "php"}, cfg.SDK.Languages)
		assert.Equal(t, "sdk", cfg.SDK.Output)
		assert.True(t, cfg.SDK.Diff.Enabled)
		assert.False(t, cfg.SDK.Diff.FailOnBreaking)
	})
}

func TestConfig_ConfigPath_Good(t *testing.T) {
	t.Run("returns correct path", func(t *testing.T) {
		path := ConfigPath("/project/root")
		assert.Equal(t, "/project/root/.core/release.yaml", path)
	})
}

func TestConfig_ConfigExists_Good(t *testing.T) {
	t.Run("returns true when config exists", func(t *testing.T) {
		dir := setupConfigTestDir(t, "version: 1")
		assert.True(t, ConfigExists(dir))
	})

	t.Run("returns false when config missing", func(t *testing.T) {
		dir := t.TempDir()
		assert.False(t, ConfigExists(dir))
	})

	t.Run("returns false when .core dir missing", func(t *testing.T) {
		dir := t.TempDir()
		assert.False(t, ConfigExists(dir))
	})
}

func TestConfig_WriteConfig_Good(t *testing.T) {
	t.Run("writes config to file", func(t *testing.T) {
		dir := t.TempDir()

		cfg := DefaultConfig()
		cfg.Project.Name = "testapp"
		cfg.Project.Repository = "owner/testapp"

		err := WriteConfig(cfg, dir)
		require.NoError(t, err)

		// Verify file exists
		assert.True(t, ConfigExists(dir))

		// Reload and verify
		loaded, err := LoadConfig(dir)
		require.NoError(t, err)
		assert.Equal(t, "testapp", loaded.Project.Name)
		assert.Equal(t, "owner/testapp", loaded.Project.Repository)
	})

	t.Run("creates .core directory if missing", func(t *testing.T) {
		dir := t.TempDir()

		cfg := DefaultConfig()
		err := WriteConfig(cfg, dir)
		require.NoError(t, err)

		// Check directory was created
		coreDir := ax.Join(dir, ConfigDir)
		info, err := ax.Stat(coreDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})
}

func TestConfig_GetRepository_Good(t *testing.T) {
	t.Run("returns repository", func(t *testing.T) {
		cfg := &Config{
			Project: ProjectConfig{
				Repository: "owner/repo",
			},
		}
		assert.Equal(t, "owner/repo", cfg.GetRepository())
	})

	t.Run("returns empty string when not set", func(t *testing.T) {
		cfg := &Config{}
		assert.Empty(t, cfg.GetRepository())
	})
}

func TestConfig_GetProjectName_Good(t *testing.T) {
	t.Run("returns project name", func(t *testing.T) {
		cfg := &Config{
			Project: ProjectConfig{
				Name: "myapp",
			},
		}
		assert.Equal(t, "myapp", cfg.GetProjectName())
	})

	t.Run("returns empty string when not set", func(t *testing.T) {
		cfg := &Config{}
		assert.Empty(t, cfg.GetProjectName())
	})
}

func TestConfig_SetVersion_Good(t *testing.T) {
	t.Run("sets version override", func(t *testing.T) {
		cfg := &Config{}
		cfg.SetVersion("v1.2.3")
		assert.Equal(t, "v1.2.3", cfg.version)
	})
}

func TestConfig_SetProjectDir_Good(t *testing.T) {
	t.Run("sets project directory", func(t *testing.T) {
		cfg := &Config{}
		cfg.SetProjectDir("/path/to/project")
		assert.Equal(t, "/path/to/project", cfg.projectDir)
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
		require.NoError(t, err)

		// Make directory read-only
		err = ax.Chmod(coreDir, 0555)
		require.NoError(t, err)
		defer func() { _ = ax.Chmod(coreDir, 0755) }()

		cfg := DefaultConfig()
		err = WriteConfig(cfg, dir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write config file")
	})

	t.Run("returns error when directory creation fails", func(t *testing.T) {
		if ax.Geteuid() == 0 {
			t.Skip("root can create directories anywhere")
		}
		// Use a path that doesn't exist and can't be created
		cfg := DefaultConfig()
		err := WriteConfig(cfg, "/nonexistent/path/that/cannot/be/created")
		assert.Error(t, err)
	})
}

func TestConfig_ApplyDefaults_Good(t *testing.T) {
	t.Run("applies version default when zero", func(t *testing.T) {
		cfg := &Config{Version: 0}
		applyDefaults(cfg)
		assert.Equal(t, 1, cfg.Version)
	})

	t.Run("preserves existing version", func(t *testing.T) {
		cfg := &Config{Version: 2}
		applyDefaults(cfg)
		assert.Equal(t, 2, cfg.Version)
	})

	t.Run("applies changelog defaults only when both empty", func(t *testing.T) {
		cfg := &Config{
			Changelog: ChangelogConfig{
				Include: []string{"feat"},
			},
		}
		applyDefaults(cfg)
		// Should not apply defaults because Include is set
		assert.Equal(t, []string{"feat"}, cfg.Changelog.Include)
		assert.Empty(t, cfg.Changelog.Exclude)
	})
}
