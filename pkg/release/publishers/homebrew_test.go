package publishers

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/io"
)

func TestHomebrew_HomebrewPublisherNameGood(t *testing.T) {
	t.Run("returns homebrew", func(t *testing.T) {
		p := NewHomebrewPublisher()
		if !stdlibAssertEqual("homebrew", p.Name()) {
			t.Fatalf("want %v, got %v", "homebrew", p.Name())
		}

	})
}

func TestHomebrew_HomebrewPublisherParseConfigGood(t *testing.T) {
	p := NewHomebrewPublisher()

	t.Run("uses defaults when no extended config", func(t *testing.T) {
		pubCfg := PublisherConfig{Type: "homebrew"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}
		cfg := p.parseConfig(pubCfg, relCfg)
		if !stdlibAssertEmpty(cfg.Tap) {
			t.Fatalf("expected empty, got %v", cfg.Tap)
		}
		if !stdlibAssertEmpty(cfg.Formula) {
			t.Fatalf("expected empty, got %v", cfg.Formula)
		}
		if !stdlibAssertNil(cfg.Official) {
			t.Fatalf("expected nil, got %v", cfg.Official)
		}

	})

	t.Run("parses tap and formula from extended config", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "homebrew",
			Extended: map[string]any{
				"tap":     "host-uk/homebrew-tap",
				"formula": "myformula",
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}
		cfg := p.parseConfig(pubCfg, relCfg)
		if !stdlibAssertEqual("host-uk/homebrew-tap", cfg.Tap) {
			t.Fatalf("want %v, got %v", "host-uk/homebrew-tap", cfg.Tap)
		}
		if !stdlibAssertEqual("myformula", cfg.Formula) {
			t.Fatalf("want %v, got %v", "myformula", cfg.Formula)
		}

	})

	t.Run("parses official config", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "homebrew",
			Extended: map[string]any{
				"official": map[string]any{
					"enabled": true,
					"output":  "dist/brew",
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
		if !stdlibAssertEqual("dist/brew", cfg.Official.Output) {
			t.Fatalf("want %v, got %v", "dist/brew", cfg.Official.Output)
		}

	})

	t.Run("handles missing official fields", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "homebrew",
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

func TestHomebrew_HomebrewPublisherToFormulaClassGood(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "core",
			expected: "Core",
		},
		{
			name:     "kebab case",
			input:    "my-cli-tool",
			expected: "MyCliTool",
		},
		{
			name:     "already capitalised",
			input:    "CLI",
			expected: "CLI",
		},
		{
			name:     "single letter",
			input:    "x",
			expected: "X",
		},
		{
			name:     "multiple dashes",
			input:    "my-super-cool-app",
			expected: "MySuperCoolApp",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := toFormulaClass(tc.input)
			if !stdlibAssertEqual(tc.expected, result) {
				t.Fatalf("want %v, got %v", tc.expected, result)
			}

		})
	}
}

