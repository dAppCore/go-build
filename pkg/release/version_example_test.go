package release

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated usage examples ---
func ExampleDetermineVersion() {
	_ = DetermineVersion(core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("DetermineVersion")
	// Output: DetermineVersion
}

func ExampleDetermineVersionWithContext() {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	_ = DetermineVersionWithContext(ctx, core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("DetermineVersionWithContext")
	// Output: DetermineVersionWithContext
}

func ExampleIncrementVersion() {
	_ = IncrementVersion("agent")
	core.Println("IncrementVersion")
	// Output: IncrementVersion
}

func ExampleIncrementMinor() {
	_ = IncrementMinor("agent")
	core.Println("IncrementMinor")
	// Output: IncrementMinor
}

func ExampleIncrementMajor() {
	_ = IncrementMajor("agent")
	core.Println("IncrementMajor")
	// Output: IncrementMajor
}

func ExampleParseVersion() {
	_ = ParseVersion("v1.2.3")
	core.Println("ParseVersion")
	// Output: ParseVersion
}

func ExampleValidateVersion() {
	_ = ValidateVersion("v1.2.3")
	core.Println("ValidateVersion")
	// Output: ValidateVersion
}

func ExampleValidateVersionIdentifier() {
	_ = ValidateVersionIdentifier("v1.2.3")
	core.Println("ValidateVersionIdentifier")
	// Output: ValidateVersionIdentifier
}

func ExampleCompareVersions() {
	_ = CompareVersions("agent", "agent")
	core.Println("CompareVersions")
	// Output: CompareVersions
}
