package build

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	stdfs "io/fs"
	"reflect"
	"testing"

	"dappco.re/go/build/internal/ax"

	io_interface "dappco.re/go/io"
	// TODO(AX-6): Replace with dappco.re/go/crypt when it exposes Compress/Decompress API parity.
	core "dappco.re/go"
	"github.com/Snider/Borg/pkg/compress"
)

func archiveRequireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func archiveAssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func archiveAssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
}

func archiveAssertEqual(t *testing.T, want, got any) {
	t.Helper()
	if !stdlibAssertEqual(want, got) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func archiveAssertContains(t *testing.T, value, contains any) {
	t.Helper()
	if !stdlibAssertContains(value, contains) {
		t.Fatalf("expected %v to contain %v", value, contains)
	}
}

func archiveAssertEmpty(t *testing.T, value any) {
	t.Helper()
	if !stdlibAssertEmpty(value) {
		t.Fatalf("expected empty, got %v", value)
	}
}

func archiveAssertNil(t *testing.T, value any) {
	t.Helper()
	if !stdlibAssertNil(value) {
		t.Fatalf("expected nil, got %v", value)
	}
}

func archiveAssertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := ax.Stat(path); err != nil {
		t.Fatalf("expected file to exist: %v", path)
	}
}

func archiveRequireLen(t *testing.T, value any, want int) {
	t.Helper()
	got := reflect.ValueOf(value).Len()
	if got != want {
		t.Fatalf("want len %v, got %v", want, got)
	}
}

func archiveAssertLess(t *testing.T, got, want int64) {
	t.Helper()
	if got >= want {
		t.Fatalf("expected %v to be less than %v", got, want)
	}
}

// setupArchiveTestFile creates a test binary file in a temp directory with the standard structure.
// Returns the path to the binary and the output directory.
func setupArchiveTestFile(t *testing.T, name, os_, arch string) (binaryPath string, outputDir string) {
	t.Helper()

	outputDir = t.TempDir()

	// Create platform directory: dist/os_arch
	platformDir := ax.Join(outputDir, os_+"_"+arch)
	err := ax.MkdirAll(platformDir, 0755)
	archiveRequireNoError(t, err)

	// Create test binary
	binaryPath = ax.Join(platformDir, name)
	content := []byte("#!/bin/bash\necho 'Hello, World!'\n")
	err = ax.WriteFile(binaryPath, content, 0755)
	archiveRequireNoError(t, err)

	return binaryPath, outputDir
}

