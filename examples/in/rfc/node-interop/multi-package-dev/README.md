# Multi-package dev example

Layout mirrors a host bootstrap (`main/main.ft`) plus library invoke packages under `auth/` and `types/`. With `node.hostMode`, `main.ft` also boots a minimal Node host shim.

```bash
task example:multipackage-dev
task example:multipackage-dev:run
forst generate -root examples/in/rfc/node-interop/multi-package-dev .
forst run -root examples/in/rfc/node-interop/multi-package-dev -- examples/in/rfc/node-interop/multi-package-dev/main/main.ft
```

Expected: the `auth` client subpath is generated; the type-only `types` package is omitted; cross-package invoke is present in the compiled binary; the hostMode nodert companion is emitted when `host.ts` is present.
