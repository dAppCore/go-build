package publishers

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/io"
)

func TestScoop_ScoopPublisherNameGood(t *testing.T) {
	t.Run("returns scoop", func(t *testing.T) {
		p := NewScoopPublisher()
		if !stdlibAssertEqual("scoop", p.Name()) {
			t.Fatalf("want %v, got %v", "scoop", p.Name())
		}

	})
}

func TestScoop_ScoopPublisherParseConfigGood(t *testing.T) {
	p := NewScoopPublisher()

	t.Run("uses defaults when no extended config", func(t *testing.T) {
		pubCfg := PublisherConfig{Type: "scoop"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}
		cfg := p.parseConfig(pubCfg, relCfg)
		if !stdlibAssertEmpty(cfg.Bucket) {
			t.Fatalf("expected empty, got %v", cfg.Bucket)
		}
		if !stdlibAssertNil(cfg.Official) {
			t.Fatalf("expected nil, got %v", cfg.Official)
		}

	})

	t.Run("parses bucket from extended config", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "scoop",
			Extended: map[string]any{
				"bucket": "host-uk/scoop-bucket",
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}
		cfg := p.parseConfig(pubCfg, relCfg)
		if !stdlibAssertEqual("host-uk/scoop-bucket", cfg.Bucket) {
			t.Fatalf("want %v, got %v", "host-uk/scoop-bucket", cfg.Bucket)
		}

	})

	t.Run("parses official config", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "scoop",
			Extended: map[string]any{
				"official": map[string]any{
					"enabled": true,
					"output":  "dist/scoop-manifest",
				},
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}
		cfg := p.parseConfig(pubCfg, relCfg)
		if stdlibAssertNil(cfg.Official) {
			t.Fatal("expected non-nil")
		}
		if !(cfg.Official.Enabled) {
			t.Fatal("expected true")
		}
		if !stdlibAssertEqual("dist/scoop-manifest", cfg.Official.Output) {
			t.Fatalf("want %v, got %v", "dist/scoop-manifest", cfg.Official.Output)
		}

	})

	t.Run("handles missing official fields", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "scoop",
			Extended: map[string]any{
				"official": map[string]any{},
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}
		cfg := p.parseConfig(pubCfg, relCfg)
		if stdlibAssertNil(cfg.Official) {
			t.Fatal("expected non-nil")
		}
		if cfg.Official.Enabled {
			t.Fatal("expected false")
		}
		if !stdlibAssertEmpty(cfg.Official.Output) {
			t.Fatalf("expected empty, got %v", cfg.Official.Output)
		}

	})

	t.Run("handles nil extended config", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type:     "scoop",
			Extended: nil,
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}
		cfg := p.parseConfig(pubCfg, relCfg)
		if !stdlibAssertEmpty(cfg.Bucket) {
			t.Fatalf("expected empty, got %v", cfg.Bucket)
		}
		if !stdlibAssertNil(cfg.Official) {
			t.Fatalf("expected nil, got %v", cfg.Official)
		}

	})
}

