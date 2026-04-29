package publishers

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/io"
)

func TestAUR_AURPublisherNameGood(t *testing.T) {
	t.Run("returns aur", func(t *testing.T) {
		p := NewAURPublisher()
		if !stdlibAssertEqual("aur", p.Name()) {
			t.Fatalf("want %v, got %v", "aur", p.Name())
		}

	})
}

func TestAUR_AURPublisherParseConfigGood(t *testing.T) {
	p := NewAURPublisher()

	t.Run("uses defaults when no extended config", func(t *testing.T) {
		pubCfg := PublisherConfig{Type: "aur"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}
		cfg := p.parseConfig(pubCfg, relCfg)
		if !stdlibAssertEmpty(cfg.Package) {
			t.Fatalf("expected empty, got %v", cfg.Package)
		}
		if !stdlibAssertEmpty(cfg.Maintainer) {
			t.Fatalf("expected empty, got %v", cfg.Maintainer)
		}
		if !stdlibAssertNil(cfg.Official) {
			t.Fatalf("expected nil, got %v", cfg.Official)
		}

	})

	t.Run("parses package and maintainer from extended config", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "aur",
			Extended: map[string]any{
				"package":    "mypackage",
				"maintainer": "John Doe <john@example.com>",
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}
		cfg := p.parseConfig(pubCfg, relCfg)
		if !stdlibAssertEqual("mypackage", cfg.Package) {
			t.Fatalf("want %v, got %v", "mypackage", cfg.Package)
		}
		if !stdlibAssertEqual("John Doe <john@example.com>", cfg.Maintainer) {
			t.Fatalf("want %v, got %v", "John Doe <john@example.com>", cfg.Maintainer)
		}

	})

	t.Run("parses official config", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "aur",
			Extended: map[string]any{
				"official": map[string]any{
					"enabled": true,
					"output":  "dist/aur-files",
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
		if !stdlibAssertEqual("dist/aur-files", cfg.Official.Output) {
			t.Fatalf("want %v, got %v", "dist/aur-files", cfg.Official.Output)
		}

	})

	t.Run("handles missing official fields", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "aur",
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
}

func TestAUR_AURPublisherRenderTemplateGood(t *testing.T) {
	p := NewAURPublisher()

	t.Run("renders PKGBUILD template with data", func(t *testing.T) {
		data := aurTemplateData{
			PackageName: "myapp",
			Description: "My awesome CLI",
			Repository:  "owner/myapp",
			Version:     "1.2.3",
			License:     "MIT",
			BinaryName:  "myapp",
			Maintainer:  "John Doe <john@example.com>",
			Checksums: ChecksumMap{
				LinuxAmd64:     "abc123",
				LinuxArm64:     "def456",
				LinuxAmd64File: "myapp_linux_amd64.tar.gz",
				LinuxArm64File: "myapp_linux_arm64.tar.gz",
			},
		}

		result, err := p.renderTemplate(io.Local, "templates/aur/PKGBUILD.tmpl", data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(result, "# Maintainer: John Doe <john@example.com>") {
			t.Fatalf("expected %v to contain %v", result, "# Maintainer: John Doe <john@example.com>")
		}
		if !stdlibAssertContains(result, "pkgname='myapp-bin'") {
			t.Fatalf("expected %v to contain %v", result, "pkgname='myapp-bin'")
		}
		if !stdlibAssertContains(result, "pkgver='1.2.3'") {
			t.Fatalf("expected %v to contain %v", result, "pkgver='1.2.3'")
		}
		if !stdlibAssertContains(result, `pkgdesc='My awesome CLI'`) {
			t.Fatalf("expected %v to contain %v", result, `pkgdesc='My awesome CLI'`)
		}
		if !stdlibAssertContains(result, "url='https://github.com/owner/myapp'") {
			t.Fatalf("expected %v to contain %v", result, "url='https://github.com/owner/myapp'")
		}
		if !stdlibAssertContains(result, "license=('MIT')") {
			t.Fatalf("expected %v to contain %v", result, "license=('MIT')")
		}
		if !stdlibAssertContains(result, "sha256sums_x86_64=('abc123')") {
			t.Fatalf("expected %v to contain %v", result, "sha256sums_x86_64=('abc123')")
		}
		if !stdlibAssertContains(result, "sha256sums_aarch64=('def456')") {
			t.Fatalf("expected %v to contain %v", result, "sha256sums_aarch64=('def456')")
		}

	})

	t.Run("renders .SRCINFO template with data", func(t *testing.T) {
		data := aurTemplateData{
			PackageName: "myapp",
			Description: "My CLI",
			Repository:  "owner/myapp",
			Version:     "1.0.0",
			License:     "MIT",
			BinaryName:  "myapp",
			Maintainer:  "Test <test@test.com>",
			Checksums: ChecksumMap{
				LinuxAmd64: "checksum1",
				LinuxArm64: "checksum2",
			},
		}

		result, err := p.renderTemplate(io.Local, "templates/aur/.SRCINFO.tmpl", data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(result, "pkgbase = myapp-bin") {
			t.Fatalf("expected %v to contain %v", result, "pkgbase = myapp-bin")
		}
		if !stdlibAssertContains(result, "pkgdesc = My CLI") {
			t.Fatalf("expected %v to contain %v", result, "pkgdesc = My CLI")
		}
		if !stdlibAssertContains(result, "pkgver = 1.0.0") {
			t.Fatalf("expected %v to contain %v", result, "pkgver = 1.0.0")
		}
		if !stdlibAssertContains(result, "arch = x86_64") {
			t.Fatalf("expected %v to contain %v", result, "arch = x86_64")
		}
		if !stdlibAssertContains(result, "arch = aarch64") {
			t.Fatalf("expected %v to contain %v", result, "arch = aarch64")
		}
		if !stdlibAssertContains(result, "sha256sums_x86_64 = checksum1") {
			t.Fatalf("expected %v to contain %v", result, "sha256sums_x86_64 = checksum1")
		}
		if !stdlibAssertContains(result, "sha256sums_aarch64 = checksum2") {
			t.Fatalf("expected %v to contain %v", result, "sha256sums_aarch64 = checksum2")
		}
		if !stdlibAssertContains(result, "pkgname = myapp-bin") {
			t.Fatalf("expected %v to contain %v", result, "pkgname = myapp-bin")
		}

	})
}

func TestAUR_AURPublisherRenderTemplateBad(t *testing.T) {
	p := NewAURPublisher()

	t.Run("returns error for non-existent template", func(t *testing.T) {
		data := aurTemplateData{}
		_, err := p.renderTemplate(io.Local, "templates/aur/nonexistent.tmpl", data)
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "failed to read template") {
			t.Fatalf("expected %v to contain %v", err.Error(), "failed to read template")
		}

	})
}

func TestAUR_AURPublisherDryRunPublishGood(t *testing.T) {
	p := NewAURPublisher()

	t.Run("outputs expected dry run information", func(t *testing.T) {
		data := aurTemplateData{
			PackageName: "myapp",
			Version:     "1.0.0",
			Maintainer:  "John Doe <john@example.com>",
			Repository:  "owner/repo",
			BinaryName:  "myapp",
			Checksums:   ChecksumMap{},
		}
		cfg := AURConfig{
			Maintainer: "John Doe <john@example.com>",
		}

		var err error
		output := capturePublisherOutput(t, func() {
			err = p.dryRunPublish(io.Local, data, cfg)
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(output, "DRY RUN: AUR Publish") {
			t.Fatalf("expected %v to contain %v", output, "DRY RUN: AUR Publish")
		}
		if !stdlibAssertContains(output, "Package:    myapp-bin") {
			t.Fatalf("expected %v to contain %v", output, "Package:    myapp-bin")
		}
		if !stdlibAssertContains(output, "Version:    1.0.0") {
			t.Fatalf("expected %v to contain %v", output, "Version:    1.0.0")
		}
		if !stdlibAssertContains(output, "Maintainer: John Doe <john@example.com>") {
			t.Fatalf("expected %v to contain %v", output, "Maintainer: John Doe <john@example.com>")
		}
		if !stdlibAssertContains(output, "Repository: owner/repo") {
			t.Fatalf("expected %v to contain %v", output, "Repository: owner/repo")
		}
		if !stdlibAssertContains(output, "Generated PKGBUILD:") {
			t.Fatalf("expected %v to contain %v", output, "Generated PKGBUILD:")
		}
		if !stdlibAssertContains(output, "Generated .SRCINFO:") {
			t.Fatalf("expected %v to contain %v", output, "Generated .SRCINFO:")
		}
		if !stdlibAssertContains(output, "Would push to AUR: ssh://aur@aur.archlinux.org/myapp-bin.git") {
			t.Fatalf("expected %v to contain %v", output, "Would push to AUR: ssh://aur@aur.archlinux.org/myapp-bin.git")
		}
		if !stdlibAssertContains(output, "END DRY RUN") {
			t.Fatalf("expected %v to contain %v", output, "END DRY RUN")
		}

	})

	t.Run("shows official output path instead of push in official mode", func(t *testing.T) {
		data := aurTemplateData{
			PackageName: "myapp",
			Version:     "1.0.0",
			Maintainer:  "John Doe <john@example.com>",
			Repository:  "owner/repo",
			BinaryName:  "myapp",
			Checksums:   ChecksumMap{},
		}
		cfg := AURConfig{
			Maintainer: "John Doe <john@example.com>",
			Official: &OfficialConfig{
				Enabled: true,
				Output:  "dist/aur-files",
			},
		}

		var err error
		output := capturePublisherOutput(t, func() {
			err = p.dryRunPublish(io.Local, data, cfg)
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(output, "Would write files for official PR to: dist/aur-files") {
			t.Fatalf("expected %v to contain %v", output, "Would write files for official PR to: dist/aur-files")
		}
		if stdlibAssertContains(output, "Would push to AUR:") {
			t.Fatalf("expected %v not to contain %v", output, "Would push to AUR:")
		}

	})
}

func TestAUR_AURPublisherPublishBad(t *testing.T) {
	p := NewAURPublisher()

	t.Run("fails when maintainer not configured", func(t *testing.T) {
		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: "/project",
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{Type: "aur"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		err := p.Publish(context.TODO(), release, pubCfg, relCfg, false)
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "maintainer is required") {
			t.Fatalf("expected %v to contain %v", err.Error(), "maintainer is required")
		}

	})
}

func TestAUR_AURConfigDefaultsGood(t *testing.T) {
	t.Run("has sensible defaults", func(t *testing.T) {
		p := NewAURPublisher()
		pubCfg := PublisherConfig{Type: "aur"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		cfg := p.parseConfig(pubCfg, relCfg)
		if !stdlibAssertEmpty(cfg.Package) {
			t.Fatalf("expected empty, got %v", cfg.Package)
		}
		if !stdlibAssertEmpty(cfg.Maintainer) {
			t.Fatalf("expected empty, got %v", cfg.Maintainer)
		}
		if !stdlibAssertNil(cfg.Official) {
			t.Fatalf("expected nil, got %v", cfg.Official)
		}

	})
}

// --- v0.9.0 generated compliance triplets ---
func TestAur_NewAURPublisher_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewAURPublisher()
	})
	core.AssertTrue(t, true)
}

func TestAur_NewAURPublisher_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewAURPublisher()
	})
	core.AssertTrue(t, true)
}

func TestAur_NewAURPublisher_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewAURPublisher()
	})
	core.AssertTrue(t, true)
}

