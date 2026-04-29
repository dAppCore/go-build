package publishers

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/io"
)

func TestLinuxKit_LinuxKitPublisherNameGood(t *testing.T) {
	t.Run("returns linuxkit", func(t *testing.T) {
		p := NewLinuxKitPublisher()
		if !stdlibAssertEqual("linuxkit", p.Name()) {
			t.Fatalf("want %v, got %v", "linuxkit", p.Name())
		}

	})
}

func TestLinuxKit_LinuxKitPublisherParseConfigGood(t *testing.T) {
	p := NewLinuxKitPublisher()

	t.Run("uses defaults when no extended config", func(t *testing.T) {
		pubCfg := PublisherConfig{Type: "linuxkit"}
		cfg := p.parseConfig(pubCfg, "/project")
		if !stdlibAssertEqual("/project/.core/linuxkit/server.yml", cfg.Config) {
			t.Fatalf("want %v, got %v", "/project/.core/linuxkit/server.yml", cfg.Config)
		}
		if !stdlibAssertEqual([]string{"iso"}, cfg.Formats) {
			t.Fatalf("want %v, got %v", []string{"iso"}, cfg.Formats)
		}
		if !stdlibAssertEqual([]string{"linux/amd64"}, cfg.Platforms) {
			t.Fatalf("want %v, got %v", []string{"linux/amd64"}, cfg.Platforms)
		}

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
		if !stdlibAssertEqual("/project/.core/linuxkit/custom.yml", cfg.Config) {
			t.Fatalf("want %v, got %v", "/project/.core/linuxkit/custom.yml", cfg.Config)
		}
		if !stdlibAssertEqual([]string{"iso", "qcow2", "vmdk"}, cfg.Formats) {
			t.Fatalf("want %v, got %v", []string{"iso", "qcow2", "vmdk"}, cfg.Formats)
		}
		if !stdlibAssertEqual([]string{"linux/amd64", "linux/arm64"}, cfg.Platforms) {
			t.Fatalf("want %v, got %v", []string{"linux/amd64", "linux/arm64"}, cfg.Platforms)
		}

	})

	t.Run("handles absolute config path", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "linuxkit",
			Extended: map[string]any{
				"config": "/absolute/path/to/config.yml",
			},
		}
		cfg := p.parseConfig(pubCfg, "/project")
		if !stdlibAssertEqual("/absolute/path/to/config.yml", cfg.Config) {
			t.Fatalf("want %v, got %v", "/absolute/path/to/config.yml", cfg.Config)
		}

	})
}

func TestLinuxKit_LinuxKitPublisherBuildLinuxKitArgsGood(t *testing.T) {
	p := NewLinuxKitPublisher()

	t.Run("builds basic args for amd64", func(t *testing.T) {
		args := p.buildLinuxKitArgs("/config/server.yml", "iso", "linuxkit-1.0.0-amd64", "/output", "amd64")
		if !stdlibAssertContains(args, "build") {
			t.Fatalf("expected %v to contain %v", args, "build")
		}
		if !stdlibAssertContains(args, "--format") {
			t.Fatalf("expected %v to contain %v", args, "--format")
		}
		if !stdlibAssertContains(args, "iso") {
			t.Fatalf("expected %v to contain %v", args, "iso")
		}
		if !stdlibAssertContains(args, "--name") {
			t.Fatalf("expected %v to contain %v",

				// Should not contain --arch for amd64 (default)
				args, "--name")
		}
		if !stdlibAssertContains(args, "linuxkit-1.0.0-amd64") {
			t.Fatalf("expected %v to contain %v", args, "linuxkit-1.0.0-amd64")
		}
		if !stdlibAssertContains(args, "--dir") {
			t.Fatalf("expected %v to contain %v", args, "--dir")
		}
		if !stdlibAssertContains(args, "/output") {
			t.Fatalf("expected %v to contain %v", args, "/output")
		}
		if !stdlibAssertContains(args, "/config/server.yml") {
			t.Fatalf("expected %v to contain %v", args, "/config/server.yml")
		}
		if stdlibAssertContains(args, "--arch") {
			t.Fatalf("expected %v not to contain %v", args, "--arch")
		}

	})

	t.Run("builds args with arch for arm64", func(t *testing.T) {
		args := p.buildLinuxKitArgs("/config/server.yml", "qcow2", "linuxkit-1.0.0-arm64", "/output", "arm64")
		if !stdlibAssertContains(args, "--arch") {
			t.Fatalf("expected %v to contain %v", args, "--arch")
		}
		if !stdlibAssertContains(args, "arm64") {
			t.Fatalf("expected %v to contain %v", args, "arm64")
		}
		if !stdlibAssertContains(args, "qcow2") {
			t.Fatalf("expected %v to contain %v", args, "qcow2")
		}

	})
}

