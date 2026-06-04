package signing

import core "dappco.re/go"

func TestSigner_DefaultSignConfig_Good(t *core.T) {
	clearSigningEnv(t, "GPG_KEY_ID")
	setSigningEnv(t, "GPG_KEY_ID", "ABC123")
	defer clearSigningEnv(t, "GPG_KEY_ID")

	cfg := DefaultSignConfig()
	core.AssertTrue(t, cfg.Enabled)
	core.AssertEqual(t, "ABC123", cfg.GPG.Key)
}

func TestSigner_DefaultSignConfig_Bad(t *core.T) {
	clearSigningEnv(t, "GPG_KEY_ID", "SIGNTOOL_CERTIFICATE")
	cfg := DefaultSignConfig()
	core.AssertTrue(t, cfg.Windows.Signtool)
	core.AssertEqual(t, "", cfg.GPG.Key)
}

func TestSigner_DefaultSignConfig_Ugly(t *core.T) {
	clearSigningEnv(t, "APPLE_TEAM_ID")
	setSigningEnv(t, "APPLE_TEAM_ID", "TEAM123")
	defer clearSigningEnv(t, "APPLE_TEAM_ID")

	cfg := DefaultSignConfig()
	core.AssertEqual(t, "TEAM123", cfg.MacOS.TeamID)
}

func TestSigner_SignConfig_ExpandEnv_Good(t *core.T) {
	clearSigningEnv(t, "GPG_KEY_ID")
	setSigningEnv(t, "GPG_KEY_ID", "ABC123")
	defer clearSigningEnv(t, "GPG_KEY_ID")

	cfg := SignConfig{GPG: GPGConfig{Key: "$GPG_KEY_ID"}}
	cfg.ExpandEnv()
	core.AssertEqual(t, "ABC123", cfg.GPG.Key)
}

func TestSigner_SignConfig_ExpandEnv_Bad(t *core.T) {
	cfg := SignConfig{GPG: GPGConfig{Key: "$"}}
	cfg.ExpandEnv()
	core.AssertEqual(t, "$", cfg.GPG.Key)
}

func TestSigner_SignConfig_ExpandEnv_Ugly(t *core.T) {
	clearSigningEnv(t, "SIGNTOOL_PASSWORD")
	setSigningEnv(t, "SIGNTOOL_PASSWORD", "secret")
	defer clearSigningEnv(t, "SIGNTOOL_PASSWORD")

	cfg := SignConfig{Windows: WindowsConfig{Password: "${SIGNTOOL_PASSWORD}"}}
	cfg.ExpandEnv()
	core.AssertEqual(t, "secret", cfg.Windows.Password)
}

func TestSigner_WindowsConfig_SetSigntool_Good(t *core.T) {
	cfg := WindowsConfig{}
	cfg.SetSigntool(false)
	core.AssertFalse(t, cfg.signtoolEnabled())
}

func TestSigner_WindowsConfig_SetSigntool_Bad(t *core.T) {
	var cfg *WindowsConfig
	core.AssertNotPanics(t, func() {
		cfg.SetSigntool(false)
	})
	core.AssertNil(t, cfg)
}

func TestSigner_WindowsConfig_SetSigntool_Ugly(t *core.T) {
	cfg := WindowsConfig{}
	core.AssertTrue(t, cfg.signtoolEnabled())
	cfg.SetSigntool(true)
	core.AssertTrue(t, cfg.signtoolEnabled())
}

// --- expandEnv parser edge cases (signer.go) ---

func TestSigner_expandEnv_Good(t *core.T) {
	// Both $VAR and ${VAR} forms expand; surrounding literals are preserved.
	clearSigningEnv(t, "EE_TOKEN")
	setSigningEnv(t, "EE_TOKEN", "xyz")
	defer clearSigningEnv(t, "EE_TOKEN")

	core.AssertEqual(t, "pre-xyz-post", expandEnv("pre-$EE_TOKEN-post"))
	core.AssertEqual(t, "[xyz]", expandEnv("[${EE_TOKEN}]"))
}

func TestSigner_expandEnv_Bad(t *core.T) {
	// A string with no '$' is returned unchanged (fast path), and an unknown
	// variable expands to empty.
	clearSigningEnv(t, "EE_UNSET_VAR")
	core.AssertEqual(t, "plain text", expandEnv("plain text"))
	core.AssertEqual(t, "()", expandEnv("($EE_UNSET_VAR)"))
}

func TestSigner_expandEnv_Ugly(t *core.T) {
	// Malformed/degenerate uses of '$' are preserved literally: an unclosed
	// brace, a '$' before a non-identifier byte, and a trailing '$'.
	core.AssertEqual(t, "${UNCLOSED", expandEnv("${UNCLOSED"))
	core.AssertEqual(t, "a$!b", expandEnv("a$!b"))
	core.AssertEqual(t, "end$", expandEnv("end$"))
}

func setSigningEnv(t *core.T, key, value string) {
	t.Helper()
	setenv := core.Setenv
	r := setenv(key, value)
	core.RequireTrue(t, r.OK, r.Error())
}

func clearSigningEnv(t *core.T, keys ...string) {
	t.Helper()
	unsetenv := core.Unsetenv
	for _, key := range keys {
		r := unsetenv(key)
		core.RequireTrue(t, r.OK, r.Error())
	}
}
