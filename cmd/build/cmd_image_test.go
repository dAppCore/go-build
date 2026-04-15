package buildcmd

import (
	"context"
	"os"
	"testing"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/build/pkg/build/builders"
	"dappco.re/go/core/io"
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

func setupFakeDockerImageCLI(t *testing.T, binDir string) {
	t.Helper()

	script := `#!/bin/sh
set -eu

log_file="${DOCKER_LOG:-}"

log() {
	if [ -n "$log_file" ]; then
		printf '%s\n' "$1" >> "$log_file"
	fi
}

case "${1:-}" in
	build)
		shift
		log "docker build $*"
		;;
	image)
		shift
		case "${1:-}" in
			load)
				shift
				log "docker image load $*"
				echo "Loaded image: imported:latest"
				;;
			tag)
				shift
				log "docker image tag $*"
				;;
			push)
				shift
				log "docker image push $*"
				;;
			*)
				log "docker image $*"
				;;
		esac
		;;
	*)
		log "docker $*"
		;;
esac
`

	require.NoError(t, ax.WriteFile(ax.Join(binDir, "docker"), []byte(script), 0o755))
}

func TestBuildCmd_AddImageCommand_Good(t *testing.T) {
	c := core.New()

	AddImageCommand(c)

	assert.True(t, c.Command("build/image").OK)
}

func TestBuildCmd_parseImageFormats_Good(t *testing.T) {
	assert.Equal(t, []string{"oci", "apple"}, parseImageFormats(" OCI , apple,Apple, oci "))
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
	setupFakeDockerImageCLI(t, binDir)
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

func TestBuildCmd_allImageArtifactsExist_RequiresMatchingCacheMetadata_Good(t *testing.T) {
	outputDir := t.TempDir()
	imageName := "core-dev"
	builder := builders.NewLinuxKitImageBuilder()
	cfg := build.LinuxKitConfig{
		Base:     "core-dev",
		Formats:  []string{"oci", "apple"},
		Packages: []string{"git", "task"},
		Mounts:   []string{"/workspace"},
	}

	require.NoError(t, ax.WriteFile(ax.Join(outputDir, "core-dev.tar"), []byte("oci image"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(outputDir, "core-dev.aci"), []byte("apple image"), 0o644))
	require.NoError(t, writeImageBuildCacheMetadata(io.Local, outputDir, imageName, cfg, "v1.2.3"))

	assert.True(t, allImageArtifactsExist(io.Local, builder, outputDir, imageName, cfg, "v1.2.3"))
	assert.False(t, allImageArtifactsExist(io.Local, builder, outputDir, imageName, cfg, "v1.2.4"))

	changedCfg := cfg
	changedCfg.GPU = true
	assert.False(t, allImageArtifactsExist(io.Local, builder, outputDir, imageName, changedCfg, "v1.2.3"))

	require.NoError(t, io.Local.Delete(imageBuildCacheMetadataPath(outputDir, imageName)))
	assert.False(t, allImageArtifactsExist(io.Local, builder, outputDir, imageName, cfg, "v1.2.3"))
}

func TestBuildCmd_allImageArtifactsExist_ValidatesVersionlessCacheMetadata_Good(t *testing.T) {
	outputDir := t.TempDir()
	imageName := "core-dev"
	builder := builders.NewLinuxKitImageBuilder()
	cfg := build.LinuxKitConfig{
		Base:     "core-dev",
		Formats:  []string{"oci", "apple"},
		Packages: []string{"git", "task"},
		Mounts:   []string{"/workspace"},
	}

	require.NoError(t, ax.WriteFile(ax.Join(outputDir, "core-dev.tar"), []byte("oci image"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(outputDir, "core-dev.aci"), []byte("apple image"), 0o644))
	require.NoError(t, writeImageBuildCacheMetadata(io.Local, outputDir, imageName, cfg, ""))

	assert.True(t, allImageArtifactsExist(io.Local, builder, outputDir, imageName, cfg, ""))

	changedCfg := cfg
	changedCfg.GPU = true
	assert.False(t, allImageArtifactsExist(io.Local, builder, outputDir, imageName, changedCfg, ""))
}

func TestBuildCmd_retainVersionedImageArtifacts_Good(t *testing.T) {
	outputDir := t.TempDir()
	tarPath := ax.Join(outputDir, "core-dev.tar")
	aciPath := ax.Join(outputDir, "core-dev.aci")

	require.NoError(t, ax.WriteFile(tarPath, []byte("oci image"), 0o644))
	require.NoError(t, ax.WriteFile(aciPath, []byte("apple image"), 0o644))

	versionedPaths, err := retainVersionedImageArtifacts(io.Local, []build.Artifact{
		{Path: tarPath},
		{Path: aciPath},
	}, "v1.2.3")
	require.NoError(t, err)

	expected := []string{
		ax.Join(outputDir, "core-dev-1.2.3.tar"),
		ax.Join(outputDir, "core-dev-1.2.3.aci"),
	}
	assert.ElementsMatch(t, expected, versionedPaths)

	for _, path := range expected {
		assert.FileExists(t, path)
	}
}

func TestBuildCmd_publishOCIImageArchive_Good(t *testing.T) {
	binDir := t.TempDir()
	logPath := ax.Join(t.TempDir(), "docker.log")
	setupFakeDockerImageCLI(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("DOCKER_LOG", logPath)

	projectDir := t.TempDir()
	artifactPath := ax.Join(projectDir, "core-dev.tar")
	require.NoError(t, ax.WriteFile(artifactPath, []byte("oci image"), 0o644))

	ref, err := publishOCIImageArchive(context.Background(), projectDir, artifactPath, "ghcr.io/dappcore", "core-dev", "v1.2.3")
	require.NoError(t, err)
	assert.Equal(t, "ghcr.io/dappcore/core-dev:1.2.3", ref)

	logContent, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(logContent), "docker image load --input "+artifactPath)
	assert.Contains(t, string(logContent), "docker image tag imported:latest ghcr.io/dappcore/core-dev:1.2.3")
	assert.Contains(t, string(logContent), "docker image push ghcr.io/dappcore/core-dev:1.2.3")
}
