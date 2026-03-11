---
title: go-build
description: Build system, release pipeline, and SDK generation for the Core ecosystem.
---

# go-build

`forge.lthn.ai/core/go-build` is the build, release, and SDK generation toolkit for Core projects. It provides:

- **Auto-detecting builders** for Go, Wails, Docker, LinuxKit, C++, and Taskfile projects
- **Cross-compilation** with per-target archiving (tar.gz, tar.xz, zip) and SHA-256 checksums
- **Code signing** -- macOS codesign with notarisation, GPG detached signatures, Windows signtool (placeholder)
- **Release automation** -- semantic versioning from git tags, conventional-commit changelogs, multi-target publishing
- **SDK generation** -- OpenAPI spec diffing for breaking-change detection, code generation for TypeScript, Python, Go, and PHP
- **CLI integration** -- registers `core build`, `core ci`, and `core sdk` commands via the Core CLI framework

## Module Path

```
forge.lthn.ai/core/go-build
```

Requires **Go 1.26+**.

## Quick Start

### Build a project

From any project directory containing a recognised marker file:

```bash
core build                          # Auto-detect type, build for configured targets
core build --targets linux/amd64    # Single target
core build --ci                     # JSON output for CI pipelines
core build --verbose                # Detailed step-by-step output
```

The builder is chosen by marker-file priority:

| Marker file       | Builder    |
|-------------------|------------|
| `wails.json`      | Wails      |
| `go.mod`          | Go         |
| `package.json`    | Node (stub)|
| `composer.json`   | PHP (stub) |
| `CMakeLists.txt`  | C++        |
| `Dockerfile`      | Docker     |
| `linuxkit.yml`    | LinuxKit   |
| `Taskfile.yml`    | Taskfile   |

### Release artifacts

```bash
core build release --we-are-go-for-launch   # Build + archive + checksum + publish
core build release                           # Dry-run (default without the flag)
core build release --draft --prerelease      # Mark as draft pre-release
```

### Publish pre-built artifacts

After `core build` has populated `dist/`:

```bash
core ci                              # Dry-run publish from dist/
core ci --we-are-go-for-launch       # Actually publish
core ci --version v1.2.3             # Override version
```

### Generate changelogs

```bash
core ci changelog                    # From latest tag to HEAD
core ci changelog --from v0.1.0 --to v0.2.0
core ci version                      # Show determined next version
core ci init                         # Scaffold .core/release.yaml
```

### SDK operations

```bash
core build sdk                       # Generate SDKs for all configured languages
core build sdk --lang typescript     # Single language
core sdk diff --base v1.0.0 --spec api/openapi.yaml   # Breaking-change check
core sdk validate                    # Validate OpenAPI spec
```

## Package Layout

```
forge.lthn.ai/core/go-build/
|
|-- cmd/
|   |-- build/          CLI commands for `core build` (build, from-path, pwa, sdk, release)
|   |-- ci/             CLI commands for `core ci` (init, changelog, version, publish)
|   +-- sdk/            CLI commands for `core sdk` (diff, validate)
|
+-- pkg/
    |-- build/              Core build types, config loading, discovery, archiving, checksums
    |   |-- builders/       Builder implementations (Go, Wails, Docker, LinuxKit, C++, Taskfile)
    |   +-- signing/        Code-signing implementations (macOS codesign, GPG, Windows stub)
    |
    |-- release/            Release orchestration, versioning, changelog, config
    |   +-- publishers/     Publisher implementations (GitHub, Docker, npm, Homebrew, Scoop, AUR, Chocolatey, LinuxKit)
    |
    +-- sdk/                OpenAPI SDK generation and breaking-change diffing
        +-- generators/     Language generators (TypeScript, Python, Go, PHP)
```

## Configuration Files

Build and release behaviour is driven by two YAML files in the `.core/` directory.

### `.core/build.yaml`

Controls compilation targets, flags, and signing:

```yaml
version: 1
project:
  name: myapp
  description: My application
  main: ./cmd/myapp
  binary: myapp
build:
  cgo: false
  flags: ["-trimpath"]
  ldflags: ["-s", "-w"]
  env: []
targets:
  - os: linux
    arch: amd64
  - os: linux
    arch: arm64
  - os: darwin
    arch: arm64
  - os: windows
    arch: amd64
sign:
  enabled: true
  gpg:
    key: $GPG_KEY_ID
  macos:
    identity: $CODESIGN_IDENTITY
    notarize: false
    apple_id: $APPLE_ID
    team_id: $APPLE_TEAM_ID
    app_password: $APPLE_APP_PASSWORD
```

When no `.core/build.yaml` exists, sensible defaults apply (CGO off, `-trimpath -s -w`, four standard targets).

### `.core/release.yaml`

Controls versioning, changelog filtering, publishers, and SDK generation:

```yaml
version: 1
project:
  name: myapp
  repository: owner/repo
build:
  targets:
    - os: linux
      arch: amd64
    - os: darwin
      arch: arm64
publishers:
  - type: github
    draft: false
    prerelease: false
  - type: homebrew
    tap: owner/homebrew-tap
  - type: docker
    registry: ghcr.io
    image: owner/myapp
    tags: ["latest", "{{.Version}}"]
changelog:
  include: [feat, fix, perf, refactor]
  exclude: [chore, docs, style, test, ci]
sdk:
  spec: api/openapi.yaml
  languages: [typescript, python, go, php]
  output: sdk
  diff:
    enabled: true
    fail_on_breaking: false
```

## Dependencies

| Dependency | Purpose |
|---|---|
| `forge.lthn.ai/core/cli` | CLI command registration and TUI styling |
| `forge.lthn.ai/core/go-io` | Filesystem abstraction (`io.Medium`, `io.Local`) |
| `forge.lthn.ai/core/go-i18n` | Internationalised CLI labels |
| `forge.lthn.ai/core/go-log` | Structured error logging |
| `github.com/Snider/Borg` | XZ compression for tar.xz archives |
| `github.com/getkin/kin-openapi` | OpenAPI spec loading and validation |
| `github.com/oasdiff/oasdiff` | OpenAPI diff and breaking-change detection |
| `gopkg.in/yaml.v3` | YAML config parsing |
| `github.com/leaanthony/debme` | Embedded filesystem anchoring (PWA templates) |
| `github.com/leaanthony/gosod` | Template extraction for PWA builds |
| `golang.org/x/net` | HTML parsing for PWA manifest detection |
| `golang.org/x/text` | Changelog section title casing |

## Licence

EUPL-1.2
