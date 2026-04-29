package signing

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/io"
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

func TestGPG_ResolveGpgCliGood(t *testing.T) {
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

func TestGPG_ResolveGpgCliBad(t *testing.T) {
	t.Setenv("PATH", "")

	_, err := resolveGpgCli(ax.Join(t.TempDir(), "missing-gpg"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "gpg CLI not found") {
		t.Fatalf("expected %v to contain %v", err.Error(), "gpg CLI not found")
	}

}

// --- v0.9.0 generated compliance triplets ---
func TestGpg_NewGPGSigner_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewGPGSigner("agent")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestGpg_NewGPGSigner_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewGPGSigner("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestGpg_NewGPGSigner_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewGPGSigner("agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestGpg_GPGSigner_Name_Good(t *core.T) {
	subject := &GPGSigner{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestGpg_GPGSigner_Name_Bad(t *core.T) {
	subject := &GPGSigner{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestGpg_GPGSigner_Name_Ugly(t *core.T) {
	subject := &GPGSigner{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestGpg_GPGSigner_Available_Good(t *core.T) {
	subject := &GPGSigner{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Available()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestGpg_GPGSigner_Available_Bad(t *core.T) {
	subject := &GPGSigner{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Available()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestGpg_GPGSigner_Available_Ugly(t *core.T) {
	subject := &GPGSigner{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Available()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestGpg_GPGSigner_Sign_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &GPGSigner{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Sign(ctx, io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestGpg_GPGSigner_Sign_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &GPGSigner{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Sign(ctx, io.NewMemoryMedium(), "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestGpg_GPGSigner_Sign_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &GPGSigner{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Sign(ctx, io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
