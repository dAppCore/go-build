package signing

import (
	"context"
	"os/exec"
	"runtime"

	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

// MacOSSigner signs binaries using macOS codesign.
type MacOSSigner struct {
	config MacOSConfig
}

// Compile-time interface check.
var _ Signer = (*MacOSSigner)(nil)

// NewMacOSSigner creates a new macOS signer.
func NewMacOSSigner(cfg MacOSConfig) *MacOSSigner {
	return &MacOSSigner{config: cfg}
}

// Name returns "codesign".
func (s *MacOSSigner) Name() string {
	return "codesign"
}

// Available checks if running on macOS with codesign and identity configured.
func (s *MacOSSigner) Available() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	if s.config.Identity == "" {
		return false
	}
	_, err := exec.LookPath("codesign")
	return err == nil
}

// Sign codesigns a binary with hardened runtime.
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

	cmd := exec.CommandContext(ctx, "codesign",
		"--sign", s.config.Identity,
		"--timestamp",
		"--options", "runtime", // Hardened runtime for notarization
		"--force",
		binary,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return coreerr.E("codesign.Sign", string(output), err)
	}

	return nil
}

// Notarize submits binary to Apple for notarization and staples the ticket.
// This blocks until Apple responds (typically 1-5 minutes).
func (s *MacOSSigner) Notarize(ctx context.Context, fs io.Medium, binary string) error {
	if s.config.AppleID == "" || s.config.TeamID == "" || s.config.AppPassword == "" {
		return coreerr.E("codesign.Notarize", "missing Apple credentials (apple_id, team_id, app_password)", nil)
	}

	// Create ZIP for submission
	zipPath := binary + ".zip"
	zipCmd := exec.CommandContext(ctx, "zip", "-j", zipPath, binary)
	if output, err := zipCmd.CombinedOutput(); err != nil {
		return coreerr.E("codesign.Notarize", "failed to create zip: "+string(output), err)
	}
	defer func() { _ = fs.Delete(zipPath) }()

	// Submit to Apple and wait
	submitCmd := exec.CommandContext(ctx, "xcrun", "notarytool", "submit",
		zipPath,
		"--apple-id", s.config.AppleID,
		"--team-id", s.config.TeamID,
		"--password", s.config.AppPassword,
		"--wait",
	)
	if output, err := submitCmd.CombinedOutput(); err != nil {
		return coreerr.E("codesign.Notarize", "notarization failed: "+string(output), err)
	}

	// Staple the ticket
	stapleCmd := exec.CommandContext(ctx, "xcrun", "stapler", "staple", binary)
	if output, err := stapleCmd.CombinedOutput(); err != nil {
		return coreerr.E("codesign.Notarize", "failed to staple: "+string(output), err)
	}

	return nil
}

// ShouldNotarize returns true if notarization is enabled.
func (s *MacOSSigner) ShouldNotarize() bool {
	return s.config.Notarize
}
