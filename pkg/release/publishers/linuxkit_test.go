package publishers

import (
	"context"
	"testing"

	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinuxKit_LinuxKitPublisherName_Good(t *testing.T) {
	t.Run("returns linuxkit", func(t *testing.T) {
		p := NewLinuxKitPublisher()
		assert.Equal(t, "linuxkit", p.Name())
	})
}

func TestLinuxKit_LinuxKitPublisherParseConfig_Good(t *testing.T) {
	p := NewLinuxKitPublisher()

	t.Run("uses defaults when no extended config", func(t *testing.T) {
		pubCfg := PublisherConfig{Type: "linuxkit"}
		cfg := p.parseConfig(pubCfg, "/project")

		assert.Equal(t, "/project/.core/linuxkit/server.yml", cfg.Config)
		assert.Equal(t, []string{"iso"}, cfg.Formats)
		assert.Equal(t, []string{"linux/amd64"}, cfg.Platforms)
	})

	t.Run("parses extended config", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "linuxkit",
			Extended: map[string]any{
				"config":    ".core/linuxkit/custom.yml",
				"formats":   []any{"iso", "qcow2", "vmdk"},
				"platforms": []any{"linux/amd64", "linux/arm64"},
			},
		}
		cfg := p.parseConfig(pubCfg, "/project")

		assert.Equal(t, "/project/.core/linuxkit/custom.yml", cfg.Config)
		assert.Equal(t, []string{"iso", "qcow2", "vmdk"}, cfg.Formats)
		assert.Equal(t, []string{"linux/amd64", "linux/arm64"}, cfg.Platforms)
	})

	t.Run("handles absolute config path", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "linuxkit",
			Extended: map[string]any{
				"config": "/absolute/path/to/config.yml",
			},
		}
		cfg := p.parseConfig(pubCfg, "/project")

		assert.Equal(t, "/absolute/path/to/config.yml", cfg.Config)
	})
}

func TestLinuxKit_LinuxKitPublisherBuildLinuxKitArgs_Good(t *testing.T) {
	p := NewLinuxKitPublisher()

	t.Run("builds basic args for amd64", func(t *testing.T) {
		args := p.buildLinuxKitArgs("/config/server.yml", "iso", "linuxkit-1.0.0-amd64", "/output", "amd64")

		assert.Contains(t, args, "build")
		assert.Contains(t, args, "--format")
		assert.Contains(t, args, "iso")
		assert.Contains(t, args, "--name")
		assert.Contains(t, args, "linuxkit-1.0.0-amd64")
		assert.Contains(t, args, "--dir")
		assert.Contains(t, args, "/output")
		assert.Contains(t, args, "/config/server.yml")
		// Should not contain --arch for amd64 (default)
		assert.NotContains(t, args, "--arch")
	})

	t.Run("builds args with arch for arm64", func(t *testing.T) {
		args := p.buildLinuxKitArgs("/config/server.yml", "qcow2", "linuxkit-1.0.0-arm64", "/output", "arm64")

		assert.Contains(t, args, "--arch")
		assert.Contains(t, args, "arm64")
		assert.Contains(t, args, "qcow2")
	})
}

func TestLinuxKit_LinuxKitPublisherBuildBaseName_Good(t *testing.T) {
	p := NewLinuxKitPublisher()

	t.Run("strips v prefix", func(t *testing.T) {
		name := p.buildBaseName("v1.2.3")
		assert.Equal(t, "linuxkit-1.2.3", name)
	})

	t.Run("handles version without v prefix", func(t *testing.T) {
		name := p.buildBaseName("1.2.3")
		assert.Equal(t, "linuxkit-1.2.3", name)
	})
}

