---
title: Stacks
description: Stack-specific behaviour for the generated build workflow and go-build builders.
---

# Stacks

The generated workflow and `pkg/build/builders` cover multiple project stacks. Detection starts with repository markers and then hands off to the matching builder.

## Wails

Wails is the primary desktop-app path and the closest match to the public `dAppCore/build@v3` action.

- Detection accepts `wails.json` directly and also Go roots that contain frontend manifests at the root, under `frontend/`, or in a subtree up to depth 2.
- Setup installs Go, Node, frontend dependencies, the Wails CLI, distro-specific Linux WebKit packages, and optional garble when obfuscation is enabled.
- Build uses `wails build` for Wails v2 and supports Wails v3 through Taskfile-driven packaging or a direct `wails3` CLI fallback.
- Windows packaging supports NSIS and the `download`, `embed`, `browser`, and `error` WebView2 modes.
- Signing supports the existing macOS and Windows signing flows exposed by the build and Apple layers.

## Node and Deno

Node-style frontend builds are also first-class:

- Detection accepts `package.json`, `deno.json`, and `deno.jsonc` at the root or in nested frontend directories.
- Setup installs Node dependencies with the detected package manager and enables Deno when `DENO_ENABLE`, `DENO_BUILD`, or Deno manifests are present.
- Build uses `DENO_BUILD` when supplied, otherwise defaults to `deno task build` for Deno projects or the package-manager `build` script for Node projects.

## Docs

Docs projects are treated as a dedicated stack instead of falling through to Node:

- Detection accepts `mkdocs.yml` and `mkdocs.yaml` at the root or under `docs/`.
- Setup installs Python plus MkDocs only when docs markers are present.
- Build runs `mkdocs build` and packages the generated site as an archiveable artifact.

## C++

C++ projects map cleanly onto the action's Conan-oriented setup story:

- Detection uses `CMakeLists.txt`.
- Setup installs Python and Conan in the generated workflow when a C++ marker is present.
- Build uses CMake and Conan profile mapping for native and cross-target builds.

## PHP

PHP projects now get the workflow setup they need:

- Detection uses `composer.json`.
- Setup verifies PHP is available and installs Composer on demand when the runner does not already provide it.
- Build runs Composer-backed dependency installation and optionally a Composer `build` script before bundling artifacts.

## Rust

Rust projects also have explicit workflow setup:

- Detection uses `Cargo.toml`.
- Setup verifies Cargo is available and bootstraps Rust with `rustup` when the runner image does not already include it.
- Build uses `cargo build --release --target ...` and collects target-specific binaries.

## Docker, LinuxKit, and Taskfile

Additional stacks are exposed through dedicated builders:

- Docker uses Buildx-backed image builds and archive-friendly export modes.
- LinuxKit supports root manifests and `.core/linuxkit/*.yml` configs.
- Taskfile acts as a generic wrapper for repositories that already encode their own build graph, including many Wails v3 projects.
- Setup installs the Task CLI when a Taskfile marker is present so Wails v3 Taskfile builds work in generated CI without extra bootstrapping.

## Future Direction

The action docs historically called out Wails v3 and C++ as future or placeholder stacks. In go-build those paths now exist, but the design still follows the same principle: keep discovery generic, keep setup conditional, and let each stack wrapper own its full pipeline.
