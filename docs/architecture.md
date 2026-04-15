---
title: Architecture
description: Internal design of the build pipeline, public action parity, setup planning, builders, and packaging layers.
---

# Architecture

go-build mirrors the modular `dAppCore/build@v3` action architecture in Go rather than reimplementing it as shell-only composites. The pipeline shape stays recognisable, but the implementation lives in typed packages, builders, and planning APIs.

## Overview

The system has one gateway shape and several specialised layers:

1. Discovery gathers repository markers, distro hints, and Git metadata.
2. Orchestration resolves the effective stack and build configuration.
3. Options computes build flags deterministically.
4. Setup planning determines which toolchains are required.
5. Builders execute stack-specific build logic.
6. Signing and packaging wrap the build outputs.
7. Release and SDK layers sit alongside the build pipeline and reuse its metadata.

## Pipeline Pattern

The public action and the Go implementation follow the same top-level flow:

```text
Gateway
  -> Discovery
  -> Orchestration
  -> Options
  -> Setup
  -> Build
  -> Sign
  -> Package
```

In the public action this is split across `actions/discovery`, `actions/options`, `actions/setup/*`, `actions/build/*`, `actions/sign`, and `actions/package`.

In go-build the equivalents are:

- `build.Pipeline` for the gateway/orchestration layer
- `build.Discover()` and `build.DiscoverFull()` for discovery
- `build.ComputeOptions()` for pure build-option computation
- `build.ComputeSetupPlan()` for action-style setup planning
- `pkg/build/builders/` for stack-specific wrappers
- `pkg/build/signing/` and `pkg/build/apple.go` for signing/notarisation
- `pkg/build/archive.go`, `pkg/build/checksum.go`, and `pkg/release/` for packaging and publishing

## Data Flow

The action design relies on outputs flowing downstream. The Go implementation keeps the same rule.

```text
PipelineRequest
  -> load config
  -> DiscoverFull()
  -> ComputeOptions()
  -> ComputeSetupPlan()
  -> resolve builder
  -> build runtime config
  -> Builder.Build()
  -> archive/checksum/release
```

The practical consequence is that discovery becomes shared context for every later phase instead of each phase re-reading the repository independently.

## Discovery

`DiscoverFull()` is the Go equivalent of the public action discovery step. It records:

- detected project types in priority order
- configured build-type overrides from `.core/build.yaml`
- host OS and architecture
- stack suggestions such as `wails2`, `cpp`, `docs`, `node`, and `go`
- root marker presence for Go, Wails, CMake, Composer, Cargo, Taskfile, and MkDocs
- frontend manifests at the root, under `frontend/`, and in visible nested trees up to depth 2
- Deno manifests and Deno-requested state
- distro-aware Linux package requirements
- GitHub/Git metadata for refs, branch/tag state, SHA, and owner/repo context

Important behaviours preserved from the action:

- Wails detection accepts direct `wails.json` projects and Go roots that contain frontend manifests.
- Docs detection treats MkDocs as a first-class stack rather than a generic Node project.
- Nested frontend discovery ignores `node_modules` and hidden directories.
- Stack suggestions preserve the action naming contract instead of only returning Go-side project types.

The build API exposes this contract through `GET /api/v1/build/discover`, including workflow-facing aliases such as `configured_build_type`, `has_subtree_package_json`, `has_subtree_deno_manifest`, `has_taskfile`, and a serialized `setup_plan`.

## Options

`ComputeOptions()` is intentionally a pure transformation step:

- config plus discovery in
- computed flags out

It currently derives:

- Go build tags
- Ubuntu 24.04+ `webkit2_41` tag injection for Wails
- obfuscation
- NSIS
- WebView2 mode
- ldflags

That mirrors the action's options phase, where flag computation is kept separate from tool installation and build execution.

## Setup Planning

`ComputeSetupPlan()` is the Go analogue of the action setup orchestrator. It does not install tools itself. It computes which toolchains a runner or caller needs:

- Go
- garble
- Task
- Node/Corepack
- Wails
- Python
- PHP and Composer
- Rust
- Conan
- MkDocs
- Deno

This keeps setup thin and compositional. A builder can depend on a setup plan without coupling its execution logic to runner bootstrap details.

## Configuration Resolution

The public action resolves configuration once with clear precedence and then passes the resolved values downstream. go-build follows the same rule.

The effective precedence is:

1. CLI or API request overrides such as `--build-tags`, `--build-obfuscate`, `--nsis`, `--deno-build`, `--wails-build-webview2`, and `--build-cache`
2. persisted `.core/build.yaml`
3. environment-driven opt-in features such as `DENO_ENABLE` and `DENO_BUILD`
4. package defaults

