package release

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated usage examples ---
func ExampleGenerate() {
	_ = Generate(core.Path(core.TempDir(), "go-build-compliance"), "agent", "agent")
	core.Println("Generate")
	// Output: Generate
}

func ExampleGenerateWithContext() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	_ = GenerateWithContext(ctx, core.Path(core.TempDir(), "go-build-compliance"), "agent", "agent")
	core.Println("GenerateWithContext")
	// Output: GenerateWithContext
}

func ExampleGenerateWithConfig() {
	_ = GenerateWithConfig(core.Path(core.TempDir(), "go-build-compliance"), "agent", "agent", &ChangelogConfig{})
	core.Println("GenerateWithConfig")
	// Output: GenerateWithConfig
}

func ExampleGenerateWithConfigWithContext() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	_ = GenerateWithConfigWithContext(ctx, core.Path(core.TempDir(), "go-build-compliance"), "agent", "agent", &ChangelogConfig{})
	core.Println("GenerateWithConfigWithContext")
	// Output: GenerateWithConfigWithContext
}

func ExampleParseCommitType() {
	_ = ParseCommitType("agent")
	core.Println("ParseCommitType")
	// Output: ParseCommitType
}
