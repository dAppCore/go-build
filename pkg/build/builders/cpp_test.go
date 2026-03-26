package builders

import (
	"testing"

	"dappco.re/go/core/build/internal/ax"

	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCPP_CPPBuilderName_Good(t *testing.T) {
	builder := NewCPPBuilder()
	assert.Equal(t, "cpp", builder.Name())
}

func TestCPP_CPPBuilderDetect_Good(t *testing.T) {
	fs := io.Local

	t.Run("detects C++ project with CMakeLists.txt", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "CMakeLists.txt"), []byte("cmake_minimum_required(VERSION 3.16)"), 0644)
		require.NoError(t, err)

		builder := NewCPPBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("returns false for non-C++ project", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module test"), 0644)
		require.NoError(t, err)

		builder := NewCPPBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		dir := t.TempDir()

		builder := NewCPPBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})
}

func TestCPP_CPPBuilderBuild_Bad(t *testing.T) {
	t.Run("returns error for nil config", func(t *testing.T) {
		builder := NewCPPBuilder()
		artifacts, err := builder.Build(nil, nil, []build.Target{{OS: "linux", Arch: "amd64"}})
		assert.Error(t, err)
		assert.Nil(t, artifacts)
		assert.Contains(t, err.Error(), "config is nil")
	})
}

func TestCPP_CPPBuilderTargetToProfile_Good(t *testing.T) {
	builder := NewCPPBuilder()

	tests := []struct {
		os, arch string
		expected string
	}{
		{"linux", "amd64", "gcc-linux-x86_64"},
		{"linux", "x86_64", "gcc-linux-x86_64"},
		{"linux", "arm64", "gcc-linux-armv8"},
		{"darwin", "arm64", "apple-clang-armv8"},
		{"darwin", "amd64", "apple-clang-x86_64"},
		{"windows", "amd64", "msvc-194-x86_64"},
	}

	for _, tt := range tests {
		t.Run(tt.os+"/"+tt.arch, func(t *testing.T) {
			profile := builder.targetToProfile(build.Target{OS: tt.os, Arch: tt.arch})
			assert.Equal(t, tt.expected, profile)
		})
	}
}

func TestCPP_CPPBuilderTargetToProfile_Bad(t *testing.T) {
	builder := NewCPPBuilder()

	t.Run("returns empty for unknown target", func(t *testing.T) {
		profile := builder.targetToProfile(build.Target{OS: "plan9", Arch: "mips"})
		assert.Empty(t, profile)
	})
}

func TestCPP_CPPBuilderFindArtifacts_Good(t *testing.T) {
	fs := io.Local

	t.Run("finds packages in build/packages", func(t *testing.T) {
		dir := t.TempDir()
		packagesDir := ax.Join(dir, "build", "packages")
		require.NoError(t, ax.MkdirAll(packagesDir, 0755))

		// Create mock package files
		require.NoError(t, ax.WriteFile(ax.Join(packagesDir, "test-1.0-linux-x86_64.tar.xz"), []byte("pkg"), 0644))
		require.NoError(t, ax.WriteFile(ax.Join(packagesDir, "test-1.0-linux-x86_64.tar.xz.sha256"), []byte("checksum"), 0644))
		require.NoError(t, ax.WriteFile(ax.Join(packagesDir, "test-1.0-linux-x86_64.rpm"), []byte("rpm"), 0644))

		builder := NewCPPBuilder()
		target := build.Target{OS: "linux", Arch: "amd64"}
		artifacts, err := builder.findArtifacts(fs, dir, target)
		require.NoError(t, err)

		// Should find tar.xz and rpm but not sha256
		assert.Len(t, artifacts, 2)
		for _, a := range artifacts {
			assert.Equal(t, "linux", a.OS)
			assert.Equal(t, "amd64", a.Arch)
			assert.False(t, ax.Ext(a.Path) == ".sha256")
		}
	})

	t.Run("falls back to binaries in build/release/src", func(t *testing.T) {
		dir := t.TempDir()
		binDir := ax.Join(dir, "build", "release", "src")
		require.NoError(t, ax.MkdirAll(binDir, 0755))

		// Create mock binary (executable)
		binPath := ax.Join(binDir, "test-daemon")
		require.NoError(t, ax.WriteFile(binPath, []byte("binary"), 0755))

		// Create a library (should be skipped)
		require.NoError(t, ax.WriteFile(ax.Join(binDir, "libcrypto.a"), []byte("lib"), 0644))

		builder := NewCPPBuilder()
		target := build.Target{OS: "linux", Arch: "amd64"}
		artifacts, err := builder.findArtifacts(fs, dir, target)
		require.NoError(t, err)

		// Should find the executable but not the library
		assert.Len(t, artifacts, 1)
		assert.Contains(t, artifacts[0].Path, "test-daemon")
	})
}

func TestCPP_CPPBuilderInterface_Good(t *testing.T) {
	var _ build.Builder = (*CPPBuilder)(nil)
	var _ build.Builder = NewCPPBuilder()
}
