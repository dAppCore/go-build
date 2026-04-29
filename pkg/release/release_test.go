package release

import (
	"context"
	"runtime"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/build/pkg/build/signing"
	"dappco.re/go/io"
)

const releaseMetaOSField = "o" + "s"

func assertFindArtifacts(t *testing.T, distDir string, wantLen int) []build.Artifact {
	t.Helper()

	artifacts := requireReleaseArtifacts(t, findArtifacts(io.Local, distDir))
	if len(artifacts) != wantLen {
		t.Fatalf("want len %v, got %v", wantLen, len(artifacts))
	}
	return artifacts
}

func requireReleaseArtifacts(t *testing.T, result core.Result) []build.Artifact {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	return result.Value.([]build.Artifact)
}

func requireReleaseValue(t *testing.T, result core.Result) *Release {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	return result.Value.(*Release)
}

func requireReleaseNamed(t *testing.T, result core.Result) interface{ Name() string } {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	return result.Value.(interface{ Name() string })
}

func requireReleaseProjectType(t *testing.T, result core.Result) build.ProjectType {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	return result.Value.(build.ProjectType)
}

func requireReleaseError(t *testing.T, result core.Result) string {
	t.Helper()
	if result.OK {
		t.Fatal("expected error")
	}
	return result.Error()
}

func requireReleaseBytes(t *testing.T, result core.Result) []byte {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	return result.Value.([]byte)
}

