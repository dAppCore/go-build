---
title: RFC
description: Authoritative spec for the build system, public GitHub Action parity, publishers, and Apple pipeline.
---

# core/build RFC

`dappco.re/go/core/build` is the build, release, signing, SDK, Apple packaging, and workflow engine behind:

- `core build`
- `core build apple`
- `core build workflow`
- `core release`
- `core sdk`
- `core ci`
- the public `dAppCore/build@v3` GitHub Action surface

## Overview

The build system has two linked faces:

1. A Go package that implements discovery, option computation, setup planning, stack-specific builds, packaging, signing, release publishing, Apple delivery, and SDK generation.
2. A public GitHub Action architecture that works as the distribution funnel for Wails and adjacent build workflows.

The action remains the public entrypoint while go-build is the reusable engine:

```text
Developer searches "wails build action"
  -> Finds dAppCore/build on GitHub
    -> Uses it successfully
      -> Discovers the wider Core tooling
```

## Builders

The original action/RFC centred on seven builder families. The Go implementation keeps those surfaces and extends them with additional language-native stacks.

| Builder | Detects | What It Builds |
|---|---|---|
| `go` | `go.mod`, `go.work` | Go binaries across a target matrix |
| `wails` | `wails.json` or Go roots with frontend manifests | Wails desktop apps, with Wails v2 direct builds and Wails v3 Taskfile or CLI fallback |
| `docker` | `Dockerfile`, `Containerfile` variants | OCI container images |
| `linuxkit` | `linuxkit.yml`, `.core/linuxkit/*.yml` | LinuxKit VM images |
| `cpp` | `CMakeLists.txt` | CMake-based C++ builds with Conan-aware setup |
| `taskfile` | `Taskfile.yml`, `Taskfile.yaml`, `Taskfile` | Task-driven build pipelines |
| `docs` | `mkdocs.yml`, `mkdocs.yaml`, `docs/mkdocs.*` | MkDocs documentation sites |
| `node` | `package.json`, `deno.json`, `deno.jsonc` | Frontend builds via package manager or Deno |
| `php` | `composer.json` | Composer-backed bundles |
| `python` | `pyproject.toml`, `requirements.txt` | Deterministic Python source bundles |
| `rust` | `Cargo.toml` | Cargo release builds |

Auto-detection follows the action-style precedence model:

```text
core build
  -> .core/build.yaml exists? use configured build.type
  -> Wails markers? Wails builder
  -> Go markers? Go builder
  -> MkDocs markers? Docs builder
  -> CMakeLists.txt? C++ builder
  -> Dockerfile or Containerfile? Docker builder
  -> Taskfile? Taskfile builder
```

## Publishers

Release publishing currently covers eight targets:

- GitHub Releases
- Docker registries
- npm
- Homebrew
- Scoop
- AUR
- Chocolatey
- LinuxKit registries

## Public GitHub Action

The public action surface remains `dAppCore/build@v3`.

### Example Usage

```yaml
- uses: dAppCore/build@v3
  with:
    build-name: myapp
    build-platform: linux/amd64
```

### Public Inputs

The generated workflow in this repo preserves the same high-signal control surface exposed by the action:

| Input | Default | Purpose |
|---|---|---|
| `build-name` | derived from config or repo name | Override output/artifact name |
| `build-platform` | matrix driven | Filter the build matrix to one target |
| `core-version` | `latest` | Pin the bootstrap Core CLI version used by the generated workflow |
| `version` | empty | Override the version embedded into build outputs and releases |
| `build` | `true` | Run the build phase |
| `sign` | `false` | Enable platform signing after build |
| `package` | `true` | Archive, checksum, upload artifacts, and publish on tags |
| `nsis` | `false` | Enable Wails Windows NSIS packaging |
| `build-tags` | empty | Forward Go build tags |
| `build-obfuscate` | `false` | Use garble or Wails obfuscation |
| `wails-version` | `latest` | Pin the Wails CLI |
| `go-version` | `1.26` | Pin Go |
| `node-version` | `22.x` | Pin Node |
| `deno-build` | empty | Override the default Deno build command |
| `wails-build-webview2` | empty | Set Windows WebView2 mode |
| `build-cache` | `true` | Restore and save build caches |
| `archive-format` | empty | Override the archive format used for packaged build artefacts; empty falls back to gzip (`.tar.gz`, or `.zip` on Windows) |