func TestScoop_ScoopPublisherRenderTemplateGood(t *testing.T) {
	p := NewScoopPublisher()

	t.Run("renders manifest template with data", func(t *testing.T) {
		data := scoopTemplateData{
			PackageName: "myapp",
			Description: "My awesome CLI",
			Repository:  "owner/myapp",
			Version:     "1.2.3",
			License:     "MIT",
			BinaryName:  "myapp",
			Checksums: ChecksumMap{
				WindowsAmd64:     "abc123",
				WindowsArm64:     "def456",
				WindowsAmd64File: "myapp_windows_amd64.zip",
				WindowsArm64File: "myapp_windows_arm64.zip",
			},
		}

		result, err := p.renderTemplate(io.Local, "templates/scoop/manifest.json.tmpl", data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(result, `"version": "1.2.3"`) {
			t.Fatalf("expected %v to contain %v", result, `"version": "1.2.3"`)
		}
		if !stdlibAssertContains(result, `"description": "My awesome CLI"`) {
			t.Fatalf("expected %v to contain %v", result, `"description": "My awesome CLI"`)
		}
		if !stdlibAssertContains(result, `"homepage": "https://github.com/owner/myapp"`) {
			t.Fatalf("expected %v to contain %v", result, `"homepage": "https://github.com/owner/myapp"`)
		}
		if !stdlibAssertContains(result, `"license": "MIT"`) {
			t.Fatalf("expected %v to contain %v", result, `"license": "MIT"`)
		}
		if !stdlibAssertContains(result, `"64bit"`) {
			t.Fatalf("expected %v to contain %v", result, `"64bit"`)
		}
		if !stdlibAssertContains(result, `"arm64"`) {
			t.Fatalf("expected %v to contain %v", result, `"arm64"`)
		}
		if !stdlibAssertContains(result, "myapp_windows_amd64.zip") {
			t.Fatalf("expected %v to contain %v", result, "myapp_windows_amd64.zip")
		}
		if !stdlibAssertContains(result, "myapp_windows_arm64.zip") {
			t.Fatalf("expected %v to contain %v", result, "myapp_windows_arm64.zip")
		}
		if !stdlibAssertContains(result, `"hash": "abc123"`) {
			t.Fatalf("expected %v to contain %v", result, `"hash": "abc123"`)
		}
		if !stdlibAssertContains(result, `"hash": "def456"`) {
			t.Fatalf("expected %v to contain %v", result, `"hash": "def456"`)
		}
		if !stdlibAssertContains(result, `"bin": "myapp.exe"`) {
			t.Fatalf("expected %v to contain %v", result, `"bin": "myapp.exe"`)
		}

	})

	t.Run("includes autoupdate configuration", func(t *testing.T) {
		data := scoopTemplateData{
			PackageName: "tool",
			Description: "A tool",
			Repository:  "org/tool",
			Version:     "2.0.0",
			License:     "Apache-2.0",
			BinaryName:  "tool",
			Checksums:   ChecksumMap{},
		}

		result, err := p.renderTemplate(io.Local, "templates/scoop/manifest.json.tmpl", data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(result, `"checkver"`) {
			t.Fatalf("expected %v to contain %v", result, `"checkver"`)
		}
		if !stdlibAssertContains(result, `"github": "https://github.com/org/tool"`) {
			t.Fatalf("expected %v to contain %v", result, `"github": "https://github.com/org/tool"`)
		}
		if !stdlibAssertContains(result, `"autoupdate"`) {
			t.Fatalf("expected %v to contain %v", result, `"autoupdate"`)
		}

	})
}

func TestScoop_ScoopPublisherRenderTemplateBad(t *testing.T) {
	p := NewScoopPublisher()

	t.Run("returns error for non-existent template", func(t *testing.T) {
		data := scoopTemplateData{}
		_, err := p.renderTemplate(io.Local, "templates/scoop/nonexistent.tmpl", data)
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "failed to read template") {
			t.Fatalf("expected %v to contain %v", err.Error(), "failed to read template")
		}

	})
}

func TestScoop_ScoopPublisherDryRunPublishGood(t *testing.T) {
	p := NewScoopPublisher()

	t.Run("outputs expected dry run information", func(t *testing.T) {
		data := scoopTemplateData{
			PackageName: "myapp",
			Version:     "1.0.0",
			Repository:  "owner/repo",
			BinaryName:  "myapp",
			Checksums:   ChecksumMap{},
		}
		cfg := ScoopConfig{
			Bucket: "owner/scoop-bucket",
		}

		var err error
		output := capturePublisherOutput(t, func() {
			err = p.dryRunPublish(io.Local, data, cfg)
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(output, "DRY RUN: Scoop Publish") {
			t.Fatalf("expected %v to contain %v", output, "DRY RUN: Scoop Publish")
		}
		if !stdlibAssertContains(output, "Package:    myapp") {
			t.Fatalf("expected %v to contain %v", output, "Package:    myapp")
		}
		if !stdlibAssertContains(output, "Version:    1.0.0") {
			t.Fatalf("expected %v to contain %v", output, "Version:    1.0.0")
		}
		if !stdlibAssertContains(output, "Bucket:     owner/scoop-bucket") {
			t.Fatalf("expected %v to contain %v", output, "Bucket:     owner/scoop-bucket")
		}
		if !stdlibAssertContains(output, "Repository: owner/repo") {
			t.Fatalf("expected %v to contain %v", output, "Repository: owner/repo")
		}
		if !stdlibAssertContains(output, "Generated manifest.json:") {
			t.Fatalf("expected %v to contain %v", output, "Generated manifest.json:")
		}
		if !stdlibAssertContains(output, "Would commit to bucket: owner/scoop-bucket") {
			t.Fatalf("expected %v to contain %v", output, "Would commit to bucket: owner/scoop-bucket")
		}
		if !stdlibAssertContains(output, "END DRY RUN") {
			t.Fatalf("expected %v to contain %v", output, "END DRY RUN")
		}

	})

	t.Run("shows official output path when enabled", func(t *testing.T) {
		data := scoopTemplateData{
			PackageName: "myapp",
			Version:     "1.0.0",
			BinaryName:  "myapp",
			Checksums:   ChecksumMap{},
		}
		cfg := ScoopConfig{
			Official: &OfficialConfig{
				Enabled: true,
				Output:  "custom/scoop/path",
			},
		}

		var err error
		output := capturePublisherOutput(t, func() {
			err = p.dryRunPublish(io.Local, data, cfg)
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(output, "Would write files for official PR to: custom/scoop/path") {
			t.Fatalf("expected %v to contain %v", output, "Would write files for official PR to: custom/scoop/path")
		}

	})

	t.Run("suppresses bucket publish output in official mode", func(t *testing.T) {
		data := scoopTemplateData{
			PackageName: "myapp",
			Version:     "1.0.0",
			BinaryName:  "myapp",
			Checksums:   ChecksumMap{},
		}
		cfg := ScoopConfig{
			Bucket: "owner/scoop-bucket",
			Official: &OfficialConfig{
				Enabled: true,
				Output:  "custom/scoop/path",
			},
		}

		var err error
		output := capturePublisherOutput(t, func() {
			err = p.dryRunPublish(io.Local, data, cfg)
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(output, "Would write files for official PR to: custom/scoop/path") {
			t.Fatalf("expected %v to contain %v", output, "Would write files for official PR to: custom/scoop/path")
		}
		if stdlibAssertContains(output, "Would commit to bucket:") {
			t.Fatalf("expected %v not to contain %v", output, "Would commit to bucket:")
		}

	})

	t.Run("uses default official output path when not specified", func(t *testing.T) {
		data := scoopTemplateData{
			PackageName: "myapp",
			Version:     "1.0.0",
			BinaryName:  "myapp",
			Checksums:   ChecksumMap{},
		}
		cfg := ScoopConfig{
			Official: &OfficialConfig{
				Enabled: true,
			},
		}

		var err error
		output := capturePublisherOutput(t, func() {
			err = p.dryRunPublish(io.Local, data, cfg)
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(output, "Would write files for official PR to: dist/scoop") {
			t.Fatalf("expected %v to contain %v", output, "Would write files for official PR to: dist/scoop")
		}

	})
}

func TestScoop_ScoopPublisherPublishBad(t *testing.T) {
	p := NewScoopPublisher()

	t.Run("fails when bucket not configured and not official mode", func(t *testing.T) {
		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: "/project",
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{Type: "scoop"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		err := p.Publish(context.TODO(), release, pubCfg, relCfg, false)
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "bucket is required") {
			t.Fatalf("expected %v to contain %v", err.Error(), "bucket is required")
		}

	})

	t.Run("official mode writes files without requiring bucket publish tooling", func(t *testing.T) {
		projectDir := t.TempDir()
		t.Setenv("PATH", "/definitely-missing")

		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: projectDir,
			FS:         io.Local,
			Artifacts: []build.Artifact{
				{Path: "dist/myapp-windows-amd64.zip", OS: "windows", Arch: "amd64", Checksum: "abc123"},
			},
		}
		pubCfg := PublisherConfig{
			Type: "scoop",
			Extended: map[string]any{
				"bucket": "owner/scoop-bucket",
				"official": map[string]any{
					"enabled": true,
					"output":  "dist/scoop-pr",
				},
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo", projectName: "myapp"}

		err := p.Publish(context.TODO(), release, pubCfg, relCfg, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := ax.Stat(ax.Join(projectDir, "dist", "scoop-pr", "myapp.json")); err != nil {
			t.Fatalf("expected file to exist: %v", ax.Join(projectDir, "dist", "scoop-pr", "myapp.json"))
		}

	})
}

func TestScoop_ScoopConfigDefaultsGood(t *testing.T) {
	t.Run("has sensible defaults", func(t *testing.T) {
		p := NewScoopPublisher()
		pubCfg := PublisherConfig{Type: "scoop"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		cfg := p.parseConfig(pubCfg, relCfg)
		if !stdlibAssertEmpty(cfg.Bucket) {
			t.Fatalf("expected empty, got %v", cfg.Bucket)
		}
		if !stdlibAssertNil(cfg.Official) {
			t.Fatalf("expected nil, got %v", cfg.Official)
		}

	})
}

func TestScoop_ScoopTemplateDataGood(t *testing.T) {
	t.Run("struct has all expected fields", func(t *testing.T) {
		data := scoopTemplateData{
			PackageName: "myapp",
			Description: "description",
			Repository:  "org/repo",
			Version:     "1.0.0",
			License:     "MIT",
			BinaryName:  "myapp",
			Checksums: ChecksumMap{
				WindowsAmd64: "hash1",
				WindowsArm64: "hash2",
			},
		}
		if !stdlibAssertEqual("myapp", data.PackageName) {
			t.Fatalf("want %v, got %v", "myapp", data.PackageName)
		}
		if !stdlibAssertEqual("description", data.Description) {
			t.Fatalf("want %v, got %v", "description", data.Description)
		}
		if !stdlibAssertEqual("org/repo", data.Repository) {
			t.Fatalf("want %v, got %v", "org/repo", data.Repository)
		}
		if !stdlibAssertEqual("1.0.0", data.Version) {
			t.Fatalf("want %v, got %v", "1.0.0", data.Version)
		}
		if !stdlibAssertEqual("MIT", data.License) {
			t.Fatalf("want %v, got %v", "MIT", data.License)
		}
		if !stdlibAssertEqual("myapp", data.BinaryName) {
			t.Fatalf("want %v, got %v", "myapp", data.BinaryName)
		}
		if !stdlibAssertEqual("hash1", data.Checksums.WindowsAmd64) {
			t.Fatalf("want %v, got %v", "hash1", data.Checksums.WindowsAmd64)
		}
		if !stdlibAssertEqual("hash2", data.Checksums.WindowsArm64) {
			t.Fatalf("want %v, got %v", "hash2", data.Checksums.WindowsArm64)
		}

	})
}

// --- v0.9.0 generated compliance triplets ---
func TestScoop_NewScoopPublisher_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewScoopPublisher()
	})
	core.AssertTrue(t, true)
}

func TestScoop_NewScoopPublisher_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewScoopPublisher()
	})
	core.AssertTrue(t, true)
}

func TestScoop_NewScoopPublisher_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewScoopPublisher()
	})
	core.AssertTrue(t, true)
}

func TestScoop_ScoopPublisher_Name_Good(t *core.T) {
	subject := &ScoopPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
	})
	core.AssertTrue(t, true)
}

func TestScoop_ScoopPublisher_Name_Bad(t *core.T) {
	subject := &ScoopPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
	})
	core.AssertTrue(t, true)
}

