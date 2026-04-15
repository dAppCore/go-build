package sdkcfg

import (
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/build/pkg/release"
	"dappco.re/go/build/pkg/sdk"
	"dappco.re/go/core/io"
)

// LoadProjectConfig resolves SDK settings from build config first, then falls
// back to release config, and finally the SDK defaults when neither file
// defines an sdk section.
func LoadProjectConfig(fs io.Medium, projectDir string) (*sdk.Config, error) {
	buildCfg, err := build.LoadConfig(fs, projectDir)
	if err != nil {
		return nil, err
	}
	if buildCfg != nil && buildCfg.SDK != nil {
		cfg := sdk.CloneConfig(buildCfg.SDK)
		cfg.ApplyDefaults()
		return cfg, nil
	}

	releaseCfg, err := release.LoadConfigWithMedium(fs, projectDir)
	if err != nil {
		return nil, err
	}
	if releaseCfg != nil && releaseCfg.SDK != nil {
		cfg := sdk.CloneConfig(releaseCfg.SDK)
		cfg.ApplyDefaults()
		return cfg, nil
	}

	cfg := sdk.DefaultConfig()
	cfg.ApplyDefaults()
	return cfg, nil
}
