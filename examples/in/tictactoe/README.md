# Tic-tac-toe example

Stateless game rules in `main/engine.ft`, shared shapes in `types.ft`, and a small `fmt` demo in `main/server.ft` (same `package main`, under `main/` so stems match the package). For `encoding/json`, use **`forst run -export-struct-fields`** / **`forst build -export-struct-fields`** (or `compiler.exportStructFields` in `ftconfig.json`, used by `forst dev`) so emitted struct fields are exported and tagged for JSON—matching the field names `forst generate` produces for TypeScript.

## Run

From the repo root:

```bash
task example:tictactoe
```

Equivalent:

```bash
cd forst && go run ./cmd/forst run -root ../examples/in/tictactoe -- ../examples/in/tictactoe/main/server.ft
```

## TypeScript client (`forst generate`)

Output lands in gitignored `.forst/client` (linked as `node_modules/@forst/tictactoe`). Run the task below after editing `.ft` sources (CI covers merge + generate via `TestGenerate_exampleManifest` and `TestExampleTictactoeMergedPackage`).

From the repo root:

```bash
task example:tictactoe:generate
```

Equivalent:

```bash
cd forst && go run ./cmd/forst generate ../examples/in/tictactoe
```

Import with `import { createForstClient } from "@forst/tictactoe"` or `import { PlayMove } from "@forst/tictactoe/main"`. For tests, prefer `startForstTestServer` from `@forst/tictactoe/$testing` (needs optional peer `@forst/cli`). Or point `FORST_BASE_URL` at a running invoke server.

### Tests (bun)

After `task example:tictactoe:generate`, from the repo root:

```bash
bun install
task test:tictactoe
```

- **`tests/tictactoe-game.simulation.test.ts`** — real-server game via `startForstTestServer` and flat imports from `@forst/tictactoe/main`.
- **`tests/tictactoe-forst-run.test.ts`** — runs **`forst run -root … main/server.ft`** and checks stdout (merged-package smoke).

Optional: `FORST_BINARY`, or `FORST_SKIP_TICTACTOE_E2E=1` to skip.

## Golden Go output

`examples/out/tictactoe/server.go` is the merged-package Go emit for `main/server.ft` entry (uses `exportStructFields` from `ftconfig.json`). Regenerate all example goldens with **`task examples:update-goldens`**, or only tictactoe:

```bash
cd forst && UPDATE_TICTACTOE_GOLDEN=1 go test ./cmd/forst -run TestExampleTictactoeMergedPackage -count=1
```

See `examples/README.md` for how `in/` / `out/` examples fit together.
