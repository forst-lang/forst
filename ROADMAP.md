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

Work is grouped below by theme.

### Core typing & declarations

| Feature | Status | Notes |
| --- | --- | --- |
| Basic type system | ✅ done | Core static typing. |
| Shape-based types | ✅ done | Structural shapes. |
| Type definitions | ✅ done | User-defined types. |
| Packages, imports, top-level `func` | ✅ done | Core compilation path (same role as Go). |
| `var`, assignments, `:=`, short declarations | ✅ done | Forst uses typed `name: Type =` and inference-friendly forms alongside Go-like patterns. |

### Optional & Result Types

| Feature | Status | Notes |
| --- | --- | --- |
| Optional / nilable value types (`T \| Nil`, `T?` sugar) | 📋 planned | Crystal-style **absence** without TypeScript `undefined` juggling; unions, narrowing, and Go lowering—see [optionals RFC hub](./examples/in/rfc/optionals/README.md) ([00](./examples/in/rfc/optionals/00-crystal-inspired-optionals.md)). |
| `Result(Success, Failure)` + structured `error` | 📋 planned | Single-return **success vs failure** in source; **failure** side stays in the **Error** family; maps to idiomatic **`(T, error)`** in Go—see [Result & error types](./examples/in/rfc/optionals/02-result-and-error-types.md) and [generics ↔ `Result`](./examples/in/rfc/generics/00-user-generics-and-type-parameters.md#result-types-generics-and-narrowing-ok-and-err). |

### Guards, `ensure`, and narrowing

| Feature | Status | Notes |
| --- | --- | --- |
| `ensure` statements (basic type assertions) | ✅ done | Validates assertions and optional blocks; does **not** narrow the subject’s type for statements **after** the `ensure` (see control-flow narrowing). |
| Shape guards (struct refinement) | ✅ done | Refinement on shapes. |
| `is` operator for `ensure` conditions | ✅ done | Parser requires `ensure … is …` (see `forst/internal/parser/ensure.go`; `ensure !ident` uses implicit `Nil()`). The typechecker enforces presence (`Present`) and type-guard subject compatibility (`forst/internal/typechecker/unify_typeguard.go`, `infer_ensure.go`); it does **not** fully validate arbitrary built-in constraint semantics beyond that. Emission: `forst/internal/transformer/go/ensure.go`, `ensure_constraint.go`. Examples: `examples/in/ensure.ft` and `task example:ensure`. |
| Type guards (beyond shape guards) | 📋 planned | Top-level guard definitions and assertion-time checks exist for shape guards; use-site **control-flow narrowing** with guard-backed `is` conditions should reuse `unify_typeguard.go` rules and will advance this row together with narrowing. |
| Immutability guarantees (`ensure`-scoped; unsafe mode for Go interop) | 📋 planned | Not implemented. |
| Binary type expressions | 🔬 experimental | Parser and AST support conjunction (`&`) and disjunction (`\|`) on type definitions (`forst/internal/parser/typedef.go`, `forst/internal/ast/typedef.go`); hashing includes `TypeDefBinaryExpr`. The typechecker does **not** yet implement meet/join semantics (no `TypeDefBinaryExpr` handling; see TODO in `forst/internal/typechecker/infer_shape.go`). Go codegen is a **placeholder** that emits `string` (`forst/internal/transformer/go/typedef_expr.go`). Do not rely on binary type semantics until inference and emit are finished. **Narrowing** and future binary types should share one internal type algebra (assertions / `TypeNode`), not a parallel representation—see control-flow narrowing. |
| Control-flow type narrowing | 🔬 experimental | **If-branch narrowing** for `if x is …` (assertion or shape RHS): refined type for the subject is recorded in the branch scope; variable types are keyed by identifier **and** source span in the typechecker so hover can differ per occurrence. Join/merge across branches and full `ensure`-successor narrowing are not done yet. Implementation: `forst/internal/typechecker/infer_if.go`, `narrow_if.go`; LSP uses `InferredTypesForVariableNode` when the variable AST node is found at the cursor. Type guards vs optionals / `Result`: [type guards & optionals](./examples/in/rfc/optionals/10-type-guards-shape-guards-and-optionals.md). |

### Generics, aliases, and nominal features

| Feature | Status | Notes |
| --- | --- | --- |
| Generic types | 📋 planned | User-declared type parameters on types and functions; see [generics RFC](./examples/in/rfc/generics/00-user-generics-and-type-parameters.md). |
| Type aliases | 🔬 experimental | Simple `type Name = BaseType` works end-to-end: definitions are stored (`forst/internal/typechecker/register.go`), alias chains resolve for field access (`forst/internal/typechecker/lookup_field.go`), and compatibility treats aliases symmetrically where wired (`forst/internal/typechecker/go_builtins.go` via `typeDefAssertionFromExpr` in `forst/internal/typechecker/utils.go`). Tests include merged-package alias returns (`forst/internal/typechecker/alias_return_test.go`). Gaps: interaction with generics and binary type expressions, and alias coverage across every compiler path—treat missing spots as bugs. |
| Methods (`func (t T) M()`) | 📋 planned | Forst is function-centric; no method declarations. |
| `interface{ }` satisfaction / embedding | 🔬 experimental | Structural shapes and Go interop differ from Go’s interface model. |
| Type assertions `x.(T)`, type switch | 📋 planned | Distinct from Forst’s `is` / `ensure` / narrowing. |
| `const` / `iota` | 🔬 experimental | `const` exists lexically; full `const`/`iota` story not aligned with Go. |

### Control flow & statements

| Feature | Status | Notes |
| --- | --- | --- |
| `if` / `else` (incl. init statement) | ✅ done | Same structural forms as Go. |
| `for` loops (infinite, condition-only, three-clause, `range`) | ✅ done | Parser, typechecker, and Go emit cover the usual Go forms; `examples/in/loop.ft` + `task example:loop`. Gaps vs Go: labeled `break`/`continue`, channel `range` nuances, Go 1.22+ integer `range`—see issues if you need them. |
| `break` / `continue` | ✅ done | Unguarded form. **Labeled** `break`/`continue` parse but are rejected in the typechecker until labels are implemented end-to-end. |
| `switch` / `case` / `default` / `fallthrough` | 📋 planned | Keywords exist in the lexer; no AST/typecheck/emit yet. **Design open:** may track Go’s **`switch`** closely, or lean toward a **`match`**-style construct (patterns, exhaustiveness) instead of—or layered on—classic **`switch`**; not decided. |
| `select` | 📋 planned | Not a Forst keyword yet; needs lexer + full statement support (see also channel row below). |
| `defer` / `go` statements | ✅ done | Matches [Go spec](https://go.dev/ref/spec): operand must be a **function or method call** (not a receive `<-ch` or other non-call). The operand **cannot be parenthesized** (`defer (f())` is rejected at parse time). Calls to certain **predeclared builtins** are forbidden—the same set Go disallows in **expression statement** context (`append`, `cap`, `complex`, `imag`, `len`, `make`, `new`, `real`, and `unsafe.*` calls listed in the spec); enforced in `forst/internal/typechecker/defer_go_validate.go`. Implementation: `forst/internal/parser/control_flow.go`, `forst/internal/typechecker/infer.go`, `forst/internal/transformer/go/statement.go`. Anonymous `go func(){ … }()` / `defer func(){ … }()` are not expressible until func literals exist in expression position. |
| Labeled statements + `goto` | 📋 planned | `goto` is lexed; no parser support. |

### Builtins, runtime, and platform

| Feature | Status | Notes |
| --- | --- | --- |
| Built-in calls (`make`, `new`, `append`, `copy`, `len`, `cap`, `close`, …) | 🔬 experimental | Predeclared Go builtins used as calls are type-checked against Go rules in `forst/internal/typechecker/go_builtins.go` (`tryDispatchGoBuiltin`). **`make` / `new` with a type argument** are rejected until the expression parser accepts **Forst type syntax** in that position (e.g. `Array(Int)`, `Map(K, V)`), not Go spellings like `[]T` or `map[K]V`. |
| Goroutines (beyond `go` call) | 🔬 experimental | `go f()` supported; scheduler/runtime is Go’s. |
| Channels (`chan`, `<-`, `range` on channel) | 🔬 experimental | Send/receive parse in some forms; `select` and full channel ergonomics missing. |
| `panic` / `recover` | 🔬 experimental | May appear in generated code; not first-class Forst keywords. |
| Build tags, `//go:build`, assembly | 📋 planned | See “Go backwards compatibility” / emit targets. |
| `unsafe` package | 🔬 experimental | Only through Go interop / generated code, not a Forst `unsafe` block. |

---

## Go interoperability

| Feature | Status | Notes |
| --- | --- | --- |
| Transpile to Go (packages, types, functions) | ✅ done | Main compiler output. |
| Runtime validation from type constraints | ✅ done | Checks emitted from types. |
| `import` of Go packages in Forst | 🔬 experimental | Common paths work; not a full Go loader yet. |
| Load & typecheck imported Go source | 🔬 experimental | `go/packages` loads import paths from `.ft` files; `TypeChecker.GoWorkspaceDir` is the directory containing `go.mod` found by walking up from the `.ft` file or compile `-root` (`internal/goload.FindModuleRoot`). `forst run` / `forst build` support optional `-root <dir>` to merge all same-package sources under that tree, matching discovery and the dev executor. |
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

| Feature | Status | Notes |
| --- | --- | --- |
| Declaration emit (`.d.ts` / TS types from Forst) | ✅ done | `forst generate` and TS transformer; contract outline: [03-forst-generate-contract.md](./examples/in/rfc/typescript-client/03-forst-generate-contract.md). |
| Merge outputs across `.ft` files | ✅ done | Shared `types.d.ts`; duplicate handling. |
| Client / helper stubs next to generated types | 🔬 experimental | Thin surface; wire to whatever runs the compiled Forst/Go side. |
| `forst dev` HTTP API + JSON contract | ✅ done | Endpoints `/health`, `/functions`, `/invoke`, `/types`, `/version` (includes **`contractVersion`** for HTTP API compatibility). Spec: [02-forst-dev-http-contract.md](./examples/in/rfc/typescript-client/02-forst-dev-http-contract.md). [`@forst/sidecar`](./packages/sidecar/README.md) reference client; `bun test` in `packages/sidecar`. |
| `forst generate` + `ftconfig` discovery | ✅ done | `forst generate` accepts **`-config`**, loads config from the target tree when omitted, and uses the same **include/exclude** discovery as `forst dev` (`forst/cmd/forst/generate.go`; tests in `generate_test.go`). |
| `@forst/sidecar` on npm and JSR | 🔬 experimental | Package metadata (`package.json`, `jsr.json`) and [publish-packages.yml](./.github/workflows/publish-packages.yml) (sidecar jobs); first registry publish is a maintainer step. |
| `@forst/cli` on npm and JSR (compiler / CLI) | 🔬 experimental | [`@forst/cli`](./packages/cli/README.md): Release Please `cli-v*` tags; lazy-download of the native `forst` from GitHub Releases; npm + JSR in [publish-packages.yml](./.github/workflows/publish-packages.yml) (CLI jobs). Compiler binaries still ship on root `v*` ([release.yml](./.github/workflows/release.yml)). Short overview: [README — npm](./README.md#npm). Not the same package as sidecar. |
| Run compiler or sidecar from Node.js | 🔬 experimental | Compiler: [`resolveForstBinary`](./packages/cli/src/resolve.ts) / [`ensureCompiler`](./packages/sidecar/src/utils.ts). Sidecar: [`@forst/sidecar`](./packages/sidecar/README.md) on npm / JSR (`npx jsr add @forst/sidecar`). Monorepo dev path unchanged. |
| Dev experience: watch + HTTP types (where applicable) | 🔬 experimental | `@forst/sidecar` + `forst dev`; **`watchRoots`**, optional **`watchGenerate`** (debounced `forst generate` after reload), **`generateTypes()`** with **`configPath`** → **`-config`**. Documented in [`packages/sidecar/README.md`](./packages/sidecar/README.md). |
| **Invocation:** stable contract from Node/TS to **running** Forst (`forst dev`) | ✅ done | HTTP `POST /invoke` JSON envelope; `ForstSidecarClient` throws typed errors (**`DevServerInvokeRejected`** when `success: false`, **`DevServerHttpFailure`** with optional **`serverErrorFromBody`** from JSON error bodies). After **`start()`**, **`versionCheck`** compares **`GET /version`** **`contractVersion`** to the sidecar build and compares compiler versions with **semver** when parseable (`version-compare.ts`). Broader integration patterns: [01-integration-profiles.md](./examples/in/rfc/typescript-client/01-integration-profiles.md). |
| **Route- or module-level Forst** (handlers + client types in one arc) | 📋 planned | Full-stack slices—not only `.d.ts` for hand-written TS handlers—not implemented. |
| OpenAPI / JSON Schema from shapes; pluggable transport (IPC, stdio); WASM / native bridges | 📋 planned | Optional tooling; not core language semantics. |
| CI: `example:sidecar-downloaded` | 🔬 experimental | May fail until a **release** ships a `forst` binary with a compatible **`dev`** subcommand; **local** `example:sidecar-local` is the CI bar today ([Taskfile.yml](./Taskfile.yml)). |

**See also:** [README (npm)](./README.md#npm), [`packages/cli/README.md`](./packages/cli/README.md), [examples/in/rfc/typescript-client/README.md](./examples/in/rfc/typescript-client/README.md) (RFC index), [examples/in/rfc/sidecar/00-sidecar.md](./examples/in/rfc/sidecar/00-sidecar.md), [examples/in/rfc/sidecar/tests](./examples/in/rfc/sidecar/tests), [examples/client-integration/README.md](./examples/client-integration/README.md).

---

## Tooling & developer experience

**LSP (`forst lsp`):** The server exposes **JSON-RPC over HTTP** (`POST /` on the listener port), not stdio—editors need a small bridge (the in-repo VS Code extension does this). **`initialize`** advertises **text sync**, **completion**, **hover**, **diagnostics**, **definition/references**, **rename** (with **prepare rename**), **document formatting**, **code actions** (`source` / `source.formatDocument`), **document/workspace symbols**, **folding**, **code lens**, plus **experimental** debug flags (`initialize.go`). `textDocument/formatting` may return **`null`** when the buffer cannot be formatted. Behavior below is what **`handleLSPMethod`** in `forst/cmd/forst/lsp/` implements. **Navigation:** **go to definition** and **find references** for top-level symbols use **merged same-package** analysis when multiple `.ft` files are open in the same directory (and on-disk peers with the same `package` clause are merged when a buffer is alone); **locals and parameters** stay binding-aware (`RestoreScope` / `LookupVariable`). **Document symbol** and **workspace symbol** (open buffers only) list top-level items—shared pipeline in `analyze.go`, `definition.go`, `references.go`, `navigation_locals.go`, `symbols.go`.

| Feature | Status | Notes |
| --- | --- | --- |
| LSP: HTTP transport & process (`forst lsp`) | ✅ done | JSON-RPC on `POST /`; `GET /health` JSON health check; panic recovery on handlers. Implementation: `server.go`. |
| LSP: `initialize` / `serverInfo` | ✅ done | Capabilities: `textDocumentSync` (open/close + incremental), `completionProvider` (trigger characters `.`, `:`, `(`, ` `; `resolveProvider`: false), `hoverProvider`, `diagnosticProvider`, definition/references, **`renameProvider`** (`prepareProvider: true`), **`documentFormattingProvider`**, **`codeActionProvider`** (`codeActionKinds`: `source`, `source.formatDocument`), document + workspace symbols, **`foldingRangeProvider`**, **`codeLensProvider`**. `serverInfo.name`: `forst-lsp`; version from compiler `Version`. `initialize.go`. |
| LSP: lifecycle (`shutdown`, `exit`) | ✅ done | `shutdown` returns `null`; `exit` acknowledged (see `shutdown.go`, tests). |
| LSP: text document sync | ✅ done | `didOpen` / `didChange` / `didClose`: in-memory `openDocuments` per URI; `didChange` stores the **last** `contentChanges` entry’s `text` as the full buffer and recompiles. Drives diagnostics on each update. |
| LSP: diagnostics | ✅ done | `compileForstFile`: lexer → parser → typechecker; builds `LSPDiagnostic` ranges from compiler errors. `didOpen` / `didChange` / `didClose` responses include `PublishDiagnosticsParams`; `sendDiagnosticsNotification` supports push-style clients. |
| LSP: hover (`textDocument/hover`) | ✅ done | **Implemented:** resolve token at LSP position, parse + typecheck when possible; hovers for **identifiers** (function signatures with optional leading `//` / `/* */` doc lines—**including doc from a merged same-package peer file** when the definition lives there—, **type** definitions, **type guards** via `Defs`, inferred variable types), **keywords** (short quick-info). **Go imports:** qualified **`pkg.Sym`** (e.g. `fmt.Println`) and **`import "path"`** string hovers show `go/types` signatures when loads succeed, plus Forst builtin or mapped return hints where applicable. **Skipped:** non-import string literals (by design). **Limits:** if parse fails, lexical keyword/identifier hovers only; `go/packages` / module layout can block Go hovers. `hover_completion.go`. |
| LSP: completion (`textDocument/completion`) | ✅ done | **Implemented:** `analyzeForstDocument`-backed completions with optional LSP **`context`** (`triggerKind`, `triggerCharacter`); **zone** heuristics (top-level vs inside block, **member-after-`.`**); **keyword** lists filtered per zone; **same-file** and **same-package** top-level symbols (merged analysis or **cross-buffer** peer open documents, deduped); **locals and parameters** via `RestoreScope` + `VisibleVariableLikeSymbols`; **member** fields/methods after `.` on Forst-typed expressions via `ListFieldNamesForType`. **`isIncomplete: true`** when other `.ft` buffers are open (index is not a full workspace scan). **Not yet:** Go **`pkg.`** exported-name completion, **`completionItem/resolve`** (`resolveProvider` stays **false**). `completion.go`. |
| LSP: go to definition | ✅ done | **`textDocument/definition`**: top-level **`func`**, **`type`**, **`is (…) Name`** type guards, **parameters**, and **local** bindings (`:=`, etc.). When **`analyzePackageGroupMerged`** applies, definitions can resolve to another **same-package** `.ft` in the merged group. **Limits:** destructured params, some nested control-flow edge cases. `definition.go`, `navigation_locals.go`. |
| LSP: find references | ✅ done | **`textDocument/references`**: occurrences that share the **same binding** as the cursor (`LookupVariable` + scope), including merged **same-package** peers when package merge is active. **`includeDeclaration`** respected. **Limits:** same as definition for tricky local patterns. `references.go`, `navigation_locals.go`. |
| LSP: rename | ✅ done | **`textDocument/prepareRename`** / **`textDocument/rename`**: same binding rules as references (locals/parameters/merged package); skips `T_*` hash idents. `rename.go`. |
| LSP: document symbols | ✅ done | **`textDocument/documentSymbol`**: flat **`SymbolInformation`** for top-level **func**, **type** (skips `T_*` hash idents), **type guard**. Parse failure → `[]`. `symbols.go`. |
| LSP: workspace symbol | ✅ done | **`workspace/symbol`**: searches **open** `.ft` buffers (`openDocuments` / didOpen sync only); substring match on symbol name (case-insensitive). No disk-wide index yet. `symbols.go` / `hover_completion.go`. |
| LSP: formatting (`textDocument/formatting`) | 🔬 experimental | **Advertised** (`documentFormattingProvider`). Handler applies the same pipeline as **Format document** code actions; may return **`null`** when formatting is not applicable. `formatting.go`. |
| LSP: folding (`textDocument/foldingRange`) | ✅ done | **Advertised** (`foldingRangeProvider`). Brace-based **region** folds from lexer tokens for top-level **function** bodies, **type** definitions (skips `T_*` hash idents), and **type guard** bodies. Parse failure → **`[]`**. `folding.go` (+ handler in `formatting.go`). |
| LSP: code actions & code lens | 🔬 experimental | **`textDocument/codeAction`**: **`source.formatDocument`** when the request allows `source` / `source.*` (empty when `only` is exclusively **`quickfix`**). **`textDocument/codeLens`**: advertised; behavior still minimal—see `server.go`. |
| LSP: custom compiler / debug methods | 🔬 experimental | `textDocument/debugInfo`, `textDocument/compilerState`, `textDocument/phaseDetails`—structured compiler state for tooling/LLM workflows (`server.go`); not general end-user editor parity. |
| LSP: protocol & client quirks | 🔬 experimental | **HTTP** transport is non-standard for LSP; stdio clients won’t work without an adapter. Methods not handled by the switch (e.g. some client notifications) get **-32601 Method not found**. |
| Error messages (line numbers, suggestions) | 🔬 experimental | Incremental improvements; quality and completeness still vary. |
| More real-world examples | 🔬 experimental | Some examples exist; broader set still wanted. |
| VS Code extension | 🔬 experimental | In-repo **`packages/vscode-forst`**: `.ft` language + grammar, HTTP LSP client, and language providers; **outline**, **folding** (when the server returns ranges), **go to definition**, **find references**, **rename**, **format document** / **format** code action, **workspace symbol** (open files) when the server resolves symbols. **Status bar:** LSP port + connection cue (idle / ready / error), click or **Forst: Focus output** opens the log. **Releases:** `vscode-forst-v*` + [publish-vscode-extension.yml](./.github/workflows/publish-vscode-extension.yml) (not tied to compiler `v*`). **Marketplace** / discoverability still open. |

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
| Colocated Go tests (`foo.go` + `foo_test.go`) | ⏳ in progress | Cursor rule **go-testing** describes the convention (stem-matched tests, exceptions for `main` / `test_utils` / generated code). New and touched production files should move toward this; LSP (`analyze`, `references`, `symbols`), `goload`, and `typechecker/go_hover` are covered first. Stem-matched tests now include e.g. `lsp/package_analysis_test.go`, `typechecker/collect_test.go`, `typechecker/unify_shape_test.go`, `typechecker/register_test.go`. |
| Deeper coverage on hot paths (LSP, goload, transformer emit) | ⏳ in progress | Packages like `internal/transformer/go` and large typechecker files benefit from **targeted** tests when behavior changes; whole-file line coverage is not the immediate bar. `internal/coveragehotspots` documents Tier-1 hotspots (prioritization, not a coverage % goal). |
| Validation codegen coverage | 🔬 experimental | Pipeline tests named `TestEmitValidation_*` in `internal/transformer/go` assert generated Go for built-in constraints and type guards; grep separately from generic coverage. Still expand emit paths over time. |
