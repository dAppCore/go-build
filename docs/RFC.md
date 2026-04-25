---
module: dappco.re/go/build
repo: core/go-build
lang: go
tier: lib
depends:
  - code/core/go
tags:
  - build
  - compilation
  - release
  - artifacts
  - packaging
---
# Core Build & Release RFC — Build automation and code signing

> The authoritative spec for `core build`, `core release`, code signing, SDK generation, and installer scripts.
> An agent should be able to implement any component from this document alone.

**Repository:** `core/go-build` (build/release), `core/cli` (CLI commands)
**Module:** `dappco.re/go/build`
**Sub-specs:** [Models](RFC.models.md) | [Commands](RFC.commands.md) | [Build Pipeline](RFC.build-pipeline.md) | [Release Pipeline](RFC.release-pipeline.md) | [SDK Generation](RFC.sdk-generation.md) | [API Provider](RFC.api-provider.md) | [CI Workflow](RFC.ci-workflow.md) | [cmd build](RFC.cmd-build.md) | [cmd sdk](RFC.cmd-sdk.md) | [Action Port](RFC.action-port.md)

---

## 1. Overview

The build system dogfoods Core to build itself. Two design modes:

- **Without `.core/`:** Simple passthrough — `core build` = `go build .`
- **With `.core/`:** Full orchestration — matrix builds, releases, signing, SDK generation

### 1.1 Design Principle

Nobody remembers long arg strings. `.core/build.yaml` remembers them for you.

---

## 2. Command Structure

### 2.1 Without Config (Passthrough)

```bash
core build                        # = go build .
core build --output ./bin         # = go build -o ./bin .
core build --targets linux/amd64  # = GOOS=linux GOARCH=amd64 go build .
```

### 2.2 With Config (Orchestrated)

```bash
core build              # .core/build.yaml → matrix build
core release            # .core/release.yaml → build + archive + publish
core release --publish  # explicit publish to GitHub/Forge
core release --tag v1.0 # specify tag
core release --draft    # create as draft
core release --apple-testflight # build Apple target and upload to TestFlight
```

---

## 3. Configuration

### 3.1 .core/build.yaml

```yaml
version: 1

project:
  name: core
  description: "Core CLI"
  main: ./cmd/core
  binary: core

build:
  type: go                          # Override auto-detected type
  cgo: false                        # Enable/disable CGO (Go only)
  obfuscate: false                  # Use garble for binary obfuscation (Go only)
  nsis: false                       # Generate Windows NSIS installer (Wails only)
  webview2: download                # WebView2 delivery: download|embed|browser|error (Wails only)
  flags: ["-trimpath"]              # Build flags
  ldflags: ["-s", "-w", "-X main.Version={{.Tag}}"]  # Linker flags
  build_tags: [tag1, tag2]          # Go build tags
  archive_format: gz                # Archive format: gz (default), xz, zip
  env: ["CGO_ENABLED=0"]            # Environment variables
  dockerfile: Dockerfile            # Path to Dockerfile (Docker only)
  registry: ghcr.io                 # Container registry (Docker)
  image: owner/repo                 # Image name (Docker)
  tags: [latest, v1.0]              # Docker image tags
  build_args:                       # Docker build arguments
    ARG1: value1
  push: false                       # Push Docker image after build
  load: false                       # Load image into local daemon (single-platform builds)
  linuxkit_config: linuxkit.yml     # Path to LinuxKit config
  formats: [iso, qcow2, raw]        # LinuxKit output formats: iso, raw, qcow2, vmdk, vhd, gcp, aws, docker, tar, kernel+initrd

cache:
  enabled: false                    # Enable build cache setup
  dir: .core/cache                  # Cache metadata directory
  key_prefix: my-project            # Cache key prefix
  paths:                            # Cache directories to create/setup
    - ~/.cache/go-build
    - ~/go/pkg/mod
  restore_keys:                     # Fallback cache key prefixes
    - go-

targets:
  - os: linux
    arch: amd64
  - os: linux
    arch: arm64
  - os: darwin
    arch: amd64
  - os: darwin
    arch: arm64
  - os: windows
    arch: amd64

sign:
  enabled: true
  gpg:
    key: $GPG_KEY_ID
  macos:
    identity: "Developer ID Application: Lethean CIC (TEAM_ID)"
    notarize: false
    apple_id: $APPLE_ID
    team_id: $APPLE_TEAM_ID
    app_password: $APPLE_APP_PASSWORD
  windows:
    signtool: false

sdk:
  spec: openapi.yaml        # or auto-detect
  languages: [typescript, python, go, php]
  output: sdk/
  diff: true                 # breaking change detection
```

