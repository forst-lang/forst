# Wire Format: JSON, Protobuf, and RPC Variants

### [00-sidecar.md](00-sidecar.md) - Back to Sidecar RFC

## Overview

The TypeScript ↔ Forst sidecar boundary is a **service-style** API: versioned contracts, codegen on both sides, and (often) high call frequency. **JSON** remains useful for **bootstrap tooling**, **human debugging**, and **simple HTTP** handlers, but it is **not** the only—or always the best—encoding for production.

This document adopts **Protocol Buffers (protobuf)** as the **contract-first** IDL for service-grade integrations, with **two RPC variants** that are both idiomatic in **Go** and well supported in **TypeScript/Node**:

1. **gRPC** — protobuf over **HTTP/2**, the default stack for Go microservices.
2. **Connect** — protobuf over **HTTP** (including HTTP/1.1 and HTTP/2), with a straightforward **Connect protocol** and strong **TypeScript** client ergonomics.

Other stacks (e.g. **gRPC-Web** behind an Envoy-style proxy) remain compatible as long as they share the same **`.proto`** definitions.

**Related:** [05-observability.md](./05-observability.md) (trace context is carried in **metadata/headers**, independent of body encoding).

---

## Why Protobuf for This Boundary

| Concern | JSON | Protobuf |
|--------|------|----------|
| **Payload size & speed** | Larger, slower parse/generate | Compact binary, fast codegen paths in Go |
| **Contracts** | Often informal or duplicated | **Single `.proto`**, generated Go + TS types |
| **Evolution** | Breakage-prone without discipline | Field numbers, optional/required patterns, tooling |
| **Go ecosystem** | `encoding/json` everywhere | **`grpc`**, `protobuf` — de facto for services |
| **Debuggability** | Excellent raw readability | Use `grpcurl`, Connect devtools, or JSON **transcoding** where needed |

JSON may still be used for **MVP HTTP** handlers and **curl-friendly** demos; the product direction for **typed, high-throughput** paths is **protobuf + one of the RPC variants below**.

---

## Variant 1: gRPC (Protobuf over HTTP/2)

**Description:** Standard **gRPC** services: protobuf messages, HTTP/2, unary and streaming RPCs, rich **metadata** for auth and **trace context** propagation (`grpc-trace-bin` / W3C via metadata conventions).

**Strengths**

- **Native Go**: `google.golang.org/grpc` is the mainstream choice for internal services.
- **Performance**: Multiplexing, flow control, efficient binary payloads.
- **Features**: Client/server streaming when Forst workflows need them.

**Trade-offs**

- **Node/TS clients** rely on `@grpc/grpc-js` (or similar); slightly heavier than thin HTTP `fetch`.
- **Browsers** require **gRPC-Web** and usually a proxy—fine for admin UIs, less ideal as the default for every browser client.

**When to prefer**

- Sidecar is treated like an **internal microservice** on the same host or LAN.
- Teams already standardize on **gRPC** for Go services.

---

## Variant 2: Connect (Protobuf over HTTP)

