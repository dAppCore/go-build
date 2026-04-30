package build

import core "dappco.re/go"

// ExampleVersionLinkerFlag references VersionLinkerFlag on this package API surface.
func ExampleVersionLinkerFlag() {
	_ = VersionLinkerFlag
	core.Println("VersionLinkerFlag")
	// Output: VersionLinkerFlag
}
