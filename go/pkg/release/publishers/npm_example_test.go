package publishers

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated usage examples ---
func ExampleNewNpmPublisher() {
	_ = NewNpmPublisher()
	core.Println("NewNpmPublisher")
	// Output: NewNpmPublisher
}

func ExampleNpmPublisher_Name() {
	subject := &NpmPublisher{}
	_ = subject.Name()
	core.Println("NpmPublisher_Name")
	// Output: NpmPublisher_Name
}

func ExampleNpmPublisher_Validate() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &NpmPublisher{}
	_ = subject.Validate(ctx, &Release{}, PublisherConfig{}, nil)
	core.Println("NpmPublisher_Validate")
	// Output: NpmPublisher_Validate
}

func ExampleNpmPublisher_Supports() {
	subject := &NpmPublisher{}
	_ = subject.Supports("linux")
	core.Println("NpmPublisher_Supports")
	// Output: NpmPublisher_Supports
}

func ExampleNpmPublisher_Publish() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &NpmPublisher{}
	_ = subject.Publish(ctx, &Release{}, PublisherConfig{}, nil, true)
	core.Println("NpmPublisher_Publish")
	// Output: NpmPublisher_Publish
}
