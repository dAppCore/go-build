package publishers

import (
	"context"
	"testing"

	core "dappco.re/go"
	storage "dappco.re/go/build/pkg/storage"
)

func TestNpm_NpmPublisherNameGood(t *testing.T) {
	t.Run("returns npm", func(t *testing.T) {
		p := NewNpmPublisher()
		if !stdlibAssertEqual("npm", p.Name()) {
			t.Fatalf("want %v, got %v", "npm", p.Name())
		}

	})
}

func TestNpm_NpmPublisherParseConfigGood(t *testing.T) {
	p := NewNpmPublisher()

	t.Run("uses defaults when no extended config", func(t *testing.T) {
		pubCfg := PublisherConfig{Type: "npm"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}
		cfg := p.parseConfig(pubCfg, relCfg)
		if !stdlibAssertEmpty(cfg.Package) {
			t.Fatalf("expected empty, got %v", cfg.Package)
		}
		if !stdlibAssertEqual("public", cfg.Access) {
			t.Fatalf("want %v, got %v", "public", cfg.Access)
		}

	})

	t.Run("parses package and access from extended config", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "npm",
			Extended: map[string]any{
				"package": "@myorg/mypackage",
				"access":  "restricted",
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}
		cfg := p.parseConfig(pubCfg, relCfg)
		if !stdlibAssertEqual("@myorg/mypackage", cfg.Package) {
			t.Fatalf("want %v, got %v", "@myorg/mypackage", cfg.Package)
		}
		if !stdlibAssertEqual("restricted", cfg.Access) {
			t.Fatalf("want %v, got %v", "restricted", cfg.Access)
		}

	})

	t.Run("keeps default access when not specified", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "npm",
			Extended: map[string]any{
				"package": "@myorg/mypackage",
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}
		cfg := p.parseConfig(pubCfg, relCfg)
		if !stdlibAssertEqual("@myorg/mypackage", cfg.Package) {
			t.Fatalf("want %v, got %v", "@myorg/mypackage", cfg.Package)
		}
		if !stdlibAssertEqual("public", cfg.Access) {
			t.Fatalf("want %v, got %v", "public", cfg.Access)
		}

	})

	t.Run("handles nil extended config", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type:     "npm",
			Extended: nil,
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}
		cfg := p.parseConfig(pubCfg, relCfg)
		if !stdlibAssertEmpty(cfg.Package) {
			t.Fatalf("expected empty, got %v", cfg.Package)
		}
		if !stdlibAssertEqual("public", cfg.Access) {
			t.Fatalf("want %v, got %v", "public", cfg.Access)
		}

	})

	t.Run("handles empty strings in config", func(t *testing.T) {
		pubCfg := PublisherConfig{
			Type: "npm",
			Extended: map[string]any{
				"package": "",
				"access":  "",
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}
		cfg := p.parseConfig(pubCfg, relCfg)
		if !stdlibAssertEmpty(cfg.Package) {
			t.Fatalf("expected empty, got %v", cfg.Package)
		}
		if !stdlibAssertEqual("public", cfg.Access) {
			t.Fatalf("want %v, got %v", "public", cfg.Access)
		}

	})
}

