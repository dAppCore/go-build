// Package release provides release automation with changelog generation and publishing.
package release

import (
	"iter"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/sdk"
	"dappco.re/go/core"
	coreio "dappco.re/go/io"
	coreerr "dappco.re/go/log"
	"gopkg.in/yaml.v3" // Note: AX-6 — no core YAMLUnmarshal yet.
)

// ConfigFileName is the name of the release configuration file.
//
// configPath := ax.Join(projectDir, release.ConfigDir, release.ConfigFileName)
const ConfigFileName = "release.yaml"

// ConfigDir is the directory where release configuration is stored.
//
// configPath := ax.Join(projectDir, release.ConfigDir, release.ConfigFileName)
const ConfigDir = ".core"

// Config holds the complete release configuration loaded from .core/release.yaml.
//
// cfg, err := release.LoadConfig(".")
type Config struct {
	// Version is the config file format version.
	Version int `yaml:"version"`
	// Project contains project metadata.
	Project ProjectConfig `yaml:"project"`
	// Build contains build settings for the release.
	Build BuildConfig `yaml:"build"`
	// Publishers defines where to publish the release.
	Publishers []PublisherConfig `yaml:"publishers"`
	// Changelog configures changelog generation.
	Changelog ChangelogConfig `yaml:"changelog"`
	// SDK configures SDK generation.
	SDK *SDKConfig `yaml:"sdk,omitempty"`
	// Checksum configures checksum generation for release artifacts.
	Checksum ChecksumConfig `yaml:"checksum,omitempty"`

	// Internal fields (not serialized)
	projectDir string // Set by LoadConfig
	version    string // Set by CLI flag
	output     coreio.Medium
	outputDir  string
}

// ProjectConfig holds project metadata for releases.
//
// cfg.Project = release.ProjectConfig{Name: "core-build", Repository: "host-uk/core-build"}
type ProjectConfig struct {
	// Name is the project name.
	Name string `yaml:"name"`
	// Repository is the GitHub repository in owner/repo format.
	Repository string `yaml:"repository"`
}

// BuildConfig holds build settings for releases.
//
// cfg.Build.Targets = []release.TargetConfig{{OS: "linux", Arch: "amd64"}}
type BuildConfig struct {
	// Targets defines the build targets.
	Targets []TargetConfig `yaml:"targets"`
	// ArchiveFormat selects the archive compression format for build outputs.
	// Supported values are "gz", "xz", and "zip"; empty uses gzip.
	ArchiveFormat string `yaml:"archive_format,omitempty"`
}

// ChecksumConfig controls release checksum generation.
type ChecksumConfig struct {
	// Algorithm selects the checksum algorithm. Currently sha256 is supported.
	Algorithm string `yaml:"algorithm,omitempty"`
	// File is the checksum file path relative to dist/ unless absolute.
	File string `yaml:"file,omitempty"`
}

// TargetConfig defines a build target.
//
// t := release.TargetConfig{OS: "linux", Arch: "arm64"}
type TargetConfig struct {
	// OS is the target operating system (e.g., "linux", "darwin", "windows").
	OS string `yaml:"os"`
	// Arch is the target architecture (e.g., "amd64", "arm64").
	Arch string `yaml:"arch"`
}