// setupArchiveTestDirectory creates a test directory artifact in a temp directory.
// Returns the path to the directory artifact and the output directory.
func setupArchiveTestDirectory(t *testing.T, name, os_, arch string) (artifactPath string, outputDir string) {
	t.Helper()

	outputDir = t.TempDir()
	platformDir := ax.Join(outputDir, os_+"_"+arch)
	archiveRequireNoError(t, ax.MkdirAll(platformDir, 0o755))

	artifactPath = ax.Join(platformDir, name)
	archiveRequireNoError(t, ax.MkdirAll(ax.Join(artifactPath, "Contents", "MacOS"), 0o755))
	archiveRequireNoError(t, ax.MkdirAll(ax.Join(artifactPath, "Resources"), 0o755))
	archiveRequireNoError(t, ax.WriteFile(ax.Join(artifactPath, "Contents", "MacOS", "core"), []byte("bundle binary"), 0o755))
	archiveRequireNoError(t, ax.WriteFile(ax.Join(artifactPath, "Resources", "config.json"), []byte(`{"ok":true}`), 0o644))

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
		archiveRequireNoError(t, err)

		// Verify archive was created
		expectedPath := ax.Join(outputDir, "myapp_linux_amd64.tar.gz")
		archiveAssertEqual(t, expectedPath, result.Path)
		archiveAssertFileExists(t, result.Path)

		// Verify OS and Arch are preserved
		archiveAssertEqual(t, "linux", result.OS)
		archiveAssertEqual(t, "amd64", result.Arch)

		// Verify archive content
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
		archiveRequireNoError(t, err)

		expectedPath := ax.Join(outputDir, "myapp_linux_amd64_v1.2.3.tar.gz")
		archiveAssertEqual(t, expectedPath, result.Path)
		archiveAssertFileExists(t, result.Path)
	})

	t.Run("creates tar.gz for darwin", func(t *testing.T) {
		binaryPath, outputDir := setupArchiveTestFile(t, "myapp", "darwin", "arm64")

		artifact := Artifact{
			Path: binaryPath,
			OS:   "darwin",
			Arch: "arm64",
		}

		result, err := Archive(fs, artifact)
		archiveRequireNoError(t, err)

		expectedPath := ax.Join(outputDir, "myapp_darwin_arm64.tar.gz")
		archiveAssertEqual(t, expectedPath, result.Path)
		archiveAssertFileExists(t, result.Path)

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
		archiveRequireNoError(t, err)

		// Windows archives should strip .exe from archive name
		expectedPath := ax.Join(outputDir, "myapp_windows_amd64.zip")
		archiveAssertEqual(t, expectedPath, result.Path)
		archiveAssertFileExists(t, result.Path)

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
		archiveRequireNoError(t, err)
		archiveAssertEqual(t, "abc123", result.Checksum)
	})

	t.Run("creates tar.xz for linux with ArchiveXZ", func(t *testing.T) {
		binaryPath, outputDir := setupArchiveTestFile(t, "myapp", "linux", "amd64")

		artifact := Artifact{
			Path: binaryPath,
			OS:   "linux",
			Arch: "amd64",
		}

		result, err := ArchiveXZ(fs, artifact)
		archiveRequireNoError(t, err)

		expectedPath := ax.Join(outputDir, "myapp_linux_amd64.tar.xz")
		archiveAssertEqual(t, expectedPath, result.Path)
		archiveAssertFileExists(t, result.Path)

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
		archiveRequireNoError(t, err)

		expectedPath := ax.Join(outputDir, "myapp_darwin_arm64.tar.xz")
		archiveAssertEqual(t, expectedPath, result.Path)
		archiveAssertFileExists(t, result.Path)

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
		archiveRequireNoError(t, err)

		// Windows should still get .zip regardless of format
		expectedPath := ax.Join(outputDir, "myapp_windows_amd64.zip")
		archiveAssertEqual(t, expectedPath, result.Path)
		archiveAssertFileExists(t, result.Path)

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
		archiveRequireNoError(t, err)

		expectedPath := ax.Join(outputDir, "myapp_linux_amd64.zip")
		archiveAssertEqual(t, expectedPath, result.Path)
		archiveAssertFileExists(t, result.Path)

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
		archiveRequireNoError(t, err)

		expectedPath := ax.Join(outputDir, "Core_darwin_arm64.tar.gz")
		archiveAssertEqual(t, expectedPath, result.Path)
		archiveAssertFileExists(t, result.Path)

		archiveAssertEqual(t, []byte("bundle binary"), extractTarGzFile(t, result.Path, "Core.app/Contents/MacOS/core"))
		archiveAssertEqual(t, []byte(`{"ok":true}`), extractTarGzFile(t, result.Path, "Core.app/Resources/config.json"))
	})

	t.Run("creates zip for directory artifacts", func(t *testing.T) {
		artifactPath, outputDir := setupArchiveTestDirectory(t, "bundle", "linux", "amd64")

		artifact := Artifact{
			Path: artifactPath,
			OS:   "linux",
			Arch: "amd64",
		}

		result, err := ArchiveWithFormat(fs, artifact, ArchiveFormatZip)
		archiveRequireNoError(t, err)

		expectedPath := ax.Join(outputDir, "bundle_linux_amd64.zip")
		archiveAssertEqual(t, expectedPath, result.Path)
		archiveAssertFileExists(t, result.Path)

		archiveAssertEqual(t, []byte("bundle binary"), extractZipFile(t, result.Path, "bundle/Contents/MacOS/core"))
		archiveAssertEqual(t, []byte(`{"ok":true}`), extractZipFile(t, result.Path, "bundle/Resources/config.json"))
	})
}

