package signing

import (
	"context"

	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

// GPGSigner signs files using GPG.
// Usage example: declare a value of type signing.GPGSigner in integrating code.
type GPGSigner struct {
	KeyID string
}

// Compile-time interface check.
var _ Signer = (*GPGSigner)(nil)

// NewGPGSigner creates a new GPG signer.
// Usage example: call signing.NewGPGSigner(...) from integrating code.
func NewGPGSigner(keyID string) *GPGSigner {
	return &GPGSigner{KeyID: keyID}
}

// Name returns "gpg".
// Usage example: call value.Name(...) from integrating code.
func (s *GPGSigner) Name() string {
	return "gpg"
}

// Available checks if gpg is installed and key is configured.
// Usage example: call value.Available(...) from integrating code.
func (s *GPGSigner) Available() bool {
	if s.KeyID == "" {
		return false
	}
	_, err := ax.LookPath("gpg")
	return err == nil
}

// Sign creates a detached ASCII-armored signature.
// For file.txt, creates file.txt.asc
// Usage example: call value.Sign(...) from integrating code.
func (s *GPGSigner) Sign(ctx context.Context, fs io.Medium, file string) error {
	if !s.Available() {
		return coreerr.E("gpg.Sign", "gpg not available or key not configured", nil)
	}

	output, err := ax.CombinedOutput(ctx, "", nil, "gpg",
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
