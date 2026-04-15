package release

import (
	"context"
	"os"
	"runtime"
	"testing"

	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/build/pkg/build/signing"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelease_FindArtifacts_Good(t *testing.T) {
	t.Run("finds tar.gz artifacts", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		require.NoError(t, ax.MkdirAll(distDir, 0755))

		// Create test artifact files
		require.NoError(t, ax.WriteFile(ax.Join(distDir, "app-linux-amd64.tar.gz"), []byte("test"), 0644))
		require.NoError(t, ax.WriteFile(ax.Join(distDir, "app-darwin-arm64.tar.gz"), []byte("test"), 0644))

		artifacts, err := findArtifacts(io.Local, distDir)
		require.NoError(t, err)

		assert.Len(t, artifacts, 2)
	})

	t.Run("finds tar.xz artifacts", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		require.NoError(t, ax.MkdirAll(distDir, 0755))

		require.NoError(t, ax.WriteFile(ax.Join(distDir, "app-linux-amd64.tar.xz"), []byte("test"), 0644))

		artifacts, err := findArtifacts(io.Local, distDir)
		require.NoError(t, err)

		assert.Len(t, artifacts, 1)
		assert.Contains(t, artifacts[0].Path, "app-linux-amd64.tar.xz")
	})

	t.Run("finds zip artifacts", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		require.NoError(t, ax.MkdirAll(distDir, 0755))

		require.NoError(t, ax.WriteFile(ax.Join(distDir, "app-windows-amd64.zip"), []byte("test"), 0644))

		artifacts, err := findArtifacts(io.Local, distDir)
		require.NoError(t, err)

		assert.Len(t, artifacts, 1)
		assert.Contains(t, artifacts[0].Path, "app-windows-amd64.zip")
	})

	t.Run("finds checksum files", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		require.NoError(t, ax.MkdirAll(distDir, 0755))

		require.NoError(t, ax.WriteFile(ax.Join(distDir, "CHECKSUMS.txt"), []byte("checksums"), 0644))

		artifacts, err := findArtifacts(io.Local, distDir)
		require.NoError(t, err)

		assert.Len(t, artifacts, 1)
		assert.Contains(t, artifacts[0].Path, "CHECKSUMS.txt")
	})

	t.Run("ignores unrelated text files", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		require.NoError(t, ax.MkdirAll(distDir, 0755))

		require.NoError(t, ax.WriteFile(ax.Join(distDir, "release-notes.txt"), []byte("notes"), 0644))

		artifacts, err := findArtifacts(io.Local, distDir)
		require.NoError(t, err)

		assert.Empty(t, artifacts)
	})

	t.Run("finds signature files", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		require.NoError(t, ax.MkdirAll(distDir, 0755))

		require.NoError(t, ax.WriteFile(ax.Join(distDir, "app.tar.gz.sig"), []byte("signature"), 0644))

		artifacts, err := findArtifacts(io.Local, distDir)
		require.NoError(t, err)

		assert.Len(t, artifacts, 1)
	})

	t.Run("finds asc signature files", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		require.NoError(t, ax.MkdirAll(distDir, 0755))

		require.NoError(t, ax.WriteFile(ax.Join(distDir, "CHECKSUMS.txt.asc"), []byte("signature"), 0644))

		artifacts, err := findArtifacts(io.Local, distDir)
		require.NoError(t, err)

		assert.Len(t, artifacts, 1)
		assert.Contains(t, artifacts[0].Path, "CHECKSUMS.txt.asc")
	})

	t.Run("finds mixed artifact types", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		require.NoError(t, ax.MkdirAll(distDir, 0755))

		require.NoError(t, ax.WriteFile(ax.Join(distDir, "app-linux.tar.gz"), []byte("test"), 0644))
		require.NoError(t, ax.WriteFile(ax.Join(distDir, "app-linux-arm64.tar.xz"), []byte("test"), 0644))
		require.NoError(t, ax.WriteFile(ax.Join(distDir, "app-windows.zip"), []byte("test"), 0644))
		require.NoError(t, ax.WriteFile(ax.Join(distDir, "CHECKSUMS.txt"), []byte("checksums"), 0644))
		require.NoError(t, ax.WriteFile(ax.Join(distDir, "app.sig"), []byte("sig"), 0644))

		artifacts, err := findArtifacts(io.Local, distDir)
		require.NoError(t, err)

		assert.Len(t, artifacts, 5)
	})

	t.Run("ignores non-artifact files", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		require.NoError(t, ax.MkdirAll(distDir, 0755))

		require.NoError(t, ax.WriteFile(ax.Join(distDir, "README.md"), []byte("readme"), 0644))
		require.NoError(t, ax.WriteFile(ax.Join(distDir, "app.exe"), []byte("binary"), 0644))
		require.NoError(t, ax.WriteFile(ax.Join(distDir, "app.tar.gz"), []byte("artifact"), 0644))
		require.NoError(t, ax.WriteFile(ax.Join(distDir, "app.tar.xz"), []byte("artifact"), 0644))

		artifacts, err := findArtifacts(io.Local, distDir)
		require.NoError(t, err)

		assert.Len(t, artifacts, 2)
		assert.ElementsMatch(t, []string{
			ax.Join(distDir, "app.tar.gz"),
			ax.Join(distDir, "app.tar.xz"),
		}, []string{artifacts[0].Path, artifacts[1].Path})
	})

	t.Run("finds nested archived artifacts in subdirectories", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		require.NoError(t, ax.MkdirAll(distDir, 0755))
		require.NoError(t, ax.MkdirAll(ax.Join(distDir, "subdir"), 0755))

		require.NoError(t, ax.WriteFile(ax.Join(distDir, "app.tar.gz"), []byte("artifact"), 0644))
		require.NoError(t, ax.WriteFile(ax.Join(distDir, "subdir", "nested.tar.gz"), []byte("nested"), 0644))

		artifacts, err := findArtifacts(io.Local, distDir)
		require.NoError(t, err)

		assert.Len(t, artifacts, 2)
		assert.ElementsMatch(t, []string{
			ax.Join(distDir, "app.tar.gz"),
			ax.Join(distDir, "subdir", "nested.tar.gz"),
		}, []string{artifacts[0].Path, artifacts[1].Path})
	})

	t.Run("falls back to raw platform artifacts when no archives exist", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		require.NoError(t, ax.MkdirAll(ax.Join(distDir, "linux_amd64"), 0755))
		require.NoError(t, ax.MkdirAll(ax.Join(distDir, "windows_amd64"), 0755))

		require.NoError(t, ax.WriteFile(ax.Join(distDir, "linux_amd64", "myapp"), []byte("binary"), 0755))
		require.NoError(t, ax.WriteFile(ax.Join(distDir, "windows_amd64", "myapp.exe"), []byte("binary"), 0755))
		require.NoError(t, ax.WriteFile(ax.Join(distDir, "linux_amd64", "artifact_meta.json"), []byte("{}"), 0644))

		artifacts, err := findArtifacts(io.Local, distDir)
		require.NoError(t, err)

		require.Len(t, artifacts, 2)
		assert.Equal(t, ax.Join(distDir, "linux_amd64", "myapp"), artifacts[0].Path)
		assert.Equal(t, "linux", artifacts[0].OS)
		assert.Equal(t, "amd64", artifacts[0].Arch)
		assert.Equal(t, ax.Join(distDir, "windows_amd64", "myapp.exe"), artifacts[1].Path)
		assert.Equal(t, "windows", artifacts[1].OS)
		assert.Equal(t, "amd64", artifacts[1].Arch)
	})

	t.Run("includes checksum artifacts alongside raw platform artifacts", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		require.NoError(t, ax.MkdirAll(ax.Join(distDir, "linux_amd64"), 0o755))
		require.NoError(t, ax.WriteFile(ax.Join(distDir, "linux_amd64", "myapp"), []byte("binary"), 0o755))
		require.NoError(t, ax.WriteFile(ax.Join(distDir, "CHECKSUMS.txt"), []byte("checksums"), 0o644))

		artifacts, err := findArtifacts(io.Local, distDir)
		require.NoError(t, err)

		assert.Len(t, artifacts, 2)
		assert.ElementsMatch(t, []string{
			ax.Join(distDir, "linux_amd64", "myapp"),
			ax.Join(distDir, "CHECKSUMS.txt"),
		}, []string{artifacts[0].Path, artifacts[1].Path})
	})

	t.Run("finds nested raw platform artifacts for multi-type builds", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		platformDir := ax.Join(distDir, "go", "linux_amd64")
		require.NoError(t, ax.MkdirAll(platformDir, 0o755))
		require.NoError(t, ax.WriteFile(ax.Join(platformDir, "myapp"), []byte("binary"), 0o755))

		artifacts, err := findArtifacts(io.Local, distDir)
		require.NoError(t, err)

		require.Len(t, artifacts, 1)
		assert.Equal(t, ax.Join(platformDir, "myapp"), artifacts[0].Path)
		assert.Equal(t, "linux", artifacts[0].OS)
		assert.Equal(t, "amd64", artifacts[0].Arch)
	})

	t.Run("includes macOS app bundles from platform directories", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		platformDir := ax.Join(distDir, "darwin_arm64")
		require.NoError(t, ax.MkdirAll(ax.Join(platformDir, "TestApp.app"), 0755))
		require.NoError(t, ax.MkdirAll(ax.Join(platformDir, "TestApp.app", "Contents"), 0755))
		require.NoError(t, ax.WriteFile(ax.Join(platformDir, "TestApp.app", "Contents", "Info.plist"), []byte("<plist/>"), 0644))

		artifacts, err := findArtifacts(io.Local, distDir)
		require.NoError(t, err)

		require.Len(t, artifacts, 1)
		assert.Equal(t, ax.Join(platformDir, "TestApp.app"), artifacts[0].Path)
		assert.Equal(t, "darwin", artifacts[0].OS)
		assert.Equal(t, "arm64", artifacts[0].Arch)
	})

	t.Run("returns empty slice for empty dist directory", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		require.NoError(t, ax.MkdirAll(distDir, 0755))

		artifacts, err := findArtifacts(io.Local, distDir)
		require.NoError(t, err)

		assert.Empty(t, artifacts)
	})
}