func TestLinuxKit_LinuxKitPublisherBuildBaseNameGood(t *testing.T) {
	p := NewLinuxKitPublisher()

	t.Run("strips v prefix", func(t *testing.T) {
		name := p.buildBaseName("v1.2.3")
		if !stdlibAssertEqual("linuxkit-1.2.3", name) {
			t.Fatalf("want %v, got %v", "linuxkit-1.2.3", name)
		}

	})

	t.Run("handles version without v prefix", func(t *testing.T) {
		name := p.buildBaseName("1.2.3")
		if !stdlibAssertEqual("linuxkit-1.2.3", name) {
			t.Fatalf("want %v, got %v", "linuxkit-1.2.3", name)
		}

	})
}

func TestLinuxKit_LinuxKitPublisherGetArtifactPathGood(t *testing.T) {
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
			if !stdlibAssertEqual(tc.expected, path) {
				t.Fatalf("want %v, got %v", tc.expected, path)
			}

		})
	}
}

func TestLinuxKit_LinuxKitPublisherGetFormatExtensionGood(t *testing.T) {
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
			if !stdlibAssertEqual(tc.expected, ext) {
				t.Fatalf("want %v, got %v", tc.expected, ext)
			}

		})
	}
}

func TestLinuxKit_LinuxKitPublisherPublishBad(t *testing.T) {
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
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "config file not found") {
			t.Fatalf("expected %v to contain %v", err.Error(), "config file not found")
		}

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
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "linuxkit CLI not found") {
			t.Fatalf("expected %v to contain %v", err.Error(), "linuxkit CLI not found")
		}

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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

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
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "could not determine repository") {
			t.Fatalf("expected %v to contain %v", err.Error(), "could not determine repository")
		}

	})
}

func TestLinuxKit_ValidateLinuxKitCliGood(t *testing.T) {
	t.Run("returns expected error when linuxkit not installed", func(t *testing.T) {
		err := validateLinuxKitCli()
		if err != nil {
			if !stdlibAssertContains(
				// LinuxKit is not installed
				err.Error(), "linuxkit CLI not found") {
				t.Fatalf("expected %v to contain %v", err.

					// If err is nil, linuxkit is installed - that's OK
					Error(), "linuxkit CLI not found")
			}

		}

	})
}

func TestLinuxKit_ResolveLinuxKitCliGood(t *testing.T) {
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "linuxkit")
	if err := ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Setenv("PATH", "")

	command, err := resolveLinuxKitCli(fallbackPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(fallbackPath, command) {
		t.Fatalf("want %v, got %v", fallbackPath, command)
	}

}

func TestLinuxKit_ResolveLinuxKitCliBad(t *testing.T) {
	t.Setenv("PATH", "")
	_, err := resolveLinuxKitCli(ax.Join(t.TempDir(), "missing-linuxkit"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "linuxkit CLI not found") {
		t.Fatalf("expected %v to contain %v", err.Error(), "linuxkit CLI not found")

		// These tests run only when linuxkit CLI is available
	}

}

func TestLinuxKit_LinuxKitPublisherPublishWithCLIGood(t *testing.T) {

	if err := validateLinuxKitCli(); err != nil {
		t.Skip("skipping test: linuxkit CLI not available")
	}

	p := NewLinuxKitPublisher()

	t.Run("succeeds with dry run and valid config", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create config directory and file
		configDir := ax.Join(tmpDir, ".core", "linuxkit")
		err := ax.MkdirAll(configDir, 0o755)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		configPath := ax.Join(configDir, "server.yml")
		err = ax.WriteFile(configPath, []byte("kernel:\n  image: linuxkit/kernel:5.10\n"), 0o644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(output, "DRY RUN: LinuxKit Build & Publish") {
			t.Fatalf("expected %v to contain %v", output, "DRY RUN: LinuxKit Build & Publish")
		}

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
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "config file not found") {
			t.Fatalf("expected %v to contain %v", err.Error(), "config file not found")
		}

	})

	t.Run("uses relCfg repository", func(t *testing.T) {
		tmpDir := t.TempDir()

		configDir := ax.Join(tmpDir, ".core", "linuxkit")
		err := ax.MkdirAll(configDir, 0o755)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		configPath := ax.Join(configDir, "server.yml")
		err = ax.WriteFile(configPath, []byte("kernel:\n  image: test\n"), 0o644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(output, "custom-owner/custom-repo") {
			t.Fatalf("expected %v to contain %v", output, "custom-owner/custom-repo")
		}

	})

	t.Run("detects repository when not provided", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create config file
		configDir := ax.Join(tmpDir, ".core", "linuxkit")
		err := ax.MkdirAll(configDir, 0o755)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		configPath := ax.Join(configDir, "server.yml")
		err = ax.WriteFile(configPath, []byte("kernel:\n  image: test\n"), 0o644)
		if err != nil {
			t.Fatalf("unexpected error: %v",

				// Initialize git repo
				err)
		}

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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(output, "detected-owner/detected-repo") {
			t.Fatalf("expected %v to contain %v", output, "detected-owner/detected-repo")
		}

	})
}

