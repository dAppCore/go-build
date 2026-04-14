package buildcmd

import (
	"testing"

	"dappco.re/go/core"
	"dappco.re/go/core/build/pkg/release"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCmd_applyReleaseArchiveFormatOverride_Good(t *testing.T) {
	cfg := release.DefaultConfig()

	err := applyReleaseArchiveFormatOverride(cfg, "xz")
	require.NoError(t, err)
	assert.Equal(t, "xz", cfg.Build.ArchiveFormat)
}

func TestBuildCmd_applyReleaseArchiveFormatOverride_Bad(t *testing.T) {
	cfg := release.DefaultConfig()

	err := applyReleaseArchiveFormatOverride(cfg, "bogus")
	require.Error(t, err)
	assert.Equal(t, "", cfg.Build.ArchiveFormat)
}

func TestBuildCmd_AddReleaseCommand_RegistersTopLevelAlias_Good(t *testing.T) {
	c := core.New()

	AddReleaseCommand(c)

	assert.True(t, c.Command("build/release").OK)
	assert.True(t, c.Command("release").OK)
}
