// Package build provides project type detection and cross-compilation for the Core build system.
package build

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	stdio "io"
	stdfs "io/fs"
	"slices"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	io_interface "dappco.re/go/io"
	// TODO(AX-6): Replace with dappco.re/go/crypt when it exposes Compress/Decompress API parity.
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
func ParseArchiveFormat(value string) core.Result {
	switch core.Trim(core.Lower(value)) {
	case "", "gz", "gzip", "tgz", "tar.gz", "tar-gz":
		return core.Ok(ArchiveFormatGzip)
	case "xz", "txz", "tar.xz", "tar-xz":
		return core.Ok(ArchiveFormatXZ)
	case "zip":
		return core.Ok(ArchiveFormatZip)
	default:
		return core.Fail(core.E("build.ParseArchiveFormat", "unsupported archive format: "+value, nil))
	}
}

// Archive creates an archive for a single artifact using gzip compression.
// Uses tar.gz for linux/darwin and zip for windows.
// The archive is created alongside the binary (e.g., dist/myapp_linux_amd64.tar.gz).
// Returns a new Artifact with Path pointing to the archive.
//
// archived, err := build.Archive(io.Local, artifact)
func Archive(fs io_interface.Medium, artifact Artifact) core.Result {
	return ArchiveWithFormat(fs, artifact, ArchiveFormatGzip)
}

// ArchiveXZ creates an archive for a single artifact using xz compression.
// Uses tar.xz for linux/darwin and zip for windows.
// Returns a new Artifact with Path pointing to the archive.
//
// archived, err := build.ArchiveXZ(io.Local, artifact)
func ArchiveXZ(fs io_interface.Medium, artifact Artifact) core.Result {
	return ArchiveWithFormat(fs, artifact, ArchiveFormatXZ)
}

// ArchiveWithFormat creates an archive for a single artifact with the specified format.
// Uses tar.gz, tar.xz, or zip depending on the requested format.
// Windows artifacts always use zip unless zip is requested explicitly.
// The archive is created alongside the binary (e.g., dist/myapp_linux_amd64.tar.xz).
// Returns a new Artifact with Path pointing to the archive.
//
// archived, err := build.ArchiveWithFormat(io.Local, artifact, build.ArchiveFormatXZ)
func ArchiveWithFormat(fs io_interface.Medium, artifact Artifact, format ArchiveFormat) core.Result {
	if artifact.Path == "" {
		return core.Fail(core.E("build.Archive", "artifact path is empty", nil))
	}

	// Verify the source file exists
	if stat := fs.Stat(artifact.Path); !stat.OK {
		return core.Fail(core.E("build.Archive", "source file not found", core.NewError(stat.Error())))
	}

	// Determine archive type based on OS and format.
	var archivePath string
	var archiveFunc func(fs io_interface.Medium, src, dst string) core.Result

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
	archived := archiveFunc(fs, artifact.Path, archivePath)
	if !archived.OK {
		return core.Fail(core.E("build.Archive", "failed to create archive", core.NewError(archived.Error())))
	}

	return core.Ok(Artifact{
		Path:     archivePath,
		OS:       artifact.OS,
		Arch:     artifact.Arch,
		Checksum: artifact.Checksum,
	})
}

// ArchiveAll archives all artifacts using gzip compression.
// Returns a slice of new artifacts pointing to the archives.
//
// archived, err := build.ArchiveAll(io.Local, artifacts)
func ArchiveAll(fs io_interface.Medium, artifacts []Artifact) core.Result {
	return ArchiveAllWithFormat(fs, artifacts, ArchiveFormatGzip)
}

// ArchiveAllXZ archives all artifacts using xz compression.
// Returns a slice of new artifacts pointing to the archives.
//
// archived, err := build.ArchiveAllXZ(io.Local, artifacts)
func ArchiveAllXZ(fs io_interface.Medium, artifacts []Artifact) core.Result {
	return ArchiveAllWithFormat(fs, artifacts, ArchiveFormatXZ)
}

