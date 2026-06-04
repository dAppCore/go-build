package builders

import (
	"context"

	core "dappco.re/go"
	"dappco.re/go/build/pkg/build"
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

// containsArg reports whether want appears among args.
func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

func TestApple_CreateDMG_ConstructsHdiutilSequence(t *core.T) {
	rec := newRecordingAppleRunner()
	b := NewAppleBuilder(WithAppleHostOS("darwin"), WithAppleCommandRunner(rec))
	fs := coreio.NewMemoryMedium()
	fs.EnsureDir("/a/Core.app")
	r := b.CreateDMG(context.Background(), fs, "/a/Core.app", AppleDMGConfig{OutputPath: "/dist/Core.dmg", VolumeName: "Core"})
	core.AssertTrue(t, r.OK)
	core.AssertLen(t, rec.calls, 4)
	core.AssertEqual(t, "hdiutil", rec.calls[0].Command)
	core.AssertEqual(t, "create", rec.calls[0].Args[0])
	core.AssertEqual(t, "attach", rec.calls[1].Args[0])
	core.AssertEqual(t, "detach", rec.calls[2].Args[0])
	core.AssertEqual(t, "convert", rec.calls[3].Args[0])
	core.AssertTrue(t, containsArg(rec.calls[3].Args, "/dist/Core.dmg")) // convert targets the real output
}

func TestApple_CreateDMG_NoPlaceholderOnDarwin(t *core.T) {
	rec := newRecordingAppleRunner()
	b := NewAppleBuilder(WithAppleHostOS("darwin"), WithAppleCommandRunner(rec))
	fs := coreio.NewMemoryMedium()
	fs.EnsureDir("/a/Core.app")
	r := b.CreateDMG(context.Background(), fs, "/a/Core.app", AppleDMGConfig{OutputPath: "/dist/Core.dmg", VolumeName: "Core"})
	core.AssertTrue(t, r.OK)
	read := fs.Read("/dist/Core.dmg")
	if read.OK {
		// hdiutil convert is the artifact on darwin; a skeleton marker would mean we clobbered it.
		core.AssertFalse(t, core.Contains(read.Value.(string), "AppleBuilder DMG skeleton"))
	}
}

func TestApple_CreateDMG_WritesPlaceholderOffDarwin(t *core.T) {
	b := NewAppleBuilder(WithAppleHostOS("linux"))
	fs := coreio.NewMemoryMedium()
	fs.EnsureDir("/a/Core.app")
	r := b.CreateDMG(context.Background(), fs, "/a/Core.app", AppleDMGConfig{OutputPath: "/dist/Core.dmg", VolumeName: "Core"})
	core.AssertTrue(t, r.OK)
	read := fs.Read("/dist/Core.dmg")
	core.AssertTrue(t, read.OK) // off-darwin hdiutil never ran, so the skeleton stands in
	core.AssertTrue(t, core.Contains(read.Value.(string), "AppleBuilder DMG skeleton"))
}

func TestApple_CreateUniversal_ConstructsLipoCreate(t *core.T) {
	rec := newRecordingAppleRunner()
	b := NewAppleBuilder(WithAppleHostOS("darwin"), WithAppleCommandRunner(rec))
	fs := coreio.NewMemoryMedium()
	fs.EnsureDir("/a/arm64.app")
	r := b.CreateUniversal(context.Background(), nil, fs, "/a/arm64.app", "/a/amd64.app", "/a/Core.app", "Core")
	core.AssertTrue(t, r.OK)
	core.AssertLen(t, rec.calls, 1)
	core.AssertEqual(t, "lipo", rec.calls[0].Command)
	core.AssertEqual(t, "-create", rec.calls[0].Args[0])
	core.AssertTrue(t, containsArg(rec.calls[0].Args, "/a/Core.app/Contents/MacOS/Core"))  // -output target
	core.AssertTrue(t, containsArg(rec.calls[0].Args, "/a/arm64.app/Contents/MacOS/Core")) // arm64 slice
	core.AssertTrue(t, containsArg(rec.calls[0].Args, "/a/amd64.app/Contents/MacOS/Core")) // amd64 slice
}

func TestApple_CreateUniversal_RunnerFailureBubbles(t *core.T) {
	rec := newRecordingAppleRunner()
	rec.result = core.Fail(core.E("test", "lipo boom", nil))
	b := NewAppleBuilder(WithAppleHostOS("darwin"), WithAppleCommandRunner(rec))
	fs := coreio.NewMemoryMedium()
	fs.EnsureDir("/a/arm64.app")
	r := b.CreateUniversal(context.Background(), nil, fs, "/a/arm64.app", "/a/amd64.app", "/a/Core.app", "Core")
	core.AssertFalse(t, r.OK) // a failing lipo run must surface, not be swallowed
}

func TestApple_CreateUniversal_RecordsOnlyOffDarwin(t *core.T) {
	rec := newRecordingAppleRunner()
	b := NewAppleBuilder(WithAppleHostOS("linux"), WithAppleCommandRunner(rec))
	fs := coreio.NewMemoryMedium()
	fs.EnsureDir("/a/arm64.app")
	r := b.CreateUniversal(context.Background(), nil, fs, "/a/arm64.app", "/a/amd64.app", "/a/Core.app", "Core")
	core.AssertTrue(t, r.OK)               // off-darwin copies arm64 + records, succeeds by design
	core.AssertEqual(t, 0, len(rec.calls)) // ...but lipo is never dispatched
}

// envContains reports whether want appears among env entries.
func envContains(env []string, want string) bool {
	for _, entry := range env {
		if entry == want {
			return true
		}
	}
	return false
}

func TestApple_BuildWailsMacOS_ConstructsWails3Build(t *core.T) {
	rec := newRecordingAppleRunner()
	b := NewAppleBuilder(WithAppleHostOS("darwin"), WithAppleCommandRunner(rec))
	fs := coreio.NewMemoryMedium()
	cfg := &build.Config{ProjectDir: "/proj", BuildTags: []string{"mlx"}}
	r := b.BuildWailsMacOS(context.Background(), fs, cfg, "/proj/dist/apple", "Core", "arm64")
	core.AssertTrue(t, r.OK)
	core.AssertLen(t, rec.calls, 1)
	core.AssertEqual(t, "wails3", rec.calls[0].Command)
	core.AssertEqual(t, "build", rec.calls[0].Args[0])
	core.AssertTrue(t, containsArg(rec.calls[0].Args, "darwin/arm64"))               // -platform target
	core.AssertTrue(t, containsArg(rec.calls[0].Args, "mlx"))                        // build tag forwarded
	core.AssertTrue(t, envContains(rec.calls[0].Env, "OUTPUT_DIR=/proj/dist/apple")) // wails3 v3 output dir
}

func TestApple_BuildWailsMacOS_NoSkeletonOnDarwin(t *core.T) {
	rec := newRecordingAppleRunner()
	b := NewAppleBuilder(WithAppleHostOS("darwin"), WithAppleCommandRunner(rec))
	fs := coreio.NewMemoryMedium()
	cfg := &build.Config{ProjectDir: "/proj"}
	r := b.BuildWailsMacOS(context.Background(), fs, cfg, "/proj/dist/apple", "Core", "arm64")
	core.AssertTrue(t, r.OK)
	// On darwin the real wails3 build is the artifact; the placeholder executable
	// must not be written or it would shadow the genuine binary.
	core.AssertFalse(t, fs.Exists("/proj/dist/apple/Core.app/Contents/MacOS/Core"))
}

func TestApple_BuildWailsMacOS_WritesSkeletonOffDarwin(t *core.T) {
	b := NewAppleBuilder(WithAppleHostOS("linux"))
	fs := coreio.NewMemoryMedium()
	cfg := &build.Config{ProjectDir: "/proj"}
	r := b.BuildWailsMacOS(context.Background(), fs, cfg, "/proj/dist/apple", "Core", "arm64")
	core.AssertTrue(t, r.OK)
	// Off-darwin wails3 never ran, so the skeleton bundle stands in for downstream lanes.
	core.AssertTrue(t, fs.Exists("/proj/dist/apple/Core.app/Contents/MacOS/Core"))
}
