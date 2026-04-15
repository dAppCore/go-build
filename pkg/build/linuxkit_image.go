package build

import "strings"

// LinuxKitImage models an immutable LinuxKit image definition.
//
//	image := build.LinuxKit(
//		build.WithBase("core-dev"),
//		build.WithPackages("git", "task"),
//		build.WithMount("/workspace"),
//		build.WithGPU(true),
//	)
type LinuxKitImage struct {
	Config LinuxKitConfig
}

// LinuxKitConfig defines an immutable LinuxKit image.
//
//	cfg := build.DefaultLinuxKitConfig()
type LinuxKitConfig struct {
	Base     string   `json:"base,omitempty" yaml:"base,omitempty"`
	Packages []string `json:"packages,omitempty" yaml:"packages,omitempty"`
	Mounts   []string `json:"mounts,omitempty" yaml:"mounts,omitempty"`
	GPU      bool     `json:"gpu,omitempty" yaml:"gpu,omitempty"`
	Formats  []string `json:"formats,omitempty" yaml:"formats,omitempty"`
	Registry string   `json:"registry,omitempty" yaml:"registry,omitempty"`
}

// LinuxKitOption configures an immutable LinuxKit image definition.
type LinuxKitOption func(*LinuxKitConfig)

// DefaultLinuxKitConfig returns the RFC defaults for immutable image builds.
func DefaultLinuxKitConfig() LinuxKitConfig {
	return LinuxKitConfig{
		Base:     "core-dev",
		Packages: []string{},
		Mounts:   []string{"/workspace"},
		GPU:      false,
		Formats:  []string{"oci", "apple"},
	}
}

// LinuxKit builds an immutable LinuxKit image definition with sensible defaults.
func LinuxKit(opts ...LinuxKitOption) *LinuxKitImage {
	cfg := DefaultLinuxKitConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	cfg = normalizeLinuxKitConfig(cfg)
	return &LinuxKitImage{Config: cfg}
}

// WithBase overrides the base image template name.
func WithBase(base string) LinuxKitOption {
	return func(cfg *LinuxKitConfig) {
		cfg.Base = strings.TrimSpace(base)
	}
}

// WithPackages appends extra OS packages to the immutable image.
func WithPackages(packages ...string) LinuxKitOption {
	return func(cfg *LinuxKitConfig) {
		cfg.Packages = append(cfg.Packages, packages...)
	}
}

// WithMount appends a writable mount point exposed inside the image.
func WithMount(path string) LinuxKitOption {
	return func(cfg *LinuxKitConfig) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		cfg.Mounts = append(cfg.Mounts, path)
	}
}

// WithGPU toggles GPU support for the immutable image.
func WithGPU(enabled bool) LinuxKitOption {
	return func(cfg *LinuxKitConfig) {
		cfg.GPU = enabled
	}
}

// WithFormats overrides the requested output formats.
func WithFormats(formats ...string) LinuxKitOption {
	return func(cfg *LinuxKitConfig) {
		cfg.Formats = normalizeLinuxKitFormats(formats)
	}
}

// WithRegistry sets the OCI registry namespace for image publication metadata.
func WithRegistry(registry string) LinuxKitOption {
	return func(cfg *LinuxKitConfig) {
		cfg.Registry = strings.TrimSpace(registry)
	}
}

func normalizeLinuxKitValues(values []string) []string {
	if len(values) == 0 {
		return values
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}

func normalizeLinuxKitFormats(values []string) []string {
	if len(values) == 0 {
		return values
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}

func normalizeLinuxKitConfig(cfg LinuxKitConfig) LinuxKitConfig {
	cfg = applyLinuxKitDefaults(cfg)

	cfg.Base = strings.TrimSpace(cfg.Base)
	cfg.Registry = strings.TrimSpace(cfg.Registry)
	cfg.Packages = normalizeLinuxKitValues(cfg.Packages)

	cfg.Mounts = normalizeLinuxKitValues(cfg.Mounts)
	cfg.Formats = normalizeLinuxKitFormats(cfg.Formats)
	cfg = applyLinuxKitDefaults(cfg)

	return cfg
}

func applyLinuxKitDefaults(cfg LinuxKitConfig) LinuxKitConfig {
	defaults := DefaultLinuxKitConfig()

	if strings.TrimSpace(cfg.Base) == "" {
		cfg.Base = defaults.Base
	}
	if len(cfg.Mounts) == 0 {
		cfg.Mounts = append([]string(nil), defaults.Mounts...)
	}
	if len(cfg.Formats) == 0 {
		cfg.Formats = append([]string(nil), defaults.Formats...)
	}

	return cfg
}
