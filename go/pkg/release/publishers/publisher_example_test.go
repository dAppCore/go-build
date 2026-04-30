package publishers

import (
	core "dappco.re/go"
	coreio "dappco.re/go/build/pkg/storage"
)

// --- v0.9.0 generated usage examples ---
func ExampleNewRelease() {
	_ = NewRelease("v1.2.3", nil, "agent", core.Path(core.TempDir(), "go-build-compliance"), coreio.NewMemoryMedium())
	core.Println("NewRelease")
	// Output: NewRelease
}

func ExampleNewReleaseWithArtifactFS() {
	_ = NewReleaseWithArtifactFS("v1.2.3", nil, "agent", core.Path(core.TempDir(), "go-build-compliance"), coreio.NewMemoryMedium(), coreio.NewMemoryMedium())
	core.Println("NewReleaseWithArtifactFS")
	// Output: NewReleaseWithArtifactFS
}

func ExampleNewPublisherConfig() {
	_ = NewPublisherConfig("agent", true, true, "agent")
	core.Println("NewPublisherConfig")
	// Output: NewPublisherConfig
}
