// Package publishers provides release publishing implementations.
package publishers

import (
	"context"
	"embed"
	"text/template"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	coreio "dappco.re/go/io"
	coreerr "dappco.re/go/log"
)

//go:embed templates/aur/*.tmpl
var aurTemplates embed.FS

// AURConfig holds AUR-specific configuration.
//
// cfg := publishers.AURConfig{Package: "core-build", Maintainer: "Jane Doe <jane@example.com>"}
type AURConfig struct {
	// Package is the AUR package name.
	Package string
	// Maintainer is the package maintainer (e.g., "Name <email>").
	Maintainer string
	// Official config for generating files for official repo PRs.
	Official *OfficialConfig
}

// AURPublisher publishes releases to AUR.
//
// pub := publishers.NewAURPublisher()
type AURPublisher struct{}

// NewAURPublisher creates a new AUR publisher.
//
// pub := publishers.NewAURPublisher()
func NewAURPublisher() *AURPublisher {
	return &AURPublisher{}
}

// Name returns the publisher's identifier.
//
// name := pub.Name() // → "aur"
func (p *AURPublisher) Name() string {
	return "aur"
}

// Validate checks the AUR publisher configuration before publishing.
func (p *AURPublisher) Validate(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig) error {
	_ = ctx
	if err := validatePublisherRelease(p.Name(), release); err != nil {
		return err
	}

	cfg := p.parseConfig(pubCfg, relCfg)
	if cfg.Maintainer == "" {
		return coreerr.E("aur.Validate", "maintainer is required (set publish.aur.maintainer in config)", nil)
	}

	return nil
}

// Supports reports whether the publisher handles the requested target.
func (p *AURPublisher) Supports(target string) bool {
	return supportsPublisherTarget(p.Name(), target)
}

