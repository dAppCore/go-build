package signing

import (
	"context"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/io"
	coreerr "dappco.re/go/log"
)

// GPGSigner signs files using GPG.
//
// s := signing.NewGPGSigner("ABCD1234")
type GPGSigner struct {
	KeyID string
}

// Compile-time interface check.
var _ Signer = (*GPGSigner)(nil)

// NewGPGSigner creates a new GPG signer.
//
// s := signing.NewGPGSigner("ABCD1234")
func NewGPGSigner(keyID string) *GPGSigner {
	return &GPGSigner{KeyID: keyID}
}

// Name returns "gpg".
//
// name := s.Name() // → "gpg"
func (s *GPGSigner) Name() string {
	return "gpg"
}

// Available checks if gpg is installed and key is configured.
//
// ok := s.Available() // → true if gpg is in PATH and key is set
func (s *GPGSigner) Available() bool {
	if s.KeyID == "" {
		return false
	}
	_, err := resolveGpgCli()
	return err == nil
}

// Sign creates a detached ASCII-armored signature.
// For file.txt, creates file.txt.asc
//
// err := s.Sign(ctx, io.Local, "dist/CHECKSUMS.txt") // creates CHECKSUMS.txt.asc
func (s *GPGSigner) Sign(ctx context.Context, fs io.Medium, file string) error {
	if s.KeyID == "" {
		return coreerr.E("gpg.Sign", "gpg not available or key not configured", nil)
	}

	gpgCommand, err := resolveGpgCli()
	if err != nil {
		return coreerr.E("gpg.Sign", "gpg not available or key not configured", err)
	}

	output, err := ax.CombinedOutput(ctx, "", nil, gpgCommand,
		"--detach-sign",
		"--armor",
		"--local-user", s.KeyID,
		"--output", file+".asc",
		file,
	)
	if err != nil {
		return coreerr.E("gpg.Sign", output, err)
	}

	return nil
}

func resolveGpgCli(paths ...string) (string, error) {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/gpg",
			"/opt/homebrew/bin/gpg",
			"/usr/local/MacGPG2/bin/gpg",
		}
	}

	command, err := ax.ResolveCommand("gpg", paths...)
	if err != nil {
		return "", coreerr.E("gpg.resolveGpgCli", "gpg CLI not found. Install it from https://gnupg.org/download/", err)
	}

	return command, nil
}
