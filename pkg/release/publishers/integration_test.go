package publishers

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	storage "dappco.re/go/build/pkg/storage"
)

// --- GitHub Publisher Integration Tests ---

func TestIntegration_GitHubPublisherIntegrationDryRunNoSideEffectsGood(t *testing.T) {
	p := NewGitHubPublisher()

	t.Run("dry run creates no files on disk", func(t *testing.T) {
		tmpDir := t.TempDir()
		release := &Release{
			Version:    "v1.0.0",
			Changelog:  "## v1.0.0\n\n- feat: initial release",
			ProjectDir: tmpDir,
			FS:         storage.Local,
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

		publishResult := core.Ok(nil)
		output := capturePublisherOutput(t, func() {
			publishResult = p.Publish(context.Background(), release, pubCfg, relCfg, true)
		})
		requirePublisherOK(t, publishResult)
		if !stdlibAssertContains(output, "DRY RUN: GitHub Release") {
			t.Fatalf("expected %v to contain %v", output, "DRY RUN: GitHub Release")
		}
		if !stdlibAssertContains(output, "Repository: test-org/test-repo") {
			t.Fatalf("expected %v to contain %v", output, "Repository: test-org/test-repo")
		}
		if !stdlibAssertContains(output, "Version:    v1.0.0") {
			t.Fatalf("expected %v to contain %v", output, "Version:    v1.0.0")
		}
		if !stdlibAssertContains(output, "Draft:      true") {
			t.Fatalf("expected %v to contain %v", output, "Draft:      true")
		}
		if !stdlibAssertContains(output, "Prerelease: true") {
			t.Fatalf("expected %v to contain %v", output, "Prerelease: true")
		}
		if !stdlibAssertContains(output, "Would upload artifacts:") {

			// Verify no files were created in the temp directory
			t.Fatalf("expected %v to contain %v", output, "Would upload artifacts:")
		}
		if !stdlibAssertContains(output, "app-linux-amd64.tar.gz") {
			t.Fatalf("expected %v to contain %v", output, "app-linux-amd64.tar.gz")
		}
		if !stdlibAssertContains(output, "app-darwin-arm64.tar.gz") {
			t.Fatalf("expected %v to contain %v", output, "app-darwin-arm64.tar.gz")
		}
		if !stdlibAssertContains(output, "CHECKSUMS.txt") {
			t.Fatalf("expected %v to contain %v", output, "CHECKSUMS.txt")
		}
		if !stdlibAssertContains(output, "gh release create") {
			t.Fatalf("expected %v to contain %v", output, "gh release create")
		}
		if !stdlibAssertContains(output, "--draft") {
			t.Fatalf("expected %v to contain %v", output, "--draft")
		}
		if !stdlibAssertContains(

			// Verify exact argument structure
			output, "--prerelease") {
			t.Fatalf("expected %v to contain %v", output, "--prerelease")
		}

		entries := requirePublisherDirEntries(t, ax.ReadDir(tmpDir))
		if !stdlibAssertEmpty(entries) {
			t.Fatal("dry run should not create any files")
		}

	})

	t.Run("dry run builds correct gh CLI command for standard release", func(t *testing.T) {
		release := &Release{
			Version:    "v2.3.0",
			Changelog:  "## v2.3.0\n\n### Features\n\n- new feature",
			ProjectDir: "/tmp",
			FS:         storage.Local,
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
		if !stdlibAssertEqual("release", args[0]) {
			t.Fatalf("want %v, got %v", "release", args[0])
		}
		if !stdlibAssertEqual("create", args[1]) {
			t.Fatalf("want %v, got %v", "create",

				// Should have --repo
				args[1])
		}
		if !stdlibAssertEqual("v2.3.0", args[2]) {
			t.Fatalf("want %v, got %v", "v2.3.0", args[2])
		}

		repoIdx := indexOf(args, "--repo")
		if repoIdx <= -1 {
			t.Fatalf("expected %v to be greater than %v", repoIdx, -1)
		}
		if !stdlibAssertEqual(

			// Should have --title
			"owner/repo", args[repoIdx+1]) {
			t.Fatalf("want %v, got %v", "owner/repo", args[repoIdx+1])
		}

		titleIdx := indexOf(args, "--title")
		if titleIdx <= -1 {
			t.Fatalf("expected %v to be greater than %v", titleIdx, -1)
		}
		if !stdlibAssertEqual(

			// Should have --notes (since changelog is non-empty)
			"v2.3.0", args[titleIdx+1]) {
			t.Fatalf("want %v, got %v", "v2.3.0", args[titleIdx+1])
		}
		if !stdlibAssertContains(

			// Should NOT have --draft or --prerelease
			args, "--notes") {
			t.Fatalf("expected %v to contain %v", args, "--notes")
		}
		if stdlibAssertContains(args, "--draft") {
			t.Fatalf("expected %v not to contain %v", args, "--draft")
		}
		if stdlibAssertContains(args, "--prerelease") {
			t.Fatalf("expected %v not to contain %v", args, "--prerelease")
		}

	})

	t.Run("dry run uses generate-notes when changelog empty", func(t *testing.T) {
		release := &Release{
			Version:    "v1.0.0",
			Changelog:  "",
			ProjectDir: "/tmp",
			FS:         storage.Local,
		}
		pubCfg := PublisherConfig{Type: "github"}

		args := p.buildCreateArgs(release, pubCfg, "owner/repo")
		if !stdlibAssertContains(args, "--generate-notes") {
			t.Fatalf("expected %v to contain %v", args, "--generate-notes")
		}
		if stdlibAssertContains(args, "--notes") {
			t.Fatalf("expected %v not to contain %v", args, "--notes")
		}

	})
}

func TestIntegration_GitHubPublisherIntegrationRepositoryDetectionGood(t *testing.T) {
	p := NewGitHubPublisher()

	t.Run("uses relCfg repository when provided", func(t *testing.T) {
		release := &Release{
			Version:    "v1.0.0",
			Changelog:  "Changes",
			ProjectDir: "/tmp",
			FS:         storage.Local,
		}
		pubCfg := PublisherConfig{Type: "github"}
		relCfg := &mockReleaseConfig{repository: "explicit/repo"}

		publishResult := core.Ok(nil)
		output := capturePublisherOutput(t, func() {
			publishResult = p.Publish(context.Background(), release, pubCfg, relCfg, true)
		})
		requirePublisherOK(t, publishResult)
		if !stdlibAssertContains(output, "Repository: explicit/repo") {
			t.Fatalf("expected %v to contain %v", output, "Repository: explicit/repo")
		}

	})

	t.Run("detects repository from git remote when relCfg empty", func(t *testing.T) {
		tmpDir := t.TempDir()

		runPublisherCommand(t, tmpDir, "git", "init")
		runPublisherCommand(t, tmpDir, "git", "remote", "add", "origin", "https://github.com/detected/from-git.git")

		release := &Release{
			Version:    "v1.0.0",
			Changelog:  "Changes",
			ProjectDir: tmpDir,
			FS:         storage.Local,
		}
		pubCfg := PublisherConfig{Type: "github"}
		relCfg := &mockReleaseConfig{repository: ""}

		publishResult := core.Ok(nil)
		output := capturePublisherOutput(t, func() {
			publishResult = p.Publish(context.Background(), release, pubCfg, relCfg, true)
		})
		requirePublisherOK(t, publishResult)
		if !stdlibAssertContains(output, "Repository: detected/from-git") {
			t.Fatalf("expected %v to contain %v", output, "Repository: detected/from-git")
		}

	})

	t.Run("fails when no repository available", func(t *testing.T) {
		tmpDir := t.TempDir()

		release := &Release{
			Version:    "v1.0.0",
			Changelog:  "Changes",
			ProjectDir: tmpDir,
			FS:         storage.Local,
		}
		pubCfg := PublisherConfig{Type: "github"}
		relCfg := &mockReleaseConfig{repository: ""}

		err := requirePublisherError(t, p.Publish(context.Background(), release, pubCfg, relCfg, true))
		if !stdlibAssertContains(err, "could not determine repository") {
			t.Fatalf("expected %v to contain %v", err, "could not determine repository")
		}

	})
}

func TestIntegration_GitHubPublisherIntegrationArtifactUploadGood(t *testing.T) {
	p := NewGitHubPublisher()

	t.Run("dry run lists all artifact types", func(t *testing.T) {
		release := &Release{
			Version:    "v1.0.0",
			Changelog:  "Release notes",
			ProjectDir: "/tmp",
			FS:         storage.Local,
			Artifacts: []build.Artifact{
				{Path: "/dist/app-linux-amd64.tar.gz", Checksum: "abc123"},
				{Path: "/dist/app-darwin-arm64.tar.gz", Checksum: "def456"},
				{Path: "/dist/app-windows-amd64.zip", Checksum: "ghi789"},
				{Path: "/dist/CHECKSUMS.txt"},
				{Path: "/dist/app-linux-amd64.tar.gz.sig"},
			},
		}
		pubCfg := PublisherConfig{Type: "github"}

		publishResult := core.Ok(nil)
		output := capturePublisherOutput(t, func() {
			publishResult = p.dryRunPublish(release, pubCfg, "owner/repo")
		})
		requirePublisherOK(t, publishResult)
		if !stdlibAssertContains(output, "Would upload artifacts:") {
			t.Fatalf("expected %v to contain %v", output, "Would upload artifacts:")
		}
		if !stdlibAssertContains(output, "app-linux-amd64.tar.gz") {
			t.Fatalf("expected %v to contain %v", output, "app-linux-amd64.tar.gz")
		}
		if !stdlibAssertContains(output, "app-darwin-arm64.tar.gz") {
			t.Fatalf("expected %v to contain %v", output, "app-darwin-arm64.tar.gz")
		}
		if !stdlibAssertContains(output, "app-windows-amd64.zip") {
			t.Fatalf("expected %v to contain %v", output, "app-windows-amd64.zip")
		}
		if !stdlibAssertContains(output, "CHECKSUMS.txt") {
			t.Fatalf("expected %v to contain %v", output, "CHECKSUMS.txt")
		}
		if !stdlibAssertContains(output, "app-linux-amd64.tar.gz.sig") {
			t.Fatalf("expected %v to contain %v", output, "app-linux-amd64.tar.gz.sig")

			// The executePublish method appends artifact paths after these base args
		}

	})

	t.Run("executePublish appends artifact paths to gh command", func(t *testing.T) {
		release := &Release{
			Version:    "v1.0.0",
			Changelog:  "Changes",
			ProjectDir: "/tmp",
			FS:         storage.Local,
			Artifacts: []build.Artifact{
				{Path: "/dist/file1.tar.gz"},
				{Path: "/dist/file2.zip"},
			},
		}
		pubCfg := PublisherConfig{Type: "github"}

		args := p.buildCreateArgs(release, pubCfg, "owner/repo")

		for _, a := range release.Artifacts {
			args = append(args, a.Path)
		}
		if !stdlibAssertEqual(

			// Verify artifacts are at end of args
			"/dist/file1.tar.gz", args[len(args)-2]) {
			t.Fatalf("want %v, got %v", "/dist/file1.tar.gz", args[len(args)-2])
		}
		if !stdlibAssertEqual("/dist/file2.zip",

			// --- Docker Publisher Integration Tests ---
			args[len(args)-1]) {
			t.Fatalf("want %v, got %v", "/dist/file2.zip", args[len(args)-1])
		}

	})
}

func TestIntegration_DockerPublisherIntegrationDryRunNoSideEffectsGood(t *testing.T) {
	if err := validateDockerCli(); !err.OK {
		t.Skip("skipping: docker CLI not available")
	}

	p := NewDockerPublisher()

	t.Run("dry run creates no images or containers", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a Dockerfile
		requirePublisherOK(t, ax.WriteFile(ax.Join(tmpDir, "Dockerfile"), []byte("FROM alpine:latest\n"), 0o644))

		release := &Release{
			Version:    "v1.2.3",
			ProjectDir: tmpDir,
			FS:         storage.Local,
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

		publishResult := core.Ok(nil)
		output := capturePublisherOutput(t, func() {
			publishResult = p.Publish(context.Background(), release, pubCfg, relCfg, true)
		})
		requirePublisherOK(t, publishResult)
		if !stdlibAssertContains(output, "DRY RUN: Docker Build & Push") {
			t.Fatalf("expected %v to contain %v", output, "DRY RUN: Docker Build & Push")
		}
		if !stdlibAssertContains(output, "Version:       v1.2.3") {
			t.Fatalf("expected %v to contain %v", output, "Version:       v1.2.3")
		}
		if !stdlibAssertContains(output, "Registry:      ghcr.io") {

			// Verify resolved tags
			t.Fatalf("expected %v to contain %v", output, "Registry:      ghcr.io")
		}
		if !stdlibAssertContains(output, "Image:         test-org/test-app") {
			t.Fatalf("expected %v to contain %v", output, "Image:         test-org/test-app")
		}
		if

		// Verify build args shown
		!stdlibAssertContains(output, "Platforms:     linux/amd64, linux/arm64") {
			t.Fatalf("expected %v to contain %v", output, "Platforms:     linux/amd64, linux/arm64")

			// Verify command
		}
		if !stdlibAssertContains(output, "ghcr.io/test-org/test-app:latest") {
			t.Fatalf("expected %v to contain %v", output, "ghcr.io/test-org/test-app:latest")
		}
		if !stdlibAssertContains(output, "ghcr.io/test-org/test-app:v1.2.3") {
			t.Fatalf("expected %v to contain %v", output, "ghcr.io/test-org/test-app:v1.2.3")
		}
		if !stdlibAssertContains(output, "ghcr.io/test-org/test-app:stable") {
			t.Fatalf("expected %v to contain %v", output, "ghcr.io/test-org/test-app:stable")
		}
		if !stdlibAssertContains(output, "Build arguments:") {
			t.Fatalf("expected %v to contain %v", output, "Build arguments:")
		}
		if !stdlibAssertContains(output,

			// Verify multi-platform string
			"GO_VERSION=1.21") {
			t.Fatalf("expected %v to contain %v", output, "GO_VERSION=1.21")
		}
		if !stdlibAssertContains(output, "docker buildx build") {
			t.Fatalf("expected %v to contain %v", output, "docker buildx build")
		}
		if !stdlibAssertContains(output, "END DRY RUN") {
			t.Fatalf("expected %v to contain %v", output, "END DRY RUN")

			// Verify tags
		}

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

		foundPlatform := false
		for i, arg := range args {
			if arg == "--platform" && i+1 < len(args) {
				foundPlatform = true
				if !stdlibAssertEqual("linux/amd64,linux/arm64,linux/arm/v7", args[i+1]) {
					t.Fatalf("want %v, got %v", "linux/amd64,linux/arm64,linux/arm/v7", args[i+1])
				}

			}
		}
		if !(foundPlatform) {
			t.Fatal("should have --platform flag")
		}
		if !stdlibAssertContains(args, "ghcr.io/org/app:latest") {
			t.Fatalf("expected %v to contain %v", args, "ghcr.io/org/app:latest")
		}
		if !stdlibAssertContains(

			// Verify build args
			args, "ghcr.io/org/app:v3.0.0") {
			t.Fatalf("expected %v to contain %v", args, "ghcr.io/org/app:v3.0.0")
		}

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
		if !(foundCustom) {
			t.Fatal("CUSTOM_ARG build arg not found")
		}
		if !(foundVersion) {
			t.Fatal("auto-added VERSION build arg not found")
		}
		if !stdlibAssertContains(

			// Verify push flag
			args, "--push") {
			t.Fatalf("expected %v to contain %v", args, "--push")
		}

	})
}

func TestIntegration_DockerPublisherIntegrationConfigParsingGood(t *testing.T) {
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

		cfg := p.parseConfig(storage.Local, pubCfg, relCfg, "/myproject")
		if !stdlibAssertEqual("registry.example.com", cfg.Registry) {
			t.Fatalf("want %v, got %v", "registry.example.com", cfg.Registry)
		}
		if !stdlibAssertEqual("myteam/myservice", cfg.Image) {
			t.Fatalf("want %v, got %v", "myteam/myservice", cfg.Image)
		}
		if !stdlibAssertEqual("/myproject/deploy/Dockerfile.prod", cfg.Dockerfile) {
			t.Fatalf("want %v, got %v", "/myproject/deploy/Dockerfile.prod", cfg.Dockerfile)
		}
		if !stdlibAssertEqual([]string{"linux/amd64"}, cfg.Platforms) {
			t.Fatalf(

				// Verify tag resolution
				"want %v, got %v", []string{"linux/amd64"}, cfg.Platforms)
		}
		if !stdlibAssertEqual([]string{"{{.Version}}", "latest", "release-{{.Version}}"}, cfg.Tags) {
			t.Fatalf(

				// --- Homebrew Publisher Integration Tests ---
				"want %v, got %v", []string{"{{.Version}}", "latest", "release-{{.Version}}"}, cfg.Tags)
		}
		if !stdlibAssertEqual("production", cfg.BuildArgs["BUILD_ENV"]) {
			t.Fatalf("want %v, got %v", "production", cfg.BuildArgs["BUILD_ENV"])
		}
		if !stdlibAssertEqual("{{.Version}}", cfg.BuildArgs["VERSION"]) {
			t.Fatalf("want %v, got %v", "{{.Version}}", cfg.BuildArgs["VERSION"])
		}

		resolved := p.resolveTags(cfg.Tags, "v2.5.0")
		if !stdlibAssertEqual([]string{"v2.5.0", "latest", "release-v2.5.0"}, resolved) {
			t.Fatalf("want %v, got %v", []string{"v2.5.0", "latest", "release-v2.5.0"}, resolved)
		}

	})
}

