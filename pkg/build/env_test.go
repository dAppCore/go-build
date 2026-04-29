package build

import core "dappco.re/go"

func TestEnv_BuildEnvironment_Good(t *core.T) {
	cfg := &Config{
		Env: []string{"APP_ENV=dev"},
		Cache: CacheConfig{
			Enabled: true,
			Paths:   []string{"cache/go-build"},
		},
	}

	env := BuildEnvironment(cfg, "EXTRA=1")
	core.AssertContains(t, env, "APP_ENV=dev")
	core.AssertContains(t, env, "GOCACHE=cache/go-build")
	core.AssertContains(t, env, "EXTRA=1")
}

func TestEnv_BuildEnvironment_Bad(t *core.T) {
	env := BuildEnvironment(nil, "EXTRA=1")
	core.AssertLen(t, env, 1)
	core.AssertEqual(t, []string{"EXTRA=1"}, env)
}

func TestEnv_BuildEnvironment_Ugly(t *core.T) {
	env := BuildEnvironment(&Config{})
	core.AssertEmpty(t, env)
	core.AssertNil(t, env)
}

func TestEnv_DenoRequested_Good(t *core.T) {
	clearBuildEnv(t, "DENO_ENABLE", "DENO_BUILD")
	setBuildEnv(t, "DENO_ENABLE", "true")
	defer clearBuildEnv(t, "DENO_ENABLE")

	core.AssertTrue(t, DenoRequested(""))
}

func TestEnv_DenoRequested_Bad(t *core.T) {
	clearBuildEnv(t, "DENO_ENABLE", "DENO_BUILD")
	requested := DenoRequested("")
	core.AssertFalse(t, requested)
	core.AssertEqual(t, false, requested)
}

func TestEnv_DenoRequested_Ugly(t *core.T) {
	clearBuildEnv(t, "DENO_ENABLE", "DENO_BUILD")
	requested := DenoRequested(" deno task build ")
	core.AssertTrue(t, requested)
	core.AssertEqual(t, true, requested)
}

func TestEnv_NpmRequested_Good(t *core.T) {
	clearBuildEnv(t, "NPM_BUILD")
	setBuildEnv(t, "NPM_BUILD", "npm run build")
	defer clearBuildEnv(t, "NPM_BUILD")

	core.AssertTrue(t, NpmRequested(""))
}

func TestEnv_NpmRequested_Bad(t *core.T) {
	clearBuildEnv(t, "NPM_BUILD")
	requested := NpmRequested("")
	core.AssertFalse(t, requested)
	core.AssertEqual(t, false, requested)
}

func TestEnv_NpmRequested_Ugly(t *core.T) {
	clearBuildEnv(t, "NPM_BUILD")
	requested := NpmRequested(" npm run assets ")
	core.AssertTrue(t, requested)
	core.AssertEqual(t, true, requested)
}

func setBuildEnv(t *core.T, key, value string) {
	t.Helper()
	setenv := core.Setenv
	r := setenv(key, value)
	core.RequireTrue(t, r.OK, r.Error())
}

func clearBuildEnv(t *core.T, keys ...string) {
	t.Helper()
	unsetenv := core.Unsetenv
	for _, key := range keys {
		r := unsetenv(key)
		core.RequireTrue(t, r.OK, r.Error())
	}
}
