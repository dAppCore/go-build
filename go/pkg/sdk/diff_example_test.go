package sdk

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated usage examples ---
func ExampleDiff() {
	_ = Diff(core.Path(core.TempDir(), "go-build-compliance"), core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("Diff")
	// Output: Diff
}

func ExampleDiffWithOptions() {
	_ = DiffWithOptions(core.Path(core.TempDir(), "go-build-compliance"), core.Path(core.TempDir(), "go-build-compliance"), DiffOptions{})
	core.Println("DiffWithOptions")
	// Output: DiffWithOptions
}

func ExampleDiffExitCode() {
	_ = DiffExitCode(&DiffResult{}, nil)
	core.Println("DiffExitCode")
	// Output: DiffExitCode
}
