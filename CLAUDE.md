# CLAUDE.md

Agent guidance for `core/go-build`.

## Project Overview

`dappco.re/go/build` is a command-registration and library module for:

- `core build`
- `core build apple`
- `core build workflow`
- `core build sdk`
- `core ci`
- `core sdk`

It also carries the reusable release workflow template that mirrors the public `dAppCore/build@v3` action surface.

## Build and Test

```bash
go build ./...
go test ./...
go test ./pkg/build/... -run TestWorkflow_WriteReleaseWorkflow_Good
go test ./pkg/build/... -run TestApple_
```

## Repo Layout

```
core/go-build/
├── go/                                ← Go module root (module dappco.re/go/build)
│   ├── cmd/                           ← CLI entry points
│   ├── internal/                      ← Go internal packages
│   ├── pkg/                           ← Go library packages
│   ├── tests/                         ← Go tests/fixtures
│   │   └── cli/
│   ├── go.mod
│   ├── go.sum
│   ├── CLAUDE.md                      ← symlink to root CLAUDE.md
│   ├── README.md                      ← symlink to root README.md
│   ├── AGENTS.md                      ← symlink to root AGENTS.md
│   └── docs                           ← symlink to root docs/
├── docs/                              ← cross-language docs (symlinked into go/)
├── locales/                           ← locale content
├── ui/                                ← language-specific UI
├── README.md
├── CLAUDE.md
├── AGENTS.md
└── ...
```

Future language siblings are expected at repo root (`php/`, `ts/`, `py/`) while Go stays in `go/`.

## Go Resolution Modes

This repo is intentionally non-workspace: a single Go module under `go/`.

| Mode | When | What runs |
|------|------|-----------|
| **Local module mode** | Standard local commands from repo root via `cd go` | Uses `go/ go.mod` and cached dependencies in module mode. |
| **`GOWORK=off`** | CI and reproducible verification | Uses `go/` module graph directly, without workspace indirection. |

```bash
cd go
go mod tidy
GOWORK=off GOFLAGS=-mod=mod go test -count=1 -short ./...
```

## Main Packages

- `pkg/build/`: discovery, config loading, caches, checksums, archives, workflow generation, Apple implementation
- `pkg/build/builders/`: Go, Wails, Node, PHP, Python, Rust, Docs, Docker, LinuxKit, C++, Taskfile
- `pkg/build/apple/`: RFC-facing Apple wrapper that exposes `core.Result`
- `pkg/build/signing/`: GPG, macOS codesign/notarisation, Windows signtool
- `pkg/release/`: versioning, changelogs, orchestration
- `pkg/release/publishers/`: GitHub, Docker, npm, Homebrew, Scoop, AUR, Chocolatey, LinuxKit
- `pkg/sdk/`: OpenAPI detection, diffing, generation

## Important Behaviour

- Discovery is richer than simple marker lookup: it handles subtree frontends, MkDocs roots, distro-aware Linux package hints, and action-facing stack suggestions
- The generated release workflow must stay aligned with the action-style inputs: `build-name`, `build-platform`, `build-tags`, `build-obfuscate`, `nsis`, `deno-build`, `wails-build-webview2`, and `build-cache`
- Workflow artifact naming is expected to follow `{build-name}_{os}_{arch}_{tag|shortsha}`
- Apple support includes universal builds, notarisation, DMG creation, Xcode Cloud script generation, TestFlight, and App Store submission

## Coding Standards

- Use `coreerr.E("package.Function", "message", err)` for wrapped errors
- Prefer UK English in user-facing strings and comments
- Keep tests in `testify` style with `_Good`, `_Bad`, and `_Ugly` naming
- Preserve env expansion support in config models and signing/apple credentials

## Extension Points

- New builder: add the implementation in `pkg/build/builders/`, register the project type in discovery/resolution, and add coverage in command and release paths
- New workflow input: update the template, workflow tests, and any CLI alias plumbing together
- New Apple capability: update both `pkg/build/apple.go` and the RFC-facing wrapper in `pkg/build/apple/`
