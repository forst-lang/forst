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

Generated Go (`z_forst_gen.go` / `z_forst_gen_test.go`) is written by `forst test` emit and gitignored; it forwards `providers` into `alpha.LogExpiry` using Forst sibling import resolution (no committed Go stub required). Forst sibling imports resolve from `.ft` + the module map; `go/packages` is used only for stdlib and external Go imports.

Go goldens: `examples/out/rfc/providers/cross_pkg/alpha/log.go` and `.../beta/handle.go`. Refresh all example goldens with **`task examples:update-goldens`** from the repo root, or only cross_pkg with:

```bash
UPDATE_PROVIDERS_CROSS_PKG_GOLDEN=1 go test ./cmd/forst -run TestExampleProvidersCrossPkgGolden -count=1
```
