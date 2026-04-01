---
title: Architecture
description: Internal design of go-build -- types, data flow, and extension points.
---

# Architecture

go-build is organised into three independent subsystems that share common types: **build**, **release**, and **sdk**. The CLI layer in `cmd/` wires them together but the library packages under `pkg/` can be used programmatically without the CLI.

## Build Subsystem

### Project Discovery

`build.Discover()` scans a directory for marker files and returns detected project types in priority order (most specific first). For example, a Wails project contains both `wails.json` and `go.mod`, so `Discover` returns `[wails, go]`. `PrimaryType()` returns only the first match.

Detection order:

1. `wails.json` -- `ProjectTypeWails`
2. `go.mod` -- `ProjectTypeGo`
3. `package.json` -- `ProjectTypeNode`
4. `composer.json` -- `ProjectTypePHP`
5. `mkdocs.yml` -- `ProjectTypeDocs`

Docker (`Dockerfile`), LinuxKit (`linuxkit.yml` or `.core/linuxkit/*.yml`), C++ (`CMakeLists.txt`), and Taskfile (`Taskfile.yml`) are detected by their respective builders' `Detect()` methods rather than the central discovery function.

### Builder Interface

Every builder implements:

```go
type Builder interface {
    Name() string
    Detect(fs io.Medium, dir string) (bool, error)
    Build(ctx context.Context, cfg *Config, targets []Target) ([]Artifact, error)
}
```

The `Config` struct carries runtime parameters (filesystem medium, project directory, output directory, binary name, version, linker flags) plus Docker- and LinuxKit-specific fields.

`Build()` returns a slice of `Artifact` values, each recording the output path, target OS, and target architecture:

```go
type Artifact struct {
    Path     string
    OS       string
    Arch     string
    Checksum string
}
```

### Builder Implementations

| Builder | Detection | Strategy |
|---|---|---|
| **GoBuilder** | `go.mod` or `wails.json` | Sets `GOOS`/`GOARCH`/`CGO_ENABLED=0`, runs `go build -trimpath` with ldflags. Output per target: `dist/{os}_{arch}/{binary}`. |
| **WailsBuilder** | `wails.json` | Checks `go.mod` for Wails v3 vs v2. V3 delegates to TaskfileBuilder; V2 runs `wails build -platform` then copies from `build/bin/` to `dist/`. |
| **NodeBuilder** | `package.json` | Detects the active package manager from lockfiles, runs the build script once per target, and collects artifacts from `dist/{os}_{arch}/`. |
| **PHPBuilder** | `composer.json` | Runs `composer install`, then `composer run-script build` when present. Falls back to a deterministic zip bundle in `dist/{os}_{arch}/`. |
| **PythonBuilder** | `pyproject.toml` or `requirements.txt` | Packages the project tree into a deterministic zip bundle in `dist/{os}_{arch}/`. |
| **RustBuilder** | `Cargo.toml` | Runs `cargo build --release --target` per platform and collects executables from `target/{triple}/release/`. |
| **DocsBuilder** | `mkdocs.yml` | Runs `mkdocs build --clean --site-dir` and packages the generated `site/` tree into a zip bundle per target. |
| **DockerBuilder** | `Dockerfile` | Validates `docker` and `buildx`, builds multi-platform images with `docker buildx build --platform`. Supports `--push` or local load/OCI tarball. |
| **LinuxKitBuilder** | `linuxkit.yml` or `.core/linuxkit/*.yml` | Validates `linuxkit` CLI, runs `linuxkit build --format --name --dir --arch`. Outputs qcow2, iso, raw, vmdk, vhd, or cloud images. Linux-only targets. |
| **CPPBuilder** | `CMakeLists.txt` | Validates `make`, runs `make configure` then `make build` then `make package` for host builds. Cross-compilation uses Conan profile targets (e.g. `make gcc-linux-armv8`). Finds artifacts in `build/packages/` or `build/release/src/`. |
| **TaskfileBuilder** | `Taskfile.yml` / `Taskfile.yaml` / `Taskfile` | Validates `task` CLI, runs `task build` with `GOOS`, `GOARCH`, `OUTPUT_DIR`, `NAME`, `VERSION` as both env vars and task vars. Discovers artifacts by platform subdirectory or filename pattern. |

