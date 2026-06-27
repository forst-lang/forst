# RFC SC-WIRE-2: IR-driven production wire (protobuf from Forst)

| **ID** | `SC-WIRE-2` |
| **Status** | Draft — **decision record** |
| **Audience** | Sidecar maintainers, compiler team, teams deploying TS ↔ Go in production |
| **Depends on** | [11-wire-format.md](./11-wire-format.md), [../typescript-client/06-forst-intermediate-representation.md](../typescript-client/06-forst-intermediate-representation.md), [../typescript-client/07-typescript-client-layered-architecture.md](../typescript-client/07-typescript-client-layered-architecture.md) |
| **Supersedes** | Informal “hand-write `.proto` alongside Forst” workflows |

---

## 1. Abstract

This RFC **decides** that **production wire schemas** (Protocol Buffers) for the TypeScript ↔ Forst/Go boundary **must be derived from [FR-IR-1 IR](../typescript-client/06-forst-intermediate-representation.md)**, not maintained as a parallel hand-written IDL. JSON-over-HTTP via `forst dev` remains the **development MVP** ([11-wire-format.md](./11-wire-format.md)). Production deployments **should** use **Connect** (TypeScript-first) or **gRPC** (Go-first) generated from the same `.proto` files emitted from IR.

---

## 2. Context

[11-wire-format.md](./11-wire-format.md) already chooses protobuf + gRPC/Connect as the **product direction** for service-grade integrations. What was **undecided** is **where `.proto` definitions come from** and how they stay aligned with Forst types and `forst generate` output.

Without this decision, teams duplicate schemas (Forst shapes + manual `.proto` + `.d.ts`), causing the same G4 drift [RFC 05 §17.1](../typescript-client/05-forst-http-gateway-signature-pipeline-rfc.md) identifies for TypeScript.

---

## 3. Decisions

### D1 — IR → proto is the canonical production path

| Stage | Owner | Output |
|-------|-------|--------|
| Typecheck + IR emit | Forst compiler | `forst.ir.json` |
| Proto emit | `forst proto` or `forst generate --proto` (name TBD) | `generated/forst.proto` (+ imports) |
| Go codegen | `protoc` / buf | Go structs + gRPC/Connect service |
| TS codegen | `@connectrpc/protoc-gen-connect-es` or equivalent | Connect client |

**Reject:** Hand-written `.proto` as the **source of truth** for symbols that originate in `.ft`.

**Allow:** Hand-written `.proto` **only** for wire types with **no** Forst counterpart (infra metadata, auth tokens) in separate proto packages.

### D2 — Dev vs prod transport profiles

| Profile | Transport | Schema source |
|---------|-----------|---------------|
| **Local dev** | JSON `forst dev` + `@forst/sidecar` | IR/TS only; no proto required |
| **Production Node ↔ Go** | Connect (default TS client) or gRPC | IR-derived proto |
| **Production Go ↔ Go** | gRPC | IR-derived proto |

**Decision:** `@forst/sidecar` JSON client **remains** the reference dev transport indefinitely. Production wire **does not** require sidecar JSON.

### D3 — Connect vs gRPC default

Align with [11-wire-format.md §Choosing Between the Two Variants](./11-wire-format.md):

| Primary consumer | Default RPC stack |
|------------------|-------------------|
| TypeScript / Node backend | **Connect** |
| Go-only internal service | **gRPC** |
| Both | Same `.proto`; implement both servers if needed |

Forst project **does not** pick a single RPC stack globally; it picks **one proto source** (IR) and **two transport options**.

### D4 — Mapping rules (outline)

Proto emit **must** document mapping from IR type refs:

| Forst / IR | Protobuf |
|------------|----------|
| `string`, `int`, `float`, `bool` | `string`, `int64`/`int32`, `double`, `bool` |
| Named struct / interface | `message` |
| Nominal error types | `message` with `forst_tag` field or `google.rpc.Status` pattern (TBD) |
| `Result<T,E>` on wire | **Not** automatic — use explicit request/response messages per RPC |
| `GatewayRequest` / `GatewayResponse` | Dedicated messages; **not** reused for arbitrary domain RPCs |
| Streaming (`chan T`) | gRPC server streaming / Connect streaming when supported |

**Decision:** Domain RPCs use **explicit request/response messages** per function in IR, not raw JSON blob fields — except gateway tunneling which may use opaque bytes for HTTP passthrough.

### D5 — Versioning

| Artifact | Version field |
|----------|---------------|
| IR | `forstIrVersion` |
| Proto package | `forst.wire.v1` (proto package name) |
| Connect/gRPC breaking change | Bump proto package or use protobuf field reservation |

JSON `contractVersion` ([02 HTTP contract](../typescript-client/02-forst-dev-http-contract.md)) is **independent** from proto package version.

### D6 — Relationship to `forst generate` TypeScript

| Layer | Technology |
|-------|------------|
| App types in TS repos | `types.d.ts` from IR (always) |
| Dev invoke | JSON sidecar |
| Prod service calls | Connect/gRPC clients from **same IR** |

TypeScript app code **may** use both `import type { GameState } from "./generated/types"` and `import { GameService } from "./generated/connect/game_connect"` — they **must** originate from one IR generation run.

---

## 4. Non-goals

- Replacing REST APIs for public browser clients (OpenAPI path is separate, [FR-ECO-1](../typescript-client/09-ecosystem-codegen-from-ir.md))
- Browser gRPC without proxy
- IPC binary wire (future RFC)
- Automatic migration from JSON sidecar to Connect in dev (opt-in)

---

## 5. Tooling layout (target)

```
project/
  generated/
    forst.ir.json      # FR-IR-1
    types.d.ts           # FR-CLIENT-1
    forst.proto          # SC-WIRE-2
    connect/             # protoc output (TS)
    go/                  # protoc output (Go) — or colocated in Go module
```

CI **should** run `buf breaking` or equivalent against committed proto baselines when IR changes.

---

## 6. Phasing

| Phase | Deliverable |
|-------|-------------|
| **WIRE-0** | Document mapping; manual proto from IR spec for one fixture |
| **WIRE-1** | `forst proto emit` experimental; golden proto snapshots |
| **WIRE-2** | Connect client example in `examples/` |
| **WIRE-3** | Production sidecar/native service template without JSON |

**Gate:** [FR-IR-1 IR-1](../typescript-client/06-forst-intermediate-representation.md) before WIRE-1.

---

## 7. Conformance

Claiming SC-WIRE-2 conformance **requires**:

1. Proto files for listed remote symbols are **generated**, not hand-edited, in official examples.
2. A fixture demonstrates Connect client calling Go server with types matching `types.d.ts` field names.
3. Release notes document proto package bumps.

---

## 8. Related documents

- [11-wire-format.md](./11-wire-format.md) — JSON vs protobuf tradeoffs
- [10-decisions.md](./10-decisions.md) — HTTP-first MVP
- [../typescript-client/06-forst-intermediate-representation.md](../typescript-client/06-forst-intermediate-representation.md)
- [../typescript-client/07-typescript-client-layered-architecture.md](../typescript-client/07-typescript-client-layered-architecture.md)
- [../typescript-client/02-forst-dev-http-contract.md](../typescript-client/02-forst-dev-http-contract.md)
