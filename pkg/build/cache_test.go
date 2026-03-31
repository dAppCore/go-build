package build

import (
	"testing"

	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCache_SetupCache_Good(t *testing.T) {
	fs := io.NewMockMedium()
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

func TestCache_SetupCache_Good_Disabled(t *testing.T) {
	fs := io.NewMockMedium()
	cfg := &CacheConfig{
		Enabled: false,
		Paths:   []string{"cache/go-build"},
	}

	err := SetupCache(fs, "/workspace/project", cfg)
	require.NoError(t, err)

	assert.Empty(t, fs.Dirs)
	assert.Empty(t, fs.Files)
	assert.Empty(t, cfg.Directory)
	assert.Equal(t, []string{"cache/go-build"}, cfg.Paths)
}

func TestCache_CacheKey_Good(t *testing.T) {
	first := CacheKey("core-build", Target{OS: "linux", Arch: "amd64"}, &CacheConfig{
		KeyPrefix: "main",
		Paths: []string{
			"cache/go-build",
			"cache/go-mod",
		},
		RestoreKeys: []string{
			"main-linux",
		},
	})
	second := CacheKey("core-build", Target{OS: "linux", Arch: "amd64"}, &CacheConfig{
		KeyPrefix: "main",
		Paths: []string{
			"cache/go-mod",
			"cache/go-build",
		},
		RestoreKeys: []string{
			"main-linux",
		},
	})
	third := CacheKey("core-build", Target{OS: "darwin", Arch: "arm64"}, &CacheConfig{
		KeyPrefix: "main",
	})

	assert.Equal(t, first, second)
	assert.NotEqual(t, first, third)
	assert.Contains(t, first, "main-linux-amd64-")
}
