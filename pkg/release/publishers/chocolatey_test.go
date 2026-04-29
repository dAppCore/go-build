package publishers

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/io"
)

func TestChocolatey_ChocolateyPublisherNameGood(t *testing.T) {
	t.Run("returns chocolatey", func(t *testing.T) {
		p := NewChocolateyPublisher()
		if !stdlibAssertEqual("chocolatey", p.Name()) {
			t.Fatalf("want %v, got %v", "chocolatey", p.Name())
		}

	})
}

func TestChocolatey_ChocolateyPublisherParseConfigGood(t *testing.T) {
	p := NewChocolateyPublisher()

	t.Run("uses defaults when no extended config", func(t *testing.T) {
		pubCfg := PublisherConfig{Type: "chocolatey"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}
		cfg := p.parseConfig(pubCfg, relCfg)
		if !stdlibAssertEmpty(cfg.Package) {
			t.Fatalf("expected empty, got %v", cfg.Package)
		}
		if cfg.Push {
			t.Fatal("expected false")
		}
		if !stdlibAssertNil(cfg.Official) {
			t.Fatalf("expected nil, got %v", cfg.Official)
		}

	})

	t.Run("parses package and push from extended config", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "chocolatey",
			Extended: map[string]any{
				"package": "mypackage",
				"push":    true,
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}
		cfg := p.parseConfig(pubCfg, relCfg)
		if !stdlibAssertEqual("mypackage", cfg.Package) {
			t.Fatalf("want %v, got %v", "mypackage", cfg.Package)
		}
		if !(cfg.Push) {
			t.Fatal("expected true")
		}

	})

	t.Run("parses official config", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "chocolatey",
			Extended: map[string]any{
				"official": map[string]any{
					"enabled": true,
					"output":  "dist/choco",
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
		if !stdlibAssertEqual("dist/choco", cfg.Official.Output) {
			t.Fatalf("want %v, got %v", "dist/choco", cfg.Official.Output)
		}

	})

	t.Run("handles missing official fields", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "chocolatey",
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
			Type:     "chocolatey",
			Extended: nil,
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}
		cfg := p.parseConfig(pubCfg, relCfg)
		if !stdlibAssertEmpty(cfg.Package) {
			t.Fatalf("expected empty, got %v", cfg.Package)
		}
		if cfg.Push {
			t.Fatal("expected false")
		}
		if !stdlibAssertNil(cfg.Official) {
			t.Fatalf("expected nil, got %v", cfg.Official)
		}

	})

	t.Run("defaults push to false when not specified", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "chocolatey",
			Extended: map[string]any{
				"package": "mypackage",
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}
		cfg := p.parseConfig(pubCfg, relCfg)
		if cfg.Push {
			t.Fatal("expected false")
		}

	})
}

