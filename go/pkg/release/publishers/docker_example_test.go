package publishers

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated usage examples ---
func ExampleNewDockerPublisher() {
	_ = NewDockerPublisher()
	core.Println("NewDockerPublisher")
	// Output: NewDockerPublisher
}

func ExampleDockerPublisher_Name() {
	subject := &DockerPublisher{}
	_ = subject.Name()
	core.Println("DockerPublisher_Name")
	// Output: DockerPublisher_Name
}

func ExampleDockerPublisher_Validate() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &DockerPublisher{}
	_ = subject.Validate(ctx, &Release{}, PublisherConfig{}, nil)
	core.Println("DockerPublisher_Validate")
	// Output: DockerPublisher_Validate
}

func ExampleDockerPublisher_Supports() {
	subject := &DockerPublisher{}
	_ = subject.Supports("linux")
	core.Println("DockerPublisher_Supports")
	// Output: DockerPublisher_Supports
}

func ExampleDockerPublisher_Publish() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &DockerPublisher{}
	_ = subject.Publish(ctx, &Release{}, PublisherConfig{}, nil, true)
	core.Println("DockerPublisher_Publish")
	// Output: DockerPublisher_Publish
}
