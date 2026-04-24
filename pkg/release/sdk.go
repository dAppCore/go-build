// Package release provides release automation with changelog generation and publishing.
package release

import (
	"context"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/build/pkg/sdk"
	"dappco.re/go/core"
	"dappco.re/go/io"
	coreerr "dappco.re/go/log"
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

	projectDir := cfg.projectDir
	if projectDir == "" {
		projectDir = "."
	}

	sdkConfig, err := resolveReleaseSDKConfig(projectDir, cfg)
	if err != nil {
		return nil, err
	}

	s := sdk.New(projectDir, sdkConfig)
	sdkConfig = s.Config()
	if sdkConfig == nil {
		return nil, coreerr.E("release.RunSDK", "failed to resolve sdk config", nil)
	}

	// Determine version
	version := cfg.version
	if version == "" {
		version, err = DetermineVersionWithContext(ctx, projectDir)
		if err != nil {
			return nil, coreerr.E("release.RunSDK", "failed to determine version", err)
		}
	}
	if err := ValidateVersionIdentifier(version); err != nil {
		return nil, coreerr.E("release.RunSDK", "invalid SDK release version override", err)
	}

	// Run diff check if enabled
	if sdkConfig.Diff.Enabled {
		breaking, err := checkBreakingChanges(ctx, projectDir, sdkConfig)
		if err != nil {
			if ctx.Err() != nil {
				return nil, coreerr.E("release.RunSDK", "diff check cancelled", ctx.Err())
			}
			// Non-fatal: warn and continue
			core.Print(nil, "Warning: diff check failed: %v", err)
		} else if breaking {
			if sdkConfig.Diff.FailOnBreaking {
				return nil, coreerr.E("release.RunSDK", "breaking API changes detected", nil)
			}
			core.Print(nil, "Warning: breaking API changes detected")
		}
	}

	// Prepare result
	output := resolveSDKOutputRoot(sdkConfig)

	result := &SDKRelease{
		Version:   version,
		Languages: append([]string(nil), sdkConfig.Languages...),
		Output:    output,
	}

	if dryRun {
		return result, nil
	}

	// Generate SDKs
	s.SetVersion(version)

	if err := s.Generate(ctx); err != nil {
		return nil, coreerr.E("release.RunSDK", "generation failed", err)
	}

	return result, nil
}

// resolveSDKOutputRoot returns the configured SDK output directory, including
// any monorepo publish path prefix.
//
// output := resolveSDKOutputRoot(cfg.SDK) // "sdk" or "packages/api-client/sdk"
func resolveSDKOutputRoot(cfg *SDKConfig) string {
	if cfg == nil {
		return "sdk"
	}

	output := cfg.Output
	if output == "" {
		output = "sdk"
	}

	if cfg.Publish.Path != "" {
		output = ax.Join(cfg.Publish.Path, output)
	}

	return output
}

// checkBreakingChanges runs oasdiff to detect breaking changes.
func checkBreakingChanges(ctx context.Context, projectDir string, cfg *SDKConfig) (bool, error) {
	// Get previous tag for comparison (uses getPreviousTag from changelog.go)
	prevTag, err := getPreviousTagWithContext(ctx, projectDir, "HEAD")
	if err != nil {
		return false, coreerr.E("release.checkBreakingChanges", "no previous tag found", err)
	}

	// Detect spec path
	specPath, err := detectSDKSpecPath(projectDir, cfg)
	if err != nil {
		return false, err
	}

	baseSpecPath, cleanup, err := materializeTaggedSDKSpec(ctx, projectDir, prevTag, specPath)
	if err != nil {
		return false, err
	}
	defer cleanup()

	// Run diff
	result, err := sdk.Diff(baseSpecPath, specPath)
	if err != nil {
		return false, err
	}

	return result.Breaking, nil
}

func detectSDKSpecPath(projectDir string, cfg *SDKConfig) (string, error) {
	specCfg := &sdk.Config{}
	if cfg != nil {
		specCfg.Spec = cfg.Spec
	}

	return sdk.New(projectDir, specCfg).DetectSpec()
}

func materializeTaggedSDKSpec(ctx context.Context, projectDir, tag, specPath string) (string, func(), error) {
	relativeSpecPath, err := ax.Rel(projectDir, specPath)
	if err != nil {
		return "", func() {}, coreerr.E("release.materializeTaggedSDKSpec", "spec path must be inside the project directory", err)
	}

	gitSpecPath := core.Replace(relativeSpecPath, ax.DS(), "/")
	content, err := ax.RunDir(ctx, projectDir, "git", "show", core.Sprintf("%s:%s", tag, gitSpecPath))
	if err != nil {
		return "", func() {}, coreerr.E("release.materializeTaggedSDKSpec", "failed to load spec from "+tag, err)
	}

	tempDir, err := ax.TempDir("core-build-sdk-diff-*")
	if err != nil {
		return "", func() {}, coreerr.E("release.materializeTaggedSDKSpec", "failed to create temp dir", err)
	}

	tempPath := ax.Join(tempDir, "base"+ax.Ext(specPath))
	if err := ax.WriteString(tempPath, content, 0o644); err != nil {
		_ = ax.RemoveAll(tempDir)
		return "", func() {}, coreerr.E("release.materializeTaggedSDKSpec", "failed to write tagged spec", err)
	}

	return tempPath, func() {
		_ = ax.RemoveAll(tempDir)
	}, nil
}

func resolveReleaseSDKConfig(projectDir string, cfg *Config) (*sdk.Config, error) {
	if cfg != nil && cfg.SDK != nil {
		resolved := toSDKConfig(cfg.SDK)
		resolved.ApplyDefaults()
		return resolved, nil
	}

	buildCfg, err := build.LoadConfig(io.Local, projectDir)
	if err != nil {
		return nil, coreerr.E("release.resolveReleaseSDKConfig", "failed to load build config", err)
	}
	if buildCfg != nil && buildCfg.SDK != nil {
		resolved := sdk.CloneConfig(buildCfg.SDK)
		resolved.ApplyDefaults()
		return resolved, nil
	}

	resolved := sdk.DefaultConfig()
	resolved.ApplyDefaults()
	return resolved, nil
}

// toSDKConfig clones release SDK config into the runtime SDK config type.
func toSDKConfig(cfg *SDKConfig) *sdk.Config {
	return sdk.CloneConfig(cfg)
}