### 3.2 .core/release.yaml

```yaml
version: 1

project:
  name: core
  repository: dappcore/core

build:
  targets:
    - os: linux
      arch: amd64
    - os: darwin
      arch: arm64
  archive_format: gz        # Archive format: gz (default), xz, zip

changelog:
  use: conventional         # Commit convention: conventional, semver
  exclude:                  # Exclude commit types/patterns
    - '^docs:'
    - '^test:'
    - '^ci:'

sdk:
  spec: openapi.yaml        # Path to OpenAPI spec or auto-detect
  languages:
    - typescript
    - python
    - go
    - php
  output: sdk/
  diff: true                # Breaking change detection

publishers:
  - type: github
    draft: false
    prerelease: false       # Auto-detect from semver (alpha/beta/rc)

  - type: npm
    package: "@dappcore/core"
    access: public

  - type: homebrew
    tap: dappcore/homebrew-tap
    formula: core
    official:               # Generate files for official Homebrew repo PR
      enabled: false
      output: dist/homebrew

  - type: docker
    registry: ghcr.io
    image: dappcore/core
    tags: [latest, v{{.Version}}]
    build_args:
      VERSION: "{{.Version}}"
    platforms:
      - linux/amd64
      - linux/arm64

  - type: linuxkit
    config: linuxkit.yml
    formats: [iso, qcow2, raw, aws, gcp]
    platforms:
      - linux/amd64
      - linux/arm64

  - type: aur
    maintainer: "Name <email@example.com>"

  - type: scoop
    bucket: dappcore/scoop-bucket
    official:               # Generate files for official Scoop repo PR
      enabled: false
      output: dist/scoop

  - type: chocolatey
    push: false             # Generate only, don't push

checksum:
  algorithm: sha256
  file: checksums.txt
```

---

## 4. Build Pipeline

```
core build
  |
  +-- .core/build.yaml exists?
  |     |
  |     +-- Yes: load config, iterate targets, build each
  |     +-- No:  simple `go build .` (passthrough)
  |
  +-- output binaries to dist/
```

### 4.0 Supported Project Types

The build system auto-detects project types and applies type-specific builders. Multi-type projects (e.g., Wails + Go) iterate all detected types.

