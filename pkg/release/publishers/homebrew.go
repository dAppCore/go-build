// Package publishers provides release publishing implementations.
package publishers

import (
	"context"       // Note: AX-6 — carries cancellation through publish and git operations.
	"embed"         // Note: AX-6 — embeds Homebrew templates for release publishing.
	"text/template" // Note: AX-6 — renders Homebrew formula templates.
	"unicode"       // Note: AX-6 — classifies runes while deriving Ruby formula class names.

	"dappco.re/go"                   // Note: AX-6 — provides approved string and formatting helpers.
	"dappco.re/go/build/internal/ax" // Note: AX-6 — Core-backed path and filesystem helpers replace banned stdlib calls.
	coreio "dappco.re/go/io"         // Note: AX-6 — Core Medium abstraction for release filesystem access.
	coreerr "dappco.re/go/log"       // Note: AX-6 — wraps publisher errors with Core logging semantics.
)

//go:embed templates/homebrew/*.tmpl
var homebrewTemplates embed.FS

// HomebrewConfig holds Homebrew-specific configuration.
//
// cfg := publishers.HomebrewConfig{Tap: "host-uk/homebrew-tap", Formula: "core-build"}
type HomebrewConfig struct {
	// Tap is the Homebrew tap repository (e.g., "host-uk/homebrew-tap").
	Tap string
	// Formula is the formula name (defaults to project name).
	Formula string
	// Official config for generating files for official repo PRs.
	Official *OfficialConfig
}

// OfficialConfig holds configuration for generating files for official repo PRs.
//
// cfg.Official = &publishers.OfficialConfig{Enabled: true, Output: "dist/homebrew"}
type OfficialConfig struct {
	// Enabled determines whether to generate files for official repos.
	Enabled bool
	// Output is the directory to write generated files.
	Output string
}

// HomebrewPublisher publishes releases to Homebrew.
//
// pub := publishers.NewHomebrewPublisher()
type HomebrewPublisher struct{}

// NewHomebrewPublisher creates a new Homebrew publisher.
//
// pub := publishers.NewHomebrewPublisher()
func NewHomebrewPublisher() *HomebrewPublisher {
	return &HomebrewPublisher{}
}

// Name returns the publisher's identifier.
//
// name := pub.Name() // → "homebrew"
func (p *HomebrewPublisher) Name() string {
	return "homebrew"
}

// Validate checks the Homebrew publisher configuration before publishing.
func (p *HomebrewPublisher) Validate(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig) core.Result {
	_ = ctx
	validated := validatePublisherRelease(p.Name(), release)
	if !validated.OK {
		return validated
	}

	cfg := p.parseConfig(pubCfg, relCfg)
	if cfg.Tap == "" && (cfg.Official == nil || !cfg.Official.Enabled) {
		return core.Fail(coreerr.E("homebrew.Validate", "tap is required (set publish.homebrew.tap in config)", nil))
	}

	return core.Ok(nil)
}

// Supports reports whether the publisher handles the requested target.
func (p *HomebrewPublisher) Supports(target string) bool {
	return supportsPublisherTarget(p.Name(), target)
}

