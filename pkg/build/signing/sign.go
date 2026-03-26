package signing

import (
	"context"
	"runtime"

	"dappco.re/go/core"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

// Artifact represents a build output that can be signed.
// This mirrors build.Artifact to avoid import cycles.
// Usage example: declare a value of type signing.Artifact in integrating code.
type Artifact struct {
	Path string
	OS   string
	Arch string
}

// SignBinaries signs macOS binaries in the artifacts list.
// Only signs darwin binaries when running on macOS with a configured identity.
// Usage example: call signing.SignBinaries(...) from integrating code.
func SignBinaries(ctx context.Context, fs io.Medium, cfg SignConfig, artifacts []Artifact) error {
	if !cfg.Enabled {
		return nil
	}

	// Only sign on macOS
	if runtime.GOOS != "darwin" {
		return nil
	}

	signer := NewMacOSSigner(cfg.MacOS)
	if !signer.Available() {
		return nil // Silently skip if not configured
	}

	for _, artifact := range artifacts {
		if artifact.OS != "darwin" {
			continue
		}

		core.Print(nil, "  Signing %s...", artifact.Path)
		if err := signer.Sign(ctx, fs, artifact.Path); err != nil {
			return coreerr.E("signing.SignBinaries", "failed to sign "+artifact.Path, err)
		}
	}

	return nil
}

// NotarizeBinaries notarizes macOS binaries if enabled.
// Usage example: call signing.NotarizeBinaries(...) from integrating code.
func NotarizeBinaries(ctx context.Context, fs io.Medium, cfg SignConfig, artifacts []Artifact) error {
	if !cfg.Enabled || !cfg.MacOS.Notarize {
		return nil
	}

	if runtime.GOOS != "darwin" {
		return nil
	}

	signer := NewMacOSSigner(cfg.MacOS)
	if !signer.Available() {
		return coreerr.E("signing.NotarizeBinaries", "notarization requested but codesign not available", nil)
	}

	for _, artifact := range artifacts {
		if artifact.OS != "darwin" {
			continue
		}

		core.Print(nil, "  Notarizing %s (this may take a few minutes)...", artifact.Path)
		if err := signer.Notarize(ctx, fs, artifact.Path); err != nil {
			return coreerr.E("signing.NotarizeBinaries", "failed to notarize "+artifact.Path, err)
		}
	}

	return nil
}

// SignChecksums signs the checksums file with GPG.
// Usage example: call signing.SignChecksums(...) from integrating code.
func SignChecksums(ctx context.Context, fs io.Medium, cfg SignConfig, checksumFile string) error {
	if !cfg.Enabled {
		return nil
	}

	signer := NewGPGSigner(cfg.GPG.Key)
	if !signer.Available() {
		return nil // Silently skip if not configured
	}

	core.Print(nil, "  Signing %s with GPG...", checksumFile)
	if err := signer.Sign(ctx, fs, checksumFile); err != nil {
		return coreerr.E("signing.SignChecksums", "failed to sign checksums file "+checksumFile, err)
	}

	return nil
}
