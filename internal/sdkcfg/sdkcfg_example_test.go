package sdkcfg

import (
	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

// --- v0.9.0 generated usage examples ---
func ExampleLoadProjectConfig() {
	_ = LoadProjectConfig(coreio.NewMemoryMedium(), core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("LoadProjectConfig")
	// Output: LoadProjectConfig
}
