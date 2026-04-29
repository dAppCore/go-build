package testassert

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated usage examples ---
func ExampleEqual() {
	_ = Equal("agent", "agent")
	core.Println("Equal")
	// Output: Equal
}

func ExampleNil() {
	_ = Nil("agent")
	core.Println("Nil")
	// Output: Nil
}

func ExampleEmpty() {
	_ = Empty("agent")
	core.Println("Empty")
	// Output: Empty
}

func ExampleZero() {
	_ = Zero("agent")
	core.Println("Zero")
	// Output: Zero
}

func ExampleContains() {
	_ = Contains("agent", "agent")
	core.Println("Contains")
	// Output: Contains
}

func ExampleElementsMatch() {
	_ = ElementsMatch("agent", "agent")
	core.Println("ElementsMatch")
	// Output: ElementsMatch
}
