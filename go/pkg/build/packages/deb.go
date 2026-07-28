// Package packages builds native OS package formats from a built binary.
//
// A .deb is an ar archive holding three members in a fixed order: debian-binary,
// control.tar.gz and data.tar.gz. Both the ar container and the tarballs inside
// it are in the Go standard library's reach, so this builds one directly rather
// than shelling out to dpkg-deb — which would restrict releases to Debian hosts
// and put a package manager in the build's dependency list.
package packages

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"strconv"

	core "dappco.re/go"
	storage "dappco.re/go/build/pkg/storage"
)

// DebSpec describes a Debian package to build.
//
//	spec := packages.DebSpec{Name: "core", Version: "0.12.3", Architecture: "amd64", BinaryPath: "dist/core"}
type DebSpec struct {
	// Name is the package name (e.g. "core").
	Name string
	// Version is the package version. A leading "v" is stripped, because
	// Debian versions must begin with a digit.
	Version string
	// Architecture is a GOARCH value; it is translated to the Debian name.
	Architecture string
	// Maintainer is the "Name <email>" field. Required by policy.
	Maintainer string
	// Homepage is an optional project URL.
	Homepage string
	// Description is the one-line summary shown by apt search.
	Description string
	// LongDescription is the optional extended description.
	LongDescription string
	// Section defaults to "devel".
	Section string
	// Priority defaults to "optional".
	Priority string
	// Depends is an optional comma-separated dependency list. A static Go
	// binary usually has none.
	Depends string
	// BinaryPath is the built binary to package.
	BinaryPath string
	// InstallPath is where the binary lands, defaulting to /usr/bin/<name>.
	InstallPath string
}

// DebArchitecture translates a GOARCH value into the Debian architecture name.
//
// The two vocabularies agree on amd64 and arm64 and disagree on the rest, which
// is the sort of difference that silently produces a package apt refuses to
// install on the machine it was built for.
//
//	packages.DebArchitecture("386") // "i386"
func DebArchitecture(goarch string) string {
	switch core.Trim(core.Lower(goarch)) {
	case "amd64", "x86_64":
		return "amd64"
	case "arm64", "aarch64":
		return "arm64"
	case "386", "i386", "x86":
		return "i386"
	case "arm":
		return "armhf"
	case "ppc64le":
		return "ppc64el"
	case "s390x":
		return "s390x"
	case "riscv64":
		return "riscv64"
	case "mips64le":
		return "mips64el"
	default:
		return core.Trim(core.Lower(goarch))
	}
}

// DebVersion normalises a release tag into a Debian version.
//
// Debian requires the version to start with a digit, so the "v" that every Go
// tag carries has to come off.
//
//	packages.DebVersion("v0.12.3") // "0.12.3"
func DebVersion(version string) string {
	trimmed := core.Trim(version)
	// Subdirectory module tags arrive as "go/v0.12.3".
	if index := core.LastIndex(trimmed, "/"); index >= 0 {
		trimmed = trimmed[index+1:]
	}
	return core.TrimPrefix(trimmed, "v")
}

// DebFileName returns the conventional package file name.
//
//	packages.DebFileName("core", "v0.12.3", "amd64") // "core_0.12.3_amd64.deb"
func DebFileName(name, version, goarch string) string {
	return core.Concat(name, "_", DebVersion(version), "_", DebArchitecture(goarch), ".deb")
}

