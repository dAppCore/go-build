package signing

import (
	"context"
	"runtime"

	core "dappco.re/go"
	coreio "dappco.re/go/build/pkg/storage"
)

// SignBinaries/NotarizeBinaries dispatch on runtime.GOOS (the real host), while
// the macOS signer's availability gates on core.Env("GOOS"). On macOS hosts we
// drive the real sign/notarise dispatch with fake tools; the signer-method
// command construction is covered host-independently in codesign_test.go.

func TestSign_SignBinaries_Good(t *core.T) {
	// Disabled config is a no-op success regardless of platform or artifacts.
	cfg := SignConfig{Enabled: false}
	result := SignBinaries(context.Background(), coreio.Local, cfg,
		[]Artifact{{Path: "/dist/darwin/app", OS: "darwin", Arch: "arm64"}})
	core.AssertTrue(t, result.OK)
}

func TestSign_SignBinaries_Bad(t *core.T) {
	// A signer that is available but fails propagates a "failed to sign" error.
	// Only reachable when the host signer dispatches and is available, so this
	// runs on macOS with a failing fake codesign and is otherwise a skip.
	if runtime.GOOS != "darwin" {
		t.Skip("SignBinaries only dispatches a signer on macOS/Windows hosts")
	}
	t.Setenv("GOOS", "darwin")
	bin := t.TempDir()
	writeFakeSigningTool(t, bin, "codesign", "#!/bin/sh\nexit 1\n")
	t.Setenv("PATH", bin)

	target := writeSigningTarget(t, "app")
	cfg := SignConfig{Enabled: true, MacOS: MacOSConfig{Identity: "Developer ID"}}
	result := SignBinaries(context.Background(), coreio.Local, cfg,
		[]Artifact{{Path: target, OS: "darwin", Arch: "arm64"}})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "failed to sign")
}

func TestSign_SignBinaries_Ugly(t *core.T) {
	// Edge case: enabled but the signer is unavailable (no identity) -> silently
	// skipped as a success; no artifact is signed.
	cfg := SignConfig{Enabled: true, MacOS: MacOSConfig{Identity: ""}}
	result := SignBinaries(context.Background(), coreio.Local, cfg,
		[]Artifact{{Path: "/dist/darwin/app", OS: "darwin", Arch: "arm64"}})
	core.AssertTrue(t, result.OK)
}

func TestSign_SignBinaries_SignsDarwinArtifacts(t *core.T) {
	// On a macOS host with a resolvable codesign, darwin artifacts are signed
	// and the call succeeds while non-darwin artifacts are skipped.
	if runtime.GOOS != "darwin" {
		t.Skip("codesign dispatch happens only on macOS hosts")
	}
	t.Setenv("GOOS", "darwin")
	bin := t.TempDir()
	writeFakeSigningTool(t, bin, "codesign", fakeToolSuccess)
	t.Setenv("PATH", bin)

	target := writeSigningTarget(t, "app")
	cfg := SignConfig{Enabled: true, MacOS: MacOSConfig{Identity: "Developer ID"}}
	result := SignBinaries(context.Background(), coreio.Local, cfg, []Artifact{
		{Path: target, OS: "darwin", Arch: "arm64"},
		{Path: "/dist/linux/app", OS: "linux", Arch: "amd64"},
	})
	core.AssertTrue(t, result.OK)
}

func TestSign_NotarizeBinaries_Good(t *core.T) {
	// On a macOS host with credentials and fake zip/xcrun, notarisation of a
	// darwin artifact succeeds.
	if runtime.GOOS != "darwin" {
		t.Skip("notarisation dispatch happens only on macOS hosts")
	}
	t.Setenv("GOOS", "darwin")
	bin := t.TempDir()
	writeFakeSigningTool(t, bin, "codesign", fakeToolSuccess)
	writeFakeSigningTool(t, bin, "zip", fakeToolSuccess)
	writeFakeSigningTool(t, bin, "xcrun", fakeToolSuccess)
	t.Setenv("PATH", bin)

	target := writeSigningTarget(t, "app")
	cfg := SignConfig{Enabled: true, MacOS: MacOSConfig{
		Identity: "Developer ID", Notarize: true,
		AppleID: "dev@example.com", TeamID: "TEAM123", AppPassword: "app-specific",
	}}
	result := NotarizeBinaries(context.Background(), coreio.Local, cfg,
		[]Artifact{{Path: target, OS: "darwin", Arch: "arm64"}})
	core.AssertTrue(t, result.OK)
}

