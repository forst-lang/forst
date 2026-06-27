# Architecture Decision Records — function requirements

**Status:** Accepted decisions for the [SPEC — Function requirements](./SPEC.md) design.

**Format:** Each record states **one** context, **one** decision, and consequences. Supersedes scattered open questions where noted.

**Normative design:** [SPEC — Function requirements](./SPEC.md)

**Disposition ledger:** [09 — Design solutions](./09-design-solutions.md) items are locked in ADR-003–046 below (replacing the former bundled ADR-016).

---

## Index

| ADR | Title |
| --- | --- |
| [001](#adr-001-two-keywords-only--use-and-with) | Two keywords: `use`, `with` |
| [002](#adr-002-contracts-are-ordinary-types) | Contracts are ordinary types |
| [003](#adr-003-no-author-written-usables-clause-on-signatures) | No author-written `usables` clause |
| [004](#adr-004-usablesf-inferred-from-use-and-transitive-calls-only) | `Usables(f)` from `use` + calls |
| [005](#adr-005-context-parameters-never-contribute-to-usablesf) | Context fields never infer |
| [006](#adr-006-usablesf-is-derived-tooling-sugar-only) | `Usables(f)` is tooling sugar |
| [007](#adr-007-always-forward-ambient-no-with-forward) | Always forward ambient |
| [008](#adr-008-nested-with-overlays-outer-wiring) | Nested `with` overlays |
| [009](#adr-009-no-postfix-with-on-calls) | No postfix `with` |
| [010](#adr-010-no-subtract-syntax-for-usables) | No subtract syntax |
| [011](#adr-011-parameters-are-data-use-is-runtime-logic) | Data vs runtime logic |
| [012](#adr-012-usables-struct-as-first-parameter) | Usables struct first param |
| [013](#adr-013-deduped-usables-struct-naming) | Deduped struct naming |
| [014](#adr-014-no-runtime-di-container-or-service-locator) | No DI container |
| [015](#adr-015-unsatisfied-usables-are-hard-compile-errors) | Unsatisfied Usables → error |
| [016](#adr-016-transitive-usables-inference) | Transitive inference |
| [017](#adr-017-no-reduced-mode-without-transitive-completeness) | No reduced mode |
| [018](#adr-018-orthogonal-to-ensure-and-result) | Orthogonal to `ensure` / `Result` |
| [019](#adr-019-superset-wiring-allowed) | Superset wiring allowed |
| [020](#adr-020-typescript-emit-excludes-usables-concept) | TS excludes Usables |
| [021](#adr-021-runnable-exports-only-when-usablesf-is-empty) | Runnable exports only |
| [022](#adr-022-hard-error-on-sidecar-export-with-non-empty-usablesf) | Sidecar export hard error |
| [023](#adr-023-go-side-discovery-json-for-usablesf) | Go-side discovery JSON |
| [024](#adr-024-known-but-unused-wiring-keys-warning) | Unused keys → warning |
| [025](#adr-025-unknown-wiring-keys-hard-error) | Unknown keys → error |
| [026](#adr-026-usables-struct-fields-by-value) | Struct fields by value |
| [027](#adr-027-first-parameter-named-usables-use-binds-from-fields) | `usables` param + `use` bind |
| [028](#adr-028-wiring-value-pointers-optional) | Wiring pointers optional |
| [029](#adr-029-mock-reuse-by-convention-not-compiler-enforced) | Mock reuse by convention |
| [030](#adr-030-with-takes-usable-shape-literals-only) | `with` shape literals only |
| [031](#adr-031-no-with-ctx-struct-forwarding) | No `with ctx` |
| [032](#adr-032-handlers-not-special-for-usewith) | Handlers not special |
| [033](#adr-033-trun--nested-with-permanent-table-test-path) | `t.Run` + nested `with` tables |
| [034](#adr-034-lsp-ships-at-feature-ga) | LSP at GA |
| [035](#adr-035-cross-package-graph-normative-before-ga) | Cross-package before GA |
| [036](#adr-036-single-graph-source-for-checker-json-and-lsp) | Single graph source |
| [037](#adr-037-nil-forbidden-in-wiring) | Nil forbidden |
| [038](#adr-038-go-_testgo-escape-hatch) | Go `_test.go` escape hatch |
| [039](#adr-039-no-optional-use-syntax) | No optional `use` |
| [040](#adr-040-no-brownfield-bridge-syntax-v1) | No brownfield bridge |
| [041](#adr-041-root-contract-ident-only-in-with-keys) | Root ident keys only |
| [042](#adr-042-usable-and-usables-vocabulary) | Usable / Usables vocabulary |
| [043](#adr-043-satisfaction-relation-for-wiring) | Satisfaction relation |
| [044](#adr-044-test-entrypoints-use-test-and-testingt) | Test entrypoints |
| [045](#adr-045-test-harness-excluded-from-usablesf) | Harness excluded |
| [046](#adr-046-ensure-in-tests-lowers-to-tfatal--terror) | `ensure` in tests |

---

## ADR-001: Two keywords only — `use` and `with`

**Context.** Prior proposals introduced `requirement`, `require`, `provide`, `supply`, `capability`, `harness`, `override`, and `test` keywords. Forst philosophy favors a small spec and English-readable control flow.

**Decision.** The surface is exactly two keywords: **`use`** binds a Usable inside a function body; **`with`** wires implementations at a scope boundary.

**Consequences.**

- No parallel declaration family for contracts.
- Integration tests use plain functions (`ciUserApiServices()`) and `with`, not a test DSL.
- Compiler and docs focus on two verbs agents and humans must learn.

---

## ADR-002: Contracts are ordinary types

**Context.** A separate `requirement` or `capability` block duplicates Go interfaces and Forst `type` shapes.

**Decision.** A contract is any type the compiler can treat as a method set: Forst shape or nominal type with methods, imported Go interface or concrete type, or thin Forst wrapper around foreign Go symbols. Structural satisfaction for implementations; **distinct nominal names = distinct Usable keys**.

**Consequences.**

- Fakes are ordinary structs + methods (same as Go).
- Go lowering emits standard `interface` types or import aliases.
- Wrapping non-Forst Go code uses native adapter structs without an Effect runtime.

---

## ADR-003: No author-written `usables` clause on signatures

**Context.** Export clauses like `uses Logger, Clock` duplicate information already in `use` sites and transitive call analysis.

**Decision.** Function signatures have **no** author-written `uses` / `usables` clause. Authors never maintain a second list that can drift from the body.

**Consequences.**

- Single source of truth is the function body ([ADR-004](./ADR.md#adr-004-usablesf-inferred-from-use-and-transitive-calls), [ADR-005](./ADR.md#adr-005-context-parameters-never-contribute-to-usablesf)).
- Tooling projects inferred sets ([ADR-006](./ADR.md#adr-006-usablesf-is-derived-tooling-sugar-only)).

---

## ADR-004: `Usables(f)` inferred from `use` and transitive calls only

**Context.** Inference must be predictable and grep-friendly at `use` sites.

**Decision.** The compiler infers **`Usables(f)`** — the set of **Usable** keys a function requires — from **`use` bindings** and **transitive calls** only.

**Consequences.**

- Leaf `use` changes propagate upward through the call graph ([ADR-016](./ADR.md#adr-016-transitive-usables-inference)).
- Context parameters are excluded ([ADR-005](./ADR.md#adr-005-context-parameters-never-contribute-to-usablesf)).

---

## ADR-005: Context parameters never contribute to `Usables(f)`

**Context.** Handlers often take `AppContext` and call `ctx.userRepo.find(...)`. Blurring context fields into inference would make `Usables(f)` depend on context shape design.

**Decision.** Method calls on **context parameter** fields are **never** inferred as Usables. Authors write `use repo: UserRepo` when a service must appear in **`Usables(f)`**.

**Consequences.**

- Parameters carry per-invocation **data** only ([ADR-011](./ADR.md#adr-011-parameters-are-data-use-is-runtime-logic)).
- ADR and SPEC agree on inference sources ([SPEC § Usables identification](./SPEC.md#usables-identification)).

---

## ADR-006: `Usables(f)` is derived tooling sugar only

**Context.** Host authors, agents, and LSP consumers need a readable projection of inferred Usables without a signature export clause.

**Decision.** **`Usables(f)`** appears only in **derived views**: LSP hover, generated docs, and discovery JSON. It is not author-written source syntax.

**Consequences.**

- Go emitted signatures still show explicit **Usables struct** parameters — honesty at the backend boundary ([ADR-012](./ADR.md#adr-012-usables-struct-as-first-parameter)).
- TypeScript never sees Usables ([ADR-020](./ADR.md#adr-020-typescript-emit-excludes-usables-concept)).

---

## ADR-007: Always forward ambient; no `with forward`

**Context.** An opt-in `with forward` modifier forced authors to remember forwarding at every inner call and duplicated merge logic in the spec.

**Decision.** Inside a wired scope, **all inner calls auto-forward** ambient Usables to callees. There is **no** `with forward` modifier and **no** opt-out.

**Consequences.**

- Simpler parser and mental model.
- Inner call sites are less verbose; wiring is visible at boundaries and in Go output.
- Less grep-ability at inner calls — mitigated by [ADR-006](./ADR.md#adr-006-usablesf-is-derived-tooling-sugar-only) and [ADR-034](./ADR.md#adr-034-lsp-ships-at-feature-ga).

---

## ADR-008: Nested `with` overlays outer wiring

**Context.** Tests often need one fake (clock, HTTP client) while reusing a shared Usables bundle for everything else.

**Decision.** **Nested** `with inner { … }` inside an outer `with` **merges** wiring: inner fields **override** outer ambient for the inner scope; unspecified fields **forward** from outer ambient. The checker verifies the merged ambient satisfies **`Usables(g)`** for callees reachable in the inner body.

**Consequences.**

- Integration tests: `with ciUserApiServices() { with { Clock: &fake } { … } }`.
- Wiring roots must supply a **complete** Usables bundle in the outermost `with`.
- Transformer emits full Usables struct literals (merge expanded at compile time).

---

## ADR-009: No postfix `with` on calls

**Context.** Postfix `f(x) with { … }` collides mentally with `ensure` and duplicates nested-block merge semantics.

**Decision.** There is **no** postfix `call() with { … }` form. Wiring overrides use **nested** `with` blocks only ([ADR-008](./ADR.md#adr-008-nested-with-overlays-outer-wiring)).

**Consequences.**

- One wiring surface form with block scoping ([ADR-030](./ADR.md#adr-030-with-takes-usable-shape-literals-only)).
- Table tests use **`t.Run` + nested `with`** ([ADR-033](./ADR.md#adr-033-trun--nested-with-permanent-table-test-path)).

---

## ADR-010: No subtract syntax for Usables

**Context.** Least-privilege scoping (`pick`, subtract) adds environment algebra the design deliberately avoids.

**Decision.** There is **no** syntax to subtract, `pick`, or remove a Usable from ambient scope.

**Consequences.**

- Always-forward ambient is permanent ([ADR-007](./ADR.md#adr-007-always-forward-ambient-no-with-forward)).
- Unsatisfied Usables remain a **hard compile error**.

---

## ADR-011: Parameters are data; `use` is runtime logic

**Context.** Go already mocks data via struct fields. DI frameworks often blur data and behavior.

**Decision.** Function **parameters** carry per-invocation **data** only. **`use`** obtains **runtime behavior** (loggers, clocks, repos, HTTP clients).

**Consequences.**

- Sidecar wire format stays data-only; host builds Usables server-side.
- Clear rule for authors and LLM agents: params = inputs, `use` = services.

---

## ADR-012: Usables struct as first parameter

**Context.** Forst must transpile to idiomatic, debuggable Go without reflection DI.

**Decision.** Each function with non-empty **`Usables(f)`** receives a synthesized **Usables struct** **prepended as first parameter (by value)**. Each contract type lowers to Go `interface` (or imported type).

**Consequences.**

- Generated code matches hand-written constructor injection.
- Field types and naming details: [ADR-013](./ADR.md#adr-013-deduped-usables-struct-naming), [ADR-026](./ADR.md#adr-026-usables-struct-fields-by-value), [ADR-027](./ADR.md#adr-027-first-parameter-named-usables).

---

## ADR-013: Deduped Usables struct naming

**Context.** Per-function struct names (`expireTokenUsables`, `createUserUsables`) explode when many functions share the same inferred set.

**Decision.** Identical inferred sets share **one** emitted struct type (e.g. `Usables_a1b2c3` or a stable descriptive name). There is **no** per-function `{fn}Usables` suffix.

**Consequences.**

- Lowering dedupes in a post-inference pass ([SPEC § Go lowering](./SPEC.md#go-lowering)).
- Agents ignore emit struct names; **`Usables(f)`** is the author-facing view.

---

## ADR-014: No runtime DI container or service locator

**Context.** Framework DI hides wiring and complicates stack traces.

**Decision.** No global registry, reflection injection, or layer graph in the language. Composition roots (`main`, tests, `ciUserApiServices()`) wire **Usables** explicitly.

**Consequences.**

- Effect/ZIO Layer lifecycle is out of scope.
- Startup ordering and `acquire`/`release` stay manual in Go.

---

## ADR-015: Unsatisfied Usables are hard compile errors

**Context.** Optional completeness or warning-only modes would defeat the feature’s value versus hand-written Go.

**Decision.** Every **`use`** must be satisfied by an enclosing **`with`**, incoming synthesized **Usables** struct, or inner **`with`** in the same function body. **No opt-out** gate for application packages.

**Consequences.**

- Wiring roots (`main`, tests, sidecar entry) must supply complete ambient Usables.
- Inner **`with`** can satisfy Usables locally without propagating obligation upward.

---

## ADR-016: Transitive Usables inference

**Context.** The feature’s value is compile-time propagation through the call graph — not keyword syntax alone.

**Decision.** The compiler **always** performs call-graph **`Usables(f)`** propagation, including **cross-package** fixed-point where modules import each other ([ADR-035](./ADR.md#adr-035-cross-package-graph-normative-before-ga)).

**Consequences.**

- Leaf `use` changes error at distant wiring roots with obligation chains.
- Ambient merge/shadow rules apply ([ADR-007](./ADR.md#adr-007-always-forward-ambient-no-with-forward), [ADR-008](./ADR.md#adr-008-nested-with-overlays-outer-wiring)).

---

## ADR-017: No reduced mode without transitive completeness

**Context.** A “syntax only” subset would type-check `use`/`with` without propagation, misleading authors about safety.

**Decision.** There is **no** reduced mode that type-checks `use`/`with` without transitive completeness and wiring-root errors.

**Consequences.**

- `use`/`with` ship as a **single complete feature** ([SPEC](./SPEC.md)).
- Implementation sequencing lives in [06 — Feasibility](./06-feasibility-analysis.md) only.

---

## ADR-018: Orthogonal to `ensure` and `Result`

**Context.** `ensure` is guards, narrowing, and failure propagation — not dependency injection.

**Decision.** Usables do not overload `ensure`, `Result`, or error types. Services via `use`/`with`; validation and errors via existing error RFCs.

**Consequences.**

- No collision with guard/shape-guard tooling.
- Effect RFC error channels remain separate from capability wiring.

---

## ADR-019: Superset wiring allowed

**Context.** A fat CI fixture such as `ciUserApiServices()` intentionally wires **more** Usables than any single callee requires.

**Decision.** An ambient Usables bundle may contain fields **not** in **`Usables(callee)`**. Lowering copies **only** fields required for the callee’s synthesized Usables struct; **extras are skipped** — not an error.

**Consequences.**

- One `ciUserApiServices()` serves many tests and callees with different **`Usables(g)`**.
- Diagnostic severities for extras: [ADR-024](./ADR.md#adr-024-known-but-unused-wiring-keys-warning), [ADR-025](./ADR.md#adr-025-unknown-wiring-keys-hard-error).

---

## ADR-020: TypeScript emit excludes Usables concept

**Context.** The sidecar contract is **data-only on the wire**; the host builds Usables server-side. Exposing Usables to TS would imply TS could supply services — which it cannot.

**Decision.** TypeScript emit does **not** mention `usables`, `@needs`, `R`, capability types, or wiring metadata. TS types describe **wire data** only.

**Consequences.**

- **`with` / `use` stay Go-side**.
- Effect-TS interop on the TS shell does not get an `R` channel from Forst.

---

## ADR-021: Runnable exports only when `Usables(f)` is empty

**Context.** Sidecar clients invoke endpoints with payload data only; host wiring happens server-side.

**Decision.** `forst generate` and sidecar discovery emit TypeScript **only for functions where `Usables(f) = ∅`** after host wiring (“runnable” exports).

**Consequences.**

- Functions with outstanding `use` Usables are Go-internal only in TS artifacts.
- Aligns with [sidecar decisions](../sidecar/10-decisions.md).

---

## ADR-022: Hard error on sidecar export with non-empty `Usables(f)`

**Context.** Silent omission from the TS manifest would hide unrunnable exports from authors.

**Decision.** Declaring a sidecar / TS export on a function with **non-empty `Usables(f)`** is a **compile error**.

**Consequences.**

- Honesty at export site; no silent manifest filtering.
- Host must wrap with a fully-wired entry point before expose.

---

## ADR-023: Go-side discovery JSON for `Usables(f)`

**Context.** Cross-package callers, agents, and host authors need machine-readable inferred Usables without TS exposure.

**Decision.** Inferred **`Usables(f)`** may appear in **Go-side** discovery JSON and LSP — never in the TS client artifact ([ADR-020](./ADR.md#adr-020-typescript-emit-excludes-usables-concept)).

**Consequences.**

- Schema: [SPEC § Discovery JSON](./SPEC.md#discovery-json).
- Invalidation: [ADR-036](./ADR.md#adr-036-single-graph-source-for-checker-json-and-lsp).

---

## ADR-024: Known-but-unused wiring keys → warning

**Context.** Fat fixtures intentionally wire more than any single callee needs; authors should get feedback without blocking CI.

**Decision.** When a wiring key names a **valid** contract ident but no reachable callee in scope requires it, the compiler emits an LSP **warning**, e.g. *"Field Metrics is not required by expireToken; ignored"*.

**Consequences.**

- Superset fixtures remain ergonomic ([ADR-019](./ADR.md#adr-019-superset-wiring-allowed)).
- Distinct from unknown-key errors ([ADR-025](./ADR.md#adr-025-unknown-wiring-keys-hard-error)).

---

## ADR-025: Unknown wiring keys → hard error

**Context.** Typos in fat fixtures (`Metricks`) must not depend on whether a distant leaf `use`s the mistyped contract.

**Decision.** Any key in a `with` wiring literal that is **not** a known contract root ident is a **compile error**.

**Consequences.**

- Typo safety at fixture definition time.
- LSP may offer quick-fixes for near-miss idents ([ADR-034](./ADR.md#adr-034-lsp-ships-at-feature-ga)).

---

## ADR-026: Usables struct fields by value

**Context.** Pointer-to-interface fields in synthesized structs are unidiomatic Go.

**Decision.** **Usables struct fields** use **contract types by value** (interfaces, shapes, imported types such as `*sql.DB`) — **do not** emit `*interface` fields.

**Consequences.**

- Normative examples: [SPEC § Go lowering](./SPEC.md#go-lowering).
- Wiring may still use pointers where assignable ([ADR-028](./ADR.md#adr-028-wiring-value-pointers-optional)).

---

## ADR-027: First parameter named `usables`; `use` binds from fields

**Context.** Generated Go should read like hand-written constructor injection.

**Decision.** The first **Usables struct** parameter is named **`usables`** (or package-scoped equivalent when shadowing). Each `use x: T` lowers to a local from the corresponding field: `logger := usables.Logger`.

**Consequences.**

- Transparent mapping from Forst `use` to Go locals.
- `with wiring { … }` lowers to struct literals filling the callee’s Usables struct ([ADR-012](./ADR.md#adr-012-usables-struct-as-first-parameter)).

---

## ADR-028: Wiring value pointers optional

**Context.** Mandating `&` on every fixture key adds ceremony without safety benefit when interface values and package vars suffice.

**Decision.** **`with` wiring values** may be any expression **assignable** to the contract type. Pointers are **allowed**, **not required**.

**Consequences.**

- Shared mutable fakes via package vars ([ADR-029](./ADR.md#adr-029-mock-reuse-by-convention-not-compiler-enforced)).
- Imported pointer contracts (`*sql.DB`) unchanged.

---

## ADR-029: Mock reuse by convention (not compiler-enforced)

**Context.** The design should enable reusing mocks across tests without mandating a framework or keyword.

**Decision.** There is **no compiler rule** requiring mock reuse or singleton fakes. Documentation recommends package-level shared fakes and `ciUserApiServices()` helpers.

**Consequences.**

- Reuse is ergonomic for Go test authors; not a language guarantee.
- Parallel tests: prefer fresh fixtures when fakes hold mutable state.

---

## ADR-030: `with` takes Usable shape literals only

**Context.** Wiring should look like supplying a **shape** (composite literal), not forwarding an opaque context struct or using a separate map type.

**Decision.** **`with` has one surface form:** `with <wiring> { <body> }`. **`<wiring>`** is a shape literal, variable, or call result assignable to a **Usables** shape — Forst **composite / shape literals** (`ShapeNode`), **not** `map[K]V` values.

**Consequences.**

- Same `{ field: value }` syntax as data shapes ([ADR-031](./ADR.md#adr-031-no-with-ctx-struct-forwarding)).
- Fat fixtures return Usables-shaped literals (`ciUserApiServices(): CIUsables`).

---

## ADR-031: No `with ctx` struct forwarding

**Context.** `with ctx` on handler service structs added implicit field→Usable projection and naming conventions in the transformer.

**Decision.** There is **no** `with ctx` — no forwarding a named handler struct with implicit field→Usable projection. Authors build Usables from struct fields manually in ordinary code if needed.

**Consequences.**

- Explicit wiring at scope boundaries.
- Handlers follow the same rules as any function ([ADR-032](./ADR.md#adr-032-handlers-not-special-for-usewith)).

---

## ADR-032: Handlers not special for `use`/`with`

**Context.** Treating HTTP handlers differently would fork checker rules and documentation.

**Decision.** Handlers use the same **`use` / `with`** rules as any other function.

**Consequences.**

- No handler-specific wiring sugar.
- Services may still be passed as ordinary parameters outside `use`/`with` — unconstrained by this feature.

---

## ADR-033: `t.Run` + nested `with` permanent table-test path

**Context.** Table-driven tests need per-row wiring deltas without postfix override syntax.

**Decision.** Table tests use **`t.Run`** subtests with **nested `with`** for per-case overrides — the **permanent** strategy, not interim sugar pending postfix `with` ([ADR-009](./ADR.md#adr-009-no-postfix-with-on-calls)).

**Consequences.**

- Normative pattern in [SPEC § Table tests](./SPEC.md#table-tests-with-trun).
- Go `_test.go` bypass remains valid for hand-written struct literals ([ADR-038](./ADR.md#adr-038-go-_testgo-escape-hatch)).

---

## ADR-034: LSP ships at feature GA

**Context.** Always-forward ambient hides Usables at inner call sites ([ADR-007](./ADR.md#adr-007-always-forward-ambient-no-with-forward)).

**Decision.** **LSP ships with feature GA** — not fast-follow. Minimum surface: derived **`Usables(f)`** hover, obligation-chain diagnostics, effective ambient Usable at cursor inside nested `with`, and severities per [ADR-024](./ADR.md#adr-024-known-but-unused-wiring-keys-warning) / [ADR-025](./ADR.md#adr-025-unknown-wiring-keys-hard-error).

**Consequences.**

- Teams without LSP investment should **Wait**.
- Normative detail: [SPEC § LSP and tooling](./SPEC.md#lsp-and-tooling).

---

## ADR-035: Cross-package graph normative before GA

**Context.** Single-package inference alone does not deliver the feature’s value versus hand-written Go.

**Decision.** [SPEC § Cross-package inference](./SPEC.md#cross-package-inference) and [SPEC § Discovery JSON](./SPEC.md#discovery-json) must specify fixed-point algorithm, cycle policy, JSON schema, and invalidation rules **before feature GA**.

**Consequences.**

- Multi-package integration tests are a GA gate ([ADR-016](./ADR.md#adr-016-transitive-usables-inference)).
- No “single-package first” reduced ship mode ([ADR-017](./ADR.md#adr-017-no-reduced-mode-without-transitive-completeness)).

---

## ADR-036: Single graph source for checker, JSON, and LSP

**Context.** Divergent graphs between typechecker, discovery JSON, and LSP would produce inconsistent obligation chains.

**Decision.** Typechecker fixed-point, discovery JSON, and LSP obligation chains derive from the **same** call-graph + Usables propagation state.

**Consequences.**

- Edit to a `use` site invalidates dependents consistently.
- One implementation owns graph build + invalidation.

---

## ADR-037: Nil forbidden in wiring

**Context.** Nil wiring values defer failures to runtime and blur optional-service semantics.

**Decision.** **`with` wiring values must not be nil.** Nil at a wiring site is a **compile error**.

**Consequences.**

- Optional behavior uses function **parameters** or noop implementations — not nil Usables ([ADR-039](./ADR.md#adr-039-no-optional-use-syntax)).
- Distinct from absent keys (also errors per [ADR-015](./ADR.md#adr-015-unsatisfied-usables-are-hard-compile-errors)).

---

## ADR-038: Go `_test.go` escape hatch

**Context.** Brownfield teams may already have hand-written Go table tests with struct literals.

**Decision.** Go `_test.go` may call exported Forst-generated helpers with hand-written Usables struct literals. This is **not** the primary pattern.

**Consequences.**

- Incremental adoption path without rewriting every test to Forst syntax.
- No automatic bridge for production `Deps` structs ([ADR-040](./ADR.md#adr-040-no-brownfield-bridge-syntax-v1)).

---

## ADR-039: No optional `use` syntax

**Context.** `use x?: T` would require optional-effect semantics the design defers.

**Decision.** There is **no** `use x?: T` or optional-Usable syntax. Every `use` is mandatory in **`Usables(f)`**.

**Consequences.**

- Optional behavior via parameters ([ADR-011](./ADR.md#adr-011-parameters-are-data-use-is-runtime-logic)) or noop impls in fat fixtures.
- Nil is forbidden in wiring ([ADR-037](./ADR.md#adr-037-nil-forbidden-in-wiring)).

---

## ADR-040: No brownfield bridge syntax (v1)

**Context.** Legacy Go services use hand-written `type ServerDeps struct { … }` passed into handlers.

**Decision.** There is **no** v1 bridge syntax (`forst:wire`, struct tags) from legacy Go `Deps` to Forst Usables. Brownfield teams rewrite to `use`/`with` or use [ADR-038](./ADR.md#adr-038-go-_testgo-escape-hatch).

**Consequences.**

- A **conversion guide** documents hand-written `Deps` → `CIUsables` mapping.
- Full production migration requires body re-authoring with `use`.

---

## ADR-041: Root contract ident only in `with` keys

**Context.** Type aliases create two names for one slot (`type AuditLogger = Logger`). Allowing alias keys in wiring duplicates keys and weakens typo detection.

**Decision.** **`with` wiring keys must use the root contract ident only.** `with { Logger: … }` is legal; `with { AuditLogger: … }` is a **compile error** when `type AuditLogger = Logger`.

**Consequences.**

- One canonical string per Usable slot in wiring literals.
- Type aliases still share one slot in **`Usables(f)`** ([ADR-042](./ADR.md#adr-042-usable-and-usables-vocabulary)).

---

## ADR-042: Usable and Usables vocabulary

**Context.** Prior drafts mixed “needs”, “requirements”, and “maps” for the same phenomenon.

**Decision.**

- **Usable** — one contract slot (root ident + contract type).
- **`Usables(f)`** — inferred set for function `f` (`Usables(A | B | C)`).
- **Usables bundle** — author-named shape typedef (e.g. `CIUsables`) or inline wiring literal.

**Consequences.**

- Pairs with **`use`** keyword; no parallel “needs” author vocabulary.
- Go emit uses deduped **Usables struct** types ([ADR-013](./ADR.md#adr-013-deduped-usables-struct-naming)).

---

## ADR-043: Satisfaction relation for wiring

**Context.** Checker, LSP, and docs need one normative test for whether ambient wiring satisfies a call.

**Decision.** At a wiring root, ambient **`U`** satisfies call to **`f`** iff **`Usables(f) ⊆ keys(U)`** with assignable field types.

**Consequences.**

- Unifies completeness errors, superset copy, and fixture typing.
- Normative in SPEC rule N10.

---

## ADR-044: Test entrypoints use `Test*` and `*testing.T`

**Context.** Bare `func testFoo()` helpers hide `t.Run`, `t.Helper`, and `go test` discovery.

**Decision.** Test entrypoints are functions named `Test[A-Z]…` with exactly one first parameter of type `*testing.T`. Tests live in `*_test.ft`. **`BenchmarkXxx`** reserved for a future milestone.

**Consequences.**

- Same discovery rule as `go test`.
- CLI spec: [forst-test RFC](../forst-test/README.md).

---

## ADR-045: Test harness excluded from `Usables(f)`

**Context.** `*testing.T` is a test harness, not a host-provided service.

**Decision.** `*testing.T` (and future `*testing.B`) are **never** part of **`Usables(f)`** and never appear in synthesized **Usables** structs.

**Consequences.**

- Test functions wire Usables like any other code inside the test body.
- Compiler: harness param exclusion in inference.

---

## ADR-046: `ensure` in tests lowers to `t.Fatal` / `t.Error`

**Context.** Test failures should use standard Go `testing` reporting.

**Decision.** In test functions, `ensure` failures lower to **`t.Helper()` + `t.Fatalf(...)`** (or `t.Errorf` when an `ensure` block is present).

**Consequences.**

- Parser/transformer treat test functions like today's `IsMainFunction()` special case for `ensure` ([ADR-018](./ADR.md#adr-018-orthogonal-to-ensure-and-result)).
- Refinement-only `ensure` in tests ([SPEC § Testing](./SPEC.md#testing-go-native)).
