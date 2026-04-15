package build

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuild_DefaultLinuxKitConfig_Good(t *testing.T) {
	cfg := DefaultLinuxKitConfig()

	assert.Equal(t, "core-dev", cfg.Base)
	assert.Equal(t, []string{"/workspace"}, cfg.Mounts)
	assert.Equal(t, []string{"oci", "apple"}, cfg.Formats)
	assert.False(t, cfg.GPU)
}

func TestBuild_LinuxKit_Good(t *testing.T) {
	image := LinuxKit(
		WithBase("core-ml"),
		WithPackages("git", "task"),
		WithMount("/src"),
		WithGPU(true),
		WithFormats("oci"),
		WithRegistry("ghcr.io/dappcore"),
	)

	require.NotNil(t, image)
	assert.Equal(t, LinuxKitConfig{
		Base:     "core-ml",
		Packages: []string{"git", "task"},
		Mounts:   []string{"/workspace", "/src"},
		GPU:      true,
		Formats:  []string{"oci"},
		Registry: "ghcr.io/dappcore",
	}, image.Config)
}

func TestBuild_LinuxKitBaseTemplate_Good(t *testing.T) {
	images := LinuxKitBaseImages()
	require.Len(t, images, 3)

	content, err := LinuxKitBaseTemplate("core-dev")
	require.NoError(t, err)
	assert.Contains(t, content, "core-dev")
}
