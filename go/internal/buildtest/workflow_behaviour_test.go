package buildtest

import (
	core "dappco.re/go"
)

// Behaviour tests drive the internal counting logic directly. The Fatalf
// failure branches of the assert helpers are not unit-testable here: their
// parameter is testing.TB, a sealed interface that cannot be implemented by a
// recording stub outside the testing package, and Fatalf calls runtime.Goexit
// rather than panicking. The success contract is already covered by the
// generated triplets; this file closes the countWorkflowMarker gap.

func TestWorkflow_CountWorkflowMarker_Good(t *core.T) {
	content := "alpha beta alpha gamma alpha"
	core.AssertEqual(t, 3, countWorkflowMarker(content, "alpha"))
}

func TestWorkflow_CountWorkflowMarker_Bad(t *core.T) {
	// An empty marker is rejected up front and counts zero rather than
	// returning len(Split)-1, which would over-count on every rune boundary.
	core.AssertEqual(t, 0, countWorkflowMarker("anything", ""))
}

func TestWorkflow_CountWorkflowMarker_Ugly(t *core.T) {
	// A marker absent from the content counts zero; an exact single match
	// counts one; overlapping-but-non-splitting input counts by split boundary.
	core.AssertEqual(t, 0, countWorkflowMarker("workflow_call:", "workflow_dispatch:"))
	core.AssertEqual(t, 1, countWorkflowMarker("workflow_call:", "workflow_call:"))
	core.AssertEqual(t, 2, countWorkflowMarker("aaaa", "aa"))
}
