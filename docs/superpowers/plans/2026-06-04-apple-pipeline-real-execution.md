# Apple Pipeline — Credential-Free Ops Real Execution — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Promote `builders.AppleBuilder`'s credential-free operations (`CreateDMG`, `CreateUniversal`, `BuildWailsMacOS`) from sandbox-safe skeleton to real execution on darwin, behind the existing runner seam, with hermetic command-construction TDD.

**Architecture:** `NewAppleBuilder` defaults its `runner` to the executing `GoProcessAppleRunner`; `runExternal` already gates execution to darwin. The three methods stop writing placeholder artifacts on darwin (the real `hdiutil`/`lipo`/`wails3` output is the result) while keeping the skeleton fallback on non-darwin. Tests inject a recording `AppleCommandRunner` via `WithAppleCommandRunner` + `WithAppleHostOS("darwin")` and assert the exact commands. Sign/Notarise/TestFlight/AppStore stay skeleton.

**Tech Stack:** Go 1.26, `dappco.re/go` (CoreGO v0.10.3), workspace mode (NO `GOWORK=off`), `core.AssertX`/`t *core.T` test idiom, AX-7 triplets.

**Spec:** `docs/superpowers/specs/2026-06-04-apple-pipeline-real-execution-design.md`

---

## Conventions (apply to every task)

- Work from `/Users/snider/Code/core/go-build/go`. Workspace mode only. NEVER `GOWORK=off`.
- Test files: `package builders`, import `core "dappco.re/go"`, `t *core.T`, assertions `core.AssertEqual/AssertTrue/AssertFalse/AssertNil/AssertNotNil`. Do NOT import `"testing"` unless using `*testing.T`. No `AssertNotPanics`/counter theatre, no tautologies, distinct Good/Bad/Ugly bodies.
- The recording runner used across tests (define once in `apple_realexec_test.go`, reuse):

```go
// recordingAppleRunner captures every RunOptions it receives and returns a
// scripted result, letting tests assert command construction without shelling
// real tools.
type recordingAppleRunner struct {
	calls  []RunOptions
	result core.Result
}

func (r *recordingAppleRunner) Run(_ core.Context, opts RunOptions) core.Result {
	r.calls = append(r.calls, opts)
	if r.result.OK || r.result.Value != nil {
		return r.result
	}
	return core.Ok(nil)
}

func newRecordingAppleRunner() *recordingAppleRunner {
	return &recordingAppleRunner{result: core.Ok(nil)}
}
```

- Build a darwin builder under test with: `NewAppleBuilder(WithAppleHostOS("darwin"), WithAppleCommandRunner(rec))`.

---

## File Structure

| File | Responsibility | Action |
|------|----------------|--------|
| `pkg/build/builders/apple.go` | `NewAppleBuilder` runner default; `BuildWailsMacOS` skeleton guard + `OUTPUT_DIR`; `CreateUniversal` unchanged logic (executes via runner) | Modify |
| `pkg/build/builders/apple_dmg.go` | `CreateDMG` placeholder guard | Modify |
| `pkg/build/builders/apple_realexec_test.go` | Recording-runner triplets for the three methods + placeholder-policy tests + shared `recordingAppleRunner` | Create |
| `pkg/build/builders/apple_test.go` | Harden any bare-builder darwin construction against the new executing default | Modify (audit) |

---

## Task 1: Default the runner to executing (and protect existing tests)

**Files:**
- Modify: `pkg/build/builders/apple.go` (`NewAppleBuilder`, ~line 132)
- Modify: `pkg/build/builders/apple_test.go` (audit bare constructions)
- Test: `pkg/build/builders/apple_realexec_test.go` (create)

- [ ] **Step 1: Audit existing apple tests for the blast radius.**

The new default means an `AppleBuilder` built without a runner will, on a darwin host (this Mac), execute real tools. Find unprotected constructions:

