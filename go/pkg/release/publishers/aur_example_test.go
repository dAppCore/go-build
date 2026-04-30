package publishers

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated usage examples ---
func ExampleNewAURPublisher() {
	_ = NewAURPublisher()
	core.Println("NewAURPublisher")
	// Output: NewAURPublisher
}

func ExampleAURPublisher_Name() {
	subject := &AURPublisher{}
	_ = subject.Name()
	core.Println("AURPublisher_Name")
	// Output: AURPublisher_Name
}

func ExampleAURPublisher_Validate() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &AURPublisher{}
	_ = subject.Validate(ctx, &Release{}, PublisherConfig{}, nil)
	core.Println("AURPublisher_Validate")
	// Output: AURPublisher_Validate
}

func ExampleAURPublisher_Supports() {
	subject := &AURPublisher{}
	_ = subject.Supports("linux")
	core.Println("AURPublisher_Supports")
	// Output: AURPublisher_Supports
}

func ExampleAURPublisher_Publish() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &AURPublisher{}
	_ = subject.Publish(ctx, &Release{}, PublisherConfig{}, nil, true)
	core.Println("AURPublisher_Publish")
	// Output: AURPublisher_Publish
}
