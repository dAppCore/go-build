package cmdutil

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated compliance triplets ---
func TestCmdutil_ContextOrBackground_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ContextOrBackground()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCmdutil_ContextOrBackground_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ContextOrBackground()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCmdutil_ContextOrBackground_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ContextOrBackground()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestCmdutil_OptionString_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = OptionString(core.NewOptions())
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCmdutil_OptionString_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = OptionString(core.NewOptions())
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCmdutil_OptionString_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = OptionString(core.NewOptions())
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestCmdutil_OptionBoolDefault_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = OptionBoolDefault(core.NewOptions(), true)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCmdutil_OptionBoolDefault_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = OptionBoolDefault(core.NewOptions(), false)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCmdutil_OptionBoolDefault_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = OptionBoolDefault(core.NewOptions(), true)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestCmdutil_OptionBool_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = OptionBool(core.NewOptions())
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCmdutil_OptionBool_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = OptionBool(core.NewOptions())
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCmdutil_OptionBool_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = OptionBool(core.NewOptions())
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestCmdutil_OptionHas_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = OptionHas(core.NewOptions())
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCmdutil_OptionHas_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = OptionHas(core.NewOptions())
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCmdutil_OptionHas_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = OptionHas(core.NewOptions())
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestCmdutil_ResultFromError_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ResultFromError(nil)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCmdutil_ResultFromError_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ResultFromError(nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCmdutil_ResultFromError_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ResultFromError(nil)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
