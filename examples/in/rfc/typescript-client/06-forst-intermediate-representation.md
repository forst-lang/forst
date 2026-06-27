# RFC FR-IR-1: Forst intermediate representation (IR) as semantic source of truth

| **ID** | `FR-IR-1` |
| **Status** | Draft — **decision record** |
| **Audience** | Compiler, `forst generate`, `forst dev`, `@forst/sidecar`, ecosystem npm packages |
| **Depends on** | [03-forst-generate-contract.md](./03-forst-generate-contract.md), [05-forst-http-gateway-signature-pipeline-rfc.md](./05-forst-http-gateway-signature-pipeline-rfc.md) |
| **Informs** | [07-typescript-client-layered-architecture.md](./07-typescript-client-layered-architecture.md), [09-ecosystem-codegen-from-ir.md](./09-ecosystem-codegen-from-ir.md), [../sidecar/13-ir-driven-production-wire.md](../sidecar/13-ir-driven-production-wire.md) |

---

## 1. Abstract

This RFC **decides** that Forst will maintain a **versioned intermediate representation (IR)** of export and signature facts as the **semantic source of truth** for all TypeScript client tooling, wire-schema emitters, and ecosystem packages. The IR is **transport-agnostic**: it does not encode HTTP routes, TanStack Query cache keys, or protobuf service names. The compiler **must** emit IR from the same typechecked program model that drives Go codegen and TypeScript emission; consumers **must not** re-parse `.ft` text for signature facts when IR is available.

---

## 2. Decision

### D1 — IR is the single semantic faucet

| Choice | Rationale |
|--------|-----------|
| **Adopt** versioned on-disk IR (`forst.ir.json` or equivalent) emitted by `forst generate` and available from `forst dev` | Eliminates drift between `.ft`, `GET /types`, hand-maintained sidecar types, and future proto/OpenAPI emitters ([RFC 05 §17.1 G4](05-forst-http-gateway-signature-pipeline-rfc.md)) |
| **Reject** parallel signature discovery in npm packages by parsing `.ft` or scraping Go source | Fragile, duplicates compiler knowledge, breaks on merge packages and hash-based type names |
| **Reject** making protobuf `.proto` the primary semantic IDL for Forst application types | Forst types are richer (refinements, guards, nominal errors); protobuf is reserved for **service wire boundaries** ([sidecar/11](../sidecar/11-wire-format.md)) |

### D2 — IR schema stays transport-neutral

The core IR schema **must not** contain:

- HTTP method, path, or status slots
- Sidecar `package` / `function` invoke strings (those live in **registration config**, [RFC 05 §15](05-forst-http-gateway-signature-pipeline-rfc.md))
- Framework-specific metadata (React Query keys, tRPC procedure names)

The core IR schema **must** contain:

- `forstIrVersion` (string, semver or integer epoch)
- Compiler release identifier (optional but recommended)
- Per-export records: stable `symbolId`, source file path, export name, kind (`function` | `type` | `error` | …)
- Resolved function signatures: parameters (name, type ref), return type ref
- Type graph references: named types, unions, nominal tags, well-known stdlib refs (e.g. `forst/gateway.GatewayHandler`)
- Tags / annotations **only** when they are transport-neutral (e.g. `streaming: { elementTypeRef }`, `remote: true` if a future linkage modifier exists)

### D3 — TypeScript emission consumes IR (eventually)

**Phase 1 (transitional):** `TypeScriptTransformer` may continue to read the typechecker directly but **must** also write IR that faithfully reflects what was emitted.

**Phase 2 (target):** `types.d.ts` and `*.client.ts` are **lowered from IR**, not from ad hoc AST walks. This enables alternative emitters without forking the driver.

### D4 — `GET /types` and on-disk IR are the same contract

| Surface | Role |
|---------|------|
| `forst generate` → `generated/forst.ir.json` | CI, diff review, ecosystem packages, proto emit |
| `forst dev` → `GET /types` | Live iteration; returns merged TS **and** IR hash or embedded IR |
| Hand-edited `.d.ts` | **Discouraged** for invoke targets; allowed for app-local augmentation only |

When IR changes, `forstTypesVersion` (or equivalent header in `types.d.ts`) **must** change. Implementations **should** expose the same IR hash from CLI output and dev server.

---

## 3. IR record shape (normative outline)

Implementations **may** extend the schema with optional fields; they **must not** remove or rename required fields without bumping `forstIrVersion`.

