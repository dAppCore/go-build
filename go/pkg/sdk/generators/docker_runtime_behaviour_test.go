package generators

import (
	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
)

// dockerRuntimeCommandState fingerprints an executable by stat + first-4KB
// hash. The existing suite drives the cache invalidation flows; these tests
// cover the stat-failure branch and the happy fingerprint on a real binary.

func TestSDK_DockerRuntimeCommandState_MissingFile_Bad(t *core.T) {
	state := dockerRuntimeCommandState(ax.Join(t.TempDir(), "no-such-binary"))
	core.AssertFalse(t, state.OK)
}

func TestSDK_DockerRuntimeCommandState_RealBinary_Good(t *core.T) {
	// /bin/echo always exists on the supported platforms; the fingerprint must
	// be a stable non-empty string composed of the command and its metadata.
	state := dockerRuntimeCommandState("/bin/echo")
	core.AssertTrue(t, state.OK, state.Error())
	fingerprint := state.Value.(string)
	core.AssertTrue(t, core.HasPrefix(fingerprint, "/bin/echo|"))

	// The fingerprint is deterministic for an unchanged file.
	again := dockerRuntimeCommandState("/bin/echo")
	core.AssertTrue(t, again.OK)
	core.AssertEqual(t, fingerprint, again.Value.(string))
}
