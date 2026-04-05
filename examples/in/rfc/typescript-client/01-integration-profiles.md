# Integration profiles (TypeScript ↔ Forst)

This document summarizes **how** different runtimes connect to Forst. For **normative** HTTP details, see [02-layer-b-http-contract.md](./02-layer-b-http-contract.md). For **`forst generate`**, see [03-layer-a-forst-generate-contract.md](./03-layer-a-forst-generate-contract.md).

## Profiles

| Profile | Runtime | What you use |
| ------- | ------- | ------------ |
| **Layer B — Node invokes local Forst** | Node runs `forst dev` (often via `@forst/sidecar`) | HTTP `GET /functions`, `POST /invoke`, etc. |
| **Layer A — Types only** | Frontends / TS packages | `forst generate` → `.d.ts` + stubs; **no** Node→Go bridge required |
| **Go service + TS clients** | Browser → your Go API | Shared shapes via Layer A; **wire protocol is yours** (REST, gRPC, …) |
| **Microservices** | TS service ↔ Go service | Network API (OpenAPI/gRPC); optional future emit from Forst types |

## Sidecar is not the whole story

`@forst/sidecar` is the **reference client** for **Layer B**. It does **not** replace **Layer A** for teams that only need **types**, and it does **not** define how **production** Go services are deployed (see [04-generated-go-execution-options.md](./04-generated-go-execution-options.md)).

## Phased roadmap (product)

1. **Phase 1 — Harden Layer B:** `forst dev` HTTP contract, sidecar package, tests, CI.
2. **Phase 2 — Layer A + operations:** Normative `forst generate` doc; document options for running emitted Go **without** the dev server where relevant.

## Comparative inspiration (non-binding)

Patterns from **Prisma** (CLI + codegen), **Protobuf/Connect** (IDL + wire), **Elm ports** (explicit message shapes) inform **documentation** and **contracts**, not a requirement to build a framework inside the compiler.

See ROADMAP TypeScript interoperability **Notes** for innovation rows (OpenAPI emit, pluggable transports, etc.).