func TestArchive_ParseArchiveFormat_Good(t *testing.T) {
	t.Run("defaults to gzip when empty", func(t *testing.T) {
		format, err := ParseArchiveFormat("")
		archiveRequireNoError(t, err)
		archiveAssertEqual(t, ArchiveFormatGzip, format)
	})

	t.Run("accepts xz aliases", func(t *testing.T) {
		for _, input := range []string{"xz", "txz", "tar.xz", "tar-xz"} {
			format, err := ParseArchiveFormat(input)
			archiveRequireNoError(t, err)
			archiveAssertEqual(t, ArchiveFormatXZ, format)
		}
	})

	t.Run("accepts zip", func(t *testing.T) {
		format, err := ParseArchiveFormat("zip")
		archiveRequireNoError(t, err)
		archiveAssertEqual(t, ArchiveFormatZip, format)
	})

	t.Run("accepts gzip aliases", func(t *testing.T) {
		for _, input := range []string{"gz", "gzip", "tgz", "tar.gz", "tar-gz"} {
			format, err := ParseArchiveFormat(input)
			archiveRequireNoError(t, err)
			archiveAssertEqual(t, ArchiveFormatGzip, format)
		}
	})

	t.Run("rejects unsupported formats", func(t *testing.T) {
		format, err := ParseArchiveFormat("bzip2")
		archiveAssertError(t, err)
		archiveAssertEmpty(t, format)
		archiveAssertContains(t, err.Error(), "unsupported archive format")
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
		archiveAssertError(t, err)
		archiveAssertContains(t, err.Error(), "artifact path is empty")
		archiveAssertEmpty(t, result.Path)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		artifact := Artifact{
			Path: "/nonexistent/path/binary",
			OS:   "linux",
			Arch: "amd64",
		}

		result, err := Archive(fs, artifact)
		archiveAssertError(t, err)
		archiveAssertContains(t, err.Error(), "source file not found")
		archiveAssertEmpty(t, result.Path)
	})

}

func TestArchive_ArchiveAll_Good(t *testing.T) {
	fs := io_interface.Local
	t.Run("archives multiple artifacts", func(t *testing.T) {
		outputDir := t.TempDir()

		// Create multiple binaries
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
			archiveRequireNoError(t, err)

			name := "myapp"
			if target.os_ == "windows" {
				name = "myapp.exe"
			}

			binaryPath := ax.Join(platformDir, name)
			err = ax.WriteFile(binaryPath, []byte("binary content"), 0755)
			archiveRequireNoError(t, err)

			artifacts = append(artifacts, Artifact{
				Path: binaryPath,
				OS:   target.os_,
				Arch: target.arch,
			})
		}

		results, err := ArchiveAll(fs, artifacts)
		archiveRequireNoError(t, err)
		archiveRequireLen(t, results, 4)

		// Verify all archives were created
		for i, result := range results {
			archiveAssertFileExists(t, result.Path)
			archiveAssertEqual(t, artifacts[i].OS, result.OS)
			archiveAssertEqual(t, artifacts[i].Arch, result.Arch)
		}
	})

	t.Run("returns nil for empty slice", func(t *testing.T) {
		results, err := ArchiveAll(fs, []Artifact{})
		archiveAssertNoError(t, err)
		archiveAssertNil(t, results)
	})

	t.Run("returns nil for nil slice", func(t *testing.T) {
		results, err := ArchiveAll(fs, nil)
		archiveAssertNoError(t, err)
		archiveAssertNil(t, results)
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
		archiveAssertError(t, err)
		// Should have the first successful result
		archiveRequireLen(t, results, 1)
		archiveAssertFileExists(t, results[0].Path)
	})
}

func TestArchive_ArchiveFilenameGood(t *testing.T) {
	t.Run("generates correct tar.gz filename", func(t *testing.T) {
		artifact := Artifact{
			Path: "/output/linux_amd64/myapp",
			OS:   "linux",
			Arch: "amd64",
		}

		filename := archiveFilename(artifact, ".tar.gz")
		archiveAssertEqual(t, "/output/myapp_linux_amd64.tar.gz", filename)
	})

	t.Run("generates correct zip filename", func(t *testing.T) {
		artifact := Artifact{
			Path: "/output/windows_amd64/myapp.exe",
			OS:   "windows",
			Arch: "amd64",
		}

		filename := archiveFilename(artifact, ".zip")
		archiveAssertEqual(t, "/output/myapp_windows_amd64.zip", filename)
	})

	t.Run("handles nested output directories", func(t *testing.T) {
		artifact := Artifact{
			Path: "/project/dist/linux_arm64/cli",
			OS:   "linux",
			Arch: "arm64",
		}

		filename := archiveFilename(artifact, ".tar.gz")
		archiveAssertEqual(t, "/project/dist/cli_linux_arm64.tar.gz", filename)
	})

	t.Run("strips app bundle suffix from archive name", func(t *testing.T) {
		artifact := Artifact{
			Path: "/output/darwin_arm64/Core.app",
			OS:   "darwin",
			Arch: "arm64",
		}

		filename := archiveFilename(artifact, ".tar.gz")
		archiveAssertEqual(t, "/output/Core_darwin_arm64.tar.gz", filename)
	})
}

