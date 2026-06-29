// Package installers generates installer shell scripts for Core CLI releases.
// Each variant targets a specific install profile (full, CI, PHP, Go, agent, dev).
package installers

import (
	"embed"         // Note: AX-6 — embeds installer templates into the package.
	"regexp"        // Note: AX-6 — validates release versions with a precompiled pattern.
	"text/template" // Note: AX-6 — renders shell installer templates.

	"dappco.re/go" // Note: AX-6 — provides approved string helpers and template writer construction.
)

//go:embed templates/*.tmpl
var installerTemplates embed.FS

var safeInstallerVersion = regexp.MustCompile(`^[A-Za-z0-9._+-]+$`)

// DefaultScriptBaseURL is the RFC-documented CDN origin for generated
// installer scripts.
const DefaultScriptBaseURL = "https://lthn.sh"

// InstallerVariant represents an installer script variant.
//
//	var v installers.InstallerVariant = installers.VariantFull
type InstallerVariant string

const (
	// VariantFull generates setup.sh — full installer with PATH setup and shell completions.
	VariantFull InstallerVariant = "full"
	// VariantCI generates ci.sh — minimal download-only installer for CI environments.
	VariantCI InstallerVariant = "ci"
	// VariantPHP generates php.sh — installs core CLI + FrankenPHP + Composer (~50MB).
	VariantPHP InstallerVariant = "php"
	// VariantGo generates go.sh — installs core CLI + Go toolchain + gopls (~200MB).
	VariantGo InstallerVariant = "go"
	// VariantAgent generates agent.sh — installs core CLI + core-agent + Claude Code (~30MB).
	VariantAgent InstallerVariant = "agent"
	// VariantAgentic is the RFC-documented alias for the AI agent installer variant.
	VariantAgentic InstallerVariant = VariantAgent
	// VariantDev generates dev.sh — installs core CLI + pulls core-dev LinuxKit image (~500MB).
	VariantDev InstallerVariant = "dev"
)

var installerVariantOrder = []InstallerVariant{
	VariantFull,
	VariantCI,
	VariantPHP,
	VariantGo,
	VariantAgent,
	VariantDev,
}

// variantTemplates maps each InstallerVariant to its embedded template filename and output script name.
var variantTemplates = map[InstallerVariant]struct {
	tmpl   string
	output string
}{
	VariantFull:  {tmpl: "templates/setup.sh.tmpl", output: "setup.sh"},
	VariantCI:    {tmpl: "templates/ci.sh.tmpl", output: "ci.sh"},
	VariantPHP:   {tmpl: "templates/php.sh.tmpl", output: "php.sh"},
	VariantGo:    {tmpl: "templates/go.sh.tmpl", output: "go.sh"},
	VariantAgent: {tmpl: "templates/agent.sh.tmpl", output: "agent.sh"},
	VariantDev:   {tmpl: "templates/dev.sh.tmpl", output: "dev.sh"},
}

// Variants returns the supported installer variants in stable output order.
func Variants() []InstallerVariant {
	return append([]InstallerVariant(nil), installerVariantOrder...)
}

// OutputName returns the generated script filename for a variant.
func OutputName(variant InstallerVariant) string {
	entry, ok := variantTemplates[canonicalVariant(variant)]
	if !ok {
		return ""
	}
	return entry.output
}

// InstallerConfig holds the values injected into installer script templates.
//
//	cfg := installers.InstallerConfig{
//	    Version:    "v1.2.3",
//	    Repo:       "dappcore/core",
//	    BinaryName: "core",
//	}
type InstallerConfig struct {
	// Version is the release tag (e.g. "v1.2.3").
	Version string
	// Repo is the GitHub repository in "owner/name" format (e.g. "dappcore/core").
	Repo string
	// BinaryName is the name of the installed binary (e.g. "core").
	BinaryName string
	// ScriptBaseURL is the public base URL that hosts the generated installer scripts.
	// Empty values default to the RFC CDN origin.
	ScriptBaseURL string
}

// GenerateInstaller renders an installer script for the given variant.
//
//	// RFC-shaped form:
//	script, err := installers.GenerateInstaller(installers.VariantCI, "v1.2.3", "dappcore/core")
//
//	// Rich form with explicit binary name and script host:
//	script, err := installers.GenerateInstaller(installers.VariantCI, installers.InstallerConfig{
//	    Version: "v1.2.3", Repo: "dappcore/core", BinaryName: "core",
//	})
func GenerateInstaller(variant InstallerVariant, args ...any) core.Result {
	cfgResult := normalizeInstallerArgs(args...)
	if !cfgResult.OK {
		return cfgResult
	}
	cfg := cfgResult.Value.(InstallerConfig)

	variant = canonicalVariant(variant)
	valid := validateInstallerVersion(cfg.Version)
	if !valid.OK {
		return core.Fail(core.E("installers.GenerateInstaller", "version is not a safe release identifier", core.NewError(valid.Error())))
	}

	entry, ok := variantTemplates[variant]
	if !ok {
		return core.Fail(core.E("installers.GenerateInstaller", "unknown variant: "+string(variant), nil))
	}

	raw, err := installerTemplates.ReadFile(entry.tmpl)
	if err != nil {
		return core.Fail(core.E("installers.GenerateInstaller", "failed to read template "+entry.tmpl, err))
	}

	tmpl, err := template.New(entry.output).Funcs(template.FuncMap{
		"shellQuote": shellQuote,
	}).Parse(string(raw))
	if err != nil {
		return core.Fail(core.E("installers.GenerateInstaller", "failed to parse template "+entry.tmpl, err))
	}

	// Note: AX-6 — core.NewBuffer is unavailable in the pinned core module;
	// core.NewBuilder is the available Core-owned writer.
	buf := core.NewBuilder()
	if err := tmpl.Execute(buf, cfg); err != nil {
		return core.Fail(core.E("installers.GenerateInstaller", "failed to render template "+entry.tmpl, err))
	}

	return core.Ok(buf.String())
}