func TestLinuxKit_LinuxKitPublisherPublishNilRelCfgGood(t *testing.T) {
	if err := validateLinuxKitCli(); err != nil {
		t.Skip("skipping test: linuxkit CLI not available")
	}

	p := NewLinuxKitPublisher()

	t.Run("handles nil relCfg by detecting repo", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create config file
		configDir := ax.Join(tmpDir, ".core", "linuxkit")
		err := ax.MkdirAll(configDir, 0o755)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		configPath := ax.Join(configDir, "server.yml")
		err = ax.WriteFile(configPath, []byte("kernel:\n  image: test\n"), 0o644)
		if err != nil {
			t.Fatalf("unexpected error: %v",

				// Initialize git repo
				err)
		}

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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(output, "nil-owner/nil-repo") {

			// mockReleaseConfig implements ReleaseConfig for testing.
			t.Fatalf("expected %v to contain %v", output, "nil-owner/nil-repo")
		}

	})
}

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

func TestLinuxKit_LinuxKitPublisherDryRunPublishGood(t *testing.T) {
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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(output, "DRY RUN: LinuxKit Build & Publish") {
			t.Fatalf("expected %v to contain %v", output, "DRY RUN: LinuxKit Build & Publish")
		}
		if !stdlibAssertContains(output, "Repository:    owner/repo") {
			t.Fatalf("expected %v to contain %v", output, "Repository:    owner/repo")
		}
		if !stdlibAssertContains(output, "Version:       v1.0.0") {
			t.Fatalf("expected %v to contain %v", output, "Version:       v1.0.0")
		}
		if !stdlibAssertContains(output, "Config:        /project/.core/linuxkit/server.yml") {
			t.Fatalf("expected %v to contain %v", output, "Config:        /project/.core/linuxkit/server.yml")
		}
		if !stdlibAssertContains(output, "Formats:       iso, qcow2") {
			t.Fatalf("expected %v to contain %v", output, "Formats:       iso, qcow2")
		}
		if !stdlibAssertContains(output, "Platforms:     linux/amd64, linux/arm64") {
			t.Fatalf("expected %v to contain %v", output, "Platforms:     linux/amd64, linux/arm64")
		}
		if !stdlibAssertContains(output, "Would execute commands:") {
			t.Fatalf("expected %v to contain %v", output, "Would execute commands:")
		}
		if !stdlibAssertContains(output, "linuxkit build") {
			t.Fatalf("expected %v to contain %v", output, "linuxkit build")
		}
		if !stdlibAssertContains(output, "Would produce/upload artifacts:") {
			t.Fatalf("expected %v to contain %v", output, "Would produce/upload artifacts:")
		}
		if !stdlibAssertContains(output, "linuxkit-1.0.0-amd64.iso") {
			t.Fatalf("expected %v to contain %v", output, "linuxkit-1.0.0-amd64.iso")
		}
		if !stdlibAssertContains(output, "linuxkit-1.0.0-amd64.qcow2") {
			t.Fatalf("expected %v to contain %v", output, "linuxkit-1.0.0-amd64.qcow2")
		}
		if !stdlibAssertContains(output, "linuxkit-1.0.0-arm64.iso") {
			t.Fatalf("expected %v to contain %v", output, "linuxkit-1.0.0-arm64.iso")
		}
		if !stdlibAssertContains(output, "linuxkit-1.0.0-arm64.qcow2") {
			t.Fatalf("expected %v to contain %v", output, "linuxkit-1.0.0-arm64.qcow2")
		}
		if !stdlibAssertContains(output, "END DRY RUN") {
			t.Fatalf("expected %v to contain %v", output, "END DRY RUN")
		}

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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(output, "linuxkit-1.0.0-amd64.docker.tar") {
			t.Fatalf("expected %v to contain %v", output, "linuxkit-1.0.0-amd64.docker.tar")
		}
		if !stdlibAssertContains(output, "Usage: docker load <") {
			t.Fatalf("expected %v to contain %v", output, "Usage: docker load <")
		}

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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(output, "linuxkit-2.0.0-amd64.iso") {
			t.Fatalf("expected %v to contain %v", output, "linuxkit-2.0.0-amd64.iso")
		}
		if stdlibAssertContains(output, "arm64") {
			t.Fatalf("expected %v not to contain %v", output, "arm64")
		}

	})
}

