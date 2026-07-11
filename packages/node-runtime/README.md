# @forst/node-runtime

Node.js runtime bootstrap and TypeScript indexer for **Forst Node interop**.

This package is Phase 0 scaffolding: manifest/index contracts, package wiring, and stub CLI/bootstrap entrypoints. RPC, spawn, and real indexing land in later phases.

---

## Status

| Component | Phase 0 |
| --- | --- |
| `forst-node-manifest-v1` schema (TS) | Types + validation |
| `forst-index-v1` schema (TS) | Types + validation |
| `forst-node-index` CLI | Stub (exit 1, not implemented) |
| Bootstrap (`bootstrap.ts`) | Stub (throws) |
| ts-morph indexer | Dependency only |

---

## Installation

In the Forst monorepo:

```bash
bun install
cd packages/node-runtime
bun run build
bun test
```

---

## Manifest contract (`forst-node-manifest-v1`)

Compile-time execution allowlist embedded in Go when `needsNodeRuntime` is true:

```json
{
  "version": 1,
  "boundaryRoot": "/abs/path/to/project",
  "exports": [
    { "moduleId": "legacy/payment.ts", "name": "create", "kind": "asyncFunction" },
    { "moduleId": "legacy/payment.ts", "name": "watchEvents", "kind": "asyncGenerator" }
  ]
}
```

Export `kind` values: `function` | `asyncFunction` | `generator` | `asyncGenerator`.

---

## Index contract (`forst-index-v1`)

Per-module TypeScript index consumed by the compiler (future indexer output):

```json
{
  "moduleId": "legacy/payment.ts",
  "exports": [
    {
      "name": "create",
      "kind": "asyncFunction",
      "parameters": [
        { "name": "amount", "type": { "kind": "number" } }
      ],
      "returnType": { "kind": "object", "fields": { "id": { "kind": "string" } } }
    }
  ]
}
```

---

## CLI (stub)

```bash
forst-node-index --root $BOUNDARY --format forst-index-v1 --files payment.ts
```

Phase 0 prints `forst-node-index: not implemented (Phase 0 scaffold)` and exits with code `1`.

---

## Development

```bash
bun run build      # tsc → dist/
bun test           # bun test
bun run typecheck  # tsc --noEmit
```

---

## See also

- [Node interop SPEC](../../.plans/node-interop/SPEC.md)
- [Phase 0 plan](../../.plans/node-interop/plan/phase-0-scaffolding.md)
