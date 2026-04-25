package publishers

import (
	"context"
	"testing"

	"dappco.re/go/io"
)

func TestNpm_NpmPublisherName_Good(t *testing.T) {
	t.Run("returns npm", func(t *testing.T) {
		p := NewNpmPublisher()
		if !stdlibAssertEqual("npm", p.Name()) {
			t.Fatalf("want %v, got %v", "npm", p.Name())
		}

	})
}

func TestNpm_NpmPublisherParseConfig_Good(t *testing.T) {
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

func TestNpm_NpmPublisherRenderTemplate_Good(t *testing.T) {
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

		result, err := p.renderTemplate(io.Local, "templates/npm/package.json.tmpl", data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
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

		result, err := p.renderTemplate(io.Local, "templates/npm/package.json.tmpl", data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
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

		result, err := p.renderTemplate(io.Local, "templates/npm/install.js.tmpl", data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
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

func TestNpm_NpmPublisherRenderTemplate_Bad(t *testing.T) {
	p := NewNpmPublisher()

	t.Run("returns error for non-existent template", func(t *testing.T) {
		data := npmTemplateData{}
		_, err := p.renderTemplate(io.Local, "templates/npm/nonexistent.tmpl", data)
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "failed to read template") {
			t.Fatalf("expected %v to contain %v", err.Error(), "failed to read template")
		}

	})
}

func TestNpm_NpmPublisherDryRunPublish_Good(t *testing.T) {
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
		var err error
		output := capturePublisherOutput(t, func() {
			err = p.dryRunPublish(io.Local, data)
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
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

		var err error
		output := capturePublisherOutput(t, func() {
			err = p.dryRunPublish(io.Local, data)
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(output, "Access:     restricted") {
			t.Fatalf("expected %v to contain %v", output, "Access:     restricted")
		}
		if !stdlibAssertContains(output, "Would run: npm publish --access restricted") {
			t.Fatalf("expected %v to contain %v", output, "Would run: npm publish --access restricted")
		}

	})
}

func TestNpm_NpmPublisherPublish_Bad(t *testing.T) {
	p := NewNpmPublisher()

	t.Run("fails when package name not configured", func(t *testing.T) {
		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: "/project",
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{Type: "npm"}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		err := p.Publish(context.TODO(), release, pubCfg, relCfg, false)
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "package name is required") {
			t.Fatalf("expected %v to contain %v", err.Error(), "package name is required")
		}

	})

	t.Run("fails when NPM_TOKEN not set in non-dry-run", func(t *testing.T) {
		t.Setenv("NPM_TOKEN", "")

		release := &Release{
			Version:    "v1.0.0",
			ProjectDir: "/project",
			FS:         io.Local,
		}
		pubCfg := PublisherConfig{
			Type: "npm",
			Extended: map[string]any{
				"package": "@test/package",
			},
		}
		relCfg := &mockReleaseConfig{repository: "owner/repo"}

		err := p.Publish(context.TODO(), release, pubCfg, relCfg, false)
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "NPM_TOKEN environment variable is required") {
			t.Fatalf("expected %v to contain %v", err.Error(), "NPM_TOKEN environment variable is required")
		}

	})
}

func TestNpm_NpmConfigDefaults_Good(t *testing.T) {
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

func TestNpm_NpmTemplateData_Good(t *testing.T) {
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
