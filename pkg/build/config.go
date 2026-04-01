// Package build provides project type detection and cross-compilation for the Core build system.
// This file handles configuration loading from .core/build.yaml files.
package build

import (
	"iter"

	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/build/signing"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
	"gopkg.in/yaml.v3"
)

// ConfigFileName is the name of the build configuration file.
//
// configPath := ax.Join(projectDir, build.ConfigDir, build.ConfigFileName)
const ConfigFileName = "build.yaml"

// ConfigDir is the directory where build configuration is stored.
//
// configPath := ax.Join(projectDir, build.ConfigDir, build.ConfigFileName)
const ConfigDir = ".core"

// BuildConfig holds the complete build configuration loaded from .core/build.yaml.
// This is distinct from Config which holds runtime build parameters.
//
// cfg, err := build.LoadConfig(io.Local, ".")
type BuildConfig struct {
	// Version is the config file format version.
	Version int `yaml:"version"`
	// Project contains project metadata.
	Project Project `yaml:"project"`
	// Build contains build settings.
	Build Build `yaml:"build"`
	// Targets defines the build targets.
	Targets []TargetConfig `yaml:"targets"`
	// Sign contains code signing configuration.
	Sign signing.SignConfig `yaml:"sign,omitempty"`
}

// Project holds project metadata.
//
// cfg.Project.Binary = "core-build"
type Project struct {
	// Name is the project name.
	Name string `yaml:"name"`
	// Description is a brief description of the project.
	Description string `yaml:"description"`
	// Main is the path to the main package (e.g., ./cmd/core).
	Main string `yaml:"main"`
	// Binary is the output binary name.
	Binary string `yaml:"binary"`
}

// Build holds build-time settings.
//
// cfg.Build.LDFlags = []string{"-s", "-w", "-X main.version=" + version}
type Build struct {
	// Type overrides project type auto-detection (e.g., "go", "wails", "docker").
	Type string `yaml:"type"`
	// CGO enables CGO for the build.
	CGO bool `yaml:"cgo"`
	// Obfuscate uses garble instead of go build for binary obfuscation.
	Obfuscate bool `yaml:"obfuscate"`
	// NSIS enables Windows NSIS installer generation (Wails projects only).
	NSIS bool `yaml:"nsis"`
	// WebView2 sets the WebView2 delivery method: download|embed|browser|error.
	WebView2 string `yaml:"webview2,omitempty"`
	// Flags are additional build flags (e.g., ["-trimpath"]).
	Flags []string `yaml:"flags"`
	// LDFlags are linker flags (e.g., ["-s", "-w"]).
	LDFlags []string `yaml:"ldflags"`
	// BuildTags are Go build tags passed through to `go build`.
	BuildTags []string `yaml:"build_tags,omitempty"`
	// ArchiveFormat selects the archive compression format for build outputs.
	// Supported values are "gz", "xz", and "zip"; empty uses gzip.
	ArchiveFormat string `yaml:"archive_format,omitempty"`
	// Env are additional environment variables.
	Env []string `yaml:"env"`
	// Cache controls build cache setup.
	Cache CacheConfig `yaml:"cache,omitempty"`
	// Dockerfile is the path to the Dockerfile used by Docker builds.
	Dockerfile string `yaml:"dockerfile,omitempty"`
	// Registry is the container registry used for Docker image references.
	Registry string `yaml:"registry,omitempty"`
	// Image is the image name used for Docker builds.
	Image string `yaml:"image,omitempty"`
	// Tags are Docker image tags to apply.
	Tags []string `yaml:"tags,omitempty"`
	// BuildArgs are Docker build arguments.
	BuildArgs map[string]string `yaml:"build_args,omitempty"`
	// Push enables pushing Docker images after build.
	Push bool `yaml:"push,omitempty"`
	// Load loads a single-platform Docker image into the local daemon after build.
	Load bool `yaml:"load,omitempty"`
	// LinuxKitConfig is the path to the LinuxKit config file.
	LinuxKitConfig string `yaml:"linuxkit_config,omitempty"`
	// Formats is the list of LinuxKit output formats.
	// Supported values include iso, raw, qcow2, vmdk, vhd, gcp, aws, docker, tar, and kernel+initrd.
	Formats []string `yaml:"formats,omitempty"`
}

