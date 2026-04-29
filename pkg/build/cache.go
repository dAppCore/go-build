// Package build provides project type detection and cross-compilation for the Core build system.
// This file handles build cache configuration and key generation.
package build

import (
	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/io"
	"gopkg.in/yaml.v3"
)

// DefaultCacheDirectory is the project-local cache metadata directory used when
// no cache directory is supplied.
//
//	cfg := build.CacheConfig{Enabled: true}
//	// SetupCache(io.Local, ".", &cfg) -> ".core/cache"
const DefaultCacheDirectory = ".core/cache"

// DefaultProcessCacheDirectory is the RFC-documented cache directory used by
// the single-argument SetupCache form when only environment wiring is needed.
const DefaultProcessCacheDirectory = "~/.cache/core-build"

// DefaultBuildCachePaths returns the project-local Go cache directories used
// when no cache paths are configured.
//
//	paths := build.DefaultBuildCachePaths("/workspace/project")
//	// ["/workspace/project/cache/go-build", "/workspace/project/cache/go-mod"]
func DefaultBuildCachePaths(baseDir string) []string {
	if core.Trim(baseDir) == "" {
		return []string{
			"cache/go-build",
			"cache/go-mod",
		}
	}

	return []string{
		ax.Join(baseDir, "cache", "go-build"),
		ax.Join(baseDir, "cache", "go-mod"),
	}
}

// CacheConfig holds build cache configuration loaded from .core/build.yaml.
//
//	cfg := build.CacheConfig{
//	    Enabled: true,
//	    Directory: ".core/cache",
//	    Paths: []string{"~/.cache/go-build", "~/go/pkg/mod"},
//	}
type CacheConfig struct {
	// Enabled turns cache setup on for the build.
	Enabled bool `json:"enabled" yaml:"enabled"`
	// Dir is where cache metadata is stored.
	Dir string `json:"dir,omitempty" yaml:"dir,omitempty"`
	// Directory is the deprecated alias for Dir.
	Directory string `json:"-" yaml:"-"`
	// KeyPrefix prefixes the generated cache key.
	KeyPrefix string `json:"key_prefix,omitempty" yaml:"key_prefix,omitempty"`
	// Paths are cache directories that should exist before the build starts.
	Paths []string `json:"paths,omitempty" yaml:"paths,omitempty"`
	// RestoreKeys are fallback prefixes used when the exact cache key is not present.
	RestoreKeys []string `json:"restore_keys,omitempty" yaml:"restore_keys,omitempty"`
}

// MarshalYAML emits the documented cache configuration shape with the Dir field.
//
//	data, err := yaml.Marshal(build.CacheConfig{Enabled: true, Dir: ".core/cache"})
func (c CacheConfig) MarshalYAML() core.Result {
	type rawCacheConfig struct {
		Enabled     bool     `yaml:"enabled"`
		Dir         string   `yaml:"dir,omitempty"`
		KeyPrefix   string   `yaml:"key_prefix,omitempty"`
		Paths       []string `yaml:"paths,omitempty"`
		RestoreKeys []string `yaml:"restore_keys,omitempty"`
	}

	return core.Ok(rawCacheConfig{
		Enabled:     c.Enabled,
		Dir:         c.effectiveDirectory(),
		KeyPrefix:   c.KeyPrefix,
		Paths:       c.Paths,
		RestoreKeys: c.RestoreKeys,
	})
}

// UnmarshalYAML accepts both the concise build config keys and the longer aliases.
//
//	err := yaml.Unmarshal([]byte("dir: .core/cache"), &cfg)
func (c *CacheConfig) UnmarshalYAML(value *yaml.Node) core.Result {
	type rawCacheConfig struct {
		Enabled     bool     `yaml:"enabled"`
		Directory   string   `yaml:"directory"`
		Dir         string   `yaml:"dir"`
		KeyPrefix   string   `yaml:"key_prefix"`
		Key         string   `yaml:"key"`
		Paths       []string `yaml:"paths"`
		RestoreKeys []string `yaml:"restore_keys"`
	}

	var raw rawCacheConfig
	if err := value.Decode(&raw); err != nil {
		return core.Fail(err)
	}

	c.Enabled = raw.Enabled
	c.Dir = firstNonEmpty(raw.Dir, raw.Directory)
	c.Directory = c.Dir
	c.KeyPrefix = firstNonEmpty(raw.KeyPrefix, raw.Key)
	c.Paths = raw.Paths
	c.RestoreKeys = raw.RestoreKeys

	return core.Ok(nil)
}

