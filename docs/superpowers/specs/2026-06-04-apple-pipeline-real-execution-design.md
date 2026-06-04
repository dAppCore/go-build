# Apple build pipeline — credential-free ops: skeleton → real execution

- **Date:** 2026-06-04
- **Status:** Approved (design) — pending implementation
- **Module:** `dappco.re/go/build`
- **Scope owner:** `pkg/build/builders` (`AppleBuilder`)
- **RFC:** `code/core/build/RFC.md` §8 (Apple Build Target) / `code/core/go/build/RFC.md` §15

## 1. Background

A gap-analysis of `dappco.re/go/build` against the build RFCs found the spec ~95%
implemented: every builder, publisher, signer, SDK generator, LinuxKit image, the
PWA/API/service surfaces, and the full Apple §8 *API surface* exist and are real.

The one genuine gap is the **Apple build pipeline**, which is a deliberate
sandbox-safe **skeleton**. It constructs the correct external commands and has a
command-runner seam, but:

1. `NewAppleBuilder` defaults its `runner` to `nil`, so `runExternal` records the
   command (via `printTODO`) and returns `Ok` **without executing** — even on
   darwin.
2. `BuildWailsMacOS`, `CreateUniversal`, and `CreateDMG` then write a **placeholder**
   `.app`/`.dmg` file, clobbering any real tool output.

All three credential-free operations carry `TODO(#484)` markers.

## 2. Goal & scope

Promote the **credential-free** Apple operations from skeleton to real execution so
`core build apple` produces genuine artifacts on macOS, while staying CI-safe and
TDD-locked.

**In scope** (`builders.AppleBuilder`):

| Method | Real command | Tooling needed |
|--------|--------------|----------------|
| `BuildWailsMacOS` | `wails3 build -platform darwin/{arch} …` | macOS + wails3 CLI |
| `CreateUniversal` | `lipo -create -output {out} {arm64} {amd64}` | macOS (Xcode CLT) |
| `CreateDMG` | `hdiutil create → attach → detach → convert` | macOS (built-in) |

**Out of scope** (remain skeleton — credential-gated, need Apple Developer secrets
not available in this environment): `Sign`, `Notarise`, `UploadTestFlight`,
`SubmitAppStore`. Also out of scope: the `pkg/build/apple` facade fn-var seams and
any public API-shape change.

## 3. Architecture

Two layers, unchanged in shape:

- **`pkg/build/apple`** — the RFC §8 facade (`apple.New`, free functions
  `BuildWailsApp`/`CreateUniversal`/`CreateDMG`/…). Thin; delegates through
  package-level function-vars (`buildWailsAppFn`, `createUniversalFn`,
  `createDMGFn`) that route into the implementation. **No changes here** — it
  inherits the hardening transitively.
- **`pkg/build/builders` (`AppleBuilder`)** — the implementation: real command
  construction, the `AppleCommandRunner` seam (`GoProcessAppleRunner` →
  `runWithOptions` → `ax.ExecWithEnv`), and the placeholder writers. **All changes
  land here.**

## 4. Design

### 4.1 Runner default policy

`NewAppleBuilder` defaults `runner` to `GoProcessAppleRunner{}` (was `nil`).

`runExternal` already gates on OS and therefore needs no change:

- **non-darwin** → `printTODO` + return `Ok` *without executing* (CI-safe record).
- **darwin** → execute the real command via the runner; non-zero exit → `core.Fail`.

Tests override the runner via the existing `WithAppleCommandRunner(rec)` plus
`WithAppleHostOS("darwin")`.

### 4.2 Placeholder removal (core change)

Each of the three methods writes its skeleton placeholder **only when
`hostOS != darwin`**. On darwin the genuine tool output is the result and must not
be overwritten.

- `CreateDMG` (`apple_dmg.go`): move the placeholder write (currently `:97-106`)
  behind the non-darwin guard.
