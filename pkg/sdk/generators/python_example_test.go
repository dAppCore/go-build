package generators

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated usage examples ---
func ExampleNewPythonGenerator() {
	_ = NewPythonGenerator()
	core.Println("NewPythonGenerator")
	// Output: NewPythonGenerator
}

func ExamplePythonGenerator_Language() {
	subject := &PythonGenerator{}
	_ = subject.Language()
	core.Println("PythonGenerator_Language")
	// Output: PythonGenerator_Language
}

func ExamplePythonGenerator_Available() {
	subject := &PythonGenerator{}
	_ = subject.Available()
	core.Println("PythonGenerator_Available")
	// Output: PythonGenerator_Available
}

func ExamplePythonGenerator_Install() {
	subject := &PythonGenerator{}
	_ = subject.Install()
	core.Println("PythonGenerator_Install")
	// Output: PythonGenerator_Install
}

func ExamplePythonGenerator_Generate() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &PythonGenerator{}
	_ = subject.Generate(ctx, Options{})
	core.Println("PythonGenerator_Generate")
	// Output: PythonGenerator_Generate
}