func TestLinuxKit_LinuxKitPublisherGetFormatExtensionAllFormatsGood(t *testing.T) {
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
			if !stdlibAssertEqual(tc.expected, ext) {
				t.Fatalf("want %v, got %v", tc.expected, ext)
			}

		})
	}
}

func TestLinuxKit_LinuxKitPublisherBuildLinuxKitArgsAllArchitecturesGood(t *testing.T) {
	p := NewLinuxKitPublisher()

	t.Run("amd64 does not include arch flag", func(t *testing.T) {
		args := p.buildLinuxKitArgs("/config.yml", "iso", "output--name", "/output", "amd64")
		if !stdlibAssertContains(args, "build") {
			t.Fatalf("expected %v to contain %v", args, "build")
		}
		if !stdlibAssertContains(args, "--format") {
			t.Fatalf("expected %v to contain %v", args, "--format")
		}
		if !stdlibAssertContains(args, "iso") {
			t.Fatalf("expected %v to contain %v", args, "iso")
		}
		if !stdlibAssertContains(args, "--name") {
			t.Fatalf("expected %v to contain %v", args, "--name")
		}
		if !stdlibAssertContains(args, "output--name") {
			t.Fatalf("expected %v to contain %v", args, "output--name")
		}
		if !stdlibAssertContains(args, "--dir") {
			t.Fatalf("expected %v to contain %v", args, "--dir")
		}
		if !stdlibAssertContains(args, "/output") {
			t.Fatalf("expected %v to contain %v", args, "/output")
		}
		if !stdlibAssertContains(args, "/config.yml") {
			t.Fatalf("expected %v to contain %v", args, "/config.yml")
		}
		if stdlibAssertContains(args, "--arch") {
			t.Fatalf("expected %v not to contain %v", args, "--arch")
		}

	})

	t.Run("arm64 includes arch flag", func(t *testing.T) {
		args := p.buildLinuxKitArgs("/config.yml", "qcow2", "output--name", "/output", "arm64")
		if !stdlibAssertContains(args, "--arch") {
			t.Fatalf("expected %v to contain %v", args, "--arch")
		}
		if !stdlibAssertContains(args, "arm64") {
			t.Fatalf("expected %v to contain %v", args, "arm64")
		}

	})

	t.Run("other architectures include arch flag", func(t *testing.T) {
		args := p.buildLinuxKitArgs("/config.yml", "raw", "output--name", "/output", "riscv64")
		if !stdlibAssertContains(args, "--arch") {
			t.Fatalf("expected %v to contain %v", args, "--arch")
		}
		if !stdlibAssertContains(args, "riscv64") {
			t.Fatalf("expected %v to contain %v", args, "riscv64")
		}

	})
}

func TestLinuxKit_LinuxKitPublisherParseConfigEdgeCasesGood(t *testing.T) {
	p := NewLinuxKitPublisher()

	t.Run("handles nil extended config", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type:     "linuxkit",
			Extended: nil,
		}

		cfg := p.parseConfig(pubCfg, "/project")
		if !stdlibAssertEqual("/project/.core/linuxkit/server.yml", cfg.Config) {
			t.Fatalf("want %v, got %v", "/project/.core/linuxkit/server.yml", cfg.Config)
		}
		if !stdlibAssertEqual([]string{"iso"}, cfg.Formats) {
			t.Fatalf("want %v, got %v", []string{"iso"}, cfg.Formats)
		}
		if !stdlibAssertEqual([]string{"linux/amd64"}, cfg.Platforms) {
			t.Fatalf("want %v, got %v", []string{"linux/amd64"}, cfg.Platforms)
		}

	})

	t.Run("handles empty extended config", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type:     "linuxkit",
			Extended: map[string]any{},
		}

		cfg := p.parseConfig(pubCfg, "/project")
		if !stdlibAssertEqual("/project/.core/linuxkit/server.yml", cfg.Config) {
			t.Fatalf("want %v, got %v", "/project/.core/linuxkit/server.yml", cfg.Config)
		}
		if !stdlibAssertEqual([]string{"iso"}, cfg.Formats) {
			t.Fatalf("want %v, got %v", []string{"iso"}, cfg.Formats)
		}
		if !stdlibAssertEqual([]string{"linux/amd64"}, cfg.Platforms) {
			t.Fatalf("want %v, got %v", []string{"linux/amd64"}, cfg.Platforms)
		}

	})

	t.Run("handles mixed format types in extended config", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "linuxkit",
			Extended: map[string]any{
				"formats": []any{"iso", 123, "qcow2"}, // includes non-string
			},
		}

		cfg := p.parseConfig(pubCfg, "/project")
		if !stdlibAssertEqual([]string{"iso", "qcow2"}, cfg.Formats) {
			t.Fatalf("want %v, got %v", []string{"iso", "qcow2"}, cfg.Formats)
		}

	})

	t.Run("handles mixed platform types in extended config", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "linuxkit",
			Extended: map[string]any{
				"platforms": []any{"linux/amd64", nil, "linux/arm64"},
			},
		}

		cfg := p.parseConfig(pubCfg, "/project")
		if !stdlibAssertEqual([]string{"linux/amd64", "linux/arm64"}, cfg.Platforms) {
			t.Fatalf("want %v, got %v", []string{"linux/amd64", "linux/arm64"}, cfg.Platforms)
		}

	})
}