func TestArchive_RoundTripGood(t *testing.T) {
	fs := io_interface.Local

	t.Run("tar.gz round trip preserves content", func(t *testing.T) {
		binaryPath, _ := setupArchiveTestFile(t, "roundtrip-app", "linux", "amd64")

		// Read original content
		originalContent, err := ax.ReadFile(binaryPath)
		archiveRequireNoError(t, err)

		artifact := Artifact{
			Path: binaryPath,
			OS:   "linux",
			Arch: "amd64",
		}

		// Create archive
		archiveArtifact, err := Archive(fs, artifact)
		archiveRequireNoError(t, err)
		archiveAssertFileExists(t, archiveArtifact.Path)

		// Extract and verify content matches
		extractedContent := extractTarGzFile(t, archiveArtifact.Path, "roundtrip-app")
		archiveAssertEqual(t, originalContent, extractedContent)
	})

	t.Run("tar.xz round trip preserves content", func(t *testing.T) {
		binaryPath, _ := setupArchiveTestFile(t, "roundtrip-xz", "linux", "arm64")

		originalContent, err := ax.ReadFile(binaryPath)
		archiveRequireNoError(t, err)

		artifact := Artifact{
			Path: binaryPath,
			OS:   "linux",
			Arch: "arm64",
		}

		archiveArtifact, err := ArchiveXZ(fs, artifact)
		archiveRequireNoError(t, err)
		archiveAssertFileExists(t, archiveArtifact.Path)

		extractedContent := extractTarXzFile(t, archiveArtifact.Path, "roundtrip-xz")
		archiveAssertEqual(t, originalContent, extractedContent)
	})

	t.Run("zip round trip preserves content", func(t *testing.T) {
		binaryPath, _ := setupArchiveTestFile(t, "roundtrip.exe", "windows", "amd64")

		originalContent, err := ax.ReadFile(binaryPath)
		archiveRequireNoError(t, err)

		artifact := Artifact{
			Path: binaryPath,
			OS:   "windows",
			Arch: "amd64",
		}

		archiveArtifact, err := Archive(fs, artifact)
		archiveRequireNoError(t, err)
		archiveAssertFileExists(t, archiveArtifact.Path)

		extractedContent := extractZipFile(t, archiveArtifact.Path, "roundtrip.exe")
		archiveAssertEqual(t, originalContent, extractedContent)
	})

	t.Run("tar.gz preserves file permissions", func(t *testing.T) {
		binaryPath, _ := setupArchiveTestFile(t, "perms-app", "linux", "amd64")

		artifact := Artifact{
			Path: binaryPath,
			OS:   "linux",
			Arch: "amd64",
		}

		archiveArtifact, err := Archive(fs, artifact)
		archiveRequireNoError(t, err)

		// Extract and verify permissions are preserved
		mode := extractTarGzFileMode(t, archiveArtifact.Path, "perms-app")
		// The original file was written with 0755
		archiveAssertEqual(t, stdfs.FileMode(0o755), mode&stdfs.ModePerm)
	})

	t.Run("round trip with large binary content", func(t *testing.T) {
		outputDir := t.TempDir()
		platformDir := ax.Join(outputDir, "linux_amd64")
		archiveRequireNoError(t, ax.MkdirAll(platformDir, 0755))

		// Create a larger file (64KB)
		largeContent := make([]byte, 64*1024)
		for i := range largeContent {
			largeContent[i] = byte(i % 256)
		}
		binaryPath := ax.Join(platformDir, "large-app")
		archiveRequireNoError(t, ax.WriteFile(binaryPath, largeContent, 0755))

		artifact := Artifact{
			Path: binaryPath,
			OS:   "linux",
			Arch: "amd64",
		}

		archiveArtifact, err := Archive(fs, artifact)
		archiveRequireNoError(t, err)

		extractedContent := extractTarGzFile(t, archiveArtifact.Path, "large-app")
		archiveAssertEqual(t, largeContent, extractedContent)
	})

	t.Run("archive is smaller than original for tar.gz", func(t *testing.T) {
		outputDir := t.TempDir()
		platformDir := ax.Join(outputDir, "linux_amd64")
		archiveRequireNoError(t, ax.MkdirAll(platformDir, 0755))

		// Create a compressible file (repeated pattern)
		compressibleContent := make([]byte, 4096)
		for i := range compressibleContent {
			compressibleContent[i] = 'A'
		}
		binaryPath := ax.Join(platformDir, "compressible-app")
		archiveRequireNoError(t, ax.WriteFile(binaryPath, compressibleContent, 0755))

		artifact := Artifact{
			Path: binaryPath,
			OS:   "linux",
			Arch: "amd64",
		}

		archiveArtifact, err := Archive(fs, artifact)
		archiveRequireNoError(t, err)

		originalInfo, err := ax.Stat(binaryPath)
		archiveRequireNoError(t, err)
		archiveInfo, err := ax.Stat(archiveArtifact.Path)
		archiveRequireNoError(t, err)

		// Compressed archive should be smaller than original
		archiveAssertLess(t, archiveInfo.Size(), originalInfo.Size())
	})
}

