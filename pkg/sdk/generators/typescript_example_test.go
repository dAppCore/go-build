package generators

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated usage examples ---
func ExampleNewTypeScriptGenerator() {
	_ = NewTypeScriptGenerator()
	core.Println("NewTypeScriptGenerator")
	// Output: NewTypeScriptGenerator
}

func ExampleTypeScriptGenerator_Language() {
	subject := &TypeScriptGenerator{}
	_ = subject.Language()
	core.Println("TypeScriptGenerator_Language")
	// Output: TypeScriptGenerator_Language
}

func ExampleTypeScriptGenerator_Available() {
	subject := &TypeScriptGenerator{}
	_ = subject.Available()
	core.Println("TypeScriptGenerator_Available")
	// Output: TypeScriptGenerator_Available
}

func ExampleTypeScriptGenerator_Install() {
	subject := &TypeScriptGenerator{}
	_ = subject.Install()
	core.Println("TypeScriptGenerator_Install")
	// Output: TypeScriptGenerator_Install
}

func ExampleTypeScriptGenerator_Generate() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &TypeScriptGenerator{}
	_ = subject.Generate(ctx, Options{})
	core.Println("TypeScriptGenerator_Generate")
	// Output: TypeScriptGenerator_Generate
}
