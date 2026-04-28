package build

import "dappco.re/go"

// BuildEnvironment returns a fresh environment slice that includes the
// configured build environment, any derived cache variables, and optional
// builder-specific values.
func BuildEnvironment(cfg *Config, extra ...string) []string {
	if cfg == nil {
		if len(extra) == 0 {
			return nil
		}
		return append([]string{}, extra...)
	}

	env := append([]string{}, cfg.Env...)
	env = append(env, CacheEnvironment(&cfg.Cache)...)
	env = append(env, extra...)

	if len(env) == 0 {
		return nil
	}

	return env
}

// DenoRequested reports whether the current build should prefer a Deno-backed
// frontend build. It honours the action-style environment overrides first and
// then the persisted/configured command override.
func DenoRequested(configuredBuild string) bool {
	if truthyEnv(core.Env("DENO_ENABLE")) {
		return true
	}

	if core.Trim(core.Env("DENO_BUILD")) != "" {
		return true
	}

	return core.Trim(configuredBuild) != ""
}

// NpmRequested reports whether the current build should prefer an npm-backed
// frontend build. It honours the action-style environment override first and
// then the persisted/configured command override.
func NpmRequested(configuredBuild string) bool {
	if core.Trim(core.Env("NPM_BUILD")) != "" {
		return true
	}

	return core.Trim(configuredBuild) != ""
}

func truthyEnv(value string) bool {
	switch core.Lower(core.Trim(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