// Publish publishes the release to Homebrew.
//
// result := pub.Publish(ctx, rel, pubCfg, relCfg, false)
func (p *HomebrewPublisher) Publish(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig, dryRun bool) core.Result {
	validated := validatePublisherRelease(p.Name(), release)
	if !validated.OK {
		return validated
	}

	// Parse config
	cfg := p.parseConfig(pubCfg, relCfg)

	// Validate configuration
	if cfg.Tap == "" && (cfg.Official == nil || !cfg.Official.Enabled) {
		return core.Fail(coreerr.E("homebrew.Publish", "tap is required (set publish.homebrew.tap in config)", nil))
	}

	// Get repository and project info
	repo := ""
	if relCfg != nil {
		repo = relCfg.GetRepository()
	}
	if repo == "" {
		detectedRepoResult := detectRepository(ctx, release.ProjectDir)
		if !detectedRepoResult.OK {
			return core.Fail(coreerr.E("homebrew.Publish", "could not determine repository", core.NewError(detectedRepoResult.Error())))
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

	formulaName := cfg.Formula
	if formulaName == "" {
		formulaName = projectName
	}

	// Strip leading 'v' from version
	version := core.TrimPrefix(release.Version, "v")

	// Build checksums map from artifacts
	checksums := buildChecksumMapFromRelease(release)

	// Template data
	data := homebrewTemplateData{
		FormulaClass: toFormulaClass(formulaName),
		Description:  core.Sprintf("%s CLI", projectName),
		Repository:   repo,
		Version:      version,
		License:      "MIT",
		BinaryName:   projectName,
		Checksums:    checksums,
	}

	if dryRun {
		return p.dryRunPublish(release.FS, data, cfg)
	}

	return p.executePublish(ctx, release.ProjectDir, data, cfg, release)
}

// homebrewTemplateData holds data for Homebrew templates.
type homebrewTemplateData struct {
	FormulaClass string
	Description  string
	Repository   string
	Version      string
	License      string
	BinaryName   string
	Checksums    ChecksumMap
}

// parseConfig extracts Homebrew-specific configuration.
func (p *HomebrewPublisher) parseConfig(pubCfg PublisherConfig, relCfg ReleaseConfig) HomebrewConfig {
	cfg := HomebrewConfig{
		Tap:     "",
		Formula: "",
	}

	if ext, ok := pubCfg.Extended.(map[string]any); ok {
		if tap, ok := ext["tap"].(string); ok && tap != "" {
			cfg.Tap = tap
		}
		if formula, ok := ext["formula"].(string); ok && formula != "" {
			cfg.Formula = formula
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

// dryRunPublish shows what would be done.
func (p *HomebrewPublisher) dryRunPublish(m coreio.Medium, data homebrewTemplateData, cfg HomebrewConfig) core.Result {
	publisherPrintln()
	publisherPrintln("=== DRY RUN: Homebrew Publish ===")
	publisherPrintln()
	publisherPrint("Formula:    %s", data.FormulaClass)
	publisherPrint("Version:    %s", data.Version)
	publisherPrint("Tap:        %s", cfg.Tap)
	publisherPrint("Repository: %s", data.Repository)
	publisherPrintln()

	// Generate and show formula
	formulaResult := p.renderTemplate(m, "templates/homebrew/formula.rb.tmpl", data)
	if !formulaResult.OK {
		return core.Fail(coreerr.E("homebrew.dryRunPublish", "failed to render template", core.NewError(formulaResult.Error())))
	}
	formula := formulaResult.Value.(string)
	publisherPrintln("Generated formula.rb:")
	publisherPrintln("---")
	publisherPrintln(formula)
	publisherPrintln("---")
	publisherPrintln()

	if cfg.Tap != "" && !homebrewOfficialMode(cfg) {
		publisherPrint("Would commit to tap: %s", cfg.Tap)
	}
	if homebrewOfficialMode(cfg) {
		output := cfg.Official.Output
		if output == "" {
			output = "dist/homebrew"
		}
		publisherPrint("Would write files for official PR to: %s", output)
	}
	publisherPrintln()
	publisherPrintln("=== END DRY RUN ===")

	return core.Ok(nil)
}

// executePublish creates the formula and commits to tap.
func (p *HomebrewPublisher) executePublish(ctx context.Context, projectDir string, data homebrewTemplateData, cfg HomebrewConfig, release *Release) core.Result {
	// Generate formula
	formulaResult := p.renderTemplate(release.FS, "templates/homebrew/formula.rb.tmpl", data)
	if !formulaResult.OK {
		return core.Fail(coreerr.E("homebrew.Publish", "failed to render formula", core.NewError(formulaResult.Error())))
	}
	formula := formulaResult.Value.(string)

	// If official config is enabled, write to output directory
	if homebrewOfficialMode(cfg) {
		output := cfg.Official.Output
		if output == "" {
			output = ax.Join(projectDir, "dist", "homebrew")
		} else if !ax.IsAbs(output) {
			output = ax.Join(projectDir, output)
		}

		created := release.FS.EnsureDir(output)
		if !created.OK {
			return core.Fail(coreerr.E("homebrew.Publish", "failed to create output directory", core.NewError(created.Error())))
		}

		formulaPath := ax.Join(output, core.Sprintf("%s.rb", core.Lower(data.FormulaClass)))
		written := release.FS.Write(formulaPath, formula)
		if !written.OK {
			return core.Fail(coreerr.E("homebrew.Publish", "failed to write formula", core.NewError(written.Error())))
		}
		publisherPrint("Wrote Homebrew formula for official PR: %s", formulaPath)
	}

	// Official repo mode generates PR-ready files and does not publish directly.
	if cfg.Tap != "" && !homebrewOfficialMode(cfg) {
		committed := p.commitToTap(ctx, cfg.Tap, data, formula)
		if !committed.OK {
			return committed
		}
	}

	return core.Ok(nil)
}

func homebrewOfficialMode(cfg HomebrewConfig) bool {
	return cfg.Official != nil && cfg.Official.Enabled
}

// commitToTap commits the formula to the tap repository.
func (p *HomebrewPublisher) commitToTap(ctx context.Context, tap string, data homebrewTemplateData, formula string) core.Result {
	// Clone tap repo to temp directory
	tmpDirResult := ax.TempDir("homebrew-tap-*")
	if !tmpDirResult.OK {
		return core.Fail(coreerr.E("homebrew.commitToTap", "failed to create temp directory", core.NewError(tmpDirResult.Error())))
	}
	tmpDir := tmpDirResult.Value.(string)
	defer func() { ax.RemoveAll(tmpDir) }()

	// Clone the tap
	publisherPrint("Cloning tap %s...", tap)
	cloned := publisherRun(ctx, "", nil, "gh", "repo", "clone", tap, tmpDir, "--", "--depth=1")
	if !cloned.OK {
		return core.Fail(coreerr.E("homebrew.commitToTap", "failed to clone tap", core.NewError(cloned.Error())))
	}

	// Ensure Formula directory exists
	formulaDir := ax.Join(tmpDir, "Formula")
	createdFormulaDir := ax.MkdirAll(formulaDir, 0o755)
	if !createdFormulaDir.OK {
		return core.Fail(coreerr.E("homebrew.commitToTap", "failed to create Formula directory", core.NewError(createdFormulaDir.Error())))
	}

	// Write formula
	formulaPath := ax.Join(formulaDir, core.Sprintf("%s.rb", core.Lower(data.FormulaClass)))
	written := ax.WriteString(formulaPath, formula, 0o644)
	if !written.OK {
		return core.Fail(coreerr.E("homebrew.commitToTap", "failed to write formula", core.NewError(written.Error())))
	}

	// Git add, commit, push
	commitMsg := core.Sprintf("Update %s to %s", data.FormulaClass, data.Version)

	added := ax.ExecDir(ctx, tmpDir, "git", "add", ".")
	if !added.OK {
		return core.Fail(coreerr.E("homebrew.commitToTap", "git add failed", core.NewError(added.Error())))
	}

	committed := publisherRun(ctx, tmpDir, nil, "git", "commit", "-m", commitMsg)
	if !committed.OK {
		return core.Fail(coreerr.E("homebrew.commitToTap", "git commit failed", core.NewError(committed.Error())))
	}

	pushed := publisherRun(ctx, tmpDir, nil, "git", "push")
	if !pushed.OK {
		return core.Fail(coreerr.E("homebrew.commitToTap", "git push failed", core.NewError(pushed.Error())))
	}

	publisherPrint("Updated Homebrew tap: %s", tap)
	return core.Ok(nil)
}

// renderTemplate renders an embedded template with the given data.
func (p *HomebrewPublisher) renderTemplate(m coreio.Medium, name string, data homebrewTemplateData) core.Result {
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
		embeddedContent, readFailure := homebrewTemplates.ReadFile(name)
		if readFailure != nil {
			return core.Fail(coreerr.E("homebrew.renderTemplate", "failed to read template "+name, readFailure))
		}
		content = embeddedContent
	}

	tmpl, parseFailure := template.New(ax.Base(name)).Funcs(publisherTemplateFuncs()).Parse(string(content))
	if parseFailure != nil {
		return core.Fail(coreerr.E("homebrew.renderTemplate", "failed to parse template "+name, parseFailure))
	}

	buf := core.NewBuffer()
	if executeFailure := tmpl.Execute(buf, data); executeFailure != nil {
		return core.Fail(coreerr.E("homebrew.renderTemplate", "failed to execute template "+name, executeFailure))
	}

	return core.Ok(buf.String())
}

// toFormulaClass converts a package name to a Ruby class name.
func toFormulaClass(name string) string {
	parts := splitFormulaClassParts(name)
	if len(parts) == 0 {
		return "Core"
	}

	for i, part := range parts {
		if len(part) == 0 {
			continue
		}
		runes := []rune(part)
		parts[i] = core.Upper(string(runes[0])) + string(runes[1:])
	}

	return core.Join("", parts...)
}

func splitFormulaClassParts(name string) []string {
	var parts []string
	start := -1

	for i, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if start == -1 {
				start = i
			}
			continue
		}

		if start != -1 {
			parts = append(parts, name[start:i])
			start = -1
		}
	}

	if start != -1 {
		parts = append(parts, name[start:])
	}

	return parts
}
