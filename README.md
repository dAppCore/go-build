<!-- SPDX-License-Identifier: EUPL-1.2 -->

# go-build

> Go build orchestrator — cmd, project detection, release publishers

[![CI](https://github.com/dappcore/go-build/actions/workflows/ci.yml/badge.svg?branch=dev)](https://github.com/dappcore/go-build/actions/workflows/ci.yml)
[![Quality Gate](https://sonarcloud.io/api/project_badges/measure?project=dappcore_go-build&metric=alert_status)](https://sonarcloud.io/dashboard?id=dappcore_go-build)
[![Coverage](https://codecov.io/gh/dappcore/go-build/branch/dev/graph/badge.svg)](https://codecov.io/gh/dappcore/go-build)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=dappcore_go-build&metric=security_rating)](https://sonarcloud.io/dashboard?id=dappcore_go-build)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=dappcore_go-build&metric=sqale_rating)](https://sonarcloud.io/dashboard?id=dappcore_go-build)
[![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=dappcore_go-build&metric=reliability_rating)](https://sonarcloud.io/dashboard?id=dappcore_go-build)
[![Code Smells](https://sonarcloud.io/api/project_badges/measure?project=dappcore_go-build&metric=code_smells)](https://sonarcloud.io/dashboard?id=dappcore_go-build)
[![Lines of Code](https://sonarcloud.io/api/project_badges/measure?project=dappcore_go-build&metric=ncloc)](https://sonarcloud.io/dashboard?id=dappcore_go-build)
[![Go Reference](https://pkg.go.dev/badge/dappco.re/go/go-build.svg)](https://pkg.go.dev/dappco.re/go/go-build)
[![License: EUPL-1.2](https://img.shields.io/badge/License-EUPL--1.2-blue.svg)](https://eupl.eu/1.2/en/)


`dappco.re/go/build` is the build, release, SDK, Apple packaging, and workflow toolkit behind `core build`, `core release`, `core sdk`, `core ci`, and the public `dAppCore/build@v3` GitHub Action surface.

## What It Covers

- Project discovery across Go, Wails, Node/Deno frontend, PHP, Python, Rust, Docs, Docker, LinuxKit, C++, and Taskfile projects
- Cross-platform artifact builds with archive and checksum generation
- Eight release publishers: GitHub, Docker, npm, Homebrew, Scoop, AUR, Chocolatey, and LinuxKit
- macOS Apple pipeline: `core build apple`, codesign, notarisation, DMG packaging, TestFlight/App Store submission, Info.plist and entitlements generation, Xcode Cloud scripts
- OpenAPI SDK generation for TypeScript, Python, Go, and PHP with `oasdiff` breaking-change checks
- Reusable release workflow generation via `core build workflow`

## Action/Workflow Parity

The generated reusable workflow mirrors the `dAppCore/build@v3` action architecture:

- Auto-detects stack markers including subtree frontend manifests and MkDocs projects
- Exposes the same discovery, option-computation, and setup-planning shape in Go through `DiscoverFull`, `ComputeOptions`, `ComputeSetupPlan`, and the reusable `Pipeline` gateway
- Installs Go, Node, Python, Conan, MkDocs, Deno, and Wails when required, plus frontend package dependencies and optional garble for obfuscated builds
- Applies distro-aware Linux WebKit dependencies for Wails builds
- Supports obfuscation, NSIS packaging, WebView2 modes, Deno frontend overrides, and build cache restore/save
- Exposes the action-style control surface: `core-version`, `go-version`, `node-version`, `wails-version`, `version`, `build`, `sign`, `package`, `build-name`, `build-platform`, `build-tags`, `build-obfuscate`, `nsis`, `deno-build`, `wails-build-webview2`, `build-cache`, and `archive-format`
- Uploads workflow artifacts using the action-style naming shape: `{build-name}_{os}_{arch}_{tag|shortsha}`

## Commands

```bash
core build
core build apple
core build workflow
core release
core release --apple-testflight
core sdk
core build sdk
core ci
```

## Module

```go
import "dappco.re/go/build/pkg/build"
```

The repository is a library/command-registration module. It does not ship its own standalone binary.
