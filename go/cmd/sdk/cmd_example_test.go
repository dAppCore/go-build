package sdkcmd

import (
	core "github.com/dappcore/go"
)

// --- v0.9.0 generated usage examples ---
func ExampleAddSDKCommands() {
	_ = AddSDKCommands(core.New())
	core.Println("AddSDKCommands")
	// Output: AddSDKCommands
}
