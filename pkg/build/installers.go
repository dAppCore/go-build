package build

import buildinstallers "dappco.re/go/core/build/pkg/build/installers"

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
	// VariantDev generates dev.sh — installs core CLI + pulls the core-dev LinuxKit image.
	VariantDev InstallerVariant = buildinstallers.VariantDev
)

// InstallerConfig supplies template values for installer generation.
type InstallerConfig = buildinstallers.InstallerConfig

// GenerateInstaller renders a single installer script variant.
func GenerateInstaller(variant InstallerVariant, cfg InstallerConfig) (string, error) {
	return buildinstallers.GenerateInstaller(variant, cfg)
}

// GenerateAll renders every installer script variant.
func GenerateAll(cfg InstallerConfig) (map[string]string, error) {
	return buildinstallers.GenerateAll(cfg)
}

// InstallerVariants returns the supported variants in stable output order.
func InstallerVariants() []InstallerVariant {
	return buildinstallers.Variants()
}

// InstallerOutputName returns the filename emitted for a variant.
func InstallerOutputName(variant InstallerVariant) string {
	return buildinstallers.OutputName(variant)
}
