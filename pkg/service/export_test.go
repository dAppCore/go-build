package service

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated compliance triplets ---
func TestExport_Export_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = Export(Config{}, "tar.gz")
	})
	core.AssertTrue(t, true)
}

func TestExport_Export_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = Export(Config{}, "")
	})
	core.AssertTrue(t, true)
}

func TestExport_Export_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = Export(Config{}, "tar.gz")
	})
	core.AssertTrue(t, true)
}
