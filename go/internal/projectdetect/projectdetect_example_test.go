package projectdetect

import (
	core "github.com/dappcore/go"
	coreio "github.com/dappcore/go-build/go/pkg/storage"
)

// --- v0.9.0 generated usage examples ---
func ExampleDetectProjectType() {
	_ = DetectProjectType(coreio.NewMemoryMedium(), core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("DetectProjectType")
	// Output: DetectProjectType
}
