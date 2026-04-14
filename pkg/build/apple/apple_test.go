package apple

import (
	"context"
	"testing"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	build "dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/build/pkg/build/signing"
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
	assert.Equal(t, "arm64", builder.options.Arch)
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

func TestAppleBuilder_Register_Good(t *testing.T) {
	c := core.New()

	result := Register(c)
	require.True(t, result.OK)

	builder, ok := result.Value.(*AppleBuilder)
	require.True(t, ok)
	assert.Equal(t, "apple", builder.Name())
	assert.True(t, c.Service("apple").OK)
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

func boolPtr(value bool) *bool {
	return &value
}
