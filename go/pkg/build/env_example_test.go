package build

import core "dappco.re/go"

// ExampleBuildEnvironment references BuildEnvironment on this package API surface.
func ExampleBuildEnvironment() {
	_ = BuildEnvironment
	core.Println("BuildEnvironment")
	// Output: BuildEnvironment
}

// ExampleDenoRequested references DenoRequested on this package API surface.
func ExampleDenoRequested() {
	_ = DenoRequested
	core.Println("DenoRequested")
	// Output: DenoRequested
}

// ExampleNpmRequested references NpmRequested on this package API surface.
func ExampleNpmRequested() {
	_ = NpmRequested
	core.Println("NpmRequested")
	// Output: NpmRequested
}
