package builders

import (
	"testing"

	"dappco.re/go/core/build/internal/ax"

	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinuxKit_LinuxKitBuilderName_Good(t *testing.T) {
	builder := NewLinuxKitBuilder()
	assert.Equal(t, "linuxkit", builder.Name())
}

func TestLinuxKit_LinuxKitBuilderDetect_Good(t *testing.T) {
	fs := io.Local

	t.Run("detects linuxkit.yml in root", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "linuxkit.yml"), []byte("kernel:\n  image: test\n"), 0644)
		require.NoError(t, err)

		builder := NewLinuxKitBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("detects linuxkit.yaml in root", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "linuxkit.yaml"), []byte("kernel:\n  image: test\n"), 0644)
		require.NoError(t, err)

		builder := NewLinuxKitBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("detects .core/linuxkit/*.yml", func(t *testing.T) {
		dir := t.TempDir()
		lkDir := ax.Join(dir, ".core", "linuxkit")
		require.NoError(t, ax.MkdirAll(lkDir, 0755))
		err := ax.WriteFile(ax.Join(lkDir, "server.yml"), []byte("kernel:\n  image: test\n"), 0644)
		require.NoError(t, err)

		builder := NewLinuxKitBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("detects .core/linuxkit/*.yaml", func(t *testing.T) {
		dir := t.TempDir()
		lkDir := ax.Join(dir, ".core", "linuxkit")
		require.NoError(t, ax.MkdirAll(lkDir, 0755))
		err := ax.WriteFile(ax.Join(lkDir, "server.yaml"), []byte("kernel:\n  image: test\n"), 0644)
		require.NoError(t, err)

		builder := NewLinuxKitBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("detects .core/linuxkit with multiple yml files", func(t *testing.T) {
		dir := t.TempDir()
		lkDir := ax.Join(dir, ".core", "linuxkit")
		require.NoError(t, ax.MkdirAll(lkDir, 0755))
		err := ax.WriteFile(ax.Join(lkDir, "server.yml"), []byte("kernel:\n"), 0644)
		require.NoError(t, err)
		err = ax.WriteFile(ax.Join(lkDir, "desktop.yml"), []byte("kernel:\n"), 0644)
		require.NoError(t, err)

		builder := NewLinuxKitBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		dir := t.TempDir()

		builder := NewLinuxKitBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})

	t.Run("returns false for non-LinuxKit project", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module test"), 0644)
		require.NoError(t, err)

		builder := NewLinuxKitBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})

	t.Run("returns false for empty .core/linuxkit directory", func(t *testing.T) {
		dir := t.TempDir()
		lkDir := ax.Join(dir, ".core", "linuxkit")
		require.NoError(t, ax.MkdirAll(lkDir, 0755))

		builder := NewLinuxKitBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})

	t.Run("returns false when .core/linuxkit has only non-yml files", func(t *testing.T) {
		dir := t.TempDir()
		lkDir := ax.Join(dir, ".core", "linuxkit")
		require.NoError(t, ax.MkdirAll(lkDir, 0755))
		err := ax.WriteFile(ax.Join(lkDir, "README.md"), []byte("# LinuxKit\n"), 0644)
		require.NoError(t, err)

		builder := NewLinuxKitBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})

	t.Run("returns false when .core/linuxkit has only non-yaml files", func(t *testing.T) {
		dir := t.TempDir()
		lkDir := ax.Join(dir, ".core", "linuxkit")
		require.NoError(t, ax.MkdirAll(lkDir, 0755))
		err := ax.WriteFile(ax.Join(lkDir, "README.md"), []byte("# LinuxKit\n"), 0644)
		require.NoError(t, err)

		builder := NewLinuxKitBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})

	t.Run("ignores subdirectories in .core/linuxkit", func(t *testing.T) {
		dir := t.TempDir()
		lkDir := ax.Join(dir, ".core", "linuxkit")
		subDir := ax.Join(lkDir, "subdir")
		require.NoError(t, ax.MkdirAll(subDir, 0755))
		// Put yml in subdir only, not in lkDir itself
		err := ax.WriteFile(ax.Join(subDir, "server.yml"), []byte("kernel:\n"), 0644)
		require.NoError(t, err)

		builder := NewLinuxKitBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})
}

func TestLinuxKit_LinuxKitBuilderGetFormatExtension_Good(t *testing.T) {
	builder := NewLinuxKitBuilder()

	tests := []struct {
		format   string
		expected string
	}{
		{"iso", ".iso"},
		{"iso-bios", ".iso"},
		{"iso-efi", ".iso"},
		{"raw", ".raw"},
		{"raw-bios", ".raw"},
		{"raw-efi", ".raw"},
		{"qcow2", ".qcow2"},
		{"qcow2-bios", ".qcow2"},
		{"qcow2-efi", ".qcow2"},
		{"vmdk", ".vmdk"},
		{"vhd", ".vhd"},
		{"gcp", ".img.tar.gz"},
		{"aws", ".raw"},
		{"docker", ".docker.tar"},
		{"tar", ".tar"},
		{"kernel+initrd", "-initrd.img"},
		{"custom", ".custom"},
	}

	for _, tc := range tests {
		t.Run(tc.format, func(t *testing.T) {
			ext := builder.getFormatExtension(tc.format)
			assert.Equal(t, tc.expected, ext)
		})
	}
}

func TestLinuxKit_LinuxKitBuilderGetArtifactPath_Good(t *testing.T) {
	builder := NewLinuxKitBuilder()

	t.Run("constructs correct path", func(t *testing.T) {
		path := builder.getArtifactPath("/dist", "server-amd64", "iso")
		assert.Equal(t, "/dist/server-amd64.iso", path)
	})

	t.Run("constructs correct path for qcow2", func(t *testing.T) {
		path := builder.getArtifactPath("/output/linuxkit", "server-arm64", "qcow2-bios")
		assert.Equal(t, "/output/linuxkit/server-arm64.qcow2", path)
	})

	t.Run("constructs correct path for docker images", func(t *testing.T) {
		path := builder.getArtifactPath("/output/linuxkit", "server-amd64", "docker")
		assert.Equal(t, "/output/linuxkit/server-amd64.docker.tar", path)
	})

	t.Run("constructs correct path for kernel+initrd images", func(t *testing.T) {
		path := builder.getArtifactPath("/output/linuxkit", "server-amd64", "kernel+initrd")
		assert.Equal(t, "/output/linuxkit/server-amd64-initrd.img", path)
	})
}

func TestLinuxKit_LinuxKitBuilderBuildLinuxKitArgs_Good(t *testing.T) {
	builder := NewLinuxKitBuilder()

	t.Run("builds args for amd64 without --arch", func(t *testing.T) {
		args := builder.buildLinuxKitArgs("/config.yml", "iso", "output", "/dist", "amd64")
		assert.Contains(t, args, "build")
		assert.Contains(t, args, "--format")
		assert.Contains(t, args, "iso")
		assert.Contains(t, args, "--name")
		assert.Contains(t, args, "output")
		assert.Contains(t, args, "--dir")
		assert.Contains(t, args, "/dist")
		assert.Contains(t, args, "/config.yml")
		assert.NotContains(t, args, "--arch")
	})

	t.Run("builds args for arm64 with --arch", func(t *testing.T) {
		args := builder.buildLinuxKitArgs("/config.yml", "qcow2", "output", "/dist", "arm64")
		assert.Contains(t, args, "--arch")
		assert.Contains(t, args, "arm64")
	})
}

func TestLinuxKit_LinuxKitBuilderFindArtifact_Good(t *testing.T) {
	fs := io.Local
	builder := NewLinuxKitBuilder()

	t.Run("finds artifact with exact extension", func(t *testing.T) {
		dir := t.TempDir()
		artifactPath := ax.Join(dir, "server-amd64.iso")
		require.NoError(t, ax.WriteFile(artifactPath, []byte("fake iso"), 0644))

		found := builder.findArtifact(fs, dir, "server-amd64", "iso")
		assert.Equal(t, artifactPath, found)
	})

	t.Run("returns empty for missing artifact", func(t *testing.T) {
		dir := t.TempDir()

		found := builder.findArtifact(fs, dir, "nonexistent", "iso")
		assert.Empty(t, found)
	})

	t.Run("finds artifact with alternate naming", func(t *testing.T) {
		dir := t.TempDir()
		// Create file matching the name prefix + known image extension
		artifactPath := ax.Join(dir, "server-amd64.qcow2")
		require.NoError(t, ax.WriteFile(artifactPath, []byte("fake qcow2"), 0644))

		found := builder.findArtifact(fs, dir, "server-amd64", "qcow2")
		assert.Equal(t, artifactPath, found)
	})

	t.Run("finds cloud image artifacts", func(t *testing.T) {
		dir := t.TempDir()
		artifactPath := ax.Join(dir, "server-amd64-gcp.img.tar.gz")
		require.NoError(t, ax.WriteFile(artifactPath, []byte("fake gcp image"), 0644))

		found := builder.findArtifact(fs, dir, "server-amd64", "gcp")
		assert.Equal(t, artifactPath, found)
	})

	t.Run("finds docker artifacts", func(t *testing.T) {
		dir := t.TempDir()
		artifactPath := ax.Join(dir, "server-amd64.docker.tar")
		require.NoError(t, ax.WriteFile(artifactPath, []byte("fake docker tar"), 0644))

		found := builder.findArtifact(fs, dir, "server-amd64", "docker")
		assert.Equal(t, artifactPath, found)
	})

	t.Run("finds kernel+initrd artifacts", func(t *testing.T) {
		dir := t.TempDir()
		artifactPath := ax.Join(dir, "server-amd64-initrd.img")
		require.NoError(t, ax.WriteFile(artifactPath, []byte("fake initrd"), 0644))

		found := builder.findArtifact(fs, dir, "server-amd64", "kernel+initrd")
		assert.Equal(t, artifactPath, found)
	})
}

func TestLinuxKit_LinuxKitBuilderInterface_Good(t *testing.T) {
	// Verify LinuxKitBuilder implements Builder interface
	var _ build.Builder = (*LinuxKitBuilder)(nil)
	var _ build.Builder = NewLinuxKitBuilder()
}

func TestLinuxKit_LinuxKitBuilderResolveLinuxKitCli_Good(t *testing.T) {
	builder := NewLinuxKitBuilder()
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "linuxkit")
	require.NoError(t, ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755))
	t.Setenv("PATH", "")

	command, err := builder.resolveLinuxKitCli(fallbackPath)
	require.NoError(t, err)
	assert.Equal(t, fallbackPath, command)
}

func TestLinuxKit_LinuxKitBuilderResolveLinuxKitCli_Bad(t *testing.T) {
	builder := NewLinuxKitBuilder()
	t.Setenv("PATH", "")

	_, err := builder.resolveLinuxKitCli(ax.Join(t.TempDir(), "missing-linuxkit"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "linuxkit CLI not found")
}
