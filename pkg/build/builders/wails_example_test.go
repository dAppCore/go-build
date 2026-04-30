package builders

import core "dappco.re/go"

// ExampleNewWailsBuilder references NewWailsBuilder on this package API surface.
func ExampleNewWailsBuilder() {
	_ = NewWailsBuilder
	core.Println("NewWailsBuilder")
	// Output: NewWailsBuilder
}

// ExampleWailsBuilder_Name references WailsBuilder.Name on this package API surface.
func ExampleWailsBuilder_Name() {
	_ = (*WailsBuilder).Name
	core.Println("WailsBuilder.Name")
	// Output: WailsBuilder.Name
}

// ExampleWailsBuilder_Detect references WailsBuilder.Detect on this package API surface.
func ExampleWailsBuilder_Detect() {
	_ = (*WailsBuilder).Detect
	core.Println("WailsBuilder.Detect")
	// Output: WailsBuilder.Detect
}

// ExampleWailsBuilder_Build references WailsBuilder.Build on this package API surface.
func ExampleWailsBuilder_Build() {
	_ = (*WailsBuilder).Build
	core.Println("WailsBuilder.Build")
	// Output: WailsBuilder.Build
}

// ExampleWailsBuilder_PreBuild references WailsBuilder.PreBuild on this package API surface.
func ExampleWailsBuilder_PreBuild() {
	_ = (*WailsBuilder).PreBuild
	core.Println("WailsBuilder.PreBuild")
	// Output: WailsBuilder.PreBuild
}