// BuildDeb writes a Debian package for spec to outputPath and returns that path.
//
//	r := packages.BuildDeb(storage.Local, spec, "dist/core_0.12.3_amd64.deb")
//	if r.OK { path := r.Value.(string) }
func BuildDeb(filesystem storage.Medium, spec DebSpec, outputPath string) core.Result {
	if filesystem == nil {
		return core.Fail(core.E("packages.BuildDeb", "no filesystem", nil))
	}
	if core.Trim(spec.Name) == "" {
		return core.Fail(core.E("packages.BuildDeb", "package name is required", nil))
	}
	if DebVersion(spec.Version) == "" {
		return core.Fail(core.E("packages.BuildDeb", "package version is required", nil))
	}
	if core.Trim(spec.BinaryPath) == "" {
		return core.Fail(core.E("packages.BuildDeb", "binary path is required", nil))
	}

	binaryResult := filesystem.Read(spec.BinaryPath)
	if !binaryResult.OK {
		return core.Fail(core.E("packages.BuildDeb", "failed to read binary: "+spec.BinaryPath, core.NewError(binaryResult.Error())))
	}
	binary, ok := binaryResult.Value.(string)
	if !ok {
		return core.Fail(core.E("packages.BuildDeb", "unexpected binary content type", nil))
	}
	if len(binary) == 0 {
		return core.Fail(core.E("packages.BuildDeb", "binary is empty: "+spec.BinaryPath, nil))
	}

	installPath := core.Trim(spec.InstallPath)
	if installPath == "" {
		installPath = "/usr/bin/" + spec.Name
	}

	dataResult := buildDataArchive(binary, installPath)
	if !dataResult.OK {
		return dataResult
	}
	data := dataResult.Value.(string)

	controlResult := buildControlArchive(spec, binary, installPath)
	if !controlResult.OK {
		return controlResult
	}
	control := controlResult.Value.(string)

	archive := buildAr([]arMember{
		// Order is not cosmetic: dpkg reads the members in sequence and
		// rejects an archive whose first member is not debian-binary.
		{Name: "debian-binary", Content: "2.0\n"},
		{Name: "control.tar.gz", Content: control},
		{Name: "data.tar.gz", Content: data},
	})

	if directory := core.PathDir(outputPath); directory != "" && directory != "." {
		if ensured := filesystem.EnsureDir(directory); !ensured.OK {
			return core.Fail(core.E("packages.BuildDeb", "failed to create output directory", core.NewError(ensured.Error())))
		}
	}

	if written := filesystem.WriteMode(outputPath, archive, 0o644); !written.OK {
		return core.Fail(core.E("packages.BuildDeb", "failed to write package: "+outputPath, core.NewError(written.Error())))
	}

	return core.Ok(outputPath)
}

// buildControlArchive produces control.tar.gz: the control file plus md5sums,
// which apt uses to detect locally modified files.
func buildControlArchive(spec DebSpec, binary, installPath string) core.Result {
	digest := md5.Sum([]byte(binary))
	// Paths in md5sums are relative to the filesystem root, without a leading
	// slash.
	md5sums := core.Concat(hex.EncodeToString(digest[:]), "  ", core.TrimPrefix(installPath, "/"), "\n")

	return buildTarGz([]tarEntry{
		{Name: "./control", Content: controlFile(spec, binary), Mode: 0o644},
		{Name: "./md5sums", Content: md5sums, Mode: 0o644},
	})
}

// buildDataArchive produces data.tar.gz holding the binary at its install path,
// preceded by the directory entries leading to it.
func buildDataArchive(binary, installPath string) core.Result {
	entries := make([]tarEntry, 0, 4)

	// dpkg wants every parent directory declared before the file inside it.
	segments := core.Split(core.TrimPrefix(installPath, "/"), "/")
	prefix := "."
	for index := 0; index < len(segments)-1; index++ {
		prefix = core.Concat(prefix, "/", segments[index])
		entries = append(entries, tarEntry{Name: prefix + "/", Mode: 0o755, Dir: true})
	}

	entries = append(entries, tarEntry{
		Name:    core.Concat("./", core.TrimPrefix(installPath, "/")),
		Content: binary,
		Mode:    0o755,
	})

	return buildTarGz(entries)
}