// PublisherConfig holds configuration for a publisher.
//
// cfg.Publishers = []release.PublisherConfig{{Type: "github", Draft: false}}
type PublisherConfig struct {
	// Type is the publisher type (e.g., "github", "linuxkit", "docker").
	Type string `yaml:"type"`
	// Prerelease marks the release as a prerelease.
	Prerelease bool `yaml:"prerelease"`
	// Draft creates the release as a draft.
	Draft bool `yaml:"draft"`

	// LinuxKit-specific configuration
	// Config is the path to the LinuxKit YAML configuration file.
	Config string `yaml:"config,omitempty"`
	// Formats are the output formats to build (iso, raw, qcow2, vmdk).
	Formats []string `yaml:"formats,omitempty"`
	// Platforms are the target platforms (linux/amd64, linux/arm64).
	Platforms []string `yaml:"platforms,omitempty"`

	// Docker-specific configuration
	// Registry is the container registry (default: ghcr.io).
	Registry string `yaml:"registry,omitempty"`
	// Image is the image name in owner/repo format.
	Image string `yaml:"image,omitempty"`
	// Dockerfile is the path to the Dockerfile (default: Dockerfile).
	Dockerfile string `yaml:"dockerfile,omitempty"`
	// Tags are the image tags to apply.
	Tags []string `yaml:"tags,omitempty"`
	// BuildArgs are additional Docker build arguments.
	BuildArgs map[string]string `yaml:"build_args,omitempty"`

	// npm-specific configuration
	// Package is the npm package name (e.g., "@host-uk/core").
	Package string `yaml:"package,omitempty"`
	// Access is the npm access level: "public" or "restricted".
	Access string `yaml:"access,omitempty"`

	// Homebrew-specific configuration
	// Tap is the Homebrew tap repository (e.g., "host-uk/homebrew-tap").
	Tap string `yaml:"tap,omitempty"`
	// Formula is the formula name (defaults to project name).
	Formula string `yaml:"formula,omitempty"`

	// Scoop-specific configuration
	// Bucket is the Scoop bucket repository (e.g., "host-uk/scoop-bucket").
	Bucket string `yaml:"bucket,omitempty"`

	// AUR-specific configuration
	// Maintainer is the AUR package maintainer (e.g., "Name <email>").
	Maintainer string `yaml:"maintainer,omitempty"`

	// Chocolatey-specific configuration
	// Push determines whether to push to Chocolatey (false = generate only).
	Push bool `yaml:"push,omitempty"`

	// Official repo configuration (for Homebrew, Scoop)
	// When enabled, generates files for PR to official repos.
	Official *OfficialConfig `yaml:"official,omitempty"`
}

// OfficialConfig holds configuration for generating files for official repo PRs.
//
// pub.Official = &release.OfficialConfig{Enabled: true, Output: "dist/homebrew"}
type OfficialConfig struct {
	// Enabled determines whether to generate files for official repos.
	Enabled bool `yaml:"enabled"`
	// Output is the directory to write generated files.
	Output string `yaml:"output,omitempty"`
}

// SDKConfig holds SDK generation configuration.
//
// cfg.SDK = &release.SDKConfig{Spec: "docs/openapi.yaml", Languages: []string{"typescript", "go"}}
type SDKConfig = sdk.Config

// SDKPackageConfig holds package naming configuration.
//
// cfg.SDK.Package = release.SDKPackageConfig{Name: "@host-uk/api-client", Version: "1.0.0"}
type SDKPackageConfig = sdk.PackageConfig

// SDKDiffConfig holds diff configuration.
//
// cfg.SDK.Diff = release.SDKDiffConfig{Enabled: true, FailOnBreaking: true}
type SDKDiffConfig = sdk.DiffConfig

// SDKPublishConfig holds monorepo publish configuration.
//
// cfg.SDK.Publish = release.SDKPublishConfig{Repo: "host-uk/ts", Path: "packages/api-client"}
type SDKPublishConfig = sdk.PublishConfig

// ChangelogConfig holds changelog generation settings.
//
// cfg.Changelog = release.ChangelogConfig{Include: []string{"feat", "fix"}, Exclude: []string{"chore"}}
type ChangelogConfig struct {
	// Use selects the changelog strategy. Conventional commits are the default.
	Use string `yaml:"use,omitempty"`
	// Include specifies commit types to include in the changelog.
	Include []string `yaml:"include"`
	// Exclude specifies commit types to exclude from the changelog.
	Exclude []string `yaml:"exclude"`
}

// PublishersIter returns an iterator for the publishers.
//
// for p := range cfg.PublishersIter() { fmt.Println(p.Type) }
func (c *Config) PublishersIter() iter.Seq[PublisherConfig] {
	return func(yield func(PublisherConfig) bool) {
		if c == nil {
			return
		}
		for _, p := range c.Publishers {
			if !yield(p) {
				return
			}
		}
	}
}

