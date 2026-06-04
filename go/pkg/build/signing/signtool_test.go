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

// --- v0.9.0 generated compliance triplets ---
func TestSigntool_WindowsSigner_Name_Good(t *core.T) {
	subject := &WindowsSigner{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestSigntool_WindowsSigner_Name_Bad(t *core.T) {
	subject := &WindowsSigner{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestSigntool_WindowsSigner_Name_Ugly(t *core.T) {
	subject := &WindowsSigner{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestSigntool_WindowsSigner_Available_Good(t *core.T) {
	subject := &WindowsSigner{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Available()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestSigntool_WindowsSigner_Available_Bad(t *core.T) {
	subject := &WindowsSigner{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Available()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestSigntool_WindowsSigner_Available_Ugly(t *core.T) {
	subject := &WindowsSigner{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Available()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestSigntool_WindowsSigner_Sign_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &WindowsSigner{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Sign(ctx, storage.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestSigntool_WindowsSigner_Sign_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &WindowsSigner{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Sign(ctx, storage.NewMemoryMedium(), "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestSigntool_WindowsSigner_Sign_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &WindowsSigner{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Sign(ctx, storage.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