`build.Pipeline` applies these overrides before discovery-dependent planning is consumed by builders, so callers do not need to mutate loaded config manually.

## Builder Layer

Every stack builder implements the same interface:

```go
type Builder interface {
    Name() string
    Detect(fs io.Medium, dir string) (bool, error)
    Build(ctx context.Context, cfg *Config, targets []Target) ([]Artifact, error)
}
```

Current builder families:

| Builder | Notes |
|---|---|
| Go | Cross-compiles binaries and supports garble plus cache wiring |
| Wails | Handles Wails v2 directly and Wails v3 through Taskfile or CLI fallback |
| Node | Uses package-manager or Deno-backed frontend builds |
| Docs | Runs MkDocs and packages the generated site |
| C++ | Uses CMake and Conan-aware setup |
| Docker | Uses Buildx-friendly image builds and export modes |
| LinuxKit | Produces configured LinuxKit artefacts |
| Taskfile | Delegates to repositories that already define their own build graph |
| PHP | Composer-backed packaging |
| Python | Deterministic source bundle packaging |
| Rust | Cargo release builds |

The design rule remains the same as the public action: discovery stays generic, setup stays conditional, and each stack wrapper owns its own build details.

## Generated Workflow

`core build workflow` writes `.github/workflows/release.yml`. The embedded template mirrors the public action surface instead of acting like a generic CI stub.

The generated workflow includes:

1. discovery outputs for markers, distro, and Git metadata
2. conditional Go, Node, Task, Deno, Wails, Python, PHP/Composer, Rust, Conan, and MkDocs setup
3. Linux distro-aware WebKit dependency selection for Wails
4. cache restore/save for `.core/cache` and `cache/`
5. `core build --archive --checksum`
6. action-style artifact naming and upload
7. tag-gated `core ci` release publishing

The workflow keeps the public action inputs visible at the CLI layer:

- `build-name`
- `build-platform`
- `build-tags`
- `build-obfuscate`
- `nsis`
- `deno-build`
- `wails-build-webview2`
- `build-cache`

## Packaging and Release

Packaging is intentionally separate from the builder layer:

- builders emit artefacts
- archive helpers compress them
- checksum helpers produce `CHECKSUMS.txt`
- CI helpers write `artifact_meta.json`
- `pkg/release` handles changelog/versioning/publishing

That separation makes it possible to reuse the same build outputs across local builds, reusable workflows, and release publishing.

## Apple, Release, and SDK Layers

The build pipeline is only one part of the module. The other major layers are:

- `pkg/build/apple.go` for macOS build/sign/notarise/package/distribute flows
- `pkg/build/apple/` for the RFC-facing wrapper exposing `core.Result`
- `pkg/release/` for versioning, changelogs, and publisher orchestration
- `pkg/sdk/` for OpenAPI detection, validation, diffing, and SDK generation

These layers reuse the same config and build metadata model rather than creating unrelated subsystems.

## Design Principles

- Modular: discovery, options, setup planning, build, sign, and package remain separate stages.
- Outputs flow downstream: discovery data becomes the shared context for later phases.
- Smart defaults: auto-detection and stack suggestion choose the right path unless config overrides them.
- Thin orchestrators: setup planning and stack routing coordinate specialised implementations instead of containing all logic themselves.
- Conditional setup: only the required toolchains and CLIs are installed for the selected stack.
- Stack ownership: each builder owns its own execution details and packaging expectations.

## Testing Strategy

The repo tests the action-parity surfaces directly instead of treating the generated workflow as opaque text:

- `pkg/build/discovery_test.go` covers marker detection, stack suggestion, distro handling, and nested frontend scanning.
- `pkg/build/options_test.go` covers deterministic flag computation and Ubuntu WebKit tag injection.
- `pkg/build/setup_test.go` covers toolchain planning and frontend directory resolution.
- `pkg/build/workflow_test.go` asserts that the embedded workflow still exposes the expected discovery outputs, setup steps, inputs, and artifact naming.
- builder tests exercise Wails v2/v3, Deno overrides, Conan-aware C++, MkDocs packaging, garble obfuscation, and WebView2 handling.
- Apple, release, and SDK packages keep their own dedicated coverage.

## Extending Stacks

Adding a new stack follows the same shape as the public action architecture:

1. Add or refine marker detection in `pkg/build/discovery.go`.
2. Add setup requirements in `pkg/build/setup.go` when new toolchains are required.
3. Implement the builder in `pkg/build/builders/`.
4. Thread any public workflow or CLI inputs through the command layer and the embedded workflow template.
5. Add fixture coverage and builder/workflow tests so the discovery and setup contract stays stable.
