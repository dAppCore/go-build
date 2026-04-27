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

func TestValidateVersionIdentifier_Bad(t *testing.T) {
	assert.NoError(t, ValidateVersionIdentifier("v1.2.3"))
	assert.NoError(t, ValidateVersionIdentifier("dev"))
	assert.Error(t, ValidateVersionIdentifier("v1.2.3\n--flag"))
}

func TestVersionFlags_ValidateVersionIdentifier_Good(t *testing.T) {
	t.Run("accepts empty version", func(t *testing.T) {
		assert.NoError(t, ValidateVersionIdentifier(""))
	})

	t.Run("accepts exact safe version", func(t *testing.T) {
		assert.NoError(t, ValidateVersionIdentifier("v1.2.3-beta.1+exp.sha"))
	})
}

func TestVersionFlags_ValidateVersionIdentifier_Ugly(t *testing.T) {
	t.Run("rejects non-ASCII identifiers", func(t *testing.T) {
		assert.Error(t, ValidateVersionIdentifier("v1.2.3-β"))
	})

	t.Run("rejects shell metacharacters", func(t *testing.T) {
		assert.Error(t, ValidateVersionIdentifier("v1.2.3 && echo unsafe"))
	})

	t.Run("rejects surrounding whitespace", func(t *testing.T) {
		assert.Error(t, ValidateVersionIdentifier("  v1.2.3-beta.1+exp.sha  "))
	})
}

func TestVersionFlags_VersionLinkerFlag_Good(t *testing.T) {
	t.Run("renders exact safe version", func(t *testing.T) {
		flag, err := VersionLinkerFlag("v1.2.3")
		require.NoError(t, err)
		assert.Equal(t, "-X main.version=v1.2.3", flag)
	})
}

func TestVersionFlags_VersionLinkerFlag_Ugly(t *testing.T) {
	t.Run("empty version is a no-op", func(t *testing.T) {
		flag, err := VersionLinkerFlag("")
		require.NoError(t, err)
		assert.Empty(t, flag)
	})

	t.Run("rejects surrounding whitespace", func(t *testing.T) {
		flag, err := VersionLinkerFlag(" v1.2.3 ")
		assert.Error(t, err)
		assert.Empty(t, flag)
	})
}
