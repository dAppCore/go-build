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
func (p *AURPublisher) Validate(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig) core.Result {
	_ = ctx
	validated := validatePublisherRelease(p.Name(), release)
	if !validated.OK {
		return validated
	}

	cfg := p.parseConfig(pubCfg, relCfg)
	if cfg.Maintainer == "" {
		return core.Fail(coreerr.E("aur.Validate", "maintainer is required (set publish.aur.maintainer in config)", nil))
	}

	return core.Ok(nil)
}

// Supports reports whether the publisher handles the requested target.
func (p *AURPublisher) Supports(target string) bool {
	return supportsPublisherTarget(p.Name(), target)
}

// Publish publishes the release to AUR.
//
// result := pub.Publish(ctx, rel, pubCfg, relCfg, false)
func (p *AURPublisher) Publish(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig, dryRun bool) core.Result {
	validated := validatePublisherRelease(p.Name(), release)
	if !validated.OK {
		return validated
	}

	cfg := p.parseConfig(pubCfg, relCfg)

	if cfg.Maintainer == "" {
		return core.Fail(coreerr.E("aur.Publish", "maintainer is required (set publish.aur.maintainer in config)", nil))
	}

	repo := ""
	if relCfg != nil {
		repo = relCfg.GetRepository()
	}
	if repo == "" {
		detectedRepoResult := detectRepository(ctx, release.ProjectDir)
		if !detectedRepoResult.OK {
			return core.Fail(coreerr.E("aur.Publish", "could not determine repository", core.NewError(detectedRepoResult.Error())))
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

func (p *AURPublisher) dryRunPublish(m coreio.Medium, data aurTemplateData, cfg AURConfig) core.Result {
	publisherPrintln()
	publisherPrintln("=== DRY RUN: AUR Publish ===")
	publisherPrintln()
	publisherPrint("Package:    %s-bin", data.PackageName)
	publisherPrint("Version:    %s", data.Version)
	publisherPrint("Maintainer: %s", data.Maintainer)
	publisherPrint("Repository: %s", data.Repository)
	publisherPrintln()

	pkgbuildResult := p.renderTemplate(m, "templates/aur/PKGBUILD.tmpl", data)
	if !pkgbuildResult.OK {
		return core.Fail(coreerr.E("aur.dryRunPublish", "failed to render PKGBUILD template", core.NewError(pkgbuildResult.Error())))
	}
	pkgbuild := pkgbuildResult.Value.(string)
	publisherPrintln("Generated PKGBUILD:")
	publisherPrintln("---")
	publisherPrintln(pkgbuild)
	publisherPrintln("---")
	publisherPrintln()

	srcinfoResult := p.renderTemplate(m, "templates/aur/.SRCINFO.tmpl", data)
	if !srcinfoResult.OK {
		return core.Fail(coreerr.E("aur.dryRunPublish", "failed to render .SRCINFO template", core.NewError(srcinfoResult.Error())))
	}
	srcinfo := srcinfoResult.Value.(string)
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

	return core.Ok(nil)
}

func (p *AURPublisher) executePublish(ctx context.Context, projectDir string, data aurTemplateData, cfg AURConfig, release *Release) core.Result {
	pkgbuildResult := p.renderTemplate(release.FS, "templates/aur/PKGBUILD.tmpl", data)
	if !pkgbuildResult.OK {
		return core.Fail(coreerr.E("aur.Publish", "failed to render PKGBUILD", core.NewError(pkgbuildResult.Error())))
	}
	pkgbuild := pkgbuildResult.Value.(string)

	srcinfoResult := p.renderTemplate(release.FS, "templates/aur/.SRCINFO.tmpl", data)
	if !srcinfoResult.OK {
		return core.Fail(coreerr.E("aur.Publish", "failed to render .SRCINFO", core.NewError(srcinfoResult.Error())))
	}
	srcinfo := srcinfoResult.Value.(string)

	// If official config is enabled, write to output directory
	if aurOfficialMode(cfg) {
		output := cfg.Official.Output
		if output == "" {
			output = ax.Join(projectDir, "dist", "aur")
		} else if !ax.IsAbs(output) {
			output = ax.Join(projectDir, output)
		}

		created := release.FS.EnsureDir(output)
		if !created.OK {
			return core.Fail(coreerr.E("aur.Publish", "failed to create output directory", core.NewError(created.Error())))
		}

		pkgbuildPath := ax.Join(output, "PKGBUILD")
		wrotePKGBUILD := release.FS.Write(pkgbuildPath, pkgbuild)
		if !wrotePKGBUILD.OK {
			return core.Fail(coreerr.E("aur.Publish", "failed to write PKGBUILD", core.NewError(wrotePKGBUILD.Error())))
		}

		srcinfoPath := ax.Join(output, ".SRCINFO")
		wroteSRCINFO := release.FS.Write(srcinfoPath, srcinfo)
		if !wroteSRCINFO.OK {
			return core.Fail(coreerr.E("aur.Publish", "failed to write .SRCINFO", core.NewError(wroteSRCINFO.Error())))
		}
		publisherPrint("Wrote AUR files: %s", output)
	}

	// Push to AUR if not in official-only mode
	if !aurOfficialMode(cfg) {
		pushed := p.pushToAUR(ctx, data, pkgbuild, srcinfo)
		if !pushed.OK {
			return pushed
		}
	}

	return core.Ok(nil)
}

func aurOfficialMode(cfg AURConfig) bool {
	return cfg.Official != nil && cfg.Official.Enabled
}

func (p *AURPublisher) pushToAUR(ctx context.Context, data aurTemplateData, pkgbuild, srcinfo string) core.Result {
	aurURL := core.Sprintf("ssh://aur@aur.archlinux.org/%s-bin.git", data.PackageName)

	tmpDirResult := ax.TempDir("aur-package-*")
	if !tmpDirResult.OK {
		return core.Fail(coreerr.E("aur.pushToAUR", "failed to create temp directory", core.NewError(tmpDirResult.Error())))
	}
	tmpDir := tmpDirResult.Value.(string)
	defer func() { ax.RemoveAll(tmpDir) }()

	// Clone existing AUR repo (or initialise new one)
	publisherPrint("Cloning AUR package %s-bin...", data.PackageName)
	cloned := ax.Exec(ctx, "git", "clone", aurURL, tmpDir)
	if !cloned.OK {
		// If clone fails, init a new repo
		initialised := ax.Exec(ctx, "git", "init", tmpDir)
		if !initialised.OK {
			return core.Fail(coreerr.E("aur.pushToAUR", "failed to initialise repo", core.NewError(initialised.Error())))
		}
		addedRemote := ax.Exec(ctx, "git", "-C", tmpDir, "remote", "add", "origin", aurURL)
		if !addedRemote.OK {
			return core.Fail(coreerr.E("aur.pushToAUR", "failed to add remote", core.NewError(addedRemote.Error())))
		}
	}

	// Write files
	wrotePKGBUILD := ax.WriteString(ax.Join(tmpDir, "PKGBUILD"), pkgbuild, 0o644)
	if !wrotePKGBUILD.OK {
		return core.Fail(coreerr.E("aur.pushToAUR", "failed to write PKGBUILD", core.NewError(wrotePKGBUILD.Error())))
	}
	wroteSRCINFO := ax.WriteString(ax.Join(tmpDir, ".SRCINFO"), srcinfo, 0o644)
	if !wroteSRCINFO.OK {
		return core.Fail(coreerr.E("aur.pushToAUR", "failed to write .SRCINFO", core.NewError(wroteSRCINFO.Error())))
	}

	commitMsg := core.Sprintf("Update to %s", data.Version)

	added := ax.ExecDir(ctx, tmpDir, "git", "add", ".")
	if !added.OK {
		return core.Fail(coreerr.E("aur.pushToAUR", "git add failed", core.NewError(added.Error())))
	}

	committed := publisherRun(ctx, tmpDir, nil, "git", "commit", "-m", commitMsg)
	if !committed.OK {
		return core.Fail(coreerr.E("aur.pushToAUR", "git commit failed", core.NewError(committed.Error())))
	}

	pushed := publisherRun(ctx, tmpDir, nil, "git", "push", "origin", "master")
	if !pushed.OK {
		return core.Fail(coreerr.E("aur.pushToAUR", "git push failed", core.NewError(pushed.Error())))
	}

	publisherPrint("Published to AUR: https://aur.archlinux.org/packages/%s-bin", data.PackageName)
	return core.Ok(nil)
}

func (p *AURPublisher) renderTemplate(m coreio.Medium, name string, data aurTemplateData) core.Result {
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
		embeddedContent, readFailure := aurTemplates.ReadFile(name)
		if readFailure != nil {
			return core.Fail(coreerr.E("aur.renderTemplate", "failed to read template "+name, readFailure))
		}
		content = embeddedContent
	}

	tmpl, parseFailure := template.New(ax.Base(name)).Funcs(publisherTemplateFuncs()).Parse(string(content))
	if parseFailure != nil {
		return core.Fail(coreerr.E("aur.renderTemplate", "failed to parse template "+name, parseFailure))
	}

	buf := core.NewBuffer()
	if executeFailure := tmpl.Execute(buf, data); executeFailure != nil {
		return core.Fail(coreerr.E("aur.renderTemplate", "failed to execute template "+name, executeFailure))
	}

	return core.Ok(buf.String())
}
