# `@forst/effect`: npm Package, Observability, and the Effect Runtime

This document analyzes whether an **`@forst/effect`** (or similarly scoped) npm package can integrate **Forst runtime logs, spans, and observability** with the **calling Effect-TS runtime**, so that codebases using Effect can offload **high-memory or high-CPU** work to Forst/Go **without** giving up Effect’s ergonomics or observability story.

It expands on [12-effect-forst-integration.md](./12-effect-forst-integration.md), [17-feasibility-analysis.md](./17-feasibility-analysis.md), [18-comprehensive-error-handling.md](./18-comprehensive-error-handling.md), and ties cross-process concerns to [Sidecar: Developer Operations and Observability](../sidecar/05-observability.md).

---

## 1. Goals

| Goal | Meaning |
|------|---------|
| **Effect-native API** | Generated services expose `Effect.Effect<Success, Error, Requirements>` (or equivalent) instead of raw promises or callbacks. |
| **Observability continuity** | Traces, structured logs, and errors remain **correlatable** with the rest of an Effect application (OTEL, `Effect` Logger, metrics). |
| **Honest boundaries** | The TS and Go runtimes are **different processes** (sidecar) or **different runtimes** (bridge); documentation must not imply a single shared in-memory Effect interpreter unless that is literally true. |
| **Adoption** | Teams already invested in Effect can adopt Forst for **hot paths** incrementally. |

---

## 2. What the RFCs Already Assume

### 2.1 `@forst/effect` as the TypeScript façade

[12-effect-forst-integration.md](./12-effect-forst-integration.md) positions **`@forst/effect`** as the package that wraps sidecar (or client) calls: `ForstEffect.call(name, args)` returns an **Effect**, implemented via `Effect.tryPromise` around a `ForstClient`. That matches the product goal: **Effect at the boundary**, Go inside.

### 2.2 Feasibility of a dedicated module

[17-feasibility-analysis.md](./17-feasibility-analysis.md) rates a dedicated Effect integration module highly. The recurring cost is **implementation and maintenance** of the **bridge** (IPC/HTTP/native), not the idea of Effect-shaped entrypoints.

### 2.3 Errors, trace IDs, and OTEL on the Forst side

[18-comprehensive-error-handling.md](./18-comprehensive-error-handling.md) describes **tagged errors**, **trace/span identifiers** on error payloads, **Go OpenTelemetry** recording, and TypeScript examples using **`Effect.log` / `Effect.logError`**. That establishes **correlation** and **error observability**; it does **not** by itself define the full **wire protocol** for every log line emitted inside Go.

### 2.4 Cross-language tracing (authoritative for the boundary)

[Sidecar: Developer Operations and Observability](../sidecar/05-observability.md) states the hard problem explicitly: **trace context propagation** across TypeScript → Go, **correlation IDs** through transports, **span relationships**, and **performance attribution**. It proposes **OpenTelemetry**, **consistent span naming**, and **correlating logs with traces** via trace IDs.

**Conclusion:** The RFC set **supports** an `@forst/effect` package and **points to** unified observability via **OTEL + propagated context**. It does **not** yet merge into a single Effect “fiber” across processes.

---

## 3. What “Smooth Integration” Can Mean

### 3.1 Realistic target (recommended)

**Distributed continuity**, not **single-runtime magic**:

1. **W3C Trace Context / OTEL** propagates on each Forst invocation (HTTP headers, or explicit metadata on IPC).
2. The TS client creates or continues a **child span** around `ForstEffect.call` (e.g. `forst.<functionName>` with transport and payload attributes).
3. Go code runs with `context.Context` carrying the same trace; internal spans use the **same tracer** and **naming conventions** as in the sidecar doc.
4. **Structured errors** returned to TS include **stable tags** and optional **trace/span** fields so `Effect.catchTag`, metrics, and OTEL error recording stay aligned with [18-comprehensive-error-handling.md](./18-comprehensive-error-handling.md).
5. **Logs**: Go emits **structured** logs with **trace id** (and optionally span id); the TS side either forwards them to the app’s logger pipeline or relies on **log backends** that correlate by trace id (same approach as any polyglot service mesh).

This gives Effect-centric apps **one trace graph** and **correlated logs**, which is what production teams usually mean by “observability integrated with Effect.”

### 3.2 What is not realistic without extra machinery

- **Every** internal `log` in Go automatically becoming an **`Effect.log`** event in the **same** Effect Logger interpretation, **in process**, without bridging.
- **Fiber-local state** in TypeScript **implicitly** visible inside Goroutines without serialization rules.
- **Perfect** static analysis of arbitrary npm dependencies (already noted in [17-feasibility-analysis.md](./17-feasibility-analysis.md)).

