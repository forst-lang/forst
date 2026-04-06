# User generics and type parameters

## Table of contents

1. [Introduction](#introduction)
2. [Purpose and benefits for the language](#purpose-and-benefits-for-the-language)
3. [Relationship to today’s Forst](#relationship-to-todays-forst)
4. [Alignment with project principles](#alignment-with-project-principles)
5. [Target capabilities (staged)](#target-capabilities-staged)
6. [Design decisions to lock early](#design-decisions-to-lock-early)
7. [Implementation phases](#implementation-phases)
8. [Risks and mitigations](#risks-and-mitigations)
9. [Testing strategy](#testing-strategy)
10. [Result types, generics, and narrowing (Ok and Err)](#result-types-generics-and-narrowing-ok-and-err)
11. [References](#references)

---

## Introduction

Forst is a statically typed language that transpiles to Go and targets **TypeScript-grade ergonomics** on the backend. Today, programmers can use **parametric built-in type constructors**—notably slices (`[]T`), pointers (`*T`), and maps (`map[K]V`)—and the compiler models those with `TypeParams` on internal [`TypeNode`](../../../../forst/internal/ast/type.go) representations. That is **not** the same as **user-defined** polymorphism: you cannot yet write a reusable `Box[T]` or `func identity[T](x T) T` that the typechecker understands end-to-end and that lowers cleanly to idiomatic Go.

This RFC describes a **roadmap** toward **comprehensive generic support** in that user-facing sense: declared type parameters on types and functions, instantiation, explicit constraints, codegen strategy (likely Go 1.18+ generics), and **incremental** interoperability with generic APIs in imported Go code. It consolidates technical direction previously captured in internal planning; it does **not** prescribe final syntax (that remains a design decision in a later revision of this document or a short ADR).

The audience is **language designers and compiler contributors** deciding what to build next, and **advanced users** who need to see how Forst intends to relate to Go’s generic ecosystem without promising a single big-bang release.

---

## Purpose and benefits for the language

### Why this work matters

1. **Abstraction without losing types**  
   Backend code repeats the same patterns for “a value of type T”, “a container of T”, or “transform A to B” where `T`, `A`, and `B` should be arbitrary **named** types. Without generics, authors either duplicate concrete definitions or fall back to **weaker** representations (`Shape`, `interface{}`, or stringly-typed APIs). User generics let those patterns stay **fully typed** and **checkable** at compile time.

2. **Closer parity with TypeScript and modern Go**  
   TypeScript users expect generic functions and generic types for collections, utilities, and API boundaries. Go 1.18+ codebases increasingly expose **generic** stdlib APIs (`slices`, `maps`, `cmp`, …). Forst that only understands **non-generic** Go signatures will **lag** those libraries unless the language can grow generic **definitions** and, over time, **interop** for instantiated generic calls. This RFC sets expectations: Forst-first generics are the **main** goal; mapping every external generic API is an **incremental** follow-on (see [Risks and mitigations](#2-go-interop-long-tail--accepted)).

3. **Safer refactors and clearer APIs**  
   Instantiated types like `Result[Int, AppError]` (syntax illustrative) document intent better than parallel `ResultInt`, `ResultString`, … Generics reduce drift between copies and make **incompatibility** explicit (`Box[Int]` is not `Box[String]`).

4. **Foundation for future features**  
   Constraints (`comparable`, ordered types, eventual interface-like bounds) connect to **binary type expressions**, **control-flow narrowing**, and **interop** work already listed on the [roadmap](../../../../ROADMAP.md). A coherent generic **core** avoids bolting on ad hoc special cases later.

5. **`Result` and discriminated narrowing**  
   User generics give **`Result(Success, Failure)`** a stable **instantiated** form (e.g. **`Result[Int, ParseError]`** in examples above)—the same **substitution** story as **`Box[Int]`**. That unlocks **`if r is Ok()`** / **`ensure r is Ok() or err`** with **narrowed** success and failure **payloads** when **`Ok()`** / **`Err(...)`** are **native type guards** on **`Result`** (see [§10](#result-types-generics-and-narrowing-ok-and-err) and the [optionals / `Result` RFC](../optionals/02-result-and-error-types.md)).

6. **Honest ergonomics vs hand-written Go**  
   Emitting **real Go generics** where appropriate keeps generated code readable and **debuggable** in mixed Go/Forst modules—aligned with Forst’s Go-backwards-compatibility story.

### What success does *not* require

- **Full** `go/types` coverage for every generic symbol in the universe on day one.
- **Maximum** type inference when it conflicts with clarity (see [PHILOSOPHY.md](../../../../PHILOSOPHY.md)).
- **Dependent types** or type-level computation beyond what this phased plan describes.

---

## Relationship to today’s Forst

| Topic | Today | After this RFC (phased) |
| ----- | ----- | ------------------------ |
| [ROADMAP “Generic types”](../../../../ROADMAP.md) | Planned / not user-defined | Progress toward experimental → done with scoped notes |
| Built-in `Array` / `Map` / `Pointer` `TypeParams` | Used for `[]T`, `map[K]V`, `*T` | Unchanged; **additional** machinery for **user** type parameters |
| [examples/in/generics.ft](../../../../examples/in/generics.ft) | Documents **builtin** parametric types | May gain a sibling example for **user** `Box[T]`-style code when implemented |
| [go_interop.go](../../../../forst/internal/typechecker/go_interop.go) | Maps basics, slices, pointers; many generic Go returns **unsupported** | **Incremental** extension for instantiated generics; long tail accepted |
| Type aliases, binary types | Experimental / partial | [Roadmap](../../../../ROADMAP.md) notes interactions with generics—**integration passes** after each stabilizes |

---

## Alignment with project principles

- **[PHILOSOPHY.md](../../../../PHILOSOPHY.md)** — Generic **constraints** should be **explicit**; type-arg **inference** only in well-defined cases where parameters are **uniquely determined**; avoid type-system side effects and unbounded metaprogramming.
- **Go interoperability** — Prefer **honest** errors over silent widening; thin Go wrappers remain valid when interop is incomplete.
- **Testing culture** — Each phase adds **reproducing** unit and integration tests ([project priorities](../../../../.cursor/rules/priorities.mdc)); examples under `examples/in/` when stable.

---

## Target capabilities (staged)

| Stage | Capability | Notes |
| ----- | ---------- | ----- |
| A | **User generic type definitions** | e.g. nominal `type Box[T] = …` / struct-like shapes with type parameters; substitution in fields and type references. |
| B | **Generic functions** | Type parameters on top-level `func`; instantiation by explicit or inferred type args at call sites. |
| C | **Constraints** | Explicit bounds (`comparable`, ordered subset for `min`/`max`, future interface-like bounds); ties to roadmap “binary types” / interfaces later. |
| D | **Interop** | `make`/`new` with Forst type syntax ([ROADMAP](../../../../ROADMAP.md)); **incremental** mapping of instantiated Go generic APIs via `go/types`—full coverage is a **long tail**, not a release blocker. |
| E | **Tooling** | LSP hover / diagnostics for type parameters and instantiation sites. |

Stages A–B unlock most **language** generics; C–D unlock **interop and stdlib**; E is ongoing.

---

## Design decisions to lock early

1. **Surface syntax** (parser): one consistent style for type parameters and applications (e.g. Go-like `func f[T](x T)`, `Box[T]`, `f[Int](1)` vs alternatives). Touches [`forst/internal/parser/type.go`](../../../../forst/internal/parser/type.go), `function.go`, `typedef.go`.
2. **Codegen strategy**: emit **Go 1.18+ generic Go** (`type Box[T any] struct`, `func Id[T any](x T) T`) vs **monomorphize** to concrete types only. Emission touches [`forst/internal/transformer/go/`](../../../../forst/internal/transformer/go/); repo `go.mod` already targets a modern Go toolchain.
3. **Constraint language (minimal first)**: start with `any` / `comparable` / **ordered** (subset already mirrored in [`builtin_type_helpers.go`](../../../../forst/internal/typechecker/builtin_type_helpers.go) for `min`/`max`) before full interface constraints.

---

## Implementation phases

### Phase 1 — AST and binding

- Extend AST for **declared type parameters** (name, optional constraint) on **type definitions** and **functions**; distinguish parameter identifiers from type constructors in [`forst/internal/ast/`](../../../../forst/internal/ast/).
- **Name resolution / scoping**: generic parameter scope for function body and type bodies (alongside existing [`register.go`](../../../../forst/internal/typechecker/register.go) / lookup paths).
- **Tests**: parser round-trips + negative tests for invalid shadowing / arity.

### Phase 2 — Typechecker: substitution and compatibility

- Introduce a **generic environment** (mapping param → concrete `TypeNode`) for each instantiation.
- Implement **substitution** on `TypeNode`, shapes, and function signatures when referencing `T` inside definitions.
- Extend **`IsTypeCompatible` / unify** ([`typeops_test.go`](../../../../forst/internal/typechecker/typeops_test.go) already has some `TypeParams` cases) for instantiated types.
- **Regression tests**: `Box[Int]` vs `Box[String]` incompatibility; identity of `Box[Int]` across sites.

### Phase 3 — Generic types (Stage A) end-to-end

- Typecheck **generic typedefs** / shapes with parameters; **instantiation** at use sites (`Box[Int]{ ... }` or equivalent).
- **Transformer**: emit `type Name[T any] struct { ... }` (or monomorphized `Name_int`) per early design decision.
- **Golden / integration**: extend [generics.ft](../../../../examples/in/generics.ft) or add `generic_types.ft` with a minimal `Box`-style example.

### Phase 4 — Generic functions (Stage B)

- Parse and store type parameter lists on **functions**; typecheck body with parameters in scope.
- **Call resolution**: explicit `f[Int](...)` first; then **inference** only when PHILOSOPHY’s “uniquely determined” rule applies (narrow cases + clear errors).
- Emit generic Go functions or monomorphized overloads.

### Phase 5 — Constraints (Stage C)

- Parse constraint clauses; enforce **assignability** to constraint (start with `any`, `comparable`, ordered builtins).
- Connect to **future** interface/union work without blocking Phases 3–4 on full interface satisfaction.

### Phase 6 — Interop (Stage D)

- Extend [`go_interop.go`](../../../../forst/internal/typechecker/go_interop.go) / `goTypeToForstType` for **instantiated** generic Go types and signatures using `go/types` (**incrementally**; full stdlib coverage is **not** a gate—see risks).
- **Expression parser** for `make`/`new` type arguments ([ROADMAP](../../../../ROADMAP.md) prerequisite).
- Tests against **one** small real API per milestone (e.g. a single `slices` or `maps` function) to prove the pipeline; add mappings over time.

### Phase 7 — Tooling and docs

- LSP: hover for generic definitions and instantiations.
- Update [ROADMAP.md](../../../../ROADMAP.md) row **Generic types** from planned → experimental/done with honest scope notes.
- Cross-cutting: roadmap flags **type aliases + generics** and **binary type expressions + generics**—schedule integration passes when each lands.

---

## Risks and mitigations

### 1. Structural hashing vs named generic instantiation identity

**Risk:** Anonymous shapes use hash-based names today; user generics need a single canonical identity for `Box[Int]` across definitions, imports, and codegen.

**Mitigations:**

- Treat **instantiated named generics** as first-class: key identity by `(generic symbol, sorted type args)` in the typechecker, not by re-hashing substituted struct bodies alone.
- Keep **structural/anonymous** types on the existing hasher path; only mix with generics where substitution is fully applied before hashing (tests that `Box[Int]` from two sites unify).
- Document when users should prefer **named** `type Box[T] = …` over anonymous shapes parameterized by `T` to avoid duplicate structural hashes.

**Examples (illustrative — final syntax TBD):**

| Allowed / goal | Not supported or out of scope for identity rules |
| ---------------- | ------------------------------------------------ |
| Two references to **`Box[Int]`** (same named generic `Box`, same args) unify to **one** instantiated type for fields, assignments, and emit. | Relying on **two independent anonymous** shapes with the same field layout to be “the same type” without a shared named definition (structural/hash territory; generics do not merge unrelated anonymous shapes). |
| **`type Pair[A, B] = { first: A, second: B }`** then **`Pair[Int, String]`** has stable identity keyed by `Pair` + `[Int, String]`. | Instantiation identity that **depends on declaration order** or **source file** rather than `(symbol, type args)` (the design should reject this). |

### 2. Go interop (long tail — accepted)

**Risk:** `go/types` exposes unbound type parameters, instantiated `Named` types, and generic signatures; mapping every future stdlib or third-party generic API is open-ended.

**Stance:** **We accept the long tail.** Shipping **Forst-side** generics and emit does **not** wait on complete Go generic interop.

**Mitigations:**

- **Incremental allowlist:** extend `goTypeToForstType` / parameter checking per **high-value** APIs with tests, not big-bang coverage.
- **Stable diagnostics:** explicit `unsupported Go … type` (or similar) with **actionable** messages (e.g. suggest a typed wrapper in `.go`) rather than silent wrong types.
- **Escape hatches:** hand-written Go helpers or thin wrappers remain valid; document gaps in ROADMAP/examples.
- **Defer edge cases:** channel/type-parameter combinations, contravariant positions, etc. behind “not yet” errors until targeted.

**Examples (illustrative):**

| Allowed (incremental) | Not supported until explicitly implemented / mapped |
| --------------------- | --------------------------------------------------- |
| Call a **non-generic** or **already-mapped** imported function; types align with today’s mapping (basics, slices, pointers, `error`). | Imported function returns **`T`** where **`T`** is still a **free type parameter** at the call boundary (e.g. `func Clone[S ~[]E](S) S` returning **`S`**) — **unsupported** until mapping exists. |
| After a milestone maps e.g. **`slices.Index`**, call with concrete **`[]Int`** and use the **concrete** result type. | Assume **every** external generic API works without tests — long tail. |
| **Escape hatch:** implement **`myClone(xs []int) []int`** in `.go`, import and call from Forst with a **non-generic** signature. | Silently treating unsupported generic returns as **`interface{}`** or wrong concrete type — **never**; must remain an **explicit error**. |
| **`make` / `new`** with Forst type syntax once parser + checker support it ([ROADMAP](../../../../ROADMAP.md)). | **`make([]T, n)`** with **`T`** only known at runtime — not valid in Go; not a goal. |

### 3. PHILOSOPHY vs inference ambiguity

**Risk:** Over-eager type-argument inference conflicts with “explicit when unclear” and hides bugs.

**Mitigations:**

- Ship **explicit** instantiation syntax first (`f[Int](x)` or equivalent); add inference only for **narrow** rules (e.g. single parameter determined by argument types) with tests.
- When inference fails, diagnostics that **name** the ambiguous parameter and suggest explicit type arguments.
- Log/trace at **debug** for inference attempts during development ([PHILOSOPHY.md](../../../../PHILOSOPHY.md)).

**Examples (illustrative — syntax TBD):**

| Allowed | Not supported (reject or require explicit args) |
| ------- | ----------------------------------------------- |
| **Explicit** type application: **`id[Int](x)`** (or as the language defines it). | **`id(x)`** when **`x`** could instantiate **`T`** to **`Int`** or **`Float`** equally — **ambiguous**; compiler must **not** guess. |
| Narrow inference **if** the ruleset proves **at most one** valid instantiation — only with tests and a specified algorithm. | Inference using **only** return context, **multiple** independent parameters, or **constraints** not in the solver yet. |
| Clear error: *cannot infer `T`; try `f[Int](…)`*. | Silent choice of **`Int`** because it was “first” internally. |

---

## Testing strategy

- **Unit tests** at each layer: parser → typechecker substitution → transformer emit → `go test` on emitted snippets.
- **Integration:** `task example:*` or dedicated `examples/in/` once syntax is stable.
- Prefer **narrow** tests per inference rule; avoid drive-by refactors.

---

## Result types, generics, and narrowing (Ok and Err)

**See also:** [optionals — `Result` types](../optionals/02-result-and-error-types.md), [optionals — `ensure` / `is`](../optionals/09-ensure-is-narrowing-and-binary-types.md), [optionals — type guards and `Result`](../optionals/10-type-guards-shape-guards-and-optionals.md), [guard RFC — generic type guards](../guard/guard.md).

The [Purpose](#purpose-and-benefits-for-the-language) section already uses **`Result[Int, AppError]`** as an illustrative **instantiated** generic type. That is **the same** **identity** story as **`Box[Int]`** (parameter symbol + concrete type arguments), but applied to Forst’s **`Result(Success, Failure)`** convention instead of a user-defined **`Box`**.

### Why this belongs in the generics RFC

- **`Result`** is **two-parameter** and **nominal** in intent: callers care about **`Result(Int, ParseError)`** vs **`Result(Int, Error)`** as **distinct** instantiated types, with **subtyping** on the **failure** parameter where defined ([optionals 02](../optionals/02-result-and-error-types.md)).
- **User generics** (this RFC) supply **substitution** and **compatibility** for those parameters—whether **`Result`** is ultimately a **builtin** type constructor or a **library** generic **`type Result[S, F] = …`**.
- **Narrowing** is **orthogonal** to emit but **depends** on knowing the **instantiated** **`Success`** and **`Failure`** types: after **`if r is Ok()`**, the **success** payload should have type **`Success`**; after **`if r is Err(...)`**, the **failure** payload should refine to **`Failure`** (or a **subtype**) ([optionals 02 §5](../optionals/02-result-and-error-types.md), [optionals 09](../optionals/09-ensure-is-narrowing-and-binary-types.md)).

### `Ok()` as a native type guard on `Result`

**Discriminated** **`Result`** is **`Ok(Success) | Err(Failure)`** ([optionals 01 §3](../optionals/01-single-return-unions-and-go-interop.md)). The **guard** subsystem should treat **`Ok()`** and **`Err(...)`** as **assertions** on that sum—**analogous** to the [guard RFC’s **generic type guards**](../guard/guard.md#generic-type-guards) example: the **guard** carries **type parameters** that **specialize** with the **`Result`** instance.

**Illustrative ergonomics** (syntax not final):

- **`if r is Ok()`** — narrow **`r`** to the **success** case; **payload** type is the **`Success`** argument of **`Result(Success, Failure)`** (e.g. **`Int`** for **`Result(Int, ParseError)`**).
- **`ensure r is Ok() or err`** — same **narrowing** in the **success** continuation; **failure** path returns **`error`** per **`ensure`** semantics ([optionals 09](../optionals/09-ensure-is-narrowing-and-binary-types.md)).
- **`if r is Err()`** / **`ensure r is Err() or err`** — mirror for the **failure** branch; **payload** narrows to **`Failure`** (subject to **`Error`** hierarchy rules in [optionals 02](../optionals/02-result-and-error-types.md)).

So **`Ok()`** is **not** an ad hoc keyword-only hack: it is the **success** discriminant of **`Result`**, composed with **generic instantiation** so the checker **knows** which **`Success`**/**`Failure`** types apply at **`r`**’s static type.

### Staging relative to this RFC’s phases

| Dependency | Notes |
| ---------- | ----- |
| **Binary unions + narrowing** ([ROADMAP](../../../../ROADMAP.md)) | **`Result`** narrowing shares the **same** **`is` / `ensure`** pipeline as **`T \| Nil`** and other sums ([optionals 10](../optionals/10-type-guards-shape-guards-and-optionals.md)). |
| **Phases 1–3** (AST, substitution, generic types) | **Instantiated** **`Result[Int, ParseError]`** (illustrative) needs **stable** type identity and **substitution**, same as **`Pair[Int, String]`** in [§Risks](#1-structural-hashing-vs-named-generic-instantiation-identity). |
| **Phase 4** (generic functions) | Combinators like **`map`/`and_then`** on **`Result`** (if expressed as **generic** functions) follow **call** resolution and **explicit** type args first. |
| **Emit** | **`Result`** still **lowers** to idiomatic **`(T, error)`** where the optionals RFC says so ([optionals 02 §4](../optionals/02-result-and-error-types.md)); **narrowing** is a **source**-level guarantee, not a requirement to emit **generic** **`Result`** structs in Go. |

**Bottom line:** Once **`Result(Success, Failure)`** and **user generics** both exist, **`if r is Ok()`** / **`ensure r is Ok() or err`** with **narrowed** payloads **matches** the **illustrative** **`Result[Int, AppError]`**-style example in this RFC: **one** **parameterized** type, **one** **discriminant** guard, **one** narrowing story.

---

## References

- [ROADMAP.md](../../../../ROADMAP.md) — feature status table.
- [PHILOSOPHY.md](../../../../PHILOSOPHY.md) — inference and explicitness.
- [examples/README.md](../../../../examples/README.md) — RFC layout conventions under `examples/in/rfc/`.
- [optionals / `Result`](../optionals/02-result-and-error-types.md), [optionals hub](../optionals/00-crystal-inspired-optionals.md) — **`Result`** vs **`T?`**, narrowing, **`Ok()`** / **`Err()`** (cross-links [§10](#result-types-generics-and-narrowing-ok-and-err)).
- Internal surfaces: [`ast.TypeNode`](../../../../forst/internal/ast/type.go), [`go_interop.go`](../../../../forst/internal/typechecker/go_interop.go), [`transformer/go`](../../../../forst/internal/transformer/go/).

---

## Document status

**Draft RFC — design and phasing only.** Syntax examples are **illustrative**; implementation tasks are **not** started by this document. Revise this file as decisions (syntax, monomorphization vs Go generics emit) are locked.
