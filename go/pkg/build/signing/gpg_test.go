package signing

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	storage "dappco.re/go/build/pkg/storage"
)

func TestGPG_GPGSignerNameGood(t *testing.T) {
	s := NewGPGSigner("ABCD1234")
	if !stdlibAssertEqual("gpg", s.Name()) {
		t.Fatalf("want %v, got %v", "gpg", s.Name())
	}

}

func TestGPG_GPGSignerAvailableGood(t *testing.T) {
	s := NewGPGSigner("ABCD1234")
	available := s.Available()
	if available && s.Name() == "" {
		t.Fatal("expected available signer to have a name")
	}
	if !stdlibAssertEqual("gpg", s.Name()) {
		t.Fatalf("want %v, got %v", "gpg", s.Name())
	}
}

func TestGPG_GPGSignerNoKeyBad(t *testing.T) {
	s := NewGPGSigner("")
	if s.Available() {
		t.Fatal("expected false")
	}

}

func TestGPG_GPGSignerSignBad(t *testing.T) {
	fs := storage.Local
	t.Run("fails when no key", func(t *testing.T) {
		s := NewGPGSigner("")
		result := s.Sign(context.Background(), fs, "test.txt")
		if result.OK {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(result.Error(), "not available or key not configured") {
			t.Fatalf("expected %v to contain %v", result.Error(), "not available or key not configured")
		}

	})
}

func TestGPG_ResolveGpgCliGood(t *testing.T) {
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "gpg")
	if result := ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	t.Setenv("PATH", "")

	result := resolveGpgCli(fallbackPath)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	command := result.Value.(string)
	if !stdlibAssertEqual(fallbackPath, command) {
		t.Fatalf("want %v, got %v", fallbackPath, command)
	}

}

func TestGPG_ResolveGpgCliBad(t *testing.T) {
	t.Setenv("PATH", "")

	result := resolveGpgCli(ax.Join(t.TempDir(), "missing-gpg"))
	if result.OK {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(result.Error(), "gpg CLI not found") {
		t.Fatalf("expected %v to contain %v", result.Error(), "gpg CLI not found")
	}

}

// --- AX-7 triplets (meaningful) ---

func TestGpg_NewGPGSigner_Good(t *core.T) {
	// Constructor stores the supplied key id and yields a usable signer name.
	signer := NewGPGSigner("ABCD1234")
	core.AssertNotNil(t, signer)
	core.AssertEqual(t, "ABCD1234", signer.KeyID)
	core.AssertEqual(t, "gpg", signer.Name())
}

func TestGpg_NewGPGSigner_Bad(t *core.T) {
	// An empty key id produces a signer that reports itself unavailable.
	signer := NewGPGSigner("")
	core.AssertEqual(t, "", signer.KeyID)
	core.AssertFalse(t, signer.Available())
}

func TestGpg_NewGPGSigner_Ugly(t *core.T) {
	// Edge case: a fingerprint-style key with spaces is preserved verbatim —
	// the signer does not normalise or trim it.
	const fingerprint = "ABCD 1234 EF56 7890"
	signer := NewGPGSigner(fingerprint)
	core.AssertEqual(t, fingerprint, signer.KeyID)
}

func TestGpg_Name_Good(t *core.T) {
	core.AssertEqual(t, "gpg", NewGPGSigner("KEY").Name())
}

func TestGpg_Name_Bad(t *core.T) {
	// Name is identity-independent: even a keyless signer reports "gpg".
	core.AssertEqual(t, "gpg", NewGPGSigner("").Name())
}

func TestGpg_Name_Ugly(t *core.T) {
	// Edge case: a zero-value struct (no constructor) still names itself.
	signer := &GPGSigner{}
	core.AssertEqual(t, "gpg", signer.Name())
}

func TestGpg_Available_Good(t *core.T) {
	// With a key configured and a resolvable gpg on PATH, the signer is
	// available.
	bin := t.TempDir()
	writeFakeSigningTool(t, bin, "gpg", fakeToolSuccess)
	t.Setenv("PATH", bin)

	core.AssertTrue(t, NewGPGSigner("ABCD1234").Available())
}

func TestGpg_Available_Bad(t *core.T) {
	// No key -> unavailable, regardless of whether gpg is installed.
	core.AssertFalse(t, NewGPGSigner("").Available())
}

func TestGpg_Available_Ugly(t *core.T) {
	// Edge case: with a key configured, availability is governed entirely by
	// whether the gpg CLI resolves. Tie the assertion to the real resolver so
	// it holds whether or not gpg is installed on the host.
	signer := NewGPGSigner("ABCD1234")
	core.AssertEqual(t, resolveGpgCli().OK, signer.Available())
}

func TestGpg_Sign_Good(t *core.T) {
	// Happy path: a resolvable gpg that exits 0 produces a successful signature.
	bin := t.TempDir()
	writeFakeSigningTool(t, bin, "gpg", fakeToolSuccess)
	t.Setenv("PATH", bin)

	target := writeSigningTarget(t, "CHECKSUMS.txt")
	result := NewGPGSigner("ABCD1234").Sign(core.Background(), storage.Local, target)
	core.AssertTrue(t, result.OK)
}

func TestGpg_Sign_Bad(t *core.T) {
	// Failure path: no key configured short-circuits before any exec.
	result := NewGPGSigner("").Sign(core.Background(), storage.Local, "anything.txt")
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "not available or key not configured")
}

func TestGpg_Sign_Ugly(t *core.T) {
	// Edge case: gpg resolves but exits non-zero (e.g. unknown key) — the tool's
	// failure is surfaced as a signing error.
	bin := t.TempDir()
	writeFakeSigningTool(t, bin, "gpg", "#!/bin/sh\necho 'gpg: signing failed: no secret key' >&2\nexit 2\n")
	t.Setenv("PATH", bin)

	target := writeSigningTarget(t, "CHECKSUMS.txt")
	result := NewGPGSigner("MISSINGKEY").Sign(core.Background(), storage.Local, target)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "gpg.Sign")
}