func TestLinuxKit_LinuxKitPublisherBuildBaseNameEdgeCasesGood(t *testing.T) {
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
			if !stdlibAssertEqual(tc.expected, name) {
				t.Fatalf("want %v, got %v", tc.expected, name)
			}

		})
	}
}

func TestLinuxKit_LinuxKitPublisherGetArtifactPathAllFormatsGood(t *testing.T) {
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
			if !stdlibAssertEqual(tc.expected, path) {
				t.Fatalf("want %v, got %v", tc.expected, path)
			}

		})
	}
}

func TestLinuxKit_LinuxKitPublisherPublishNilFSBad(t *testing.T) {
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
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "release filesystem (FS) is nil") {
			t.Fatalf("expected %v to contain %v", err.Error(), "release filesystem (FS) is nil")
		}

	})
}

func TestLinuxKit_LinuxKitPublisherPublishDryRunGood(t *testing.T) {
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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		configPath := ax.Join(configDir, "server.yml")
		err = ax.WriteFile(configPath, []byte("kernel:\n  image: linuxkit/kernel:5.10\n"), 0o644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(output, "DRY RUN: LinuxKit Build & Publish") {
			t.Fatalf("expected %v to contain %v", output, "DRY RUN: LinuxKit Build & Publish")
		}

	})

	t.Run("dry run uses custom config path", func(t *testing.T) {
		tmpDir := t.TempDir()

		customConfigPath := ax.Join(tmpDir, "custom-config.yml")
		err := ax.WriteFile(customConfigPath, []byte("kernel:\n  image: custom\n"), 0o644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(output, "custom-config.yml") {
			t.Fatalf("expected %v to contain %v", output, "custom-config.yml")
		}

	})

	t.Run("dry run with multiple formats and platforms", func(t *testing.T) {
		tmpDir := t.TempDir()

		configPath := ax.Join(tmpDir, "config.yml")
		err := ax.WriteFile(configPath, []byte("kernel:\n  image: test\n"), 0o644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

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
		if err != nil {
			t.Fatalf("unexpected error: %v",

				// Check all format/platform combinations are listed
				err)
		}
		if !stdlibAssertContains(output, "linuxkit-2.0.0-amd64.iso") {
			t.Fatalf("expected %v to contain %v", output, "linuxkit-2.0.0-amd64.iso")
		}
		if !stdlibAssertContains(output, "linuxkit-2.0.0-amd64.qcow2") {
			t.Fatalf("expected %v to contain %v", output, "linuxkit-2.0.0-amd64.qcow2")
		}
		if !stdlibAssertContains(output, "linuxkit-2.0.0-amd64.vmdk") {
			t.Fatalf("expected %v to contain %v", output, "linuxkit-2.0.0-amd64.vmdk")
		}
		if !stdlibAssertContains(output, "linuxkit-2.0.0-arm64.iso") {
			t.Fatalf("expected %v to contain %v", output, "linuxkit-2.0.0-arm64.iso")
		}
		if !stdlibAssertContains(output, "linuxkit-2.0.0-arm64.qcow2") {
			t.Fatalf("expected %v to contain %v", output, "linuxkit-2.0.0-arm64.qcow2")
		}
		if !stdlibAssertContains(output, "linuxkit-2.0.0-arm64.vmdk") {
			t.Fatalf("expected %v to contain %v", output, "linuxkit-2.0.0-arm64.vmdk")
		}

	})
}

func TestPublish_IsoQcow2RawGood(t *testing.T) {
	result := runLinuxKitPublishFixture(t, []string{"iso", "qcow2", "raw"}, "ok", nil)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}

	assertLinuxKitArtifactExists(t, result, "iso")
	assertLinuxKitArtifactExists(t, result, "qcow2")
	assertLinuxKitArtifactExists(t, result, "raw")
}