// Publish publishes the release to AUR.
//
// err := pub.Publish(ctx, rel, pubCfg, relCfg, false)
func (p *AURPublisher) Publish(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig, dryRun bool) error {
	if err := validatePublisherRelease(p.Name(), release); err != nil {
		return err
	}

	cfg := p.parseConfig(pubCfg, relCfg)

	if cfg.Maintainer == "" {
		return coreerr.E("aur.Publish", "maintainer is required (set publish.aur.maintainer in config)", nil)
	}

	repo := ""
	if relCfg != nil {
		repo = relCfg.GetRepository()
	}
	if repo == "" {
		detectedRepo, err := detectRepository(ctx, release.ProjectDir)
		if err != nil {
			return coreerr.E("aur.Publish", "could not determine repository", err)
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
	checksums := buildChecksumMapFromRelease(release)

	data := aurTemplateData{
		PackageName: packageName,
		Description: core.Sprintf("%s CLI", projectName),
		Repository:  repo,
		Version:     version,
		License:     "MIT",
		BinaryName:  projectName,
		Maintainer:  cfg.Maintainer,
		Checksums:   checksums,
	}

	if dryRun {
		return p.dryRunPublish(release.FS, data, cfg)
	}

	return p.executePublish(ctx, release.ProjectDir, data, cfg, release)
}

type aurTemplateData struct {
	PackageName string
	Description string
	Repository  string
	Version     string
	License     string
	BinaryName  string
	Maintainer  string
	Checksums   ChecksumMap
}

func (p *AURPublisher) parseConfig(pubCfg PublisherConfig, relCfg ReleaseConfig) AURConfig {
	cfg := AURConfig{}

	if ext, ok := pubCfg.Extended.(map[string]any); ok {
		if pkg, ok := ext["package"].(string); ok && pkg != "" {
			cfg.Package = pkg
		}
		if maintainer, ok := ext["maintainer"].(string); ok && maintainer != "" {
			cfg.Maintainer = maintainer
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

func (p *AURPublisher) dryRunPublish(m coreio.Medium, data aurTemplateData, cfg AURConfig) error {
	publisherPrintln()
	publisherPrintln("=== DRY RUN: AUR Publish ===")
	publisherPrintln()
	publisherPrint("Package:    %s-bin", data.PackageName)
	publisherPrint("Version:    %s", data.Version)
	publisherPrint("Maintainer: %s", data.Maintainer)
	publisherPrint("Repository: %s", data.Repository)
	publisherPrintln()

	pkgbuild, err := p.renderTemplate(m, "templates/aur/PKGBUILD.tmpl", data)
	if err != nil {
		return coreerr.E("aur.dryRunPublish", "failed to render PKGBUILD template", err)
	}
	publisherPrintln("Generated PKGBUILD:")
	publisherPrintln("---")
	publisherPrintln(pkgbuild)
	publisherPrintln("---")
	publisherPrintln()

	srcinfo, err := p.renderTemplate(m, "templates/aur/.SRCINFO.tmpl", data)
	if err != nil {
		return coreerr.E("aur.dryRunPublish", "failed to render .SRCINFO template", err)
	}
	publisherPrintln("Generated .SRCINFO:")
	publisherPrintln("---")
	publisherPrintln(srcinfo)
	publisherPrintln("---")
	publisherPrintln()

	if aurOfficialMode(cfg) {
		output := cfg.Official.Output
		if output == "" {
			output = "dist/aur"
		}
		publisherPrint("Would write files for official PR to: %s", output)
	} else {
		publisherPrint("Would push to AUR: ssh://aur@aur.archlinux.org/%s-bin.git", data.PackageName)
	}
	publisherPrintln()
	publisherPrintln("=== END DRY RUN ===")

	return nil
}

func (p *AURPublisher) executePublish(ctx context.Context, projectDir string, data aurTemplateData, cfg AURConfig, release *Release) error {
	pkgbuild, err := p.renderTemplate(release.FS, "templates/aur/PKGBUILD.tmpl", data)
	if err != nil {
		return coreerr.E("aur.Publish", "failed to render PKGBUILD", err)
	}

	srcinfo, err := p.renderTemplate(release.FS, "templates/aur/.SRCINFO.tmpl", data)
	if err != nil {
		return coreerr.E("aur.Publish", "failed to render .SRCINFO", err)
	}

	// If official config is enabled, write to output directory
	if aurOfficialMode(cfg) {
		output := cfg.Official.Output
		if output == "" {
			output = ax.Join(projectDir, "dist", "aur")
		} else if !ax.IsAbs(output) {
			output = ax.Join(projectDir, output)
		}

		if err := release.FS.EnsureDir(output); err != nil {
			return coreerr.E("aur.Publish", "failed to create output directory", err)
		}

		pkgbuildPath := ax.Join(output, "PKGBUILD")
		if err := release.FS.Write(pkgbuildPath, pkgbuild); err != nil {
			return coreerr.E("aur.Publish", "failed to write PKGBUILD", err)
		}

		srcinfoPath := ax.Join(output, ".SRCINFO")
		if err := release.FS.Write(srcinfoPath, srcinfo); err != nil {
			return coreerr.E("aur.Publish", "failed to write .SRCINFO", err)
		}
		publisherPrint("Wrote AUR files: %s", output)
	}

	// Push to AUR if not in official-only mode
	if !aurOfficialMode(cfg) {
		if err := p.pushToAUR(ctx, data, pkgbuild, srcinfo); err != nil {
			return err
		}
	}

	return nil
}

func aurOfficialMode(cfg AURConfig) bool {
	return cfg.Official != nil && cfg.Official.Enabled
}

func (p *AURPublisher) pushToAUR(ctx context.Context, data aurTemplateData, pkgbuild, srcinfo string) error {
	aurURL := core.Sprintf("ssh://aur@aur.archlinux.org/%s-bin.git", data.PackageName)

	tmpDir, err := ax.TempDir("aur-package-*")
	if err != nil {
		return coreerr.E("aur.pushToAUR", "failed to create temp directory", err)
	}
	defer func() { _ = ax.RemoveAll(tmpDir) }()

	// Clone existing AUR repo (or initialise new one)
	publisherPrint("Cloning AUR package %s-bin...", data.PackageName)
	if err := ax.Exec(ctx, "git", "clone", aurURL, tmpDir); err != nil {
		// If clone fails, init a new repo
		if err := ax.Exec(ctx, "git", "init", tmpDir); err != nil {
			return coreerr.E("aur.pushToAUR", "failed to initialise repo", err)
		}
		if err := ax.Exec(ctx, "git", "-C", tmpDir, "remote", "add", "origin", aurURL); err != nil {
			return coreerr.E("aur.pushToAUR", "failed to add remote", err)
		}
	}

	// Write files
	if err := ax.WriteString(ax.Join(tmpDir, "PKGBUILD"), pkgbuild, 0o644); err != nil {
		return coreerr.E("aur.pushToAUR", "failed to write PKGBUILD", err)
	}
	if err := ax.WriteString(ax.Join(tmpDir, ".SRCINFO"), srcinfo, 0o644); err != nil {
		return coreerr.E("aur.pushToAUR", "failed to write .SRCINFO", err)
	}

	commitMsg := core.Sprintf("Update to %s", data.Version)

	if err := ax.ExecDir(ctx, tmpDir, "git", "add", "."); err != nil {
		return coreerr.E("aur.pushToAUR", "git add failed", err)
	}

	if err := publisherRun(ctx, tmpDir, nil, "git", "commit", "-m", commitMsg); err != nil {
		return coreerr.E("aur.pushToAUR", "git commit failed", err)
	}

	if err := publisherRun(ctx, tmpDir, nil, "git", "push", "origin", "master"); err != nil {
		return coreerr.E("aur.pushToAUR", "git push failed", err)
	}

	publisherPrint("Published to AUR: https://aur.archlinux.org/packages/%s-bin", data.PackageName)
	return nil
}

func (p *AURPublisher) renderTemplate(m coreio.Medium, name string, data aurTemplateData) (string, error) {
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
		content, err = aurTemplates.ReadFile(name)
		if err != nil {
			return "", coreerr.E("aur.renderTemplate", "failed to read template "+name, err)
		}
	}

	tmpl, err := template.New(ax.Base(name)).Funcs(publisherTemplateFuncs()).Parse(string(content))
	if err != nil {
		return "", coreerr.E("aur.renderTemplate", "failed to parse template "+name, err)
	}

	buf := core.NewBuffer()
	if err := tmpl.Execute(buf, data); err != nil {
		return "", coreerr.E("aur.renderTemplate", "failed to execute template "+name, err)
	}

	return buf.String(), nil
}
