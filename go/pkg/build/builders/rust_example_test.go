package builders

import core "dappco.re/go"

// ExampleNewRustBuilder references NewRustBuilder on this package API surface.
func ExampleNewRustBuilder() {
	_ = NewRustBuilder
	core.Println("NewRustBuilder")
	// Output: NewRustBuilder
}

// ExampleRustBuilder_Name references RustBuilder.Name on this package API surface.
func ExampleRustBuilder_Name() {
	_ = (*RustBuilder).Name
	core.Println("RustBuilder.Name")
	// Output: RustBuilder.Name
}

// ExampleRustBuilder_Detect references RustBuilder.Detect on this package API surface.
func ExampleRustBuilder_Detect() {
	_ = (*RustBuilder).Detect
	core.Println("RustBuilder.Detect")
	// Output: RustBuilder.Detect
}

// ExampleRustBuilder_Build references RustBuilder.Build on this package API surface.
func ExampleRustBuilder_Build() {
	_ = (*RustBuilder).Build
	core.Println("RustBuilder.Build")
	// Output: RustBuilder.Build
}