// controlFile renders the Debian control stanza.
func controlFile(spec DebSpec, binary string) string {
	section := core.Trim(spec.Section)
	if section == "" {
		section = "devel"
	}
	priority := core.Trim(spec.Priority)
	if priority == "" {
		priority = "optional"
	}
	maintainer := core.Trim(spec.Maintainer)
	if maintainer == "" {
		maintainer = "unknown <unknown@localhost>"
	}
	summary := core.Trim(spec.Description)
	if summary == "" {
		summary = spec.Name
	}

	// Installed-Size is in kibibytes, rounded up: apt reports it before
	// downloading and a zero reads as a broken package.
	installedSize := (len(binary) + 1023) / 1024

	out := core.Concat(
		"Package: ", spec.Name, "\n",
		"Version: ", DebVersion(spec.Version), "\n",
		"Section: ", section, "\n",
		"Priority: ", priority, "\n",
		"Architecture: ", DebArchitecture(spec.Architecture), "\n",
		"Maintainer: ", maintainer, "\n",
		"Installed-Size: ", strconv.Itoa(installedSize), "\n",
	)
	if depends := core.Trim(spec.Depends); depends != "" {
		out = core.Concat(out, "Depends: ", depends, "\n")
	}
	if homepage := core.Trim(spec.Homepage); homepage != "" {
		out = core.Concat(out, "Homepage: ", homepage, "\n")
	}

	out = core.Concat(out, "Description: ", summary, "\n")
	// Continuation lines are indented by one space; a blank line inside the
	// description has to be written as " ." or the stanza ends early.
	for _, line := range core.Split(core.Trim(spec.LongDescription), "\n") {
		if core.Trim(line) == "" {
			continue
		}
		out = core.Concat(out, " ", line, "\n")
	}

	return out
}

// tarEntry is one file or directory to place in a tarball.
type tarEntry struct {
	Name    string
	Content string
	Mode    int64
	Dir     bool
}

// buildTarGz writes entries into a gzip-compressed tar archive.
func buildTarGz(entries []tarEntry) core.Result {
	var buffer bytes.Buffer
	compressor := gzip.NewWriter(&buffer)
	archive := tar.NewWriter(compressor)

	for _, entry := range entries {
		header := &tar.Header{
			Name:     entry.Name,
			Mode:     entry.Mode,
			Size:     int64(len(entry.Content)),
			Format:   tar.FormatGNU,
			Typeflag: tar.TypeReg,
		}
		if entry.Dir {
			header.Typeflag = tar.TypeDir
			header.Size = 0
		}

		if err := archive.WriteHeader(header); err != nil {
			return core.Fail(core.E("packages.buildTarGz", "failed to write tar header: "+entry.Name, err))
		}
		if !entry.Dir {
			if _, err := archive.Write([]byte(entry.Content)); err != nil {
				return core.Fail(core.E("packages.buildTarGz", "failed to write tar entry: "+entry.Name, err))
			}
		}
	}

	if err := archive.Close(); err != nil {
		return core.Fail(core.E("packages.buildTarGz", "failed to close tar", err))
	}
	if err := compressor.Close(); err != nil {
		return core.Fail(core.E("packages.buildTarGz", "failed to close gzip", err))
	}

	return core.Ok(buffer.String())
}

// arMember is one member of an ar archive.
type arMember struct {
	Name    string
	Content string
}

// buildAr assembles members into a BSD-style ar archive.
//
// The format is small enough to write directly: an 8-byte magic, then per
// member a fixed 60-byte header of space-padded ASCII fields followed by the
// content, padded to an even length.
func buildAr(members []arMember) string {
	var buffer bytes.Buffer
	buffer.WriteString("!<arch>\n")

	for _, member := range members {
		// dpkg reads the timestamp, owner and group but does not act on them;
		// zeroing them keeps the package byte-identical between builds.
		buffer.WriteString(arField(member.Name, 16))
		buffer.WriteString(arField("0", 12))     // mtime
		buffer.WriteString(arField("0", 6))      // owner
		buffer.WriteString(arField("0", 6))      // group
		buffer.WriteString(arField("100644", 8)) // mode
		buffer.WriteString(arField(strconv.Itoa(len(member.Content)), 10))
		buffer.WriteString("`\n")

		buffer.WriteString(member.Content)
		if len(member.Content)%2 == 1 {
			buffer.WriteString("\n")
		}
	}

	return buffer.String()
}

// arField renders value left-aligned in a fixed-width space-padded field.
func arField(value string, width int) string {
	if len(value) >= width {
		return value[:width]
	}
	padded := value
	for len(padded) < width {
		padded += " "
	}
	return padded
}
