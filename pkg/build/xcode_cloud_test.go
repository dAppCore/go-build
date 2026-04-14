package build

import (
	"testing"

	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXcodeCloud_HasXcodeCloudConfig_Good(t *testing.T) {
	assert.False(t, HasXcodeCloudConfig(nil))
	assert.False(t, HasXcodeCloudConfig(&BuildConfig{}))
	assert.True(t, HasXcodeCloudConfig(&BuildConfig{
		Apple: AppleConfig{
			XcodeCloud: XcodeCloudConfig{
				Workflow: "CoreGUI Release",
			},
		},
	}))
	assert.True(t, HasXcodeCloudConfig(&BuildConfig{
		Apple: AppleConfig{
			XcodeCloud: XcodeCloudConfig{
				Triggers: []XcodeCloudTrigger{{Branch: "main", Action: "testflight"}},
			},
		},
	}))
}

func TestXcodeCloud_GenerateXcodeCloudScripts_Good(t *testing.T) {
	scripts := GenerateXcodeCloudScripts("/tmp/project", &BuildConfig{
		Project: Project{
			Name:   "Core",
			Binary: "Core",
		},
		Apple: AppleConfig{
			BundleID: "ai.lthn.core",
			TeamID:   "ABC123DEF4",
			Arch:     "universal",
			Notarise: boolPtr(false),
			DMG:      boolPtr(true),
			AppStore: boolPtr(true),
		},
	})

	require.Len(t, scripts, 3)
	assert.Contains(t, scripts[XcodeCloudPostCloneScriptName], "go install github.com/wailsapp/wails/v3/cmd/wails3@latest")
	assert.Contains(t, scripts[XcodeCloudPostCloneScriptName], "find . -maxdepth 3 -name package.json")
	assert.Contains(t, scripts[XcodeCloudPostCloneScriptName], "deno_requested()")
	assert.Contains(t, scripts[XcodeCloudPostCloneScriptName], "DENO_ENABLE")
	assert.Contains(t, scripts[XcodeCloudPostCloneScriptName], "DENO_BUILD")
	assert.Contains(t, scripts[XcodeCloudPreXcodebuildScriptName], `core build apple --arch "universal" --config ".core/build.yaml" --notarise=false --dmg --appstore --bundle-id "ai.lthn.core" --team-id "ABC123DEF4"`)
	assert.Contains(t, scripts[XcodeCloudPostXcodebuildScriptName], `BUNDLE_PATH="dist/apple/Core.app"`)
	assert.Contains(t, scripts[XcodeCloudPostXcodebuildScriptName], "codesign --verify --deep --strict")
}

func TestXcodeCloud_WriteXcodeCloudScripts_Good(t *testing.T) {
	projectDir := t.TempDir()

	paths, err := WriteXcodeCloudScripts(io.Local, projectDir, &BuildConfig{
		Project: Project{
			Name:   "Core",
			Binary: "Core",
		},
		Apple: AppleConfig{
			XcodeCloud: XcodeCloudConfig{
				Workflow: "CoreGUI Release",
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{
		ax.Join(projectDir, XcodeCloudScriptsDir, XcodeCloudPostCloneScriptName),
		ax.Join(projectDir, XcodeCloudScriptsDir, XcodeCloudPreXcodebuildScriptName),
		ax.Join(projectDir, XcodeCloudScriptsDir, XcodeCloudPostXcodebuildScriptName),
	}, paths)

	for _, path := range paths {
		content, err := io.Local.Read(path)
		require.NoError(t, err)
		assert.NotEmpty(t, content)

		info, err := io.Local.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, 0o755, int(info.Mode().Perm()))
	}
}

func TestXcodeCloud_WriteXcodeCloudScripts_Bad(t *testing.T) {
	_, err := WriteXcodeCloudScripts(nil, t.TempDir(), DefaultConfig())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "filesystem medium is required")
}

func boolPtr(value bool) *bool {
	return &value
}
