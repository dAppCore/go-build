package service

import (
	core "dappco.re/go"
)

type EventEmitter = daemonEventEmitter

// --- v0.9.0 generated usage examples ---
func ExampleRun() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	_ = Run(ctx, Config{})
	core.Println("Run")
	// Output: Run
}

func ExampleEventEmitter_Emit() {
	subject := daemonEventEmitter{}
	subject.Emit("agent", "agent")
	core.Println("EventEmitter_Emit")
	// Output: EventEmitter_Emit
}
