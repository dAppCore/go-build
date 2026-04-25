---
Status: Aspirational
module: dappco.re/go/build
repo: core/go-build
lang: go
tier: lib
depends:
  - code/core/go
  - code/core/go/io
tags:
  - build
  - installer
  - cdn
  - release
---

# Installer CDN Delivery RFC

This sub-spec extends [RFC.md §12](RFC.md#12-installer-scripts) with the delivery contract for generated installer scripts. The scripts covered are `setup.sh`, `ci.sh`, `php.sh`, `go.sh`, `agent.sh`, and `dev.sh`.

The CDN delivery layer is aspirational in this worktree: installer generation exists, but R2 provisioning, `lthn.sh` routing, signed index generation, and automated CDN upload are not treated as shipped here. Manual hosting is acceptable for the MVP when it preserves the signed index and SHA verification contract below.

## 1. Hosting Decision

Default hosting is Cloudflare R2 behind `https://lthn.sh/`.

R2 is the preferred default because it gives the installer CDN a simple object-store model, edge-cache friendly public reads, and an S3-compatible write API that fits the `io.Medium` abstraction. Writes are private and restricted to the release automation credentials; reads are public through the `lthn.sh` hostname.

Object policy:

| Object | Mutability | Cache policy |
|--------|------------|--------------|
| `/{channel}/{script}-{version}.sh` | immutable | `public, max-age=31536000, immutable` |
| `/{channel}/{script}-{version}.sh.sbom.*` | immutable | `public, max-age=31536000, immutable` |
| `/{channel}/{script}-{version}.sh.provenance.*` | immutable | `public, max-age=31536000, immutable` |
| `/index.json` | mutable channel pointer | `public, max-age=60, must-revalidate` |
| `/index.json.asc` | mutable detached signature | `public, max-age=60, must-revalidate` |

Fallback options remain valid but are not the default:

| Option | Use when | Tradeoff |
|--------|----------|----------|
| S3 + CloudFront | AWS is already the deployment control plane | Mature path, but more operational surface and cost management |
| Self-hosted via go-io Medium | Lethean infra should own the whole path | Simpler ownership, but less CDN reach and more server hardening |
| Forge release artifacts | The release system is the only available publisher | Easy MVP path, but weaker short-domain and channel alias control |

## 2. URL Scheme

Canonical versioned installer URLs use:

```text
https://lthn.sh/{channel}/{script}-{version}.sh
```

`channel` is one of:

| Channel | Meaning |
|---------|---------|
| `stable` | Promoted release suitable for default installation |
| `beta` | Pre-release candidate intended for wider testing |
| `edge` | Commit-addressed build from the development line |
| `rolling` | Latest successful installer generation for internal or fast-moving use |

`script` is the basename without `.sh`: `setup`, `ci`, `php`, `go`, `agent`, or `dev`.

Examples:

```text
https://lthn.sh/stable/setup-v1.4.0.sh
https://lthn.sh/beta/ci-v1.5.0-beta.1.sh
https://lthn.sh/edge/dev-20260425.abcdef0.sh
```

The compatibility aliases from RFC §12, such as `https://lthn.sh/setup.sh`, may remain, but they are bootstraps only. A compatibility alias must resolve the signed `index.json`, select the current `stable` entry for its script, fetch the pinned versioned object, verify SHA-256, and execute the verified temporary file. High-assurance installation should prefer a trusted Core binary or pinned bootstrap that performs the same verification before execution.

## 3. Version Pinning

Every uploaded installer script is pinned by SHA-256 in:

```text
https://lthn.sh/index.json
https://lthn.sh/index.json.asc
```

`index.json` is the only mutable channel pointer. Versioned script objects are immutable and must not be overwritten. Promotion, rollback, and channel movement happen by replacing the signed index, not by changing existing script content.

Minimum index shape:

```json
{
  "schema": 1,
  "generated_at": "2026-04-25T00:00:00Z",
  "signing_key": {
    "id": "lethean-installer-2026",
    "fingerprint": "OPENPGP-FINGERPRINT-HERE"
  },
  "channels": {
    "stable": {
      "setup": {
        "version": "v1.4.0",
        "url": "https://lthn.sh/stable/setup-v1.4.0.sh",
        "sha256": "SHA256-HERE",
        "size": 12345,
        "sbom": "https://lthn.sh/stable/setup-v1.4.0.sh.sbom.spdx.json",
        "provenance": "https://lthn.sh/stable/setup-v1.4.0.sh.provenance.intoto.jsonl"
      }
    }
  }
}
```

For `stable` and `beta`, `version` is the release tag. For `edge` and `rolling`, `version` may include a UTC build date and short commit SHA. The object path still carries that version so a client can pin to an exact script even when following a mutable channel.

## 4. Signature Verification

Installers use a HEAD-and-verify pattern:

1. Load the embedded Lethean installer public key and expected OpenPGP fingerprint.
2. Fetch `/index.json` and `/index.json.asc`.
3. Verify the detached GPG signature before reading installer entries.
4. Resolve the requested `{channel, script}` to a versioned URL and expected SHA-256.
5. Send `HEAD` for the versioned script and reject obvious mismatches when `Content-Length` or an `x-lthn-sha256` metadata header disagrees with the signed index. Headers are an optimization only.
6. Fetch the script body to a temporary file.
7. Compute SHA-256 locally and compare it with the signed index entry.
8. Execute only the verified temporary file.

Unsigned indexes, unknown keys, revoked keys, missing entries, missing SHA values, SHA mismatches, and unexpected redirects outside `https://lthn.sh/` are hard failures.

The `curl https://lthn.sh/setup.sh | bash` style remains a compatibility path, but it cannot verify the bootstrap content before the shell starts. The bootstrap must therefore be minimal, stable, and limited to the signed-index verification flow above.

## 5. Update Flow

`core build push --installer` owns publication.

Expected flow:

1. Regenerate installer scripts for `setup`, `ci`, `php`, `go`, `agent`, and `dev`.
2. Produce SBOM and provenance sidecars for each script.
3. Compute SHA-256 and size metadata for every script and sidecar.
4. Upload immutable versioned script and sidecar objects first.
5. Regenerate `index.json` with the new channel pins.
6. Sign `index.json` with the Lethean installer GPG key.
7. Upload `/index.json` and `/index.json.asc` last.
8. Purge or revalidate only the mutable index objects.

The automated path depends on `code/core/go/io` gaining an R2-backed `io.Medium`. Until that backend exists, the MVP may generate the same publish directory locally and upload it manually through the R2 dashboard, an S3-compatible client pointed at R2, S3 + CloudFront, self-hosted storage, or Forge release artifacts. Manual uploads must still publish versioned objects first and the signed index last.

## 6. Rollback

Rollback is a signed index change.

To roll back a channel, update that channel entry in `index.json` to the prior version and SHA, sign the new index, upload `/index.json` and `/index.json.asc`, then purge or revalidate the index cache. Existing clients pinned to an old SHA continue to work because versioned artifacts are immutable. Clients following a channel observe the rollback after their next signed index refresh.

If a script artifact is known compromised, remove it from all channel entries, publish a signed index that points elsewhere, purge the mutable index cache, and remove the compromised immutable object only after confirming no supported channel references it.

## 7. Supply Chain

Every uploaded installer artifact carries SBOM and provenance sidecars through go-build's existing release integrity pipeline.

Required sidecars:

| Sidecar | Purpose |
|---------|---------|
| `{script}-{version}.sh.sbom.spdx.json` | SBOM for the generated script and embedded install metadata |
| `{script}-{version}.sh.provenance.intoto.jsonl` | Build provenance linking source revision, builder, and release invocation |
| `CHECKSUMS.txt` | Optional channel or release checksum bundle for offline audit |
| `CHECKSUMS.txt.asc` | Detached GPG signature when a checksum bundle is published |

`index.json` references the SBOM and provenance URLs for each installer entry. The signed index remains the client trust root for online installation; SBOM, provenance, and checksum bundles support audit, release review, and incident response.
