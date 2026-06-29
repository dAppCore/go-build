package signing

import (
	"context"
	"runtime"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	storage "dappco.re/go/build/pkg/storage"
)

func TestCodesign_MacOSSignerNameGood(t *testing.T) {
	s := NewMacOSSigner(MacOSConfig{Identity: "Developer ID Application: Test"})
	if !stdlibAssertEqual("codesign", s.Name()) {
		t.Fatalf("want %v, got %v", "codesign", s.Name())
	}

}

func TestCodesign_MacOSSignerAvailableGood(t *testing.T) {
	s := NewMacOSSigner(MacOSConfig{Identity: "Developer ID Application: Test"})

	if runtime.GOOS == "darwin" {
		// Just verify it doesn't panic
		_ = s.Available()
	} else {
		if s.Available() {
			t.Fatal("expected false")
		}

	}
}

func TestCodesign_MacOSSignerNoIdentityBad(t *testing.T) {
	s := NewMacOSSigner(MacOSConfig{})
	if s.Available() {
		t.Fatal("expected false")
	}

}

func TestCodesign_MacOSSignerSignBad(t *testing.T) {
	t.Run("fails when not available", func(t *testing.T) {
		if runtime.GOOS == "darwin" {
			t.Skip("skipping on macOS")
		}
		fs := storage.Local
		s := NewMacOSSigner(MacOSConfig{Identity: "test"})
		result := s.Sign(context.Background(), fs, "test")
		if result.OK {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(result.Error(), "only available on macOS") {
			t.Fatalf("expected %v to contain %v", result.Error(), "only available on macOS")
		}

	})
}

func TestCodesign_MacOSSignerNotarizeBad(t *testing.T) {
	fs := storage.Local
	t.Run("fails with missing credentials", func(t *testing.T) {
		s := NewMacOSSigner(MacOSConfig{})
		result := s.Notarize(context.Background(), fs, "test")
		if result.OK {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(result.Error(), "missing Apple credentials") {
			t.Fatalf("expected %v to contain %v", result.Error(), "missing Apple credentials")
		}

	})
}

func TestCodesign_MacOSSignerShouldNotarizeGood(t *testing.T) {
	s := NewMacOSSigner(MacOSConfig{Notarize: true})
	if !(s.ShouldNotarize()) {
		t.Fatal("expected true")
	}

	s2 := NewMacOSSigner(MacOSConfig{Notarize: false})
	if s2.ShouldNotarize() {
		t.Fatal("expected false")
	}

}

func TestCodesign_ResolveCodesignCliGood(t *testing.T) {
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "codesign")
	if result := ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	t.Setenv("PATH", "")

	result := resolveCodesignCli(fallbackPath)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	command := result.Value.(string)
	if !stdlibAssertEqual(fallbackPath, command) {
		t.Fatalf("want %v, got %v", fallbackPath, command)
	}

}

func TestCodesign_ResolveCodesignCliBad(t *testing.T) {
	t.Setenv("PATH", "")

	result := resolveCodesignCli(ax.Join(t.TempDir(), "missing-codesign"))
	if result.OK {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(result.Error(), "codesign tool not found") {
		t.Fatalf("expected %v to contain %v", result.Error(), "codesign tool not found")
	}

}

func TestCodesign_ResolveZipCliGood(t *testing.T) {
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "zip")
	if result := ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	t.Setenv("PATH", "")

	result := resolveZipCli(fallbackPath)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	command := result.Value.(string)
	if !stdlibAssertEqual(fallbackPath, command) {
		t.Fatalf("want %v, got %v", fallbackPath, command)
	}

}

func TestCodesign_ResolveZipCliBad(t *testing.T) {
	t.Setenv("PATH", "")

	result := resolveZipCli(ax.Join(t.TempDir(), "missing-zip"))
	if result.OK {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(result.Error(), "zip tool not found") {
		t.Fatalf("expected %v to contain %v", result.Error(), "zip tool not found")
	}

}

func TestCodesign_ResolveXcrunCliGood(t *testing.T) {
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "xcrun")
	if result := ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	t.Setenv("PATH", "")

	result := resolveXcrunCli(fallbackPath)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	command := result.Value.(string)
	if !stdlibAssertEqual(fallbackPath, command) {
		t.Fatalf("want %v, got %v", fallbackPath, command)
	}

}

func TestCodesign_ResolveXcrunCliBad(t *testing.T) {
	t.Setenv("PATH", "")

	result := resolveXcrunCli(ax.Join(t.TempDir(), "missing-xcrun"))
	if result.OK {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(result.Error(), "xcrun tool not found") {
		t.Fatalf("expected %v to contain %v", result.Error(), "xcrun tool not found")
	}

}

// --- AX-7 triplets (meaningful) ---
//
// MacOSSigner.Available/Sign gate on core.Env("GOOS") (not runtime.GOOS), so
// these tests set GOOS=darwin and supply fake codesign/zip/xcrun tools on PATH
// to drive the real command-construction paths deterministically on any host.

func TestCodesign_NewMacOSSigner_Good(t *core.T) {
	signer := NewMacOSSigner(MacOSConfig{Identity: "Developer ID Application: Acme (TEAM123)", Notarize: true})
	core.AssertNotNil(t, signer)
	core.AssertEqual(t, "codesign", signer.Name())
	core.AssertTrue(t, signer.ShouldNotarize())
}

func TestCodesign_NewMacOSSigner_Bad(t *core.T) {
	// An empty config yields a signer that never notarises and is unavailable.
	signer := NewMacOSSigner(MacOSConfig{})
	core.AssertFalse(t, signer.ShouldNotarize())
	core.AssertFalse(t, signer.Available())
}

func TestCodesign_NewMacOSSigner_Ugly(t *core.T) {
	// Edge case: an identity without notarisation credentials still constructs
	// a named signer; notarisation stays opt-in via the Notarize flag.
	signer := NewMacOSSigner(MacOSConfig{Identity: "Developer ID"})
	core.AssertEqual(t, "codesign", signer.Name())
	core.AssertFalse(t, signer.ShouldNotarize())
}

func TestCodesign_Available_Good(t *core.T) {
	// On (simulated) macOS with an identity and a resolvable codesign, the
	// signer reports available.
	t.Setenv("GOOS", "darwin")
	bin := t.TempDir()
	writeFakeSigningTool(t, bin, "codesign", fakeToolSuccess)
	t.Setenv("PATH", bin)

	core.AssertTrue(t, NewMacOSSigner(MacOSConfig{Identity: "Developer ID"}).Available())
}

func TestCodesign_Available_Bad(t *core.T) {
	// Off macOS the signer is never available, even with an identity set.
	t.Setenv("GOOS", "linux")
	core.AssertFalse(t, NewMacOSSigner(MacOSConfig{Identity: "Developer ID"}).Available())
}

func TestCodesign_Available_Ugly(t *core.T) {
	// Edge case: on macOS but with no identity configured -> unavailable, the
	// identity check short-circuits before resolving the tool.
	t.Setenv("GOOS", "darwin")
	core.AssertFalse(t, NewMacOSSigner(MacOSConfig{}).Available())
}

func TestCodesign_Sign_Good(t *core.T) {
	// Happy path: codesign resolves and exits 0.
	t.Setenv("GOOS", "darwin")
	bin := t.TempDir()
	writeFakeSigningTool(t, bin, "codesign", fakeToolSuccess)
	t.Setenv("PATH", bin)

	target := writeSigningTarget(t, "myapp")
	result := NewMacOSSigner(MacOSConfig{Identity: "Developer ID Application: Acme"}).
		Sign(core.Background(), storage.Local, target)
	core.AssertTrue(t, result.OK)
}

func TestCodesign_Sign_Bad(t *core.T) {
	// Failure path: not on macOS -> the platform guard error is returned and no
	// tool is invoked.
	t.Setenv("GOOS", "linux")
	result := NewMacOSSigner(MacOSConfig{Identity: "Developer ID"}).
		Sign(core.Background(), storage.Local, "myapp")
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "only available on macOS")
}

func TestCodesign_Sign_Ugly(t *core.T) {
	// Edge case: on macOS with an identity, but codesign exits non-zero (e.g.
	// the identity is not in the keychain) — the tool failure is surfaced.
	t.Setenv("GOOS", "darwin")
	bin := t.TempDir()
	writeFakeSigningTool(t, bin, "codesign", "#!/bin/sh\necho 'error: no identity found' >&2\nexit 1\n")
	t.Setenv("PATH", bin)

	target := writeSigningTarget(t, "myapp")
	result := NewMacOSSigner(MacOSConfig{Identity: "Missing Identity"}).
		Sign(core.Background(), storage.Local, target)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "codesign.Sign")
}