func TestIntegration_HomebrewPublisherIntegrationDryRunNoSideEffectsGood(t *testing.T) {
	p := NewHomebrewPublisher()

	t.Run("dry run generates formula without writing files", func(t *testing.T) {
		tmpDir := t.TempDir()

		release := &Release{
			Version:    "v2.1.0",
			ProjectDir: tmpDir,
			FS:         storage.Local,
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

		publishResult := core.Ok(nil)
		output := capturePublisherOutput(t, func() {
			publishResult = p.Publish(context.Background(), release, pubCfg, relCfg, true)
		})
		requirePublisherOK(t, publishResult)
		if !stdlibAssertContains(output, "DRY RUN: Homebrew Publish") {
			t.Fatalf("expected %v to contain %v", output, "DRY RUN: Homebrew Publish")
		}
		if !stdlibAssertContains(output, "Formula:    MyCli") {
			t.Fatalf("expected %v to contain %v", output, "Formula:    MyCli")
		}
		if !stdlibAssertContains(output, "Version:    2.1.0") {

			// Verify generated formula content
			t.Fatalf("expected %v to contain %v", output, "Version:    2.1.0")
		}
		if !stdlibAssertContains(output, "Tap:        test-org/homebrew-tap") {
			t.Fatalf("expected %v to contain %v", output, "Tap:        test-org/homebrew-tap")
		}
		if !stdlibAssertContains(output, "Repository: test-org/my-cli") {
			t.Fatalf("expected %v to contain %v", output, "Repository: test-org/my-cli")
		}
		if !stdlibAssertContains(output, "class MyCli < Formula") {
			t.Fatalf(

				// Verify no files created
				"expected %v to contain %v", output, "class MyCli < Formula")
		}
		if !stdlibAssertContains(output, `version '2.1.0'`) {
			t.Fatalf("expected %v to contain %v", output, `version '2.1.0'`)
		}
		if !stdlibAssertContains(output, "sha256_darwin_amd64") {
			t.Fatalf("expected %v to contain %v", output, "sha256_darwin_amd64")
		}
		if !stdlibAssertContains(output, "sha256_darwin_arm64") {
			t.Fatalf("expected %v to contain %v", output, "sha256_darwin_arm64")
		}
		if !stdlibAssertContains(output, "sha256_linux_amd64") {
			t.Fatalf("expected %v to contain %v", output, "sha256_linux_amd64")
		}
		if !stdlibAssertContains(output, "sha256_linux_arm64") {
			t.Fatalf("expected %v to contain %v", output, "sha256_linux_arm64")
		}
		if !stdlibAssertContains(output, "Would commit to tap: test-org/homebrew-tap") {
			t.Fatalf("expected %v to contain %v", output, "Would commit to tap: test-org/homebrew-tap")
		}

		entries := requirePublisherDirEntries(t, ax.ReadDir(tmpDir))
		if !stdlibAssertEmpty(entries) {
			t.Fatal("dry run should not create any files")
		}

	})

	t.Run("dry run with official config shows output path", func(t *testing.T) {
		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: "/project",
			FS:         storage.Local,
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

		publishResult := core.Ok(nil)
		output := capturePublisherOutput(t, func() {
			publishResult = p.Publish(context.Background(), release, pubCfg, relCfg, true)
		})
		requirePublisherOK(t, publishResult)
		if !stdlibAssertContains(output, "Would write files for official PR to: dist/homebrew-official") {
			t.Fatalf("expected %v to contain %v", output, "Would write files for official PR to: dist/homebrew-official")
		}

	})
}

func TestIntegration_HomebrewPublisherIntegrationFormulaGenerationGood(t *testing.T) {
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

		formula := requirePublisherString(t, p.renderTemplate(storage.Local, "templates/homebrew/formula.rb.tmpl", data))
		if !stdlibAssertContains(formula, "class CoreCli < Formula") {
			t.Fatalf("expected %v to contain %v",

				// Verify metadata
				formula, "class CoreCli < Formula")
		}
		if !stdlibAssertContains(formula, `desc 'Core CLI tool'`) {
			t.Fatalf("expected %v to contain %v", formula, `desc 'Core CLI tool'`)

			// Verify checksums for all platforms
		}
		if !stdlibAssertContains(formula, `version '3.0.0'`) {
			t.Fatalf("expected %v to contain %v", formula, `version '3.0.0'`)
		}
		if !stdlibAssertContains(formula, `license 'MIT'`) {
			t.Fatalf("expected %v to contain %v", formula, `license 'MIT'`)

			// Verify binary install
		}
		if !stdlibAssertContains(formula, "a1b2c3d4e5f6") {
			t.Fatalf("expected %v to contain %v", formula, "a1b2c3d4e5f6")
		}
		if !stdlibAssertContains(formula, "f6e5d4c3b2a1") {
			t.Fatalf("expected %v to contain %v", formula, "f6e5d4c3b2a1")
		}
		if !stdlibAssertContains(formula, "112233445566") {
			t.Fatalf("expected %v to contain %v", formula, "112233445566")
		}
		if !stdlibAssertContains(formula, "665544332211") {
			t.Fatalf("expected %v to contain %v", formula, "665544332211")
		}
		if !stdlibAssertContains(formula, `bin.install 'core'`) {
			t.Fatalf("expected %v to contain %v", formula,

				// --- Scoop Publisher Integration Tests ---
				`bin.install 'core'`)
		}

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
				if !stdlibAssertEqual(tc.expected, result) {
					t.Fatalf("want %v, got %v", tc.expected, result)
				}

			})
		}
	})
}

