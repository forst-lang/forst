# Forst Roadmap

Forst aims to give backend developers TypeScript-grade ergonomics while compiling to Go and interoperating with TypeScript clients ([README](./README.md)). Principles and boundaries live in [PHILOSOPHY.md](./PHILOSOPHY.md).

## How this roadmap works

Each section below is a **feature parity** table: **Feature** | **Status** | **Notes**. Status values:

- ✅ **done** — Believed complete for current scope; report gaps as bugs.
- ⏳ **in progress** — Actively being implemented **toward** the **done** bar; remaining gaps are temporary.
- 🔬 **experimental** — Something **exists** (often usable), but scope, stability, or polish are **not** at the **done** bar yet—may be a thin surface, partial coverage, or “works, but don’t rely on the contract long-term” while the surface is still maturing.
- 📋 **planned** — On the roadmap but **not yet delivered**: no usable implementation yet.

**In progress vs experimental:** **In progress** means implementation is **underway** toward **done**. **Experimental** means the feature is **out in the wild** in some form, but we are **not** yet treating it as complete—whether because large pieces are missing, behavior may change, or advertised capabilities are still stubs.

Themes group work (language, interop, tooling, docs, infrastructure). We do not publish dates here; use GitHub issues and milestones for concrete execution.

---

## Language & types

| Feature | Status | Notes |
| --- | --- | --- |
| Basic type system | ✅ done | Core static typing. |
| Shape-based types | ✅ done | Structural shapes. |
| Type definitions | ✅ done | User-defined types. |
| `ensure` statements (basic type assertions) | ✅ done | Basic assertions. |
| Shape guards (struct refinement) | ✅ done | Refinement on shapes. |
| `is` operator for `ensure` conditions | ✅ done | Parser requires `ensure … is …` (see `forst/internal/parser/ensure.go`; `ensure !ident` uses implicit `Nil()`). The typechecker enforces presence (`Present`) and type-guard subject compatibility (`forst/internal/typechecker/unify_typeguard.go`, `infer_ensure.go`); it does **not** fully validate arbitrary built-in constraint semantics beyond that. Emission: `forst/internal/transformer/go/ensure.go`, `ensure_constraint.go`. Examples: `examples/in/ensure.ft` and `task example:ensure`. |
| Type guards (beyond shape guards) | 📋 planned | Not implemented. |
| Immutability guarantees (`ensure`-scoped; unsafe mode for Go interop) | 📋 planned | Not implemented. |
| Binary type expressions | 📋 planned | Not implemented. |
| Type aliases | 📋 planned | Not implemented. |
| Generic types | 📋 planned | Not implemented. |
| `for` loops (infinite, condition-only, three-clause, `range`) | ✅ done | Parser, typechecker, and Go emit cover the usual Go forms; `examples/in/loop.ft` + `task example:loop`. Gaps: labeled `break`/`continue`, channel `range`, Go 1.22+ integer `range`—see issues if you need them. |
| `break` / `continue` | ✅ done | Unguarded form; labels not implemented yet. |
| `switch` / `case` / `select` | 📋 planned | Keywords exist in the lexer; statement support not wired through. |

---

## Go interoperability

| Feature | Status | Notes |
| --- | --- | --- |
| Transpile to Go (packages, types, functions) | ✅ done | Main compiler output. |
| Runtime validation from type constraints | ✅ done | Checks emitted from types. |
| `import` of Go packages in Forst | 🔬 experimental | Common paths work; not a full Go loader yet. |
| Load & typecheck imported Go source | 🔬 experimental | `go/packages` loads import paths from `.ft` files; `TypeChecker.GoWorkspaceDir` defaults to the source file directory (compiler + LSP). |
| Type-check Forst↔Go calls | 🔬 experimental | Qualified calls `pkg.Func` are checked against loaded Go signatures when imports resolve; primitives, slices, pointers, `error`, and `interface{}` (incl. variadic) are mapped; other Go types report an unsupported diagnostic. Builtin table still supplies return types when both exist. |
| Match Go idioms where it matters (`error`, naming) | 🔬 experimental | Iterative polish; conventions still evolving. |
| Expose Forst functions to non-Forst callers (HTTP, RPC, subprocess) from **generated Go** | 🔬 experimental | Compose servers in Go; Forst-native handler patterns not in place yet. |

