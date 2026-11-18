package sdkcfg

import (
	"github.com/dappcore/go"
	"github.com/dappcore/go-build/go/pkg/build"
	"github.com/dappcore/go-build/go/pkg/release"
	"github.com/dappcore/go-build/go/pkg/sdk"
	storage "github.com/dappcore/go-build/go/pkg/storage"
)

// LoadProjectConfig resolves SDK settings from build config first, then falls
// back to release config, and finally the SDK defaults when neither file
// defines an sdk section.
func LoadProjectConfig(fs storage.Medium, projectDir string) core.Result {
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