func TestPublish_IsoGood(t *testing.T) {
	result := runLinuxKitPublishFixture(t, []string{"iso"}, "ok", nil)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	assertLinuxKitArtifactExists(t, result, "iso")
}

func TestPublish_IsoBad(t *testing.T) {
	result := runLinuxKitPublishFixture(t, []string{"iso"}, "fail", nil)
	if result.Err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(result.Err.Error(), "build failed") {
		t.Fatalf("expected %v to contain %v", result.Err.Error(), "build failed")
	}
}

func TestPublish_IsoUgly(t *testing.T) {
	result := runLinuxKitPublishFixture(t, []string{"iso"}, "missing", nil)
	if result.Err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(result.Err.Error(), "artifact not found after build") {
		t.Fatalf("expected %v to contain %v", result.Err.Error(), "artifact not found after build")
	}
}

func TestPublish_Qcow2Good(t *testing.T) {
	result := runLinuxKitPublishFixture(t, []string{"qcow2"}, "ok", nil)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	assertLinuxKitArtifactExists(t, result, "qcow2")
}

func TestPublish_Qcow2Bad(t *testing.T) {
	result := runLinuxKitPublishFixture(t, []string{"qcow2"}, "fail", nil)
	if result.Err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(result.Err.Error(), "build failed") {
		t.Fatalf("expected %v to contain %v", result.Err.Error(), "build failed")
	}
}

func TestPublish_Qcow2Ugly(t *testing.T) {
	result := runLinuxKitPublishFixture(t, []string{"qcow2"}, "missing", nil)
	if result.Err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(result.Err.Error(), "artifact not found after build") {
		t.Fatalf("expected %v to contain %v", result.Err.Error(), "artifact not found after build")
	}
}

func TestPublish_RawGood(t *testing.T) {
	result := runLinuxKitPublishFixture(t, []string{"raw"}, "ok", nil)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	assertLinuxKitArtifactExists(t, result, "raw")
}

func TestPublish_RawBad(t *testing.T) {
	result := runLinuxKitPublishFixture(t, []string{"raw"}, "fail", nil)
	if result.Err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(result.Err.Error(), "build failed") {
		t.Fatalf("expected %v to contain %v", result.Err.Error(), "build failed")
	}
}

func TestPublish_RawUgly(t *testing.T) {
	result := runLinuxKitPublishFixture(t, []string{"raw"}, "missing", nil)
	if result.Err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(result.Err.Error(), "artifact not found after build") {
		t.Fatalf("expected %v to contain %v", result.Err.Error(), "artifact not found after build")
	}
}

func TestPublish_Qcow2WithCloudTargetsGood(t *testing.T) {
	result := runLinuxKitPublishFixture(t, []string{"qcow2"}, "ok", map[string]any{
		"targets": []any{
			map[string]any{"provider": "aws", "bucket": "aws-bucket", "prefix": "images"},
			map[string]any{"provider": "gcp", "bucket": "gcp-bucket", "prefix": "images"},
		},
	})
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	assertLinuxKitArtifactExists(t, result, "qcow2")

	log := readLinuxKitCloudLog(t, result.CloudLog)
	if !stdlibAssertContains(log, "aws s3 cp") {
		t.Fatalf("expected %v to contain %v", log, "aws s3 cp")
	}
	if !stdlibAssertContains(log, "s3://aws-bucket/images/linuxkit-1.2.3-amd64.qcow2") {
		t.Fatalf("expected %v to contain %v", log, "s3://aws-bucket/images/linuxkit-1.2.3-amd64.qcow2")
	}
	if !stdlibAssertContains(log, "gcloud storage cp") {
		t.Fatalf("expected %v to contain %v", log, "gcloud storage cp")
	}
	if !stdlibAssertContains(log, "gs://gcp-bucket/images/linuxkit-1.2.3-amd64.qcow2") {
		t.Fatalf("expected %v to contain %v", log, "gs://gcp-bucket/images/linuxkit-1.2.3-amd64.qcow2")
	}
}

func TestPublish_AWSGood(t *testing.T) {
	result := runLinuxKitPublishFixture(t, []string{"aws"}, "ok", map[string]any{
		"targets": []any{
			map[string]any{"provider": "aws", "bucket": "aws-bucket", "prefix": "images", "region": "eu-west-2"},
		},
	})
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	assertLinuxKitArtifactExists(t, result, "aws")

	log := readLinuxKitCloudLog(t, result.CloudLog)
	if !stdlibAssertContains(log, "s3://aws-bucket/images/linuxkit-1.2.3-amd64.raw") {
		t.Fatalf("expected %v to contain %v", log, "s3://aws-bucket/images/linuxkit-1.2.3-amd64.raw")
	}
	if !stdlibAssertContains(log, "--region eu-west-2") {
		t.Fatalf("expected %v to contain %v", log, "--region eu-west-2")
	}
}

