package buildtest

import (
	core "dappco.re/go"
)

func generatedWorkflowContentForCompliance() string {
	signInput := "sign:\n        description: Enable platform signing after build.\n        required: false\n        type: boolean\n        default: false"
	triggers := "workflow_call:\nworkflow_dispatch:"
	return core.Concat(triggers, "\n", core.Join("\n", releaseWorkflowExpectedMarkers...), "\n", signInput, "\n", signInput)
}

// --- v0.9.0 generated compliance triplets ---
func TestWorkflow_AssertReleaseWorkflowContent_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		AssertReleaseWorkflowContent(t, generatedWorkflowContentForCompliance())
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestWorkflow_AssertReleaseWorkflowContent_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		AssertReleaseWorkflowContent(t, generatedWorkflowContentForCompliance())
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestWorkflow_AssertReleaseWorkflowContent_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		AssertReleaseWorkflowContent(t, generatedWorkflowContentForCompliance())
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestWorkflow_AssertReleaseWorkflowTriggers_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		AssertReleaseWorkflowTriggers(t, generatedWorkflowContentForCompliance())
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestWorkflow_AssertReleaseWorkflowTriggers_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		AssertReleaseWorkflowTriggers(t, generatedWorkflowContentForCompliance())
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestWorkflow_AssertReleaseWorkflowTriggers_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		AssertReleaseWorkflowTriggers(t, generatedWorkflowContentForCompliance())
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