func TestCodesign_Sign_NoIdentity(t *core.T) {
	// On macOS with no identity, Sign reports the missing-identity error before
	// attempting any execution.
	t.Setenv("GOOS", "darwin")
	bin := t.TempDir()
	writeFakeSigningTool(t, bin, "codesign", fakeToolSuccess)
	t.Setenv("PATH", bin)

	result := NewMacOSSigner(MacOSConfig{}).Sign(core.Background(), storage.Local, "myapp")
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "identity not configured")
}

func TestCodesign_Notarize_Good(t *core.T) {
	// Happy path: full zip -> notarytool submit -> stapler staple, all exiting 0.
	t.Setenv("GOOS", "darwin")
	bin := t.TempDir()
	writeFakeSigningTool(t, bin, "zip", fakeToolSuccess)
	writeFakeSigningTool(t, bin, "xcrun", fakeToolSuccess)
	t.Setenv("PATH", bin)

	target := writeSigningTarget(t, "myapp")
	result := NewMacOSSigner(MacOSConfig{
		AppleID: "dev@example.com", TeamID: "TEAM123", AppPassword: "app-specific",
	}).Notarize(core.Background(), storage.Local, target)
	core.AssertTrue(t, result.OK)
}

func TestCodesign_Notarize_Bad(t *core.T) {
	// Failure path: missing Apple credentials short-circuits before any exec.
	result := NewMacOSSigner(MacOSConfig{}).Notarize(core.Background(), storage.Local, "myapp")
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "missing Apple credentials")
}

