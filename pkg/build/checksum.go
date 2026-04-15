// Package build provides project type detection and cross-compilation for the Core build system.
package build

import (
	"crypto/sha256"
	"encoding/hex"
	stdio "io"
	"slices"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	io_interface "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

// Checksum computes SHA256 for an artifact and returns the artifact with the Checksum field filled.
//
// cs, err := build.Checksum(io.Local, artifact)
func Checksum(fs io_interface.Medium, artifact Artifact) (Artifact, error) {
	if artifact.Path == "" {
		return Artifact{}, coreerr.E("build.Checksum", "artifact path is empty", nil)
	}

	// Open the file
	file, err := fs.Open(artifact.Path)
	if err != nil {
		return Artifact{}, coreerr.E("build.Checksum", "failed to open file", err)
	}
	defer func() { _ = file.Close() }()

	// Compute SHA256 hash
	hasher := sha256.New()
	if _, err := stdio.Copy(hasher, file); err != nil {
		return Artifact{}, coreerr.E("build.Checksum", "failed to hash file", err)
	}

	checksum := hex.EncodeToString(hasher.Sum(nil))

	return Artifact{
		Path:     artifact.Path,
		OS:       artifact.OS,
		Arch:     artifact.Arch,
		Checksum: checksum,
	}, nil
}

// ChecksumAll computes checksums for all artifacts.
// Returns a slice of artifacts with their Checksum fields filled.
//
// checked, err := build.ChecksumAll(io.Local, artifacts)
func ChecksumAll(fs io_interface.Medium, artifacts []Artifact) ([]Artifact, error) {
	if len(artifacts) == 0 {
		return nil, nil
	}

	var checksummed []Artifact
	for _, artifact := range artifacts {
		cs, err := Checksum(fs, artifact)
		if err != nil {
			return checksummed, coreerr.E("build.ChecksumAll", "failed to checksum "+artifact.Path, err)
		}
		checksummed = append(checksummed, cs)
	}

	return checksummed, nil
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
func WriteChecksumFile(fs io_interface.Medium, artifacts []Artifact, path string) error {
	if len(artifacts) == 0 {
		return nil
	}

	// Build the content
	var lines []string
	for _, artifact := range artifacts {
		if artifact.Checksum == "" {
			return coreerr.E("build.WriteChecksumFile", "artifact "+artifact.Path+" has no checksum", nil)
		}
		filename := checksumFilename(path, artifact.Path)
		lines = append(lines, core.Sprintf("%s  %s", artifact.Checksum, filename))
	}

	// Sort lines for consistent output
	slices.Sort(lines)

	content := core.Concat(core.Join("\n", lines...), "\n")

	// Write the file using the medium (which handles directory creation in Write)
	if err := fs.Write(path, content); err != nil {
		return coreerr.E("build.WriteChecksumFile", "failed to write file", err)
	}

	return nil
}

func checksumFilename(checksumPath, artifactPath string) string {
	baseDir := ax.Dir(checksumPath)
	relativePath, err := ax.Rel(baseDir, artifactPath)
	if err == nil {
		relativePath = ax.Clean(relativePath)
		if relativePath != "" &&
			relativePath != "." &&
			relativePath != ".." &&
			!ax.IsAbs(relativePath) &&
			!core.HasPrefix(relativePath, ".."+ax.DS()) {
			return core.Replace(relativePath, ax.DS(), "/")
		}
	}

	return core.PathBase(artifactPath)
}
