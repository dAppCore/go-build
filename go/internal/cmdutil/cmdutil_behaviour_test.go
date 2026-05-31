package cmdutil

import (
	core "dappco.re/go"
)

// Behaviour tests drive the real option-resolution branches rather than the
// generated no-panic triplets: first-non-empty selection, bool coercion from
// both native bool and parseable strings, the parse-failure and missing-key
// fall-backs, and the error adaptation.

func opts(pairs ...core.Option) core.Options {
	return core.NewOptions(pairs...)
}

func TestCmdutil_OptionString_Behaviour_Good(t *core.T) {
	o := opts(core.Option{Key: "name", Value: "agent"})
	core.AssertEqual(t, "agent", OptionString(o, "name"))
}

func TestCmdutil_OptionString_Behaviour_Bad(t *core.T) {
	// No keys supplied and an empty value both yield the empty string.
	core.AssertEqual(t, "", OptionString(opts()))
	o := opts(core.Option{Key: "name", Value: ""})
	core.AssertEqual(t, "", OptionString(o, "name"))
}

func TestCmdutil_OptionString_Behaviour_Ugly(t *core.T) {
	// First key empty, second key populated: the loop skips the empty one.
	o := opts(
		core.Option{Key: "build-name", Value: ""},
		core.Option{Key: "name", Value: "fallback"},
	)
	core.AssertEqual(t, "fallback", OptionString(o, "build-name", "name"))
}

func TestCmdutil_OptionBoolDefault_Behaviour_Good(t *core.T) {
	o := opts(core.Option{Key: "obfuscate", Value: true})
	core.AssertTrue(t, OptionBoolDefault(o, false, "obfuscate"))

	o = opts(core.Option{Key: "obfuscate", Value: false})
	core.AssertFalse(t, OptionBoolDefault(o, true, "obfuscate"))
}

func TestCmdutil_OptionBoolDefault_Behaviour_Bad(t *core.T) {
	// Missing key falls back to the supplied default.
	core.AssertTrue(t, OptionBoolDefault(opts(), true, "missing"))
	core.AssertFalse(t, OptionBoolDefault(opts(), false, "missing"))
}

func TestCmdutil_OptionBoolDefault_Behaviour_Ugly(t *core.T) {
	// Parseable string values coerce to bool.
	o := opts(core.Option{Key: "nsis", Value: "true"})
	core.AssertTrue(t, OptionBoolDefault(o, false, "nsis"))
	o = opts(core.Option{Key: "nsis", Value: "0"})
	core.AssertFalse(t, OptionBoolDefault(o, true, "nsis"))

	// Unparseable string falls through to the default.
	o = opts(core.Option{Key: "nsis", Value: "maybe"})
	core.AssertTrue(t, OptionBoolDefault(o, true, "nsis"))

	// Non-bool, non-string value is ignored and the default returns.
	o = opts(core.Option{Key: "nsis", Value: 42})
	core.AssertFalse(t, OptionBoolDefault(o, false, "nsis"))

	// First key missing, second key present: the loop continues to the hit.
	o = opts(core.Option{Key: "deno-build", Value: true})
	core.AssertTrue(t, OptionBoolDefault(o, false, "missing", "deno-build"))
}

func TestCmdutil_OptionBool_Behaviour_Good(t *core.T) {
	o := opts(core.Option{Key: "cache", Value: true})
	core.AssertTrue(t, OptionBool(o, "cache"))
}

func TestCmdutil_OptionBool_Behaviour_Bad(t *core.T) {
	// OptionBool defaults to false when nothing matches.
	core.AssertFalse(t, OptionBool(opts(), "cache"))
}

func TestCmdutil_OptionHas_Behaviour_Good(t *core.T) {
	o := opts(core.Option{Key: "wails-build-webview2", Value: "embed"})
	core.AssertTrue(t, OptionHas(o, "wails-build-webview2"))
}

func TestCmdutil_OptionHas_Behaviour_Bad(t *core.T) {
	core.AssertFalse(t, OptionHas(opts(), "wails-build-webview2"))
}

func TestCmdutil_OptionHas_Behaviour_Ugly(t *core.T) {
	// First key absent, second present.
	o := opts(core.Option{Key: "build-platform", Value: "linux/amd64"})
	core.AssertTrue(t, OptionHas(o, "platform", "build-platform"))
}

func TestCmdutil_ResultFromError_Behaviour_Good(t *core.T) {
	r := ResultFromError(nil)
	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, nil, r.Value)
}

func TestCmdutil_ResultFromError_Behaviour_Bad(t *core.T) {
	err := core.E("cmdutil.test", "boom", nil)
	r := ResultFromError(err)
	core.AssertFalse(t, r.OK)
	core.AssertEqual(t, err, r.Value)
}

func TestCmdutil_ContextOrBackground_Behaviour_Good(t *core.T) {
	// Outside a live CLI dispatch currentCLIContext recovers and we fall back
	// to a non-nil background context.
	ctx := ContextOrBackground()
	core.AssertFalse(t, ctx == nil)
}