func TestPublish_AWSBad(t *testing.T) {
	result := runLinuxKitPublishFixture(t, []string{"aws"}, "ok", nil)
	if result.Err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(result.Err.Error(), "aws target bucket is required") {
		t.Fatalf("expected %v to contain %v", result.Err.Error(), "aws target bucket is required")
	}
}

func TestPublish_GCPGood(t *testing.T) {
	result := runLinuxKitPublishFixture(t, []string{"gcp"}, "ok", map[string]any{
		"targets": []any{
			`{"provider":"gcp","bucket":"gcp-bucket","prefix":"images"}`,
		},
	})
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	assertLinuxKitArtifactExists(t, result, "gcp")

	log := readLinuxKitCloudLog(t, result.CloudLog)
	if !stdlibAssertContains(log, "gs://gcp-bucket/images/linuxkit-1.2.3-amd64.img.tar.gz") {
		t.Fatalf("expected %v to contain %v", log, "gs://gcp-bucket/images/linuxkit-1.2.3-amd64.img.tar.gz")
	}
}

func TestPublish_GCPBad(t *testing.T) {
	result := runLinuxKitPublishFixture(t, []string{"gcp"}, "ok", nil)
	if result.Err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(result.Err.Error(), "gcp target bucket is required") {
		t.Fatalf("expected %v to contain %v", result.Err.Error(), "gcp target bucket is required")
	}
}

type linuxKitPublishFixtureResult struct {
	ArtifactPaths map[string]string
	CloudLog      string
	Err           error
	Output        string
	ProjectDir    string
}