// LoadConfig loads release configuration from the .core/release.yaml file in the given directory.
// If the config file does not exist, it returns DefaultConfig().
// Returns an error if the file exists but cannot be parsed.
//
// cfg, err := release.LoadConfig(".")
func LoadConfig(dir string) (*Config, error) {
	return LoadConfigWithMedium(coreio.Local, dir)
}

// LoadConfigWithMedium loads release configuration from the provided medium.
// This mirrors build config loading so callers that virtualise project files
// via io.Medium can still resolve release settings consistently.
//
// cfg, err := release.LoadConfigWithMedium(io.NewMemoryMedium(), "project")
func LoadConfigWithMedium(filesystem coreio.Medium, dir string) (*Config, error) {
	cfg, err := LoadConfigAtPath(filesystem, ConfigPath(dir))
	if err != nil {
		return nil, err
	}

	cfg.projectDir = dir
	return cfg, nil
}

// LoadConfigAtPath loads release configuration from an explicit path in the
// provided medium. If the path does not point to a file, it returns
// DefaultConfig().
//
// cfg, err := release.LoadConfigAtPath(io.Local, "/tmp/project/.core/release.yaml")
func LoadConfigAtPath(filesystem coreio.Medium, configPath string) (*Config, error) {
	if filesystem == nil {
		filesystem = coreio.Local
	}

	content, err := filesystem.Read(configPath)
	if err != nil {
		if !filesystem.IsFile(configPath) {
			return DefaultConfig(), nil
		}
		return nil, coreerr.E("release.LoadConfigAtPath", "failed to read config file", err)
	}

	var cfg Config
	if err := yaml.Unmarshal([]byte(content), &cfg); err != nil {
		return nil, coreerr.E("release.LoadConfigAtPath", "failed to parse config file", err)
	}

	// Apply defaults for any missing fields.
	applyDefaults(&cfg)
	cfg.ExpandEnv()
	if cfg.SDK != nil {
		cfg.SDK.ApplyDefaults()
	}

	return &cfg, nil
}

// DefaultConfig returns sensible defaults for release configuration.
//
// cfg := release.DefaultConfig()
func DefaultConfig() *Config {
	return &Config{
		Version: 1,
		Project: ProjectConfig{
			Name:       "",
			Repository: "",
		},
		Build: BuildConfig{
			Targets: defaultTargetConfigs(),
		},
		Publishers: []PublisherConfig{
			{
				Type:       "github",
				Prerelease: false,
				Draft:      false,
			},
		},
		Changelog: ChangelogConfig{
			Use:     "conventional",
			Include: []string{"feat", "fix", "perf", "refactor"},
			Exclude: []string{"chore", "docs", "style", "test", "ci"},
		},
		Checksum: ChecksumConfig{
			Algorithm: "sha256",
			File:      defaultChecksumFileName,
		},
	}
}

// ScaffoldConfig returns the config shape written by `core ci init`.
//
// cfg := release.ScaffoldConfig()
func ScaffoldConfig() *Config {
	cfg := DefaultConfig()
	cfg.SDK = &SDKConfig{
		Spec:      "api/openapi.yaml",
		Languages: []string{"typescript", "python", "go", "php"},
		Output:    "sdk",
		Diff: SDKDiffConfig{
			Enabled:        true,
			FailOnBreaking: false,
		},
	}
	return cfg
}

// applyDefaults fills in default values for any empty fields in the config.
func applyDefaults(cfg *Config) {
	defaults := DefaultConfig()

	if cfg.Version == 0 {
		cfg.Version = defaults.Version
	}

	if len(cfg.Publishers) == 0 {
		cfg.Publishers = defaults.Publishers
	}

	if cfg.Changelog.Use == "" {
		cfg.Changelog.Use = defaults.Changelog.Use
	}

	if len(cfg.Changelog.Include) == 0 && len(cfg.Changelog.Exclude) == 0 {
		cfg.Changelog.Include = defaults.Changelog.Include
		cfg.Changelog.Exclude = defaults.Changelog.Exclude
	}

	if cfg.Checksum.Algorithm == "" {
		cfg.Checksum.Algorithm = defaults.Checksum.Algorithm
	}
	if cfg.Checksum.File == "" {
		cfg.Checksum.File = defaults.Checksum.File
	}
}

