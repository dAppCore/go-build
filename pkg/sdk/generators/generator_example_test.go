package generators

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated usage examples ---
func ExampleNewRegistry() {
	_ = NewRegistry()
	core.Println("NewRegistry")
	// Output: NewRegistry
}

func ExampleRegistry_Get() {
	subject := NewRegistry()
	_, _ = subject.Get("go")
	core.Println("Registry_Get")
	// Output: Registry_Get
}

func ExampleRegistry_Register() {
	subject := NewRegistry()
	subject.Register(NewGoGenerator())
	core.Println("Registry_Register")
	// Output: Registry_Register
}

func ExampleRegistry_Languages() {
	subject := NewRegistry()
	_ = subject.Languages()
	core.Println("Registry_Languages")
	// Output: Registry_Languages
}

func ExampleRegistry_LanguagesIter() {
	subject := NewRegistry()
	_ = subject.LanguagesIter()
	core.Println("Registry_LanguagesIter")
	// Output: Registry_LanguagesIter
}