**Description:** **[Connect](https://connectrpc.com/)** (or equivalent **protobuf-over-HTTP** RPC) exposes RPCs as **plain HTTP** with `Content-Type: application/proto` (or JSON **codec** for debug). Unary and limited streaming map cleanly to **fetch** and Node APIs.

**Strengths**

- **TypeScript ergonomics**: Generated clients, minimal ceremony, works well with **Node** and **browser** (where CORS applies).
- **Operations**: Often easier to **inspect** than raw gRPC (HTTP/1.1-friendly paths, standard load balancers).
- **Same `.proto` files** as gRPC: **one schema**, two transports if both are implemented.

**Trade-offs**

- **Ecosystem** is smaller than gRPC’s in Go, but **Connect** provides a **Go server** that is production-ready for many teams.
- **Feature parity** with gRPC streaming models should be validated per release (check current Connect capabilities for streaming needs).

**When to prefer**

- **Primary** client is **TypeScript** (Node or browser) and you want **minimal** RPC friction.
- You want **HTTP-shaped** observability (path, method, status) without giving up **protobuf** bodies.

---

## Choosing Between the Two Variants

| Criterion | Prefer **gRPC** | Prefer **Connect** |
|-----------|-----------------|---------------------|
| **Existing org standard** | Already gRPC everywhere | HTTP-first APIs |
| **Primary client** | Another Go service | **TypeScript** app |
| **Browser** | Via gRPC-Web + proxy | **Connect** often simpler |
| **Streaming** | Mature gRPC streaming | **Evaluate** Connect streaming support for your version |
| **Debugging** | `grpcurl`, rich tooling | **curl** + JSON codec option |

Implementations may support **both** behind the same generated **business logic**: one **protobuf** module, two **transport adapters** (gRPC server + Connect server).

---

## Protobuf ↔ JSON for tooling and development

Binary protobuf on the wire is efficient but awkward for **quick experiments**, **support**, and **docs**. The intended story is **not** a separate hand-written JSON schema: it is the **standard protobuf JSON mapping** (often called **proto JSON**), where the **same `.proto`** defines both encodings.

### Single contract, two encodings

- **Go**: `google.golang.org/protobuf/encoding/protojson` (`Marshal` / `Unmarshal`).
- **TypeScript**: generated code from `protoc` / Buf typically includes JSON helpers, or use the same mapping rules via Connect’s **JSON codec** when available.
- **Rules**: Field names are **JSON names** from the schema (often `camelCase` per `json_name`), **enums** as strings, **`int64`/`uint64`** as strings in JSON to avoid precision loss—teams should treat the **language-specific protojson docs** as normative, not arbitrary REST JSON.

This keeps **tooling** and **production** aligned: curl and notebooks talk **JSON**; services still **compile** from one `.proto`.

### Development and operator workflows

| Use case | Typical approach |
|----------|------------------|
| **Ad-hoc RPC from terminal** | **`grpcurl`** with `-d` and a **JSON** payload (mapped to the request message); reflection or packaged `.proto` files. |
| **Explore APIs** | **grpcui**, **Buf Studio**, **Postman gRPC**, or **Connect** clients with **JSON** request bodies in dev. |
| **Scripting / CI** | Same **proto JSON** fixtures checked into `testdata/`; assert responses using JSON **or** decoded protobuf for stability. |
| **Local sidecar** | Optional **dev-only** mode: Connect **JSON codec**, or a **second listener** that accepts `application/json` for unary RPCs (still generated from `.proto`). |
| **Gateways** | **gRPC JSON transcoding** (e.g. Envoy/Cloud endpoints) exposes HTTP/JSON to browsers while backends stay gRPC+protobuf—same protos, not a forked API. |

### What to avoid

- Maintaining **parallel** “REST JSON” types that **drift** from the `.proto`.
- Treating **JSON** as the **canonical** wire format for **high-QPS** production paths without profiling (see [04-tradeoffs.md](./04-tradeoffs.md)).
- Logging **full** JSON bodies in production without **redaction** and **size** limits—prefer structured fields and trace correlation ([05-observability.md](./05-observability.md)).

### Summary for this story

- **Production default**: binary **protobuf** over **gRPC** or **Connect**.
- **Tooling / dev default**: **Proto JSON** for the **same messages**, via official mappings and off-the-shelf clients—so debugging stays **readable** without splitting the contract.

---

## Observability and Metadata

- **Distributed tracing** (W3C `traceparent`, baggage) should ride in **RPC metadata** (gRPC) or **HTTP headers** (Connect), not in the protobuf **message** unless the product explicitly requires replay without headers.
- **Span attributes** for payloads should use **byte size** or **redacted** summaries, not assume JSON (e.g. `protobuf.size` or `grpc.request.size_bytes`).

---

## MVP vs Production

1. **MVP / dev**: JSON-over-HTTP or JSON-over-IPC is acceptable for **velocity** and **debugging** (see [10-decisions.md](./10-decisions.md) HTTP-first strategy). Prefer **proto-mapped JSON** (above) when the API is already defined by `.proto`, so dev does not fork the contract.
2. **Production hot paths**: Prefer **binary protobuf** + **gRPC or Connect** for **contracts**, **performance**, and alignment with **Go service** norms.
3. **Migration**: Same `.proto` backs binary and JSON encodings; **admin/debug** surfaces may stay JSON-only behind auth while core traffic stays protobuf.

---

## Summary

- **Protobuf** is the **recommended contract layer** for service-style integration between TypeScript and Forst/Go.
- **gRPC** and **Connect** are the **two supported variants**: choose **gRPC** for Go-centric, **HTTP/2**-native services; choose **Connect** for **TS-first** clients and **HTTP-friendly** operations.
- **Proto JSON** (standard mapping from the same `.proto`) is the **tooling/dev** story: `grpcurl`, UIs, fixtures, and optional dev servers—without maintaining a second schema.
- **Binary protobuf** remains the **default for high-throughput** production; JSON is for **velocity**, **debuggability**, and **operator** workflows, not a competing contract.
