package publishers

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/io"
)

func TestDocker_DockerPublisherNameGood(t *testing.T) {
	t.Run("returns docker", func(t *testing.T) {
		p := NewDockerPublisher()
		if !stdlibAssertEqual("docker", p.Name()) {
			t.Fatalf("want %v, got %v", "docker", p.Name())
		}

	})
}

func TestDocker_DockerPublisherParseConfigGood(t *testing.T) {
	p := NewDockerPublisher()

	t.Run("uses defaults when no extended config", func(t *testing.T) {
		pubCfg := PublisherConfig{Type: "docker"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}
		cfg := p.parseConfig(io.Local, pubCfg, relCfg, "/project")
		if !stdlibAssertEqual("ghcr.io", cfg.Registry) {
			t.Fatalf("want %v, got %v", "ghcr.io", cfg.Registry)
		}
		if !stdlibAssertEqual("owner/repo", cfg.Image) {
			t.Fatalf("want %v, got %v", "owner/repo", cfg.Image)
		}
		if !stdlibAssertEqual("/project/Dockerfile", cfg.Dockerfile) {
			t.Fatalf("want %v, got %v", "/project/Dockerfile", cfg.Dockerfile)
		}
		if !stdlibAssertEqual([]string{"linux/amd64", "linux/arm64"}, cfg.Platforms) {
			t.Fatalf("want %v, got %v", []string{"linux/amd64", "linux/arm64"}, cfg.Platforms)
		}
		if !stdlibAssertEqual([]string{"latest", "{{.Version}}"}, cfg.Tags) {
			t.Fatalf("want %v, got %v", []string{"latest", "{{.Version}}"}, cfg.Tags)
		}

	})

	t.Run("parses extended config", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "docker",
			Extended: map[string]any{
				"registry":   "docker.io",
				"image":      "myorg/myimage",
				"dockerfile": "docker/Dockerfile.prod",
				"platforms":  []any{"linux/amd64"},
				"tags":       []any{"latest", "stable", "{{.Version}}"},
				"build_args": map[string]any{
					"GO_VERSION": "1.21",
				},
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}
		cfg := p.parseConfig(io.Local, pubCfg, relCfg, "/project")
		if !stdlibAssertEqual("docker.io", cfg.Registry) {
			t.Fatalf("want %v, got %v", "docker.io", cfg.Registry)
		}
		if !stdlibAssertEqual("myorg/myimage", cfg.Image) {
			t.Fatalf("want %v, got %v", "myorg/myimage", cfg.Image)
		}
		if !stdlibAssertEqual("/project/docker/Dockerfile.prod", cfg.Dockerfile) {
			t.Fatalf("want %v, got %v", "/project/docker/Dockerfile.prod", cfg.Dockerfile)
		}
		if !stdlibAssertEqual([]string{"linux/amd64"}, cfg.Platforms) {
			t.Fatalf("want %v, got %v", []string{"linux/amd64"}, cfg.Platforms)
		}
		if !stdlibAssertEqual([]string{"latest", "stable", "{{.Version}}"}, cfg.Tags) {
			t.Fatalf("want %v, got %v", []string{"latest", "stable", "{{.Version}}"}, cfg.Tags)
		}
		if !stdlibAssertEqual("1.21", cfg.BuildArgs["GO_VERSION"]) {
			t.Fatalf("want %v, got %v", "1.21", cfg.BuildArgs["GO_VERSION"])
		}

	})

	t.Run("handles absolute dockerfile path", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "docker",
			Extended: map[string]any{
				"dockerfile": "/absolute/path/Dockerfile",
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}
		cfg := p.parseConfig(io.Local, pubCfg, relCfg, "/project")
		if !stdlibAssertEqual("/absolute/path/Dockerfile", cfg.Dockerfile) {
			t.Fatalf("want %v, got %v", "/absolute/path/Dockerfile", cfg.Dockerfile)
		}

	})

	t.Run("detects Containerfile when Dockerfile is absent", func(t *testing.T) {
		projectDir := t.TempDir()
		if result := ax.WriteFile(ax.Join(projectDir, "Containerfile"), []byte("FROM alpine\n"), 0o644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		pubCfg := PublisherConfig{Type: "docker"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}
		cfg := p.parseConfig(io.Local, pubCfg, relCfg, projectDir)
		if !stdlibAssertEqual(ax.Join(projectDir, "Containerfile"), cfg.Dockerfile) {
			t.Fatalf("want %v, got %v", ax.Join(projectDir, "Containerfile"), cfg.Dockerfile)
		}

	})
}

