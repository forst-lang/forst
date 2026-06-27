# Needs map typing — Usable shape constraint (leading option)

**Status:** **Leading option locked** as [ADR-017 — Usable shape constraint](./ADR.md#adr-017-usable-shape-constraint). Name analysis below; union/generic alternatives remain in this doc for comparison.

**Term:** **Usable** — a **shape type** describing what may be **`use`d** in a scope; **shape literals** `{ Logger: impl, … }` in **`with`** must satisfy an Usable (explicit typedef or inferred minimal shape from completeness).

**Not Go maps:** Wiring uses Forst **composite literals** (shapes / `ShapeNode` in the parser) — the same `{ field: value }` form as data shapes — **not** `map[K]V` unless you explicitly choose a map type elsewhere.

**Related:** [SPEC](./SPEC.md), [ADR](./ADR.md), [09](./09-design-solutions.md), [08](./08-design-analysis.md).

---

## Leading option (summary)

**Syntax stays shape literals** ([ADR-015](./ADR.md#adr-015-with-takes-usable-shape-literals-only)). **Typing** is an ordinary **shape** whose fields are requirement contracts — we call that shape an **Usable**.

```forst
type CIUsables = {
    Logger:      Logger,
    UserRepo:    UserRepo,
    HttpClient:  HttpClient,
    EmailSender: EmailSender,
    Metrics:     Metrics,
}

func ciUserApiServices(): CIUsables {
    return {
        Logger:      NopLogger {},
        UserRepo:    InMemoryUserRepo { users: map[String]User{} },
        HttpClient:  BlockedHttp {},
        EmailSender: NoopEmail {},
        Metrics:     NopMetrics {},
    }
}

with ciUserApiServices() {
    handleCreateUser(req)
}

with ciUserApiServices() {
    with { Clock: FakeClock { fixedMs: 2000 } } {
        expireToken(token)
    }
}
```

### Shape literals vs `with ctx` (important)

| Allowed | Not allowed |
| --- | --- |
| `with { Logger: NopLogger {}, UserRepo: db } { … }` — **anonymous shape literal** satisfying `CIUsables` | `with ctx { … }` — **named struct param** with implicit field→requirement projection |
| `with ciUserApiServices() { … }` — variable holding same literal shape | Passing `AppServices` struct as a function param for services ([ADR-006](./ADR.md#adr-006-parameters-are-data-requirements-are-runtime-logic)) |
| Nested `with { Clock: fake } { … }` — partial shape overlay | Postfix `f() with { … }` |

At compile time, `{ Logger: x, … }` is a **shape literal** in Forst (same AST path as other composite literals). Lowering to Go produces a **struct literal** for `{fn}Needs` — not a Go `map[string]any`.

### Checker rules

| Rule | Behavior |
| --- | --- |
| **Fixture defs** | Return type / annotation is an **Usable** shape → wrong field name (`Metricks`) → **error** |
| **Completeness** | Inferred `Needs(f)` for the block must be covered by merged ambient ([ADR-009](./ADR.md#adr-009-transitive-inference-and-mandatory-completeness)) |
| **Superset** | Extra Usable fields OK; lowering copies only `Needs(callee)`; unused keys → **warning** ([ADR-012](./ADR.md#adr-012-superset-wiring-extras-allowed)) |
| **Nested `with`** | Partial shape overlays outer; inner fields shadow |
| **Field names** | **Root** contract idents only (`Logger`, not alias `AuditLogger` when `type AuditLogger = Logger`) |
| **Values** | Assignable to contract type; **nil forbidden** ([ADR-016](./ADR.md#adr-016-post-critique-decisions-09--locked)) |
| **Go emit** | Usable shape ↔ `{fn}Needs` struct fields (same names); dedupe identical sets ([ADR-016](./ADR.md#adr-016-post-critique-decisions-09--locked)) |

**No new keyword required** — `Usable` is the **concept name** for “shape used as wiring constraint.” Typedefs use plain shape syntax `{ … }`. Optional future marker `Usable { … }` in typedef bodies is sugar only.

---

## Naming analysis — what to call the wiring shape?

We need a short English name for: *“shape of requirement contracts that a `with` wiring expression must satisfy.”* It should pair with **`use`** / **`with`**, not collide with inferred **`Needs(f)`**, and stay informal enough for fixtures (`ciUserApiServices()`).

### Candidates

| Name | Example | Pros | Cons | Score |
| --- | --- | --- | --- | ---: |
| **Usable** / **Usables** | `type CIUsables = { Logger: Logger, … }` | Pairs with **`use`** (“what you can use”); short; not a third keyword; plural **Usables** names bundles naturally | Adjective-as-noun is slightly informal; must document vs English “usable API” | **9** |
| **Wiring** | `type CIWiring = { … }` | Describes `with` accurately | Doesn’t pair with **`use`**; verb-ish process, not the map | 7 |
| **Supply** / **Supplies** | `type CISupplies = { … }` | Matches “supply implementations” | Old RFC **`supply`** keyword rejected; easy confusion | 5 |
| **Needs** / **NeedsMap** | `type CINeeds = { … }` | Obvious | Collides with inferred **`Needs(f)`** and tooling JSON field `needs` | 4 |
| **Deps** | `type CIDeps = { … }` | Go developers recognize it | Collides with “data params vs deps”; implies struct param pattern | 6 |
| **Services** | `type CIServices = { … }` | Handler language | **`AppServices`** brownfield confusion; sounds like microservices | 5 |
| **Runtime** | `type CIRuntime = { … }` | Matches [ADR-006](./ADR.md#adr-006-parameters-are-data-requirements-are-runtime-logic) “runtime logic” | Abstract; doesn’t read as a map | 6 |
| **Kit** / **Fixture** | `type CIKit = { … }` | Test-helper tone | Sounds test-only; weak for `main` / production wiring | 6 |
| **Bundle** | `type CIBundle = { … }` | Fat-fixture metaphor | Generic; sidecar “bundle” overload | 6 |
| **Host** | `type CIHost = { … }` | Sidecar host owns it | Narrow; TS/host-only connotation | 5 |
| **Equipped** | `type Equipped = { … }` | “Equipped with Logger…” | Awkward typedef names (`CIEquipped`); past participle | 5 |
| **Context** | `type CIContext = { … }` | Familiar from other langs | **`with ctx`** struct pattern explicitly dropped | 3 |
| **Capabilities** | `type CICapabilities = { … }` | Effect-ish | **`capability`** declaration family rejected | 4 |

### Recommendation: **Usable** (concept), **Usables** (concrete typedef)

- **Concept (docs, ADR, LSP):** “This map must satisfy **Usable** *X*.”
- **Author typedefs:** name the bundle **`…Usables`** — `CIUsables`, `ProdUsables`, `TestUsables`.
- **Not a keyword:** an Usable is any shape that meets the rules in [ADR-017](./ADR.md#adr-017-usable-shape-constraint); no mandatory `Usable` token in source for now.
- **Optional later sugar:** `type CIUsables = Usable { Logger: Logger, … }` — marker for readability only; desugars to shape + Usable kind flag in checker.

**Avoid:** **`Needs`** as the shape name (inference wins that word). **`Supply`** (dead keyword). **`Context`** (dropped pattern).

### Plain-language glossary

| Term | Meaning |
| --- | --- |
| **Contract** | Type a function can `use` — `Logger`, `UserRepo`, … |
| **Usable** | Shape listing contracts + field types; constraint on a wiring map |
| **Usables** | A named Usable typedef (`CIUsables`) or a **shape literal** `{ Logger: impl, … }` satisfying one |
| **`Needs(f)`** | Inferred set of contracts function `f` requires — derived, not the typedef name |

---

## Open details (implementation, not naming)

Locked elsewhere:

- Map values not required to be pointers ([ADR-016](./ADR.md#adr-016-post-critique-decisions-09--locked)).
- **`type AuditLogger = Logger`** shares slot `Logger`; field names use **root** ident.
- No `forst:wire` bridge ([ADR-016](./ADR.md#adr-016-post-critique-decisions-09--locked)).

Still to specify in compiler:

- Exact assignability rules (extras, partial nested maps, unknown keys).
- Whether dev-only trace comments ([Q4](#cross-cutting-ambient-traceability-q4)) ship with Usable spike.

---

## Existing Forst unions and intersections (reuse baseline)

Forst already has **binary type expressions** on typedef bodies — the same **`TypeNode` algebra** used for errors, optionals, and control-flow join. Requirements map typing should **extend** this, not introduce a unrelated `NeedsMap` keyword without weighing unions/shapes first.

### Spec (RFC / roadmap)

| Source | What it says |
| --- | --- |
| [ROADMAP.md](../../../../ROADMAP.md) | **Binary type expressions** (`T \| U`, `T & U`) — parser/AST; full meet/join + Go emit still listed as incomplete in roadmap text |
| [optionals/01 — unions & Go interop](../optionals/01-single-return-unions-and-go-interop.md) | One primitive: `\|` = union, `&` = intersection; `T?` ≈ `T \| Nil`; shares narrowing with `ensure` / `is` |
| [optionals/02 — Result & error unions](../optionals/02-result-and-error-types.md) | Finite **unions of nominal error types** assignable to `Error` |
| [union_error_types.ft](../../union_error_types.ft) | `type ErrKind = ParseError \| IoError` — golden example |

### Implemented today (compiler audit)

| Layer | Status | Location |
| --- | --- | --- |
| **Parse** | `A \| B`, `A & B` in typedef bodies | `forst/internal/parser/typedef.go`, `ast.TypeDefBinaryExpr` |
| **AST** | `TypeUnion`, `TypeIntersection`; flatten + dedupe | `forst/internal/ast/type.go` (`NewUnionType`, …) |
| **Typecheck** | Lower typedef body → `TypeNode`; **join** / **meet**; assignability with union on RHS | `typedef_binary.go`, `typeops.go`, `go_builtins.go` |
| **Flow** | `JoinAfterIfMerge` at if-merge (union of branch refinements) | `typeops.go`, `flow_fact_test.go` |
| **Go emit — errors** | Closed union of **nominal errors** → **sealed interface** + marker methods (or `error` when uniformly error-kinded) | `error_union_sealed.go`, `examples/out/union_error_types.go` |
| **Go emit — other unions** | Non-error unions → `any` (placeholder) | `transformer/go/type.go` |
| **TS emit** | `A \| B` → `A \| B`; `A & B` → `A & B` | `transformer/ts/type_mapping.go` |
| **Examples** | `task example:union-error-types`, narrowing variant | `union_error_types.ft`, `union_error_narrowing.ft` |

**Not implemented for requirements:** map key validation against a typedef union, wiring-map types, or sealed “requirement kind” interfaces for contracts (only the **error** sealed-union path exists).

**Note:** [ROADMAP.md](../../../../ROADMAP.md) understates checker progress (join/meet and error unions **do** run in tests); treat roadmap as directional, this table as **current code**.

### Why this matters for needs maps

| Needs-map problem | Existing Forst mechanism |
| --- | --- |
| **Closed fixture key set** (catch `Metricks`) | Typedef **union of contract nominals**: `type CIKeys = Logger \| UserRepo \| …` — checker: key ∈ union members |
| **Product of slots** (all keys required together) | **Usable** shape — leading option ([ADR-017](./ADR.md#adr-017-usable-shape-constraint)) |
| **Value assignability** (fake satisfies contract) | Existing **structural** `IsTypeCompatible` (already used for contracts) |
| **Branch-dependent needs** | **`JoinTypes`** on `Needs(f)` across paths (same as if-merge union story) |
| **Closed enum of implementations** (future) | Same **sealed union** pattern as nominal errors — only if we need runtime tagging; likely **overkill** for wiring maps |

---

## Questions to answer

| ID | Question |
| --- | --- |
| **Q1** | What is the static type of `{ Logger: x, UserRepo: y }` in `with`? |
| **Q2** | How do fat fixtures (`ciUserApiServices()`) catch typos like `Metricks`? |
| **Q3** | What Go type(s) does lowering emit for wiring and for `{fn}Needs` fields? |
| **Q4** | How do humans debug “which ambient supplied this call?” without new keywords? |
| **Q5** | How do `type Alias = Root` and `with` keys interact? |
| **Q6** | How does brownfield Go (`AppServices struct`) connect — without a dedicated bridge keyword? |

---

## Alternatives (not leading)

### Option A — Open map (untyped baseline)

**Surface:** Map literal with requirement idents as keys; values any expression assignable to contract type.

```forst
with {
    Logger:   NopLogger {},
    UserRepo: pg,
} { ... }
```

| | |
| --- | --- |
| **Q1** | Built-in open map type keyed by ident (checker validates keys against known contracts). |
| **Q2** | Unknown key → **error** (locked). Unused key in literal → warning. Typos in *values* caught by assignability. |
| **Q3** | Emit TBD: likely `map[string]any` internally or per-call struct literal; `{fn}Needs` fields = contract types (interfaces / imported types). |
| **Q4** | LSP breadcrumbs + optional dev emit comments (no source syntax). |
| **Q5** | Key = **root** contract ident; alias `AuditLogger = Logger` → key `Logger`. |
| **Q6** | User writes ordinary conversion to map literal once; no compiler magic. |

**Status:** Fallback if Usable assignability is too strict for nested overlays; strict unknown-key errors still apply.

---

## Option B — `NeedsMap<…>` generic (not recommended)

**Surface:** Hypothetical generic listing allowed keys ( **not** the same as typedef `|` unions today).

```forst
type CI = NeedsMap<Logger, UserRepo, HttpClient, Metrics>
```

| | |
| --- | --- |
| **Q1** | New generic/native type family — **parallel** to `Logger \| UserRepo \| …` unless generics RFC lands |
| **Q2** | Closed at fixture definition; best typo safety **if** we accept new surface syntax |
| **Q3** | Lower to dedicated struct per key set or canonical hash struct ([ADR-016](./ADR.md#adr-016-post-critique-decisions) impl detail) |
| **Q4** | Same as A — tooling |
| **Q5** | Keys are **root** idents only; aliases collapse to root in the key set |
| **Q6** | Conversion function returns `NeedsMap<…>`; still user-written unless codegen added later |

**Pros:** Fixture typos caught early; self-documenting CI helpers.  
**Cons:** **Duplicates** finite unions (`A \| B \| …`) already in AST; arity/generics cost; critics rejected “special syntax” without proof shapes/unions fail.

**Compare first:** [Option F — typedef union](#option-f--typedef-union-of-contract-nominals-closed-keys) before committing here.

---

## Option C — Usable shape constraint (**leading — [ADR-017](./ADR.md#adr-017-usable-shape-constraint)**)

**Surface:** Wiring is a **shape literal** (or variable/call) that must **satisfy** an **Usable** — a shape whose fields are requirement contracts.

```forst
type CIUsables = {
    Logger:     Logger,
    UserRepo:   UserRepo,
    HttpClient: HttpClient,
}

func ciUserApiServices(): CIUsables {
    return {
        Logger:     NopLogger {},
        UserRepo:   InMemoryUserRepo { users: map[String]User{} },
        HttpClient: BlockedHttp {},
    }
}

with ciUserApiServices() { handleGetUser("42") }
```

No `with CIUsables { … }` passing the type as value — **ADR-015**: `with` takes a **wiring expression** (`ciUserApiServices()` or `{ … }` shape literal), not struct forwarding.

| | |
| --- | --- |
| **Q1** | Static type = Usable shape; shape literal checked for field names + value assignability |
| **Q2** | Wrong field → shape error; catches `Metricks` |
| **Q3** | Usable → Go struct with same field names; aligns with `{fn}Needs` emit |
| **Q4** | LSP / optional trace comments |
| **Q5** | Fields = **root** contract idents |
| **Q6** | Hand-written conversion returning `{ … }` typed as Usable |

**Pros:** Reuses shapes; idiomatic Go emit; pairs with **`use`**; no new keyword.  
**Cons:** Must define partial-map rules for nested `with` (merged ambient, not full CIUsables every time).

---

## Option D — Union / intersection types (extend existing binary types)

**Idea:** Use **`type … = A | B | …`** and **`A & B`** already in the compiler — **no new `NeedsMap` keyword** — for (some of) Q1–Q2–Q5.

This replaces the old “tagged union of implementations” sketch with options grounded in **shipped AST + checker** and the **sealed error-union** Go emit path.

### D1 — Union of contract **nominals** = closed **key** set (Option F below)

Typedef union whose members are **requirement contract types** (shapes / interfaces with methods):

```forst
type AppContracts = Logger | UserRepo | HttpClient | EmailSender | Metrics

func ciUserApiServices(): NeedsMapFor<AppContracts> {
    return {
        Logger:  NopLogger {},
        Metrics: NopMetrics {},
    }
}
```

(`NeedsMapFor<…>` = checker name for “map literal constrained by union members” — may be **pure constraint**, no new runtime type.)

| | |
| --- | --- |
| **Q1** | Map literal type = “record keyed by members of `AppContracts`” (product indexed by union members). |
| **Q2** | Key must be **ident of a union member** → `Metricks` fails at compile time. |
| **Q5** | Union members use **root** idents; `type AuditLogger = Logger` collapses into slot `Logger`, not a separate union member unless distinct nominal def. |

**Checker:** Expand `AppContracts` via `TypeDefExprToTypeNode` / `flattenUnionTypeParams`; validate map keys ⊆ member idents. Reuse error-union flatten helpers.

**Go emit:** Static check only — **no** sealed interface required for keys (contrast: error unions need runtime seal). Wiring still lowers to struct literals / `{fn}Needs`.

### D2 — Union on **values** (when several fakes satisfy one contract)

When one slot accepts multiple concrete implementations:

```forst
// conceptual: field type is ClientDoer; value may be any structurally compatible type
with { HttpClient: clientDoerOrBlocked() } { ... }
```

Assignability already flows through **`IsTypeCompatible`**; explicit **`ClientDoer | BlockedHttp`** as a **value** type only helps if we add **expression** unions (not in typedef-only unions today). **Lean:** rely on structural satisfaction; defer value-unions unless a real ambiguity appears.

### D3 — Intersection `A & B` (meet) — usually **not** the map story

`Logger & UserRepo` as **meet** of unrelated nominals is typically **empty** (`meetTypeDefPair` fails). Intersections matter for:

- **Overlapping** contracts (shared methods) — rare for distinct capabilities.
- **Future** bounds on generics ([generics RFC](../generics/00-user-generics-and-type-parameters.md)).

**Prefer shapes** for “must provide Logger **and** UserRepo” (product), not `&` of nominals.

### D4 — Sealed union **emit** pattern (borrow from errors)

Nominal error unions emit:

```go
// closed union of nominal errors — pattern from union_error_types.go
type ErrKind interface { isErrKind() }
```

**Possible reuse:** if we ever need a **runtime** discriminant for requirement kinds in Go (unlikely for wiring maps). For **typing only**, D1 does not need sealed emit.

| | |
| --- | --- |
| **Pros** | Reuses parser, AST, join/meet, tests; aligns with optionals roadmap; TS `\|` already works |
| **Cons** | “Map keyed by union members” is **new checker rules**, not copy-paste errors; non-error union Go emit still `any` until generalized |
| **vs Option B** | Same typo safety **without** new `NeedsMap<>` generic if D1 suffices |

---

## Option F — Typedef union of contract nominals (closed keys)

**Recommended union-first alternative to Option B.**

```forst
type CIContracts = Logger | UserRepo | HttpClient | EmailSender | Metrics

func ciUserApiServices() {
    return {
        Logger:      NopLogger {},
        UserRepo:    InMemoryUserRepo { users: map[String]User{} },
        HttpClient:  BlockedHttp {},
        EmailSender: NoopEmail {},
        Metrics:     NopMetrics {},
    } satisfies MapFor<CIContracts>   // `satisfies` spelling TBD — may be implicit from return type
}
```

| | |
| --- | --- |
| **Q1–Q2** | Closed keys via **existing** `\|` typedef; typos = unknown key |
| **Q3** | Same as Option A/C emit (struct literal → `{fn}Needs`) |
| **Q5** | Union lists **root** contract idents only ([ADR-016](./ADR.md#adr-016-post-critique-decisions)) |

**Implementation sketch:**

1. Mark typedef unions used as key sets (attribute or convention `*Contracts` suffix).
2. `flattenUnionTypeParams` → allowed key idents.
3. Map literal checker: keys ⊆ allowed; values assignable to contract type of member `K`.
4. No new parser tokens — only typedef `\|` already parsed.

**Open:** overlay literals in nested `with` — validate keys against **callee `Needs` ∪ known contracts**, not full `CIContracts`.

---

## Option E — `with ctx` on opaque service struct (rejected)

**Surface:** `with appServices { … }` where `appServices` is an existing handler struct (`AppServices`) with implicit field→requirement projection — **no** explicit Usable shape literal.

**Conflicts with [ADR-015](./ADR.md#adr-015-with-takes-usable-shape-literals-only)** — wiring must be an expression assignable to an **Usable** (shape literal, typed fixture return, etc.), not struct forwarding sugar.

Not recommended — listed for completeness. Brownfield connection stays a hand-written conversion to `{ Logger: appServices.Logger, … }` or a fixture function ([Q6](./10-needs-map-typing-options.md#questions-to-answer)).

---

## Cross-cutting: emit lowering (Q3)

Independent of Q1 surface, generated Go likely needs:

| Piece | Candidates |
| --- | --- |
| **`{fn}Needs` fields** | Interface / contract type per field (preferred over `*interface`) |
| **Call-site wiring** | Struct literal copying from ambient; or map lookup |
| **Dedup** | Identical need sets → one emitted struct name ([ADR-016](./ADR.md#adr-016-post-critique-decisions) — implementation detail) |
| **Nil** | **Disallowed** at wiring sites — compile error if nil assignable value ([ADR-016](./ADR.md#adr-016-post-critique-decisions)) |

**Pointer values** allowed when assignable (`*sql.DB`, shared fakes); **never required** for all keys.

---

## Cross-cutting: ambient traceability (Q4)

No new keywords. Candidates (can combine):

| Mechanism | Owner |
| --- | --- |
| LSP: “ambient from line N, keys {…}” | Compiler / LSP |
| `forst build -trace-needs` comments in generated Go | Transformer flag |
| Discovery JSON: per-function `Needs(f)` | Already planned |
| Source `with` nesting | **By design** — explicit scopes ([ADR-004](./ADR.md#adr-004-always-forward-ambient-no-with-forward)) |

**Not in scope:** subtract / `pick` / least-privilege scopes.

---

## Cross-cutting: alias keys in `with` (Q5)

**Locked:** `type AuditLogger = Logger` → **one slot** `Logger`.

**TBD (lean):**

| Rule | Effect |
| --- | --- |
| **Keys must be root ident** | `with { Logger: x }` ✓ — `with { AuditLogger: x }` ✗ when alias |
| **Keys may be any alias of root** | Normalized to root at check time |

Recommendation to decide in pilot: **root ident only** in map literals (simplest grep, clearest fixtures).

---

## Cross-cutting: brownfield (Q6)

**Locked:** No `forst:wire` tags, no mandatory bridge ADR.

Allowed without new syntax:

- Hand-written function returning **shape literal** typed as **Usable** / `…Usables` typedef.
- Future: optional **codegen** from Go struct → map (tooling, not language).

---

## Comparison matrix

| Criterion | **Usable (C)** | A open map | B NeedsMap | D/F union |
| --- | --- | --- | --- | --- |
| Pairs with `use` / `with` | **Yes** | Partial | No | Partial |
| Fixture typo safety | **Strong** | Weak | Strong | Strong |
| New syntax | **None** (shape only) | Low | High | Low |
| Reuses Forst machinery | **Shapes** | Minimal | New | `\|` typedef |
| ADR-015 shape literal only | **Yes** | Yes | Yes | Yes |
| Go emit clarity | **High** | Medium | High | Medium |

---

## Recommended next step

1. Implement **Usable** rules in checker: shape field = contract; shape literal assignability; root field names; nil forbid.
2. Spike **`ciUserApiServices(): CIUsables`** in `examples/in/rfc/requirements/` + golden Go.
3. Nested **`with`** merge + completeness against inferred shape (not full CIUsables on every inner map).
4. Wire **Usable → `{fn}Needs`** lowering; dedupe identical shapes.
5. Union typedef (F) only if unknown-key errors without Usable typedef prove insufficient.

See [ADR-017](./ADR.md#adr-017-usable-shape-constraint).

---

## References

- [ADR-017 — Usable shape constraint](./ADR.md#adr-017-usable-shape-constraint)
- [09 — Design solutions](./09-design-solutions.md)
- [08 — Design analysis](./08-design-analysis.md)
- [SPEC — Function requirements](./SPEC.md)
- [optionals/01 — Unions & binary types](../optionals/01-single-return-unions-and-go-interop.md)
- [optionals/02 — Error unions](../optionals/02-result-and-error-types.md)
- [union_error_types.ft](../../union_error_types.ft) — `\|` typedef example + golden Go
- [ROADMAP.md](../../../../ROADMAP.md) — binary types (roadmap text; see audit table above)
- Compiler: `forst/internal/typechecker/typedef_binary.go`, `forst/internal/transformer/go/error_union_sealed.go`
