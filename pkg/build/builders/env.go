package builders

import (
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/core/io"
)

// appendConfiguredEnv returns a fresh environment slice that includes the
// configured build environment, derived cache variables, and any
// builder-specific values.
func appendConfiguredEnv(cfg *build.Config, extra ...string) []string {
	return build.BuildEnvironment(cfg, extra...)
}

// ensureBuildFilesystem returns the filesystem associated with cfg, falling
// back to io.Local for zero-value configs. When cfg is non-nil, the fallback is
// also written back so downstream helpers that read cfg.FS stay safe.
func ensureBuildFilesystem(cfg *build.Config) io.Medium {
	if cfg == nil {
		return io.Local
	}
	if cfg.FS == nil {
		cfg.FS = io.Local
	}
	return cfg.FS
}