Run: `grep -n 'NewAppleBuilder(' pkg/build/builders/*_test.go`
For each match, confirm it ALSO passes `WithAppleHostOS("linux")` (or another non-darwin) OR `WithAppleCommandRunner(...)`. List any that pass neither — those are the ones to fix in Step 6.

- [ ] **Step 2: Write the failing test (runner executes on darwin, records on non-darwin).**

Create `pkg/build/builders/apple_realexec_test.go`:

```go
// SPDX-License-Identifier: EUPL-1.2

package builders

import (
	"context"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	coreio "dappco.re/go/build/pkg/storage"
)

type recordingAppleRunner struct {
	calls  []RunOptions
	result core.Result
}

func (r *recordingAppleRunner) Run(_ core.Context, opts RunOptions) core.Result {
	r.calls = append(r.calls, opts)
	if !r.result.OK {
		return r.result
	}
	return core.Ok(nil)
}

func newRecordingAppleRunner() *recordingAppleRunner {
	return &recordingAppleRunner{result: core.Ok(nil)}
}

func TestApple_NewAppleBuilder_DefaultRunnerExecutesOnDarwin_Good(t *core.T) {
	rec := newRecordingAppleRunner()
	b := NewAppleBuilder(WithAppleHostOS("darwin"), WithAppleCommandRunner(rec))
	r := b.CreateUniversal(context.Background(), nil, coreio.NewMemory(), "/a/arm64.app", "/a/amd64.app", "/a/out.app", "App")
	core.AssertTrue(t, r.OK)
	core.AssertFalse(t, len(rec.calls) == 0) // lipo was actually dispatched to the runner
}

func TestApple_NewAppleBuilder_DefaultRunnerSkipsExecutionOffDarwin_Bad(t *core.T) {
	rec := newRecordingAppleRunner()
	b := NewAppleBuilder(WithAppleHostOS("linux"), WithAppleCommandRunner(rec))
	r := b.CreateUniversal(context.Background(), nil, coreio.NewMemory(), "/a/arm64.app", "/a/amd64.app", "/a/out.app", "App")
	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, 0, len(rec.calls)) // non-darwin records-only, runner not invoked
}

func TestApple_NewAppleBuilder_DefaultRunnerNonNil_Ugly(t *core.T) {
	// A default builder carries an executing runner (not nil) so production
	// darwin runs real tools without explicit wiring.
	b := NewAppleBuilder(WithAppleHostOS("linux")) // linux => safe, no execution
	core.AssertNotNil(t, b.runner)
}
```

> Note: `coreio.NewMemory()` is the in-memory `storage.Medium` — confirm its constructor name with `grep -rn 'func NewMemory\|func Memory(' pkg/build/storage/ pkg/storage/` and adjust. If the arm64 path must exist for `CopyMediumPath`, seed it: `fs := coreio.NewMemory(); fs.EnsureDir("/a/arm64.app")` before the call.

- [ ] **Step 3: Run the test, verify it fails.**

Run: `go test ./pkg/build/builders/ -run 'TestApple_NewAppleBuilder_DefaultRunner' -count=1 -v`
Expected: FAIL — `Ugly` fails (`b.runner` is nil today); `Good` fails (no calls recorded because default runner is nil).

- [ ] **Step 4: Implement the runner default.**

In `pkg/build/builders/apple.go`, `NewAppleBuilder` (~line 133-137), add the default runner:

```go
func NewAppleBuilder(options ...AppleBuilderOption) *AppleBuilder {
	builder := &AppleBuilder{
		Options:    DefaultAppleBuilderOptions(),
		hostOS:     runtime.GOOS,
		todoWriter: core.Stdout(),
		runner:     GoProcessAppleRunner{},
	}
	for _, option := range options {
		if option != nil {
			option(builder)
		}
	}
	return builder
}
```

- [ ] **Step 5: Run the test, verify it passes.**

