package sdkcfg

import (
	core "github.com/dappcore/go"
	coreio "github.com/dappcore/go-build/go/pkg/storage"
)

// --- v0.9.0 generated usage examples ---
func ExampleLoadProjectConfig() {
	_ = LoadProjectConfig(coreio.NewMemoryMedium(), core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("LoadProjectConfig")
	// Output: LoadProjectConfig
}