func TestIntegration_ScoopPublisherIntegrationDryRunNoSideEffectsGood(t *testing.T) {
	p := NewScoopPublisher()

	t.Run("dry run generates manifest without writing files", func(t *testing.T) {
		tmpDir := t.TempDir()

		release := &Release{
			Version:    "v1.5.0",
			ProjectDir: tmpDir,
			FS:         storage.Local,
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

		publishResult := core.Ok(nil)
		output := capturePublisherOutput(t, func() {
			publishResult = p.Publish(context.Background(), release, pubCfg, relCfg, true)
		})
		requirePublisherOK(t, publishResult)
		if !stdlibAssertContains(output, "DRY RUN: Scoop Publish") {
			t.Fatalf("expected %v to contain %v", output, "DRY RUN: Scoop Publish")
		}
		if !stdlibAssertContains(output, "Package:    myapp") {
			t.Fatalf("expected %v to contain %v", output, "Package:    myapp")
		}
		if !stdlibAssertContains(output, "Version:    1.5.0") {
			t.Fatalf("expected %v to contain %v", output, "Version:    1.5.0")
		}
		if !stdlibAssertContains(output,

			// Verify no files created
			"Bucket:     test-org/scoop-bucket") {
			t.Fatalf("expected %v to contain %v", output, "Bucket:     test-org/scoop-bucket")
		}
		if !stdlibAssertContains(

			// --- AUR Publisher Integration Tests ---
			output, "Generated manifest.json:") {
			t.Fatalf("expected %v to contain %v", output, "Generated manifest.json:")
		}
		if !stdlibAssertContains(output, `"version": "1.5.0"`) {
			t.Fatalf("expected %v to contain %v", output, `"version": "1.5.0"`)
		}
		if !stdlibAssertContains(output, "Would commit to bucket: test-org/scoop-bucket") {
			t.Fatalf("expected %v to contain %v", output, "Would commit to bucket: test-org/scoop-bucket")
		}

		entries := requirePublisherDirEntries(t, ax.ReadDir(tmpDir))
		if !stdlibAssertEmpty(entries) {
			t.Fatalf("expected empty, got %v", entries)
		}

	})
}

func TestIntegration_AURPublisherIntegrationDryRunNoSideEffectsGood(t *testing.T) {
	p := NewAURPublisher()

	t.Run("dry run generates PKGBUILD and SRCINFO without writing files", func(t *testing.T) {
		tmpDir := t.TempDir()

		release := &Release{
			Version:    "v2.0.0",
			ProjectDir: tmpDir,
			FS:         storage.Local,
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

		publishResult := core.Ok(nil)
		output := capturePublisherOutput(t, func() {
			publishResult = p.Publish(context.Background(), release, pubCfg, relCfg, true)
		})
		requirePublisherOK(t, publishResult)
		if !stdlibAssertContains(output, "DRY RUN: AUR Publish") {
			t.Fatalf("expected %v to contain %v", output, "DRY RUN: AUR Publish")
		}
		if !stdlibAssertContains(output, "Package:    myapp-bin") {
			t.Fatalf("expected %v to contain %v", output, "Package:    myapp-bin")
		}
		if !stdlibAssertContains(output, "Version:    2.0.0") {
			t.Fatalf("expected %v to contain %v", output, "Version:    2.0.0")
		}
		if !stdlibAssertContains(output, "Maintainer: Test User <test@example.com>") {
			t.Fatalf("expected %v to contain %v", output, "Maintainer: Test User <test@example.com>")

			// Verify no files created
		}
		if !stdlibAssertContains(output, "Generated PKGBUILD:") {
			t.Fatalf("expected %v to contain %v", output, "Generated PKGBUILD:")

			// --- Chocolatey Publisher Integration Tests ---
		}
		if !stdlibAssertContains(output, "pkgname='myapp-bin'") {
			t.Fatalf("expected %v to contain %v", output, "pkgname='myapp-bin'")
		}
		if !stdlibAssertContains(output, "pkgver='2.0.0'") {
			t.Fatalf("expected %v to contain %v", output, "pkgver='2.0.0'")
		}
		if !stdlibAssertContains(output, "Generated .SRCINFO:") {
			t.Fatalf("expected %v to contain %v", output, "Generated .SRCINFO:")
		}
		if !stdlibAssertContains(output, "pkgbase = myapp-bin") {
			t.Fatalf("expected %v to contain %v", output, "pkgbase = myapp-bin")
		}
		if !stdlibAssertContains(output, "Would push to AUR:") {
			t.Fatalf("expected %v to contain %v", output, "Would push to AUR:")
		}

		entries := requirePublisherDirEntries(t, ax.ReadDir(tmpDir))
		if !stdlibAssertEmpty(entries) {
			t.Fatalf("expected empty, got %v", entries)
		}

	})
}

func TestIntegration_ChocolateyPublisherIntegrationDryRunNoSideEffectsGood(t *testing.T) {
	p := NewChocolateyPublisher()

	t.Run("dry run generates nuspec and install script without side effects", func(t *testing.T) {
		tmpDir := t.TempDir()

		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: tmpDir,
			FS:         storage.Local,
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

		publishResult := core.Ok(nil)
		output := capturePublisherOutput(t, func() {
			publishResult = p.Publish(context.Background(), release, pubCfg, relCfg, true)
		})
		requirePublisherOK(t, publishResult)
		if !stdlibAssertContains(output, "DRY RUN: Chocolatey Publish") {
			t.Fatalf("expected %v to contain %v", output, "DRY RUN: Chocolatey Publish")
		}
		if !stdlibAssertContains(output, "Package:    my-cli-tool") {
			t.Fatalf("expected %v to contain %v", output, "Package:    my-cli-tool")
		}
		if !stdlibAssertContains(output, "Version:    1.0.0") {
			t.Fatalf("expected %v to contain %v", output, "Version:    1.0.0")
		}
		if !stdlibAssertContains(output, "Push:       false") {
			t.Fatalf(

				// Verify no files created
				"expected %v to contain %v", output, "Push:       false")
		}
		if !stdlibAssertContains(output, "Generated package.nuspec:") {
			t.Fatalf(

				// --- npm Publisher Integration Tests ---
				"expected %v to contain %v", output, "Generated package.nuspec:")
		}
		if !stdlibAssertContains(output, "<id>my-cli-tool</id>") {
			t.Fatalf("expected %v to contain %v", output, "<id>my-cli-tool</id>")
		}
		if !stdlibAssertContains(output, "Generated chocolateyinstall.ps1:") {
			t.Fatalf("expected %v to contain %v", output, "Generated chocolateyinstall.ps1:")
		}
		if !stdlibAssertContains(output, "Would generate package files only") {
			t.Fatalf("expected %v to contain %v", output, "Would generate package files only")
		}

		entries := requirePublisherDirEntries(t, ax.ReadDir(tmpDir))
		if !stdlibAssertEmpty(entries) {
			t.Fatalf("expected empty, got %v", entries)
		}

	})
}

func TestIntegration_NpmPublisherIntegrationDryRunNoSideEffectsGood(t *testing.T) {
	p := NewNpmPublisher()

	t.Run("dry run generates package.json without writing files or publishing", func(t *testing.T) {
		tmpDir := t.TempDir()

		release := &Release{
			Version:    "v3.0.0",
			ProjectDir: tmpDir,
			FS:         storage.Local,
		}
		pubCfg := PublisherConfig{
			Type: "npm",
			Extended: map[string]any{
				"package": "@test-org/my-cli",
				"access":  "public",
			},
		}
		relCfg := &mockReleaseConfig{repository: "test-org/my-cli", projectName: "my-cli"}

		publishResult := core.Ok(nil)
		output := capturePublisherOutput(t, func() {
			publishResult = p.Publish(context.Background(), release, pubCfg, relCfg, true)
		})
		requirePublisherOK(t, publishResult)
		if !stdlibAssertContains(output, "DRY RUN: npm Publish") {
			t.Fatalf("expected %v to contain %v", output, "DRY RUN: npm Publish")
		}
		if !stdlibAssertContains(output, "Package:    @test-org/my-cli") {
			t.Fatalf("expected %v to contain %v", output, "Package:    @test-org/my-cli")
		}
		if !stdlibAssertContains(output, "Version:    3.0.0") {
			t.Fatalf("expected %v to contain %v", output, "Version:    3.0.0")
		}
		if !stdlibAssertContains(output, "Access:     public") {
			t.Fatalf(

				// Verify no files created
				"expected %v to contain %v", output, "Access:     public")
		}
		if !stdlibAssertContains(output, "Generated package.json:") {
			t.Fatalf(

				// --- LinuxKit Publisher Integration Tests ---
				"expected %v to contain %v", output, "Generated package.json:")
		}
		if !stdlibAssertContains(output, `"name": "@test-org/my-cli"`) {
			t.Fatalf("expected %v to contain %v", output, `"name": "@test-org/my-cli"`)
		}
		if !stdlibAssertContains(output, `"version": "3.0.0"`) {
			t.Fatalf("expected %v to contain %v", output, `"version": "3.0.0"`)
		}
		if !stdlibAssertContains(output, "Would run: npm publish --access public") {

			// Create config file
			t.Fatalf("expected %v to contain %v", output, "Would run: npm publish --access public")
		}

		entries := requirePublisherDirEntries(t, ax.ReadDir(tmpDir))
		if !stdlibAssertEmpty(entries) {
			t.Fatalf("expected empty, got %v", entries)
		}

	})
}

func TestIntegration_LinuxKitPublisherIntegrationDryRunNoSideEffectsGood(t *testing.T) {
	if err := validateLinuxKitCli(); !err.OK {
		t.Skip("skipping: linuxkit CLI not available")
	}

	p := NewLinuxKitPublisher()

	t.Run("dry run with multiple formats and platforms", func(t *testing.T) {
		tmpDir := t.TempDir()

		configDir := ax.Join(tmpDir, ".core", "linuxkit")
		if result := ax.MkdirAll(configDir, 0o755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(configDir, "server.yml"), []byte("kernel:\n  image: test\n"), 0o644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: tmpDir,
			FS:         storage.Local,
		}
		pubCfg := PublisherConfig{
			Type: "linuxkit",
			Extended: map[string]any{
				"formats":   []any{"iso", "qcow2", "docker"},
				"platforms": []any{"linux/amd64", "linux/arm64"},
			},
		}
		relCfg := &mockReleaseConfig{repository: "test-org/my-os"}

		publishResult := core.Ok(nil)
		output := capturePublisherOutput(t, func() {
			publishResult = p.Publish(context.Background(), release, pubCfg, relCfg, true)
		})
		requirePublisherOK(t, publishResult)
		if !stdlibAssertContains(output, "DRY RUN: LinuxKit Build & Publish") {
			t.Fatalf("expected %v to contain %v", output, "DRY RUN: LinuxKit Build & Publish")
		}
		if !stdlibAssertContains(output, "Formats:       iso, qcow2, docker") {

			// Verify all combinations listed
			t.Fatalf("expected %v to contain %v", output, "Formats:       iso, qcow2, docker")
		}
		if !stdlibAssertContains(output, "Platforms:     linux/amd64, linux/arm64") {
			t.Fatalf("expected %v to contain %v", output, "Platforms:     linux/amd64, linux/arm64")
		}
		if !stdlibAssertContains(output, "linuxkit-1.0.0-amd64.iso") {
			t.Fatalf("expected %v to contain %v", output, "linuxkit-1.0.0-amd64.iso")

			// Verify docker usage hint
		}
		if !stdlibAssertContains(output, "linuxkit-1.0.0-amd64.qcow2") {
			t.Fatalf(

				// Verify no files created in dist
				"expected %v to contain %v", output, "linuxkit-1.0.0-amd64.qcow2")
		}
		if !stdlibAssertContains(output, "linuxkit-1.0.0-amd64.docker.tar") {
			t.Fatalf("expected %v to contain %v",

				// --- Cross-Publisher Integration Tests ---
				output, "linuxkit-1.0.0-amd64.docker.tar")
		}
		if !stdlibAssertContains(output, "linuxkit-1.0.0-arm64.iso") {
			t.Fatalf("expected %v to contain %v", output, "linuxkit-1.0.0-arm64.iso")
		}
		if !stdlibAssertContains(output, "linuxkit-1.0.0-arm64.qcow2") {
			t.Fatalf("expected %v to contain %v", output, "linuxkit-1.0.0-arm64.qcow2")
		}
		if !stdlibAssertContains(output, "linuxkit-1.0.0-arm64.docker.tar") {
			t.Fatalf("expected %v to contain %v", output, "linuxkit-1.0.0-arm64.docker.tar")
		}
		if !stdlibAssertContains(output, "docker load") {
			t.Fatalf("expected %v to contain %v", output, "docker load")
		}

		distDir := ax.Join(tmpDir, "dist")
		if ax.Exists(distDir) {
			t.Fatal("dry run should not create dist directory")
		}

	})
}

func TestIntegration_AllPublishersIntegrationNameUniquenessGood(t *testing.T) {
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
			if names[name] {
				t.Fatalf("duplicate publisher name: %s", name)
			}

			names[name] = true
			if stdlibAssertEmpty(name) {
				t.Fatal("publisher name should not be empty")
			}

		}
		if len(names) != 8 {
			t.Fatal("should have 8 unique publishers")
		}

	})
}