---

## 4. `@forst/effect`: Responsibilities

The package should be a **thin, opinionated layer** over the generic client (`@forst/sidecar` or successor):

| Responsibility | Notes |
|----------------|--------|
| **Effect return type** | `Effect.tryPromise` / `Effect.async` / `Effect.try` with typed errors. |
| **Tagged errors** | Map wire errors to `Data.TaggedError` unions; align with [18-comprehensive-error-handling.md](./18-comprehensive-error-handling.md). |
| **Tracing** | Optional helpers: `withForstSpan`, `forstCall`, reading **active span** from `@opentelemetry/api` or Effect’s OTEL integration and injecting **traceparent** into the request. |
| **Logging** | Optional: bridge **structured log events** from responses or a side channel into `Effect.log` / user callbacks. |
| **Configuration** | `Layer` or env-based setup for default tracer, service name, redaction. |

Naming: the feasibility doc discusses **`@effect/forst`** as an alternative; **scope and ownership** (Forst-owned vs Effect org) are product/publishing decisions, not technical blockers.

---

## 5. Observability Model (End-to-End)

### 5.1 Trace graph

```
[Effect app: HTTP handler span]
  └── [forst.getUserById]  (client span, TS)
        └── [forst.getUserById]  (server/child span, Go)
              └── [db.QueryRow ...]  (optional children in Go)
```

Alignment with [05-observability.md](../sidecar/05-observability.md): same **tracer names**, **`forst.<function>`** conventions, and **transport** attributes so dashboards can split **serialization** vs **Go CPU** vs **network**.

### 5.2 Logs

- **TS**: Prefer **structured** logs with `trace_id` from the active context (Effect Logger + OTEL resource fields).
- **Go**: Same **trace_id** on every line or log record (OpenTelemetry logger SDK or structured `slog`/zap fields).
- **Optional enhancement**: **log forwarding** stream (or batched) from sidecar to TS for dev UX; **not** required for production parity if log aggregation is centralized.

### 5.3 Metrics

- Histograms for **latency**, **payload size**, **errors by `_tag`** on the client; mirror **server-side** metrics with the same labels where possible ([05-observability.md](../sidecar/05-observability.md) transport-aware metrics pattern).

---

## 6. API Sketches (Non-Normative)

Illustrative only; real APIs follow Effect and sidecar releases.

### 6.1 Call with automatic span

```typescript
import { Effect } from "effect";
import { ForstEffect } from "@forst/effect";

const getUser = (id: number) =>
  ForstEffect.call("getUserById", { id }, {
    spanName: "forst.getUserById",
    // inject trace context from OTEL active context by default
  });
```

### 6.2 Layer for defaults

```typescript
ForstEffect.layer({
  tracer: Tracer,
  serviceName: "billing-api",
  redact: (args) => ({ ...args, token: "[REDACTED]" }),
});
```

### 6.3 Error mapping

Preserve **`_tag`** and optional **`traceId` / `spanId`** on failures so `Effect.catchTag` and dashboards stay consistent with [18-comprehensive-error-handling.md](./18-comprehensive-error-handling.md).

---

## 7. Gaps to Close in Follow-Up RFCs / Specs

1. **Canonical wire format** for trace context (headers vs body field; IPC framing).
2. **Whether** Go emits **span ids** back to the client for **log correlation** only, or full **event** payloads.
3. **Log forwarding**: dev-only vs production; backpressure; PII.
4. **Effect version matrix**: which Effect major versions the package supports and how `Layer` / `Scope` interact with `@forst/effect` configuration.
5. **Naming**: publish as **`@forst/effect`** vs **`@effect/forst`** and governance.

---

## 8. Summary

| Question | Answer |
|----------|--------|
| Can we have **`@forst/effect`**? | **Yes** — [12-effect-forst-integration.md](./12-effect-forst-integration.md) and [17-feasibility-analysis.md](./17-feasibility-analysis.md) support a dedicated npm module. |
| Can **logs/spans** feel integrated with Effect? | **Yes**, via **OTEL + propagated trace context**, **child spans** around calls, **structured correlated logs**, and **tagged errors** — not via a single merged runtime. |
| Where is the boundary spec? | **Cross-process** details live in [05-observability.md](../sidecar/05-observability.md); **error/trace fields** in [18-comprehensive-error-handling.md](./18-comprehensive-error-handling.md). |
| What remains to specify? | **Wire format**, optional **log streaming**, and **versioning** (Section 7). |

This positions Forst as a **backend offload** for Effect-heavy TypeScript codebases while preserving the **stdlib-like** developer experience of Effect on the **application** side — which, as this RFC argues, is achieved through **clear boundaries** and **distributed observability**, not process fusion.
