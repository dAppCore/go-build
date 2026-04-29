package release

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated usage examples ---
func ExampleRunSDK() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	_, _ = RunSDK(ctx, &Config{}, true)
	core.Println("RunSDK")
	// Output: RunSDK
}
