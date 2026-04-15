package sdkcfg

import (
	"testing"

	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/build/pkg/release"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadProjectConfig_Good(t *testing.T) {
	t.Run("falls back to release config in the provided medium", func(t *testing.T) {
		medium := io.NewMemoryMedium()
		projectDir := "project"

		require.NoError(t, medium.EnsureDir(ax.Join(projectDir, release.ConfigDir)))
		require.NoError(t, medium.Write(release.ConfigPath(projectDir), `
version: 1
sdk:
  spec: docs/openapi.yaml
  languages: [php]
  output: generated/sdk
`))

		cfg, err := LoadProjectConfig(medium, projectDir)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		assert.Equal(t, "docs/openapi.yaml", cfg.Spec)
		assert.Equal(t, []string{"php"}, cfg.Languages)
		assert.Equal(t, "generated/sdk", cfg.Output)
	})

	t.Run("prefers build config over release config in the provided medium", func(t *testing.T) {
		medium := io.NewMemoryMedium()
		projectDir := "project"

		require.NoError(t, medium.EnsureDir(ax.Join(projectDir, build.ConfigDir)))
		require.NoError(t, medium.Write(build.ConfigPath(projectDir), `
version: 1
sdk:
  spec: openapi.yaml
  languages: [typescript]
`))
		require.NoError(t, medium.EnsureDir(ax.Join(projectDir, release.ConfigDir)))
		require.NoError(t, medium.Write(release.ConfigPath(projectDir), `
version: 1
sdk:
  spec: docs/openapi.yaml
  languages: [python]
`))

		cfg, err := LoadProjectConfig(medium, projectDir)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		assert.Equal(t, "openapi.yaml", cfg.Spec)
		assert.Equal(t, []string{"typescript"}, cfg.Languages)
	})
}
