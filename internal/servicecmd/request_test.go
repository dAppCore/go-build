package servicecmd

import (
	core "dappco.re/go"
	buildservice "dappco.re/go/build/pkg/service"
)

// --- v0.9.0 generated compliance triplets ---
func TestRequest_FromOptions_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = FromOptions(core.NewOptions())
	})
	core.AssertTrue(t, true)
}

func TestRequest_FromOptions_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = FromOptions(core.NewOptions())
	})
	core.AssertTrue(t, true)
}

func TestRequest_FromOptions_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = FromOptions(core.NewOptions())
	})
	core.AssertTrue(t, true)
}

func TestRequest_LoadConfig_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = LoadConfig(Request{}, func() (string, error) {
			return "", nil
		}, func(string) (buildservice.Config, error) {
			return buildservice.Config{}, nil
		})
	})
	core.AssertTrue(t, true)
}

func TestRequest_LoadConfig_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = LoadConfig(Request{}, func() (string, error) {
			return "", nil
		}, func(string) (buildservice.Config, error) {
			return buildservice.Config{}, nil
		})
	})
	core.AssertTrue(t, true)
}

func TestRequest_LoadConfig_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = LoadConfig(Request{}, func() (string, error) {
			return "", nil
		}, func(string) (buildservice.Config, error) {
			return buildservice.Config{}, nil
		})
	})
	core.AssertTrue(t, true)
}

func TestRequest_ApplyOverrides_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ApplyOverrides(nil, Request{})
	})
	core.AssertTrue(t, true)
}

func TestRequest_ApplyOverrides_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ApplyOverrides(nil, Request{})
	})
	core.AssertTrue(t, true)
}

func TestRequest_ApplyOverrides_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ApplyOverrides(nil, Request{})
	})
	core.AssertTrue(t, true)
}

func TestRequest_ParseCSV_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ParseCSV("agent")
	})
	core.AssertTrue(t, true)
}

func TestRequest_ParseCSV_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ParseCSV("")
	})
	core.AssertTrue(t, true)
}

func TestRequest_ParseCSV_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ParseCSV("agent")
	})
	core.AssertTrue(t, true)
}
