package build

import (
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/io"
)

// setupChecksumTestFile creates a test file with known content.
func setupChecksumTestFile(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := ax.Join(dir, "testfile")
	result := ax.WriteFile(path, []byte(content), 0644)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	return path
}

func requireChecksumArtifact(t *testing.T, result core.Result) Artifact {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	return result.Value.(Artifact)
}

func requireChecksumArtifacts(t *testing.T, result core.Result) []Artifact {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result.Value == nil {
		return nil
	}
	return result.Value.([]Artifact)
}

func requireChecksumOK(t *testing.T, result core.Result) {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
}

func requireChecksumBytes(t *testing.T, result core.Result) []byte {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	return result.Value.([]byte)
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

		result := requireChecksumArtifact(t, Checksum(fs, artifact))
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

		result := requireChecksumArtifact(t, Checksum(fs, artifact))
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

		result := requireChecksumArtifact(t, Checksum(fs, artifact))
		if len(result.Checksum) != 64 {
			t.Fatalf("want len %v, got %v", 64, len(result.Checksum))
		}

	})

	t.Run("different content produces different checksums", func(t *testing.T) {
		path1 := setupChecksumTestFile(t, "content one")
		path2 := setupChecksumTestFile(t, "content two")

		result1 := requireChecksumArtifact(t, Checksum(fs, Artifact{Path: path1, OS: "linux", Arch: "amd64"}))

		result2 := requireChecksumArtifact(t, Checksum(fs, Artifact{Path: path2, OS: "linux", Arch: "amd64"}))
		if stdlibAssertEqual(result1.Checksum, result2.Checksum) {
			t.Fatalf("did not want %v", result2.Checksum)
		}

	})

	t.Run("same content produces same checksum", func(t *testing.T) {
		content := "identical content"
		path1 := setupChecksumTestFile(t, content)
		path2 := setupChecksumTestFile(t, content)

		result1 := requireChecksumArtifact(t, Checksum(fs, Artifact{Path: path1, OS: "linux", Arch: "amd64"}))

		result2 := requireChecksumArtifact(t, Checksum(fs, Artifact{Path: path2, OS: "linux", Arch: "amd64"}))
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

		result := Checksum(fs, artifact)
		if result.OK {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(result.Error(), "artifact path is empty") {
			t.Fatalf("expected %v to contain %v", result.Error(), "artifact path is empty")
		}

	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		artifact := Artifact{
			Path: "/nonexistent/path/file",
			OS:   "linux",
			Arch: "amd64",
		}

		result := Checksum(fs, artifact)
		if result.OK {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(result.Error(), "failed to open file") {
			t.Fatalf("expected %v to contain %v", result.Error(), "failed to open file")
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

		results := requireChecksumArtifacts(t, ChecksumAll(fs, artifacts))
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
		results := requireChecksumArtifacts(t, ChecksumAll(fs, []Artifact{}))
		if !stdlibAssertNil(results) {
			t.Fatalf("expected nil, got %v", results)
		}

	})

	t.Run("returns nil for nil slice", func(t *testing.T) {
		results := requireChecksumArtifacts(t, ChecksumAll(fs, nil))
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

		result := ChecksumAll(fs, artifacts)
		if result.OK {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(result.Error(), "failed to checksum") {
			t.Fatalf("expected %v to contain failed to checksum", result.Error())
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

		requireChecksumOK(t, WriteChecksumFile(fs, artifacts, checksumPath))

		content := requireChecksumBytes(t, ax.ReadFile(checksumPath))

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

		requireChecksumOK(t, WriteChecksumFile(fs, artifacts, checksumPath))
		if result := ax.Stat(checksumPath); !result.OK {
			t.Fatalf("expected file to exist: %v", checksumPath)
		}

	})

	t.Run("does nothing for empty artifacts", func(t *testing.T) {
		dir := t.TempDir()
		checksumPath := ax.Join(dir, "CHECKSUMS.txt")

		requireChecksumOK(t, WriteChecksumFile(fs, []Artifact{}, checksumPath))
		if ax.Exists(checksumPath) {
			t.Fatal("expected false")
		}

	})

	t.Run("does nothing for nil artifacts", func(t *testing.T) {
		dir := t.TempDir()
		checksumPath := ax.Join(dir, "CHECKSUMS.txt")

		requireChecksumOK(t, WriteChecksumFile(fs, nil, checksumPath))

	})

	t.Run("uses only basename for filenames", func(t *testing.T) {
		dir := t.TempDir()
		checksumPath := ax.Join(dir, "CHECKSUMS.txt")

		artifacts := []Artifact{
			{Path: "/some/deep/nested/path/myapp_linux_amd64.tar.gz", Checksum: "checksum123", OS: "linux", Arch: "amd64"},
		}

		requireChecksumOK(t, WriteChecksumFile(fs, artifacts, checksumPath))

		content := requireChecksumBytes(t, ax.ReadFile(checksumPath))
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
		requireChecksumOK(t, ax.MkdirAll(ax.Dir(artifactPath), 0o755))

		artifacts := []Artifact{
			{Path: artifactPath, Checksum: "checksum123", OS: "linux", Arch: "amd64"},
		}

		requireChecksumOK(t, WriteChecksumFile(fs, artifacts, checksumPath))

		content := requireChecksumBytes(t, ax.ReadFile(checksumPath))
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

		result := WriteChecksumFile(fs, artifacts, checksumPath)
		if result.OK {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(result.Error(), "has no checksum") {
			t.Fatalf("expected %v to contain %v", result.Error(), "has no checksum")
		}

	})
}

// --- v0.9.0 generated compliance triplets ---
func TestChecksum_Checksum_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Checksum(io.NewMemoryMedium(), Artifact{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestChecksum_ChecksumAll_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ChecksumAll(io.NewMemoryMedium(), nil)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestChecksum_WriteChecksumFile_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WriteChecksumFile(io.NewMemoryMedium(), nil, core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
