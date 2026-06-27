# Cross-package Providers cookbook

`alpha` exports `LogExpiry` with `use logger: Logger` for cross-package calls. That is **not** a TypeScript/sidecar entry point (ADR-022 applies to `package main` only). Wire `Logger` at the host (`beta.Handle` tests, `main`, or sidecar bootstrap).

Run:

```bash
task test:providers-cross_pkg
```

Files:

- `alpha/log.ft` — contract + direct `use`
- `beta/handle.ft` — imports alpha; `Logger` typedef for wiring keys
- `beta/handle_test.ft` — `with { Logger: … }` integration test

Generated Go (`z_forst_gen.go` / test emit) forwards `providers` into `alpha.LogExpiry` using Forst sibling import resolution (no Go stub required).
