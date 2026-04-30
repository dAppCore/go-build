package signing

import core "dappco.re/go"

// ExampleSignBinaries references SignBinaries on this package API surface.
func ExampleSignBinaries() {
	_ = SignBinaries
	core.Println("SignBinaries")
	// Output: SignBinaries
}

// ExampleNotarizeBinaries references NotarizeBinaries on this package API surface.
func ExampleNotarizeBinaries() {
	_ = NotarizeBinaries
	core.Println("NotarizeBinaries")
	// Output: NotarizeBinaries
}

// ExampleSignChecksums references SignChecksums on this package API surface.
func ExampleSignChecksums() {
	_ = SignChecksums
	core.Println("SignChecksums")
	// Output: SignChecksums
}