// ArchiveAllWithFormat archives all artifacts with the specified format.
// Returns a slice of new artifacts pointing to the archives.
//
// archived, err := build.ArchiveAllWithFormat(io.Local, artifacts, build.ArchiveFormatXZ)
func ArchiveAllWithFormat(fs io_interface.Medium, artifacts []Artifact, format ArchiveFormat) core.Result {
	if len(artifacts) == 0 {
		return core.Ok([]Artifact(nil))
	}

	var archived []Artifact
	for _, artifact := range artifacts {
		arch := ArchiveWithFormat(fs, artifact, format)
		if !arch.OK {
			return core.Fail(core.E("build.ArchiveAll", "failed to archive "+artifact.Path, core.NewError(arch.Error())))
		}
		archived = append(archived, arch.Value.(Artifact))
	}

	return core.Ok(archived)
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
	if !archiveBaseNameHasPlatformSuffix(binaryName, artifact.OS, artifact.Arch) {
		binaryName = core.Sprintf("%s_%s_%s", binaryName, artifact.OS, artifact.Arch)
	}

	// Construct archive name: myapp_linux_amd64.tar.gz
	archiveName := core.Concat(binaryName, ext)

	return ax.Join(outputDir, archiveName)
}

func archiveBaseName(path string) string {
	name := ax.Base(path)
	name = core.TrimSuffix(name, ".exe")
	name = core.TrimSuffix(name, ".app")
	return name
}

func archiveBaseNameHasPlatformSuffix(name, os, arch string) bool {
	if name == "" || os == "" || arch == "" {
		return false
	}

	platform := core.Sprintf("_%s_%s", os, arch)
	return core.HasSuffix(name, platform) || core.Contains(name, platform+"_")
}

// createTarXzArchive creates a tar.xz archive containing a file or directory tree.
// TODO(AX-6): Replace Borg compression with dappco.re/go/crypt once API parity exists.
func createTarXzArchive(fs io_interface.Medium, src, dst string) core.Result {
	// Create tar archive in memory
	tarBuf := core.NewBuffer()
	tarWriter := tar.NewWriter(tarBuf)
	written := writeTarTree(fs, tarWriter, src, src)
	if !written.OK {
		return written
	}

	if err := tarWriter.Close(); err != nil {
		return core.Fail(core.E("build.createTarXzArchive", "failed to close tar writer", err))
	}

	// Compress with xz using the deferred Borg API.
	xzData, err := compress.Compress(tarBuf.Bytes(), "xz")
	if err != nil {
		return core.Fail(core.E("build.createTarXzArchive", "failed to compress with xz", err))
	}

	return writeArchiveBytes(fs, dst, xzData, "build.createTarXzArchive")
}

// createTarGzArchive creates a tar.gz archive containing a file or directory tree.
func createTarGzArchive(fs io_interface.Medium, src, dst string) core.Result {
	buf := core.NewBuffer()

	// Create gzip writer
	gzWriter := gzip.NewWriter(buf)

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)

	written := writeTarTree(fs, tarWriter, src, src)
	if !written.OK {
		tarWriter.Close()
		gzWriter.Close()
		return written
	}
	if err := tarWriter.Close(); err != nil {
		return core.Fail(core.E("build.createTarGzArchive", "failed to close tar writer", err))
	}
	if err := gzWriter.Close(); err != nil {
		return core.Fail(core.E("build.createTarGzArchive", "failed to close gzip writer", err))
	}

	return writeArchiveBytes(fs, dst, buf.Bytes(), "build.createTarGzArchive")
}

// createZipArchive creates a zip archive containing a file or directory tree.
func createZipArchive(fs io_interface.Medium, src, dst string) core.Result {
	buf := core.NewBuffer()

	// Create zip writer
	zipWriter := zip.NewWriter(buf)

	written := writeZipTree(fs, zipWriter, src, src)
	if !written.OK {
		zipWriter.Close()
		return written
	}
	if err := zipWriter.Close(); err != nil {
		return core.Fail(core.E("build.createZipArchive", "failed to close zip writer", err))
	}

	return writeArchiveBytes(fs, dst, buf.Bytes(), "build.createZipArchive")
}

func writeArchiveBytes(fs io_interface.Medium, dst string, data []byte, operation string) core.Result {
	written := fs.Write(dst, string(data))
	if !written.OK {
		return core.Fail(core.E(operation, "failed to write archive file", core.NewError(written.Error())))
	}

	return core.Ok(nil)
}

