package builders

import (
	"testing"

	"dappco.re/go/core/build/internal/ax"

	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocker_DockerBuilderName_Good(t *testing.T) {
	builder := NewDockerBuilder()
	assert.Equal(t, "docker", builder.Name())
}

func TestDocker_DockerBuilderDetect_Good(t *testing.T) {
	fs := io.Local

	t.Run("detects Dockerfile", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "Dockerfile"), []byte("FROM alpine\n"), 0644)
		require.NoError(t, err)

		builder := NewDockerBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		dir := t.TempDir()

		builder := NewDockerBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})

	t.Run("returns false for non-Docker project", func(t *testing.T) {
		dir := t.TempDir()
		// Create a Go project instead
		err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module test"), 0644)
		require.NoError(t, err)

		builder := NewDockerBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})

	t.Run("does not match docker-compose.yml", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "docker-compose.yml"), []byte("version: '3'\n"), 0644)
		require.NoError(t, err)

		builder := NewDockerBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})

	t.Run("does not match Dockerfile in subdirectory", func(t *testing.T) {
		dir := t.TempDir()
		subDir := ax.Join(dir, "subdir")
		require.NoError(t, ax.MkdirAll(subDir, 0755))
		err := ax.WriteFile(ax.Join(subDir, "Dockerfile"), []byte("FROM alpine\n"), 0644)
		require.NoError(t, err)

		builder := NewDockerBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})
}

func TestDocker_DockerBuilderInterface_Good(t *testing.T) {
	// Verify DockerBuilder implements Builder interface
	var _ build.Builder = (*DockerBuilder)(nil)
	var _ build.Builder = NewDockerBuilder()
}
