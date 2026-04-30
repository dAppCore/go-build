package builders

import core "dappco.re/go"

// ExampleNewDockerBuilder references NewDockerBuilder on this package API surface.
func ExampleNewDockerBuilder() {
	_ = NewDockerBuilder
	core.Println("NewDockerBuilder")
	// Output: NewDockerBuilder
}

// ExampleDockerBuilder_Name references DockerBuilder.Name on this package API surface.
func ExampleDockerBuilder_Name() {
	_ = (*DockerBuilder).Name
	core.Println("DockerBuilder.Name")
	// Output: DockerBuilder.Name
}

// ExampleDockerBuilder_Detect references DockerBuilder.Detect on this package API surface.
func ExampleDockerBuilder_Detect() {
	_ = (*DockerBuilder).Detect
	core.Println("DockerBuilder.Detect")
	// Output: DockerBuilder.Detect
}

// ExampleDockerBuilder_Build references DockerBuilder.Build on this package API surface.
func ExampleDockerBuilder_Build() {
	_ = (*DockerBuilder).Build
	core.Println("DockerBuilder.Build")
	// Output: DockerBuilder.Build
}
