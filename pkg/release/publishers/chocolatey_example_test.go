package publishers

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated usage examples ---
func ExampleNewChocolateyPublisher() {
	_ = NewChocolateyPublisher()
	core.Println("NewChocolateyPublisher")
	// Output: NewChocolateyPublisher
}

func ExampleChocolateyPublisher_Name() {
	subject := &ChocolateyPublisher{}
	_ = subject.Name()
	core.Println("ChocolateyPublisher_Name")
	// Output: ChocolateyPublisher_Name
}

func ExampleChocolateyPublisher_Validate() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &ChocolateyPublisher{}
	_ = subject.Validate(ctx, &Release{}, PublisherConfig{}, nil)
	core.Println("ChocolateyPublisher_Validate")
	// Output: ChocolateyPublisher_Validate
}

func ExampleChocolateyPublisher_Supports() {
	subject := &ChocolateyPublisher{}
	_ = subject.Supports("linux")
	core.Println("ChocolateyPublisher_Supports")
	// Output: ChocolateyPublisher_Supports
}

func ExampleChocolateyPublisher_Publish() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &ChocolateyPublisher{}
	_ = subject.Publish(ctx, &Release{}, PublisherConfig{}, nil, true)
	core.Println("ChocolateyPublisher_Publish")
	// Output: ChocolateyPublisher_Publish
}
