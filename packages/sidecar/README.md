# @forst/sidecar

**TypeScript integration for Forst** in Node.js applications: start or attach to a `forst dev` server, invoke compiled Forst functions over HTTP, and keep generated types aligned with your repository layout.

This package is aimed at teams adopting Forst **incrementally**—without replacing their existing Express/TypeScript stack in one step.

---

## Table of contents

1. [Overview](#overview)
2. [At a glance](#at-a-glance)
3. [Use cases](#use-cases)
4. [Requirements](#requirements)
5. [Features](#features)
6. [Installation](#installation)
7. [Quick start](#quick-start)
8. [Express integration](#express-integration)
9. [Architecture: spawn vs connect](#architecture-spawn-vs-connect)
10. [Configuration](#configuration)
11. [Environment variables](#environment-variables)
12. [Versioning and codegen](#versioning-and-codegen)
13. [Troubleshooting](#troubleshooting)
14. [API reference](#api-reference)
15. [Forst sources layout](#forst-sources-layout)
16. [Development](#development)
17. [Package layout](#package-layout)
18. [Performance](#performance)
19. [Error handling](#error-handling)
20. [Publishing and distribution](#publishing-and-distribution)
21. [See also](#see-also)
22. [Support](#support)
23. [Contributing](#contributing)
24. [License](#license)

---

## Overview

The sidecar orchestrates the **Forst development server** (`forst dev`) and a small **HTTP client** so TypeScript code can discover functions, invoke them with JSON payloads, and run `forst generate` with the same roots and config as the dev server.

Two runtime modes matter in practice:

| Mode | Role |
| --- | --- |
| **Spawn** | The sidecar starts `forst dev` as a child process (typical local development). |
| **Connect** | The sidecar only talks HTTP to an already-running `forst dev` (typical second process, CI shard, or monorepo package). |

**Compiler dependency:** [`@forst/cli`](https://www.npmjs.com/package/@forst/cli) supplies the native `forst` binary (download/cache). Shared environment variables: [`FORST_BINARY`, `FORST_CACHE_DIR`, `FORST_CLI_VERIFY`](../cli/README.md#environment-variables). Published `@forst/sidecar` packages declare `@forst/cli` with a **caret** range (`^x.y.z`) fixed at release time—bump **sidecar** for API changes, adjust **`@forst/cli`** when you only need a newer compiler.

**Peer dependency:** [`express`](https://www.npmjs.com/package/express) **^5**—install Express in the host application; it is not bundled here.

## At a glance

| | |
| --- | --- |
| **Role** | Dev server lifecycle + HTTP client + optional Express middleware |
| **Typical stack** | Node.js 18+, Express 5, `@forst/cli` (transitive or direct) |
| **Protocols** | JSON over HTTP to `forst dev` ([contract](../../examples/in/rfc/typescript-client/02-forst-dev-http-contract.md)) |
| **Registries** | [npm](https://www.npmjs.com/package/@forst/sidecar) · [JSR](https://jsr.io/@forst/sidecar) |

## Use cases

- **Local development:** one `ForstSidecar` with hot reload over a tree of `.ft` files; iterate from TypeScript without hand-managing the compiler binary.
- **Monorepos:** run a single `forst dev` at the repo root; other packages use **connect** mode so only the HTTP client runs (no duplicate servers or port collisions).
- **Automation and CI:** shared `FORST_DEV_URL`, health checks, and `generateTypes()` from the same configuration as dev.

## Requirements

- **Node.js** 18 or later (`engines` in `package.json`)
- **Express** 5.x in the host application (`peerDependencies`)
- **`@forst/cli`** resolvable at runtime (dependency of this package or pinned explicitly) so the native compiler can be located

## Features

- Zero-config defaults for common layouts
- Hot reload when `.ft` files change (spawn mode)
- TypeScript-oriented workflow (`forst generate`, optional watch)
- HTTP/JSON interaction with `forst dev`
- Express middleware to expose the sidecar on `req.forst`
- Health and version endpoints for coordination with orchestration

## Installation

**From the public registry:**

```bash
npm install @forst/sidecar express
```

Express is a **peer dependency**—if your app already satisfies `express@^5`, you do not need a second install line.

**Working inside this monorepo** (contributors):

```bash
cd packages/sidecar
bun install
bun run build
```

## Quick start

```typescript
import { autoStart } from "@forst/sidecar";

async function main() {
  const sidecar = await autoStart({
    forstDir: "./forst",
    port: 8080,
  });

  const functions = await sidecar.discoverFunctions();
  console.log("Available functions:", functions);

  const result = await sidecar.invoke("myPackage", "myFunction", [
    { arg: "value" },
  ]);
  console.log("Result:", result);
}

main().catch(console.error);
```

The third argument to `invoke` is the **positional JSON argument array** expected by the Forst executor.

## Express integration

```typescript
import express from "express";
import { ForstSidecar, createExpressMiddleware } from "@forst/sidecar";

const app = express();

const sidecar = new ForstSidecar({
  forstDir: "./forst",
  port: 8080,
});
await sidecar.start();

app.use(createExpressMiddleware(sidecar));

app.post("/process-data", async (req, res) => {
  try {
    const result = await req.forst.invoke("data", "process", [req.body]);
    res.json(result);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

app.listen(3000, () => {
  console.log("Server running on port 3000");
});
```

## Architecture: spawn vs connect

**Spawn (default when no dev URL is set)**

- The sidecar resolves the `forst` binary (via `@forst/cli`) and starts `forst dev` as a subprocess.
- Use for local development and whenever this process should own the compiler lifecycle.

**Connect**

- Set `sidecarRuntime: "connect"` and `devServerUrl`, or set `FORST_DEV_URL` (see [Environment variables](#environment-variables)).
- The sidecar does **not** spawn a child; it only issues HTTP requests to an existing server.
- Use when another task or package already runs `forst dev`, or in CI where the server is started separately.

`start()` resolves the compiler binary only in **spawn** mode.

## Configuration

### Project layout and roots

`forst dev -root` uses `rootDir` if set, otherwise `forstDir`, otherwise `./forst`, so discovery matches your `.ft` tree. In **spawn** mode, hot reload uses **chokidar** on `watchRoots` when set; otherwise it watches the same directory as the default watch root (`forstDir`, then `rootDir`, then the same default as `-root`). Set `rootDir` to your **repository or app root** when `.ft` files span multiple packages; use `watchRoots` to include several folders without scanning unrelated subtrees.

### `ftconfig.json`

Set `configPath` to pass `-config` to `forst dev` when you need a canonical config path in monorepos. The compiler still discovers `ftconfig.json` by walking upward from the process working directory (`-root`); `configPath` pins a specific file when discovery-by-walk is ambiguous.

### Monorepos and mixed Forst + TypeScript

- **One dev server per repository (recommended):** a single `ForstSidecar` with `rootDir` at the monorepo root so discovery sees `**/*.ft` under that tree (subject to `ftconfig.json` include/exclude). Other packages use **connect** mode.
- **Stable checked-in types:** prefer `forst generate` (or `generateTypes()`) for CI and commits; `GET /types` on the dev server is for live iteration—see [01-integration-profiles.md](../../examples/in/rfc/typescript-client/01-integration-profiles.md).
- **Turborepo / Nx:** model `forst dev` as one root task; dependents use **connect** mode or call the HTTP API with a shared base URL.

Example layout:

```text
repo/
  ftconfig.json
  apps/api/
    handler.ts
    handler.ft
  packages/core/
    util.ts
    util.ft
```

```typescript
// One process — spawns `forst dev`
await new ForstSidecar({
  rootDir: ".",
  configPath: "./ftconfig.json",
}).start();

// Another terminal or package — attach only
await new ForstSidecar({
  sidecarRuntime: "connect",
  devServerUrl: "http://127.0.0.1:8080",
}).start();
```

### Example `ForstConfig` shape

```typescript
import { ForstConfig } from "@forst/sidecar";

const config: ForstConfig = {
  mode: "development",
  forstDir: "./forst",
  outputDir: "./dist/forst",
  port: 8080,
  host: "localhost",
  logLevel: "info",
  transports: {
    development: {
      mode: "http",
      http: {
        port: 8080,
        cors: true,
        healthCheck: "/health",
      },
    },
    production: {
      mode: "http",
      http: {
        port: 8080,
        cors: true,
        healthCheck: "/health",
      },
    },
  },
};
```

## Environment variables

Precedence: explicit fields on `ForstConfig` override these (see `mergeForstSidecarEnv` in the codebase).

| Variable | Role |
| --- | --- |
| `NODE_ENV` | Influences default `mode` where applicable. |
| `FORST_DIR` | Default for `forstDir` when not set in config. |
| `FORST_PORT` | Default `port` when not set (spawn and health checks). |
| `FORST_DEV_URL` | Base URL of an existing `forst dev` (e.g. `http://127.0.0.1:8080`). When set, **`sidecarRuntime` defaults to `connect`** unless you pass `sidecarRuntime: "spawn"`. |
| `FORST_SKIP_SPAWN` | If `1`, forces **connect** semantics; you must still supply `devServerUrl` or `FORST_DEV_URL`. |

If `FORST_DEV_URL` is set but you need a local **spawn** anyway, set `sidecarRuntime: "spawn"` in code.

## Versioning and codegen

**`versionCheck`** on `ForstConfig` (`off` | `warn` | `strict`, default **`warn`**): after `start()`, compares `GET /version`’s **`contractVersion`** with what this `@forst/sidecar` build expects, then compares the local `forst` binary (`forst version`) to the server’s version (semver when both parse, else exact string). **`strict`** throws `ContractVersionMismatch` or `ServerVersionMismatch`. Older `forst dev` builds without a usable `/version` log a warning unless `versionCheck` is `off`.

**`generateTypes()`** runs `forst generate` with the same effective project root as `forst dev -root` (`rootDir` / `forstDir`). If **`configPath`** is set, `-config` is passed so include/exclude matches `forst dev`. Output goes under `generated/`—no HTTP server required. In monorepos, pair with **connect** mode: one task runs `forst dev`, another runs codegen.

**`watchGenerate`:** when `true`, runs `forst generate` after debounced hot-reload restarts when `.ft` files change (**spawn** mode only; same roots as `generateTypes`).

## Troubleshooting

| Symptom | What to check |
| --- | --- |
| `ContractVersionMismatch` / `ServerVersionMismatch` | Align `@forst/sidecar` with the `forst dev` build, or set `versionCheck: "off"` temporarily. |
| Port already in use | Another **spawn** on the same `FORST_PORT`; use **connect** + `FORST_DEV_URL` for a second process. |
| `DevServerInvokeRejected` | HTTP 200 with `success: false` from the executor—inspect `invokeResponse` on the error. |
| Compiler never downloads | Network access to GitHub Releases, or set `FORST_BINARY` (see [CLI README](../cli/README.md#environment-variables)). |

## API reference

### `ForstSidecar`

Main API for managing the sidecar.

| Method | Description |
| --- | --- |
| `start()` | Start `forst dev` (**spawn**) or attach (**connect**). |
| `stop()` | Stop the child process or disconnect the client. |
| `discoverFunctions()` | List discoverable Forst functions. |
| `invoke(package, function, args)` | Call a function; `args` is the positional JSON array. On HTTP 200 with `success: false`, throws **`DevServerInvokeRejected`**. Non-2xx: **`DevServerHttpFailure`**; JSON bodies may set **`serverErrorFromBody`**. |
| `invokeStream(...)` | NDJSON stream: omit the last arg for `for await`, or pass `onResult` to consume by callback. |
| `healthCheck()` | Server health. |
| `getVersion()` | `GET /version` (compiler + contract metadata). |
| `generateTypes()` | Run `forst generate` at the configured root. |
| `isRunning()` | Whether the server is considered up. |

### `ForstClient`

HTTP client for the same JSON API; method semantics match `ForstSidecar` where applicable.

### `ForstServer`

Process management for the development server (`start` / `stop` / `restart` / `getServerInfo`).

## Forst sources layout

Organize `.ft` files under a directory such as `forst/` (names are project-specific):

```text
forst/
├── routes/
│   ├── process_data.ft
│   ├── search.ft
│   └── calculations.ft
└── utils/
    ├── validation.ft
    └── helpers.ft
```

Expose public functions callable from TypeScript—for example:

```go
// forst/routes/process_data.ft
package routes

type ProcessDataInput = {
  records: Array({ id: String, data: String })
}

func processData(input ProcessDataInput) {
  return { processed: len(input.records), status: "success" }
}
```

## Development

| Task | Command |
| --- | --- |
| Build | `bun run build` |
| Watch | `bun run dev` |
| Tests | `bun test` |
| Example | `bun run example` or `node dist/examples/basic.js` after build |

## Package layout

```text
packages/sidecar/
├── src/
│   ├── index.ts          # Barrel exports
│   ├── sidecar.ts        # ForstSidecar, middleware, autoStart
│   ├── client.ts         # HTTP client
│   ├── server.ts         # Development server
│   ├── types.ts          # TypeScript types
│   └── utils.ts          # Utilities
├── examples/basic.ts
├── dist/                 # Build output
├── package.json
├── tsconfig.json         # IDE + typecheck (includes tests)
├── tsconfig.build.json   # Emit for publish
└── README.md
```

## Performance

Forst compiles selected logic to native code; CPU-bound paths may outperform equivalent hot spots left in interpreted TypeScript. **Measure in your own environment**—the sidecar handles integration, not application benchmarking.

## Error handling

- **Compilation:** errors surface from `forst dev` logs in **spawn** mode.
- **Invoke:** use `DevServerInvokeRejected` and `DevServerHttpFailure` for structured handling ([API reference](#api-reference)).
- **HTTP client:** retries with backoff where appropriate for transient failures.
- **Health:** use `healthCheck()` or your HTTP health route before depending on traffic.

## Publishing and distribution

| Topic | Details |
| --- | --- |
| **Versioning** | [Release Please](https://github.com/googleapis/release-please) for `packages/sidecar`; tags like `sidecar-v*` update `package.json` and `jsr.json` ([config](https://github.com/forst-lang/forst/blob/main/.release-please-config.json)). |
| **CI** | [.github/workflows/publish-packages.yml](https://github.com/forst-lang/forst/blob/main/.github/workflows/publish-packages.yml) publishes to npm and JSR after a Release Please **GitHub Release**. |
| **npm** | [trusted publishing](https://docs.npmjs.com/trusted-publishers/) (OIDC) when the package is configured on npmjs.com; optional `NPM_TOKEN` as fallback. With reusable workflows, the workflow name trusted by npm may need to match the **caller** (e.g. `release.yml`)—see npm’s troubleshooting. |
| **JSR** | Repository OIDC or `JSR_TOKEN`. |
| **Manual** | From `packages/sidecar`: `npm publish` / `npx jsr publish`; `npx jsr publish --dry-run` to validate. |
| **`repository.url`** | Same as CLI: for npm **trusted publishing** from GitHub, `repository.url` must match the repo OIDC sees ([npm docs](https://docs.npmjs.com/trusted-publishers/)). This package uses `git+https://github.com/forst-lang/forst.git`; change it if you publish from a fork. |

## See also

- **`forst dev` HTTP contract:** [02-forst-dev-http-contract.md](../../examples/in/rfc/typescript-client/02-forst-dev-http-contract.md)
- **Roadmap:** [ROADMAP.md](../../ROADMAP.md)

## Support

Open an issue in the [tracker](https://github.com/forst-lang/forst/issues) with `@forst/sidecar` and `@forst/cli` versions, **spawn** vs **connect**, and logs. For compiler or binary issues, include `npx forst --forst-cli-info` from [`@forst/cli`](../cli/README.md).

## Contributing

1. Fork the repository  
2. Create a feature branch  
3. Make changes and add tests where appropriate  
4. Open a pull request  

## License

MIT — see the [`LICENSE`](./LICENSE) file in this package.