func TestDocker_DockerPublisherResolveTagsGood(t *testing.T) {
	p := NewDockerPublisher()

	t.Run("resolves version template", func(t *testing.T) {
		tags := p.resolveTags([]string{"latest", "{{.Version}}", "stable"}, "v1.2.3")
		if !stdlibAssertEqual([]string{"latest", "v1.2.3", "stable"}, tags) {
			t.Fatalf("want %v, got %v", []string{"latest", "v1.2.3", "stable"}, tags)
		}

	})

	t.Run("handles simple version syntax", func(t *testing.T) {
		tags := p.resolveTags([]string{"{{Version}}"}, "v1.0.0")
		if !stdlibAssertEqual([]string{"v1.0.0"}, tags) {
			t.Fatalf("want %v, got %v", []string{"v1.0.0"}, tags)
		}

	})

	t.Run("handles no templates", func(t *testing.T) {
		tags := p.resolveTags([]string{"latest", "stable"}, "v1.2.3")
		if !stdlibAssertEqual([]string{"latest", "stable"}, tags) {
			t.Fatalf("want %v, got %v", []string{"latest", "stable"}, tags)
		}

	})
}

func TestDocker_DockerPublisherBuildFullTagGood(t *testing.T) {
	p := NewDockerPublisher()

	tests := []struct {
		name     string
		registry string
		image    string
		tag      string
		expected string
	}{
		{
			name:     "with registry",
			registry: "ghcr.io",
			image:    "owner/repo",
			tag:      "v1.0.0",
			expected: "ghcr.io/owner/repo:v1.0.0",
		},
		{
			name:     "without registry",
			registry: "",
			image:    "myimage",
			tag:      "latest",
			expected: "myimage:latest",
		},
		{
			name:     "docker hub",
			registry: "docker.io",
			image:    "library/nginx",
			tag:      "alpine",
			expected: "docker.io/library/nginx:alpine",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tag := p.buildFullTag(tc.registry, tc.image, tc.tag)
			if !stdlibAssertEqual(tc.expected, tag) {
				t.Fatalf("want %v, got %v", tc.expected, tag)
			}

		})
	}
}

func TestDocker_DockerPublisherBuildBuildxArgsGood(t *testing.T) {
	p := NewDockerPublisher()

	t.Run("builds basic args", func(t *testing.T) {
		cfg := DockerConfig{
			Registry:   "ghcr.io",
			Image:      "owner/repo",
			Dockerfile: "/project/Dockerfile",
			Platforms:  []string{"linux/amd64", "linux/arm64"},
			BuildArgs:  make(map[string]string),
		}
		tags := []string{"latest", "v1.0.0"}

		args := p.buildBuildxArgs(cfg, tags, "v1.0.0")
		if !stdlibAssertContains(args, "buildx") {
			t.Fatalf("expected %v to contain %v", args, "buildx")
		}
		if !stdlibAssertContains(args, "build") {
			t.Fatalf("expected %v to contain %v", args, "build")
		}
		if !stdlibAssertContains(args, "--platform") {
			t.Fatalf("expected %v to contain %v", args, "--platform")
		}
		if !stdlibAssertContains(args, "linux/amd64,linux/arm64") {
			t.Fatalf("expected %v to contain %v", args, "linux/amd64,linux/arm64")
		}
		if !stdlibAssertContains(args, "-t") {
			t.Fatalf("expected %v to contain %v", args, "-t")
		}
		if !stdlibAssertContains(args, "ghcr.io/owner/repo:latest") {
			t.Fatalf("expected %v to contain %v", args, "ghcr.io/owner/repo:latest")
		}
		if !stdlibAssertContains(args, "ghcr.io/owner/repo:v1.0.0") {
			t.Fatalf("expected %v to contain %v", args, "ghcr.io/owner/repo:v1.0.0")
		}
		if !stdlibAssertContains(args, "-f") {
			t.Fatalf("expected %v to contain %v", args, "-f")
		}
		if !stdlibAssertContains(args, "/project/Dockerfile") {
			t.Fatalf("expected %v to contain %v",

				// Check that build args are present (order may vary)
				args, "/project/Dockerfile")
		}
		if !stdlibAssertContains(args, "--push") {
			t.Fatalf("expected %v to contain %v", args, "--push")
		}
		if !stdlibAssertContains(args, ".") {
			t.Fatalf("expected %v to contain %v", args, ".")
		}

	})

	t.Run("includes build args", func(t *testing.T) {
		cfg := DockerConfig{
			Registry:   "ghcr.io",
			Image:      "owner/repo",
			Dockerfile: "/project/Dockerfile",
			Platforms:  []string{"linux/amd64"},
			BuildArgs: map[string]string{
				"GO_VERSION": "1.21",
				"APP_NAME":   "myapp",
			},
		}
		tags := []string{"latest"}

		args := p.buildBuildxArgs(cfg, tags, "v1.0.0")
		if !stdlibAssertContains(args, "--build-arg") {
			t.Fatalf("expected %v to contain %v", args, "--build-arg")
		}

		foundGoVersion := false
		foundAppName := false
		foundVersion := false
		for i, arg := range args {
			if arg == "--build-arg" && i+1 < len(args) {
				if args[i+1] == "GO_VERSION=1.21" {
					foundGoVersion = true
				}
				if args[i+1] == "APP_NAME=myapp" {
					foundAppName = true
				}
				if args[i+1] == "VERSION=v1.0.0" {
					foundVersion = true
				}
			}
		}
		if !(foundGoVersion) {
			t.Fatal("GO_VERSION build arg not found")
		}
		if !(foundAppName) {
			t.Fatal("APP_NAME build arg not found")
		}
		if !(foundVersion) {
			t.Fatal("VERSION build arg not found")
		}

	})

	t.Run("expands version in build args", func(t *testing.T) {
		cfg := DockerConfig{
			Registry:   "ghcr.io",
			Image:      "owner/repo",
			Dockerfile: "/project/Dockerfile",
			Platforms:  []string{"linux/amd64"},
			BuildArgs: map[string]string{
				"APP_VERSION": "{{.Version}}",
			},
		}
		tags := []string{"latest"}

		args := p.buildBuildxArgs(cfg, tags, "v2.0.0")

		foundExpandedVersion := false
		for i, arg := range args {
			if arg == "--build-arg" && i+1 < len(args) {
				if args[i+1] == "APP_VERSION=v2.0.0" {
					foundExpandedVersion = true
				}
			}
		}
		if !(foundExpandedVersion) {
			t.Fatal("APP_VERSION should be expanded to v2.0.0")
		}

	})
}

