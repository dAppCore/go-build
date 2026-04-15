package build

import (
	"testing"

	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuild_RuntimeConfigFromBuildConfig_Good(t *testing.T) {
	source := &BuildConfig{
		Project: Project{
			Name:   "Core",
			Main:   "./cmd/core",
			Binary: "core",
		},
		Build: Build{
			CGO:            true,
			Obfuscate:      true,
			DenoBuild:      "deno task bundle",
			NSIS:           true,
			WebView2:       "embed",
			Flags:          []string{"-mod=readonly"},
			LDFlags:        []string{"-s", "-w"},
			BuildTags:      []string{"integration"},
			Env:            []string{"FOO=bar"},
			Cache:          CacheConfig{Enabled: true, Paths: []string{"/tmp/go-build"}},
			Dockerfile:     "build/Dockerfile",
			Registry:       "ghcr.io",
			Image:          "host-uk/core",
			Tags:           []string{"latest"},
			BuildArgs:      map[string]string{"VERSION": "1.2.3"},
			Push:           false,
			Load:           true,
			LinuxKitConfig: ".core/linuxkit/core.yaml",
			Formats:        []string{"iso", "qcow2"},
		},
		LinuxKit: LinuxKitConfig{
			Base:     "core-dev",
			Packages: []string{"git"},
			Mounts:   []string{"/workspace"},
			GPU:      true,
			Formats:  []string{"oci", "apple"},
			Registry: "ghcr.io/dappcore",
		},
	}

	cfg := RuntimeConfigFromBuildConfig(io.Local, "/workspace/core", "/workspace/core/dist", "core-bin", source, true, "override/image", "v1.2.3")
	require.NotNil(t, cfg)

	assert.Equal(t, io.Local, cfg.FS)
	assert.Equal(t, source.Project, cfg.Project)
	assert.Equal(t, "/workspace/core", cfg.ProjectDir)
	assert.Equal(t, "/workspace/core/dist", cfg.OutputDir)
	assert.Equal(t, "core-bin", cfg.Name)
	assert.Equal(t, "v1.2.3", cfg.Version)
	assert.Equal(t, []string{"-mod=readonly"}, cfg.Flags)
	assert.Equal(t, []string{"-s", "-w"}, cfg.LDFlags)
	assert.Equal(t, []string{"integration"}, cfg.BuildTags)
	assert.Equal(t, []string{"FOO=bar"}, cfg.Env)
	assert.Equal(t, CacheConfig{Enabled: true, Paths: []string{"/tmp/go-build"}}, cfg.Cache)
	assert.True(t, cfg.CGO)
	assert.True(t, cfg.Obfuscate)
	assert.Equal(t, "deno task bundle", cfg.DenoBuild)
	assert.True(t, cfg.NSIS)
	assert.Equal(t, "embed", cfg.WebView2)
	assert.Equal(t, "build/Dockerfile", cfg.Dockerfile)
	assert.Equal(t, "ghcr.io", cfg.Registry)
	assert.Equal(t, "override/image", cfg.Image)
	assert.Equal(t, []string{"latest"}, cfg.Tags)
	assert.Equal(t, map[string]string{"VERSION": "1.2.3"}, cfg.BuildArgs)
	assert.True(t, cfg.Push)
	assert.True(t, cfg.Load)
	assert.Equal(t, ".core/linuxkit/core.yaml", cfg.LinuxKitConfig)
	assert.Equal(t, []string{"iso", "qcow2"}, cfg.Formats)
	assert.Equal(t, LinuxKitConfig{
		Base:     "core-dev",
		Packages: []string{"git"},
		Mounts:   []string{"/workspace"},
		GPU:      true,
		Formats:  []string{"oci", "apple"},
		Registry: "ghcr.io/dappcore",
	}, cfg.LinuxKit)

	cfg.Flags[0] = "-trimpath"
	cfg.LDFlags[0] = "-X"
	cfg.BuildTags[0] = "ui"
	cfg.Env[0] = "BAR=baz"
	cfg.Tags[0] = "stable"
	cfg.BuildArgs["VERSION"] = "2.0.0"
	cfg.LinuxKit.Packages[0] = "task"

	assert.Equal(t, []string{"-mod=readonly"}, source.Build.Flags)
	assert.Equal(t, []string{"-s", "-w"}, source.Build.LDFlags)
	assert.Equal(t, []string{"integration"}, source.Build.BuildTags)
	assert.Equal(t, []string{"FOO=bar"}, source.Build.Env)
	assert.Equal(t, []string{"latest"}, source.Build.Tags)
	assert.Equal(t, map[string]string{"VERSION": "1.2.3"}, source.Build.BuildArgs)
	assert.Equal(t, []string{"git"}, source.LinuxKit.Packages)
}

func TestBuild_RuntimeConfigFromBuildConfig_ExpandsVersionTemplates_Good(t *testing.T) {
	source := &BuildConfig{
		Build: Build{
			Flags:   []string{"-X-build=v{{.Version}}"},
			LDFlags: []string{"-X main.Version={{.Tag}}"},
			Env:     []string{"RELEASE_TAG={{.Tag}}", "IMAGE_TAG=v{{.Version}}"},
		},
	}

	cfg := RuntimeConfigFromBuildConfig(io.Local, "/workspace/core", "/workspace/core/dist", "core-bin", source, false, "", "v1.2.3")
	require.NotNil(t, cfg)

	assert.Equal(t, []string{"-X-build=v1.2.3"}, cfg.Flags)
	assert.Equal(t, []string{"-X main.Version=v1.2.3"}, cfg.LDFlags)
	assert.Equal(t, []string{"RELEASE_TAG=v1.2.3", "IMAGE_TAG=v1.2.3"}, cfg.Env)
	assert.Equal(t, []string{"-X-build=v{{.Version}}"}, source.Build.Flags)
	assert.Equal(t, []string{"-X main.Version={{.Tag}}"}, source.Build.LDFlags)
	assert.Equal(t, []string{"RELEASE_TAG={{.Tag}}", "IMAGE_TAG=v{{.Version}}"}, source.Build.Env)
}
