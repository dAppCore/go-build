package apple

import (
	"context"
	"testing"

	"dappco.re/go/core"
	"dappco.re/go/build/internal/ax"
	build "dappco.re/go/build/pkg/build"
	"dappco.re/go/build/pkg/build/signing"
	coreio "dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppleBuilder_New_Good(t *testing.T) {
	builder := New(
		WithArch("arm64"),
		WithSign(false),
		WithNotarise(false),
		WithDMG(true),
		WithTestFlight(true),
		WithAppStore(true),
	)

	require.NotNil(t, builder)
	assert.Equal(t, "apple", builder.Name())
	require.NotNil(t, builder.ServiceRuntime)
	assert.Equal(t, "arm64", builder.options.Arch)
	assert.Equal(t, "arm64", builder.Options().Arch)
	assert.False(t, builder.options.Sign)
	assert.False(t, builder.options.Notarise)
	assert.True(t, builder.options.DMG)
	assert.True(t, builder.options.TestFlight)
	assert.True(t, builder.options.AppStore)
	assert.True(t, builder.explicit.arch)
	assert.True(t, builder.explicit.sign)
	assert.True(t, builder.explicit.notarise)
	assert.True(t, builder.explicit.dmg)
	assert.True(t, builder.explicit.testFlight)
	assert.True(t, builder.explicit.appStore)
}

func TestAppleBuilder_New_PreservesExplicitDefaultValuedOptions_Good(t *testing.T) {
	builder := New(
		WithArch("universal"),
		WithSign(true),
		WithNotarise(true),
	)

	require.NotNil(t, builder)
	assert.Equal(t, "universal", builder.options.Arch)
	assert.True(t, builder.options.Sign)
	assert.True(t, builder.options.Notarise)
	assert.True(t, builder.explicit.arch)
	assert.True(t, builder.explicit.sign)
	assert.True(t, builder.explicit.notarise)
}

func TestAppleBuilder_Register_Good(t *testing.T) {
	c := core.New()

	result := Register(c)
	require.True(t, result.OK)

	builder, ok := result.Value.(*AppleBuilder)
	require.True(t, ok)
	assert.Equal(t, "apple", builder.Name())
	require.NotNil(t, builder.ServiceRuntime)
	assert.Same(t, c, builder.Core())
	assert.True(t, c.Service("apple").OK)
	assert.True(t, c.RegistryOf("services").Has("apple"))
}

func TestAppleBuilder_Detect_Good(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, ax.WriteFile(ax.Join(dir, "wails.json"), []byte("{}"), 0o644))

	result := New().Detect(coreio.Local, dir)
	require.True(t, result.OK)

	detected, ok := result.Value.(bool)
	require.True(t, ok)
	assert.True(t, detected)
}

func TestAppleBuilder_Build_Good(t *testing.T) {
	projectDir := t.TempDir()

	oldLoadConfig := loadConfigFn
	oldBuildApple := buildAppleFn
	oldDetermineVersion := determineVersion
	oldGetwd := getwdFn
	oldRunDir := runDirFn
	oldWriteXcodeCloudScripts := writeXcodeCloudScriptsFn
	t.Cleanup(func() {
		loadConfigFn = oldLoadConfig
		buildAppleFn = oldBuildApple
		determineVersion = oldDetermineVersion
		getwdFn = oldGetwd
		runDirFn = oldRunDir
		writeXcodeCloudScriptsFn = oldWriteXcodeCloudScripts
	})

	loadConfigFn = func(fs coreio.Medium, dir string) (*build.BuildConfig, error) {
		require.Equal(t, projectDir, dir)
		return &build.BuildConfig{
			Project: build.Project{
				Name:   "Core",
				Binary: "Core",
			},
			Build: build.Build{
				LDFlags: []string{"-s", "-w"},
			},
			Apple: build.AppleConfig{
				BundleID: "ai.lthn.core",
				Sign:     boolPtr(false),
			},
			Sign: signing.SignConfig{
				MacOS: signing.MacOSConfig{
					Identity: "Developer ID Application: Lethean CIC (ABC123DEF4)",
					TeamID:   "ABC123DEF4",
				},
			},
		}, nil
	}
	determineVersion = func(ctx context.Context, dir string) (string, error) {
		require.Equal(t, projectDir, dir)
		return "v1.2.3", nil
	}
	getwdFn = func() (string, error) {
		return projectDir, nil
	}
	runDirFn = func(ctx context.Context, dir, command string, args ...string) (string, error) {
		require.Equal(t, projectDir, dir)
		require.Equal(t, "git", command)
		assert.Equal(t, []string{"rev-list", "--count", "HEAD"}, args)
		return "42", nil
	}
	buildAppleFn = func(ctx context.Context, cfg *build.Config, options build.AppleOptions, buildNumber string) (*build.AppleBuildResult, error) {
		assert.Equal(t, ax.Join(projectDir, "dist", "apple"), cfg.OutputDir)
		assert.Equal(t, "Core", cfg.Name)
		assert.Equal(t, "v1.2.3", cfg.Version)
		assert.Equal(t, "42", buildNumber)
		assert.Equal(t, "ai.lthn.core", options.BundleID)
		assert.True(t, options.Sign)
		assert.Equal(t, "arm64", options.Arch)
		return &build.AppleBuildResult{
			BundlePath: ax.Join(cfg.OutputDir, "Core.app"),
		}, nil
	}

	result := New(WithArch("arm64"), WithSign(true)).Build(context.Background(), nil)
	require.True(t, result.OK)
	assert.Equal(t, ax.Join(projectDir, "dist", "apple", "Core.app"), result.Value)
}

