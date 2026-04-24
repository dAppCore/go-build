package publishers

import (
	"context"
	"testing"

	"dappco.re/go/core/io"
)

func TestPublishers_PublishRejectsUnsafeVersion_Good(t *testing.T) {
	release := &Release{
		Version:    "v1.2.3;rm -rf /",
		ProjectDir: t.TempDir(),
		FS:         io.Local,
	}

	relCfg := &mockReleaseConfig{
		repository:  "owner/repo",
		projectName: "project",
	}

	tests := []struct {
		name      string
		publisher Publisher
	}{
		{name: "github", publisher: NewGitHubPublisher()},
		{name: "docker", publisher: NewDockerPublisher()},
		{name: "homebrew", publisher: NewHomebrewPublisher()},
		{name: "chocolatey", publisher: NewChocolateyPublisher()},
		{name: "aur", publisher: NewAURPublisher()},
		{name: "npm", publisher: NewNpmPublisher()},
		{name: "scoop", publisher: NewScoopPublisher()},
		{name: "linuxkit", publisher: NewLinuxKitPublisher()},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.publisher.Publish(context.Background(), release, PublisherConfig{Type: tc.name}, relCfg, true)
			if err == nil {
				t.Fatal("expected error")
			}
			if !stdlibAssertContains(err.Error(), "release version contains unsupported characters") {
				t.Fatalf("expected %v to contain %v", err.Error(), "release version contains unsupported characters")
			}

		})
	}
}
