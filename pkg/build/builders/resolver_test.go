package builders

import (
	"io/fs"
	"testing"

	"dappco.re/go/core/build/pkg/build"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveBuilder_Good(t *testing.T) {
	t.Run("returns Go builder for go project type", func(t *testing.T) {
		builder, err := ResolveBuilder(build.ProjectTypeGo)
		require.NoError(t, err)
		assert.Equal(t, "go", builder.Name())
	})

	t.Run("returns Docker builder for docker project type", func(t *testing.T) {
		builder, err := ResolveBuilder(build.ProjectTypeDocker)
		require.NoError(t, err)
		assert.Equal(t, "docker", builder.Name())
	})
}

func TestResolveBuilder_Bad(t *testing.T) {
	_, err := ResolveBuilder(build.ProjectType("unknown"))
	assert.ErrorIs(t, err, fs.ErrNotExist)
}
