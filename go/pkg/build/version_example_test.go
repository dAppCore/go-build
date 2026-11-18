package build

import core "github.com/dappcore/go"

// ExampleValidateVersionString references ValidateVersionString on this package API surface.
func ExampleValidateVersionString() {
	_ = ValidateVersionString
	core.Println("ValidateVersionString")
	// Output: ValidateVersionString
}

// ExampleValidateVersionIdentifier references ValidateVersionIdentifier on this package API surface.
func ExampleValidateVersionIdentifier() {
	_ = ValidateVersionIdentifier
	core.Println("ValidateVersionIdentifier")
	// Output: ValidateVersionIdentifier
}
