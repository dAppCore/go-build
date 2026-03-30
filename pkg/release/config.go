// Package release provides release automation with changelog generation and publishing.
package release

import (
	"iter"

	"dappco.re/go/core/build/internal/ax"
	coreerr "dappco.re/go/core/log"
	"gopkg.in/yaml.v3"
)

// ConfigFileName is the name of the release configuration file.
// Usage example: reference release.ConfigFileName from package consumers.
const ConfigFileName = "release.yaml"

// ConfigDir is the directory where release configuration is stored.
// Usage example: reference release.ConfigDir from package consumers.
const ConfigDir = ".core"

// Config holds the complete release configuration loaded from .core/release.yaml.
// Usage example: declare a value of type release.Config in integrating code.
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

	// Internal fields (not serialized)
	projectDir string // Set by LoadConfig
	version    string // Set by CLI flag
}

// ProjectConfig holds project metadata for releases.
// Usage example: declare a value of type release.ProjectConfig in integrating code.
type ProjectConfig struct {
	// Name is the project name.
	Name string `yaml:"name"`
	// Repository is the GitHub repository in owner/repo format.
	Repository string `yaml:"repository"`
}

// BuildConfig holds build settings for releases.
// Usage example: declare a value of type release.BuildConfig in integrating code.
type BuildConfig struct {
	// Targets defines the build targets.
	Targets []TargetConfig `yaml:"targets"`
}

// TargetConfig defines a build target.
// Usage example: declare a value of type release.TargetConfig in integrating code.
type TargetConfig struct {
	// OS is the target operating system (e.g., "linux", "darwin", "windows").
	OS string `yaml:"os"`
	// Arch is the target architecture (e.g., "amd64", "arm64").
	Arch string `yaml:"arch"`
}

// PublisherConfig holds configuration for a publisher.
// Usage example: declare a value of type release.PublisherConfig in integrating code.
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
// Usage example: declare a value of type release.OfficialConfig in integrating code.
type OfficialConfig struct {
	// Enabled determines whether to generate files for official repos.
	Enabled bool `yaml:"enabled"`
	// Output is the directory to write generated files.
	Output string `yaml:"output,omitempty"`
}

// SDKConfig holds SDK generation configuration.
// Usage example: declare a value of type release.SDKConfig in integrating code.
type SDKConfig struct {
	// Spec is the path to the OpenAPI spec file.
	Spec string `yaml:"spec,omitempty"`
	// Languages to generate.
	Languages []string `yaml:"languages,omitempty"`
	// Output directory (default: sdk/).
	Output string `yaml:"output,omitempty"`
	// Package naming.
	Package SDKPackageConfig `yaml:"package,omitempty"`
	// Diff configuration.
	Diff SDKDiffConfig `yaml:"diff,omitempty"`
	// Publish configuration.
	Publish SDKPublishConfig `yaml:"publish,omitempty"`
}

// SDKPackageConfig holds package naming configuration.
// Usage example: declare a value of type release.SDKPackageConfig in integrating code.
type SDKPackageConfig struct {
	Name    string `yaml:"name,omitempty"`
	Version string `yaml:"version,omitempty"`
}

// SDKDiffConfig holds diff configuration.
// Usage example: declare a value of type release.SDKDiffConfig in integrating code.
type SDKDiffConfig struct {
	Enabled        bool `yaml:"enabled,omitempty"`
	FailOnBreaking bool `yaml:"fail_on_breaking,omitempty"`
}

// SDKPublishConfig holds monorepo publish configuration.
// Usage example: declare a value of type release.SDKPublishConfig in integrating code.
type SDKPublishConfig struct {
	Repo string `yaml:"repo,omitempty"`
	Path string `yaml:"path,omitempty"`
}

// ChangelogConfig holds changelog generation settings.
// Usage example: declare a value of type release.ChangelogConfig in integrating code.
type ChangelogConfig struct {
	// Include specifies commit types to include in the changelog.
	Include []string `yaml:"include"`
	// Exclude specifies commit types to exclude from the changelog.
	Exclude []string `yaml:"exclude"`
}