| Project Type | Marker Files | Builder |
|-------------|--------------|---------|
| **Go** | go.mod, go.work | GoBuilder — `go build` with matrix targets |
| **Wails** | wails.json | WailsBuilder — desktop app with WebView2/macOS/Windows support |
| **Node** | package.json | NodeBuilder — npm/yarn with cross-platform bundling |
| **PHP** | composer.json | PHPBuilder — Composer + code generation |
| **Python** | pyproject.toml, requirements.txt | PythonBuilder — pip/poetry with wheel packaging |
| **C++** | CMakeLists.txt | CPPBuilder — CMake + Conan with cross-compile profiles |
| **Rust** | Cargo.toml | RustBuilder — cargo build |
| **Taskfile** | Taskfile.yml, Taskfile.yaml, Taskfile | TaskfileBuilder — Task runner automation |
| **Docker** | Dockerfile | DockerBuilder — Docker buildx with multi-arch |
| **LinuxKit** | linuxkit.yml, .core/linuxkit/*.yml | LinuxKitBuilder — VM image generation (iso, qcow2, raw, aws, gcp) |
| **Docs** | mkdocs.yml, mkdocs.yaml, docs/mkdocs.yml | DocsBuilder — Static site generation with bundling |

### 4.1 Package Structure

```
go-build/
  pkg/build/
    config.go              # Config loading (.core/build.yaml, .core/release.yaml)
    discovery.go           # Project type detection (Go, Wails, Node, PHP, Python, Rust, C++, Docker, LinuxKit, Taskfile, Docs)
    ci.go                  # CI environment detection (GitHub Actions)
    workflow.go            # Release workflow generation + template embedding
    cache.go               # Build cache configuration
    builders/
      go.go                # Go builder
      wails.go             # Wails desktop application builder
      node.go              # Node.js/npm builder
      docker.go            # Docker/OCI builder
      linuxkit.go          # LinuxKit VM builder
      linuxkit_image.go    # Immutable image spec, base resolution, Apple Container + OCI output
      php.go               # PHP builder (Composer)
      python.go            # Python builder (pyproject.toml, requirements.txt)
      cpp.go               # C++ builder (CMake + Conan)
      rust.go              # Rust builder (Cargo)
      taskfile.go          # Taskfile task runner builder
      docs.go              # Documentation builder (mkdocs)
      zip_deterministic.go # Deterministic ZIP archive support
    images/
      core-dev.yml         # LinuxKit YAML for core-dev base image
      core-ml.yml          # LinuxKit YAML for core-ml base image
      core-minimal.yml     # LinuxKit YAML for core-minimal base image
    signing/
      signer.go            # Signer interface + SignConfig
      gpg.go               # GPG checksums signing
      codesign.go          # macOS codesign + notarize
      signtool.go          # Windows Authenticode signing (signtool)
  pkg/release/
    release.go             # Archive + checksums + multi-platform publish
    config.go              # Release configuration + SDK settings
    changelog.go           # Changelog generation
    publishers/
      github.go            # GitHub Releases publisher
      npm.go               # npm package publisher
      homebrew.go          # Homebrew formula publisher
      docker.go            # Docker image publisher
      linuxkit.go          # LinuxKit image publisher
      aur.go               # Arch User Repository publisher
      scoop.go             # Scoop bucket publisher
      chocolatey.go        # Chocolatey package publisher
  pkg/sdk/
    sdk.go                 # SDK generation orchestration
    detect.go              # OpenAPI spec detection
    diff.go                # Breaking change detection (oasdiff)
    generators/
      generator.go         # Generator interface
      typescript.go        # TypeScript SDK generator
      python.go            # Python SDK generator
      go.go                # Go SDK generator
      php.go               # PHP SDK generator
  pkg/api/
    provider.go            # REST API provider wrapper + WebSocket support
  cmd/build/
    cmd_project.go         # Main project build orchestration
    cmd_pwa.go             # PWA building (local path or URL)
    cmd_workflow.go        # Release workflow file generation
    cmd_build.go           # build command
    cmd_image.go           # `core build image` — immutable LinuxKit image building
    cmd_release.go         # release command
    cmd_sdk.go             # SDK commands
    cmd_commands.go        # Utility commands
```

---

## 5. Release Pipeline

```
core release
  → load .core/release.yaml
  → build (uses .core/build.yaml)
  → sign macOS binaries (codesign)
  → notarise if enabled (wait for Apple)
  → create archives (tar.gz/zip)
  → generate CHECKSUMS.txt + GPG sign
  → publish to configured targets
```

`core release --apple-testflight` and `core release --target apple-testflight` bypass `.core/release.yaml` and run the Apple build pipeline with TestFlight upload enabled. The Apple pipeline still loads `.core/build.yaml`, writes Xcode Cloud helper scripts when configured, signs with the configured Apple distribution identity, and uploads via `xcrun altool --upload-app`.

### 5.1 Publisher Interface

```go
type Publisher interface {
    Name() string
    Validate(ctx context.Context, cfg *Config) error
    Publish(ctx context.Context, release *Release) error
    Supports(target string) bool
}
```

Package structure: `pkg/release/publishers/` with one file per platform + `templates/` for generated files.

### 5.2 Release Targets

**Philosophy:** Make paywalled release features "just a feature". GoReleaser Pro charges $165/yr for npm, Chocolatey, AUR. OSS developers shouldn't pay to distribute free software.

#### Tier 1 — Must Have

| Platform | Publisher | Method | Notes |
|----------|-----------|--------|-------|
| **GitHub Releases** | GitHubPublisher | go-github | Foundation — artifacts + checksums, all others reference these |
| **npm** | NpmPublisher | npm CLI + binary wrapper | `@dappcore/core` — installs correct binary per platform |
| **Homebrew** | HomebrewPublisher | Formula + tap PR | `dappcore/homebrew-tap` — generates formula for official repo PR |

#### Tier 2 — High Impact

| Platform | Publisher | Method | Notes |
|----------|-----------|--------|-------|
| **Docker** | DockerPublisher | buildx multi-arch | `ghcr.io/dappcore/core` — push to registry or load locally |
| **AUR** | AURPublisher | PKGBUILD template | `core-bin` package with official repo PR generation |
| **Scoop** | ScoopPublisher | JSON manifest | `dappcore/scoop-bucket` — generates manifest for official repo PR |
| **Chocolatey** | ChocolateyPublisher | NuSpec + push | Optional push to Chocolatey (false = generate only) |
| **LinuxKit** | LinuxKitPublisher | LinuxKit CLI | ISO, qcow2, raw, vmdk, vhd, aws, gcp outputs |

#### Tier 3 (Future)

Snapcraft, Flatpak, Fury.io, WinGet, APT/YUM repos.

### 5.3 Official Repository Configuration

When publishing to official package repositories (Homebrew, Scoop, AUR), use the `official` block to generate PR-ready files instead of publishing directly:

```yaml
publishers:
  - type: homebrew
    tap: dappcore/homebrew-tap
    formula: core
    official:
      enabled: true
      output: dist/homebrew     # Generated files for official Homebrew repo PR

  - type: scoop
    bucket: dappcore/scoop-bucket
    official:
      enabled: true
      output: dist/scoop        # Generated files for official Scoop repo PR

  - type: aur
    maintainer: "Name <email>"
    # AUR uses PKGBUILD template generation for official submissions
```

Official configurations generate the necessary files (formulas, manifests, PKGBUILDs) without pushing directly to official repositories.

---

## 6. Code Signing

### 6.1 Architecture

```
Build binaries
    |
Sign macOS binaries (codesign --sign --timestamp --options runtime)
    |
Notarise if enabled (xcrun notarytool submit --wait → xcrun stapler staple)
    |
Create archives (tar.gz / zip)
    |
Generate CHECKSUMS.txt (SHA-256)
    |
GPG sign CHECKSUMS.txt → CHECKSUMS.txt.asc
```

### 6.2 Signer Interface

```go
type Signer interface {
    Name() string
    Available() bool
    Sign(ctx context.Context, path string) error
}
```

### 6.3 Implementations

| Signer | Platform | What It Signs | Output |
|--------|----------|---------------|--------|
| GPG | All | CHECKSUMS.txt | CHECKSUMS.txt.asc (detached ASCII armour) |
| macOS codesign | darwin | Binary files | In-place signature + hardened runtime |
| macOS notarytool | darwin | Binaries (via zip) | Stapled notarisation ticket |
| Windows signtool | windows | Binaries | Authenticode signature |

### 6.4 User Verification

```bash
gpg --verify CHECKSUMS.txt.asc CHECKSUMS.txt
sha256sum -c CHECKSUMS.txt
```

### 6.5 CLI Flags

```bash
core build                # Sign with defaults (GPG + codesign if configured)
core build --no-sign      # Skip all signing
core build --notarize     # Enable macOS notarisation (overrides config)
```

### 6.6 Environment Variables

| Variable | Purpose |
|----------|---------|
| `GPG_KEY_ID` | GPG key fingerprint |
| `CODESIGN_IDENTITY` | macOS Developer ID (fallback) |
| `APPLE_ID` | Apple account email |
| `APPLE_TEAM_ID` | Apple Developer Team ID |
| `APPLE_APP_PASSWORD` | App-specific password for notarisation |

---

## 7. SDK Generation

### 7.1 Overview

Generate typed API clients from OpenAPI specs. Hybrid approach: native generators where available, Docker openapi-generator fallback.

### 7.2 Detection Flow

```
1. Check .core/build.yaml sdk.spec field
2. Scan common paths: openapi.yaml, openapi.json, docs/openapi.yaml
3. Try Laravel Scramble (php artisan scramble:export)
4. Fail if no spec found
```

### 7.3 Generators

| Language | Tool | Native? |
|----------|------|---------|
| TypeScript | openapi-typescript-codegen | Yes |
| Python | openapi-python-client | Yes |
| Go | oapi-codegen | Yes |
| PHP | openapi-generator (Docker) | No |

### 7.4 Breaking Change Detection

Uses `oasdiff` library to compare current spec against previously generated spec:

```bash
core sdk diff                    # Show breaking changes
core sdk diff --fail-on-warn     # Exit 1 on warnings too
core sdk generate                # Generate all configured SDKs
core sdk generate --lang ts      # Generate TypeScript only
```

### 7.5 Output Structure

```
sdk/
  typescript/
    package.json
    src/
  python/
    pyproject.toml
    src/
  go/
    go.mod
    client.go
  php/
    composer.json
    src/
```

### 7.6 Release Integration

```bash
core release --target sdk                    # Generate SDKs only
core release --target sdk --version v1.2.3   # Explicit version
core release --target sdk --dry-run          # Preview
```

SDK version matches release version from git tags.

### 7.7 Package Structure

```
pkg/sdk/
  sdk.go              # Main SDK type, orchestration
  detect.go           # OpenAPI spec detection
  diff.go             # Breaking change detection (oasdiff)
  generators/
    generator.go      # Generator interface
    typescript.go     # openapi-typescript-codegen
    python.go         # openapi-python-client
    go.go             # oapi-codegen
    php.go            # openapi-generator (Docker)
  templates/          # Package scaffolding templates
    typescript/
    python/
    go/
    php/
```

---

## 8. Project Detection & Multi-Type Projects

### 8.1 Auto-Detection

The build system auto-detects project types based on marker files and directory structure. Projects can combine multiple types (e.g., a Wails project is detected as both Wails + Go).

```
core build
  → Discover project types
  → Iterate all detected types
  → Run type-specific builder for each
```

### 8.2 Detection Order

Detection checks in priority order:

1. **Wails** (wails.json) — if present, always checked first
2. **Go** (go.mod, go.work)
3. **Node** (package.json, subtree npm detection)
4. **PHP** (composer.json)
5. **Python** (pyproject.toml, requirements.txt)
6. **Rust** (Cargo.toml)
7. **C++** (CMakeLists.txt)
8. **Docker** (Dockerfile)
9. **LinuxKit** (linuxkit.yml, .core/linuxkit/*.yml)
10. **Taskfile** (Taskfile.yml, Taskfile.yaml, Taskfile)
11. **Docs** (mkdocs.yml, docs/mkdocs.yml)

### 8.3 Override Detection

To skip auto-detection and force a specific type:

```yaml
# .core/build.yaml
build:
  type: go  # Forces Go builder, skips detection
```

---

## 9. PWA & Legacy GUI Building

### 9.1 Local PWA Build

Build a local Progressive Web App directory into a desktop application:

```bash
core build pwa --path ./my-pwa
```

### 9.2 Live PWA Build

Download a PWA from a URL and package it:

```bash
core build pwa --url https://example.com/app
```

The builder:
1. Fetches the HTML entry point
2. Downloads all linked assets (CSS, JS, images, manifest)
3. Extracts metadata (title, description, icons)
4. Packages assets locally
5. Invokes the main build pipeline

---

## 9.3 LinuxKit Immutable Images

go-build produces LinuxKit images for Apple Containers and Docker. The image is the controlled environment — agents work in a known OS with enforced toolchains, not whatever the host happens to have installed.

### 9.3.1 Design

The rootfs is immutable. The OS layer, installed packages, and toolchain versions are defined by the image spec at build time. Runtime cannot modify the base — only the mounted `/workspace` volume is read-write. This eliminates environment drift between agent runs and guarantees reproducible builds across all dispatch hosts.

### 9.3.2 Base Images

| Base | Contents | Use Case |
|------|----------|----------|
| `core-dev` | Go toolchain, git, task, core CLI, linters | Standard agent dispatch (code generation, AX sweeps) |
| `core-ml` | Go toolchain, MLX framework, model loaders | ML inference tasks (go-mlx inside container) |
| `core-minimal` | Go toolchain only | Lightweight builds, CI runners |

### 9.3.3 Image Spec

```go
// Build an immutable LinuxKit image for agent dispatch
//
//   image := build.LinuxKit(
//       build.WithBase("core-dev"),
//       build.WithPackages("git", "task"),
//       build.WithMount("/workspace"),
//       build.WithGPU(true),
//   )
func LinuxKit(opts ...LinuxKitOption) *LinuxKitImage {
    cfg := &LinuxKitConfig{
        Base:     "core-dev",
        Packages: []string{},
        Mounts:   []string{"/workspace"},
        GPU:      false,
    }
    for _, opt := range opts {
        opt(cfg)
    }
    return &LinuxKitImage{Config: cfg}
}

// LinuxKitConfig defines an immutable container image.
type LinuxKitConfig struct {
    Base     string   // base image: core-dev, core-ml, core-minimal
    Packages []string // additional OS packages
    Mounts   []string // volume mount points (read-write)
    GPU      bool     // Metal passthrough support (Apple) or NVIDIA (Docker)
}
```

### 9.3.4 Output Formats

LinuxKit images build to multiple output formats from a single spec:

| Format | Target Runtime | Extension |
|--------|---------------|-----------|
| OCI image | Docker, Podman | `.tar` (OCI bundle) |
| Apple Container image | Apple Containers (macOS 26+) | `.aci` |
| Raw disk | QEMU, bare metal | `.raw` |
| ISO | Boot media, CI runners | `.iso` |

The OCI format is Docker/Podman compatible — the same image runs on Linux hosts, CI systems, and macOS Docker Desktop. The Apple Container format is macOS-native with hardware VM isolation and sub-second startup.

### 9.3.5 CLI

```bash
core build image core-dev              # Build the core-dev image (default formats: OCI + Apple)
core build image core-ml               # Build the ML-capable image
core build image core-minimal          # Build the minimal image
core build image core-dev --format oci # Build OCI format only
core build image --list                # List available base images and their versions
core build image --rebuild             # Force rebuild (ignores cache)
```

### 9.3.6 Versioning

Images are versioned alongside Core releases. The image tag matches the Core version that built it — `core-dev:0.8.0` is built by Core v0.8.0. When Core updates, `core build image` rebuilds with the new toolchain. Old images are retained for rollback.

### 9.3.7 Configuration

```yaml
# .core/build.yaml — LinuxKit image section
linuxkit:
  base: core-dev
  packages:
    - git
    - task
    - gopls
  mounts:
    - /workspace
  gpu: false
  formats: [oci, apple]               # Output formats
  registry: ghcr.io/dappcore          # Push OCI images to registry
```

### 9.3.8 Package Structure

```
go-build/
  pkg/build/
    builders/
      linuxkit.go          # LinuxKit builder (existing)
      linuxkit_image.go    # Immutable image spec, base resolution, format output
    images/
      core-dev.yml         # LinuxKit YAML for core-dev base
      core-ml.yml          # LinuxKit YAML for core-ml base
      core-minimal.yml     # LinuxKit YAML for core-minimal base
  cmd/build/
    cmd_image.go           # `core build image` command
```

---

## 10. CI Integration & Workflow Generation

### 10.1 GitHub Actions Environment Detection

The build system detects GitHub Actions environment and provides:

```go
ci := build.DetectCI()
if ci != nil {
    // Inside GitHub Actions
    ci.Tag       // "v1.2.3" (if triggered by tag)
    ci.SHA       // Full commit hash
    ci.ShortSHA  // First 7 chars
    ci.Ref       // Full git ref
    ci.IsTag     // Boolean tag detection
    ci.Repo      // "owner/repo"
    ci.Owner     // "owner"
}
```

### 10.2 Release Workflow Generation

Generate a GitHub Actions workflow file for releases on tags:

```bash
core build workflow --output .github/workflows/
```

Generates `.github/workflows/release.yml` with:
- Trigger on version tags (`v*.*.*`)
- Checkout, build, sign, and publish steps
- Matrix builds for multiple platforms
- Automatic changelog generation
- Multi-target publishing

Workflow path resolution supports:
- Directory: `core build workflow --output ci` → `./ci/release.yml`
- File: `core build workflow --output ci/release.yml`
- Absolute: `core build workflow --output /tmp/release.yml`
- Default: `core build workflow` → `.github/workflows/release.yml`

### 10.3 GitHub Annotations

Format build messages as GitHub Actions annotations:

```go
s := build.FormatGitHubAnnotation("error", "main.go", 42, "undefined: foo")
// → "::error file=main.go,line=42::undefined: foo"
```

Support levels: error, warning, notice, debug

---

## 11. REST API Provider

### 11.1 Overview

The `pkg/api/BuildProvider` wraps build, release, and SDK operations as a REST API with WebSocket event streaming.

Implements:
- `Provider` — route group registration
- `Streamable` — WebSocket event emission
- `Describable` — OpenAPI documentation
- `Renderable` — UI element specification

### 11.2 Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/api/v1/build` | Trigger build |
| POST | `/api/v1/build/release` | Trigger release |
| POST | `/api/v1/build/sdk` | Generate SDKs |
| WS | `/api/v1/build/events` | Subscribe to build events |

### 11.3 Usage

```go
hub := ws.NewHub()
p := api.NewProvider(".", hub)
router.Use(p.Register(router, hub))
// Routes registered at /api/v1/build/*
```

---

## 12. Installer Scripts

Hosted at `https://lthn.sh/`

| Script | Variant | Usage |
|--------|---------|-------|
| `setup.sh` | full | Default install |
| `ci.sh` | ci | Minimal CI builds |
| `php.sh` | php | PHP development |
| `go.sh` | go | Go development |
| `agent.sh` | agentic | AI agent variant |
| `dev.sh` | dev | Multi-repo dev |

```bash
curl -sL https://lthn.sh/setup.sh | bash
```

---

## 14. Service & Daemon

```bash
core service install    # Register with OS service manager
core service start      # Start daemon
core service stop       # Stop daemon
core service uninstall  # Remove registration
core service export     # Dump native config (systemd/launchd/etc)
```

Daemon runs Core runtime with:
- File watcher (auto-rebuild)
- API server (IDE/remote control)
- Task scheduler (periodic jobs)
- MCP server (AI tool integration)
- Agentic (AI agent orchestration)

Implementation: kardianos/service for abstraction + native config export.

---

## 15. Implementation Priority

### Core Build Features

| Feature | Description | Status |
|---------|-------------|--------|
| Cross-compile matrix | Build from `.core/build.yaml` target definitions | ✓ |
| Project type detection | Auto-detect Go, Wails, Node, PHP, Python, Rust, C++, Docker, LinuxKit, Taskfile, Docs | ✓ |
| Go builder | Standard Go cross-compile | ✓ |
| Wails builder | Desktop app with WebView2/macOS/Windows support | ✓ |
| Node builder | npm/yarn with bundling | ✓ |
| PHP builder | Composer package handling | ✓ |
| Python builder | pip/poetry wheel support | ✓ |
| Rust builder | Cargo build system | ✓ |
| C++ builder | CMake + Conan cross-compile | ✓ |
| Docker builder | Multi-arch Docker images with buildx | ✓ |
| LinuxKit builder | VM image generation (iso, qcow2, raw, aws, gcp, vmdk, vhd) | ✓ |
| Taskfile builder | Task runner automation | ✓ |
| Docs builder | mkdocs static site generation | ✓ |

### Archive & Packaging

| Feature | Description | Status |
|---------|-------------|--------|
| Archive creation | tar.gz, xz, zip packaging | ✓ |
| Deterministic ZIP | Reproducible ZIP archives | ✓ |
| CHECKSUMS.txt generation | SHA-256 checksums for artifacts | ✓ |
| Build caching | Cache configuration with restore keys | ✓ |

### Code Signing

| Feature | Description | Status |
|---------|-------------|--------|
| Signer interface | Pluggable signing abstraction | ✓ |
| GPG signing | Detached ASCII armour signature for checksums | ✓ |
| macOS codesign | Code signing with Developer ID + hardened runtime | ✓ |
| macOS notarisation | notarytool submit + stapler staple | ✓ |
| Windows signtool | Authenticode signing via signtool | ✓ |
| CLI signing flags | `--no-sign`, `--notarize` overrides | ✓ |

### Release Publishing

| Feature | Description | Status |
|---------|-------------|--------|
| GitHub Releases | go-github publisher | ✓ |
| npm publisher | Binary wrapper + metadata | ✓ |
| Homebrew | Formula generation + tap PR | ✓ |
| Docker | Multi-arch image push | ✓ |
| AUR | PKGBUILD template generation | ✓ |
| Scoop | JSON manifest generation | ✓ |
| Chocolatey | NuSpec generation + optional push | ✓ |
| LinuxKit | Image publishing | ✓ |
| Official repo generation | Homebrew/Scoop official repo PR files | ✓ |

### SDK Generation

| Feature | Description | Status |
|---------|-------------|--------|
| OpenAPI spec detection | Auto-scan or manual spec path | ✓ |
| Breaking change detection | oasdiff comparison | ✓ |
| TypeScript generator | openapi-typescript-codegen | ✓ |
| Python generator | openapi-python-client | ✓ |
| Go generator | oapi-codegen | ✓ |
| PHP generator | openapi-generator (Docker) | ✓ |
| SDK CLI commands | `core sdk diff`, `core sdk generate` | ✓ |
| SDK release integration | `core release --target sdk` | ✓ |

### Installer Scripts

| Feature | Description | Status |
|---------|-------------|--------|
| setup.sh | Full installer with PATH + completions | ✓ |
| ci.sh | Minimal CI-only installer | ✓ |
| php.sh | PHP development variant | ✓ |
| go.sh | Go development variant | ✓ |
| agent.sh | AI agent variant | ✓ |
| dev.sh | Multi-repo development variant | ✓ |
| CDN hosting | lthn.sh script distribution | ✓ |

### CI & Workflow

| Feature | Description | Status |
|---------|-------------|--------|
| GitHub Actions detection | CIContext with SHA, tag, ref, repo | ✓ |
| GitHub Annotations | Format build messages as annotations | ✓ |
| Release workflow generation | Embedded template output to .github/workflows/ | ✓ |
| Workflow path resolution | Directory/file path handling | ✓ |

### Advanced Features

| Feature | Description | Status |
|---------|-------------|--------|
| PWA building | Local path or URL download + packaging | ✓ |
| REST API provider | Build operations as HTTP endpoints | ✓ |
| WebSocket events | Real-time build event streaming | ✓ |
| kardianos/service | Cross-platform daemon service | ✓ |
| File watcher | Auto-rebuild on source changes | ✓ |
| API server | IDE/remote control interface | ✓ |
| `core service` commands | install, start, stop, uninstall, export | ✓ |

### Build Configuration Features

| Feature | Description | Status |
|---------|-------------|--------|
| Type override | Force builder via `.core/build.yaml` | ✓ |
| CGO control | Enable/disable CGO (Go) | ✓ |
| Binary obfuscation | garble integration for Go | ✓ |
| Build tags | Go build tag support | ✓ |
| NSIS installer | Windows installer generation (Wails) | ✓ |
| WebView2 delivery | download\|embed\|browser\|error modes (Wails) | ✓ |
| Docker load | Load image into local daemon | ✓ |
| LinuxKit formats | 9 output formats (iso, raw, qcow2, vmdk, vhd, aws, gcp, docker, tar, kernel+initrd) | ✓ |
| Immutable base images | core-dev, core-ml, core-minimal LinuxKit images for agent dispatch | ✓ |
| Apple Container output | Native macOS 26 container image format (.aci) | ✓ |
| `core build image` | CLI command to build/rebuild immutable images | ✓ |
| Image versioning | Images tagged with Core release version | ✓ |

---

## 16. I/O Medium

Build artifacts are stored and retrieved via `io.Medium` — output to local filesystem, S3, or DataCube. See `code/core/go/io/RFC.md §Medium` for the interface.

```go
build.Run(build.WithOutput(io.S3("releases.lthn.io/v0.8.0")))
```

---

## 17. Reference Material

| Resource | Location |
|----------|----------|
| go-build repo | `dappco.re/go/build` |
| Core Go RFC | `code/core/go/RFC.md` |
| I/O Medium interface | `code/core/go/io/RFC.md` |
| Build pipeline sub-spec | `code/core/go/build/RFC.build-pipeline.md` |
| Release pipeline sub-spec | `code/core/go/build/RFC.release-pipeline.md` |
| SDK generation sub-spec | `code/core/go/build/RFC.sdk-generation.md` |

---

## Changelog

| Date | Change |
|------|--------|
| 2026-04-08 | Added §9.3 LinuxKit Immutable Images — base images (core-dev, core-ml, core-minimal), Apple Container + OCI output formats, `core build image` CLI, versioned alongside Core releases |
| 2026-04-08 | Added §16 I/O Medium (build artifacts via io.Medium) |
