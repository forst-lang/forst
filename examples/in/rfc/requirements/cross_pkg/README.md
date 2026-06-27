# Cross-package Providers cookbook

`alpha` exports `LogExpiry` with `use logger: Logger` for cross-package calls. That is **not** a TypeScript/sidecar entry point (ADR-022 applies to `package main` only). Wire `Logger` at the host (`beta.Handle` tests, `main`, or sidecar bootstrap).

Run:

```bash
task test:providers-cross_pkg
```

Files:

- `alpha/log.ft` — contract + direct `use`
- `alpha/stub.go` — Go stub for cross-package call resolution
- `beta/handle.ft` — imports alpha, repeats `Logger` typedef for wiring keys
- `beta/handle_test.ft` — `with { Logger: … }` integration test
