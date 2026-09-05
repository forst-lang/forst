# Go interop example

Shows two ways Forst calls Go:

| File | Role |
|------|------|
| **`helpers.go`** | Hand-written Go in the same package. Exported funcs are visible to Forst **without** an `import` line. |
| **`stdlib.ft`** | Stdlib import: `import "os/exec"` with FFI (subslices, variadic spread, field/method access). |
| **`custom.ft`** | Same-package calls into `helpers.go`. |
| **`main.ft`** | Entry that runs both demos. Relies on merged same-package analysis (LSP / `-root`). |
| **`cli.ft`** | Single-file compile target for `task example:go-interop` and golden tests. |
| **`http_handle.ft`** | `net/http.HandleFunc` with a Forst function literal (callback FFI). |

## CLI

Ad-hoc `forst run` copies same-package hand-written `.go` files into its execution sandbox automatically, so `forst run cli.ft` works without configuration.

When `ftconfig.json` configures `generate.go` (or when `-o` is set), `forst run` emits beside the package and wraps `go run .` so generated Go stays co-located with your hand-written `.go`. Embedded invoke / bridge host mode always use an isolated sandbox with copied sibling `.go` sources.

For testing, `forst test ./...` discovers both `*_test.ft` and native Go `*_test.go` files and runs `go test`.

```bash
task example:go-interop
```

Or manually from `forst/`:

```bash
go run ./cmd/forst run ../examples/in/go_interop/cli.ft
# or: generate then go run .
go run ./cmd/forst generate ../examples/in/go_interop
cd ../examples/in/go_interop && go run .
```

Or with explicit CLI overrides (mirrors `ftconfig` field names):

```bash
go run ./cmd/forst generate --go-entry=../examples/in/go_interop/cli.ft --go-out=../examples/in/go_interop/main.gen.go --skip-client ../examples/in/go_interop
```

Golden: `examples/out/go_interop/cli.go` (`task examples:update-goldens` from repo root).
