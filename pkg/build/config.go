// Package build provides project type detection and cross-compilation for the Core build system.
// This file handles configuration loading from .core/build.yaml files.
package build

import (
	"iter"

	"dappco.re/go/core"
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
	Version int `json:"version" yaml:"version"`
	// Project contains project metadata.
	Project Project `json:"project" yaml:"project"`
	// Build contains build settings.
	Build Build `json:"build" yaml:"build"`
	// Apple contains macOS Apple pipeline settings.
	Apple AppleConfig `json:"apple,omitempty" yaml:"apple,omitempty"`
	// Targets defines the build targets.
	Targets []TargetConfig `json:"targets" yaml:"targets"`
	// Sign contains code signing configuration.
	Sign signing.SignConfig `json:"sign,omitempty" yaml:"sign,omitempty"`
	// LinuxKit contains immutable image configuration for `core build image`.
	LinuxKit LinuxKitConfig `json:"linuxkit,omitempty" yaml:"linuxkit,omitempty"`
}

// Project holds project metadata.
//
// cfg.Project.Binary = "core-build"
type Project struct {
	// Name is the project name.
	Name string `json:"name" yaml:"name"`
	// Description is a brief description of the project.
	Description string `json:"description" yaml:"description"`
	// Main is the path to the main package (e.g., ./cmd/core).
	Main string `json:"main" yaml:"main"`
	// Binary is the output binary name.
	Binary string `json:"binary" yaml:"binary"`
}

// Build holds build-time settings.
//
// cfg.Build.LDFlags = []string{"-s", "-w", "-X main.version=" + version}
type Build struct {
	// Type overrides project type auto-detection (e.g., "go", "wails", "docker").
	Type string `json:"type" yaml:"type"`
	// CGO enables CGO for the build.
	CGO bool `json:"cgo" yaml:"cgo"`
	// Obfuscate uses garble instead of go build for binary obfuscation.
	Obfuscate bool `json:"obfuscate" yaml:"obfuscate"`
	// DenoBuild overrides the default `deno task build` invocation for Deno-backed builds.
	DenoBuild string `json:"deno_build,omitempty" yaml:"deno_build,omitempty"`
	// NSIS enables Windows NSIS installer generation (Wails projects only).
	NSIS bool `json:"nsis" yaml:"nsis"`
	// WebView2 sets the WebView2 delivery method: download|embed|browser|error.
	WebView2 string `json:"webview2,omitempty" yaml:"webview2,omitempty"`
	// Flags are additional build flags (e.g., ["-trimpath"]).
	Flags []string `json:"flags" yaml:"flags"`
	// LDFlags are linker flags (e.g., ["-s", "-w"]).
	LDFlags []string `json:"ldflags" yaml:"ldflags"`
	// BuildTags are Go build tags passed through to `go build`.
	BuildTags []string `json:"build_tags,omitempty" yaml:"build_tags,omitempty"`
	// ArchiveFormat selects the archive compression format for build outputs.
	// Supported values are "gz", "xz", and "zip"; empty uses gzip.
	ArchiveFormat string `json:"archive_format,omitempty" yaml:"archive_format,omitempty"`
	// Env are additional environment variables.
	Env []string `json:"env" yaml:"env"`
	// Cache controls build cache setup.
	Cache CacheConfig `json:"cache,omitempty" yaml:"cache,omitempty"`
	// Dockerfile is the path to the Dockerfile used by Docker builds.
	Dockerfile string `json:"dockerfile,omitempty" yaml:"dockerfile,omitempty"`
	// Registry is the container registry used for Docker image references.
	Registry string `json:"registry,omitempty" yaml:"registry,omitempty"`
	// Image is the image name used for Docker builds.
	Image string `json:"image,omitempty" yaml:"image,omitempty"`
	// Tags are Docker image tags to apply.
	Tags []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	// BuildArgs are Docker build arguments.
	BuildArgs map[string]string `json:"build_args,omitempty" yaml:"build_args,omitempty"`
	// Push enables pushing Docker images after build.
	Push bool `json:"push,omitempty" yaml:"push,omitempty"`
	// Load loads a single-platform Docker image into the local daemon after build.
	Load bool `json:"load,omitempty" yaml:"load,omitempty"`
	// LinuxKitConfig is the path to the LinuxKit config file.
	LinuxKitConfig string `json:"linuxkit_config,omitempty" yaml:"linuxkit_config,omitempty"`
	// Formats is the list of LinuxKit output formats.
	// Supported values include iso, raw, qcow2, vmdk, vhd, gcp, aws, docker, tar, and kernel+initrd.
	Formats []string `json:"formats,omitempty" yaml:"formats,omitempty"`
}

