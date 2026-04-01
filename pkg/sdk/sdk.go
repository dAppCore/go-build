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
//
// cfg := &sdk.Config{Languages: []string{"typescript"}, Output: "sdk"}
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
//
// cfg.Package = sdk.PackageConfig{Name: "@host-uk/api-client", Version: "1.0.0"}
type PackageConfig struct {
	// Name is the base package name.
	Name string `yaml:"name,omitempty"`
	// Version is the SDK version (supports templates like {{.Version}}).
	Version string `yaml:"version,omitempty"`
}

// DiffConfig holds breaking change detection configuration.
//
// cfg.Diff = sdk.DiffConfig{Enabled: true, FailOnBreaking: true}
type DiffConfig struct {
	// Enabled determines whether to run diff checks.
	Enabled bool `yaml:"enabled,omitempty"`
	// FailOnBreaking fails the release if breaking changes are detected.
	FailOnBreaking bool `yaml:"fail_on_breaking,omitempty"`
}

// PublishConfig holds monorepo publishing configuration.
//
// cfg.Publish = sdk.PublishConfig{Repo: "host-uk/ts", Path: "packages/api-client"}
type PublishConfig struct {
	// Repo is the SDK monorepo (e.g., "myorg/sdks").
	Repo string `yaml:"repo,omitempty"`
	// Path is the subdirectory for this SDK (e.g., "packages/myapi").
	Path string `yaml:"path,omitempty"`
}

// SDK orchestrates OpenAPI SDK generation.
//
// s := sdk.New(".", cfg)
type SDK struct {
	config     *Config
	projectDir string
	version    string
}

// New creates a new SDK instance.
//
// s := sdk.New(".", &sdk.Config{Languages: []string{"typescript"}, Output: "sdk"})
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
//
// s.SetVersion("v1.2.3")
func (s *SDK) SetVersion(version string) {
	s.version = version
	if s.config != nil && !containsVersionTemplate(s.config.Package.Version) {
		s.config.Package.Version = version
	}
}

// DefaultConfig returns sensible defaults for SDK configuration.
//
// cfg := sdk.DefaultConfig() // languages: typescript, python, go, php
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
//
// err := s.Generate(ctx) // generates sdk/typescript/, sdk/python/, etc.
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
//
// err := s.GenerateLanguage(ctx, "typescript") // generates sdk/typescript/
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
	}

	outputDir := ax.Join(s.projectDir, s.config.Output, lang)
	opts := generators.Options{
		SpecPath:    specPath,
		OutputDir:   outputDir,
		PackageName: s.config.Package.Name,
		Version:     s.resolvePackageVersion(),
	}

	core.Print(nil, "Generating %s SDK...", lang)
	if err := gen.Generate(ctx, opts); err != nil {
		return coreerr.E("sdk.GenerateLanguage", lang+" generation failed", err)
	}
	core.Print(nil, "Generated %s SDK at %s", lang, outputDir)

	return nil
}

// resolvePackageVersion renders the configured package version against the
// current SDK version when a template placeholder is present.
//
// resolved := s.resolvePackageVersion() // "v1.2.3" or "1.2.3-beta"
func (s *SDK) resolvePackageVersion() string {
	if s == nil || s.config == nil {
		return ""
	}

	packageVersion := s.config.Package.Version
	if packageVersion == "" {
		return s.version
	}

	if !containsVersionTemplate(packageVersion) {
		return packageVersion
	}

	if s.version == "" {
		return packageVersion
	}

	resolved := core.Replace(packageVersion, "{{.Version}}", s.version)
	resolved = core.Replace(resolved, "{{Version}}", s.version)
	return resolved
}

// containsVersionTemplate reports whether a package version uses a version
// placeholder that should be rendered at generation time.
func containsVersionTemplate(value string) bool {
	return core.Contains(value, "{{.Version}}") || core.Contains(value, "{{Version}}")
}
