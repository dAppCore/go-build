# CLAUDE.md

## Project Overview

`core/go-build` is the build system, release pipeline, and SDK generation tool. Three main packages: build (cross-compilation, archiving, signing), release (versioning, changelog, publishers), and sdk (OpenAPI diff, code generation).

## Build & Development

```bash
go test ./...
go build ./...
```

## Architecture

- `build/` — Build config (.core/build.yaml), project discovery, archiving, checksums
- `build/builders/` — Go, Wails, Docker, LinuxKit, C++, Taskfile builders
- `build/signing/` — Code signing (macOS notarisation, GPG)
- `build/buildcmd/` — CLI commands for `core build`
- `release/` — Versioning, changelog generation, release orchestration
- `release/publishers/` — GitHub, Homebrew, Scoop, AUR, npm, Docker, Chocolatey
- `sdk/` — OpenAPI spec diffing, SDK code generation
- `cmd/ci/` — CI/release pipeline commands
- `cmd/sdk/` — SDK validation commands

## Dependencies

- `cli` — Command registration
- `go-io` — File utilities
- `go-i18n` — Internationalisation
- `go-log` — Structured logging
- Borg — Compression
- kin-openapi, oasdiff — OpenAPI tooling

## Coding Standards

- UK English, strict types, testify, EUPL-1.2