- `BuildWailsMacOS` (`apple.go`): call `createAppleBundleSkeleton` only on
  non-darwin.
- `CreateUniversal` (`apple.go`): write the placeholder `.app` only on non-darwin.

The OS check uses the same helper the methods already use
(`firstNonEmptyApple(b.hostOS, runtime.GOOS) == "darwin"`), so `WithAppleHostOS`
controls it deterministically in tests.

Method **return values are unchanged** — the methods return what they return today
(`core.Ok(nil)` / the existing value); the facade supplies the artifact path. This
change is about *producing* the real artifact, not altering the return contract.

### 4.3 Error handling

No new error paths. Real tool failure surfaces through the existing
`runExternal → runner → core.Fail` chain (non-zero exit or command-resolve
failure), matching `pkg/build/signing` behaviour. A missing `wails3`/`lipo`/
`hdiutil` on darwin fails honestly. Non-darwin always returns `Ok` (records +
skeleton output for downstream lanes).

## 5. TDD strategy

Red → green per method; hermetic; runs in CI and on this Mac.

1. **Command-construction tests (primary).** A `recordingRunner` test double
   implements `AppleCommandRunner` and captures each `RunOptions`. Injected via
   `WithAppleCommandRunner(rec)` + `WithAppleHostOS("darwin")`. Assertions on the
   exact command + arg sequence:
   - `CreateDMG` → `hdiutil create -volname … -srcfolder … -format UDRW`, then
     `attach`, `detach`, `convert -format UDZO -o {out}`.
   - `CreateUniversal` → `lipo -create -output {out} {arm64bin} {amd64bin}`.
   - `BuildWailsMacOS` → `wails3 build -platform darwin/{arch} …` (+ tags, ldflags,
     env as already constructed).
   AX-7 `Good/Bad/Ugly` triplets per method, with distinct real assertions (no
   `AssertNotPanics`/counter theatre, no tautologies).
2. **Placeholder-policy tests.** Assert: darwin (recording runner) leaves the output
   path *without* a placeholder overwrite; non-darwin writes the skeleton marker.
3. **Optional darwin real-tool test (confidence).** Skip-if-absent execution of real
   `lipo`/`hdiutil` against a fixture bundle (mirrors the signing fake-tool-on-PATH
   pattern) exercising `GoProcessAppleRunner → ax.ExecWithEnv`. Skips cleanly in CI
   and where the tool is missing.

All tests file-aware per the v0.9.0 audit (`Test<File>_<Symbol>_{Good,Bad,Ugly}` in
the matching `_test.go`).

## 6. Acceptance criteria

- `NewAppleBuilder` defaults to an executing runner; darwin execution wired.
- The three methods do not overwrite real output with placeholders on darwin;
  non-darwin still yields a skeleton file.
- Command-construction triplets pass; placeholder-policy tests pass.
- `go build ./...`, `go vet ./...`, `go test ./...` green (workspace mode, no
  `GOWORK=off`).
- No regression in the `pkg/build/builders` audit dimensions (no new theatre).
- `Sign`/`Notarise`/`TestFlight`/`AppStore` behaviour unchanged.

## 7. File-level change list (anticipated)

- `pkg/build/builders/apple.go` — runner default in `NewAppleBuilder`; placeholder
  guards in `BuildWailsMacOS` + `CreateUniversal`.
- `pkg/build/builders/apple_dmg.go` — placeholder guard in `CreateDMG`.
- `pkg/build/builders/apple_*_test.go` — recording-runner triplets + placeholder-policy
  tests (+ optional darwin real-tool test).

Planning will trace the facade fn-var chain
(`createDMGFn`/`buildWailsAppFn`/`createUniversalFn`) to confirm it routes into
these `builders.AppleBuilder` methods (the placeholder writers + runner seam are
confirmed to live there). If a stage routes elsewhere, the same guard pattern
applies at that stage; no facade API change is anticipated either way.
