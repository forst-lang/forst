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

## Go backwards compatibility

**Story:** Forst emits Go so teams can adopt it beside hand-written Go in the same module and rely on the standard toolchain. Parity here means **generated code that compiles and behaves predictably**, **sensible defaults for which Go version we assume**, and (over time) **explicit knobs** when output must run on older toolchains.

| Feature | Status | Notes |
| --- | --- | --- |
| Emitted code is valid Go for the **supported compiler/toolchain** (see `forst/go.mod`) | done | Primary guarantee today: output matches what we build and test against. |
| Forst syntax that mirrors Go (subset) maps to familiar Go constructs | done | See README “backwards-compatible with Go” examples. |
| Mixed packages: Forst (`.ft`) alongside `.go` in one module / tree | prototype | Works in common layouts; edge cases still shaken out with imports and discovery. |
| Idiomatic, readable generated Go (names, structure, `error` handling) | prototype | Ongoing polish; not frozen. |
| **Selectable minimum Go version** for emitted code (e.g. emit for older `go` than the compiler) | not ready | Single emit path; no `-target` / compatibility mode yet. |
| Emit avoids language/stdlib features newer than chosen target (when targets exist) | not ready | Depends on versioned emit; backlog. |
| Output routinely **`gofmt`-clean** (or documented exceptions) | prototype | Aim for gofmt-friendly layout; not asserted everywhere in tests yet. |
| Policy for **stdlib / API deprecation** in generated code as Go releases ship | prototype | Track Go release notes over time; no separate audit pipeline yet. |
| Build tags, file splits, or shims for **per-version or per-GOOS** generated code | not ready | Backlog unless a concrete interop need lands first. |

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

**LSP (`forst lsp`):** The server exposes **JSON-RPC over HTTP** (`POST /` on the listener port), not stdio—editors need a small bridge (the in-repo VS Code extension does this). **`initialize`** returns a wide **ServerCapabilities** set (hover, completion, diagnostics, definition/references, symbols, formatting, code actions, code lens, folding, plus **experimental** debug flags). Behavior below is what **`handleLSPMethod`** in `forst/cmd/forst/lsp/` actually implements vs advertises. **Navigation:** same-file **go to definition**, **find references**, **document symbol**, and **workspace symbol** (open buffers only) for top-level functions, user types, and type guards—shared pipeline in `analyze.go`, `definition.go`, `references.go`, `symbols.go`.

| Feature | Status | Notes |
| --- | --- | --- |
| LSP: HTTP transport & process (`forst lsp`) | done | JSON-RPC on `POST /`; `GET /health` JSON health check; panic recovery on handlers. Implementation: `server.go`. |
| LSP: `initialize` / `serverInfo` | done | Returns capabilities (e.g. `textDocumentSync` with open/close + incremental, `completionProvider` with trigger characters, `hoverProvider`, `diagnosticProvider`, and providers for navigation/formatting/actions—many of the latter are still stubs). `serverInfo.name`: `forst-lsp`; version from compiler `Version`. |
| LSP: lifecycle (`shutdown`, `exit`) | done | `shutdown` returns `null`; `exit` acknowledged (see `shutdown.go`, tests). |
| LSP: text document sync | done | `didOpen` / `didChange` / `didClose`: in-memory `openDocuments` per URI; `didChange` stores the **last** `contentChanges` entry’s `text` as the full buffer and recompiles. Drives diagnostics on each update. |
| LSP: diagnostics | done | `compileForstFile`: lexer → parser → typechecker; builds `LSPDiagnostic` ranges from compiler errors. `didOpen` / `didChange` / `didClose` responses include `PublishDiagnosticsParams`; `sendDiagnosticsNotification` supports push-style clients. |
| LSP: hover (`textDocument/hover`) | in progress | **Implemented:** resolve token at LSP position, parse + typecheck when possible; hovers for **identifiers** (function signatures with optional leading `//` / `/* */` doc lines, type definitions, inferred variable types), **keywords** (short quick-info). **Skipped:** literals (by design). **Limits:** if parse fails, hover often returns nothing (errors show in diagnostics). |
| LSP: completion (`textDocument/completion`) | prototype | Returns **all** Forst keywords from `lexer.Keywords` as keyword items—**not** filtered by position or scope, **no** semantic or identifier completion. `completionProvider.resolveProvider` is **false** until `completionItem/resolve` exists. |
| LSP: go to definition | done | **`textDocument/definition`** resolves identifiers in the open buffer to defining tokens in the **same file**: top-level **`func`**, **`type`** (incl. shape/alias defs), and top-level **`is (…) Name`** type guards (brace depth avoids `ensure … is …`). Variables/parameters not yet. Implementation: `cmd/forst/lsp/definition.go`. |
| LSP: find references | done | **`textDocument/references`**: same-file identifier occurrences for symbols that match **go-to-definition** rules (top-level **func**, **type**, **type guard**). **`includeDeclaration`** respected. Locals/cross-file not yet. `references.go`. |
| LSP: document symbols | done | **`textDocument/documentSymbol`**: flat **`SymbolInformation`** for top-level **func**, **type** (skips `T_*` hash idents), **type guard**. Parse failure → `[]`. `symbols.go`. |
| LSP: workspace symbol | done | **`workspace/symbol`**: searches **open** `.ft` buffers (`openDocuments` / didOpen sync only); substring match on symbol name (case-insensitive). No disk-wide index yet. `symbols.go` / `hover_completion.go`. |
| LSP: formatting | prototype | `textDocument/formatting` returns **`null`** (no formatter yet). |
| LSP: code actions, code lens, folding | prototype | `textDocument/codeAction` / `codeLens` / `foldingRange` return **empty** arrays (or no-op); capabilities still advertised in `initialize`. |
| LSP: custom compiler / debug methods | prototype | `textDocument/debugInfo`, `textDocument/compilerState`, `textDocument/phaseDetails`—structured compiler state for tooling/LLM workflows (`server.go`); not general end-user editor parity. |
| LSP: protocol & client quirks | prototype | **HTTP** transport is non-standard for LSP; stdio clients won’t work without an adapter. Methods not handled by the switch (e.g. some client notifications) get **-32601 Method not found**. |
| Error messages (line numbers, suggestions) | prototype | Incremental improvements; parity not the only priority. |
| More real-world examples | prototype | Some examples exist; broader set still wanted. |
| VS Code extension | prototype | In-repo **`packages/vscode-forst`**: `.ft` language + grammar, HTTP LSP client, and language providers; **outline**, **go to definition**, **find references**, **workspace symbol** (open files) when the server resolves symbols. **Marketplace** / discoverability still open. |

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
