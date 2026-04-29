// Package build provides project type detection and cross-compilation for the Core build system.
package build

import (
	"crypto/sha256"
	"encoding/hex"
	stdio "io"
	"slices"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	io_interface "dappco.re/go/io"
)

// Checksum computes SHA256 for an artifact and returns the artifact with the Checksum field filled.
//
// cs, err := build.Checksum(io.Local, artifact)
func Checksum(fs io_interface.Medium, artifact Artifact) core.Result {
	if artifact.Path == "" {
		return core.Fail(core.E("build.Checksum", "artifact path is empty", nil))
	}

	// Open the file
	file := fs.Open(artifact.Path)
	if !file.OK {
		return core.Fail(core.E("build.Checksum", "failed to open file", core.NewError(file.Error())))
	}
	stream := file.Value.(core.FsFile)
	defer func() { _ = stream.Close() }()

	// Compute SHA256 hash
	hasher := sha256.New()
	if _, err := stdio.Copy(hasher, stream); err != nil {
		return core.Fail(core.E("build.Checksum", "failed to hash file", err))
	}

	checksum := hex.EncodeToString(hasher.Sum(nil))

	return core.Ok(Artifact{
		Path:     artifact.Path,
		OS:       artifact.OS,
		Arch:     artifact.Arch,
		Checksum: checksum,
	})
}

// ChecksumAll computes checksums for all artifacts.
// Returns a slice of artifacts with their Checksum fields filled.
//
// checked, err := build.ChecksumAll(io.Local, artifacts)
func ChecksumAll(fs io_interface.Medium, artifacts []Artifact) core.Result {
	if len(artifacts) == 0 {
		return core.Ok([]Artifact(nil))
	}

	var checksummed []Artifact
	for _, artifact := range artifacts {
		cs := Checksum(fs, artifact)
		if !cs.OK {
			return core.Fail(core.E("build.ChecksumAll", "failed to checksum "+artifact.Path, core.NewError(cs.Error())))
		}
		checksummed = append(checksummed, cs.Value.(Artifact))
	}

	return core.Ok(checksummed)
}

// WriteChecksumFile writes a CHECKSUMS.txt file with the format:
//
//	sha256hash  filename1
//	sha256hash  filename2
//
// The artifacts should have their Checksum fields filled (call ChecksumAll first).
// Filenames are relative to the output directory (just the basename).
//
// err := build.WriteChecksumFile(io.Local, artifacts, "dist/CHECKSUMS.txt")
func WriteChecksumFile(fs io_interface.Medium, artifacts []Artifact, path string) core.Result {
	if len(artifacts) == 0 {
		return core.Ok(nil)
	}

	// Build the content
	var lines []string
	for _, artifact := range artifacts {
		if artifact.Checksum == "" {
			return core.Fail(core.E("build.WriteChecksumFile", "artifact "+artifact.Path+" has no checksum", nil))
		}
		filename := checksumFilename(path, artifact.Path)
		lines = append(lines, core.Sprintf("%s  %s", artifact.Checksum, filename))
	}

	// Sort lines for consistent output
	slices.Sort(lines)

	content := core.Concat(core.Join("\n", lines...), "\n")

	// Write the file using the medium (which handles directory creation in Write)
	written := fs.Write(path, content)
	if !written.OK {
		return core.Fail(core.E("build.WriteChecksumFile", "failed to write file", core.NewError(written.Error())))
	}

	return core.Ok(nil)
}

func checksumFilename(checksumPath, artifactPath string) string {
	baseDir := ax.Dir(checksumPath)
	relativePath := ax.Rel(baseDir, artifactPath)
	if relativePath.OK {
		relativePathValue := ax.Clean(relativePath.Value.(string))
		if relativePathValue != "" &&
			relativePathValue != "." &&
			relativePathValue != ".." &&
			!ax.IsAbs(relativePathValue) &&
			!core.HasPrefix(relativePathValue, ".."+ax.DS()) {
			return core.Replace(relativePathValue, ax.DS(), "/")
		}
	}

	return core.PathBase(artifactPath)
}