Run: `go test ./pkg/build/builders/ -run 'TestApple_NewAppleBuilder_DefaultRunner' -count=1 -v`
Expected: PASS.

- [ ] **Step 6: Protect any unprotected existing tests found in Step 1.**

For each bare `NewAppleBuilder(...)` from Step 1 that passes neither a runner nor a non-darwin host, add `WithAppleHostOS("linux")` (if the test is asserting skeleton/recording behaviour) or `WithAppleCommandRunner(newRecordingAppleRunner())` (if it should record). Then:

Run: `go test ./pkg/build/builders/ -count=1`
Expected: PASS (no test now shells a real tool on this darwin Mac).

- [ ] **Step 7: Commit.**

```bash
git add pkg/build/builders/apple.go pkg/build/builders/apple_realexec_test.go pkg/build/builders/apple_test.go
git commit -m "feat(apple): default AppleBuilder to executing runner on darwin"
```

---

## Task 2: CreateDMG — guard the placeholder behind non-darwin

**Files:**
- Modify: `pkg/build/builders/apple_dmg.go` (placeholder write, lines 97-106)
- Test: `pkg/build/builders/apple_realexec_test.go`

- [ ] **Step 1: Write the failing tests (command sequence + placeholder policy).**

Append to `apple_realexec_test.go`:

```go
func TestAppleDMG_CreateDMG_ConstructsHdiutilSequence_Good(t *core.T) {
	rec := newRecordingAppleRunner()
	b := NewAppleBuilder(WithAppleHostOS("darwin"), WithAppleCommandRunner(rec))
	fs := coreio.NewMemory()
	r := b.CreateDMG(context.Background(), fs, "/build/App.app", AppleDMGConfig{OutputPath: "/dist/App.dmg", VolumeName: "App"})
	core.AssertTrue(t, r.OK)
	// Four hdiutil invocations in order: create, attach, detach, convert.
	core.AssertEqual(t, 4, len(rec.calls))
	core.AssertEqual(t, "hdiutil", rec.calls[0].Command)
	core.AssertEqual(t, "create", rec.calls[0].Args[0])
	core.AssertEqual(t, "convert", rec.calls[3].Args[0])
	core.AssertTrue(t, containsArg(rec.calls[3].Args, "/dist/App.dmg"))
}

func TestAppleDMG_CreateDMG_NoPlaceholderOnDarwin_Ugly(t *core.T) {
	rec := newRecordingAppleRunner()
	b := NewAppleBuilder(WithAppleHostOS("darwin"), WithAppleCommandRunner(rec))
	fs := coreio.NewMemory()
	_ = b.CreateDMG(context.Background(), fs, "/build/App.app", AppleDMGConfig{OutputPath: "/dist/App.dmg", VolumeName: "App"})
	// On darwin the real hdiutil output is the artifact — the skeleton must NOT
	// write a placeholder text file over it.
	read := fs.Read("/dist/App.dmg")
	if read.OK {
		core.AssertFalse(t, core.Contains(read.Value.(string), "AppleBuilder DMG skeleton"))
	}
}

func TestAppleDMG_CreateDMG_WritesPlaceholderOffDarwin_Bad(t *core.T) {
	b := NewAppleBuilder(WithAppleHostOS("linux"))
	fs := coreio.NewMemory()
	r := b.CreateDMG(context.Background(), fs, "/build/App.app", AppleDMGConfig{OutputPath: "/dist/App.dmg", VolumeName: "App"})
	core.AssertTrue(t, r.OK)
	read := fs.Read("/dist/App.dmg")
	core.AssertTrue(t, read.OK)
	core.AssertTrue(t, core.Contains(read.Value.(string), "AppleBuilder DMG skeleton"))
}

// containsArg reports whether args contains want.
func containsArg(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}
```

> Confirm `core.Contains` exists (`grep -rn 'func Contains' external/go/`); if it is `core.StringContains` or similar, adjust. Confirm `coreio.Memory` `Read` returns the content in `Value.(string)`.

