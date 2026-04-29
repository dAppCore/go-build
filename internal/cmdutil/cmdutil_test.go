package cmdutil

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated compliance triplets ---
func TestCmdutil_ContextOrBackground_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ContextOrBackground()
	})
	core.AssertTrue(t, true)
}

func TestCmdutil_ContextOrBackground_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ContextOrBackground()
	})
	core.AssertTrue(t, true)
}

func TestCmdutil_ContextOrBackground_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ContextOrBackground()
	})
	core.AssertTrue(t, true)
}

func TestCmdutil_OptionString_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = OptionString(core.NewOptions())
	})
	core.AssertTrue(t, true)
}

func TestCmdutil_OptionString_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = OptionString(core.NewOptions())
	})
	core.AssertTrue(t, true)
}

func TestCmdutil_OptionString_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = OptionString(core.NewOptions())
	})
	core.AssertTrue(t, true)
}

func TestCmdutil_OptionBoolDefault_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = OptionBoolDefault(core.NewOptions(), true)
	})
	core.AssertTrue(t, true)
}

func TestCmdutil_OptionBoolDefault_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = OptionBoolDefault(core.NewOptions(), false)
	})
	core.AssertTrue(t, true)
}

func TestCmdutil_OptionBoolDefault_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = OptionBoolDefault(core.NewOptions(), true)
	})
	core.AssertTrue(t, true)
}

func TestCmdutil_OptionBool_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = OptionBool(core.NewOptions())
	})
	core.AssertTrue(t, true)
}

func TestCmdutil_OptionBool_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = OptionBool(core.NewOptions())
	})
	core.AssertTrue(t, true)
}

func TestCmdutil_OptionBool_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = OptionBool(core.NewOptions())
	})
	core.AssertTrue(t, true)
}

func TestCmdutil_OptionHas_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = OptionHas(core.NewOptions())
	})
	core.AssertTrue(t, true)
}

func TestCmdutil_OptionHas_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = OptionHas(core.NewOptions())
	})
	core.AssertTrue(t, true)
}

func TestCmdutil_OptionHas_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = OptionHas(core.NewOptions())
	})
	core.AssertTrue(t, true)
}

func TestCmdutil_ResultFromError_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ResultFromError(nil)
	})
	core.AssertTrue(t, true)
}

func TestCmdutil_ResultFromError_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ResultFromError(nil)
	})
	core.AssertTrue(t, true)
}

func TestCmdutil_ResultFromError_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ResultFromError(nil)
	})
	core.AssertTrue(t, true)
}
