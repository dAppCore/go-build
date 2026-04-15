package builders

import (
	"testing"

	"dappco.re/go/core/build/pkg/build"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolver_InitRegistersDefaultBuilderResolver_Good(t *testing.T) {
	resolver := build.DefaultBuilderResolver()
	require.NotNil(t, resolver)

	builder, err := resolver(build.ProjectTypeGo)
	require.NoError(t, err)
	require.NotNil(t, builder)
	assert.Equal(t, "go", builder.Name())
}
