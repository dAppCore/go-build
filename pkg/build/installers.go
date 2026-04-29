package build

import (
	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	buildinstallers "dappco.re/go/build/pkg/build/installers"
)

// InstallerVariant identifies an installer script profile.
type InstallerVariant = buildinstallers.InstallerVariant

const (
	// VariantFull generates setup.sh — full installer with PATH setup and shell completions.
	VariantFull InstallerVariant = buildinstallers.VariantFull
	// VariantCI generates ci.sh — minimal download-only installer for CI environments.
	VariantCI InstallerVariant = buildinstallers.VariantCI
	// VariantPHP generates php.sh — installs core CLI + FrankenPHP + Composer.
	VariantPHP InstallerVariant = buildinstallers.VariantPHP
	// VariantGo generates go.sh — installs core CLI + Go toolchain + gopls.
	VariantGo InstallerVariant = buildinstallers.VariantGo
	// VariantAgent generates agent.sh — installs core CLI + core-agent + Claude Code.
	VariantAgent InstallerVariant = buildinstallers.VariantAgent
	// VariantAgentic is the RFC-documented alias for the AI agent installer variant.
	VariantAgentic InstallerVariant = buildinstallers.VariantAgentic
	// VariantDev generates dev.sh — installs core CLI + pulls the core-dev LinuxKit image.
	VariantDev InstallerVariant = buildinstallers.VariantDev
)

// GenerateInstallerScript renders a single installer script variant from the
// release version and repository.
//
//	script, err := build.GenerateInstallerScript(build.VariantCI, "v1.2.3", "dappcore/core")
//	// script starts with the ci.sh template rendered for core binaries
func GenerateInstallerScript(variant InstallerVariant, version, repo string) (string, error) {
	return buildinstallers.GenerateInstaller(variant, installerConfig(version, repo))
}

// GenerateInstaller is the backwards-compatible alias for GenerateInstallerScript.
func GenerateInstaller(variant InstallerVariant, version, repo string) (string, error) {
	return GenerateInstallerScript(variant, version, repo)
}

// GenerateAllInstallerScripts renders every installer script variant from the
// release version and repository.
//
//	scripts, err := build.GenerateAllInstallerScripts("v1.2.3", "dappcore/core")
//	// scripts["setup.sh"], scripts["ci.sh"], scripts["go.sh"], ...
func GenerateAllInstallerScripts(version, repo string) (map[string]string, error) {
	return buildinstallers.GenerateAll(installerConfig(version, repo))
}

// GenerateAll is the backwards-compatible alias for GenerateAllInstallerScripts.
func GenerateAll(version, repo string) (map[string]string, error) {
	return GenerateAllInstallerScripts(version, repo)
}

// InstallerVariants returns the supported variants in stable output order.
func InstallerVariants() []InstallerVariant {
	return buildinstallers.Variants()
}

// InstallerOutputName returns the filename emitted for a variant.
func InstallerOutputName(variant InstallerVariant) string {
	return buildinstallers.OutputName(variant)
}

func installerConfig(version, repo string) buildinstallers.InstallerConfig {
	repo = core.Trim(repo)
	binaryName := ""
	if repo != "" {
		binaryName = core.TrimSuffix(ax.Base(repo), ".git")
		if binaryName == "" {
			binaryName = repo
		}
	}

	return buildinstallers.InstallerConfig{
		Version:    core.Trim(version),
		Repo:       repo,
		BinaryName: binaryName,
	}
}
