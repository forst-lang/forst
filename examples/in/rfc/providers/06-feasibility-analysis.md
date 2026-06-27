# Function requirements (05) — feasibility analysis

**Audience:** Compiler implementers, RFC coordinators, tech leads.

**Status:** Engineering assessment — evaluates [SPEC — Function requirements](./SPEC.md) against the current Forst compiler.

**Depends on:** [SPEC — Function requirements](./SPEC.md), [00 — Prior art §5](./00-prior-art.md#5-forst-codebase-audit-current-state).

---

## 1. Executive summary

**Verdict: Partially feasible — go with phased delivery; do not ship 05 as a single release.**

The design’s **Go lowering target** (interfaces + per-function needs struct + struct literals at boundaries) aligns with how the transformer already emits functions, shapes, and imports. The **compiler value proposition** — transitive need inference and entry-point completeness — requires a **new analysis pass** that does not exist today and depends on several **prerequisite language features** the design assumes but the codebase lacks.

| Area | Assessment |
| --- | --- |
| Parser (`use`, `with`) | **Feasible (S)** — same pattern as `ensure`; keyword map in `lexer/tokens.go` |
| Contract types (shapes + methods) | **Blocked for MVP (L)** — no receiver-method grammar; shape fields are data-only |
| Requirement inference + completeness | **Hard (XL)** — no call graph, no scope requirement scope |
| Go imported interfaces as contracts | **Partial (M–L)** — `go/types` loads packages; interface identity is opaque (`TYPE_IMPLICIT`) |
| Needs struct + `with` lowering | **Feasible once analysis exists (M)** — extends `transformFunction` / call sites |
| Integration with handlers / sidecar | **Feasible (S–M)** — explicit `ctx` params already match sidecar data-only wire format |

**Recommended phasing**

1. **MVP (L):** Receiver methods + method signatures in contract shapes; `use` binding; explicit `with` map blocks; **manual** needs struct emission (one struct per function from collected `use` sites); **no** transitive completeness yet.
2. **v1 (XL):** Call-graph need propagation, scope `with` scopes (always-forward), `with ctx` field forwarding, postfix override merge, completeness diagnostics at wiring roots.
3. **v2 (L):** Imported Go interface contracts with stable identity, cross-package fixed-point, Go-side discovery/LSP `needs` JSON (not TypeScript).

**Go/no-go:** **Conditional go** — proceed if MVP accepts shipping **syntax + lowering first** and treats inference as the mandatory v1 milestone (per [04 § executive summary](./04-redesign.md): without the checker, keywords are ~85% redundant with hand-written Go).

---

## 2. Design recap

[SPEC](./SPEC.md) adds two keywords: **`use`** binds a requirement (contract type) inside a function body; **`with`** supplies implementations at a scope boundary or call site. Contracts are ordinary Forst types — shapes with methods, nominal types, or imported Go types — not a parallel declaration family. The compiler **infers** each function’s need set from `use` sites (and optionally method calls on designated context parameters), **propagates** needs transitively through calls, and **rejects** wiring roots where any need lacks a supplier. Lowering synthesizes a `fnNeeds` struct and prepends it to function signatures, matching idiomatic Go constructor injection; `with` blocks become struct literals and field copies.

---

## 3. Feasibility by component

| Component | Verdict | Effort | Risk | Depends on |
| --- | --- | --- | --- | --- |
| **Parser: `use` stmt** | Feasible | S | Low | Lexer keyword; mirror `parseEnsureStatement` (`parser/ensure.go`) |
| **Parser: `with` blocks / postfix** | Feasible | M | Medium | Block stmt in `parseBlockStatement`; postfix needs Pratt/call suffix in `parser/expression.go` |
| **Typechecker: requirement inference** | Not present | XL | High | Call graph over `tc.Functions`; new `Needs` map per function |
| **Typechecker: transitive closure** | Not present | L | High | Fixed-point over call edges; inner `with` subtracts satisfied keys |
| **Typechecker: completeness at entry points** | Not present | M | Medium | Wiring-root detection (`main`, exported handlers); scope scope stack |
| **Typechecker: structural vs nominal satisfaction** | Partial | L | High | `shapesAreStructurallyIdentical` (`go_builtins.go`) compares **fields only**; distinct nominal errors already leak structural equality (`error_nominal_test.go`) |
| **Typechecker: imported Go interface resolution** | Partial | M | Medium | `initGoImportPackages` + `goPackageForImportLocal` (`go_interop.go`); interfaces become `TYPE_IMPLICIT` in `goTypeToForstType` |
| **Transformer: synthesized `fnNeeds` struct** | Not present | M | Low | Pointer fields per [ADR-013](./ADR.md#adr-013-pointer-fields-in-needs-structs); emit via `Output.AddType` |
| **Transformer: `with` → struct literals** | Not present | M | Medium | Scope-aware call transformer; rewrites arity at call sites |
| **Transformer: `with ctx` field extraction** | Not present | M | Medium | Shape field walk + assignability to need keys |
| **Transformer: scope merge / override at call site** | Not present | M | Medium | ProviderScope binding stack in checker; merge override keys with scope |
| **Go wrap syntax for foreign symbols** | Partial | M | Medium | Qualified types parse (`parser/type.go`); alias typedef lowers to named Go type; adapter methods need receiver methods |
| **Integration with handler/context patterns** | Feasible | S | Low | Handler service structs (e.g. `UserApiServices`) register via `registerShapeType` (`register.go`); sidecar stays data-only |

---

## 4. Compiler pipeline changes

### Current pipeline

```
lexer → parser → typechecker.CheckTypes (collect + infer) → transformergo.TransformForstFileToGo → generators.GenerateGoCode
```

See `compiler/compile_pipeline.go`.

### Likely touched packages / files

| Layer | Files / packages | Change |
| --- | --- | --- |
| **Lexer** | `internal/lexer/tokens.go`, `internal/ast/tokens.go` | `TokenUse`, `TokenWith` |
| **AST** | New: `internal/ast/use.go`, `internal/ast/with.go`; extend `internal/ast/node.go` kinds | `UseNode`, `WithBlockNode`, `WithCallSuffixNode`, optional `RequirementMapNode` |
| **Parser** | `internal/parser/statement.go`, new `use.go` / `with.go`, `internal/parser/expression.go` | Statement and postfix forms; **also** receiver methods in `function.go`, method fields in `shape.go` |
| **Printer** | `internal/printer/printer.go` | Round-trip for new nodes |
| **Typechecker** | New: `requirements.go`, `requirements_infer.go`, `requirements_satisfy.go`; extend `collect.go`, `infer.go`, `infer_expression.go` | Needs collection, propagation, scope env, diagnostics |
| **Typechecker (prereq)** | `register.go`, `infer_expression.go`, `builtins.go` | Method registry for Forst receiver methods; method-set satisfaction |
| **Transformer** | `internal/transformer/go/function.go`, new `requirements.go`, `statement.go`, `expression.go` | Prepend needs param; strip `use` to locals; rewrite calls inside `with` |
| **Discovery / LSP** | `internal/discovery/discovery.go` | Go-side `needs: []string` on `FunctionInfo` (v2); filtered from TS manifest per [ADR-011](./ADR.md#adr-011-typescript-never-sees-requirements) |
| **Examples / tests** | `examples/in/rfc/providers/`, `examples/out/` | Golden pipeline tests per phase |

### New AST nodes (minimum)

```text
UseNode          { Name Ident, ContractType TypeNode, Span }
WithBlockNode    { Supply WithSupplyExpr, Body []Node }   // Supply: ident | map
WithCallNode     { Call FunctionCallNode, Supply WithSupplyExpr, Forward bool }
MethodDefNode    { Receiver SimpleParamNode, Name Ident, Params, ReturnTypes, Body }  // prerequisite
ShapeMethodField { ... }  // or extend ShapeFieldNode with signature
```

### New typechecker state (minimum)

```text
FunctionNeeds    map[Identifier]set[TypeIdent]   // direct + propagated
ScopeRequirements stack per WithBlockNode
RequirementKey     TypeIdent                     // nominal identity (N6)
```

A **third pass** (after infer, before transform) is the cleanest insertion point for propagation + completeness; alternatively extend `CheckTypes` with pass 2b.

---

## 5. Hard problems (ranked by severity)

### 1. Inference fixed-point across packages — **Critical**

`CheckTypes` runs per merged package (`compiler/package.go` loads same-package `.ft` files). There is **no** cross-package call graph or exported-needs summary. Transitive rule N4 requires `Needs(f) ⊇ Needs(g)` for every call `f → g`, including across files and (eventually) packages.

**Mitigation:** Single-package fixed-point in v1; export inferred needs in discovery JSON for cross-package v2; document that incomplete cross-package wiring may slip until v2.

### 2. Ambiguity: `use x: T` when T is shape vs interface vs concrete — **High**

- Shape typedefs lower to **Go structs** today (`transformTypeDef` → `TypeDefShapeExpr` → struct), not interfaces.
- Parser `interface { … }` accepts method syntax but discards signatures → `TYPE_OBJECT` (`parser/type.go` lines 68–117).
- Imported Go interfaces map to **`TYPE_IMPLICIT`** (`go_interop.go` `goTypeToForstType`), losing method-set identity for satisfaction checks.

**Mitigation:** Introduce explicit **contract kind** in typedef (05 avoids `capability` keyword but compiler still needs a shape-with-methods → interface emission path); prefer Go alias typedefs for foreign interfaces in MVP.

### 3. `with` merge / shadow semantics — **High**

Requires an **scope requirement environment** threaded through scopes (similar to `ScopeStack` in `scope_stack.go` but keyed by requirement type, not variable). Inner maps override outer per key; **all inner calls auto-forward** scope to callees (no opt-in modifier). No existing pattern — `ensure` scopes are for narrowing, not supply.

**Mitigation:** Implement explicit map + `with ctx` + always-forward merge; strict override rules with tests per row in 05’s merge table and [ADR-004/005](./ADR.md).

### 4. Go import type identity in Forst typechecker — **High**

`forstAssignableToGoType` bridges Forst ↔ Go at call boundaries but contract checking needs **stable TypeIdent → go/types.Type** for interfaces like `io.Writer`. Today only `variableGoTypes` tracks locals from Go calls; typedef `type W = io.Writer` is an unresolved qualified name unless registered.

**Mitigation:** Extend `Defs` registration for `pkg.Type` aliases when `goPkgsByLocal` resolves; use `types.AssignableTo` for satisfaction at `with` sites.

### 5. Wrapping arbitrary Go funcs (generics, methods, variadics) — **Medium**

Go interop handles qualified calls and tracked receivers (`checkGoMethodCall`, `checkGoQualifiedCall`). Generics, variadic spread, and method expressions are unevenly covered. Wrapper adapters in 05 require **Forst receiver methods** — **not implemented** (see §6).

**Mitigation:** Document supported wrap patterns; hand-write thin adapters in Go for edge cases in MVP.

### 6. Interaction with `ensure`, Result, shape guards — **Medium**

- `ensure` occupies statement prefix syntax and failure/narrowing semantics (`ast/ensure.go`, `infer_ensure.go`) — orthogonal but parser must disambiguate `use` vs mistyped `ensure`.
- `Result` method calls (`Ok`/`Err`) are special-cased in `inferMethodCallType` — must not be classified as requirements.
- Shape guards / `is` narrowing use structural comparison (`shapesAreStructurallyIdentical`) — reusable for **implementation** checking, not for distinct requirement keys (N6 conflicts with today’s nominal-error structural leak).

**Mitigation:** Requirement keys always use **typedef ident**, never structural hash; exclude `Result` and guard-only types from need inference (N7).

---

## 6. What already exists

| 05 feature | Existing capability | Location |
| --- | --- | --- |
| Go `import` paths | Parsed and collected | `ast/import.go`, `parser/imports.go`, `collect.go` |
| Go package loading | `go/packages` via goload | `goload/load.go`, `typechecker/initGoImportPackages` in `go_interop.go` |
| Qualified Go calls | Type-checked with `go/types` | `checkGoQualifiedCall`, tests in `go_interop_test.go` |
| Go method calls on tracked receivers | `variableGoTypes` + `checkGoMethodCall` | `typechecker.go`, `infer_expression.go` |
| Forst ↔ Go assignability | `forstAssignableToGoType` | `go_interop.go` (incl. opaque pointer → interface) |
| Structural shape comparison | Field-wise identity | `shapesAreStructurallyIdentical` in `go_builtins.go` |
| Shape / typedef registration | `registerType`, `registerShapeType` | `register.go` |
| Function signatures + call checking | `tc.Functions`, arity checks | `registerFunction`, `infer_expression.go` |
| Scope stack for nesting | push/pop/restore | `scope_stack.go`, used by `ensure`, `if`, `for` |
| Function → Go lowering | 1:1 params + body | `transformer/go/function.go` |
| Struct literal emission | Shape lowering | `expression_shape_literal.go`, `shape.go` |
| Import emission in Go output | `Output.AddImport` | `transformer/go/output.go` |
| Handler-style context structs | Shape typedefs (data fields) | Patterns in RFC `02-examples.ft`; `registerShapeType` |
| Sidecar data-only boundary | Documented | `sidecar/10-decisions.md`, `00-prior-art.md` §4.3 |
| Statement keyword extension precedent | `ensure` | `lexer/tokens.go`, `parser/ensure.go`, `transformer/go/statement_ensure.go` |
| Merged multi-file packages | Collect order independence | `collect_order.go`, `compiler/package.go` |
| Sealed interface emission (pattern) | Nominal error unions | `error_union_sealed.go` — shows interface + method emission machinery |

---

## 7. What is net-new

| Capability | Notes |
| --- | --- |
| **`use` / `with` grammar and AST** | Zero matches in `forst/internal` today |
| **Forst receiver methods** | `parseFunctionDefinition` only parses `func name(...)` — no `func (recv Type) method(...)` (`parser/function.go`) |
| **Method signatures in shape types** | `parseShapeTypeField` expects `field: Type` or bare ident; `info(msg String)` is not parsed |
| **Method registry + user-defined method calls** | `CheckBuiltinMethod` returns error for non-builtin types (`builtins.go`); no lookup of Forst methods on user structs |
| **Contract → Go interface emission** | Shapes lower to structs; no capability interface pass |
| **Per-function needs struct synthesis** | Transformer emits only explicit params |
| **Call-site rewriting for injected needs** | All calls use declared arity from `tc.Functions` |
| **Requirement inference pass** | No `Needs(f)` collection or transitive closure |
| **ProviderScope requirement scopes** | Distinct from variable scopes |
| **Completeness diagnostics** | No wiring-root analysis |
| **`with forward` postfix** | Removed by [ADR-004](./ADR.md); scope always forwards |
| **Field → requirement key convention** | `logger` → `Logger` heuristic undocumented in compiler |
| **Discovery `needs` metadata** | `discovery.FunctionInfo` has params only |
| **Imported type as first-class contract** | Qualified typedefs not linked to `go/types` in `Defs` |

---

## 8. Phased implementation plan

### MVP — syntax, contracts foundation, manual wiring lowering

**Effort: L (6–10 engineer-weeks)**

**Scope**

- Receiver method syntax + method registry on Forst types.
- Method-shaped fields in contract typedefs **or** explicit `interface`-like typedef form that preserves signatures (compiler-internal; no `capability` keyword required).
- `use x: T` / `use T` statements; type-check contract type.
- `with { Key: value, … } { … }` and `call(...) with { … }` (no `forward`, no `with ctx` yet).
- Emit `type expireTokenNeeds struct { … }` from **direct** `use` sites in each function; prepend to signature; lower `use` to local field reads.
- Single-file completeness: `main` must `with`-wrap calls whose callee needs are unsatisfied in empty scope env.

**Acceptance criteria**

- `examples/in/rfc/providers/SPEC.md` token-expiry example compiles through `task example:*` golden (subset without `with ctx`).
- Unit tests: parse `use`/`with`; needs struct emission; one integration test calling `expireToken` with literal needs struct.
- Receiver method call `logger.info(...)` type-checks on `StdLogger`.

**Not in MVP**

- Transitive propagation, `with ctx`, scope always-forward + override merge, imported `io.Writer` as contract, cross-package.

---

### v1 — inference + handler ergonomics

**Effort: XL (10–16 engineer-weeks)**

**Scope**

- Build call graph from function bodies; compute `Needs(f)` with rules N1–N5.
- ProviderScope env for nested `with`; inner shadowing.
- `with ctx { … }` field forwarding when `ctx` is shape-typed and fields satisfy needs.
- ProviderScope auto-forward and postfix override merge ([ADR-004](./ADR.md), [ADR-005](./ADR.md)).
- Wiring-root completeness errors with chain diagnostics (as sketched in 05).
- Designated context param detection (`ctx`, `deps`, `app`).

**Acceptance criteria**

- Adding `use metrics: Metrics` in a leaf errors at handler/`main` with transitive path.
- Handler example from 05 (`handleRefresh` / `expireToken`) passes compile + golden.
- Regression: `ensure`, `Result`, shape guards unchanged (dedicated tests).

---

### v2 — Go-native contracts + tooling

**Effort: L (6–10 engineer-weeks)**

**Scope**

- Typedef alias to imported Go interface (`type W = io.Writer`) with `go/types` identity in needs struct.
- Cross-package needs summary (discovery JSON / LSP).
- Optional needs-struct merge optimization for identical need sets (05 open Q1).
- TS generate `@needs` from same inference graph — **removed** ([ADR-011](./ADR.md#adr-011-typescript-never-sees-requirements)).

**Acceptance criteria**

- `use w: io.Writer` lowers to `io.Writer` in needs struct without duplicate interface.
- `discovery` exposes `needs` for exported functions.
- CI integration test pattern (`ciUserApiServices`) compiles in merged package test.

---

## 9. Performance / Go output quality

| Concern | Assessment |
| --- | --- |
| **Allocation** | One needs struct per call when using postfix `with` — same as hand-written Go; `with ctx` block amortizes one literal per scope. |
| **Interface boxing** | Contracts emitted as Go interfaces; concrete fakes stay structs — standard Go pattern; no extra boxing beyond interface dispatch the design already chooses. |
| **Inlining** | Extra indirection through interface methods may inhibit inlining vs concrete deps — acceptable for testability boundary; document that hot paths may keep data params concrete. |
| **Code size** | Per-function needs structs duplicate field lists when needs sets repeat — v2 merge pass reduces bloat. |
| **Compile time** | Fixed-point inference adds one pass over function graph; negligible vs existing `go/types` load for imports. |

---

## 10. Testing strategy

| Layer | Approach |
| --- | --- |
| **Parser** | Colocated `use_test.go`, `with_test.go` — token streams like `ensure_test.go` |
| **Typechecker** | Unit tests for `Needs` collection, propagation, shadow, completeness errors; table-driven satisfaction cases |
| **Transformer** | Golden Go snippets for needs struct + call rewrite; extend `pipeline_integration_test.go` |
| **Integration** | New `examples/in/rfc/providers/providers_mvp.ft` + `examples/out/` golden via `task example:providers` |
| **Regression** | Run existing `task test:integration` / full `go test ./...` — requirements must not break `ensure`, Result, guards |
| **Interop** | Extend `go_interop_test.go` patterns for `io.Writer` contract wiring |

Prefer **precise test names** per project priorities (e.g. `TestProviders_propagateNeedsThroughCallChain`).

---

## 11. Blockers and mitigations

| Blocker | Severity | Mitigation |
| --- | --- | --- |
| No receiver methods | **Ship-stopper for 05 examples** | MVP phase 0: receiver syntax + method table before `use` |
| Shapes ≠ interfaces in emit | **High** | Contract typedef flag or method-bearing shape → interface transform |
| `TYPE_IMPLICIT` for Go interfaces | **High** | Register resolved `go/types` for imported contract aliases |
| No call graph | **High** | v1 dedicated pass; MVP limits to direct needs + explicit call-site `with` |
| Nominal vs structural confusion (N6) | **Medium** | Requirement keys = typedef name only; tighten nominal error compat separately |
| Keyword collision (`use` is common) | **Low** | Statement-only context (like `ensure`); not valid as expression identifier in statement position |
| 02-examples.ft / 05 use invalid syntax today | **Low** | Treat RFC sources as spec; add compiling examples per phase |

---

## 12. Verdict

**Recommendation: Conditional go.**

Proceed with [SPEC](./SPEC.md) as the **surface-area target** (two keywords, types as contracts) **only if** the team commits to:

1. **Receiver methods + contract interface emission** as MVP prerequisites — the design’s examples do not parse or type-check today.
2. **v1 inference as non-optional** — otherwise stop after MVP and document manual needs structs (same break-even as [04](./04-redesign.md)).
3. **Explicit deferral** of cross-package fixed-point and full generic Go wrap to v2.

**Do not** block on a `capability` keyword or `requirement` blocks — none required. **Do** block on method semantics and checker infrastructure before marketing compile-time completeness.

**Effort summary**

| Phase | Effort | Cumulative |
| --- | --- | --- |
| MVP | **L** | L |
| v1 | **XL** | L + XL |
| v2 | **L** | L + XL + L |

**Risk-adjusted timeline:** ~2–3 quarters for v1-quality 05 (one experienced compiler engineer, part-time RFC coordination), assuming no major sidecar or Effect monad scope creep.

---

## References (codebase)

- Lexer keywords: `forst/internal/lexer/tokens.go`
- Parser statements: `forst/internal/parser/statement.go`, `ensure.go`, `function.go`
- Typechecker pipeline: `forst/internal/typechecker/typechecker.go` (`CheckTypes`)
- Go interop: `forst/internal/typechecker/go_interop.go`
- Structural compare: `forst/internal/typechecker/go_builtins.go` (`shapesAreStructurallyIdentical`)
- Function lowering: `forst/internal/transformer/go/function.go`
- Compile pipeline: `forst/internal/compiler/compile_pipeline.go`
- Prior art audit: `examples/in/rfc/providers/00-prior-art.md` §5
