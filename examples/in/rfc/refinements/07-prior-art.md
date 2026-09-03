# 07 — Prior art: who fell in, who split the problem

**Status:** Evidence for [00](./00-the-trap.md) and [05](./05-solution-space.md). Not a survey of every type system.

---

## 1. The split that works in industry

**TypeScript** put **literal unions** and **discriminated unions** in the type grammar. Control-flow analysis (occurrence typing, incomplete) narrows `switch (x.kind)` and `===` literals. User `function isFoo(x): x is Foo { return … }` is **opaque and unsound**. Product lesson: **enums and tagged JSON are types**; do not wait for a theorem prover. Negative lesson: opaque guards. Forst should copy the first, refuse the second ([05 M](./05-solution-space.md)).

**Typed Racket** did occurrence typing **properly**: tests carry propositions; the grammar of tests is limited. That is the theory of Forst `if x is A()` and of restricted guard `if`. It is not Prolog.

**Kotlin sealed classes / Rust enums** make **sums** the enum. There is no `{x: String | x ∈ S}` as a first-class JSON pattern; you wrap. Forst cannot require a wrapper for every wire status without punishing TS interop. Literal unions are the TS-shaped fill. Rust/Kotlin remain the model for **kind 3** (tag split), not kind 1.

## 2. The slope (SMT / hybrid checking)

**Liquid Haskell** (Rondon, Jhala, …): refinements in a **decidable logic**, discharged by SMT. Users still hit “cannot prove,” slow checks, and a second language (the logic). Excellent research; wrong product for “replace TypeScript on the backend.”

**F\***: dependent types + SMT. Proof burden.

**Sage** (Knowles & Flanagan, hybrid type checking): mix static and dynamic for refinements that do not prove. Complexity and blame tracking leaked into the language story. Warning: “we will check what we can and defer the rest” becomes a second semantics users cannot predict. Forst already wants **explicit** `ensure` for the deferred part — that is the honest hybrid: **static atoms vs runtime `ensure`**, not a hidden residual.

**Whiley**: constrained types, SMT, academic timeline. Existence proof that “just add predicates to types” is a research career, not a v1.

Forst [SEMANTICS_NARROWING](../../../../forst/internal/typechecker/SEMANTICS_NARROWING.md) already listed **no SMT**. This RFC is why that line exists.

## 3. Opaque predicates as a product

**Python `TypeGuard` / `TypeIs`**, **Clojure spec**, **Flow** user predicates: runtime truth, weak or no extraction. Fine if types are elsewhere. Forst’s LoggedIn **is** the type. Opaque is a regression.

**Zod / io-ts / valibot**: a **schema** DSL that *generates* types. Two artifacts unless the schema *is* the type. Forst’s `is`/`ensure`/`String.Min` already is that DSL. Adding Zod-shaped `OneOf` beside `|` is a second DSL ([05 G, N](./05-solution-space.md)).

## 4. Logic programming

**Prolog**: clauses + unification + recursion = Turing-complete. **Datalog**: no function symbols / bounded recursion depending on the dialect; polynomial. Type-guard bodies as **non-recursive Datalog over `is` atoms** is the complexity class TG-5 asked for. Recursion and `for` leave that class. We do not want SLD resolution in the compiler.

**JSON Schema `anyOf` / `oneOf`**: DNF explosion is a known interoperability and performance issue. Same warning as TG-5. Cap size; do not cleverly distribute forever.

## 5. What Forst already copied, accidentally

| Forst today | Closest prior art | Risk |
| --- | --- | --- |
| Sequential `ensure` in guards | Horn clause / conjunctive refinement | Too weak, not unsound |
| `if x is A()` in functions | Occurrence typing | Join at merge still pre-if type |
| `if` in guards + `return true` | Neither TR nor TS — **fall-through success** | Unsound extraction |
| Typedef `A \| B` | TS unions / Crystal unions | Emit `any` for general unions |
| `ensure … else Err` | Forst-specific; not logical or | Must not be reused. Historical docs said `or` |
| Guide-level `return len` | TS user predicates | Contradicts TG-1 |

The recommendation is: look like **TS on literal and discriminated unions**, look like **Typed Racket on tests**, look like **Datalog on guard bodies**, look like **Liquid Types only in the sense that the logic is closed and language-defined** — without the SMT backend.

## 6. References (entry points)

- Freeman, Pfenning — Refinement types for ML (1991).
- Rondon, Kawaguchi, Jhala — Liquid Types (PLDI 2008).
- Knowles, Flanagan — Hybrid type checking (Sage).
- Tobin-Hochstadt, Felleisen — The design and implementation of Typed Scheme / occurrence typing.
- Pierce — TAPL (unions, subtyping). Not dependent types.
- TypeScript Handbook — Narrowing; literal types; discriminated unions.
- [guard.md](../guard/guard.md), [optionals 01](../optionals/01-single-return-unions-and-go-interop.md), [switch-match](../switch-match/00-switch-and-match.md).
