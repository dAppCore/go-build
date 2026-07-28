package packages

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	stdio "io"

	core "dappco.re/go"
	storage "dappco.re/go/build/pkg/storage"
)

func newDebFS(t *core.T) (storage.Medium, string) {
	filesystem := storage.Local
	base := t.TempDir()
	core.AssertTrue(t, filesystem.Write(core.PathJoin(base, "core"), "BINARYCONTENT").OK)
	return filesystem, base
}

func TestDeb_DebArchitecture_Good(t *core.T) {
	core.AssertEqual(t, "amd64", DebArchitecture("amd64"))
	core.AssertEqual(t, "arm64", DebArchitecture("arm64"))
	// The vocabularies diverge here — a package built "386" must say i386.
	core.AssertEqual(t, "i386", DebArchitecture("386"))
	core.AssertEqual(t, "armhf", DebArchitecture("arm"))
	core.AssertEqual(t, "ppc64el", DebArchitecture("ppc64le"))
}

func TestDeb_DebArchitecture_Bad(t *core.T) {
	// Unknown values pass through rather than guessing.
	core.AssertEqual(t, "sparc", DebArchitecture("SPARC"))
	core.AssertEqual(t, "", DebArchitecture(""))
}

func TestDeb_DebVersion_Good(t *core.T) {
	core.AssertEqual(t, "0.12.3", DebVersion("v0.12.3"))
	core.AssertEqual(t, "0.12.3", DebVersion("0.12.3"))
	// Go tags a module in a subdirectory as go/vX.Y.Z.
	core.AssertEqual(t, "0.12.3", DebVersion("go/v0.12.3"))
}

func TestDeb_DebFileName_Good(t *core.T) {
	core.AssertEqual(t, "core_0.12.3_amd64.deb", DebFileName("core", "v0.12.3", "amd64"))
	core.AssertEqual(t, "core_0.12.3_i386.deb", DebFileName("core", "v0.12.3", "386"))
}

func TestDeb_BuildDeb_Good(t *core.T) {
	filesystem, base := newDebFS(t)
	out := core.PathJoin(base, "core_0.12.3_amd64.deb")

	result := BuildDeb(filesystem, DebSpec{
		Name:         "core",
		Version:      "v0.12.3",
		Architecture: "amd64",
		Maintainer:   "Lethean <dev@lethean.io>",
		Description:  "Core developer CLI",
		BinaryPath:   core.PathJoin(base, "core"),
	}, out)

	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, out, result.Value.(string))

	content := filesystem.Read(out)
	core.AssertTrue(t, content.OK)
	archive := content.Value.(string)

	// dpkg identifies a .deb by the ar magic and rejects anything whose first
	// member is not debian-binary.
	core.AssertTrue(t, core.HasPrefix(archive, "!<arch>\n"))
	core.AssertTrue(t, core.Contains(archive, "debian-binary"))
	core.AssertTrue(t, core.Contains(archive, "control.tar.gz"))
	core.AssertTrue(t, core.Contains(archive, "data.tar.gz"))
}

func TestDeb_BuildDeb_Control_Good(t *core.T) {
	filesystem, base := newDebFS(t)
	out := core.PathJoin(base, "core.deb")

	core.AssertTrue(t, BuildDeb(filesystem, DebSpec{
		Name:            "core",
		Version:         "v0.12.3",
		Architecture:    "amd64",
		Maintainer:      "Lethean <dev@lethean.io>",
		Homepage:        "https://dappco.re",
		Description:     "Core developer CLI",
		LongDescription: "Builds, tests and releases CoreGO projects.",
		BinaryPath:      core.PathJoin(base, "core"),
	}, out).OK)

	control := readDebMember(t, filesystem.Read(out).Value.(string), "control.tar.gz", "./control")

	core.AssertTrue(t, core.Contains(control, "Package: core\n"))
	// The leading v must be gone: Debian versions start with a digit.
	core.AssertTrue(t, core.Contains(control, "Version: 0.12.3\n"))
	core.AssertTrue(t, core.Contains(control, "Architecture: amd64\n"))
	core.AssertTrue(t, core.Contains(control, "Maintainer: Lethean <dev@lethean.io>\n"))
	core.AssertTrue(t, core.Contains(control, "Homepage: https://dappco.re\n"))
	core.AssertTrue(t, core.Contains(control, "Description: Core developer CLI\n"))
	// Continuation lines carry a leading space or the stanza ends early.
	core.AssertTrue(t, core.Contains(control, "\n Builds, tests and releases CoreGO projects.\n"))
	// Installed-Size is in KiB rounded up, so a small binary is 1, never 0.
	core.AssertTrue(t, core.Contains(control, "Installed-Size: 1\n"))
	// Defaults applied.
	core.AssertTrue(t, core.Contains(control, "Section: devel\n"))
	core.AssertTrue(t, core.Contains(control, "Priority: optional\n"))
}