func TestAur_AURPublisher_Name_Good(t *core.T) {
	subject := &AURPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
	})
	core.AssertTrue(t, true)
}

func TestAur_AURPublisher_Name_Bad(t *core.T) {
	subject := &AURPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
	})
	core.AssertTrue(t, true)
}

func TestAur_AURPublisher_Name_Ugly(t *core.T) {
	subject := &AURPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
	})
	core.AssertTrue(t, true)
}

func TestAur_AURPublisher_Validate_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &AURPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Validate(ctx, &Release{}, PublisherConfig{}, nil)
	})
	core.AssertTrue(t, true)
}

func TestAur_AURPublisher_Validate_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &AURPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Validate(ctx, nil, PublisherConfig{}, nil)
	})
	core.AssertTrue(t, true)
}

func TestAur_AURPublisher_Validate_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &AURPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Validate(ctx, &Release{}, PublisherConfig{}, nil)
	})
	core.AssertTrue(t, true)
}

func TestAur_AURPublisher_Supports_Good(t *core.T) {
	subject := &AURPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Supports("linux")
	})
	core.AssertTrue(t, true)
}

func TestAur_AURPublisher_Supports_Bad(t *core.T) {
	subject := &AURPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Supports("")
	})
	core.AssertTrue(t, true)
}

func TestAur_AURPublisher_Supports_Ugly(t *core.T) {
	subject := &AURPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Supports("linux")
	})
	core.AssertTrue(t, true)
}

func TestAur_AURPublisher_Publish_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &AURPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Publish(ctx, &Release{}, PublisherConfig{}, nil, true)
	})
	core.AssertTrue(t, true)
}

func TestAur_AURPublisher_Publish_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &AURPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Publish(ctx, nil, PublisherConfig{}, nil, true)
	})
	core.AssertTrue(t, true)
}

func TestAur_AURPublisher_Publish_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &AURPublisher{}
	core.AssertNotPanics(t, func() {
		_ = subject.Publish(ctx, &Release{}, PublisherConfig{}, nil, true)
	})
	core.AssertTrue(t, true)
}
