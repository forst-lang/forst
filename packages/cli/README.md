# @forst/cli

Install the **Forst** compiler for Node.js without a separate Go toolchain. The package ships a small wrapper that downloads the official **native binary** for your OS/arch from [GitHub Releases](https://github.com/forst-lang/forst/releases) (same artifacts as `task build:release`).

The **npm package version** matches the compiler release (e.g. `@forst/cli@0.0.19` uses assets under tag `v0.0.19`).

## Install

```bash
npm install -D @forst/cli
# or
npx @forst/cli version
```

With a local dependency, `node_modules/.bin/forst` runs the wrapper.

## Environment

| Variable | Purpose |
| --- | --- |
| `FORST_BINARY` | Absolute path to a `forst` executable; skips download. |
| `FORST_CACHE_DIR` | Base directory for cached binaries (default: `~/.cache/forst-cli` on Unix, `%LOCALAPPDATA%/forst-cli/cache` on Windows). Each compiler version is stored in a subdirectory. |
| `FORST_CLI_VERIFY` | **Default:** sha256 verification is **required** using digests from [GitHub release metadata](https://docs.github.com/en/rest/releases/releases). Set to `0` or `false` to skip (not recommended). |

Downloads use retries on transient HTTP errors, an exclusive lock when two processes install at once, and an atomic write so a partial file never replaces the binary. In CI or air-gapped environments, prefer `FORST_BINARY`, or set `FORST_CLI_VERIFY=0` only if the GitHub API is unreachable or the release has no digest metadata.

## API

```typescript
import { resolveForstBinary, spawnForst, getCliPackageVersion } from "@forst/cli";

const bin = await resolveForstBinary();
// spawnForst(["generate", "-h"], {}, { version: "0.0.19" });
```

## Relation to `@forst/sidecar`

[`@forst/sidecar`](../sidecar/README.md) depends on this package for [`resolveForstBinary`](./src/resolve.ts) so dev-server integration and the CLI share one download/cache implementation.

## Compiler CLI

The native binary implements `forst dev`, `forst generate`, `forst lsp`, `forst fmt`, etc. See the Go entrypoint in [`forst/cmd/forst/main.go`](../../forst/cmd/forst/main.go).
