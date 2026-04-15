package build

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionLinkerFlag_Good(t *testing.T) {
	flag, err := VersionLinkerFlag("v1.2.3-beta.1+exp.sha")
	require.NoError(t, err)
	assert.Equal(t, "-X main.version=v1.2.3-beta.1+exp.sha", flag)
}

func TestVersionLinkerFlag_Bad(t *testing.T) {
	flag, err := VersionLinkerFlag("v1.2.3;rm -rf /")
	assert.Error(t, err)
	assert.Empty(t, flag)
}