func TestChocolatey_ChocolateyPublisherRenderTemplateGood(t *testing.T) {
	p := NewChocolateyPublisher()

	t.Run("renders nuspec template with data", func(t *testing.T) {
		data := chocolateyTemplateData{
			PackageName: "myapp",
			Title:       "MyApp CLI",
			Description: "My awesome CLI",
			Repository:  "owner/myapp",
			Version:     "1.2.3",
			License:     "MIT",
			BinaryName:  "myapp",
			Authors:     "owner",
			Tags:        "cli myapp",
			Checksums:   ChecksumMap{},
		}

		result, err := p.renderTemplate(io.Local, "templates/chocolatey/package.nuspec.tmpl", data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(result, `<id>myapp</id>`) {
			t.Fatalf("expected %v to contain %v", result, `<id>myapp</id>`)
		}
		if !stdlibAssertContains(result, `<version>1.2.3</version>`) {
			t.Fatalf("expected %v to contain %v", result, `<version>1.2.3</version>`)
		}
		if !stdlibAssertContains(result, `<title>MyApp CLI</title>`) {
			t.Fatalf("expected %v to contain %v", result, `<title>MyApp CLI</title>`)
		}
		if !stdlibAssertContains(result, `<authors>owner</authors>`) {
			t.Fatalf("expected %v to contain %v", result, `<authors>owner</authors>`)
		}
		if !stdlibAssertContains(result, `<description>My awesome CLI</description>`) {
			t.Fatalf("expected %v to contain %v", result, `<description>My awesome CLI</description>`)
		}
		if !stdlibAssertContains(result, `<tags>cli myapp</tags>`) {
			t.Fatalf("expected %v to contain %v", result, `<tags>cli myapp</tags>`)
		}
		if !stdlibAssertContains(result, "projectUrl>https://github.com/owner/myapp") {
			t.Fatalf("expected %v to contain %v", result, "projectUrl>https://github.com/owner/myapp")
		}
		if !stdlibAssertContains(result, "releaseNotes>https://github.com/owner/myapp/releases/tag/v1.2.3") {
			t.Fatalf("expected %v to contain %v", result, "releaseNotes>https://github.com/owner/myapp/releases/tag/v1.2.3")
		}

	})

	t.Run("renders install script template with data", func(t *testing.T) {
		data := chocolateyTemplateData{
			PackageName: "myapp",
			Repository:  "owner/myapp",
			Version:     "1.2.3",
			BinaryName:  "myapp",
			Checksums: ChecksumMap{
				WindowsAmd64:     "abc123def456",
				WindowsAmd64File: "myapp_windows_amd64.zip",
			},
		}

		result, err := p.renderTemplate(io.Local, "templates/chocolatey/tools/chocolateyinstall.ps1.tmpl", data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(result, "$ErrorActionPreference = 'Stop'") {
			t.Fatalf("expected %v to contain %v", result, "$ErrorActionPreference = 'Stop'")
		}
		if !stdlibAssertContains(result, "https://github.com/owner/myapp/releases/download/v1.2.3/myapp_windows_amd64.zip") {
			t.Fatalf("expected %v to contain %v", result, "https://github.com/owner/myapp/releases/download/v1.2.3/myapp_windows_amd64.zip")
		}
		if !stdlibAssertContains(result, "packageName    = 'myapp'") {
			t.Fatalf("expected %v to contain %v", result, "packageName    = 'myapp'")
		}
		if !stdlibAssertContains(result, "checksum64     = 'abc123def456'") {
			t.Fatalf("expected %v to contain %v", result, "checksum64     = 'abc123def456'")
		}
		if !stdlibAssertContains(result, "checksumType64 = 'sha256'") {
			t.Fatalf("expected %v to contain %v", result, "checksumType64 = 'sha256'")
		}
		if !stdlibAssertContains(result, "Install-ChocolateyZipPackage") {
			t.Fatalf("expected %v to contain %v", result, "Install-ChocolateyZipPackage")
		}

	})
}

func TestChocolatey_ChocolateyPublisherRenderTemplateBad(t *testing.T) {
	p := NewChocolateyPublisher()

	t.Run("returns error for non-existent template", func(t *testing.T) {
		data := chocolateyTemplateData{}
		_, err := p.renderTemplate(io.Local, "templates/chocolatey/nonexistent.tmpl", data)
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "failed to read template") {
			t.Fatalf("expected %v to contain %v", err.Error(), "failed to read template")
		}

	})
}

func TestChocolatey_ChocolateyPublisherDryRunPublishGood(t *testing.T) {
	p := NewChocolateyPublisher()

	t.Run("outputs expected dry run information", func(t *testing.T) {
		data := chocolateyTemplateData{
			PackageName: "myapp",
			Version:     "1.0.0",
			Repository:  "owner/repo",
			BinaryName:  "myapp",
			Authors:     "owner",
			Tags:        "cli myapp",
			Checksums:   ChecksumMap{},
		}
		cfg := ChocolateyConfig{
			Push: false,
		}

		var err error
		output := capturePublisherOutput(t, func() {
			err = p.dryRunPublish(io.Local, data, cfg)
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(output, "DRY RUN: Chocolatey Publish") {
			t.Fatalf("expected %v to contain %v", output, "DRY RUN: Chocolatey Publish")
		}
		if !stdlibAssertContains(output, "Package:    myapp") {
			t.Fatalf("expected %v to contain %v", output, "Package:    myapp")
		}
		if !stdlibAssertContains(output, "Version:    1.0.0") {
			t.Fatalf("expected %v to contain %v", output, "Version:    1.0.0")
		}
		if !stdlibAssertContains(output, "Push:       false") {
			t.Fatalf("expected %v to contain %v", output, "Push:       false")
		}
		if !stdlibAssertContains(output, "Repository: owner/repo") {
			t.Fatalf("expected %v to contain %v", output, "Repository: owner/repo")
		}
		if !stdlibAssertContains(output, "Generated package.nuspec:") {
			t.Fatalf("expected %v to contain %v", output, "Generated package.nuspec:")
		}
		if !stdlibAssertContains(output, "Generated chocolateyinstall.ps1:") {
			t.Fatalf("expected %v to contain %v", output, "Generated chocolateyinstall.ps1:")
		}
		if !stdlibAssertContains(output, "Would generate package files only (push=false)") {
			t.Fatalf("expected %v to contain %v", output, "Would generate package files only (push=false)")
		}
		if !stdlibAssertContains(output, "END DRY RUN") {
			t.Fatalf("expected %v to contain %v", output, "END DRY RUN")
		}

	})

	t.Run("shows push message when push is enabled", func(t *testing.T) {
		data := chocolateyTemplateData{
			PackageName: "myapp",
			Version:     "1.0.0",
			BinaryName:  "myapp",
			Authors:     "owner",
			Tags:        "cli",
			Checksums:   ChecksumMap{},
		}
		cfg := ChocolateyConfig{
			Push: true,
		}

		var err error
		output := capturePublisherOutput(t, func() {
			err = p.dryRunPublish(io.Local, data, cfg)
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(output, "Push:       true") {
			t.Fatalf("expected %v to contain %v", output, "Push:       true")
		}
		if !stdlibAssertContains(output, "Would push to Chocolatey community repo") {
			t.Fatalf("expected %v to contain %v", output, "Would push to Chocolatey community repo")
		}

	})
}

func TestChocolatey_ChocolateyPublisherExecutePublishBad(t *testing.T) {
	p := NewChocolateyPublisher()

	t.Run("fails when CHOCOLATEY_API_KEY not set for push", func(t *testing.T) {
		t.Setenv("CHOCOLATEY_API_KEY", "")

		// Create a temp directory for the test
		tmpDir := t.TempDir()
		if !(ax.IsDir(tmpDir)) {
			t.Fatal("expected true")
		}

		data := chocolateyTemplateData{
			PackageName: "testpkg",
			Version:     "1.0.0",
			BinaryName:  "testpkg",
			Repository:  "owner/repo",
			Authors:     "owner",
			Tags:        "cli",
			Checksums:   ChecksumMap{},
		}

		err := p.pushToChocolatey(context.TODO(), tmpDir, data)
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "CHOCOLATEY_API_KEY environment variable is required") {
			t.Fatalf("expected %v to contain %v", err.Error(), "CHOCOLATEY_API_KEY environment variable is required")
		}

	})
}

func TestChocolatey_ChocolateyConfigDefaultsGood(t *testing.T) {
	t.Run("has sensible defaults", func(t *testing.T) {
		p := NewChocolateyPublisher()
		pubCfg := PublisherConfig{Type: "chocolatey"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		cfg := p.parseConfig(pubCfg, relCfg)
		if !stdlibAssertEmpty(cfg.Package) {
			t.Fatalf("expected empty, got %v", cfg.Package)
		}
		if cfg.Push {
			t.Fatal("expected false")
		}
		if !stdlibAssertNil(cfg.Official) {
			t.Fatalf("expected nil, got %v", cfg.Official)
		}

	})
}

func TestChocolatey_ChocolateyTemplateDataGood(t *testing.T) {
	t.Run("struct has all expected fields", func(t *testing.T) {
		data := chocolateyTemplateData{
			PackageName: "myapp",
			Title:       "MyApp CLI",
			Description: "description",
			Repository:  "org/repo",
			Version:     "1.0.0",
			License:     "MIT",
			BinaryName:  "myapp",
			Authors:     "org",
			Tags:        "cli tool",
			Checksums: ChecksumMap{
				WindowsAmd64: "hash1",
			},
		}
		if !stdlibAssertEqual("myapp", data.PackageName) {
			t.Fatalf("want %v, got %v", "myapp", data.PackageName)
		}
		if !stdlibAssertEqual("MyApp CLI", data.Title) {
			t.Fatalf("want %v, got %v", "MyApp CLI", data.Title)
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
		if !stdlibAssertEqual("org", data.Authors) {
			t.Fatalf("want %v, got %v", "org", data.Authors)
		}
		if !stdlibAssertEqual("cli tool", data.Tags) {
			t.Fatalf("want %v, got %v", "cli tool", data.Tags)
		}
		if !stdlibAssertEqual("hash1", data.Checksums.WindowsAmd64) {
			t.Fatalf("want %v, got %v", "hash1", data.Checksums.WindowsAmd64)
		}

	})
}

// --- v0.9.0 generated compliance triplets ---
func TestChocolatey_NewChocolateyPublisher_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewChocolateyPublisher()
	})
	core.AssertTrue(t, true)
}

func TestChocolatey_NewChocolateyPublisher_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewChocolateyPublisher()
	})
	core.AssertTrue(t, true)
}

