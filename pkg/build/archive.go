// Package build provides project type detection and cross-compilation for the Core build system.
package build

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	io_interface "forge.lthn.ai/core/go-io"
	coreerr "forge.lthn.ai/core/go-log"
	"github.com/Snider/Borg/pkg/compress"
)

// ArchiveFormat specifies the compression format for archives.
type ArchiveFormat string

const (
	// ArchiveFormatGzip uses tar.gz (gzip compression) - widely compatible.
	ArchiveFormatGzip ArchiveFormat = "gz"
	// ArchiveFormatXZ uses tar.xz (xz/LZMA2 compression) - better compression ratio.
	ArchiveFormatXZ ArchiveFormat = "xz"
	// ArchiveFormatZip uses zip - for Windows.
	ArchiveFormatZip ArchiveFormat = "zip"
)

// Archive creates an archive for a single artifact using gzip compression.
// Uses tar.gz for linux/darwin and zip for windows.
// The archive is created alongside the binary (e.g., dist/myapp_linux_amd64.tar.gz).
// Returns a new Artifact with Path pointing to the archive.
func Archive(fs io_interface.Medium, artifact Artifact) (Artifact, error) {
	return ArchiveWithFormat(fs, artifact, ArchiveFormatGzip)
}

// ArchiveXZ creates an archive for a single artifact using xz compression.
// Uses tar.xz for linux/darwin and zip for windows.
// Returns a new Artifact with Path pointing to the archive.
func ArchiveXZ(fs io_interface.Medium, artifact Artifact) (Artifact, error) {
	return ArchiveWithFormat(fs, artifact, ArchiveFormatXZ)
}

// ArchiveWithFormat creates an archive for a single artifact with the specified format.
// Uses tar.gz or tar.xz for linux/darwin and zip for windows.
// The archive is created alongside the binary (e.g., dist/myapp_linux_amd64.tar.xz).
// Returns a new Artifact with Path pointing to the archive.
func ArchiveWithFormat(fs io_interface.Medium, artifact Artifact, format ArchiveFormat) (Artifact, error) {
	if artifact.Path == "" {
		return Artifact{}, coreerr.E("build.Archive", "artifact path is empty", nil)
	}

	// Verify the source file exists
	info, err := fs.Stat(artifact.Path)
	if err != nil {
		return Artifact{}, coreerr.E("build.Archive", "source file not found", err)
	}
	if info.IsDir() {
		return Artifact{}, coreerr.E("build.Archive", "source path is a directory, expected file", nil)
	}

	// Determine archive type based on OS and format
	var archivePath string
	var archiveFunc func(fs io_interface.Medium, src, dst string) error

	if artifact.OS == "windows" {
		archivePath = archiveFilename(artifact, ".zip")
		archiveFunc = createZipArchive
	} else {
		switch format {
		case ArchiveFormatXZ:
			archivePath = archiveFilename(artifact, ".tar.xz")
			archiveFunc = createTarXzArchive
		default:
			archivePath = archiveFilename(artifact, ".tar.gz")
			archiveFunc = createTarGzArchive
		}
	}

	// Create the archive
	if err := archiveFunc(fs, artifact.Path, archivePath); err != nil {
		return Artifact{}, coreerr.E("build.Archive", "failed to create archive", err)
	}

	return Artifact{
		Path:     archivePath,
		OS:       artifact.OS,
		Arch:     artifact.Arch,
		Checksum: artifact.Checksum,
	}, nil
}

// ArchiveAll archives all artifacts using gzip compression.
// Returns a slice of new artifacts pointing to the archives.
func ArchiveAll(fs io_interface.Medium, artifacts []Artifact) ([]Artifact, error) {
	return ArchiveAllWithFormat(fs, artifacts, ArchiveFormatGzip)
}

// ArchiveAllXZ archives all artifacts using xz compression.
// Returns a slice of new artifacts pointing to the archives.
func ArchiveAllXZ(fs io_interface.Medium, artifacts []Artifact) ([]Artifact, error) {
	return ArchiveAllWithFormat(fs, artifacts, ArchiveFormatXZ)
}

// ArchiveAllWithFormat archives all artifacts with the specified format.
// Returns a slice of new artifacts pointing to the archives.
func ArchiveAllWithFormat(fs io_interface.Medium, artifacts []Artifact, format ArchiveFormat) ([]Artifact, error) {
	if len(artifacts) == 0 {
		return nil, nil
	}

	var archived []Artifact
	for _, artifact := range artifacts {
		arch, err := ArchiveWithFormat(fs, artifact, format)
		if err != nil {
			return archived, coreerr.E("build.ArchiveAll", "failed to archive "+artifact.Path, err)
		}
		archived = append(archived, arch)
	}

	return archived, nil
}

// archiveFilename generates the archive filename based on the artifact and extension.
// Format: dist/myapp_linux_amd64.tar.gz (binary name taken from artifact path).
func archiveFilename(artifact Artifact, ext string) string {
	// Get the directory containing the binary (e.g., dist/linux_amd64)
	dir := filepath.Dir(artifact.Path)
	// Go up one level to the output directory (e.g., dist)
	outputDir := filepath.Dir(dir)

	// Get the binary name without extension
	binaryName := filepath.Base(artifact.Path)
	binaryName = strings.TrimSuffix(binaryName, ".exe")

	// Construct archive name: myapp_linux_amd64.tar.gz
	archiveName := fmt.Sprintf("%s_%s_%s%s", binaryName, artifact.OS, artifact.Arch, ext)

	return filepath.Join(outputDir, archiveName)
}