func TestNpm_NpmPublisherRenderTemplateGood(t *testing.T) {
	p := NewNpmPublisher()

	t.Run("renders package.json template with data", func(t *testing.T) {
		data := npmTemplateData{
			Package:     "@myorg/mycli",
			Version:     "1.2.3",
			Description: "My awesome CLI",
			License:     "MIT",
			Repository:  "owner/myapp",
			BinaryName:  "myapp",
			ProjectName: "myapp",
			Access:      "public",
		}

		result := requirePublisherString(t, p.renderTemplate(storage.Local, "templates/npm/package.json.tmpl", data))
		if !stdlibAssertContains(result, `"name": "@myorg/mycli"`) {
			t.Fatalf("expected %v to contain %v", result, `"name": "@myorg/mycli"`)
		}
		if !stdlibAssertContains(result, `"version": "1.2.3"`) {
			t.Fatalf("expected %v to contain %v", result, `"version": "1.2.3"`)
		}
		if !stdlibAssertContains(result, `"description": "My awesome CLI"`) {
			t.Fatalf("expected %v to contain %v", result, `"description": "My awesome CLI"`)
		}
		if !stdlibAssertContains(result, `"license": "MIT"`) {
			t.Fatalf("expected %v to contain %v", result, `"license": "MIT"`)
		}
		if !stdlibAssertContains(result, "owner/myapp") {
			t.Fatalf("expected %v to contain %v", result, "owner/myapp")
		}
		if !stdlibAssertContains(result, `"myapp": "./bin/run.js"`) {
			t.Fatalf("expected %v to contain %v", result, `"myapp": "./bin/run.js"`)
		}
		if !stdlibAssertContains(result, `"access": "public"`) {
			t.Fatalf("expected %v to contain %v", result, `"access": "public"`)
		}

	})

	t.Run("renders restricted access correctly", func(t *testing.T) {
		data := npmTemplateData{
			Package:     "@private/cli",
			Version:     "1.0.0",
			Description: "Private CLI",
			License:     "MIT",
			Repository:  "org/repo",
			BinaryName:  "cli",
			ProjectName: "cli",
			Access:      "restricted",
		}

		result := requirePublisherString(t, p.renderTemplate(storage.Local, "templates/npm/package.json.tmpl", data))
		if !stdlibAssertContains(result, `"access": "restricted"`) {
			t.Fatalf("expected %v to contain %v", result, `"access": "restricted"`)
		}

	})

	t.Run("renders install.js with resolved release asset names", func(t *testing.T) {
		data := npmTemplateData{
			Package:     "@myorg/mycli",
			Version:     "1.2.3",
			Description: "My awesome CLI",
			License:     "MIT",
			Repository:  "owner/myapp",
			BinaryName:  "myapp",
			ProjectName: "myapp",
			Access:      "public",
			Checksums: ChecksumMap{
				LinuxAmd64:       "abc123",
				LinuxAmd64File:   "myapp_linux_amd64.tar.gz",
				WindowsAmd64:     "def456",
				WindowsAmd64File: "myapp_windows_amd64.zip",
				ChecksumFile:     "CHECKSUMS.txt",
			},
		}

		result := requirePublisherString(t, p.renderTemplate(storage.Local, "templates/npm/install.js.tmpl", data))
		if !stdlibAssertContains(result, `const CHECKSUM_FILE = "CHECKSUMS.txt";`) {
			t.Fatalf("expected %v to contain %v", result, `const CHECKSUM_FILE = "CHECKSUMS.txt";`)
		}
		if !stdlibAssertContains(result, `myapp_linux_amd64.tar.gz`) {
			t.Fatalf("expected %v to contain %v", result, `myapp_linux_amd64.tar.gz`)
		}
		if !stdlibAssertContains(result, `myapp_windows_amd64.zip`) {
			t.Fatalf("expected %v to contain %v", result, `myapp_windows_amd64.zip`)
		}
		if stdlibAssertContains(result, `/checksums.txt`) {
			t.Fatalf("expected %v not to contain %v", result, `/checksums.txt`)
		}

	})
}

func TestNpm_NpmPublisherRenderTemplateBad(t *testing.T) {
	p := NewNpmPublisher()

	t.Run("returns error for non-existent template", func(t *testing.T) {
		data := npmTemplateData{}
		err := requirePublisherError(t, p.renderTemplate(storage.Local, "templates/npm/nonexistent.tmpl", data))
		if !stdlibAssertContains(err, "failed to read template") {
			t.Fatalf("expected %v to contain %v", err, "failed to read template")
		}

	})
}