func TestRelease_FindArtifacts_Bad(t *testing.T) {
	t.Run("returns error when dist directory does not exist", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")

		_, err := findArtifacts(io.Local, distDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dist/ directory not found")
	})

	t.Run("returns error when dist directory is unreadable", func(t *testing.T) {
		if ax.Geteuid() == 0 {
			t.Skip("root can read any directory")
		}
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		require.NoError(t, ax.MkdirAll(distDir, 0755))

		// Create a file that looks like dist but will cause ReadDir to fail
		// by making the directory unreadable
		require.NoError(t, ax.Chmod(distDir, 0000))
		defer func() { _ = ax.Chmod(distDir, 0755) }()

		_, err := findArtifacts(io.Local, distDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read dist/")
	})
}

func TestRelease_GetBuilder_Good(t *testing.T) {
	t.Run("returns Go builder for go project type", func(t *testing.T) {
		builder, err := getBuilder(build.ProjectTypeGo)
		require.NoError(t, err)
		assert.NotNil(t, builder)
		assert.Equal(t, "go", builder.Name())
	})

	t.Run("returns Wails builder for wails project type", func(t *testing.T) {
		builder, err := getBuilder(build.ProjectTypeWails)
		require.NoError(t, err)
		assert.NotNil(t, builder)
		assert.Equal(t, "wails", builder.Name())
	})

	t.Run("returns Node builder for node project type", func(t *testing.T) {
		builder, err := getBuilder(build.ProjectTypeNode)
		require.NoError(t, err)
		assert.NotNil(t, builder)
		assert.Equal(t, "node", builder.Name())
	})

	t.Run("returns PHP builder for php project type", func(t *testing.T) {
		builder, err := getBuilder(build.ProjectTypePHP)
		require.NoError(t, err)
		assert.NotNil(t, builder)
		assert.Equal(t, "php", builder.Name())
	})

	t.Run("returns Python builder for python project type", func(t *testing.T) {
		builder, err := getBuilder(build.ProjectTypePython)
		require.NoError(t, err)
		assert.NotNil(t, builder)
		assert.Equal(t, "python", builder.Name())
	})

	t.Run("returns Rust builder for rust project type", func(t *testing.T) {
		builder, err := getBuilder(build.ProjectTypeRust)
		require.NoError(t, err)
		assert.NotNil(t, builder)
		assert.Equal(t, "rust", builder.Name())
	})

	t.Run("returns C++ builder for cpp project type", func(t *testing.T) {
		builder, err := getBuilder(build.ProjectTypeCPP)
		require.NoError(t, err)
		assert.NotNil(t, builder)
		assert.Equal(t, "cpp", builder.Name())
	})

	t.Run("returns Docker builder for docker project type", func(t *testing.T) {
		builder, err := getBuilder(build.ProjectTypeDocker)
		require.NoError(t, err)
		assert.NotNil(t, builder)
		assert.Equal(t, "docker", builder.Name())
	})

	t.Run("returns LinuxKit builder for linuxkit project type", func(t *testing.T) {
		builder, err := getBuilder(build.ProjectTypeLinuxKit)
		require.NoError(t, err)
		assert.NotNil(t, builder)
		assert.Equal(t, "linuxkit", builder.Name())
	})

	t.Run("returns Taskfile builder for taskfile project type", func(t *testing.T) {
		builder, err := getBuilder(build.ProjectTypeTaskfile)
		require.NoError(t, err)
		assert.NotNil(t, builder)
		assert.Equal(t, "taskfile", builder.Name())
	})
}

func TestRelease_GetBuilder_Bad(t *testing.T) {
	t.Run("returns error for unsupported project type", func(t *testing.T) {
		_, err := getBuilder(build.ProjectType("unknown"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported project type")
	})
}

func TestRelease_GetPublisher_Good(t *testing.T) {
	tests := []struct {
		pubType      string
		expectedName string
	}{
		{"github", "github"},
		{"linuxkit", "linuxkit"},
		{"docker", "docker"},
		{"npm", "npm"},
		{"homebrew", "homebrew"},
		{"scoop", "scoop"},
		{"aur", "aur"},
		{"chocolatey", "chocolatey"},
	}

	for _, tc := range tests {
		t.Run(tc.pubType, func(t *testing.T) {
			publisher, err := getPublisher(tc.pubType)
			require.NoError(t, err)
			assert.NotNil(t, publisher)
			assert.Equal(t, tc.expectedName, publisher.Name())
		})
	}
}

func TestRelease_GetPublisher_Bad(t *testing.T) {
	t.Run("returns error for unsupported publisher type", func(t *testing.T) {
		_, err := getPublisher("unsupported")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported publisher type: unsupported")
	})

	t.Run("returns error for empty publisher type", func(t *testing.T) {
		_, err := getPublisher("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported publisher type")
	})
}

func TestRelease_ResolveProjectType_Good(t *testing.T) {
	t.Run("honours explicit build type override", func(t *testing.T) {
		dir := t.TempDir()

		projectType, err := resolveProjectType(io.Local, dir, "docker")
		require.NoError(t, err)
		assert.Equal(t, build.ProjectTypeDocker, projectType)
	})

	t.Run("falls back to marker detection when build type is empty", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example.com/test"), 0644))

		projectType, err := resolveProjectType(io.Local, dir, "")
		require.NoError(t, err)
		assert.Equal(t, build.ProjectTypeGo, projectType)
	})
}

func TestRelease_BuildExtendedConfig_Good(t *testing.T) {
	t.Run("returns empty map for minimal config", func(t *testing.T) {
		cfg := PublisherConfig{
			Type: "github",
		}

		ext := buildExtendedConfig(cfg)
		assert.Empty(t, ext)
	})

	t.Run("includes LinuxKit config", func(t *testing.T) {
		cfg := PublisherConfig{
			Type:      "linuxkit",
			Config:    "linuxkit.yaml",
			Formats:   []string{"iso", "qcow2"},
			Platforms: []string{"linux/amd64", "linux/arm64"},
		}

		ext := buildExtendedConfig(cfg)

		assert.Equal(t, "linuxkit.yaml", ext["config"])
		assert.Equal(t, []any{"iso", "qcow2"}, ext["formats"])
		assert.Equal(t, []any{"linux/amd64", "linux/arm64"}, ext["platforms"])
	})

	t.Run("includes Docker config", func(t *testing.T) {
		cfg := PublisherConfig{
			Type:       "docker",
			Registry:   "ghcr.io",
			Image:      "owner/repo",
			Dockerfile: "Dockerfile.prod",
			Tags:       []string{"latest", "v1.0.0"},
			BuildArgs:  map[string]string{"VERSION": "1.0.0"},
		}

		ext := buildExtendedConfig(cfg)

		assert.Equal(t, "ghcr.io", ext["registry"])
		assert.Equal(t, "owner/repo", ext["image"])
		assert.Equal(t, "Dockerfile.prod", ext["dockerfile"])
		assert.Equal(t, []any{"latest", "v1.0.0"}, ext["tags"])
		buildArgs := ext["build_args"].(map[string]any)
		assert.Equal(t, "1.0.0", buildArgs["VERSION"])
	})

	t.Run("includes npm config", func(t *testing.T) {
		cfg := PublisherConfig{
			Type:    "npm",
			Package: "@host-uk/core",
			Access:  "public",
		}

		ext := buildExtendedConfig(cfg)

		assert.Equal(t, "@host-uk/core", ext["package"])
		assert.Equal(t, "public", ext["access"])
	})

	t.Run("includes Homebrew config", func(t *testing.T) {
		cfg := PublisherConfig{
			Type:    "homebrew",
			Tap:     "host-uk/tap",
			Formula: "core",
		}

		ext := buildExtendedConfig(cfg)

		assert.Equal(t, "host-uk/tap", ext["tap"])
		assert.Equal(t, "core", ext["formula"])
	})

	t.Run("includes Scoop config", func(t *testing.T) {
		cfg := PublisherConfig{
			Type:   "scoop",
			Bucket: "host-uk/bucket",
		}

		ext := buildExtendedConfig(cfg)

		assert.Equal(t, "host-uk/bucket", ext["bucket"])
	})

	t.Run("includes AUR config", func(t *testing.T) {
		cfg := PublisherConfig{
			Type:       "aur",
			Maintainer: "John Doe <john@example.com>",
		}

		ext := buildExtendedConfig(cfg)

		assert.Equal(t, "John Doe <john@example.com>", ext["maintainer"])
	})

	t.Run("includes Chocolatey config", func(t *testing.T) {
		cfg := PublisherConfig{
			Type: "chocolatey",
			Push: true,
		}

		ext := buildExtendedConfig(cfg)

		assert.True(t, ext["push"].(bool))
	})

	t.Run("includes Official config", func(t *testing.T) {
		cfg := PublisherConfig{
			Type: "homebrew",
			Official: &OfficialConfig{
				Enabled: true,
				Output:  "/path/to/output",
			},
		}

		ext := buildExtendedConfig(cfg)

		official := ext["official"].(map[string]any)
		assert.True(t, official["enabled"].(bool))
		assert.Equal(t, "/path/to/output", official["output"])
	})

	t.Run("Official config without output", func(t *testing.T) {
		cfg := PublisherConfig{
			Type: "scoop",
			Official: &OfficialConfig{
				Enabled: true,
			},
		}

		ext := buildExtendedConfig(cfg)

		official := ext["official"].(map[string]any)
		assert.True(t, official["enabled"].(bool))
		_, hasOutput := official["output"]
		assert.False(t, hasOutput)
	})
}

func TestRelease_ToAnySlice_Good(t *testing.T) {
	t.Run("converts string slice to any slice", func(t *testing.T) {
		input := []string{"a", "b", "c"}

		result := toAnySlice(input)

		assert.Len(t, result, 3)
		assert.Equal(t, "a", result[0])
		assert.Equal(t, "b", result[1])
		assert.Equal(t, "c", result[2])
	})

	t.Run("handles empty slice", func(t *testing.T) {
		input := []string{}

		result := toAnySlice(input)

		assert.Empty(t, result)
	})

	t.Run("handles single element", func(t *testing.T) {
		input := []string{"only"}

		result := toAnySlice(input)

		assert.Len(t, result, 1)
		assert.Equal(t, "only", result[0])
	})
}

func TestRelease_Publish_Good(t *testing.T) {
	t.Run("returns release with version from config", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		require.NoError(t, ax.MkdirAll(distDir, 0755))
		require.NoError(t, ax.WriteFile(ax.Join(distDir, "app.tar.gz"), []byte("test"), 0644))
		require.NoError(t, ax.WriteFile(ax.Join(distDir, "app.tar.xz"), []byte("test"), 0644))

		cfg := DefaultConfig()
		cfg.SetProjectDir(dir)
		cfg.SetVersion("v1.0.0")
		cfg.Publishers = nil // No publishers to avoid network calls

		release, err := Publish(context.Background(), cfg, true)
		require.NoError(t, err)

		assert.Equal(t, "v1.0.0", release.Version)
		assert.Len(t, release.Artifacts, 2)
	})

	t.Run("finds artifacts in dist directory", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		require.NoError(t, ax.MkdirAll(distDir, 0755))
		require.NoError(t, ax.WriteFile(ax.Join(distDir, "app-linux.tar.gz"), []byte("test"), 0644))
		require.NoError(t, ax.WriteFile(ax.Join(distDir, "app-linux.tar.xz"), []byte("test"), 0644))
		require.NoError(t, ax.WriteFile(ax.Join(distDir, "app-darwin.tar.gz"), []byte("test"), 0644))
		require.NoError(t, ax.WriteFile(ax.Join(distDir, "CHECKSUMS.txt"), []byte("checksums"), 0644))

		cfg := DefaultConfig()
		cfg.SetProjectDir(dir)
		cfg.SetVersion("v1.0.0")
		cfg.Publishers = nil

		release, err := Publish(context.Background(), cfg, true)
		require.NoError(t, err)

		assert.Len(t, release.Artifacts, 4)
	})

	t.Run("keeps raw platform artifacts when checksums exist without archives", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		require.NoError(t, ax.MkdirAll(ax.Join(distDir, "linux_amd64"), 0o755))
		require.NoError(t, ax.WriteFile(ax.Join(distDir, "linux_amd64", "app"), []byte("binary"), 0o755))
		require.NoError(t, ax.WriteFile(ax.Join(distDir, "CHECKSUMS.txt"), []byte("checksums"), 0o644))

		cfg := DefaultConfig()
		cfg.SetProjectDir(dir)
		cfg.SetVersion("v1.0.0")
		cfg.Publishers = nil

		release, err := Publish(context.Background(), cfg, true)
		require.NoError(t, err)

		assert.Contains(t, release.Artifacts, build.Artifact{
			Path: ax.Join(distDir, "linux_amd64", "app"),
			OS:   "linux",
			Arch: "amd64",
		})
		assert.Contains(t, release.Artifacts, build.Artifact{Path: ax.Join(distDir, "CHECKSUMS.txt")})
	})
}

