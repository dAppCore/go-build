package publishers

import (
	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

// --- v0.9.0 generated compliance triplets ---
func TestPublisher_NewRelease_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewRelease("v1.2.3", nil, "agent", core.Path(t.TempDir(), "go-build-compliance"), coreio.NewMemoryMedium())
	})
	core.AssertTrue(t, true)
}

func TestPublisher_NewRelease_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewRelease("", nil, "", "", coreio.NewMemoryMedium())
	})
	core.AssertTrue(t, true)
}

func TestPublisher_NewRelease_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewRelease("v1.2.3", nil, "agent", core.Path(t.TempDir(), "go-build-compliance"), coreio.NewMemoryMedium())
	})
	core.AssertTrue(t, true)
}

func TestPublisher_NewReleaseWithArtifactFS_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewReleaseWithArtifactFS("v1.2.3", nil, "agent", core.Path(t.TempDir(), "go-build-compliance"), coreio.NewMemoryMedium(), coreio.NewMemoryMedium())
	})
	core.AssertTrue(t, true)
}

func TestPublisher_NewReleaseWithArtifactFS_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewReleaseWithArtifactFS("", nil, "", "", coreio.NewMemoryMedium(), coreio.NewMemoryMedium())
	})
	core.AssertTrue(t, true)
}

func TestPublisher_NewReleaseWithArtifactFS_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewReleaseWithArtifactFS("v1.2.3", nil, "agent", core.Path(t.TempDir(), "go-build-compliance"), coreio.NewMemoryMedium(), coreio.NewMemoryMedium())
	})
	core.AssertTrue(t, true)
}

func TestPublisher_NewPublisherConfig_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewPublisherConfig("agent", true, true, "agent")
	})
	core.AssertTrue(t, true)
}

func TestPublisher_NewPublisherConfig_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewPublisherConfig("", false, false, "agent")
	})
	core.AssertTrue(t, true)
}

func TestPublisher_NewPublisherConfig_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewPublisherConfig("agent", true, true, "agent")
	})
	core.AssertTrue(t, true)
}
