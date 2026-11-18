package build

import core "github.com/dappcore/go"

// ExampleVersionLinkerFlag references VersionLinkerFlag on this package API surface.
func ExampleVersionLinkerFlag() {
	_ = VersionLinkerFlag
	core.Println("VersionLinkerFlag")
	// Output: VersionLinkerFlag
}