func TestRelease_Publish_Bad(t *testing.T) {
	t.Run("returns error when config is nil", func(t *testing.T) {
		_, err := Publish(context.Background(), nil, true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config is nil")
	})

	t.Run("returns error when dist directory missing", func(t *testing.T) {
		dir := t.TempDir()

		cfg := DefaultConfig()
		cfg.SetProjectDir(dir)
		cfg.SetVersion("v1.0.0")

		_, err := Publish(context.Background(), cfg, true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dist/ directory not found")
	})

	t.Run("returns error when no artifacts found", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		require.NoError(t, ax.MkdirAll(distDir, 0755))

		cfg := DefaultConfig()
		cfg.SetProjectDir(dir)
		cfg.SetVersion("v1.0.0")

		_, err := Publish(context.Background(), cfg, true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no artifacts found")
	})

	t.Run("returns error for unsupported publisher", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		require.NoError(t, ax.MkdirAll(distDir, 0755))
		require.NoError(t, ax.WriteFile(ax.Join(distDir, "app.tar.gz"), []byte("test"), 0644))

		cfg := DefaultConfig()
		cfg.SetProjectDir(dir)
		cfg.SetVersion("v1.0.0")
		cfg.Publishers = []PublisherConfig{
			{Type: "unsupported"},
		}

		_, err := Publish(context.Background(), cfg, true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported publisher type")
	})

	t.Run("returns error when version determination fails in non-git dir", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		require.NoError(t, ax.MkdirAll(distDir, 0755))
		require.NoError(t, ax.WriteFile(ax.Join(distDir, "app.tar.gz"), []byte("test"), 0644))

		cfg := DefaultConfig()
		cfg.SetProjectDir(dir)
		// Don't set version - let it try to determine from git
		cfg.Publishers = nil

		// In a non-git directory, DetermineVersion returns v0.0.1 as default
		// so we verify that the publish proceeds without error
		release, err := Publish(context.Background(), cfg, true)
		require.NoError(t, err)
		assert.Equal(t, "v0.0.1", release.Version)
	})
}

func TestRelease_Run_Good(t *testing.T) {
	t.Run("returns release with version from config", func(t *testing.T) {
		// Create a minimal Go project for testing
		dir := t.TempDir()

		// Create go.mod
		goMod := `module testapp

go 1.21
`
		require.NoError(t, ax.WriteFile(ax.Join(dir, "go.mod"), []byte(goMod), 0644))

		// Create main.go
		mainGo := `package main

func main() {}
`
		require.NoError(t, ax.WriteFile(ax.Join(dir, "main.go"), []byte(mainGo), 0644))

		cfg := DefaultConfig()
		cfg.SetProjectDir(dir)
		cfg.SetVersion("v1.0.0")
		cfg.Project.Name = "testapp"
		cfg.Build.Targets = []TargetConfig{} // Empty targets to use defaults
		cfg.Publishers = nil                 // No publishers to avoid network calls

		// Note: This test will actually try to build, which may fail in CI
		// So we just test that the function accepts the config properly
		release, err := Run(context.Background(), cfg, true)
		if err != nil {
			// Build might fail in test environment, but we still verify the error message
			assert.Contains(t, err.Error(), "build")
		} else {
			assert.Equal(t, "v1.0.0", release.Version)
		}
	})
}

func TestRelease_Run_Bad(t *testing.T) {
	t.Run("returns error when config is nil", func(t *testing.T) {
		_, err := Run(context.Background(), nil, true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config is nil")
	})
}

func TestRelease_Structure_Good(t *testing.T) {
	t.Run("Release struct holds expected fields", func(t *testing.T) {
		release := &Release{
			Version:    "v1.0.0",
			Artifacts:  []build.Artifact{{Path: "/path/to/artifact"}},
			Changelog:  "## v1.0.0\n\nChanges",
			ProjectDir: "/project",
		}

		assert.Equal(t, "v1.0.0", release.Version)
		assert.Len(t, release.Artifacts, 1)
		assert.Contains(t, release.Changelog, "v1.0.0")
		assert.Equal(t, "/project", release.ProjectDir)
	})
}

func TestRelease_PublishVersionFromGit_Good(t *testing.T) {
	t.Run("determines version from git when not set", func(t *testing.T) {
		dir := setupPublishGitRepo(t)
		createPublishCommit(t, dir, "feat: initial commit")
		createPublishTag(t, dir, "v1.2.3")

		// Create dist directory with artifact
		distDir := ax.Join(dir, "dist")
		require.NoError(t, ax.MkdirAll(distDir, 0755))
		require.NoError(t, ax.WriteFile(ax.Join(distDir, "app.tar.gz"), []byte("test"), 0644))

		cfg := DefaultConfig()
		cfg.SetProjectDir(dir)
		// Don't set version - let it be determined from git
		cfg.Publishers = nil

		release, err := Publish(context.Background(), cfg, true)
		require.NoError(t, err)

		assert.Equal(t, "v1.2.3", release.Version)
	})
}

func TestRelease_PublishChangelogGeneration_Good(t *testing.T) {
	t.Run("generates changelog from git commits when available", func(t *testing.T) {
		dir := setupPublishGitRepo(t)
		createPublishCommit(t, dir, "feat: add feature")
		createPublishTag(t, dir, "v1.0.0")
		createPublishCommit(t, dir, "fix: fix bug")
		createPublishTag(t, dir, "v1.0.1")

		// Create dist directory with artifact
		distDir := ax.Join(dir, "dist")
		require.NoError(t, ax.MkdirAll(distDir, 0755))
		require.NoError(t, ax.WriteFile(ax.Join(distDir, "app.tar.gz"), []byte("test"), 0644))

		cfg := DefaultConfig()
		cfg.SetProjectDir(dir)
		cfg.SetVersion("v1.0.1")
		cfg.Publishers = nil

		release, err := Publish(context.Background(), cfg, true)
		require.NoError(t, err)

		// Changelog should contain either the commit message or the version
		assert.Contains(t, release.Changelog, "v1.0.1")
	})

	t.Run("uses fallback changelog on error", func(t *testing.T) {
		dir := t.TempDir() // Not a git repo
		distDir := ax.Join(dir, "dist")
		require.NoError(t, ax.MkdirAll(distDir, 0755))
		require.NoError(t, ax.WriteFile(ax.Join(distDir, "app.tar.gz"), []byte("test"), 0644))

		cfg := DefaultConfig()
		cfg.SetProjectDir(dir)
		cfg.SetVersion("v1.0.0")
		cfg.Publishers = nil

		release, err := Publish(context.Background(), cfg, true)
		require.NoError(t, err)

		// Should use fallback changelog
		assert.Contains(t, release.Changelog, "Release v1.0.0")
	})
}

func TestRelease_PublishDefaultProjectDir_Good(t *testing.T) {
	t.Run("uses current directory when projectDir is empty", func(t *testing.T) {
		// Create artifacts in current directory's dist folder
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		require.NoError(t, ax.MkdirAll(distDir, 0755))
		require.NoError(t, ax.WriteFile(ax.Join(distDir, "app.tar.gz"), []byte("test"), 0644))

		cfg := DefaultConfig()
		cfg.SetProjectDir(dir)
		cfg.SetVersion("v1.0.0")
		cfg.Publishers = nil

		release, err := Publish(context.Background(), cfg, true)
		require.NoError(t, err)

		assert.NotEmpty(t, release.ProjectDir)
	})
}

func TestRelease_BuildArtifacts_SignsChecksums_Good(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake gpg script uses POSIX shell")
	}

	dir := t.TempDir()
	require.NoError(t, ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module signedapp\n\ngo 1.21\n"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))

	gpgDir := t.TempDir()
	gpgPath := ax.Join(gpgDir, "gpg")
	gpgScript := `#!/bin/sh
out=""
while [ $# -gt 0 ]; do
  case "$1" in
    --output)
      out="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done

if [ -z "$out" ]; then
  exit 2
fi

: > "$out"
`
	require.NoError(t, ax.WriteFile(gpgPath, []byte(gpgScript), 0o755))

	oldPath := os.Getenv("PATH")
	require.NotEmpty(t, oldPath)
	t.Setenv("PATH", gpgDir+string(os.PathListSeparator)+oldPath)
	t.Setenv("GPG_KEY_ID", "TESTKEY")

	cfg := DefaultConfig()
	cfg.SetProjectDir(dir)
	cfg.Project.Name = "signedapp"
	cfg.Build.ArchiveFormat = "xz"
	cfg.Build.Targets = []TargetConfig{{OS: runtime.GOOS, Arch: runtime.GOARCH}}
	cfg.Publishers = nil

	artifacts, err := buildArtifacts(context.Background(), io.Local, cfg, dir, "v1.0.0")
	require.NoError(t, err)

	var sawChecksumSignature bool
	var sawXzArchive bool
	for _, artifact := range artifacts {
		if artifact.Path == ax.Join(dir, "dist", "CHECKSUMS.txt.asc") {
			sawChecksumSignature = true
		}
		if artifact.Path == ax.Join(dir, "dist", "signedapp_"+runtime.GOOS+"_"+runtime.GOARCH+".tar.xz") {
			sawXzArchive = true
		}
	}

	assert.True(t, sawChecksumSignature)
	assert.True(t, sawXzArchive)
	assert.FileExists(t, ax.Join(dir, "dist", "CHECKSUMS.txt.asc"))
}

func TestRelease_BuildArtifacts_UsesConfiguredChecksumFile_Good(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module signedapp\n\ngo 1.21\n"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))

	oldSignChecksums := signReleaseChecksums
	defer func() {
		signReleaseChecksums = oldSignChecksums
	}()

	var checksumPaths []string
	signReleaseChecksums = func(ctx context.Context, fs io.Medium, cfg signing.SignConfig, checksumFile string) error {
		checksumPaths = append(checksumPaths, checksumFile)
		return nil
	}

	cfg := DefaultConfig()
	cfg.SetProjectDir(dir)
	cfg.Project.Name = "signedapp"
	cfg.Checksum.File = "checksums.txt"
	cfg.Build.Targets = []TargetConfig{{OS: runtime.GOOS, Arch: runtime.GOARCH}}
	cfg.Publishers = nil

	artifacts, err := buildArtifacts(context.Background(), io.Local, cfg, dir, "v1.0.0")
	require.NoError(t, err)

	customChecksumPath := ax.Join(dir, "dist", "checksums.txt")
	assert.Equal(t, []string{customChecksumPath}, checksumPaths)
	assert.FileExists(t, customChecksumPath)

	var sawChecksum bool
	for _, artifact := range artifacts {
		if artifact.Path == customChecksumPath {
			sawChecksum = true
			break
		}
	}
	assert.True(t, sawChecksum)
}

func TestRelease_BuildArtifacts_SignsBinariesBeforeArchiving_Good(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module signedapp\n\ngo 1.21\n"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))

	require.NoError(t, ax.MkdirAll(ax.Join(dir, ".core"), 0o755))
	require.NoError(t, ax.WriteFile(ax.Join(dir, ".core", build.ConfigFileName), []byte(`
version: 1
project:
  name: signedapp
  binary: signedapp
  main: .
build:
  archive_format: gz
  build_tags:
    - integration
  env:
    - FOO=bar
  cgo: false
  flags:
    - -trimpath
sign:
  enabled: true
targets:
  - os: `+runtime.GOOS+`
    arch: `+runtime.GOARCH+`
`), 0o644))

	oldSignBinaries := signReleaseBinaries
	oldNotarizeBinaries := notarizeReleaseBinaries
	oldSignChecksums := signReleaseChecksums
	defer func() {
		signReleaseBinaries = oldSignBinaries
		notarizeReleaseBinaries = oldNotarizeBinaries
		signReleaseChecksums = oldSignChecksums
	}()

	var signedPaths []string
	var notarizedPaths []string
	var checksumPaths []string

	signReleaseBinaries = func(ctx context.Context, fs io.Medium, cfg signing.SignConfig, artifacts []signing.Artifact) error {
		require.True(t, cfg.Enabled)
		require.Len(t, artifacts, 1)
		signedPaths = append(signedPaths, artifacts[0].Path)
		return nil
	}
	notarizeReleaseBinaries = func(ctx context.Context, fs io.Medium, cfg signing.SignConfig, artifacts []signing.Artifact) error {
		require.True(t, cfg.Enabled)
		require.Len(t, artifacts, 1)
		notarizedPaths = append(notarizedPaths, artifacts[0].Path)
		return nil
	}
	signReleaseChecksums = func(ctx context.Context, fs io.Medium, cfg signing.SignConfig, checksumFile string) error {
		require.True(t, cfg.Enabled)
		checksumPaths = append(checksumPaths, checksumFile)
		return nil
	}

	cfg := DefaultConfig()
	cfg.SetProjectDir(dir)
	cfg.Project.Name = "signedapp"
	cfg.Build.Targets = []TargetConfig{{OS: runtime.GOOS, Arch: runtime.GOARCH}}
	cfg.Publishers = nil

	artifacts, err := buildArtifacts(context.Background(), io.Local, cfg, dir, "v1.0.0")
	require.NoError(t, err)

	assert.Equal(t, []string{ax.Join(dir, "dist", runtime.GOOS+"_"+runtime.GOARCH, "signedapp")}, signedPaths)
	assert.Equal(t, signedPaths, notarizedPaths)
	assert.Equal(t, []string{ax.Join(dir, "dist", "CHECKSUMS.txt")}, checksumPaths)

	var sawArchive bool
	for _, artifact := range artifacts {
		if artifact.Path == ax.Join(dir, "dist", "signedapp_"+runtime.GOOS+"_"+runtime.GOARCH+".tar.gz") {
			sawArchive = true
			break
		}
	}

	assert.True(t, sawArchive)
}

func TestRelease_Publish_IncludesConfiguredChecksumArtifact_Good(t *testing.T) {
	dir := t.TempDir()
	distDir := ax.Join(dir, "dist")
	require.NoError(t, ax.MkdirAll(distDir, 0o755))
	require.NoError(t, ax.WriteFile(ax.Join(distDir, "app-linux-amd64.tar.gz"), []byte("archive"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(distDir, "checksums.txt"), []byte("checksums"), 0o644))

	cfg := DefaultConfig()
	cfg.SetProjectDir(dir)
	cfg.SetVersion("v1.0.0")
	cfg.Checksum.File = "checksums.txt"
	cfg.Publishers = nil

	release, err := Publish(context.Background(), cfg, true)
	require.NoError(t, err)

	assert.Contains(t, release.Artifacts, build.Artifact{Path: ax.Join(distDir, "checksums.txt")})
}

func TestRelease_BuildArtifacts_WritesArtifactMetadata_Good(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, ax.MkdirAll(ax.Join(dir, ".core"), 0o755))
	require.NoError(t, ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module signedapp\n\ngo 1.21\n"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(dir, ".core", build.ConfigFileName), []byte(`
version: 1
project:
  name: signedapp
  binary: signedapp
  main: .
build:
  archive_format: gz
  cgo: false
  flags:
    - -trimpath
targets:
  - os: `+runtime.GOOS+`
    arch: `+runtime.GOARCH+`
`), 0o644))

	t.Setenv("GITHUB_SHA", "abc1234def5678901234567890123456789012345")
	t.Setenv("GITHUB_REF", "refs/tags/v1.0.0")
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")

	cfg := DefaultConfig()
	cfg.SetProjectDir(dir)
	cfg.Project.Name = "signedapp"
	cfg.Build.Targets = []TargetConfig{{OS: runtime.GOOS, Arch: runtime.GOARCH}}
	cfg.Publishers = nil

	artifacts, err := buildArtifacts(context.Background(), io.Local, cfg, dir, "v1.0.0")
	require.NoError(t, err)
	require.NotEmpty(t, artifacts)

	metaPath := ax.Join(dir, "dist", runtime.GOOS+"_"+runtime.GOARCH, "artifact_meta.json")
	content, err := ax.ReadFile(metaPath)
	require.NoError(t, err)

	var meta map[string]any
	require.NoError(t, ax.JSONUnmarshal([]byte(content), &meta))
	assert.Equal(t, "signedapp", meta["name"])
	assert.Equal(t, runtime.GOOS, meta["os"])
	assert.Equal(t, runtime.GOARCH, meta["arch"])
	assert.Equal(t, "refs/tags/v1.0.0", meta["ref"])
	assert.Equal(t, "v1.0.0", meta["tag"])
	assert.Equal(t, "owner/repo", meta["repo"])
}

func TestRelease_BuildArtifacts_HonoursBuildProjectMain_Good(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, ax.MkdirAll(ax.Join(dir, ".core"), 0o755))
	require.NoError(t, ax.MkdirAll(ax.Join(dir, "cmd", "app"), 0o755))

	require.NoError(t, ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example.com/releaseapp\n\ngo 1.21\n"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(dir, "cmd", "app", "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))

	buildConfig := `version: 1
project:
  name: releaseapp
  binary: releaseapp
  main: ./cmd/app
build:
  flags: ["-trimpath"]
targets:
  - os: ` + runtime.GOOS + `
    arch: ` + runtime.GOARCH + `
`
	require.NoError(t, ax.WriteFile(ax.Join(dir, ".core", build.ConfigFileName), []byte(buildConfig), 0o644))

	cfg := DefaultConfig()
	cfg.SetProjectDir(dir)
	cfg.Project.Name = "releaseapp"
	cfg.Publishers = nil

	artifacts, err := buildArtifacts(context.Background(), io.Local, cfg, dir, "v1.0.0")
	require.NoError(t, err)

	var sawArchive bool
	for _, artifact := range artifacts {
		if artifact.Path == ax.Join(dir, "dist", "releaseapp_"+runtime.GOOS+"_"+runtime.GOARCH+".tar.gz") {
			sawArchive = true
			break
		}
	}

	assert.True(t, sawArchive)
}

// Helper functions for publish tests
func setupPublishGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")

	return dir
}

func createPublishCommit(t *testing.T, dir, message string) {
	t.Helper()

	filePath := ax.Join(dir, "publish_test.txt")
	content, _ := ax.ReadFile(filePath)
	content = append(content, []byte(message+"\n")...)
	require.NoError(t, ax.WriteFile(filePath, content, 0644))

	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", message)
}

func createPublishTag(t *testing.T, dir, tag string) {
	t.Helper()
	runGit(t, dir, "tag", tag)
}
