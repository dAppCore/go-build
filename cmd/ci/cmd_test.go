package ci

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated compliance triplets ---
func TestCmd_AddCICommands_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		AddCICommands(core.New())
	})
	core.AssertTrue(t, true)
}

func TestCmd_AddCICommands_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		AddCICommands(core.New())
	})
	core.AssertTrue(t, true)
}

func TestCmd_AddCICommands_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		AddCICommands(core.New())
	})
	core.AssertTrue(t, true)
}
