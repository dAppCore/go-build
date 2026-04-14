package projectdetect

import (
	"testing"

	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectProjectType_Good(t *testing.T) {
	fs := io.Local

	t.Run("prefers configured build type from .core/build.yaml even without markers", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.MkdirAll(ax.Join(dir, ".core"), 0o755))
		require.NoError(t, ax.WriteFile(ax.Join(dir, ".core", "build.yaml"), []byte("build:\n  type: docker\n"), 0o644))

		projectType, err := DetectProjectType(fs, dir)
		require.NoError(t, err)
		assert.Equal(t, build.ProjectTypeDocker, projectType)
	})

	t.Run("prefers core marker types over fallback builders", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example"), 0o644))
		require.NoError(t, ax.WriteFile(ax.Join(dir, "Dockerfile"), []byte("FROM alpine"), 0o644))

		projectType, err := DetectProjectType(fs, dir)
		require.NoError(t, err)
		assert.Equal(t, build.ProjectTypeGo, projectType)
	})

	t.Run("detects Go workspaces", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "go.work"), []byte("go 1.22\nuse ./app"), 0o644))

		projectType, err := DetectProjectType(fs, dir)
		require.NoError(t, err)
		assert.Equal(t, build.ProjectTypeGo, projectType)
	})

	t.Run("detects Docker projects", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "Dockerfile"), []byte("FROM alpine"), 0o644))

		projectType, err := DetectProjectType(fs, dir)
		require.NoError(t, err)
		assert.Equal(t, build.ProjectTypeDocker, projectType)
	})

	t.Run("detects LinuxKit projects", func(t *testing.T) {
		dir := t.TempDir()
		linuxkitDir := ax.Join(dir, ".core", "linuxkit")
		require.NoError(t, ax.MkdirAll(linuxkitDir, 0o755))
		require.NoError(t, ax.WriteFile(ax.Join(linuxkitDir, "server.yml"), []byte("kernel:\n  image: test"), 0o644))

		projectType, err := DetectProjectType(fs, dir)
		require.NoError(t, err)
		assert.Equal(t, build.ProjectTypeLinuxKit, projectType)
	})

	t.Run("detects C++ projects", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "CMakeLists.txt"), []byte("cmake_minimum_required(VERSION 3.16)"), 0o644))

		projectType, err := DetectProjectType(fs, dir)
		require.NoError(t, err)
		assert.Equal(t, build.ProjectTypeCPP, projectType)
	})

	t.Run("detects Taskfile projects", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "Taskfile.yml"), []byte("version: '3'"), 0o644))

		projectType, err := DetectProjectType(fs, dir)
		require.NoError(t, err)
		assert.Equal(t, build.ProjectTypeTaskfile, projectType)
	})

	t.Run("detects nested Node.js projects", func(t *testing.T) {
		dir := t.TempDir()
		nested := ax.Join(dir, "apps", "web")
		require.NoError(t, ax.MkdirAll(nested, 0o755))
		require.NoError(t, ax.WriteFile(ax.Join(nested, "package.json"), []byte("{}"), 0o644))

		projectType, err := DetectProjectType(fs, dir)
		require.NoError(t, err)
		assert.Equal(t, build.ProjectTypeNode, projectType)
	})

	t.Run("detects nested Deno projects", func(t *testing.T) {
		dir := t.TempDir()
		nested := ax.Join(dir, "apps", "site")
		require.NoError(t, ax.MkdirAll(nested, 0o755))
		require.NoError(t, ax.WriteFile(ax.Join(nested, "deno.jsonc"), []byte("{}"), 0o644))

		projectType, err := DetectProjectType(fs, dir)
		require.NoError(t, err)
		assert.Equal(t, build.ProjectTypeNode, projectType)
	})

	t.Run("detects Wails projects from go.mod and root package.json", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example"), 0o644))
		require.NoError(t, ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0o644))

		projectType, err := DetectProjectType(fs, dir)
		require.NoError(t, err)
		assert.Equal(t, build.ProjectTypeWails, projectType)
	})

	t.Run("detects Wails monorepos from go.mod and nested frontend manifests", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example"), 0o644))
		nested := ax.Join(dir, "apps", "web")
		require.NoError(t, ax.MkdirAll(nested, 0o755))
		require.NoError(t, ax.WriteFile(ax.Join(nested, "package.json"), []byte("{}"), 0o644))

		projectType, err := DetectProjectType(fs, dir)
		require.NoError(t, err)
		assert.Equal(t, build.ProjectTypeWails, projectType)
	})
}

func TestDetectProjectType_Bad(t *testing.T) {
	fs := io.Local

	t.Run("returns empty type for empty directory", func(t *testing.T) {
		projectType, err := DetectProjectType(fs, t.TempDir())
		require.NoError(t, err)
		assert.Empty(t, projectType)
	})
}
