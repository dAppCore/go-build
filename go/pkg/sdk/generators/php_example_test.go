package generators

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated usage examples ---
func ExampleNewPHPGenerator() {
	_ = NewPHPGenerator()
	core.Println("NewPHPGenerator")
	// Output: NewPHPGenerator
}

func ExamplePHPGenerator_Language() {
	subject := &PHPGenerator{}
	_ = subject.Language()
	core.Println("PHPGenerator_Language")
	// Output: PHPGenerator_Language
}

func ExamplePHPGenerator_Available() {
	subject := &PHPGenerator{}
	_ = subject.Available()
	core.Println("PHPGenerator_Available")
	// Output: PHPGenerator_Available
}

func ExamplePHPGenerator_Install() {
	subject := &PHPGenerator{}
	_ = subject.Install()
	core.Println("PHPGenerator_Install")
	// Output: PHPGenerator_Install
}

func ExamplePHPGenerator_Generate() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &PHPGenerator{}
	_ = subject.Generate(ctx, Options{})
	core.Println("PHPGenerator_Generate")
	// Output: PHPGenerator_Generate
}
