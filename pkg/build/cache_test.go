package build

import (
	"testing"

	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCache_SetupCache_Good(t *testing.T) {
	fs := io.NewMemoryMedium()
	cfg := &CacheConfig{
		Enabled: true,
		Paths: []string{
			"cache/go-build",
			"cache/go-mod",
		},
	}

	err := SetupCache(fs, "/workspace/project", cfg)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "/workspace/project/.core/cache", cfg.Directory)
	assert.Equal(t, []string{
		"/workspace/project/cache/go-build",
		"/workspace/project/cache/go-mod",
	}, cfg.Paths)

	assert.True(t, fs.Exists("/workspace/project/.core/cache"))
	assert.True(t, fs.Exists("/workspace/project/cache/go-build"))
	assert.True(t, fs.Exists("/workspace/project/cache/go-mod"))
}

func TestCache_SetupBuildCache_Good(t *testing.T) {
	fs := io.NewMemoryMedium()
	cfg := &BuildConfig{
		Build: Build{
			Cache: CacheConfig{
				Enabled: true,
				Paths: []string{
					"cache/go-build",
				},
			},
		},
	}

	err := SetupBuildCache(fs, "/workspace/project", cfg)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "/workspace/project/.core/cache", cfg.Build.Cache.Directory)
	assert.Equal(t, []string{"/workspace/project/cache/go-build"}, cfg.Build.Cache.Paths)
	assert.True(t, fs.Exists("/workspace/project/.core/cache"))
	assert.True(t, fs.Exists("/workspace/project/cache/go-build"))
}

func TestCache_SetupCache_Good_DefaultPathsWhenEnabled(t *testing.T) {
	fs := io.NewMemoryMedium()
	cfg := &CacheConfig{
		Enabled: true,
	}

	err := SetupCache(fs, "/workspace/project", cfg)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "/workspace/project/.core/cache", cfg.Directory)
	assert.Equal(t, []string{
		"/workspace/project/cache/go-build",
		"/workspace/project/cache/go-mod",
	}, cfg.Paths)
	assert.True(t, fs.Exists("/workspace/project/.core/cache"))
	assert.True(t, fs.Exists("/workspace/project/cache/go-build"))
	assert.True(t, fs.Exists("/workspace/project/cache/go-mod"))
}

func TestCache_SetupBuildCache_Good_DefaultPathsWhenEnabled(t *testing.T) {
	fs := io.NewMemoryMedium()
	cfg := &BuildConfig{
		Build: Build{
			Cache: CacheConfig{
				Enabled: true,
			},
		},
	}

	err := SetupBuildCache(fs, "/workspace/project", cfg)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "/workspace/project/.core/cache", cfg.Build.Cache.Directory)
	assert.Equal(t, []string{
		"/workspace/project/cache/go-build",
		"/workspace/project/cache/go-mod",
	}, cfg.Build.Cache.Paths)
	assert.True(t, fs.Exists("/workspace/project/.core/cache"))
	assert.True(t, fs.Exists("/workspace/project/cache/go-build"))
	assert.True(t, fs.Exists("/workspace/project/cache/go-mod"))
}

func TestCache_SetupCache_Good_Disabled(t *testing.T) {
	fs := io.NewMemoryMedium()
	cfg := &CacheConfig{
		Enabled: false,
		Paths:   []string{"cache/go-build"},
	}

	err := SetupCache(fs, "/workspace/project", cfg)
	require.NoError(t, err)

	assert.False(t, fs.Exists("/workspace/project/.core/cache"))
	assert.False(t, fs.Exists("/workspace/project/cache/go-build"))
	assert.Empty(t, cfg.Directory)
	assert.Equal(t, []string{"cache/go-build"}, cfg.Paths)
}

func TestCache_SetupBuildCache_Good_Disabled(t *testing.T) {
	fs := io.NewMemoryMedium()
	cfg := &BuildConfig{
		Build: Build{
			Cache: CacheConfig{
				Enabled: false,
				Paths:   []string{"cache/go-build"},
			},
		},
	}

	err := SetupBuildCache(fs, "/workspace/project", cfg)
	require.NoError(t, err)

	assert.False(t, fs.Exists("/workspace/project/.core/cache"))
	assert.Empty(t, cfg.Build.Cache.Directory)
	assert.Equal(t, []string{"cache/go-build"}, cfg.Build.Cache.Paths)
}

func TestCache_CacheKey_Good(t *testing.T) {
	fs := io.NewMemoryMedium()
	require.NoError(t, fs.Write("/workspace/project/go.sum", "module.example v1.0.0 h1:abc123"))
	require.NoError(t, fs.Write("/workspace/project/go.work.sum", "workspace.example v1.0.0 h1:def456"))

	first := CacheKey(fs, "/workspace/project", "linux", "amd64")
	second := CacheKey(fs, "/workspace/project", "linux", "amd64")
	third := CacheKey(fs, "/workspace/project", "darwin", "arm64")

	assert.Equal(t, first, second)
	assert.NotEqual(t, first, third)
	assert.Contains(t, first, "go-linux-amd64-")
}

func TestCache_CacheKey_Good_GoWorkSumChangesKey(t *testing.T) {
	fs := io.NewMemoryMedium()
	require.NoError(t, fs.Write("/workspace/project/go.sum", "module.example v1.0.0 h1:abc123"))

	baseline := CacheKey(fs, "/workspace/project", "linux", "amd64")
	require.NoError(t, fs.Write("/workspace/project/go.work.sum", "workspace.example v1.0.0 h1:def456"))
	updated := CacheKey(fs, "/workspace/project", "linux", "amd64")

	assert.NotEqual(t, baseline, updated)
}

func TestCache_CacheEnvironment_Good(t *testing.T) {
	t.Run("maps cache directory and Go cache paths to env vars", func(t *testing.T) {
		env := CacheEnvironment(&CacheConfig{
			Enabled: true,
			Paths: []string{
				"/workspace/project/cache/go-build",
				"/workspace/project/cache/go-mod",
				"/workspace/project/cache/go-build",
			},
		})

		assert.Equal(t, []string{
			"GOCACHE=/workspace/project/cache/go-build",
			"GOMODCACHE=/workspace/project/cache/go-mod",
		}, env)
	})

	t.Run("disabled cache returns no env vars", func(t *testing.T) {
		assert.Nil(t, CacheEnvironment(&CacheConfig{Enabled: false}))
	})
}
