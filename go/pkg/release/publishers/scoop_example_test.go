package publishers

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated usage examples ---
func ExampleNewScoopPublisher() {
	_ = NewScoopPublisher()
	core.Println("NewScoopPublisher")
	// Output: NewScoopPublisher
}

func ExampleScoopPublisher_Name() {
	subject := &ScoopPublisher{}
	_ = subject.Name()
	core.Println("ScoopPublisher_Name")
	// Output: ScoopPublisher_Name
}

func ExampleScoopPublisher_Validate() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &ScoopPublisher{}
	_ = subject.Validate(ctx, &Release{}, PublisherConfig{}, nil)
	core.Println("ScoopPublisher_Validate")
	// Output: ScoopPublisher_Validate
}

func ExampleScoopPublisher_Supports() {
	subject := &ScoopPublisher{}
	_ = subject.Supports("linux")
	core.Println("ScoopPublisher_Supports")
	// Output: ScoopPublisher_Supports
}

func ExampleScoopPublisher_Publish() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &ScoopPublisher{}
	_ = subject.Publish(ctx, &Release{}, PublisherConfig{}, nil, true)
	core.Println("ScoopPublisher_Publish")
	// Output: ScoopPublisher_Publish
}
