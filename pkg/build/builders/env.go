package builders

import "dappco.re/go/build/pkg/build"

// appendConfiguredEnv returns a fresh environment slice that includes the
// configured build environment, derived cache variables, and any
// builder-specific values.
func appendConfiguredEnv(cfg *build.Config, extra ...string) []string {
	return build.BuildEnvironment(cfg, extra...)
}
