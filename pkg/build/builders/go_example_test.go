package builders

import core "dappco.re/go"

// ExampleNewGoBuilder references NewGoBuilder on this package API surface.
func ExampleNewGoBuilder() {
	_ = NewGoBuilder
	core.Println("NewGoBuilder")
	// Output: NewGoBuilder
}

// ExampleGoBuilder_Name references GoBuilder.Name on this package API surface.
func ExampleGoBuilder_Name() {
	_ = (*GoBuilder).Name
	core.Println("GoBuilder.Name")
	// Output: GoBuilder.Name
}

// ExampleGoBuilder_Detect references GoBuilder.Detect on this package API surface.
func ExampleGoBuilder_Detect() {
	_ = (*GoBuilder).Detect
	core.Println("GoBuilder.Detect")
	// Output: GoBuilder.Detect
}

// ExampleGoBuilder_Build references GoBuilder.Build on this package API surface.
func ExampleGoBuilder_Build() {
	_ = (*GoBuilder).Build
	core.Println("GoBuilder.Build")
	// Output: GoBuilder.Build
}