// AppleConfig holds macOS Apple pipeline settings loaded from .core/build.yaml.
// Pointer booleans preserve the difference between an explicit false and an unset field.
type AppleConfig struct {
	TeamID       string `json:"team_id,omitempty" yaml:"team_id,omitempty"`
	BundleID     string `json:"bundle_id,omitempty" yaml:"bundle_id,omitempty"`
	Arch         string `json:"arch,omitempty" yaml:"arch,omitempty"`
	CertIdentity string `json:"cert_identity,omitempty" yaml:"cert_identity,omitempty"`
	ProfilePath  string `json:"profile_path,omitempty" yaml:"profile_path,omitempty"`
	KeychainPath string `json:"keychain_path,omitempty" yaml:"keychain_path,omitempty"`
	MetadataPath string `json:"metadata_path,omitempty" yaml:"metadata_path,omitempty"`

	Sign       *bool `json:"sign,omitempty" yaml:"sign,omitempty"`
	Notarise   *bool `json:"notarise,omitempty" yaml:"notarise,omitempty"`
	DMG        *bool `json:"dmg,omitempty" yaml:"dmg,omitempty"`
	TestFlight *bool `json:"testflight,omitempty" yaml:"testflight,omitempty"`
	AppStore   *bool `json:"appstore,omitempty" yaml:"appstore,omitempty"`

	APIKeyID       string `json:"api_key_id,omitempty" yaml:"api_key_id,omitempty"`
	APIKeyIssuerID string `json:"api_key_issuer_id,omitempty" yaml:"api_key_issuer_id,omitempty"`
	APIKeyPath     string `json:"api_key_path,omitempty" yaml:"api_key_path,omitempty"`
	AppleID        string `json:"apple_id,omitempty" yaml:"apple_id,omitempty"`
	Password       string `json:"password,omitempty" yaml:"password,omitempty"`

	BundleDisplayName string           `json:"bundle_display_name,omitempty" yaml:"bundle_display_name,omitempty"`
	MinSystemVersion  string           `json:"min_system_version,omitempty" yaml:"min_system_version,omitempty"`
	Category          string           `json:"category,omitempty" yaml:"category,omitempty"`
	Copyright         string           `json:"copyright,omitempty" yaml:"copyright,omitempty"`
	PrivacyPolicyURL  string           `json:"privacy_policy_url,omitempty" yaml:"privacy_policy_url,omitempty"`
	DMGBackground     string           `json:"dmg_background,omitempty" yaml:"dmg_background,omitempty"`
	DMGVolumeName     string           `json:"dmg_volume_name,omitempty" yaml:"dmg_volume_name,omitempty"`
	EntitlementsPath  string           `json:"entitlements_path,omitempty" yaml:"entitlements_path,omitempty"`
	XcodeCloud        XcodeCloudConfig `json:"xcode_cloud,omitempty" yaml:"xcode_cloud,omitempty"`
}

// XcodeCloudConfig defines the Xcode Cloud workflow metadata stored in build config.
type XcodeCloudConfig struct {
	Workflow string              `json:"workflow,omitempty" yaml:"workflow,omitempty"`
	Triggers []XcodeCloudTrigger `json:"triggers,omitempty" yaml:"triggers,omitempty"`
}

// XcodeCloudTrigger defines a single Xcode Cloud trigger rule.
type XcodeCloudTrigger struct {
	Branch string `json:"branch,omitempty" yaml:"branch,omitempty"`
	Tag    string `json:"tag,omitempty" yaml:"tag,omitempty"`
	Action string `json:"action,omitempty" yaml:"action,omitempty"`
}

// TargetConfig defines a build target in the config file.
// This is separate from Target to allow for additional config-specific fields.
//
// cfg.Targets = []build.TargetConfig{{OS: "linux", Arch: "amd64"}, {OS: "darwin", Arch: "arm64"}}
type TargetConfig struct {
	// OS is the target operating system (e.g., "linux", "darwin", "windows").
	OS string `json:"os" yaml:"os"`
	// Arch is the target architecture (e.g., "amd64", "arm64").
	Arch string `json:"arch" yaml:"arch"`
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
		return nil, coreerr.E("build.LoadConfigAtPath", "failed to read config file", err)
	}

	cfg := DefaultConfig()
	data := []byte(content)
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, coreerr.E("build.LoadConfigAtPath", "failed to parse config file", err)
	}

	// Apply defaults for any missing fields
	applyDefaults(cfg)

	// Expand environment variables after defaults so overrides can still be
	// expressed declaratively in config files.
	cfg.ExpandEnv()

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
		Sign:     signing.DefaultSignConfig(),
		LinuxKit: DefaultLinuxKitConfig(),
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

	if cfg.LinuxKit.Base == "" {
		cfg.LinuxKit.Base = defaults.LinuxKit.Base
	}
	if cfg.LinuxKit.Mounts == nil {
		cfg.LinuxKit.Mounts = append([]string(nil), defaults.LinuxKit.Mounts...)
	}
	if cfg.LinuxKit.Formats == nil {
		cfg.LinuxKit.Formats = append([]string(nil), defaults.LinuxKit.Formats...)
	}

}