### What the Action Shape Does

The action and generated workflow preserve the same phase ordering:

1. Discovery of project markers, distro metadata, and Git context.
2. Option computation for tags, obfuscation, NSIS, and WebView2.
3. Conditional toolchain setup for the detected stack.
4. Stack-specific build execution.
5. Platform signing when requested.
6. Artifact upload and tag-gated release publishing.

Artifact naming follows the action convention:

```text
{build-name}_{os}_{arch}_{tag|shortsha}
```

### History

The public action architecture evolved in stages:

```text
wails-build-action@v2
  -> proved cross-platform Wails CI packaging
    -> dAppCore/build@v3
      -> decomposed into reusable action phases
        -> core/go-build
          -> Go implementation of discovery, setup, build, release, Apple, and SDK flows
```

The key point is architectural continuity: go-build is not an unrelated tool, it is the typed implementation of the public action model.

## Action Architecture and Parity

The public action is organised as composable phases:

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

The Go equivalents are:

- `build.Pipeline` for the gateway/orchestration layer
- `build.Discover()` and `build.DiscoverFull()` for discovery
- `build.ComputeOptions()` for deterministic flag derivation
- `build.ComputeSetupPlan()` for setup orchestration
- `pkg/build/builders/` for stack-specific build wrappers
- `pkg/build/signing/` and `pkg/build/apple.go` for signing/notarisation
- `pkg/build/archive.go`, `pkg/build/checksum.go`, and `pkg/release/` for packaging and publishing

### Discovery Contract

Discovery preserves the richer `dAppCore/build@v3` model rather than stopping at simple marker lookup:

- Wails detection accepts `wails.json` and also Go roots with frontend manifests.
- Frontend discovery scans the root, `frontend/`, and nested trees up to depth 2.
- Docs detection accepts `mkdocs.yml` and `mkdocs.yaml` in the root or `docs/`.
- Stack suggestions preserve action naming such as `wails2`, `cpp`, `docs`, and `node`.
- Linux distro detection feeds Ubuntu-aware WebKit dependency selection.
- Git metadata is surfaced for artifact naming and release decisions.
- The build API exposes workflow-facing aliases such as `configured_build_type`, `has_subtree_package_json`, `has_subtree_deno_manifest`, `has_taskfile`, root Composer/Cargo markers, and a serialized `setup_plan`.

### Ported Action Behaviours

The features that originally lived in the action now exist in Go and in the generated workflow:

| Behaviour | Status in go-build |
|---|---|
| Ubuntu-aware WebKit dependency selection | Implemented |
| `webkit2_41` tag injection for Ubuntu 24.04+ Wails builds | Implemented |
| Subtree frontend scanning to depth 2 | Implemented |
| MkDocs stack detection | Implemented |
| Stack suggestion aliases (`wails2`, `cpp`, `docs`, `node`) | Implemented |
| Conan-aware C++ setup planning | Implemented |
| Windows NSIS packaging for Wails | Implemented |
| garble-based obfuscation | Implemented |
| Deno setup and `DENO_BUILD` overrides | Implemented |
| Build cache wiring under `.core/cache` and `cache/` | Implemented |
| Windows WebView2 modes (`download`, `embed`, `browser`, `error`) | Implemented |

## SDK Generation

`pkg/sdk` detects OpenAPI specs, validates them, compares revisions with `oasdiff`, and generates SDKs for:

| Language | Primary Generator Path | Fallback |
|---|---|---|
| TypeScript | native or `npx` tooling | Docker where needed |
| Python | native client generator | Docker |
| Go | native Go tooling | none |
| PHP | generator CLI | Docker |

Breaking-change detection remains part of the release/SDK story through `oasdiff`.

## Code Signing

Signing and trust surfaces include:

| Signer | Platform | Purpose |
|---|---|---|
| GPG | all | `CHECKSUMS.txt.asc` |
| `codesign` | macOS | app bundles, frameworks, binaries, installers |
| `notarytool` | macOS | notarisation and stapling |
| `signtool` | Windows | Windows binary and installer signing integration points |

Credentials are loaded from `.core/build.yaml` and support environment expansion.

## Apple Build Target

The Apple surface covers build, sign, notarise, package, and distribute macOS applications through:

- `core build apple`
- `pkg/build/apple.go`
- the RFC-facing `pkg/build/apple/` wrapper that exposes `core.Result` contracts

