package signing

import (
	"context"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/io"
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
	if core.Env("GOOS") != "darwin" {
		return false
	}
	if s.config.Identity == "" {
		return false
	}
	return resolveCodesignCli().OK
}

// Sign codesigns a binary with hardened runtime.
//
// err := s.Sign(ctx, io.Local, "dist/myapp")
func (s *MacOSSigner) Sign(ctx context.Context, fs io.Medium, binary string) core.Result {
	if !s.Available() {
		if core.Env("GOOS") != "darwin" {
			return core.Fail(core.E("codesign.Sign", "codesign is only available on macOS", nil))
		}
		if s.config.Identity == "" {
			return core.Fail(core.E("codesign.Sign", "codesign identity not configured", nil))
		}
		return core.Fail(core.E("codesign.Sign", "codesign tool not found in PATH", nil))
	}

	codesignCommand := resolveCodesignCli()
	if !codesignCommand.OK {
		return core.Fail(core.E("codesign.Sign", "codesign tool not found in PATH", core.NewError(codesignCommand.Error())))
	}

	output := ax.CombinedOutput(ctx, "", nil, codesignCommand.Value.(string),
		"--sign", s.config.Identity,
		"--timestamp",
		"--options", `runtime`, // Hardened runtime for notarization
		"--force",
		binary,
	)
	if !output.OK {
		return core.Fail(core.E("codesign.Sign", output.Error(), core.NewError(output.Error())))
	}

	return core.Ok(nil)
}

// Notarize submits binary to Apple for notarization and staples the ticket.
// This blocks until Apple responds (typically 1-5 minutes).
//
// err := s.Notarize(ctx, io.Local, "dist/myapp")
func (s *MacOSSigner) Notarize(ctx context.Context, fs io.Medium, binary string) core.Result {
	if s.config.AppleID == "" || s.config.TeamID == "" || s.config.AppPassword == "" {
		return core.Fail(core.E("codesign.Notarize", "missing Apple credentials (apple_id, team_id, app_password)", nil))
	}

	zipCommand := resolveZipCli()
	if !zipCommand.OK {
		return core.Fail(core.E("codesign.Notarize", "zip tool not found in PATH", core.NewError(zipCommand.Error())))
	}

	xcrunCommand := resolveXcrunCli()
	if !xcrunCommand.OK {
		return core.Fail(core.E("codesign.Notarize", "xcrun tool not found in PATH", core.NewError(xcrunCommand.Error())))
	}

	// Create ZIP for submission
	zipPath := binary + ".zip"
	if output := ax.CombinedOutput(ctx, "", nil, zipCommand.Value.(string), "-j", zipPath, binary); !output.OK {
		return core.Fail(core.E("codesign.Notarize", "failed to create zip: "+output.Error(), core.NewError(output.Error())))
	}
	defer func() { _ = fs.Delete(zipPath) }()

	// Submit to Apple and wait
	if output := ax.CombinedOutput(ctx, "", nil, xcrunCommand.Value.(string), "notarytool", "submit",
		zipPath,
		"--apple-id", s.config.AppleID,
		"--team-id", s.config.TeamID,
		"--password", s.config.AppPassword,
		"--wait",
	); !output.OK {
		return core.Fail(core.E("codesign.Notarize", "notarization failed: "+output.Error(), core.NewError(output.Error())))
	}

	// Staple the ticket
	if output := ax.CombinedOutput(ctx, "", nil, xcrunCommand.Value.(string), "stapler", "staple", binary); !output.OK {
		return core.Fail(core.E("codesign.Notarize", "failed to staple: "+output.Error(), core.NewError(output.Error())))
	}

	return core.Ok(nil)
}

// ShouldNotarize returns true if notarization is enabled.
//
// if s.ShouldNotarize() { ... }
func (s *MacOSSigner) ShouldNotarize() bool {
	return s.config.Notarize
}

func resolveCodesignCli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			"/usr/bin/codesign",
			"/usr/local/bin/codesign",
			"/opt/homebrew/bin/codesign",
		}
	}

	command := ax.ResolveCommand("codesign", paths...)
	if !command.OK {
		return core.Fail(core.E("codesign.resolveCodesignCli", "codesign tool not found. Install Xcode Command Line Tools on macOS.", core.NewError(command.Error())))
	}

	return command
}

func resolveZipCli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			"/usr/bin/zip",
			"/usr/local/bin/zip",
			"/opt/homebrew/bin/zip",
		}
	}

	command := ax.ResolveCommand("zip", paths...)
	if !command.OK {
		return core.Fail(core.E("codesign.resolveZipCli", "zip tool not found. Install the zip utility for notarisation packaging.", core.NewError(command.Error())))
	}

	return command
}

func resolveXcrunCli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			"/usr/bin/xcrun",
			"/usr/local/bin/xcrun",
			"/opt/homebrew/bin/xcrun",
		}
	}

	command := ax.ResolveCommand("xcrun", paths...)
	if !command.OK {
		return core.Fail(core.E("codesign.resolveXcrunCli", "xcrun tool not found. Install Xcode Command Line Tools on macOS.", core.NewError(command.Error())))
	}

	return command
}