// ExpandEnv expands environment variables across the build config.
//
// cfg.ExpandEnv() // expands $APP_NAME, $IMAGE_TAG, $GPG_KEY_ID, etc.
func (cfg *BuildConfig) ExpandEnv() {
	if cfg == nil {
		return
	}

	cfg.Project.Name = expandEnv(cfg.Project.Name)
	cfg.Project.Description = expandEnv(cfg.Project.Description)
	cfg.Project.Main = expandEnv(cfg.Project.Main)
	cfg.Project.Binary = expandEnv(cfg.Project.Binary)

	cfg.Build.Type = expandEnv(cfg.Build.Type)
	cfg.Build.DenoBuild = expandEnv(cfg.Build.DenoBuild)
	cfg.Build.WebView2 = expandEnv(cfg.Build.WebView2)
	cfg.Build.ArchiveFormat = expandEnv(cfg.Build.ArchiveFormat)
	cfg.Build.Dockerfile = expandEnv(cfg.Build.Dockerfile)
	cfg.Build.Registry = expandEnv(cfg.Build.Registry)
	cfg.Build.Image = expandEnv(cfg.Build.Image)
	cfg.Build.LinuxKitConfig = expandEnv(cfg.Build.LinuxKitConfig)

	cfg.Apple.TeamID = expandEnv(cfg.Apple.TeamID)
	cfg.Apple.BundleID = expandEnv(cfg.Apple.BundleID)
	cfg.Apple.Arch = expandEnv(cfg.Apple.Arch)
	cfg.Apple.CertIdentity = expandEnv(cfg.Apple.CertIdentity)
	cfg.Apple.ProfilePath = expandEnv(cfg.Apple.ProfilePath)
	cfg.Apple.KeychainPath = expandEnv(cfg.Apple.KeychainPath)
	cfg.Apple.MetadataPath = expandEnv(cfg.Apple.MetadataPath)
	cfg.Apple.APIKeyID = expandEnv(cfg.Apple.APIKeyID)
	cfg.Apple.APIKeyIssuerID = expandEnv(cfg.Apple.APIKeyIssuerID)
	cfg.Apple.APIKeyPath = expandEnv(cfg.Apple.APIKeyPath)
	cfg.Apple.AppleID = expandEnv(cfg.Apple.AppleID)
	cfg.Apple.Password = expandEnv(cfg.Apple.Password)
	cfg.Apple.BundleDisplayName = expandEnv(cfg.Apple.BundleDisplayName)
	cfg.Apple.MinSystemVersion = expandEnv(cfg.Apple.MinSystemVersion)
	cfg.Apple.Category = expandEnv(cfg.Apple.Category)
	cfg.Apple.Copyright = expandEnv(cfg.Apple.Copyright)
	cfg.Apple.PrivacyPolicyURL = expandEnv(cfg.Apple.PrivacyPolicyURL)
	cfg.Apple.DMGBackground = expandEnv(cfg.Apple.DMGBackground)
	cfg.Apple.DMGVolumeName = expandEnv(cfg.Apple.DMGVolumeName)
	cfg.Apple.EntitlementsPath = expandEnv(cfg.Apple.EntitlementsPath)
	cfg.Apple.XcodeCloud.Workflow = expandEnv(cfg.Apple.XcodeCloud.Workflow)

	cfg.Build.Flags = expandEnvSlice(cfg.Build.Flags)
	cfg.Build.LDFlags = expandEnvSlice(cfg.Build.LDFlags)
	cfg.Build.BuildTags = expandEnvSlice(cfg.Build.BuildTags)
	cfg.Build.Env = expandEnvSlice(cfg.Build.Env)
	cfg.Build.Tags = expandEnvSlice(cfg.Build.Tags)
	cfg.Build.Formats = expandEnvSlice(cfg.Build.Formats)
	cfg.LinuxKit.Base = expandEnv(cfg.LinuxKit.Base)
	cfg.LinuxKit.Packages = expandEnvSlice(cfg.LinuxKit.Packages)
	cfg.LinuxKit.Mounts = expandEnvSlice(cfg.LinuxKit.Mounts)
	cfg.LinuxKit.Formats = expandEnvSlice(cfg.LinuxKit.Formats)
	cfg.LinuxKit.Registry = expandEnv(cfg.LinuxKit.Registry)
	cfg.Apple.XcodeCloud.Triggers = expandXcodeCloudTriggers(cfg.Apple.XcodeCloud.Triggers)

	cfg.Build.Cache.Directory = expandEnv(cfg.Build.Cache.Directory)
	cfg.Build.Cache.KeyPrefix = expandEnv(cfg.Build.Cache.KeyPrefix)
	cfg.Build.Cache.Paths = expandEnvSlice(cfg.Build.Cache.Paths)
	cfg.Build.Cache.RestoreKeys = expandEnvSlice(cfg.Build.Cache.RestoreKeys)

	cfg.Build.BuildArgs = expandEnvMap(cfg.Build.BuildArgs)
	cfg.Targets = expandTargetConfigs(cfg.Targets)

	cfg.Sign.ExpandEnv()
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

func expandXcodeCloudTriggers(values []XcodeCloudTrigger) []XcodeCloudTrigger {
	if len(values) == 0 {
		return values
	}

	result := make([]XcodeCloudTrigger, len(values))
	for i, value := range values {
		result[i] = XcodeCloudTrigger{
			Branch: expandEnv(value.Branch),
			Tag:    expandEnv(value.Tag),
			Action: expandEnv(value.Action),
		}
	}
	return result
}

// CloneStringMap returns a shallow copy of a string map.
//
// clone := build.CloneStringMap(map[string]string{"VERSION": "v1.2.3"})
func CloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return values
	}

	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