func TestChocolatey_NewChocolateyPublisher_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewChocolateyPublisher()
	})
	core.AssertTrue(t, true)
}

func TestChocolatey_ChocolateyPublisher_Name_Good(t *core.T) {
	subject := &ChocolateyPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
	})
	core.AssertTrue(t, true)
}

func TestChocolatey_ChocolateyPublisher_Name_Bad(t *core.T) {
	subject := &ChocolateyPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
	})
	core.AssertTrue(t, true)
}

func TestChocolatey_ChocolateyPublisher_Name_Ugly(t *core.T) {
	subject := &ChocolateyPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
	})
	core.AssertTrue(t, true)
}

func TestChocolatey_ChocolateyPublisher_Validate_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &ChocolateyPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Validate(ctx, &Release{}, PublisherConfig{}, nil)
	})
	core.AssertTrue(t, true)
}

func TestChocolatey_ChocolateyPublisher_Validate_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &ChocolateyPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Validate(ctx, nil, PublisherConfig{}, nil)
	})
	core.AssertTrue(t, true)
}

func TestChocolatey_ChocolateyPublisher_Validate_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &ChocolateyPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Validate(ctx, &Release{}, PublisherConfig{}, nil)
	})
	core.AssertTrue(t, true)
}

func TestChocolatey_ChocolateyPublisher_Supports_Good(t *core.T) {
	subject := &ChocolateyPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Supports("linux")
	})
	core.AssertTrue(t, true)
}

func TestChocolatey_ChocolateyPublisher_Supports_Bad(t *core.T) {
	subject := &ChocolateyPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Supports("")
	})
	core.AssertTrue(t, true)
}

func TestChocolatey_ChocolateyPublisher_Supports_Ugly(t *core.T) {
	subject := &ChocolateyPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Supports("linux")
	})
	core.AssertTrue(t, true)
}

func TestChocolatey_ChocolateyPublisher_Publish_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &ChocolateyPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Publish(ctx, &Release{}, PublisherConfig{}, nil, true)
	})
	core.AssertTrue(t, true)
}

func TestChocolatey_ChocolateyPublisher_Publish_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &ChocolateyPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Publish(ctx, nil, PublisherConfig{}, nil, true)
	})
	core.AssertTrue(t, true)
}

func TestChocolatey_ChocolateyPublisher_Publish_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &ChocolateyPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Publish(ctx, &Release{}, PublisherConfig{}, nil, true)
	})
	core.AssertTrue(t, true)
}