// ExpandEnv expands environment variables across the release config.
//
//	cfg.ExpandEnv() // expands $REPO, $PACKAGE_NAME, $SDK_SPEC, etc.
func (c *Config) ExpandEnv() {
	if c == nil {
		return
	}

	c.Project.Name = expandEnv(c.Project.Name)
	c.Project.Repository = expandEnv(c.Project.Repository)

	c.Build.ArchiveFormat = expandEnv(c.Build.ArchiveFormat)
	c.Build.Targets = expandTargetConfigs(c.Build.Targets)

	c.Publishers = expandPublisherConfigs(c.Publishers)

	c.Changelog.Use = expandEnv(c.Changelog.Use)
	c.Changelog.Include = expandEnvSlice(c.Changelog.Include)
	c.Changelog.Exclude = expandEnvSlice(c.Changelog.Exclude)
	c.Checksum.Algorithm = expandEnv(c.Checksum.Algorithm)
	c.Checksum.File = expandEnv(c.Checksum.File)

	if c.SDK != nil {
		c.SDK.Spec = expandEnv(c.SDK.Spec)
		c.SDK.Languages = expandEnvSlice(c.SDK.Languages)
		c.SDK.Output = expandEnv(c.SDK.Output)
		c.SDK.Package.Name = expandEnv(c.SDK.Package.Name)
		c.SDK.Package.Version = expandEnv(c.SDK.Package.Version)
		c.SDK.Publish.Repo = expandEnv(c.SDK.Publish.Repo)
		c.SDK.Publish.Path = expandEnv(c.SDK.Publish.Path)
	}
}

func defaultTargetConfigs() []TargetConfig {
	return []TargetConfig{
		{OS: "linux", Arch: "amd64"},
		{OS: "linux", Arch: "arm64"},
		{OS: "darwin", Arch: "amd64"},
		{OS: "darwin", Arch: "arm64"},
		{OS: "windows", Arch: "amd64"},
	}
}

// SetProjectDir sets the project directory on the config.
//
// cfg.SetProjectDir("/home/user/my-project")
func (c *Config) SetProjectDir(dir string) {
	if c == nil {
		return
	}
	c.projectDir = dir
}

// SetVersion sets the version override on the config.
//
// cfg.SetVersion("v1.2.3")
func (c *Config) SetVersion(version string) {
	if c == nil {
		return
	}
	c.version = version
}

// SetOutput configures the medium and root used for release artifacts.
//
// cfg.SetOutput(io.NewMemoryMedium(), "releases")
func (c *Config) SetOutput(medium coreio.Medium, dir string) {
	if c == nil {
		return
	}
	c.output = medium
	c.outputDir = dir
}

// SetOutputMedium overrides the medium used for release artifacts.
//
// cfg.SetOutputMedium(io.NewMemoryMedium())
func (c *Config) SetOutputMedium(medium coreio.Medium) {
	if c == nil {
		return
	}
	c.output = medium
}

// SetOutputDir overrides the root directory or key prefix used for release artifacts.
//
// cfg.SetOutputDir("releases")
func (c *Config) SetOutputDir(dir string) {
	if c == nil {
		return
	}
	c.outputDir = dir
}

func expandPublisherConfigs(publishers []PublisherConfig) []PublisherConfig {
	if len(publishers) == 0 {
		return publishers
	}

	result := make([]PublisherConfig, len(publishers))
	copy(result, publishers)

	for i := range result {
		result[i].Type = expandEnv(result[i].Type)
		result[i].Config = expandEnv(result[i].Config)
		result[i].Formats = expandEnvSlice(result[i].Formats)
		result[i].Platforms = expandEnvSlice(result[i].Platforms)
		result[i].Registry = expandEnv(result[i].Registry)
		result[i].Image = expandEnv(result[i].Image)
		result[i].Dockerfile = expandEnv(result[i].Dockerfile)
		result[i].Tags = expandEnvSlice(result[i].Tags)
		result[i].BuildArgs = expandEnvMap(result[i].BuildArgs)
		result[i].Package = expandEnv(result[i].Package)
		result[i].Access = expandEnv(result[i].Access)
		result[i].Tap = expandEnv(result[i].Tap)
		result[i].Formula = expandEnv(result[i].Formula)
		result[i].Bucket = expandEnv(result[i].Bucket)
		result[i].Maintainer = expandEnv(result[i].Maintainer)
		if result[i].Official != nil {
			result[i].Official.Output = expandEnv(result[i].Official.Output)
		}
	}

	return result
}

