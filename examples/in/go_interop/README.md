# Go interop example

Shows two ways Forst calls Go:

| File | Role |
|------|------|
| **`helpers.go`** | Hand-written Go in the same package. Exported funcs are visible to Forst **without** an `import` line. |
| **`stdlib.ft`** | Stdlib import: `import "os/exec"` with FFI (subslices, variadic spread, field/method access). |
| **`custom.ft`** | Same-package calls into `helpers.go`. |
| **`main.ft`** | Entry that runs both demos. Relies on merged same-package analysis (LSP / `-root`). |
| **`cli.ft`** | Single-file compile target for `task example:go-interop` and golden tests. |

## CLI

`forst run` emits transpiled Go into a temp sandbox (`.forst/run/`); **hand-written `.go` is not included**. For same-package Go stubs, use `forst build` and the standard Go toolchain:

```bash
task example:go-interop
```

Or manually from `forst/`:

```bash
go run ./cmd/forst build -o ../examples/in/go_interop/main.gen.go -- ../examples/in/go_interop/cli.ft
cd ../examples/in/go_interop && go run .
```

Golden: `examples/out/go_interop/cli.go` (`task examples:update-goldens` from repo root).
