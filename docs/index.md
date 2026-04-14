---
title: go-build
description: Build, release, Apple packaging, SDK generation, and GitHub workflow tooling for Core projects.
---

# go-build

`dappco.re/go/core/build` is the build system and release engine used by the Core CLI and the public `dAppCore/build@v3` GitHub Action.

## Highlights

- Auto-detecting builders for Go, Wails, Node, PHP, Python, Rust, Docs, Docker, LinuxKit, C++, and Taskfile projects
- Action-oriented discovery hints for `wails2`, `cpp`, `docs`, `node`, and `go`
- Generated reusable GitHub release workflow with Go/Node/Python/Deno setup, Conan/MkDocs hooks, distro-aware WebKit packages, cache restore/save, and canonical artifact naming
- macOS Apple pipeline with `core build apple`, DMG packaging, notarisation, Xcode Cloud script generation, TestFlight, and App Store submission
- Release orchestration with eight publishers
- OpenAPI SDK generation with breaking-change detection

## Commands

```bash
core build
core build apple
core build workflow
core release
core sdk
core build sdk
core ci
```

## Build Surfaces

| Surface | Purpose |
|---|---|
| `pkg/build/` | Discovery, config, caches, archives, checksums, workflow generation, Apple pipeline |
| `pkg/build/builders/` | Builder implementations for all supported stacks |
| `pkg/build/apple/` | RFC-facing Apple wrapper that exposes `core.Result` contracts |
| `pkg/build/signing/` | GPG, macOS codesign/notarisation, Windows signtool |
| `pkg/release/` | Versioning, changelog generation, publishing orchestration |
| `pkg/release/publishers/` | GitHub, Docker, npm, Homebrew, Scoop, AUR, Chocolatey, LinuxKit |
| `pkg/sdk/` | Spec detection, diffing, and SDK generation |
| `cmd/build/` | `core build`, `core build apple`, `core build workflow`, `core build sdk`, `core release` |
| `cmd/ci/` | `core ci` publish/version/changelog commands |
| `cmd/sdk/` | `core sdk`, `core sdk diff`, and `core sdk validate` |

See also: [Architecture](architecture.md) and [Stacks](stacks.md).

## Builder Detection

Discovery checks the project root and selected nested paths:

| Marker | Result |
|---|---|
| `.core/build.yaml` | Config-driven override |
| `wails.json` or `go.mod`/`go.work` plus frontend manifests | Wails |
| `go.mod` or `go.work` | Go |
| `package.json`, `deno.json`, `deno.jsonc` | Node/Deno |
| `mkdocs.yml`, `mkdocs.yaml`, `docs/mkdocs.yml`, `docs/mkdocs.yaml` | Docs |
| `CMakeLists.txt` | C++ |
| `Dockerfile`, `Containerfile` variants | Docker |
| `linuxkit.yml`, `linuxkit.yaml`, `.core/linuxkit/*.yml` | LinuxKit |
| `Taskfile.yml`, `Taskfile.yaml`, `Taskfile` variants | Taskfile |
| `composer.json`, `pyproject.toml`, `requirements.txt`, `Cargo.toml` | PHP, Python, Rust |

Monorepo frontend discovery scans subtree manifests to depth 2 and ignores `node_modules` and hidden directories.

## GitHub Workflow Generation

`core build workflow` writes a reusable release workflow that:

1. Detects the required toolchains from the repository contents.
2. Installs Go, Node, Python, Conan, MkDocs, Deno, and Wails only when needed, plus frontend package dependencies and optional garble for obfuscated builds.
3. Restores build caches under `.core/cache` and `cache/`.
4. Applies Ubuntu 24.04 WebKit 4.1 handling for Wails Linux builds.
5. Runs `core build --archive --checksum`.
6. Uploads artifacts with action-style names and publishes with `core ci`.

## Apple Pipeline

The Apple surface is available both through `pkg/build/apple/` and `core build apple`. It supports:

- universal, arm64, and amd64 app builds
- codesign and notarisation
- DMG creation
- TestFlight and App Store submission
- generated `Info.plist` and entitlements
- Xcode Cloud helper scripts checked into the project

## Module Path

```go
import "dappco.re/go/core/build/pkg/build"
```

Requires Go 1.26+.
