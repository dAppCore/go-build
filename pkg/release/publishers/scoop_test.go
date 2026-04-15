package publishers

import (
	"context"
	"testing"

	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/io"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScoop_ScoopPublisherName_Good(t *testing.T) {
	t.Run("returns scoop", func(t *testing.T) {
		p := NewScoopPublisher()
		assert.Equal(t, "scoop", p.Name())
	})
}

func TestScoop_ScoopPublisherParseConfig_Good(t *testing.T) {
	p := NewScoopPublisher()

	t.Run("uses defaults when no extended config", func(t *testing.T) {
		pubCfg := PublisherConfig{Type: "scoop"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}
		cfg := p.parseConfig(pubCfg, relCfg)

		assert.Empty(t, cfg.Bucket)
		assert.Nil(t, cfg.Official)
	})

	t.Run("parses bucket from extended config", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "scoop",
			Extended: map[string]any{
				"bucket": "host-uk/scoop-bucket",
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}
		cfg := p.parseConfig(pubCfg, relCfg)

		assert.Equal(t, "host-uk/scoop-bucket", cfg.Bucket)
	})

	t.Run("parses official config", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "scoop",
			Extended: map[string]any{
				"official": map[string]any{
					"enabled": true,
					"output":  "dist/scoop-manifest",
				},
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}
		cfg := p.parseConfig(pubCfg, relCfg)

		require.NotNil(t, cfg.Official)
		assert.True(t, cfg.Official.Enabled)
		assert.Equal(t, "dist/scoop-manifest", cfg.Official.Output)
	})

	t.Run("handles missing official fields", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "scoop",
			Extended: map[string]any{
				"official": map[string]any{},
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}
		cfg := p.parseConfig(pubCfg, relCfg)

		require.NotNil(t, cfg.Official)
		assert.False(t, cfg.Official.Enabled)
		assert.Empty(t, cfg.Official.Output)
	})

	t.Run("handles nil extended config", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type:     "scoop",
			Extended: nil,
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}
		cfg := p.parseConfig(pubCfg, relCfg)

		assert.Empty(t, cfg.Bucket)
		assert.Nil(t, cfg.Official)
	})
}

func TestScoop_ScoopPublisherRenderTemplate_Good(t *testing.T) {
	p := NewScoopPublisher()

	t.Run("renders manifest template with data", func(t *testing.T) {
		data := scoopTemplateData{
			PackageName: "myapp",
			Description: "My awesome CLI",
			Repository:  "owner/myapp",
			Version:     "1.2.3",
			License:     "MIT",
			BinaryName:  "myapp",
			Checksums: ChecksumMap{
				WindowsAmd64: "abc123",
				WindowsArm64: "def456",
			},
		}

		result, err := p.renderTemplate(io.Local, "templates/scoop/manifest.json.tmpl", data)
		require.NoError(t, err)

		assert.Contains(t, result, `"version": "1.2.3"`)
		assert.Contains(t, result, `"description": "My awesome CLI"`)
		assert.Contains(t, result, `"homepage": "https://github.com/owner/myapp"`)
		assert.Contains(t, result, `"license": "MIT"`)
		assert.Contains(t, result, `"64bit"`)
		assert.Contains(t, result, `"arm64"`)
		assert.Contains(t, result, "myapp-windows-amd64.zip")
		assert.Contains(t, result, "myapp-windows-arm64.zip")
		assert.Contains(t, result, `"hash": "abc123"`)
		assert.Contains(t, result, `"hash": "def456"`)
		assert.Contains(t, result, `"bin": "myapp.exe"`)
	})

	t.Run("includes autoupdate configuration", func(t *testing.T) {
		data := scoopTemplateData{
			PackageName: "tool",
			Description: "A tool",
			Repository:  "org/tool",
			Version:     "2.0.0",
			License:     "Apache-2.0",
			BinaryName:  "tool",
			Checksums:   ChecksumMap{},
		}

		result, err := p.renderTemplate(io.Local, "templates/scoop/manifest.json.tmpl", data)
		require.NoError(t, err)

		assert.Contains(t, result, `"checkver"`)
		assert.Contains(t, result, `"github": "https://github.com/org/tool"`)
		assert.Contains(t, result, `"autoupdate"`)
	})
}

