package service

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated usage examples ---
func ExampleResolveConfig() {
	_, _ = ResolveConfig(core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("ResolveConfig")
	// Output: ResolveConfig
}

func ExampleDefaultConfig() {
	_ = DefaultConfig(core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("DefaultConfig")
	// Output: DefaultConfig
}

func ExampleConfig_Normalized() {
	subject := Config{}
	_ = subject.Normalized()
	core.Println("Config_Normalized")
	// Output: Config_Normalized
}

func ExampleResolveNativeFormat() {
	_, _ = ResolveNativeFormat("tar.gz")
	core.Println("ResolveNativeFormat")
	// Output: ResolveNativeFormat
}
