package build

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	stdfs "io/fs"
	"testing"

	"dappco.re/go/build/internal/ax"

	io_interface "dappco.re/go/core/io"
	"github.com/Snider/Borg/pkg/compress"
	"os"
)

type closeErrorWriteCloser struct {
	bytes.Buffer
	closeErr error
}

func (w *closeErrorWriteCloser) Close() error {
	return w.closeErr
}

type closeErrorMedium struct {
	io_interface.Medium
	closeErr error
}

func (m closeErrorMedium) Create(path string) (io.WriteCloser, error) {
	_ = path
	return &closeErrorWriteCloser{closeErr: m.closeErr}, nil
}

// setupArchiveTestFile creates a test binary file in a temp directory with the standard structure.
// Returns the path to the binary and the output directory.
func setupArchiveTestFile(t *testing.T, name, os_, arch string) (binaryPath string, outputDir string) {
	t.Helper()

	outputDir = t.TempDir()

	// Create platform directory: dist/os_arch
	platformDir := ax.Join(outputDir, os_+"_"+arch)
	err := ax.MkdirAll(platformDir, 0755)
	if err != nil {
		t.Fatalf("unexpected error: %v",

			// Create test binary
			err)
	}

	binaryPath = ax.Join(platformDir, name)
	content := []byte("#!/bin/bash\necho 'Hello, World!'\n")
	err = ax.WriteFile(binaryPath, content, 0755)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return binaryPath, outputDir
}

