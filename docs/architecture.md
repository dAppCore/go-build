---
title: Architecture
description: Internal design of the build, action/workflow, Apple, release, and SDK layers.
---

# Architecture

go-build mirrors the modular `dAppCore/build@v3` action architecture in Go instead of shell composites. The same pipeline shape is preserved, but the implementation lives in typed packages and builders instead of `actions/*`.

## Overview

The repo has four major surfaces that share types but can be used independently:

1. `pkg/build` for discovery, setup planning, builders, artifacts, caches, workflow generation, and Apple packaging
2. `pkg/release` for versioning, changelogs, and publishers
3. `pkg/sdk` for OpenAPI diffing and SDK generation
4. `cmd/` for registering `core build`, `core ci`, and `core sdk`

## Pipeline Shape

The public action and the Go implementation follow the same high-level gateway pattern:

```text
Gateway
  -> Discovery
  -> Orchestration / stack suggestion
  -> Option computation
  -> Setup planning
  -> Toolchain setup
  -> Stack-specific build
  -> Sign
  -> Package / release
```

In the public action this is split across `actions/discovery`, `actions/options`, `actions/setup/*`, `actions/build/*`, `actions/sign`, and `actions/package`.

In go-build the equivalents are:

- `build.Pipeline` for the gateway/orchestration layer that resolves config, discovery, setup, and the selected builder
- `build.Discover()` and `build.DiscoverFull()`
- `build.ComputeOptions()`
- `build.ComputeSetupPlan()`
- builder implementations in `pkg/build/builders/`
- release orchestration in `pkg/release`

## Directory Structure

The action repository documents its pipeline in terms of `actions/*` folders. In go-build the same responsibilities are expressed as packages and builders:

| Action Surface | Go Surface | Responsibility |
|---|---|---|
| `actions/discovery/` | `pkg/build/discovery.go` | Marker scanning, distro detection, Git metadata, stack suggestion |
| `actions/options/` | `pkg/build/options.go` | Deterministic build-flag computation |
| `actions/setup/` and `actions/setup/*` | `pkg/build/setup.go` plus builder/toolchain helpers | Thin setup planning and toolchain-specific requirements |
| `actions/build/{stack}/` | `pkg/build/builders/` | Stack-specific build orchestration |
| `actions/sign/` | `pkg/build/signing/` and `pkg/build/apple.go` | macOS and Windows signing plus Apple notarisation |
| `actions/package/` | `pkg/build/archive.go`, `pkg/build/checksum.go`, `pkg/release/` | Archiving, checksums, artifact naming, release publishing |

This keeps the action architecture recognisable without copying the composite-action layout literally.

## Discovery and Stack Suggestion

`build.Discover()` and `build.DiscoverFull()` implement the action-style discovery pass.

They record:

- detected project types in priority order
- raw marker presence, including root Go/Wails/CMake markers and subtree frontend manifests
- whether a frontend exists at the root, in `frontend/`, or in a subtree up to depth 2
- distro-aware Linux package requirements
- action-facing stack suggestions such as `wails2`, `cpp`, `docs`, `node`, and `go`
- Git metadata for artifact naming and release behavior

Important detection behaviour:

- Wails detection accepts `wails.json` and also Go roots that contain frontend manifests
- Docs detection accepts `mkdocs.yml` and `mkdocs.yaml` in the root or `docs/`
- Docker detection accepts `Dockerfile` and `Containerfile` variants
- LinuxKit detection accepts root manifests and `.core/linuxkit/*.yml`
- Taskfile detection accepts common case variants

The build API exposes this richer discovery contract through `GET /api/v1/build/discover`, including workflow-facing aliases such as `configured_build_type`, `has_subtree_package_json`, `has_taskfile`, root Composer/Cargo markers, and a serialized `setup_plan`.

## Option and Setup Planning

`ComputeOptions()` is the Go equivalent of the action's pure options step. It folds config and discovery into a deterministic option set:

- Go build tags
- `webkit2_41` injection for Wails on Ubuntu 24.04+
- NSIS
- WebView2
- obfuscation
- ldflags

`ComputeSetupPlan()` is the thin orchestration layer for setup. It does not install tools itself; it computes what is needed:

- Go
- garble
- Task
- Node
- Wails
- Python
- PHP / Composer
- Rust
- Conan
- MkDocs
- Deno

That mirrors the public action's "thin orchestrator + specialised setup actions" design instead of collapsing setup into one monolithic script.

## Configuration Resolution

The public action resolves configuration once and then passes the resolved values downstream. go-build follows the same rule:

- CLI or workflow inputs override persisted config.
- Environment overrides are honoured for action-style features such as `DENO_ENABLE` and `DENO_BUILD`.
- Defaults fill the gaps only after explicit inputs and config have been considered.

In practice that means the public action's `inputs > environment > defaults` rule becomes:

- CLI request fields such as `--build-tags`, `--build-obfuscate`, `--nsis`, `--deno-build`, and `--wails-build-webview2`
- persisted `.core/build.yaml`
- environment-driven overrides for opt-in features that the action also exposes through `env:`
- package defaults such as cache paths, archive format, and stack fallbacks

`build.Pipeline` now resolves those action-style overrides directly from `PipelineRequest` before discovery, option computation, setup planning, and builder execution, so callers do not need to mutate `BuildConfig` up front.

