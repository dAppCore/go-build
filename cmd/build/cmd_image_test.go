package buildcmd

import (
	"context"
	"os"
	"testing"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupFakeLinuxKitImageCLI(t *testing.T, binDir string) {
	t.Helper()

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
esac

mkdir -p "$dir"
printf 'linuxkit image\n' > "$dir/$name$ext"
`

	require.NoError(t, ax.WriteFile(ax.Join(binDir, "linuxkit"), []byte(script), 0o755))
}

func TestBuildCmd_AddImageCommand_Good(t *testing.T) {
	c := core.New()

	AddImageCommand(c)

	assert.True(t, c.Command("build/image").OK)
}

func TestBuildCmd_parseImageFormats_Good(t *testing.T) {
	assert.Equal(t, []string{"oci", "apple"}, parseImageFormats("oci, apple,oci"))
}

func TestBuildCmd_buildPwaCommandAcceptsPath_Good(t *testing.T) {
	c := core.New()
	AddBuildCommands(c)

	command := c.Command("build/pwa").Value.(*core.Command)

	original := runLocalPwaBuild
	defer func() { runLocalPwaBuild = original }()

	calledPath := ""
	runLocalPwaBuild = func(ctx context.Context, projectDir string) error {
		calledPath = projectDir
		return nil
	}

	opts := core.NewOptions(core.Option{Key: "path", Value: "/tmp/pwa"})
	result := command.Run(opts)
	assert.True(t, result.OK)
	assert.Equal(t, "/tmp/pwa", calledPath)
}

func TestBuildCmd_runBuildImage_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeLinuxKitImageCLI(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	outputDir := t.TempDir()

	err := runBuildImage(ImageBuildRequest{
		Context:   context.Background(),
		Base:      "core-minimal",
		Format:    "oci,apple",
		OutputDir: outputDir,
	})
	require.NoError(t, err)

	assert.FileExists(t, ax.Join(outputDir, "core-minimal.tar"))
	assert.FileExists(t, ax.Join(outputDir, "core-minimal.aci"))

	t.Setenv("PATH", "/definitely-missing")
	err = runBuildImage(ImageBuildRequest{
		Context:   context.Background(),
		Base:      "core-minimal",
		Format:    "oci,apple",
		OutputDir: outputDir,
	})
	require.NoError(t, err)
}
