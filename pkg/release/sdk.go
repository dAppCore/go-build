// Package release provides release automation with changelog generation and publishing.
package release

import (
	"context"

	"dappco.re/go/core"
	"dappco.re/go/core/build/pkg/sdk"
	coreerr "dappco.re/go/core/log"
)

// SDKRelease holds the result of an SDK release.
//
// rel, err := release.RunSDK(ctx, cfg, false)
type SDKRelease struct {
	// Version is the SDK version.
	Version string
	// Languages that were generated.
	Languages []string
	// Output directory.
	Output string
}

// RunSDK executes SDK-only release: diff check + generate.
//
// rel, err := release.RunSDK(ctx, cfg, false) // dryRun=true to preview
func RunSDK(ctx context.Context, cfg *Config, dryRun bool) (*SDKRelease, error) {
	if cfg == nil {
		return nil, coreerr.E("release.RunSDK", "config is nil", nil)
	}
	if cfg.SDK == nil {
		return nil, coreerr.E("release.RunSDK", "sdk not configured in .core/release.yaml", nil)
	}

	projectDir := cfg.projectDir
	if projectDir == "" {
		projectDir = "."
	}

	// Determine version
	version := cfg.version
	if version == "" {
		var err error
		version, err = DetermineVersionWithContext(ctx, projectDir)
		if err != nil {
			return nil, coreerr.E("release.RunSDK", "failed to determine version", err)
		}
	}

	// Run diff check if enabled
	if cfg.SDK.Diff.Enabled {
		breaking, err := checkBreakingChanges(ctx, projectDir, cfg.SDK)
		if err != nil {
			if ctx.Err() != nil {
				return nil, coreerr.E("release.RunSDK", "diff check cancelled", ctx.Err())
			}
			// Non-fatal: warn and continue
			core.Print(nil, "Warning: diff check failed: %v", err)
		} else if breaking {
			if cfg.SDK.Diff.FailOnBreaking {
				return nil, coreerr.E("release.RunSDK", "breaking API changes detected", nil)
			}
			core.Print(nil, "Warning: breaking API changes detected")
		}
	}

	// Prepare result
	output := cfg.SDK.Output
	if output == "" {
		output = "sdk"
	}

	result := &SDKRelease{
		Version:   version,
		Languages: cfg.SDK.Languages,
		Output:    output,
	}

	if dryRun {
		return result, nil
	}

	// Generate SDKs
	sdkCfg := toSDKConfig(cfg.SDK)
	s := sdk.New(projectDir, sdkCfg)
	s.SetVersion(version)

	if err := s.Generate(ctx); err != nil {
		return nil, coreerr.E("release.RunSDK", "generation failed", err)
	}

	return result, nil
}

// checkBreakingChanges runs oasdiff to detect breaking changes.
func checkBreakingChanges(ctx context.Context, projectDir string, cfg *SDKConfig) (bool, error) {
	// Get previous tag for comparison (uses getPreviousTag from changelog.go)
	prevTag, err := getPreviousTagWithContext(ctx, projectDir, "HEAD")
	if err != nil {
		return false, coreerr.E("release.checkBreakingChanges", "no previous tag found", err)
	}

	// Detect spec path
	specPath := cfg.Spec
	if specPath == "" {
		s := sdk.New(projectDir, nil)
		specPath, err = s.DetectSpec()
		if err != nil {
			return false, err
		}
	}

	// Run diff
	result, err := sdk.Diff(prevTag, specPath)
	if err != nil {
		return false, err
	}

	return result.Breaking, nil
}

// toSDKConfig converts release.SDKConfig to sdk.Config.
func toSDKConfig(cfg *SDKConfig) *sdk.Config {
	if cfg == nil {
		return nil
	}
	return &sdk.Config{
		Spec:      cfg.Spec,
		Languages: cfg.Languages,
		Output:    cfg.Output,
		Package: sdk.PackageConfig{
			Name:    cfg.Package.Name,
			Version: cfg.Package.Version,
		},
		Diff: sdk.DiffConfig{
			Enabled:        cfg.Diff.Enabled,
			FailOnBreaking: cfg.Diff.FailOnBreaking,
		},
	}
}
