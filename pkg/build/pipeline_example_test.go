package build

import core "dappco.re/go"

// ExamplePipeline_Plan references Pipeline.Plan on this package API surface.
func ExamplePipeline_Plan() {
	_ = (*Pipeline).Plan
	core.Println("Pipeline.Plan")
	// Output: Pipeline.Plan
}

// ExamplePipeline_Run references Pipeline.Run on this package API surface.
func ExamplePipeline_Run() {
	_ = (*Pipeline).Run
	core.Println("Pipeline.Run")
	// Output: Pipeline.Run
}

// ExampleResolveBuildName references ResolveBuildName on this package API surface.
func ExampleResolveBuildName() {
	_ = ResolveBuildName
	core.Println("ResolveBuildName")
	// Output: ResolveBuildName
}
