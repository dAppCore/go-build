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
	coreio "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

//go:embed templates/homebrew/*.tmpl
var homebrewTemplates embed.FS

// HomebrewConfig holds Homebrew-specific configuration.
// Usage example: declare a value of type publishers.HomebrewConfig in integrating code.
type HomebrewConfig struct {
	// Tap is the Homebrew tap repository (e.g., "host-uk/homebrew-tap").
	Tap string
	// Formula is the formula name (defaults to project name).
	Formula string
	// Official config for generating files for official repo PRs.
	Official *OfficialConfig
}

// OfficialConfig holds configuration for generating files for official repo PRs.
// Usage example: declare a value of type publishers.OfficialConfig in integrating code.
type OfficialConfig struct {
	// Enabled determines whether to generate files for official repos.
	Enabled bool
	// Output is the directory to write generated files.
	Output string
}

// HomebrewPublisher publishes releases to Homebrew.
// Usage example: declare a value of type publishers.HomebrewPublisher in integrating code.
type HomebrewPublisher struct{}

// NewHomebrewPublisher creates a new Homebrew publisher.
// Usage example: call publishers.NewHomebrewPublisher(...) from integrating code.
func NewHomebrewPublisher() *HomebrewPublisher {
	return &HomebrewPublisher{}
}

// Name returns the publisher's identifier.
// Usage example: call value.Name(...) from integrating code.
func (p *HomebrewPublisher) Name() string {
	return "homebrew"
}