```json
{
  "forstIrVersion": "1",
  "forstTypesVersion": "2026.06.0",
  "compilerVersion": "0.x.y",
  "packageRoot": ".",
  "exports": [
    {
      "symbolId": "engine.PlayMove",
      "kind": "function",
      "file": "engine.ft",
      "name": "PlayMove",
      "parameters": [
        { "name": "state", "typeRef": "GameState" },
        { "name": "move", "typeRef": "Move" }
      ],
      "returnTypeRef": "GameState",
      "streaming": null,
      "tags": []
    }
  ],
  "types": [
    {
      "symbolId": "GameState",
      "kind": "interface",
      "file": "engine.ft",
      "name": "GameState",
      "fields": []
    }
  ]
}
```

Exact `typeRef` resolution rules follow the typechecker’s named and hash-based types; the IR **must** use stable `symbolId` strings that do not change when unrelated files are added to the merge package.

---

## 4. Emission points

| Command / endpoint | IR required? | TS required? |
|--------------------|--------------|--------------|
| `forst generate` | **Yes** | **Yes** |
| `forst dev` `GET /types` | **Yes** (or hash + fetch full IR) | **Yes** |
| `forst build` (future) | **Yes** | Optional |
| Go-only compile (no TS consumer) | Optional | No |

Failure policy: if typecheck fails, **no IR and no TS** are written (same as [03 contract](./03-forst-generate-contract.md)).

---

## 5. Consumers (non-exhaustive)

| Consumer | Uses IR for |
|----------|-------------|
| `internal/transformer/ts` | `.d.ts`, `*.client.ts` |
| `@forst/sidecar` | Validating invoke targets against registered symbols; future typed facade |
| `@forst/react-query` (future npm) | `queryKey`, `queryFn` signatures |
| `@forst/procedures` (future npm) | tRPC-like procedure routers |
| Proto emitter ([sidecar/13](../sidecar/13-ir-driven-production-wire.md)) | `.proto` messages and RPC methods |
| OpenAPI emitter (future) | REST schemas from shapes |
| CI lint | Diff on signature change, breaking-change detection |

---

## 6. Versioning and breaking changes

- **`forstIrVersion`:** Bump on incompatible IR schema changes (field removal, semantic change).
- **`forstTypesVersion`:** Bump when TS mapping rules change ([03 contract](./03-forst-generate-contract.md)); may align with `FORST_TS_CONTRACT` when introduced.
- **`contractVersion`** ([02 HTTP contract](./02-forst-dev-http-contract.md)): Remains the **wire** version for `POST /invoke`; independent of IR but **must** be updated in lockstep when gateway `result` JSON changes.

---

## 7. Non-goals

- Full program AST serialization in IR
- Storing inferred constraint proofs or guard expressions as executable code
- LSP protocol as IR transport
- Browser-consumable IR without a build step (apps use generated TS)

---

## 8. Conformance

Implementations claiming FR-IR-1 conformance **must**:

1. Emit IR on successful `forst generate` for merge packages.
2. Include `forstIrVersion` and at least one export record per public function intended for remote invoke.
3. Pass golden tests: IR snapshot for fixture projects matches committed `testdata` after intentional updates.
4. Document IR location and versioning in release notes when schema changes.

---

## 9. Phased implementation

| Phase | Deliverable |
|-------|-------------|
| **IR-0** | JSON dump of signatures from existing transformer (flat outline §3 OK) |
| **IR-1** | §12 graph schema + `contentHash` + CI snapshot tests; `GET /types` returns hash |
| **IR-2** | TS emitter reads IR graph; sidecar validates invoke targets |
| **IR-3** | Proto/OpenAPI emitters; ecosystem npm packages |

---

## 10. Related documents

- [05-forst-http-gateway-signature-pipeline-rfc.md](./05-forst-http-gateway-signature-pipeline-rfc.md) — IR architecture principles (§14)
- [07-typescript-client-layered-architecture.md](./07-typescript-client-layered-architecture.md) — what consumes IR vs what stays in npm
- [08-generated-client-runtime-integration.md](./08-generated-client-runtime-integration.md) — typed client stubs from IR/TS
- [../optionals/03-typescript-emission-optionals-and-result.md](../optionals/03-typescript-emission-optionals-and-result.md) — type mapping into IR/TS

---

## 11. Prior art — how other languages structure “IR” for clients

Forst’s IR is **not** a compiler lowering IR (LLVM bitcode, Rust MIR, TypeScript checker internal types). It is a **public, versioned export/signature graph** — closer to cross-language IDL metadata. Comparable systems:

