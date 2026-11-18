package sdk

import (
	core "github.com/dappcore/go"
)

// --- v0.9.0 generated usage examples ---
func ExampleSDK_ValidateSpec() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &SDK{}
	_ = subject.ValidateSpec(ctx)
	core.Println("SDK_ValidateSpec")
	// Output: SDK_ValidateSpec
}