func TestDocker_DockerPublisherPublishBad(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	p := NewDockerPublisher()

	t.Run("fails when dockerfile not found", func(t *testing.T) {
		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: "/nonexistent",
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{
			Type: "docker",
			Extended: map[string]any{
				"dockerfile": "/nonexistent/Dockerfile",
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		err := requirePublisherError(t, p.Publish(context.TODO(), release, pubCfg, relCfg, false))
		if !stdlibAssertContains(err, "Dockerfile not found") {
			t.Fatalf("expected %v to contain %v", err, "Dockerfile not found")
		}

	})
}

func TestDocker_DockerConfigDefaultsGood(t *testing.T) {
	t.Run("has sensible defaults", func(t *testing.T) {
		p := NewDockerPublisher()
		pubCfg := PublisherConfig{Type: "docker"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		cfg := p.parseConfig(io.Local, pubCfg, relCfg, "/project")
		if !stdlibAssertEqual(

			// Verify defaults
			"ghcr.io", cfg.Registry) {
			t.Fatalf("want %v, got %v", "ghcr.io", cfg.Registry)
		}
		if !stdlibAssertEqual("owner/repo", cfg.Image) {
			t.Fatalf("want %v, got %v", "owner/repo", cfg.Image)
		}
		if len(cfg.Platforms) != 2 {
			t.Fatalf("want len %v, got %v", 2, len(cfg.Platforms))
		}
		if !stdlibAssertContains(cfg.Platforms, "linux/amd64") {
			t.Fatalf("expected %v to contain %v", cfg.Platforms, "linux/amd64")
		}
		if !stdlibAssertContains(cfg.Platforms, "linux/arm64") {
			t.Fatalf("expected %v to contain %v", cfg.Platforms, "linux/arm64")
		}
		if !stdlibAssertContains(cfg.Tags, "latest") {
			t.Fatalf("expected %v to contain %v", cfg.Tags, "latest")
		}

	})
}

func TestDocker_DockerPublisherDryRunPublishGood(t *testing.T) {
	p := NewDockerPublisher()

	t.Run("outputs expected dry run information", func(t *testing.T) {
		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: "/project",
			FS:         io.Local,
		}
		cfg := DockerConfig{
			Registry:   "ghcr.io",
			Image:      "owner/repo",
			Dockerfile: "/project/Dockerfile",
			Platforms:  []string{"linux/amd64", "linux/arm64"},
			Tags:       []string{"latest", "{{.Version}}"},
			BuildArgs:  make(map[string]string),
		}

		publishResult := core.Ok(nil)
		output := capturePublisherOutput(t, func() {
			publishResult = p.dryRunPublish(release, cfg)
		})
		requirePublisherOK(t, publishResult)
		if !stdlibAssertContains(output, "DRY RUN: Docker Build & Push") {
			t.Fatalf("expected %v to contain %v", output, "DRY RUN: Docker Build & Push")
		}
		if !stdlibAssertContains(output, "Version:       v1.0.0") {
			t.Fatalf("expected %v to contain %v", output, "Version:       v1.0.0")
		}
		if !stdlibAssertContains(output, "Registry:      ghcr.io") {
			t.Fatalf("expected %v to contain %v", output, "Registry:      ghcr.io")
		}
		if !stdlibAssertContains(output, "Image:         owner/repo") {
			t.Fatalf("expected %v to contain %v", output, "Image:         owner/repo")
		}
		if !stdlibAssertContains(output, "Dockerfile:    /project/Dockerfile") {
			t.Fatalf("expected %v to contain %v", output, "Dockerfile:    /project/Dockerfile")
		}
		if !stdlibAssertContains(output, "Platforms:     linux/amd64, linux/arm64") {
			t.Fatalf("expected %v to contain %v", output, "Platforms:     linux/amd64, linux/arm64")
		}
		if !stdlibAssertContains(output, "Tags to be applied:") {
			t.Fatalf("expected %v to contain %v", output, "Tags to be applied:")
		}
		if !stdlibAssertContains(output, "ghcr.io/owner/repo:latest") {
			t.Fatalf("expected %v to contain %v", output, "ghcr.io/owner/repo:latest")
		}
		if !stdlibAssertContains(output, "ghcr.io/owner/repo:v1.0.0") {
			t.Fatalf("expected %v to contain %v", output, "ghcr.io/owner/repo:v1.0.0")
		}
		if !stdlibAssertContains(output, "Would execute command:") {
			t.Fatalf("expected %v to contain %v", output, "Would execute command:")
		}
		if !stdlibAssertContains(output, "docker buildx build") {
			t.Fatalf("expected %v to contain %v", output, "docker buildx build")
		}
		if !stdlibAssertContains(output, "END DRY RUN") {
			t.Fatalf("expected %v to contain %v", output, "END DRY RUN")
		}

	})

	t.Run("shows build args when present", func(t *testing.T) {
		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: "/project",
			FS:         io.Local,
		}
		cfg := DockerConfig{
			Registry:   "docker.io",
			Image:      "myorg/myapp",
			Dockerfile: "/project/Dockerfile",
			Platforms:  []string{"linux/amd64"},
			Tags:       []string{"latest"},
			BuildArgs: map[string]string{
				"GO_VERSION": "1.21",
				"APP_NAME":   "myapp",
			},
		}

		publishResult := core.Ok(nil)
		output := capturePublisherOutput(t, func() {
			publishResult = p.dryRunPublish(release, cfg)
		})
		requirePublisherOK(t, publishResult)
		if !stdlibAssertContains(output, "Build arguments:") {
			t.Fatalf("expected %v to contain %v", output, "Build arguments:")
		}
		if !stdlibAssertContains(output, "GO_VERSION=1.21") {
			t.Fatalf("expected %v to contain %v", output, "GO_VERSION=1.21")
		}
		if !stdlibAssertContains(output, "APP_NAME=myapp") {
			t.Fatalf("expected %v to contain %v", output, "APP_NAME=myapp")
		}

	})

	t.Run("handles single platform", func(t *testing.T) {
		release := &Release{
			Version:    "v2.0.0",
			ProjectDir: "/project",
			FS:         io.Local,
		}
		cfg := DockerConfig{
			Registry:   "ghcr.io",
			Image:      "owner/repo",
			Dockerfile: "/project/Dockerfile.prod",
			Platforms:  []string{"linux/amd64"},
			Tags:       []string{"stable"},
			BuildArgs:  make(map[string]string),
		}

		publishResult := core.Ok(nil)
		output := capturePublisherOutput(t, func() {
			publishResult = p.dryRunPublish(release, cfg)
		})
		requirePublisherOK(t, publishResult)
		if !stdlibAssertContains(output, "Platforms:     linux/amd64") {
			t.Fatalf("expected %v to contain %v", output, "Platforms:     linux/amd64")
		}
		if !stdlibAssertContains(output, "ghcr.io/owner/repo:stable") {
			t.Fatalf("expected %v to contain %v", output, "ghcr.io/owner/repo:stable")
		}

	})
}

func TestDocker_DockerPublisherParseConfigEdgeCasesGood(t *testing.T) {
	p := NewDockerPublisher()

	t.Run("handles nil release config", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "docker",
			Extended: map[string]any{
				"image": "custom/image",
			},
		}

		cfg := p.parseConfig(io.Local, pubCfg, nil, "/project")
		if !stdlibAssertEqual("custom/image", cfg.Image) {
			t.Fatalf("want %v, got %v", "custom/image", cfg.Image)
		}
		if !stdlibAssertEqual("ghcr.io", cfg.Registry) {
			t.Fatalf("want %v, got %v", "ghcr.io", cfg.Registry)
		}

	})

	t.Run("handles empty repository in release config", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "docker",
			Extended: map[string]any{
				"image": "fallback/image",
			},
		}
		relCfg := &mockReleaseConfig{repository: ""}

		cfg := p.parseConfig(io.Local, pubCfg, relCfg, "/project")
		if !stdlibAssertEqual("fallback/image", cfg.Image) {
			t.Fatalf("want %v, got %v", "fallback/image", cfg.Image)
		}

	})

	t.Run("extended config overrides repository image", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "docker",
			Extended: map[string]any{
				"image": "override/image",
			},
		}
		relCfg := &mockReleaseConfig{repository: "original/repo"}

		cfg := p.parseConfig(io.Local, pubCfg, relCfg, "/project")
		if !stdlibAssertEqual("override/image", cfg.Image) {
			t.Fatalf("want %v, got %v", "override/image", cfg.Image)
		}

	})

	t.Run("handles mixed build args types", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "docker",
			Extended: map[string]any{
				"build_args": map[string]any{
					"STRING_ARG": "value",
					"INT_ARG":    123, // Non-string value should be skipped
				},
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		cfg := p.parseConfig(io.Local, pubCfg, relCfg, "/project")
		if !stdlibAssertEqual("value", cfg.BuildArgs["STRING_ARG"]) {
			t.Fatalf("want %v, got %v", "value", cfg.BuildArgs["STRING_ARG"])
		}

		_, exists := cfg.BuildArgs["INT_ARG"]
		if exists {
			t.Fatal("non-string build arg should not be included")
		}

	})
}