func TestSign_NotarizeBinaries_Bad(t *core.T) {
	// On a macOS host, notarisation requested but codesign unavailable (no
	// identity) is a hard failure.
	if runtime.GOOS != "darwin" {
		t.Skip("notarisation availability check is macOS-specific")
	}
	t.Setenv("GOOS", "linux") // makes the macOS signer report unavailable
	cfg := SignConfig{Enabled: true, MacOS: MacOSConfig{Notarize: true}}
	result := NotarizeBinaries(context.Background(), coreio.Local, cfg,
		[]Artifact{{Path: "/dist/darwin/app", OS: "darwin", Arch: "arm64"}})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "codesign not available")
}

func TestSign_NotarizeBinaries_Ugly(t *core.T) {
	// Edge case: notarisation disabled in config is a no-op success even with a
	// darwin artifact present.
	cfg := SignConfig{Enabled: true, MacOS: MacOSConfig{Notarize: false}}
	result := NotarizeBinaries(context.Background(), coreio.Local, cfg,
		[]Artifact{{Path: "/dist/darwin/app", OS: "darwin", Arch: "arm64"}})
	core.AssertTrue(t, result.OK)
}

func TestSign_SignChecksums_Good(t *core.T) {
	// With a GPG key and a resolvable gpg, the checksums file is signed.
	bin := t.TempDir()
	writeFakeSigningTool(t, bin, "gpg", fakeToolSuccess)
	t.Setenv("PATH", bin)

	target := writeSigningTarget(t, "CHECKSUMS.txt")
	cfg := SignConfig{Enabled: true, GPG: GPGConfig{Key: "ABCD1234"}}
	result := SignChecksums(context.Background(), coreio.Local, cfg, target)
	core.AssertTrue(t, result.OK)
}

func TestSign_SignChecksums_Bad(t *core.T) {
	// A configured GPG key but a gpg that exits non-zero yields a checksum
	// signing failure.
	bin := t.TempDir()
	writeFakeSigningTool(t, bin, "gpg", "#!/bin/sh\nexit 3\n")
	t.Setenv("PATH", bin)

	target := writeSigningTarget(t, "CHECKSUMS.txt")
	cfg := SignConfig{Enabled: true, GPG: GPGConfig{Key: "ABCD1234"}}
	result := SignChecksums(context.Background(), coreio.Local, cfg, target)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "failed to sign checksums file")
}

func TestSign_SignChecksums_Ugly(t *core.T) {
	// Edge case: no GPG key -> the signer is unavailable and signing is silently
	// skipped as a success.
	cfg := SignConfig{Enabled: true, GPG: GPGConfig{Key: ""}}
	result := SignChecksums(context.Background(), coreio.Local, cfg, "/tmp/CHECKSUMS.txt")
	core.AssertTrue(t, result.OK)
}

func TestSign_signArtifactsWithSigner_Good(t *core.T) {
	// The helper signs only artifacts matching the target OS, in order.
	signer := &mockSigner{name: "mock", available: true}
	result := signArtifactsWithSigner(context.Background(), coreio.Local, signer, "darwin", []Artifact{
		{Path: "/dist/darwin/a", OS: "darwin", Arch: "arm64"},
		{Path: "/dist/linux/b", OS: "linux", Arch: "amd64"},
		{Path: "/dist/darwin/c", OS: "darwin", Arch: "amd64"},
	})
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, []string{"/dist/darwin/a", "/dist/darwin/c"}, signer.signedPaths)
}

func TestSign_signArtifactsWithSigner_Bad(t *core.T) {
	// A signer failure aborts and is wrapped with the failing artifact path.
	signer := &mockSigner{name: "mock", available: true, signError: core.NewError("boom")}
	result := signArtifactsWithSigner(context.Background(), coreio.Local, signer, "darwin",
		[]Artifact{{Path: "/dist/darwin/a", OS: "darwin", Arch: "arm64"}})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "/dist/darwin/a")
}

func TestSign_signArtifactsWithSigner_Ugly(t *core.T) {
	// Edge case: no artifact matches the target OS -> nothing is signed and the
	// call succeeds.
	signer := &mockSigner{name: "mock", available: true}
	result := signArtifactsWithSigner(context.Background(), coreio.Local, signer, "windows",
		[]Artifact{{Path: "/dist/linux/a", OS: "linux", Arch: "amd64"}})
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, 0, len(signer.signedPaths))
}
