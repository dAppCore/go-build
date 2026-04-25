// Package publishers provides release publishing implementations.
package publishers

import (
	"context"
	"embed"
	"text/template"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/core"
	coreio "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

//go:embed templates/npm/*.tmpl
var npmTemplates embed.FS

// NpmConfig holds npm-specific configuration.
//
// cfg := publishers.NpmConfig{Package: "@host-uk/core-build", Access: "public"}
type NpmConfig struct {
	// Package is the npm package name (e.g., "@host-uk/core").
	Package string
	// Access is the npm access level: "public" or "restricted".
	Access string
}

// NpmPublisher publishes releases to npm using the binary wrapper pattern.
//
// pub := publishers.NewNpmPublisher()
type NpmPublisher struct{}

// NewNpmPublisher creates a new npm publisher.
//
// pub := publishers.NewNpmPublisher()
func NewNpmPublisher() *NpmPublisher {
	return &NpmPublisher{}
}

// Name returns the publisher's identifier.
//
// name := pub.Name() // → "npm"
func (p *NpmPublisher) Name() string {
	return "npm"
}

// Validate checks the npm publisher configuration before publishing.
func (p *NpmPublisher) Validate(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig) error {
	_ = ctx
	if err := validatePublisherRelease(p.Name(), release); err != nil {
		return err
	}

	npmCfg := p.parseConfig(pubCfg, relCfg)
	if npmCfg.Package == "" {
		return coreerr.E("npm.Validate", "package name is required (set publish.npm.package in config)", nil)
	}

	return nil
}

// Supports reports whether the publisher handles the requested target.
func (p *NpmPublisher) Supports(target string) bool {
	return supportsPublisherTarget(p.Name(), target)
}

// Publish publishes the release to npm as a binary wrapper package.
//
// err := pub.Publish(ctx, rel, pubCfg, relCfg, false)
func (p *NpmPublisher) Publish(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig, dryRun bool) error {
	// Parse npm config
	npmCfg := p.parseConfig(pubCfg, relCfg)

	// Validate configuration
	if npmCfg.Package == "" {
		return coreerr.E("npm.Publish", "package name is required (set publish.npm.package in config)", nil)
	}

	// Get repository
	repo := ""
	if relCfg != nil {
		repo = relCfg.GetRepository()
	}
	if repo == "" {
		detectedRepo, err := detectRepository(ctx, release.ProjectDir)
		if err != nil {
			return coreerr.E("npm.Publish", "could not determine repository", err)
		}
		repo = detectedRepo
	}

	// Get project name (binary name)
	projectName := ""
	if relCfg != nil {
		projectName = relCfg.GetProjectName()
	}
	if projectName == "" {
		// Try to infer from package name
		parts := core.Split(npmCfg.Package, "/")
		projectName = parts[len(parts)-1]
	}

	// Strip leading 'v' from version for npm
	version := core.TrimPrefix(release.Version, "v")
	checksums := buildChecksumMapFromRelease(release)

	// Template data
	data := npmTemplateData{
		Package:     npmCfg.Package,
		Version:     version,
		Description: core.Sprintf("%s CLI", projectName),
		License:     "MIT",
		Repository:  repo,
		BinaryName:  projectName,
		ProjectName: projectName,
		Access:      npmCfg.Access,
		Checksums:   checksums,
	}

	if dryRun {
		return p.dryRunPublish(release.FS, data)
	}

	return p.executePublish(ctx, release.FS, data, &npmCfg)
}

// parseConfig extracts npm-specific configuration from the publisher config.
func (p *NpmPublisher) parseConfig(pubCfg PublisherConfig, relCfg ReleaseConfig) NpmConfig {
	cfg := NpmConfig{
		Package: "",
		Access:  "public",
	}

	// Override from extended config if present
	if ext, ok := pubCfg.Extended.(map[string]any); ok {
		if pkg, ok := ext["package"].(string); ok && pkg != "" {
			cfg.Package = pkg
		}
		if access, ok := ext["access"].(string); ok && access != "" {
			cfg.Access = access
		}
	}

	return cfg
}

// npmTemplateData holds data for npm templates.
type npmTemplateData struct {
	Package     string
	Version     string
	Description string
	License     string
	Repository  string
	BinaryName  string
	ProjectName string
	Access      string
	Checksums   ChecksumMap
}

// dryRunPublish shows what would be done without actually publishing.
func (p *NpmPublisher) dryRunPublish(m coreio.Medium, data npmTemplateData) error {
	publisherPrintln()
	publisherPrintln("=== DRY RUN: npm Publish ===")
	publisherPrintln()
	publisherPrint("Package:    %s", data.Package)
	publisherPrint("Version:    %s", data.Version)
	publisherPrint("Access:     %s", data.Access)
	publisherPrint("Repository: %s", data.Repository)
	publisherPrint("Binary:     %s", data.BinaryName)
	publisherPrintln()

	// Generate and show package.json
	pkgJSON, err := p.renderTemplate(m, "templates/npm/package.json.tmpl", data)
	if err != nil {
		return coreerr.E("npm.dryRunPublish", "failed to render template", err)
	}
	publisherPrintln("Generated package.json:")
	publisherPrintln("---")
	publisherPrintln(pkgJSON)
	publisherPrintln("---")
	publisherPrintln()

	publisherPrintln("Would run: npm publish --access", data.Access)
	publisherPrintln()
	publisherPrintln("=== END DRY RUN ===")

	return nil
}

// executePublish actually creates and publishes the npm package.
func (p *NpmPublisher) executePublish(ctx context.Context, m coreio.Medium, data npmTemplateData, cfg *NpmConfig) error {
	// Check for NPM_TOKEN
	npmToken := core.Env("NPM_TOKEN")
	if npmToken == "" {
		return coreerr.E("npm.Publish", "NPM_TOKEN environment variable is required", nil)
	}

	// Create temp directory for package
	tmpDir, err := ax.TempDir("npm-publish-*")
	if err != nil {
		return coreerr.E("npm.Publish", "failed to create temp directory", err)
	}
	defer func() { _ = ax.RemoveAll(tmpDir) }()

	// Create bin directory
	binDir := ax.Join(tmpDir, "bin")
	if err := ax.MkdirAll(binDir, 0o755); err != nil {
		return coreerr.E("npm.Publish", "failed to create bin directory", err)
	}

	// Generate package.json
	pkgJSON, err := p.renderTemplate(m, "templates/npm/package.json.tmpl", data)
	if err != nil {
		return coreerr.E("npm.Publish", "failed to render package.json", err)
	}
	if err := ax.WriteString(ax.Join(tmpDir, "package.json"), pkgJSON, 0o644); err != nil {
		return coreerr.E("npm.Publish", "failed to write package.json", err)
	}

	// Generate install.js
	installJS, err := p.renderTemplate(m, "templates/npm/install.js.tmpl", data)
	if err != nil {
		return coreerr.E("npm.Publish", "failed to render install.js", err)
	}
	if err := ax.WriteString(ax.Join(tmpDir, "install.js"), installJS, 0o644); err != nil {
		return coreerr.E("npm.Publish", "failed to write install.js", err)
	}

	// Generate run.js
	runJS, err := p.renderTemplate(m, "templates/npm/run.js.tmpl", data)
	if err != nil {
		return coreerr.E("npm.Publish", "failed to render run.js", err)
	}
	if err := ax.WriteString(ax.Join(binDir, "run.js"), runJS, 0o644); err != nil {
		return coreerr.E("npm.Publish", "failed to write run.js", err)
	}

	// Create .npmrc with token
	npmrc := "//registry.npmjs.org/:_authToken=${NPM_TOKEN}\n"
	if err := ax.WriteString(ax.Join(tmpDir, ".npmrc"), npmrc, 0o644); err != nil {
		return coreerr.E("npm.Publish", "failed to write .npmrc", err)
	}

	// Run npm publish
	publisherPrint("Publishing %s@%s to npm...", data.Package, data.Version)
	if err := publisherRun(ctx, tmpDir, []string{"NPM_TOKEN=" + npmToken}, "npm", "publish", "--access", data.Access); err != nil {
		return coreerr.E("npm.Publish", "npm publish failed", err)
	}

	publisherPrint("Published %s@%s to npm", data.Package, data.Version)
	publisherPrint("  https://www.npmjs.com/package/%s", data.Package)

	return nil
}

// renderTemplate renders an embedded template with the given data.
func (p *NpmPublisher) renderTemplate(m coreio.Medium, name string, data npmTemplateData) (string, error) {
	var content []byte
	var err error

	// Try custom template from medium
	customPath := ax.Join(".core", name)
	if m != nil && m.IsFile(customPath) {
		customContent, err := m.Read(customPath)
		if err == nil {
			content = []byte(customContent)
		}
	}

	// Fallback to embedded template
	if content == nil {
		content, err = npmTemplates.ReadFile(name)
		if err != nil {
			return "", coreerr.E("npm.renderTemplate", "failed to read template "+name, err)
		}
	}

	tmpl, err := template.New(ax.Base(name)).Funcs(publisherTemplateFuncs()).Parse(string(content))
	if err != nil {
		return "", coreerr.E("npm.renderTemplate", "failed to parse template "+name, err)
	}

	buf := core.NewBuffer()
	if err := tmpl.Execute(buf, data); err != nil {
		return "", coreerr.E("npm.renderTemplate", "failed to execute template "+name, err)
	}

	return buf.String(), nil
}