// GenerateAll renders all installer variants and returns a map of output filename → script content.
//
//	// RFC-shaped form:
//	scripts, err := installers.GenerateAll("v1.2.3", "dappcore/core")
//
//	// Rich form with explicit binary name and script host:
//	scripts, err := installers.GenerateAll(installers.InstallerConfig{
//	    Version: "v1.2.3", Repo: "dappcore/core", BinaryName: "core",
//	})
//	for name, content := range scripts {
//	    // name: "setup.sh", content: "#!/usr/bin/env bash\n..."
//	}
func GenerateAll(args ...any) core.Result {
	cfgResult := normalizeInstallerArgs(args...)
	if !cfgResult.OK {
		return cfgResult
	}
	cfg := cfgResult.Value.(InstallerConfig)

	valid := validateInstallerVersion(cfg.Version)
	if !valid.OK {
		return core.Fail(core.E("installers.GenerateAll", "version is not a safe release identifier", core.NewError(valid.Error())))
	}

	out := make(map[string]string, len(installerVariantOrder))

	for _, variant := range installerVariantOrder {
		entry := variantTemplates[variant]
		script := GenerateInstaller(variant, cfg)
		if !script.OK {
			return core.Fail(core.E("installers.GenerateAll", "failed to generate variant "+string(variant), core.NewError(script.Error())))
		}
		out[entry.output] = script.Value.(string)
	}

	return core.Ok(out)
}

func normalizeInstallerArgs(args ...any) core.Result {
	switch len(args) {
	case 1:
		switch cfg := args[0].(type) {
		case InstallerConfig:
			return core.Ok(normalizeInstallerConfig(cfg))
		case *InstallerConfig:
			if cfg == nil {
				return core.Ok(normalizeInstallerConfig(InstallerConfig{}))
			}
			return core.Ok(normalizeInstallerConfig(*cfg))
		default:
			return core.Fail(core.E("installers.normalizeInstallerArgs", "expected InstallerConfig or *InstallerConfig", nil))
		}
	case 2:
		version, ok := args[0].(string)
		if !ok {
			return core.Fail(core.E("installers.normalizeInstallerArgs", "version must be a string", nil))
		}
		repo, ok := args[1].(string)
		if !ok {
			return core.Fail(core.E("installers.normalizeInstallerArgs", "repo must be a string", nil))
		}
		return core.Ok(normalizeInstallerConfig(InstallerConfig{
			Version:    version,
			Repo:       repo,
			BinaryName: defaultInstallerBinaryName(repo),
		}))
	default:
		return core.Fail(core.E("installers.normalizeInstallerArgs", "expected either InstallerConfig or version/repo arguments", nil))
	}
}

func normalizeInstallerConfig(cfg InstallerConfig) InstallerConfig {
	baseURL := trimTrailingSlashes(core.Trim(cfg.ScriptBaseURL))
	if baseURL == "" {
		baseURL = DefaultScriptBaseURL
	}
	cfg.ScriptBaseURL = baseURL
	if core.Trim(cfg.BinaryName) == "" {
		cfg.BinaryName = defaultInstallerBinaryName(cfg.Repo)
	}
	return cfg
}

func defaultInstallerBinaryName(repo string) string {
	repo = core.Trim(repo)
	if repo == "" {
		return ""
	}

	parts := core.Split(core.Replace(repo, "\\", "/"), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" {
			return parts[i]
		}
	}

	return ""
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}

	return "'" + core.Replace(value, "'", `'"'"'`) + "'"
}

func canonicalVariant(variant InstallerVariant) InstallerVariant {
	normalized := InstallerVariant(core.Lower(core.Trim(string(variant))))
	if normalized == "agentic" {
		return VariantAgent
	}
	return normalized
}

func validateInstallerVersion(version string) core.Result {
	trimmed := core.Trim(version)
	if trimmed == "" {
		return core.Ok(nil)
	}
	if version != trimmed {
		return core.Fail(core.E("installers.validateInstallerVersion", "version contains unsupported whitespace", nil))
	}
	if !safeInstallerVersion.MatchString(version) {
		return core.Fail(core.E("installers.validateInstallerVersion", "version contains unsupported characters", nil))
	}

	return core.Ok(nil)
}

func trimTrailingSlashes(value string) string {
	for core.HasSuffix(value, "/") {
		value = core.TrimSuffix(value, "/")
	}
	return value
}
