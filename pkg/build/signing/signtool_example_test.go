package signing

import core "dappco.re/go"

// ExampleNewWindowsSigner references NewWindowsSigner on this package API surface.
func ExampleNewWindowsSigner() {
	_ = NewWindowsSigner
	core.Println("NewWindowsSigner")
	// Output: NewWindowsSigner
}

// ExampleWindowsSigner_Name references WindowsSigner.Name on this package API surface.
func ExampleWindowsSigner_Name() {
	_ = (*WindowsSigner).Name
	core.Println("WindowsSigner.Name")
	// Output: WindowsSigner.Name
}

// ExampleWindowsSigner_Available references WindowsSigner.Available on this package API surface.
func ExampleWindowsSigner_Available() {
	_ = (*WindowsSigner).Available
	core.Println("WindowsSigner.Available")
	// Output: WindowsSigner.Available
}

// ExampleWindowsSigner_Sign references WindowsSigner.Sign on this package API surface.
func ExampleWindowsSigner_Sign() {
	_ = (*WindowsSigner).Sign
	core.Println("WindowsSigner.Sign")
	// Output: WindowsSigner.Sign
}
