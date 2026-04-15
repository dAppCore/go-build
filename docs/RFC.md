---
title: RFC
description: Authoritative overview of the build system, GitHub Action parity, publishers, and Apple pipeline.
---

# core/build RFC

`dappco.re/go/core/build` is the build system, release engine, Apple packaging layer, SDK generator, and reusable workflow surface behind:

- `core build`
- `core build apple`
- `core build workflow`
- `core release`
- `core sdk`
- `core ci`
- the public `dAppCore/build@v3` GitHub Action architecture

## Overview

The project has two linked jobs:

1. Provide a Go package that implements discovery, setup planning, stack-specific builds, packaging, signing, publishing, Apple delivery, and SDK generation.
2. Preserve parity with the public `dAppCore/build@v3` action so the Go implementation and the action evolve around the same pipeline model.

That keeps the public action as the distribution funnel while `go-build` remains the reusable engine:

```text
Developer searches "wails build action"
  -> Finds dAppCore/build on GitHub
    -> Uses it successfully
      -> Discovers the wider Core tooling
```

## Builders

| Builder | Detects | Notes |
|---|---|---|
| `go` | `go.mod`, `go.work` | Cross-compiles binaries and supports garble obfuscation |
| `wails` | `wails.json` or Go roots with frontend manifests | Handles Wails v2 directly and Wails v3 through Taskfile or CLI fallback |
| `node` | `package.json`, `deno.json`, `deno.jsonc` | Supports package-manager builds and Deno overrides |
| `docs` | `mkdocs.yml`, `mkdocs.yaml`, `docs/mkdocs.*` | Builds MkDocs sites and packages the output |
| `cpp` | `CMakeLists.txt` | Uses CMake with Conan-aware setup |
| `docker` | `Dockerfile`, `Containerfile` variants | Uses Buildx-backed image builds and archive-friendly exports |
| `linuxkit` | `linuxkit.yml`, `.core/linuxkit/*.yml` | Produces LinuxKit images in configured formats |
| `taskfile` | `Taskfile.yml`, `Taskfile.yaml`, `Taskfile` | Generic wrapper for repos that already define their own build graph |
| `php`, `python`, `rust` | language-native markers | Deterministic packaging or native release builds per stack |

Auto-detection follows the same high-level order as the public action:

```text
core build
  -> .core/build.yaml exists? use configured type
  -> Wails markers? Wails builder
  -> Go markers? Go builder
  -> MkDocs markers? Docs builder
  -> CMakeLists.txt? C++ builder
  -> Dockerfile/Containerfile? Docker builder
  -> Taskfile? Taskfile builder
```

## Publishers

Release publishing currently covers:

- GitHub Releases
- Docker registries
- npm
- Homebrew
- Scoop
- AUR
- Chocolatey
- LinuxKit registries

## GitHub Action Surface

The public action remains `dAppCore/build@v3`.

Its main responsibilities are:

- install Go, Node, Wails, and optional Deno or Conan tooling
- detect the stack from repository markers and distro hints
- compute action-style build options such as obfuscation, NSIS, and WebView2
- run the build and signing phases
- upload workflow artifacts and publish releases on tags

The generated workflow in this repo preserves the same user-facing control surface:

- `core-version`
- `go-version`
- `node-version`
- `wails-version`
- `build`
- `sign`
- `package`
- `build-name`
- `build-platform`
- `build-tags`
- `build-obfuscate`
- `nsis`
- `deno-build`
- `wails-build-webview2`
- `build-cache`

## Action Architecture

The generated workflow mirrors the composable `dAppCore/build@v3` structure:

```text
Gateway
  -> Discovery
  -> Option computation
  -> Setup planning / toolchain setup
  -> Stack-specific build
  -> Sign
  -> Package / release
```

The Go equivalents are:

- `DiscoverFull()` for the action-style discovery pass
- `ComputeOptions()` for deterministic build flag derivation
- `ComputeSetupPlan()` for setup orchestration inputs
- builder implementations in `pkg/build/builders/` for stack-specific execution

This keeps the Go package aligned with the action architecture without copying the action repository's bash and PowerShell split directly.

## Discovery Contract

Discovery preserves the richer `dAppCore/build@v3` model:

- Wails detection accepts direct `wails.json` projects and also Go roots with frontend manifests.
- Frontend discovery scans the root, `frontend/`, and nested trees up to depth 2.
- Docs detection treats MkDocs as a dedicated stack instead of falling through to Node.
- Linux distro detection feeds Ubuntu-aware WebKit dependency selection.
- Stack suggestions preserve action naming such as `wails2`, `cpp`, `docs`, and `node`.
- Git metadata is surfaced for artifact naming and release behavior.
- The build API exposes action-compatible aliases such as `configured_build_type`, `has_subtree_package_json`, `has_subtree_deno_manifest`, `has_taskfile`, root Composer/Cargo markers, and a serialized `setup_plan`.

## Ported Action Behaviours

The Go implementation intentionally carries forward the higher-signal action features:

- Ubuntu 20.04/22.04 vs 24.04 WebKit dependency handling
- `webkit2_41` build-tag injection for Wails on Ubuntu 24.04+
- subtree frontend scanning for monorepos
- setup planning for Go, Node, Wails, Deno, Task, Python, Conan, MkDocs, PHP/Composer, and Rust toolchains
- MkDocs detection and setup hooks
- Conan setup hooks for C++
- NSIS packaging for Windows Wails builds
- WebView2 modes: `download`, `embed`, `browser`, `error`
- garble-based obfuscation
- `DENO_BUILD` and `DENO_ENABLE` support
- build cache restore/save wiring under `.core/cache` and `cache/`

## Apple Target

The Apple surface provides:

- `core build apple`
- arm64, amd64, and universal macOS app builds
- the RFC-facing `pkg/build/apple/` wrapper with `core.Result`-based `Builder`, `AppleBuilder`, and functional options such as `WithArch`, `WithSign`, `WithNotarise`, `WithDMG`, `WithTestFlight`, and `WithAppStore`
- generated `Info.plist` and entitlements
- codesign and notarisation
- DMG creation for direct distribution
- TestFlight and App Store upload flows
- App Store preflight checks for metadata, privacy policy URL, minimum macOS version, licence declaration, and private API scanning
- Xcode Cloud helper script generation from `.core/build.yaml`

The runtime Apple pipeline follows this order:

1. Build a Wails macOS app bundle for the requested architecture.
2. Merge arch-specific bundles with `lipo` for universal builds.
3. Generate `Info.plist` and entitlements from config.
4. Sign with `codesign`.
5. Package a DMG when requested.
6. Notarise with `xcrun notarytool`.
7. Upload to TestFlight or App Store Connect when requested.

Xcode Cloud generation writes the checked-in scripts expected by the spec:

- `ci_scripts/ci_post_clone.sh`
- `ci_scripts/ci_pre_xcodebuild.sh`
- `ci_scripts/ci_post_xcodebuild.sh`

## Code Signing

Signing and trust surfaces currently include:

- GPG for `CHECKSUMS.txt.asc`
- `codesign` for macOS bundles and related artefacts
- `notarytool` for Apple notarisation and stapling
- `signtool` placeholders and integration points for Windows signing

Credentials are loaded from `.core/build.yaml` with environment expansion.

## SDK Generation

`pkg/sdk` detects OpenAPI specs, validates them, compares revisions with `oasdiff`, and generates SDKs for:

- TypeScript
- Python
- Go
- PHP

Generators prefer native tooling first and fall back to `npx` or Docker where appropriate.

## Supporting Docs

- [Architecture](architecture.md)
- [Stacks](stacks.md)
- [Development](development.md)