- [ ] **Step 2: Run the tests, verify they fail.**

Run: `go test ./pkg/build/builders/ -run 'TestAppleDMG_CreateDMG' -count=1 -v`
Expected: FAIL — `NoPlaceholderOnDarwin_Ugly` fails (placeholder is written unconditionally today).

- [ ] **Step 3: Guard the placeholder write.**

In `pkg/build/builders/apple_dmg.go`, replace the unconditional placeholder block (lines 97-106) with a non-darwin guard:

```go
	// On non-darwin hosts hdiutil did not execute; write a skeleton marker so
	// downstream lanes still receive a file. On darwin the real hdiutil convert
	// output above is the artifact and must not be overwritten.
	if firstNonEmptyApple(b.hostOS, runtime.GOOS) != "darwin" {
		placeholder := core.Sprintf(
			"AppleBuilder DMG skeleton\napp=%s\nvolume=%s\nbackground=%s\n",
			appPath,
			cfg.VolumeName,
			cfg.BackgroundPath,
		)
		written := filesystem.WriteMode(cfg.OutputPath, placeholder, 0o644)
		if !written.OK {
			return core.Fail(core.E("AppleBuilder.CreateDMG", "failed to write placeholder DMG", written))
		}
	}

	return core.Ok(nil)
```

Add `"runtime"` to the `apple_dmg.go` import block if not present (`firstNonEmptyApple` lives in `apple.go`, same package). Note the `core.NewError(written.Error())` → `written` change applies the Result-propagation idiom.

- [ ] **Step 4: Run the tests, verify they pass.**

Run: `go test ./pkg/build/builders/ -run 'TestAppleDMG_CreateDMG' -count=1 -v`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
git add pkg/build/builders/apple_dmg.go pkg/build/builders/apple_realexec_test.go
git commit -m "feat(apple): CreateDMG runs hdiutil for real on darwin (placeholder only off-darwin)"
```

---

## Task 3: CreateUniversal — lock the lipo command construction

`CreateUniversal` already copies the arm64 bundle then runs `lipo` via the runner — with Task 1's executing default it now merges for real on darwin. This task locks the command construction with tests (no production change expected; if a test reveals a defect, fix minimally).

**Files:**
- Test: `pkg/build/builders/apple_realexec_test.go`
- Modify (only if a test fails): `pkg/build/builders/apple.go` (`CreateUniversal`, ~line 361)

- [ ] **Step 1: Write the failing/locking test.**

```go
func TestApple_CreateUniversal_ConstructsLipoCreate_Good(t *core.T) {
	rec := newRecordingAppleRunner()
	b := NewAppleBuilder(WithAppleHostOS("darwin"), WithAppleCommandRunner(rec))
	fs := coreio.NewMemory()
	fs.EnsureDir("/a/arm64.app")
	r := b.CreateUniversal(context.Background(), nil, fs, "/a/arm64.app", "/a/amd64.app", "/a/Core.app", "Core")
	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, 1, len(rec.calls))
	core.AssertEqual(t, "lipo", rec.calls[0].Command)
	core.AssertEqual(t, "-create", rec.calls[0].Args[0])
	core.AssertTrue(t, containsArg(rec.calls[0].Args, "/a/Core.app/Contents/MacOS/Core"))     // -output target
	core.AssertTrue(t, containsArg(rec.calls[0].Args, "/a/arm64.app/Contents/MacOS/Core"))
	core.AssertTrue(t, containsArg(rec.calls[0].Args, "/a/amd64.app/Contents/MacOS/Core"))
}

