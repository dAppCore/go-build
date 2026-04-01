// Package build provides project type detection and cross-compilation for the Core build system.
// This file handles build cache configuration and key generation.
package build

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
	"gopkg.in/yaml.v3"
)

// CacheConfig holds build cache configuration loaded from .core/build.yaml.
//
//	cfg := build.CacheConfig{
//	    Enabled: true,
//	    Directory: ".core/cache",
//	    Paths: []string{"~/.cache/go-build", "~/go/pkg/mod"},
//	}
type CacheConfig struct {
	// Enabled turns cache setup on for the build.
	Enabled bool `yaml:"enabled"`
	// Directory is where cache metadata is stored.
	Directory string `yaml:"dir,omitempty"`
	// KeyPrefix prefixes the generated cache key.
	KeyPrefix string `yaml:"key_prefix,omitempty"`
	// Paths are cache directories that should exist before the build starts.
	Paths []string `yaml:"paths,omitempty"`
	// RestoreKeys are fallback prefixes used when the exact cache key is not present.
	RestoreKeys []string `yaml:"restore_keys,omitempty"`
}

// UnmarshalYAML accepts both the concise build config keys and the longer aliases.
//
//	err := yaml.Unmarshal([]byte("dir: .core/cache"), &cfg)
func (c *CacheConfig) UnmarshalYAML(value *yaml.Node) error {
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
		return err
	}

	c.Enabled = raw.Enabled
	c.Directory = firstNonEmpty(raw.Directory, raw.Dir)
	c.KeyPrefix = firstNonEmpty(raw.KeyPrefix, raw.Key)
	c.Paths = raw.Paths
	c.RestoreKeys = raw.RestoreKeys

	return nil
}

// SetupCache normalises cache paths and ensures the cache directories exist.
//
//	err := build.SetupCache(io.Local, ".", &build.CacheConfig{
//	    Enabled: true,
//	    Paths: []string{"~/.cache/go-build", "~/go/pkg/mod"},
//	})
func SetupCache(fs io.Medium, dir string, cfg *CacheConfig) error {
	if fs == nil || cfg == nil || !cfg.Enabled {
		return nil
	}

	if cfg.Directory == "" {
		cfg.Directory = ax.Join(dir, ConfigDir, "cache")
	}
	cfg.Directory = normaliseCachePath(dir, cfg.Directory)

	if err := fs.EnsureDir(cfg.Directory); err != nil {
		return coreerr.E("build.SetupCache", "failed to create cache directory", err)
	}

	normalisedPaths := make([]string, 0, len(cfg.Paths))
	for _, path := range cfg.Paths {
		path = normaliseCachePath(dir, path)
		if path == "" {
			continue
		}
		if err := fs.EnsureDir(path); err != nil {
			return coreerr.E("build.SetupCache", "failed to create cache path "+path, err)
		}
		normalisedPaths = append(normalisedPaths, path)
	}
	cfg.Paths = deduplicateStrings(normalisedPaths)

	return nil
}

// SetupBuildCache prepares the cache configuration stored on a build config.
//
//	err := build.SetupBuildCache(io.Local, ".", cfg)
func SetupBuildCache(fs io.Medium, dir string, cfg *BuildConfig) error {
	if fs == nil || cfg == nil {
		return nil
	}

	return SetupCache(fs, dir, &cfg.Build.Cache)
}

// CacheKey returns a deterministic cache key for the build configuration and target.
//
//	key := build.CacheKey("core-build", build.Target{OS: "linux", Arch: "amd64"}, &build.CacheConfig{
//	    KeyPrefix: "main",
//	})
func CacheKey(buildName string, target Target, cfg *CacheConfig) string {
	if buildName == "" {
		buildName = "build"
	}

	keyPrefix := buildName
	if cfg != nil && cfg.KeyPrefix != "" {
		keyPrefix = cfg.KeyPrefix
	}

	snapshot := cacheKeySnapshot(buildName, target, cfg)
	sum := sha256.Sum256([]byte(snapshot))
	suffix := hex.EncodeToString(sum[:])[:12]

	return core.Join("-", keyPrefix, target.OS, target.Arch, suffix)
}

func cacheKeySnapshot(buildName string, target Target, cfg *CacheConfig) string {
	parts := []string{
		"build",
		buildName,
		target.OS,
		target.Arch,
	}

	if cfg == nil {
		return core.Join("\n", parts...)
	}

	parts = append(parts,
		strconv.FormatBool(cfg.Enabled),
		cfg.Directory,
		cfg.KeyPrefix,
	)

	paths := deduplicateStrings(append([]string(nil), cfg.Paths...))
	sort.Strings(paths)
	parts = append(parts, "paths:"+core.Join(",", paths...))

	restoreKeys := deduplicateStrings(append([]string(nil), cfg.RestoreKeys...))
	sort.Strings(restoreKeys)
	parts = append(parts, "restore:"+core.Join(",", restoreKeys...))

	return core.Join("\n", parts...)
}

func normaliseCachePath(baseDir, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	if strings.HasPrefix(path, "~") {
		home := core.Env("HOME")
		if home != "" {
			if path == "~" {
				return ax.Clean(home)
			}
			if strings.HasPrefix(path, "~/") {
				return ax.Join(home, strings.TrimPrefix(path, "~/"))
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
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
