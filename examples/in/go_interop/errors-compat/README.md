# Go error-library compatibility corpus

Separate Go module so third-party error libraries never enter the compiler's `go.mod`.

Run from repo root:

```bash
task test:errors-compat
```

Stdlib-only cases (`errors`, `fmt`, `io`, `context`) live in `forst/internal/typechecker/errors_compat_stdlib_test.go` and run on every PR without network.

Third-party libraries (pkg/errors, cockroachdb/errors, multierr) are exercised via this module's Go tests (`errors_compat_test.go`).
