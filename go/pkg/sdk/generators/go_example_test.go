package generators

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated usage examples ---
func ExampleNewGoGenerator() {
	_ = NewGoGenerator()
	core.Println("NewGoGenerator")
	// Output: NewGoGenerator
}

func ExampleGoGenerator_Language() {
	subject := &GoGenerator{}
	_ = subject.Language()
	core.Println("GoGenerator_Language")
	// Output: GoGenerator_Language
}

func ExampleGoGenerator_Available() {
	subject := &GoGenerator{}
	_ = subject.Available()
	core.Println("GoGenerator_Available")
	// Output: GoGenerator_Available
}

func ExampleGoGenerator_Install() {
	subject := &GoGenerator{}
	_ = subject.Install()
	core.Println("GoGenerator_Install")
	// Output: GoGenerator_Install
}

func ExampleGoGenerator_Generate() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &GoGenerator{}
	_ = subject.Generate(ctx, Options{})
	core.Println("GoGenerator_Generate")
	// Output: GoGenerator_Generate
}