// Publish publishes the release to Homebrew.
// Usage example: call value.Publish(...) from integrating code.
func (p *HomebrewPublisher) Publish(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig, dryRun bool) error {
	// Parse config
	cfg := p.parseConfig(pubCfg, relCfg)

	// Validate configuration
	if cfg.Tap == "" && (cfg.Official == nil || !cfg.Official.Enabled) {
		return coreerr.E("homebrew.Publish", "tap is required (set publish.homebrew.tap in config)", nil)
	}

	// Get repository and project info
	repo := ""
	if relCfg != nil {
		repo = relCfg.GetRepository()
	}
	if repo == "" {
		detectedRepo, err := detectRepository(release.ProjectDir)
		if err != nil {
			return coreerr.E("homebrew.Publish", "could not determine repository", err)
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

	formulaName := cfg.Formula
	if formulaName == "" {
		formulaName = projectName
	}

	// Strip leading 'v' from version
	version := core.TrimPrefix(release.Version, "v")

	// Build checksums map from artifacts
	checksums := buildChecksumMap(release.Artifacts)

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

// ChecksumMap holds checksums for different platform/arch combinations.
// Usage example: declare a value of type publishers.ChecksumMap in integrating code.
type ChecksumMap struct {
	DarwinAmd64  string
	DarwinArm64  string
	LinuxAmd64   string
	LinuxArm64   string
	WindowsAmd64 string
	WindowsArm64 string
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
func (p *HomebrewPublisher) dryRunPublish(m coreio.Medium, data homebrewTemplateData, cfg HomebrewConfig) error {
	publisherPrintln()
	publisherPrintln("=== DRY RUN: Homebrew Publish ===")
	publisherPrintln()
	publisherPrint("Formula:    %s", data.FormulaClass)
	publisherPrint("Version:    %s", data.Version)
	publisherPrint("Tap:        %s", cfg.Tap)
	publisherPrint("Repository: %s", data.Repository)
	publisherPrintln()

	// Generate and show formula
	formula, err := p.renderTemplate(m, "templates/homebrew/formula.rb.tmpl", data)
	if err != nil {
		return coreerr.E("homebrew.dryRunPublish", "failed to render template", err)
	}
	publisherPrintln("Generated formula.rb:")
	publisherPrintln("---")
	publisherPrintln(formula)
	publisherPrintln("---")
	publisherPrintln()

	if cfg.Tap != "" {
		publisherPrint("Would commit to tap: %s", cfg.Tap)
	}
	if cfg.Official != nil && cfg.Official.Enabled {
		output := cfg.Official.Output
		if output == "" {
			output = "dist/homebrew"
		}
		publisherPrint("Would write files for official PR to: %s", output)
	}
	publisherPrintln()
	publisherPrintln("=== END DRY RUN ===")

	return nil
}

// executePublish creates the formula and commits to tap.
func (p *HomebrewPublisher) executePublish(ctx context.Context, projectDir string, data homebrewTemplateData, cfg HomebrewConfig, release *Release) error {
	// Generate formula
	formula, err := p.renderTemplate(release.FS, "templates/homebrew/formula.rb.tmpl", data)
	if err != nil {
		return coreerr.E("homebrew.Publish", "failed to render formula", err)
	}

	// If official config is enabled, write to output directory
	if cfg.Official != nil && cfg.Official.Enabled {
		output := cfg.Official.Output
		if output == "" {
			output = ax.Join(projectDir, "dist", "homebrew")
		} else if !ax.IsAbs(output) {
			output = ax.Join(projectDir, output)
		}

		if err := release.FS.EnsureDir(output); err != nil {
			return coreerr.E("homebrew.Publish", "failed to create output directory", err)
		}

		formulaPath := ax.Join(output, core.Sprintf("%s.rb", core.Lower(data.FormulaClass)))
		if err := release.FS.Write(formulaPath, formula); err != nil {
			return coreerr.E("homebrew.Publish", "failed to write formula", err)
		}
		publisherPrint("Wrote Homebrew formula for official PR: %s", formulaPath)
	}

	// If tap is configured, commit to it
	if cfg.Tap != "" {
		if err := p.commitToTap(ctx, cfg.Tap, data, formula); err != nil {
			return err
		}
	}

	return nil
}

// commitToTap commits the formula to the tap repository.
func (p *HomebrewPublisher) commitToTap(ctx context.Context, tap string, data homebrewTemplateData, formula string) error {
	// Clone tap repo to temp directory
	tmpDir, err := ax.TempDir("homebrew-tap-*")
	if err != nil {
		return coreerr.E("homebrew.commitToTap", "failed to create temp directory", err)
	}
	defer func() { _ = ax.RemoveAll(tmpDir) }()

	// Clone the tap
	publisherPrint("Cloning tap %s...", tap)
	if err := publisherRun(ctx, "", nil, "gh", "repo", "clone", tap, tmpDir, "--", "--depth=1"); err != nil {
		return coreerr.E("homebrew.commitToTap", "failed to clone tap", err)
	}

	// Ensure Formula directory exists
	formulaDir := ax.Join(tmpDir, "Formula")
	if err := coreio.Local.EnsureDir(formulaDir); err != nil {
		return coreerr.E("homebrew.commitToTap", "failed to create Formula directory", err)
	}

	// Write formula
	formulaPath := ax.Join(formulaDir, core.Sprintf("%s.rb", core.Lower(data.FormulaClass)))
	if err := coreio.Local.Write(formulaPath, formula); err != nil {
		return coreerr.E("homebrew.commitToTap", "failed to write formula", err)
	}

	// Git add, commit, push
	commitMsg := core.Sprintf("Update %s to %s", data.FormulaClass, data.Version)

	if err := ax.ExecDir(ctx, tmpDir, "git", "add", "."); err != nil {
		return coreerr.E("homebrew.commitToTap", "git add failed", err)
	}

	if err := publisherRun(ctx, tmpDir, nil, "git", "commit", "-m", commitMsg); err != nil {
		return coreerr.E("homebrew.commitToTap", "git commit failed", err)
	}

	if err := publisherRun(ctx, tmpDir, nil, "git", "push"); err != nil {
		return coreerr.E("homebrew.commitToTap", "git push failed", err)
	}

	publisherPrint("Updated Homebrew tap: %s", tap)
	return nil
}

// renderTemplate renders an embedded template with the given data.
func (p *HomebrewPublisher) renderTemplate(m coreio.Medium, name string, data homebrewTemplateData) (string, error) {
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
		content, err = homebrewTemplates.ReadFile(name)
		if err != nil {
			return "", coreerr.E("homebrew.renderTemplate", "failed to read template "+name, err)
		}
	}

	tmpl, err := template.New(ax.Base(name)).Parse(string(content))
	if err != nil {
		return "", coreerr.E("homebrew.renderTemplate", "failed to parse template "+name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", coreerr.E("homebrew.renderTemplate", "failed to execute template "+name, err)
	}

	return buf.String(), nil
}

// toFormulaClass converts a package name to a Ruby class name.
func toFormulaClass(name string) string {
	// Convert kebab-case to PascalCase
	parts := core.Split(name, "-")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = core.Upper(part[:1]) + part[1:]
		}
	}
	return core.Join("", parts...)
}

// buildChecksumMap extracts checksums from artifacts into a structured map.
func buildChecksumMap(artifacts []build.Artifact) ChecksumMap {
	checksums := ChecksumMap{}

	for _, a := range artifacts {
		// Parse artifact name to determine platform
		name := ax.Base(a.Path)
		checksum := a.Checksum

		switch {
		case core.Contains(name, "darwin-amd64"):
			checksums.DarwinAmd64 = checksum
		case core.Contains(name, "darwin-arm64"):
			checksums.DarwinArm64 = checksum
		case core.Contains(name, "linux-amd64"):
			checksums.LinuxAmd64 = checksum
		case core.Contains(name, "linux-arm64"):
			checksums.LinuxArm64 = checksum
		case core.Contains(name, "windows-amd64"):
			checksums.WindowsAmd64 = checksum
		case core.Contains(name, "windows-arm64"):
			checksums.WindowsArm64 = checksum
		}
	}

	return checksums
}
