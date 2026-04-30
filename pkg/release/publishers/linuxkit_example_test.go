package publishers

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated usage examples ---
func ExampleNewLinuxKitPublisher() {
	_ = NewLinuxKitPublisher()
	core.Println("NewLinuxKitPublisher")
	// Output: NewLinuxKitPublisher
}

func ExampleLinuxKitPublisher_Name() {
	subject := &LinuxKitPublisher{}
	_ = subject.Name()
	core.Println("LinuxKitPublisher_Name")
	// Output: LinuxKitPublisher_Name
}

func ExampleLinuxKitPublisher_Validate() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &LinuxKitPublisher{}
	_ = subject.Validate(ctx, &Release{}, PublisherConfig{}, nil)
	core.Println("LinuxKitPublisher_Validate")
	// Output: LinuxKitPublisher_Validate
}

func ExampleLinuxKitPublisher_Supports() {
	subject := &LinuxKitPublisher{}
	_ = subject.Supports("linux")
	core.Println("LinuxKitPublisher_Supports")
	// Output: LinuxKitPublisher_Supports
}

func ExampleLinuxKitPublisher_Publish() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &LinuxKitPublisher{}
	_ = subject.Publish(ctx, &Release{}, PublisherConfig{}, nil, true)
	core.Println("LinuxKitPublisher_Publish")
	// Output: LinuxKitPublisher_Publish
}
