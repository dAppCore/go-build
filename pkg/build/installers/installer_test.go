package installers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validConfig is a fully populated InstallerConfig used as the happy-path baseline.
var validConfig = InstallerConfig{
	Version:    "v1.2.3",
	Repo:       "dappcore/core",
	BinaryName: "core",
}

// TestInstaller_GenerateInstaller_Good verifies that each known variant produces a non-empty
// shell script containing the expected shebang, binary name, version, and repo strings.
func TestInstaller_GenerateInstaller_Good(t *testing.T) {
	allVariants := []InstallerVariant{
		VariantFull,
		VariantCI,
		VariantPHP,
		VariantGo,
		VariantAgent,
		VariantDev,
	}

	for _, variant := range allVariants {
		v := variant // capture range variable
		t.Run(string(v), func(t *testing.T) {
			script, err := GenerateInstaller(v, validConfig)
			require.NoError(t, err)
			assert.NotEmpty(t, script)
			assert.Contains(t, script, "#!/usr/bin/env bash", "script must start with bash shebang")
			assert.Contains(t, script, validConfig.BinaryName, "script must reference binary name")
			assert.Contains(t, script, validConfig.Version, "script must reference version")
			assert.Contains(t, script, validConfig.Repo, "script must reference repo")
		})
	}
}

// TestInstaller_GenerateInstaller_Bad verifies that an unknown variant returns an error and empty output.
func TestInstaller_GenerateInstaller_Bad(t *testing.T) {
	t.Run("unknown variant returns error", func(t *testing.T) {
		script, err := GenerateInstaller("nonexistent", validConfig)
		assert.Error(t, err)
		assert.Empty(t, script)
	})

	t.Run("empty variant string returns error", func(t *testing.T) {
		script, err := GenerateInstaller("", validConfig)
		assert.Error(t, err)
		assert.Empty(t, script)
	})
}

// TestInstaller_GenerateInstaller_Ugly verifies that empty config fields are rendered without
// panicking — the template may produce incomplete scripts but must not error.
func TestInstaller_GenerateInstaller_Ugly(t *testing.T) {
	t.Run("empty Version renders without error", func(t *testing.T) {
		cfg := InstallerConfig{Version: "", Repo: "dappcore/core", BinaryName: "core"}
		script, err := GenerateInstaller(VariantFull, cfg)
		require.NoError(t, err)
		assert.NotEmpty(t, script)
	})

	t.Run("empty Repo renders without error", func(t *testing.T) {
		cfg := InstallerConfig{Version: "v1.0.0", Repo: "", BinaryName: "core"}
		script, err := GenerateInstaller(VariantCI, cfg)
		require.NoError(t, err)
		assert.NotEmpty(t, script)
	})

	t.Run("empty BinaryName renders without error", func(t *testing.T) {
		cfg := InstallerConfig{Version: "v1.0.0", Repo: "dappcore/core", BinaryName: ""}
		script, err := GenerateInstaller(VariantAgent, cfg)
		require.NoError(t, err)
		assert.NotEmpty(t, script)
	})

	t.Run("fully empty config renders without error", func(t *testing.T) {
		script, err := GenerateInstaller(VariantDev, InstallerConfig{})
		require.NoError(t, err)
		assert.NotEmpty(t, script)
	})
}

// TestInstaller_GenerateAll_Good verifies that GenerateAll returns one entry per variant
// and that each script is a non-empty bash script.
func TestInstaller_GenerateAll_Good(t *testing.T) {
	scripts, err := GenerateAll(validConfig)
	require.NoError(t, err)

	expectedOutputs := []string{
		"setup.sh",
		"ci.sh",
		"php.sh",
		"go.sh",
		"agent.sh",
		"dev.sh",
	}

	assert.Len(t, scripts, len(variantTemplates), "GenerateAll must return one entry per variant")

	for _, name := range expectedOutputs {
		t.Run(name, func(t *testing.T) {
			content, ok := scripts[name]
			assert.True(t, ok, "GenerateAll must include %s", name)
			assert.NotEmpty(t, content)
			assert.Contains(t, content, "#!/usr/bin/env bash")
			assert.Contains(t, content, validConfig.BinaryName)
		})
	}
}

func TestInstaller_Variants_Good(t *testing.T) {
	assert.Equal(t, []InstallerVariant{
		VariantFull,
		VariantCI,
		VariantPHP,
		VariantGo,
		VariantAgent,
		VariantDev,
	}, Variants())
}

func TestInstaller_OutputName_Good(t *testing.T) {
	assert.Equal(t, "setup.sh", OutputName(VariantFull))
	assert.Equal(t, "ci.sh", OutputName(VariantCI))
	assert.Equal(t, "php.sh", OutputName(VariantPHP))
	assert.Equal(t, "go.sh", OutputName(VariantGo))
	assert.Equal(t, "agent.sh", OutputName(VariantAgent))
	assert.Equal(t, "dev.sh", OutputName(VariantDev))
	assert.Empty(t, OutputName("bogus"))
}
