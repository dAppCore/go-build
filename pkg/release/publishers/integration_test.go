package publishers

import (
	"context"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- GitHub Publisher Integration Tests ---

func TestIntegration_GitHubPublisherIntegrationDryRunNoSideEffects_Good(t *testing.T) {
	p := NewGitHubPublisher()

	t.Run("dry run creates no files on disk", func(t *testing.T) {
		tmpDir := t.TempDir()
		release := &Release{
			Version:    "v1.0.0",
			Changelog:  "## v1.0.0\n\n- feat: initial release",
			ProjectDir: tmpDir,
			FS:         io.Local,
			Artifacts: []build.Artifact{
				{Path: ax.Join(tmpDir, "app-linux-amd64.tar.gz")},
				{Path: ax.Join(tmpDir, "app-darwin-arm64.tar.gz")},
				{Path: ax.Join(tmpDir, "CHECKSUMS.txt")},
			},
		}
		pubCfg := PublisherConfig{
			Type:       "github",
			Draft:      true,
			Prerelease: true,
		}
		relCfg := &mockReleaseConfig{repository: "test-org/test-repo", projectName: "testapp"}

		var err error
		output := capturePublisherOutput(t, func() {
			err = p.Publish(context.Background(), release, pubCfg, relCfg, true)
		})
		require.NoError(t, err)

		// Verify dry run output contains expected information
		assert.Contains(t, output, "DRY RUN: GitHub Release")
		assert.Contains(t, output, "Repository: test-org/test-repo")
		assert.Contains(t, output, "Version:    v1.0.0")
		assert.Contains(t, output, "Draft:      true")
		assert.Contains(t, output, "Prerelease: true")
		assert.Contains(t, output, "Would upload artifacts:")
		assert.Contains(t, output, "app-linux-amd64.tar.gz")
		assert.Contains(t, output, "app-darwin-arm64.tar.gz")
		assert.Contains(t, output, "CHECKSUMS.txt")
		assert.Contains(t, output, "gh release create")
		assert.Contains(t, output, "--draft")
		assert.Contains(t, output, "--prerelease")

		// Verify no files were created in the temp directory
		entries, err := ax.ReadDir(tmpDir)
		require.NoError(t, err)
		assert.Empty(t, entries, "dry run should not create any files")
	})

	t.Run("dry run builds correct gh CLI command for standard release", func(t *testing.T) {
		release := &Release{
			Version:    "v2.3.0",
			Changelog:  "## v2.3.0\n\n### Features\n\n- new feature",
			ProjectDir: "/tmp",
			FS:         io.Local,
			Artifacts: []build.Artifact{
				{Path: "/dist/app-linux-amd64.tar.gz"},
			},
		}
		pubCfg := PublisherConfig{
			Type:       "github",
			Draft:      false,
			Prerelease: false,
		}

		args := p.buildCreateArgs(release, pubCfg, "owner/repo")

		// Verify exact argument structure
		assert.Equal(t, "release", args[0])
		assert.Equal(t, "create", args[1])
		assert.Equal(t, "v2.3.0", args[2])

		// Should have --repo
		repoIdx := indexOf(args, "--repo")
		assert.Greater(t, repoIdx, -1)
		assert.Equal(t, "owner/repo", args[repoIdx+1])

		// Should have --title
		titleIdx := indexOf(args, "--title")
		assert.Greater(t, titleIdx, -1)
		assert.Equal(t, "v2.3.0", args[titleIdx+1])

		// Should have --notes (since changelog is non-empty)
		assert.Contains(t, args, "--notes")

		// Should NOT have --draft or --prerelease
		assert.NotContains(t, args, "--draft")
		assert.NotContains(t, args, "--prerelease")
	})

	t.Run("dry run uses generate-notes when changelog empty", func(t *testing.T) {
		release := &Release{
			Version:    "v1.0.0",
			Changelog:  "",
			ProjectDir: "/tmp",
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{Type: "github"}

		args := p.buildCreateArgs(release, pubCfg, "owner/repo")

		assert.Contains(t, args, "--generate-notes")
		assert.NotContains(t, args, "--notes")
	})
}

func TestIntegration_GitHubPublisherIntegrationRepositoryDetection_Good(t *testing.T) {
	p := NewGitHubPublisher()

	t.Run("uses relCfg repository when provided", func(t *testing.T) {
		release := &Release{
			Version:    "v1.0.0",
			Changelog:  "Changes",
			ProjectDir: "/tmp",
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{Type: "github"}
		relCfg := &mockReleaseConfig{repository: "explicit/repo"}

		var err error
		output := capturePublisherOutput(t, func() {
			err = p.Publish(context.Background(), release, pubCfg, relCfg, true)
		})
		require.NoError(t, err)
		assert.Contains(t, output, "Repository: explicit/repo")
	})

	t.Run("detects repository from git remote when relCfg empty", func(t *testing.T) {
		tmpDir := t.TempDir()

		runPublisherCommand(t, tmpDir, "git", "init")
		runPublisherCommand(t, tmpDir, "git", "remote", "add", "origin", "https://github.com/detected/from-git.git")

		release := &Release{
			Version:    "v1.0.0",
			Changelog:  "Changes",
			ProjectDir: tmpDir,
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{Type: "github"}
		relCfg := &mockReleaseConfig{repository: ""}

		var err error
		output := capturePublisherOutput(t, func() {
			err = p.Publish(context.Background(), release, pubCfg, relCfg, true)
		})
		require.NoError(t, err)
		assert.Contains(t, output, "Repository: detected/from-git")
	})

	t.Run("fails when no repository available", func(t *testing.T) {
		tmpDir := t.TempDir()

		release := &Release{
			Version:    "v1.0.0",
			Changelog:  "Changes",
			ProjectDir: tmpDir,
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{Type: "github"}
		relCfg := &mockReleaseConfig{repository: ""}

		err := p.Publish(context.Background(), release, pubCfg, relCfg, true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not determine repository")
	})
}

func TestIntegration_GitHubPublisherIntegrationArtifactUpload_Good(t *testing.T) {
	p := NewGitHubPublisher()

	t.Run("dry run lists all artifact types", func(t *testing.T) {
		release := &Release{
			Version:    "v1.0.0",
			Changelog:  "Release notes",
			ProjectDir: "/tmp",
			FS:         io.Local,
			Artifacts: []build.Artifact{
				{Path: "/dist/app-linux-amd64.tar.gz", Checksum: "abc123"},
				{Path: "/dist/app-darwin-arm64.tar.gz", Checksum: "def456"},
				{Path: "/dist/app-windows-amd64.zip", Checksum: "ghi789"},
				{Path: "/dist/CHECKSUMS.txt"},
				{Path: "/dist/app-linux-amd64.tar.gz.sig"},
			},
		}
		pubCfg := PublisherConfig{Type: "github"}

		var err error
		output := capturePublisherOutput(t, func() {
			err = p.dryRunPublish(release, pubCfg, "owner/repo")
		})
		require.NoError(t, err)

		assert.Contains(t, output, "Would upload artifacts:")
		assert.Contains(t, output, "app-linux-amd64.tar.gz")
		assert.Contains(t, output, "app-darwin-arm64.tar.gz")
		assert.Contains(t, output, "app-windows-amd64.zip")
		assert.Contains(t, output, "CHECKSUMS.txt")
		assert.Contains(t, output, "app-linux-amd64.tar.gz.sig")
	})

	t.Run("executePublish appends artifact paths to gh command", func(t *testing.T) {
		release := &Release{
			Version:    "v1.0.0",
			Changelog:  "Changes",
			ProjectDir: "/tmp",
			FS:         io.Local,
			Artifacts: []build.Artifact{
				{Path: "/dist/file1.tar.gz"},
				{Path: "/dist/file2.zip"},
			},
		}
		pubCfg := PublisherConfig{Type: "github"}

		args := p.buildCreateArgs(release, pubCfg, "owner/repo")

		// The executePublish method appends artifact paths after these base args
		for _, a := range release.Artifacts {
			args = append(args, a.Path)
		}

		// Verify artifacts are at end of args
		assert.Equal(t, "/dist/file1.tar.gz", args[len(args)-2])
		assert.Equal(t, "/dist/file2.zip", args[len(args)-1])
	})
}

// --- Docker Publisher Integration Tests ---

func TestIntegration_DockerPublisherIntegrationDryRunNoSideEffects_Good(t *testing.T) {
	if err := validateDockerCli(); err != nil {
		t.Skip("skipping: docker CLI not available")
	}

	p := NewDockerPublisher()

	t.Run("dry run creates no images or containers", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a Dockerfile
		err := ax.WriteFile(ax.Join(tmpDir, "Dockerfile"), []byte("FROM alpine:latest\n"), 0o644)
		require.NoError(t, err)

		release := &Release{
			Version:    "v1.2.3",
			ProjectDir: tmpDir,
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{
			Type: "docker",
			Extended: map[string]any{
				"registry":  "ghcr.io",
				"image":     "test-org/test-app",
				"platforms": []any{"linux/amd64", "linux/arm64"},
				"tags":      []any{"latest", "{{.Version}}", "stable"},
				"build_args": map[string]any{
					"APP_VERSION": "{{.Version}}",
					"GO_VERSION":  "1.21",
				},
			},
		}
		relCfg := &mockReleaseConfig{repository: "test-org/test-app"}

		output := capturePublisherOutput(t, func() {
			err = p.Publish(context.Background(), release, pubCfg, relCfg, true)
		})
		require.NoError(t, err)

		// Verify dry run output
		assert.Contains(t, output, "DRY RUN: Docker Build & Push")
		assert.Contains(t, output, "Version:       v1.2.3")
		assert.Contains(t, output, "Registry:      ghcr.io")
		assert.Contains(t, output, "Image:         test-org/test-app")
		assert.Contains(t, output, "Platforms:     linux/amd64, linux/arm64")

		// Verify resolved tags
		assert.Contains(t, output, "ghcr.io/test-org/test-app:latest")
		assert.Contains(t, output, "ghcr.io/test-org/test-app:v1.2.3")
		assert.Contains(t, output, "ghcr.io/test-org/test-app:stable")

		// Verify build args shown
		assert.Contains(t, output, "Build arguments:")
		assert.Contains(t, output, "GO_VERSION=1.21")

		// Verify command
		assert.Contains(t, output, "docker buildx build")
		assert.Contains(t, output, "END DRY RUN")
	})

	t.Run("dry run produces correct buildx command for multi-platform", func(t *testing.T) {
		cfg := DockerConfig{
			Registry:   "ghcr.io",
			Image:      "org/app",
			Dockerfile: "/project/Dockerfile",
			Platforms:  []string{"linux/amd64", "linux/arm64", "linux/arm/v7"},
			Tags:       []string{"latest", "{{.Version}}"},
			BuildArgs: map[string]string{
				"CUSTOM_ARG": "custom_value",
			},
		}
		tags := p.resolveTags(cfg.Tags, "v3.0.0")
		args := p.buildBuildxArgs(cfg, tags, "v3.0.0")

		// Verify multi-platform string
		foundPlatform := false
		for i, arg := range args {
			if arg == "--platform" && i+1 < len(args) {
				foundPlatform = true
				assert.Equal(t, "linux/amd64,linux/arm64,linux/arm/v7", args[i+1])
			}
		}
		assert.True(t, foundPlatform, "should have --platform flag")

		// Verify tags
		assert.Contains(t, args, "ghcr.io/org/app:latest")
		assert.Contains(t, args, "ghcr.io/org/app:v3.0.0")

		// Verify build args
		foundCustom := false
		foundVersion := false
		for i, arg := range args {
			if arg == "--build-arg" && i+1 < len(args) {
				if args[i+1] == "CUSTOM_ARG=custom_value" {
					foundCustom = true
				}
				if args[i+1] == "VERSION=v3.0.0" {
					foundVersion = true
				}
			}
		}
		assert.True(t, foundCustom, "CUSTOM_ARG build arg not found")
		assert.True(t, foundVersion, "auto-added VERSION build arg not found")

		// Verify push flag
		assert.Contains(t, args, "--push")
	})
}

func TestIntegration_DockerPublisherIntegrationConfigParsing_Good(t *testing.T) {
	p := NewDockerPublisher()

	t.Run("full config round-trip from PublisherConfig to DockerConfig", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "docker",
			Extended: map[string]any{
				"registry":   "registry.example.com",
				"image":      "myteam/myservice",
				"dockerfile": "deploy/Dockerfile.prod",
				"platforms":  []any{"linux/amd64"},
				"tags":       []any{"{{.Version}}", "latest", "release-{{.Version}}"},
				"build_args": map[string]any{
					"BUILD_ENV": "production",
					"VERSION":   "{{.Version}}",
				},
			},
		}
		relCfg := &mockReleaseConfig{repository: "fallback/repo"}

		cfg := p.parseConfig(io.Local, pubCfg, relCfg, "/myproject")

		assert.Equal(t, "registry.example.com", cfg.Registry)
		assert.Equal(t, "myteam/myservice", cfg.Image)
		assert.Equal(t, "/myproject/deploy/Dockerfile.prod", cfg.Dockerfile)
		assert.Equal(t, []string{"linux/amd64"}, cfg.Platforms)
		assert.Equal(t, []string{"{{.Version}}", "latest", "release-{{.Version}}"}, cfg.Tags)
		assert.Equal(t, "production", cfg.BuildArgs["BUILD_ENV"])
		assert.Equal(t, "{{.Version}}", cfg.BuildArgs["VERSION"])

		// Verify tag resolution
		resolved := p.resolveTags(cfg.Tags, "v2.5.0")
		assert.Equal(t, []string{"v2.5.0", "latest", "release-v2.5.0"}, resolved)
	})
}

// --- Homebrew Publisher Integration Tests ---

func TestIntegration_HomebrewPublisherIntegrationDryRunNoSideEffects_Good(t *testing.T) {
	p := NewHomebrewPublisher()

	t.Run("dry run generates formula without writing files", func(t *testing.T) {
		tmpDir := t.TempDir()

		release := &Release{
			Version:    "v2.1.0",
			ProjectDir: tmpDir,
			FS:         io.Local,
			Artifacts: []build.Artifact{
				{Path: "/dist/myapp-darwin-amd64.tar.gz", Checksum: "sha256_darwin_amd64"},
				{Path: "/dist/myapp-darwin-arm64.tar.gz", Checksum: "sha256_darwin_arm64"},
				{Path: "/dist/myapp-linux-amd64.tar.gz", Checksum: "sha256_linux_amd64"},
				{Path: "/dist/myapp-linux-arm64.tar.gz", Checksum: "sha256_linux_arm64"},
			},
		}
		pubCfg := PublisherConfig{
			Type: "homebrew",
			Extended: map[string]any{
				"tap":     "test-org/homebrew-tap",
				"formula": "my-cli",
			},
		}
		relCfg := &mockReleaseConfig{repository: "test-org/my-cli", projectName: "my-cli"}

		var err error
		output := capturePublisherOutput(t, func() {
			err = p.Publish(context.Background(), release, pubCfg, relCfg, true)
		})
		require.NoError(t, err)

		// Verify dry run output
		assert.Contains(t, output, "DRY RUN: Homebrew Publish")
		assert.Contains(t, output, "Formula:    MyCli")
		assert.Contains(t, output, "Version:    2.1.0")
		assert.Contains(t, output, "Tap:        test-org/homebrew-tap")
		assert.Contains(t, output, "Repository: test-org/my-cli")

		// Verify generated formula content
		assert.Contains(t, output, "class MyCli < Formula")
		assert.Contains(t, output, `version "2.1.0"`)
		assert.Contains(t, output, "sha256_darwin_amd64")
		assert.Contains(t, output, "sha256_darwin_arm64")
		assert.Contains(t, output, "sha256_linux_amd64")
		assert.Contains(t, output, "sha256_linux_arm64")

		assert.Contains(t, output, "Would commit to tap: test-org/homebrew-tap")

		// Verify no files created
		entries, err := ax.ReadDir(tmpDir)
		require.NoError(t, err)
		assert.Empty(t, entries, "dry run should not create any files")
	})

	t.Run("dry run with official config shows output path", func(t *testing.T) {
		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: "/project",
			FS:         io.Local,
			Artifacts:  []build.Artifact{},
		}
		pubCfg := PublisherConfig{
			Type: "homebrew",
			Extended: map[string]any{
				"official": map[string]any{
					"enabled": true,
					"output":  "dist/homebrew-official",
				},
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo", projectName: "repo"}

		var err error
		output := capturePublisherOutput(t, func() {
			err = p.Publish(context.Background(), release, pubCfg, relCfg, true)
		})
		require.NoError(t, err)
		assert.Contains(t, output, "Would write files for official PR to: dist/homebrew-official")
	})
}

func TestIntegration_HomebrewPublisherIntegrationFormulaGeneration_Good(t *testing.T) {
	p := NewHomebrewPublisher()

	t.Run("generated formula contains correct Ruby class structure", func(t *testing.T) {
		data := homebrewTemplateData{
			FormulaClass: "CoreCli",
			Description:  "Core CLI tool",
			Repository:   "host-uk/core-cli",
			Version:      "3.0.0",
			License:      "MIT",
			BinaryName:   "core",
			Checksums: ChecksumMap{
				DarwinAmd64: "a1b2c3d4e5f6",
				DarwinArm64: "f6e5d4c3b2a1",
				LinuxAmd64:  "112233445566",
				LinuxArm64:  "665544332211",
			},
		}

		formula, err := p.renderTemplate(io.Local, "templates/homebrew/formula.rb.tmpl", data)
		require.NoError(t, err)

		// Verify class definition
		assert.Contains(t, formula, "class CoreCli < Formula")

		// Verify metadata
		assert.Contains(t, formula, `desc "Core CLI tool"`)
		assert.Contains(t, formula, `version "3.0.0"`)
		assert.Contains(t, formula, `license "MIT"`)

		// Verify checksums for all platforms
		assert.Contains(t, formula, "a1b2c3d4e5f6")
		assert.Contains(t, formula, "f6e5d4c3b2a1")
		assert.Contains(t, formula, "112233445566")
		assert.Contains(t, formula, "665544332211")

		// Verify binary install
		assert.Contains(t, formula, `bin.install "core"`)
	})

	t.Run("toFormulaClass handles various naming patterns", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"my-app", "MyApp"},
			{"core", "Core"},
			{"go-devops", "GoDevops"},
			{"a-b-c-d", "ABCD"},
			{"single", "Single"},
			{"UPPER", "UPPER"},
		}

		for _, tc := range tests {
			t.Run(tc.input, func(t *testing.T) {
				result := toFormulaClass(tc.input)
				assert.Equal(t, tc.expected, result)
			})
		}
	})
}

// --- Scoop Publisher Integration Tests ---

func TestIntegration_ScoopPublisherIntegrationDryRunNoSideEffects_Good(t *testing.T) {
	p := NewScoopPublisher()

	t.Run("dry run generates manifest without writing files", func(t *testing.T) {
		tmpDir := t.TempDir()

		release := &Release{
			Version:    "v1.5.0",
			ProjectDir: tmpDir,
			FS:         io.Local,
			Artifacts: []build.Artifact{
				{Path: "/dist/myapp-windows-amd64.zip", Checksum: "win64hash"},
				{Path: "/dist/myapp-windows-arm64.zip", Checksum: "winarm64hash"},
			},
		}
		pubCfg := PublisherConfig{
			Type: "scoop",
			Extended: map[string]any{
				"bucket": "test-org/scoop-bucket",
			},
		}
		relCfg := &mockReleaseConfig{repository: "test-org/myapp", projectName: "myapp"}

		var err error
		output := capturePublisherOutput(t, func() {
			err = p.Publish(context.Background(), release, pubCfg, relCfg, true)
		})
		require.NoError(t, err)

		assert.Contains(t, output, "DRY RUN: Scoop Publish")
		assert.Contains(t, output, "Package:    myapp")
		assert.Contains(t, output, "Version:    1.5.0")
		assert.Contains(t, output, "Bucket:     test-org/scoop-bucket")
		assert.Contains(t, output, "Generated manifest.json:")
		assert.Contains(t, output, `"version": "1.5.0"`)
		assert.Contains(t, output, "Would commit to bucket: test-org/scoop-bucket")

		// Verify no files created
		entries, err := ax.ReadDir(tmpDir)
		require.NoError(t, err)
		assert.Empty(t, entries)
	})
}

// --- AUR Publisher Integration Tests ---

func TestIntegration_AURPublisherIntegrationDryRunNoSideEffects_Good(t *testing.T) {
	p := NewAURPublisher()

	t.Run("dry run generates PKGBUILD and SRCINFO without writing files", func(t *testing.T) {
		tmpDir := t.TempDir()

		release := &Release{
			Version:    "v2.0.0",
			ProjectDir: tmpDir,
			FS:         io.Local,
			Artifacts: []build.Artifact{
				{Path: "/dist/myapp-linux-amd64.tar.gz", Checksum: "amd64hash"},
				{Path: "/dist/myapp-linux-arm64.tar.gz", Checksum: "arm64hash"},
			},
		}
		pubCfg := PublisherConfig{
			Type: "aur",
			Extended: map[string]any{
				"maintainer": "Test User <test@example.com>",
			},
		}
		relCfg := &mockReleaseConfig{repository: "test-org/myapp", projectName: "myapp"}

		var err error
		output := capturePublisherOutput(t, func() {
			err = p.Publish(context.Background(), release, pubCfg, relCfg, true)
		})
		require.NoError(t, err)

		assert.Contains(t, output, "DRY RUN: AUR Publish")
		assert.Contains(t, output, "Package:    myapp-bin")
		assert.Contains(t, output, "Version:    2.0.0")
		assert.Contains(t, output, "Maintainer: Test User <test@example.com>")
		assert.Contains(t, output, "Generated PKGBUILD:")
		assert.Contains(t, output, "pkgname=myapp-bin")
		assert.Contains(t, output, "pkgver=2.0.0")
		assert.Contains(t, output, "Generated .SRCINFO:")
		assert.Contains(t, output, "pkgbase = myapp-bin")
		assert.Contains(t, output, "Would push to AUR:")

		// Verify no files created
		entries, err := ax.ReadDir(tmpDir)
		require.NoError(t, err)
		assert.Empty(t, entries)
	})
}

// --- Chocolatey Publisher Integration Tests ---

func TestIntegration_ChocolateyPublisherIntegrationDryRunNoSideEffects_Good(t *testing.T) {
	p := NewChocolateyPublisher()

	t.Run("dry run generates nuspec and install script without side effects", func(t *testing.T) {
		tmpDir := t.TempDir()

		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: tmpDir,
			FS:         io.Local,
			Artifacts: []build.Artifact{
				{Path: "/dist/myapp-windows-amd64.zip", Checksum: "choco_hash"},
			},
		}
		pubCfg := PublisherConfig{
			Type: "chocolatey",
			Extended: map[string]any{
				"package": "my-cli-tool",
				"push":    false,
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/my-cli-tool", projectName: "my-cli-tool"}

		var err error
		output := capturePublisherOutput(t, func() {
			err = p.Publish(context.Background(), release, pubCfg, relCfg, true)
		})
		require.NoError(t, err)

		assert.Contains(t, output, "DRY RUN: Chocolatey Publish")
		assert.Contains(t, output, "Package:    my-cli-tool")
		assert.Contains(t, output, "Version:    1.0.0")
		assert.Contains(t, output, "Push:       false")
		assert.Contains(t, output, "Generated package.nuspec:")
		assert.Contains(t, output, "<id>my-cli-tool</id>")
		assert.Contains(t, output, "Generated chocolateyinstall.ps1:")
		assert.Contains(t, output, "Would generate package files only")

		// Verify no files created
		entries, err := ax.ReadDir(tmpDir)
		require.NoError(t, err)
		assert.Empty(t, entries)
	})
}

// --- npm Publisher Integration Tests ---

func TestIntegration_NpmPublisherIntegrationDryRunNoSideEffects_Good(t *testing.T) {
	p := NewNpmPublisher()

	t.Run("dry run generates package.json without writing files or publishing", func(t *testing.T) {
		tmpDir := t.TempDir()

		release := &Release{
			Version:    "v3.0.0",
			ProjectDir: tmpDir,
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{
			Type: "npm",
			Extended: map[string]any{
				"package": "@test-org/my-cli",
				"access":  "public",
			},
		}
		relCfg := &mockReleaseConfig{repository: "test-org/my-cli", projectName: "my-cli"}

		var err error
		output := capturePublisherOutput(t, func() {
			err = p.Publish(context.Background(), release, pubCfg, relCfg, true)
		})
		require.NoError(t, err)

		assert.Contains(t, output, "DRY RUN: npm Publish")
		assert.Contains(t, output, "Package:    @test-org/my-cli")
		assert.Contains(t, output, "Version:    3.0.0")
		assert.Contains(t, output, "Access:     public")
		assert.Contains(t, output, "Generated package.json:")
		assert.Contains(t, output, `"name": "@test-org/my-cli"`)
		assert.Contains(t, output, `"version": "3.0.0"`)
		assert.Contains(t, output, "Would run: npm publish --access public")

		// Verify no files created
		entries, err := ax.ReadDir(tmpDir)
		require.NoError(t, err)
		assert.Empty(t, entries)
	})
}

// --- LinuxKit Publisher Integration Tests ---

func TestIntegration_LinuxKitPublisherIntegrationDryRunNoSideEffects_Good(t *testing.T) {
	if err := validateLinuxKitCli(); err != nil {
		t.Skip("skipping: linuxkit CLI not available")
	}

	p := NewLinuxKitPublisher()

	t.Run("dry run with multiple formats and platforms", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create config file
		configDir := ax.Join(tmpDir, ".core", "linuxkit")
		require.NoError(t, ax.MkdirAll(configDir, 0o755))
		require.NoError(t, ax.WriteFile(ax.Join(configDir, "server.yml"), []byte("kernel:\n  image: test\n"), 0o644))

		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: tmpDir,
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{
			Type: "linuxkit",
			Extended: map[string]any{
				"formats":   []any{"iso", "qcow2", "docker"},
				"platforms": []any{"linux/amd64", "linux/arm64"},
			},
		}
		relCfg := &mockReleaseConfig{repository: "test-org/my-os"}

		var err error
		output := capturePublisherOutput(t, func() {
			err = p.Publish(context.Background(), release, pubCfg, relCfg, true)
		})
		require.NoError(t, err)

		assert.Contains(t, output, "DRY RUN: LinuxKit Build & Publish")
		assert.Contains(t, output, "Formats:       iso, qcow2, docker")
		assert.Contains(t, output, "Platforms:     linux/amd64, linux/arm64")

		// Verify all combinations listed
		assert.Contains(t, output, "linuxkit-1.0.0-amd64.iso")
		assert.Contains(t, output, "linuxkit-1.0.0-amd64.qcow2")
		assert.Contains(t, output, "linuxkit-1.0.0-amd64.docker.tar")
		assert.Contains(t, output, "linuxkit-1.0.0-arm64.iso")
		assert.Contains(t, output, "linuxkit-1.0.0-arm64.qcow2")
		assert.Contains(t, output, "linuxkit-1.0.0-arm64.docker.tar")

		// Verify docker usage hint
		assert.Contains(t, output, "docker load")

		// Verify no files created in dist
		distDir := ax.Join(tmpDir, "dist")
		assert.False(t, ax.Exists(distDir), "dry run should not create dist directory")
	})
}

// --- Cross-Publisher Integration Tests ---

func TestIntegration_AllPublishersIntegrationNameUniqueness_Good(t *testing.T) {
	t.Run("all publishers have unique names", func(t *testing.T) {
		publishers := []Publisher{
			NewGitHubPublisher(),
			NewDockerPublisher(),
			NewHomebrewPublisher(),
			NewNpmPublisher(),
			NewScoopPublisher(),
			NewAURPublisher(),
			NewChocolateyPublisher(),
			NewLinuxKitPublisher(),
		}

		names := make(map[string]bool)
		for _, pub := range publishers {
			name := pub.Name()
			assert.False(t, names[name], "duplicate publisher name: %s", name)
			names[name] = true
			assert.NotEmpty(t, name, "publisher name should not be empty")
		}

		assert.Len(t, names, 8, "should have 8 unique publishers")
	})
}

func TestIntegration_AllPublishersIntegrationNilRelCfg_Good(t *testing.T) {
	t.Run("github handles nil relCfg with git repo", func(t *testing.T) {
		tmpDir := t.TempDir()

		runPublisherCommand(t, tmpDir, "git", "init")
		runPublisherCommand(t, tmpDir, "git", "remote", "add", "origin", "git@github.com:niltest/repo.git")

		release := &Release{
			Version:    "v1.0.0",
			Changelog:  "Changes",
			ProjectDir: tmpDir,
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{Type: "github"}

		var err error
		output := capturePublisherOutput(t, func() {
			err = NewGitHubPublisher().Publish(context.Background(), release, pubCfg, nil, true)
		})
		require.NoError(t, err)
		assert.Contains(t, output, "niltest/repo")
	})
}

func TestIntegration_BuildChecksumMapIntegration_Good(t *testing.T) {
	t.Run("maps all platforms correctly from realistic artifacts", func(t *testing.T) {
		artifacts := []build.Artifact{
			{Path: "/dist/core-v1.0.0-darwin-amd64.tar.gz", Checksum: "da64"},
			{Path: "/dist/core-v1.0.0-darwin-arm64.tar.gz", Checksum: "da65"},
			{Path: "/dist/core-v1.0.0-linux-amd64.tar.gz", Checksum: "la64"},
			{Path: "/dist/core-v1.0.0-linux-arm64.tar.gz", Checksum: "la65"},
			{Path: "/dist/core-v1.0.0-windows-amd64.zip", Checksum: "wa64"},
			{Path: "/dist/core-v1.0.0-windows-arm64.zip", Checksum: "wa65"},
			{Path: "/dist/CHECKSUMS.txt"}, // No checksum for checksum file
		}

		checksums := buildChecksumMap(artifacts)

		assert.Equal(t, "da64", checksums.DarwinAmd64)
		assert.Equal(t, "da65", checksums.DarwinArm64)
		assert.Equal(t, "la64", checksums.LinuxAmd64)
		assert.Equal(t, "la65", checksums.LinuxArm64)
		assert.Equal(t, "wa64", checksums.WindowsAmd64)
		assert.Equal(t, "wa65", checksums.WindowsArm64)
	})
}

// indexOf returns the index of an element in a string slice, or -1 if not found.
func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}

// Compile-time check: all publishers implement Publisher interface
var _ Publisher = (*GitHubPublisher)(nil)
var _ Publisher = (*DockerPublisher)(nil)
var _ Publisher = (*HomebrewPublisher)(nil)
var _ Publisher = (*NpmPublisher)(nil)
var _ Publisher = (*ScoopPublisher)(nil)
var _ Publisher = (*AURPublisher)(nil)
var _ Publisher = (*ChocolateyPublisher)(nil)
var _ Publisher = (*LinuxKitPublisher)(nil)
