package build

import core "dappco.re/go"

// ExampleTarget_String references Target.String on this package API surface.
func ExampleTarget_String() {
	_ = (*Target).String
	core.Println("Target.String")
	// Output: Target.String
}
