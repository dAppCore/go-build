package builders

import core "dappco.re/go"

// ExampleNewNodeBuilder references NewNodeBuilder on this package API surface.
func ExampleNewNodeBuilder() {
	_ = NewNodeBuilder
	core.Println("NewNodeBuilder")
	// Output: NewNodeBuilder
}

// ExampleNodeBuilder_Name references NodeBuilder.Name on this package API surface.
func ExampleNodeBuilder_Name() {
	_ = (*NodeBuilder).Name
	core.Println("NodeBuilder.Name")
	// Output: NodeBuilder.Name
}

// ExampleNodeBuilder_Detect references NodeBuilder.Detect on this package API surface.
func ExampleNodeBuilder_Detect() {
	_ = (*NodeBuilder).Detect
	core.Println("NodeBuilder.Detect")
	// Output: NodeBuilder.Detect
}

// ExampleNodeBuilder_Build references NodeBuilder.Build on this package API surface.
func ExampleNodeBuilder_Build() {
	_ = (*NodeBuilder).Build
	core.Println("NodeBuilder.Build")
	// Output: NodeBuilder.Build
}