func TestAppleBuilder_Build_PartialRuntimeOptionsPreservePipelineDefaults_Good(t *testing.T) {
	projectDir := t.TempDir()

	oldLoadConfig := loadConfigFn
	oldBuildApple := buildAppleFn
	oldDetermineVersion := determineVersion
	oldGetwd := getwdFn
	oldRunDir := runDirFn
	t.Cleanup(func() {
		loadConfigFn = oldLoadConfig
		buildAppleFn = oldBuildApple
		determineVersion = oldDetermineVersion
		getwdFn = oldGetwd
		runDirFn = oldRunDir
	})

	loadConfigFn = func(fs coreio.Medium, dir string) (*build.BuildConfig, error) {
		require.Equal(t, projectDir, dir)
		return &build.BuildConfig{
			Project: build.Project{
				Name:   "Core",
				Binary: "Core",
			},
			Apple: build.AppleConfig{
				BundleID: "ai.lthn.core",
				DMG:      boolPtr(true),
			},
			Sign: signing.SignConfig{
				MacOS: signing.MacOSConfig{
					Identity: "Developer ID Application: Lethean CIC (ABC123DEF4)",
					TeamID:   "ABC123DEF4",
				},
			},
		}, nil
	}
	determineVersion = func(ctx context.Context, dir string) (string, error) {
		return "v1.2.3", nil
	}
	getwdFn = func() (string, error) {
		return projectDir, nil
	}
	runDirFn = func(ctx context.Context, dir, command string, args ...string) (string, error) {
		return "42", nil
	}
	buildAppleFn = func(ctx context.Context, cfg *build.Config, options build.AppleOptions, buildNumber string) (*build.AppleBuildResult, error) {
		assert.Equal(t, "ai.lthn.override", options.BundleID)
		assert.True(t, options.Sign)
		assert.True(t, options.Notarise)
		assert.True(t, options.DMG)
		assert.False(t, options.TestFlight)
		assert.False(t, options.AppStore)
		return &build.AppleBuildResult{
			BundlePath: ax.Join(cfg.OutputDir, "Core.app"),
		}, nil
	}

	result := New().Build(context.Background(), &AppleOptions{
		BundleID: "ai.lthn.override",
	})
	require.True(t, result.OK)
	assert.Equal(t, ax.Join(projectDir, "dist", "apple", "Core.app"), result.Value)
}

func TestAppleBuilder_Build_SetsUpBuildCache_Good(t *testing.T) {
	projectDir := t.TempDir()

	oldLoadConfig := loadConfigFn
	oldBuildApple := buildAppleFn
	oldDetermineVersion := determineVersion
	oldGetwd := getwdFn
	oldRunDir := runDirFn
	t.Cleanup(func() {
		loadConfigFn = oldLoadConfig
		buildAppleFn = oldBuildApple
		determineVersion = oldDetermineVersion
		getwdFn = oldGetwd
		runDirFn = oldRunDir
	})

	loadConfigFn = func(fs coreio.Medium, dir string) (*build.BuildConfig, error) {
		require.Equal(t, projectDir, dir)
		return &build.BuildConfig{
			Project: build.Project{
				Name:   "Core",
				Binary: "Core",
			},
			Build: build.Build{
				Cache: build.CacheConfig{
					Enabled: true,
					Paths: []string{
						"cache/go-build",
						"cache/go-mod",
					},
				},
			},
			Apple: build.AppleConfig{
				BundleID: "ai.lthn.core",
				Sign:     boolPtr(false),
			},
		}, nil
	}
	determineVersion = func(ctx context.Context, dir string) (string, error) {
		return "v1.2.3", nil
	}
	getwdFn = func() (string, error) {
		return projectDir, nil
	}
	runDirFn = func(ctx context.Context, dir, command string, args ...string) (string, error) {
		return "42", nil
	}
	buildAppleFn = func(ctx context.Context, cfg *build.Config, options build.AppleOptions, buildNumber string) (*build.AppleBuildResult, error) {
		assert.Equal(t, []string{
			ax.Join(projectDir, "cache", "go-build"),
			ax.Join(projectDir, "cache", "go-mod"),
		}, cfg.Cache.Paths)
		assert.True(t, cfg.FS.Exists(ax.Join(projectDir, ".core", "cache")))
		assert.True(t, cfg.FS.Exists(ax.Join(projectDir, "cache", "go-build")))
		assert.True(t, cfg.FS.Exists(ax.Join(projectDir, "cache", "go-mod")))
		return &build.AppleBuildResult{
			BundlePath: ax.Join(cfg.OutputDir, "Core.app"),
		}, nil
	}

	result := New().Build(context.Background(), nil)
	require.True(t, result.OK)
	assert.Equal(t, ax.Join(projectDir, "dist", "apple", "Core.app"), result.Value)
}