---

## Go backwards compatibility

**Story:** Forst emits Go so teams can adopt it beside hand-written Go in the same module and rely on the standard toolchain. Parity here means **generated code that compiles and behaves predictably**, **sensible defaults for which Go version we assume**, and **explicit knobs** when output must run on older toolchains.

| Feature | Status | Notes |
| --- | --- | --- |
| Emitted code is valid Go for the **supported compiler/toolchain** (see `forst/go.mod`) | ✅ done | Primary guarantee today: output matches what we build and test against. |
| Forst syntax that mirrors Go (subset) maps to familiar Go constructs | ✅ done | See README “backwards-compatible with Go” examples; includes `for`/`range`/`break`/`continue`, `if` with init, and boolean literals as keywords. |
| Mixed packages: Forst (`.ft`) alongside `.go` in one module / tree | 🔬 experimental | Works in common layouts; edge cases still shaken out with imports and discovery. |
| Idiomatic, readable generated Go (names, structure, `error` handling) | 🔬 experimental | Ongoing polish; not frozen. |
| **Selectable minimum Go version** for emitted code (e.g. emit for older `go` than the compiler) | 📋 planned | Single emit path; no `-target` / compatibility mode yet. |
| Emit avoids language/stdlib features newer than chosen target (when targets exist) | 📋 planned | Depends on versioned emit; not in place until targets exist. |
| Output routinely **`gofmt`-clean** (or documented exceptions) | 🔬 experimental | Aim for gofmt-friendly layout; not asserted everywhere in tests yet. |
| Policy for **stdlib / API deprecation** in generated code as Go releases ship | 🔬 experimental | Track Go release notes over time; no separate audit pipeline yet. |
| Build tags, file splits, or shims for **per-version or per-GOOS** generated code | 📋 planned | Not implemented. |

---

## TypeScript interoperability

**Story:** `forst generate` yields **types + a stub**; **calling** Forst still means wiring Node/TS to the **Go process** that runs transpiled code. A **small, documented invocation contract** (fewer ad-hoc wires) is an open thread in the rows below. **Whole routes or TS-facing modules** implemented in Forst—not only type mirrors—are a **larger** concern than types + stub + wiring alone.

| Feature | Status | Notes |
| --- | --- | --- |
| Declaration emit (`.d.ts` / TS types from Forst) | ✅ done | `forst generate` and TS transformer. |
| Merge outputs across `.ft` files | ✅ done | Shared `types.d.ts`; duplicate handling. |
| Client / helper stubs next to generated types | 🔬 experimental | Thin surface; wire to whatever runs the compiled Forst/Go side. |
| NPM package (installable compiler / CLI) | 📋 planned | Distribution for JS/TS ecosystems. |
| Run compiler or sidecar from Node.js | 🔬 experimental | Experimental paths; installation and packaging still evolving. |
| Dev experience: watch + HTTP types (where applicable) | 🔬 experimental | Watch and HTTP-assisted workflows exist; workflow still evolving. |
| **Invocation:** stable contract from Node/TS to **running** Forst (compiled Go) | 🔬 experimental | No single blessed pattern; integration is still mostly ad hoc. |
| **Route- or module-level Forst** (handlers + client types in one arc) | 📋 planned | Full-stack slices in Forst—not only `.d.ts` for hand-written TS handlers—not implemented. |

---

## Tooling & developer experience

**LSP (`forst lsp`):** The server exposes **JSON-RPC over HTTP** (`POST /` on the listener port), not stdio—editors need a small bridge (the in-repo VS Code extension does this). **`initialize`** advertises **text sync**, **completion**, **hover**, **diagnostics**, **definition/references**, **document/workspace symbols**, **folding**, plus **experimental** debug flags (`initialize.go`). **Document formatting**, **code actions**, and **code lens** are **not** advertised until they are non–no-op; `textDocument/formatting` may still return `null` if called. Behavior below is what **`handleLSPMethod`** in `forst/cmd/forst/lsp/` implements. **Navigation:** same-file **go to definition**, **find references**, **document symbol**, and **workspace symbol** (open buffers only) for top-level functions, user types, and type guards, plus **locals and parameters** (binding-aware references via `RestoreScope` / `LookupVariable`)—shared pipeline in `analyze.go`, `definition.go`, `references.go`, `navigation_locals.go`, `symbols.go`.

