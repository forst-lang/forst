# Forst — VS Code extension

Provides `.ft` language registration and connects to the Forst compiler’s **HTTP LSP** (`forst lsp`) for diagnostics on open, edit, and close.

## LSP over HTTP

The server listens on `POST /` with **`Content-Type: application/json`**. Each request body is a single **JSON-RPC 2.0** object (`jsonrpc`, `id`, `method`, `params`). The response is one JSON-RPC result or error object. This is **not** stdio LSP; editors need a small HTTP bridge (this extension implements one).

- **`GET /`** returns a short JSON description of the endpoint (for probes or manual checks).
- **`GET /health`** returns a health JSON payload.

Unsupported methods return JSON-RPC error **`-32601` Method not found** (except `initialized` and `$/cancelRequest`, which are acknowledged with a null result when posted with an `id`).

## Requirements

- `forst` on `PATH`, or set **Forst: Path** (`forst.path`) to the executable.
- Default LSP URL: `http://127.0.0.1:8081` (configurable via **Forst: Lsp: Port**).

## Develop (F5)

Use **VS Code** (or a fork that supports the **Extension Development Host**). Run `bun install` at the repo root once; TypeScript is compiled automatically when you F5 (see below) or run `task build:vscode` / `bun run compile` in `packages/vscode-forst`.

**Important:** `--extensionDevelopmentPath` must point at **`packages/vscode-forst`**, not the monorepo root. Otherwise the extension never loads (no **Forst** output channel, no Problems).

1. Build the compiler so `forst` is on `PATH`, or set **Forst: Path** later.
2. **If you opened the whole Forst repo as the workspace** (recommended): use the root [`.vscode/launch.json`](../../.vscode/launch.json) configuration **“Run Extension (vscode-forst)”** and press F5. That sets `extensionDevelopmentPath` to `packages/vscode-forst` and runs the **bun: compile vscode-forst** preLaunch task (`bun run compile` in this package) so `out/` is up to date.
3. **If you opened only `packages/vscode-forst` as the workspace**: use [.vscode/launch.json](.vscode/launch.json) **“Run Extension (this folder = vscode-forst)”** and F5; its preLaunch task compiles the same way.

In the **Extension Development Host** window, **File → Open Folder** and choose a tree that contains `.ft` files (e.g. `examples/in`). Open an `.ft` file; the status bar language mode should be **Forst**. Check **View → Output** and pick **Forst** — you should see “Forst extension activated.” and LSP logs after that.

### Troubleshooting

- No **Forst** channel: the dev host did not load this extension; fix `extensionDevelopmentPath` as above.
- Language mode **Plain Text** on `.ft`: extension not loaded, or another extension stole the association; confirm the dev host and try **Change Language Mode** → Forst.
- **Cursor:** extension debugging may differ; try VS Code if F5 does nothing.

## Settings

| Setting | Default | Description |
| --- | --- | --- |
| `forst.path` | _(empty)_ | Path to `forst`; empty uses `PATH`. |
| `forst.lsp.port` | `8081` | Port for `forst lsp`. |
| `forst.lsp.logLevel` | `info` | Passed to `-log-level`. |
| `forst.lsp.autoStart` | `true` | Spawn `forst lsp` locally; set `false` if you start the server yourself. |

## Status bar

When the extension activates, the status bar shows the configured LSP port (right side, e.g. `Forst :8081`). The icon reflects LSP state: outline (idle/starting), checkmark (initialized), or error icon if the last operation failed. **Click** the item to open the **Forst** output channel (same as command **Forst: Focus output**).

## Command palette

- **Forst: Focus output** — opens the **Forst** log channel (HTTP LSP client, spawn, health).
- **Forst: Restart language server** — stops the spawned process (if any), clears diagnostics, and re-syncs open `.ft` files.

## CI

From the repo root: `task build:vscode` compiles this package. The same compile runs at the start of `task ci:test` (see [Taskfile.yml](../../Taskfile.yml)). GitHub Actions uses `package.json` `packageManager` with [setup-bun](https://github.com/oven-sh/setup-bun) to pin the Bun version.

## Releases and Open VSX

The extension is versioned separately from the compiler: Release Please uses **`packages/vscode-forst`** and tags like **`vscode-forst-v0.0.19`**. When that GitHub Release is published, [`.github/workflows/publish-vscode-extension.yml`](../../.github/workflows/publish-vscode-extension.yml) builds `dist/forst-vscode-<version>.vsix`, uploads it to the release, and runs **`npx ovsx publish`** when the repository has an Actions secret **`OVSX_PAT`** (see [Auto publishing extensions](https://github.com/EclipseFdn/open-vsx.org/wiki/Auto-Publishing-Extensions)). If the secret is missing, Open VSX publish is skipped and the workflow still succeeds.