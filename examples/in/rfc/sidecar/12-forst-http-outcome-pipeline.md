# Forst HTTP: `GatewayRequest` / `GatewayResponse`, and the Express / sidecar bridge

| Field | Value |
|-------|--------|
| **Type** | Sidecar-focused design note; **authoritative** cross-language contract: [TypeScript RFC 05](../typescript-client/05-forst-http-gateway-signature-pipeline-rfc.md) |
| **Node package** | [`@forst/sidecar`](../../../../packages/sidecar/) |
| **HTTP envelope** (normative) | [../typescript-client/02-forst-dev-http-contract.md](../typescript-client/02-forst-dev-http-contract.md) |

## Purpose

**Naming:** The **Forst** **gateway** story in [RFC 05](../typescript-client/05-forst-http-gateway-signature-pipeline-rfc.md) — merge path e.g. **`forst/gateway`**, types **`GatewayRequest` / `GatewayResponse`**, **`GatewayHandler`** — should **not** be confused with [Go’s `net/http`](https://pkg.go.dev/net/http) in documentation, generated `package` lines, or onboarding.

**Forst** **gateway** **middleware** means the **Express** integration in **`@forst/sidecar`** (the *middleware* that turns **`invoke` `result`** into HTTP / `next()`). **“Gateway”** *alone* is a fine short label once the reader knows you mean *Forst* + **sidecar**.

This document ties **`@forst/sidecar`** to that model so that:

- **TypeScript** types in the sidecar **mirror** the **`invoke` `result`** **JSON** after **encoding** the `Ok` `GatewayResponse` binary from `forst/gateway` (constructors may use a shorter module alias; handler *return* in Forst is **`Result<GatewayResponse, E>`** on the *recommended* path, per RFC 05).  
- **Express** *Forst* **gateway** *middleware* **branches** on **`kind`**: *pass* (delegate to `next()` with `locals` / `req` merge) vs *answer* (set status, headers, body) — *one* `result` *shape* (RFC 05 §12).  

## Forst program shape

Authors target the types defined in [RFC 05](../typescript-client/05-forst-http-gateway-signature-pipeline-rfc.md) §5, §9 — summarized here as:

- **Parameter:** `GatewayRequest` in **`forst/gateway`** (same *shape* as `ForstRoutedRequest` + `bodyBase64` in [express-middleware.ts](../../../../packages/sidecar/src/express-middleware.ts)). Validate **HTTP** request data (query, path, body) with **request** guards (RFC 05 §13).  
- **Return:** Typically `Result<GatewayResponse, E>`; the `Ok` payload is `GatewayResponse` from **stdlib** constructors only (e.g. `text`, `pass` — see RFC 05 §9).  
- **Typedef:** `GatewayHandler` — the **one line** to search for in docs and code review (older drafts used `RoutedHandler` / `http.Incoming`+`http.Outcome` spellings).  

**Registration** to **`POST /invoke`** is a **build / dev** concern: **discovery** and/or **config** (RFC 05), **not** a core language keyword and **not** a decorator.

## What `@forst/sidecar` owns (normative target)

| Surface | Role |
|---------|--------|
| `ForstRoutedRequest`, `ForstRoutedResponse` / split types | Wire-aligned, `kind`-discriminated `result` (RFC 05); same logical request shape as Forst `GatewayRequest`. |
| `createRouteToForstMiddleware` | `Promise<void>`-typed async handler; **`await`**-safe in **tests**; `mapRequest` hook. Part of **Forst** **gateway** **middleware** setup. |
| `isPass` / `isAnswer` | **Narrowing** on **`invoke` `result`** for custom middleware: **`kind === "pass"`** vs **`kind === "answer"`** (RFC §12). |
| `ForstSidecarClient` | `POST /invoke` / `invoke/raw` **serialization** of the **request** / **`GatewayRequest`**-equivalent *payload* for the *gateway* path. |

**Hand-authored** **TypeScript** may **later** do `import type` from **standard** **`forst generate` output**; until then, types stay **in** [packages/sidecar](../../../../packages/sidecar) and are **bumped** with **wire** + **Forst** **gateway** types in lockstep (RFC 05: **no** types-only orphan package).  

## End-to-end checklist (for README “happy path”)

1. On the gateway path, the Forst handler return is **`Result<GatewayResponse, E>`** (or equivalent); **`Ok`** carries **`GatewayResponse`**, not a raw map (RFC 05 §10).  
2. **Function** is a **listed** **invoke** **target** per **dev** / **config** (RFC 05).  
3. **Express** uses **Forst** **gateway** **middleware** (e.g. **`createRouteToForstMiddleware`**) with `package` + `function` **strings** **matching** registration.  
4. **`tsc`**: `Locals` / `request` **generics** **match** app **augmentation** (see sidecar README).  
5. Optional: **`import type`** from **generated** `.d.ts` if/when the **standard** **pipeline** lands—see [03-forst-generate-contract.md](../typescript-client/03-forst-generate-contract.md).

## Out of sidecar scope (here)

- **Defining** **gateway** *constructors* in **`forst/gateway`** — Forst *stdlib* / **merge** package.  
- **IR schema** and **`forst generate`** layout — **TypeScript** RFCs **05** and **03**.  
- **Compiler**-side **shims** and **entrypoint** **discovery** — `forst` **CLI** / compiler **design**.  

## Related

- [11-wire-format.md](./11-wire-format.md) — broader **wire** and **format** **tradeoffs** (gRPC, JSON, **etc.**).  
- [08-development-workflow.md](./08-development-workflow.md) — **dev** **workflow** and **sidecar** **connect**.  
- [02-forst-dev-http-contract.md](../typescript-client/02-forst-dev-http-contract.md) — **`/invoke`**, **`/types`**, **`result`**, **`contractVersion`**.  

The **authoritative** end-to-end **gateway** + **IR/TS** design is [05-forst-http-gateway-signature-pipeline-rfc.md](../typescript-client/05-forst-http-gateway-signature-pipeline-rfc.md); this file is the **bridge** to **concrete** **TypeScript** **files** in the repo.