func TestHomebrew_HomebrewPublisherBuildChecksumMapGood(t *testing.T) {
	t.Run("maps artifacts to checksums by platform", func(t *testing.T) {
		artifacts := []build.Artifact{
			{Path: "/dist/myapp-darwin-amd64.tar.gz", OS: "darwin", Arch: "amd64", Checksum: "abc123"},
			{Path: "/dist/myapp-darwin-arm64.tar.gz", OS: "darwin", Arch: "arm64", Checksum: "def456"},
			{Path: "/dist/myapp-linux-amd64.tar.gz", OS: "linux", Arch: "amd64", Checksum: "ghi789"},
			{Path: "/dist/myapp-linux-arm64.tar.gz", OS: "linux", Arch: "arm64", Checksum: "jkl012"},
			{Path: "/dist/myapp-windows-amd64.zip", OS: "windows", Arch: "amd64", Checksum: "mno345"},
			{Path: "/dist/myapp-windows-arm64.zip", OS: "windows", Arch: "arm64", Checksum: "pqr678"},
		}

		checksums := buildChecksumMap(artifacts)
		if !stdlibAssertEqual("abc123", checksums.DarwinAmd64) {
			t.Fatalf("want %v, got %v", "abc123", checksums.DarwinAmd64)
		}
		if !stdlibAssertEqual("def456", checksums.DarwinArm64) {
			t.Fatalf("want %v, got %v", "def456", checksums.DarwinArm64)
		}
		if !stdlibAssertEqual("ghi789", checksums.LinuxAmd64) {
			t.Fatalf("want %v, got %v", "ghi789", checksums.LinuxAmd64)
		}
		if !stdlibAssertEqual("jkl012", checksums.LinuxArm64) {
			t.Fatalf("want %v, got %v", "jkl012", checksums.LinuxArm64)
		}
		if !stdlibAssertEqual("mno345", checksums.WindowsAmd64) {
			t.Fatalf("want %v, got %v", "mno345", checksums.WindowsAmd64)
		}
		if !stdlibAssertEqual("pqr678", checksums.WindowsArm64) {
			t.Fatalf("want %v, got %v", "pqr678", checksums.WindowsArm64)
		}

	})

	t.Run("handles empty artifacts", func(t *testing.T) {
		checksums := buildChecksumMap([]build.Artifact{})
		if !stdlibAssertEmpty(checksums.DarwinAmd64) {
			t.Fatalf("expected empty, got %v", checksums.DarwinAmd64)
		}
		if !stdlibAssertEmpty(checksums.DarwinArm64) {
			t.Fatalf("expected empty, got %v", checksums.DarwinArm64)
		}
		if !stdlibAssertEmpty(checksums.LinuxAmd64) {
			t.Fatalf("expected empty, got %v", checksums.LinuxAmd64)
		}
		if !stdlibAssertEmpty(checksums.LinuxArm64) {
			t.Fatalf("expected empty, got %v", checksums.LinuxArm64)
		}

	})

	t.Run("handles partial platform coverage", func(t *testing.T) {
		artifacts := []build.Artifact{
			{Path: "/dist/myapp-darwin-arm64.tar.gz", Checksum: "def456"},
			{Path: "/dist/myapp-linux-amd64.tar.gz", Checksum: "ghi789"},
		}

		checksums := buildChecksumMap(artifacts)
		if !stdlibAssertEmpty(checksums.DarwinAmd64) {
			t.Fatalf("expected empty, got %v", checksums.DarwinAmd64)
		}
		if !stdlibAssertEqual("def456", checksums.DarwinArm64) {
			t.Fatalf("want %v, got %v", "def456", checksums.DarwinArm64)
		}
		if !stdlibAssertEqual("ghi789", checksums.LinuxAmd64) {
			t.Fatalf("want %v, got %v", "ghi789", checksums.LinuxAmd64)
		}
		if !stdlibAssertEmpty(checksums.LinuxArm64) {
			t.Fatalf("expected empty, got %v", checksums.LinuxArm64)
		}

	})
}

func TestHomebrew_HomebrewPublisherRenderTemplateGood(t *testing.T) {
	p := NewHomebrewPublisher()

	t.Run("renders formula template with data", func(t *testing.T) {
		data := homebrewTemplateData{
			FormulaClass: "MyApp",
			Description:  "My awesome CLI",
			Repository:   "owner/myapp",
			Version:      "1.2.3",
			License:      "MIT",
			BinaryName:   "myapp",
			Checksums: ChecksumMap{
				DarwinAmd64: "abc123",
				DarwinArm64: "def456",
				LinuxAmd64:  "ghi789",
				LinuxArm64:  "jkl012",
			},
		}

		result := requirePublisherString(t, p.renderTemplate(io.Local, "templates/homebrew/formula.rb.tmpl", data))
		if !stdlibAssertContains(result, "class MyApp < Formula") {
			t.Fatalf("expected %v to contain %v", result, "class MyApp < Formula")
		}
		if !stdlibAssertContains(result, `desc 'My awesome CLI'`) {
			t.Fatalf("expected %v to contain %v", result, `desc 'My awesome CLI'`)
		}
		if !stdlibAssertContains(result, `version '1.2.3'`) {
			t.Fatalf("expected %v to contain %v", result, `version '1.2.3'`)
		}
		if !stdlibAssertContains(result, `license 'MIT'`) {
			t.Fatalf("expected %v to contain %v", result, `license 'MIT'`)
		}
		if !stdlibAssertContains(result, "owner/myapp") {
			t.Fatalf("expected %v to contain %v", result, "owner/myapp")
		}
		if !stdlibAssertContains(result, "abc123") {
			t.Fatalf("expected %v to contain %v", result, "abc123")
		}
		if !stdlibAssertContains(result, "def456") {
			t.Fatalf("expected %v to contain %v", result, "def456")
		}
		if !stdlibAssertContains(result, "ghi789") {
			t.Fatalf("expected %v to contain %v", result, "ghi789")
		}
		if !stdlibAssertContains(result, "jkl012") {
			t.Fatalf("expected %v to contain %v", result, "jkl012")
		}
		if !stdlibAssertContains(result, `bin.install 'myapp'`) {
			t.Fatalf("expected %v to contain %v", result, `bin.install 'myapp'`)
		}

	})
}

