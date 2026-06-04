package builders

import core "dappco.re/go"

// ExampleNewCPPBuilder references NewCPPBuilder on this package API surface.
func ExampleNewCPPBuilder() {
	_ = NewCPPBuilder
	core.Println("NewCPPBuilder")
	// Output: NewCPPBuilder
}

// ExampleCPPBuilder_Name references CPPBuilder.Name on this package API surface.
func ExampleCPPBuilder_Name() {
	_ = (*CPPBuilder).Name
	core.Println("CPPBuilder.Name")
	// Output: CPPBuilder.Name
}

// ExampleCPPBuilder_Detect references CPPBuilder.Detect on this package API surface.
func ExampleCPPBuilder_Detect() {
	_ = (*CPPBuilder).Detect
	core.Println("CPPBuilder.Detect")
	// Output: CPPBuilder.Detect
}

// ExampleCPPBuilder_Build references CPPBuilder.Build on this package API surface.
func ExampleCPPBuilder_Build() {
	_ = (*CPPBuilder).Build
	core.Println("CPPBuilder.Build")
	// Output: CPPBuilder.Build
}