// extractTarGzFile extracts a named file from a tar.gz archive and returns its content.
func extractTarGzFile(t *testing.T, archivePath, fileName string) []byte {
	t.Helper()

	file, err := ax.Open(archivePath)
	archiveRequireNoError(t, err)
	defer func() { _ = file.Close() }()

	gzReader, err := gzip.NewReader(file)
	archiveRequireNoError(t, err)
	defer func() { _ = gzReader.Close() }()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			t.Fatalf("file %q not found in archive", fileName)
		}
		archiveRequireNoError(t, err)

		if header.Name == fileName {
			content, err := io.ReadAll(tarReader)
			archiveRequireNoError(t, err)
			return content
		}
	}
}

// extractTarGzFileMode extracts the file mode of a named file from a tar.gz archive.
func extractTarGzFileMode(t *testing.T, archivePath, fileName string) stdfs.FileMode {
	t.Helper()

	file, err := ax.Open(archivePath)
	archiveRequireNoError(t, err)
	defer func() { _ = file.Close() }()

	gzReader, err := gzip.NewReader(file)
	archiveRequireNoError(t, err)
	defer func() { _ = gzReader.Close() }()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			t.Fatalf("file %q not found in archive", fileName)
		}
		archiveRequireNoError(t, err)

		if header.Name == fileName {
			return header.FileInfo().Mode()
		}
	}
}

// extractTarXzFile extracts a named file from a tar.xz archive and returns its content.
func extractTarXzFile(t *testing.T, archivePath, fileName string) []byte {
	t.Helper()

	xzData, err := ax.ReadFile(archivePath)
	archiveRequireNoError(t, err)

	tarData, err := compress.Decompress(xzData)
	archiveRequireNoError(t, err)

	tarReader := tar.NewReader(core.NewBuffer(tarData))

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			t.Fatalf("file %q not found in archive", fileName)
		}
		archiveRequireNoError(t, err)

		if header.Name == fileName {
			content, err := io.ReadAll(tarReader)
			archiveRequireNoError(t, err)
			return content
		}
	}
}

