package builders

import (
	"os"
	"path/filepath"
	"testing"

	"forge.lthn.ai/core/go-build/pkg/build"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskfileBuilder_Name_Good(t *testing.T) {
	builder := NewTaskfileBuilder()
	assert.Equal(t, "taskfile", builder.Name())
}

func TestTaskfileBuilder_Detect_Good(t *testing.T) {
	fs := io.Local

	t.Run("detects Taskfile.yml", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte("version: '3'\n"), 0644)
		require.NoError(t, err)

		builder := NewTaskfileBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("detects Taskfile.yaml", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "Taskfile.yaml"), []byte("version: '3'\n"), 0644)
		require.NoError(t, err)

		builder := NewTaskfileBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("detects Taskfile (no extension)", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "Taskfile"), []byte("version: '3'\n"), 0644)
		require.NoError(t, err)

		builder := NewTaskfileBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("detects lowercase taskfile.yml", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "taskfile.yml"), []byte("version: '3'\n"), 0644)
		require.NoError(t, err)

		builder := NewTaskfileBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("detects lowercase taskfile.yaml", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "taskfile.yaml"), []byte("version: '3'\n"), 0644)
		require.NoError(t, err)

		builder := NewTaskfileBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		dir := t.TempDir()

		builder := NewTaskfileBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})

	t.Run("returns false for non-Taskfile project", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte("all:\n\techo hello\n"), 0644)
		require.NoError(t, err)

		builder := NewTaskfileBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})

	t.Run("does not match Taskfile in subdirectory", func(t *testing.T) {
		dir := t.TempDir()
		subDir := filepath.Join(dir, "subdir")
		require.NoError(t, os.MkdirAll(subDir, 0755))
		err := os.WriteFile(filepath.Join(subDir, "Taskfile.yml"), []byte("version: '3'\n"), 0644)
		require.NoError(t, err)

		builder := NewTaskfileBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})
}

func TestTaskfileBuilder_FindArtifacts_Good(t *testing.T) {
	fs := io.Local
	builder := NewTaskfileBuilder()

	t.Run("finds files in output directory", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "myapp"), []byte("binary"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "myapp.tar.gz"), []byte("archive"), 0644))

		artifacts := builder.findArtifacts(fs, dir)
		assert.Len(t, artifacts, 2)
	})

	t.Run("skips hidden files", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "myapp"), []byte("binary"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".hidden"), []byte("hidden"), 0644))

		artifacts := builder.findArtifacts(fs, dir)
		assert.Len(t, artifacts, 1)
		assert.Contains(t, artifacts[0].Path, "myapp")
	})

	t.Run("skips CHECKSUMS.txt", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "myapp"), []byte("binary"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "CHECKSUMS.txt"), []byte("sha256"), 0644))

		artifacts := builder.findArtifacts(fs, dir)
		assert.Len(t, artifacts, 1)
		assert.Contains(t, artifacts[0].Path, "myapp")
	})

	t.Run("skips directories", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "myapp"), []byte("binary"), 0755))
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "subdir"), 0755))

		artifacts := builder.findArtifacts(fs, dir)
		assert.Len(t, artifacts, 1)
	})

	t.Run("returns empty for empty directory", func(t *testing.T) {
		dir := t.TempDir()

		artifacts := builder.findArtifacts(fs, dir)
		assert.Empty(t, artifacts)
	})

	t.Run("returns empty for nonexistent directory", func(t *testing.T) {
		artifacts := builder.findArtifacts(fs, "/nonexistent/path")
		assert.Empty(t, artifacts)
	})
}

func TestTaskfileBuilder_FindArtifactsForTarget_Good(t *testing.T) {
	fs := io.Local
	builder := NewTaskfileBuilder()

	t.Run("finds artifacts in platform subdirectory", func(t *testing.T) {
		dir := t.TempDir()
		platformDir := filepath.Join(dir, "linux_amd64")
		require.NoError(t, os.MkdirAll(platformDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(platformDir, "myapp"), []byte("binary"), 0755))

		target := build.Target{OS: "linux", Arch: "amd64"}
		artifacts := builder.findArtifactsForTarget(fs, dir, target)
		assert.Len(t, artifacts, 1)
		assert.Equal(t, "linux", artifacts[0].OS)
		assert.Equal(t, "amd64", artifacts[0].Arch)
	})

	t.Run("finds artifacts by name pattern in root", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "myapp-linux-amd64"), []byte("binary"), 0755))

		target := build.Target{OS: "linux", Arch: "amd64"}
		artifacts := builder.findArtifactsForTarget(fs, dir, target)
		assert.NotEmpty(t, artifacts)
	})

	t.Run("returns empty when no matching artifacts", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "myapp"), []byte("binary"), 0755))

		target := build.Target{OS: "linux", Arch: "arm64"}
		artifacts := builder.findArtifactsForTarget(fs, dir, target)
		assert.Empty(t, artifacts)
	})

	t.Run("handles .app bundles on darwin", func(t *testing.T) {
		dir := t.TempDir()
		platformDir := filepath.Join(dir, "darwin_arm64")
		appDir := filepath.Join(platformDir, "MyApp.app")
		require.NoError(t, os.MkdirAll(appDir, 0755))

		target := build.Target{OS: "darwin", Arch: "arm64"}
		artifacts := builder.findArtifactsForTarget(fs, dir, target)
		assert.Len(t, artifacts, 1)
		assert.Contains(t, artifacts[0].Path, "MyApp.app")
	})
}

func TestTaskfileBuilder_MatchPattern_Good(t *testing.T) {
	builder := NewTaskfileBuilder()

	t.Run("matches simple glob", func(t *testing.T) {
		assert.True(t, builder.matchPattern("myapp-linux-amd64", "*-linux-amd64"))
	})

	t.Run("does not match different pattern", func(t *testing.T) {
		assert.False(t, builder.matchPattern("myapp-linux-amd64", "*-darwin-arm64"))
	})

	t.Run("matches wildcard", func(t *testing.T) {
		assert.True(t, builder.matchPattern("test_linux_arm64.bin", "*_linux_arm64*"))
	})
}

func TestTaskfileBuilder_Interface_Good(t *testing.T) {
	// Verify TaskfileBuilder implements Builder interface
	var _ build.Builder = (*TaskfileBuilder)(nil)
	var _ build.Builder = NewTaskfileBuilder()
}