func TestLinuxKit_LinuxKitPublisherGetArtifactPath_Good(t *testing.T) {
	p := NewLinuxKitPublisher()

	tests := []struct {
		name       string
		outputDir  string
		outputName string
		format     string
		expected   string
	}{
		{
			name:       "ISO format",
			outputDir:  "/dist/linuxkit",
			outputName: "linuxkit-1.0.0-amd64",
			format:     "iso",
			expected:   "/dist/linuxkit/linuxkit-1.0.0-amd64.iso",
		},
		{
			name:       "raw format",
			outputDir:  "/dist/linuxkit",
			outputName: "linuxkit-1.0.0-amd64",
			format:     "raw",
			expected:   "/dist/linuxkit/linuxkit-1.0.0-amd64.raw",
		},
		{
			name:       "qcow2 format",
			outputDir:  "/dist/linuxkit",
			outputName: "linuxkit-1.0.0-arm64",
			format:     "qcow2",
			expected:   "/dist/linuxkit/linuxkit-1.0.0-arm64.qcow2",
		},
		{
			name:       "vmdk format",
			outputDir:  "/dist/linuxkit",
			outputName: "linuxkit-1.0.0-amd64",
			format:     "vmdk",
			expected:   "/dist/linuxkit/linuxkit-1.0.0-amd64.vmdk",
		},
		{
			name:       "gcp format",
			outputDir:  "/dist/linuxkit",
			outputName: "linuxkit-1.0.0-amd64",
			format:     "gcp",
			expected:   "/dist/linuxkit/linuxkit-1.0.0-amd64.img.tar.gz",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := p.getArtifactPath(tc.outputDir, tc.outputName, tc.format)
			assert.Equal(t, tc.expected, path)
		})
	}
}

func TestLinuxKit_LinuxKitPublisherGetFormatExtension_Good(t *testing.T) {
	p := NewLinuxKitPublisher()

	tests := []struct {
		format   string
		expected string
	}{
		{"iso", ".iso"},
		{"raw", ".raw"},
		{"qcow2", ".qcow2"},
		{"vmdk", ".vmdk"},
		{"vhd", ".vhd"},
		{"gcp", ".img.tar.gz"},
		{"aws", ".raw"},
		{"unknown", ".unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.format, func(t *testing.T) {
			ext := p.getFormatExtension(tc.format)
			assert.Equal(t, tc.expected, ext)
		})
	}
}

