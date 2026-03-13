// Package generators provides SDK code generators for different languages.
package generators

import (
	"context"
	"fmt"
	"iter"
	"maps"
	"os"
	"runtime"
	"slices"
)

// Options holds common generation options.
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
type Registry struct {
	generators map[string]Generator
}

// NewRegistry creates a registry with all available generators.
func NewRegistry() *Registry {
	r := &Registry{
		generators: make(map[string]Generator),
	}
	// Generators will be registered in subsequent tasks
	return r
}

// Get returns a generator by language.
func (r *Registry) Get(lang string) (Generator, bool) {
	g, ok := r.generators[lang]
	return g, ok
}

// Register adds a generator to the registry.
func (r *Registry) Register(g Generator) {
	r.generators[g.Language()] = g
}

// Languages returns all registered language identifiers.
func (r *Registry) Languages() []string {
	return slices.Collect(r.LanguagesIter())
}

// LanguagesIter returns an iterator for all registered language identifiers.
func (r *Registry) LanguagesIter() iter.Seq[string] {
	return func(yield func(string) bool) {
		// Sort keys for deterministic iteration
		for _, lang := range slices.Sorted(maps.Keys(r.generators)) {
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
	return []string{"--user", fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid())}
}
