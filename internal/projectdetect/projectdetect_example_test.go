package projectdetect

import (
	core "dappco.re/go"
	coreio "dappco.re/go/build/pkg/storage"
)

// --- v0.9.0 generated usage examples ---
func ExampleDetectProjectType() {
	_ = DetectProjectType(coreio.NewMemoryMedium(), core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("DetectProjectType")
	// Output: DetectProjectType
}
