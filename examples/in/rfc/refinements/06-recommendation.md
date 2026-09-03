# 06 — Recommendation (historical)

**Status:** **Superseded** by [12 — Accepted decision](./12-accepted-decision.md) and [13 — Refinement stability](./13-refinement-stability.md). Kept as the draft that 12 modified.

**Conflicts with 12 (do not implement these from this file):**

- Guard unfold into **DNF** as language semantics (§3, §5.5). Use `All`/`Any`/`Atom` ([12 §20](./12-accepted-decision.md#20-avoid-mandatory-dnf-expansion)).
- Implementation order that skipped recursive guard-language enforcement, failure blocks in guards, defined-vs-alias types, and runtime-parameter types.
- **`ensure … or <error>`** as the failure keyword. [12](./12-accepted-decision.md) uses **`else`**.
- Mutation as an afterthought. [13](./13-refinement-stability.md) is the full stability model: drop facts, do not lock fields, Go untrusted.

**Still aligned with 12:** assertion `or` vs typed `else`; `|` for types; fail-closed guard `if`; literal unions; no SMT; no boolean `ensure`.

The original text follows.

---

---

## 1. Decision

Forst treats refinements as a **closed atom algebra** with **Meet** (`.`, sequential `ensure`, typedef `&`) and **Join** (`|`). Type guards **name** a formula in that algebra. They are not general programs.

**Logical or** is `|`. **`ensure` … `or`** remains **failure only**.

Three user constructs, one lattice:

1. **Literal unions** for finite value sets (enums).
2. **Assertion `|`** for disjunction on one carrier, including call-site `ensure`.
3. **Restricted `if` in type guards** for tag-split / different fields per arm, **fail closed**.

No SMT. No boolean `ensure`. No `return` in guards. No loops in guards. No opaque user `x is T`.

## 2. What authors write

### Enum / allowed values

```ft
type GameStatus = "playing" | "x_won" | "o_won" | "draw"

type GameState = {
  cells: []String,
  nextPlayer: String,
  status: GameStatus,
}

func parseStatus(raw String): Result(GameStatus, BadStatus) {
  ensure raw is GameStatus or BadStatus()
  return raw
}
```

Desugar: `String.Value("playing") | …`. Kernel `Value` unions are valid without sugar.

### Refinement or (same carrier)

```ft
type PhoneNumber =
  String.Min(3).Max(10) & (
    String.HasPrefix("+")
    | String.HasPrefix("0")
  )

func checkPrefix(p String): Result(String, BadPhone) {
  ensure p is HasPrefix("+") | HasPrefix("0") or BadPhone()
  return p
}
```

### Named conjunctive guard (unchanged)

```ft
is (ctx AppContext) LoggedIn() {
  ensure ctx.sessionId is Present()
  ensure ctx.user is Present()
}
```

### Named disjunctive guard (assertion `|`, not empty `if`)

```ft
is (p String) PhonePrefix {
  ensure p is HasPrefix("+") | HasPrefix("0")
}
```

### Tag split (restricted `if`, implicit fail)

```ft
is (m Shape) ValidMessage {
  if m.kind is Value("login") {
    ensure m.user is Min(1)
    ensure m.password is Min(8)
  } else if m.kind is Value("token") {
    ensure m.token is Min(20)
  }
}
```

Unmatched `kind` → guard false. Generated Go must **not** fall through to `return true`.

## 3. Semantics (short)

- Unfold each guard once to a DNF of atoms ([04](./04-guard-bodies.md), [08](./08-complexity-bounds.md)).
- `ensure x is A | B` succeeds if `A` or `B` holds; successor type is Join(`A`,`B`) Meet current type.
- Subtyping between user formulas: nominal tag `G`, plus syntactic atom inclusion after unfold, plus **built-in-only** lattices (literal `⊆`, `Min`/`Max` intervals). No general implication.
- Recursion through guard names: reject.
- Guard `if` condition: `is` only. Unmatched: fail.

## 4. What never ships (from [05](./05-solution-space.md))

Boolean `ensure`; logical `or`; opaque `return` in guards; non-`is` guard conditions; loops in guards; SMT; dependent `Π`; Zod-split schemas; `OneOf` as a second enum theory if literals exist; encoding enums as type-guard `if` chains in docs/examples.

## 5. Implementation order

1. **Fail closed** on type-guard `if` (soundness bug, no new syntax). Tests: unmatched arm → `G_*` returns false; typechecker treats residual as not `G`.
2. **Assertion `|`** in `ensure` / `is` (parser + Join in `InferAssertionType` + runtime `||` of checks). Tests: Phone prefix; diagnostic if user writes `or` instead of `|`.
3. **Literal unions** in typedefs (`"a" | "b"` and/or `String.Value("a") | …` emit + `ensure x is GameStatus`). Tests: tictactoe-shaped status; TS emit of string unions; Go named string + check.
4. **Typedef constraint Join** actually **checked and lowered** for one-carrier refinements (PhoneNumber example stops being skip-only).
5. **DNF size cap** (TG-5) with a hard error, not a hang.
6. **Join of tagged shapes** as real union types (hover/assignability). Can trail 1–4; `T.G` as a name is enough until then.
7. **`match`** remains a later RFC on this algebra.

Do not wait for 6 or 7 to ship 1–3. Enums and `ensure … is A | B or Err()` are the consumer pain.

**Use-site precision ([09](./09-nominal-proxies.md)):** it is coherent to **keep call-site `ensure` as a single name** and to **not unfold** a guard’s internal `|` / `if` into the caller’s type. Nested `ensure is H()` stays a **name in a conjunctive export**. That is enough for “runtime checked, statically branded.” Unfolding `φ_G` into callers (this document’s PhonePrefix Join at the use site) is extra precision, not required for soundness — as long as names that hold on **some** paths are never exported as if they held on **all** paths.

## 6. Docs and RFC fallout

- Withdraw `return len(…)` examples in [guard.md](../guard/guard.md) as non-normative; point here.
- Errors RFC snippets that say `ensure n > 0` stay invalid; they mean `ensure n is GreaterThan(0)`.
- Sidecar `OneOf([...])` examples retarget to literal unions / `Value` Join.
- User docs: one cheat sheet — `.` and sequential `ensure` = and; `|` = or; `or` = error; type guards = named formula; enums = literal unions.

## 7. Success criteria

A new backend author can:

1. Type a JSON status field as `"playing" | "x_won" | …` without a type guard.
2. `ensure phone is HasPrefix("+") | HasPrefix("0") or BadPhone()` and get narrowing + a Go `if`/`||`.
3. Write `LoggedIn` as two `ensure`s and still get field presence.
4. Write `ValidMessage` as `if is` cases and understand that a third tag **fails**.
5. Never see “the refinement checker could not prove this.”

If (5) starts happening, we have left the fragment. Stop and delete the feature that caused it.
