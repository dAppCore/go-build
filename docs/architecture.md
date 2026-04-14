---
title: Architecture
description: Internal design of the build, action/workflow, Apple, release, and SDK layers.
---

# Architecture

go-build has four major surfaces that share types but can be used independently:

1. `pkg/build` for discovery, builders, artifacts, caches, workflow generation, and Apple packaging
2. `pkg/release` for versioning, changelogs, and publishers
3. `pkg/sdk` for OpenAPI diffing and SDK generation
4. `cmd/` for registering `core build`, `core ci`, and `core sdk`

## Discovery and Stack Suggestion

`build.Discover()` and `build.DiscoverFull()` implement the action-style discovery pass.

They record:

- detected project types in priority order
- raw marker presence
- whether a frontend exists at the root, in `frontend/`, or in a subtree up to depth 2
- distro-aware Linux package requirements
- an action-facing stack suggestion (`wails2`, `cpp`, `docs`, `node`, `go`)

Important detection behaviour:

- Wails detection accepts `wails.json` and also Go roots that contain frontend manifests
- Docs detection accepts `mkdocs.yml` and `mkdocs.yaml` in the root or `docs/`
- Docker detection accepts `Dockerfile` and `Containerfile` variants
- LinuxKit detection accepts root manifests and `.core/linuxkit/*.yml`
- Taskfile detection accepts common case variants

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
| PHP | Composer-backed builds with deterministic zip fallback |
| Python | Deterministic source bundle packaging |
| Rust | Cargo release builds by target triple |
| Docs | MkDocs build plus zipped site output |
| Docker | Buildx-backed image builds with push/load/archive modes |
| LinuxKit | LinuxKit image generation in configured formats |
| C++ | Make + Conan orchestration with profile-based cross-builds |
| Taskfile | Generic task-backed build pipeline used heavily by Wails v3 projects |

## Generated GitHub Workflow

`core build workflow` writes `.github/workflows/release.yml`. The generated workflow mirrors the modular `dAppCore/build@v3` action pipeline:

1. Checkout
2. Discovery by file markers through `hashFiles(...)`
3. Toolchain setup for Go, Node, Python, Conan, MkDocs, Deno, and Wails
4. Linux distro-aware WebKit dependency setup for Wails
5. Cache restore under `.core/cache` and `cache/`
6. `core build --archive --checksum`
7. Artifact upload with action-style names: `{build-name}_{os}_{arch}_{tag|shortsha}`
8. Release publishing through `core ci`

The workflow keeps the action inputs exposed at the CLI layer:

- `build-name`
- `build-platform`
- `build-tags`
- `build-obfuscate`
- `nsis`
- `deno-build`
- `wails-build-webview2`
- `build-cache`

## Apple Pipeline

The Apple implementation lives in `pkg/build/apple.go`, with an RFC-facing wrapper in `pkg/build/apple/`.

Key pieces:

- `AppleOptions` for the runtime pipeline
- `BuildApple()` for the end-to-end macOS build flow
- `BuildWailsApp()` and `CreateUniversal()` for architecture-specific and universal app bundles
- `Sign()`, `Notarise()`, `CreateDMG()`, `UploadTestFlight()`, and `SubmitAppStore()` for post-build delivery
- generated `Info.plist` and entitlements
- Xcode Cloud script generation from `.core/build.yaml`

`cmd/build/cmd_apple.go` wires this into `core build apple`.

## Release Layer

`pkg/release` owns:

- semver version resolution from git tags
- changelog generation from conventional commits
- building or reusing `dist/` artifacts
- checksum and artifact metadata handling
- publisher orchestration

Publishers currently cover GitHub, Docker, npm, Homebrew, Scoop, AUR, Chocolatey, and LinuxKit.

## SDK Layer

`pkg/sdk` detects an OpenAPI spec, validates it, compares revisions with `oasdiff`, and generates SDKs for:

- TypeScript
- Python
- Go
- PHP

Generators prefer native tooling first and fall back to `npx` or Docker where appropriate.
