# TypeScript client generation (RFC notes)

Design notes and the implementation roadmap for `forst generate`, merged `types.d.ts`, and TypeScript types for consuming apps.

- [00-implementation-plan.md](./00-implementation-plan.md) — prioritized work (P0–P3), acceptance criteria, and sequencing.
- [01-integration-profiles.md](./01-integration-profiles.md) — sidecar vs codegen-only profiles, phased adoption.
- [02-forst-dev-http-contract.md](./02-forst-dev-http-contract.md) — **normative** HTTP API for `forst dev` (Node / `@forst/sidecar`).
- [03-forst-generate-contract.md](./03-forst-generate-contract.md) — **`forst generate`** outputs and guarantees.
- [04-generated-go-execution-options.md](./04-generated-go-execution-options.md) — Phase 2 exploration: running emitted Go without the dev server.

A short introduction for new readers is in the root [README.md](../../../../README.md#typescript-client-output) (“TypeScript client output”).

Related: sidecar RFC and examples live under [../sidecar/](../sidecar/).