func TestIntegration_AllPublishersIntegrationNilRelCfgGood(t *testing.T) {
	t.Run("github handles nil relCfg with git repo", func(t *testing.T) {
		tmpDir := t.TempDir()

		runPublisherCommand(t, tmpDir, "git", "init")
		runPublisherCommand(t, tmpDir, "git", "remote", "add", "origin", "git@github.com:niltest/repo.git")

		release := &Release{
			Version:    "v1.0.0",
			Changelog:  "Changes",
			ProjectDir: tmpDir,
			FS:         storage.Local,
		}
		pubCfg := PublisherConfig{Type: "github"}

		publishResult := core.Ok(nil)
		output := capturePublisherOutput(t, func() {
			publishResult = NewGitHubPublisher().Publish(context.Background(), release, pubCfg, nil, true)
		})
		requirePublisherOK(t, publishResult)
		if !stdlibAssertContains(output, "niltest/repo") {
			t.Fatalf("expected %v to contain %v", output, "niltest/repo")
		}

	})
}

func TestIntegration_BuildChecksumMapIntegrationGood(t *testing.T) {
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
		if !stdlibAssertEqual("da64", checksums.DarwinAmd64) {
			t.Fatalf("want %v, got %v", "da64", checksums.DarwinAmd64)
		}
		if !stdlibAssertEqual("da65", checksums.DarwinArm64) {
			t.Fatalf("want %v, got %v", "da65", checksums.DarwinArm64)
		}
		if !stdlibAssertEqual("la64", checksums.LinuxAmd64) {
			t.Fatalf("want %v, got %v", "la64", checksums.

				// indexOf returns the index of an element in a string slice, or -1 if not found.
				LinuxAmd64)
		}
		if !stdlibAssertEqual("la65", checksums.LinuxArm64) {
			t.Fatalf("want %v, got %v", "la65", checksums.LinuxArm64)
		}
		if !stdlibAssertEqual("wa64", checksums.WindowsAmd64) {
			t.Fatalf("want %v, got %v", "wa64",

				// Compile-time check: all publishers implement Publisher interface
				checksums.WindowsAmd64)
		}
		if !stdlibAssertEqual("wa65", checksums.WindowsArm64) {
			t.Fatalf("want %v, got %v", "wa65", checksums.WindowsArm64)
		}

	})
}

func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}

var _ Publisher = (*GitHubPublisher)(nil)
var _ Publisher = (*DockerPublisher)(nil)
var _ Publisher = (*HomebrewPublisher)(nil)
var _ Publisher = (*NpmPublisher)(nil)
var _ Publisher = (*ScoopPublisher)(nil)
var _ Publisher = (*AURPublisher)(nil)
var _ Publisher = (*ChocolateyPublisher)(nil)
var _ Publisher = (*LinuxKitPublisher)(nil)