### CLI Shape

```text
core build apple [flags]
```

Key flags:

| Flag | Default | Purpose |
|---|---|---|
| `--arch` | `universal` | `arm64`, `amd64`, or `universal` |
| `--sign` | `true` | Enable Apple code signing |
| `--notarise` | `true` | Submit to Apple notarisation |
| `--dmg` | `false` | Produce a distributable DMG |
| `--testflight` | `false` | Upload to TestFlight |
| `--appstore` | `false` | Submit to App Store Connect |
| `--team-id` | config | Apple Developer Team ID |
| `--bundle-id` | config | Bundle identifier |
| `--version` | Git or override | Version string |
| `--build-number` | generated | Integer build number |

### Builder and Options Surface

The RFC-facing wrapper exposes:

- `AppleBuilder`
- the `Builder` interface returning `core.Result` values from `Detect()` and `Build()`
- functional options such as `WithArch`, `WithSign`, `WithNotarise`, `WithDMG`, `WithTestFlight`, and `WithAppStore`
- `AppleOptions` mirroring the CLI/runtime pipeline, including signing identity, App Store Connect credentials, and delivery toggles

This wrapper keeps the Apple contract consistent with the wider Core service pattern while delegating the concrete implementation to `pkg/build/apple.go`.

### Wails macOS Build

Apple builds wrap Wails app generation for macOS:

1. Build an app bundle for `arm64`, `amd64`, or both.
2. Inject version metadata and build tags.
3. Merge arch-specific bundles with `lipo` for universal builds.
4. Produce `{OutputDir}/AppName.app`.

The Apple implementation exposes helper functions such as:

- `BuildWailsApp()`
- `CreateUniversal()`

`BuildWailsApp()` keeps the RFC-facing `LDFlags` field as a single string in `pkg/build/apple/` and converts it to the lower-level slice form expected by `pkg/build/apple.go`, so the wrapper stays stable while the build package remains CLI-friendly.

### Signing, Notarisation, and DMG Packaging

The Apple pipeline supports:

- inside-out signing of frameworks, helpers, binaries, and the `.app` bundle
- notarisation with `xcrun notarytool`
- stapling and verification
- DMG packaging for direct distribution, including background assets and `/Applications` symlink staging

Notarisation supports both auth paths from the RFC:

- App Store Connect API key authentication via `APIKeyID`, `APIKeyIssuerID`, and `APIKeyPath`
- Apple ID plus app-specific password as a fallback when API key credentials are not supplied

The runtime order is:

1. Build app bundle.
2. Generate `Info.plist` and entitlements.
3. Sign with `codesign`.
4. Package DMG when requested.
5. Notarise the app or DMG.
6. Upload to TestFlight or App Store Connect when requested.

### Xcode Cloud

Xcode Cloud support is configured in `.core/build.yaml` and generates the checked-in helper scripts expected by the Apple pipeline:

- `ci_scripts/ci_post_clone.sh`
- `ci_scripts/ci_pre_xcodebuild.sh`
- `ci_scripts/ci_post_xcodebuild.sh`

The generated flow is designed around:

- branch-triggered TestFlight delivery
- tag-triggered App Store delivery
- prebuild invocation of `core build apple`

### TestFlight and App Store Connect

The Apple surface includes runtime support for:

- TestFlight uploads
- App Store Connect submission
- App Store preflight checks for metadata, privacy policy URL, minimum macOS version, and distribution mode

### Info.plist and Entitlements

The Apple pipeline generates `Info.plist` and entitlements from config rather than treating them as static assets.

Important metadata includes:

- bundle identifier and names
- user-facing version and build number
- minimum macOS version
- App Store category
- copyright and licence notice

Entitlement generation distinguishes direct-distribution and App Store profiles so the build can request:

- network client and server access
- file access for user-selected and downloads locations
- Metal access
- sandboxing for App Store builds
- JIT-related entitlements only where that distribution mode allows them

## Reference Material

- Go module path: `dappco.re/go/core/build`
- Public action: `dAppCore/build@v3`
- Build config companion RFC: `code/core/config/RFC.md`

## Supporting Documents

- [Architecture](architecture.md)
- [Stacks](stacks.md)
- [Development](development.md)

## Changelog

- 2026-04-15: Synced the in-repo RFC with the current generated workflow surface by documenting `core-version`, `archive-format`, and the fuller Apple wrapper/notarisation contract.
