package sdk

import (
	"context"
	"testing"

	"dappco.re/go/build/internal/ax"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSDK_SetVersion_Good(t *testing.T) {
	s := New("/tmp", nil)
	s.SetVersion("v1.2.3")

	assert.Equal(t, "v1.2.3", s.version)
}

func TestSDK_VersionPassedToGenerator_Good(t *testing.T) {
	config := &Config{
		Languages: []string{"typescript"},
		Output:    "sdk",
		Package: PackageConfig{
			Name: "test-sdk",
		},
	}
	s := New("/tmp", config)
	s.SetVersion("v2.0.0")

	assert.Equal(t, "v2.0.0", s.config.Package.Version)
}

func TestSDK_VersionTemplateIsRendered_Good(t *testing.T) {
	config := &Config{
		Package: PackageConfig{
			Name:    "test-sdk",
			Version: "{{.Version}}-beta",
		},
	}
	s := New("/tmp", config)
	s.SetVersion("v2.0.0")

	assert.Equal(t, "{{.Version}}-beta", s.config.Package.Version)
	assert.Equal(t, "v2.0.0-beta", s.resolvePackageVersion())
}

func TestSDK_DefaultConfig_Good(t *testing.T) {
	cfg := DefaultConfig()
	assert.Contains(t, cfg.Languages, "typescript")
	assert.Equal(t, "sdk", cfg.Output)
	assert.True(t, cfg.Diff.Enabled)
}

func TestSDK_ApplyDefaultsNormalisesLanguageAliases_Good(t *testing.T) {
	cfg := &Config{
		Languages: []string{"ts", "python", "py", "golang", "go", "php"},
	}

	cfg.ApplyDefaults()

	assert.Equal(t, []string{"typescript", "python", "go", "php"}, cfg.Languages)
}

func TestSDK_normaliseLanguage_Good(t *testing.T) {
	assert.Equal(t, "typescript", normaliseLanguage("ts"))
	assert.Equal(t, "typescript", normaliseLanguage("TypeScript"))
	assert.Equal(t, "python", normaliseLanguage("py"))
	assert.Equal(t, "go", normaliseLanguage("golang"))
	assert.Equal(t, "php", normaliseLanguage("php"))
}

func TestSDK_New_Good(t *testing.T) {
	t.Run("with nil config", func(t *testing.T) {
		s := New("/tmp", nil)
		assert.NotNil(t, s.config)
		assert.Equal(t, "sdk", s.config.Output)
	})

	t.Run("with custom config", func(t *testing.T) {
		cfg := &Config{Output: "custom"}
		s := New("/tmp", cfg)
		assert.Equal(t, "custom", s.config.Output)
		assert.True(t, s.config.Diff.Enabled)
	})

	t.Run("applies defaults and does not mutate the caller config", func(t *testing.T) {
		cfg := &Config{
			Languages: []string{"ts", "python", "py"},
		}

		s := New("/tmp", cfg)

		assert.Equal(t, []string{"typescript", "python"}, s.config.Languages)
		assert.Equal(t, "sdk", s.config.Output)
		assert.True(t, s.config.Diff.Enabled)
		assert.Equal(t, []string{"ts", "python", "py"}, cfg.Languages)
		assert.Empty(t, cfg.Output)
	})
}

func TestSDK_GenerateLanguage_Bad(t *testing.T) {

	t.Run("unknown language", func(t *testing.T) {

		tmpDir := t.TempDir()

		specPath := ax.Join(tmpDir, "openapi.yaml")

		err := ax.WriteFile(specPath, []byte("openapi: 3.0.0"), 0644)

		require.NoError(t, err)

		s := New(tmpDir, nil)

		err = s.GenerateLanguage(context.Background(), "invalid-lang")

		assert.Error(t, err)

		assert.Contains(t, err.Error(), "unknown language")

	})

}