func TestScoop_ScoopPublisherRenderTemplate_Bad(t *testing.T) {
	p := NewScoopPublisher()

	t.Run("returns error for non-existent template", func(t *testing.T) {
		data := scoopTemplateData{}
		_, err := p.renderTemplate(io.Local, "templates/scoop/nonexistent.tmpl", data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read template")
	})
}

func TestScoop_ScoopPublisherDryRunPublish_Good(t *testing.T) {
	p := NewScoopPublisher()

	t.Run("outputs expected dry run information", func(t *testing.T) {
		data := scoopTemplateData{
			PackageName: "myapp",
			Version:     "1.0.0",
			Repository:  "owner/repo",
			BinaryName:  "myapp",
			Checksums:   ChecksumMap{},
		}
		cfg := ScoopConfig{
			Bucket: "owner/scoop-bucket",
		}

		var err error
		output := capturePublisherOutput(t, func() {
			err = p.dryRunPublish(io.Local, data, cfg)
		})
		require.NoError(t, err)

		assert.Contains(t, output, "DRY RUN: Scoop Publish")
		assert.Contains(t, output, "Package:    myapp")
		assert.Contains(t, output, "Version:    1.0.0")
		assert.Contains(t, output, "Bucket:     owner/scoop-bucket")
		assert.Contains(t, output, "Repository: owner/repo")
		assert.Contains(t, output, "Generated manifest.json:")
		assert.Contains(t, output, "Would commit to bucket: owner/scoop-bucket")
		assert.Contains(t, output, "END DRY RUN")
	})

	t.Run("shows official output path when enabled", func(t *testing.T) {
		data := scoopTemplateData{
			PackageName: "myapp",
			Version:     "1.0.0",
			BinaryName:  "myapp",
			Checksums:   ChecksumMap{},
		}
		cfg := ScoopConfig{
			Official: &OfficialConfig{
				Enabled: true,
				Output:  "custom/scoop/path",
			},
		}

		var err error
		output := capturePublisherOutput(t, func() {
			err = p.dryRunPublish(io.Local, data, cfg)
		})
		require.NoError(t, err)
		assert.Contains(t, output, "Would write files for official PR to: custom/scoop/path")
	})

	t.Run("suppresses bucket publish output in official mode", func(t *testing.T) {
		data := scoopTemplateData{
			PackageName: "myapp",
			Version:     "1.0.0",
			BinaryName:  "myapp",
			Checksums:   ChecksumMap{},
		}
		cfg := ScoopConfig{
			Bucket: "owner/scoop-bucket",
			Official: &OfficialConfig{
				Enabled: true,
				Output:  "custom/scoop/path",
			},
		}

		var err error
		output := capturePublisherOutput(t, func() {
			err = p.dryRunPublish(io.Local, data, cfg)
		})
		require.NoError(t, err)
		assert.Contains(t, output, "Would write files for official PR to: custom/scoop/path")
		assert.NotContains(t, output, "Would commit to bucket:")
	})

	t.Run("uses default official output path when not specified", func(t *testing.T) {
		data := scoopTemplateData{
			PackageName: "myapp",
			Version:     "1.0.0",
			BinaryName:  "myapp",
			Checksums:   ChecksumMap{},
		}
		cfg := ScoopConfig{
			Official: &OfficialConfig{
				Enabled: true,
			},
		}

		var err error
		output := capturePublisherOutput(t, func() {
			err = p.dryRunPublish(io.Local, data, cfg)
		})
		require.NoError(t, err)
		assert.Contains(t, output, "Would write files for official PR to: dist/scoop")
	})
}

func TestScoop_ScoopPublisherPublish_Bad(t *testing.T) {
	p := NewScoopPublisher()

	t.Run("fails when bucket not configured and not official mode", func(t *testing.T) {
		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: "/project",
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{Type: "scoop"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		err := p.Publish(context.TODO(), release, pubCfg, relCfg, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bucket is required")
	})

	t.Run("official mode writes files without requiring bucket publish tooling", func(t *testing.T) {
		projectDir := t.TempDir()
		t.Setenv("PATH", "/definitely-missing")

		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: projectDir,
			FS:         io.Local,
			Artifacts: []build.Artifact{
				{Path: "dist/myapp-windows-amd64.zip", OS: "windows", Arch: "amd64", Checksum: "abc123"},
			},
		}
		pubCfg := PublisherConfig{
			Type: "scoop",
			Extended: map[string]any{
				"bucket": "owner/scoop-bucket",
				"official": map[string]any{
					"enabled": true,
					"output":  "dist/scoop-pr",
				},
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo", projectName: "myapp"}

		err := p.Publish(context.TODO(), release, pubCfg, relCfg, false)
		require.NoError(t, err)
		assert.FileExists(t, ax.Join(projectDir, "dist", "scoop-pr", "myapp.json"))
	})
}

func TestScoop_ScoopConfigDefaults_Good(t *testing.T) {
	t.Run("has sensible defaults", func(t *testing.T) {
		p := NewScoopPublisher()
		pubCfg := PublisherConfig{Type: "scoop"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		cfg := p.parseConfig(pubCfg, relCfg)

		assert.Empty(t, cfg.Bucket)
		assert.Nil(t, cfg.Official)
	})
}

func TestScoop_ScoopTemplateData_Good(t *testing.T) {
	t.Run("struct has all expected fields", func(t *testing.T) {
		data := scoopTemplateData{
			PackageName: "myapp",
			Description: "description",
			Repository:  "org/repo",
			Version:     "1.0.0",
			License:     "MIT",
			BinaryName:  "myapp",
			Checksums: ChecksumMap{
				WindowsAmd64: "hash1",
				WindowsArm64: "hash2",
			},
		}

		assert.Equal(t, "myapp", data.PackageName)
		assert.Equal(t, "description", data.Description)
		assert.Equal(t, "org/repo", data.Repository)
		assert.Equal(t, "1.0.0", data.Version)
		assert.Equal(t, "MIT", data.License)
		assert.Equal(t, "myapp", data.BinaryName)
		assert.Equal(t, "hash1", data.Checksums.WindowsAmd64)
		assert.Equal(t, "hash2", data.Checksums.WindowsArm64)
	})
}
