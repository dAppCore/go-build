package builders

import (
	"github.com/dappcore/go"
	"github.com/dappcore/go-build/go/internal/ax"
	storage "github.com/dappcore/go-build/go/pkg/storage"
)

type packageJSONManifest struct {
	PackageManager string `json:"packageManager"`
}

// detectDeclaredPackageManager reads package.json and returns the declared package manager.
//
// manager := detectDeclaredPackageManager(storage.Local, ".")
func detectDeclaredPackageManager(fs storage.Medium, dir string) string {
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