// CloneBuildConfig returns a deep copy of a build config so callers can apply
// runtime overrides without mutating the persisted or caller-owned config.
//
//	clone := build.CloneBuildConfig(cfg)
func CloneBuildConfig(cfg *BuildConfig) *BuildConfig {
	if cfg == nil {
		return nil
	}

	clone := *cfg
	clone.Build = cloneBuild(cfg.Build)
	clone.Apple = cloneAppleConfig(cfg.Apple)
	clone.LinuxKit = cloneLinuxKitConfig(cfg.LinuxKit)
	clone.Targets = append([]TargetConfig(nil), cfg.Targets...)

	return &clone
}

func cloneBuild(value Build) Build {
	return Build{
		Type:           value.Type,
		CGO:            value.CGO,
		Obfuscate:      value.Obfuscate,
		DenoBuild:      value.DenoBuild,
		NSIS:           value.NSIS,
		WebView2:       value.WebView2,
		Flags:          append([]string(nil), value.Flags...),
		LDFlags:        append([]string(nil), value.LDFlags...),
		BuildTags:      append([]string(nil), value.BuildTags...),
		ArchiveFormat:  value.ArchiveFormat,
		Env:            append([]string(nil), value.Env...),
		Cache:          cloneCacheConfig(value.Cache),
		Dockerfile:     value.Dockerfile,
		Registry:       value.Registry,
		Image:          value.Image,
		Tags:           append([]string(nil), value.Tags...),
		BuildArgs:      CloneStringMap(value.BuildArgs),
		Push:           value.Push,
		Load:           value.Load,
		LinuxKitConfig: value.LinuxKitConfig,
		Formats:        append([]string(nil), value.Formats...),
	}
}

func cloneCacheConfig(value CacheConfig) CacheConfig {
	return CacheConfig{
		Enabled:     value.Enabled,
		Directory:   value.Directory,
		KeyPrefix:   value.KeyPrefix,
		Paths:       append([]string(nil), value.Paths...),
		RestoreKeys: append([]string(nil), value.RestoreKeys...),
	}
}

func cloneLinuxKitConfig(value LinuxKitConfig) LinuxKitConfig {
	return LinuxKitConfig{
		Base:     value.Base,
		Packages: append([]string(nil), value.Packages...),
		Mounts:   append([]string(nil), value.Mounts...),
		GPU:      value.GPU,
		Formats:  append([]string(nil), value.Formats...),
		Registry: value.Registry,
	}
}

func cloneAppleConfig(value AppleConfig) AppleConfig {
	clone := value

	if value.Sign != nil {
		sign := *value.Sign
		clone.Sign = &sign
	}
	if value.Notarise != nil {
		notarise := *value.Notarise
		clone.Notarise = &notarise
	}
	if value.DMG != nil {
		dmg := *value.DMG
		clone.DMG = &dmg
	}
	if value.TestFlight != nil {
		testFlight := *value.TestFlight
		clone.TestFlight = &testFlight
	}
	if value.AppStore != nil {
		appStore := *value.AppStore
		clone.AppStore = &appStore
	}

	clone.XcodeCloud = XcodeCloudConfig{
		Workflow: value.XcodeCloud.Workflow,
		Triggers: append([]XcodeCloudTrigger(nil), value.XcodeCloud.Triggers...),
	}

	return clone
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
