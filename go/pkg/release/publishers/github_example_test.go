package publishers

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated usage examples ---
func ExampleNewGitHubPublisher() {
	_ = NewGitHubPublisher()
	core.Println("NewGitHubPublisher")
	// Output: NewGitHubPublisher
}

func ExampleDetectGitHubRepository() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	_ = DetectGitHubRepository(ctx, core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("DetectGitHubRepository")
	// Output: DetectGitHubRepository
}

func ExampleGitHubPublisher_Name() {
	subject := &GitHubPublisher{}
	_ = subject.Name()
	core.Println("GitHubPublisher_Name")
	// Output: GitHubPublisher_Name
}

func ExampleGitHubPublisher_Validate() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &GitHubPublisher{}
	_ = subject.Validate(ctx, &Release{}, PublisherConfig{}, nil)
	core.Println("GitHubPublisher_Validate")
	// Output: GitHubPublisher_Validate
}

func ExampleGitHubPublisher_Supports() {
	subject := &GitHubPublisher{}
	_ = subject.Supports("linux")
	core.Println("GitHubPublisher_Supports")
	// Output: GitHubPublisher_Supports
}

func ExampleGitHubPublisher_Publish() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &GitHubPublisher{}
	_ = subject.Publish(ctx, &Release{}, PublisherConfig{}, nil, true)
	core.Println("GitHubPublisher_Publish")
	// Output: GitHubPublisher_Publish
}

func ExampleUploadArtifact() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	_ = UploadArtifact(ctx, "owner/repo", "v1.2.3", core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("UploadArtifact")
	// Output: UploadArtifact
}

func ExampleDeleteRelease() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	_ = DeleteRelease(ctx, "owner/repo", "v1.2.3")
	core.Println("DeleteRelease")
	// Output: DeleteRelease
}

func ExampleReleaseExists() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	_ = ReleaseExists(ctx, "owner/repo", "v1.2.3")
	core.Println("ReleaseExists")
	// Output: ReleaseExists
}