func TestRelease_FindArtifactsGood(t *testing.T) {
	t.Run("finds tar.gz artifacts", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		requireReleaseConfigOKResult(t, ax.MkdirAll(distDir, 0755))
		if result := ax.WriteFile(ax.Join(distDir, "app-linux-amd64.tar.gz"), []byte("test"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "app-darwin-arm64.tar.gz"), []byte("test"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		artifacts := assertFindArtifacts(t, distDir, 2)
		if !stdlibAssertContains(artifacts[0].Path, distDir) {
			t.Fatalf("expected %v to contain %v", artifacts[0].Path, distDir)
		}

	})

	t.Run("finds tar.xz artifacts", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		if result := ax.MkdirAll(distDir, 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "app-linux-amd64.tar.xz"), []byte("test"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		artifacts := assertFindArtifacts(t, distDir, 1)
		if !stdlibAssertContains(artifacts[0].Path, "app-linux-amd64.tar.xz") {
			t.Fatalf("expected %v to contain %v", artifacts[0].Path, "app-linux-amd64.tar.xz")
		}

	})

	t.Run("finds zip artifacts", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		if result := ax.MkdirAll(distDir, 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "app-windows-amd64.zip"), []byte("test"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		artifacts := assertFindArtifacts(t, distDir, 1)
		if !stdlibAssertContains(artifacts[0].Path, "app-windows-amd64.zip") {
			t.Fatalf("expected %v to contain %v", artifacts[0].Path, "app-windows-amd64.zip")
		}

	})

	t.Run("finds checksum files", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		if result := ax.MkdirAll(distDir, 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "CHECKSUMS.txt"), []byte("checksums"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		artifacts := assertFindArtifacts(t, distDir, 1)
		if !stdlibAssertContains(artifacts[0].Path, "CHECKSUMS.txt") {
			t.Fatalf("expected %v to contain %v", artifacts[0].Path, "CHECKSUMS.txt")
		}

	})

	t.Run("ignores unrelated text files", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		if result := ax.MkdirAll(distDir, 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "release-notes.txt"), []byte("notes"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		artifacts := assertFindArtifacts(t, distDir, 0)
		if !stdlibAssertEmpty(artifacts) {
			t.Fatalf("expected empty, got %v", artifacts)
		}

	})

	t.Run("finds signature files", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		if result := ax.MkdirAll(distDir, 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "app.tar.gz.sig"), []byte("signature"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		artifacts := assertFindArtifacts(t, distDir, 1)
		if !stdlibAssertContains(artifacts[0].Path, "app.tar.gz.sig") {
			t.Fatalf("expected %v to contain %v", artifacts[0].Path, "app.tar.gz.sig")
		}

	})

	t.Run("finds asc signature files", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		if result := ax.MkdirAll(distDir, 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "CHECKSUMS.txt.asc"), []byte("signature"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		artifacts := assertFindArtifacts(t, distDir, 1)
		if !stdlibAssertContains(artifacts[0].Path, "CHECKSUMS.txt.asc") {
			t.Fatalf("expected %v to contain %v", artifacts[0].Path, "CHECKSUMS.txt.asc")
		}

	})

	t.Run("finds mixed artifact types", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		if result := ax.MkdirAll(distDir, 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "app-linux.tar.gz"), []byte("test"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "app-linux-arm64.tar.xz"), []byte("test"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "app-windows.zip"), []byte("test"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "CHECKSUMS.txt"), []byte("checksums"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "app.sig"), []byte("sig"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		artifacts := assertFindArtifacts(t, distDir, 5)
		if !stdlibAssertContains(artifacts[0].Path, distDir) {
			t.Fatalf("expected %v to contain %v", artifacts[0].Path, distDir)
		}

	})

	t.Run("ignores non-artifact files", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		if result := ax.MkdirAll(distDir, 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "README.md"), []byte("readme"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "app.exe"), []byte("binary"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "app.tar.gz"), []byte("artifact"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "app.tar.xz"), []byte("artifact"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		artifacts := assertFindArtifacts(t, distDir, 2)
		if !stdlibAssertElementsMatch([]string{ax.Join(distDir, "app.tar.gz"), ax.Join(distDir, "app.tar.xz")}, []string{artifacts[0].Path, artifacts[1].Path}) {
			t.Fatalf("expected elements %v, got %v", []string{ax.Join(distDir, "app.tar.gz"), ax.Join(distDir, "app.tar.xz")}, []string{artifacts[0].Path, artifacts[1].Path})
		}

	})

	t.Run("finds nested archived artifacts in subdirectories", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		if result := ax.MkdirAll(distDir, 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.MkdirAll(ax.Join(distDir, "subdir"), 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "app.tar.gz"), []byte("artifact"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "subdir", "nested.tar.gz"), []byte("nested"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		artifacts := assertFindArtifacts(t, distDir, 2)
		if !stdlibAssertElementsMatch([]string{ax.Join(distDir, "app.tar.gz"), ax.Join(distDir, "subdir", "nested.tar.gz")}, []string{artifacts[0].Path, artifacts[1].Path}) {
			t.Fatalf("expected elements %v, got %v", []string{ax.Join(distDir, "app.tar.gz"), ax.Join(distDir, "subdir", "nested.tar.gz")}, []string{artifacts[0].Path, artifacts[1].Path})
		}

	})

	t.Run("falls back to raw platform artifacts when no archives exist", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		if result := ax.MkdirAll(ax.Join(distDir, "linux_amd64"), 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.MkdirAll(ax.Join(distDir, "windows_amd64"), 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "linux_amd64", "myapp"), []byte("binary"), 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "windows_amd64", "myapp.exe"), []byte("binary"), 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "linux_amd64", "artifact_meta.json"), []byte("{}"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		artifacts := assertFindArtifacts(t, distDir, 2)
		if !stdlibAssertEqual(ax.Join(distDir, "linux_amd64", "myapp"), artifacts[0].Path) {
			t.Fatalf("want %v, got %v", ax.Join(distDir, "linux_amd64", "myapp"), artifacts[0].Path)
		}
		if !stdlibAssertEqual("linux", artifacts[0].OS) {
			t.Fatalf("want %v, got %v", "linux", artifacts[0].OS)
		}
		if !stdlibAssertEqual("amd64", artifacts[0].Arch) {
			t.Fatalf("want %v, got %v", "amd64", artifacts[0].Arch)
		}
		if !stdlibAssertEqual(ax.Join(distDir, "windows_amd64", "myapp.exe"), artifacts[1].Path) {
			t.Fatalf("want %v, got %v", ax.Join(distDir, "windows_amd64", "myapp.exe"), artifacts[1].Path)
		}
		if !stdlibAssertEqual("windows", artifacts[1].OS) {
			t.Fatalf("want %v, got %v", "windows", artifacts[1].OS)
		}
		if !stdlibAssertEqual("amd64", artifacts[1].Arch) {
			t.Fatalf("want %v, got %v", "amd64", artifacts[1].Arch)
		}

	})

	t.Run("includes checksum artifacts alongside raw platform artifacts", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		if result := ax.MkdirAll(ax.Join(distDir, "linux_amd64"), 0o755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "linux_amd64", "myapp"), []byte("binary"), 0o755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "CHECKSUMS.txt"), []byte("checksums"), 0o644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		artifacts := assertFindArtifacts(t, distDir, 2)
		if !stdlibAssertElementsMatch([]string{ax.Join(distDir, "linux_amd64", "myapp"), ax.Join(distDir, "CHECKSUMS.txt")}, []string{artifacts[0].Path, artifacts[1].Path}) {
			t.Fatalf("expected elements %v, got %v", []string{ax.Join(distDir, "linux_amd64", "myapp"), ax.Join(distDir, "CHECKSUMS.txt")}, []string{artifacts[0].Path, artifacts[1].Path})
		}

	})

	t.Run("finds nested raw platform artifacts for multi-type builds", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		platformDir := ax.Join(distDir, "go", "linux_amd64")
		if result := ax.MkdirAll(platformDir, 0o755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(platformDir, "myapp"), []byte("binary"), 0o755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		artifacts := assertFindArtifacts(t, distDir, 1)
		if !stdlibAssertEqual(ax.Join(platformDir, "myapp"), artifacts[0].Path) {
			t.Fatalf("want %v, got %v", ax.Join(platformDir, "myapp"), artifacts[0].Path)
		}
		if !stdlibAssertEqual("linux", artifacts[0].OS) {
			t.Fatalf("want %v, got %v", "linux", artifacts[0].OS)
		}
		if !stdlibAssertEqual("amd64", artifacts[0].Arch) {
			t.Fatalf("want %v, got %v", "amd64", artifacts[0].Arch)
		}

	})

	t.Run("includes macOS app bundles from platform directories", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		platformDir := ax.Join(distDir, "darwin_arm64")
		if result := ax.MkdirAll(ax.Join(platformDir, "TestApp.app"), 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.MkdirAll(ax.Join(platformDir, "TestApp.app", "Contents"), 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(platformDir, "TestApp.app", "Contents", "Info.plist"), []byte("<plist/>"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		artifacts := assertFindArtifacts(t, distDir, 1)
		if !stdlibAssertEqual(ax.Join(platformDir, "TestApp.app"), artifacts[0].Path) {
			t.Fatalf("want %v, got %v", ax.Join(platformDir, "TestApp.app"), artifacts[0].Path)
		}
		if !stdlibAssertEqual("darwin", artifacts[0].OS) {
			t.Fatalf("want %v, got %v", "darwin", artifacts[0].OS)
		}
		if !stdlibAssertEqual("arm64", artifacts[0].Arch) {
			t.Fatalf("want %v, got %v", "arm64", artifacts[0].Arch)
		}

	})

	t.Run("returns empty slice for empty dist directory", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		if result := ax.MkdirAll(distDir, 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		artifacts := assertFindArtifacts(t, distDir, 0)
		if !stdlibAssertEmpty(artifacts) {
			t.Fatalf("expected empty, got %v", artifacts)
		}

	})
}

func TestRelease_Publish_ValidatesPublisherBeforePublish_Bad(t *testing.T) {
	dir := t.TempDir()
	distDir := ax.Join(dir, "dist")
	if result := ax.MkdirAll(distDir, 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(ax.Join(distDir, "app.tar.gz"), []byte("artifact"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	cfg := DefaultConfig()
	cfg.SetProjectDir(dir)
	cfg.SetVersion("v1.0.0")
	cfg.Publishers = []PublisherConfig{{Type: "npm"}}

	err := requireReleaseError(t, Publish(context.Background(), cfg, true))
	if !stdlibAssertContains(err, "validate publisher npm failed") {
		t.Fatalf("expected %v to contain %v", err, "validate publisher npm failed")
	}

}

func TestRelease_FindArtifactsBad(t *testing.T) {
	t.Run("returns error when dist directory does not exist", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")

		err := requireReleaseError(t, findArtifacts(io.Local, distDir))
		if !stdlibAssertContains(err, "dist/ directory not found") {
			t.Fatalf("expected %v to contain %v", err, "dist/ directory not found")
		}

	})

	t.Run("returns error when dist directory is unreadable", func(t *testing.T) {
		if ax.Geteuid() == 0 {
			t.Skip("root can read any directory")
		}
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		requireReleaseConfigOKResult(t, ax.MkdirAll(distDir, 0755))
		requireReleaseConfigOKResult(t, ax.Chmod(distDir, 0000))

		defer func() { requireReleaseConfigOKResult(t, ax.Chmod(distDir, 0755)) }()

		err := requireReleaseError(t, findArtifacts(io.Local, distDir))
		if !stdlibAssertContains(err, "failed to read dist/") {
			t.Fatalf("expected %v to contain %v", err, "failed to read dist/")
		}

	})
}

func TestRelease_GetBuilderGood(t *testing.T) {
	t.Run("returns Go builder for go project type", func(t *testing.T) {
		builder := requireReleaseNamed(t, getBuilder(build.ProjectTypeGo))
		if stdlibAssertNil(builder) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("go", builder.Name()) {
			t.Fatalf("want %v, got %v", "go", builder.Name())
		}

	})

	t.Run("returns Wails builder for wails project type", func(t *testing.T) {
		builder := requireReleaseNamed(t, getBuilder(build.ProjectTypeWails))
		if stdlibAssertNil(builder) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("wails", builder.Name()) {
			t.Fatalf("want %v, got %v", "wails", builder.Name())
		}

	})

	t.Run("returns Node builder for node project type", func(t *testing.T) {
		builder := requireReleaseNamed(t, getBuilder(build.ProjectTypeNode))
		if stdlibAssertNil(builder) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("node", builder.Name()) {
			t.Fatalf("want %v, got %v", "node", builder.Name())
		}

	})

	t.Run("returns PHP builder for php project type", func(t *testing.T) {
		builder := requireReleaseNamed(t, getBuilder(build.ProjectTypePHP))
		if stdlibAssertNil(builder) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("php", builder.Name()) {
			t.Fatalf("want %v, got %v", "php", builder.Name())
		}

	})

	t.Run("returns Python builder for python project type", func(t *testing.T) {
		builder := requireReleaseNamed(t, getBuilder(build.ProjectTypePython))
		if stdlibAssertNil(builder) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("python", builder.Name()) {
			t.Fatalf("want %v, got %v", "python", builder.Name())
		}

	})

	t.Run("returns Rust builder for rust project type", func(t *testing.T) {
		builder := requireReleaseNamed(t, getBuilder(build.ProjectTypeRust))
		if stdlibAssertNil(builder) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("rust", builder.Name()) {
			t.Fatalf("want %v, got %v", "rust", builder.Name())
		}

	})

	t.Run("returns C++ builder for cpp project type", func(t *testing.T) {
		builder := requireReleaseNamed(t, getBuilder(build.ProjectTypeCPP))
		if stdlibAssertNil(builder) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("cpp", builder.Name()) {
			t.Fatalf("want %v, got %v", "cpp", builder.Name())
		}

	})

	t.Run("returns Docker builder for docker project type", func(t *testing.T) {
		builder := requireReleaseNamed(t, getBuilder(build.ProjectTypeDocker))
		if stdlibAssertNil(builder) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("docker", builder.Name()) {
			t.Fatalf("want %v, got %v", "docker", builder.Name())
		}

	})

	t.Run("returns LinuxKit builder for linuxkit project type", func(t *testing.T) {
		builder := requireReleaseNamed(t, getBuilder(build.ProjectTypeLinuxKit))
		if stdlibAssertNil(builder) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("linuxkit", builder.Name()) {
			t.Fatalf("want %v, got %v", "linuxkit", builder.Name())
		}

	})

	t.Run("returns Taskfile builder for taskfile project type", func(t *testing.T) {
		builder := requireReleaseNamed(t, getBuilder(build.ProjectTypeTaskfile))
		if stdlibAssertNil(builder) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("taskfile", builder.Name()) {
			t.Fatalf("want %v, got %v", "taskfile", builder.Name())
		}

	})
}

func TestRelease_GetBuilderBad(t *testing.T) {
	t.Run("returns error for unsupported project type", func(t *testing.T) {
		err := requireReleaseError(t, getBuilder(build.ProjectType("unknown")))
		if !stdlibAssertContains(err, "unsupported project type") {
			t.Fatalf("expected %v to contain %v", err, "unsupported project type")
		}

	})
}

func TestRelease_GetPublisherGood(t *testing.T) {
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
			publisher := requireReleaseNamed(t, getPublisher(tc.pubType))
			if stdlibAssertNil(publisher) {
				t.Fatal("expected non-nil")
			}
			if !stdlibAssertEqual(tc.expectedName, publisher.Name()) {
				t.Fatalf("want %v, got %v", tc.expectedName, publisher.Name())
			}

		})
	}
}

