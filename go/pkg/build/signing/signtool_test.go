package signing

import (
	"context"
	"runtime"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	storage "dappco.re/go/build/pkg/storage"
)

func TestSigntool_NewWindowsSigner_Good(t *testing.T) {
	signer := NewWindowsSigner(WindowsConfig{
		Signtool:    true,
		Certificate: "cert.pfx",
		Password:    "secret",
	})
	if !stdlibAssertEqual("signtool", signer.Name()) {
		t.Fatalf("want %v, got %v", "signtool", signer.Name())
	}

}

func TestSigntool_NewWindowsSigner_Bad(t *testing.T) {
	t.Run("available is false when the explicit toggle disables signtool", func(t *testing.T) {
		signer := NewWindowsSigner(WindowsConfig{
			Signtool:         false,
			Certificate:      "cert.pfx",
			signtoolExplicit: true,
		})
		if signer.Available() {
			t.Fatal("expected false")
		}

	})
}

func TestSigntool_NewWindowsSigner_Ugly(t *testing.T) {
	t.Run("available is false without a certificate", func(t *testing.T) {
		signer := NewWindowsSigner(WindowsConfig{Signtool: true})
		if signer.Available() {
			t.Fatal("expected false")
		}

	})
}

func TestSigntool_Available_Good(t *testing.T) {
	signer := NewWindowsSigner(WindowsConfig{Signtool: true, Certificate: "cert.pfx"})
	if runtime.GOOS != "windows" {
		if signer.Available() {
			t.Fatal("expected signtool to be unavailable on non-Windows hosts")
		}
		return
	}
	if !stdlibAssertEqual("signtool", signer.Name()) {
		t.Fatalf("want %v, got %v", "signtool", signer.Name())
	}
}

func TestSigntool_Sign_Bad(t *testing.T) {
	t.Run("returns the platform guard on non-Windows hosts", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("this assertion is specific to non-Windows hosts")
		}

		signer := NewWindowsSigner(WindowsConfig{
			Signtool:    true,
			Certificate: "cert.pfx",
		})

		result := signer.Sign(context.Background(), storage.Local, "test.exe")
		if result.OK {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(result.Error(), "only available on Windows") {
			t.Fatalf("expected %v to contain %v", result.Error(), "only available on Windows")
		}

	})
}

func TestSigntool_Sign_Good(t *testing.T) {
	signer := NewWindowsSigner(WindowsConfig{Signtool: true, Certificate: "cert.pfx"})
	result := signer.Sign(context.Background(), storage.Local, "test.exe")
	if runtime.GOOS != "windows" {
		if result.OK {
			t.Fatal("expected non-Windows platform guard")
		}
		return
	}
	if !result.OK && !stdlibAssertContains(result.Error(), "signtool") {
		t.Fatalf("expected signtool-related result, got %v", result.Error())
	}
}

func TestSigntool_ResolveSigntoolCliGood(t *testing.T) {
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "signtool.exe")
	if result := ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	t.Setenv("PATH", "")

	result := resolveSigntoolCli(fallbackPath)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	command := result.Value.(string)
	if !stdlibAssertEqual(fallbackPath, command) {
		t.Fatalf("want %v, got %v", fallbackPath, command)
	}

}

func TestSigntool_ResolveSigntoolCliBad(t *testing.T) {
	t.Setenv("PATH", "")

	result := resolveSigntoolCli(ax.Join(t.TempDir(), "missing-signtool.exe"))
	if result.OK {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(result.Error(), "signtool tool not found") {
		t.Fatalf("expected %v to contain %v", result.Error(), "signtool tool not found")
	}

}

// --- AX-7 triplets (meaningful) ---
//
// WindowsSigner.Available/Sign gate on runtime.GOOS == "windows", which cannot
// be overridden in-process. On non-Windows hosts the signer is always
// unavailable and Sign returns the platform guard; the real signtool execution
// path is therefore covered only on Windows and is skipped here (see report).
// These tests assert the host-independent logic: the signtool toggle, the
// certificate requirement, naming, and the validation/error branches.

func TestSigntool_NewWindowsSigner_Constructed_Good(t *core.T) {
	signer := NewWindowsSigner(WindowsConfig{Signtool: true, Certificate: "cert.pfx", Password: "secret"})
	core.AssertNotNil(t, signer)
	core.AssertEqual(t, "signtool", signer.Name())
}

func TestSigntool_Name_Good(t *core.T) {
	core.AssertEqual(t, "signtool", NewWindowsSigner(WindowsConfig{Certificate: "cert.pfx"}).Name())
}

func TestSigntool_Name_Bad(t *core.T) {
	// Name is configuration-independent: a disabled signer still names itself.
	signer := NewWindowsSigner(WindowsConfig{})
	signer.config.SetSigntool(false)
	core.AssertEqual(t, "signtool", signer.Name())
}

func TestSigntool_Name_Ugly(t *core.T) {
	// Edge case: a zero-value struct names itself without a constructor.
	core.AssertEqual(t, "signtool", (&WindowsSigner{}).Name())
}

func TestSigntool_Available_Disabled_Bad(t *core.T) {
	// The explicit signtool toggle disables availability regardless of OS.
	signer := NewWindowsSigner(WindowsConfig{Certificate: "cert.pfx"})
	signer.config.SetSigntool(false)
	core.AssertFalse(t, signer.Available())
}

func TestSigntool_Available_NoCertificate_Ugly(t *core.T) {
	// With signtool enabled but no certificate, the signer is unavailable; on a
	// non-Windows host it is unavailable in any case, so the result is false
	// either way.
	core.AssertFalse(t, NewWindowsSigner(WindowsConfig{Signtool: true}).Available())
}

func TestSigntool_Available_TracksResolution_Good(t *core.T) {
	// On a non-Windows host the signer is never available even when fully
	// configured; on Windows availability tracks signtool resolution. The
	// assertion is pinned to the host so it holds on both.
	signer := NewWindowsSigner(WindowsConfig{Signtool: true, Certificate: "cert.pfx"})
	if runtime.GOOS != "windows" {
		core.AssertFalse(t, signer.Available())
		return
	}
	core.AssertEqual(t, resolveSigntoolCli().OK, signer.Available())
}

func TestSigntool_Sign_ToggleDisabled_Bad(t *core.T) {
	// The explicit toggle off makes the signer unavailable; Sign then reports a
	// guard error rather than attempting to run signtool.
	signer := NewWindowsSigner(WindowsConfig{Certificate: "cert.pfx"})
	signer.config.SetSigntool(false)
	result := signer.Sign(context.Background(), storage.Local, "app.exe")
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "signtool")
}

func TestSigntool_Sign_NoCertificate_Ugly(t *core.T) {
	// Edge case: enabled with no certificate. On non-Windows the platform guard
	// fires first; on Windows the missing-certificate guard fires. Both are
	// failures with a signtool-prefixed error.
	signer := NewWindowsSigner(WindowsConfig{Signtool: true})
	result := signer.Sign(context.Background(), storage.Local, "app.exe")
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "signtool.Sign")
}
