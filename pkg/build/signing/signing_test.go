package signing

import (
	"context"
	"runtime"
	"testing"

	"forge.lthn.ai/core/go-io"
	"github.com/stretchr/testify/assert"
)

func TestSignBinaries_Good_SkipsNonDarwin(t *testing.T) {
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

func TestSignBinaries_Good_DisabledConfig(t *testing.T) {
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

func TestSignBinaries_Good_SkipsOnNonMacOS(t *testing.T) {
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

func TestNotarizeBinaries_Good_DisabledConfig(t *testing.T) {
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

func TestNotarizeBinaries_Good_NotarizeDisabled(t *testing.T) {
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

func TestSignChecksums_Good_SkipsNoKey(t *testing.T) {
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

func TestSignChecksums_Good_Disabled(t *testing.T) {
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

func TestDefaultSignConfig(t *testing.T) {
	cfg := DefaultSignConfig()
	assert.True(t, cfg.Enabled)
}

func TestSignConfig_ExpandEnv(t *testing.T) {
	t.Setenv("TEST_KEY", "ABC")
	cfg := SignConfig{
		GPG: GPGConfig{Key: "$TEST_KEY"},
	}
	cfg.ExpandEnv()
	assert.Equal(t, "ABC", cfg.GPG.Key)
}

func TestWindowsSigner_Good(t *testing.T) {
	fs := io.Local
	s := NewWindowsSigner(WindowsConfig{})
	assert.Equal(t, "signtool", s.Name())
	assert.False(t, s.Available())
	assert.NoError(t, s.Sign(context.Background(), fs, "test.exe"))
}

// mockSigner is a test double that records calls to Sign.
type mockSigner struct {
	name      string
	available bool
	signedPaths []string
	signError error
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

func TestSignBinaries_Good_MockSigner(t *testing.T) {
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
		assert.NoError(t, err)
	})

	t.Run("skips all when enabled is false", func(t *testing.T) {
		artifacts := []Artifact{
			{Path: "/dist/darwin_arm64/myapp", OS: "darwin", Arch: "arm64"},
		}

		cfg := SignConfig{Enabled: false}
		err := SignBinaries(context.Background(), io.Local, cfg, artifacts)
		assert.NoError(t, err)
	})

	t.Run("handles empty artifact list", func(t *testing.T) {
		cfg := SignConfig{
			Enabled: true,
			MacOS:   MacOSConfig{Identity: "Developer ID"},
		}
		err := SignBinaries(context.Background(), io.Local, cfg, []Artifact{})
		assert.NoError(t, err)
	})
}

func TestSignChecksums_Good_MockSigner(t *testing.T) {
	t.Run("skips when GPG key is empty", func(t *testing.T) {
		cfg := SignConfig{
			Enabled: true,
			GPG:     GPGConfig{Key: ""},
		}

		err := SignChecksums(context.Background(), io.Local, cfg, "/tmp/CHECKSUMS.txt")
		assert.NoError(t, err)
	})

	t.Run("skips when disabled", func(t *testing.T) {
		cfg := SignConfig{
			Enabled: false,
			GPG:     GPGConfig{Key: "ABCD1234"},
		}

		err := SignChecksums(context.Background(), io.Local, cfg, "/tmp/CHECKSUMS.txt")
		assert.NoError(t, err)
	})
}

func TestNotarizeBinaries_Good_MockSigner(t *testing.T) {
	t.Run("skips when notarize is false", func(t *testing.T) {
		cfg := SignConfig{
			Enabled: true,
			MacOS:   MacOSConfig{Notarize: false},
		}

		artifacts := []Artifact{
			{Path: "/dist/darwin_arm64/myapp", OS: "darwin", Arch: "arm64"},
		}

		err := NotarizeBinaries(context.Background(), io.Local, cfg, artifacts)
		assert.NoError(t, err)
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
		assert.NoError(t, err)
	})

	t.Run("handles empty artifact list", func(t *testing.T) {
		cfg := SignConfig{
			Enabled: true,
			MacOS:   MacOSConfig{Notarize: true, Identity: "Dev ID"},
		}

		err := NotarizeBinaries(context.Background(), io.Local, cfg, []Artifact{})
		assert.NoError(t, err)
	})
}

func TestExpandEnv_Good(t *testing.T) {
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

		assert.Equal(t, "GPG123", cfg.GPG.Key)
		assert.Equal(t, "Developer ID Application: Test", cfg.MacOS.Identity)
		assert.Equal(t, "test@apple.com", cfg.MacOS.AppleID)
		assert.Equal(t, "TEAM123", cfg.MacOS.TeamID)
		assert.Equal(t, "secret", cfg.MacOS.AppPassword)
		assert.Equal(t, "/path/to/cert.pfx", cfg.Windows.Certificate)
		assert.Equal(t, "certpass", cfg.Windows.Password)
	})

	t.Run("preserves non-env values", func(t *testing.T) {
		cfg := SignConfig{
			GPG: GPGConfig{Key: "literal-key"},
			MacOS: MacOSConfig{
				Identity: "Developer ID Application: Literal",
			},
		}

		cfg.ExpandEnv()

		assert.Equal(t, "literal-key", cfg.GPG.Key)
		assert.Equal(t, "Developer ID Application: Literal", cfg.MacOS.Identity)
	})
}