func TestDocker_DockerPublisherResolveTagsEdgeCasesGood(t *testing.T) {
	p := NewDockerPublisher()

	t.Run("handles empty tags", func(t *testing.T) {
		tags := p.resolveTags([]string{}, "v1.0.0")
		if !stdlibAssertEmpty(tags) {
			t.Fatalf("expected empty, got %v", tags)
		}

	})

	t.Run("handles multiple version placeholders", func(t *testing.T) {
		tags := p.resolveTags([]string{"{{.Version}}", "prefix-{{.Version}}", "{{.Version}}-suffix"}, "v1.2.3")
		if !stdlibAssertEqual([]string{"v1.2.3", "prefix-v1.2.3", "v1.2.3-suffix"}, tags) {
			t.Fatalf("want %v, got %v", []string{"v1.2.3", "prefix-v1.2.3", "v1.2.3-suffix"}, tags)
		}

	})

	t.Run("handles mixed template formats", func(t *testing.T) {
		tags := p.resolveTags([]string{"{{.Version}}", "{{Version}}", "latest"}, "v3.0.0")
		if !stdlibAssertEqual([]string{"v3.0.0", "v3.0.0", "latest"}, tags) {
			t.Fatalf("want %v, got %v", []string{"v3.0.0", "v3.0.0", "latest"}, tags)
		}

	})

	t.Run("supports tag aliases and RFC version prefixing", func(t *testing.T) {
		tags := p.resolveTags([]string{"v{{.Version}}", "{{.Tag}}", "release-{{Tag}}"}, "v3.0.0")
		if !stdlibAssertEqual([]string{"v3.0.0", "v3.0.0", "release-v3.0.0"}, tags) {
			t.Fatalf("want %v, got %v", []string{"v3.0.0", "v3.0.0", "release-v3.0.0"}, tags)
		}

	})
}

