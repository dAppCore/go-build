package signing

import (
	"context"

	"dappco.re/go/core/io"
)

// WindowsSigner signs binaries using Windows signtool (placeholder).
// Usage example: declare a value of type signing.WindowsSigner in integrating code.
type WindowsSigner struct {
	config WindowsConfig
}

// Compile-time interface check.
var _ Signer = (*WindowsSigner)(nil)

// NewWindowsSigner creates a new Windows signer.
// Usage example: call signing.NewWindowsSigner(...) from integrating code.
func NewWindowsSigner(cfg WindowsConfig) *WindowsSigner {
	return &WindowsSigner{config: cfg}
}

// Name returns "signtool".
// Usage example: call value.Name(...) from integrating code.
func (s *WindowsSigner) Name() string {
	return "signtool"
}

// Available returns false (not yet implemented).
// Usage example: call value.Available(...) from integrating code.
func (s *WindowsSigner) Available() bool {
	return false
}

// Sign is a placeholder that does nothing.
// Usage example: call value.Sign(...) from integrating code.
func (s *WindowsSigner) Sign(ctx context.Context, fs io.Medium, binary string) error {
	// TODO: Implement Windows signing
	return nil
}
