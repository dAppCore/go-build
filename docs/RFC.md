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

The repository has two linked jobs:

1. Provide a Go package that implements discovery, stack-specific builds, packaging, signing, publishing, and SDK generation.
2. Preserve parity with the public action architecture so the Go implementation and the GitHub Action evolve around the same pipeline model.

The runtime flow is:

1. Discovery gathers repository markers, frontend layout, distro hints, and git metadata.
2. Option computation merges config, CLI overrides, and discovery-derived flags.
3. Toolchain setup stays conditional and stack-aware.
4. Builders own stack-specific execution.
5. Signing, checksums, archiving, release publishing, and workflow packaging happen last.

## GitHub Action Surface

The public action surface remains `dAppCore/build@v3`.

Its main responsibilities are:

- install Go, Node, Wails, and optional Deno or Conan tooling
- detect the stack from repository markers and distro hints
- compute action-style build options such as obfuscation, NSIS, and WebView2
- run the build and signing phases
- upload workflow artifacts and publish releases on tags

The generated workflow in this repo preserves the action-style input model:

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

## Builders

| Builder | Detects | Notes |
|---|---|---|
| `go` | `go.mod`, `go.work` | Cross-compiles binaries and supports garble obfuscation |
| `wails` | `wails.json` or Go roots with frontend manifests | Handles Wails v2 directly and Wails v3 via Taskfile or CLI fallback |
| `node` | `package.json`, `deno.json`, `deno.jsonc` | Supports package-manager builds and Deno overrides |
| `docs` | `mkdocs.yml`, `mkdocs.yaml`, `docs/mkdocs.*` | Builds MkDocs sites and packages the output |
| `cpp` | `CMakeLists.txt` | Uses CMake with Conan-aware setup |
| `docker` | `Dockerfile`, `Containerfile` variants | Uses Buildx-backed image builds and archive-friendly exports |
| `linuxkit` | `linuxkit.yml`, `.core/linuxkit/*.yml` | Produces LinuxKit images in configured formats |
| `taskfile` | `Taskfile.yml`, `Taskfile.yaml`, `Taskfile` | Generic wrapper for repos that already define their own build graph |
| `php`, `python`, `rust` | language-native markers | Deterministic packaging or native release builds per stack |

## Discovery Contract

Discovery preserves the richer `dAppCore/build@v3` action model:

- Wails detection accepts direct `wails.json` projects and Go roots with frontend manifests.
- Frontend discovery scans the root, `frontend/`, and nested trees up to depth 2.
- Docs detection treats MkDocs as a dedicated stack instead of falling through to Node.
- Linux distro detection feeds Ubuntu-aware WebKit dependency selection.
- Stack suggestions preserve action naming such as `wails2`, `cpp`, `docs`, and `node`.
- Git metadata is surfaced for artifact naming and release behavior.

## Action Parity

The Go implementation intentionally ports the high-signal action features:

- Ubuntu 20.04/22.04 vs 24.04 WebKit dependency handling
- `webkit2_41` build-tag injection for Wails on Ubuntu 24.04+
- subtree frontend scanning for monorepos
- MkDocs detection and setup hooks
- Conan setup hooks for C++
- NSIS packaging for Windows Wails builds
- WebView2 modes: `download`, `embed`, `browser`, `error`
- garble-based obfuscation
- `DENO_BUILD` and `DENO_ENABLE` support
- build cache restore/save wiring under `.core/cache` and `cache/`

`core build workflow` writes `.github/workflows/release.yml` to mirror that action pipeline.

The local docs in this repo track the architecture docs from the public action:

- discovery runs first and exports marker, git, and distro context downstream
- option computation is deterministic and side-effect free
- setup stays thin and conditional instead of becoming a monolithic shell script
- stack wrappers own full pipeline execution for Wails, Docs, C++, Docker, LinuxKit, and Taskfile builds

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

Xcode Cloud generation writes the checked-in scripts expected by the spec:

- `ci_scripts/ci_post_clone.sh`
- `ci_scripts/ci_pre_xcodebuild.sh`
- `ci_scripts/ci_post_xcodebuild.sh`

The RFC-facing wrapper lives in `pkg/build/apple/` and exposes `core.Result`-based contracts for Apple builder APIs.

## SDK Generation

`pkg/sdk` detects OpenAPI specs, validates them, compares revisions with `oasdiff`, and generates SDKs for:

- TypeScript
- Python
- Go
- PHP

## Supporting Docs

- [Architecture](architecture.md)
- [Stacks](stacks.md)
- [Development](development.md)
