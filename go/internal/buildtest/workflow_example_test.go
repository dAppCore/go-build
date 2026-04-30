package buildtest

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated usage examples ---
func ExampleAssertReleaseWorkflowContent() {
	content := generatedWorkflowContentForCompliance()
	func() {
		defer func() { _ = recover() }()
		AssertReleaseWorkflowContent((*core.T)(nil), content)
	}()
	core.Println("AssertReleaseWorkflowContent")
	// Output: AssertReleaseWorkflowContent
}

func ExampleAssertReleaseWorkflowTriggers() {
	content := generatedWorkflowContentForCompliance()
	func() {
		defer func() { _ = recover() }()
		AssertReleaseWorkflowTriggers((*core.T)(nil), content)
	}()
	core.Println("AssertReleaseWorkflowTriggers")
	// Output: AssertReleaseWorkflowTriggers
}
