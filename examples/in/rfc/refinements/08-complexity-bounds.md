# 08 — Complexity bounds and not confusing the user

**Status:** How TG-5 stays true. **DNF size cap as language semantics is withdrawn** ([12 §20](./12-accepted-decision.md#20-avoid-mandatory-dnf-expansion)). Bound **tree depth / node count** of `All`/`Any` if needed, do not expand.

---

## 1. What “polynomial” applies to

Checking a **use** of `ensure x is G()` is: look up `G`, emit `G_*`, Meet the **name** `G` and `must(G)` with `x`’s type. That is cheap if `G` is stored as an `All`/`Any`/`Atom` tree.

Defining `G` is a walk of a small AST into that tree. **Do not** distribute `(A∨B) ∧ (C∨D)` into four conjuncts.

Implementation may reject a `GuardExpr` whose node count or depth exceeds a compiler limit (hangs, not semantics). Do not silently drop children.

Inlining other guards into the **runtime** `G_*` is fine; inlining their **internal** `Any` into the caller’s exported facts is not ([12 §18–19](./12-accepted-decision.md)). Recursion is already forbidden.

## 2. Built-in lattices vs user formulas

**Cheap, closed, language-defined:**

| Theory | Implication we may implement | Not |
| --- | --- | --- |
| Literal sets | `{a,b} ⊆ {a,b,c}` | Regex “looks like a status” |
| `Min`/`Max` / `LessThan` on the same numeric field | interval inclusion | `Min(x)` with `x` not a constant |
| `Present` / `Nil` | obvious | |

**Forbidden:** SMT over user functions; proving `HasPrefix("+") ∨ HasPrefix("0")` equivalent to a regex; widening across field *paths* without occurrence keys (already deferred in SEMANTICS_NARROWING).

If `Min(12)` does not yet imply `Min(10)` in the checker, that is a **builtin lattice** TODO, not a reason to add Z3.

## 3. User-facing complexity: one model, three spellings

People get lost when **the same `∨` has four syntaxes**. This RFC allows two spellings of Join and one of tag-split:

| Intent | Spelling | Do not also teach |
| --- | --- | --- |
| And | sequential `ensure`, `.` chain | `&&` in `ensure` |
| Or (same subject) | `or` | `\|` in `is`, `\|\|`, `else` as Join, empty `if`/`else if` |
| Cases (different fields) | `if … is` in a **type guard**, fail closed | function-body `if` as if it defined `φ` |

Function-body `if x is A()` remains **occurrence typing for that function**, not a way to define a reusable `φ`. Reuse = type, assertion, or type guard.

**Complicated predicates** (checksum, “at least two of three classes,” graph reachability):

1. Try to write them as **named atoms + Meet/Join**. `VeryStrong` = three `ensure is`.
2. If that fails, it is a **`func … : Bool`**. Call it from `if`. It does **not** narrow. Optional later: `ensure ok is True() else err` after `ok := checksum(x)` — still no refinement of `x`.
3. Do not add `opaque` guards in v1 ([05 S](./05-solution-space.md)).

That is how we allow “more complicated predicates without confusing the user”: **the type language stays small**; complexity lives in ordinary functions whose types do not pretend to be `φ`.

## 4. Diagnostics that teach the algebra

| User writes | Error should say |
| --- | --- |
| `ensure x is A() or Error()` intending failure | `` `or` starts another constraint; use `else` for errors `` |
| `ensure x is A() \| B()` | `` use `or` for assertion alternatives; `\|` is for types `` |
| `ensure x == "a"` | `ensure` needs `is`; for a set use a literal union or `Value("a") \| …` |
| `return …` in a guard | type guards name a formula; use `ensure` / `if is`; functions return bool but do not narrow |
| `if x.role == "admin"` in a guard | condition must be `is` (TG-2); `Equals`/`Value` or a literal type |
| `for` in a guard | no loops in guards; use element type `"" \| "X" \| "O"` or a function |
| DNF cap exceeded | *(withdrawn)* split the guard only if the `All`/`Any` tree itself is huge |
| Unmatched tags in `ValidMeswsage` | (not an error) runtime false; optional lint: “no else; other kinds fail the guard” |

## 5. What we do *not* optimize for

- Proving two differently written `φ` equal.
- Global best-DNF minimization. **Do not DNF.**
- Making every runtime validator a type. That is the trap.

## 6. Relation to `match` and join-at-merge

When union Join after `if` in **functions** starts using the same lattice ([SEMANTICS_NARROWING](../../../../forst/internal/typechecker/SEMANTICS_NARROWING.md) currently keeps the pre-if type), it must use the **same** Join as assertion `or`. Guard-body Join and function-body join are one implementation. `match` exhaustiveness is “the Join of arms covers the scrutinee type.” None of that requires SMT.
