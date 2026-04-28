package signing

import (
	"context"
	"runtime"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/io"
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

		err := signer.Sign(context.Background(), io.Local, "test.exe")
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "only available on Windows") {
			t.Fatalf("expected %v to contain %v", err.Error(), "only available on Windows")
		}

	})
}

func TestSigntool_Sign_Good(t *testing.T) {
	signer := NewWindowsSigner(WindowsConfig{Signtool: true, Certificate: "cert.pfx"})
	err := signer.Sign(context.Background(), io.Local, "test.exe")
	if runtime.GOOS != "windows" {
		if err == nil {
			t.Fatal("expected non-Windows platform guard")
		}
		return
	}
	if err != nil && !stdlibAssertContains(err.Error(), "signtool") {
		t.Fatalf("expected signtool-related result, got %v", err)
	}
}

func TestSigntool_ResolveSigntoolCli_Good(t *testing.T) {
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "signtool.exe")
	if err := ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Setenv("PATH", "")

	command, err := resolveSigntoolCli(fallbackPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(fallbackPath, command) {
		t.Fatalf("want %v, got %v", fallbackPath, command)
	}

}

func TestSigntool_ResolveSigntoolCli_Bad(t *testing.T) {
	t.Setenv("PATH", "")

	_, err := resolveSigntoolCli(ax.Join(t.TempDir(), "missing-signtool.exe"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "signtool tool not found") {
		t.Fatalf("expected %v to contain %v", err.Error(), "signtool tool not found")
	}

}