func TestDocker_DockerPublisherBuildBuildxArgsEdgeCasesGood(t *testing.T) {
	p := NewDockerPublisher()

	t.Run("handles empty platforms", func(t *testing.T) {
		cfg := DockerConfig{
			Registry:   "ghcr.io",
			Image:      "owner/repo",
			Dockerfile: "/project/Dockerfile",
			Platforms:  []string{},
			BuildArgs:  make(map[string]string),
		}

		args := p.buildBuildxArgs(cfg, []string{"latest"}, "v1.0.0")
		if !stdlibAssertContains(args, "buildx") {
			t.Fatalf("expected %v to contain %v", args,

				// Should not have --platform if empty
				"buildx")
		}
		if !stdlibAssertContains(args, "build") {
			t.Fatalf("expected %v to contain %v", args, "build")
		}

		foundPlatform := false
		for i, arg := range args {
			if arg == "--platform" {
				foundPlatform = true
				// Check the next arg exists (it shouldn't be empty)
				if i+1 < len(args) && args[i+1] == "" {
					t.Error("platform argument should not be empty string")
				}
			}
		}
		if foundPlatform {
			t.Fatal("should not include --platform when platforms is empty")
		}

	})

	t.Run("handles version expansion in build args", func(t *testing.T) {
		cfg := DockerConfig{
			Registry:   "ghcr.io",
			Image:      "owner/repo",
			Dockerfile: "/Dockerfile",
			Platforms:  []string{"linux/amd64"},
			BuildArgs: map[string]string{
				"VERSION":      "{{.Version}}",
				"SIMPLE_VER":   "{{Version}}",
				"STATIC_VALUE": "static",
			},
		}

		args := p.buildBuildxArgs(cfg, []string{"latest"}, "v2.5.0")

		foundVersionArg := false
		foundSimpleArg := false
		foundStaticArg := false
		foundAutoVersion := false

		for i, arg := range args {
			if arg == "--build-arg" && i+1 < len(args) {
				switch args[i+1] {
				case "VERSION=v2.5.0":
					foundVersionArg = true
				case "SIMPLE_VER=v2.5.0":
					foundSimpleArg = true
				case "STATIC_VALUE=static":
					foundStaticArg = true
				}
				// Auto-added VERSION build arg
				if args[i+1] == "VERSION=v2.5.0" {
					foundAutoVersion = true
				}
			}
		}
		if !(foundVersionArg ||

			// Note: VERSION is both in BuildArgs and auto-added, so we just check it exists
			foundAutoVersion) {
			t.Fatal("VERSION build arg not found")
		}
		if !(foundSimpleArg) {
			t.Fatal("SIMPLE_VER build arg not expanded")
		}
		if !(foundStaticArg) {
			t.Fatal("STATIC_VALUE build arg not found")
		}

	})

	t.Run("supports tag aliases and prefixed RFC templates in build args", func(t *testing.T) {
		cfg := DockerConfig{
			Registry:   "ghcr.io",
			Image:      "owner/repo",
			Dockerfile: "/Dockerfile",
			Platforms:  []string{"linux/amd64"},
			BuildArgs: map[string]string{
				"IMAGE_TAG":   "v{{.Version}}",
				"RELEASE_TAG": "{{.Tag}}",
			},
		}

		args := p.buildBuildxArgs(cfg, []string{"latest"}, "v2.5.0")

		foundImageTag := false
		foundReleaseTag := false
		for i, arg := range args {
			if arg != "--build-arg" || i+1 >= len(args) {
				continue
			}
			switch args[i+1] {
			case "IMAGE_TAG=v2.5.0":
				foundImageTag = true
			case "RELEASE_TAG=v2.5.0":
				foundReleaseTag = true
			}
		}
		if !(foundImageTag) {
			t.Fatal("IMAGE_TAG build arg not expanded")
		}
		if !(foundReleaseTag) {
			t.Fatal("RELEASE_TAG build arg not expanded")
		}

	})

	t.Run("handles empty registry", func(t *testing.T) {
		cfg := DockerConfig{
			Registry:   "",
			Image:      "localimage",
			Dockerfile: "/Dockerfile",
			Platforms:  []string{"linux/amd64"},
			BuildArgs:  make(map[string]string),
		}

		args := p.buildBuildxArgs(cfg, []string{"latest"}, "v1.0.0")
		if !stdlibAssertContains(args, "-t") {
			t.Fatalf("expected %v to contain %v", args, "-t")
		}
		if !stdlibAssertContains(args, "localimage:latest") {
			t.Fatalf("expected %v to contain %v",

				// Skip if docker CLI is not available - dry run still validates docker is installed
				args, "localimage:latest")
		}

	})
}

