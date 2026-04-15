// Package build provides project type detection and cross-compilation for the Core build system.
package build

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	stdio "io"
	stdfs "io/fs"
	"slices"

	"dappco.re/go/core"
	"dappco.re/go/build/internal/ax"
	io_interface "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
	"github.com/Snider/Borg/pkg/compress"
)

// ArchiveFormat specifies the compression format for archives.
//
// var fmt build.ArchiveFormat = build.ArchiveFormatGzip
type ArchiveFormat string

const (
	// ArchiveFormatGzip uses tar.gz (gzip compression) - widely compatible.
	ArchiveFormatGzip ArchiveFormat = "gz"
	// ArchiveFormatXZ uses tar.xz (xz/LZMA2 compression) - better compression ratio.
	ArchiveFormatXZ ArchiveFormat = "xz"
	// ArchiveFormatZip uses zip archives on any platform.
	ArchiveFormatZip ArchiveFormat = "zip"
)

// ParseArchiveFormat converts a user-facing archive format string into an ArchiveFormat.
//
//	format, err := build.ParseArchiveFormat("xz")  // → build.ArchiveFormatXZ
//	format, err := build.ParseArchiveFormat("zip") // → build.ArchiveFormatZip
func ParseArchiveFormat(value string) (ArchiveFormat, error) {
	switch core.Trim(core.Lower(value)) {
	case "", "gz", "gzip", "tgz", "tar.gz", "tar-gz":
		return ArchiveFormatGzip, nil
	case "xz", "txz", "tar.xz", "tar-xz":
		return ArchiveFormatXZ, nil
	case "zip":
		return ArchiveFormatZip, nil
	default:
		return "", coreerr.E("build.ParseArchiveFormat", "unsupported archive format: "+value, nil)
	}
}

// Archive creates an archive for a single artifact using gzip compression.
// Uses tar.gz for linux/darwin and zip for windows.
// The archive is created alongside the binary (e.g., dist/myapp_linux_amd64.tar.gz).
// Returns a new Artifact with Path pointing to the archive.
//
// archived, err := build.Archive(io.Local, artifact)
func Archive(fs io_interface.Medium, artifact Artifact) (Artifact, error) {
	return ArchiveWithFormat(fs, artifact, ArchiveFormatGzip)
}

// ArchiveXZ creates an archive for a single artifact using xz compression.
// Uses tar.xz for linux/darwin and zip for windows.
// Returns a new Artifact with Path pointing to the archive.
//
// archived, err := build.ArchiveXZ(io.Local, artifact)
func ArchiveXZ(fs io_interface.Medium, artifact Artifact) (Artifact, error) {
	return ArchiveWithFormat(fs, artifact, ArchiveFormatXZ)
}

