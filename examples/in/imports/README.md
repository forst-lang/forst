# LSP: merged package (one tiny app)

Tiny “hello” program split across three files so you can see **how the language server joins open buffers** in the same folder with the same `package main`.

| File | Role |
|------|------|
| **`greeting.ft`** | Forst-only helper: `greeting()` returns a string. Like shared app code in its own file. |
| **`fmt_only.ft`** | Only `import "fmt"`. In normal Go each file imports what it uses; here the import lives in **this** buffer so the LSP demo can show that **`fmt` is still available in `main.ft`** when both are open and merged. |
| **`main.ft`** | Entry: prints the greeting. It does **not** repeat `import "fmt"` and does **not** define `greeting`—it relies on the other open files. |

**`cli.ft`** is the same program in **one** file. Use it with `forst run` or `task example:imports`. The compiler only reads **one** path per invocation; the split layout is for the editor.

## Golden outputs (`examples/out/`)

`fmt_only.ft` → `fmt_only.go`, `greeting.ft` → `greeting.go`, `cli.ft` → `cli.go`. There is no `main.go`: `main.ft` does not compile alone. Integration coverage for the merged case is `TestProcessForstFile_crossFileGoImportSharedAcrossBuffers` in `cmd/forst/lsp/cross_file_test.go`.

## CLI

```bash
task example:imports
```

From `forst/`: `go run ./cmd/forst run -- ../examples/in/imports/cli.ft`
