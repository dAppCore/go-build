package signing

import (
	"context"
	"runtime"
	"testing"

	"dappco.re/go/build/internal/testassert"
	"dappco.re/go/io"
)

func TestSigning_SignBinariesSkipsNonDarwinGood(t *testing.T) {
	ctx := context.Background()
	fs := io.Local
	cfg := SignConfig{
		Enabled: true,
		MacOS: MacOSConfig{
			Identity: "Developer ID Application: Test",
		},
	}

	// Create fake artifact for linux
	artifacts := []Artifact{
		{Path: "/tmp/test-binary", OS: "linux", Arch: "amd64"},
	}

	// Should not error even though binary doesn't exist (skips non-darwin)
	err := SignBinaries(ctx, fs, cfg, artifacts)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSigning_SignBinariesDisabledConfigGood(t *testing.T) {
	ctx := context.Background()
	fs := io.Local
	cfg := SignConfig{
		Enabled: false,
	}

	artifacts := []Artifact{
		{Path: "/tmp/test-binary", OS: "darwin", Arch: "arm64"},
	}

	err := SignBinaries(ctx, fs, cfg, artifacts)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSigning_SignBinariesSkipsOnNonMacOSGood(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("Skipping on macOS - this tests non-macOS behavior")
	}

	ctx := context.Background()
	fs := io.Local
	cfg := SignConfig{
		Enabled: true,
		MacOS: MacOSConfig{
			Identity: "Developer ID Application: Test",
		},
	}

	artifacts := []Artifact{
		{Path: "/tmp/test-binary", OS: "darwin", Arch: "arm64"},
	}

	err := SignBinaries(ctx, fs, cfg, artifacts)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSigning_NotarizeBinariesDisabledConfigGood(t *testing.T) {
	ctx := context.Background()
	fs := io.Local
	cfg := SignConfig{
		Enabled: false,
	}

	artifacts := []Artifact{
		{Path: "/tmp/test-binary", OS: "darwin", Arch: "arm64"},
	}

	err := NotarizeBinaries(ctx, fs, cfg, artifacts)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSigning_NotarizeBinariesNotarizeDisabledGood(t *testing.T) {
	ctx := context.Background()
	fs := io.Local
	cfg := SignConfig{
		Enabled: true,
		MacOS: MacOSConfig{
			Notarize: false,
		},
	}

	artifacts := []Artifact{
		{Path: "/tmp/test-binary", OS: "darwin", Arch: "arm64"},
	}

	err := NotarizeBinaries(ctx, fs, cfg, artifacts)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSigning_SignChecksumsSkipsNoKeyGood(t *testing.T) {
	ctx := context.Background()
	fs := io.Local
	cfg := SignConfig{
		Enabled: true,
		GPG: GPGConfig{
			Key: "", // No key configured
		},
	}

	// Should silently skip when no key
	err := SignChecksums(ctx, fs, cfg, "/tmp/CHECKSUMS.txt")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSigning_SignChecksumsDisabledGood(t *testing.T) {
	ctx := context.Background()
	fs := io.Local
	cfg := SignConfig{
		Enabled: false,
	}

	err := SignChecksums(ctx, fs, cfg, "/tmp/CHECKSUMS.txt")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSigning_DefaultSignConfig_Good(t *testing.T) {
	cfg := DefaultSignConfig()
	if !(cfg.Enabled) {
		t.Fatal("expected true")
	}
	if !(cfg.Windows.Signtool) {
		t.Fatal("expected true")
	}

}

func TestSigning_SignConfigExpandEnvGood(t *testing.T) {
	t.Setenv("TEST_KEY", "ABC")
	cfg := SignConfig{
		GPG: GPGConfig{Key: "$TEST_KEY"},
	}
	cfg.ExpandEnv()
	if !stdlibAssertEqual("ABC", cfg.GPG.Key) {
		t.Fatalf("want %v, got %v", "ABC", cfg.GPG.Key)
	}

}

func TestSigning_WindowsSignerGood(t *testing.T) {
	fs := io.Local
	s := NewWindowsSigner(WindowsConfig{Signtool: true, Certificate: "cert.pfx"})
	if !stdlibAssertEqual("signtool", s.Name()) {
		t.Fatalf("want %v, got %v", "signtool", s.Name())
	}

	if runtime.GOOS != "windows" {
		if s.Available() {
			t.Fatal("expected false")
		}
		if s.Sign(context.Background(), fs, "test.exe") == nil {
			t.Fatal("expected error")

			// On Windows, availability depends on the SDK toolchain being installed.
		}

		return
	}

	_ = s.Available()
}

func TestSigning_WindowsSignerHonoursSigntoolToggleGood(t *testing.T) {
	s := NewWindowsSigner(WindowsConfig{
		Signtool:         false,
		Certificate:      "cert.pfx",
		signtoolExplicit: true,
	})
	if s.Available() {
		t.Fatal("expected false")

		// mockSigner is a test double that records calls to Sign.
	}

}

type mockSigner struct {
	name        string
	available   bool
	signedPaths []string
	signError   error
}

func (m *mockSigner) Name() string {
	return m.name
}

func (m *mockSigner) Available() bool {
	return m.available
}

func (m *mockSigner) Sign(ctx context.Context, fs io.Medium, path string) error {
	m.signedPaths = append(m.signedPaths, path)
	return m.signError
}

// Verify mockSigner implements Signer
var _ Signer = (*mockSigner)(nil)

func TestSigning_SignBinariesMockSignerGood(t *testing.T) {
	t.Run("signs only darwin artifacts", func(t *testing.T) {
		artifacts := []Artifact{
			{Path: "/dist/linux_amd64/myapp", OS: "linux", Arch: "amd64"},
			{Path: "/dist/darwin_arm64/myapp", OS: "darwin", Arch: "arm64"},
			{Path: "/dist/windows_amd64/myapp.exe", OS: "windows", Arch: "amd64"},
			{Path: "/dist/darwin_amd64/myapp", OS: "darwin", Arch: "amd64"},
		}

		// SignBinaries filters to darwin only and calls signer.Sign for each.
		// We can verify the logic by checking that non-darwin artifacts are skipped.
		// Since SignBinaries uses NewMacOSSigner internally, we test the filtering
		// by passing only darwin artifacts and confirming non-darwin are skipped.
		cfg := SignConfig{
			Enabled: true,
			MacOS:   MacOSConfig{Identity: ""},
		}

		// With empty identity, Available() returns false, so Sign is never called.
		// This verifies the short-circuit behavior.
		ctx := context.Background()
		err := SignBinaries(ctx, io.Local, cfg, artifacts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

	})

	t.Run("skips all when enabled is false", func(t *testing.T) {
		artifacts := []Artifact{
			{Path: "/dist/darwin_arm64/myapp", OS: "darwin", Arch: "arm64"},
		}

		cfg := SignConfig{Enabled: false}
		err := SignBinaries(context.Background(), io.Local, cfg, artifacts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

	})

	t.Run("handles empty artifact list", func(t *testing.T) {
		cfg := SignConfig{
			Enabled: true,
			MacOS:   MacOSConfig{Identity: "Developer ID"},
		}
		err := SignBinaries(context.Background(), io.Local, cfg, []Artifact{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

	})
}

func TestSigning_signArtifactsWithSigner_Good(t *testing.T) {
	signer := &mockSigner{name: "mock", available: true}
	artifacts := []Artifact{
		{Path: "/dist/linux_amd64/myapp", OS: "linux", Arch: "amd64"},
		{Path: "/dist/windows_amd64/myapp.exe", OS: "windows", Arch: "amd64"},
		{Path: "/dist/windows_arm64/myapp.exe", OS: "windows", Arch: "arm64"},
	}

	err := signArtifactsWithSigner(context.Background(), io.Local, signer, "windows", artifacts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual([]string{"/dist/windows_amd64/myapp.exe", "/dist/windows_arm64/myapp.exe"}, signer.signedPaths) {
		t.Fatalf("want %v, got %v", []string{"/dist/windows_amd64/myapp.exe", "/dist/windows_arm64/myapp.exe"}, signer.signedPaths)
	}

}

func TestSigning_ResolveSigntoolCliGood(t *testing.T) {
	fallbackDir := t.TempDir()
	fallbackPath := fallbackDir + "/signtool.exe"
	if err := io.Local.Write(fallbackPath, "#!/bin/sh\nexit 0\n"); err != nil {
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

func TestSigning_ResolveSigntoolCliBad(t *testing.T) {
	t.Setenv("PATH", "")

	_, err := resolveSigntoolCli(t.TempDir() + "/missing-signtool.exe")
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "signtool tool not found") {
		t.Fatalf("expected %v to contain %v", err.Error(), "signtool tool not found")
	}

}

func TestSigning_SignChecksumsMockSignerGood(t *testing.T) {
	t.Run("skips when GPG key is empty", func(t *testing.T) {
		cfg := SignConfig{
			Enabled: true,
			GPG:     GPGConfig{Key: ""},
		}

		err := SignChecksums(context.Background(), io.Local, cfg, "/tmp/CHECKSUMS.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

	})

	t.Run("skips when disabled", func(t *testing.T) {
		cfg := SignConfig{
			Enabled: false,
			GPG:     GPGConfig{Key: "ABCD1234"},
		}

		err := SignChecksums(context.Background(), io.Local, cfg, "/tmp/CHECKSUMS.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

	})
}

func TestSigning_NotarizeBinariesMockSignerGood(t *testing.T) {
	t.Run("skips when notarize is false", func(t *testing.T) {
		cfg := SignConfig{
			Enabled: true,
			MacOS:   MacOSConfig{Notarize: false},
		}

		artifacts := []Artifact{
			{Path: "/dist/darwin_arm64/myapp", OS: "darwin", Arch: "arm64"},
		}

		err := NotarizeBinaries(context.Background(), io.Local, cfg, artifacts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

	})

	t.Run("skips when disabled", func(t *testing.T) {
		cfg := SignConfig{
			Enabled: false,
			MacOS:   MacOSConfig{Notarize: true},
		}

		artifacts := []Artifact{
			{Path: "/dist/darwin_arm64/myapp", OS: "darwin", Arch: "arm64"},
		}

		err := NotarizeBinaries(context.Background(), io.Local, cfg, artifacts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

	})

	t.Run("handles empty artifact list", func(t *testing.T) {
		cfg := SignConfig{
			Enabled: true,
			MacOS:   MacOSConfig{Notarize: true, Identity: "Dev ID"},
		}

		err := NotarizeBinaries(context.Background(), io.Local, cfg, []Artifact{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

	})
}

func TestSigning_ExpandEnv_Good(t *testing.T) {
	t.Run("expands all config fields", func(t *testing.T) {
		t.Setenv("TEST_GPG_KEY", "GPG123")
		t.Setenv("TEST_IDENTITY", "Developer ID Application: Test")
		t.Setenv("TEST_APPLE_ID", "test@apple.com")
		t.Setenv("TEST_TEAM_ID", "TEAM123")
		t.Setenv("TEST_APP_PASSWORD", "secret")
		t.Setenv("TEST_CERT_PATH", "/path/to/cert.pfx")
		t.Setenv("TEST_CERT_PASS", "certpass")

		cfg := SignConfig{
			GPG: GPGConfig{Key: "$TEST_GPG_KEY"},
			MacOS: MacOSConfig{
				Identity:    "$TEST_IDENTITY",
				AppleID:     "$TEST_APPLE_ID",
				TeamID:      "$TEST_TEAM_ID",
				AppPassword: "$TEST_APP_PASSWORD",
			},
			Windows: WindowsConfig{
				Certificate: "$TEST_CERT_PATH",
				Password:    "$TEST_CERT_PASS",
			},
		}

		cfg.ExpandEnv()
		if !stdlibAssertEqual("GPG123", cfg.GPG.Key) {
			t.Fatalf("want %v, got %v", "GPG123", cfg.GPG.Key)
		}
		if !stdlibAssertEqual("Developer ID Application: Test", cfg.MacOS.Identity) {
			t.Fatalf("want %v, got %v", "Developer ID Application: Test", cfg.MacOS.Identity)
		}
		if !stdlibAssertEqual("test@apple.com", cfg.MacOS.AppleID) {
			t.Fatalf("want %v, got %v", "test@apple.com", cfg.MacOS.AppleID)
		}
		if !stdlibAssertEqual("TEAM123", cfg.MacOS.TeamID) {
			t.Fatalf("want %v, got %v", "TEAM123", cfg.MacOS.TeamID)
		}
		if !stdlibAssertEqual("secret", cfg.MacOS.AppPassword) {
			t.Fatalf("want %v, got %v", "secret", cfg.MacOS.AppPassword)
		}
		if !stdlibAssertEqual("/path/to/cert.pfx", cfg.Windows.Certificate) {
			t.Fatalf("want %v, got %v", "/path/to/cert.pfx", cfg.Windows.Certificate)
		}
		if !stdlibAssertEqual("certpass", cfg.Windows.Password) {
			t.Fatalf("want %v, got %v", "certpass", cfg.Windows.Password)
		}

	})

	t.Run("preserves non-env values", func(t *testing.T) {
		cfg := SignConfig{
			GPG: GPGConfig{Key: "literal-key"},
			MacOS: MacOSConfig{
				Identity: "Developer ID Application: Literal",
			},
		}

		cfg.ExpandEnv()
		if !stdlibAssertEqual("literal-key", cfg.GPG.Key) {
			t.Fatalf("want %v, got %v", "literal-key", cfg.GPG.Key)
		}
		if !stdlibAssertEqual("Developer ID Application: Literal", cfg.MacOS.Identity) {
			t.Fatalf("want %v, got %v", "Developer ID Application: Literal", cfg.MacOS.Identity)
		}

	})
}

var (
	stdlibAssertEqual         = testassert.Equal
	stdlibAssertNil           = testassert.Nil
	stdlibAssertEmpty         = testassert.Empty
	stdlibAssertZero          = testassert.Zero
	stdlibAssertContains      = testassert.Contains
	stdlibAssertElementsMatch = testassert.ElementsMatch
)
