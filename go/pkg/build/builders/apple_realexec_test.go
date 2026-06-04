package builders

import (
	"context"
	"runtime"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
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

// fatBinaryArchs returns the architectures in a Mach-O file via real lipo, or an
// empty slice if the file is thin / lipo errors.
func fatBinaryArchs(ctx core.Context, path string) []string {
	out := ax.Run(ctx, "lipo", "-archs", path)
	if !out.OK {
		return nil
	}
	var archs []string
	for _, tok := range core.Split(core.Trim(out.Value.(string)), " ") {
		if core.Trim(tok) != "" {
			archs = append(archs, tok)
		}
	}
	return archs
}

// TestApple_CreateUniversal_RealLipo proves the default executing runner
// (GoProcessAppleRunner -> ax.ExecWithEnv) drives a genuine lipo merge through
// CreateUniversal, with NO recording runner injected.
//
// Approach: the PRIMARY plan (stage real thin slices from a universal system
// binary, then merge them via CreateUniversal). The plan suggested thinning to
// literal arm64/x86_64, but modern macOS system binaries ship x86_64 + arm64e
// (pointer-auth ABI), so we read the fixture's ACTUAL archs and extract the
// first two by their real names — keeping the test robust across Macs. Every
// failure mode (non-darwin, missing lipo, non-fat fixture, extraction failure)
// is a clean t.Skip so CI/linux and credential-free machines stay green.
func TestApple_CreateUniversal_RealLipo(t *core.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("requires darwin")
	}
	if !ax.ResolveCommand("lipo").OK {
		t.Skip("requires lipo")
	}
	ctx := context.Background()

	// Find a universal system binary with at least two slices to extract.
	var sysBin string
	var archs []string
	for _, candidate := range []string{"/bin/ls", "/usr/bin/true", "/bin/cp"} {
		if found := fatBinaryArchs(ctx, candidate); len(found) >= 2 {
			sysBin = candidate
			archs = found
			break
		}
	}
	if sysBin == "" {
		t.Skip("no universal (multi-arch) system binary available to stage from")
	}
	archA, archB := archs[0], archs[1]

	tmp := t.TempDir()
	armApp := ax.Join(tmp, "arm64.app")
	amdApp := ax.Join(tmp, "amd64.app")
	outApp := ax.Join(tmp, "Core.app")

	// Stage two real thin binaries, one per slice, at the bundle's executable path.
	for app, arch := range map[string]string{armApp: archA, amdApp: archB} {
		macosDir := ax.Join(app, "Contents", "MacOS")
		if !ax.MkdirAll(macosDir, 0o755).OK {
			t.Skip("could not create staging bundle dir " + macosDir)
		}
		thin := ax.Run(ctx, "lipo", sysBin, "-thin", arch, "-output", ax.Join(macosDir, "Core"))
		if !thin.OK {
			t.Skip("could not extract " + arch + " slice from " + sysBin + ": " + thin.Error())
		}
	}

	// Default executing runner — the whole point: no WithAppleCommandRunner here.
	b := NewAppleBuilder(WithAppleHostOS("darwin"))
	r := b.CreateUniversal(ctx, nil, coreio.Local, armApp, amdApp, outApp, "Core")
	core.AssertTrue(t, r.OK)

	mergedArchs := fatBinaryArchs(ctx, ax.Join(outApp, "Contents", "MacOS", "Core"))
	core.AssertTrue(t, containsArg(mergedArchs, archA)) // first slice survived the real lipo -create
	core.AssertTrue(t, containsArg(mergedArchs, archB)) // second slice survived too => genuinely universal
}