func TestNpm_NpmPublisherDryRunPublishGood(t *testing.T) {
	p := NewNpmPublisher()

	t.Run("outputs expected dry run information", func(t *testing.T) {
		data := npmTemplateData{
			Package:     "@myorg/mycli",
			Version:     "1.0.0",
			Access:      "public",
			Repository:  "owner/repo",
			BinaryName:  "mycli",
			Description: "My CLI",
		}
		publishResult := core.Ok(nil)
		output := capturePublisherOutput(t, func() {
			publishResult = p.dryRunPublish(storage.Local, data)
		})
		requirePublisherOK(t, publishResult)
		if !stdlibAssertContains(output, "DRY RUN: npm Publish") {
			t.Fatalf("expected %v to contain %v", output, "DRY RUN: npm Publish")
		}
		if !stdlibAssertContains(output, "Package:    @myorg/mycli") {
			t.Fatalf("expected %v to contain %v", output, "Package:    @myorg/mycli")
		}
		if !stdlibAssertContains(output, "Version:    1.0.0") {
			t.Fatalf("expected %v to contain %v", output, "Version:    1.0.0")
		}
		if !stdlibAssertContains(output, "Access:     public") {
			t.Fatalf("expected %v to contain %v", output, "Access:     public")
		}
		if !stdlibAssertContains(output, "Repository: owner/repo") {
			t.Fatalf("expected %v to contain %v", output, "Repository: owner/repo")
		}
		if !stdlibAssertContains(output, "Binary:     mycli") {
			t.Fatalf("expected %v to contain %v", output, "Binary:     mycli")
		}
		if !stdlibAssertContains(output, "Generated package.json:") {
			t.Fatalf("expected %v to contain %v", output, "Generated package.json:")
		}
		if !stdlibAssertContains(output, "Would run: npm publish --access public") {
			t.Fatalf("expected %v to contain %v", output, "Would run: npm publish --access public")
		}
		if !stdlibAssertContains(output, "END DRY RUN") {
			t.Fatalf("expected %v to contain %v", output, "END DRY RUN")
		}

	})

	t.Run("shows restricted access correctly", func(t *testing.T) {
		data := npmTemplateData{
			Package:    "@private/cli",
			Version:    "2.0.0",
			Access:     "restricted",
			Repository: "org/repo",
			BinaryName: "cli",
		}

		publishResult := core.Ok(nil)
		output := capturePublisherOutput(t, func() {
			publishResult = p.dryRunPublish(storage.Local, data)
		})
		requirePublisherOK(t, publishResult)
		if !stdlibAssertContains(output, "Access:     restricted") {
			t.Fatalf("expected %v to contain %v", output, "Access:     restricted")
		}
		if !stdlibAssertContains(output, "Would run: npm publish --access restricted") {
			t.Fatalf("expected %v to contain %v", output, "Would run: npm publish --access restricted")
		}

	})
}

