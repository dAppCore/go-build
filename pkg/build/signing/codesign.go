package signing

import (
	"context"
	"runtime"

	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

// MacOSSigner signs binaries using macOS codesign.
// Usage example: declare a value of type signing.MacOSSigner in integrating code.
type MacOSSigner struct {
	config MacOSConfig
}

// Compile-time interface check.
var _ Signer = (*MacOSSigner)(nil)

// NewMacOSSigner creates a new macOS signer.
// Usage example: call signing.NewMacOSSigner(...) from integrating code.
func NewMacOSSigner(cfg MacOSConfig) *MacOSSigner {
	return &MacOSSigner{config: cfg}
}

// Name returns "codesign".
// Usage example: call value.Name(...) from integrating code.
func (s *MacOSSigner) Name() string {
	return "codesign"
}

// Available checks if running on macOS with codesign and identity configured.
// Usage example: call value.Available(...) from integrating code.
func (s *MacOSSigner) Available() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	if s.config.Identity == "" {
		return false
	}
	_, err := ax.LookPath("codesign")
	return err == nil
}

// Sign codesigns a binary with hardened runtime.
// Usage example: call value.Sign(...) from integrating code.
func (s *MacOSSigner) Sign(ctx context.Context, fs io.Medium, binary string) error {
	if !s.Available() {
		if runtime.GOOS != "darwin" {
			return coreerr.E("codesign.Sign", "codesign is only available on macOS", nil)
		}
		if s.config.Identity == "" {
			return coreerr.E("codesign.Sign", "codesign identity not configured", nil)
		}
		return coreerr.E("codesign.Sign", "codesign tool not found in PATH", nil)
	}

	output, err := ax.CombinedOutput(ctx, "", nil, "codesign",
		"--sign", s.config.Identity,
		"--timestamp",
		"--options", "runtime", // Hardened runtime for notarization
		"--force",
		binary,
	)
	if err != nil {
		return coreerr.E("codesign.Sign", output, err)
	}

	return nil
}

// Notarize submits binary to Apple for notarization and staples the ticket.
// This blocks until Apple responds (typically 1-5 minutes).
// Usage example: call value.Notarize(...) from integrating code.
func (s *MacOSSigner) Notarize(ctx context.Context, fs io.Medium, binary string) error {
	if s.config.AppleID == "" || s.config.TeamID == "" || s.config.AppPassword == "" {
		return coreerr.E("codesign.Notarize", "missing Apple credentials (apple_id, team_id, app_password)", nil)
	}

	// Create ZIP for submission
	zipPath := binary + ".zip"
	if output, err := ax.CombinedOutput(ctx, "", nil, "zip", "-j", zipPath, binary); err != nil {
		return coreerr.E("codesign.Notarize", "failed to create zip: "+output, err)
	}
	defer func() { _ = fs.Delete(zipPath) }()

	// Submit to Apple and wait
	if output, err := ax.CombinedOutput(ctx, "", nil, "xcrun", "notarytool", "submit",
		zipPath,
		"--apple-id", s.config.AppleID,
		"--team-id", s.config.TeamID,
		"--password", s.config.AppPassword,
		"--wait",
	); err != nil {
		return coreerr.E("codesign.Notarize", "notarization failed: "+output, err)
	}

	// Staple the ticket
	if output, err := ax.CombinedOutput(ctx, "", nil, "xcrun", "stapler", "staple", binary); err != nil {
		return coreerr.E("codesign.Notarize", "failed to staple: "+output, err)
	}

	return nil
}

// ShouldNotarize returns true if notarization is enabled.
// Usage example: call value.ShouldNotarize(...) from integrating code.
func (s *MacOSSigner) ShouldNotarize() bool {
	return s.config.Notarize
}