// SetupCache normalises cache paths and ensures the cache directories exist.
//
// The canonical form is the 3-argument variant:
//
//	err := build.SetupCache(io.Local, ".", &build.CacheConfig{
//	    Enabled: true,
//	    Paths: []string{"~/.cache/go-build", "~/go/pkg/mod"},
//	})
//
// A compatibility 1-argument form is also supported for the RFC-shaped API:
//
//	err := build.SetupCache(build.CacheConfig{Enabled: true})
func SetupCache(args ...any) core.Result {
	switch len(args) {
	case 1:
		cfg, ok := cacheConfigArg(args[0])
		if !ok || cfg == nil || !cfg.Enabled {
			return core.Ok(nil)
		}

		// The single-argument form configures the process environment for callers
		// that only need cache wiring and do not have a filesystem/project root.
		if cfg.effectiveDirectory() == "" {
			cfg.Dir = DefaultProcessCacheDirectory
			cfg.Directory = DefaultProcessCacheDirectory
		}
		if len(cfg.Paths) == 0 {
			cfg.Paths = []string{"~/.cache/go-build", "~/go/pkg/mod"}
		}
		applyCacheEnvironment(cfg)
		return core.Ok(nil)
	case 3:
		fs, _ := args[0].(io.Medium)
		dir, _ := args[1].(string)
		cfg, ok := args[2].(*CacheConfig)
		if !ok {
			return core.Fail(core.E("build.SetupCache", "third argument must be *CacheConfig", nil))
		}
		return setupCacheWithMedium(fs, dir, cfg)
	default:
		return core.Fail(core.E("build.SetupCache", "expected 1 or 3 arguments", nil))
	}
}

func cacheConfigArg(arg any) (*CacheConfig, bool) {
	switch cfg := arg.(type) {
	case CacheConfig:
		return &cfg, true
	case *CacheConfig:
		return cfg, true
	default:
		return nil, false
	}
}

func setupCacheWithMedium(fs io.Medium, dir string, cfg *CacheConfig) core.Result {
	if fs == nil || cfg == nil || !cfg.Enabled {
		return core.Ok(nil)
	}

	directory := cfg.effectiveDirectory()
	if directory == "" {
		directory = ax.Join(dir, DefaultCacheDirectory)
	}
	directory = normaliseCachePath(dir, directory)
	cfg.Dir = directory
	cfg.Directory = directory
	if len(cfg.Paths) == 0 {
		cfg.Paths = DefaultBuildCachePaths(dir)
	}

	created := fs.EnsureDir(directory)
	if !created.OK {
		return core.Fail(core.E("build.SetupCache", "failed to create cache directory", core.NewError(created.Error())))
	}

	normalisedPaths := make([]string, 0, len(cfg.Paths))
	for _, path := range cfg.Paths {
		path = normaliseCachePath(dir, path)
		if path == "" {
			continue
		}
		created = fs.EnsureDir(path)
		if !created.OK {
			return core.Fail(core.E("build.SetupCache", "failed to create cache path "+path, core.NewError(created.Error())))
		}
		normalisedPaths = append(normalisedPaths, path)
	}
	cfg.Paths = deduplicateStrings(normalisedPaths)

	return core.Ok(nil)
}

// SetupBuildCache prepares the cache configuration stored on a build config.
//
//	err := build.SetupBuildCache(io.Local, ".", cfg)
func SetupBuildCache(fs io.Medium, dir string, cfg *BuildConfig) core.Result {
	if fs == nil || cfg == nil {
		return core.Ok(nil)
	}

	return setupCacheWithMedium(fs, dir, &cfg.Build.Cache)
}