func TestHomebrew_HomebrewPublisherRenderTemplateBad(t *testing.T) {
	p := NewHomebrewPublisher()

	t.Run("returns error for non-existent template", func(t *testing.T) {
		data := homebrewTemplateData{}
		err := requirePublisherError(t, p.renderTemplate(io.Local, "templates/homebrew/nonexistent.tmpl", data))
		if !stdlibAssertContains(err, "failed to read template") {
			t.Fatalf("expected %v to contain %v", err, "failed to read template")
		}

	})
}

func TestHomebrew_HomebrewPublisherDryRunPublishGood(t *testing.T) {
	p := NewHomebrewPublisher()

	t.Run("outputs expected dry run information", func(t *testing.T) {
		data := homebrewTemplateData{
			FormulaClass: "MyApp",
			Description:  "My CLI",
			Repository:   "owner/repo",
			Version:      "1.0.0",
			License:      "MIT",
			BinaryName:   "myapp",
			Checksums:    ChecksumMap{},
		}
		cfg := HomebrewConfig{
			Tap: "owner/homebrew-tap",
		}

		publishResult := core.Ok(nil)
		output := capturePublisherOutput(t, func() {
			publishResult = p.dryRunPublish(io.Local, data, cfg)
		})
		requirePublisherOK(t, publishResult)
		if !stdlibAssertContains(output, "DRY RUN: Homebrew Publish") {
			t.Fatalf("expected %v to contain %v", output, "DRY RUN: Homebrew Publish")
		}
		if !stdlibAssertContains(output, "Formula:    MyApp") {
			t.Fatalf("expected %v to contain %v", output, "Formula:    MyApp")
		}
		if !stdlibAssertContains(output, "Version:    1.0.0") {
			t.Fatalf("expected %v to contain %v", output, "Version:    1.0.0")
		}
		if !stdlibAssertContains(output, "Tap:        owner/homebrew-tap") {
			t.Fatalf("expected %v to contain %v", output, "Tap:        owner/homebrew-tap")
		}
		if !stdlibAssertContains(output, "Repository: owner/repo") {
			t.Fatalf("expected %v to contain %v", output, "Repository: owner/repo")
		}
		if !stdlibAssertContains(output, "Would commit to tap: owner/homebrew-tap") {
			t.Fatalf("expected %v to contain %v", output, "Would commit to tap: owner/homebrew-tap")
		}
		if !stdlibAssertContains(output, "END DRY RUN") {
			t.Fatalf("expected %v to contain %v", output, "END DRY RUN")
		}

	})

	t.Run("shows official output path when enabled", func(t *testing.T) {
		data := homebrewTemplateData{
			FormulaClass: "MyApp",
			Version:      "1.0.0",
			BinaryName:   "myapp",
			Checksums:    ChecksumMap{},
		}
		cfg := HomebrewConfig{
			Official: &OfficialConfig{
				Enabled: true,
				Output:  "custom/path",
			},
		}

		publishResult := core.Ok(nil)
		output := capturePublisherOutput(t, func() {
			publishResult = p.dryRunPublish(io.Local, data, cfg)
		})
		requirePublisherOK(t, publishResult)
		if !stdlibAssertContains(output, "Would write files for official PR to: custom/path") {
			t.Fatalf("expected %v to contain %v", output, "Would write files for official PR to: custom/path")
		}

	})

	t.Run("suppresses tap publish output in official mode", func(t *testing.T) {
		data := homebrewTemplateData{
			FormulaClass: "MyApp",
			Version:      "1.0.0",
			BinaryName:   "myapp",
			Checksums:    ChecksumMap{},
		}
		cfg := HomebrewConfig{
			Tap: "owner/homebrew-tap",
			Official: &OfficialConfig{
				Enabled: true,
				Output:  "custom/path",
			},
		}

		publishResult := core.Ok(nil)
		output := capturePublisherOutput(t, func() {
			publishResult = p.dryRunPublish(io.Local, data, cfg)
		})
		requirePublisherOK(t, publishResult)
		if !stdlibAssertContains(output, "Would write files for official PR to: custom/path") {
			t.Fatalf("expected %v to contain %v", output, "Would write files for official PR to: custom/path")
		}
		if stdlibAssertContains(output, "Would commit to tap:") {
			t.Fatalf("expected %v not to contain %v", output, "Would commit to tap:")
		}

	})

	t.Run("uses default official output path when not specified", func(t *testing.T) {
		data := homebrewTemplateData{
			FormulaClass: "MyApp",
			Version:      "1.0.0",
			BinaryName:   "myapp",
			Checksums:    ChecksumMap{},
		}
		cfg := HomebrewConfig{
			Official: &OfficialConfig{
				Enabled: true,
			},
		}

		publishResult := core.Ok(nil)
		output := capturePublisherOutput(t, func() {
			publishResult = p.dryRunPublish(io.Local, data, cfg)
		})
		requirePublisherOK(t, publishResult)
		if !stdlibAssertContains(output, "Would write files for official PR to: dist/homebrew") {
			t.Fatalf("expected %v to contain %v", output, "Would write files for official PR to: dist/homebrew")
		}

	})
}

