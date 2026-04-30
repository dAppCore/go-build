package ci

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated compliance triplets ---
func TestCmd_AddCICommands_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		AddCICommands(core.New())
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCmd_AddCICommands_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		AddCICommands(core.New())
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCmd_AddCICommands_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		AddCICommands(core.New())
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
