package build

import core "dappco.re/go"

// ExampleChecksum references Checksum on this package API surface.
func ExampleChecksum() {
	_ = Checksum
	core.Println("Checksum")
	// Output: Checksum
}

// ExampleChecksumAll references ChecksumAll on this package API surface.
func ExampleChecksumAll() {
	_ = ChecksumAll
	core.Println("ChecksumAll")
	// Output: ChecksumAll
}

// ExampleWriteChecksumFile references WriteChecksumFile on this package API surface.
func ExampleWriteChecksumFile() {
	_ = WriteChecksumFile
	core.Println("WriteChecksumFile")
	// Output: WriteChecksumFile
}
