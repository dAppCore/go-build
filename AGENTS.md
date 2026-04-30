# go-build Repository Guide

This repository contains the Go implementation of the dAppCore build,
release, SDK, API, and service tooling. The module path is
`dappco.re/go/build`; public packages are intended to be consumed by other
Core projects, while `cmd/*` packages wire those capabilities into CLI
commands.

The repository is organized around a small set of product areas:

- `cmd/build`, `cmd/ci`, `cmd/sdk`, and `cmd/service` register CLI commands
  against `dappco.re/go` command primitives.
- `pkg/build` owns project discovery, build configuration, builders,
  artifact packaging, checksums, installer scripts, workflows, Apple
  packaging, LinuxKit images, and Xcode Cloud helpers.
- `pkg/release` owns release configuration, version and changelog resolution,
  release orchestration, SDK release orchestration, and publisher handoff.
- `pkg/release/publishers` contains the concrete publisher implementations
  for GitHub, Docker, Homebrew, Scoop, Chocolatey, AUR, NPM, and LinuxKit.
- `pkg/sdk` and `pkg/sdk/generators` handle OpenAPI detection, validation,
  diffing, and language-specific SDK generation.
- `pkg/api` exposes the build provider surface for the Core API runtime.
- `pkg/service` runs the local build daemon, exports service manager
  configuration, and bridges daemon events into API, websocket, MCP, and
  agentic channels.
- `internal/*` contains shared implementation support that is not public API:
  Core-compatible filesystem/process wrappers, command option helpers,
  service command request parsing, project detection glue, SDK config loading,
  workflow assertions, and legacy assertion helpers.

Keep tests beside the source file they exercise. Public functions and methods
use the AX-7 triplet shape `Test<File>_<Symbol>_{Good,Bad,Ugly}` in the
matching `<file>_test.go`; usage examples live in matching
`<file>_example_test.go` files and print through `dappco.re/go` helpers.
Avoid new monolithic test files because the compliance audit is file-aware.

Do not import banned standard-library convenience packages directly in Go
files. Use the Core wrappers from `dappco.re/go` and the compatibility modules
under `.compat/` when a legacy package path is still required by consumers.
The audit gate treats tests, examples, CLI code, internal packages, and
compatibility shims the same way.

Before handing work off, run the repository checks from the root with
`GOWORK=off`: `go mod tidy`, `go vet ./...`, `go test -count=1 ./...`,
`gofmt -l .`, and the v0.9.0 compliance audit.
