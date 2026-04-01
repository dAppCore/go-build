package buildcmd

import (
	"encoding/json"
	"testing"

	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCmd_GetBuilder_Good(t *testing.T) {
	t.Run("returns Python builder for python project type", func(t *testing.T) {
		builder, err := getBuilder(build.ProjectTypePython)
		require.NoError(t, err)
		assert.NotNil(t, builder)
		assert.Equal(t, "python", builder.Name())
	})
}

func TestBuildCmd_buildRuntimeConfig_Good(t *testing.T) {
	buildConfig := &build.BuildConfig{
		Project: build.Project{
			Name: "sample",
		},
		Build: build.Build{
			LDFlags:        []string{"-s", "-w"},
			Flags:          []string{"-trimpath"},
			Env:            []string{"FOO=bar"},
			CGO:            true,
			Obfuscate:      true,
			NSIS:           true,
			WebView2:       "embed",
			Dockerfile:     "Dockerfile.custom",
			Registry:       "ghcr.io",
			Image:          "owner/repo",
			Tags:           []string{"latest", "{{.Version}}"},
			BuildArgs:      map[string]string{"VERSION": "{{.Version}}"},
			Push:           true,
			LinuxKitConfig: ".core/linuxkit/server.yml",
			Formats:        []string{"iso", "qcow2"},
		},
	}

	cfg := buildRuntimeConfig(io.Local, "/project", "/project/dist", "binary", buildConfig, false, "")

	assert.Equal(t, []string{"-s", "-w"}, cfg.LDFlags)
	assert.Equal(t, []string{"-trimpath"}, cfg.Flags)
	assert.Equal(t, []string{"FOO=bar"}, cfg.Env)
	assert.True(t, cfg.CGO)
	assert.True(t, cfg.Obfuscate)
	assert.True(t, cfg.NSIS)
	assert.Equal(t, "embed", cfg.WebView2)
	assert.Equal(t, "Dockerfile.custom", cfg.Dockerfile)
	assert.Equal(t, "ghcr.io", cfg.Registry)
	assert.Equal(t, "owner/repo", cfg.Image)
	assert.Equal(t, []string{"latest", "{{.Version}}"}, cfg.Tags)
	assert.Equal(t, map[string]string{"VERSION": "{{.Version}}"}, cfg.BuildArgs)
	assert.True(t, cfg.Push)
	assert.Equal(t, ".core/linuxkit/server.yml", cfg.LinuxKitConfig)
	assert.Equal(t, []string{"iso", "qcow2"}, cfg.Formats)
}

func TestBuildCmd_buildRuntimeConfig_ImageOverride_Good(t *testing.T) {
	buildConfig := &build.BuildConfig{
		Build: build.Build{
			Image: "owner/repo",
		},
	}

	cfg := buildRuntimeConfig(io.Local, "/project", "/project/dist", "binary", buildConfig, true, "cli/image")

	assert.Equal(t, "cli/image", cfg.Image)
	assert.True(t, cfg.Push)
}

func TestBuildCmd_writeArtifactMetadata_Good(t *testing.T) {
	t.Setenv("GITHUB_SHA", "abc1234def5678")
	t.Setenv("GITHUB_REF", "refs/tags/v1.2.3")
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")

	fs := io.Local
	dir := t.TempDir()

	linuxDir := ax.Join(dir, "linux_amd64")
	windowsDir := ax.Join(dir, "windows_amd64")
	require.NoError(t, ax.MkdirAll(linuxDir, 0755))
	require.NoError(t, ax.MkdirAll(windowsDir, 0755))

	artifacts := []build.Artifact{
		{Path: ax.Join(linuxDir, "sample"), OS: "linux", Arch: "amd64"},
		{Path: ax.Join(windowsDir, "sample.exe"), OS: "windows", Arch: "amd64"},
	}

	err := writeArtifactMetadata(fs, "sample", artifacts)
	require.NoError(t, err)

	verifyArtifactMeta := func(path string, expectedOS string, expectedArch string) {
		content, readErr := ax.ReadFile(path)
		require.NoError(t, readErr)

		var meta map[string]any
		require.NoError(t, json.Unmarshal(content, &meta))

		assert.Equal(t, "sample", meta["name"])
		assert.Equal(t, expectedOS, meta["os"])
		assert.Equal(t, expectedArch, meta["arch"])
		assert.Equal(t, "v1.2.3", meta["tag"])
		assert.Equal(t, "owner/repo", meta["repo"])
	}

	verifyArtifactMeta(ax.Join(linuxDir, "artifact_meta.json"), "linux", "amd64")
	verifyArtifactMeta(ax.Join(windowsDir, "artifact_meta.json"), "windows", "amd64")
}