func TestAppleBuilder_Build_WritesXcodeCloudScripts_Good(t *testing.T) {
	projectDir := t.TempDir()

	oldLoadConfig := loadConfigFn
	oldBuildApple := buildAppleFn
	oldDetermineVersion := determineVersion
	oldGetwd := getwdFn
	oldRunDir := runDirFn
	oldWriteXcodeCloudScripts := writeXcodeCloudScriptsFn
	t.Cleanup(func() {
		loadConfigFn = oldLoadConfig
		buildAppleFn = oldBuildApple
		determineVersion = oldDetermineVersion
		getwdFn = oldGetwd
		runDirFn = oldRunDir
		writeXcodeCloudScriptsFn = oldWriteXcodeCloudScripts
	})

	loadConfigFn = func(fs coreio.Medium, dir string) (*build.BuildConfig, error) {
		require.Equal(t, projectDir, dir)
		return &build.BuildConfig{
			Project: build.Project{
				Name:   "Core",
				Binary: "Core",
			},
			Apple: build.AppleConfig{
				BundleID: "ai.lthn.core",
				Sign:     boolPtr(false),
				XcodeCloud: build.XcodeCloudConfig{
					Workflow: "CoreGUI Release",
				},
			},
		}, nil
	}
	determineVersion = func(ctx context.Context, dir string) (string, error) {
		return "v1.2.3", nil
	}
	getwdFn = func() (string, error) {
		return projectDir, nil
	}
	runDirFn = func(ctx context.Context, dir, command string, args ...string) (string, error) {
		return "42", nil
	}

	var scriptsWritten bool
	writeXcodeCloudScriptsFn = func(fs coreio.Medium, dir string, cfg *build.BuildConfig) ([]string, error) {
		require.Equal(t, projectDir, dir)
		require.Equal(t, "CoreGUI Release", cfg.Apple.XcodeCloud.Workflow)
		scriptsWritten = true
		return []string{ax.Join(dir, build.XcodeCloudScriptsDir, build.XcodeCloudPreXcodebuildScriptName)}, nil
	}
	buildAppleFn = func(ctx context.Context, cfg *build.Config, options build.AppleOptions, buildNumber string) (*build.AppleBuildResult, error) {
		return &build.AppleBuildResult{
			BundlePath: ax.Join(cfg.OutputDir, "Core.app"),
		}, nil
	}

	result := New().Build(context.Background(), nil)
	require.True(t, result.OK)
	assert.True(t, scriptsWritten)
	assert.Equal(t, ax.Join(projectDir, "dist", "apple", "Core.app"), result.Value)
}

func TestAppleBuilder_resolveOptions_BoolOnlyRuntimeOverride_Good(t *testing.T) {
	builder := New()

	options := builder.resolveOptions(&build.BuildConfig{
		Apple: build.AppleConfig{
			BundleID: "ai.lthn.core",
			DMG:      boolPtr(true),
		},
	}, &AppleOptions{
		Sign:     false,
		Notarise: false,
		DMG:      false,
		AppStore: true,
	})

	assert.False(t, options.Sign)
	assert.False(t, options.Notarise)
	assert.False(t, options.DMG)
	assert.True(t, options.AppStore)
}

func TestApple_BuildWailsApp_UsesCurrentDirectoryAndStringLDFlags_Good(t *testing.T) {
	projectDir := t.TempDir()

	oldBuildWails := buildWailsAppFn
	oldGetwd := getwdFn
	t.Cleanup(func() {
		buildWailsAppFn = oldBuildWails
		getwdFn = oldGetwd
	})

	getwdFn = func() (string, error) {
		return projectDir, nil
	}

	buildWailsAppFn = func(ctx context.Context, cfg build.WailsBuildConfig) (string, error) {
		assert.Equal(t, projectDir, cfg.ProjectDir)
		assert.Equal(t, "Core", cfg.Name)
		assert.Equal(t, "arm64", cfg.Arch)
		assert.Equal(t, []string{"integration"}, cfg.BuildTags)
		assert.Equal(t, []string{"-s -w -X main.version=1.2.3"}, cfg.LDFlags)
		assert.Equal(t, "1.2.3", cfg.Version)
		assert.Equal(t, []string{"FOO=bar"}, cfg.Env)
		return ax.Join(projectDir, "dist", "Core.app"), nil
	}

	result := BuildWailsApp(context.Background(), WailsBuildConfig{
		Name:      "Core",
		Arch:      "arm64",
		BuildTags: []string{"integration"},
		LDFlags:   "-s -w -X main.version=1.2.3",
		OutputDir: ax.Join(projectDir, "dist"),
		Version:   "1.2.3",
		Env:       []string{"FOO=bar"},
	})

	require.True(t, result.OK)
	assert.Equal(t, ax.Join(projectDir, "dist", "Core.app"), result.Value)
}

func boolPtr(value bool) *bool {
	return &value
}