func TestApple_CreateUniversal_RunnerFailureBubbles_Bad(t *core.T) {
	rec := newRecordingAppleRunner()
	rec.result = core.Fail(core.E("test", "lipo boom", nil))
	b := NewAppleBuilder(WithAppleHostOS("darwin"), WithAppleCommandRunner(rec))
	fs := coreio.NewMemory()
	fs.EnsureDir("/a/arm64.app")
	r := b.CreateUniversal(context.Background(), nil, fs, "/a/arm64.app", "/a/amd64.app", "/a/Core.app", "Core")
	core.AssertFalse(t, r.OK)
}

func TestApple_CreateUniversal_RecordsOnlyOffDarwin_Ugly(t *core.T) {
	rec := newRecordingAppleRunner()
	b := NewAppleBuilder(WithAppleHostOS("linux"), WithAppleCommandRunner(rec))
	fs := coreio.NewMemory()
	fs.EnsureDir("/a/arm64.app")
	r := b.CreateUniversal(context.Background(), nil, fs, "/a/arm64.app", "/a/amd64.app", "/a/Core.app", "Core")
	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, 0, len(rec.calls))
}
```

- [ ] **Step 2: Run the tests.**

Run: `go test ./pkg/build/builders/ -run 'TestApple_CreateUniversal' -count=1 -v`
Expected: PASS (logic already correct). If `Good` fails on the `-output` ordering, inspect `CreateUniversal` (apple.go:382-385) and align the assertion to the actual arg order — do NOT change production unless the order is genuinely wrong.

- [ ] **Step 3: Commit.**

```bash
git add pkg/build/builders/apple_realexec_test.go
git commit -m "test(apple): lock CreateUniversal lipo command construction"
```

---

## Task 4: BuildWailsMacOS — real wails3 output dir + skeleton guard

On non-darwin the skeleton `.app` is correct. On darwin, `wails3` must be told where to output (it uses the `OUTPUT_DIR` env, per `wails.go:163`), and the skeleton `.app` write must be skipped so the real bundle is the result.

**Files:**
- Modify: `pkg/build/builders/apple.go` (`BuildWailsMacOS`, lines 323-358)
- Test: `pkg/build/builders/apple_realexec_test.go`

- [ ] **Step 1: Write the failing tests.**

```go
func TestApple_BuildWailsMacOS_ConstructsWails3Build_Good(t *core.T) {
	rec := newRecordingAppleRunner()
	b := NewAppleBuilder(WithAppleHostOS("darwin"), WithAppleCommandRunner(rec))
	fs := coreio.NewMemory()
	cfg := &build.Config{ProjectDir: "/proj", BuildTags: []string{"mlx"}}
	r := b.BuildWailsMacOS(context.Background(), fs, cfg, "/proj/dist/apple", "Core", "arm64")
	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, 1, len(rec.calls))
	core.AssertEqual(t, "wails3", rec.calls[0].Command)
	core.AssertEqual(t, "build", rec.calls[0].Args[0])
	core.AssertTrue(t, containsArg(rec.calls[0].Args, "darwin/arm64"))
	core.AssertTrue(t, containsArg(rec.calls[0].Args, "mlx"))
	// wails3 v3 is told the output dir via OUTPUT_DIR env (see wails.go).
	core.AssertTrue(t, envContains(rec.calls[0].Env, "OUTPUT_DIR=/proj/dist/apple"))
}

func TestApple_BuildWailsMacOS_NoSkeletonOnDarwin_Ugly(t *core.T) {
	rec := newRecordingAppleRunner()
	b := NewAppleBuilder(WithAppleHostOS("darwin"), WithAppleCommandRunner(rec))
	fs := coreio.NewMemory()
	cfg := &build.Config{ProjectDir: "/proj"}
	_ = b.BuildWailsMacOS(context.Background(), fs, cfg, "/proj/dist/apple", "Core", "arm64")
	// No placeholder bundle written on darwin (real wails3 output is the result).
	core.AssertFalse(t, fs.Exists("/proj/dist/apple/Core.app/Contents/Info.plist"))
}