// CacheKey returns a deterministic cache key from go.sum, go.work.sum, and the target platform.
//
//	key := build.CacheKey(io.Local, ".", "linux", "amd64") // "go-linux-amd64-abc123..."
func CacheKey(fs io.Medium, dir, goos, goarch string) string {
	var seed []byte

	if fs != nil {
		for _, name := range []string{"go.sum", "go.work.sum"} {
			if content := fs.Read(ax.Join(dir, name)); content.OK {
				seed = append(seed, content.Value.(string)...)
				seed = append(seed, '\n')
			}
		}
		if len(seed) == 0 {
			seed = append(seed, '\n')
		}
	}

	seed = append(seed, goos...)
	seed = append(seed, '\n')
	seed = append(seed, goarch...)

	suffix := core.SHA256Hex(seed)[:12]

	return core.Join("-", "go", goos, goarch, suffix)
}

// CacheKeyWithConfig returns a deterministic cache key and applies the optional
// cache key prefix from configuration.
//
//	key := build.CacheKeyWithConfig(io.Local, ".", "linux", "amd64", &cfg.Cache)
//	// "demo-go-linux-amd64-abc123..."
func CacheKeyWithConfig(fs io.Medium, dir, goos, goarch string, cfg *CacheConfig) string {
	key := CacheKey(fs, dir, goos, goarch)
	if cfg == nil {
		return key
	}

	prefix := core.Trim(cfg.KeyPrefix)
	if prefix == "" {
		return key
	}

	return core.Join("-", prefix, key)
}

// CacheRestoreKeys returns the configured restore-key prefixes in stable order.
//
//	keys := build.CacheRestoreKeys(&build.CacheConfig{
//	    KeyPrefix: "demo",
//	    RestoreKeys: []string{"go-", "core-"},
//	})
//	// ["demo", "go-", "core-"]
func CacheRestoreKeys(cfg *CacheConfig) []string {
	if cfg == nil {
		return nil
	}

	keys := make([]string, 0, 1+len(cfg.RestoreKeys))
	if prefix := core.Trim(cfg.KeyPrefix); prefix != "" {
		keys = append(keys, prefix)
	}
	keys = append(keys, cfg.RestoreKeys...)

	return deduplicateStrings(keys)
}

// CacheEnvironment returns environment variables derived from the cache config.
//
//	env := build.CacheEnvironment(&build.CacheConfig{Enabled: true, Paths: []string{"/tmp/go-build"}})
func CacheEnvironment(cfg *CacheConfig) []string {
	if cfg == nil || !cfg.Enabled {
		return nil
	}

	var env []string

	for _, path := range cfg.Paths {
		switch cacheEnvironmentName(path) {
		case "GOCACHE":
			env = appendIfMissing(env, "GOCACHE="+path)
		case "GOMODCACHE":
			env = appendIfMissing(env, "GOMODCACHE="+path)
		}
	}

	return deduplicateStrings(env)
}

func cacheEnvironmentName(path string) string {
	base := core.Lower(ax.Base(path))

	switch base {
	case "go-build", "gocache":
		return "GOCACHE"
	case "go-mod", "gomodcache":
		return "GOMODCACHE"
	default:
		return ""
	}
}

func appendIfMissing(values []string, value string) []string {
	for _, current := range values {
		if current == value {
			return values
		}
	}
	return append(values, value)
}

func applyCacheEnvironment(cfg *CacheConfig) {
	setenv := core.Setenv
	for _, env := range CacheEnvironment(cfg) {
		parts := core.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		if set := setenv(parts[0], parts[1]); !set.OK {
			continue
		}
	}
}

func normaliseCachePath(baseDir, path string) string {
	path = core.Trim(path)
	if path == "" {
		return ""
	}

	if core.HasPrefix(path, "~") {
		home := core.Env("HOME")
		if home != "" {
			if path == "~" {
				return ax.Clean(home)
			}
			if core.HasPrefix(path, "~/") {
				return ax.Join(home, core.TrimPrefix(path, "~/"))
			}
		}
	}

	if ax.IsAbs(path) {
		return ax.Clean(path)
	}

	return ax.Join(baseDir, path)
}

func deduplicateStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if core.Trim(value) != "" {
			return value
		}
	}
	return ""
}

func (c CacheConfig) effectiveDirectory() string {
	if core.Trim(c.Dir) != "" {
		return c.Dir
	}
	return c.Directory
}
