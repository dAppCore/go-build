# Build UI Service Provider + Lit Custom Elements

**Date**: 14 March 2026
**Module**: `forge.lthn.ai/core/go-build`
**Pattern**: Follows `go-scm/pkg/api/` provider + `go-scm/ui/` Lit elements

## Overview

Add a service provider (`BuildProvider`) that exposes the existing build, release, and SDK subsystems as REST endpoints, plus a set of Lit custom elements for GUI display within the Core IDE.

The provider wraps existing functions — no business logic is reimplemented.

## Architecture

```
go-build/
├── pkg/api/
│   ├── provider.go      # BuildProvider (Provider + Streamable + Describable + Renderable)
│   ├── embed.go          # //go:embed for UI assets
│   └── ui/dist/          # Built JS bundle (populated by npm run build)
├── pkg/api/
│   └── provider_test.go  # Identity + endpoint tests
└── ui/
    ├── package.json
    ├── tsconfig.json
    ├── vite.config.ts
    ├── index.html         # Demo page
    └── src/
        ├── index.ts               # Bundle entry
        ├── build-panel.ts         # <core-build-panel> — tabs container
        ├── build-config.ts        # <core-build-config> — .core/build.yaml + discovery
        ├── build-artifacts.ts     # <core-build-artifacts> — dist/ contents
        ├── build-release.ts       # <core-build-release> — version, changelog, publish
        ├── build-sdk.ts           # <core-build-sdk> — OpenAPI diff, SDK generation
        └── shared/
            ├── api.ts             # Typed fetch wrapper for /api/v1/build/*
            └── events.ts          # WS event connection for build.* channels
```

## REST Endpoints

All under `/api/v1/build`:

| Method | Path                | Handler          | Wraps                              |
|--------|---------------------|------------------|------------------------------------|
| GET    | /config             | getConfig        | `build.LoadConfig(io.Local, cwd)`  |
| GET    | /discover           | discoverProject  | `build.Discover(io.Local, cwd)`    |
| POST   | /build              | triggerBuild     | Full build pipeline                |
| GET    | /artifacts          | listArtifacts    | Scan `dist/` directory             |
| GET    | /release/version    | getVersion       | `release.DetermineVersion(cwd)`    |
| GET    | /release/changelog  | getChangelog     | `release.Generate(cwd, "", "")`    |
| POST   | /release            | triggerRelease   | `release.Run()` or `Publish()`     |
| GET    | /sdk/diff           | getSdkDiff       | `sdk.Diff(base, revision)`         |
| POST   | /sdk/generate       | generateSdk      | `sdk.SDK.Generate(ctx)`            |

## WS Channels

- `build.started` — Build commenced (includes project type, targets)
- `build.complete` — Build finished (includes artifact list)
- `build.failed` — Build error (includes error message)
- `release.started` — Release pipeline started
- `release.complete` — Release published
- `sdk.generated` — SDK generation complete

## Custom Elements

### `<core-build-panel>` (build-panel.ts)
Top-level tabbed container with tabs: Config, Build, Release, SDK.
Follows HLCRF layout from go-scm.

### `<core-build-config>` (build-config.ts)
- Displays `.core/build.yaml` fields (project name, binary, targets, flags, signing)
- Shows detected project type from discovery
- Read-only display of current configuration

### `<core-build-artifacts>` (build-artifacts.ts)
- Lists files in `dist/` with size, checksum status
- "Build" button with confirmation dialogue (POST /build is destructive)
- Real-time progress via WS events

### `<core-build-release>` (build-release.ts)
- Current version from git tags
- Changelog preview (rendered markdown)
- Publisher targets from `.core/release.yaml`
- "Release" button with confirmation dialogue

### `<core-build-sdk>` (build-sdk.ts)
- OpenAPI diff results (breaking/non-breaking changes)
- SDK generation controls (language selection)
- Generation status

## Dependencies Added to go.mod

```
forge.lthn.ai/core/api v0.1.0
forge.lthn.ai/core/go-ws v0.1.0
github.com/gin-gonic/gin v1.11.0
```

## Safety Considerations

- POST /build and POST /release are destructive operations
- UI elements include confirmation dialogues before triggering
- Provider accepts a `projectDir` parameter (defaults to CWD)
- Build/release operations run synchronously; WS events provide progress

## Implementation Tasks

1. Create `pkg/api/provider.go` — BuildProvider struct + all handlers
2. Create `pkg/api/provider_test.go` — Identity + config/discover tests
3. Create `pkg/api/embed.go` — //go:embed directive
4. Create `ui/` directory with full Lit element suite
5. Update `go.mod` — add api, go-ws, gin dependencies
6. Build UI (`cd ui && npm install && npm run build`)
7. Verify Go compilation (`go build ./...`)