func TestCodesign_Notarize_Ugly(t *core.T) {
	// Edge case: credentials present and the zip succeeds, but notarytool exits
	// non-zero (rejected submission) — the notarisation error is surfaced.
	t.Setenv("GOOS", "darwin")
	bin := t.TempDir()
	writeFakeSigningTool(t, bin, "zip", fakeToolSuccess)
	writeFakeSigningTool(t, bin, "xcrun", "#!/bin/sh\necho 'notarytool: rejected' >&2\nexit 1\n")
	t.Setenv("PATH", bin)

	target := writeSigningTarget(t, "myapp")
	result := NewMacOSSigner(MacOSConfig{
		AppleID: "dev@example.com", TeamID: "TEAM123", AppPassword: "app-specific",
	}).Notarize(core.Background(), storage.Local, target)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "notarization failed")
}

func TestCodesign_Notarize_StaplerFails(t *core.T) {
	// Submission succeeds but stapling the ticket fails: the staple error is
	// surfaced. The fake xcrun succeeds for notarytool and fails for stapler.
	t.Setenv("GOOS", "darwin")
	bin := t.TempDir()
	writeFakeSigningTool(t, bin, "zip", fakeToolSuccess)
	writeFakeSigningTool(t, bin, "xcrun",
		"#!/bin/sh\ncase \"$1\" in\n  stapler) echo 'stapler: ticket not found' >&2; exit 1;;\n  *) exit 0;;\nesac\n")
	t.Setenv("PATH", bin)

	target := writeSigningTarget(t, "myapp")
	result := NewMacOSSigner(MacOSConfig{
		AppleID: "dev@example.com", TeamID: "TEAM123", AppPassword: "app-specific",
	}).Notarize(core.Background(), storage.Local, target)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "failed to staple")
}

func TestCodesign_Notarize_ZipMissing(t *core.T) {
	// With credentials present but no zip tool resolvable, notarisation fails at
	// the packaging step.
	t.Setenv("GOOS", "darwin")
	t.Setenv("PATH", t.TempDir()) // empty: defeats fallback for zip? see below
	result := NewMacOSSigner(MacOSConfig{
		AppleID: "dev@example.com", TeamID: "TEAM123", AppPassword: "app-specific",
	}).Notarize(core.Background(), storage.Local, "myapp")
	// zip and xcrun resolve via hard-coded fallbacks on a real macOS host, so we
	// only assert the outcome is a failure originating from notarisation rather
	// than asserting a specific missing-tool message.
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "codesign.Notarize")
}

func TestCodesign_ShouldNotarize_Good(t *core.T) {
	core.AssertTrue(t, NewMacOSSigner(MacOSConfig{Notarize: true}).ShouldNotarize())
}

func TestCodesign_ShouldNotarize_Bad(t *core.T) {
	core.AssertFalse(t, NewMacOSSigner(MacOSConfig{Notarize: false}).ShouldNotarize())
}

func TestCodesign_ShouldNotarize_Ugly(t *core.T) {
	// Edge case: a zero-value signer (no constructor) defaults to not notarising.
	signer := &MacOSSigner{}
	core.AssertFalse(t, signer.ShouldNotarize())
}
