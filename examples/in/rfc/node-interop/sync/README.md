# Node interop sync example (runtime E2E)

Fixture for **Phase 2** synchronous Node RPC: compiled Go binary calls `legacy/payment.ts` via `@forst/node-runtime` protobuf RPC.

## Layout

| Path | Purpose |
| --- | --- |
| `ftconfig.json` | Boundary with `node.runtimeEnabled: true` |
| `legacy/payment.ts` | Sync TypeScript export (`create`) |
| `main.ft` | Forst entry calling `payment.create` at runtime |

## Run

From repo root:

```bash
task example:node-interop-sync
```

Requires `@forst/node-runtime` built (`task build:node-runtime`) and Phase 2 `nodert` wiring in the compiler.

## See also

- [compile-time fixture](../) — Phase 1
- [phase-2 plan](../../../../.plans/node-interop/plan/phase-2-node-runtime.md)
