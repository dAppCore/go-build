package signing

import (
	"context"
	"runtime"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/io"
)

func TestCodesign_MacOSSignerName_Good(t *testing.T) {
	s := NewMacOSSigner(MacOSConfig{Identity: "Developer ID Application: Test"})
	if !stdlibAssertEqual("codesign", s.Name()) {
		t.Fatalf("want %v, got %v", "codesign", s.Name())
	}

}

func TestCodesign_MacOSSignerAvailable_Good(t *testing.T) {
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

func TestCodesign_MacOSSignerNoIdentity_Bad(t *testing.T) {
	s := NewMacOSSigner(MacOSConfig{})
	if s.Available() {
		t.Fatal("expected false")
	}

}

func TestCodesign_MacOSSignerSign_Bad(t *testing.T) {
	t.Run("fails when not available", func(t *testing.T) {
		if runtime.GOOS == "darwin" {
			t.Skip("skipping on macOS")
		}
		fs := io.Local
		s := NewMacOSSigner(MacOSConfig{Identity: "test"})
		err := s.Sign(context.Background(), fs, "test")
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "only available on macOS") {
			t.Fatalf("expected %v to contain %v", err.Error(), "only available on macOS")
		}

	})
}

func TestCodesign_MacOSSignerNotarize_Bad(t *testing.T) {
	fs := io.Local
	t.Run("fails with missing credentials", func(t *testing.T) {
		s := NewMacOSSigner(MacOSConfig{})
		err := s.Notarize(context.Background(), fs, "test")
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "missing Apple credentials") {
			t.Fatalf("expected %v to contain %v", err.Error(), "missing Apple credentials")
		}

	})
}

func TestCodesign_MacOSSignerShouldNotarize_Good(t *testing.T) {
	s := NewMacOSSigner(MacOSConfig{Notarize: true})
	if !(s.ShouldNotarize()) {
		t.Fatal("expected true")
	}

	s2 := NewMacOSSigner(MacOSConfig{Notarize: false})
	if s2.ShouldNotarize() {
		t.Fatal("expected false")
	}

}

func TestCodesign_ResolveCodesignCli_Good(t *testing.T) {
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "codesign")
	if err := ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Setenv("PATH", "")

	command, err := resolveCodesignCli(fallbackPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(fallbackPath, command) {
		t.Fatalf("want %v, got %v", fallbackPath, command)
	}

}

func TestCodesign_ResolveCodesignCli_Bad(t *testing.T) {
	t.Setenv("PATH", "")

	_, err := resolveCodesignCli(ax.Join(t.TempDir(), "missing-codesign"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "codesign tool not found") {
		t.Fatalf("expected %v to contain %v", err.Error(), "codesign tool not found")
	}

}

func TestCodesign_ResolveZipCli_Good(t *testing.T) {
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "zip")
	if err := ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Setenv("PATH", "")

	command, err := resolveZipCli(fallbackPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(fallbackPath, command) {
		t.Fatalf("want %v, got %v", fallbackPath, command)
	}

}

func TestCodesign_ResolveZipCli_Bad(t *testing.T) {
	t.Setenv("PATH", "")

	_, err := resolveZipCli(ax.Join(t.TempDir(), "missing-zip"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "zip tool not found") {
		t.Fatalf("expected %v to contain %v", err.Error(), "zip tool not found")
	}

}

func TestCodesign_ResolveXcrunCli_Good(t *testing.T) {
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "xcrun")
	if err := ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Setenv("PATH", "")

	command, err := resolveXcrunCli(fallbackPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(fallbackPath, command) {
		t.Fatalf("want %v, got %v", fallbackPath, command)
	}

}

func TestCodesign_ResolveXcrunCli_Bad(t *testing.T) {
	t.Setenv("PATH", "")

	_, err := resolveXcrunCli(ax.Join(t.TempDir(), "missing-xcrun"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "xcrun tool not found") {
		t.Fatalf("expected %v to contain %v", err.Error(), "xcrun tool not found")
	}

}
