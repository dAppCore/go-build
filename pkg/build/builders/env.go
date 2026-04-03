package builders

// appendConfiguredEnv returns a fresh environment slice with the configured
// build environment prepended to any builder-specific values.
func appendConfiguredEnv(base []string, extra ...string) []string {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}

	env := append([]string{}, base...)
	env = append(env, extra...)
	return env
}
