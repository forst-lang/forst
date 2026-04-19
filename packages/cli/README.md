# @forst/cli

**Node.js distribution of the Forst compiler.** This package ships a small wrapper that downloads and caches the official **native `forst` binary** for your platform from [GitHub Releases](https://github.com/forst-lang/forst/releases)—the same artifacts built with `task build:release` in the main repository. No local Go toolchain is required.

Use it in npm scripts, developer tooling, or as a dependency of libraries such as [`@forst/sidecar`](../sidecar/README.md).

---

## Table of contents

1. [Overview](#overview)
2. [What’s included](#whats-included)
3. [Requirements](#requirements)
4. [Installation](#installation)
5. [Command-line usage](#command-line-usage)
6. [Environment variables](#environment-variables)
7. [How the wrapper works](#how-the-wrapper-works)
8. [Programmatic API](#programmatic-api)
9. [Releases and publishing](#releases-and-publishing)
10. [Security](#security)
11. [Upgrading](#upgrading)
12. [Troubleshooting](#troubleshooting)
13. [Relationship to `@forst/sidecar`](#relationship-to-forstsidecar)
14. [Native compiler CLI](#native-compiler-cli)
15. [Support](#support)
16. [License](#license)

---

## Overview

`@forst/cli` bridges JavaScript/TypeScript workflows and the Forst compiler: the **npm package version** determines which **native binary** is downloaded (matching that semver on GitHub Releases). The binary is cached per version so repeat invocations avoid redundant downloads.

The wrapper is responsible for resolution, verification, and concurrency-safe installation—not for implementing compiler features (those live in the native `forst` executable).

## What’s included

| Deliverable | Description |
| --- | --- |
| **`forst` on `PATH`** | Via `node_modules/.bin/forst` when installed as a dependency, or `npx @forst/cli` / `npx forst` without a global install. |
| **JavaScript API** | `resolveForstBinary`, `spawnForst`, `getCliPackageVersion`, etc., for tools that need to locate or run the binary programmatically. |
| **Diagnostics** | `npx forst --forst-cli-info` prints package semver, resolved binary path, and `forst version` output—use this in bug reports and CI logs. |

## Requirements

| Requirement | Details |
| --- | --- |
| **Node.js** | Version 18 or later (`engines` in `package.json`). |
| **Network** | HTTPS access to `github.com` on first use (binary download and release metadata), unless you bypass download with `FORST_BINARY`. |
| **Platforms** | Linux, macOS, and Windows on CPU architectures published for your package version on the project’s GitHub Releases. |

## Installation

**As a dev dependency (typical for apps and libraries):**

```bash
npm install -D @forst/cli
```

**One-off invocation without adding to `package.json`:**

```bash
npx @forst/cli version
```

After install, the wrapper is available as `node_modules/.bin/forst`.

On **first use**, the wrapper downloads the binary for the **installed package version** from GitHub. For offline or locked-down CI, set `FORST_BINARY` to a pre-installed executable (see [Environment variables](#environment-variables)).

## Command-line usage

The wrapper forwards arguments to the native `forst` binary once it is resolved. Examples:

```bash
# After local install
npx forst version
npx forst dev --help
npx forst generate
```

**In `package.json` scripts:**

```json
{
  "scripts": {
    "forst": "forst",
    "dev:forst": "forst dev"
  }
}
```

For full command coverage (`dev`, `generate`, `lsp`, `fmt`, …), see [Native compiler CLI](#native-compiler-cli).

## Environment variables

| Variable | Purpose |
| --- | --- |
| `FORST_BINARY` | Absolute path to a `forst` executable; skips download entirely. |
| `FORST_CACHE_DIR` | Base directory for cached binaries. Default: `~/.cache/forst-cli` on Unix, `%LOCALAPPDATA%/forst-cli/cache` on Windows. Each compiler version is stored in a subdirectory. |
| `FORST_CLI_VERIFY` | **Default:** SHA-256 verification is **required** using digests from [GitHub release metadata](https://docs.github.com/en/rest/releases/releases). Set to `0` or `false` to skip (not recommended for production). |

**Behavior:** downloads retry on transient HTTP errors, use an exclusive lock when two processes install concurrently, and write the binary atomically so a partial file never replaces a good one. In air-gapped or API-restricted environments, prefer `FORST_BINARY`, or use `FORST_CLI_VERIFY=0` only when release metadata has no digests or `api.github.com` is unreachable.

## How the wrapper works

1. Read the **semver** of the installed `@forst/cli` package (not “latest” from the registry globally).
2. Resolve the corresponding **release assets** on GitHub for that version.
3. Download (unless `FORST_BINARY` is set), **verify** when `FORST_CLI_VERIFY` is enabled, and place the binary in the cache.
4. Execute the native `forst` with your arguments.

Upgrading the **npm package** is therefore the supported way to pin a compiler version in Node workflows.

## Programmatic API

```typescript
import { resolveForstBinary, spawnForst, getCliPackageVersion } from "@forst/cli";

const bin = await resolveForstBinary();
// spawnForst(["generate", "-h"], {}, { version: "0.0.19" });
```

See TypeScript definitions under `dist/` after build, or source in [`src/`](./src/).

## Releases and publishing

| Topic | Details |
| --- | --- |
| **Versioning** | [Release Please](https://github.com/googleapis/release-please) manages `packages/cli`; tags like `cli-v*` bump `package.json` and `jsr.json` (see [`.release-please-config.json`](https://github.com/forst-lang/forst/blob/main/.release-please-config.json)). |
| **Automation** | When a GitHub Release is published, [.github/workflows/publish-packages.yml](https://github.com/forst-lang/forst/blob/main/.github/workflows/publish-packages.yml) publishes **@forst/cli** to npm and JSR. |
| **npm** | [Package page](https://www.npmjs.com/package/@forst/cli). CI may use [trusted publishing](https://docs.npmjs.com/trusted-publishers/) (OIDC); a repository `NPM_TOKEN` can still be used where a classic token fallback is required. |
| **JSR** | [jsr.io/@forst/cli](https://jsr.io/@forst/cli)—link the repository for OIDC or configure `JSR_TOKEN` as documented by JSR. |
| **Manual JSR** | From `packages/cli`: `npx jsr publish --dry-run` or `npx jsr publish`. A clean git tree is expected unless you pass `--allow-dirty`. |

## Security

- **Integrity:** binaries are verified by default using SHA-256 digests from GitHub’s release API (`FORST_CLI_VERIFY`).
- **Install safety:** atomic writes and locking reduce the risk of corrupted executables under parallel installs.
- **Operational hardening:** in restricted environments, supply a vetted binary via `FORST_BINARY` and prefer OIDC or short-lived credentials in CI over long-lived tokens in logs.

## Upgrading

Change the **`@forst/cli`** version in `package.json` (or your lockfile) when you need a compiler release that shipped after your current dependency. The wrapper always binds to the **installed npm semver**, not an implicit “latest” on the registry—stale dependencies mean a stale compiler until you upgrade.

After upgrading, run:

```bash
npx forst --forst-cli-info
```

Confirm that the reported package version, binary path, and `forst version` match your expectations.

## Troubleshooting

| Symptom | What to check |
| --- | --- |
| Download fails or 404 on an asset | A GitHub release must exist for this package version’s artifacts. Unpublished semvers or forks may need `FORST_BINARY` or a published CLI version. |
| Wrong OS or architecture | Override with `FORST_BINARY` for custom builds or unsupported matrices. |
| SHA-256 / verification errors | Digests must exist in release metadata; proxies blocking `api.github.com` often surface here. Disable verification only as a last resort (`FORST_CLI_VERIFY=0`). |
| Concurrent installs | Locking prevents corruption; different versions use separate cache subdirectories. |

## Relationship to `@forst/sidecar`

[`@forst/sidecar`](../sidecar/README.md) depends on this package and uses [`resolveForstBinary`](./src/resolve.ts) so dev-server integration and the CLI share one download and cache implementation.

## Native compiler CLI

The native binary implements `forst dev`, `forst generate`, `forst lsp`, `forst fmt`, and related commands. Implementation entry point: [`forst/cmd/forst/main.go`](../../forst/cmd/forst/main.go). Refer to the main repository for the full CLI reference and language documentation.

## Support

Report issues and feature requests in the [issue tracker](https://github.com/forst-lang/forst/issues). Include `npx forst --forst-cli-info` output and your OS/architecture when reporting download or verification problems.

## License

MIT — see [`LICENSE`](./LICENSE).