| Feature | Status | Notes |
| --- | --- | --- |
| LSP: HTTP transport & process (`forst lsp`) | ✅ done | JSON-RPC on `POST /`; `GET /health` JSON health check; panic recovery on handlers. Implementation: `server.go`. |
| LSP: `initialize` / `serverInfo` | ✅ done | Capabilities: `textDocumentSync` (open/close + incremental), `completionProvider` (trigger characters `.`, `:`, `(`, ` `; `resolveProvider`: false), `hoverProvider`, `diagnosticProvider`, definition/references, document + workspace symbols, **`foldingRangeProvider`**. Formatting / code action / code lens providers omitted until implemented. `serverInfo.name`: `forst-lsp`; version from compiler `Version`. `initialize.go`. |
| LSP: lifecycle (`shutdown`, `exit`) | ✅ done | `shutdown` returns `null`; `exit` acknowledged (see `shutdown.go`, tests). |
| LSP: text document sync | ✅ done | `didOpen` / `didChange` / `didClose`: in-memory `openDocuments` per URI; `didChange` stores the **last** `contentChanges` entry’s `text` as the full buffer and recompiles. Drives diagnostics on each update. |
| LSP: diagnostics | ✅ done | `compileForstFile`: lexer → parser → typechecker; builds `LSPDiagnostic` ranges from compiler errors. `didOpen` / `didChange` / `didClose` responses include `PublishDiagnosticsParams`; `sendDiagnosticsNotification` supports push-style clients. |
| LSP: hover (`textDocument/hover`) | ⏳ in progress | **Implemented:** resolve token at LSP position, parse + typecheck when possible; hovers for **identifiers** (function signatures with optional leading `//` / `/* */` doc lines, type definitions, inferred variable types), **keywords** (short quick-info). **Go imports:** qualified **`pkg.Sym`** (e.g. `fmt.Println`) and **`import "path"`** string hovers show `go/types` signatures when loads succeed, plus Forst builtin or mapped return hints where applicable. **Skipped:** non-import string literals (by design). **Limits:** if parse fails, hover often returns nothing (errors show in diagnostics). |
| LSP: completion (`textDocument/completion`) | ⏳ in progress | **Implemented:** `analyzeForstDocument`-backed completions with optional LSP **`context`** (`triggerKind`, `triggerCharacter`); **zone** heuristics (top-level vs inside block, **member-after-`.`**); **keyword** lists filtered per zone; **same-file** functions / user types / type guards; **locals and parameters** via `RestoreScope` + `VisibleVariableLikeSymbols`; **member** fields/methods after `.` via expression inference + `ListFieldNamesForType`; **parse/diagnostic-aware** positioning helpers (`error_position.go`) for stable request handling. Skips strings/comments. **Not yet:** cross-file/import-qualified symbols, `completionItem/resolve` (`resolveProvider` stays **false**). `completion.go`. |
| LSP: go to definition | ✅ done | **`textDocument/definition`** resolves identifiers in the open buffer to defining tokens in the **same file**: top-level **`func`**, **`type`** (incl. shape/alias defs), top-level **`is (…) Name`** type guards (brace depth avoids `ensure … is …`), plus **parameters** and **local** bindings (`:=`, etc.) using scope restoration and AST/token mapping. **Limits:** destructured params, some nested control-flow edge cases, and **cross-file** symbols are not covered yet. `definition.go`, `navigation_locals.go`. |
| LSP: find references | ✅ done | **`textDocument/references`**: same-file occurrences that share the **same variable binding** as the cursor (via `LookupVariable` + defining scope), for top-level navigable symbols **and** locals/parameters. **`includeDeclaration`** respected. **Limits:** same as definition for locals; **cross-file** not yet. `references.go`, `navigation_locals.go`. |
| LSP: document symbols | ✅ done | **`textDocument/documentSymbol`**: flat **`SymbolInformation`** for top-level **func**, **type** (skips `T_*` hash idents), **type guard**. Parse failure → `[]`. `symbols.go`. |
| LSP: workspace symbol | ✅ done | **`workspace/symbol`**: searches **open** `.ft` buffers (`openDocuments` / didOpen sync only); substring match on symbol name (case-insensitive). No disk-wide index yet. `symbols.go` / `hover_completion.go`. |
| LSP: formatting (`textDocument/formatting`) | 🔬 experimental | **Not** advertised in `initialize`. Handler returns **`null`** (no formatter yet). `formatting.go`. |
| LSP: folding (`textDocument/foldingRange`) | ✅ done | **Advertised** (`foldingRangeProvider`). Brace-based **region** folds from lexer tokens for top-level **function** bodies, **type** definitions (skips `T_*` hash idents), and **type guard** bodies. Parse failure → **`[]`**. `folding.go` (+ handler in `formatting.go`). |
| LSP: code actions & code lens | 📋 planned | **Not** advertised in `initialize`. `textDocument/codeAction` / `codeLens` handlers return **empty** arrays if invoked. |
| LSP: custom compiler / debug methods | 🔬 experimental | `textDocument/debugInfo`, `textDocument/compilerState`, `textDocument/phaseDetails`—structured compiler state for tooling/LLM workflows (`server.go`); not general end-user editor parity. |
| LSP: protocol & client quirks | 🔬 experimental | **HTTP** transport is non-standard for LSP; stdio clients won’t work without an adapter. Methods not handled by the switch (e.g. some client notifications) get **-32601 Method not found**. |
| Error messages (line numbers, suggestions) | 🔬 experimental | Incremental improvements; quality and completeness still vary. |
| More real-world examples | 🔬 experimental | Some examples exist; broader set still wanted. |
| VS Code extension | 🔬 experimental | In-repo **`packages/vscode-forst`**: `.ft` language + grammar, HTTP LSP client, and language providers; **outline**, **folding** (when the server returns ranges), **go to definition**, **find references**, **workspace symbol** (open files) when the server resolves symbols. **Marketplace** / discoverability still open. |

