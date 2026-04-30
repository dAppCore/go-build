// Package build provides project type detection and cross-compilation for the Core build system.
// This file handles build options computation from config + discovery.
package build

import (
	"strconv"

	"dappco.re/go"
)

// BuildOptions holds computed build flags from config + discovery.
//
//	opts := build.ComputeOptions(cfg, discovery)
//	fmt.Println(opts.String()) // "-tags webkit2_41"
type BuildOptions struct {
	// Obfuscate uses garble instead of go build for obfuscation.
	Obfuscate bool
	// Tags holds de-duplicated Go build tags.
	Tags []string
	// NSIS enables Windows NSIS installer generation (Wails only).
	NSIS bool
	// WebView2 sets the WebView2 delivery method: download|embed|browser|error.
	WebView2 string
	// LDFlags holds linker flags merged from config.
	LDFlags []string
}

// ComputeOptions merges config + discovery into build flags.
// Handles distro-aware WebKit tag injection for Ubuntu 24.04+ Wails builds.
// Returns safe defaults when cfg or discovery is nil.
//
//	opts := build.ComputeOptions(cfg, result)
//	if opts.Obfuscate { /* use garble */ }
func ComputeOptions(cfg *BuildConfig, discovery *DiscoveryResult) *BuildOptions {
	options := &BuildOptions{}

	if cfg != nil {
		options.Obfuscate = cfg.Build.Obfuscate
		options.NSIS = cfg.Build.NSIS
		options.WebView2 = cfg.Build.WebView2
		options.LDFlags = append(options.LDFlags, cfg.Build.LDFlags...)
		options.Tags = append(options.Tags, cfg.Build.BuildTags...)
	}

	// Inject webkit2_41 for Ubuntu 24.04+ Wails builds.
	if shouldInjectWebKitTag(cfg, discovery) {
		options.Tags = InjectWebKitTag(options.Tags, discovery.Distro)
	}

	// De-duplicate tags
	options.Tags = deduplicateTags(options.Tags)

	return options
}

// ApplyOptions copies computed build options onto a runtime build config.
//
// build.ApplyOptions(cfg, build.ComputeOptions(config, discovery))
func ApplyOptions(cfg *Config, options *BuildOptions) {
	if cfg == nil || options == nil {
		return
	}

	if options.Obfuscate {
		cfg.Obfuscate = true
	}
	if options.NSIS {
		cfg.NSIS = true
	}
	if options.WebView2 != "" {
		cfg.WebView2 = options.WebView2
	}

	if len(options.LDFlags) > 0 {
		cfg.LDFlags = append([]string{}, options.LDFlags...)
	}

	if len(options.Tags) > 0 {
		cfg.BuildTags = deduplicateTags(append(cfg.BuildTags, options.Tags...))
	}
}

// InjectWebKitTag adds webkit2_41 tag for Ubuntu 24.04+ if not already present.
// Called automatically by ComputeOptions when discovery detects Linux.
//
//	tags := build.InjectWebKitTag(tags, "24.04")  // ["webkit2_41"]
//	tags := build.InjectWebKitTag(tags, "22.04")  // unchanged
func InjectWebKitTag(tags []string, distro string) []string {
	if distro == "" {
		return tags
	}

	// Check if the distro version is 24.04 or newer
	if !isUbuntu2404OrNewer(distro) {
		return tags
	}

	// Check if tag is already present
	for _, tag := range tags {
		if tag == "webkit2_41" {
			return tags
		}
	}

	return append([]string{"webkit2_41"}, tags...)
}

// String returns the options as a CLI flag string.
//
//	s := opts.String()  // "-tags webkit2_41 -ldflags '-s -w'"
func (o *BuildOptions) String() string {
	if o == nil {
		return ""
	}

	var parts []string

	if o.Obfuscate {
		parts = append(parts, "-obfuscated")
	}

	if len(o.Tags) > 0 {
		parts = append(parts, "-tags "+core.Join(",", o.Tags...))
	}

	if o.NSIS {
		parts = append(parts, "-nsis")
	}

	if o.WebView2 != "" {
		parts = append(parts, "-webview2 "+o.WebView2)
	}

	if len(o.LDFlags) > 0 {
		parts = append(parts, "-ldflags '"+core.Join(" ", o.LDFlags...)+"'")
	}

	return core.Join(" ", parts...)
}

func shouldInjectWebKitTag(cfg *BuildConfig, discovery *DiscoveryResult) bool {
	if discovery == nil || discovery.Distro == "" {
		return false
	}

	if discovery.OS != "" && core.Lower(core.Trim(discovery.OS)) != "linux" {
		return false
	}

	if cfg != nil && core.Lower(core.Trim(cfg.Build.Type)) == string(ProjectTypeWails) {
		return true
	}

	if core.Lower(core.Trim(discovery.ConfiguredType)) == string(ProjectTypeWails) {
		return true
	}

	if discovery.PrimaryStack == string(ProjectTypeWails) {
		return true
	}

	for _, projectType := range discovery.Types {
		if projectType == ProjectTypeWails {
			return true
		}
	}

	return false
}

// isUbuntu2404OrNewer checks if the distro version string represents Ubuntu 24.04+.
// Compares major.minor version numerically.
//
//	isUbuntu2404OrNewer("24.04") // true
//	isUbuntu2404OrNewer("22.04") // false
//	isUbuntu2404OrNewer("25.10") // true
func isUbuntu2404OrNewer(distro string) bool {
	parts := core.Split(distro, ".")
	if len(parts) != 2 {
		return false
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}

	// 24.04 or newer: major > 24, or major == 24 and minor >= 4
	if major > 24 {
		return true
	}
	if major == 24 && minor >= 4 {
		return true
	}
	return false
}

// deduplicateTags removes duplicate entries from a tag slice while preserving order.
//
//	deduplicateTags([]string{"a", "b", "a"}) // ["a", "b"]
func deduplicateTags(tags []string) []string {
	if len(tags) == 0 {
		return tags
	}

	seen := make(map[string]bool, len(tags))
	result := make([]string, 0, len(tags))

	for _, tag := range tags {
		if tag == "" {
			continue
		}
		if !seen[tag] {
			seen[tag] = true
			result = append(result, tag)
		}
	}

	return result
}
