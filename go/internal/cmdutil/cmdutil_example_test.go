package cmdutil

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated usage examples ---
func ExampleContextOrBackground() {
	_ = ContextOrBackground()
	core.Println("ContextOrBackground")
	// Output: ContextOrBackground
}

func ExampleOptionString() {
	_ = OptionString(core.NewOptions())
	core.Println("OptionString")
	// Output: OptionString
}

func ExampleOptionBoolDefault() {
	_ = OptionBoolDefault(core.NewOptions(), true)
	core.Println("OptionBoolDefault")
	// Output: OptionBoolDefault
}

func ExampleOptionBool() {
	_ = OptionBool(core.NewOptions())
	core.Println("OptionBool")
	// Output: OptionBool
}

func ExampleOptionHas() {
	_ = OptionHas(core.NewOptions())
	core.Println("OptionHas")
	// Output: OptionHas
}

func ExampleResultFromError() {
	_ = ResultFromError(nil)
	core.Println("ResultFromError")
	// Output: ResultFromError
}
