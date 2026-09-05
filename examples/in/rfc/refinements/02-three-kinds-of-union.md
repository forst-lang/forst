# 02 — Three kinds of “union” (do not encode them as one)

**Status:** Distinctions. Mixing these is how users get hacky type guards for enums and how the checker gets an untyped `∨`. Assertion Join is spelled **`or`** ([12 §5](./12-accepted-decision.md)); **`|`** is type union. Examples below that show `|` inside `is` are historical spelling.

---

## 1. Same English word, three type constructors

| Kind | Inhabitant | Theory | Forst should write | Not |
| --- | --- | --- | --- | --- |
| **Literal / enum** | A value of a **fixed carrier** drawn from a **finite set** | Singleton types; Join is set union; subtyping is `⊆` | `type Status = "playing" \| "x_won" \| "o_won" \| "draw"` | A type guard with four `if`s |
| **Refinement Join** | A value of **one** carrier satisfying **P or Q** | `{x:T \| φ ∨ ψ}` | `String.HasPrefix("+") \| String.HasPrefix("0")` | `ensure a \|\| b` |
| **Discriminated sum** | **Different shapes**, disjoint by a **tag** | Ordinary union of records; occurrence typing on the tag | `if m.kind is Value("login") { ensure … } else if m.kind is Value("token") { … }` inside a guard, or a typedef of two shapes | `{x: Shape \| true}` with overlapping fields and no tag |

TypeScript exposes (1) and (3) in the **type grammar** and treats (2) as “write a union of branded/literal types.” It does **not** ask user `function isFoo(x): x is Foo` to extract `φ`. That split is why TS enums-as-string-unions feel cheap and why TS user type guards are a soundness hole. Forst wants the cheap part without the hole.

## 2. Literal / enum unions — the thing that feels like an enum

tictactoe today:

```ft
// GameState.status is one of: "playing" | "x_won" | "o_won" | "draw"
type GameState = {cells: []String, nextPlayer: String, status: String}
```

The comment **is** the type the author wanted. Encoding it as a type guard is the hack:

```ft
// DO NOT do this — four arms, no literal type, hover is a name not a set
is (s String) GameStatus {
  if s is Value("playing") {}
  else if s is Value("x_won") {}
  else if s is Value("o_won") {}
  else if s is Value("draw") {}
}
```

Even with implicit-fail `if` ([04](./04-guard-bodies.md)), this is the wrong construct: the interesting object is the **set**, not a predicate name. Call sites want `status: GameStatus` on the shape and `ensure g.status is GameStatus` only when the static type is still `String` (JSON boundary).

**Go `const` / `iota`** already exists for **numeric** Go-shaped enums. It does not give JSON string status a type, and it does not emit TypeScript `"playing" | "x_won"`. Forst’s TS-interop goal makes **string (and int) literal unions** the enum solution, not `iota` and not type guards.

**Kernel spelling** (already have `Value` constraints and typedef `|`):

```ft
type GameStatus =
  String.Value("playing")
  | String.Value("x_won")
  | String.Value("o_won")
  | String.Value("draw")
```

**Sugar** (TypeScript-shaped, recommended):

```ft
type GameStatus = "playing" | "x_won" | "o_won" | "draw"
```

Sugar desugars to `Value` unions on a common carrier inferred from the literals (all strings → `String`, all ints → `Int`, mix → error).

**Runtime:** membership test (`==` chain or small set). **Static:** the type **is** the set. After `ensure s is GameStatus`, `s` has type `GameStatus`, not `String`. Assigning `"nope"` is a type error when the source is a literal; dynamic `String` still needs `ensure`.

**`OneOf(["a","b"])`** (sidecar RFC sketches) is the same set with extra vocabulary. Prefer not to ship it if literals exist. If a builtin is wanted for `ensure` without a named type, `ensure s is Value("a") | Value("b")` is enough.

## 3. Refinement Join — one carrier, two predicates

PhoneNumber in the guard RFC is the canonical example: still a `String`, still one Go `string`, but `φ = Min(3) ∧ Max(10) ∧ (HasPrefix("+") ∨ HasPrefix("0"))`.

This is **not** a discriminated union. There is no tag field. Go lowering stays `string` plus a check. TS lowering can be `string` with a comment, or a branded alias; it cannot be a TS union of two string types that the JS runtime distinguishes — prefixes are not tags.

**Where to write it:**

- **Type:** `type PhoneNumber = String.Min(3).Max(10) & (String.HasPrefix("+") | String.HasPrefix("0"))`
- **ensure:** `ensure phone is HasPrefix("+") | HasPrefix("0") else BadPhone()` when the length bounds already live on the type
- **Guard (named formula):** sequential `ensure` of a union assertion, not an `if` with empty bodies

Empty-bodied `if` for prefix-or is worse than `|` on the assertion. `|` is the thing the user can hover.

## 4. Discriminated sums — different fields per variant

```ft
// Login has user+password; Token has token. Not a String predicate.
is (m Shape) ValidMessage {
  if m.kind is Value("login") {
    ensure m.user is Min(1)
    ensure m.password is Min(8)
  } else if m.kind is Value("token") {
    ensure m.token is Min(20)
  }
}
```

Success type is a **Join of shapes**, i.e. a real union:

```text
{ kind: "login", user: String.Min(1), password: String.Min(8), … }
| { kind: "token", token: String.Min(20), … }
```

This **requires** union types at join (already experimental; [SEMANTICS_NARROWING](../../../../forst/internal/typechecker/SEMANTICS_NARROWING.md) currently widens to the pre-if type). Until Join is real, the guard still **runs** as a DNF of field tests; hover may only show `ValidMessage`. That is acceptable v1. Inventing a second “enum keyword” for this is not: it is [switch-match](../switch-match/00-switch-and-match.md)’s `match` job on the same algebra.

**Why type guards (restricted `if`) earn their keep here and not for `GameStatus`:** each arm **adds different atoms on different paths**. `|` of two prefix constraints does not. Using `if` only for tag-split keeps the user’s model: **`|` = or of refinements on the same subject; `if is` = split the subject.**

## 5. What Go and TypeScript emit (do not lie)

| Kind | Go | TypeScript |
| --- | --- | --- |
| Literal union of strings | `string` + check, or named `type GameStatus string` with consts | `"playing" \| "x_won" \| …` |
| Refinement Join on `String` | `string` + check | `string` (optionally branded alias). **Not** a TS union unless the atoms are literals |
| Discriminated shapes | struct/interface union; same problem as general `A \| B` (today `any` for non-error unions) | `{kind: "login", …} \| {kind: "token", …}` — this is the TS form users expect |
| Named error union | Sealed interface (already) | TS union of error shapes (already aimed) |

General `String | Int` staying `any` in Go is an **emit** limit, not a reason to refuse `|` on refinements of **one** carrier. One-carrier refinement Join is the lowerable subset [optionals 01](../optionals/01-single-return-unions-and-go-interop.md) already asked for (“restrict surface unions to lowerable subsets”).

## 6. Decision rule for authors (and for us)

```text
Finite known values of one primitive?     → literal union type.
Same fields, predicate P or Q?            → `|` in the assertion / typedef.
Different fields depending on a tag?      → restricted if in a type guard (and later match).
Need a runtime bool the checker cannot see? → func, not a type guard.
```

If a design uses type guards for the first bullet, it has already failed the end consumer. That is the tictactoe comment.
