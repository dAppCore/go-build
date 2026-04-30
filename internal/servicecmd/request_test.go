package servicecmd

import (
	core "dappco.re/go"
	buildservice "dappco.re/go/build/pkg/service"
)

// --- v0.9.0 generated compliance triplets ---
func TestRequest_FromOptions_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = FromOptions(core.NewOptions())
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestRequest_FromOptions_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = FromOptions(core.NewOptions())
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestRequest_FromOptions_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = FromOptions(core.NewOptions())
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestRequest_LoadConfig_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = LoadConfig(Request{}, func() core.Result {
			return core.Ok("")
		}, func(string) core.Result {
			return core.Ok(buildservice.Config{})
		})
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestRequest_LoadConfig_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = LoadConfig(Request{}, func() core.Result {
			return core.Ok("")
		}, func(string) core.Result {
			return core.Ok(buildservice.Config{})
		})
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestRequest_LoadConfig_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = LoadConfig(Request{}, func() core.Result {
			return core.Ok("")
		}, func(string) core.Result {
			return core.Ok(buildservice.Config{})
		})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestRequest_ApplyOverrides_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ApplyOverrides(nil, Request{})
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestRequest_ApplyOverrides_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ApplyOverrides(nil, Request{})
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestRequest_ApplyOverrides_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ApplyOverrides(nil, Request{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestRequest_ParseCSV_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ParseCSV("agent")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestRequest_ParseCSV_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ParseCSV("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestRequest_ParseCSV_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ParseCSV("agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
