# Node interop promises example (Phase 3)

Fixture for **Phase 3** async Node RPC: compiled Go binary calls ordinary TypeScript `export async function` exports via `forst.node/callAsync`.

## Layout

| Path | Purpose |
| --- | --- |
| `ftconfig.json` | Boundary with `node.runtimeEnabled: true` |
| `legacy/payment.ts` | Ordinary async TypeScript exports (no special annotations) |
| `main.ft` | Forst entry calling `payment.create` and `payment.concurrentEcho` |

## Run

From repo root:

```bash
task example:bridge-interop-promises
```

Requires `@forst/runtime` built (`task build:runtime`) and Phase 3 `CallAsync` wiring in the compiler.

## See also

- [sync](../sync/) — sync RPC E2E (Phase 2)
- [phase-3 plan](../../../../.plans/bridge-interop/plan/phase-3-promises.md)