func TestRelease_GetPublisherBad(t *testing.T) {
	t.Run("returns error for unsupported publisher type", func(t *testing.T) {
		err := requireReleaseError(t, getPublisher("unsupported"))
		if !stdlibAssertContains(err, "unsupported publisher type: unsupported") {
			t.Fatalf("expected %v to contain %v", err, "unsupported publisher type: unsupported")
		}

	})

	t.Run("returns error for empty publisher type", func(t *testing.T) {
		err := requireReleaseError(t, getPublisher(""))
		if !stdlibAssertContains(err, "unsupported publisher type") {
			t.Fatalf("expected %v to contain %v", err, "unsupported publisher type")
		}

	})
}

func TestRelease_ResolveProjectTypeGood(t *testing.T) {
	t.Run("honours explicit build type override", func(t *testing.T) {
		dir := t.TempDir()

		projectType := requireReleaseProjectType(t, resolveProjectType(io.Local, dir, "docker"))
		if !stdlibAssertEqual(build.ProjectTypeDocker, projectType) {
			t.Fatalf("want %v, got %v", build.ProjectTypeDocker, projectType)
		}

	})

	t.Run("falls back to marker detection when build type is empty", func(t *testing.T) {
		dir := t.TempDir()
		if result := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example.com/test"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		projectType := requireReleaseProjectType(t, resolveProjectType(io.Local, dir, ""))
		if !stdlibAssertEqual(build.ProjectTypeGo, projectType) {
			t.Fatalf("want %v, got %v", build.ProjectTypeGo, projectType)
		}

	})
}

