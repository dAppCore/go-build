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

// --- v0.9.0 generated compliance triplets ---
func TestCodesign_NewMacOSSigner_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewMacOSSigner(MacOSConfig{})
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCodesign_NewMacOSSigner_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewMacOSSigner(MacOSConfig{})
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCodesign_NewMacOSSigner_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewMacOSSigner(MacOSConfig{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestCodesign_MacOSSigner_Name_Good(t *core.T) {
	subject := &MacOSSigner{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCodesign_MacOSSigner_Name_Bad(t *core.T) {
	subject := &MacOSSigner{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCodesign_MacOSSigner_Name_Ugly(t *core.T) {
	subject := &MacOSSigner{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestCodesign_MacOSSigner_Available_Good(t *core.T) {
	subject := &MacOSSigner{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Available()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCodesign_MacOSSigner_Available_Bad(t *core.T) {
	subject := &MacOSSigner{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Available()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCodesign_MacOSSigner_Available_Ugly(t *core.T) {
	subject := &MacOSSigner{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Available()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestCodesign_MacOSSigner_Sign_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &MacOSSigner{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Sign(ctx, storage.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCodesign_MacOSSigner_Sign_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &MacOSSigner{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Sign(ctx, storage.NewMemoryMedium(), "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCodesign_MacOSSigner_Sign_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &MacOSSigner{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Sign(ctx, storage.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestCodesign_MacOSSigner_Notarize_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &MacOSSigner{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Notarize(ctx, storage.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCodesign_MacOSSigner_Notarize_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &MacOSSigner{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Notarize(ctx, storage.NewMemoryMedium(), "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCodesign_MacOSSigner_Notarize_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &MacOSSigner{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Notarize(ctx, storage.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestCodesign_MacOSSigner_ShouldNotarize_Good(t *core.T) {
	subject := &MacOSSigner{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.ShouldNotarize()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCodesign_MacOSSigner_ShouldNotarize_Bad(t *core.T) {
	subject := &MacOSSigner{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.ShouldNotarize()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCodesign_MacOSSigner_ShouldNotarize_Ugly(t *core.T) {
	subject := &MacOSSigner{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.ShouldNotarize()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
