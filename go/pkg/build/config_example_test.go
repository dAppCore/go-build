package build

import core "dappco.re/go"

// ExampleTargetConfig_MarshalYAML references TargetConfig.MarshalYAML on this package API surface.
func ExampleTargetConfig_MarshalYAML() {
	_ = (*TargetConfig).MarshalYAML
	core.Println("TargetConfig.MarshalYAML")
	// Output: TargetConfig.MarshalYAML
}

// ExampleTargetConfig_UnmarshalYAML references TargetConfig.UnmarshalYAML on this package API surface.
func ExampleTargetConfig_UnmarshalYAML() {
	_ = (*TargetConfig).UnmarshalYAML
	core.Println("TargetConfig.UnmarshalYAML")
	// Output: TargetConfig.UnmarshalYAML
}

// ExampleBuildConfig_UnmarshalYAML references BuildConfig.UnmarshalYAML on this package API surface.
func ExampleBuildConfig_UnmarshalYAML() {
	_ = (*BuildConfig).UnmarshalYAML
	core.Println("BuildConfig.UnmarshalYAML")
	// Output: BuildConfig.UnmarshalYAML
}

// ExampleBuildConfig_MarshalYAML references BuildConfig.MarshalYAML on this package API surface.
func ExampleBuildConfig_MarshalYAML() {
	_ = (*BuildConfig).MarshalYAML
	core.Println("BuildConfig.MarshalYAML")
	// Output: BuildConfig.MarshalYAML
}

// ExampleLoadConfig references LoadConfig on this package API surface.
func ExampleLoadConfig() {
	_ = LoadConfig
	core.Println("LoadConfig")
	// Output: LoadConfig
}

// ExampleLoadConfigAtPath references LoadConfigAtPath on this package API surface.
func ExampleLoadConfigAtPath() {
	_ = LoadConfigAtPath
	core.Println("LoadConfigAtPath")
	// Output: LoadConfigAtPath
}

// ExampleDefaultConfig references DefaultConfig on this package API surface.
func ExampleDefaultConfig() {
	_ = DefaultConfig
	core.Println("DefaultConfig")
	// Output: DefaultConfig
}

// ExampleResolveOutputMedium references ResolveOutputMedium on this package API surface.
func ExampleResolveOutputMedium() {
	_ = ResolveOutputMedium
	core.Println("ResolveOutputMedium")
	// Output: ResolveOutputMedium
}

// ExampleMediumIsLocal references MediumIsLocal on this package API surface.
func ExampleMediumIsLocal() {
	_ = MediumIsLocal
	core.Println("MediumIsLocal")
	// Output: MediumIsLocal
}

// ExampleCopyMediumPath references CopyMediumPath on this package API surface.
func ExampleCopyMediumPath() {
	_ = CopyMediumPath
	core.Println("CopyMediumPath")
	// Output: CopyMediumPath
}

// ExampleBuildConfig_ExpandEnv references BuildConfig.ExpandEnv on this package API surface.
func ExampleBuildConfig_ExpandEnv() {
	_ = (*BuildConfig).ExpandEnv
	core.Println("BuildConfig.ExpandEnv")
	// Output: BuildConfig.ExpandEnv
}

// ExampleCloneStringMap references CloneStringMap on this package API surface.
func ExampleCloneStringMap() {
	_ = CloneStringMap
	core.Println("CloneStringMap")
	// Output: CloneStringMap
}

// ExampleCloneBuildConfig references CloneBuildConfig on this package API surface.
func ExampleCloneBuildConfig() {
	_ = CloneBuildConfig
	core.Println("CloneBuildConfig")
	// Output: CloneBuildConfig
}

// ExampleConfigPath references ConfigPath on this package API surface.
func ExampleConfigPath() {
	_ = ConfigPath
	core.Println("ConfigPath")
	// Output: ConfigPath
}

// ExampleConfigExists references ConfigExists on this package API surface.
func ExampleConfigExists() {
	_ = ConfigExists
	core.Println("ConfigExists")
	// Output: ConfigExists
}

// ExampleBuildConfig_TargetsIter references BuildConfig.TargetsIter on this package API surface.
func ExampleBuildConfig_TargetsIter() {
	_ = (*BuildConfig).TargetsIter
	core.Println("BuildConfig.TargetsIter")
	// Output: BuildConfig.TargetsIter
}

// ExampleBuildConfig_ToTargets references BuildConfig.ToTargets on this package API surface.
func ExampleBuildConfig_ToTargets() {
	_ = (*BuildConfig).ToTargets
	core.Println("BuildConfig.ToTargets")
	// Output: BuildConfig.ToTargets
}