func TestApple_BuildWailsMacOS_WritesSkeletonOffDarwin_Bad(t *core.T) {
	b := NewAppleBuilder(WithAppleHostOS("linux"))
	fs := coreio.NewMemory()
	cfg := &build.Config{ProjectDir: "/proj"}
	r := b.BuildWailsMacOS(context.Background(), fs, cfg, "/proj/dist/apple", "Core", "arm64")
	core.AssertTrue(t, r.OK)
	core.AssertTrue(t, fs.Exists("/proj/dist/apple/Core.app")) // skeleton bundle present off-darwin
}

func envContains(env []string, want string) bool {
	for _, e := range env {
		if e == want {
			return true
		}
	}
	return false
}
```

> Confirm `createAppleBundleSkeleton` writes `Contents/Info.plist` (read `apple.go:612`); if it writes a different marker path, adjust the `NoSkeletonOnDarwin` assertion to that path. Confirm `build.BuildEnvironment` is variadic-string (`apple.go:346`) so `OUTPUT_DIR=...` can be appended.

- [ ] **Step 2: Run the tests, verify they fail.**

Run: `go test ./pkg/build/builders/ -run 'TestApple_BuildWailsMacOS' -count=1 -v`
Expected: FAIL — `OUTPUT_DIR` not in env; skeleton written on darwin.

- [ ] **Step 3: Implement OUTPUT_DIR env + skeleton guard.**

In `pkg/build/builders/apple.go`, modify `BuildWailsMacOS` (lines 342-357):

```go
	// TODO(#484 resolved for credential-free build): wails3 v3 takes the output
	// location via OUTPUT_DIR (see WailsBuilder.buildV3Target). On darwin the
	// runner executes the real build; off-darwin runExternal records only.
	ran := b.runExternal(ctx, "wails-build", RunOptions{
		Command: "wails3",
		Args:    args,
		Dir:     cfg.ProjectDir,
		Env:     build.BuildEnvironment(cfg, "GOOS=darwin", "GOARCH="+arch, "CGO_ENABLED=1", "OUTPUT_DIR="+outputDir),
	})
	if !ran.OK {
		return ran
	}

	bundlePath := ax.Join(outputDir, name+".app")
	// On non-darwin the real wails3 build did not run; write a skeleton bundle so
	// downstream lanes have a path. On darwin the real .app produced by wails3 is
	// the artifact.
	if firstNonEmptyApple(b.hostOS, runtime.GOOS) != "darwin" {
		createdBundle := createAppleBundleSkeleton(filesystem, bundlePath, name, arch)
		if !createdBundle.OK {
			return createdBundle
		}
	}
	return core.Ok(bundlePath)
```

- [ ] **Step 4: Run the tests, verify they pass.**

Run: `go test ./pkg/build/builders/ -run 'TestApple_BuildWailsMacOS' -count=1 -v`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
git add pkg/build/builders/apple.go pkg/build/builders/apple_realexec_test.go
git commit -m "feat(apple): BuildWailsMacOS passes OUTPUT_DIR + skips skeleton on darwin"
```

> Real-Mac verification item (out of hermetic scope): on a darwin host with `wails3` installed, confirm the produced `.app` lands at `outputDir/Core.app` (wails3 may nest under a platform subdir; if so, a follow-up adds bundle resolution). Covered by the skip-if-absent test in Task 5.

---

## Task 5: Optional darwin real-tool test (skip-if-absent)

Confidence that `GoProcessAppleRunner → ax.ExecWithEnv` actually drives a real tool. Uses `lipo` (present on any macOS with Xcode CLT); skips elsewhere.

**Files:**
- Test: `pkg/build/builders/apple_realexec_test.go`

- [ ] **Step 1: Write the skip-if-absent real-tool test.**