// ArchiveWithFormat creates an archive for a single artifact with the specified format.
// Uses tar.gz, tar.xz, or zip depending on the requested format.
// Windows artifacts always use zip unless zip is requested explicitly.
// The archive is created alongside the binary (e.g., dist/myapp_linux_amd64.tar.xz).
// Returns a new Artifact with Path pointing to the archive.
//
// archived, err := build.ArchiveWithFormat(io.Local, artifact, build.ArchiveFormatXZ)
func ArchiveWithFormat(fs io_interface.Medium, artifact Artifact, format ArchiveFormat) (Artifact, error) {
	if artifact.Path == "" {
		return Artifact{}, coreerr.E("build.Archive", "artifact path is empty", nil)
	}

	// Verify the source file exists
	if _, err := fs.Stat(artifact.Path); err != nil {
		return Artifact{}, coreerr.E("build.Archive", "source file not found", err)
	}

	// Determine archive type based on OS and format.
	var archivePath string
	var archiveFunc func(fs io_interface.Medium, src, dst string) error

	switch {
	case format == ArchiveFormatZip || artifact.OS == "windows":
		archivePath = archiveFilename(artifact, ".zip")
		archiveFunc = createZipArchive
	case format == ArchiveFormatXZ:
		archivePath = archiveFilename(artifact, ".tar.xz")
		archiveFunc = createTarXzArchive
	default:
		archivePath = archiveFilename(artifact, ".tar.gz")
		archiveFunc = createTarGzArchive
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
//
// archived, err := build.ArchiveAll(io.Local, artifacts)
func ArchiveAll(fs io_interface.Medium, artifacts []Artifact) ([]Artifact, error) {
	return ArchiveAllWithFormat(fs, artifacts, ArchiveFormatGzip)
}

// ArchiveAllXZ archives all artifacts using xz compression.
// Returns a slice of new artifacts pointing to the archives.
//
// archived, err := build.ArchiveAllXZ(io.Local, artifacts)
func ArchiveAllXZ(fs io_interface.Medium, artifacts []Artifact) ([]Artifact, error) {
	return ArchiveAllWithFormat(fs, artifacts, ArchiveFormatXZ)
}

// ArchiveAllWithFormat archives all artifacts with the specified format.
// Returns a slice of new artifacts pointing to the archives.
//
// archived, err := build.ArchiveAllWithFormat(io.Local, artifacts, build.ArchiveFormatXZ)
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
	dir := ax.Dir(artifact.Path)
	// Go up one level to the output directory (e.g., dist)
	outputDir := ax.Dir(dir)

	// Get the binary or bundle name without packaging extensions.
	binaryName := archiveBaseName(artifact.Path)

	// Construct archive name: myapp_linux_amd64.tar.gz
	archiveName := core.Sprintf("%s_%s_%s%s", binaryName, artifact.OS, artifact.Arch, ext)

	return ax.Join(outputDir, archiveName)
}

func archiveBaseName(path string) string {
	name := ax.Base(path)
	name = core.TrimSuffix(name, ".exe")
	name = core.TrimSuffix(name, ".app")
	return name
}

// createTarXzArchive creates a tar.xz archive containing a file or directory tree.
// Uses Borg's compress package for xz compression.
func createTarXzArchive(fs io_interface.Medium, src, dst string) error {
	// Create tar archive in memory
	var tarBuf bytes.Buffer
	tarWriter := tar.NewWriter(&tarBuf)
	if err := writeTarTree(fs, tarWriter, src, src); err != nil {
		return err
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

// createTarGzArchive creates a tar.gz archive containing a file or directory tree.
func createTarGzArchive(fs io_interface.Medium, src, dst string) error {
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

	return writeTarTree(fs, tarWriter, src, src)
}

// createZipArchive creates a zip archive containing a file or directory tree.
func createZipArchive(fs io_interface.Medium, src, dst string) error {
	// Create the destination file
	dstFile, err := fs.Create(dst)
	if err != nil {
		return coreerr.E("build.createZipArchive", "failed to create archive file", err)
	}
	defer func() { _ = dstFile.Close() }()

	// Create zip writer
	zipWriter := zip.NewWriter(dstFile)
	defer func() { _ = zipWriter.Close() }()

	return writeZipTree(fs, zipWriter, src, src)
}

func writeTarTree(fs io_interface.Medium, writer *tar.Writer, rootPath, currentPath string) error {
	info, err := fs.Stat(currentPath)
	if err != nil {
		return coreerr.E("build.writeTarTree", "failed to stat archive entry", err)
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return coreerr.E("build.writeTarTree", "failed to create tar header", err)
	}
	header.Name = archiveEntryName(rootPath, currentPath)
	if info.IsDir() {
		header.Name += "/"
	}

	if err := writer.WriteHeader(header); err != nil {
		return coreerr.E("build.writeTarTree", "failed to write tar header", err)
	}

	if info.IsDir() {
		entries, err := fs.List(currentPath)
		if err != nil {
			return coreerr.E("build.writeTarTree", "failed to list archive directory", err)
		}
		sortDirEntries(entries)
		for _, entry := range entries {
			if err := writeTarTree(fs, writer, rootPath, ax.Join(currentPath, entry.Name())); err != nil {
				return err
			}
		}
		return nil
	}

	source, err := fs.Open(currentPath)
	if err != nil {
		return coreerr.E("build.writeTarTree", "failed to open archive entry", err)
	}
	defer func() { _ = source.Close() }()

	if _, err := stdio.Copy(writer, source); err != nil {
		return coreerr.E("build.writeTarTree", "failed to write file content to tar", err)
	}

	return nil
}

func writeZipTree(fs io_interface.Medium, writer *zip.Writer, rootPath, currentPath string) error {
	info, err := fs.Stat(currentPath)
	if err != nil {
		return coreerr.E("build.writeZipTree", "failed to stat archive entry", err)
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return coreerr.E("build.writeZipTree", "failed to create zip header", err)
	}
	header.Name = archiveEntryName(rootPath, currentPath)

	if info.IsDir() {
		header.Name += "/"
		if _, err := writer.CreateHeader(header); err != nil {
			return coreerr.E("build.writeZipTree", "failed to create zip directory entry", err)
		}

		entries, err := fs.List(currentPath)
		if err != nil {
			return coreerr.E("build.writeZipTree", "failed to list archive directory", err)
		}
		sortDirEntries(entries)
		for _, entry := range entries {
			if err := writeZipTree(fs, writer, rootPath, ax.Join(currentPath, entry.Name())); err != nil {
				return err
			}
		}
		return nil
	}

	header.Method = zip.Deflate
	zipEntry, err := writer.CreateHeader(header)
	if err != nil {
		return coreerr.E("build.writeZipTree", "failed to create zip entry", err)
	}

	source, err := fs.Open(currentPath)
	if err != nil {
		return coreerr.E("build.writeZipTree", "failed to open archive entry", err)
	}
	defer func() { _ = source.Close() }()

	if _, err := stdio.Copy(zipEntry, source); err != nil {
		return coreerr.E("build.writeZipTree", "failed to write file content to zip", err)
	}

	return nil
}

func archiveEntryName(rootPath, currentPath string) string {
	rootName := ax.Base(rootPath)
	if currentPath == rootPath {
		return rootName
	}

	relPath, err := ax.Rel(rootPath, currentPath)
	if err != nil || relPath == "" || relPath == "." {
		return rootName
	}

	return core.Replace(ax.Join(rootName, relPath), ax.DS(), "/")
}

func sortDirEntries(entries []stdfs.DirEntry) {
	slices.SortFunc(entries, func(a, b stdfs.DirEntry) int {
		if a.Name() < b.Name() {
			return -1
		}
		if a.Name() > b.Name() {
			return 1
		}
		return 0
	})
}
