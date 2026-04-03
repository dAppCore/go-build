package signing

import (
	"context"
	"runtime"

	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

// MacOSSigner signs binaries using macOS codesign.
//
// s := signing.NewMacOSSigner(cfg.MacOS)
type MacOSSigner struct {
	config MacOSConfig
}

// Compile-time interface check.
var _ Signer = (*MacOSSigner)(nil)

// NewMacOSSigner creates a new macOS signer.
//
// s := signing.NewMacOSSigner(cfg.MacOS)
func NewMacOSSigner(cfg MacOSConfig) *MacOSSigner {
	return &MacOSSigner{config: cfg}
}

// Name returns "codesign".
//
// name := s.Name() // → "codesign"
func (s *MacOSSigner) Name() string {
	return "codesign"
}

// Available checks if running on macOS with codesign and identity configured.
//
// ok := s.Available() // → true if on macOS with identity set
func (s *MacOSSigner) Available() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	if s.config.Identity == "" {
		return false
	}
	_, err := resolveCodesignCli()
	return err == nil
}

// Sign codesigns a binary with hardened runtime.
//
// err := s.Sign(ctx, io.Local, "dist/myapp")
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

	codesignCommand, err := resolveCodesignCli()
	if err != nil {
		return coreerr.E("codesign.Sign", "codesign tool not found in PATH", err)
	}

	output, err := ax.CombinedOutput(ctx, "", nil, codesignCommand,
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
//
// err := s.Notarize(ctx, io.Local, "dist/myapp")
func (s *MacOSSigner) Notarize(ctx context.Context, fs io.Medium, binary string) error {
	if s.config.AppleID == "" || s.config.TeamID == "" || s.config.AppPassword == "" {
		return coreerr.E("codesign.Notarize", "missing Apple credentials (apple_id, team_id, app_password)", nil)
	}

	zipCommand, err := resolveZipCli()
	if err != nil {
		return coreerr.E("codesign.Notarize", "zip tool not found in PATH", err)
	}

	xcrunCommand, err := resolveXcrunCli()
	if err != nil {
		return coreerr.E("codesign.Notarize", "xcrun tool not found in PATH", err)
	}

	// Create ZIP for submission
	zipPath := binary + ".zip"
	if output, err := ax.CombinedOutput(ctx, "", nil, zipCommand, "-j", zipPath, binary); err != nil {
		return coreerr.E("codesign.Notarize", "failed to create zip: "+output, err)
	}
	defer func() { _ = fs.Delete(zipPath) }()

	// Submit to Apple and wait
	if output, err := ax.CombinedOutput(ctx, "", nil, xcrunCommand, "notarytool", "submit",
		zipPath,
		"--apple-id", s.config.AppleID,
		"--team-id", s.config.TeamID,
		"--password", s.config.AppPassword,
		"--wait",
	); err != nil {
		return coreerr.E("codesign.Notarize", "notarization failed: "+output, err)
	}

	// Staple the ticket
	if output, err := ax.CombinedOutput(ctx, "", nil, xcrunCommand, "stapler", "staple", binary); err != nil {
		return coreerr.E("codesign.Notarize", "failed to staple: "+output, err)
	}

	return nil
}

// ShouldNotarize returns true if notarization is enabled.
//
// if s.ShouldNotarize() { ... }
func (s *MacOSSigner) ShouldNotarize() bool {
	return s.config.Notarize
}

func resolveCodesignCli(paths ...string) (string, error) {
	if len(paths) == 0 {
		paths = []string{
			"/usr/bin/codesign",
			"/usr/local/bin/codesign",
			"/opt/homebrew/bin/codesign",
		}
	}

	command, err := ax.ResolveCommand("codesign", paths...)
	if err != nil {
		return "", coreerr.E("codesign.resolveCodesignCli", "codesign tool not found. Install Xcode Command Line Tools on macOS.", err)
	}

	return command, nil
}

func resolveZipCli(paths ...string) (string, error) {
	if len(paths) == 0 {
		paths = []string{
			"/usr/bin/zip",
			"/usr/local/bin/zip",
			"/opt/homebrew/bin/zip",
		}
	}

	command, err := ax.ResolveCommand("zip", paths...)
	if err != nil {
		return "", coreerr.E("codesign.resolveZipCli", "zip tool not found. Install the zip utility for notarisation packaging.", err)
	}

	return command, nil
}

func resolveXcrunCli(paths ...string) (string, error) {
	if len(paths) == 0 {
		paths = []string{
			"/usr/bin/xcrun",
			"/usr/local/bin/xcrun",
			"/opt/homebrew/bin/xcrun",
		}
	}

	command, err := ax.ResolveCommand("xcrun", paths...)
	if err != nil {
		return "", coreerr.E("codesign.resolveXcrunCli", "xcrun tool not found. Install Xcode Command Line Tools on macOS.", err)
	}

	return command, nil
}
