package signing

import core "dappco.re/go"

// ExampleNewMacOSSigner references NewMacOSSigner on this package API surface.
func ExampleNewMacOSSigner() {
	_ = NewMacOSSigner
	core.Println("NewMacOSSigner")
	// Output: NewMacOSSigner
}

// ExampleMacOSSigner_Name references MacOSSigner.Name on this package API surface.
func ExampleMacOSSigner_Name() {
	_ = (*MacOSSigner).Name
	core.Println("MacOSSigner.Name")
	// Output: MacOSSigner.Name
}

// ExampleMacOSSigner_Available references MacOSSigner.Available on this package API surface.
func ExampleMacOSSigner_Available() {
	_ = (*MacOSSigner).Available
	core.Println("MacOSSigner.Available")
	// Output: MacOSSigner.Available
}

// ExampleMacOSSigner_Sign references MacOSSigner.Sign on this package API surface.
func ExampleMacOSSigner_Sign() {
	_ = (*MacOSSigner).Sign
	core.Println("MacOSSigner.Sign")
	// Output: MacOSSigner.Sign
}

// ExampleMacOSSigner_Notarize references MacOSSigner.Notarize on this package API surface.
func ExampleMacOSSigner_Notarize() {
	_ = (*MacOSSigner).Notarize
	core.Println("MacOSSigner.Notarize")
	// Output: MacOSSigner.Notarize
}

// ExampleMacOSSigner_ShouldNotarize references MacOSSigner.ShouldNotarize on this package API surface.
func ExampleMacOSSigner_ShouldNotarize() {
	_ = (*MacOSSigner).ShouldNotarize
	core.Println("MacOSSigner.ShouldNotarize")
	// Output: MacOSSigner.ShouldNotarize
}