func TestLinuxKit_LinuxKitPublisherPublish_Bad(t *testing.T) {
	p := NewLinuxKitPublisher()

	t.Run("fails when config file not found with linuxkit installed", func(t *testing.T) {
		if err := validateLinuxKitCli(); err != nil {
			t.Skip("skipping test: linuxkit CLI not available")
		}

		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: "/nonexistent",
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{
			Type: "linuxkit",
			Extended: map[string]any{
				"config": "/nonexistent/config.yml",
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		err := p.Publish(context.TODO(), release, pubCfg, relCfg, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config file not found")
	})

	t.Run("fails when linuxkit CLI not available", func(t *testing.T) {
		if err := validateLinuxKitCli(); err == nil {
			t.Skip("skipping test: linuxkit CLI is available")
		}

		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: "/tmp",
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{Type: "linuxkit"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		err := p.Publish(context.TODO(), release, pubCfg, relCfg, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "linuxkit CLI not found")
	})

	t.Run("fails when repository cannot be detected and not provided", func(t *testing.T) {
		if err := validateLinuxKitCli(); err != nil {
			t.Skip("skipping test: linuxkit CLI not available")
		}

		// Create temp directory that is NOT a git repo
		tmpDir := t.TempDir()

		// Create a config file
		configPath := ax.Join(tmpDir, "config.yml")
		err := ax.WriteFile(configPath, []byte("kernel:\n  image: test\n"), 0o644)
		require.NoError(t, err)

		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: tmpDir,
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{
			Type: "linuxkit",
			Extended: map[string]any{
				"config": "config.yml",
			},
		}
		relCfg := &mockReleaseConfig{repository: ""} // Empty repository

		err = p.Publish(context.TODO(), release, pubCfg, relCfg, true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not determine repository")
	})
}

func TestLinuxKit_ValidateLinuxKitCli_Good(t *testing.T) {
	t.Run("returns expected error when linuxkit not installed", func(t *testing.T) {
		err := validateLinuxKitCli()
		if err != nil {
			// LinuxKit is not installed
			assert.Contains(t, err.Error(), "linuxkit CLI not found")
		}
		// If err is nil, linuxkit is installed - that's OK
	})
}

func TestLinuxKit_ResolveLinuxKitCli_Good(t *testing.T) {
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "linuxkit")
	require.NoError(t, ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755))

	command, err := resolveLinuxKitCli(fallbackPath)
	require.NoError(t, err)
	assert.Equal(t, fallbackPath, command)
}

func TestLinuxKit_ResolveLinuxKitCli_Bad(t *testing.T) {
	_, err := resolveLinuxKitCli(ax.Join(t.TempDir(), "missing-linuxkit"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "linuxkit CLI not found")
}

func TestLinuxKit_LinuxKitPublisherPublishWithCLI_Good(t *testing.T) {
	// These tests run only when linuxkit CLI is available
	if err := validateLinuxKitCli(); err != nil {
		t.Skip("skipping test: linuxkit CLI not available")
	}

	p := NewLinuxKitPublisher()

	t.Run("succeeds with dry run and valid config", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create config directory and file
		configDir := ax.Join(tmpDir, ".core", "linuxkit")
		err := ax.MkdirAll(configDir, 0o755)
		require.NoError(t, err)

		configPath := ax.Join(configDir, "server.yml")
		err = ax.WriteFile(configPath, []byte("kernel:\n  image: linuxkit/kernel:5.10\n"), 0o644)
		require.NoError(t, err)

		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: tmpDir,
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{Type: "linuxkit"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		output := capturePublisherOutput(t, func() {
			err = p.Publish(context.TODO(), release, pubCfg, relCfg, true)
		})
		require.NoError(t, err)
		assert.Contains(t, output, "DRY RUN: LinuxKit Build & Publish")
	})

	t.Run("fails with missing config file", func(t *testing.T) {
		tmpDir := t.TempDir()

		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: tmpDir,
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{Type: "linuxkit"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		err := p.Publish(context.TODO(), release, pubCfg, relCfg, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config file not found")
	})

	t.Run("uses relCfg repository", func(t *testing.T) {
		tmpDir := t.TempDir()

		configDir := ax.Join(tmpDir, ".core", "linuxkit")
		err := ax.MkdirAll(configDir, 0o755)
		require.NoError(t, err)

		configPath := ax.Join(configDir, "server.yml")
		err = ax.WriteFile(configPath, []byte("kernel:\n  image: test\n"), 0o644)
		require.NoError(t, err)

		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: tmpDir,
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{Type: "linuxkit"}
		relCfg := &mockReleaseConfig{repository: "custom-owner/custom-repo"}

		output := capturePublisherOutput(t, func() {
			err = p.Publish(context.TODO(), release, pubCfg, relCfg, true)
		})
		require.NoError(t, err)
		assert.Contains(t, output, "custom-owner/custom-repo")
	})

	t.Run("detects repository when not provided", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create config file
		configDir := ax.Join(tmpDir, ".core", "linuxkit")
		err := ax.MkdirAll(configDir, 0o755)
		require.NoError(t, err)

		configPath := ax.Join(configDir, "server.yml")
		err = ax.WriteFile(configPath, []byte("kernel:\n  image: test\n"), 0o644)
		require.NoError(t, err)

		// Initialize git repo
		runPublisherCommand(t, tmpDir, "git", "init")
		runPublisherCommand(t, tmpDir, "git", "remote", "add", "origin", "git@github.com:detected-owner/detected-repo.git")

		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: tmpDir,
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{Type: "linuxkit"}
		relCfg := &mockReleaseConfig{repository: ""} // Empty to trigger detection

		output := capturePublisherOutput(t, func() {
			err = p.Publish(context.TODO(), release, pubCfg, relCfg, true)
		})
		require.NoError(t, err)
		assert.Contains(t, output, "detected-owner/detected-repo")
	})
}

func TestLinuxKit_LinuxKitPublisherPublishNilRelCfg_Good(t *testing.T) {
	if err := validateLinuxKitCli(); err != nil {
		t.Skip("skipping test: linuxkit CLI not available")
	}

	p := NewLinuxKitPublisher()

	t.Run("handles nil relCfg by detecting repo", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create config file
		configDir := ax.Join(tmpDir, ".core", "linuxkit")
		err := ax.MkdirAll(configDir, 0o755)
		require.NoError(t, err)

		configPath := ax.Join(configDir, "server.yml")
		err = ax.WriteFile(configPath, []byte("kernel:\n  image: test\n"), 0o644)
		require.NoError(t, err)

		// Initialize git repo
		runPublisherCommand(t, tmpDir, "git", "init")
		runPublisherCommand(t, tmpDir, "git", "remote", "add", "origin", "git@github.com:nil-owner/nil-repo.git")

		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: tmpDir,
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{Type: "linuxkit"}

		output := capturePublisherOutput(t, func() {
			err = p.Publish(context.TODO(), release, pubCfg, nil, true)
		})
		require.NoError(t, err)
		assert.Contains(t, output, "nil-owner/nil-repo")
	})
}

// mockReleaseConfig implements ReleaseConfig for testing.
type mockReleaseConfig struct {
	repository  string
	projectName string
}

func (m *mockReleaseConfig) GetRepository() string {
	return m.repository
}

func (m *mockReleaseConfig) GetProjectName() string {
	return m.projectName
}

func TestLinuxKit_LinuxKitPublisherDryRunPublish_Good(t *testing.T) {
	p := NewLinuxKitPublisher()

	t.Run("outputs expected dry run information", func(t *testing.T) {
		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: "/project",
			FS:         io.Local,
		}
		cfg := LinuxKitConfig{
			Config:    "/project/.core/linuxkit/server.yml",
			Formats:   []string{"iso", "qcow2"},
			Platforms: []string{"linux/amd64", "linux/arm64"},
		}

		var err error
		output := capturePublisherOutput(t, func() {
			err = p.dryRunPublish(release, cfg, "owner/repo")
		})
		require.NoError(t, err)

		assert.Contains(t, output, "DRY RUN: LinuxKit Build & Publish")
		assert.Contains(t, output, "Repository:    owner/repo")
		assert.Contains(t, output, "Version:       v1.0.0")
		assert.Contains(t, output, "Config:        /project/.core/linuxkit/server.yml")
		assert.Contains(t, output, "Formats:       iso, qcow2")
		assert.Contains(t, output, "Platforms:     linux/amd64, linux/arm64")
		assert.Contains(t, output, "Would execute commands:")
		assert.Contains(t, output, "linuxkit build")
		assert.Contains(t, output, "Would upload artifacts to release:")
		assert.Contains(t, output, "linuxkit-1.0.0-amd64.iso")
		assert.Contains(t, output, "linuxkit-1.0.0-amd64.qcow2")
		assert.Contains(t, output, "linuxkit-1.0.0-arm64.iso")
		assert.Contains(t, output, "linuxkit-1.0.0-arm64.qcow2")
		assert.Contains(t, output, "END DRY RUN")
	})

	t.Run("shows docker format usage hint", func(t *testing.T) {
		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: "/project",
			FS:         io.Local,
		}
		cfg := LinuxKitConfig{
			Config:    "/config.yml",
			Formats:   []string{"docker"},
			Platforms: []string{"linux/amd64"},
		}

		var err error
		output := capturePublisherOutput(t, func() {
			err = p.dryRunPublish(release, cfg, "owner/repo")
		})
		require.NoError(t, err)

		assert.Contains(t, output, "linuxkit-1.0.0-amd64.docker.tar")
		assert.Contains(t, output, "Usage: docker load <")
	})

	t.Run("handles single platform and format", func(t *testing.T) {
		release := &Release{
			Version:    "v2.0.0",
			ProjectDir: "/project",
			FS:         io.Local,
		}
		cfg := LinuxKitConfig{
			Config:    "/config.yml",
			Formats:   []string{"iso"},
			Platforms: []string{"linux/amd64"},
		}

		var err error
		output := capturePublisherOutput(t, func() {
			err = p.dryRunPublish(release, cfg, "owner/repo")
		})
		require.NoError(t, err)

		assert.Contains(t, output, "linuxkit-2.0.0-amd64.iso")
		assert.NotContains(t, output, "arm64")
	})
}

func TestLinuxKit_LinuxKitPublisherGetFormatExtensionAllFormats_Good(t *testing.T) {
	p := NewLinuxKitPublisher()

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
		{"docker", ".docker.tar"},
		{"tar", ".tar"},
		{"kernel+initrd", "-initrd.img"},
		{"custom--format", ".custom--format"},
	}

	for _, tc := range tests {
		t.Run(tc.format, func(t *testing.T) {
			ext := p.getFormatExtension(tc.format)
			assert.Equal(t, tc.expected, ext)
		})
	}
}

func TestLinuxKit_LinuxKitPublisherBuildLinuxKitArgsAllArchitectures_Good(t *testing.T) {
	p := NewLinuxKitPublisher()

	t.Run("amd64 does not include arch flag", func(t *testing.T) {
		args := p.buildLinuxKitArgs("/config.yml", "iso", "output--name", "/output", "amd64")

		assert.Contains(t, args, "build")
		assert.Contains(t, args, "--format")
		assert.Contains(t, args, "iso")
		assert.Contains(t, args, "--name")
		assert.Contains(t, args, "output--name")
		assert.Contains(t, args, "--dir")
		assert.Contains(t, args, "/output")
		assert.Contains(t, args, "/config.yml")
		assert.NotContains(t, args, "--arch")
	})

	t.Run("arm64 includes arch flag", func(t *testing.T) {
		args := p.buildLinuxKitArgs("/config.yml", "qcow2", "output--name", "/output", "arm64")

		assert.Contains(t, args, "--arch")
		assert.Contains(t, args, "arm64")
	})

	t.Run("other architectures include arch flag", func(t *testing.T) {
		args := p.buildLinuxKitArgs("/config.yml", "raw", "output--name", "/output", "riscv64")

		assert.Contains(t, args, "--arch")
		assert.Contains(t, args, "riscv64")
	})
}

func TestLinuxKit_LinuxKitPublisherParseConfigEdgeCases_Good(t *testing.T) {
	p := NewLinuxKitPublisher()

	t.Run("handles nil extended config", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type:     "linuxkit",
			Extended: nil,
		}

		cfg := p.parseConfig(pubCfg, "/project")

		assert.Equal(t, "/project/.core/linuxkit/server.yml", cfg.Config)
		assert.Equal(t, []string{"iso"}, cfg.Formats)
		assert.Equal(t, []string{"linux/amd64"}, cfg.Platforms)
	})

	t.Run("handles empty extended config", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type:     "linuxkit",
			Extended: map[string]any{},
		}

		cfg := p.parseConfig(pubCfg, "/project")

		assert.Equal(t, "/project/.core/linuxkit/server.yml", cfg.Config)
		assert.Equal(t, []string{"iso"}, cfg.Formats)
		assert.Equal(t, []string{"linux/amd64"}, cfg.Platforms)
	})

	t.Run("handles mixed format types in extended config", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "linuxkit",
			Extended: map[string]any{
				"formats": []any{"iso", 123, "qcow2"}, // includes non-string
			},
		}

		cfg := p.parseConfig(pubCfg, "/project")

		assert.Equal(t, []string{"iso", "qcow2"}, cfg.Formats)
	})

	t.Run("handles mixed platform types in extended config", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "linuxkit",
			Extended: map[string]any{
				"platforms": []any{"linux/amd64", nil, "linux/arm64"},
			},
		}

		cfg := p.parseConfig(pubCfg, "/project")

		assert.Equal(t, []string{"linux/amd64", "linux/arm64"}, cfg.Platforms)
	})
}

