package signing

import (
	"context"

	"dappco.re/go/core/io"
)

// WindowsSigner signs binaries using Windows signtool (placeholder).
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

// Available returns false (not yet implemented).
//
// ok := s.Available() // → false (placeholder)
func (s *WindowsSigner) Available() bool {
	return false
}

// Sign is a placeholder that does nothing.
//
// err := s.Sign(ctx, io.Local, "dist/myapp.exe") // no-op until implemented
func (s *WindowsSigner) Sign(ctx context.Context, fs io.Medium, binary string) error {
	// TODO: Implement Windows signing
	return nil
}