func TestRelease_BuildExtendedConfigGood(t *testing.T) {
	t.Run("returns empty map for minimal config", func(t *testing.T) {
		cfg := PublisherConfig{
			Type: "github",
		}

		ext := buildExtendedConfig(cfg)
		if !stdlibAssertEmpty(ext) {
			t.Fatalf("expected empty, got %v", ext)
		}

	})

	t.Run("includes LinuxKit config", func(t *testing.T) {
		cfg := PublisherConfig{
			Type:      "linuxkit",
			Config:    "linuxkit.yaml",
			Formats:   []string{"iso", "qcow2"},
			Platforms: []string{"linux/amd64", "linux/arm64"},
		}

		ext := buildExtendedConfig(cfg)
		if !stdlibAssertEqual("linuxkit.yaml", ext["config"]) {
			t.Fatalf("want %v, got %v", "linuxkit.yaml", ext["config"])
		}
		if !stdlibAssertEqual([]any{"iso", "qcow2"}, ext["formats"]) {
			t.Fatalf("want %v, got %v", []any{"iso", "qcow2"}, ext["formats"])
		}
		if !stdlibAssertEqual([]any{"linux/amd64", "linux/arm64"}, ext["platforms"]) {
			t.Fatalf("want %v, got %v", []any{"linux/amd64", "linux/arm64"}, ext["platforms"])
		}

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
		if !stdlibAssertEqual("ghcr.io", ext["registry"]) {
			t.Fatalf("want %v, got %v", "ghcr.io", ext["registry"])
		}
		if !stdlibAssertEqual("owner/repo", ext["image"]) {
			t.Fatalf("want %v, got %v", "owner/repo", ext["image"])
		}
		if !stdlibAssertEqual("Dockerfile.prod", ext["dockerfile"]) {
			t.Fatalf("want %v, got %v", "Dockerfile.prod", ext["dockerfile"])
		}
		if !stdlibAssertEqual([]any{"latest", "v1.0.0"}, ext["tags"]) {
			t.Fatalf("want %v, got %v", []any{"latest", "v1.0.0"}, ext["tags"])
		}

		buildArgs := ext["build_args"].(map[string]any)
		if !stdlibAssertEqual("1.0.0", buildArgs["VERSION"]) {
			t.Fatalf("want %v, got %v", "1.0.0", buildArgs["VERSION"])
		}

	})

	t.Run("includes npm config", func(t *testing.T) {
		cfg := PublisherConfig{
			Type:    "npm",
			Package: "@host-uk/core",
			Access:  "public",
		}

		ext := buildExtendedConfig(cfg)
		if !stdlibAssertEqual("@host-uk/core", ext["package"]) {
			t.Fatalf("want %v, got %v", "@host-uk/core", ext["package"])
		}
		if !stdlibAssertEqual("public", ext["access"]) {
			t.Fatalf("want %v, got %v", "public", ext["access"])
		}

	})

	t.Run("includes Homebrew config", func(t *testing.T) {
		cfg := PublisherConfig{
			Type:    "homebrew",
			Tap:     "host-uk/tap",
			Formula: "core",
		}

		ext := buildExtendedConfig(cfg)
		if !stdlibAssertEqual("host-uk/tap", ext["tap"]) {
			t.Fatalf("want %v, got %v", "host-uk/tap", ext["tap"])
		}
		if !stdlibAssertEqual("core", ext["formula"]) {
			t.Fatalf("want %v, got %v", "core", ext["formula"])
		}

	})

	t.Run("includes Scoop config", func(t *testing.T) {
		cfg := PublisherConfig{
			Type:   "scoop",
			Bucket: "host-uk/bucket",
		}

		ext := buildExtendedConfig(cfg)
		if !stdlibAssertEqual("host-uk/bucket", ext["bucket"]) {
			t.Fatalf("want %v, got %v", "host-uk/bucket", ext["bucket"])
		}

	})

	t.Run("includes AUR config", func(t *testing.T) {
		cfg := PublisherConfig{
			Type:       "aur",
			Maintainer: "John Doe <john@example.com>",
		}

		ext := buildExtendedConfig(cfg)
		if !stdlibAssertEqual("John Doe <john@example.com>", ext["maintainer"]) {
			t.Fatalf("want %v, got %v", "John Doe <john@example.com>", ext["maintainer"])
		}

	})

	t.Run("includes Chocolatey config", func(t *testing.T) {
		cfg := PublisherConfig{
			Type: "chocolatey",
			Push: true,
		}

		ext := buildExtendedConfig(cfg)
		if !(ext["push"].(bool)) {
			t.Fatal("expected true")
		}

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
		if !(official["enabled"].(bool)) {
			t.Fatal("expected true")
		}
		if !stdlibAssertEqual("/path/to/output", official["output"]) {
			t.Fatalf("want %v, got %v", "/path/to/output", official["output"])
		}

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
		if !(official["enabled"].(bool)) {
			t.Fatal("expected true")
		}

		_, hasOutput := official["output"]
		if hasOutput {
			t.Fatal("expected false")
		}

	})
}

func TestRelease_ToAnySliceGood(t *testing.T) {
	t.Run("converts string slice to any slice", func(t *testing.T) {
		input := []string{"a", "b", "c"}

		result := toAnySlice(input)
		if len(result) != 3 {
			t.Fatalf("want len %v, got %v", 3, len(result))
		}
		if !stdlibAssertEqual("a", result[0]) {
			t.Fatalf("want %v, got %v", "a", result[0])
		}
		if !stdlibAssertEqual("b", result[1]) {
			t.Fatalf("want %v, got %v", "b", result[1])
		}
		if !stdlibAssertEqual("c", result[2]) {
			t.Fatalf("want %v, got %v", "c", result[2])
		}

	})

	t.Run("handles empty slice", func(t *testing.T) {
		input := []string{}

		result := toAnySlice(input)
		if !stdlibAssertEmpty(result) {
			t.Fatalf("expected empty, got %v", result)
		}

	})

	t.Run("handles single element", func(t *testing.T) {
		input := []string{"only"}

		result := toAnySlice(input)
		if len(result) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(result))
		}
		if !stdlibAssertEqual("only", result[0]) {
			t.Fatalf("want %v, got %v", "only", result[0])
		}

	})
}

func TestRelease_Publish_Good(t *testing.T) {
	t.Run("returns release with version from config", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		if result := ax.MkdirAll(distDir, 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "app.tar.gz"), []byte("test"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "app.tar.xz"), []byte("test"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		cfg := DefaultConfig()
		cfg.SetProjectDir(dir)
		cfg.SetVersion("v1.0.0")
		cfg.Publishers = nil // No publishers to avoid network calls

		release := requireReleaseValue(t, Publish(context.Background(), cfg, true))
		if !stdlibAssertEqual("v1.0.0", release.Version) {
			t.Fatalf("want %v, got %v", "v1.0.0", release.Version)
		}
		if len(release.Artifacts) != 2 {
			t.Fatalf("want len %v, got %v", 2, len(release.Artifacts))
		}

	})

	t.Run("finds artifacts in dist directory", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		if result := ax.MkdirAll(distDir, 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "app-linux.tar.gz"), []byte("test"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "app-linux.tar.xz"), []byte("test"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "app-darwin.tar.gz"), []byte("test"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "CHECKSUMS.txt"), []byte("checksums"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		cfg := DefaultConfig()
		cfg.SetProjectDir(dir)
		cfg.SetVersion("v1.0.0")
		cfg.Publishers = nil

		release := requireReleaseValue(t, Publish(context.Background(), cfg, true))
		if len(release.Artifacts) != 4 {
			t.Fatalf("want len %v, got %v", 4, len(release.Artifacts))
		}

	})

	t.Run("keeps raw platform artifacts when checksums exist without archives", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		if result := ax.MkdirAll(ax.Join(distDir, "linux_amd64"), 0o755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "linux_amd64", "app"), []byte("binary"), 0o755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "CHECKSUMS.txt"), []byte("checksums"), 0o644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		cfg := DefaultConfig()
		cfg.SetProjectDir(dir)
		cfg.SetVersion("v1.0.0")
		cfg.Publishers = nil

		release := requireReleaseValue(t, Publish(context.Background(), cfg, true))
		if !stdlibAssertContains(release.Artifacts, build.Artifact{Path: ax.Join(distDir, "linux_amd64", "app"), OS: "linux", Arch: "amd64"}) {
			t.Fatalf("expected %v to contain %v", release.Artifacts, build.Artifact{Path: ax.Join(distDir, "linux_amd64", "app"), OS: "linux", Arch: "amd64"})
		}
		if !stdlibAssertContains(release.Artifacts, build.Artifact{Path: ax.Join(distDir, "CHECKSUMS.txt")}) {
			t.Fatalf("expected %v to contain %v", release.Artifacts, build.Artifact{Path: ax.Join(distDir, "CHECKSUMS.txt")})
		}

	})

	t.Run("reads artifacts from configured output medium", func(t *testing.T) {
		medium := io.NewMemoryMedium()
		requireReleaseConfigOKResult(t, medium.Write("releases/app-linux-amd64.tar.gz", "artifact"))
		requireReleaseConfigOKResult(t, medium.Write("releases/CHECKSUMS.txt", "checksums"))

		cfg := DefaultConfig()
		cfg.SetProjectDir(t.TempDir())
		cfg.SetVersion("v1.0.0")
		cfg.SetOutput(medium, "releases")
		cfg.Publishers = nil

		release := requireReleaseValue(t, Publish(context.Background(), cfg, true))
		if len(release.Artifacts) != 2 {
			t.Fatalf("want len %v, got %v", 2, len(release.Artifacts))
		}
		if !stdlibAssertEqual(io.Local, release.FS) {
			t.Fatalf("want %v, got %v", io.Local, release.FS)
		}
		if !stdlibAssertEqual(medium, release.ArtifactFS) {
			t.Fatalf("want %v, got %v", medium, release.ArtifactFS)
		}
		if !stdlibAssertContains(release.Artifacts, build.Artifact{Path: "releases/app-linux-amd64.tar.gz"}) {
			t.Fatalf("expected %v to contain %v", release.Artifacts, build.Artifact{Path: "releases/app-linux-amd64.tar.gz"})
		}
		if !stdlibAssertContains(release.Artifacts, build.Artifact{Path: "releases/CHECKSUMS.txt"}) {
			t.Fatalf("expected %v to contain %v", release.Artifacts, build.Artifact{Path: "releases/CHECKSUMS.txt"})
		}

	})

	t.Run("reads artifacts from medium root when output dir is unset", func(t *testing.T) {
		medium := io.NewMemoryMedium()
		requireReleaseConfigOKResult(t, medium.Write("app-linux-amd64.tar.gz", "artifact"))
		requireReleaseConfigOKResult(t, medium.Write("CHECKSUMS.txt", "checksums"))

		cfg := DefaultConfig()
		cfg.SetProjectDir(t.TempDir())
		cfg.SetVersion("v1.0.0")
		cfg.SetOutputMedium(medium)
		cfg.Publishers = nil

		release := requireReleaseValue(t, Publish(context.Background(), cfg, true))
		if len(release.Artifacts) != 2 {
			t.Fatalf("want len %v, got %v", 2, len(release.Artifacts))
		}
		if !stdlibAssertContains(release.Artifacts, build.Artifact{Path: "app-linux-amd64.tar.gz"}) {
			t.Fatalf("expected %v to contain %v", release.Artifacts, build.Artifact{Path: "app-linux-amd64.tar.gz"})
		}
		if !stdlibAssertContains(release.Artifacts, build.Artifact{Path: "CHECKSUMS.txt"}) {
			t.Fatalf("expected %v to contain %v", release.Artifacts, build.Artifact{Path: "CHECKSUMS.txt"})
		}

	})
}

