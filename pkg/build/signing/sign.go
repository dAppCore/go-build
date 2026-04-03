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
//
// a := signing.Artifact{Path: "dist/myapp", OS: "darwin", Arch: "arm64"}
type Artifact struct {
	Path string
	OS   string
	Arch string
}

// SignBinaries signs binaries for the current host OS in the artifacts list.
// On macOS it signs darwin artifacts with codesign; on Windows it signs windows
// artifacts with signtool when the relevant credentials are configured.
//
// err := signing.SignBinaries(ctx, io.Local, cfg, artifacts)
func SignBinaries(ctx context.Context, fs io.Medium, cfg SignConfig, artifacts []Artifact) error {
	if !cfg.Enabled {
		return nil
	}

	var signer Signer
	var targetOS string

	switch runtime.GOOS {
	case "darwin":
		signer = NewMacOSSigner(cfg.MacOS)
		targetOS = "darwin"
	case "windows":
		signer = NewWindowsSigner(cfg.Windows)
		targetOS = "windows"
	default:
		return nil
	}

	if !signer.Available() {
		return nil // Silently skip if not configured
	}

	return signArtifactsWithSigner(ctx, fs, signer, targetOS, artifacts)
}

// NotarizeBinaries notarizes macOS binaries if enabled.
//
// err := signing.NotarizeBinaries(ctx, io.Local, cfg, artifacts)
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
//
// err := signing.SignChecksums(ctx, io.Local, cfg, "dist/CHECKSUMS.txt")
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

func signArtifactsWithSigner(ctx context.Context, fs io.Medium, signer Signer, targetOS string, artifacts []Artifact) error {
	_ = fs

	for _, artifact := range artifacts {
		if artifact.OS != targetOS {
			continue
		}

		core.Print(nil, "  Signing %s...", artifact.Path)
		if err := signer.Sign(ctx, fs, artifact.Path); err != nil {
			return coreerr.E("signing.SignBinaries", "failed to sign "+artifact.Path, err)
		}
	}

	return nil
}
