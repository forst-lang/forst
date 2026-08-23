# Node interop generators example (Phase 4)

Fixture for **Phase 4** generator RPC: compiled Go binary iterates TypeScript `function*` and `async function*` exports via `genOpen` / `genNext` / `genClose`.

## Layout

| Path | Purpose |
| --- | --- |
| `ftconfig.json` | Boundary with `node.runtimeEnabled: true` |
| `legacy/generators.ts` | Ordinary sync and async generator exports |
| `main.ft` | Forst entry iterating generator exports |

## Run

From repo root:

```bash
task example:bridge-interop-generators
```

Requires `@forst/runtime` built (`task build:runtime`).

## See also

- [sync](../sync/) — Phase 2 sync RPC
- [phase-4 plan](../../../../.plans/bridge-interop/plan/phase-4-generators.md)
