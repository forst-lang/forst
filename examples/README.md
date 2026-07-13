# Examples

## Go interop layout profiles

| Profile | Example | `go.mod` | `.ft` sources | Workflow |
| --- | --- | --- | --- | --- |
| **Go-native** | [`in/go_interop/`](in/go_interop/) | project root | mixed with `.go` | `forst build` + `go run .` |
| **Node-primary** | [`in/rfc/node-interop/remix-serve/`](in/rfc/node-interop/remix-serve/) | `.forst-gomod/` | `main/` under boundary | `forst generate` + `forst run` |
| **Compiler dev** | any out-of-tree copy | either | either | `FORST_GOMOD_ROOT` points at the compiler module |

In **Node-primary** layouts, `replace` paths in `.forst-gomod/go.mod` are relative to the **project boundary** (the directory with `ftconfig.json`), not to `.forst-gomod/` itself.

## Compiler inputs and golden outputs

- **`in/`** — Forst `.ft` sources used by Task targets (e.g. `task example:basic`). Paths are mirrored under **`out/`** with expected Go output for integration checks (`in` → `out`).

- **`in/*.ft`** at the root of `in/` (e.g. `basic.ft`, `go_builtins.ft`, `generics.ft`, `slices.ft`, `ensure.ft`, `nominal_error.ft`, `union_error_types.ft`) are the small, primary examples referenced by the Taskfile and [testing rules](../.cursor/rules/testing.mdc). **`in/go_interop/`** is the Go FFI + same-package hand-written Go showcase (`cli.ft`, `helpers.go`). Full compile integration for these lives in **`cmd/forst` `TestExamples`**; `internal/compiler` keeps a minimal smoke compile only. Regenerate committed Go goldens under **`out/`** with **`task examples:update-goldens`** (from repo root). That task runs `git add -f examples/out/` so new or updated goldens are staged even though the root `out` ignore rule would otherwise hide them from `git status`.

- **`in/imports/`** — multi-file “imports” demo (LSP merged package + `task example:imports` via `cli.ft`); see `in/imports/README.md`.

- **`in/tictactoe/`** — multi-file `package main` under `main/` (types + engine + `fmt` demo `server.ft`). Run with `task example:tictactoe` (`forst run -root …/tictactoe -- …/main/server.ft`). Regenerate TS with `task example:tictactoe:generate`. Golden Go for the merged compile is `out/tictactoe/server.go` (uses `exportStructFields` from `in/tictactoe/ftconfig.json`). Refresh all Go goldens with **`task examples:update-goldens`** from the repo root, or only tictactoe with `UPDATE_TICTACTOE_GOLDEN=1 go test ./cmd/forst -run TestExampleTictactoeMergedPackage -count=1` from `forst/`.

- **`in/forst-generate-ts-examples.json`** — lists example directories (under `in/`) that CI exercises with `forst generate` plus optional `mustContain` checks (`TestGenerate_exampleManifest` in `forst/cmd/forst`). Add a path when an example has `ftconfig.json` + `.ft` sources and should keep emitting valid `generated/` + `client/` TypeScript. Separate `tsc --noEmit` smoke runs in `generate_tsc_test.go`.

## `in/rfc/`

Design notes (Markdown), sample `.ft` files, and sometimes TypeScript or config files are grouped **by topic**. That folder is **not only** minimal “hello world” examples: it is **RFC-style documentation + runnable snippets** kept together. Each topic usually has a `README.md` index (except where noted). Implementation status in the wild is tracked separately in [ROADMAP.md](../ROADMAP.md); **stage** here describes the RFC’s *intent and maturity as documentation*, not a guarantee that every bullet is shipped.

### RFC index (by topic)

**Stages (how to read this table):**

| Stage | Meaning |
| ----- | ------- |
| **Exploration** | Alternatives and tradeoffs; no single committed delivery tied to the whole RFC set. |
| **Specification / design** | Normative contracts, architecture, or phased plans meant to guide implementation. |
| **Planned (language)** | Aligns with [ROADMAP](../ROADMAP.md) language rows still **planned** or early; design-only until implemented. |
| **Partially realized** | Core pieces exist in the compiler or tooling; the full RFC scope is larger than what is shipped. |
| **Experimental (tooling)** | End-user or dev workflows (sidecar, HTTP, NPM) expected to evolve; not the core language definition. |

| Topic | Index / entry point | Stage | Notes |
| ----- | -------------------- | ----- | ----- |
| **Effect-TS integration** | [in/rfc/effect/README.md](in/rfc/effect/README.md) | Exploration | Many documents (native effects, interop, batching, strategy); long-horizon options for Effect-like patterns. |
| **Error system** | [in/rfc/errors/README.md](in/rfc/errors/README.md) | Specification / design | Broad architecture (hierarchy, OTel, factories); compiler has typed errors—full vision is wider than today’s emit. |
| **ES modules / Node** | [in/rfc/esm/README.md](in/rfc/esm/README.md) | Exploration | Forst as native ESM, addons, enhanced sidecar variants. |
| **User generics** | [in/rfc/generics/README.md](in/rfc/generics/README.md) | Specification / design · **Planned (language)** | Phased plan for user type parameters; see [00-user-generics-and-type-parameters.md](in/rfc/generics/00-user-generics-and-type-parameters.md). |
| **Crystal-inspired optionals** | [in/rfc/optionals/README.md](in/rfc/optionals/README.md) | Exploration | **Hub** [00](in/rfc/optionals/00-crystal-inspired-optionals.md); topics **01–10** (e.g. [10](in/rfc/optionals/10-type-guards-shape-guards-and-optionals.md) type/shape guards). |
| **Guards** | [in/rfc/guard/guard.md](in/rfc/guard/guard.md) (no folder README) | Partially realized | Shape guards and `ensure` are in the language; [anonymous_objects.md](in/rfc/guard/anonymous_objects.md), [interop.md](in/rfc/guard/interop.md) extend the story. |
| **Sidecar & TS adoption** | [in/rfc/sidecar/00-sidecar.md](in/rfc/sidecar/00-sidecar.md), [10-decisions.md](in/rfc/sidecar/10-decisions.md) | Experimental (tooling) + specification | HTTP sidecar, NPM, tests under `sidecar/tests/`; strategic decisions doc. (Ignore `node_modules/`—third-party deps, not RFC text.) |
| **TypeScript client** | [in/rfc/typescript-client/README.md](in/rfc/typescript-client/README.md) | Specification / design · **in progress (tooling)** | `forst generate`, dev HTTP contract, integration profiles (P0–P3 plan in [00-implementation-plan.md](in/rfc/typescript-client/00-implementation-plan.md)). |

## `client-integration/`

Standalone demo of generated client usage; not part of the default `task example:*` paths unless documented in the Taskfile.
