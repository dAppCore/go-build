package builders

import (
	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/io"
)

type packageJSONManifest struct {
	PackageManager string `json:"packageManager"`
}

// detectDeclaredPackageManager reads package.json and returns the declared package manager.
//
// manager := detectDeclaredPackageManager(io.Local, ".")
func detectDeclaredPackageManager(fs io.Medium, dir string) string {
	content, err := fs.Read(ax.Join(dir, "package.json"))
	if err != nil {
		return ""
	}

	var manifest packageJSONManifest
	if err := ax.JSONUnmarshal([]byte(content), &manifest); err != nil {
		return ""
	}

	return normalisePackageManager(manifest.PackageManager)
}

// normalisePackageManager trims any pinned version from a packageManager declaration.
//
// manager := normalisePackageManager("pnpm@9.12.0")
func normalisePackageManager(value string) string {
	value = core.Trim(value)
	if value == "" {
		return ""
	}

	parts := core.SplitN(value, "@", 2)
	manager := parts[0]

	switch manager {
	case "bun", "pnpm", "yarn", "npm":
		return manager
	default:
		return ""
	}
}