func TestNpm_NpmPublisherPublishBad(t *testing.T) {
	p := NewNpmPublisher()

	t.Run("fails when package name not configured", func(t *testing.T) {
		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: "/project",
			FS:         storage.Local,
		}
		pubCfg := PublisherConfig{Type: "npm"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		err := requirePublisherError(t, p.Publish(context.TODO(), release, pubCfg, relCfg, false))
		if !stdlibAssertContains(err, "package name is required") {
			t.Fatalf("expected %v to contain %v", err, "package name is required")
		}

	})

	t.Run("fails when NPM_TOKEN not set in non-dry-run", func(t *testing.T) {
		t.Setenv("NPM_TOKEN", "")

		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: "/project",
			FS:         storage.Local,
		}
		pubCfg := PublisherConfig{
			Type: "npm",
			Extended: map[string]any{
				"package": "@test/package",
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		err := requirePublisherError(t, p.Publish(context.TODO(), release, pubCfg, relCfg, false))
		if !stdlibAssertContains(err, "NPM_TOKEN environment variable is required") {
			t.Fatalf("expected %v to contain %v", err, "NPM_TOKEN environment variable is required")
		}

	})
}

func TestNpm_NpmConfigDefaultsGood(t *testing.T) {
	t.Run("has sensible defaults", func(t *testing.T) {
		p := NewNpmPublisher()
		pubCfg := PublisherConfig{Type: "npm"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		cfg := p.parseConfig(pubCfg, relCfg)
		if !stdlibAssertEmpty(cfg.Package) {
			t.Fatalf("expected empty, got %v", cfg.Package)
		}
		if !stdlibAssertEqual("public", cfg.Access) {
			t.Fatalf("want %v, got %v", "public", cfg.Access)
		}

	})
}

func TestNpm_NpmTemplateDataGood(t *testing.T) {
	t.Run("struct has all expected fields", func(t *testing.T) {
		data := npmTemplateData{
			Package:     "@myorg/package",
			Version:     "1.0.0",
			Description: "description",
			License:     "MIT",
			Repository:  "org/repo",
			BinaryName:  "cli",
			ProjectName: "cli",
			Access:      "public",
		}
		if !stdlibAssertEqual("@myorg/package", data.Package) {
			t.Fatalf("want %v, got %v", "@myorg/package", data.Package)
		}
		if !stdlibAssertEqual("1.0.0", data.Version) {
			t.Fatalf("want %v, got %v", "1.0.0", data.Version)
		}
		if !stdlibAssertEqual("description", data.Description) {
			t.Fatalf("want %v, got %v", "description", data.Description)
		}
		if !stdlibAssertEqual("MIT", data.License) {
			t.Fatalf("want %v, got %v", "MIT", data.License)
		}
		if !stdlibAssertEqual("org/repo", data.Repository) {
			t.Fatalf("want %v, got %v", "org/repo", data.Repository)
		}
		if !stdlibAssertEqual("cli", data.BinaryName) {
			t.Fatalf("want %v, got %v", "cli", data.BinaryName)
		}
		if !stdlibAssertEqual("cli", data.ProjectName) {
			t.Fatalf("want %v, got %v", "cli", data.ProjectName)
		}
		if !stdlibAssertEqual("public", data.Access) {
			t.Fatalf("want %v, got %v", "public", data.Access)
		}

	})
}

// --- v0.9.0 generated compliance triplets ---
func TestNpm_NewNpmPublisher_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewNpmPublisher()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestNpm_NewNpmPublisher_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewNpmPublisher()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestNpm_NewNpmPublisher_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewNpmPublisher()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestNpm_NpmPublisher_Name_Good(t *core.T) {
	subject := &NpmPublisher{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestNpm_NpmPublisher_Name_Bad(t *core.T) {
	subject := &NpmPublisher{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestNpm_NpmPublisher_Name_Ugly(t *core.T) {
	subject := &NpmPublisher{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestNpm_NpmPublisher_Validate_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &NpmPublisher{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Validate(ctx, &Release{}, PublisherConfig{}, nil)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestNpm_NpmPublisher_Validate_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &NpmPublisher{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Validate(ctx, nil, PublisherConfig{}, nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestNpm_NpmPublisher_Validate_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &NpmPublisher{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Validate(ctx, &Release{}, PublisherConfig{}, nil)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestNpm_NpmPublisher_Supports_Good(t *core.T) {
	subject := &NpmPublisher{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Supports("linux")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestNpm_NpmPublisher_Supports_Bad(t *core.T) {
	subject := &NpmPublisher{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Supports("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestNpm_NpmPublisher_Supports_Ugly(t *core.T) {
	subject := &NpmPublisher{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Supports("linux")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestNpm_NpmPublisher_Publish_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &NpmPublisher{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Publish(ctx, &Release{}, PublisherConfig{}, nil, true)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestNpm_NpmPublisher_Publish_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &NpmPublisher{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Publish(ctx, nil, PublisherConfig{}, nil, true)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestNpm_NpmPublisher_Publish_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &NpmPublisher{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Publish(ctx, &Release{}, PublisherConfig{}, nil, true)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
