package builders

import core "dappco.re/go"

// ExampleNewPythonBuilder references NewPythonBuilder on this package API surface.
func ExampleNewPythonBuilder() {
	_ = NewPythonBuilder
	core.Println("NewPythonBuilder")
	// Output: NewPythonBuilder
}

// ExamplePythonBuilder_Name references PythonBuilder.Name on this package API surface.
func ExamplePythonBuilder_Name() {
	_ = (*PythonBuilder).Name
	core.Println("PythonBuilder.Name")
	// Output: PythonBuilder.Name
}

// ExamplePythonBuilder_Detect references PythonBuilder.Detect on this package API surface.
func ExamplePythonBuilder_Detect() {
	_ = (*PythonBuilder).Detect
	core.Println("PythonBuilder.Detect")
	// Output: PythonBuilder.Detect
}

// ExamplePythonBuilder_Build references PythonBuilder.Build on this package API surface.
func ExamplePythonBuilder_Build() {
	_ = (*PythonBuilder).Build
	core.Println("PythonBuilder.Build")
	// Output: PythonBuilder.Build
}
