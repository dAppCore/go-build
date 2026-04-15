package release

import (
	"context"
	"testing"

	"dappco.re/go/build/internal/ax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runReleaseGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	require.NoError(t, ax.ExecDir(context.Background(), dir, "git", args...))
}

func TestSDK_RunSDKNilConfig_Bad(t *testing.T) {
	_, err := RunSDK(context.Background(), nil, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config is nil")
}

func TestSDK_RunSDKNoSDKConfig_FallsBackToDefaults_Good(t *testing.T) {
	cfg := &Config{}
	cfg.projectDir = t.TempDir()
	cfg.version = "v1.0.0"

	result, err := RunSDK(context.Background(), cfg, true)
	require.NoError(t, err)
	assert.Equal(t, "v1.0.0", result.Version)
	assert.Equal(t, "sdk", result.Output)
	assert.Equal(t, []string{"typescript", "python", "go", "php"}, result.Languages)
}

func TestSDK_RunSDKNoSDKConfig_UsesBuildConfig_Good(t *testing.T) {
	projectDir := t.TempDir()
	buildConfig := `version: 1
sdk:
  spec: api/openapi.yaml
  languages: [typescript, go]
  output: generated/sdk
`
	require.NoError(t, ax.MkdirAll(ax.Join(projectDir, ".core"), 0o755))
	require.NoError(t, ax.WriteFile(ax.Join(projectDir, ".core", "build.yaml"), []byte(buildConfig), 0o644))

	cfg := &Config{}
	cfg.projectDir = projectDir
	cfg.version = "v2.0.0"

	result, err := RunSDK(context.Background(), cfg, true)
	require.NoError(t, err)
	assert.Equal(t, "v2.0.0", result.Version)
	assert.Equal(t, "generated/sdk", result.Output)
	assert.Equal(t, []string{"typescript", "go"}, result.Languages)
}

func TestSDK_RunSDKDryRun_Good(t *testing.T) {
	cfg := &Config{
		SDK: &SDKConfig{
			Languages: []string{"typescript", "python"},
			Output:    "sdk",
		},
	}
	cfg.projectDir = "/tmp"
	cfg.version = "v1.0.0"

	result, err := RunSDK(context.Background(), cfg, true)
	require.NoError(t, err)

	assert.Equal(t, "v1.0.0", result.Version)
	assert.Len(t, result.Languages, 2)
	assert.Contains(t, result.Languages, "typescript")
	assert.Contains(t, result.Languages, "python")
	assert.Equal(t, "sdk", result.Output)
}

func TestSDK_RunSDKDryRunDefaultOutput_Good(t *testing.T) {
	cfg := &Config{
		SDK: &SDKConfig{
			Languages: []string{"go"},
			Output:    "", // Empty output, should default to "sdk"
		},
	}
	cfg.projectDir = "/tmp"
	cfg.version = "v2.0.0"

	result, err := RunSDK(context.Background(), cfg, true)
	require.NoError(t, err)

	assert.Equal(t, "sdk", result.Output)
}

func TestSDK_RunSDKDryRunDefaultsLanguages_Good(t *testing.T) {
	cfg := &Config{
		SDK: &SDKConfig{},
	}
	cfg.projectDir = t.TempDir()
	cfg.version = "v2.0.0"

	result, err := RunSDK(context.Background(), cfg, true)
	require.NoError(t, err)

	assert.Equal(t, "sdk", result.Output)
	assert.Equal(t, []string{"typescript", "python", "go", "php"}, result.Languages)
}

func TestSDK_RunSDKDryRunDefaultProjectDir_Good(t *testing.T) {
	cfg := &Config{
		SDK: &SDKConfig{
			Languages: []string{"typescript"},
			Output:    "out",
		},
	}
	// projectDir is empty, should default to "."
	cfg.version = "v1.0.0"

	result, err := RunSDK(context.Background(), cfg, true)
	require.NoError(t, err)

	assert.Equal(t, "v1.0.0", result.Version)
}

func TestSDK_RunSDKBreakingChangesFailOnBreaking_Bad(t *testing.T) {
	// This test verifies that when diff.FailOnBreaking is true and breaking changes
	// are detected, RunSDK returns an error. However, since we can't easily mock
	// the diff check, this test verifies the config is correctly processed.
	// The actual breaking change detection is tested in pkg/sdk/diff_test.go.
	cfg := &Config{
		SDK: &SDKConfig{
			Languages: []string{"typescript"},
			Output:    "sdk",
			Diff: SDKDiffConfig{
				Enabled:        true,
				FailOnBreaking: true,
			},
		},
	}
	cfg.projectDir = "/tmp"
	cfg.version = "v1.0.0"

	// In dry run mode with no git repo, diff check will fail gracefully
	// (non-fatal warning), so this should succeed
	result, err := RunSDK(context.Background(), cfg, true)
	require.NoError(t, err)
	assert.Equal(t, "v1.0.0", result.Version)
}

func TestSDK_ToSDKConfig_Good(t *testing.T) {
	sdkCfg := &SDKConfig{
		Spec:      "api/openapi.yaml",
		Languages: []string{"typescript", "go"},
		Output:    "sdk",
		Package: SDKPackageConfig{
			Name:    "myapi",
			Version: "v1.0.0",
		},
		Diff: SDKDiffConfig{
			Enabled:        true,
			FailOnBreaking: true,
		},
		Publish: SDKPublishConfig{
			Repo: "owner/sdk-monorepo",
			Path: "packages/api-client",
		},
	}

	result := toSDKConfig(sdkCfg)

	assert.Equal(t, "api/openapi.yaml", result.Spec)
	assert.Equal(t, []string{"typescript", "go"}, result.Languages)
	assert.Equal(t, "sdk", result.Output)
	assert.Equal(t, "myapi", result.Package.Name)
	assert.Equal(t, "v1.0.0", result.Package.Version)
	assert.True(t, result.Diff.Enabled)
	assert.True(t, result.Diff.FailOnBreaking)
	assert.Equal(t, "owner/sdk-monorepo", result.Publish.Repo)
	assert.Equal(t, "packages/api-client", result.Publish.Path)
}

func TestSDK_ToSDKConfigNilInput_Good(t *testing.T) {
	result := toSDKConfig(nil)
	assert.Nil(t, result)
}

func TestSDK_RunSDKWithDiffEnabledNoFailOnBreaking_Good(t *testing.T) {
	// Tests diff enabled but FailOnBreaking=false (should warn but not fail)
	cfg := &Config{
		SDK: &SDKConfig{
			Languages: []string{"typescript"},
			Output:    "sdk",
			Diff: SDKDiffConfig{
				Enabled:        true,
				FailOnBreaking: false,
			},
		},
	}
	cfg.projectDir = "/tmp"
	cfg.version = "v1.0.0"

	// Dry run should succeed even without git repo (diff check fails gracefully)
	result, err := RunSDK(context.Background(), cfg, true)
	require.NoError(t, err)
	assert.Equal(t, "v1.0.0", result.Version)
	assert.Contains(t, result.Languages, "typescript")
}

func TestSDK_RunSDKMultipleLanguages_Good(t *testing.T) {
	// Tests multiple language support
	cfg := &Config{
		SDK: &SDKConfig{
			Languages: []string{"typescript", "python", "go", "java"},
			Output:    "multi-sdk",
		},
	}
	cfg.projectDir = "/tmp"
	cfg.version = "v3.0.0"

	result, err := RunSDK(context.Background(), cfg, true)
	require.NoError(t, err)

	assert.Equal(t, "v3.0.0", result.Version)
	assert.Len(t, result.Languages, 4)
	assert.Equal(t, "multi-sdk", result.Output)
}

func TestSDK_RunSDKWithPackageConfig_Good(t *testing.T) {
	// Tests that package config is properly handled
	cfg := &Config{
		SDK: &SDKConfig{
			Spec:      "openapi.yaml",
			Languages: []string{"typescript"},
			Output:    "sdk",
			Package: SDKPackageConfig{
				Name:    "my-custom-sdk",
				Version: "v2.5.0",
			},
		},
	}
	cfg.projectDir = "/tmp"
	cfg.version = "v1.0.0"

	result, err := RunSDK(context.Background(), cfg, true)
	require.NoError(t, err)
	assert.Equal(t, "v1.0.0", result.Version)
}

func TestSDK_ToSDKConfigEmptyPackageConfig_Good(t *testing.T) {
	// Tests conversion with empty package config
	sdkCfg := &SDKConfig{
		Languages: []string{"go"},
		Output:    "sdk",
		// Package is empty struct
	}

	result := toSDKConfig(sdkCfg)

	assert.Equal(t, []string{"go"}, result.Languages)
	assert.Equal(t, "sdk", result.Output)
	assert.Empty(t, result.Package.Name)
	assert.Empty(t, result.Package.Version)
}

func TestSDK_ToSDKConfigDiffDisabled_Good(t *testing.T) {
	// Tests conversion with diff disabled
	sdkCfg := &SDKConfig{
		Languages: []string{"typescript"},
		Output:    "sdk",
		Diff: SDKDiffConfig{
			Enabled:        false,
			FailOnBreaking: false,
		},
	}

	result := toSDKConfig(sdkCfg)

	assert.False(t, result.Diff.Enabled)
	assert.False(t, result.Diff.FailOnBreaking)
}

func TestSDK_ResolveSDKOutputRoot_Good(t *testing.T) {
	t.Run("uses the default sdk root when no publish path is configured", func(t *testing.T) {
		assert.Equal(t, "sdk", resolveSDKOutputRoot(&SDKConfig{}))
	})

	t.Run("prefixes the configured publish path", func(t *testing.T) {
		cfg := &SDKConfig{
			Output: "generated",
			Publish: SDKPublishConfig{
				Path: "packages/api-client",
			},
		}

		assert.Equal(t, ax.Join("packages/api-client", "generated"), resolveSDKOutputRoot(cfg))
	})
}

func TestSDK_CheckBreakingChanges_UsesPreviousTaggedSpec_Good(t *testing.T) {
	dir := t.TempDir()
	runReleaseGit(t, dir, "init")
	runReleaseGit(t, dir, "config", "user.email", "test@example.com")
	runReleaseGit(t, dir, "config", "user.name", "Test User")

	baseSpec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /health:
    get:
      operationId: getHealth
      responses:
        "200":
          description: OK
  /users:
    get:
      operationId: getUsers
      responses:
        "200":
          description: OK
`

	currentSpec := `openapi: "3.0.0"
info:
  title: Test API
  version: "2.0.0"
paths:
  /health:
    get:
      operationId: getHealth
      responses:
        "200":
          description: OK
`

	specPath := ax.Join(dir, "openapi.yaml")
	require.NoError(t, ax.WriteFile(specPath, []byte(baseSpec), 0o644))
	runReleaseGit(t, dir, "add", "openapi.yaml")
	runReleaseGit(t, dir, "commit", "-m", "feat: add initial spec")
	runReleaseGit(t, dir, "tag", "v1.0.0")

	require.NoError(t, ax.WriteFile(specPath, []byte(currentSpec), 0o644))
	runReleaseGit(t, dir, "add", "openapi.yaml")
	runReleaseGit(t, dir, "commit", "-m", "feat: remove users endpoint")

	breaking, err := checkBreakingChanges(context.Background(), dir, &SDKConfig{Spec: "openapi.yaml"})
	require.NoError(t, err)
	assert.True(t, breaking)
}
