# Node interop host mode example

Fixture for **host mode**: Go spawns an app shim (`node app/server.mjs`) and connects via a Unix socket. Auto-injected `register.mjs` starts the host; the server entry calls `signalForstAppReady()` after seeding shared state.

## Layout

| Path | Purpose |
| --- | --- |
| `ftconfig.json` | `node.hostMode: true`, `node.args` for the app shim |
| `app/server.mjs` | Mock app entry — `signalForstAppReady()` after init |
| `legacy/counter.ts` | TypeScript export reading `globalThis.__forstTest` |
| `main.ft` | Forst entry calling `counter.inc` at runtime |

## Host environment variables

Go sets `FORST_NODE_HOST`, `FORST_NODE_HOST_LEADER`, `FORST_NODE_SOCKET`, and `FORST_NODE_HOST_READY` on the direct shim child. See [Call JavaScript from Forst — host mode environment variables](https://docs.forst.dev/interop/node/call-javascript#host-mode-environment-variables) for semantics, readiness phases, and worker vs leader behavior.

## Compile

From repo root:

```bash
task example:node-interop-host
```

Requires `@forst/node-runtime` built (`task build:node-runtime`).

## Run (manual)

Integration tests in `forst/nodert/host_integration_test.go` cover E2E host mode with strict readiness ordering.

## See also

- [remix-serve](../remix-serve/) — third-party shim with `hostAppReadyModule`
- [sync](../sync/) — bootstrap mode (dedicated Node child, stdio RPC)
- [Call JavaScript from Forst](/interop/node/call-javascript) — host vs bootstrap comparison
