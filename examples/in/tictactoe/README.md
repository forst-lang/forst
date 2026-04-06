# Tic-tac-toe example

Stateless game rules in `engine.ft`, shared shapes in `types.ft`, and a small `fmt` demo in `server.ft` (same `package main`). For `encoding/json`, use **`forst run -export-struct-fields`** / **`forst build -export-struct-fields`** (or `compiler.exportStructFields` in `ftconfig.json`, used by `forst dev`) so emitted struct fields are exported and tagged for JSON—matching the field names `forst generate` produces for TypeScript.

## Run

From the repo root:

```bash
task example:tictactoe
```

Equivalent:

```bash
cd forst && go run ./cmd/forst run -root ../examples/in/tictactoe -- ../examples/in/tictactoe/server.ft
```

## TypeScript client (`forst generate`)

`generated/` and `client/` under this folder are **gitignored**; run the task below after editing `.ft` sources (CI covers merge + generate via `TestGenerateCommand_tictactoeMergedPackage`).

From the repo root, regenerate `generated/*.ts` and `client/` (merged `types.d.ts`, per-file `*.client.ts`, and `@forst/client` wrapper):

```bash
task example:tictactoe:generate
```

Equivalent:

```bash
cd forst && go run ./cmd/forst generate ../examples/in/tictactoe
```

Use `npm install` or `pnpm install` inside `client/` if you want to type-check against `@forst/sidecar` (see `client/package.json`). Point `FORST_BASE_URL` at a running `forst dev` sidecar when calling `NewGame()` / `PlayMove()` from TypeScript.

## Golden Go output

`examples/out/tictactoe/server.go` is the merged-package Go emit for `server.ft` entry. Regenerate:

```bash
cd forst && UPDATE_TICTACTOE_GOLDEN=1 go test ./cmd/forst -run TestExampleTictactoeMergedPackage -count=1
```

See `examples/README.md` for how `in/` / `out/` examples fit together.
