// Package generators provides SDK code generators for different languages.
package generators

import (
	"context"
	"iter"
	"runtime"
	"slices"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
)

// Options holds common generation options.
//
// opts := generators.Options{SpecPath: "docs/openapi.yaml", OutputDir: "sdk/typescript", PackageName: "@host-uk/api-client", Version: "1.0.0"}
type Options struct {
	// SpecPath is the path to the OpenAPI spec file.
	SpecPath string
	// OutputDir is where to write the generated SDK.
	OutputDir string
	// PackageName is the package/module name.
	PackageName string
	// Version is the SDK version.
	Version string
}

// Generator defines the interface for SDK generators.
//
// gen := generators.NewTypeScriptGenerator()
// err := gen.Generate(ctx, opts)
type Generator interface {
	// Language returns the generator's target language identifier.
	Language() string

	// Generate creates SDK from OpenAPI spec.
	Generate(ctx context.Context, opts Options) error

	// Available checks if generator dependencies are installed.
	Available() bool

	// Install returns instructions for installing the generator.
	Install() string
}

// Registry holds available generators.
//
// r := generators.NewRegistry()
type Registry struct {
	generators map[string]Generator
}

// NewRegistry creates a registry with all available generators.
//
// r := generators.NewRegistry()
func NewRegistry() *Registry {
	r := &Registry{
		generators: make(map[string]Generator),
	}

	// Register the built-in generators so callers get a ready-to-use registry.
	r.Register(NewTypeScriptGenerator())
	r.Register(NewPythonGenerator())
	r.Register(NewGoGenerator())
	r.Register(NewPHPGenerator())

	return r
}

// Get returns a generator by language.
//
// gen, ok := r.Get("typescript")
func (r *Registry) Get(lang string) (Generator, bool) {
	g, ok := r.generators[lang]
	return g, ok
}

// Register adds a generator to the registry.
//
// r.Register(generators.NewTypeScriptGenerator())
func (r *Registry) Register(g Generator) {
	r.generators[g.Language()] = g
}

// Languages returns all registered language identifiers.
//
// langs := r.Languages() // → ["go", "php", "python", "typescript"]
func (r *Registry) Languages() []string {
	var languages []string
	for lang := range r.LanguagesIter() {
		languages = append(languages, lang)
	}
	return languages
}

// LanguagesIter returns an iterator for all registered language identifiers.
//
// for lang := range r.LanguagesIter() { fmt.Println(lang) }
func (r *Registry) LanguagesIter() iter.Seq[string] {
	return func(yield func(string) bool) {
		// Sort keys for deterministic iteration
		keys := make([]string, 0, len(r.generators))
		for lang := range r.generators {
			keys = append(keys, lang)
		}
		slices.Sort(keys)
		for _, lang := range keys {
			if !yield(lang) {
				return
			}
		}
	}
}

// dockerUserArgs returns Docker --user args for the current user on Unix systems.
// On Windows, Docker handles permissions differently, so no args are returned.
func dockerUserArgs() []string {
	if runtime.GOOS == "windows" {
		return nil
	}
	return []string{"--user", core.Sprintf("%d:%d", ax.Getuid(), ax.Getgid())}
}
