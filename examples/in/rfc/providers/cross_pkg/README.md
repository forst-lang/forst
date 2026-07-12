# Cross-package Providers cookbook

`auth` exports `LogEvent` with `use logger: Logger` for cross-package calls. That is **not** a TypeScript/sidecar entry point (ADR-022 applies to `package main` only). Wire `Logger` at the host (`api.HandleRequest` tests, `main`, or sidecar bootstrap).

Run:

```bash
task test:providers-cross_pkg
```

Files:

- `auth/log.ft` — contract + direct `use`
- `api/handle.ft` — imports auth; `Logger` typedef for wiring keys
- `api/handle_test.ft` — `with { Logger: … }` integration test

Generated Go for tests lives under `.forst/gen/test/` (session-scoped); it forwards `providers` into `auth.LogEvent` using Forst sibling import resolution (no committed Go stub required). Forst sibling imports resolve from `.ft` + the module map; `go/packages` is used only for stdlib and external Go imports.

Go goldens: `examples/out/rfc/providers/cross_pkg/auth/log.go` and `.../api/handle.go`. Refresh all example goldens with **`task examples:update-goldens`** from the repo root, or only cross_pkg with:

```bash
UPDATE_PROVIDERS_CROSS_PKG_GOLDEN=1 go test ./cmd/forst -run TestExampleProvidersCrossPkgGolden -count=1
```