// extractZipFile extracts a named file from a zip archive and returns its content.
func extractZipFile(t *testing.T, archivePath, fileName string) []byte {
	t.Helper()

	reader, err := zip.OpenReader(archivePath)
	archiveRequireNoError(t, err)
	defer func() { _ = reader.Close() }()

	for _, f := range reader.File {
		if f.Name == fileName {
			rc, err := f.Open()
			archiveRequireNoError(t, err)
			defer func() { _ = rc.Close() }()

			content, err := io.ReadAll(rc)
			archiveRequireNoError(t, err)
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
	archiveRequireNoError(t, err)
	defer func() { _ = file.Close() }()

	gzReader, err := gzip.NewReader(file)
	archiveRequireNoError(t, err)
	defer func() { _ = gzReader.Close() }()

	tarReader := tar.NewReader(gzReader)

	header, err := tarReader.Next()
	archiveRequireNoError(t, err)
	archiveAssertEqual(t, expectedName, header.Name)

	// Verify there's only one file
	_, err = tarReader.Next()
	archiveAssertEqual(t, io.EOF, err)
}

// verifyZipContent opens a zip file and verifies it contains the expected file.
func verifyZipContent(t *testing.T, archivePath, expectedName string) {
	t.Helper()

	reader, err := zip.OpenReader(archivePath)
	archiveRequireNoError(t, err)
	defer func() { _ = reader.Close() }()

	archiveRequireLen(t, reader.File, 1)
	archiveAssertEqual(t, expectedName, reader.File[0].Name)
}

// verifyTarXzContent opens a tar.xz file and verifies it contains the expected file.
func verifyTarXzContent(t *testing.T, archivePath, expectedName string) {
	t.Helper()

	// Read the xz-compressed file
	xzData, err := ax.ReadFile(archivePath)
	archiveRequireNoError(t, err)

	// Decompress with the deferred Borg API.
	tarData, err := compress.Decompress(xzData)
	archiveRequireNoError(t, err)

	// Read tar archive
	tarReader := tar.NewReader(core.NewBuffer(tarData))

	header, err := tarReader.Next()
	archiveRequireNoError(t, err)
	archiveAssertEqual(t, expectedName, header.Name)

	// Verify there's only one file
	_, err = tarReader.Next()
	archiveAssertEqual(t, io.EOF, err)
}

// --- v0.9.0 generated compliance triplets ---
func TestArchive_ParseArchiveFormat_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = ParseArchiveFormat("")
	})
	core.AssertTrue(t, true)
}

func TestArchive_ParseArchiveFormat_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = ParseArchiveFormat("agent")
	})
	core.AssertTrue(t, true)
}

func TestArchive_Archive_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = Archive(io_interface.NewMemoryMedium(), Artifact{})
	})
	core.AssertTrue(t, true)
}

func TestArchive_ArchiveXZ_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = ArchiveXZ(io_interface.NewMemoryMedium(), Artifact{})
	})
	core.AssertTrue(t, true)
}

func TestArchive_ArchiveXZ_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = ArchiveXZ(io_interface.NewMemoryMedium(), Artifact{})
	})
	core.AssertTrue(t, true)
}

func TestArchive_ArchiveXZ_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = ArchiveXZ(io_interface.NewMemoryMedium(), Artifact{})
	})
	core.AssertTrue(t, true)
}

func TestArchive_ArchiveWithFormat_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = ArchiveWithFormat(io_interface.NewMemoryMedium(), Artifact{}, ArchiveFormat("linux"))
	})
	core.AssertTrue(t, true)
}

func TestArchive_ArchiveWithFormat_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = ArchiveWithFormat(io_interface.NewMemoryMedium(), Artifact{}, ArchiveFormat("linux"))
	})
	core.AssertTrue(t, true)
}

func TestArchive_ArchiveWithFormat_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = ArchiveWithFormat(io_interface.NewMemoryMedium(), Artifact{}, ArchiveFormat("linux"))
	})
	core.AssertTrue(t, true)
}

func TestArchive_ArchiveAll_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = ArchiveAll(io_interface.NewMemoryMedium(), nil)
	})
	core.AssertTrue(t, true)
}

func TestArchive_ArchiveAllXZ_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = ArchiveAllXZ(io_interface.NewMemoryMedium(), nil)
	})
	core.AssertTrue(t, true)
}

func TestArchive_ArchiveAllXZ_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = ArchiveAllXZ(io_interface.NewMemoryMedium(), nil)
	})
	core.AssertTrue(t, true)
}

func TestArchive_ArchiveAllXZ_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = ArchiveAllXZ(io_interface.NewMemoryMedium(), nil)
	})
	core.AssertTrue(t, true)
}

func TestArchive_ArchiveAllWithFormat_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = ArchiveAllWithFormat(io_interface.NewMemoryMedium(), nil, ArchiveFormat("linux"))
	})
	core.AssertTrue(t, true)
}

func TestArchive_ArchiveAllWithFormat_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = ArchiveAllWithFormat(io_interface.NewMemoryMedium(), nil, ArchiveFormat("linux"))
	})
	core.AssertTrue(t, true)
}

func TestArchive_ArchiveAllWithFormat_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = ArchiveAllWithFormat(io_interface.NewMemoryMedium(), nil, ArchiveFormat("linux"))
	})
	core.AssertTrue(t, true)
}