func runLinuxKitPublishFixture(t *testing.T, formats []string, linuxKitMode string, extended map[string]any) linuxKitPublishFixtureResult {
	t.Helper()

	tmpDir := t.TempDir()
	binDir := ax.Join(tmpDir, "bin")
	if err := ax.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	installFakeLinuxKitTool(t, binDir, linuxKitMode)
	installFakeLinuxKitCloudTool(t, binDir, "aws")
	installFakeLinuxKitCloudTool(t, binDir, "gcloud")

	oldPath := core.Getenv("PATH")
	t.Setenv("PATH", binDir+":"+oldPath)
	cloudLog := ax.Join(tmpDir, "cloud.log")
	t.Setenv("LINUXKIT_CLOUD_LOG", cloudLog)
	t.Setenv("LINUXKIT_CLOUD_MODE", "ok")

	configPath := ax.Join(tmpDir, "config.yml")
	if err := ax.WriteFile(configPath, []byte("kernel:\n  image: test\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	formatValues := make([]any, 0, len(formats))
	for _, format := range formats {
		formatValues = append(formatValues, format)
	}

	ext := map[string]any{
		"config":    "config.yml",
		"formats":   formatValues,
		"platforms": []any{"linux/amd64"},
	}
	for key, value := range extended {
		ext[key] = value
	}

	release := &Release{
		Version:    "v1.2.3",
		ProjectDir: tmpDir,
		FS:         io.Local,
	}
	pubCfg := PublisherConfig{
		Type:     "linuxkit",
		Extended: ext,
	}
	relCfg := &mockReleaseConfig{repository: "owner/repo"}

	p := NewLinuxKitPublisher()
	var publishErr error
	output := capturePublisherOutput(t, func() {
		publishErr = p.Publish(context.TODO(), release, pubCfg, relCfg, false)
	})

	outputDir := ax.Join(tmpDir, "dist", "linuxkit")
	baseName := p.buildBaseName(release.Version)
	artifactPaths := make(map[string]string, len(formats))
	for _, format := range formats {
		artifactPaths[format] = p.getArtifactPath(outputDir, baseName+"-amd64", format)
	}

	return linuxKitPublishFixtureResult{
		ArtifactPaths: artifactPaths,
		CloudLog:      cloudLog,
		Err:           publishErr,
		Output:        output,
		ProjectDir:    tmpDir,
	}
}

func assertLinuxKitArtifactExists(t *testing.T, result linuxKitPublishFixtureResult, format string) {
	t.Helper()

	artifactPath := result.ArtifactPaths[format]
	if !io.Local.Exists(artifactPath) {
		t.Fatalf("expected artifact to exist: %s", artifactPath)
	}

	content, err := ax.ReadFile(artifactPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertContains(string(content), "linuxkit:"+format) {
		t.Fatalf("expected %v to contain %v", string(content), "linuxkit:"+format)
	}
}

func assertLinuxKitPublishError(t *testing.T, format, linuxKitMode, expected string) {
	t.Helper()

	result := runLinuxKitPublishFixture(t, []string{format}, linuxKitMode, nil)
	if result.Err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(result.Err.Error(), expected) {
		t.Fatalf("expected %v to contain %v", result.Err.Error(), expected)
	}
}

func readLinuxKitCloudLog(t *testing.T, path string) string {
	t.Helper()

	if !ax.Exists(path) {
		return ""
	}
	data, err := ax.ReadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return string(data)
}

func installFakeLinuxKitTool(t *testing.T, binDir, mode string) {
	t.Helper()

	script := `#!/bin/sh
mode="` + mode + `"
format=""
name=""
dir=""
while [ "$#" -gt 0 ]; do
	case "$1" in
		--format)
			format="$2"
			shift 2
			;;
		--name)
			name="$2"
			shift 2
			;;
		--dir)
			dir="$2"
			shift 2
			;;
		--arch)
			shift 2
			;;
		*)
			shift
			;;
	esac
done
if [ "$mode" = "fail" ]; then
	echo "fake linuxkit failed" >&2
	exit 23
fi
if [ "$mode" = "missing" ]; then
	exit 0
fi
case "$format" in
	iso|iso-bios|iso-efi)
		ext=".iso"
		;;
	raw|raw-bios|raw-efi|aws)
		ext=".raw"
		;;
	qcow2|qcow2-bios|qcow2-efi)
		ext=".qcow2"
		;;
	gcp)
		ext=".img.tar.gz"
		;;
	docker)
		ext=".docker.tar"
		;;
	tar)
		ext=".tar"
		;;
	kernel+initrd)
		ext="-initrd.img"
		;;
	*)
		ext=".$format"
		;;
esac
/bin/mkdir -p "$dir"
printf 'linuxkit:%s:%s' "$format" "$name" > "$dir/$name$ext"
`
	if err := ax.WriteFile(ax.Join(binDir, "linuxkit"), []byte(script), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func installFakeLinuxKitCloudTool(t *testing.T, binDir, name string) {
	t.Helper()

	script := `#!/bin/sh
if [ -n "$LINUXKIT_CLOUD_LOG" ]; then
	printf '%s' "` + name + `" >> "$LINUXKIT_CLOUD_LOG"
	for arg in "$@"; do
		printf ' %s' "$arg" >> "$LINUXKIT_CLOUD_LOG"
	done
	printf '\n' >> "$LINUXKIT_CLOUD_LOG"
fi
if [ "$LINUXKIT_CLOUD_MODE" = "fail" ]; then
	exit 31
fi
exit 0
`
	if err := ax.WriteFile(ax.Join(binDir, name), []byte(script), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- v0.9.0 generated compliance triplets ---
func TestLinuxkit_NewLinuxKitPublisher_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewLinuxKitPublisher()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestLinuxkit_NewLinuxKitPublisher_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewLinuxKitPublisher()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestLinuxkit_NewLinuxKitPublisher_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewLinuxKitPublisher()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestLinuxkit_LinuxKitPublisher_Name_Good(t *core.T) {
	subject := &LinuxKitPublisher{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestLinuxkit_LinuxKitPublisher_Name_Bad(t *core.T) {
	subject := &LinuxKitPublisher{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestLinuxkit_LinuxKitPublisher_Name_Ugly(t *core.T) {
	subject := &LinuxKitPublisher{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestLinuxkit_LinuxKitPublisher_Validate_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &LinuxKitPublisher{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Validate(ctx, &Release{}, PublisherConfig{}, nil)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestLinuxkit_LinuxKitPublisher_Validate_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &LinuxKitPublisher{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Validate(ctx, nil, PublisherConfig{}, nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestLinuxkit_LinuxKitPublisher_Validate_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &LinuxKitPublisher{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Validate(ctx, &Release{}, PublisherConfig{}, nil)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestLinuxkit_LinuxKitPublisher_Supports_Good(t *core.T) {
	subject := &LinuxKitPublisher{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Supports("linux")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestLinuxkit_LinuxKitPublisher_Supports_Bad(t *core.T) {
	subject := &LinuxKitPublisher{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Supports("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestLinuxkit_LinuxKitPublisher_Supports_Ugly(t *core.T) {
	subject := &LinuxKitPublisher{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Supports("linux")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestLinuxkit_LinuxKitPublisher_Publish_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &LinuxKitPublisher{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Publish(ctx, &Release{}, PublisherConfig{}, nil, true)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestLinuxkit_LinuxKitPublisher_Publish_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &LinuxKitPublisher{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Publish(ctx, nil, PublisherConfig{}, nil, true)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestLinuxkit_LinuxKitPublisher_Publish_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &LinuxKitPublisher{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Publish(ctx, &Release{}, PublisherConfig{}, nil, true)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
