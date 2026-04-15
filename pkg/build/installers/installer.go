// Package installers generates installer shell scripts for Core CLI releases.
// Each variant targets a specific install profile (full, CI, PHP, Go, agent, dev).
package installers

import (
	"bytes"
	"embed"
	"regexp"
	"strings"
	"text/template"

	coreerr "dappco.re/go/core/log"
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
//	script, err := installers.GenerateInstaller(installers.VariantCI, installers.InstallerConfig{
//	    Version: "v1.2.3", Repo: "dappcore/core", BinaryName: "core",
//	})
func GenerateInstaller(variant InstallerVariant, cfg InstallerConfig) (string, error) {
	cfg = normalizeInstallerConfig(cfg)
	variant = canonicalVariant(variant)
	if err := validateInstallerVersion(cfg.Version); err != nil {
		return "", coreerr.E("installers.GenerateInstaller", "version is not a safe release identifier", err)
	}

	entry, ok := variantTemplates[variant]
	if !ok {
		return "", coreerr.E("installers.GenerateInstaller", "unknown variant: "+string(variant), nil)
	}

	raw, err := installerTemplates.ReadFile(entry.tmpl)
	if err != nil {
		return "", coreerr.E("installers.GenerateInstaller", "failed to read template "+entry.tmpl, err)
	}

	tmpl, err := template.New(entry.output).Funcs(template.FuncMap{
		"shellQuote": shellQuote,
	}).Parse(string(raw))
	if err != nil {
		return "", coreerr.E("installers.GenerateInstaller", "failed to parse template "+entry.tmpl, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return "", coreerr.E("installers.GenerateInstaller", "failed to render template "+entry.tmpl, err)
	}

	return buf.String(), nil
}

// GenerateAll renders all installer variants and returns a map of output filename → script content.
//
//	scripts, err := installers.GenerateAll(installers.InstallerConfig{
//	    Version: "v1.2.3", Repo: "dappcore/core", BinaryName: "core",
//	})
//	for name, content := range scripts {
//	    // name: "setup.sh", content: "#!/usr/bin/env bash\n..."
//	}
func GenerateAll(cfg InstallerConfig) (map[string]string, error) {
	if err := validateInstallerVersion(cfg.Version); err != nil {
		return nil, coreerr.E("installers.GenerateAll", "version is not a safe release identifier", err)
	}

	out := make(map[string]string, len(installerVariantOrder))

	for _, variant := range installerVariantOrder {
		entry := variantTemplates[variant]
		script, err := GenerateInstaller(variant, cfg)
		if err != nil {
			return nil, coreerr.E("installers.GenerateAll", "failed to generate variant "+string(variant), err)
		}
		out[entry.output] = script
	}

	return out, nil
}

func normalizeInstallerConfig(cfg InstallerConfig) InstallerConfig {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.ScriptBaseURL), "/")
	if baseURL == "" {
		baseURL = DefaultScriptBaseURL
	}
	cfg.ScriptBaseURL = baseURL
	return cfg
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}

	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func canonicalVariant(variant InstallerVariant) InstallerVariant {
	normalized := InstallerVariant(strings.ToLower(strings.TrimSpace(string(variant))))
	if normalized == "agentic" {
		return VariantAgent
	}
	return normalized
}

func validateInstallerVersion(version string) error {
	version = strings.TrimSpace(version)
	if version == "" {
		return nil
	}
	if !safeInstallerVersion.MatchString(version) {
		return coreerr.E("installers.validateInstallerVersion", "version contains unsupported characters", nil)
	}

	return nil
}