### Post-Build Pipeline

After building, the CLI orchestrates three optional steps:

1. **Signing** -- `signing.SignBinaries()` codesigns darwin artifacts with hardened runtime. `signing.NotarizeBinaries()` submits to Apple via `xcrun notarytool` and staples. `signing.SignChecksums()` creates GPG detached signatures (`.asc`).

2. **Archiving** -- `build.ArchiveAll()` (or `ArchiveAllXZ()`) wraps each artifact. Linux/macOS get `tar.gz` (or `tar.xz`); Windows gets `zip`. XZ compression uses the Borg library. Archive filenames follow the pattern `{binary}_{os}_{arch}.tar.gz`.

3. **Checksums** -- `build.ChecksumAll()` computes SHA-256 for each archive. `build.WriteChecksumFile()` writes a sorted `CHECKSUMS.txt` in the standard `sha256  filename` format.

### Signing Architecture

The `Signer` interface:

```go
type Signer interface {
    Name() string
    Available() bool
    Sign(ctx context.Context, fs io.Medium, path string) error
}
```

Three implementations:

- **GPGSigner** -- `gpg --detach-sign --armor --local-user {key}`. Produces `.asc` files.
- **MacOSSigner** -- `codesign --sign {identity} --timestamp --options runtime --force`. Notarisation via `xcrun notarytool submit --wait` then `xcrun stapler staple`.
- **WindowsSigner** -- Uses `signtool` on Windows when a certificate is configured.

Configuration supports `$ENV` expansion in all credential fields, so secrets can come from environment variables without being written to YAML.

### Configuration Loading

`build.LoadConfig(fs, dir)` reads `.core/build.yaml`. If the file is missing, `DefaultConfig()` provides:

- Version 1 format
- Main package: `.`
- Flags: `["-trimpath"]`
- LDFlags: `["-s", "-w"]`
- CGO: disabled
- Targets: `linux/amd64`, `linux/arm64`, `darwin/arm64`, `windows/amd64`
- Signing: enabled, credentials from environment

Fields present in the YAML override defaults; omitted fields inherit defaults via `applyDefaults()`.

### Filesystem Abstraction

All file operations go through `io.Medium` from `forge.lthn.ai/core/go-io`. Production code uses `io.Local` (real filesystem); tests can inject mock mediums. This makes builders unit-testable without touching the real filesystem for detection and configuration loading.

---

## Release Subsystem

### Version Resolution

`release.DetermineVersion(dir)` resolves the release version:

1. If HEAD has an exact git tag, use it.
2. If there is a previous tag, increment its patch number (e.g. `v1.2.3` becomes `v1.2.4`).
3. If no tags exist, default to `v0.0.1`.

Helper functions `IncrementMinor()` and `IncrementMajor()` are available for manual version bumps. `ParseVersion()` decomposes a semver string into major, minor, patch, pre-release, and build components. `CompareVersions()` returns -1, 0, or 1.

All versions are normalised to include a `v` prefix.

### Changelog Generation

`release.Generate(dir, fromRef, toRef)` parses git history between two refs and produces grouped Markdown.

Commits are parsed against the conventional-commit regex:

```
^(\w+)(?:\(([^)]+)\))?(!)?:\s*(.+)$
```

This matches patterns like `feat: add feature`, `fix(scope): fix bug`, or `feat!: breaking change`.

Parsed commits are grouped by type and rendered in a fixed order: breaking changes first, then features, bug fixes, performance, refactoring, and so on. Each entry includes the optional scope (bolded) and the short commit hash.

`GenerateWithConfig()` adds include/exclude filtering by commit type, driven by the `changelog` section in `.core/release.yaml`.

### Release Orchestration

Two entry points:

- **`release.Run()`** -- Full pipeline: determine version, generate changelog, build artifacts (via the build subsystem), archive, checksum, then publish to all configured targets.
- **`release.Publish()`** -- Publish-only: expects pre-built artifacts in `dist/`, generates changelog, then publishes. This supports the separated `core build` then `core ci` workflow.

Both accept a `dryRun` parameter. When true, publishers print what would happen without executing.