// createTarXzArchive creates a tar.xz archive containing a single file.
// Uses Borg's compress package for xz compression.
func createTarXzArchive(fs io_interface.Medium, src, dst string) error {
	// Open the source file
	srcFile, err := fs.Open(src)
	if err != nil {
		return coreerr.E("build.createTarXzArchive", "failed to open source file", err)
	}
	defer func() { _ = srcFile.Close() }()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return coreerr.E("build.createTarXzArchive", "failed to stat source file", err)
	}

	// Create tar archive in memory
	var tarBuf bytes.Buffer
	tarWriter := tar.NewWriter(&tarBuf)

	// Create tar header
	header, err := tar.FileInfoHeader(srcInfo, "")
	if err != nil {
		return coreerr.E("build.createTarXzArchive", "failed to create tar header", err)
	}
	header.Name = filepath.Base(src)

	if err := tarWriter.WriteHeader(header); err != nil {
		return coreerr.E("build.createTarXzArchive", "failed to write tar header", err)
	}

	if _, err := io.Copy(tarWriter, srcFile); err != nil {
		return coreerr.E("build.createTarXzArchive", "failed to write file content to tar", err)
	}

	if err := tarWriter.Close(); err != nil {
		return coreerr.E("build.createTarXzArchive", "failed to close tar writer", err)
	}

	// Compress with xz using Borg
	xzData, err := compress.Compress(tarBuf.Bytes(), "xz")
	if err != nil {
		return coreerr.E("build.createTarXzArchive", "failed to compress with xz", err)
	}

	// Write to destination file
	dstFile, err := fs.Create(dst)
	if err != nil {
		return coreerr.E("build.createTarXzArchive", "failed to create archive file", err)
	}
	defer func() { _ = dstFile.Close() }()

	if _, err := dstFile.Write(xzData); err != nil {
		return coreerr.E("build.createTarXzArchive", "failed to write archive file", err)
	}

	return nil
}

// createTarGzArchive creates a tar.gz archive containing a single file.
func createTarGzArchive(fs io_interface.Medium, src, dst string) error {
	// Open the source file
	srcFile, err := fs.Open(src)
	if err != nil {
		return coreerr.E("build.createTarGzArchive", "failed to open source file", err)
	}
	defer func() { _ = srcFile.Close() }()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return coreerr.E("build.createTarGzArchive", "failed to stat source file", err)
	}

	// Create the destination file
	dstFile, err := fs.Create(dst)
	if err != nil {
		return coreerr.E("build.createTarGzArchive", "failed to create archive file", err)
	}
	defer func() { _ = dstFile.Close() }()

	// Create gzip writer
	gzWriter := gzip.NewWriter(dstFile)
	defer func() { _ = gzWriter.Close() }()

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer func() { _ = tarWriter.Close() }()

	// Create tar header
	header, err := tar.FileInfoHeader(srcInfo, "")
	if err != nil {
		return coreerr.E("build.createTarGzArchive", "failed to create tar header", err)
	}
	// Use just the filename, not the full path
	header.Name = filepath.Base(src)

	// Write header
	if err := tarWriter.WriteHeader(header); err != nil {
		return coreerr.E("build.createTarGzArchive", "failed to write tar header", err)
	}

	// Write file content
	if _, err := io.Copy(tarWriter, srcFile); err != nil {
		return coreerr.E("build.createTarGzArchive", "failed to write file content to tar", err)
	}

	return nil
}

// createZipArchive creates a zip archive containing a single file.
func createZipArchive(fs io_interface.Medium, src, dst string) error {
	// Open the source file
	srcFile, err := fs.Open(src)
	if err != nil {
		return coreerr.E("build.createZipArchive", "failed to open source file", err)
	}
	defer func() { _ = srcFile.Close() }()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return coreerr.E("build.createZipArchive", "failed to stat source file", err)
	}

	// Create the destination file
	dstFile, err := fs.Create(dst)
	if err != nil {
		return coreerr.E("build.createZipArchive", "failed to create archive file", err)
	}
	defer func() { _ = dstFile.Close() }()

	// Create zip writer
	zipWriter := zip.NewWriter(dstFile)
	defer func() { _ = zipWriter.Close() }()

	// Create zip header
	header, err := zip.FileInfoHeader(srcInfo)
	if err != nil {
		return coreerr.E("build.createZipArchive", "failed to create zip header", err)
	}
	// Use just the filename, not the full path
	header.Name = filepath.Base(src)
	header.Method = zip.Deflate

	// Create file in archive
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return coreerr.E("build.createZipArchive", "failed to create zip entry", err)
	}

	// Write file content
	if _, err := io.Copy(writer, srcFile); err != nil {
		return coreerr.E("build.createZipArchive", "failed to write file content to zip", err)
	}

	return nil
}
