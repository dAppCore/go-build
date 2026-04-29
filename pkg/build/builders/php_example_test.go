package builders

import core "dappco.re/go"

// ExampleNewPHPBuilder references NewPHPBuilder on this package API surface.
func ExampleNewPHPBuilder() {
	_ = NewPHPBuilder
	core.Println("NewPHPBuilder")
	// Output: NewPHPBuilder
}

// ExamplePHPBuilder_Name references PHPBuilder.Name on this package API surface.
func ExamplePHPBuilder_Name() {
	_ = (*PHPBuilder).Name
	core.Println("PHPBuilder.Name")
	// Output: PHPBuilder.Name
}

// ExamplePHPBuilder_Detect references PHPBuilder.Detect on this package API surface.
func ExamplePHPBuilder_Detect() {
	_ = (*PHPBuilder).Detect
	core.Println("PHPBuilder.Detect")
	// Output: PHPBuilder.Detect
}

// ExamplePHPBuilder_Build references PHPBuilder.Build on this package API surface.
func ExamplePHPBuilder_Build() {
	_ = (*PHPBuilder).Build
	core.Println("PHPBuilder.Build")
	// Output: PHPBuilder.Build
}
