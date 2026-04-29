// Package release provides release automation with changelog generation and publishing.
package release

import (
	"context"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/build/pkg/sdk"
	"dappco.re/go/io"
	coreerr "dappco.re/go/log"
)

// SDKRelease holds the result of an SDK release.
//
// result := release.RunSDK(ctx, cfg, false)
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
// result := release.RunSDK(ctx, cfg, false) // dryRun=true to preview
func RunSDK(ctx context.Context, cfg *Config, dryRun bool) core.Result {
	if cfg == nil {
		return core.Fail(coreerr.E("release.RunSDK", "config is nil", nil))
	}

	projectDir := cfg.projectDir
	if projectDir == "" {
		projectDir = "."
	}

	sdkConfigResult := resolveReleaseSDKConfig(projectDir, cfg)
	if !sdkConfigResult.OK {
		return sdkConfigResult
	}
	sdkConfig := sdkConfigResult.Value.(*sdk.Config)

	s := sdk.New(projectDir, sdkConfig)
	sdkConfig = s.Config()
	if sdkConfig == nil {
		return core.Fail(coreerr.E("release.RunSDK", "failed to resolve sdk config", nil))
	}

	// Determine version
	version := cfg.version
	if version == "" {
		versionResult := DetermineVersionWithContext(ctx, projectDir)
		if !versionResult.OK {
			return core.Fail(coreerr.E("release.RunSDK", "failed to determine version", core.NewError(versionResult.Error())))
		}
		version = versionResult.Value.(string)
	}
	validatedVersion := ValidateVersionIdentifier(version)
	if !validatedVersion.OK {
		return core.Fail(coreerr.E("release.RunSDK", "invalid SDK release version override", core.NewError(validatedVersion.Error())))
	}

	// Run diff check if enabled
	if sdkConfig.Diff.Enabled {
		breakingResult := checkBreakingChanges(ctx, projectDir, sdkConfig)
		if !breakingResult.OK {
			if ctx.Err() != nil {
				return core.Fail(coreerr.E("release.RunSDK", "diff check cancelled", ctx.Err()))
			}
			// Non-fatal: warn and continue
			core.Print(nil, "Warning: diff check failed: %v", breakingResult.Error())
		} else if breakingResult.Value.(bool) {
			if sdkConfig.Diff.FailOnBreaking {
				return core.Fail(coreerr.E("release.RunSDK", "breaking API changes detected", nil))
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
		return core.Ok(result)
	}

	// Generate SDKs
	s.SetVersion(version)

	generated := s.Generate(ctx)
	if !generated.OK {
		return core.Fail(coreerr.E("release.RunSDK", "generation failed", core.NewError(generated.Error())))
	}

	return core.Ok(result)
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
func checkBreakingChanges(ctx context.Context, projectDir string, cfg *SDKConfig) core.Result {
	// Get previous tag for comparison (uses getPreviousTag from changelog.go)
	prevTagResult := getPreviousTagWithContext(ctx, projectDir, "HEAD")
	if !prevTagResult.OK {
		return core.Fail(coreerr.E("release.checkBreakingChanges", "no previous tag found", core.NewError(prevTagResult.Error())))
	}
	prevTag := prevTagResult.Value.(string)

	// Detect spec path
	specPathResult := detectSDKSpecPath(projectDir, cfg)
	if !specPathResult.OK {
		return specPathResult
	}
	specPath := specPathResult.Value.(string)

	taggedSpecResult := materializeTaggedSDKSpec(ctx, projectDir, prevTag, specPath)
	if !taggedSpecResult.OK {
		return taggedSpecResult
	}
	taggedSpec := taggedSpecResult.Value.(taggedSDKSpec)
	defer taggedSpec.cleanup()

	// Run diff
	result := sdk.Diff(taggedSpec.path, specPath)
	if !result.OK {
		return result
	}
	diffResult := result.Value.(*sdk.DiffResult)

	return core.Ok(diffResult.Breaking)
}

func detectSDKSpecPath(projectDir string, cfg *SDKConfig) core.Result {
	specCfg := &sdk.Config{}
	if cfg != nil {
		specCfg.Spec = cfg.Spec
	}

	return sdk.New(projectDir, specCfg).DetectSpec()
}

type taggedSDKSpec struct {
	path    string
	cleanup func()
}

func materializeTaggedSDKSpec(ctx context.Context, projectDir, tag, specPath string) core.Result {
	relativeSpecPathResult := ax.Rel(projectDir, specPath)
	if !relativeSpecPathResult.OK {
		return core.Fail(coreerr.E("release.materializeTaggedSDKSpec", "spec path must be inside the project directory", core.NewError(relativeSpecPathResult.Error())))
	}
	relativeSpecPath := relativeSpecPathResult.Value.(string)

	gitSpecPath := core.Replace(relativeSpecPath, ax.DS(), "/")
	contentResult := ax.RunDir(ctx, projectDir, "git", "show", core.Sprintf("%s:%s", tag, gitSpecPath))
	if !contentResult.OK {
		return core.Fail(coreerr.E("release.materializeTaggedSDKSpec", "failed to load spec from "+tag, core.NewError(contentResult.Error())))
	}
	content := contentResult.Value.(string)

	tempDirResult := ax.TempDir("core-build-sdk-diff-*")
	if !tempDirResult.OK {
		return core.Fail(coreerr.E("release.materializeTaggedSDKSpec", "failed to create temp dir", core.NewError(tempDirResult.Error())))
	}
	tempDir := tempDirResult.Value.(string)

	tempPath := ax.Join(tempDir, "base"+ax.Ext(specPath))
	written := ax.WriteString(tempPath, content, 0o644)
	if !written.OK {
		cleaned := ax.RemoveAll(tempDir)
		if !cleaned.OK {
			return core.Fail(coreerr.E("release.materializeTaggedSDKSpec", "failed to clean up temp dir", core.NewError(cleaned.Error())))
		}
		return core.Fail(coreerr.E("release.materializeTaggedSDKSpec", "failed to write tagged spec", core.NewError(written.Error())))
	}

	return core.Ok(taggedSDKSpec{path: tempPath, cleanup: func() { ax.RemoveAll(tempDir) }})
}

func resolveReleaseSDKConfig(projectDir string, cfg *Config) core.Result {
	if cfg != nil && cfg.SDK != nil {
		resolved := toSDKConfig(cfg.SDK)
		resolved.ApplyDefaults()
		return core.Ok(resolved)
	}

	buildCfgResult := build.LoadConfig(io.Local, projectDir)
	if !buildCfgResult.OK {
		return core.Fail(coreerr.E("release.resolveReleaseSDKConfig", "failed to load build config", core.NewError(buildCfgResult.Error())))
	}
	buildCfg := buildCfgResult.Value.(*build.BuildConfig)
	if buildCfg != nil && buildCfg.SDK != nil {
		resolved := sdk.CloneConfig(buildCfg.SDK)
		resolved.ApplyDefaults()
		return core.Ok(resolved)
	}

	resolved := sdk.DefaultConfig()
	resolved.ApplyDefaults()
	return core.Ok(resolved)
}

// toSDKConfig clones release SDK config into the runtime SDK config type.
func toSDKConfig(cfg *SDKConfig) *sdk.Config {
	return sdk.CloneConfig(cfg)
}
