package build

import core "dappco.re/go"

// ExampleDefaultBuildCachePaths references DefaultBuildCachePaths on this package API surface.
func ExampleDefaultBuildCachePaths() {
	_ = DefaultBuildCachePaths
	core.Println("DefaultBuildCachePaths")
	// Output: DefaultBuildCachePaths
}

// ExampleCacheConfig_MarshalYAML references CacheConfig.MarshalYAML on this package API surface.
func ExampleCacheConfig_MarshalYAML() {
	_ = (*CacheConfig).MarshalYAML
	core.Println("CacheConfig.MarshalYAML")
	// Output: CacheConfig.MarshalYAML
}

// ExampleCacheConfig_UnmarshalYAML references CacheConfig.UnmarshalYAML on this package API surface.
func ExampleCacheConfig_UnmarshalYAML() {
	_ = (*CacheConfig).UnmarshalYAML
	core.Println("CacheConfig.UnmarshalYAML")
	// Output: CacheConfig.UnmarshalYAML
}

// ExampleSetupCache references SetupCache on this package API surface.
func ExampleSetupCache() {
	_ = SetupCache
	core.Println("SetupCache")
	// Output: SetupCache
}

// ExampleSetupBuildCache references SetupBuildCache on this package API surface.
func ExampleSetupBuildCache() {
	_ = SetupBuildCache
	core.Println("SetupBuildCache")
	// Output: SetupBuildCache
}

// ExampleCacheKey references CacheKey on this package API surface.
func ExampleCacheKey() {
	_ = CacheKey
	core.Println("CacheKey")
	// Output: CacheKey
}

// ExampleCacheKeyWithConfig references CacheKeyWithConfig on this package API surface.
func ExampleCacheKeyWithConfig() {
	_ = CacheKeyWithConfig
	core.Println("CacheKeyWithConfig")
	// Output: CacheKeyWithConfig
}

// ExampleCacheRestoreKeys references CacheRestoreKeys on this package API surface.
func ExampleCacheRestoreKeys() {
	_ = CacheRestoreKeys
	core.Println("CacheRestoreKeys")
	// Output: CacheRestoreKeys
}

// ExampleCacheEnvironment references CacheEnvironment on this package API surface.
func ExampleCacheEnvironment() {
	_ = CacheEnvironment
	core.Println("CacheEnvironment")
	// Output: CacheEnvironment
}
