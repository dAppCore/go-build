package signing

import (
	"context"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/io"
)

func TestGPG_GPGSignerName_Good(t *testing.T) {
	s := NewGPGSigner("ABCD1234")
	if !stdlibAssertEqual("gpg", s.Name()) {
		t.Fatalf("want %v, got %v", "gpg", s.Name())
	}

}

func TestGPG_GPGSignerAvailable_Good(t *testing.T) {
	s := NewGPGSigner("ABCD1234")
	_ = s.Available()
}

func TestGPG_GPGSignerNoKey_Bad(t *testing.T) {
	s := NewGPGSigner("")
	if s.Available() {
		t.Fatal("expected false")
	}

}

func TestGPG_GPGSignerSign_Bad(t *testing.T) {
	fs := io.Local
	t.Run("fails when no key", func(t *testing.T) {
		s := NewGPGSigner("")
		err := s.Sign(context.Background(), fs, "test.txt")
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "not available or key not configured") {
			t.Fatalf("expected %v to contain %v", err.Error(), "not available or key not configured")
		}

	})
}

func TestGPG_ResolveGpgCli_Good(t *testing.T) {
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "gpg")
	if err := ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Setenv("PATH", "")

	command, err := resolveGpgCli(fallbackPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(fallbackPath, command) {
		t.Fatalf("want %v, got %v", fallbackPath, command)
	}

}

func TestGPG_ResolveGpgCli_Bad(t *testing.T) {
	t.Setenv("PATH", "")

	_, err := resolveGpgCli(ax.Join(t.TempDir(), "missing-gpg"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "gpg CLI not found") {
		t.Fatalf("expected %v to contain %v", err.Error(), "gpg CLI not found")
	}

}
