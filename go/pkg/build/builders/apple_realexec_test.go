package builders

import (
	"context"

	core "dappco.re/go"
	coreio "dappco.re/go/build/pkg/storage"
)

// recordingAppleRunner captures every RunOptions dispatched to it and returns a
// configurable result. The zero value records but reports failure; prefer
// newRecordingAppleRunner so callers default to a successful runner.
type recordingAppleRunner struct {
	calls  []RunOptions
	result core.Result
}

// Run implements AppleCommandRunner: it records opts, then returns the
// configured result (Ok when none was set to fail).
func (r *recordingAppleRunner) Run(_ core.Context, opts RunOptions) core.Result {
	r.calls = append(r.calls, opts)
	if !r.result.OK {
		return r.result
	}
	return core.Ok(nil)
}

// newRecordingAppleRunner returns a recorder that reports success by default.
func newRecordingAppleRunner() *recordingAppleRunner {
	return &recordingAppleRunner{result: core.Ok(nil)}
}

var _ AppleCommandRunner = (*recordingAppleRunner)(nil)

func TestApple_NewAppleBuilder_DefaultRunnerExecutesOnDarwin(t *core.T) {
	rec := newRecordingAppleRunner()
	b := NewAppleBuilder(WithAppleHostOS("darwin"), WithAppleCommandRunner(rec))
	fs := coreio.NewMemoryMedium()
	fs.EnsureDir("/a/arm64.app")
	r := b.CreateUniversal(context.Background(), nil, fs, "/a/arm64.app", "/a/amd64.app", "/a/out.app", "App")
	core.AssertTrue(t, r.OK)
	core.AssertLen(t, rec.calls, 1) // exactly one lipo call dispatched to the runner
}

func TestApple_NewAppleBuilder_DefaultRunnerRecordsOnlyOffDarwin(t *core.T) {
	rec := newRecordingAppleRunner()
	b := NewAppleBuilder(WithAppleHostOS("linux"), WithAppleCommandRunner(rec))
	fs := coreio.NewMemoryMedium()
	fs.EnsureDir("/a/arm64.app")
	r := b.CreateUniversal(context.Background(), nil, fs, "/a/arm64.app", "/a/amd64.app", "/a/out.app", "App")
	core.AssertTrue(t, r.OK)               // off-darwin succeeds by design
	core.AssertEqual(t, 0, len(rec.calls)) // ...but records only, no dispatch
}

func TestApple_NewAppleBuilder_DefaultRunnerNonNil(t *core.T) {
	b := NewAppleBuilder(WithAppleHostOS("linux")) // linux => safe, no execution
	core.AssertNotNil(t, b.runner)
}