func TestLinuxKit_LinuxKitPublisherBuildBaseNameEdgeCases_Good(t *testing.T) {
	p := NewLinuxKitPublisher()

	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{"strips v prefix", "v1.2.3", "linuxkit-1.2.3"},
		{"no v prefix", "1.2.3", "linuxkit-1.2.3"},
		{"prerelease version", "v1.0.0-alpha.1", "linuxkit-1.0.0-alpha.1"},
		{"build metadata", "v1.0.0+build.123", "linuxkit-1.0.0+build.123"},
		{"only v", "v", "linuxkit-"},
		{"empty string", "", "linuxkit-"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			name := p.buildBaseName(tc.version)
			assert.Equal(t, tc.expected, name)
		})
	}
}

func TestLinuxKit_LinuxKitPublisherGetArtifactPathAllFormats_Good(t *testing.T) {
	p := NewLinuxKitPublisher()

	tests := []struct {
		name       string
		outputDir  string
		outputName string
		format     string
		expected   string
	}{
		{
			name:       "ISO format",
			outputDir:  "/dist",
			outputName: "linuxkit-1.0.0-amd64",
			format:     "iso",
			expected:   "/dist/linuxkit-1.0.0-amd64.iso",
		},
		{
			name:       "ISO-BIOS format",
			outputDir:  "/dist",
			outputName: "linuxkit-1.0.0-amd64",
			format:     "iso-bios",
			expected:   "/dist/linuxkit-1.0.0-amd64.iso",
		},
		{
			name:       "docker format",
			outputDir:  "/output",
			outputName: "linuxkit-2.0.0-arm64",
			format:     "docker",
			expected:   "/output/linuxkit-2.0.0-arm64.docker.tar",
		},
		{
			name:       "tar format",
			outputDir:  "/output",
			outputName: "linuxkit-1.0.0",
			format:     "tar",
			expected:   "/output/linuxkit-1.0.0.tar",
		},
		{
			name:       "kernel+initrd format",
			outputDir:  "/output",
			outputName: "linuxkit-1.0.0",
			format:     "kernel+initrd",
			expected:   "/output/linuxkit-1.0.0-initrd.img",
		},
		{
			name:       "GCP format",
			outputDir:  "/output",
			outputName: "linuxkit-1.0.0",
			format:     "gcp",
			expected:   "/output/linuxkit-1.0.0.img.tar.gz",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := p.getArtifactPath(tc.outputDir, tc.outputName, tc.format)
			assert.Equal(t, tc.expected, path)
		})
	}
}

