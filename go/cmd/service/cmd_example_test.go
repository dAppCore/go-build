package servicecmd

import (
	core "github.com/dappcore/go"
)

// --- v0.9.0 generated usage examples ---
func ExampleAddServiceCommands() {
	_ = AddServiceCommands(core.New())
	core.Println("AddServiceCommands")
	// Output: AddServiceCommands
}