| System | What it stores | Structure | Lessons for Forst |
|--------|----------------|-----------|-------------------|
| **Protobuf `FileDescriptorSet`** | Messages, enums, services, field numbers | Cross-linked descriptor graph; binary + JSON | One schema → many emitters (Go, TS Connect); **wire** layer separate from app types |
| **WIT `Resolve`** ([wit-parser](https://docs.rs/wit-parser/latest/wit_parser/struct.Resolve.html)) | Packages, interfaces, worlds, type defs | **Arena + index IDs**; types topologically sorted; packages merged | **Index-based type refs**, not string names; **worlds** = import/export contract (analog: registration config, not core IR) |
| **Swift Symbol Graph** ([SymbolKit](https://github.com/swiftlang/swift-docc-symbolkit)) | Module declarations | **Nodes (symbols) + edges (relationships)**; symbols never nest | Split **symbols** from **relationships**; stable `precise` identifier + human path |
| **Prisma DMMF** | Datamodel AST as JSON | Flat JSON from Rust engine; drives all generators | **Single faucet** pattern matches Forst; DMMF was **internal/unstable** — Forst IR must be **public + versioned from v1** |
| **TypeScript compiler** | Full program types | **No public JSON IR**; consumers use Compiler API or `.d.ts` emit | Ecosystem reads **declarations**, not AST — Forst ships IR **and** `.d.ts` |
| **OpenAPI 3** | Paths + `components/schemas` | Document with reusable schema components | **Paths are transport** — belong outside core IR ([D2](#d2--ir-schema-stays-transport-neutral)) |
| **Rust `.rmeta`** | Incremental compile metadata | Binary, compiler-internal | Too low-level; not a model for client IR |

**Patterns to adopt**

1. **Graph, not nested tree** — types and exports as nodes; references by stable ID (WIT arenas, Symbol Graph edges, protobuf descriptors).
2. **Topological type ordering** — dependents after dependencies (WIT `types` arena).
3. **Contract surface separate from package graph** — WIT *worlds* (imports/exports) ↔ Forst **registration manifest** in `ftconfig.json`, not in core IR.
4. **Derivative wire IDL** — protobuf `.proto` emitted **from** IR ([SC-WIRE-2](../sidecar/13-ir-driven-production-wire.md)), not co-equal source of truth.
5. **Public contract** — unlike Prisma DMMF, document schema, semver `forstIrVersion`, and breaking-change policy in release notes.

**Patterns to reject**

1. **String-only `typeRef`** without interned IDs — breaks merge packages and hash-based structural types.
2. **HTTP paths / invoke strings in core IR** — OpenAPI-style coupling; use sidecar registration config.
3. **Full AST or guard expression bytecode** — compiler-internal; consumers need signatures only.
4. **Framework slots in driver IR** — TanStack keys, React props; ecosystem npm reads neutral IR.

---

## 12. Recommended Forst IR structure (normative target)

Replace the flat outline in §3 with a **three-blob model**: **meta**, **graph** (types + symbols), **relationships**. Registration and transport stay **outside** the file or in companion config.

### 12.1 Top-level envelope

```json
{
  "forstIrVersion": "1",
  "meta": {
    "forstTypesVersion": "1",
    "compilerVersion": "0.x.y",
    "packageRoot": ".",
    "contentHash": "sha256:…",
    "inputs": ["engine.ft", "server.ft"]
  },
  "graph": { "packages": [], "types": [], "symbols": [] },
  "relationships": []
}
```

| Field | Role |
|-------|------|
| `meta.contentHash` | Same bytes from CLI and `GET /types`; enables drift detection ([D4](#d4--gettypes-and-on-disk-ir-are-the-same-contract)) |
| `graph` | All definitional nodes |
| `relationships` | Optional edges (implements, memberOf, returns, uses) for discovery and docs |

### 12.2 Packages (merge-unit indexing)

One entry per **Forst merge stem / logical package** (matches `*.client.ts` factory and invoke `package` string):

```json
{ "id": 0, "name": "engine", "rootFile": "engine.ft" }
```

- `id`: **u32** arena index, stable within one IR document
- `name`: invoke/sidecar package string
- WIT analogue: `Package` in `Resolve`

### 12.3 Types arena (interned, topologically sorted)

Types **must** appear before any reference to their `id`. Use **index refs**, not bare names, for structural/hash types:

```json
{
  "id": 2,
  "packageId": 0,
  "kind": "record",
  "name": "GameState",
  "symbolId": "engine.GameState",
  "fields": [
    { "name": "board", "typeId": 1, "optional": false }
  ],
  "source": { "file": "engine.ft" }
}
```

| `kind` | Maps to |
|--------|---------|
| `record` | shape / interface |
| `union` | binary sum / discriminated union |
| `alias` | typedef |
| `nominal_error` | `error X { … }` |
| `builtin` | `string`, `int`, … |
| `applied` | `[]T`, `map[K,V]`, `*T`, `Result<T,E>`, `chan T` with `args: [typeId…]` |

- **`symbolId`**: stable string for humans, CI, registration (`engine.GameState`)
- **`typeId`**: index into `types[]` for edges and signatures
- WIT analogue: `TypeDef` arena; Protobuf analogue: message descriptor index

### 12.4 Symbols arena (exports / callable surface)

```json
{
  "id": 0,
  "symbolId": "engine.PlayMove",
  "kind": "function",
  "packageId": 0,
  "name": "PlayMove",
  "signature": {
    "parameters": [
      { "name": "state", "typeId": 2 },
      { "name": "move", "typeId": 3 }
    ],
    "returnTypeId": 2,
    "streaming": null
  },
  "tags": [],
  "source": { "file": "engine.ft" }
}
```

| `kind` | Use |
|--------|-----|
| `function` | Remote-invokable exports |
| `type_export` | Re-exported type only |

**`tags`** (transport-neutral, optional): `remote`, `streaming`, future linkage modifiers ([RFC 05 §17.4](../typescript-client/05-forst-http-gateway-signature-pipeline-rfc.md)). **Not** HTTP paths or query keys.

**Gateway discovery:** emit relationship `implements` → well-known `forst/gateway.GatewayHandler` instead of duplicating handler shape in tags.

### 12.5 Relationships (Symbol Graph pattern)

```json
{
  "kind": "implements",
  "sourceSymbolId": "server.handlePlay",
  "target": { "wellKnown": "forst/gateway.GatewayHandler" }
}
```

| `kind` | Purpose |
|--------|---------|
| `implements` | Gateway handler, future trait bounds |
| `usesType` | Symbol → type dependency (emit ordering, proto) |
| `memberOf` | Nested declarations (optional) |

Symbols **must not** embed other symbols inline ([Symbol Graph rule](https://github.com/swiftlang/swift-docc-symbolkit)).

### 12.6 What lives outside `forst.ir.json`

| Concern | Location | Analogue |
|---------|----------|----------|
| `(package, function)` invoke allowlist | `ftconfig.json` → `compiler.invokeTargets` (name TBD) | WIT **world** imports/exports |
| HTTP routes, Express paths | App TS + sidecar middleware | OpenAPI **paths** |
| TanStack `queryKey` | `@forst/react-query` generator | — |
| `.proto` services | `generated/forst.proto` from IR | Protobuf **services** layer |

---

## 13. Emitter and consumer rules

1. **Producer (compiler):** Build IR from typechecker **after** merge typecheck; assign `typeId` / `symbolId`; compute `contentHash`; write JSON (pretty in dev, compact in CI optional).
2. **TS emitter:** Lower `graph.types` + `graph.symbols` → `types.d.ts` + `*.client.ts` ([FR-CLIENT-1](./08-generated-client-runtime-integration.md)).
3. **Proto emitter:** Walk `symbols` with `tags` containing `remote` + `types` arena → messages/RPC ([SC-WIRE-2](../sidecar/13-ir-driven-production-wire.md)).
4. **Sidecar:** Validate `POST /invoke` `(package, function)` against registration config **and** `graph.symbols` entry.
5. **Ecosystem npm:** Read IR only; never `.ft` ([FR-ECO-1](./09-ecosystem-codegen-from-ir.md)).

### 13.1 Merge semantics (multi-file `forst generate`)

When merging per-file chunks ([`merge.go`](../../../../forst/internal/transformer/ts/merge.go)):

- **Types:** Dedupe by `symbolId` if structurally identical; **error** on same `symbolId` different shape (same as TS merge today).
- **Symbols:** Dedupe by `symbolId`; **error** on signature conflict.
- **IDs:** Reindex into single arena per output document; `contentHash` covers full merged IR.

---

## 14. Migration from §3 flat outline

§3 remains a **minimal illustration**. Implementations **should** target §12 for IR-1. Transitional IR-0 may emit flat `exports` + `types` with string `typeRef`; IR-1 **must** adopt index-based `typeId` graph before proto or ecosystem packages ship.

---

## 15. Open points

- Binary IR encoding (protobuf meta-schema for IR itself?) — defer until JSON schema stabilizes
- Source spans in IR for LSP/diagnostics ([RFC 05 L8](./05-forst-http-gateway-signature-pipeline-rfc.md))
- Whether `symbolId` uses `package.Name` or file stem — **decision:** `package.name` from ftconfig merge stem, matching invoke wire today
