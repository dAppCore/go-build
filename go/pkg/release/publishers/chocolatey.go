// Package publishers provides release publishing implementations.
package publishers

import (
	"context"
	"embed"
	"text/template"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	storage "dappco.re/go/build/pkg/storage"
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

// Validate checks that the Chocolatey publisher has a release to publish.
func (p *ChocolateyPublisher) Validate(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig) core.Result {
	_ = ctx
	_ = pubCfg
	_ = relCfg
	return validatePublisherRelease(p.Name(), release)
}

// Supports reports whether the publisher handles the requested target.
func (p *ChocolateyPublisher) Supports(target string) bool {
	return supportsPublisherTarget(p.Name(), target)
}

// Publish publishes the release to Chocolatey.
//
// result := pub.Publish(ctx, rel, pubCfg, relCfg, false)
func (p *ChocolateyPublisher) Publish(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig, dryRun bool) core.Result {
	validated := validatePublisherRelease(p.Name(), release)
	if !validated.OK {
		return validated
	}

	cfg := p.parseConfig(pubCfg, relCfg)

	repo := ""
	if relCfg != nil {
		repo = relCfg.GetRepository()
	}
	if repo == "" {
		detectedRepoResult := detectRepository(ctx, release.ProjectDir)
		if !detectedRepoResult.OK {
			return core.Fail(core.E("chocolatey.Publish", "could not determine repository", core.NewError(detectedRepoResult.Error())))
		}
		repo = detectedRepoResult.Value.(string)
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
	checksums := buildChecksumMapFromRelease(release)

	// Extract authors from repository
	authors := core.Split(repo, "/")[0]

	data := chocolateyTemplateData{
		PackageName: packageName,
		Title:       core.Sprintf("%s CLI", title(projectName)),
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

func title(text string) string {
	if text == "" {
		return ""
	}
	return core.Upper(text[:1]) + text[1:]
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

func (p *ChocolateyPublisher) dryRunPublish(m storage.Medium, data chocolateyTemplateData, cfg ChocolateyConfig) core.Result {
	publisherPrintln()
	publisherPrintln("=== DRY RUN: Chocolatey Publish ===")
	publisherPrintln()
	publisherPrint("Package:    %s", data.PackageName)
	publisherPrint("Version:    %s", data.Version)
	publisherPrint("Push:       %t", cfg.Push)
	publisherPrint("Repository: %s", data.Repository)
	publisherPrintln()

	nuspecResult := p.renderTemplate(m, "templates/chocolatey/package.nuspec.tmpl", data)
	if !nuspecResult.OK {
		return core.Fail(core.E("chocolatey.dryRunPublish", "failed to render nuspec", core.NewError(nuspecResult.Error())))
	}
	nuspec := nuspecResult.Value.(string)
	publisherPrintln("Generated package.nuspec:")
	publisherPrintln("---")
	publisherPrintln(nuspec)
	publisherPrintln("---")
	publisherPrintln()

	installResult := p.renderTemplate(m, "templates/chocolatey/tools/chocolateyinstall.ps1.tmpl", data)
	if !installResult.OK {
		return core.Fail(core.E("chocolatey.dryRunPublish", "failed to render install script", core.NewError(installResult.Error())))
	}
	install := installResult.Value.(string)
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

	return core.Ok(nil)
}

func (p *ChocolateyPublisher) executePublish(ctx context.Context, projectDir string, data chocolateyTemplateData, cfg ChocolateyConfig, release *Release) core.Result {
	nuspecResult := p.renderTemplate(release.FS, "templates/chocolatey/package.nuspec.tmpl", data)
	if !nuspecResult.OK {
		return core.Fail(core.E("chocolatey.Publish", "failed to render nuspec", core.NewError(nuspecResult.Error())))
	}
	nuspec := nuspecResult.Value.(string)

	installResult := p.renderTemplate(release.FS, "templates/chocolatey/tools/chocolateyinstall.ps1.tmpl", data)
	if !installResult.OK {
		return core.Fail(core.E("chocolatey.Publish", "failed to render install script", core.NewError(installResult.Error())))
	}
	install := installResult.Value.(string)

	// Create package directory
	output := ax.Join(projectDir, "dist", "chocolatey")
	if cfg.Official != nil && cfg.Official.Enabled && cfg.Official.Output != "" {
		output = cfg.Official.Output
		if !ax.IsAbs(output) {
			output = ax.Join(projectDir, output)
		}
	}

	toolsDir := ax.Join(output, "tools")
	created := release.FS.EnsureDir(toolsDir)
	if !created.OK {
		return core.Fail(core.E("chocolatey.Publish", "failed to create output directory", core.NewError(created.Error())))
	}

	// Write files
	nuspecPath := ax.Join(output, core.Sprintf("%s.nuspec", data.PackageName))
	wroteNuspec := release.FS.Write(nuspecPath, nuspec)
	if !wroteNuspec.OK {
		return core.Fail(core.E("chocolatey.Publish", "failed to write nuspec", core.NewError(wroteNuspec.Error())))
	}

	installPath := ax.Join(toolsDir, "chocolateyinstall.ps1")
	wroteInstall := release.FS.Write(installPath, install)
	if !wroteInstall.OK {
		return core.Fail(core.E("chocolatey.Publish", "failed to write install script", core.NewError(wroteInstall.Error())))
	}

	publisherPrint("Wrote Chocolatey package files: %s", output)

	// Push to Chocolatey if configured
	if cfg.Push {
		pushed := p.pushToChocolatey(ctx, output, data)
		if !pushed.OK {
			return pushed
		}
	}

	return core.Ok(nil)
}

func (p *ChocolateyPublisher) pushToChocolatey(ctx context.Context, packageDir string, data chocolateyTemplateData) core.Result {
	// Check for CHOCOLATEY_API_KEY
	apiKey := core.Env("CHOCOLATEY_API_KEY")
	if apiKey == "" {
		return core.Fail(core.E("chocolatey.Publish", "CHOCOLATEY_API_KEY environment variable is required for push", nil))
	}

	// Pack the package
	nupkgPath := ax.Join(packageDir, core.Sprintf("%s.%s.nupkg", data.PackageName, data.Version))

	packed := publisherRun(ctx, "", nil, "choco", "pack", ax.Join(packageDir, core.Sprintf("%s.nuspec", data.PackageName)), "-OutputDirectory", packageDir)
	if !packed.OK {
		return core.Fail(core.E("chocolatey.Publish", "choco pack failed", core.NewError(packed.Error())))
	}

	// Push the package — pass API key via environment variable to avoid exposing it in process listings
	pushed := publisherRun(ctx, "", []string{"chocolateyApiKey=" + apiKey}, "choco", "push", nupkgPath, "--source", "https://push.chocolatey.org/")
	if !pushed.OK {
		return core.Fail(core.E("chocolatey.Publish", "choco push failed", core.NewError(pushed.Error())))
	}

	publisherPrint("Published to Chocolatey: https://community.chocolatey.org/packages/%s", data.PackageName)
	return core.Ok(nil)
}

func (p *ChocolateyPublisher) renderTemplate(m storage.Medium, name string, data chocolateyTemplateData) core.Result {
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
		embeddedContent, readFailure := chocolateyTemplates.ReadFile(name)
		if readFailure != nil {
			return core.Fail(core.E("chocolatey.renderTemplate", "failed to read template "+name, readFailure))
		}
		content = embeddedContent
	}

	tmpl, parseFailure := template.New(ax.Base(name)).Funcs(publisherTemplateFuncs()).Parse(string(content))
	if parseFailure != nil {
		return core.Fail(core.E("chocolatey.renderTemplate", "failed to parse template "+name, parseFailure))
	}

	buf := core.NewBuffer()
	if executeFailure := tmpl.Execute(buf, data); executeFailure != nil {
		return core.Fail(core.E("chocolatey.renderTemplate", "failed to execute template "+name, executeFailure))
	}

	return core.Ok(buf.String())
}
