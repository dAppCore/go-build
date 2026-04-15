// Package sdk provides OpenAPI SDK generation and diff capabilities.
package sdk

import (
	"context"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/sdk/generators"
	coreerr "dappco.re/go/core/log"
	"gopkg.in/yaml.v3"
)

// Config holds SDK generation configuration for SDK commands and releases.
//
// cfg := &sdk.Config{Languages: []string{"typescript"}, Output: "sdk"}
type Config struct {
	// Spec is the path to the OpenAPI spec file (auto-detected if empty).
	Spec string `json:"spec,omitempty" yaml:"spec,omitempty"`
	// Languages to generate SDKs for.
	Languages []string `json:"languages,omitempty" yaml:"languages,omitempty"`
	// Output directory (default: sdk/).
	Output string `json:"output,omitempty" yaml:"output,omitempty"`
	// Package naming configuration.
	Package PackageConfig `json:"package,omitempty" yaml:"package,omitempty"`
	// Diff configuration for breaking change detection.
	Diff DiffConfig `json:"diff,omitempty" yaml:"diff,omitempty"`
	// Publish configuration for monorepo publishing.
	Publish PublishConfig `json:"publish,omitempty" yaml:"publish,omitempty"`
}

// PackageConfig holds package naming configuration.
//
// cfg.Package = sdk.PackageConfig{Name: "@host-uk/api-client", Version: "1.0.0"}
type PackageConfig struct {
	// Name is the base package name.
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	// Version is the SDK version (supports templates like {{.Version}}).
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}

// DiffConfig holds breaking change detection configuration.
//
// cfg.Diff = sdk.DiffConfig{Enabled: true, FailOnBreaking: true}
type DiffConfig struct {
	// Enabled determines whether to run diff checks.
	Enabled bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	// FailOnBreaking fails the release if breaking changes are detected.
	FailOnBreaking bool `json:"fail_on_breaking,omitempty" yaml:"fail_on_breaking,omitempty"`

	enabledConfigured bool `yaml:"-" json:"-"`
}

// PublishConfig holds monorepo publishing configuration.
//
// cfg.Publish = sdk.PublishConfig{Repo: "host-uk/ts", Path: "packages/api-client"}
type PublishConfig struct {
	// Repo is the SDK monorepo (e.g., "myorg/sdks").
	Repo string `json:"repo,omitempty" yaml:"repo,omitempty"`
	// Path is the subdirectory for this SDK (e.g., "packages/myapi").
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
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

// CloneConfig returns a deep copy of an SDK config.
func CloneConfig(config *Config) *Config {
	if config == nil {
		return nil
	}

	clone := *config
	clone.Languages = append([]string(nil), config.Languages...)
	return &clone
}

// ApplyDefaults fills in documented defaults for partial SDK configs.
func (c *Config) ApplyDefaults() {
	if c == nil {
		return
	}

	defaults := DefaultConfig()
	if len(c.Languages) == 0 {
		c.Languages = append([]string(nil), defaults.Languages...)
	}
	if c.Output == "" {
		c.Output = defaults.Output
	}
	if !c.Diff.enabledConfigured {
		c.Diff.Enabled = defaults.Diff.Enabled
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
			Enabled:           true,
			FailOnBreaking:    false,
			enabledConfigured: true,
		},
	}
}

// UnmarshalYAML accepts either `diff: true` or the expanded mapping form.
func (c *DiffConfig) UnmarshalYAML(value *yaml.Node) error {
	type diffConfigAlias struct {
		Enabled        *bool `yaml:"enabled,omitempty"`
		FailOnBreaking bool  `yaml:"fail_on_breaking,omitempty"`
	}

	switch value.Kind {
	case yaml.ScalarNode:
		var enabled bool
		if err := value.Decode(&enabled); err != nil {
			return err
		}
		*c = DiffConfig{
			Enabled:           enabled,
			enabledConfigured: true,
		}
		return nil
	case yaml.MappingNode:
		var alias diffConfigAlias
		if err := value.Decode(&alias); err != nil {
			return err
		}

		*c = DiffConfig{
			FailOnBreaking: alias.FailOnBreaking,
		}
		if alias.Enabled != nil {
			c.Enabled = *alias.Enabled
			c.enabledConfigured = true
		}
		return nil
	default:
		var alias diffConfigAlias
		if err := value.Decode(&alias); err != nil {
			return err
		}
		*c = DiffConfig{
			FailOnBreaking: alias.FailOnBreaking,
		}
		if alias.Enabled != nil {
			c.Enabled = *alias.Enabled
			c.enabledConfigured = true
		}
		return nil
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

// outputRoot returns the directory that should contain generated SDKs.
//
// root := s.outputRoot() // "sdk" or "packages/myapi/sdk"
func (s *SDK) outputRoot() string {
	if s == nil || s.config == nil {
		return "sdk"
	}

	output := s.config.Output
	if output == "" {
		output = "sdk"
	}

	if s.config.Publish.Path != "" {
		output = ax.Join(s.config.Publish.Path, output)
	}

	return output
}

// outputDir returns the language-specific SDK directory.
//
// dir := s.outputDir("typescript") // "sdk/typescript" or "packages/myapi/sdk/typescript"
func (s *SDK) outputDir(lang string) string {
	return ax.Join(s.projectDir, s.outputRoot(), lang)
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

	gen, ok := registry.Get(lang)
	if !ok {
		return coreerr.E("sdk.GenerateLanguage", "unknown language: "+lang, nil)
	}

	if !gen.Available() {
		core.Print(nil, "Warning: %s generator not available. Install with: %s", lang, gen.Install())
	}

	outputDir := s.outputDir(lang)
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
