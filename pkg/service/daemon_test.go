package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldSkipWatchPath_Good(t *testing.T) {
	projectDir := t.TempDir()

	assert.True(t, shouldSkipWatchPath(projectDir, filepath.Join(projectDir, ".git", "HEAD")))
	assert.True(t, shouldSkipWatchPath(projectDir, filepath.Join(projectDir, "dist", "core-build")))
	assert.True(t, shouldSkipWatchPath(projectDir, filepath.Join(projectDir, ".core", "cache", "state.json")))
	assert.False(t, shouldSkipWatchPath(projectDir, filepath.Join(projectDir, ".core", "build.yaml")))
	assert.False(t, shouldSkipWatchPath(projectDir, filepath.Join(projectDir, "cmd", "main.go")))
}

func TestSnapshotFiles_ExcludesGeneratedOutputs_Good(t *testing.T) {
	projectDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "cmd"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "dist"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "cmd", "main.go"), []byte("package main"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "dist", "app"), []byte("binary"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, ".git", "HEAD"), []byte("ref: refs/heads/dev"), 0o644))

	snapshot, err := snapshotFiles(Config{ProjectDir: projectDir, WatchPaths: []string{projectDir}})
	require.NoError(t, err)

	assert.Contains(t, snapshot, filepath.Join(projectDir, "cmd", "main.go"))
	assert.NotContains(t, snapshot, filepath.Join(projectDir, "dist", "app"))
	assert.NotContains(t, snapshot, filepath.Join(projectDir, ".git", "HEAD"))
}

func TestDiffSnapshots_Good(t *testing.T) {
	now := time.Now()
	before := map[string]time.Time{
		"/tmp/a": now,
		"/tmp/b": now,
	}
	after := map[string]time.Time{
		"/tmp/a": now.Add(time.Second),
		"/tmp/c": now,
	}

	changed := diffSnapshots(before, after)

	assert.Equal(t, []string{"/tmp/a", "/tmp/b", "/tmp/c"}, changed)
}
