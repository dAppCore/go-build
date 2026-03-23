package builders

import (
	"os"
	"path/filepath"
	"testing"

	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinuxKitBuilder_Name_Good(t *testing.T) {
	builder := NewLinuxKitBuilder()
	assert.Equal(t, "linuxkit", builder.Name())
}

func TestLinuxKitBuilder_Detect_Good(t *testing.T) {
	fs := io.Local

	t.Run("detects linuxkit.yml in root", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "linuxkit.yml"), []byte("kernel:\n  image: test\n"), 0644)
		require.NoError(t, err)

		builder := NewLinuxKitBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("detects .core/linuxkit/*.yml", func(t *testing.T) {
		dir := t.TempDir()
		lkDir := filepath.Join(dir, ".core", "linuxkit")
		require.NoError(t, os.MkdirAll(lkDir, 0755))
		err := os.WriteFile(filepath.Join(lkDir, "server.yml"), []byte("kernel:\n  image: test\n"), 0644)
		require.NoError(t, err)

		builder := NewLinuxKitBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("detects .core/linuxkit with multiple yml files", func(t *testing.T) {
		dir := t.TempDir()
		lkDir := filepath.Join(dir, ".core", "linuxkit")
		require.NoError(t, os.MkdirAll(lkDir, 0755))
		err := os.WriteFile(filepath.Join(lkDir, "server.yml"), []byte("kernel:\n"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(lkDir, "desktop.yml"), []byte("kernel:\n"), 0644)
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
		err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)
		require.NoError(t, err)

		builder := NewLinuxKitBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})

	t.Run("returns false for empty .core/linuxkit directory", func(t *testing.T) {
		dir := t.TempDir()
		lkDir := filepath.Join(dir, ".core", "linuxkit")
		require.NoError(t, os.MkdirAll(lkDir, 0755))

		builder := NewLinuxKitBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})

	t.Run("returns false when .core/linuxkit has only non-yml files", func(t *testing.T) {
		dir := t.TempDir()
		lkDir := filepath.Join(dir, ".core", "linuxkit")
		require.NoError(t, os.MkdirAll(lkDir, 0755))
		err := os.WriteFile(filepath.Join(lkDir, "README.md"), []byte("# LinuxKit\n"), 0644)
		require.NoError(t, err)

		builder := NewLinuxKitBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})

	t.Run("ignores subdirectories in .core/linuxkit", func(t *testing.T) {
		dir := t.TempDir()
		lkDir := filepath.Join(dir, ".core", "linuxkit")
		subDir := filepath.Join(lkDir, "subdir")
		require.NoError(t, os.MkdirAll(subDir, 0755))
		// Put yml in subdir only, not in lkDir itself
		err := os.WriteFile(filepath.Join(subDir, "server.yml"), []byte("kernel:\n"), 0644)
		require.NoError(t, err)

		builder := NewLinuxKitBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})
}

func TestLinuxKitBuilder_GetFormatExtension_Good(t *testing.T) {
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
		{"custom", ".custom"},
	}

	for _, tc := range tests {
		t.Run(tc.format, func(t *testing.T) {
			ext := builder.getFormatExtension(tc.format)
			assert.Equal(t, tc.expected, ext)
		})
	}
}

func TestLinuxKitBuilder_GetArtifactPath_Good(t *testing.T) {
	builder := NewLinuxKitBuilder()

	t.Run("constructs correct path", func(t *testing.T) {
		path := builder.getArtifactPath("/dist", "server-amd64", "iso")
		assert.Equal(t, "/dist/server-amd64.iso", path)
	})

	t.Run("constructs correct path for qcow2", func(t *testing.T) {
		path := builder.getArtifactPath("/output/linuxkit", "server-arm64", "qcow2-bios")
		assert.Equal(t, "/output/linuxkit/server-arm64.qcow2", path)
	})
}

func TestLinuxKitBuilder_BuildLinuxKitArgs_Good(t *testing.T) {
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

func TestLinuxKitBuilder_FindArtifact_Good(t *testing.T) {
	fs := io.Local
	builder := NewLinuxKitBuilder()

	t.Run("finds artifact with exact extension", func(t *testing.T) {
		dir := t.TempDir()
		artifactPath := filepath.Join(dir, "server-amd64.iso")
		require.NoError(t, os.WriteFile(artifactPath, []byte("fake iso"), 0644))

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
		artifactPath := filepath.Join(dir, "server-amd64.qcow2")
		require.NoError(t, os.WriteFile(artifactPath, []byte("fake qcow2"), 0644))

		found := builder.findArtifact(fs, dir, "server-amd64", "qcow2")
		assert.Equal(t, artifactPath, found)
	})
}

func TestLinuxKitBuilder_Interface_Good(t *testing.T) {
	// Verify LinuxKitBuilder implements Builder interface
	var _ build.Builder = (*LinuxKitBuilder)(nil)
	var _ build.Builder = NewLinuxKitBuilder()
}