// ConfigPath returns the path to the release config file for a given directory.
//
// path := release.ConfigPath("/home/user/my-project") // → "/home/user/my-project/.core/release.yaml"
func ConfigPath(dir string) string {
	return ax.Join(dir, ConfigDir, ConfigFileName)
}

// ConfigExists checks if a release config file exists in the given directory.
//
// if release.ConfigExists(".") { ... }
func ConfigExists(dir string) bool {
	configPath := ConfigPath(dir)
	absPath, err := ax.Abs(configPath)
	if err != nil {
		return false
	}
	return ax.IsFile(absPath)
}

// GetRepository returns the repository from the config.
//
// repo := cfg.GetRepository() // → "host-uk/core-build"
func (c *Config) GetRepository() string {
	if c == nil {
		return ""
	}
	return c.Project.Repository
}

// GetProjectName returns the project name from the config.
//
// name := cfg.GetProjectName() // → "core-build"
func (c *Config) GetProjectName() string {
	if c == nil {
		return ""
	}
	return c.Project.Name
}

// WriteConfig writes the config to the .core/release.yaml file.
//
// err := release.WriteConfig(cfg, ".")
func WriteConfig(cfg *Config, dir string) error {
	configPath := ConfigPath(dir)

	// Resolve path with AX-aware helpers.
	absPath, err := ax.Abs(configPath)
	if err != nil {
		return coreerr.E("release.WriteConfig", "failed to resolve path", err)
	}

	// Ensure directory exists
	configDir := ax.Dir(absPath)
	if err := ax.MkdirAll(configDir, 0o755); err != nil {
		return coreerr.E("release.WriteConfig", "failed to create directory", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return coreerr.E("release.WriteConfig", "failed to marshal config", err)
	}

	if err := ax.WriteString(absPath, string(data), 0o644); err != nil {
		return coreerr.E("release.WriteConfig", "failed to write config file", err)
	}

	return nil
}

func expandEnvSlice(values []string) []string {
	if len(values) == 0 {
		return values
	}

	result := make([]string, len(values))
	for i, value := range values {
		result[i] = expandEnv(value)
	}
	return result
}

func expandEnvMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return values
	}

	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = expandEnv(value)
	}
	return result
}

func expandTargetConfigs(values []TargetConfig) []TargetConfig {
	if len(values) == 0 {
		return values
	}

	result := make([]TargetConfig, len(values))
	for i, value := range values {
		result[i] = TargetConfig{
			OS:   expandEnv(value.OS),
			Arch: expandEnv(value.Arch),
		}
	}
	return result
}

// expandEnv expands $VAR or ${VAR} using the current process environment.
func expandEnv(s string) string {
	if !core.Contains(s, "$") {
		return s
	}

	buf := core.NewBuilder()
	for i := 0; i < len(s); {
		if s[i] != '$' {
			buf.WriteByte(s[i])
			i++
			continue
		}

		if i+1 < len(s) && s[i+1] == '{' {
			j := i + 2
			for j < len(s) && s[j] != '}' {
				j++
			}
			if j < len(s) {
				buf.WriteString(core.Env(s[i+2 : j]))
				i = j + 1
				continue
			}
		}

		j := i + 1
		for j < len(s) {
			c := s[j]
			if c != '_' && (c < '0' || c > '9') && (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') {
				break
			}
			j++
		}
		if j > i+1 {
			buf.WriteString(core.Env(s[i+1 : j]))
			i = j
			continue
		}

		buf.WriteByte(s[i])
		i++
	}

	return buf.String()
}
