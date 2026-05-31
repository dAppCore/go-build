package generators

import (
	"context"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
)

// allGenerators returns one instance of each language generator so the shared
// error-path contract can be asserted uniformly. These branches are reachable
// without any external CLI (oapi-codegen, openapi-generator, docker, npx) being
// installed, which is why the existing suite skipped them.
func allGenerators() []Generator {
	return []Generator{
		NewGoGenerator(),
		NewPythonGenerator(),
		NewPHPGenerator(),
		NewTypeScriptGenerator(),
	}
}

func TestGenerators_Generate_CancelledContext_Bad(t *core.T) {
	for _, g := range allGenerators() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		result := g.Generate(ctx, Options{
			SpecPath:    "spec.yaml",
			OutputDir:   ax.Join(t.TempDir(), "out"),
			PackageName: "client",
		})
		core.AssertFalse(t, result.OK, g.Language())
		core.AssertTrue(t, core.Contains(result.Error(), "cancelled"), g.Language())
	}
}

func TestGenerators_Generate_OutputDirBlocked_Bad(t *core.T) {
	for _, g := range allGenerators() {
		// A regular file standing where the output directory should be makes the
		// MkdirAll step fail, driving the create-output-dir error branch.
		blocker := ax.Join(t.TempDir(), "blocker")
		core.AssertTrue(t, ax.WriteString(blocker, "i am a file", 0o644).OK, g.Language())

		result := g.Generate(context.Background(), Options{
			SpecPath:    "spec.yaml",
			OutputDir:   ax.Join(blocker, "out"),
			PackageName: "client",
		})
		core.AssertFalse(t, result.OK, g.Language())
	}
}
