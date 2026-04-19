# @forst/sidecar

Sidecar integration for TypeScript applications to use Forst for high-performance backend operations.

**Compiler:** depends on [`@forst/cli`](https://www.npmjs.com/package/@forst/cli) for the same native `forst` binary download/cache. Env vars (`FORST_BINARY`, `FORST_CACHE_DIR`, `FORST_CLI_VERIFY`) apply here too—see [CLI README](../cli/README.md#environment). Published tarballs pin `@forst/cli` to a **caret range** (`^x.y.z`) matching the CLI version at release time—upgrade **sidecar** when you need a newer sidecar API, and let the range pull a newer CLI, or bump **`@forst/cli`** explicitly in your app if you only need a newer compiler.

**Peer:** [`express`](https://www.npmjs.com/package/express) **^5** (see `peerDependencies`). Install it in your app; the sidecar does not bundle Express.

## Overview

The Forst sidecar enables gradual adoption of Forst within TypeScript applications by providing a seamless bridge between TypeScript and high-performance Forst-compiled Go binaries. This allows teams to solve performance bottlenecks incrementally without requiring a complete rewrite.

## Features

- **Zero-config setup**: Automatic detection and compilation of Forst files
- **Hot reloading**: Automatic recompilation when `.ft` files change
- **Type safety**: Automatic TypeScript interface generation
- **HTTP transport**: Familiar REST/JSON patterns for easy debugging
- **Express.js integration**: Drop-in middleware for Express applications
- **Health monitoring**: Built-in health checks and status monitoring

## Quick Start

### Installation

From the npm registry (Node 18+):

```bash
npm install @forst/sidecar express
```

(`express` is a peer dependency—install the major version range your app already uses if it satisfies `^5`.)

**Monorepo / contributing:** clone the repo, then:

```bash
cd packages/sidecar
bun install
bun run build
```

**Releases:** `@forst/sidecar` is versioned by [Release Please](https://github.com/googleapis/release-please) (`packages/sidecar` in [`.release-please-config.json`](https://github.com/forst-lang/forst/blob/main/.release-please-config.json)). Merging the release PR bumps `package.json` and `jsr.json`. When GitHub publishes a release whose tag contains `sidecar-v`, [publish-packages.yml](https://github.com/forst-lang/forst/blob/main/.github/workflows/publish-packages.yml) runs **Publish packages (npm / JSR)**. npm: [trusted publishing](https://docs.npmjs.com/trusted-publishers/) (OIDC from GitHub Actions) when each package is configured on npmjs.com; optional `NPM_TOKEN` secret still works as fallback. JSR: link the repo for OIDC or set `JSR_TOKEN`. With `workflow_call`, the trusted publisher workflow name on npm may need to match the **caller** (e.g. `release.yml`)—see npm’s troubleshooting for reusable workflows.

**Local npm / JSR:** from `packages/sidecar`, run `npm publish` or `npx jsr publish` if you must publish outside CI; `npx jsr publish --dry-run` validates the JSR package. CI publishes only when a Release Please **GitHub Release** is published (workflow **Publish packages (npm / JSR)**).

**Registries:** [npm](https://www.npmjs.com/package/@forst/sidecar) · [JSR](https://jsr.io/@forst/sidecar).

### Basic Usage

```typescript
import { autoStart } from "@forst/sidecar";

async function main() {
  // Start the sidecar with zero configuration
  const sidecar = await autoStart({
    forstDir: "./forst", // Directory containing your .ft files
    port: 8080,
  });

  // Discover available functions
  const functions = await sidecar.discoverFunctions();
  console.log("Available functions:", functions);

  // Call a Forst function (third argument is the JSON args array for the Forst call)
  const result = await sidecar.invoke("myPackage", "myFunction", [
    { arg: "value" },
  ]);
  console.log("Result:", result);
}

main().catch(console.error);
```

### Express.js Integration

```typescript
import express from "express";
import { ForstSidecar, createExpressMiddleware } from "@forst/sidecar";

const app = express();

// Create and start the sidecar
const sidecar = new ForstSidecar({
  forstDir: "./forst",
  port: 8080,
});
await sidecar.start();

// Add middleware to make sidecar available in routes
app.use(createExpressMiddleware(sidecar));

// Use Forst functions in your routes
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

## Configuration

**Project layout:** `forst dev -root` uses `rootDir` if set, otherwise `forstDir`, otherwise `./forst`, so discovery matches your `.ft` tree. Hot-reload (spawn mode) uses **chokidar** on `watchRoots` if set; otherwise it watches the same directory as **default watch root**: `forstDir`, then `rootDir`, then the same default as `-root`. Set `rootDir` to your **repository or app root** when `.ft` files live in multiple packages; use `watchRoots` to watch several folders without scanning unrelated subtrees.

**Explicit `ftconfig.json`:** set `configPath` to pass `-config` to `forst dev`. The compiler still discovers `ftconfig.json` by walking up from the process working directory (`-root`); `configPath` is for pinning a canonical file in monorepos.

### Monorepos and mixed Forst + TypeScript

- **One dev server per repo (recommended):** run a single `ForstSidecar` with `rootDir` pointing at the monorepo root so discovery sees every `**/*.ft` under that tree (subject to `ftconfig.json` include/exclude). In other packages or processes, use **connect mode** so only the HTTP client runs (no second `forst dev` child, no port collision).
- **Connect mode:** `sidecarRuntime: "connect"` with `devServerUrl`, or set `FORST_DEV_URL` (see below). `start()` resolves the compiler binary only in **spawn** mode.
- **Stable types per package:** for checked-in `.d.ts` and CI, prefer `forst generate` where applicable; the dev server’s `GET /types` is for live iteration—see [examples/in/rfc/typescript-client/01-integration-profiles.md](../../examples/in/rfc/typescript-client/01-integration-profiles.md).
- **Turborepo / Nx:** model `forst dev` as one task (or one sidecar `spawn` at the root); dependent tasks use the sidecar in **connect** mode or call the HTTP API with a shared base URL.

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
// Package A — spawns `forst dev`
await new ForstSidecar({
  rootDir: ".",
  configPath: "./ftconfig.json",
}).start();

// Package B — same process, another terminal, or CI helper — attach only
await new ForstSidecar({
  sidecarRuntime: "connect",
  devServerUrl: "http://127.0.0.1:8080",
}).start();
```

### Basic Configuration

```typescript
import { ForstConfig } from "@forst/sidecar";

const config: ForstConfig = {
  mode: "development", // 'development' | 'production' | 'testing'
  forstDir: "./forst", // Directory containing .ft files
  outputDir: "./dist/forst", // Output directory for compiled files
  port: 8080, // HTTP server port
  host: "localhost", // HTTP server host
  logLevel: "info", // 'debug' | 'info' | 'warn' | 'error'
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

### Environment Variables

Precedence: explicit fields on `ForstConfig` win over these variables (see `mergeForstSidecarEnv`).

- `NODE_ENV`: Used by helpers for default `mode` where applicable.
- `FORST_DIR`: Default for `forstDir` when not set in config.
- `FORST_PORT`: Default `port` when not set in config (spawn and health checks).
- `FORST_DEV_URL`: Base URL of an existing `forst dev` (e.g. `http://127.0.0.1:8080`). When set, **`sidecarRuntime` defaults to `connect`** unless you pass `sidecarRuntime: "spawn"` explicitly.
- `FORST_SKIP_SPAWN`: If `1`, same as forcing **connect** mode (you must still provide a URL via `devServerUrl` or `FORST_DEV_URL`).

If `FORST_DEV_URL` is present but you need to **spawn** locally anyway, set `sidecarRuntime: "spawn"` in code so the sidecar does not attach to the remote URL.

**Version check:** `versionCheck` on `ForstConfig` (`off` | `warn` | `strict`, default **`warn`**) runs after `start()`: it checks that `GET /version`’s **`contractVersion`** matches what this `@forst/sidecar` release expects (HTTP API compatibility), then compares the local `forst` binary (`forst version`) to the server’s reported version using **semver** when both sides parse as semver (otherwise exact string match). In **`strict`**, mismatches throw `ContractVersionMismatch` or `ServerVersionMismatch`. Older `forst dev` builds without a usable `/version` log a warning unless `versionCheck` is `off`.

**Codegen (`forst generate`):** call `forstSidecar.generateTypes()` to run `forst generate` with the same **effective project root** as `forst dev -root` (`rootDir` / `forstDir`). If you set **`configPath`**, the sidecar passes **`-config`** so file discovery matches `forst dev` (include/exclude in `ftconfig.json`). Writes `generated/` under that directory—no HTTP server required. Pair with **connect** mode in monorepos: one task runs `forst dev`, another runs `generateTypes` or a script that calls it.

**Watch + generate:** set **`watchGenerate: true`** to run `forst generate` after each debounced hot-reload restart when `.ft` files change (spawn mode only; uses the same root and optional `-config` as `generateTypes`).

### Troubleshooting

| Symptom | What to check |
| --- | --- |
| `ContractVersionMismatch` / `ServerVersionMismatch` | `versionCheck` (default `warn`): align `@forst/sidecar` with the `forst dev` build, or set `versionCheck: "off"` temporarily while debugging. |
| Port already in use | Another `forst dev` or sidecar **spawn** on the same `FORST_PORT`; use **connect** mode + `FORST_DEV_URL` for a second process. |
| `invoke` throws `DevServerInvokeRejected` | HTTP 200 with `success: false` from the executor—inspect `invokeResponse` on the error, not only `message`. |
| Compiler never downloads | Same as CLI: network to GitHub Releases, or set `FORST_BINARY` in CI. |

## API Reference

### ForstSidecar

Main class for managing the sidecar integration.

#### Methods

- `start()`: Start the development server (spawn `forst dev`) or attach in **connect** mode
- `stop()`: Stop the spawned child or disconnect the client
- `discoverFunctions()`: Discover available Forst functions
- `invoke(package, function, args)`: Call a Forst function (`args`: positional JSON array for the executor). If the dev server returns HTTP 200 with `success: false`, throws **`DevServerInvokeRejected`** (structured `invokeResponse` on the error). Non-2xx responses throw **`DevServerHttpFailure`**; when the body is JSON from `forst dev`, **`serverErrorFromBody`** holds the `error` field.
- `invokeStreaming(package, function, args, onResult)`: Same `args` shape as `invoke` (positional array), with streaming
- `healthCheck()`: Check server health
- `getVersion()`: Read `GET /version` (compiler + HTTP contract metadata)
- `generateTypes()`: Run `forst generate` on the configured project root (writes `generated/` TS)
- `isRunning()`: Check if server is running

### ForstClient

HTTP client for communicating with the Forst server.

#### Methods

- `discoverFunctions()`: Get list of available functions
- `invoke(package, function, args)`: Call a function (positional `args` array); throws **`DevServerInvokeRejected`** / **`DevServerHttpFailure`** on failure (same semantics as `ForstSidecar.invoke`)
- `invokeStreaming(package, function, args, onResult)`: Stream results; same positional `args` as `invoke`
- `healthCheck()`: Check server health
- `getVersion()`: Read `GET /version`

### ForstServer

Manages the Forst development server process.

#### Methods

- `start()`: Start the server
- `stop()`: Stop the server
- `restart()`: Restart the server
- `getServerInfo()`: Get server status information

## Development

### Building

```bash
bun run build
```

### Development Mode

```bash
bun run dev
```

### Testing

```bash
bun test
```

### Running Examples

```bash
# Run the basic example
bun run example

# Or run the compiled example
node dist/examples/basic.js
```

## Project Structure

```
packages/sidecar/
├── src/
│   ├── index.ts          # Barrel exports
│   ├── sidecar.ts        # ForstSidecar, middleware, autoStart
│   ├── client.ts         # HTTP client
│   ├── server.ts         # Development server
│   ├── types.ts          # TypeScript types
│   └── utils.ts          # Utility functions
├── examples/
│   └── basic.ts          # Basic usage example
├── dist/                 # Compiled output
├── package.json
├── tsconfig.json        # IDE + `tsc --noEmit` (includes tests, Bun types)
├── tsconfig.build.json  # `npm run build` emit (excludes *.test.ts)
└── README.md
```

## Forst File Structure

The sidecar expects Forst files to be organized as follows:

```
forst/
├── routes/
│   ├── process_data.ft
│   ├── search.ft
│   └── calculations.ft
└── utils/
    ├── validation.ft
    └── helpers.ft
```

Each `.ft` file should contain public functions that can be called from TypeScript:

```go
// forst/routes/process_data.ft
package routes

type ProcessDataInput = {
  records: Array({ id: String, data: String })
}

func processData(input ProcessDataInput) {
  // High-performance processing logic
  return { processed: len(input.records), status: "success" }
}
```

## Performance Benefits

- **10x faster latency** compared to TypeScript for CPU-intensive operations
- **Memory efficiency** through Go's garbage collector
- **Concurrent processing** with goroutines
- **Compiled performance** vs interpreted TypeScript

## Error Handling

The sidecar provides comprehensive error handling:

- **Compilation errors**: Graceful handling of Forst compilation failures
- **Runtime errors**: Proper error propagation from Forst to TypeScript
- **Network errors**: Retry logic with exponential backoff
- **Health monitoring**: Automatic detection of server issues

## See also

- **`forst dev` HTTP contract** (JSON API): [examples/in/rfc/typescript-client/02-forst-dev-http-contract.md](../../examples/in/rfc/typescript-client/02-forst-dev-http-contract.md)
- **Roadmap** (TypeScript interoperability): [ROADMAP.md](../../ROADMAP.md)

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see LICENSE file for details.
