package generators

import (
	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
)

// These tests drive the TypeScript output-staging pipeline end to end without
// any external generator: finalizeTypeScriptOutput, the recursive copy
// helpers, the src-placement decision, and package.json synthesis. The
// existing suite reached these only when openapi-generator/npx were installed.

func writeStagingFile(t *core.T, dir, rel, content string) {
	t.Helper()
	path := ax.Join(dir, rel)
	core.AssertTrue(t, ax.MkdirAll(ax.Dir(path), 0o755).OK)
	core.AssertTrue(t, ax.WriteFile(path, []byte(content), 0o644).OK)
}

func TestTypeScript_ShouldPlaceInSrc_Good(t *core.T) {
	// TypeScript source files and the known source directories land in src.
	core.AssertTrue(t, shouldPlaceTypeScriptInSrc("client.ts", false))
	core.AssertTrue(t, shouldPlaceTypeScriptInSrc("Widget.TSX", false))
	core.AssertTrue(t, shouldPlaceTypeScriptInSrc("models", true))
	core.AssertTrue(t, shouldPlaceTypeScriptInSrc("apis", true))
}

func TestTypeScript_ShouldPlaceInSrc_Bad(t *core.T) {
	// Non-source files and unknown directories stay at the package root.
	core.AssertFalse(t, shouldPlaceTypeScriptInSrc("README.md", false))
	core.AssertFalse(t, shouldPlaceTypeScriptInSrc("package.json", false))
	core.AssertFalse(t, shouldPlaceTypeScriptInSrc("docs", true))
	core.AssertFalse(t, shouldPlaceTypeScriptInSrc("", false))
	core.AssertFalse(t, shouldPlaceTypeScriptInSrc("   ", true))
}

func TestTypeScript_FinalizeOutput_Good(t *core.T) {
	root := t.TempDir()
	staging := ax.Join(root, "staging")
	output := ax.Join(root, "out")

	// A representative generator staging tree: a src dir, a placed-in-src model
	// directory, a root-level source file, and a non-source doc file.
	writeStagingFile(t, staging, "src/index.ts", "export {};\n")
	writeStagingFile(t, staging, "models/Pet.ts", "export interface Pet {}\n")
	writeStagingFile(t, staging, "client.ts", "export const c = 1;\n")
	writeStagingFile(t, staging, "README.md", "# generated\n")

	result := finalizeTypeScriptOutput(staging, Options{
		OutputDir:   output,
		PackageName: "@scope/client",
		Version:     "2.1.0",
	})
	core.AssertTrue(t, result.OK, result.Error())

	core.AssertTrue(t, ax.IsFile(ax.Join(output, "src", "index.ts")))
	core.AssertTrue(t, ax.IsFile(ax.Join(output, "src", "models", "Pet.ts")))
	core.AssertTrue(t, ax.IsFile(ax.Join(output, "src", "client.ts")))
	core.AssertTrue(t, ax.IsFile(ax.Join(output, "README.md")))

	manifest := ax.ReadFile(ax.Join(output, "package.json"))
	core.AssertTrue(t, manifest.OK)
	parsed := map[string]any{}
	core.AssertTrue(t, core.JSONUnmarshal(manifest.Value.([]byte), &parsed).OK)
	core.AssertEqual(t, "@scope/client", parsed["name"])
	core.AssertEqual(t, "2.1.0", parsed["version"])
	core.AssertEqual(t, "module", parsed["type"])
}

func TestTypeScript_FinalizeOutput_DefaultsMetadata_Ugly(t *core.T) {
	root := t.TempDir()
	staging := ax.Join(root, "staging")
	output := ax.Join(root, "named-output")

	// No package name or version: name falls back to the output dir base and
	// version defaults to 0.0.0. An index.ts also seeds types + exports.
	writeStagingFile(t, staging, "src/index.ts", "export {};\n")

	result := finalizeTypeScriptOutput(staging, Options{OutputDir: output})
	core.AssertTrue(t, result.OK, result.Error())

	manifest := ax.ReadFile(ax.Join(output, "package.json"))
	core.AssertTrue(t, manifest.OK)
	parsed := map[string]any{}
	core.AssertTrue(t, core.JSONUnmarshal(manifest.Value.([]byte), &parsed).OK)
	core.AssertEqual(t, "named-output", parsed["name"])
	core.AssertEqual(t, "0.0.0", parsed["version"])
	core.AssertEqual(t, "./src/index.ts", parsed["types"])
}

