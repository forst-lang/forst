# Node interop host mode example

Fixture for **host mode**: Go spawns an app shim (`node app/server.mjs`) and connects via a Unix socket. Auto-injected `register.mjs` starts the host; the server entry calls `signalForstAppReady()` after seeding shared state.

## Layout

| Path | Purpose |
| --- | --- |
| `ftconfig.json` | `node.hostMode: true`, `node.args` for the app shim |
| `app/server.mjs` | Mock app entry — `signalForstAppReady()` after init |
| `legacy/counter.ts` | TypeScript export reading `globalThis.__forstTest` |
| `main.ft` | Forst entry calling `counter.inc` at runtime |

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
