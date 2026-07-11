# @forst/node-runtime

Node runtime for **Forst → TypeScript** interop. Compiled Go binaries call your legacy `.ts` and `.js` modules over a closed RPC channel. The Forst compiler uses this package at build time to index TypeScript exports.

**Status:** experimental. Pin this package and verify with the [examples](https://github.com/forst-lang/forst/tree/main/examples/in/rfc/node-interop) before production use.

[Full guide → Call JavaScript from Forst](https://docs.forst.dev/interop/node/call-javascript)

## Install

```bash
npm install @forst/node-runtime
```

```bash
npx jsr add @forst/node-runtime
```

Requires **Node.js 18+**. When your Forst program uses `import node`, you also need **tsx** on the path for TypeScript loading.

| Registry | Package |
| --- | --- |
| npm | [@forst/node-runtime](https://www.npmjs.com/package/@forst/node-runtime) |
| JSR | [@forst/node-runtime](https://jsr.io/@forst/node-runtime) |

## What you get

| Piece | Role |
| --- | --- |
| `bootstrap.js` | RPC server process Go spawns in bootstrap mode |
| `@forst/node-runtime/host` | In process RPC when your app runs the Node child |
| `forst-node-index` | CLI the compiler invokes to read TypeScript exports |
| Schema types | `forst-node-manifest-v1` and `forst-index-v1` validation |

## Project setup

Enable node interop in `ftconfig.json`:

```json
{
  "files": {
    "include": ["**/*.ft", "**/*.ts"]
  },
  "node": {
    "enabled": true,
    "runtimeEnabled": true
  }
}
```

Use opt in imports in Forst source:

```ft
import node "./legacy/payment"

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

**Bootstrap (default):** Go starts a dedicated Node child that runs `dist/bootstrap.js`. Isolated process. Logs on stderr. Stdout is RPC only.

**Host:** Go starts your app (`node.binary` + `node.args`). RPC listens on a local socket inside that process so module cache and globals stay shared. Import from `@forst/node-runtime/host` and call `signalForstAppReady()` when your app is ready.

See [runtime modes](https://docs.forst.dev/interop/node/call-javascript#runtime-modes-bootstrap-vs-host) in the docs.

## CLI

```bash
forst-node-index --root . --format forst-index-v1 --files legacy/payment.ts
```

The compiler calls this during type checking. You rarely run it yourself.

## Environment variables

| Variable | Purpose |
| --- | --- |
| `FORST_NODE_BOOTSTRAP` | Absolute path to `bootstrap.js` |
| `FORST_NODE_HOST` | Set by Go when host mode is active |
| `FORST_NODE_SOCKET` | Socket path for host mode RPC |
| `FORST_NODE_HOST_READY` | Ready file path for host mode |

## Development

From the monorepo:

```bash
cd packages/node-runtime
bun run build
bun test
```

## Publishing

Release Please tags `node-runtime-v*` bump `package.json` and `jsr.json`. CI publishes to npm and JSR via [.github/workflows/publish-packages.yml](https://github.com/forst-lang/forst/blob/main/.github/workflows/publish-packages.yml).

Manual publish from `packages/node-runtime`:

```bash
bun run build
npm publish --access public
npx jsr publish
```

Dry run:

```bash
bun run pack:dry
npx jsr publish --dry-run
```

## License

MIT. See [LICENSE](./LICENSE).
