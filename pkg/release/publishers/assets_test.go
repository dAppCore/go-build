package publishers

import (
	"testing"

	"dappco.re/go/build/pkg/build"
	"dappco.re/go/core/io"
)

func TestAssets_BuildChecksumMap_ParsesRFCArchiveNames_Good(t *testing.T) {
	artifacts := []build.Artifact{
		{Path: "/dist/myapp_darwin_amd64.tar.gz", Checksum: "darwin-amd64"},
		{Path: "/dist/myapp_darwin_arm64.tar.gz", Checksum: "darwin-arm64"},
		{Path: "/dist/myapp_linux_amd64.tar.gz", Checksum: "linux-amd64"},
		{Path: "/dist/myapp_linux_arm64.tar.gz", Checksum: "linux-arm64"},
		{Path: "/dist/myapp_windows_amd64.zip", Checksum: "windows-amd64"},
		{Path: "/dist/myapp_windows_arm64.zip", Checksum: "windows-arm64"},
		{Path: "/dist/CHECKSUMS.txt"},
	}

	checksums := buildChecksumMap(artifacts)
	if !stdlibAssertEqual("darwin-amd64", checksums.DarwinAmd64) {
		t.Fatalf("want %v, got %v", "darwin-amd64", checksums.DarwinAmd64)
	}
	if !stdlibAssertEqual("darwin-arm64", checksums.DarwinArm64) {
		t.Fatalf("want %v, got %v", "darwin-arm64", checksums.DarwinArm64)
	}
	if !stdlibAssertEqual("linux-amd64", checksums.LinuxAmd64) {
		t.Fatalf("want %v, got %v", "linux-amd64", checksums.LinuxAmd64)
	}
	if !stdlibAssertEqual("linux-arm64", checksums.LinuxArm64) {
		t.Fatalf("want %v, got %v", "linux-arm64", checksums.LinuxArm64)
	}
	if !stdlibAssertEqual("windows-amd64", checksums.WindowsAmd64) {
		t.Fatalf("want %v, got %v", "windows-amd64", checksums.WindowsAmd64)
	}
	if !stdlibAssertEqual("windows-arm64", checksums.WindowsArm64) {
		t.Fatalf("want %v, got %v", "windows-arm64", checksums.WindowsArm64)
	}
	if !stdlibAssertEqual("myapp_darwin_amd64.tar.gz", checksums.DarwinAmd64File) {
		t.Fatalf("want %v, got %v", "myapp_darwin_amd64.tar.gz", checksums.DarwinAmd64File)
	}
	if !stdlibAssertEqual("myapp_darwin_arm64.tar.gz", checksums.DarwinArm64File) {
		t.Fatalf("want %v, got %v", "myapp_darwin_arm64.tar.gz", checksums.DarwinArm64File)
	}
	if !stdlibAssertEqual("myapp_linux_amd64.tar.gz", checksums.LinuxAmd64File) {
		t.Fatalf("want %v, got %v", "myapp_linux_amd64.tar.gz", checksums.LinuxAmd64File)
	}
	if !stdlibAssertEqual("myapp_linux_arm64.tar.gz", checksums.LinuxArm64File) {
		t.Fatalf("want %v, got %v", "myapp_linux_arm64.tar.gz", checksums.LinuxArm64File)
	}
	if !stdlibAssertEqual("myapp_windows_amd64.zip", checksums.WindowsAmd64File) {
		t.Fatalf("want %v, got %v", "myapp_windows_amd64.zip", checksums.WindowsAmd64File)
	}
	if !stdlibAssertEqual("myapp_windows_arm64.zip", checksums.WindowsArm64File) {
		t.Fatalf("want %v, got %v", "myapp_windows_arm64.zip", checksums.WindowsArm64File)
	}
	if !stdlibAssertEqual("CHECKSUMS.txt", checksums.ChecksumFile) {
		t.Fatalf("want %v, got %v", "CHECKSUMS.txt", checksums.ChecksumFile)
	}

}

func TestAssets_BuildChecksumMapFromRelease_UsesChecksumFileFallback_Good(t *testing.T) {
	artifactFS := io.NewMemoryMedium()
	if err := artifactFS.Write("releases/checksums.txt", ""+"abc123  myapp_linux_amd64.tar.gz\n"+"def456  myapp_darwin_arm64.tar.gz\n"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	release := &Release{
		Artifacts: []build.Artifact{
			{Path: "releases/myapp_linux_amd64.tar.gz", OS: "linux", Arch: "amd64"},
			{Path: "releases/myapp_darwin_arm64.tar.gz"},
			{Path: "releases/checksums.txt"},
		},
		FS:         artifactFS,
		ArtifactFS: artifactFS,
	}

	checksums := buildChecksumMapFromRelease(release)
	if !stdlibAssertEqual("abc123", checksums.LinuxAmd64) {
		t.Fatalf("want %v, got %v", "abc123", checksums.LinuxAmd64)
	}
	if !stdlibAssertEqual("def456", checksums.DarwinArm64) {
		t.Fatalf("want %v, got %v", "def456", checksums.DarwinArm64)
	}
	if !stdlibAssertEqual("myapp_linux_amd64.tar.gz", checksums.LinuxAmd64File) {
		t.Fatalf("want %v, got %v", "myapp_linux_amd64.tar.gz", checksums.LinuxAmd64File)
	}
	if !stdlibAssertEqual("myapp_darwin_arm64.tar.gz", checksums.DarwinArm64File) {
		t.Fatalf("want %v, got %v", "myapp_darwin_arm64.tar.gz", checksums.DarwinArm64File)
	}
	if !stdlibAssertEqual("checksums.txt", checksums.ChecksumFile) {
		t.Fatalf("want %v, got %v", "checksums.txt", checksums.ChecksumFile)
	}

}