func TestLinuxKit_LinuxKitPublisherPublishNilFS_Bad(t *testing.T) {
	if err := validateLinuxKitCli(); err != nil {
		t.Skip("skipping test: linuxkit CLI not available")
	}

	p := NewLinuxKitPublisher()

	t.Run("returns error when release FS is nil", func(t *testing.T) {
		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: "/tmp",
			FS:         nil, // nil FS should be guarded
		}
		pubCfg := PublisherConfig{Type: "linuxkit"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		err := p.Publish(context.TODO(), release, pubCfg, relCfg, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "release filesystem (FS) is nil")
	})
}

func TestLinuxKit_LinuxKitPublisherPublishDryRun_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Skip if linuxkit CLI is not available
	if err := validateLinuxKitCli(); err != nil {
		t.Skip("skipping test: linuxkit CLI not available")
	}

	p := NewLinuxKitPublisher()

	t.Run("dry run succeeds with valid config file", func(t *testing.T) {
		// Create temp directory with config file
		tmpDir := t.TempDir()
		configDir := ax.Join(tmpDir, ".core", "linuxkit")
		err := ax.MkdirAll(configDir, 0o755)
		require.NoError(t, err)

		configPath := ax.Join(configDir, "server.yml")
		err = ax.WriteFile(configPath, []byte("kernel:\n  image: linuxkit/kernel:5.10\n"), 0o644)
		require.NoError(t, err)

		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: tmpDir,
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{Type: "linuxkit"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		output := capturePublisherOutput(t, func() {
			err = p.Publish(context.TODO(), release, pubCfg, relCfg, true)
		})
		require.NoError(t, err)
		assert.Contains(t, output, "DRY RUN: LinuxKit Build & Publish")
	})

	t.Run("dry run uses custom config path", func(t *testing.T) {
		tmpDir := t.TempDir()

		customConfigPath := ax.Join(tmpDir, "custom-config.yml")
		err := ax.WriteFile(customConfigPath, []byte("kernel:\n  image: custom\n"), 0o644)
		require.NoError(t, err)

		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: tmpDir,
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{
			Type: "linuxkit",
			Extended: map[string]any{
				"config": customConfigPath,
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		output := capturePublisherOutput(t, func() {
			err = p.Publish(context.TODO(), release, pubCfg, relCfg, true)
		})
		require.NoError(t, err)
		assert.Contains(t, output, "custom-config.yml")
	})

	t.Run("dry run with multiple formats and platforms", func(t *testing.T) {
		tmpDir := t.TempDir()

		configPath := ax.Join(tmpDir, "config.yml")
		err := ax.WriteFile(configPath, []byte("kernel:\n  image: test\n"), 0o644)
		require.NoError(t, err)

		release := &Release{
			Version:    "v2.0.0",
			ProjectDir: tmpDir,
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{
			Type: "linuxkit",
			Extended: map[string]any{
				"config":    "config.yml",
				"formats":   []any{"iso", "qcow2", "vmdk"},
				"platforms": []any{"linux/amd64", "linux/arm64"},
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		output := capturePublisherOutput(t, func() {
			err = p.Publish(context.TODO(), release, pubCfg, relCfg, true)
		})
		require.NoError(t, err)

		// Check all format/platform combinations are listed
		assert.Contains(t, output, "linuxkit-2.0.0-amd64.iso")
		assert.Contains(t, output, "linuxkit-2.0.0-amd64.qcow2")
		assert.Contains(t, output, "linuxkit-2.0.0-amd64.vmdk")
		assert.Contains(t, output, "linuxkit-2.0.0-arm64.iso")
		assert.Contains(t, output, "linuxkit-2.0.0-arm64.qcow2")
		assert.Contains(t, output, "linuxkit-2.0.0-arm64.vmdk")
	})
}
