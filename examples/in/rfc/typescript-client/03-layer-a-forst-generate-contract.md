# Layer A: `forst generate` contract (TypeScript declarations)

Normative outline for **file-based** TypeScript interop: what `forst generate` produces and what it guarantees. **Not** the HTTP `forst dev` surface (see [02-layer-b-http-contract.md](./02-layer-b-http-contract.md)).

**Producer:** `forst` compiler (release version). **Consumer:** `tsc` / bundlers reading generated files.

## Scope

| In scope | Out of scope |
| -------- | ------------- |
| `.d.ts` and stub files from **`forst generate`** | Your production HTTP API between browser and Go |
| CLI inputs (paths, flags), exit codes, merge behavior | **`forst dev`** / sidecar HTTP (Layer B) |
| Type mapping rules from Forst → TS | Runtime validation in TS unless you add it |

## Outputs (baseline)

Paths are relative to the generator output directory (CLI-dependent); typical layout:

| Artifact | Role |
| -------- | ---- |
| `generated/types.d.ts` | Merged declarations for all inputs |
| Per-stem `*.client.ts` | Stubs importing `./types` |
| `client/index.ts` | Re-exports |

Merge semantics (duplicates, conflicts): see [00-implementation-plan.md](./00-implementation-plan.md) and tests in `forst/cmd/forst/generate_test.go`, `internal/transformer/ts/merge_test.go`.

## Semantic guarantees

- Type mapping follows `internal/transformer/ts/type_mapping.go` and tests (`generate_tsc_test.go` where enabled).
- If generation fails, the CLI should exit non-zero without claiming success.
- Generated TS is **descriptive** for clients; **behavior** is defined by transpiled Go and your server.

## Versioning

Breaking changes to output layout or mapping should be documented in release notes; a dedicated `FORST_TS_CONTRACT` version may be introduced later.

## Non-goals

- No guarantee that generated TS matches arbitrary JSON on the wire without separate schema or tests.
- LSP behavior is not part of this contract.

## Conformance

- Go tests: `TestGenerate_*`, `merge_test`, `generate_tsc_test` (when TypeScript toolchain available).
