package sdkcfg

import (
	"dappco.re/go"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/build/pkg/release"
	"dappco.re/go/build/pkg/sdk"
	"dappco.re/go/io"
)

// LoadProjectConfig resolves SDK settings from build config first, then falls
// back to release config, and finally the SDK defaults when neither file
// defines an sdk section.
func LoadProjectConfig(fs io.Medium, projectDir string) core.Result {
	buildLoaded := build.LoadConfig(fs, projectDir)
	if !buildLoaded.OK {
		return buildLoaded
	}
	buildCfg := buildLoaded.Value.(*build.BuildConfig)
	if buildCfg != nil && buildCfg.SDK != nil {
		cfg := sdk.CloneConfig(buildCfg.SDK)
		cfg.ApplyDefaults()
		return core.Ok(cfg)
	}

	releaseLoaded := release.LoadConfigWithMedium(fs, projectDir)
	if !releaseLoaded.OK {
		return releaseLoaded
	}
	releaseCfg := releaseLoaded.Value.(*release.Config)
	if releaseCfg != nil && releaseCfg.SDK != nil {
		cfg := sdk.CloneConfig(releaseCfg.SDK)
		cfg.ApplyDefaults()
		return core.Ok(cfg)
	}

	cfg := sdk.DefaultConfig()
	cfg.ApplyDefaults()
	return core.Ok(cfg)
}
