package installers

import core "dappco.re/go"

// ExampleVariants references Variants on this package API surface.
func ExampleVariants() {
	_ = Variants
	core.Println("Variants")
	// Output: Variants
}

// ExampleOutputName references OutputName on this package API surface.
func ExampleOutputName() {
	_ = OutputName
	core.Println("OutputName")
	// Output: OutputName
}

// ExampleGenerateInstaller references GenerateInstaller on this package API surface.
func ExampleGenerateInstaller() {
	_ = GenerateInstaller
	core.Println("GenerateInstaller")
	// Output: GenerateInstaller
}

// ExampleGenerateAll references GenerateAll on this package API surface.
func ExampleGenerateAll() {
	_ = GenerateAll
	core.Println("GenerateAll")
	// Output: GenerateAll
}
