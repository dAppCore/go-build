// Package publishers provides release publishing implementations.
package publishers

import (
	"context"       // Note: AX-6 — carries cancellation through publish and npm operations.
	"embed"         // Note: AX-6 — embeds npm templates for release publishing.
	"text/template" // Note: AX-6 — renders npm package templates.

	"dappco.re/go"                          // Note: AX-6 — provides approved string and formatting helpers.
	"dappco.re/go/build/internal/ax"        // Note: AX-6 — Core-backed path and filesystem helpers replace banned stdlib calls.
	coreio "dappco.re/go/build/pkg/storage" // Note: AX-6 — Core Medium abstraction for release filesystem access.
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
func (p *NpmPublisher) Validate(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig) core.Result {
	_ = ctx
	validated := validatePublisherRelease(p.Name(), release)
	if !validated.OK {
		return validated
	}

	npmCfg := p.parseConfig(pubCfg, relCfg)
	if npmCfg.Package == "" {
		return core.Fail(core.E("npm.Validate", "package name is required (set publish.npm.package in config)", nil))
	}

	return core.Ok(nil)
}

// Supports reports whether the publisher handles the requested target.
func (p *NpmPublisher) Supports(target string) bool {
	return supportsPublisherTarget(p.Name(), target)
}

// Publish publishes the release to npm as a binary wrapper package.
//
// result := pub.Publish(ctx, rel, pubCfg, relCfg, false)
func (p *NpmPublisher) Publish(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig, dryRun bool) core.Result {
	validated := validatePublisherRelease(p.Name(), release)
	if !validated.OK {
		return validated
	}

	// Parse npm config
	npmCfg := p.parseConfig(pubCfg, relCfg)

	// Validate configuration
	if npmCfg.Package == "" {
		return core.Fail(core.E("npm.Publish", "package name is required (set publish.npm.package in config)", nil))
	}

	// Get repository
	repo := ""
	if relCfg != nil {
		repo = relCfg.GetRepository()
	}
	if repo == "" {
		detectedRepoResult := detectRepository(ctx, release.ProjectDir)
		if !detectedRepoResult.OK {
			return core.Fail(core.E("npm.Publish", "could not determine repository", core.NewError(detectedRepoResult.Error())))
		}
		repo = detectedRepoResult.Value.(string)
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
func (p *NpmPublisher) dryRunPublish(m coreio.Medium, data npmTemplateData) core.Result {
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
	pkgJSONResult := p.renderTemplate(m, "templates/npm/package.json.tmpl", data)
	if !pkgJSONResult.OK {
		return core.Fail(core.E("npm.dryRunPublish", "failed to render template", core.NewError(pkgJSONResult.Error())))
	}
	pkgJSON := pkgJSONResult.Value.(string)
	publisherPrintln("Generated package.json:")
	publisherPrintln("---")
	publisherPrintln(pkgJSON)
	publisherPrintln("---")
	publisherPrintln()

	publisherPrintln("Would run: npm publish --access", data.Access)
	publisherPrintln()
	publisherPrintln("=== END DRY RUN ===")

	return core.Ok(nil)
}

// executePublish actually creates and publishes the npm package.
func (p *NpmPublisher) executePublish(ctx context.Context, m coreio.Medium, data npmTemplateData, cfg *NpmConfig) core.Result {
	// Check for NPM_TOKEN
	npmToken := core.Env("NPM_TOKEN")
	if npmToken == "" {
		return core.Fail(core.E("npm.Publish", "NPM_TOKEN environment variable is required", nil))
	}

	// Create temp directory for package
	tmpDirResult := ax.TempDir("npm-publish-*")
	if !tmpDirResult.OK {
		return core.Fail(core.E("npm.Publish", "failed to create temp directory", core.NewError(tmpDirResult.Error())))
	}
	tmpDir := tmpDirResult.Value.(string)
	defer func() { _ = ax.RemoveAll(tmpDir) }()

	// Create bin directory
	binDir := ax.Join(tmpDir, "bin")
	createdBin := ax.MkdirAll(binDir, 0o755)
	if !createdBin.OK {
		return core.Fail(core.E("npm.Publish", "failed to create bin directory", core.NewError(createdBin.Error())))
	}

	// Generate package.json
	pkgJSONResult := p.renderTemplate(m, "templates/npm/package.json.tmpl", data)
	if !pkgJSONResult.OK {
		return core.Fail(core.E("npm.Publish", "failed to render package.json", core.NewError(pkgJSONResult.Error())))
	}
	pkgJSON := pkgJSONResult.Value.(string)
	wrotePackage := ax.WriteString(ax.Join(tmpDir, "package.json"), pkgJSON, 0o644)
	if !wrotePackage.OK {
		return core.Fail(core.E("npm.Publish", "failed to write package.json", core.NewError(wrotePackage.Error())))
	}

	// Generate install.js
	installJSResult := p.renderTemplate(m, "templates/npm/install.js.tmpl", data)
	if !installJSResult.OK {
		return core.Fail(core.E("npm.Publish", "failed to render install.js", core.NewError(installJSResult.Error())))
	}
	installJS := installJSResult.Value.(string)
	wroteInstall := ax.WriteString(ax.Join(tmpDir, "install.js"), installJS, 0o644)
	if !wroteInstall.OK {
		return core.Fail(core.E("npm.Publish", "failed to write install.js", core.NewError(wroteInstall.Error())))
	}

	// Generate run.js
	runJSResult := p.renderTemplate(m, "templates/npm/run.js.tmpl", data)
	if !runJSResult.OK {
		return core.Fail(core.E("npm.Publish", "failed to render run.js", core.NewError(runJSResult.Error())))
	}
	runJS := runJSResult.Value.(string)
	wroteRun := ax.WriteString(ax.Join(binDir, "run.js"), runJS, 0o644)
	if !wroteRun.OK {
		return core.Fail(core.E("npm.Publish", "failed to write run.js", core.NewError(wroteRun.Error())))
	}

	// Create .npmrc with token
	npmrc := "//registry.npmjs.org/:_authToken=${NPM_TOKEN}\n"
	wroteNPMRC := ax.WriteString(ax.Join(tmpDir, ".npmrc"), npmrc, 0o644)
	if !wroteNPMRC.OK {
		return core.Fail(core.E("npm.Publish", "failed to write .npmrc", core.NewError(wroteNPMRC.Error())))
	}

	// Run npm publish
	publisherPrint("Publishing %s@%s to npm...", data.Package, data.Version)
	published := publisherRun(ctx, tmpDir, []string{"NPM_TOKEN=" + npmToken}, "npm", "publish", "--access", data.Access)
	if !published.OK {
		return core.Fail(core.E("npm.Publish", "npm publish failed", core.NewError(published.Error())))
	}

	publisherPrint("Published %s@%s to npm", data.Package, data.Version)
	publisherPrint("  https://www.npmjs.com/package/%s", data.Package)

	return core.Ok(nil)
}

// renderTemplate renders an embedded template with the given data.
func (p *NpmPublisher) renderTemplate(m coreio.Medium, name string, data npmTemplateData) core.Result {
	var content []byte

	// Try custom template from medium
	customPath := ax.Join(".core", name)
	if m != nil && m.IsFile(customPath) {
		customContent := m.Read(customPath)
		if customContent.OK {
			content = []byte(customContent.Value.(string))
		}
	}

	// Fallback to embedded template
	if content == nil {
		embeddedContent, readFailure := npmTemplates.ReadFile(name)
		if readFailure != nil {
			return core.Fail(core.E("npm.renderTemplate", "failed to read template "+name, readFailure))
		}
		content = embeddedContent
	}

	tmpl, parseFailure := template.New(ax.Base(name)).Funcs(publisherTemplateFuncs()).Parse(string(content))
	if parseFailure != nil {
		return core.Fail(core.E("npm.renderTemplate", "failed to parse template "+name, parseFailure))
	}

	buf := core.NewBuffer()
	if executeFailure := tmpl.Execute(buf, data); executeFailure != nil {
		return core.Fail(core.E("npm.renderTemplate", "failed to execute template "+name, executeFailure))
	}

	return core.Ok(buf.String())
}
