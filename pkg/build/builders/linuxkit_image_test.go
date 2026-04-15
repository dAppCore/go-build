package builders

import (
	"context"
	"os"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupFakeLinuxKitImageToolchain(t *testing.T, binDir string) {
	t.Helper()

	dockerScript := `#!/bin/sh
exit 0
`
	require.NoError(t, ax.WriteFile(ax.Join(binDir, "docker"), []byte(dockerScript), 0o755))

	script := `#!/bin/sh
set -eu

format=""
dir=""
name=""
while [ $# -gt 0 ]; do
	case "$1" in
	build)
		;;
	--format)
		shift
		format="${1:-}"
		;;
	--dir)
		shift
		dir="${1:-}"
		;;
	--name)
		shift
		name="${1:-}"
		;;
	esac
	shift
done

ext=".img"
case "$format" in
	tar)
		ext=".tar"
		;;
	iso|iso-bios|iso-efi)
		ext=".iso"
		;;
	raw|raw-bios|raw-efi)
		ext=".raw"
		;;
	qcow2|qcow2-bios|qcow2-efi)
		ext=".qcow2"
		;;
esac

mkdir -p "$dir"
printf 'linuxkit image\n' > "$dir/$name$ext"
`

	require.NoError(t, ax.WriteFile(ax.Join(binDir, "linuxkit"), []byte(script), 0o755))
}

func TestLinuxKitImage_LinuxKitImageBuilderName_Good(t *testing.T) {
	builder := NewLinuxKitImageBuilder()
	assert.Equal(t, "linuxkit-image", builder.Name())
}

func TestLinuxKitImage_LinuxKitImageBuilderArtifactPath_Good(t *testing.T) {
	builder := NewLinuxKitImageBuilder()

	assert.Equal(t, "/dist/core-dev.tar", builder.ArtifactPath("/dist", "core-dev", "oci"))
	assert.Equal(t, "/dist/core-dev.aci", builder.ArtifactPath("/dist", "core-dev", "apple"))
	assert.Equal(t, "/dist/core-dev.iso", builder.ArtifactPath("/dist", "core-dev", "iso"))
}

func TestLinuxKitImage_RenderTemplateUsesImmutableServiceImage_Good(t *testing.T) {
	builder := NewLinuxKitImageBuilder()
	baseImage, ok := build.LookupLinuxKitBaseImage("core-dev")
	require.True(t, ok)

	rendered, err := builder.renderTemplate(baseImage, build.LinuxKitConfig{
		Base:     "core-dev",
		Mounts:   []string{"/workspace"},
		Formats:  []string{"oci"},
		Packages: []string{"gh"},
	}, "v1.2.3", "core-build-linuxkit/core-dev:test")
	require.NoError(t, err)

	assert.Contains(t, rendered, `image: "core-build-linuxkit/core-dev:test"`)
	assert.Contains(t, rendered, "tail -f /dev/null")
	assert.NotContains(t, rendered, "apk add --no-cache")
}

func TestLinuxKitImage_RenderTemplateRestoresDefaultWorkspaceMount_Good(t *testing.T) {
	builder := NewLinuxKitImageBuilder()
	baseImage, ok := build.LookupLinuxKitBaseImage("core-dev")
	require.True(t, ok)

	rendered, err := builder.renderTemplate(baseImage, build.LinuxKitConfig{
		Base:    "core-dev",
		Mounts:  []string{""},
		Formats: []string{"oci"},
	}, "v1.2.3", "core-build-linuxkit/core-dev:test")
	require.NoError(t, err)

	assert.Contains(t, rendered, "binds:")
	assert.Contains(t, rendered, "- /workspace:/workspace")
}

func TestLinuxKitImage_LinuxKitImageBuilderBuild_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeLinuxKitImageToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectDir := t.TempDir()
	outputDir := t.TempDir()

	builder := NewLinuxKitImageBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "core-dev",
		Version:    "v1.2.3",
		LinuxKit: build.LinuxKitConfig{
			Base:     "core-dev",
			Packages: []string{"gh"},
			Mounts:   []string{"/workspace"},
			Formats:  []string{"oci", "apple"},
		},
	}

	artifacts, err := builder.Build(context.Background(), cfg)
	require.NoError(t, err)
	require.Len(t, artifacts, 2)

	assert.FileExists(t, ax.Join(outputDir, "core-dev.tar"))
	assert.FileExists(t, ax.Join(outputDir, "core-dev.aci"))
	assert.NoFileExists(t, ax.Join(outputDir, ".core-dev-linuxkit.yml"))
}