```go
func TestApple_CreateUniversal_RealLipo_Good(t *core.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("real lipo test requires darwin")
	}
	if !ax.ResolveCommand("lipo").OK {
		t.Skip("lipo not installed")
	}
	// Default (executing) runner — no recording override.
	b := NewAppleBuilder(WithAppleHostOS("darwin"))
	dir := t.TempDir()
	// Build two trivial thin Mach-O binaries via clang if available; otherwise skip.
	if !ax.ResolveCommand("clang").OK {
		t.Skip("clang not installed")
	}
	// ... construct minimal arm64/amd64 .app dirs with thin binaries (clang -arch),
	// then call b.CreateUniversal and assert ax.ResolveCommand("file") reports a
	// universal binary at the output. (Full fixture in the test.)
	_ = dir
}
```

> This test imports `"runtime"`. Keep the fixture construction inside the test; it must `t.Skip` cleanly when `clang`/`lipo` are absent so CI (linux) and credential-free machines stay green. If building thin binaries proves flaky, downgrade this to asserting `lipo` is invoked against pre-staged fixture files and that the call returns OK — still real execution, no synthetic Mach-O needed.

- [ ] **Step 2: Run it.**

Run: `go test ./pkg/build/builders/ -run 'TestApple_CreateUniversal_RealLipo' -count=1 -v`
Expected: PASS or SKIP (skips if not darwin / tools absent).

- [ ] **Step 3: Commit.**

```bash
git add pkg/build/builders/apple_realexec_test.go
git commit -m "test(apple): real-lipo execution smoke (skip-if-absent)"
```

---

## Task 6: Full verification + audit + facade trace

**Files:** none (verification)

- [ ] **Step 1: Whole-package + dependent build/test green.**

Run: `gofmt -l pkg/build/builders/ && go vet ./pkg/build/builders/ && go test ./pkg/build/builders/ ./pkg/build/apple/ -count=1`
Expected: gofmt empty; vet clean; tests PASS.

- [ ] **Step 2: Confirm the facade still works (it delegates via fn-vars).**

Run: `go test ./pkg/build/apple/ -count=1 -v` and `grep -rn 'createDMGFn\|buildWailsAppFn\|createUniversalFn' pkg/build/apple/`
Expected: the facade tests pass; confirm the fn-vars route into the `builders.AppleBuilder` methods (or `build.*` wrappers thereof). No facade edit expected.

- [ ] **Step 3: Whole-module green (the push gate).**

Run: `go build ./... && go test ./... -count=1 -short`
Expected: all packages `ok`.

- [ ] **Step 4: Re-run the v0.9.0 audit; no new theatre.**

Run: `bash /Users/snider/Code/core/go/tests/cli/v090-upgrade/audit.sh . 2>/dev/null | sed 's/\x1b\[[0-9;]*m//g' | grep -E 'tautological|identical-triplets|test-stubs|unreferenced|ax7-triplet'`
Expected: no increase attributable to the new apple tests (distinct triplet bodies, no `AssertNotPanics`, no tautologies).

- [ ] **Step 5: Final summary commit (if any verification fixups were needed).**

```bash
git add -A pkg/build/
git commit -m "test(apple): verification fixups for credential-free real execution"
```

---

## Self-review notes

- **Spec coverage:** runner default (Task 1) ✓; placeholder guard CreateDMG (Task 2) ✓; CreateUniversal real exec (Tasks 1+3) ✓; BuildWailsMacOS real output + guard (Task 4) ✓; recording-runner command-construction triplets (Tasks 2-4) ✓; placeholder-policy tests (Tasks 2,4) ✓; optional darwin real-tool test (Task 5) ✓; out-of-scope Sign/Notarise/TestFlight/AppStore untouched ✓.
- **Open verification (flagged, not faked):** `coreio.NewMemory` constructor name; `core.Contains` name; `createAppleBundleSkeleton` marker path; `build.BuildEnvironment` variadic shape; wails3 `.app` final location on a real Mac. Each step names the grep to confirm and the adjustment if the name differs — these are confirm-and-adjust, not placeholders.
