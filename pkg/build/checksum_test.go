package build

import (
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/core"
	"dappco.re/go/core/io"
	"os"
)

// setupChecksumTestFile creates a test file with known content.
func setupChecksumTestFile(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := ax.Join(dir, "testfile")
	err := ax.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return path
}

func TestChecksum_Checksum_Good(t *testing.T) {
	fs := io.Local
	t.Run("computes SHA256 checksum", func(t *testing.T) {
		// Known SHA256 of "Hello, World!\n"
		path := setupChecksumTestFile(t, "Hello, World!\n")
		expectedChecksum := "c98c24b677eff44860afea6f493bbaec5bb1c4cbb209c6fc2bbb47f66ff2ad31"

		artifact := Artifact{
			Path: path,
			OS:   "linux",
			Arch: "amd64",
		}

		result, err := Checksum(fs, artifact)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(expectedChecksum, result.Checksum) {
			t.Fatalf("want %v, got %v", expectedChecksum, result.Checksum)
		}

	})

	t.Run("preserves artifact fields", func(t *testing.T) {
		path := setupChecksumTestFile(t, "test content")

		artifact := Artifact{
			Path: path,
			OS:   "darwin",
			Arch: "arm64",
		}

		result, err := Checksum(fs, artifact)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(path, result.Path) {
			t.Fatalf("want %v, got %v", path, result.Path)
		}
		if !stdlibAssertEqual("darwin", result.OS) {
			t.Fatalf("want %v, got %v", "darwin", result.OS)
		}
		if !stdlibAssertEqual("arm64", result.Arch) {
			t.Fatalf("want %v, got %v", "arm64", result.Arch)
		}
		if stdlibAssertEmpty(result.Checksum) {
			t.Fatal("expected non-empty")
		}

	})

	t.Run("produces 64 character hex string", func(t *testing.T) {
		path := setupChecksumTestFile(t, "any content")

		artifact := Artifact{Path: path, OS: "linux", Arch: "amd64"}

		result, err := Checksum(fs, artifact)
		if err != nil {
			t.Fatalf("unexpected error: %v",

				// SHA256 produces 32 bytes = 64 hex characters
				err)
		}
		if len(result.Checksum) != 64 {
			t.Fatalf("want len %v, got %v", 64, len(result.Checksum))
		}

	})

	t.Run("different content produces different checksums", func(t *testing.T) {
		path1 := setupChecksumTestFile(t, "content one")
		path2 := setupChecksumTestFile(t, "content two")

		result1, err := Checksum(fs, Artifact{Path: path1, OS: "linux", Arch: "amd64"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result2, err := Checksum(fs, Artifact{Path: path2, OS: "linux", Arch: "amd64"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertEqual(result1.Checksum, result2.Checksum) {
			t.Fatalf("did not want %v", result2.Checksum)
		}

	})

	t.Run("same content produces same checksum", func(t *testing.T) {
		content := "identical content"
		path1 := setupChecksumTestFile(t, content)
		path2 := setupChecksumTestFile(t, content)

		result1, err := Checksum(fs, Artifact{Path: path1, OS: "linux", Arch: "amd64"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result2, err := Checksum(fs, Artifact{Path: path2, OS: "linux", Arch: "amd64"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(result1.Checksum, result2.Checksum) {
			t.Fatalf("want %v, got %v", result1.Checksum, result2.Checksum)
		}

	})
}

func TestChecksum_Checksum_Bad(t *testing.T) {
	fs := io.Local
	t.Run("returns error for empty path", func(t *testing.T) {
		artifact := Artifact{
			Path: "",
			OS:   "linux",
			Arch: "amd64",
		}

		result, err := Checksum(fs, artifact)
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "artifact path is empty") {
			t.Fatalf("expected %v to contain %v", err.Error(), "artifact path is empty")
		}
		if !stdlibAssertEmpty(result.Checksum) {
			t.Fatalf("expected empty, got %v", result.Checksum)
		}

	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		artifact := Artifact{
			Path: "/nonexistent/path/file",
			OS:   "linux",
			Arch: "amd64",
		}

		result, err := Checksum(fs, artifact)
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "failed to open file") {
			t.Fatalf("expected %v to contain %v", err.Error(), "failed to open file")
		}
		if !stdlibAssertEmpty(result.Checksum) {
			t.Fatalf("expected empty, got %v", result.Checksum)
		}

	})
}

func TestChecksum_ChecksumAll_Good(t *testing.T) {
	fs := io.Local
	t.Run("checksums multiple artifacts", func(t *testing.T) {
		paths := []string{
			setupChecksumTestFile(t, "content one"),
			setupChecksumTestFile(t, "content two"),
			setupChecksumTestFile(t, "content three"),
		}

		artifacts := []Artifact{
			{Path: paths[0], OS: "linux", Arch: "amd64"},
			{Path: paths[1], OS: "darwin", Arch: "arm64"},
			{Path: paths[2], OS: "windows", Arch: "amd64"},
		}

		results, err := ChecksumAll(fs, artifacts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 3 {
			t.Fatalf("want len %v, got %v", 3, len(results))
		}

		for i, result := range results {
			if !stdlibAssertEqual(artifacts[i].Path, result.Path) {
				t.Fatalf("want %v, got %v", artifacts[i].Path, result.Path)
			}
			if !stdlibAssertEqual(artifacts[i].OS, result.OS) {
				t.Fatalf("want %v, got %v", artifacts[i].OS, result.OS)
			}
			if !stdlibAssertEqual(artifacts[i].Arch, result.Arch) {
				t.Fatalf("want %v, got %v", artifacts[i].Arch, result.Arch)
			}
			if stdlibAssertEmpty(result.Checksum) {
				t.Fatal("expected non-empty")
			}

		}
	})

	t.Run("returns nil for empty slice", func(t *testing.T) {
		results, err := ChecksumAll(fs, []Artifact{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertNil(results) {
			t.Fatalf("expected nil, got %v", results)
		}

	})

	t.Run("returns nil for nil slice", func(t *testing.T) {
		results, err := ChecksumAll(fs, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertNil(results) {
			t.Fatalf("expected nil, got %v", results)
		}

	})
}

func TestChecksum_ChecksumAll_Bad(t *testing.T) {
	fs := io.Local
	t.Run("returns partial results on error", func(t *testing.T) {
		path := setupChecksumTestFile(t, "valid content")

		artifacts := []Artifact{
			{Path: path, OS: "linux", Arch: "amd64"},
			{Path: "/nonexistent/file", OS: "linux", Arch: "arm64"}, // This will fail
		}

		results, err := ChecksumAll(fs, artifacts)
		if err == nil {
			t.Fatal("expected error")

			// Should have the first successful result
		}
		if len(results) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(results))
		}
		if stdlibAssertEmpty(results[0].Checksum) {
			t.Fatal("expected non-empty")
		}

	})
}

func TestChecksum_WriteChecksumFile_Good(t *testing.T) {
	fs := io.Local
	t.Run("writes checksum file with correct format", func(t *testing.T) {
		dir := t.TempDir()
		checksumPath := ax.Join(dir, "CHECKSUMS.txt")

		artifacts := []Artifact{
			{Path: "/output/app_linux_amd64.tar.gz", Checksum: "abc123def456", OS: "linux", Arch: "amd64"},
			{Path: "/output/app_darwin_arm64.tar.gz", Checksum: "789xyz000111", OS: "darwin", Arch: "arm64"},
		}

		err := WriteChecksumFile(fs, artifacts, checksumPath)
		if err != nil {
			t.Fatalf("unexpected error: %v",

				// Read and verify content
				err)
		}

		content, err := ax.ReadFile(checksumPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		lines := core.Split(core.Trim(string(content)), "\n")
		if len(lines) != 2 {
			t.Fatalf("want len %v, got %v",

				// Lines should be sorted alphabetically
				2, len(lines))
		}
		if !stdlibAssertEqual("789xyz000111  app_darwin_arm64.tar.gz", lines[0]) {
			t.Fatalf("want %v, got %v", "789xyz000111  app_darwin_arm64.tar.gz", lines[0])
		}
		if !stdlibAssertEqual("abc123def456  app_linux_amd64.tar.gz", lines[1]) {
			t.Fatalf("want %v, got %v", "abc123def456  app_linux_amd64.tar.gz", lines[1])
		}

	})

	t.Run("creates parent directories", func(t *testing.T) {
		dir := t.TempDir()
		checksumPath := ax.Join(dir, "nested", "deep", "CHECKSUMS.txt")

		artifacts := []Artifact{
			{Path: "/output/app.tar.gz", Checksum: "abc123", OS: "linux", Arch: "amd64"},
		}

		err := WriteChecksumFile(fs, artifacts, checksumPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := os.Stat(checksumPath); err != nil {
			t.Fatalf("expected file to exist: %v", checksumPath)
		}

	})

	t.Run("does nothing for empty artifacts", func(t *testing.T) {
		dir := t.TempDir()
		checksumPath := ax.Join(dir, "CHECKSUMS.txt")

		err := WriteChecksumFile(fs, []Artifact{}, checksumPath)
		if err != nil {
			t.Fatalf("unexpected error: %v",

				// File should not exist
				err)
		}
		if ax.Exists(checksumPath) {
			t.Fatal("expected false")
		}

	})

	t.Run("does nothing for nil artifacts", func(t *testing.T) {
		dir := t.TempDir()
		checksumPath := ax.Join(dir, "CHECKSUMS.txt")

		err := WriteChecksumFile(fs, nil, checksumPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

	})

	t.Run("uses only basename for filenames", func(t *testing.T) {
		dir := t.TempDir()
		checksumPath := ax.Join(dir, "CHECKSUMS.txt")

		artifacts := []Artifact{
			{Path: "/some/deep/nested/path/myapp_linux_amd64.tar.gz", Checksum: "checksum123", OS: "linux", Arch: "amd64"},
		}

		err := WriteChecksumFile(fs, artifacts, checksumPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := ax.ReadFile(checksumPath)
		if err != nil {
			t.Fatalf("unexpected error: %v",

				// Should only contain the basename
				err)
		}
		if !stdlibAssertContains(string(content), "myapp_linux_amd64.tar.gz") {
			t.Fatalf("expected %v to contain %v", string(content), "myapp_linux_amd64.tar.gz")
		}
		if stdlibAssertContains(string(content), "/some/deep/nested/path/") {
			t.Fatalf("expected %v not to contain %v", string(content), "/some/deep/nested/path/")
		}

	})

	t.Run("uses relative paths for nested artifacts inside the output tree", func(t *testing.T) {
		dir := t.TempDir()
		checksumPath := ax.Join(dir, "CHECKSUMS.txt")
		artifactPath := ax.Join(dir, "go", "myapp_linux_amd64.tar.gz")
		if err := ax.MkdirAll(ax.Dir(artifactPath), 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		artifacts := []Artifact{
			{Path: artifactPath, Checksum: "checksum123", OS: "linux", Arch: "amd64"},
		}

		err := WriteChecksumFile(fs, artifacts, checksumPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := ax.ReadFile(checksumPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(string(content), "go/myapp_linux_amd64.tar.gz") {
			t.Fatalf("expected %v to contain %v", string(content), "go/myapp_linux_amd64.tar.gz")
		}

	})
}

func TestChecksum_WriteChecksumFile_Bad(t *testing.T) {
	fs := io.Local
	t.Run("returns error for artifact without checksum", func(t *testing.T) {
		dir := t.TempDir()
		checksumPath := ax.Join(dir, "CHECKSUMS.txt")

		artifacts := []Artifact{
			{Path: "/output/app.tar.gz", Checksum: "", OS: "linux", Arch: "amd64"}, // No checksum
		}

		err := WriteChecksumFile(fs, artifacts, checksumPath)
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "has no checksum") {
			t.Fatalf("expected %v to contain %v", err.Error(), "has no checksum")
		}

	})
}
