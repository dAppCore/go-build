package builders

import (
	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/io"
)

type packageJSONManifest struct {
	PackageManager string `json:"packageManager"`
}

// detectDeclaredPackageManager reads package.json and returns the declared package manager.
//
// manager := detectDeclaredPackageManager(io.Local, ".")
func detectDeclaredPackageManager(fs io.Medium, dir string) string {
	contentResult := fs.Read(ax.Join(dir, "package.json"))
	if !contentResult.OK {
		return ""
	}
	content := contentResult.Value.(string)

	var manifest packageJSONManifest
	decoded := ax.JSONUnmarshal([]byte(content), &manifest)
	if !decoded.OK {
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
