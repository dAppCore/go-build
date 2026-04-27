---
title: Stacks
description: Stack-specific behaviour for detection, setup planning, builders, and the generated GitHub workflow.
---

# Stacks

The public action historically centred on Wails v2, and go-build keeps that as the default desktop-app path while carrying the same architecture across Wails, frontend, docs, C++, container, and language-native stacks.

## Wails v2

Wails remains the primary desktop-app path and the closest match to the public `dAppCore/build@v3` action.

- This is the default stack shape in the generated workflow when Wails markers are present.
- Detection accepts `wails.json` directly and also Go roots that contain frontend manifests at the root, under `frontend/`, or in a visible subtree up to depth 2.
- Setup installs Go, Node, frontend dependencies, the Wails CLI, distro-specific Linux WebKit packages, and optional garble when obfuscation is enabled.
- Build uses `wails build` for Wails v2 and forwards build-name, build tags, ldflags, obfuscation, NSIS, and WebView2 options.
- Windows packaging supports NSIS plus the `download`, `embed`, `browser`, and `error` WebView2 modes.
- Signing integrates with the existing macOS and Windows signing layers.

## Wails v3

Wails v3 support is implemented rather than treated as a future placeholder.

- Detection still enters through the Wails stack because the repository shape is a Wails app.
- Build prefers a project Taskfile when present because many Wails v3 repositories already encode their own packaging flow there.
- When no Taskfile exists, the builder falls back to `wails3` directly.
- The same action-style options still apply: build tags, version ldflags, obfuscation, NSIS, and WebView2 where relevant.

## Node

Node-style frontend projects are first-class stacks in their own right.

- Detection accepts `package.json` at the root and in visible nested directories up to depth 2.
- Setup installs Node plus frontend dependencies and honours the declared package manager when present.
- Build runs the package-manager `build` script and collects outputs from target-specific directories.
- The same nested-frontend discovery rules are reused by Wails-backed prebuilds.

## Deno

Deno is integrated into the frontend path instead of being treated as a separate product surface.

- It can run as a standalone frontend stack or as a companion prebuild step for Wails and Node-backed repositories.
- Detection accepts `deno.json` and `deno.jsonc` at the root, under `frontend/`, or in visible nested directories up to depth 2.
- Setup enables Deno when manifests are present or when `DENO_ENABLE`, `DENO_BUILD`, or the `deno-build` input explicitly request it.
- Build honours `DENO_BUILD` first and otherwise defaults to `deno task build`.
- The same Deno rules apply to standalone frontend projects and Wails frontend prebuilds.

## Docs

Docs projects are treated as a dedicated stack rather than falling through to generic frontend handling.

- Detection accepts `mkdocs.yml` and `mkdocs.yaml` at the root or under `docs/`.
- Setup installs Python and MkDocs only when docs markers are present.
- Build runs `mkdocs build` and packages the generated site as an archive-friendly artefact.
- Docs detection intentionally outranks generic Node markers so a docs repository with frontend assets is still understood as a docs stack first.

## C++

C++ projects map onto the action's Conan-oriented setup story.

- Detection uses `CMakeLists.txt`.
- Setup installs Python and Conan when a C++ marker is present.
- Build uses CMake with Conan-aware preparation and target-specific output handling.
- The stack is exposed through discovery suggestions as `cpp`.

## PHP

PHP projects have explicit workflow setup and build support.

- Detection uses `composer.json`.
- Setup verifies PHP and installs Composer when required.
- Build runs Composer-backed dependency installation and can bundle deterministic release artefacts.

## Python

Python projects are also explicit build targets.

- Detection uses `pyproject.toml` or `requirements.txt`.
- Setup is intentionally light because packaging is deterministic and language-native.
- Build produces predictable source-oriented artefacts suitable for archive and release flows.

## Rust

Rust projects participate in the same pipeline rather than needing a separate release tool.

- Detection uses `Cargo.toml`.
- Setup verifies Cargo and bootstraps Rust when required.
- Build uses `cargo build --release --target ...` and collects the target-specific binaries.

## Docker

Container builds are handled by a dedicated stack instead of shelling out from a generic builder.

- Detection accepts `Dockerfile` and `Containerfile` variants.
- Setup stays minimal because Docker/Buildx is expected to exist on the runner that requested the stack.
- Build supports image tags, push/load behaviour, and archive-friendly export modes.

## LinuxKit

LinuxKit is treated as its own stack rather than a Docker special case.

- Detection accepts root `linuxkit.yml` or `linuxkit.yaml` files and `.core/linuxkit/*.yml`.
- Build produces the configured image formats through the LinuxKit builder.
- Release publishing can target LinuxKit registry-oriented flows alongside standard artefact packaging.

## Taskfile

Taskfile acts as the generic escape hatch for repositories that already define their own build graph.

- Detection accepts common Taskfile name variants.
- Setup installs the Task CLI when required.
- Build delegates to the Task targets defined by the repository.
- This is especially important for Wails v3 repositories that already ship a Taskfile-based packaging flow.

## Future and Extended Stacks

The public action docs originally called out Wails v3 and C++ as future or placeholder surfaces. In go-build those paths now exist directly.

The design rule stays the same for any future stack:

- keep discovery generic
- keep setup conditional
- let the stack wrapper own its full build details
