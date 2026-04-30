package publishers

import (
	core "dappco.re/go"
	coreio "dappco.re/go/build/pkg/storage"
)

// --- v0.9.0 generated compliance triplets ---
func TestPublisher_NewRelease_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewRelease("v1.2.3", nil, "agent", core.Path(t.TempDir(), "go-build-compliance"), coreio.NewMemoryMedium())
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestPublisher_NewRelease_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewRelease("", nil, "", "", coreio.NewMemoryMedium())
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestPublisher_NewRelease_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewRelease("v1.2.3", nil, "agent", core.Path(t.TempDir(), "go-build-compliance"), coreio.NewMemoryMedium())
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestPublisher_NewReleaseWithArtifactFS_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewReleaseWithArtifactFS("v1.2.3", nil, "agent", core.Path(t.TempDir(), "go-build-compliance"), coreio.NewMemoryMedium(), coreio.NewMemoryMedium())
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestPublisher_NewReleaseWithArtifactFS_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewReleaseWithArtifactFS("", nil, "", "", coreio.NewMemoryMedium(), coreio.NewMemoryMedium())
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestPublisher_NewReleaseWithArtifactFS_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewReleaseWithArtifactFS("v1.2.3", nil, "agent", core.Path(t.TempDir(), "go-build-compliance"), coreio.NewMemoryMedium(), coreio.NewMemoryMedium())
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestPublisher_NewPublisherConfig_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewPublisherConfig("agent", true, true, "agent")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestPublisher_NewPublisherConfig_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewPublisherConfig("", false, false, "agent")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestPublisher_NewPublisherConfig_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewPublisherConfig("agent", true, true, "agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