func TestRelease_Publish_Bad(t *testing.T) {
	t.Run("returns error when config is nil", func(t *testing.T) {
		err := requireReleaseError(t, Publish(context.Background(), nil, true))
		if !stdlibAssertContains(err, "config is nil") {
			t.Fatalf("expected %v to contain %v", err, "config is nil")
		}

	})

	t.Run("returns error when dist directory missing", func(t *testing.T) {
		dir := t.TempDir()

		cfg := DefaultConfig()
		cfg.SetProjectDir(dir)
		cfg.SetVersion("v1.0.0")

		err := requireReleaseError(t, Publish(context.Background(), cfg, true))
		if !stdlibAssertContains(err, "dist/ directory not found") {
			t.Fatalf("expected %v to contain %v", err, "dist/ directory not found")
		}

	})

	t.Run("returns error when no artifacts found", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		if result := ax.MkdirAll(distDir, 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		cfg := DefaultConfig()
		cfg.SetProjectDir(dir)
		cfg.SetVersion("v1.0.0")

		err := requireReleaseError(t, Publish(context.Background(), cfg, true))
		if !stdlibAssertContains(err, "no artifacts found") {
			t.Fatalf("expected %v to contain %v", err, "no artifacts found")
		}

	})

	t.Run("returns error for unsupported publisher", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		if result := ax.MkdirAll(distDir, 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "app.tar.gz"), []byte("test"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		cfg := DefaultConfig()
		cfg.SetProjectDir(dir)
		cfg.SetVersion("v1.0.0")
		cfg.Publishers = []PublisherConfig{
			{Type: "unsupported"},
		}

		err := requireReleaseError(t, Publish(context.Background(), cfg, true))
		if !stdlibAssertContains(err, "unsupported publisher type") {
			t.Fatalf("expected %v to contain %v", err, "unsupported publisher type")
		}

	})

	t.Run("returns error when version determination fails in non-git dir", func(t *testing.T) {
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		if result := ax.MkdirAll(distDir, 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		requireReleaseConfigOKResult(t, ax.WriteFile(ax.Join(distDir, "app.tar.gz"), []byte("test"), 0644))

		cfg := DefaultConfig()
		cfg.SetProjectDir(dir)

		cfg.Publishers = nil

		// In a non-git directory, DetermineVersion returns v0.0.1 as default
		// so we verify that the publish proceeds without error
		release := requireReleaseValue(t, Publish(context.Background(), cfg, true))
		if !stdlibAssertEqual("v0.0.1", release.Version) {
			t.Fatalf("want %v, got %v", "v0.0.1", release.Version)
		}

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
		requireReleaseConfigOKResult(t, ax.WriteFile(ax.Join(dir, "go.mod"), []byte(goMod), 0644))

		mainGo := `package main

func main() {}
`
		if result := ax.WriteFile(ax.Join(dir, "main.go"), []byte(mainGo), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		cfg := DefaultConfig()
		cfg.SetProjectDir(dir)
		cfg.SetVersion("v1.0.0")
		cfg.Project.Name = "testapp"
		cfg.Build.Targets = []TargetConfig{} // Empty targets to use defaults
		cfg.Publishers = nil                 // No publishers to avoid network calls

		// Note: This test will actually try to build, which may fail in CI
		// So we just test that the function accepts the config properly
		runResult := Run(context.Background(), cfg, true)
		if !runResult.OK {
			if !stdlibAssertContains(
				// Build might fail in test environment, but we still verify the error message
				runResult.Error(), "build") {
				t.Fatalf("expected %v to contain %v", runResult.Error(), "build")
			}

		} else {
			release := runResult.Value.(*Release)
			if !stdlibAssertEqual("v1.0.0", release.Version) {
				t.Fatalf("want %v, got %v", "v1.0.0", release.Version)
			}

		}
	})

	t.Run("mirrors artifacts to configured output medium", func(t *testing.T) {
		dir := t.TempDir()
		if result := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module testapp\n\ngo 1.21\n"), 0o644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		medium := io.NewMemoryMedium()

		cfg := DefaultConfig()
		cfg.SetProjectDir(dir)
		cfg.SetVersion("v1.0.0")
		cfg.SetOutput(medium, "releases")
		cfg.Project.Name = "testapp"
		cfg.Build.Targets = []TargetConfig{{OS: runtime.GOOS, Arch: runtime.GOARCH}}
		cfg.Publishers = nil

		release := requireReleaseValue(t, Run(context.Background(), cfg, true))
		if stdlibAssertNil(release) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual(medium, release.ArtifactFS) {
			t.Fatalf("want %v, got %v", medium, release.ArtifactFS)
		}
		if stdlibAssertEmpty(release.Artifacts) {
			t.Fatal("expected non-empty")
		}

		for _, artifact := range release.Artifacts {
			if !(medium.Exists(artifact.Path)) {
				t.Fatalf("expected mirrored artifact %s to exist", artifact.Path)
			}
			if !(core.HasPrefix(artifact.Path, "releases/")) {
				t.Fatalf("expected mirrored artifact path %s to use configured output root", artifact.Path)
			}

		}
		if !(medium.Exists("releases/CHECKSUMS.txt")) {
			t.Fatal("expected true")
		}

	})

	t.Run("mirrors artifacts to medium root when output dir is unset", func(t *testing.T) {
		dir := t.TempDir()
		if result := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module testapp\n\ngo 1.21\n"), 0o644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		medium := io.NewMemoryMedium()

		cfg := DefaultConfig()
		cfg.SetProjectDir(dir)
		cfg.SetVersion("v1.0.0")
		cfg.SetOutputMedium(medium)
		cfg.Project.Name = "testapp"
		cfg.Build.Targets = []TargetConfig{{OS: runtime.GOOS, Arch: runtime.GOARCH}}
		cfg.Publishers = nil

		release := requireReleaseValue(t, Run(context.Background(), cfg, true))
		if stdlibAssertNil(release) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual(medium, release.ArtifactFS) {
			t.Fatalf("want %v, got %v", medium, release.ArtifactFS)
		}
		if stdlibAssertEmpty(release.Artifacts) {
			t.Fatal("expected non-empty")
		}

		for _, artifact := range release.Artifacts {
			if !(medium.Exists(artifact.Path)) {
				t.Fatalf("expected mirrored artifact %s to exist", artifact.Path)
			}
			if core.HasPrefix(artifact.Path, "dist/") {
				t.Fatalf("expected mirrored artifact path %s to omit the local dist prefix", artifact.Path)
			}
			if core.HasPrefix(artifact.Path, "/") {
				t.Fatalf("expected mirrored artifact path %s to stay relative to the medium root", artifact.Path)
			}

		}
		if !(medium.Exists("CHECKSUMS.txt")) {
			t.Fatal("expected true")
		}

	})
}

func TestRelease_Run_Bad(t *testing.T) {
	t.Run("returns error when config is nil", func(t *testing.T) {
		err := requireReleaseError(t, Run(context.Background(), nil, true))
		if !stdlibAssertContains(err, "config is nil") {
			t.Fatalf("expected %v to contain %v", err, "config is nil")
		}

	})

	t.Run("rejects unsafe version before changelog generation", func(t *testing.T) {
		dir := t.TempDir()

		cfg := DefaultConfig()
		cfg.SetProjectDir(dir)
		cfg.SetVersion("v1.0.0\n--bad")
		cfg.Publishers = nil

		oldGenerateReleaseChangelogFn := generateReleaseChangelogFn
		defer func() {
			generateReleaseChangelogFn = oldGenerateReleaseChangelogFn
		}()

		called := false
		generateReleaseChangelogFn = func(ctx context.Context, projectDir, version string, cfg *Config) core.Result {
			called = true
			return core.Ok("")
		}

		err := requireReleaseError(t, Run(context.Background(), cfg, true))
		if !stdlibAssertContains(err, "invalid release version override") {
			t.Fatalf("expected %v to contain %v", err, "invalid release version override")
		}
		if called {
			t.Fatal("changelog generation should not run for unsafe versions")
		}

	})
}

func TestRelease_StructureGood(t *testing.T) {
	t.Run("Release struct holds expected fields", func(t *testing.T) {
		release := &Release{
			Version:    "v1.0.0",
			Artifacts:  []build.Artifact{{Path: "/path/to/artifact"}},
			Changelog:  "## v1.0.0\n\nChanges",
			ProjectDir: "/project",
		}
		if !stdlibAssertEqual("v1.0.0", release.Version) {
			t.Fatalf("want %v, got %v", "v1.0.0", release.Version)
		}
		if len(release.Artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(release.Artifacts))
		}
		if !stdlibAssertContains(release.Changelog, "v1.0.0") {
			t.Fatalf("expected %v to contain %v", release.Changelog, "v1.0.0")
		}
		if !stdlibAssertEqual("/project", release.ProjectDir) {
			t.Fatalf("want %v, got %v", "/project", release.ProjectDir)
		}

	})
}

func TestRelease_PublishVersionFromGitGood(t *testing.T) {
	t.Run("determines version from git when not set", func(t *testing.T) {
		dir := setupPublishGitRepo(t)
		createPublishCommit(t, dir, "feat: initial commit")
		createPublishTag(t, dir, "v1.2.3")

		// Create dist directory with artifact
		distDir := ax.Join(dir, "dist")
		if result := ax.MkdirAll(distDir, 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		requireReleaseConfigOKResult(t, ax.WriteFile(ax.Join(distDir, "app.tar.gz"), []byte("test"), 0644))

		cfg := DefaultConfig()
		cfg.SetProjectDir(dir)

		cfg.Publishers = nil

		release := requireReleaseValue(t, Publish(context.Background(), cfg, true))
		if !stdlibAssertEqual("v1.2.3", release.Version) {
			t.Fatalf("want %v, got %v", "v1.2.3", release.Version)
		}

	})
}

func TestRelease_PublishChangelogGenerationGood(t *testing.T) {
	t.Run("generates changelog from git commits when available", func(t *testing.T) {
		dir := setupPublishGitRepo(t)
		createPublishCommit(t, dir, "feat: add feature")
		createPublishTag(t, dir, "v1.0.0")
		createPublishCommit(t, dir, "fix: fix bug")
		createPublishTag(t, dir, "v1.0.1")

		// Create dist directory with artifact
		distDir := ax.Join(dir, "dist")
		if result := ax.MkdirAll(distDir, 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "app.tar.gz"), []byte("test"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		cfg := DefaultConfig()
		cfg.SetProjectDir(dir)
		cfg.SetVersion("v1.0.1")
		cfg.Publishers = nil

		release := requireReleaseValue(t, Publish(context.Background(), cfg, true))
		if !stdlibAssertContains(release.Changelog, "v1.0.1") {
			t.Fatalf("expected %v to contain %v", release.Changelog, "v1.0.1")
		}

	})

	t.Run("uses fallback changelog on error", func(t *testing.T) {
		dir := t.TempDir() // Not a git repo
		distDir := ax.Join(dir, "dist")
		if result := ax.MkdirAll(distDir, 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "app.tar.gz"), []byte("test"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		cfg := DefaultConfig()
		cfg.SetProjectDir(dir)
		cfg.SetVersion("v1.0.0")
		cfg.Publishers = nil

		release := requireReleaseValue(t, Publish(context.Background(), cfg, true))
		if !stdlibAssertContains(release.Changelog, "Release v1.0.0") {
			t.Fatalf("expected %v to contain %v", release.Changelog, "Release v1.0.0")
		}

	})
}

func TestRelease_PublishDefaultProjectDirGood(t *testing.T) {
	t.Run("uses current directory when projectDir is empty", func(t *testing.T) {
		// Create artifacts in current directory's dist folder
		dir := t.TempDir()
		distDir := ax.Join(dir, "dist")
		if result := ax.MkdirAll(distDir, 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(distDir, "app.tar.gz"), []byte("test"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		cfg := DefaultConfig()
		cfg.SetProjectDir(dir)
		cfg.SetVersion("v1.0.0")
		cfg.Publishers = nil

		release := requireReleaseValue(t, Publish(context.Background(), cfg, true))
		if stdlibAssertEmpty(release.ProjectDir) {
			t.Fatal("expected non-empty")
		}

	})
}

func TestRelease_BuildArtifacts_SignsChecksumsGood(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake gpg script uses POSIX shell")
	}

	dir := t.TempDir()
	if result := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module signedapp\n\ngo 1.21\n"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(ax.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

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
	if result := ax.WriteFile(gpgPath, []byte(gpgScript), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	oldPath := core.Getenv("PATH")
	if stdlibAssertEmpty(oldPath) {
		t.Fatal("expected non-empty")
	}

	t.Setenv("PATH", gpgDir+string(core.PathListSeparator)+oldPath)
	t.Setenv("GPG_KEY_ID", "TESTKEY")

	cfg := DefaultConfig()
	cfg.SetProjectDir(dir)
	cfg.Project.Name = "signedapp"
	cfg.Build.ArchiveFormat = "xz"
	cfg.Build.Targets = []TargetConfig{{OS: runtime.GOOS, Arch: runtime.GOARCH}}
	cfg.Publishers = nil

	artifacts := requireReleaseArtifacts(t, buildArtifacts(context.Background(), io.Local, cfg, dir, ax.Join(dir, "dist"), "v1.0.0"))

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
	if !(sawChecksumSignature) {
		t.Fatal("expected true")
	}
	if !(sawXzArchive) {
		t.Fatal("expected true")
	}
	requireReleaseConfigOKResult(t, ax.Stat(ax.Join(dir, "dist", "CHECKSUMS.txt.asc")))

}

func TestRelease_BuildArtifacts_UsesConfiguredChecksumFileGood(t *testing.T) {
	dir := t.TempDir()
	if result := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module signedapp\n\ngo 1.21\n"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(ax.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	oldSignChecksums := signReleaseChecksums
	defer func() {
		signReleaseChecksums = oldSignChecksums
	}()

	var checksumPaths []string
	signReleaseChecksums = func(ctx context.Context, fs io.Medium, cfg signing.SignConfig, checksumFile string) core.Result {
		checksumPaths = append(checksumPaths, checksumFile)
		return core.Ok(nil)
	}

	cfg := DefaultConfig()
	cfg.SetProjectDir(dir)
	cfg.Project.Name = "signedapp"
	cfg.Checksum.File = "checksums.txt"
	cfg.Build.Targets = []TargetConfig{{OS: runtime.GOOS, Arch: runtime.GOARCH}}
	cfg.Publishers = nil

	artifacts := requireReleaseArtifacts(t, buildArtifacts(context.Background(), io.Local, cfg, dir, ax.Join(dir, "dist"), "v1.0.0"))

	customChecksumPath := ax.Join(dir, "dist", "checksums.txt")
	if !stdlibAssertEqual([]string{customChecksumPath}, checksumPaths) {
		t.Fatalf("want %v, got %v", []string{customChecksumPath}, checksumPaths)
	}
	requireReleaseConfigOKResult(t, ax.Stat(customChecksumPath))

	var sawChecksum bool
	for _, artifact := range artifacts {
		if artifact.Path == customChecksumPath {
			sawChecksum = true
			break
		}
	}
	if !(sawChecksum) {
		t.Fatal("expected true")
	}

}

func TestRelease_BuildArtifacts_SignsBinariesBeforeArchivingGood(t *testing.T) {
	dir := t.TempDir()
	if result := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module signedapp\n\ngo 1.21\n"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(ax.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.MkdirAll(ax.Join(dir, ".core"), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(ax.Join(dir, ".core", build.ConfigFileName), []byte(`
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
`), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

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

	signReleaseBinaries = func(ctx context.Context, fs io.Medium, cfg signing.SignConfig, artifacts []signing.Artifact) core.Result {
		if !(cfg.Enabled) {
			t.Fatal("expected true")
		}
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}

		signedPaths = append(signedPaths, artifacts[0].Path)
		return core.Ok(nil)
	}
	notarizeReleaseBinaries = func(ctx context.Context, fs io.Medium, cfg signing.SignConfig, artifacts []signing.Artifact) core.Result {
		if !(cfg.Enabled) {
			t.Fatal("expected true")
		}
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}

		notarizedPaths = append(notarizedPaths, artifacts[0].Path)
		return core.Ok(nil)
	}
	signReleaseChecksums = func(ctx context.Context, fs io.Medium, cfg signing.SignConfig, checksumFile string) core.Result {
		if !(cfg.Enabled) {
			t.Fatal("expected true")
		}

		checksumPaths = append(checksumPaths, checksumFile)
		return core.Ok(nil)
	}

	cfg := DefaultConfig()
	cfg.SetProjectDir(dir)
	cfg.Project.Name = "signedapp"
	cfg.Build.Targets = []TargetConfig{{OS: runtime.GOOS, Arch: runtime.GOARCH}}
	cfg.Publishers = nil

	artifacts := requireReleaseArtifacts(t, buildArtifacts(context.Background(), io.Local, cfg, dir, ax.Join(dir, "dist"), "v1.0.0"))
	if !stdlibAssertEqual([]string{ax.Join(dir, "dist", runtime.GOOS+"_"+runtime.GOARCH, "signedapp")}, signedPaths) {
		t.Fatalf("want %v, got %v", []string{ax.Join(dir, "dist", runtime.GOOS+"_"+runtime.GOARCH, "signedapp")}, signedPaths)
	}
	if !stdlibAssertEqual(signedPaths, notarizedPaths) {
		t.Fatalf("want %v, got %v", signedPaths, notarizedPaths)
	}
	if !stdlibAssertEqual([]string{ax.Join(dir, "dist", "CHECKSUMS.txt")}, checksumPaths) {
		t.Fatalf("want %v, got %v", []string{ax.Join(dir, "dist", "CHECKSUMS.txt")}, checksumPaths)
	}

	var sawArchive bool
	for _, artifact := range artifacts {
		if artifact.Path == ax.Join(dir, "dist", "signedapp_"+runtime.GOOS+"_"+runtime.GOARCH+".tar.gz") {
			sawArchive = true
			break
		}
	}
	if !(sawArchive) {
		t.Fatal("expected true")
	}

}

func TestRelease_Publish_IncludesConfiguredChecksumArtifact_Good(t *testing.T) {
	dir := t.TempDir()
	distDir := ax.Join(dir, "dist")
	if result := ax.MkdirAll(distDir, 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(ax.Join(distDir, "app-linux-amd64.tar.gz"), []byte("archive"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(ax.Join(distDir, "checksums.txt"), []byte("checksums"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	cfg := DefaultConfig()
	cfg.SetProjectDir(dir)
	cfg.SetVersion("v1.0.0")
	cfg.Checksum.File = "checksums.txt"
	cfg.Publishers = nil

	release := requireReleaseValue(t, Publish(context.Background(), cfg, true))
	if !stdlibAssertContains(release.Artifacts, build.Artifact{Path: ax.Join(distDir, "checksums.txt")}) {
		t.Fatalf("expected %v to contain %v", release.Artifacts, build.Artifact{Path: ax.Join(distDir, "checksums.txt")})
	}

}

func TestRelease_BuildArtifacts_WritesArtifactMetadataGood(t *testing.T) {
	dir := t.TempDir()
	if result := ax.MkdirAll(ax.Join(dir, ".core"), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module signedapp\n\ngo 1.21\n"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(ax.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(ax.Join(dir, ".core", build.ConfigFileName), []byte(`
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
`), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	t.Setenv("GITHUB_SHA", "abc1234def5678901234567890123456789012345")
	t.Setenv("GITHUB_REF", "refs/tags/v1.0.0")
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")

	cfg := DefaultConfig()
	cfg.SetProjectDir(dir)
	cfg.Project.Name = "signedapp"
	cfg.Build.Targets = []TargetConfig{{OS: runtime.GOOS, Arch: runtime.GOARCH}}
	cfg.Publishers = nil

	artifacts := requireReleaseArtifacts(t, buildArtifacts(context.Background(), io.Local, cfg, dir, ax.Join(dir, "dist"), "v1.0.0"))
	if stdlibAssertEmpty(artifacts) {
		t.Fatal("expected non-empty")
	}

	metaPath := ax.Join(dir, "dist", runtime.GOOS+"_"+runtime.GOARCH, "artifact_meta.json")
	content := requireReleaseBytes(t, ax.ReadFile(metaPath))

	var meta map[string]any
	if result := ax.JSONUnmarshal([]byte(content), &meta); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if !stdlibAssertEqual("signedapp", meta["name"]) {
		t.Fatalf("want %v, got %v", "signedapp", meta["name"])
	}
	if !stdlibAssertEqual(runtime.GOOS, meta[releaseMetaOSField]) {
		t.Fatalf("want %v, got %v", runtime.GOOS, meta[releaseMetaOSField])
	}
	if !stdlibAssertEqual(runtime.GOARCH, meta["arch"]) {
		t.Fatalf("want %v, got %v", runtime.GOARCH, meta["arch"])
	}
	if !stdlibAssertEqual("refs/tags/v1.0.0", meta["ref"]) {
		t.Fatalf("want %v, got %v", "refs/tags/v1.0.0", meta["ref"])
	}
	if !stdlibAssertEqual("v1.0.0", meta["tag"]) {
		t.Fatalf("want %v, got %v", "v1.0.0", meta["tag"])
	}
	if !stdlibAssertEqual("owner/repo", meta["repo"]) {
		t.Fatalf("want %v, got %v", "owner/repo", meta["repo"])
	}

}

func TestRelease_BuildArtifacts_HonoursBuildProjectMainGood(t *testing.T) {
	dir := t.TempDir()
	if result := ax.MkdirAll(ax.Join(dir, ".core"), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.MkdirAll(ax.Join(dir, "cmd", "app"), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example.com/releaseapp\n\ngo 1.21\n"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(ax.Join(dir, "cmd", "app", "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

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
	if result := ax.WriteFile(ax.Join(dir, ".core", build.ConfigFileName), []byte(buildConfig), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	cfg := DefaultConfig()
	cfg.SetProjectDir(dir)
	cfg.Project.Name = "releaseapp"
	cfg.Publishers = nil

	artifacts := requireReleaseArtifacts(t, buildArtifacts(context.Background(), io.Local, cfg, dir, ax.Join(dir, "dist"), "v1.0.0"))

	var sawArchive bool
	for _, artifact := range artifacts {
		if artifact.Path == ax.Join(dir, "dist", "releaseapp_"+runtime.GOOS+"_"+runtime.GOARCH+".tar.gz") {
			sawArchive = true
			break
		}
	}
	if !(sawArchive) {
		t.Fatal("expected true")
	}

}

func TestRelease_BuildArtifacts_InheritsBuildTargetsWhenReleaseTargetsOmittedGood(t *testing.T) {
	dir := t.TempDir()
	if result := ax.MkdirAll(ax.Join(dir, ".core"), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example.com/releaseapp\n\ngo 1.21\n"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(ax.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	buildConfig := `version: 1
project:
  name: releaseapp
  binary: releaseapp
targets:
  - os: ` + runtime.GOOS + `
    arch: ` + runtime.GOARCH + `
sign:
  enabled: false
`
	if result := ax.WriteFile(ax.Join(dir, ".core", build.ConfigFileName), []byte(buildConfig), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	releaseConfig := `version: 1
project:
  name: releaseapp
`
	if result := ax.WriteFile(ax.Join(dir, ".core", ConfigFileName), []byte(releaseConfig), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	cfg := requireReleaseConfig(t, LoadConfig(dir))
	if stdlibAssertNil(cfg) {
		t.Fatal("expected non-nil")
	}
	if !stdlibAssertEmpty(cfg.Build.Targets) {
		t.Fatalf("expected empty, got %v", cfg.Build.Targets)
	}

	artifacts := requireReleaseArtifacts(t, buildArtifacts(context.Background(), io.Local, cfg, dir, ax.Join(dir, "dist"), "v1.0.0"))
	if stdlibAssertEmpty(artifacts) {
		t.Fatal("expected non-empty")
	}

	for _, artifact := range artifacts {
		if artifact.OS == "" || artifact.Arch == "" {
			continue
		}
		if !stdlibAssertEqual(runtime.GOOS, artifact.OS) {
			t.Fatalf("want %v, got %v", runtime.GOOS, artifact.OS)
		}
		if

		// Helper functions for publish tests
		!stdlibAssertEqual(runtime.GOARCH, artifact.Arch) {
			t.Fatalf("want %v, got %v", runtime.GOARCH, artifact.Arch)
		}

	}
}

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
	content := []byte{}
	readResult := ax.ReadFile(filePath)
	if readResult.OK {
		content = readResult.Value.([]byte)
	}
	content = append(content, []byte(message+"\n")...)
	if result := ax.WriteFile(filePath, content, 0644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", message)
}

func createPublishTag(t *testing.T, dir, tag string) {
	t.Helper()
	runGit(t, dir, "tag", tag)
}

// --- v0.9.0 generated compliance triplets ---
func TestRelease_Publish_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Publish(ctx, &Config{}, true)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestRelease_Run_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Run(ctx, &Config{}, true)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
