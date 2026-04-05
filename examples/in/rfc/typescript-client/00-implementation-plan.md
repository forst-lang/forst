# TypeScript client generation — implementation plan

Prioritized work to make generated TypeScript satisfying for consuming apps. Higher items matter most to users.

---

## P0 — Fix broken or misleading workflows

### 1. Aggregate `types.d.ts` for multi-file `forst generate`

**Problem:** Each `.ft` pass used to overwrite `generated/types.d.ts`, so directory runs dropped earlier files.

**Implementation:** Shared pipeline and merge:

- `forst/internal/transformer/ts/forst_file.go` — `TransformForstFileFromPath` (strict for CLI, relaxed for dev server).
- `forst/internal/transformer/ts/merge.go` — `MergeTypeScriptOutputs` (dedupe identical type blocks; conflict detection on duplicate function names with different signatures).
- `forst/cmd/forst/generate.go` — sorts inputs, merges once, writes `types.d.ts`, then per-stem `*.client.ts`; `client/index.ts` only lists stems that succeeded.
- `forst/cmd/forst/dev_server.go` — `GenerateTypesForFunctions` uses the same transform + merge path.

### 2. End-to-end tests

**Coverage:** `internal/transformer/ts/merge_test.go`, `forst_file_test.go`; `cmd/forst/generate_test.go` (`TestGenerateCommand_directoryMergesTypesIntoSingleTypesDotDts`); dev server tests unchanged in spirit (still exercise single-file transform via `generateTypesForFile`).

---

## P1 — Type mapping quality

**Implemented:**

- **`GetTypeScriptType`** (`forst/internal/transformer/ts/type_mapping.go`): `[]T` → `T[]` (parenthesized when needed for unions); `map[K]V` → `Record<K, V>`; `*T` → `(T) | null`; `error` → `unknown`. Unparameterized `[]` / `map` still fall back to `any[]` / `Record<any, any>`.
- **Function return types** (`function.go`): same `GetTypeScriptType` path as parameters; default when no inferred or explicit return is `unknown` instead of `any`.
- **Tests:** `type_mapping_test.go` (array, map, pointer, error, union array wrapping); `transformer_test.go` (`TestTransformForstFileToTypeScript_arrayMapAndPointerInShape`).

---

## P2 — Ergonomics

**Implemented:**

- **`*.client.ts`** (`internal/transformer/ts/client.go`): type bodies removed; `import type { … } from './types'` lists `ExportedTypeNames` collected when emitting type defs (`output.go`, `transformer.go`). `MergeTypeScriptOutputs` merges and dedupes exported names (`merge.go`, `exported_names.go`).
- **`client/index.ts`** (`generate.go`): `export * from './types'` so the stub package re-exports declarations.
- **Docs:** [README.md](../../../../README.md#typescript-client-output) — short intro to `forst generate` for new readers.

---

## P3 — Module augmentation

- Only if product needs `declare module` patches; require `tsc --noEmit` clean; avoid invalid augmentation blobs.

---

## Testing (TypeScript E2E)

`go test ./cmd/forst/...` includes **`TestGenerate_typescriptTypechecks_*`** in [`generate_tsc_test.go`](../../../../forst/cmd/forst/generate_tsc_test.go), which runs `tsc --noEmit` on generated output using shims under [`forst/cmd/forst/testdata/typescript/`](../../../../forst/cmd/forst/testdata/typescript/). Requires a local TypeScript install (e.g. `bun install` at repo root for `typescript` in devDependencies) or `bun` / `tsc` / `npx` on `PATH`. Set **`FORST_SKIP_TS_E2E=1`** to skip.

## Suggested sequencing

1. P0 merge + tests (this document’s P0).
2. P1 type mapping.
3. P2 docs and client ergonomics.
4. P3 augmentation (optional).
