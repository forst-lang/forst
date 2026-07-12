# Multi-package dev example

Layout mirrors a host bootstrap (`main.ft`) plus library invoke package (`bcrypt.ft`) and type-only package (`types.ft`). With `node.hostMode`, `main.ft` also boots a minimal Node host shim.

```bash
task example:multipackage-dev
task example:multipackage-dev:run
forst generate -root examples/in/rfc/node-interop/multi-package-dev .
forst run -root examples/in/rfc/node-interop/multi-package-dev -- examples/in/rfc/node-interop/multi-package-dev/main.ft
```

Expected: `bcrypt.client.ts` generated; `types.client.ts` omitted; cross-package invoke in compiled binary; hostMode nodert companion when `host.ts` is present.