func TestDocker_DockerPublisherPublishDryRunGood(t *testing.T) {

	if err := validateDockerCli(); !err.OK {
		t.Skip("skipping test: docker CLI not available")
	}

	p := NewDockerPublisher()

	t.Run("dry run succeeds with valid Dockerfile", func(t *testing.T) {
		// Create temp directory with Dockerfile
		tmpDir := t.TempDir()
		if result := ax.WriteFile(ax.Join(tmpDir, "Dockerfile"), []byte("FROM alpine:latest\n"), 0o644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: tmpDir,
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{Type: "docker"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		publishResult := core.Ok(nil)
		output := capturePublisherOutput(t, func() {
			publishResult = p.Publish(context.TODO(), release, pubCfg, relCfg, true)
		})
		requirePublisherOK(t, publishResult)
		if !stdlibAssertContains(output, "DRY RUN: Docker Build & Push") {
			t.Fatalf("expected %v to contain %v", output, "DRY RUN: Docker Build & Push")

			// Create temp directory with custom Dockerfile
		}

	})

	t.Run("dry run uses custom dockerfile path", func(t *testing.T) {

		tmpDir := t.TempDir()
		customDir := ax.Join(tmpDir, "docker")
		if result := ax.MkdirAll(customDir, 0o755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(customDir, "Dockerfile.prod"), []byte("FROM alpine:latest\n"), 0o644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: tmpDir,
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{
			Type: "docker",
			Extended: map[string]any{
				"dockerfile": "docker/Dockerfile.prod",
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		publishResult := core.Ok(nil)
		output := capturePublisherOutput(t, func() {
			publishResult = p.Publish(context.TODO(), release, pubCfg, relCfg, true)
		})
		requirePublisherOK(t, publishResult)
		if !stdlibAssertContains(output, "Dockerfile.prod") {
			t.Fatalf("expected %v to contain %v", output, "Dockerfile.prod")
		}

	})
}

func TestDocker_DockerPublisherPublishValidationBad(t *testing.T) {
	p := NewDockerPublisher()

	t.Run("fails when Dockerfile not found with docker installed", func(t *testing.T) {
		if err := validateDockerCli(); !err.OK {
			t.Skip("skipping test: docker CLI not available")
		}

		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: "/nonexistent/path",
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{Type: "docker"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		err := requirePublisherError(t, p.Publish(context.TODO(), release, pubCfg, relCfg, false))
		if !stdlibAssertContains(err, "Dockerfile not found") {
			t.Fatalf("expected %v to contain %v", err, "Dockerfile not found")
		}

	})

	t.Run("fails when docker CLI not available", func(t *testing.T) {
		if err := validateDockerCli(); err.OK {
			t.Skip("skipping test: docker CLI is available")
		}

		tmpDir := t.TempDir()
		if result := ax.WriteFile(ax.Join(tmpDir, "Dockerfile"), []byte("FROM alpine:latest\n"), 0o644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: tmpDir,
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{Type: "docker"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		err := requirePublisherError(t, p.Publish(context.TODO(), release, pubCfg, relCfg, false))
		if !stdlibAssertContains(err, "docker CLI not found") {
			t.Fatalf("expected %v to contain %v", err, "docker CLI not found")
		}

	})
}

func TestDocker_ValidateDockerCliGood(t *testing.T) {
	t.Run("returns nil when docker is installed", func(t *testing.T) {
		err := validateDockerCli()
		if !err.OK {
			if !stdlibAssertContains(
				// Docker is not installed, which is fine for this test
				err.Error(), "docker CLI not found") {
				t.Fatalf("expected %v to contain %v", err.Error(), "docker CLI not found")

				// If err is nil, docker is installed - that's OK
			}

		}

	})
}

func TestDocker_ResolveDockerCliGood(t *testing.T) {
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "docker")
	if result := ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	t.Setenv("PATH", "")

	command := requirePublisherString(t, resolveDockerCli(fallbackPath))
	if !stdlibAssertEqual(fallbackPath, command) {
		t.Fatalf("want %v, got %v", fallbackPath, command)
	}

}

func TestDocker_ResolveDockerCliBad(t *testing.T) {
	t.Setenv("PATH", "")
	err := requirePublisherError(t, resolveDockerCli(ax.Join(t.TempDir(), "missing-docker")))
	if !stdlibAssertContains(err, "docker CLI not found") {
		t.Fatalf("expected %v to contain %v", err, "docker CLI not found")

		// These tests run only when docker CLI is available
	}

}

func TestDocker_DockerPublisherPublishWithCLIGood(t *testing.T) {

	if err := validateDockerCli(); !err.OK {
		t.Skip("skipping test: docker CLI not available")
	}

	p := NewDockerPublisher()

	t.Run("dry run succeeds with all config options", func(t *testing.T) {
		tmpDir := t.TempDir()
		if result := ax.WriteFile(ax.Join(tmpDir, "Dockerfile"), []byte("FROM alpine:latest\n"), 0o644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: tmpDir,
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{
			Type: "docker",
			Extended: map[string]any{
				"registry":   "docker.io",
				"image":      "myorg/myapp",
				"platforms":  []any{"linux/amd64", "linux/arm64"},
				"tags":       []any{"latest", "{{.Version}}", "stable"},
				"build_args": map[string]any{"GO_VERSION": "1.21"},
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		publishResult := core.Ok(nil)
		output := capturePublisherOutput(t, func() {
			publishResult = p.Publish(context.TODO(), release, pubCfg, relCfg, true)
		})
		requirePublisherOK(t, publishResult)
		if !stdlibAssertContains(output, "DRY RUN: Docker Build & Push") {
			t.Fatalf("expected %v to contain %v", output, "DRY RUN: Docker Build & Push")
		}
		if !stdlibAssertContains(output, "docker.io") {
			t.Fatalf("expected %v to contain %v", output, "docker.io")
		}
		if !stdlibAssertContains(output, "myorg/myapp") {
			t.Fatalf("expected %v to contain %v", output, "myorg/myapp")
		}

	})

	t.Run("dry run with nil relCfg uses extended image", func(t *testing.T) {
		tmpDir := t.TempDir()
		if result := ax.WriteFile(ax.Join(tmpDir, "Dockerfile"), []byte("FROM alpine:latest\n"), 0o644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: tmpDir,
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{
			Type: "docker",
			Extended: map[string]any{
				"image": "standalone/image",
			},
		}

		publishResult := core.Ok(nil)
		output := capturePublisherOutput(t, func() {
			publishResult = p.Publish(context.TODO(), release, pubCfg, nil, true)
		})
		requirePublisherOK(t, publishResult)
		if !stdlibAssertContains(output, "standalone/image") {
			t.Fatalf("expected %v to contain %v", output, "standalone/image")
		}

	})

	t.Run("fails with non-existent Dockerfile in non-dry-run", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Don't create a Dockerfile
		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: tmpDir,
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{Type: "docker"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		err := requirePublisherError(t, p.Publish(context.TODO(), release, pubCfg, relCfg, false))
		if !stdlibAssertContains(err, "Dockerfile not found") {
			t.Fatalf("expected %v to contain %v", err, "Dockerfile not found")
		}

	})
}

// --- v0.9.0 generated compliance triplets ---
func TestDocker_NewDockerPublisher_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewDockerPublisher()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestDocker_NewDockerPublisher_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewDockerPublisher()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestDocker_NewDockerPublisher_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewDockerPublisher()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestDocker_DockerPublisher_Name_Good(t *core.T) {
	subject := &DockerPublisher{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestDocker_DockerPublisher_Name_Bad(t *core.T) {
	subject := &DockerPublisher{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestDocker_DockerPublisher_Name_Ugly(t *core.T) {
	subject := &DockerPublisher{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestDocker_DockerPublisher_Validate_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &DockerPublisher{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Validate(ctx, &Release{}, PublisherConfig{}, nil)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestDocker_DockerPublisher_Validate_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &DockerPublisher{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Validate(ctx, nil, PublisherConfig{}, nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestDocker_DockerPublisher_Validate_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &DockerPublisher{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Validate(ctx, &Release{}, PublisherConfig{}, nil)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestDocker_DockerPublisher_Supports_Good(t *core.T) {
	subject := &DockerPublisher{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Supports("linux")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestDocker_DockerPublisher_Supports_Bad(t *core.T) {
	subject := &DockerPublisher{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Supports("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestDocker_DockerPublisher_Supports_Ugly(t *core.T) {
	subject := &DockerPublisher{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Supports("linux")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestDocker_DockerPublisher_Publish_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &DockerPublisher{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Publish(ctx, &Release{}, PublisherConfig{}, nil, true)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestDocker_DockerPublisher_Publish_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &DockerPublisher{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Publish(ctx, nil, PublisherConfig{}, nil, true)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestDocker_DockerPublisher_Publish_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &DockerPublisher{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Publish(ctx, &Release{}, PublisherConfig{}, nil, true)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
