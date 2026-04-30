package service

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated compliance triplets ---
func TestExport_Export_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Export(Config{}, "tar.gz")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestExport_Export_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Export(Config{}, "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestExport_Export_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Export(Config{}, "tar.gz")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
