package signing

import (
	"context"
	"runtime"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	storage "dappco.re/go/build/pkg/storage"
)

// WindowsSigner signs binaries using Windows signtool.
//
// s := signing.NewWindowsSigner(cfg.Windows)
type WindowsSigner struct {
	config WindowsConfig
}

// Compile-time interface check.
var _ Signer = (*WindowsSigner)(nil)

// NewWindowsSigner creates a new Windows signer.
//
// s := signing.NewWindowsSigner(cfg.Windows)
func NewWindowsSigner(cfg WindowsConfig) *WindowsSigner {
	return &WindowsSigner{config: cfg}
}

// Name returns "signtool".
//
// name := s.Name() // → "signtool"
func (s *WindowsSigner) Name() string {
	return "signtool"
}

// Available checks if running on Windows with signtool and certificate configured.
//
// ok := s.Available() // → true if on Windows with certificate configured
func (s *WindowsSigner) Available() bool {
	if !s.config.signtoolEnabled() {
		return false
	}
	if runtime.GOOS != "windows" {
		return false
	}
	if s.config.Certificate == "" {
		return false
	}
	return resolveSigntoolCli().OK
}

// Sign signs a binary using signtool and a PFX certificate.
//
// err := s.Sign(ctx, storage.Local, "dist/myapp.exe")
func (s *WindowsSigner) Sign(ctx context.Context, fs storage.Medium, binary string) core.Result {
	_ = fs

	if !s.Available() {
		if runtime.GOOS != "windows" {
			return core.Fail(core.E("signtool.Sign", "signtool is only available on Windows", nil))
		}
		if s.config.Certificate == "" {
			return core.Fail(core.E("signtool.Sign", "signtool certificate not configured", nil))
		}
		return core.Fail(core.E("signtool.Sign", "signtool tool not found in PATH", nil))
	}

	signtoolCommand := resolveSigntoolCli()
	if !signtoolCommand.OK {
		return core.Fail(core.E("signtool.Sign", "signtool tool not found in PATH", core.NewError(signtoolCommand.Error())))
	}

	args := []string{
		"sign",
		"/f", s.config.Certificate,
		"/fd", "sha256",
		"/tr", "http://timestamp.digicert.com",
		"/td", "sha256",
	}
	if s.config.Password != "" {
		args = append(args, "/p", s.config.Password)
	}
	args = append(args, binary)

	output := ax.CombinedOutput(ctx, "", nil, signtoolCommand.Value.(string), args...)
	if !output.OK {
		return core.Fail(core.E("signtool.Sign", output.Error(), core.NewError(output.Error())))
	}

	return core.Ok(nil)
}

func resolveSigntoolCli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			`C:\\Program Files (x86)\\Windows Kits\\10\\bin\\x64\\signtool.exe`,
			`C:\\Program Files (x86)\\Windows Kits\\10\\bin\\x86\\signtool.exe`,
			`C:\\Program Files\\Windows Kits\\10\\bin\\x64\\signtool.exe`,
			`C:\\Program Files\\Windows Kits\\10\\bin\\x86\\signtool.exe`,
		}
	}

	command := ax.ResolveCommand("signtool", paths...)
	if !command.OK {
		return core.Fail(core.E("signtool.resolveSigntoolCli", "signtool tool not found. Install the Windows SDK.", core.NewError(command.Error())))
	}

	return command
}
