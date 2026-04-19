# @forst/cli

Install the **Forst** compiler for Node.js without a separate Go toolchain. This package provides a small Node.js wrapper that downloads and caches the official **native `forst` binary** for your platform from [GitHub Releases](https://github.com/forst-lang/forst/releases) (the same artifacts produced by `task build:release` in the main repository).

Use it when you want the compiler in JavaScript/TypeScript workflows (npm scripts, tooling, or libraries such as [`@forst/sidecar`](../sidecar/README.md)) without building Go from source.

## Requirements

- **Node.js** 18 or later (`engines` in `package.json`)
- **Network access** to `github.com` on first use (binary download and release metadata), unless you point to a pre-installed binary with `FORST_BINARY`
- **Supported platforms:** Linux, macOS, and Windows on architectures published on the project’s GitHub Releases for your package version

The **npm package version** is chosen by Release Please for `packages/cli`; the wrapper downloads the native binary for that semver from [GitHub Releases](https://github.com/forst-lang/forst/releases) (typically aligned with the compiler).

**Releases:** Tags `cli-v*` (same pattern as `sidecar-v*`, see [`.release-please-config.json`](https://github.com/forst-lang/forst/blob/main/.release-please-config.json) `packages/cli`) bump `package.json` and `jsr.json`. When GitHub publishes that release, [publish-packages.yml](https://github.com/forst-lang/forst/blob/main/.github/workflows/publish-packages.yml) publishes **@forst/cli** to npm and JSR (link [jsr.io/@forst/cli](https://jsr.io/@forst/cli) to this repo for OIDC, or set `JSR_TOKEN`).

**Local JSR dry-run / publish:** from `packages/cli`, run `npx jsr publish --dry-run` or `npx jsr publish` (CI only runs on GitHub Release). Committed trees are required unless you pass `--allow-dirty`.

**Registries:** [npm](https://www.npmjs.com/package/@forst/cli) · [JSR](https://jsr.io/@forst/cli).

**CI publishes** use GitHub Actions OIDC ([trusted publishing](https://docs.npmjs.com/trusted-publishers/)) when configured on npm; a classic `NPM_TOKEN` in repository secrets may be used where a fallback is required. JSR publishes use OIDC linked to the repository, or a `JSR_TOKEN` secret when applicable.

## Security

The wrapper **verifies** downloaded binaries by default (`FORST_CLI_VERIFY`) using SHA-256 digests from GitHub’s release API. Downloads use atomic writes and file locking so concurrent installs cannot leave a truncated binary in place. For maximum control in locked-down environments, supply a vetted compiler with `FORST_BINARY` and avoid exposing long-lived tokens in logs (CI should use OIDC or short-lived credentials where possible).

## Install

```bash
npm install -D @forst/cli
# or
npx @forst/cli version
```

With a local dependency, `node_modules/.bin/forst` runs the wrapper.

On **first use**, the wrapper downloads the binary for your installed package version (HTTPS to GitHub). To skip the download—for example in offline CI—set `FORST_BINARY` to a pre-installed executable (see **Environment** below).

**Diagnostics:** `npx forst --forst-cli-info` prints the npm package semver, the resolved native binary path, and the output of `forst version` (useful for bug reports and CI).

## Environment

| Variable | Purpose |
| --- | --- |
| `FORST_BINARY` | Absolute path to a `forst` executable; skips download. |
| `FORST_CACHE_DIR` | Base directory for cached binaries (default: `~/.cache/forst-cli` on Unix, `%LOCALAPPDATA%/forst-cli/cache` on Windows). Each compiler version is stored in a subdirectory. |
| `FORST_CLI_VERIFY` | **Default:** sha256 verification is **required** using digests from [GitHub release metadata](https://docs.github.com/en/rest/releases/releases). Set to `0` or `false` to skip (not recommended). |

Downloads use retries on transient HTTP errors, an exclusive lock when two processes install at once, and an atomic write so a partial file never replaces the binary. In CI or air-gapped environments, prefer `FORST_BINARY`, or set `FORST_CLI_VERIFY=0` only if the GitHub API is unreachable or the release has no digest metadata.

## Upgrading

Bump the **`@forst/cli`** version in `package.json` when you need a compiler release that shipped after your current lockfile entry. The wrapper always resolves the native binary for the **installed npm package semver**, not “latest” globally—so a stale dependency means a stale compiler until you upgrade. After upgrading, run `npx forst --forst-cli-info` to confirm the binary and `forst version` match expectations.

## Troubleshooting

| Symptom | What to check |
| --- | --- |
| Download fails or 404 on asset | A GitHub release must exist for this package version’s compiler artifacts. If you use a git fork or unpublished semver, pin `FORST_BINARY` or publish/install a released CLI version. |
| Wrong OS/arch binary | Rare on supported platforms; override with `FORST_BINARY` for custom builds. |
| sha256 / verify errors | Release metadata must include digests; otherwise verification fails unless `FORST_CLI_VERIFY=0`. Corporate proxies blocking `api.github.com` show up here too. |
| Two installs race | Locking avoids corruption; if both processes use different versions, each gets its own cache subdirectory. |

## API

```typescript
import { resolveForstBinary, spawnForst, getCliPackageVersion } from "@forst/cli";

const bin = await resolveForstBinary();
// spawnForst(["generate", "-h"], {}, { version: "0.0.19" });
```

## Relation to `@forst/sidecar`

[`@forst/sidecar`](../sidecar/README.md) depends on this package for [`resolveForstBinary`](./src/resolve.ts) so dev-server integration and the CLI share one download/cache implementation.

## Compiler CLI

The native binary implements `forst dev`, `forst generate`, `forst lsp`, `forst fmt`, and related commands. Refer to the Go entry point in [`forst/cmd/forst/main.go`](../../forst/cmd/forst/main.go) and the main project documentation for full CLI behavior.

## Support

Report issues and feature requests in the [issue tracker](https://github.com/forst-lang/forst/issues). Include the output of `npx forst --forst-cli-info` and your OS/architecture when reporting download or verification problems.

## License

MIT — see [`LICENSE`](./LICENSE).