---

## Docs & community

| Feature | Status | Notes |
| --- | --- | --- |
| “Getting Started” guide | 🔬 experimental | README is the main entry; a fuller guide is still open. |
| Example project showcasing core features | 🔬 experimental | `examples/` today; a dedicated showcase repo still open. |
| Contributing guide (dev setup) | 📋 planned | Contributor onboarding. |

---

## Infrastructure

### CI & releases

| Feature | Status | Notes |
| --- | --- | --- |
| CI pipeline (lint, tests, compiler build) | ✅ done | GitHub Actions; `task ci:test` in `forst/`. |
| Release automation | ✅ done | release-please + git tags. |

### Test coverage

| Feature | Status | Notes |
| --- | --- | --- |
| Unit / library coverage (Go) | ✅ done | `go test -cover` on `cmd/forst/...` and `internal/...`; Coveralls upload in CI. |
| Integration & example runs in CI | ✅ done | Same pipeline runs example tasks (e.g. sidecar) after unit tests. |
| Colocated Go tests (`foo.go` + `foo_test.go`) | ⏳ in progress | Cursor rule **go-testing** describes the convention (stem-matched tests, exceptions for `main` / `test_utils` / generated code). New and touched production files should move toward this; LSP (`analyze`, `references`, `symbols`), `goload`, and `typechecker/go_hover` are covered first. |
| Deeper coverage on hot paths (LSP, goload, transformer emit) | ⏳ in progress | Packages like `internal/transformer/go` and large typechecker files benefit from **targeted** tests when behavior changes; whole-file line coverage is not the immediate bar. |
| Validation codegen coverage | 📋 planned | Track **emit** (compiler output for constraints/guards) and **generated validation Go** separately from generic `internal/` coverage. |
