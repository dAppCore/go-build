package release

import (
	core "github.com/dappcore/go"
)

// --- v0.9.0 generated usage examples ---
func ExampleRunSDK() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	_ = RunSDK(ctx, &Config{}, true)
	core.Println("RunSDK")
	// Output: RunSDK
}