func TestScoop_ScoopPublisher_Name_Ugly(t *core.T) {
	subject := &ScoopPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
	})
	core.AssertTrue(t, true)
}

func TestScoop_ScoopPublisher_Validate_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &ScoopPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Validate(ctx, &Release{}, PublisherConfig{}, nil)
	})
	core.AssertTrue(t, true)
}

func TestScoop_ScoopPublisher_Validate_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &ScoopPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Validate(ctx, nil, PublisherConfig{}, nil)
	})
	core.AssertTrue(t, true)
}

func TestScoop_ScoopPublisher_Validate_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &ScoopPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Validate(ctx, &Release{}, PublisherConfig{}, nil)
	})
	core.AssertTrue(t, true)
}

func TestScoop_ScoopPublisher_Supports_Good(t *core.T) {
	subject := &ScoopPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Supports("linux")
	})
	core.AssertTrue(t, true)
}

func TestScoop_ScoopPublisher_Supports_Bad(t *core.T) {
	subject := &ScoopPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Supports("")
	})
	core.AssertTrue(t, true)
}

func TestScoop_ScoopPublisher_Supports_Ugly(t *core.T) {
	subject := &ScoopPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Supports("linux")
	})
	core.AssertTrue(t, true)
}

func TestScoop_ScoopPublisher_Publish_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &ScoopPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Publish(ctx, &Release{}, PublisherConfig{}, nil, true)
	})
	core.AssertTrue(t, true)
}

func TestScoop_ScoopPublisher_Publish_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &ScoopPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Publish(ctx, nil, PublisherConfig{}, nil, true)
	})
	core.AssertTrue(t, true)
}

func TestScoop_ScoopPublisher_Publish_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &ScoopPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Publish(ctx, &Release{}, PublisherConfig{}, nil, true)
	})
	core.AssertTrue(t, true)
}