func TestHomebrew_HomebrewPublisherPublishBad(t *testing.T) {
	p := NewHomebrewPublisher()

	t.Run("fails when tap not configured and not official mode", func(t *testing.T) {
		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: "/project",
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{Type: "homebrew"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		err := requirePublisherError(t, p.Publish(context.TODO(), release, pubCfg, relCfg, false))
		if !stdlibAssertContains(err, "tap is required") {
			t.Fatalf("expected %v to contain %v", err, "tap is required")
		}

	})

	t.Run("official mode writes files without requiring tap publish tooling", func(t *testing.T) {
		projectDir := t.TempDir()
		t.Setenv("PATH", "/definitely-missing")

		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: projectDir,
			FS:         io.Local,
			Artifacts: []build.Artifact{
				{Path: "dist/myapp-linux-amd64.tar.gz", OS: "linux", Arch: "amd64", Checksum: "abc123"},
			},
		}
		pubCfg := PublisherConfig{
			Type: "homebrew",
			Extended: map[string]any{
				"tap": "owner/homebrew-tap",
				"official": map[string]any{
					"enabled": true,
					"output":  "dist/homebrew-pr",
				},
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo", projectName: "myapp"}

		requirePublisherOK(t, p.Publish(context.TODO(), release, pubCfg, relCfg, false))
		requirePublisherOK(t, ax.Stat(ax.Join(projectDir, "dist", "homebrew-pr", "myapp.rb")))

	})
}

func TestHomebrew_HomebrewConfigDefaultsGood(t *testing.T) {
	t.Run("has sensible defaults", func(t *testing.T) {
		p := NewHomebrewPublisher()
		pubCfg := PublisherConfig{Type: "homebrew"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		cfg := p.parseConfig(pubCfg, relCfg)
		if !stdlibAssertEmpty(cfg.Tap) {
			t.Fatalf("expected empty, got %v", cfg.Tap)
		}
		if !stdlibAssertEmpty(cfg.Formula) {
			t.Fatalf("expected empty, got %v", cfg.Formula)
		}
		if !stdlibAssertNil(cfg.Official) {
			t.Fatalf("expected nil, got %v", cfg.Official)
		}

	})
}

// --- v0.9.0 generated compliance triplets ---
func TestHomebrew_NewHomebrewPublisher_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewHomebrewPublisher()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestHomebrew_NewHomebrewPublisher_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewHomebrewPublisher()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestHomebrew_NewHomebrewPublisher_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewHomebrewPublisher()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestHomebrew_HomebrewPublisher_Name_Good(t *core.T) {
	subject := &HomebrewPublisher{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestHomebrew_HomebrewPublisher_Name_Bad(t *core.T) {
	subject := &HomebrewPublisher{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestHomebrew_HomebrewPublisher_Name_Ugly(t *core.T) {
	subject := &HomebrewPublisher{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestHomebrew_HomebrewPublisher_Validate_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &HomebrewPublisher{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Validate(ctx, &Release{}, PublisherConfig{}, nil)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestHomebrew_HomebrewPublisher_Validate_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &HomebrewPublisher{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Validate(ctx, nil, PublisherConfig{}, nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestHomebrew_HomebrewPublisher_Validate_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &HomebrewPublisher{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Validate(ctx, &Release{}, PublisherConfig{}, nil)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestHomebrew_HomebrewPublisher_Supports_Good(t *core.T) {
	subject := &HomebrewPublisher{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Supports("linux")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestHomebrew_HomebrewPublisher_Supports_Bad(t *core.T) {
	subject := &HomebrewPublisher{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Supports("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestHomebrew_HomebrewPublisher_Supports_Ugly(t *core.T) {
	subject := &HomebrewPublisher{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Supports("linux")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestHomebrew_HomebrewPublisher_Publish_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &HomebrewPublisher{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Publish(ctx, &Release{}, PublisherConfig{}, nil, true)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestHomebrew_HomebrewPublisher_Publish_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &HomebrewPublisher{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Publish(ctx, nil, PublisherConfig{}, nil, true)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestHomebrew_HomebrewPublisher_Publish_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &HomebrewPublisher{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Publish(ctx, &Release{}, PublisherConfig{}, nil, true)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
