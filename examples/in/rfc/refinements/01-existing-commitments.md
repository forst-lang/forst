# 01 — Existing commitments this RFC must not break

**Status:** Inventory. Sources of truth stay in the cited RFCs; this document only records what they already forbade or required.

---

## 1. Type guards exist to be extracted, not to be general functions

[guard.md](../guard/guard.md) Summary: guards must be **predictable, composable, efficient to analyze statically**, and safe for narrowing, Go emit, and `.d.ts`.

The **guide-level** examples still show `return len(password) >= 12`. The **normative rules (TG-1–TG-7)** contradict that. The compiler implements the rules, not the early examples:

| Rule | Meaning | Compiler today |
| --- | --- | --- |
| **TG-1** | No `return`. Only `if` / `else if` / `else` and `ensure`. | [`inferTypeGuardNode`](../../../../forst/internal/typechecker/infer_typeguard_node.go) rejects `return` and anything except `if`/`ensure`. |
| **TG-2** | `if` conditions must be `is`, not `==` or calls. | Same file: condition must be `BinaryExpression` with `TokenIs`. |
| **TG-3** | Only `ensure` refines. **No typed failure** in guards. | [`parseEnsureStatement`](../../../../forst/internal/parser/ensure.go): `or` in a type-guard context is a parse error today; [12](./12-accepted-decision.md) makes that `else` and also forbids failure blocks. |
| **TG-4** | Nested shape refinement does not drop fields. | Shape-guard path; not the disjunction issue. |
| **TG-5** | Polynomial-time analysis; reject combinatorial paths. | Size cap not yet enforced; the *intent* is binding. |
| **TG-6** | Bare `{}` for shapes. | Unrelated to `∨`. |
| **TG-7** | No identifiers except receiver and guard parameters. | Keeps `φ` closed (no I/O, no globals). |

**Why no `return`:** a boolean return is an **opaque** predicate. The checker would have to execute or prove the expression. That is [00 §2](./00-the-trap.md). The early `return len(…)` examples are **withdrawn** by TG-1; this RFC does not revive them.

**Why no typed `else` / failure blocks inside guards:** typed failure is the **failure continuation** of `ensure` in functions ([12](./12-accepted-decision.md) §3.2; [errors 01](../errors/01-ensure-only-failure-returns.md)). A guard is not allowed to return an error; it returns whether `φ` holds. `or` was that continuation until 12 made `else` failure and `or` assertion alternative. Type `|` is not assertion `or`.

**Why `if` was allowed at all:** TG-1 is the original sketch of **disjunction as a chain of `is` tests**, not as boolean algebra. The implementation never defined what an **unmatched** `if` means, and lowering **succeeds** by default ([00 §8](./00-the-trap.md)). This RFC supplies that meaning rather than deleting `if`.

## 2. `ensure` is a test of an assertion, plus an optional failure

Call-site `ensure` is **not** a boolean statement. The parser requires `is` (or `ensure !ident` as `Nil()`). Comparisons are rejected:

```text
ensure requires 'is' with a constraint (not a comparison)
```

So `ensure n > 0` and `ensure x == "a" || x == "b"` are **not** in the language, even though some older RFC snippets write the first. The **live** grammar today is:

```text
ensure <subject> is <assertion> [ { block } ] [ or <error> ]
```

[12](./12-accepted-decision.md) uses `else` for typed failure, names the block a **failure block**, and forbids combining it with typed failure. Assertion alternative is **`or`**. `<assertion>` is a **chain of constraints** (dots), joined by `or`. It is **not** a Forst expression. `|` stays on typedefs.

Typed failure is **[errors 02](../errors/02-first-class-errors-normative.md)**: the only normative way to introduce `Result` failure. It must stay a **separate production** after a complete assertion. See [03](./03-or-vs-pipe.md).

## 3. Binary types already are Meet and Join

[ROADMAP](../../../../ROADMAP.md) and [optionals 01](../optionals/01-single-return-unions-and-go-interop.md): `A | B` and `A & B` share **one type algebra** with narrowing. Typedefs already parse `|` / `&` ([`parseTypeDefExpr`](../../../../forst/internal/parser/typedef.go)). The checker lowers them with `Join` / `Meet` ([`typedef_binary.go`](../../../../forst/internal/typechecker/typedef_binary.go)).

The PhoneNumber example in the guard RFC **and** [`input_validation.skip.ft`](../../../input_validation.skip.ft) already writes **type-level disjunction of refinements**:

```ft
type PhoneNumber =
  String.Min(3).Max(10) & (
    String.HasPrefix("+")
    | String.HasPrefix("0")
  )
```

That is the desired **kernel** for problem (2) in [00 §9](./00-the-trap.md). It is not wired through `ensure` yet, and Go emit for general unions is still `any` ([union docs](../../../../docs/language/union-and-intersection-types.mdx)). The design is not “invent unions.” It is “stop leaving `|` only on typedefs.”

## 4. Narrowing already refuses SMT and loops

[SEMANTICS_NARROWING.md](../../../../forst/internal/typechecker/SEMANTICS_NARROWING.md) non-goals: **no SMT**, no CFG join for arbitrary loops, compound `ensure` subjects deferred.

[optionals 09](../optionals/09-ensure-is-narrowing-and-binary-types.md): one pipeline for `if` / `is` / `ensure` / unions.

This RFC does not add a second analyzer. Guard-body DNF is **the same** Meet/Join.

## 5. `match` is deferred; this RFC must not wait on it

[switch-match](../switch-match/00-switch-and-match.md): Go `switch` shipped; `match` is sugar for `if x is A / else if x is B` plus exhaustiveness. Disjunction in **types** and **guards** is the substrate `match` needs. Shipping **`or` in assertions** and fixing guard `if` semantics is **not** a `match` RFC.

## 6. Errors vs refinements stay orthogonal

[providers ADR-018](../providers/ADR.md) and errors RFCs: do not overload `ensure` for DI, effects, or anything that is not validation + failure. Boolean `||` inside `ensure` would be a third overload. `|` on types does not touch failure.

## 7. What is *not* a commitment (gaps)

| Gap | Today | This RFC |
| --- | --- | --- |
| User `return` in guards | Forbidden | Stays forbidden |
| `Equals` / `OneOf` builtins | RFC examples and sidecar sketches; **not** in the constraint table | Prefer **literal unions** and `Value`; do not add a parallel `OneOf` unless literals slip |
| String literal types `"a" \| "b"` | Not parsed as typedef members | Recommended for enums |
| Assertion-position `or` | `parseAssertionChain` is dots only; `or` is still failure | Recommended ([12](./12-accepted-decision.md) §5) |
| Assertion-position `\|` | not used in `is` | Types only |
| Unmatched `if` in a guard | Succeeds (`return true`) | Must **fail** |
| Implication `Min(12) ⇒ Min(10)` | Not a general prover | Optional **built-in interval** lattice only |
| Opaque “2 of 3 character classes” as a guard | Would need `\|\|` of `Contains` | Encode as named atoms + Meet, or keep as a **function** that does not narrow |

## 8. End-consumer invariant

The person writing a handler should learn **one** sentence:

> `is` / `ensure` test an **assertion**. Assertions compose with **`.` (and)** and **`or` (or)**. `else` after `ensure` is **typed failure**. `|` is **type** union. Type guards **name** an assertion. They are not a second language.

Everything in [12](./12-accepted-decision.md) is that sentence made precise.
