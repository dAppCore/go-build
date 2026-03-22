# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`core/go-build` is the build system, release pipeline, and SDK generation tool for the Core ecosystem. Three subsystems under `pkg/` — **build**, **release**, **sdk** — can be used as libraries or wired together via CLI commands in `cmd/`. This repo produces no standalone binary; `cmd/` packages register commands via `cli.RegisterCommands()` in `init()` functions, compiled into the `core` binary from `forge.lthn.ai/core/cli`. Module path: `dappco.re/go/core/build`.

## Build & Test

```bash
go build ./...                                          # compile all packages
go test ./...                                           # run all tests
go test ./pkg/build/... -run TestLoadConfig_Good        # single test by name
go test -race ./...                                     # with race detection
```

**Go workspace**: this module is part of `~/Code/go.work`. Run `go work sync` after cloning. Set `GOPRIVATE=dappco.re/*,forge.lthn.ai/*` for private module fetching.

## Architecture

The three subsystems share common types but are independent:

- **`pkg/build/`** — Config loading (`.core/build.yaml`), project discovery via marker files, `Builder` interface, archiving (tar.gz/zip), checksums (SHA-256)
- **`pkg/build/builders/`** — Go, Wails, Docker, LinuxKit, C++, Taskfile builder implementations
- **`pkg/build/signing/`** — `Signer` interface with GPG, macOS codesign/notarisation, Windows (placeholder). Credentials support `$ENV` expansion
- **`pkg/release/`** — Version resolution from git tags, conventional-commit changelog generation, release orchestration. Two entry points: `Run()` (full pipeline) and `Publish()` (pre-built artifacts from `dist/`)
- **`pkg/release/publishers/`** — `Publisher` interface: GitHub, Docker, npm, Homebrew, Scoop, AUR, Chocolatey, LinuxKit
- **`pkg/sdk/`** — OpenAPI spec detection, breaking-change diff via oasdiff, SDK code generation
- **`pkg/sdk/generators/`** — `Generator` interface with registry. TypeScript, Python, Go, PHP generators (native tool -> npx -> Docker fallback)
- **`cmd/build/`** — `core build` commands (build, from-path, pwa, sdk, release)
- **`cmd/ci/`** — `core ci` commands (publish, init, changelog, version)
- **`cmd/sdk/`** — `core sdk` commands (diff, validate)

### Key Data Flow

```
.core/build.yaml -> LoadConfig() -> BuildConfig
project dir      -> Discover()   -> ProjectType -> getBuilder() -> Builder.Build() -> []Artifact
                                    -> SignBinaries() -> ArchiveAll() -> ChecksumAll() -> Publisher.Publish()
```

### Key Interfaces

- `build.Builder` — `Name()`, `Detect(fs, dir)`, `Build(ctx, cfg, targets)`
- `publishers.Publisher` — `Name()`, `Publish(ctx, release, pubCfg, relCfg, dryRun)`
- `signing.Signer` — `Name()`, `Available()`, `Sign(ctx, fs, path)`
- `generators.Generator` — `Language()`, `Generate(ctx, opts)`, `Available()`, `Install()`

### Filesystem Abstraction

All file operations use `io.Medium` from `dappco.re/go/core/io`. Production uses `io.Local`; tests inject mocks for isolation.

### Configuration Files

- `.core/build.yaml` — Build config (targets, flags, signing)
- `.core/release.yaml` — Release config (publishers, changelog, SDK settings)

## Coding Standards

- **UK English** in comments and strings (colour, organisation, notarisation)
- **Strict types** — all parameters and return types explicitly typed
- **Error wrapping** — `coreerr.E("package.Function", "message", err)` via `coreerr "dappco.re/go/core/log"`
- **testify** (`assert`/`require`) for assertions
- **Test naming** — `_Good` (happy path), `_Bad` (expected errors), `_Ugly` (edge cases)
- **Conventional commits** — `type(scope): description`
- **Licence** — EUPL-1.2

## Extension Points

**New builder**: implement `build.Builder` in `pkg/build/builders/`, add to `getBuilder()` in `cmd/build/cmd_project.go` and `pkg/release/release.go`, optionally add `ProjectType` to `pkg/build/build.go` and marker to `pkg/build/discovery.go`.

**New publisher**: implement `publishers.Publisher` in `pkg/release/publishers/`, add to `getPublisher()` in `pkg/release/release.go`, add config fields to `PublisherConfig` in `pkg/release/config.go` and `buildExtendedConfig()`.

**New SDK generator**: implement `generators.Generator` in `pkg/sdk/generators/`, register in `pkg/sdk/sdk.go` `GenerateLanguage()`.

## Dependencies

- `forge.lthn.ai/core/cli` — Command registration (`cli.RegisterCommands`, `cli.Command`) *(not yet migrated)*
- `dappco.re/go/core/io` — Filesystem abstraction (`io.Medium`, `io.Local`)
- `dappco.re/go/core/i18n` — Internationalisation (`i18n.T()`, `i18n.Label()`)
- `dappco.re/go/core/log` — Structured logging
- `github.com/Snider/Borg` — XZ compression for archives
- `github.com/getkin/kin-openapi` + `github.com/oasdiff/oasdiff` — OpenAPI parsing and diff
