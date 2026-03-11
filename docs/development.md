---
title: Development
description: Building, testing, and contributing to go-build.
---

# Development

## Prerequisites

- **Go 1.26+** (the module declares `go 1.26.0`)
- **Go workspace** -- this module is part of the workspace at `~/Code/go.work`. After cloning, run `go work sync` to ensure local replacements resolve correctly.
- `GOPRIVATE=forge.lthn.ai/*` must be set for private module fetching.

## Building

```bash
cd /Users/snider/Code/core/go-build
go build ./...
```

There is no standalone binary produced by this repository. The `cmd/` packages register CLI commands that are compiled into the `core` binary from `forge.lthn.ai/core/cli`.

To build the full CLI with these commands included:

```bash
cd /Users/snider/Code/core/cli
core build          # or: go build -o bin/core ./cmd/core
```

## Running Tests

```bash
go test ./...
```

To run a single test by name:

```bash
go test ./pkg/build/... -run TestLoadConfig_Good
go test ./pkg/release/... -run TestIncrementVersion
go test ./pkg/sdk/... -run TestDiff
```

To run tests with race detection:

```bash
go test -race ./...
```

### Test Naming Convention

Tests follow the `_Good`, `_Bad`, `_Ugly` suffix pattern used across the Core ecosystem:

- `_Good` -- Happy-path tests. Valid inputs produce expected outputs.
- `_Bad` -- Expected error conditions. Invalid inputs are handled gracefully.
- `_Ugly` -- Edge cases, panics, and boundary conditions.

Example:

```go
func TestLoadConfig_Good(t *testing.T) {
    // Valid .core/build.yaml is loaded correctly
}

func TestLoadConfig_Bad(t *testing.T) {
    // Malformed YAML returns a parse error
}

func TestChecksum_Ugly(t *testing.T) {
    // Empty artifact path returns an error
}
```

### Test Helpers

Tests use `t.TempDir()` for filesystem isolation and `io.Local` as the medium:

```go
func setupConfigTestDir(t *testing.T, configContent string) string {
    t.Helper()
    dir := t.TempDir()
    if configContent != "" {
        coreDir := filepath.Join(dir, ConfigDir)
        err := os.MkdirAll(coreDir, 0755)
        require.NoError(t, err)
        err = os.WriteFile(
            filepath.Join(coreDir, ConfigFileName),
            []byte(configContent), 0644,
        )
        require.NoError(t, err)
    }
    return dir
}
```

### Testing Libraries

- **testify** (`assert` and `require`) for assertions.
- `io.Local` from `forge.lthn.ai/core/go-io` as the filesystem medium.

## Code Style

- **UK English** in comments and user-facing strings (colour, organisation, centre, notarisation).
- **Strict types** -- all parameters and return types are explicitly typed.
- **Error format** -- use `fmt.Errorf("package.Function: descriptive message: %w", err)` for wrapped errors.
- **PSR-style** formatting via `gofmt` / `goimports`.

## Adding a New Builder

1. Create `pkg/build/builders/mybuilder.go` implementing `build.Builder`:

```go
type MyBuilder struct{}

func NewMyBuilder() *MyBuilder { return &MyBuilder{} }

func (b *MyBuilder) Name() string { return "mybuilder" }

func (b *MyBuilder) Detect(fs io.Medium, dir string) (bool, error) {
    return fs.IsFile(filepath.Join(dir, "mymarker.toml")), nil
}

func (b *MyBuilder) Build(ctx context.Context, cfg *build.Config, targets []build.Target) ([]build.Artifact, error) {
    // Build logic here
    return artifacts, nil
}

var _ build.Builder = (*MyBuilder)(nil)  // Compile-time check
```

2. Add the builder to the `getBuilder()` switch in both `cmd/build/cmd_project.go` and `pkg/release/release.go`.

3. Optionally add a `ProjectType` constant and marker to `pkg/build/build.go` and `pkg/build/discovery.go` if the new type should participate in auto-discovery.

4. Write tests in `pkg/build/builders/mybuilder_test.go` following the `_Good`/`_Bad`/`_Ugly` pattern.

## Adding a New Publisher

1. Create `pkg/release/publishers/mypub.go` implementing `publishers.Publisher`:

```go
type MyPublisher struct{}

func NewMyPublisher() *MyPublisher { return &MyPublisher{} }

func (p *MyPublisher) Name() string { return "mypub" }

func (p *MyPublisher) Publish(ctx context.Context, release *Release, pubCfg PublisherConfig, relCfg ReleaseConfig, dryRun bool) error {
    if dryRun {
        // Print what would happen
        return nil
    }
    // Publish logic here
    return nil
}
```

2. Add the publisher to the `getPublisher()` switch in `pkg/release/release.go`.

3. Add any publisher-specific fields to `PublisherConfig` in `pkg/release/config.go` and map them in `buildExtendedConfig()` in `pkg/release/release.go`.

4. Write tests in `pkg/release/publishers/mypub_test.go`.

## Adding a New SDK Generator

1. Create `pkg/sdk/generators/mylang.go` implementing `generators.Generator`:

```go
type MyLangGenerator struct{}

func NewMyLangGenerator() *MyLangGenerator { return &MyLangGenerator{} }

func (g *MyLangGenerator) Language() string { return "mylang" }

func (g *MyLangGenerator) Available() bool {
    _, err := exec.LookPath("mylang-codegen")
    return err == nil
}

func (g *MyLangGenerator) Install() string {
    return "pip install mylang-codegen"
}

func (g *MyLangGenerator) Generate(ctx context.Context, opts Options) error {
    // Try native, then Docker fallback
    return nil
}
```

2. Register it in `pkg/sdk/sdk.go` inside `GenerateLanguage()`:

```go
registry.Register(generators.NewMyLangGenerator())
```

3. Write tests in `pkg/sdk/generators/mylang_test.go`.

## Directory Conventions

- **`pkg/`** -- Library code. Importable by other modules.
- **`cmd/`** -- CLI command registration. Each subdirectory registers commands via `cli.RegisterCommands()` in an `init()` function. These packages are imported by the CLI binary.
- **`.core/`** -- Per-project configuration directory (not part of this repository; created in consumer projects).

## Commit Guidelines

Follow conventional commits:

```
type(scope): description
```

Types: `feat`, `fix`, `perf`, `refactor`, `docs`, `style`, `test`, `build`, `ci`, `chore`.

Include the co-author trailer:

```
Co-Authored-By: Virgil <virgil@lethean.io>
```

## Licence

EUPL-1.2. See `LICENSE` in the repository root.
