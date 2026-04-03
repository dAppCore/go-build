// Package publishers provides release publishing implementations.
package publishers

import (
	"bytes"
	"context"
	"embed"
	"text/template"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/i18n"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

//go:embed templates/chocolatey/*.tmpl templates/chocolatey/tools/*.tmpl
var chocolateyTemplates embed.FS

// ChocolateyConfig holds Chocolatey-specific configuration.
//
// cfg := publishers.ChocolateyConfig{Package: "core-build", Push: true}
type ChocolateyConfig struct {
	// Package is the Chocolatey package name.
	Package string
	// Push determines whether to push to Chocolatey (false = generate only).
	Push bool
	// Official config for generating files for official repo PRs.
	Official *OfficialConfig
}

// ChocolateyPublisher publishes releases to Chocolatey.
//
// pub := publishers.NewChocolateyPublisher()
type ChocolateyPublisher struct{}

// NewChocolateyPublisher creates a new Chocolatey publisher.
//
// pub := publishers.NewChocolateyPublisher()
func NewChocolateyPublisher() *ChocolateyPublisher {
	return &ChocolateyPublisher{}
}

// Name returns the publisher's identifier.
//
// name := pub.Name() // → "chocolatey"
func (p *ChocolateyPublisher) Name() string {
	return "chocolatey"
}

// Publish publishes the release to Chocolatey.
//
// err := pub.Publish(ctx, rel, pubCfg, relCfg, false)
func (p *ChocolateyPublisher) Publish(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig, dryRun bool) error {
	cfg := p.parseConfig(pubCfg, relCfg)

	repo := ""
	if relCfg != nil {
		repo = relCfg.GetRepository()
	}
	if repo == "" {
		detectedRepo, err := detectRepository(ctx, release.ProjectDir)
		if err != nil {
			return coreerr.E("chocolatey.Publish", "could not determine repository", err)
		}
		repo = detectedRepo
	}

	projectName := ""
	if relCfg != nil {
		projectName = relCfg.GetProjectName()
	}
	if projectName == "" {
		parts := core.Split(repo, "/")
		projectName = parts[len(parts)-1]
	}

	packageName := cfg.Package
	if packageName == "" {
		packageName = projectName
	}

	version := core.TrimPrefix(release.Version, "v")
	checksums := buildChecksumMap(release.Artifacts)

	// Extract authors from repository
	authors := core.Split(repo, "/")[0]

	data := chocolateyTemplateData{
		PackageName: packageName,
		Title:       core.Sprintf("%s CLI", i18n.Title(projectName)),
		Description: core.Sprintf("%s CLI", projectName),
		Repository:  repo,
		Version:     version,
		License:     "MIT",
		BinaryName:  projectName,
		Authors:     authors,
		Tags:        core.Sprintf("cli %s", projectName),
		Checksums:   checksums,
	}

	if dryRun {
		return p.dryRunPublish(release.FS, data, cfg)
	}

	return p.executePublish(ctx, release.ProjectDir, data, cfg, release)
}

type chocolateyTemplateData struct {
	PackageName string
	Title       string
	Description string
	Repository  string
	Version     string
	License     string
	BinaryName  string
	Authors     string
	Tags        string
	Checksums   ChecksumMap
}

func (p *ChocolateyPublisher) parseConfig(pubCfg PublisherConfig, relCfg ReleaseConfig) ChocolateyConfig {
	cfg := ChocolateyConfig{
		Push: false, // Default to generate only
	}

	if ext, ok := pubCfg.Extended.(map[string]any); ok {
		if pkg, ok := ext["package"].(string); ok && pkg != "" {
			cfg.Package = pkg
		}
		if push, ok := ext["push"].(bool); ok {
			cfg.Push = push
		}
		if official, ok := ext["official"].(map[string]any); ok {
			cfg.Official = &OfficialConfig{}
			if enabled, ok := official["enabled"].(bool); ok {
				cfg.Official.Enabled = enabled
			}
			if output, ok := official["output"].(string); ok {
				cfg.Official.Output = output
			}
		}
	}

	return cfg
}

func (p *ChocolateyPublisher) dryRunPublish(m io.Medium, data chocolateyTemplateData, cfg ChocolateyConfig) error {
	publisherPrintln()
	publisherPrintln("=== DRY RUN: Chocolatey Publish ===")
	publisherPrintln()
	publisherPrint("Package:    %s", data.PackageName)
	publisherPrint("Version:    %s", data.Version)
	publisherPrint("Push:       %t", cfg.Push)
	publisherPrint("Repository: %s", data.Repository)
	publisherPrintln()

	nuspec, err := p.renderTemplate(m, "templates/chocolatey/package.nuspec.tmpl", data)
	if err != nil {
		return coreerr.E("chocolatey.dryRunPublish", "failed to render nuspec", err)
	}
	publisherPrintln("Generated package.nuspec:")
	publisherPrintln("---")
	publisherPrintln(nuspec)
	publisherPrintln("---")
	publisherPrintln()

	install, err := p.renderTemplate(m, "templates/chocolatey/tools/chocolateyinstall.ps1.tmpl", data)
	if err != nil {
		return coreerr.E("chocolatey.dryRunPublish", "failed to render install script", err)
	}
	publisherPrintln("Generated chocolateyinstall.ps1:")
	publisherPrintln("---")
	publisherPrintln(install)
	publisherPrintln("---")
	publisherPrintln()

	if cfg.Push {
		publisherPrintln("Would push to Chocolatey community repo")
	} else {
		publisherPrintln("Would generate package files only (push=false)")
	}
	publisherPrintln()
	publisherPrintln("=== END DRY RUN ===")

	return nil
}

func (p *ChocolateyPublisher) executePublish(ctx context.Context, projectDir string, data chocolateyTemplateData, cfg ChocolateyConfig, release *Release) error {
	nuspec, err := p.renderTemplate(release.FS, "templates/chocolatey/package.nuspec.tmpl", data)
	if err != nil {
		return coreerr.E("chocolatey.Publish", "failed to render nuspec", err)
	}

	install, err := p.renderTemplate(release.FS, "templates/chocolatey/tools/chocolateyinstall.ps1.tmpl", data)
	if err != nil {
		return coreerr.E("chocolatey.Publish", "failed to render install script", err)
	}

	// Create package directory
	output := ax.Join(projectDir, "dist", "chocolatey")
	if cfg.Official != nil && cfg.Official.Enabled && cfg.Official.Output != "" {
		output = cfg.Official.Output
		if !ax.IsAbs(output) {
			output = ax.Join(projectDir, output)
		}
	}

	toolsDir := ax.Join(output, "tools")
	if err := release.FS.EnsureDir(toolsDir); err != nil {
		return coreerr.E("chocolatey.Publish", "failed to create output directory", err)
	}

	// Write files
	nuspecPath := ax.Join(output, core.Sprintf("%s.nuspec", data.PackageName))
	if err := release.FS.Write(nuspecPath, nuspec); err != nil {
		return coreerr.E("chocolatey.Publish", "failed to write nuspec", err)
	}

	installPath := ax.Join(toolsDir, "chocolateyinstall.ps1")
	if err := release.FS.Write(installPath, install); err != nil {
		return coreerr.E("chocolatey.Publish", "failed to write install script", err)
	}

	publisherPrint("Wrote Chocolatey package files: %s", output)

	// Push to Chocolatey if configured
	if cfg.Push {
		if err := p.pushToChocolatey(ctx, output, data); err != nil {
			return err
		}
	}

	return nil
}

func (p *ChocolateyPublisher) pushToChocolatey(ctx context.Context, packageDir string, data chocolateyTemplateData) error {
	// Check for CHOCOLATEY_API_KEY
	apiKey := core.Env("CHOCOLATEY_API_KEY")
	if apiKey == "" {
		return coreerr.E("chocolatey.Publish", "CHOCOLATEY_API_KEY environment variable is required for push", nil)
	}

	// Pack the package
	nupkgPath := ax.Join(packageDir, core.Sprintf("%s.%s.nupkg", data.PackageName, data.Version))

	if err := publisherRun(ctx, "", nil, "choco", "pack", ax.Join(packageDir, core.Sprintf("%s.nuspec", data.PackageName)), "-OutputDirectory", packageDir); err != nil {
		return coreerr.E("chocolatey.Publish", "choco pack failed", err)
	}

	// Push the package — pass API key via environment variable to avoid exposing it in process listings
	if err := publisherRun(ctx, "", []string{"chocolateyApiKey=" + apiKey}, "choco", "push", nupkgPath, "--source", "https://push.chocolatey.org/"); err != nil {
		return coreerr.E("chocolatey.Publish", "choco push failed", err)
	}

	publisherPrint("Published to Chocolatey: https://community.chocolatey.org/packages/%s", data.PackageName)
	return nil
}

func (p *ChocolateyPublisher) renderTemplate(m io.Medium, name string, data chocolateyTemplateData) (string, error) {
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
		content, err = chocolateyTemplates.ReadFile(name)
		if err != nil {
			return "", coreerr.E("chocolatey.renderTemplate", "failed to read template "+name, err)
		}
	}

	tmpl, err := template.New(ax.Base(name)).Parse(string(content))
	if err != nil {
		return "", coreerr.E("chocolatey.renderTemplate", "failed to parse template "+name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", coreerr.E("chocolatey.renderTemplate", "failed to execute template "+name, err)
	}

	return buf.String(), nil
}

// Ensure build package is used
var _ = build.Artifact{}
