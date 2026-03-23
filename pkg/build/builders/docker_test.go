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

func TestDockerBuilder_Name_Good(t *testing.T) {
	builder := NewDockerBuilder()
	assert.Equal(t, "docker", builder.Name())
}

func TestDockerBuilder_Detect_Good(t *testing.T) {
	fs := io.Local

	t.Run("detects Dockerfile", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM alpine\n"), 0644)
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
		err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)
		require.NoError(t, err)

		builder := NewDockerBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})

	t.Run("does not match docker-compose.yml", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("version: '3'\n"), 0644)
		require.NoError(t, err)

		builder := NewDockerBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})

	t.Run("does not match Dockerfile in subdirectory", func(t *testing.T) {
		dir := t.TempDir()
		subDir := filepath.Join(dir, "subdir")
		require.NoError(t, os.MkdirAll(subDir, 0755))
		err := os.WriteFile(filepath.Join(subDir, "Dockerfile"), []byte("FROM alpine\n"), 0644)
		require.NoError(t, err)

		builder := NewDockerBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})
}

func TestDockerBuilder_Interface_Good(t *testing.T) {
	// Verify DockerBuilder implements Builder interface
	var _ build.Builder = (*DockerBuilder)(nil)
	var _ build.Builder = NewDockerBuilder()
}