### Publisher Interface

```go
type Publisher interface {
    Name() string
    Publish(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig, dryRun bool) error
}
```

Eight publishers are implemented:

| Publisher | Mechanism |
|---|---|
| **GitHub** | `gh release create` via the GitHub CLI. Auto-detects repository from git remote. Uploads all artifacts as release assets. |
| **Docker** | `docker buildx build` with multi-platform support. Pushes to configured registry with version tags. |
| **npm** | `npm publish` with configurable access level and package name. |
| **Homebrew** | Generates a Ruby formula file. Optionally targets an official tap repository. |
| **Scoop** | Generates a JSON manifest for a Scoop bucket. |
| **AUR** | Generates a PKGBUILD file for the Arch User Repository. |
| **Chocolatey** | Generates a `.nuspec` and `chocolateyinstall.ps1`. Optionally pushes via `choco push`. |
| **LinuxKit** | Builds LinuxKit VM images in specified formats and uploads them as release assets. |

Publisher-specific configuration (registry, tap, bucket, image, etc.) is carried in `PublisherConfig` fields and mapped to an `Extended` map at runtime.

### SDK Release Integration

`release.RunSDK()` handles SDK-specific releases: it runs a breaking-change diff (if enabled), then generates SDKs via the SDK subsystem. This can be wired into a CI pipeline to auto-generate client libraries on each release.

---

## SDK Subsystem

### Spec Detection

`sdk.DetectSpec()` locates the OpenAPI specification:

1. If a path is configured in `.core/release.yaml` under `sdk.spec`, use it.
2. Check common paths: `api/openapi.yaml`, `api/openapi.json`, `openapi.yaml`, `openapi.json`, `docs/api.yaml`, `docs/api.json`, `swagger.yaml`, `swagger.json`.
3. Check for Laravel Scramble in `composer.json` (export not yet implemented).

### Breaking-Change Detection

`sdk.Diff(basePath, revisionPath)` loads two OpenAPI specs via `kin-openapi`, computes a structural diff via `oasdiff`, and runs the `oasdiff/checker` backward-compatibility checks at error level. Returns a `DiffResult` with a boolean `Breaking` flag, a list of change descriptions, and a human-readable summary.

`DiffExitCode()` maps results to CI exit codes: 0 (clean), 1 (breaking changes), 2 (error).

### Code Generation

The `generators.Generator` interface:

```go
type Generator interface {
    Language() string
    Generate(ctx context.Context, opts Options) error
    Available() bool
    Install() string
}
```

Generators are held in a `Registry` and looked up by language identifier. Each generator tries three strategies in order:

1. **Native tool** -- e.g. `oapi-codegen` for Go, `openapi-typescript-codegen` for TypeScript.
2. **npx** -- Falls back to `npx` invocation where applicable (TypeScript).
3. **Docker** -- Uses the `openapitools/openapi-generator-cli` image as a last resort.

| Language | Native Tool | Docker Generator |
|---|---|---|
| TypeScript | `openapi-typescript-codegen` or `npx` | `typescript-fetch` |
| Python | `openapi-python-client` | `python` |
| Go | `oapi-codegen` | `go` |
| PHP | `openapi-generator-cli` via Docker | `php` |

On Unix systems, Docker containers run with `--user {uid}:{gid}` to match host file ownership.

---

## Data Flow Summary

```
.core/build.yaml -----> LoadConfig() -----> BuildConfig
                                                |
project directory ----> Discover() -----------> ProjectType
                                                |
                                          getBuilder()
                                                |
                                           Builder.Build()
                                                |
                                          []Artifact (raw binaries)
                                                |
                         +----------------------+---------------------+
                         |                      |                     |
                    SignBinaries()         ArchiveAll()          (optional)
                         |                      |              NotarizeBinaries()
                         |               []Artifact (archives)
                         |                      |
                         |               ChecksumAll()
                         |                      |
                         |            []Artifact (with checksums)
                         |                      |
                         |            WriteChecksumFile()
                         |                      |
                         +----------+-----------+
                                    |
                           SignChecksums() (GPG)
                                    |
                             Publisher.Publish()
                                    |
                    GitHub / Docker / npm / Homebrew / ...
```