func TestDeb_BuildDeb_Payload_Good(t *core.T) {
	filesystem, base := newDebFS(t)
	out := core.PathJoin(base, "core.deb")

	core.AssertTrue(t, BuildDeb(filesystem, DebSpec{
		Name:         "core",
		Version:      "v0.12.3",
		Architecture: "amd64",
		BinaryPath:   core.PathJoin(base, "core"),
	}, out).OK)

	payload := readDebMember(t, filesystem.Read(out).Value.(string), "data.tar.gz", "./usr/bin/core")
	core.AssertEqual(t, "BINARYCONTENT", payload)
}

func TestDeb_BuildDeb_InstallPath_Good(t *core.T) {
	filesystem, base := newDebFS(t)
	out := core.PathJoin(base, "core.deb")

	core.AssertTrue(t, BuildDeb(filesystem, DebSpec{
		Name:         "core",
		Version:      "v0.12.3",
		Architecture: "amd64",
		BinaryPath:   core.PathJoin(base, "core"),
		InstallPath:  "/opt/lethean/bin/core",
	}, out).OK)

	payload := readDebMember(t, filesystem.Read(out).Value.(string), "data.tar.gz", "./opt/lethean/bin/core")
	core.AssertEqual(t, "BINARYCONTENT", payload)
}

func TestDeb_BuildDeb_Bad(t *core.T) {
	filesystem, base := newDebFS(t)
	binary := core.PathJoin(base, "core")
	out := core.PathJoin(base, "x.deb")

	core.AssertFalse(t, BuildDeb(nil, DebSpec{Name: "core", Version: "v1", BinaryPath: binary}, out).OK)
	core.AssertFalse(t, BuildDeb(filesystem, DebSpec{Version: "v1", BinaryPath: binary}, out).OK)
	core.AssertFalse(t, BuildDeb(filesystem, DebSpec{Name: "core", BinaryPath: binary}, out).OK)
	core.AssertFalse(t, BuildDeb(filesystem, DebSpec{Name: "core", Version: "v1"}, out).OK)
	// A version of just "v" normalises to empty and must not build.
	core.AssertFalse(t, BuildDeb(filesystem, DebSpec{Name: "core", Version: "v", BinaryPath: binary}, out).OK)
}

func TestDeb_BuildDeb_Ugly(t *core.T) {
	filesystem, base := newDebFS(t)

	// A missing binary must fail rather than produce an empty package.
	core.AssertFalse(t, BuildDeb(filesystem, DebSpec{
		Name: "core", Version: "v1.0.0", BinaryPath: core.PathJoin(base, "absent"),
	}, core.PathJoin(base, "a.deb")).OK)

	// So must an empty one: dpkg would install a zero-byte executable.
	core.AssertTrue(t, filesystem.Write(core.PathJoin(base, "empty"), "").OK)
	core.AssertFalse(t, BuildDeb(filesystem, DebSpec{
		Name: "core", Version: "v1.0.0", BinaryPath: core.PathJoin(base, "empty"),
	}, core.PathJoin(base, "b.deb")).OK)
}

// An ar member's content must be padded to an even length, or every member
// after an odd-sized one is misaligned and dpkg reports a corrupt archive.
func TestDeb_buildAr_Padding_Ugly(t *core.T) {
	archive := buildAr([]arMember{
		{Name: "odd", Content: "abc"},
		{Name: "next", Content: "xy"},
	})

	core.AssertTrue(t, core.Contains(archive, "abc\n"))
	// 8 magic + 60 header + 3 content + 1 pad + 60 header + 2 content.
	core.AssertEqual(t, 134, len(archive))
}

// readDebMember pulls one file out of a tarball nested in the .deb.
func readDebMember(t *core.T, deb, member, path string) string {
	body := arMemberContent(t, deb, member)

	decompressor, err := gzip.NewReader(bytes.NewReader([]byte(body)))
	core.AssertNoError(t, err)
	archive := tar.NewReader(decompressor)

	for {
		header, err := archive.Next()
		if err == stdio.EOF {
			break
		}
		core.AssertNoError(t, err)
		if header.Name != path {
			continue
		}
		content, err := stdio.ReadAll(archive)
		core.AssertNoError(t, err)
		return string(content)
	}

	t.Fatalf("member %s not found in %s", path, member)
	return ""
}

// arMemberContent walks the ar headers to find one member's bytes.
func arMemberContent(t *core.T, archive, name string) string {
	offset := len("!<arch>\n")
	for offset+60 <= len(archive) {
		header := archive[offset : offset+60]
		memberName := core.Trim(header[0:16])
		size := core.Trim(header[48:58])
		length := 0
		for _, digit := range size {
			length = length*10 + int(digit-'0')
		}
		start := offset + 60
		if memberName == name {
			return archive[start : start+length]
		}
		offset = start + length
		if length%2 == 1 {
			offset++
		}
	}
	t.Fatalf("ar member %s not found", name)
	return ""
}
