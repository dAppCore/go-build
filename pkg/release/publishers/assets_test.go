package publishers

import (
	"testing"

	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssets_BuildChecksumMap_ParsesRFCArchiveNames_Good(t *testing.T) {
	artifacts := []build.Artifact{
		{Path: "/dist/myapp_darwin_amd64.tar.gz", Checksum: "darwin-amd64"},
		{Path: "/dist/myapp_darwin_arm64.tar.gz", Checksum: "darwin-arm64"},
		{Path: "/dist/myapp_linux_amd64.tar.gz", Checksum: "linux-amd64"},
		{Path: "/dist/myapp_linux_arm64.tar.gz", Checksum: "linux-arm64"},
		{Path: "/dist/myapp_windows_amd64.zip", Checksum: "windows-amd64"},
		{Path: "/dist/myapp_windows_arm64.zip", Checksum: "windows-arm64"},
		{Path: "/dist/CHECKSUMS.txt"},
	}

	checksums := buildChecksumMap(artifacts)

	assert.Equal(t, "darwin-amd64", checksums.DarwinAmd64)
	assert.Equal(t, "darwin-arm64", checksums.DarwinArm64)
	assert.Equal(t, "linux-amd64", checksums.LinuxAmd64)
	assert.Equal(t, "linux-arm64", checksums.LinuxArm64)
	assert.Equal(t, "windows-amd64", checksums.WindowsAmd64)
	assert.Equal(t, "windows-arm64", checksums.WindowsArm64)
	assert.Equal(t, "myapp_darwin_amd64.tar.gz", checksums.DarwinAmd64File)
	assert.Equal(t, "myapp_darwin_arm64.tar.gz", checksums.DarwinArm64File)
	assert.Equal(t, "myapp_linux_amd64.tar.gz", checksums.LinuxAmd64File)
	assert.Equal(t, "myapp_linux_arm64.tar.gz", checksums.LinuxArm64File)
	assert.Equal(t, "myapp_windows_amd64.zip", checksums.WindowsAmd64File)
	assert.Equal(t, "myapp_windows_arm64.zip", checksums.WindowsArm64File)
	assert.Equal(t, "CHECKSUMS.txt", checksums.ChecksumFile)
}

func TestAssets_BuildChecksumMapFromRelease_UsesChecksumFileFallback_Good(t *testing.T) {
	artifactFS := io.NewMemoryMedium()
	require.NoError(t, artifactFS.Write("releases/checksums.txt", ""+
		"abc123  myapp_linux_amd64.tar.gz\n"+
		"def456  myapp_darwin_arm64.tar.gz\n",
	))

	release := &Release{
		Artifacts: []build.Artifact{
			{Path: "releases/myapp_linux_amd64.tar.gz", OS: "linux", Arch: "amd64"},
			{Path: "releases/myapp_darwin_arm64.tar.gz"},
			{Path: "releases/checksums.txt"},
		},
		FS:         artifactFS,
		ArtifactFS: artifactFS,
	}

	checksums := buildChecksumMapFromRelease(release)

	assert.Equal(t, "abc123", checksums.LinuxAmd64)
	assert.Equal(t, "def456", checksums.DarwinArm64)
	assert.Equal(t, "myapp_linux_amd64.tar.gz", checksums.LinuxAmd64File)
	assert.Equal(t, "myapp_darwin_arm64.tar.gz", checksums.DarwinArm64File)
	assert.Equal(t, "checksums.txt", checksums.ChecksumFile)
}
