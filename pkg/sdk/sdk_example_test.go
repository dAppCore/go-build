package sdk

import (
	core "dappco.re/go"
	yaml "gopkg.in/yaml.v3"
)

// --- v0.9.0 generated usage examples ---
func ExampleNew() {
	_ = New(core.Path(core.TempDir(), "go-build-compliance"), &Config{})
	core.Println("New")
	// Output: New
}

func ExampleCloneConfig() {
	_ = CloneConfig(&Config{})
	core.Println("CloneConfig")
	// Output: CloneConfig
}

func ExampleSDK_Config() {
	subject := &SDK{}
	_ = subject.Config()
	core.Println("SDK_Config")
	// Output: SDK_Config
}

func ExampleConfig_ApplyDefaults() {
	subject := &Config{}
	subject.ApplyDefaults()
	core.Println("Config_ApplyDefaults")
	// Output: Config_ApplyDefaults
}

func ExampleSDK_SetVersion() {
	subject := &SDK{}
	subject.SetVersion("v1.2.3")
	core.Println("SDK_SetVersion")
	// Output: SDK_SetVersion
}

func ExampleDefaultConfig() {
	_ = DefaultConfig()
	core.Println("DefaultConfig")
	// Output: DefaultConfig
}

func ExampleDiffConfig_UnmarshalYAML() {
	subject := &DiffConfig{}
	_ = subject.UnmarshalYAML(&yaml.Node{Kind: yaml.ScalarNode, Value: "false"})
	core.Println("DiffConfig_UnmarshalYAML")
	// Output: DiffConfig_UnmarshalYAML
}

func ExampleSDK_Generate() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &SDK{}
	_ = subject.Generate(ctx)
	core.Println("SDK_Generate")
	// Output: SDK_Generate
}

func ExampleSDK_GenerateWithStatus() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &SDK{}
	_, _ = subject.GenerateWithStatus(ctx)
	core.Println("SDK_GenerateWithStatus")
	// Output: SDK_GenerateWithStatus
}

func ExampleSDK_GenerateLanguage() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &SDK{}
	_ = subject.GenerateLanguage(ctx, "go")
	core.Println("SDK_GenerateLanguage")
	// Output: SDK_GenerateLanguage
}

func ExampleSDK_GenerateLanguageWithStatus() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &SDK{}
	_, _ = subject.GenerateLanguageWithStatus(ctx, "go")
	core.Println("SDK_GenerateLanguageWithStatus")
	// Output: SDK_GenerateLanguageWithStatus
}
