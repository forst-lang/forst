# Embedded invoke example

Compiled Go binary exposes `POST /invoke` on `http://127.0.0.1:8081` (no `forst dev`, no per-call `go run`).

## Compile

```bash
task example:embedded-invoke
```

## Run

```bash
task example:embedded-invoke:run
```

In another terminal:

```bash
curl -s -X POST http://127.0.0.1:8081/invoke \
  -H 'Content-Type: application/json' \
  -d '{"package":"main","function":"Echo","args":[{"message":"hello"}]}'
```

## Host mode + embedded invoke

In a host-mode deployment (e.g. `remix-serve`), three channels coexist:

| Port / socket | Direction |
| --- | --- |
| `:3000` | Browser → Remix |
| `:8081` | TS loader → Forst (`POST /invoke`) |
| `.forst/node.sock` | Forst → legacy TS (nodert) |

Enable both `server.embedded` and `node.hostMode` in `ftconfig.json`. Call **`ForstInvokeWaitForShutdown()`** in `main` so the invoke server stays up while the Node host runs.

## Generated client

```bash
forst generate ./examples/in/rfc/embedded-invoke
FORST_BASE_URL=http://127.0.0.1:8081 node -e "
  const { createInvokeClient } = require('@forst/client');
  // use generated client/index.ts after forst generate
"
```
