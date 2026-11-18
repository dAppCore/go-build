package buildcmd

import core "github.com/dappcore/go"

// ExampleAddAppleCommand references AddAppleCommand on this package API surface.
func ExampleAddAppleCommand() {
	_ = AddAppleCommand
	core.Println("AddAppleCommand")
	// Output: AddAppleCommand
}