func TestTypeScript_FinalizeOutput_Bad(t *core.T) {
	// An empty output dir is rejected before any filesystem work.
	result := finalizeTypeScriptOutput(t.TempDir(), Options{OutputDir: "   "})
	core.AssertFalse(t, result.OK)
	core.AssertTrue(t, core.Contains(result.Error(), "output dir is required"))
}

func TestTypeScript_EnsurePackageMetadata_MergesExisting_Ugly(t *core.T) {
	output := t.TempDir()
	// A pre-existing manifest with custom fields is preserved while name/version
	// are overlaid.
	existing := []byte(`{"name":"old","scripts":{"build":"tsc"},"version":"9.9.9"}`)
	core.AssertTrue(t, ax.WriteFile(ax.Join(output, "package.json"), existing, 0o644).OK)

	result := ensureTypeScriptPackageMetadata(output, "fresh", "")
	core.AssertTrue(t, result.OK, result.Error())

	manifest := ax.ReadFile(ax.Join(output, "package.json"))
	parsed := map[string]any{}
	core.AssertTrue(t, core.JSONUnmarshal(manifest.Value.([]byte), &parsed).OK)
	core.AssertEqual(t, "fresh", parsed["name"])
	// Existing version is kept because none was supplied.
	core.AssertEqual(t, "9.9.9", parsed["version"])
	// Untouched custom fields survive the merge.
	scripts, ok := parsed["scripts"].(map[string]any)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "tsc", scripts["build"])
}

func TestTypeScript_CopyDirectoryContents_Bad(t *core.T) {
	// A missing source directory fails the listing step.
	result := copyTypeScriptDirectoryContents(ax.Join(t.TempDir(), "nope"), t.TempDir())
	core.AssertFalse(t, result.OK)
	core.AssertTrue(t, core.Contains(result.Error(), "failed to list source dir"))
}

func TestTypeScript_CopyPath_Bad(t *core.T) {
	// Statting a non-existent source path fails.
	result := copyTypeScriptPath(ax.Join(t.TempDir(), "missing.ts"), ax.Join(t.TempDir(), "out.ts"))
	core.AssertFalse(t, result.OK)
	core.AssertTrue(t, core.Contains(result.Error(), "failed to stat source path"))
}

func TestTypeScript_ResolveNativeCli_Fallback_Ugly(t *core.T) {
	// openapi-typescript-codegen is not a PATH-installed tool in CI, so the
	// fallback path is deterministically taken. (npx is intentionally not
	// tested this way: it is commonly present on PATH and would resolve there
	// rather than via the fabricated fallback.)
	g := NewTypeScriptGenerator()
	fallback := ax.Join(t.TempDir(), "openapi-typescript-codegen")
	core.AssertTrue(t, ax.WriteString(fallback, "#!/bin/sh\n", 0o755).OK)
	resolved := g.resolveNativeCli("/no/such/tool", fallback)
	core.AssertTrue(t, resolved.OK)
	core.AssertEqual(t, fallback, resolved.Value.(string))
}

func TestTypeScript_ResolveNativeCli_AllMissing_Bad(t *core.T) {
	g := NewTypeScriptGenerator()
	resolved := g.resolveNativeCli("/no/such/tool-a", "/no/such/tool-b")
	core.AssertFalse(t, resolved.OK)
	core.AssertTrue(t, core.Contains(resolved.Error(), "openapi-typescript-codegen not found"))
}

func TestGenerator_LanguagesIter_EarlyBreak_Ugly(t *core.T) {
	registry := NewRegistry()
	registry.Register(NewGoGenerator())
	registry.Register(NewPythonGenerator())
	registry.Register(NewPHPGenerator())

	// Breaking out of the range loop drives the yield-returns-false early-return
	// branch of LanguagesIter.
	seen := 0
	for range registry.LanguagesIter() {
		seen++
		break
	}
	core.AssertEqual(t, 1, seen)
}
