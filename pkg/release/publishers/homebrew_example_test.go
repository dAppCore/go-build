package publishers

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated usage examples ---
func ExampleNewHomebrewPublisher() {
	_ = NewHomebrewPublisher()
	core.Println("NewHomebrewPublisher")
	// Output: NewHomebrewPublisher
}

func ExampleHomebrewPublisher_Name() {
	subject := &HomebrewPublisher{}
	_ = subject.Name()
	core.Println("HomebrewPublisher_Name")
	// Output: HomebrewPublisher_Name
}

func ExampleHomebrewPublisher_Validate() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &HomebrewPublisher{}
	_ = subject.Validate(ctx, &Release{}, PublisherConfig{}, nil)
	core.Println("HomebrewPublisher_Validate")
	// Output: HomebrewPublisher_Validate
}

func ExampleHomebrewPublisher_Supports() {
	subject := &HomebrewPublisher{}
	_ = subject.Supports("linux")
	core.Println("HomebrewPublisher_Supports")
	// Output: HomebrewPublisher_Supports
}

func ExampleHomebrewPublisher_Publish() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &HomebrewPublisher{}
	_ = subject.Publish(ctx, &Release{}, PublisherConfig{}, nil, true)
	core.Println("HomebrewPublisher_Publish")
	// Output: HomebrewPublisher_Publish
}
