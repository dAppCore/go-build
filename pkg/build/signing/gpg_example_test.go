package signing

import core "dappco.re/go"

// ExampleNewGPGSigner references NewGPGSigner on this package API surface.
func ExampleNewGPGSigner() {
	_ = NewGPGSigner
	core.Println("NewGPGSigner")
	// Output: NewGPGSigner
}

// ExampleGPGSigner_Name references GPGSigner.Name on this package API surface.
func ExampleGPGSigner_Name() {
	_ = (*GPGSigner).Name
	core.Println("GPGSigner.Name")
	// Output: GPGSigner.Name
}

// ExampleGPGSigner_Available references GPGSigner.Available on this package API surface.
func ExampleGPGSigner_Available() {
	_ = (*GPGSigner).Available
	core.Println("GPGSigner.Available")
	// Output: GPGSigner.Available
}

// ExampleGPGSigner_Sign references GPGSigner.Sign on this package API surface.
func ExampleGPGSigner_Sign() {
	_ = (*GPGSigner).Sign
	core.Println("GPGSigner.Sign")
	// Output: GPGSigner.Sign
}
