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

`forst run` emits transpiled Go into a temp sandbox (`.forst/run/`); **hand-written `.go` is not included**. For same-package Go stubs, configure `generate.go` in `ftconfig.json` (see `ftconfig.json` in this directory) or pass matching CLI flags:

```bash
task example:go-interop
```

Or manually from `forst/`:

```bash
go run ./cmd/forst generate ../examples/in/go_interop
cd ../examples/in/go_interop && go run .
```

Or with explicit CLI overrides (mirrors `ftconfig` field names):

```bash
go run ./cmd/forst generate --go-entry=../examples/in/go_interop/cli.ft --go-out=../examples/in/go_interop/main.gen.go --skip-client ../examples/in/go_interop
```

Golden: `examples/out/go_interop/cli.go` (`task examples:update-goldens` from repo root).
