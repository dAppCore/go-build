package builders

import core "dappco.re/go"

// ExampleAppleBuilder_Notarise references AppleBuilder.Notarise on this package API surface.
func ExampleAppleBuilder_Notarise() {
	_ = (*AppleBuilder).Notarise
	core.Println("AppleBuilder.Notarise")
	// Output: AppleBuilder.Notarise
}
