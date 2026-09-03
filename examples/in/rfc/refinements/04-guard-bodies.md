# 04 — Type-guard bodies: restricted `if`, no loops

**Status:** Semantics of `is (x T) G { … }`. **Normative fail-closed / restricted `if` still stand.** Unfolding the body into DNF is **withdrawn**; the compiler stores `All`/`Any`/`Atom` ([12 §20](./12-accepted-decision.md#20-avoid-mandatory-dnf-expansion)). Failure blocks inside guards are **forbidden** ([12 §12](./12-accepted-decision.md#12-failure-blocks-are-forbidden-inside-type-guards)).

---

## 1. A guard body denotes a formula, then a boolean function

At definition time the checker lowers the body into a **`GuardExpr`** (`Atom` / `All` / `Any`), not a DNF. At runtime, generated `G_*` evaluates that formula and returns `bool`. Static uses (`ensure x is G()`, `if x is G()`) attach **`G`** and the **universal** facts of `G` ([12 §18–19](./12-accepted-decision.md#18-type-guards-are-primarily-nominal-facts)).

There is no third interpretation (no error return, no mutation, no “best effort” skip).

## 2. Conjunction — sequential `ensure` (keep)

```ft
is (ctx AppContext) LoggedIn() {
  ensure ctx.sessionId is Present()
  ensure ctx.user is Present()
}
```

```text
φ = Present(ctx.sessionId) ∧ Present(ctx.user)
```

This stays the default style. It is what authors already write. Meet of atoms; merge of narrowing names on the subject ([`RegisterSymbolWithNarrowing`](../../../../forst/internal/typechecker/scope.go)).

A union assertion is still **one** `ensure` (one conjunct that is itself a Join):

```ft
is (p String) PhonePrefix {
  ensure p is HasPrefix("+") or HasPrefix("0")
}
```

```text
φ = HasPrefix("+") ∨ HasPrefix("0")
```

Do not expand this into empty `if` arms.

## 3. Disjunction of **paths** — restricted `if` (fix + keep)

Use `if` only when **different arms refine different structure** ([02](./02-three-kinds-of-union.md) kind 3).

**Restrictions (normative):**

| Allowed | Forbidden |
| --- | --- |
| `if` / `else if` / `else` | `for`, `range`, `switch`, `goto`, `defer`, `go` |
| Condition: `subject` or `subject.field` **`is` assertion** (TG-2) | `==`, `\|\|`, calls, `isAdmin(x)` |
| Body: `ensure` **without** a failure block, and nested `if` of this form | `return`, assignment, `:=`, mutating calls, `ensure { … }` |
| Identifiers: receiver + guard params (TG-7) | globals, imports, closures |

**Unmatched = `φ` is false.** There is no implicit success. Lowering: if no arm ran (or `else` is missing and no `else if` matched), `return false`. Today’s `return true` after a skipped `if` is a bug relative to extraction.

**`else`:**

- `else { ensure … }` is another arm (can still fail if those ensures fail).
- `else { }` empty: that arm succeeds with **no extra atoms** (only what the `else if` chain already excluded, when exclusion is known). Prefer not to use empty `else` except when the residual type is already the intended refinement.
- **No `else`:** residual values fail the guard. That is the closed-world default for tag splits.

**Success type of the guard:** the name `T.G`, plus facts true on **every** successful path (`must(All)` = union, `must(Any)` = intersection). Do not Join arm post-types into the caller type unless those facts are universal.

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

```text
φ = (kind=login ∧ Min(user,1) ∧ Min(password,8))
  ∨ (kind=token ∧ Min(token,20))
```

No third arm: other `kind` values → false.

## 4. Why this is occurrence typing, not “we allowed if so we allowed logic”

The condition **is** an atom. The arm **is** a conjunction (`All`). The chain **is** a Join (`Any`). Nested `if` stays nested in the `GuardExpr` tree. **Do not distribute into DNF.**

That is the “restricted in just the right way” answer: **the restriction is the grammar of tests**, not a vibe. The moment `if x.role == "admin"` is legal in a guard, `φ` is no longer in the atom language and TG-5 is theatre.

## 5. Why no loops

A loop in a predicate is either:

- **Bounded scan the compiler cannot see** (`for i := 0; i < n; i++` with `n` runtime) → `φ` would need a quantifier `∀i`. That is a jump from QF atoms to quantified logic (SMT).
- **Invariant-shaped “all elements satisfy P”** → that is a **builtin** (`Array` of `T.P`, or a future `All(P)` constraint), not a user `for`.

“All cells are `"" | "X" | "O"`” is a **literal union on the element type**, or a builtin on arrays, not a type-guard loop. tictactoe’s `ValidBoard` as `Min(9)`/`Max(9)` is the honest current fragment; a loop over cells would be the trap.

## 6. Composition of named guards

```ft
is (p Password) VeryStrong {
  ensure p is Strong()
  ensure p is HasUppercase()
  ensure p is HasNumber()
}
```

`φ_VeryStrong = All(Strong, HasUppercase, HasNumber)` as names, not an inlined DNF of `Strong`’s body. Recursion (`is (x T) G { ensure x is G() }`) is rejected. Mutual recursion is rejected.

**Join of named guards:**

```ft
is (p Password) Acceptable {
  ensure p is Strong() or Passkey()
}
```

Not:

```ft
is (p Password) Acceptable {
  if p is Strong() {}
  else if p is Passkey() {}
}
```

Empty arms are how you confuse authors. `or` is the spelling of “either named formula.”

## 7. What we tell users (without teaching DNF)

- List of `ensure` = all must hold.
- `is A() or B()` = either holds.
- `if … is … { ensure … }` = **cases**. If none match, the guard fails. Like a `switch` that does not succeed by falling out.
- Need a loop or `==`? Write a **function**. It will not narrow. That is the feature.

## 8. Lowering sketch

For a DNF with arms `c_i`:

```go
func G_ValidMessage(m T) bool {
  if m.Kind == "login" {
    if len(m.User) < 1 { return false }
    if len(m.Password) < 8 { return false }
    return true
  }
  if m.Kind == "token" {
    if len(m.Token) < 20 { return false }
    return true
  }
  return false
}
```

Conjunctive playlists stay a sequence of `if !atom { return false }`; final `return true` is correct **only when there is no residual unmatched `if`**.
