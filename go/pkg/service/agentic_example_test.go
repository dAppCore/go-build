package service

import (
	core "dappco.re/go"
)

type Agentic = daemonAgentic

// --- v0.9.0 generated usage examples ---
func ExampleAgentic_Run() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := newDaemonAgentic(Config{}, nil, nil).(*daemonAgentic)
	subject.Run(ctx)
	core.Println("Agentic_Run")
	// Output: Agentic_Run
}

func ExampleAgentic_Notify() {
	subject := newDaemonAgentic(Config{}, nil, nil).(*daemonAgentic)
	subject.Notify("agent", "agent")
	core.Println("Agentic_Notify")
	// Output: Agentic_Notify
}