## Builder Layer

Every builder implements:

```go
type Builder interface {
    Name() string
    Detect(fs io.Medium, dir string) (bool, error)
    Build(ctx context.Context, cfg *Config, targets []Target) ([]Artifact, error)
}
```

Current implementations:

| Builder | Notes |
|---|---|
| Go | Cross-compiles binaries and supports garble obfuscation plus cache env wiring |
| Wails | Handles Wails v2 directly and Wails v3 through Taskfile or CLI fallback; supports NSIS, WebView2, Deno, subtree frontends, and obfuscation |
| Node | Detects package manager, supports Deno manifests, and builds nested frontend projects |
| Docs | MkDocs build plus zipped site output |
| C++ | Make + Conan orchestration with profile-based cross-builds |
| Docker | Buildx-backed image builds with push/load/archive modes |
| LinuxKit | LinuxKit image generation in configured formats |
| Taskfile | Generic task-backed build pipeline used heavily by Wails v3 projects |
| PHP | Composer-backed builds with deterministic zip fallback |
| Python | Deterministic source bundle packaging |
| Rust | Cargo release builds by target triple |

The action principle still applies here: stack wrappers own the full pipeline for their technology instead of forcing everything through a single generic build command.

## Generated GitHub Workflow

`core build workflow` writes `.github/workflows/release.yml`. The generated workflow mirrors the modular `dAppCore/build@v3` pipeline:

1. Checkout
2. Discovery by repository markers and Git metadata, exported as workflow step outputs
3. Toolchain setup for Go, Node, PHP/Composer, Python, Rust, Deno, Task, Conan, MkDocs, and Wails
4. Linux distro-aware WebKit dependency setup for Wails
5. Cache restore under `.core/cache` and `cache/`
6. `core build --archive --checksum`
7. Artifact upload with action-style names: `{build-name}_{os}_{arch}_{tag|shortsha}`
8. Release publishing through `core ci`

The workflow keeps the public action inputs exposed at the CLI layer:

- `build-name`
- `build-platform`
- `build-tags`
- `build-obfuscate`
- `nsis`
- `deno-build`
- `wails-build-webview2`
- `build-cache`

## Ported Action Behaviours

The Go implementation intentionally preserves the higher-signal behaviours from the public action:

- Ubuntu-aware WebKit dependency selection and `webkit2_41` build-tag injection for Wails on 24.04+
- frontend manifest scanning at the root, under `frontend/`, and in nested trees up to depth 2
- MkDocs project detection and setup hooks
- Conan installation hooks and C++ build support
- Deno setup and `DENO_BUILD` overrides
- garble-based obfuscation, NSIS packaging, WebView2 modes, and build cache wiring

## Testing Strategy

The repo keeps the action-parity surfaces under test rather than treating the generated workflow as opaque output.

- `go test ./...` covers discovery, options, setup planning, workflow generation, builders, Apple packaging, release publishing, and SDK generation.
- `pkg/build/testdata/` provides fixture projects for stacks such as Wails, Go, docs, C++, Node, Python, Rust, PHP, and monorepo frontends.
- `pkg/build/workflow_test.go` asserts that the generated workflow still exposes the expected action-style inputs, discovery outputs, setup steps, and artifact naming.
- Builder-specific tests exercise stack behaviour directly, including Wails v2/v3 routing, Deno overrides, Conan integration, MkDocs packaging, garble obfuscation, and WebView2 handling.

## Apple, Release, and SDK Layers

The Apple implementation lives in `pkg/build/apple.go`, with an RFC-facing wrapper in `pkg/build/apple/`.

Key Apple pieces:

- `AppleOptions` for the runtime pipeline
- `BuildApple()` for the end-to-end macOS build flow
- `BuildWailsApp()` and `CreateUniversal()` for architecture-specific and universal app bundles
- `Sign()`, `Notarise()`, `CreateDMG()`, `UploadTestFlight()`, and `SubmitAppStore()` for post-build delivery
- generated `Info.plist` and entitlements
- Xcode Cloud script generation from `.core/build.yaml`

`pkg/release` owns version resolution, changelog generation, artifact reuse, checksum handling, and publisher orchestration.

`pkg/sdk` detects an OpenAPI spec, validates it, compares revisions with `oasdiff`, and generates SDKs for TypeScript, Python, Go, and PHP.

## Design Principles

- Modular: discovery, options, setup planning, build, sign, and package remain separate stages.
- Outputs flow downstream: discovery data becomes the shared context for later phases.
- Smart defaults: auto-detection and stack suggestion choose the right path unless config overrides them.
- Thin orchestrators: setup planning and stack routing coordinate specialised implementations instead of containing all logic themselves.
- Conditional setup: only the required toolchains and CLIs are installed for the selected stack.
- Stack ownership: each builder owns its own execution details and packaging expectations.

## Extending Stacks

Adding a new stack follows the same shape as the public action architecture:

1. Add or refine the marker detection in `pkg/build/discovery.go`.
2. Add setup requirements in `pkg/build/setup.go` when the stack needs new toolchains.
3. Implement the stack builder in `pkg/build/builders/`.
4. Thread any public workflow or CLI inputs through the generated workflow and command layer.
5. Add fixture coverage and builder/workflow tests so the discovery and setup contract stays stable.
