# TypeScript client generation (RFC notes)

Design notes and the implementation roadmap for `forst generate`, merged `types.d.ts`, and TypeScript types for consuming apps.

- [00-implementation-plan.md](./00-implementation-plan.md) — prioritized work (P0–P3), acceptance criteria, and sequencing.
- [01-integration-profiles.md](./01-integration-profiles.md) — sidecar vs codegen-only profiles, phased adoption.
- [02-forst-dev-http-contract.md](./02-forst-dev-http-contract.md) — **normative** HTTP API for `forst dev` (Node / `@forst/sidecar`).
- [03-forst-generate-contract.md](./03-forst-generate-contract.md) — **`forst generate`** outputs and guarantees.
- [04-generated-go-execution-options.md](./04-generated-go-execution-options.md) — Phase 2 exploration: running emitted Go without the dev server.
- [05-forst-http-gateway-signature-pipeline-rfc.md](./05-forst-http-gateway-signature-pipeline-rfc.md) — **`forst/gateway`**: **`GatewayRequest` / `GatewayResponse`**, **`GatewayHandler`**, **Forst** **gateway** **middleware** (`@forst/sidecar` Express), **invoke** **`result`**, **IR** + **standard** **TS** **pipeline** (no decorators, no **HTTP** *parser* in the *core* grammar; see **T4** in RFC 05).

### Decision records (TypeScript client architecture)

Major decisions guiding future compiler, sidecar, and ecosystem work:

- [06-forst-intermediate-representation.md](./06-forst-intermediate-representation.md) — **`FR-IR-1`**: versioned **IR** (`forst.ir.json`) as semantic source of truth for TS, proto, and ecosystem emitters.
- [07-typescript-client-layered-architecture.md](./07-typescript-client-layered-architecture.md) — **`FR-TS-ARCH-1`**: three-layer stack (compiler faucet / transport / ecosystem npm); no TanStack or tRPC in the driver.
- [08-generated-client-runtime-integration.md](./08-generated-client-runtime-integration.md) — **`FR-CLIENT-1`**: generated `*.client.ts` + `@forst/sidecar` integration, wire–type fidelity tiers, dev loop.
- [09-ecosystem-codegen-from-ir.md](./09-ecosystem-codegen-from-ir.md) — **`FR-ECO-1`**: TanStack Query, tRPC-style procedures, OpenAPI as optional IR consumers outside the compiler.

A short introduction for new readers is in the root [README.md](../../../../README.md#typescript-client-output) (“TypeScript client output”).

Related: sidecar RFC and examples live under [../sidecar/](../sidecar/). **Sidecar bridge** for the same work: [../sidecar/12-forst-http-outcome-pipeline.md](../sidecar/12-forst-http-outcome-pipeline.md). **Production wire from IR:** [../sidecar/13-ir-driven-production-wire.md](../sidecar/13-ir-driven-production-wire.md).