func writeTarTree(fs io_interface.Medium, writer *tar.Writer, rootPath, currentPath string) core.Result {
	info := fs.Stat(currentPath)
	if !info.OK {
		return core.Fail(core.E("build.writeTarTree", "failed to stat archive entry", core.NewError(info.Error())))
	}
	fileInfo := info.Value.(stdfs.FileInfo)

	header, err := tar.FileInfoHeader(fileInfo, "")
	if err != nil {
		return core.Fail(core.E("build.writeTarTree", "failed to create tar header", err))
	}
	header.Name = archiveEntryName(rootPath, currentPath)
	if fileInfo.IsDir() {
		header.Name += "/"
	}

	if err := writer.WriteHeader(header); err != nil {
		return core.Fail(core.E("build.writeTarTree", "failed to write tar header", err))
	}

	if fileInfo.IsDir() {
		entries := fs.List(currentPath)
		if !entries.OK {
			return core.Fail(core.E("build.writeTarTree", "failed to list archive directory", core.NewError(entries.Error())))
		}
		dirEntries := entries.Value.([]core.FsDirEntry)
		sortDirEntries(dirEntries)
		for _, entry := range dirEntries {
			written := writeTarTree(fs, writer, rootPath, ax.Join(currentPath, entry.Name()))
			if !written.OK {
				return written
			}
		}
		return core.Ok(nil)
	}

	source := fs.Open(currentPath)
	if !source.OK {
		return core.Fail(core.E("build.writeTarTree", "failed to open archive entry", core.NewError(source.Error())))
	}
	stream := source.Value.(core.FsFile)
	defer stream.Close()

	if _, err := stdio.Copy(writer, stream); err != nil {
		return core.Fail(core.E("build.writeTarTree", "failed to write file content to tar", err))
	}

	return core.Ok(nil)
}

func writeZipTree(fs io_interface.Medium, writer *zip.Writer, rootPath, currentPath string) core.Result {
	info := fs.Stat(currentPath)
	if !info.OK {
		return core.Fail(core.E("build.writeZipTree", "failed to stat archive entry", core.NewError(info.Error())))
	}
	fileInfo := info.Value.(stdfs.FileInfo)

	header, err := zip.FileInfoHeader(fileInfo)
	if err != nil {
		return core.Fail(core.E("build.writeZipTree", "failed to create zip header", err))
	}
	header.Name = archiveEntryName(rootPath, currentPath)

	if fileInfo.IsDir() {
		header.Name += "/"
		if _, err := writer.CreateHeader(header); err != nil {
			return core.Fail(core.E("build.writeZipTree", "failed to create zip directory entry", err))
		}

		entries := fs.List(currentPath)
		if !entries.OK {
			return core.Fail(core.E("build.writeZipTree", "failed to list archive directory", core.NewError(entries.Error())))
		}
		dirEntries := entries.Value.([]core.FsDirEntry)
		sortDirEntries(dirEntries)
		for _, entry := range dirEntries {
			written := writeZipTree(fs, writer, rootPath, ax.Join(currentPath, entry.Name()))
			if !written.OK {
				return written
			}
		}
		return core.Ok(nil)
	}

	header.Method = zip.Deflate
	zipEntry, err := writer.CreateHeader(header)
	if err != nil {
		return core.Fail(core.E("build.writeZipTree", "failed to create zip entry", err))
	}

	source := fs.Open(currentPath)
	if !source.OK {
		return core.Fail(core.E("build.writeZipTree", "failed to open archive entry", core.NewError(source.Error())))
	}
	stream := source.Value.(core.FsFile)
	defer func() { _ = stream.Close() }()

	if _, err := stdio.Copy(zipEntry, stream); err != nil {
		return core.Fail(core.E("build.writeZipTree", "failed to write file content to zip", err))
	}

	return core.Ok(nil)
}

func archiveEntryName(rootPath, currentPath string) string {
	rootName := ax.Base(rootPath)
	if currentPath == rootPath {
		return rootName
	}

	relPathResult := ax.Rel(rootPath, currentPath)
	if !relPathResult.OK {
		return rootName
	}
	relPath := relPathResult.Value.(string)
	if relPath == "" || relPath == "." {
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
