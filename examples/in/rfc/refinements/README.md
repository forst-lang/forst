# Analyzable refinements, disjunction, and enums (RFC)

**Status:** **Accepted** — [12](./12-accepted-decision.md) is the normative `ensure` / guard / assertion `or` / typed `else` / type `|` decision. [13](./13-refinement-stability.md) is the normative mutation / aliasing / Go-interop decision. [14](./14-type-targets.md) is the normative `ensure` type-target decision (types vs assertions on the RHS of `is`). Background analysis is [00](./00-the-trap.md)–[11](./11-ensure-worth-and-user-types.md). Implementation: [`.plans/analyzable-refinements`](../../../../.plans/analyzable-refinements/README.md).

**Audience:** Language designers and compiler contributors.

**Depends on:** [type guards](../guard/guard.md) (TG-1–TG-7), [`ensure` / `is` / binary types](../optionals/09-ensure-is-narrowing-and-binary-types.md), [guards vs optionals](../optionals/10-type-guards-shape-guards-and-optionals.md), [`switch` vs `match`](../switch-match/00-switch-and-match.md), [SEMANTICS_NARROWING](../../../../forst/internal/typechecker/SEMANTICS_NARROWING.md), [PHILOSOPHY](../../../../PHILOSOPHY.md) (no dependent types, no type-level computation, no SMT).

---

## The problem in one paragraph

Today a type guard is a **playlist of conjunctive `ensure` statements**. That is a Horn clause: the subject is in the guard iff every atom holds. Real programs need **or**: a status is one of four strings; a phone starts with `+` **or** `0`; a message is a login shape **or** a token shape. If we satisfy that by letting guard bodies become ordinary programs (`return`, `==`, `||`, loops), the compiler can no longer **extract** a refinement. Subtyping `{x:T | P} <: {x:T | Q}` then requires proving `P ⇒ Q`. That problem is undecidable for arbitrary `P`, and it is the slope Liquid Haskell, Sage, and Whiley went down. Forst already forbade that slope (TG-1, TG-5, no SMT). This RFC keeps the prohibition and still gives users disjunction.

## Accepted model (one paragraph)

**Types** say what values may exist (including literal unions). **Guards** name reusable invariants on **domain types**. **`ensure`** establishes a refinement against a **type or an assertion** ([14](./14-type-targets.md)). **`or`** is assertion alternative on one place. **`else`** is typed failure. **`|`** is type union, not assertion disjunction. **`ensure … else { … }`** is a failure-handling block (ordinary functions / `main` only). Restricted `if is` stays in guards and **fails closed**. Callers see the **guard name**, plus facts that hold on **every** successful path. The compiler stores `All`/`Any`/`Atom`, not DNF. No SMT. Details: [12](./12-accepted-decision.md), [14](./14-type-targets.md). After a fact is established, it remains only while its dependencies are known not to have changed: [13](./13-refinement-stability.md).

## Topic index

| Doc | Topic |
| --- | --- |
| **[12 — Accepted decision](./12-accepted-decision.md)** | **Normative** for `ensure` / guards. Assertion **`or`**; typed **`else`**; **`|`** for types; failure blocks; fail-closed `if`; literal unions; domain types; `must()`; no DNF. Mutation is a pointer to 13 |
| **[13 — Refinement stability](./13-refinement-stability.md)** | **Normative** for mutation / aliasing / Go. Facts with deps; overlap invalidates; no borrow checker; Forst effects inferred; Go untrusted |
| **[14 — Type targets](./14-type-targets.md)** | **Normative.** `ensure place is Type` (no parens) vs `ensure place is Guard()`. Enum subsets are types, not assertion `or` |
| [00 — The trap](./00-the-trap.md) | Refinement extraction, predicate implication, Datalog vs Prolog, why new languages fail |
| [01 — Existing commitments](./01-existing-commitments.md) | What TG-1–TG-7, PHILOSOPHY, and the narrowing snapshot already decided. **`or` as failure is superseded by `else`** |
| [02 — Three kinds of union](./02-three-kinds-of-union.md) | Literal/enum vs refinement-of-one-carrier vs discriminated/tag split |
| [03 — `else` vs `or`](./03-or-vs-pipe.md) | Typed failure vs assertion alternative. Filename kept; **`or` is assertion `or`**; **`|` is types** |
| [04 — Guard bodies](./04-guard-bodies.md) | Conjunction, restricted `if` as Join, implicit fail, no loops — **DNF unfold superseded by 12 §20** |
| [05 — Solution space](./05-solution-space.md) | Every alternative, compared |
| [06 — Recommendation](./06-recommendation.md) | **Superseded** by [12](./12-accepted-decision.md) where they conflict |
| [07 — Prior art](./07-prior-art.md) | TypeScript, Typed Racket, Liquid Haskell, F*, Sage, Whiley, Rust, Kotlin, Zod, Prolog/Datalog |
| [08 — Complexity bounds](./08-complexity-bounds.md) | Polynomial analysis — **DNF cap as semantics withdrawn**; `All`/`Any` depth may still be bounded |
| [09 — Nominal proxies](./09-nominal-proxies.md) | Guard name as caller fact; universal export. **Call-site assertion `or` is allowed** (12 §5) |
| [10 — Predicates are not sums](./10-predicates-are-not-sums.md) | Named guards can replace **anonymous predicate `\|`**. They do **not** make type unions moot |
| [11 — Worth it? User types, not `String.Foo`](./11-ensure-worth-and-user-types.md) | Go has no refinements; `ensure` is Effective Go’s early return. Guards on builtins are a waste. **Aliases to `String` are still `String`.** |

## Non-goals of this folder

- Designing `match` syntax. [switch-match](../switch-match/00-switch-and-match.md) already deferred that; this RFC supplies the **algebra** `match` would sit on.
- Full `T | Nil` / `Result` construction. Those stay in the [optionals hub](../optionals/README.md).

## Document status

**Accepted** in [12](./12-accepted-decision.md), [13](./13-refinement-stability.md), and [14](./14-type-targets.md). Implement via [`.plans/analyzable-refinements`](../../../../.plans/analyzable-refinements/README.md). Do not add boolean `ensure`, `|` as assertion disjunction, enum variants as assertions, opaque `return` in guards, SMT, or a borrow checker.
