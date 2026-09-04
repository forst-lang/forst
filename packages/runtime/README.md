# @forst/runtime

Node runtime for **Forst → TypeScript** interop. Compiled Go binaries call your legacy `.ts` and `.js` modules over a closed RPC channel. The Forst compiler uses this package at build time to index TypeScript exports.

**Status:** experimental. Pin this package and verify with the [examples](https://github.com/forst-lang/forst/tree/main/examples/in/rfc/bridge-interop) before production use.

[Full guide → Call JavaScript from Forst](https://docs.forst.dev/interop/bridge)

## Install

```bash
npm install @forst/runtime
```

```bash
npx jsr add @forst/runtime
```

Requires **Node.js 18+**. When your Forst program uses a JS import (`import "./path" node`), you also need **tsx** on the path for TypeScript loading.

| Registry | Package |
| --- | --- |
| npm | [@forst/runtime](https://www.npmjs.com/package/@forst/runtime) |
| JSR | [@forst/runtime](https://jsr.io/@forst/runtime) |

## What you get

The runtime is built on [Effect](https://effect.website): structured logs via `Effect.log*` and `Effect.fn` spans, with `ForstRuntimeLayer` (stderr pretty logging, tracing, and `FORST_BRIDGE_LOG_LEVEL`) provided at process boundaries via `NodeRuntime.runMain` or `Effect.runPromise`.

RPC and runtime hot paths are wrapped in `Effect.fn` spans (`Rpc.dispatch`, `Runtime.handleSyncCall`, …). Span attributes (`rpc_method`, `module_id`, …) appear on log lines when `FORST_BRIDGE_LOG_LEVEL` is `debug` or `trace`.

### Custom Effect runtime

Default entrypoints (`bootstrap.js`, `@forst/runtime/host`) use `ForstRuntimeLayer`. To bring your own logging, tracing, or services, build a setup and pass it at the process boundary:

```typescript
import { NodeRuntime } from "@effect/platform-node";
import { Effect, Layer } from "effect";
import {
  bootstrapMain,
  bootstrapFatal,
  createNodeRuntimeSetup,
  makeForstRuntimeLayer,
  startForstNodeHost,
} from "@forst/runtime";

// Standalone child: Forst owns stderr logging and tracing.
const myLayer = makeForstRuntimeLayer();
const { layer, runtime } = createNodeRuntimeSetup(myLayer);

// Embedded in an Effect app: keep the parent logger and tracer.
const embeddedLayer = Layer.merge(
  appLayer,
  makeForstRuntimeLayer({ replaceLogger: false })
);
const embedded = createNodeRuntimeSetup(embeddedLayer);

// Bootstrap child (local socket RPC). disablePrettyLogger avoids duplicate stdout logging.
NodeRuntime.runMain(
  bootstrapMain({ runtime }).pipe(
    Effect.catchAllDefect((cause) => bootstrapFatal(cause)),
    Effect.provide(layer)
  ),
  { disablePrettyLogger: true }
);

// Host mode: pass runtimeLayer so RPC forks use the same setup
await Effect.runPromise(
  startForstNodeHost({ runtimeLayer: embedded.layer }).pipe(
    Effect.provide(embedded.layer)
  )
);
```

Use the same `layer` for `Effect.provide` and the matching `runtime` for async RPC dispatch sites (`bootstrapMain`, host connection forks).

### OpenTelemetry (optional)

Install peer packages when you want RPC spans exported to OTLP or the console:

```bash
npm install @effect/opentelemetry @opentelemetry/api @opentelemetry/sdk-trace-base @opentelemetry/sdk-trace-node @opentelemetry/exporter-trace-otlp-http
```

**Bootstrap auto-export:** the bootstrap child calls `resolveForstRuntimeLayer()` at startup. When `OTEL_EXPORTER_OTLP_ENDPOINT` or `FORST_BRIDGE_OTEL=1` is set and the peers above are installed, Effect spans (`Rpc.dispatch`, `Runtime.handleSyncCall`, …) export through OpenTelemetry automatically.

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318/v1/traces \
OTEL_SERVICE_NAME=my-app \
FORST_BRIDGE_LOG_LEVEL=debug \
npx forst run -root ./my-service ./main.ft
```

If OTEL env is set but peers are missing, bootstrap logs a warning and continues with stderr-only spans.

**Host / embedded apps:** merge OpenTelemetry at your app boundary so you do not duplicate exporters:

```typescript
import { Layer } from "effect";
import { createNodeRuntimeSetup } from "@forst/runtime";
import {
  mergeForstNodeRuntimeWithOpenTelemetry,
  openTelemetryLayerFromEnv,
} from "@forst/runtime/opentelemetry";

const layer = mergeForstNodeRuntimeWithOpenTelemetry(
  openTelemetryLayerFromEnv(),
  { replaceLogger: false }
);
const { layer: runtimeLayer, runtime } = createNodeRuntimeSetup(
  Layer.merge(appLayer, layer)
);
```

Import helpers from `@forst/runtime/opentelemetry`. The main `@forst/runtime` entry also exports `resolveForstRuntimeLayer` for bootstrap-style resolution without static OTEL imports.

| Piece | Role |
| --- | --- |
| `bootstrap.js` | RPC server process Go spawns in bootstrap mode |
| `@forst/runtime/host` | In process RPC when your app runs the Node child |
| `forst-runtime-index` | CLI the compiler invokes to read TypeScript exports |
| Schema types | `forst-node-manifest-v1` and `forst-index-v1` validation |

## Project setup

Enable node interop in `ftconfig.json`:

```json
{
  "files": {
    "include": ["**/*.ft", "**/*.ts"]
  },
  "bridge": {
    "enabled": true,
    "runtimeEnabled": true
  }
}
```

Use opt in imports in Forst source:

```ft
import "./legacy/payment" js

func main() {
    result := payment.create(100.0, "USD")
}
```

Build and run with the Forst compiler ([`@forst/cli`](../cli/README.md)):

```bash
npx forst build -root . ./main.ft
npx forst run -root . ./main.ft
```

## Runtime modes

**Bootstrap (default):** Go starts a dedicated Node child that runs `dist/bootstrap.js`. Isolated process. RPC over a local socket (default `.forst/node-bootstrap.sock`); child stdout and stderr are forwarded as logs.

**Host:** Go starts your app (`bridge.binary` + `bridge.args`). RPC listens on a local socket inside that process so module cache and globals stay shared. Import from `@forst/runtime/host` and call `signalForstAppReady()` when your app is ready.

See [runtime modes](https://docs.forst.dev/interop/bridge/build-and-runtime#choose-how-javascript-runs) in the docs.

## CLI

```bash
forst-runtime-index --root . --format forst-index-v1 --files legacy/payment.ts
```

The compiler calls this during type checking. You rarely run it yourself.

## Environment variables

### Bootstrap and logging

| Variable | Purpose |
| --- | --- |
| `FORST_BRIDGE_LOG_LEVEL` | Log verbosity: `trace`, `debug`, `info`, `warn`, or `error` (default `info`). Per-RPC flow uses Effect spans; set `debug` or `trace` to see span annotations (`effect.spanName`, `rpc_method`, …) on log lines. |
| `FORST_BRIDGE_OTEL` | When `"1"`, enables OpenTelemetry export in bootstrap (console spans when no OTLP endpoint is set). Requires optional OTEL peer packages. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Standard OTLP traces URL (e.g. `http://localhost:4318/v1/traces`). When set, bootstrap exports RPC spans to this endpoint. |
| `OTEL_SERVICE_NAME` | OpenTelemetry service name (default `forst-runtime`). |
| `OTEL_EXPORTER_OTLP_HEADERS` | Optional OTLP headers (`key=value,key2=value2`). |
| `FORST_BRIDGE_LOG_FORMAT` | Log format: `pretty` (default) or `json` for structured stderr lines. |
| `FORST_BRIDGE_BOOTSTRAP` | Absolute path to `bootstrap.js` (bootstrap mode spawn planning) |
| `FORST_BRIDGE_MODULES_DIR` | Absolute path to compiled bridge modules (`.js` manifest IDs resolve here in compiled format) |
| `FORST_BRIDGE_SOCKET` | Absolute Unix socket path (TCP URL on Windows) for Go↔Node RPC. Bootstrap default: `{boundaryRoot}/.forst/node-bootstrap.sock`. Host default: `{boundaryRoot}/.forst/node.sock`. |
| `FORST_BRIDGE_HOST_READY` | Absolute path to JSON readiness file (`{socket}.ready`); Go waits for `phase: "app"` before dialing. |

### Host mode (set by Go on the direct app-shim child)

| Variable | Purpose |
| --- | --- |
| `FORST_BRIDGE_HOST` | When `"1"`, enables in-process host RPC (`startForstNodeHost`). Unset in bootstrap mode. |
| `FORST_BRIDGE_HOST_LEADER` | When `"1"`, marks the Go-spawned leader process. Required together with `register.mjs` in `process.execArgv`; workers skip binding. |
| `FORST_BRIDGE_APP_READY_MODULE` | Optional path to a module loaded before app readiness when `bridge.hostAppReadyModule` is configured. |

See [Environment variables for socket RPC](https://docs.forst.dev/interop/bridge/build-and-runtime#environment-variables-for-socket-rpc) for spawn layout, readiness phases, and troubleshooting.

## Development

From the monorepo:

```bash
cd packages/runtime
bun run build
bun test
```

## Publishing

Release Please tags `runtime-v*` bump `package.json` and `jsr.json`. CI publishes to npm and JSR via [.github/workflows/publish-packages.yml](https://github.com/forst-lang/forst/blob/main/.github/workflows/publish-packages.yml).

Manual publish from `packages/runtime`:

```bash
bun run build
npm publish --access public --workspaces=false
npx jsr publish
```

Dry run:

```bash
bun run pack:dry
npx jsr publish --dry-run
```

### npm trusted publishing (one-time)

Before CI can publish, add a **trusted publisher** for `@forst/runtime` on [npmjs.com](https://www.npmjs.com/package/@forst/runtime) (same settings as `@forst/cli` — each package needs its own):

- Repository: `forst-lang/forst`
- Workflow: `publish-packages.yml`
- Environment: (none, unless you use one for other packages)

Without this, CI fails with `OIDC token exchange error - package not found` / `ENEEDAUTH`. Re-publishing an already-published version is a no-op in CI.

## License

MIT. See [LICENSE](./LICENSE).
