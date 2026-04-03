# Forst Roadmap

Forst aims to give backend developers TypeScript-grade ergonomics while compiling to Go and interoperating with TypeScript clients ([README](./README.md)). Principles and boundaries live in [PHILOSOPHY.md](./PHILOSOPHY.md).

## How this roadmap works

Each section below is a **feature parity** table: **Feature** | **Status** | **Notes**. Status values:

- **done** — Believed complete for current scope; report gaps as bugs.
- **in progress** — Active development; some areas incomplete.
- **prototype** — Experimental or placeholder; limited expectations.
- **not ready** — Not yet shipped, or not usable for general use yet.

Themes group work (language, interop, tooling, docs, infrastructure). Priority and ordering are reflected in **Notes** where it helps (e.g. near-term focus vs backlog). We do not publish dates here; use GitHub issues and milestones for concrete execution.

---

## Language & types

| Feature | Status | Notes |
| --- | --- | --- |
| Basic type system | done | Core static typing. |
| Shape-based types | done | Structural shapes. |
| Type definitions | done | User-defined types. |
| `ensure` statements (basic type assertions) | done | Basic assertions. |
| Shape guards (struct refinement) | done | Refinement on shapes. |
| `is` operator for `ensure` conditions | not ready | Near-term priority. |
| Type guards (beyond shape guards) | not ready | Near-term priority. |
| Immutability guarantees (`ensure`-scoped; unsafe mode for Go interop) | not ready | Backlog. |
| Binary type expressions | not ready | Backlog. |
| Type aliases | not ready | Backlog. |
| Generic types | not ready | Backlog. |

---

## Go interoperability

| Feature | Status | Notes |
| --- | --- | --- |
| Transpile to Go (packages, types, functions) | done | Main compiler output. |
| Runtime validation from type constraints | done | Checks emitted from types. |
| `import` of Go packages in Forst | prototype | Common paths work; not a full Go loader yet. |
| Load & typecheck imported Go source | not ready | Understand symbols beyond import lines. |
| Type-check Forst↔Go calls | not ready | Args and results at boundary calls. |
| Match Go idioms where it matters (`error`, naming) | prototype | Iterative polish; not the main focus every cycle. |
| Expose Forst functions to non-Forst callers (HTTP, RPC, subprocess) from **generated Go** | prototype | Compose servers in Go today; Forst-native handler patterns later. |

---

## TypeScript interoperability

**Story:** `forst generate` yields **types + a stub**; **calling** Forst still means wiring Node/TS to the **Go process** that runs transpiled code. A **small, documented invocation contract** (fewer ad-hoc wires) is an open thread in the rows below. **Further out:** **whole routes or TS-facing modules** implemented in Forst—not only type mirrors—once emit, packaging, and call semantics are solid.

| Feature | Status | Notes |
| --- | --- | --- |
| Declaration emit (`.d.ts` / TS types from Forst) | done | `forst generate` and TS transformer. |
| Merge outputs across `.ft` files | done | Shared `types.d.ts`; duplicate handling. |
| Client / helper stubs next to generated types | prototype | Thin surface; wire to whatever runs the compiled Forst/Go side. |
| NPM package (installable compiler / CLI) | not ready | Distribution for JS/TS ecosystems. |
| Run compiler or sidecar from Node.js | prototype | Experimental paths; not the default install yet. |
| Dev experience: watch + HTTP types (where applicable) | prototype | Iteration story; not the main path yet. |
| **Invocation:** stable contract from Node/TS to **running** Forst (compiled Go) | prototype | No single blessed pattern yet; fewer bespoke wires over time. |
| **Route- or module-level Forst** (handlers + client types in one arc) | not ready | Aspirational: full-stack slices in Forst, not only `.d.ts` for hand-written TS handlers. |

---

## Tooling & developer experience

| Feature | Status | Notes |
| --- | --- | --- |
| LSP: transport & server (`forst lsp`) | in progress | JSON-RPC over HTTP `POST /`; not stdio. See `forst/cmd/forst/lsp/`. |
| LSP: text document sync | in progress | `didOpen` / `didChange` / `didClose` wired to compilation. |
| LSP: diagnostics | in progress | From compiler pipeline; feeds editor diagnostics. |
| LSP: completion | prototype | Keyword completions; not full semantic completion. |
| LSP: hover | prototype | Placeholder hover content; not AST-aware yet. |
| LSP: go to definition | prototype | Handler exists; returns no result (`null`) for now. |
| LSP: find references | prototype | Returns empty; not implemented beyond stub. |
| LSP: document & workspace symbols | prototype | Returns empty; not implemented beyond stub. |
| LSP: formatting, code actions, code lens, folding | prototype | Advertised capabilities; stubs return empty/null. |
| LSP: compiler debug (custom methods) | prototype | debugInfo, compilerState, phaseDetails—niche tooling, not core editor parity. |
| Error messages (line numbers, suggestions) | prototype | Incremental improvements; parity not the only priority. |
| More real-world examples | prototype | Some examples exist; broader set still wanted. |
| VS Code extension | not ready | Marketplace extension and wiring to the language service. |

---

## Docs & community

| Feature | Status | Notes |
| --- | --- | --- |
| “Getting Started” guide | prototype | README is the main entry; a fuller guide is still open. |
| Example project showcasing core features | prototype | `examples/` today; a dedicated showcase repo still open. |
| Contributing guide (dev setup) | not ready | Contributor onboarding. |

---

## Infrastructure

### CI & releases

| Feature | Status | Notes |
| --- | --- | --- |
| CI pipeline (lint, tests, compiler build) | done | GitHub Actions; `task ci:test` in `forst/`. |
| Release automation | done | release-please + git tags. |

### Test coverage

| Feature | Status | Notes |
| --- | --- | --- |
| Unit / library coverage (Go) | done | `go test -cover` on `cmd/forst/...` and `internal/...`; Coveralls upload in CI. |
| Integration & example runs in CI | done | Same pipeline runs example tasks (e.g. sidecar) after unit tests. |
| Validation codegen coverage | not ready | Track **emit** (compiler output for constraints/guards) and **generated validation Go** separately from generic `internal/` coverage. |
