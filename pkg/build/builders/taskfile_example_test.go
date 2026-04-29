package builders

import core "dappco.re/go"

// ExampleNewTaskfileBuilder references NewTaskfileBuilder on this package API surface.
func ExampleNewTaskfileBuilder() {
	_ = NewTaskfileBuilder
	core.Println("NewTaskfileBuilder")
	// Output: NewTaskfileBuilder
}

// ExampleTaskfileBuilder_Name references TaskfileBuilder.Name on this package API surface.
func ExampleTaskfileBuilder_Name() {
	_ = (*TaskfileBuilder).Name
	core.Println("TaskfileBuilder.Name")
	// Output: TaskfileBuilder.Name
}

// ExampleTaskfileBuilder_Detect references TaskfileBuilder.Detect on this package API surface.
func ExampleTaskfileBuilder_Detect() {
	_ = (*TaskfileBuilder).Detect
	core.Println("TaskfileBuilder.Detect")
	// Output: TaskfileBuilder.Detect
}

// ExampleTaskfileBuilder_Build references TaskfileBuilder.Build on this package API surface.
func ExampleTaskfileBuilder_Build() {
	_ = (*TaskfileBuilder).Build
	core.Println("TaskfileBuilder.Build")
	// Output: TaskfileBuilder.Build
}
