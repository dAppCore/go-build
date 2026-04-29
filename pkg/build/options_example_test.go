package build

import core "dappco.re/go"

// ExampleComputeOptions references ComputeOptions on this package API surface.
func ExampleComputeOptions() {
	_ = ComputeOptions
	core.Println("ComputeOptions")
	// Output: ComputeOptions
}

// ExampleApplyOptions references ApplyOptions on this package API surface.
func ExampleApplyOptions() {
	_ = ApplyOptions
	core.Println("ApplyOptions")
	// Output: ApplyOptions
}

// ExampleInjectWebKitTag references InjectWebKitTag on this package API surface.
func ExampleInjectWebKitTag() {
	_ = InjectWebKitTag
	core.Println("InjectWebKitTag")
	// Output: InjectWebKitTag
}

// ExampleBuildOptions_String references BuildOptions.String on this package API surface.
func ExampleBuildOptions_String() {
	_ = (*BuildOptions).String
	core.Println("BuildOptions.String")
	// Output: BuildOptions.String
}
