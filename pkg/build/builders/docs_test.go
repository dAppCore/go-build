package builders

import (
	"archive/zip"
	"context"
	stdio "io"
	"os"
	"runtime"
	"testing"

	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocs_DocsBuilderName_Good(t *testing.T) {
	builder := NewDocsBuilder()
	assert.Equal(t, "docs", builder.Name())
}

func TestDocs_DocsBuilderDetect_Good(t *testing.T) {
	fs := io.Local

	t.Run("detects mkdocs.yml", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "mkdocs.yml"), []byte("site_name: Demo\n"), 0o644)
		require.NoError(t, err)

		builder := NewDocsBuilder()
		detected, err := builder.Detect(fs, dir)
		require.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("detects mkdocs.yaml", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "mkdocs.yaml"), []byte("site_name: Demo\n"), 0o644)
		require.NoError(t, err)

		builder := NewDocsBuilder()
		detected, err := builder.Detect(fs, dir)
		require.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("returns false without mkdocs.yml", func(t *testing.T) {
		builder := NewDocsBuilder()
		detected, err := builder.Detect(fs, t.TempDir())
		require.NoError(t, err)
		assert.False(t, detected)
	})
}

func TestDocs_DocsBuilderBuild_Good(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mkdocs test fixture uses a shell script")
	}

	dir := t.TempDir()
	require.NoError(t, ax.WriteFile(ax.Join(dir, "mkdocs.yaml"), []byte("site_name: Demo\n"), 0o644))

	binDir := t.TempDir()
	mkdocsPath := ax.Join(binDir, "mkdocs")
	script := "#!/bin/sh\nset -eu\nsite_dir=\"\"\nwhile [ $# -gt 0 ]; do\n  if [ \"$1\" = \"--site-dir\" ]; then\n    shift\n    site_dir=\"$1\"\n  fi\n  shift\ndone\nmkdir -p \"$site_dir\"\nprintf '%s' 'demo docs' > \"$site_dir/index.html\"\n"
	require.NoError(t, ax.WriteFile(mkdocsPath, []byte(script), 0o755))
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: dir,
		OutputDir:  ax.Join(dir, "dist"),
		Name:       "demo-site",
	}

	builder := NewDocsBuilder()
	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	artifact := artifacts[0]
	assert.Equal(t, "linux", artifact.OS)
	assert.Equal(t, "amd64", artifact.Arch)
	assert.FileExists(t, artifact.Path)

	reader, err := zip.OpenReader(artifact.Path)
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	require.Len(t, reader.File, 1)
	assert.Equal(t, "index.html", reader.File[0].Name)

	file, err := reader.File[0].Open()
	require.NoError(t, err)
	defer func() { _ = file.Close() }()

	data, err := stdio.ReadAll(file)
	require.NoError(t, err)
	assert.Equal(t, "demo docs", string(data))
}

func TestDocs_DocsBuilderBuild_Bad(t *testing.T) {
	builder := NewDocsBuilder()

	t.Run("returns error when config is nil", func(t *testing.T) {
		artifacts, err := builder.Build(context.Background(), nil, []build.Target{{OS: "linux", Arch: "amd64"}})
		require.Error(t, err)
		assert.Nil(t, artifacts)
	})

	t.Run("returns error when mkdocs.yml is missing", func(t *testing.T) {
		cfg := &build.Config{
			FS:         io.Local,
			ProjectDir: t.TempDir(),
			OutputDir:  t.TempDir(),
		}

		artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
		require.Error(t, err)
		assert.Nil(t, artifacts)
	})
}