// TargetConfig defines a build target in the config file.
// This is separate from Target to allow for additional config-specific fields.
//
// cfg.Targets = []build.TargetConfig{{OS: "linux", Arch: "amd64"}, {OS: "darwin", Arch: "arm64"}}
type TargetConfig struct {
	// OS is the target operating system (e.g., "linux", "darwin", "windows").
	OS string `yaml:"os"`
	// Arch is the target architecture (e.g., "amd64", "arm64").
	Arch string `yaml:"arch"`
}

// LoadConfig loads build configuration from the .core/build.yaml file in the given directory.
// If the config file does not exist, it returns DefaultConfig().
// Returns an error if the file exists but cannot be parsed.
//
// cfg, err := build.LoadConfig(io.Local, ".")
func LoadConfig(fs io.Medium, dir string) (*BuildConfig, error) {
	return LoadConfigAtPath(fs, ax.Join(dir, ConfigDir, ConfigFileName))
}

// LoadConfigAtPath loads build configuration from an explicit file path.
// If the file does not exist, it returns DefaultConfig().
// Returns an error if the file exists but cannot be parsed.
//
// cfg, err := build.LoadConfigAtPath(io.Local, "/tmp/project/build.yaml")
func LoadConfigAtPath(fs io.Medium, configPath string) (*BuildConfig, error) {
	content, err := fs.Read(configPath)
	if err != nil {
		if !fs.Exists(configPath) {
			return DefaultConfig(), nil
		}
		return nil, coreerr.E("build.LoadConfig", "failed to read config file", err)
	}

	cfg := DefaultConfig()
	data := []byte(content)
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, coreerr.E("build.LoadConfig", "failed to parse config file", err)
	}

	// Apply defaults for any missing fields
	applyDefaults(cfg)

	return cfg, nil
}

// DefaultConfig returns sensible defaults for Go projects.
//
// cfg := build.DefaultConfig()
func DefaultConfig() *BuildConfig {
	return &BuildConfig{
		Version: 1,
		Project: Project{
			Name:   "",
			Main:   ".",
			Binary: "",
		},
		Build: Build{
			CGO:     false,
			Flags:   []string{"-trimpath"},
			LDFlags: []string{"-s", "-w"},
			Env:     []string{},
		},
		Targets: []TargetConfig{
			{OS: "linux", Arch: "amd64"},
			{OS: "linux", Arch: "arm64"},
			{OS: "darwin", Arch: "arm64"},
			{OS: "windows", Arch: "amd64"},
		},
		Sign: signing.DefaultSignConfig(),
	}
}

// applyDefaults fills in default values for any empty fields in the config.
func applyDefaults(cfg *BuildConfig) {
	defaults := DefaultConfig()

	if cfg.Version == 0 {
		cfg.Version = defaults.Version
	}

	if cfg.Project.Main == "" {
		cfg.Project.Main = defaults.Project.Main
	}

	if cfg.Build.Flags == nil {
		cfg.Build.Flags = defaults.Build.Flags
	}

	if cfg.Build.LDFlags == nil {
		cfg.Build.LDFlags = defaults.Build.LDFlags
	}

	if cfg.Build.Env == nil {
		cfg.Build.Env = defaults.Build.Env
	}

	if cfg.Targets == nil {
		cfg.Targets = defaults.Targets
	}

	// Expand environment variables in sign config
	cfg.Sign.ExpandEnv()
}

// ConfigPath returns the path to the build config file for a given directory.
//
// path := build.ConfigPath("/home/user/my-project") // → "/home/user/my-project/.core/build.yaml"
func ConfigPath(dir string) string {
	return ax.Join(dir, ConfigDir, ConfigFileName)
}

// ConfigExists checks if a build config file exists in the given directory.
//
// if build.ConfigExists(io.Local, ".") { ... }
func ConfigExists(fs io.Medium, dir string) bool {
	return fileExists(fs, ConfigPath(dir))
}

// TargetsIter returns an iterator for the build targets.
//
// for t := range cfg.TargetsIter() { fmt.Println(t.OS, t.Arch) }
func (cfg *BuildConfig) TargetsIter() iter.Seq[TargetConfig] {
	return func(yield func(TargetConfig) bool) {
		for _, t := range cfg.Targets {
			if !yield(t) {
				return
			}
		}
	}
}

// ToTargets converts TargetConfig slice to Target slice for use with builders.
//
// targets := cfg.ToTargets()
func (cfg *BuildConfig) ToTargets() []Target {
	targets := make([]Target, len(cfg.Targets))
	for i, t := range cfg.Targets {
		targets[i] = Target{OS: t.OS, Arch: t.Arch}
	}
	return targets
}
