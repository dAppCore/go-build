package builders

import core "dappco.re/go"

// ExampleAppleBuilder_CreateDMG references AppleBuilder.CreateDMG on this package API surface.
func ExampleAppleBuilder_CreateDMG() {
	_ = (*AppleBuilder).CreateDMG
	core.Println("AppleBuilder.CreateDMG")
	// Output: AppleBuilder.CreateDMG
}