// setupArchiveTestDirectory creates a test directory artifact in a temp directory.
// Returns the path to the directory artifact and the output directory.
func setupArchiveTestDirectory(t *testing.T, name, os_, arch string) (artifactPath string, outputDir string) {
	t.Helper()

	outputDir = t.TempDir()
	platformDir := ax.Join(outputDir, os_+"_"+arch)
	if err := ax.MkdirAll(platformDir, 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	artifactPath = ax.Join(platformDir, name)
	if err := ax.MkdirAll(ax.Join(artifactPath, "Contents", "MacOS"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.MkdirAll(ax.Join(artifactPath, "Resources"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(artifactPath, "Contents", "MacOS", "core"), []byte("bundle binary"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(artifactPath, "Resources", "config.json"), []byte(`{"ok":true}`), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return artifactPath, outputDir
}

func TestArchive_Archive_Good(t *testing.T) {
	fs := io_interface.Local
	t.Run("creates tar.gz for linux", func(t *testing.T) {
		binaryPath, outputDir := setupArchiveTestFile(t, "myapp", "linux", "amd64")

		artifact := Artifact{
			Path: binaryPath,
			OS:   "linux",
			Arch: "amd64",
		}

		result, err := Archive(fs, artifact)
		if err != nil {
			t.Fatalf("unexpected error: %v",

				// Verify archive was created
				err)
		}

		expectedPath := ax.Join(outputDir, "myapp_linux_amd64.tar.gz")
		if !stdlibAssertEqual(expectedPath, result.Path) {
			t.Fatalf("want %v, got %v", expectedPath,

				// Verify OS and Arch are preserved
				result.Path)
		}
		if _, err := os.Stat(result.Path); err != nil {
			t.Fatalf("expected file to exist: %v", result.Path)
		}
		if !stdlibAssertEqual(

			// Verify archive content
			"linux", result.OS) {
			t.Fatalf("want %v, got %v", "linux", result.OS)
		}
		if !stdlibAssertEqual("amd64", result.Arch) {
			t.Fatalf("want %v, got %v", "amd64", result.Arch)
		}

		verifyTarGzContent(t, result.Path, "myapp")
	})

	t.Run("keeps CI-stamped binary names without double-appending the platform", func(t *testing.T) {
		binaryPath, outputDir := setupArchiveTestFile(t, "myapp_linux_amd64_v1.2.3", "linux", "amd64")

		artifact := Artifact{
			Path: binaryPath,
			OS:   "linux",
			Arch: "amd64",
		}

		result, err := Archive(fs, artifact)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedPath := ax.Join(outputDir, "myapp_linux_amd64_v1.2.3.tar.gz")
		if !stdlibAssertEqual(expectedPath, result.Path) {
			t.Fatalf("want %v, got %v", expectedPath, result.Path)
		}
		if _, err := os.Stat(result.Path); err != nil {
			t.Fatalf("expected file to exist: %v", result.Path)
		}

	})

	t.Run("creates tar.gz for darwin", func(t *testing.T) {
		binaryPath, outputDir := setupArchiveTestFile(t, "myapp", "darwin", "arm64")

		artifact := Artifact{
			Path: binaryPath,
			OS:   "darwin",
			Arch: "arm64",
		}

		result, err := Archive(fs, artifact)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedPath := ax.Join(outputDir, "myapp_darwin_arm64.tar.gz")
		if !stdlibAssertEqual(expectedPath, result.Path) {
			t.Fatalf("want %v, got %v", expectedPath, result.Path)
		}
		if _, err := os.Stat(result.Path); err != nil {
			t.Fatalf("expected file to exist: %v", result.Path)
		}

		verifyTarGzContent(t, result.Path, "myapp")
	})

	t.Run("creates zip for windows", func(t *testing.T) {
		binaryPath, outputDir := setupArchiveTestFile(t, "myapp.exe", "windows", "amd64")

		artifact := Artifact{
			Path: binaryPath,
			OS:   "windows",
			Arch: "amd64",
		}

		result, err := Archive(fs, artifact)
		if err != nil {
			t.Fatalf("unexpected error: %v",

				// Windows archives should strip .exe from archive name
				err)
		}

		expectedPath := ax.Join(outputDir, "myapp_windows_amd64.zip")
		if !stdlibAssertEqual(expectedPath, result.Path) {
			t.Fatalf("want %v, got %v", expectedPath, result.Path)
		}
		if _, err := os.Stat(result.Path); err != nil {
			t.Fatalf("expected file to exist: %v", result.Path)
		}

		verifyZipContent(t, result.Path, "myapp.exe")
	})

	t.Run("preserves checksum field", func(t *testing.T) {
		binaryPath, _ := setupArchiveTestFile(t, "myapp", "linux", "amd64")

		artifact := Artifact{
			Path:     binaryPath,
			OS:       "linux",
			Arch:     "amd64",
			Checksum: "abc123",
		}

		result, err := Archive(fs, artifact)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("abc123", result.Checksum) {
			t.Fatalf("want %v, got %v", "abc123", result.Checksum)
		}

	})

	t.Run("creates tar.xz for linux with ArchiveXZ", func(t *testing.T) {
		binaryPath, outputDir := setupArchiveTestFile(t, "myapp", "linux", "amd64")

		artifact := Artifact{
			Path: binaryPath,
			OS:   "linux",
			Arch: "amd64",
		}

		result, err := ArchiveXZ(fs, artifact)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedPath := ax.Join(outputDir, "myapp_linux_amd64.tar.xz")
		if !stdlibAssertEqual(expectedPath, result.Path) {
			t.Fatalf("want %v, got %v", expectedPath, result.Path)
		}
		if _, err := os.Stat(result.Path); err != nil {
			t.Fatalf("expected file to exist: %v", result.Path)
		}

		verifyTarXzContent(t, result.Path, "myapp")
	})

	t.Run("creates tar.xz for darwin with ArchiveWithFormat", func(t *testing.T) {
		binaryPath, outputDir := setupArchiveTestFile(t, "myapp", "darwin", "arm64")

		artifact := Artifact{
			Path: binaryPath,
			OS:   "darwin",
			Arch: "arm64",
		}

		result, err := ArchiveWithFormat(fs, artifact, ArchiveFormatXZ)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedPath := ax.Join(outputDir, "myapp_darwin_arm64.tar.xz")
		if !stdlibAssertEqual(expectedPath, result.Path) {
			t.Fatalf("want %v, got %v", expectedPath, result.Path)
		}
		if _, err := os.Stat(result.Path); err != nil {
			t.Fatalf("expected file to exist: %v", result.Path)
		}

		verifyTarXzContent(t, result.Path, "myapp")
	})

	t.Run("windows still uses zip even with xz format", func(t *testing.T) {
		binaryPath, outputDir := setupArchiveTestFile(t, "myapp.exe", "windows", "amd64")

		artifact := Artifact{
			Path: binaryPath,
			OS:   "windows",
			Arch: "amd64",
		}

		result, err := ArchiveWithFormat(fs, artifact, ArchiveFormatXZ)
		if err != nil {
			t.Fatalf("unexpected error: %v",

				// Windows should still get .zip regardless of format
				err)
		}

		expectedPath := ax.Join(outputDir, "myapp_windows_amd64.zip")
		if !stdlibAssertEqual(expectedPath, result.Path) {
			t.Fatalf("want %v, got %v", expectedPath, result.Path)
		}
		if _, err := os.Stat(result.Path); err != nil {
			t.Fatalf("expected file to exist: %v", result.Path)
		}

		verifyZipContent(t, result.Path, "myapp.exe")
	})

	t.Run("creates zip for linux when explicitly requested", func(t *testing.T) {
		binaryPath, outputDir := setupArchiveTestFile(t, "myapp", "linux", "amd64")

		artifact := Artifact{
			Path: binaryPath,
			OS:   "linux",
			Arch: "amd64",
		}

		result, err := ArchiveWithFormat(fs, artifact, ArchiveFormatZip)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedPath := ax.Join(outputDir, "myapp_linux_amd64.zip")
		if !stdlibAssertEqual(expectedPath, result.Path) {
			t.Fatalf("want %v, got %v", expectedPath, result.Path)
		}
		if _, err := os.Stat(result.Path); err != nil {
			t.Fatalf("expected file to exist: %v", result.Path)
		}

		verifyZipContent(t, result.Path, "myapp")
	})

	t.Run("creates tar.gz for directory artifacts", func(t *testing.T) {
		artifactPath, outputDir := setupArchiveTestDirectory(t, "Core.app", "darwin", "arm64")

		artifact := Artifact{
			Path: artifactPath,
			OS:   "darwin",
			Arch: "arm64",
		}

		result, err := Archive(fs, artifact)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedPath := ax.Join(outputDir, "Core_darwin_arm64.tar.gz")
		if !stdlibAssertEqual(expectedPath, result.Path) {
			t.Fatalf("want %v, got %v", expectedPath, result.Path)
		}
		if _, err := os.Stat(result.Path); err != nil {
			t.Fatalf("expected file to exist: %v", result.Path)
		}
		if !stdlibAssertEqual([]byte("bundle binary"), extractTarGzFile(t, result.Path, "Core.app/Contents/MacOS/core")) {
			t.Fatalf("want %v, got %v", []byte("bundle binary"), extractTarGzFile(t, result.Path, "Core.app/Contents/MacOS/core"))
		}
		if !stdlibAssertEqual([]byte(`{"ok":true}`), extractTarGzFile(t, result.Path, "Core.app/Resources/config.json")) {
			t.Fatalf("want %v, got %v", []byte(`{"ok":true}`), extractTarGzFile(t, result.Path, "Core.app/Resources/config.json"))
		}

	})

	t.Run("creates zip for directory artifacts", func(t *testing.T) {
		artifactPath, outputDir := setupArchiveTestDirectory(t, "bundle", "linux", "amd64")

		artifact := Artifact{
			Path: artifactPath,
			OS:   "linux",
			Arch: "amd64",
		}

		result, err := ArchiveWithFormat(fs, artifact, ArchiveFormatZip)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedPath := ax.Join(outputDir, "bundle_linux_amd64.zip")
		if !stdlibAssertEqual(expectedPath, result.Path) {
			t.Fatalf("want %v, got %v", expectedPath, result.Path)
		}
		if _, err := os.Stat(result.Path); err != nil {
			t.Fatalf("expected file to exist: %v", result.Path)
		}
		if !stdlibAssertEqual([]byte("bundle binary"), extractZipFile(t, result.Path, "bundle/Contents/MacOS/core")) {
			t.Fatalf("want %v, got %v", []byte("bundle binary"), extractZipFile(t, result.Path, "bundle/Contents/MacOS/core"))
		}
		if !stdlibAssertEqual([]byte(`{"ok":true}`), extractZipFile(t, result.Path, "bundle/Resources/config.json")) {
			t.Fatalf("want %v, got %v", []byte(`{"ok":true}`), extractZipFile(t, result.Path, "bundle/Resources/config.json"))
		}

	})
}

func TestArchive_ParseArchiveFormat_Good(t *testing.T) {
	t.Run("defaults to gzip when empty", func(t *testing.T) {
		format, err := ParseArchiveFormat("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(ArchiveFormatGzip, format) {
			t.Fatalf("want %v, got %v", ArchiveFormatGzip, format)
		}

	})

	t.Run("accepts xz aliases", func(t *testing.T) {
		for _, input := range []string{"xz", "txz", "tar.xz", "tar-xz"} {
			format, err := ParseArchiveFormat(input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !stdlibAssertEqual(ArchiveFormatXZ, format) {
				t.Fatalf("want %v, got %v", ArchiveFormatXZ, format)
			}

		}
	})

	t.Run("accepts zip", func(t *testing.T) {
		format, err := ParseArchiveFormat("zip")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(ArchiveFormatZip, format) {
			t.Fatalf("want %v, got %v", ArchiveFormatZip, format)
		}

	})

	t.Run("accepts gzip aliases", func(t *testing.T) {
		for _, input := range []string{"gz", "gzip", "tgz", "tar.gz", "tar-gz"} {
			format, err := ParseArchiveFormat(input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !stdlibAssertEqual(ArchiveFormatGzip, format) {
				t.Fatalf("want %v, got %v", ArchiveFormatGzip, format)
			}

		}
	})

	t.Run("rejects unsupported formats", func(t *testing.T) {
		format, err := ParseArchiveFormat("bzip2")
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertEmpty(format) {
			t.Fatalf("expected empty, got %v", format)
		}
		if !stdlibAssertContains(err.Error(), "unsupported archive format") {
			t.Fatalf("expected %v to contain %v", err.Error(), "unsupported archive format")
		}

	})
}

func TestArchive_Archive_Bad(t *testing.T) {
	fs := io_interface.Local
	t.Run("returns error for empty path", func(t *testing.T) {
		artifact := Artifact{
			Path: "",
			OS:   "linux",
			Arch: "amd64",
		}

		result, err := Archive(fs, artifact)
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "artifact path is empty") {
			t.Fatalf("expected %v to contain %v", err.Error(), "artifact path is empty")
		}
		if !stdlibAssertEmpty(result.Path) {
			t.Fatalf("expected empty, got %v", result.Path)
		}

	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		artifact := Artifact{
			Path: "/nonexistent/path/binary",
			OS:   "linux",
			Arch: "amd64",
		}

		result, err := Archive(fs, artifact)
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "source file not found") {
			t.Fatalf("expected %v to contain %v", err.Error(), "source file not found")
		}
		if !stdlibAssertEmpty(result.Path) {
			t.Fatalf("expected empty, got %v", result.Path)
		}

	})

	t.Run("returns error when the tar.gz writer cannot be closed", func(t *testing.T) {
		binaryPath, _ := setupArchiveTestFile(t, "myapp", "linux", "amd64")

		artifact := Artifact{
			Path: binaryPath,
			OS:   "linux",
			Arch: "amd64",
		}

		result, err := Archive(closeErrorMedium{Medium: io_interface.Local, closeErr: errors.New("close failed")}, artifact)
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "failed to close archive file") {
			t.Fatalf("expected %v to contain %v", err.Error(), "failed to close archive file")
		}
		if !stdlibAssertContains(err.Error(), "close failed") {
			t.Fatalf("expected %v to contain %v", err.Error(), "close failed")
		}
		if !stdlibAssertEmpty(result.Path) {
			t.Fatalf("expected empty, got %v", result.Path)
		}

	})

	t.Run("returns error when the zip writer cannot be closed", func(t *testing.T) {
		binaryPath, _ := setupArchiveTestFile(t, "myapp.exe", "windows", "amd64")

		artifact := Artifact{
			Path: binaryPath,
			OS:   "windows",
			Arch: "amd64",
		}

		result, err := Archive(closeErrorMedium{Medium: io_interface.Local, closeErr: errors.New("close failed")}, artifact)
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "failed to close archive file") {
			t.Fatalf("expected %v to contain %v", err.Error(), "failed to close archive file")
		}
		if !stdlibAssertContains(err.Error(), "close failed") {
			t.Fatalf("expected %v to contain %v", err.Error(), "close failed")
		}
		if !stdlibAssertEmpty(result.Path) {
			t.Fatalf("expected empty, got %v", result.Path)
		}

	})

	t.Run("returns error when the tar.xz archive file cannot be closed", func(t *testing.T) {
		binaryPath, _ := setupArchiveTestFile(t, "myapp", "linux", "amd64")

		artifact := Artifact{
			Path: binaryPath,
			OS:   "linux",
			Arch: "amd64",
		}

		result, err := ArchiveXZ(closeErrorMedium{Medium: io_interface.Local, closeErr: errors.New("close failed")}, artifact)
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "failed to close archive file") {
			t.Fatalf("expected %v to contain %v", err.Error(), "failed to close archive file")
		}
		if !stdlibAssertContains(err.Error(), "close failed") {
			t.Fatalf("expected %v to contain %v", err.Error(), "close failed")
		}
		if !stdlibAssertEmpty(result.Path) {
			t.Fatalf(

				// Create multiple binaries
				"expected empty, got %v", result.Path)
		}

	})

}

func TestArchive_ArchiveAll_Good(t *testing.T) {
	fs := io_interface.Local
	t.Run("archives multiple artifacts", func(t *testing.T) {
		outputDir := t.TempDir()

		var artifacts []Artifact
		targets := []struct {
			os_  string
			arch string
		}{
			{"linux", "amd64"},
			{"linux", "arm64"},
			{"darwin", "arm64"},
			{"windows", "amd64"},
		}

		for _, target := range targets {
			platformDir := ax.Join(outputDir, target.os_+"_"+target.arch)
			err := ax.MkdirAll(platformDir, 0755)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			name := "myapp"
			if target.os_ == "windows" {
				name = "myapp.exe"
			}

			binaryPath := ax.Join(platformDir, name)
			err = ax.WriteFile(binaryPath, []byte("binary content"), 0755)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			artifacts = append(artifacts, Artifact{
				Path: binaryPath,
				OS:   target.os_,
				Arch: target.arch,
			})
		}

		results, err := ArchiveAll(fs, artifacts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) !=

			// Verify all archives were created
			4 {
			t.Fatalf("want len %v, got %v", 4, len(results))
		}

		for i, result := range results {
			if _, err := os.Stat(result.Path); err != nil {
				t.Fatalf("expected file to exist: %v", result.Path)
			}
			if !stdlibAssertEqual(artifacts[i].OS, result.OS) {
				t.Fatalf("want %v, got %v", artifacts[i].OS, result.OS)
			}
			if !stdlibAssertEqual(artifacts[i].Arch, result.Arch) {
				t.Fatalf("want %v, got %v", artifacts[i].Arch, result.Arch)
			}

		}
	})

	t.Run("returns nil for empty slice", func(t *testing.T) {
		results, err := ArchiveAll(fs, []Artifact{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertNil(results) {
			t.Fatalf("expected nil, got %v", results)
		}

	})

	t.Run("returns nil for nil slice", func(t *testing.T) {
		results, err := ArchiveAll(fs, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertNil(results) {
			t.Fatalf("expected nil, got %v", results)
		}

	})
}

func TestArchive_ArchiveAll_Bad(t *testing.T) {
	fs := io_interface.Local
	t.Run("returns partial results on error", func(t *testing.T) {
		binaryPath, _ := setupArchiveTestFile(t, "myapp", "linux", "amd64")

		artifacts := []Artifact{
			{Path: binaryPath, OS: "linux", Arch: "amd64"},
			{Path: "/nonexistent/binary", OS: "linux", Arch: "arm64"}, // This will fail
		}

		results, err := ArchiveAll(fs, artifacts)
		if err == nil {
			t.Fatal("expected error")

			// Should have the first successful result
		}
		if len(results) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(results))
		}
		if _, err := os.Stat(results[0].Path); err != nil {
			t.Fatalf("expected file to exist: %v", results[0].Path)
		}

	})
}

func TestArchive_ArchiveFilename_Good(t *testing.T) {
	t.Run("generates correct tar.gz filename", func(t *testing.T) {
		artifact := Artifact{
			Path: "/output/linux_amd64/myapp",
			OS:   "linux",
			Arch: "amd64",
		}

		filename := archiveFilename(artifact, ".tar.gz")
		if !stdlibAssertEqual("/output/myapp_linux_amd64.tar.gz", filename) {
			t.Fatalf("want %v, got %v", "/output/myapp_linux_amd64.tar.gz", filename)
		}

	})

	t.Run("generates correct zip filename", func(t *testing.T) {
		artifact := Artifact{
			Path: "/output/windows_amd64/myapp.exe",
			OS:   "windows",
			Arch: "amd64",
		}

		filename := archiveFilename(artifact, ".zip")
		if !stdlibAssertEqual("/output/myapp_windows_amd64.zip", filename) {
			t.Fatalf("want %v, got %v", "/output/myapp_windows_amd64.zip", filename)
		}

	})

	t.Run("handles nested output directories", func(t *testing.T) {
		artifact := Artifact{
			Path: "/project/dist/linux_arm64/cli",
			OS:   "linux",
			Arch: "arm64",
		}

		filename := archiveFilename(artifact, ".tar.gz")
		if !stdlibAssertEqual("/project/dist/cli_linux_arm64.tar.gz", filename) {
			t.Fatalf("want %v, got %v", "/project/dist/cli_linux_arm64.tar.gz", filename)
		}

	})

	t.Run("strips app bundle suffix from archive name", func(t *testing.T) {
		artifact := Artifact{
			Path: "/output/darwin_arm64/Core.app",
			OS:   "darwin",
			Arch: "arm64",
		}

		filename := archiveFilename(artifact, ".tar.gz")
		if !stdlibAssertEqual("/output/Core_darwin_arm64.tar.gz", filename) {
			t.Fatalf("want %v, got %v", "/output/Core_darwin_arm64.tar.gz", filename)
		}

	})
}

func TestArchive_RoundTrip_Good(t *testing.T) {
	fs := io_interface.Local

	t.Run("tar.gz round trip preserves content", func(t *testing.T) {
		binaryPath, _ := setupArchiveTestFile(t, "roundtrip-app", "linux", "amd64")

		// Read original content
		originalContent, err := ax.ReadFile(binaryPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		artifact := Artifact{
			Path: binaryPath,
			OS:   "linux",
			Arch: "amd64",
		}

		// Create archive
		archiveArtifact, err := Archive(fs, artifact)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := os.Stat(archiveArtifact.

			// Extract and verify content matches
			Path); err != nil {
			t.Fatalf("expected file to exist: %v", archiveArtifact.Path)
		}

		extractedContent := extractTarGzFile(t, archiveArtifact.Path, "roundtrip-app")
		if !stdlibAssertEqual(originalContent, extractedContent) {
			t.Fatalf("want %v, got %v", originalContent, extractedContent)
		}

	})

	t.Run("tar.xz round trip preserves content", func(t *testing.T) {
		binaryPath, _ := setupArchiveTestFile(t, "roundtrip-xz", "linux", "arm64")

		originalContent, err := ax.ReadFile(binaryPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		artifact := Artifact{
			Path: binaryPath,
			OS:   "linux",
			Arch: "arm64",
		}

		archiveArtifact, err := ArchiveXZ(fs, artifact)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := os.Stat(archiveArtifact.Path); err != nil {
			t.Fatalf("expected file to exist: %v", archiveArtifact.Path)
		}

		extractedContent := extractTarXzFile(t, archiveArtifact.Path, "roundtrip-xz")
		if !stdlibAssertEqual(originalContent, extractedContent) {
			t.Fatalf("want %v, got %v", originalContent, extractedContent)
		}

	})

	t.Run("zip round trip preserves content", func(t *testing.T) {
		binaryPath, _ := setupArchiveTestFile(t, "roundtrip.exe", "windows", "amd64")

		originalContent, err := ax.ReadFile(binaryPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		artifact := Artifact{
			Path: binaryPath,
			OS:   "windows",
			Arch: "amd64",
		}

		archiveArtifact, err := Archive(fs, artifact)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := os.Stat(archiveArtifact.Path); err != nil {
			t.Fatalf("expected file to exist: %v", archiveArtifact.Path)
		}

		extractedContent := extractZipFile(t, archiveArtifact.Path, "roundtrip.exe")
		if !stdlibAssertEqual(originalContent, extractedContent) {
			t.Fatalf("want %v, got %v", originalContent, extractedContent)
		}

	})

	t.Run("tar.gz preserves file permissions", func(t *testing.T) {
		binaryPath, _ := setupArchiveTestFile(t, "perms-app", "linux", "amd64")

		artifact := Artifact{
			Path: binaryPath,
			OS:   "linux",
			Arch: "amd64",
		}

		archiveArtifact, err := Archive(fs, artifact)
		if err != nil {
			t.Fatalf("unexpected error: %v",

				// Extract and verify permissions are preserved
				err)
		}

		mode := extractTarGzFileMode(t, archiveArtifact.Path, "perms-app")
		if !stdlibAssertEqual(
			// The original file was written with 0755
			stdfs.FileMode(0o755), mode&stdfs.ModePerm) {
			t.Fatalf("want %v, got %v", stdfs.FileMode(0o755), mode&stdfs.ModePerm)
		}

	})

	t.Run("round trip with large binary content", func(t *testing.T) {
		outputDir := t.TempDir()
		platformDir := ax.Join(outputDir, "linux_amd64")
		if err := ax.MkdirAll(platformDir, 0755); err != nil {
			t.Fatalf("unexpected error: %v",

				// Create a larger file (64KB)
				err)
		}

		largeContent := make([]byte, 64*1024)
		for i := range largeContent {
			largeContent[i] = byte(i % 256)
		}
		binaryPath := ax.Join(platformDir, "large-app")
		if err := ax.WriteFile(binaryPath, largeContent, 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		artifact := Artifact{
			Path: binaryPath,
			OS:   "linux",
			Arch: "amd64",
		}

		archiveArtifact, err := Archive(fs, artifact)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		extractedContent := extractTarGzFile(t, archiveArtifact.Path, "large-app")
		if !stdlibAssertEqual(largeContent, extractedContent) {
			t.Fatalf("want %v, got %v", largeContent, extractedContent)
		}

	})

	t.Run("archive is smaller than original for tar.gz", func(t *testing.T) {
		outputDir := t.TempDir()
		platformDir := ax.Join(outputDir, "linux_amd64")
		if err := ax.MkdirAll(platformDir, 0755); err != nil {
			t.Fatalf("unexpected error: %v",

				// Create a compressible file (repeated pattern)
				err)
		}

		compressibleContent := make([]byte, 4096)
		for i := range compressibleContent {
			compressibleContent[i] = 'A'
		}
		binaryPath := ax.Join(platformDir, "compressible-app")
		if err := ax.WriteFile(binaryPath, compressibleContent, 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		artifact := Artifact{
			Path: binaryPath,
			OS:   "linux",
			Arch: "amd64",
		}

		archiveArtifact, err := Archive(fs, artifact)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		originalInfo, err := ax.Stat(binaryPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		archiveInfo, err := ax.Stat(archiveArtifact.Path)
		if err != nil {
			t.Fatalf("unexpected error: %v",

				// Compressed archive should be smaller than original
				err)
		}
		if archiveInfo.Size() >= originalInfo.Size() {
			t.Fatalf("expected %v to be less than %v", archiveInfo.Size(),

				// extractTarGzFile extracts a named file from a tar.gz archive and returns its content.
				originalInfo.Size())
		}

	})
}

func extractTarGzFile(t *testing.T, archivePath, fileName string) []byte {
	t.Helper()

	file, err := ax.Open(archivePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defer func() { _ = file.Close() }()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defer func() { _ = gzReader.Close() }()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			t.Fatalf("file %q not found in archive", fileName)
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if header.Name == fileName {
			content, err := io.ReadAll(tarReader)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			return content
		}
	}
}

// extractTarGzFileMode extracts the file mode of a named file from a tar.gz archive.
func extractTarGzFileMode(t *testing.T, archivePath, fileName string) stdfs.FileMode {
	t.Helper()

	file, err := ax.Open(archivePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defer func() { _ = file.Close() }()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defer func() { _ = gzReader.Close() }()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			t.Fatalf("file %q not found in archive", fileName)
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if header.Name == fileName {
			return header.FileInfo().Mode()
		}
	}
}

// extractTarXzFile extracts a named file from a tar.xz archive and returns its content.
func extractTarXzFile(t *testing.T, archivePath, fileName string) []byte {
	t.Helper()

	xzData, err := ax.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tarData, err := compress.Decompress(xzData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tarReader := tar.NewReader(bytes.NewReader(tarData))

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			t.Fatalf("file %q not found in archive", fileName)
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if header.Name == fileName {
			content, err := io.ReadAll(tarReader)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			return content
		}
	}
}

// extractZipFile extracts a named file from a zip archive and returns its content.
func extractZipFile(t *testing.T, archivePath, fileName string) []byte {
	t.Helper()

	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defer func() { _ = reader.Close() }()

	for _, f := range reader.File {
		if f.Name == fileName {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			defer func() { _ = rc.Close() }()

			content, err := io.ReadAll(rc)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			return content
		}
	}

	t.Fatalf("file %q not found in zip archive", fileName)
	return nil
}

// verifyTarGzContent opens a tar.gz file and verifies it contains the expected file.
func verifyTarGzContent(t *testing.T, archivePath, expectedName string) {
	t.Helper()

	file, err := ax.Open(archivePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defer func() { _ = file.Close() }()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defer func() { _ = gzReader.Close() }()

	tarReader := tar.NewReader(gzReader)

	header, err := tarReader.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(expectedName,

		// Verify there's only one file
		header.Name) {
		t.Fatalf("want %v, got %v", expectedName, header.Name)
	}

	_, err = tarReader.Next()
	if !stdlibAssertEqual(io.EOF, err) {
		t.

			// verifyZipContent opens a zip file and verifies it contains the expected file.
			Fatalf("want %v, got %v", io.EOF, err)
	}

}

func verifyZipContent(t *testing.T, archivePath, expectedName string) {
	t.Helper()

	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defer func() { _ = reader.Close() }()
	if len(reader.File) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(reader.File))
	}
	if !stdlibAssertEqual(

		// verifyTarXzContent opens a tar.xz file and verifies it contains the expected file.
		expectedName, reader.File[0].Name) {
		t.Fatalf("want %v, got %v", expectedName, reader.File[0].Name)
	}

}

func verifyTarXzContent(t *testing.T, archivePath, expectedName string) {
	t.Helper()

	// Read the xz-compressed file
	xzData, err := ax.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("unexpected error: %v",

			// Decompress with Borg
			err)
	}

	tarData, err := compress.Decompress(xzData)
	if err != nil {
		t.Fatalf("unexpected error: %v",

			// Read tar archive
			err)
	}

	tarReader := tar.NewReader(bytes.NewReader(tarData))

	header, err := tarReader.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(expectedName,

		// Verify there's only one file
		header.Name) {
		t.Fatalf("want %v, got %v", expectedName, header.Name)
	}

	_, err = tarReader.Next()
	if !stdlibAssertEqual(io.EOF, err) {
		t.Fatalf("want %v, got %v", io.EOF, err)
	}

}
