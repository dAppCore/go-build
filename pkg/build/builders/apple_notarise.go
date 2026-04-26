package builders

import (
	"context"

	coreerr "dappco.re/go/log"
	"dappco.re/go/process"
)

// AppleNotariseConfig defines a notarisation request for a built Apple artifact.
type AppleNotariseConfig struct {
	AppPath        string
	Profile        string
	APIKeyID       string
	APIKeyIssuerID string
	APIKeyPath     string
	TeamID         string
	AppleID        string
	Password       string
}

// Notarise records notarytool submit and stapler staple invocations.
// A real run requires Apple Developer credentials, either through a
// notarytool keychain profile, App Store Connect API key, or Apple ID credentials.
func (b *AppleBuilder) Notarise(ctx context.Context, artifactPath string, options AppleOptions) error {
	if artifactPath == "" {
		return coreerr.E("AppleBuilder.Notarise", "artifact path is required", nil)
	}

	submitArgs := []string{
		"notarytool",
		"submit",
		artifactPath,
		"--wait",
	}
	submitArgs = append(submitArgs, appleNotaryAuthArgs(options)...)

	// TODO(#484): xcrun notarytool requires macOS and Apple Developer
	// credentials. The skeleton records the go-process invocation only.
	if err := b.runExternal(ctx, "notarytool-submit", process.RunOptions{
		Command: "xcrun",
		Args:    submitArgs,
	}); err != nil {
		return err
	}

	// TODO(#484): xcrun stapler requires a notarised artifact on macOS.
	return b.runExternal(ctx, "stapler-staple", process.RunOptions{
		Command: "xcrun",
		Args:    []string{"stapler", "staple", artifactPath},
	})
}

func appleNotaryAuthArgs(options AppleOptions) []string {
	if profile := options.notarisationProfile(); profile != "" {
		return []string{"--keychain-profile", profile}
	}

	if options.APIKeyID != "" {
		return []string{
			"--key", options.APIKeyPath,
			"--key-id", options.APIKeyID,
			"--issuer", options.APIKeyIssuerID,
		}
	}

	return []string{
		"--apple-id", options.AppleID,
		"--password", firstNonEmptyApple(options.AppPassword, options.Password),
		"--team-id", options.TeamID,
	}
}