// PublishersIter returns an iterator for the publishers.
// Usage example: call value.PublishersIter(...) from integrating code.
func (c *Config) PublishersIter() iter.Seq[PublisherConfig] {
	return func(yield func(PublisherConfig) bool) {
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
// Usage example: call release.LoadConfig(...) from integrating code.
func LoadConfig(dir string) (*Config, error) {
	configPath := ax.Join(dir, ConfigDir, ConfigFileName)

	// Resolve path with AX-aware helpers.
	absPath, err := ax.Abs(configPath)
	if err != nil {
		return nil, coreerr.E("release.LoadConfig", "failed to resolve path", err)
	}

	content, err := ax.ReadFile(absPath)
	if err != nil {
		if !ax.IsFile(absPath) {
			cfg := DefaultConfig()
			cfg.projectDir = dir
			return cfg, nil
		}
		return nil, coreerr.E("release.LoadConfig", "failed to read config file", err)
	}

	var cfg Config
	if err := yaml.Unmarshal([]byte(content), &cfg); err != nil {
		return nil, coreerr.E("release.LoadConfig", "failed to parse config file", err)
	}

	// Apply defaults for any missing fields
	applyDefaults(&cfg)
	cfg.projectDir = dir

	return &cfg, nil
}

// DefaultConfig returns sensible defaults for release configuration.
// Usage example: call release.DefaultConfig(...) from integrating code.
func DefaultConfig() *Config {
	return &Config{
		Version: 1,
		Project: ProjectConfig{
			Name:       "",
			Repository: "",
		},
		Build: BuildConfig{
			Targets: []TargetConfig{
				{OS: "linux", Arch: "amd64"},
				{OS: "linux", Arch: "arm64"},
				{OS: "darwin", Arch: "arm64"},
				{OS: "windows", Arch: "amd64"},
			},
		},
		Publishers: []PublisherConfig{
			{
				Type:       "github",
				Prerelease: false,
				Draft:      false,
			},
		},
		Changelog: ChangelogConfig{
			Include: []string{"feat", "fix", "perf", "refactor"},
			Exclude: []string{"chore", "docs", "style", "test", "ci"},
		},
	}
}

// applyDefaults fills in default values for any empty fields in the config.
func applyDefaults(cfg *Config) {
	defaults := DefaultConfig()

	if cfg.Version == 0 {
		cfg.Version = defaults.Version
	}

	if len(cfg.Build.Targets) == 0 {
		cfg.Build.Targets = defaults.Build.Targets
	}

	if len(cfg.Publishers) == 0 {
		cfg.Publishers = defaults.Publishers
	}

	if len(cfg.Changelog.Include) == 0 && len(cfg.Changelog.Exclude) == 0 {
		cfg.Changelog.Include = defaults.Changelog.Include
		cfg.Changelog.Exclude = defaults.Changelog.Exclude
	}
}

// SetProjectDir sets the project directory on the config.
// Usage example: call value.SetProjectDir(...) from integrating code.
func (c *Config) SetProjectDir(dir string) {
	c.projectDir = dir
}

// SetVersion sets the version override on the config.
// Usage example: call value.SetVersion(...) from integrating code.
func (c *Config) SetVersion(version string) {
	c.version = version
}

// ConfigPath returns the path to the release config file for a given directory.
// Usage example: call release.ConfigPath(...) from integrating code.
func ConfigPath(dir string) string {
	return ax.Join(dir, ConfigDir, ConfigFileName)
}

// ConfigExists checks if a release config file exists in the given directory.
// Usage example: call release.ConfigExists(...) from integrating code.
func ConfigExists(dir string) bool {
	configPath := ConfigPath(dir)
	absPath, err := ax.Abs(configPath)
	if err != nil {
		return false
	}
	return ax.IsFile(absPath)
}

// GetRepository returns the repository from the config.
// Usage example: call value.GetRepository(...) from integrating code.
func (c *Config) GetRepository() string {
	return c.Project.Repository
}

// GetProjectName returns the project name from the config.
// Usage example: call value.GetProjectName(...) from integrating code.
func (c *Config) GetProjectName() string {
	return c.Project.Name
}

// WriteConfig writes the config to the .core/release.yaml file.
// Usage example: call release.WriteConfig(...) from integrating code.
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
