package build

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
