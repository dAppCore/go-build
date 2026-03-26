// Package sdk provides OpenAPI SDK generation and diff capabilities.
package sdk

import (
	"context"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/sdk/generators"
	coreerr "dappco.re/go/core/log"
)

// Config holds SDK generation configuration from .core/release.yaml.
// Usage example: declare a value of type sdk.Config in integrating code.
type Config struct {
	// Spec is the path to the OpenAPI spec file (auto-detected if empty).
	Spec string `yaml:"spec,omitempty"`
	// Languages to generate SDKs for.
	Languages []string `yaml:"languages,omitempty"`
	// Output directory (default: sdk/).
	Output string `yaml:"output,omitempty"`
	// Package naming configuration.
	Package PackageConfig `yaml:"package,omitempty"`
	// Diff configuration for breaking change detection.
	Diff DiffConfig `yaml:"diff,omitempty"`
	// Publish configuration for monorepo publishing.
	Publish PublishConfig `yaml:"publish,omitempty"`
}

// PackageConfig holds package naming configuration.
// Usage example: declare a value of type sdk.PackageConfig in integrating code.
type PackageConfig struct {
	// Name is the base package name.
	Name string `yaml:"name,omitempty"`
	// Version is the SDK version (supports templates like {{.Version}}).
	Version string `yaml:"version,omitempty"`
}

// DiffConfig holds breaking change detection configuration.
// Usage example: declare a value of type sdk.DiffConfig in integrating code.
type DiffConfig struct {
	// Enabled determines whether to run diff checks.
	Enabled bool `yaml:"enabled,omitempty"`
	// FailOnBreaking fails the release if breaking changes are detected.
	FailOnBreaking bool `yaml:"fail_on_breaking,omitempty"`
}

// PublishConfig holds monorepo publishing configuration.
// Usage example: declare a value of type sdk.PublishConfig in integrating code.
type PublishConfig struct {
	// Repo is the SDK monorepo (e.g., "myorg/sdks").
	Repo string `yaml:"repo,omitempty"`
	// Path is the subdirectory for this SDK (e.g., "packages/myapi").
	Path string `yaml:"path,omitempty"`
}

// SDK orchestrates OpenAPI SDK generation.
// Usage example: declare a value of type sdk.SDK in integrating code.
type SDK struct {
	config     *Config
	projectDir string
	version    string
}

// New creates a new SDK instance.
// Usage example: call sdk.New(...) from integrating code.
func New(projectDir string, config *Config) *SDK {
	if config == nil {
		config = DefaultConfig()
	}
	return &SDK{
		config:     config,
		projectDir: projectDir,
	}
}

// SetVersion sets the SDK version for generation.
// This updates both the internal version field and the config's Package.Version.
// Usage example: call value.SetVersion(...) from integrating code.
func (s *SDK) SetVersion(version string) {
	s.version = version
	if s.config != nil {
		s.config.Package.Version = version
	}
}

// DefaultConfig returns sensible defaults for SDK configuration.
// Usage example: call sdk.DefaultConfig(...) from integrating code.
func DefaultConfig() *Config {
	return &Config{
		Languages: []string{"typescript", "python", "go", "php"},
		Output:    "sdk",
		Diff: DiffConfig{
			Enabled:        true,
			FailOnBreaking: false,
		},
	}
}

// Generate generates SDKs for all configured languages.
// Usage example: call value.Generate(...) from integrating code.
func (s *SDK) Generate(ctx context.Context) error {
	// Generate for each language
	for _, lang := range s.config.Languages {
		if err := s.GenerateLanguage(ctx, lang); err != nil {
			return err
		}
	}

	return nil
}

// GenerateLanguage generates SDK for a specific language.
// Usage example: call value.GenerateLanguage(...) from integrating code.
func (s *SDK) GenerateLanguage(ctx context.Context, lang string) error {
	specPath, err := s.DetectSpec()
	if err != nil {
		return err
	}

	registry := generators.NewRegistry()
	registry.Register(generators.NewTypeScriptGenerator())
	registry.Register(generators.NewPythonGenerator())
	registry.Register(generators.NewGoGenerator())
	registry.Register(generators.NewPHPGenerator())

	gen, ok := registry.Get(lang)
	if !ok {
		return coreerr.E("sdk.GenerateLanguage", "unknown language: "+lang, nil)
	}

	if !gen.Available() {
		core.Print(nil, "Warning: %s generator not available. Install with: %s", lang, gen.Install())
		core.Print(nil, "Falling back to Docker...")
	}

	outputDir := ax.Join(s.projectDir, s.config.Output, lang)
	opts := generators.Options{
		SpecPath:    specPath,
		OutputDir:   outputDir,
		PackageName: s.config.Package.Name,
		Version:     s.config.Package.Version,
	}

	core.Print(nil, "Generating %s SDK...", lang)
	if err := gen.Generate(ctx, opts); err != nil {
		return coreerr.E("sdk.GenerateLanguage", lang+" generation failed", err)
	}
	core.Print(nil, "Generated %s SDK at %s", lang, outputDir)

	return nil
}
