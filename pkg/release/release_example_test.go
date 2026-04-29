package release

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated usage examples ---
func ExamplePublish() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	_, _ = Publish(ctx, &Config{}, true)
	core.Println("Publish")
	// Output: Publish
}

func ExampleRun() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	_, _ = Run(ctx, &Config{}, true)
	core.Println("Run")
	// Output: Run
}
