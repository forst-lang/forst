# 09 — Nominal proxies (export rule)

**Status:** The **export / abstraction-boundary** rule is **accepted** ([12 §18–19](./12-accepted-decision.md)). Call-site `ensure` **may** contain assertion `or` ([12 §5](./12-accepted-decision.md)). `|` in `is` assertions is **not** used.

**Kept:** After `ensure x is G()`, callers know `G`, plus facts true on every successful path. Nested guards are fine. Flattening names under `∨` into a conjunctive chain is unsound.

**Not kept:** Forbidding `ensure phone is HasPrefix("+") or HasPrefix("0") else InvalidPhone()`.

The original analysis of nested guards and imports follows; read historical “call-site `|`” bans as “call-site `or` is allowed.”

---

**Idea under test:** Call-site `ensure` stays a single nominal atom (`Min`, `LoggedIn`, `GameStatus`). Disjunctive predicates exist **only in type-guard bodies**. The checker does not unfold those bodies into a DNF of underlying conditions. It records the **chain of names** the author wrote (`ensure x is Strong()`, `ensure x is HasUppercase()`) and uses those names as **proxies**. Runtime still executes the real `φ`, including `or` / fail-closed `if`. A type guard does not have to *be* a sophisticated type. It has to be a **checked brand**.

---

## 1. What this actually claims

Two layers:

| Layer | What it is | Who sees it |
| --- | --- | --- |
| **Implementation** (guard body) | May contain Join (`or`, restricted `if`). Lowered to `G_* : T → bool`. | Compiler emit, runtime |
| **Interface** (use site) | After `ensure x is G()`, `x` has the **name** `G` (and the conjunctive names `G` **exports**). Not the DNF of `G`’s internals. | Typechecker, hover, assignability |

This is an **abstraction barrier**, the same move as an abstract predicate in separation logic or a sealed brand: callers know `G`, not `φ_G`. Nested guards are calls across that barrier, not inlining.

Call-site grammar is one place plus constraint alternatives (`or`). Historical “no call-site `|`” is withdrawn; [12 §5](./12-accepted-decision.md) allows:

```ft
ensure ctx is LoggedIn() else Unauthorized()
ensure raw is GameStatus else BadStatus()     // named type, still one atom
ensure p is PhonePrefix() else BadPhone()     // named guard that *hides* a Join
ensure p is HasPrefix("+") or HasPrefix("0") else BadPhone()
```

That Join lives only in `PhonePrefix`’s body.

## 2. Nested guards are why this works — if you do not conjunctivize `∨`

Guards already call guards:

```ft
is (p Password) Strong {
  ensure p is Min(12)
}

is (p Password) VeryStrong {
  ensure p is Strong()
  ensure p is HasUppercase()
}
```

**Export of `VeryStrong` (conjunctive body):** the name chain `{VeryStrong, Strong, HasUppercase}`. Optionally still the builtin `Min(12)` if we keep today’s merge for **conjunctive** builtin atoms (that is LoggedIn / `Present`). We do **not** need to look *inside* `Strong` to know the name `Strong` was required. Runtime: `G_VeryStrong` calls `G_Strong`, which checks `Min(12)`. Everything is checked. The proxy **is** what the author wrote: “VeryStrong means Strong and HasUppercase.”

Now the disjunctive case:

```ft
is (p Password) Acceptable {
  ensure p is Strong() | Passkey()
}
```

**Sound export:** `{Acceptable}` only. Callers know `Acceptable`, not Strong, not Passkey.

**Cursed export:** `{Acceptable, Strong, Passkey}` as a **conjunctive** chain. Then the checker believes Acceptable ⇒ Strong ∧ Passkey. Runtime only proved Strong ∨ Passkey. **Unsound.** That is the entire curse. It is not “guards calling guards.” It is **flattening a Join into a Meet of names**.

Same bug with restricted `if`:

```ft
is (m Shape) ValidMessage {
  if m.kind is Value("login") {
    ensure m.user is Min(1)
  } else if m.kind is Value("token") {
    ensure m.token is Min(20)
  }
}
```

Export `{ValidMessage}` as a brand. Do **not** export `Min(user)` and `Min(token)` as if both fields were proven. They were proven on **different arms**.

**Rule (normative for this idea):**

> Names and builtin atoms are added to a guard’s **exported Meet** only if they hold on **every successful path**. Anything that holds on some paths only stays behind the guard’s own name. Disjunction is never copied into the export as a list of names.

Nested calls are then boring:

- `VeryStrong` conjunctively ensures `Strong` → export includes `Strong`. `Strong` may hide a Join; callers of `VeryStrong` still do not see that Join. They see `Strong`, which is what the author conjunctively demanded.
- `Acceptable` disjunctively mentions `Strong | Passkey` → export is `Acceptable` only. Callers cannot pass `Acceptable` where `Strong` is required. Correct: an Acceptable value might be a Passkey.

No truncation of a DNF. There is **no DNF at the use site**. There is a Meet of names that were **universal** in the body, plus the guard’s own name.

## 3. Does the type need to be sophisticated?

No. Under this idea `T.G` is a **nominal refinement / brand** plus whatever **conjunctive** facts we still export (field `Present`, `Min` on the same path, other guard names required on all paths).

That is enough if the product goal is:

1. Runtime: the wire value was actually checked (`G_*` ran, including internal `|`).
2. Static: you cannot forget the check (`ensure x is G()` or the type is already `T.G`).
3. Static: conjunctive structure the author wrote as a **playlist of names** is visible (`VeryStrong` implies `Strong`).

That is **not** enough if the product goal is:

1. After `PhonePrefix`, treat the value as `HasPrefix("+") | HasPrefix("0")` (Join in the type).
2. After `ValidMessage`, have a discriminated union of two shapes (Join of records).
3. After inline `ensure x is A() | B()`, skip naming a guard.

Those are [06](./06-recommendation.md)’s extra precision. They need Join in the **type**, not only in `G_*`. This document says: you can ship runtime disjunction **without** that precision, and it is coherent.

LoggedIn still wants conjunctive builtin export (`Present` on two fields). That is the **playlist** case, already implemented. This idea does not take that away. It refuses to pretend a disjunctive body is the same kind of playlist.

## 4. Relation to unfolding ([04](./04-guard-bodies.md) / [06](./06-recommendation.md))

| | Unfold `φ` (06) | Nominal proxy (this doc) |
| --- | --- | --- |
| Call-site `ensure … is A \| B` | Yes | **No** — name a guard (or a typedef) |
| Guard body `|` / `if` | Yes, becomes Join in `φ_G` | Yes, **runtime only** |
| Use-site type | `G` **and** unfolded `φ_G` | `G` **and** conjunctive exported names only |
| `Acceptable` vs `Strong` | Can see `Strong ∨ Passkey` | `Acceptable` ≰ `Strong` unless you unfold |
| DNF cap ([08](./08-complexity-bounds.md)) | Needed at definition unfold | Needed only if you later unfold; v1 can skip |
| Hover | Can show the or | Shows the name; the or is “see definition” |
| Nested guards | Inline under size cap | **Names as proxies; do not inline** |

They compose: a later compiler can start attaching `φ_G` for hover without changing runtime. The trap is attaching a **wrong** `φ` (Meet of names that were Joined).

## 5. Enums still should not be type guards

`type GameStatus = "playing" | …` is a **type**, one atom at `ensure raw is GameStatus`. That is not call-site `|` of two constraints. This idea does not force GameStatus to be `is (s String) GameStatus { ensure s is Value("playing") | … }`. Doing that would make every status a brand with no TS string-union emit unless you special-case it. Keep [02](./02-three-kinds-of-union.md) kind 1 as types.

Optional: a typedef Join of `Value` is a named type; `ensure x is PhoneNumber` where `PhoneNumber` is the typedef with `|` is still **one nominal atom** at the call site. The Join lives in the **type definition**, which is the same abstraction barrier (name vs body) as a type guard. Fine. What this idea forbids is **anonymous** `|` in `ensure`.

## 6. What the typechecker does (operational)

When checking `is (x T) G { body }`:

1. Typecheck `body` under TG-1–TG-7 (and fail-closed `if` — [00 §8](./00-the-trap.md) still applies: unmatched must not `return true`).
2. Compute `Export(G)`:
   - Always include `G`.
   - For each **sequential** `ensure x is A()` (and field-path conjunctive ensures): add `A` and, if `A` is a user guard, **do not** add `Export(A)` unless you want one-level name inlining of **Meets only**. Safer v1: add the name `A` only, not `Export(A)`. Then `VeryStrong` exports `{VeryStrong, Strong, HasUppercase}` and a use site that needs `Min(12)` still has to go through `Strong` or you one-level-inline conjunctive builtins from `Strong` only.
   - For `ensure x is A() | B()`: add **nothing** from `A`/`B` to `Export(G)`.
   - For `if` arms: `Export` = Meet of (what every successful arm exports). If arms export incomparable field facts, Meet is empty besides `G`.
3. Store `Export(G)`. Do not store a Join formula unless you are implementing 06 later.

When checking `ensure x is G()` in a function:

1. Runtime/emit: call `G_*` (which recursively calls nested `G_*`).
2. Static: Meet `x`’s type with `{G} ∪ Export(G)` (minus the duplicate `G`).
3. Do not walk `G`’s AST looking for `|`.

**One-level inline of conjunctive builtins** (`Strong` → also tag `Min(12)`) is optional sugar so hover matches today. It must **stop** at the first Join in `Strong`’s body.

## 7. Imports: the names you rely on stay public

This idea does **not** mean “any guard we rely on must be unexported / not reused.” That would be the wrong reading of the barrier.

A type guard is a **declaration**, like a function. It is usable wherever a single `is` atom is legal: other guards, function `ensure`, `if x is G()`, parameter types `T.G`, other packages. [guard.md — Module Boundaries](../guard/guard.md) already wanted that. Nominal proxies **require** it: the use site only needs the **name** (and `Export(G)`), not `G`’s AST.

| Thing | Importable / reusable? |
| --- | --- |
| `LoggedIn`, `Strong`, `PhonePrefix`, `ValidMessage` | **Yes.** That is how handlers `ensure ctx is auth.LoggedIn()`. |
| Nested use `ensure p is Strong()` inside `VeryStrong` | **Yes**, including `ensure p is auth.Strong()` from another package. |
| The **anonymous** Join `HasPrefix("+") \| HasPrefix("0")` at a call-site `ensure` | **No** under this idea — not because `HasPrefix` is private, but because Join is not a call-site assertion. `HasPrefix("+")` alone remains a normal builtin atom. |
| `Strong` and `Passkey` after you only have `Acceptable` | Independently **yes**. `Acceptable` does not imply them. You `ensure is Strong()` if you need Strong. |

Across packages the checker **must not** depend on inlining the imported body. Package B sees `auth.Strong` as an atom whose runtime is `G_Strong` in package A (or inlined at link/emit, irrelevant to types). `Export(auth.Strong)` can travel as a small list of names / field facts next to the symbol. If that list is missing, B still has the brand `Strong`. That is weaker, not unsound, and it is exactly how imported predicates should work.

**Unexported guards** are a **style** choice (helper used only inside one disjunctive body), same as an unexported `func`. Soundness does not require it. Hiding `Passkey` does not make `Acceptable` imply `Strong`.

What you **cannot** do from another file is see through a Join you did not name: after `ensure x is auth.Acceptable()`, you do not get `Strong` for free, whether `Strong` is public or not. Public `Strong` stays usable on its own.

## 8. Failure modes (when people will say it is cursed)

| Situation | Cursed if | Fine if |
| --- | --- | --- |
| `Acceptable` = `Strong \| Passkey` | Export both names | Export `Acceptable` only |
| `ValidMessage` tag split | Export both arms’ fields | Export the name; fields stay unrefined at use site |
| `VeryStrong` → `Strong` | Forget to export `Strong` (too weak, not unsound) | Export the conjunctive names the author wrote |
| Deep nesting `A`→`B`→`C` with Join inside `C` | Unfold `C` into `A`’s export | `A` only sees `B`; `B` only sees `C` |
| Call-site wants `HasPrefix("+") \| …` without a name | User cannot write it | User writes `PhonePrefix` — the point |
| Expect `T.G` to be a TS discriminated union | It will not be | Use a typedef union or wait for 06 Join |

Too-weak (not unsound) is acceptable for v1. Unsound flattening is not.

## 9. Verdict

**Works.** It is the right description of what Forst mostly does already: names on the symbol, `G_*` at runtime, nested `ensure is H()` as a name in the chain. Disjunction belongs **behind** a name so the chain stays a Meet.

**Not cursed by nested type guards.** Nested guards are the mechanism that *prevents* the trap: the inner Join never becomes a list of outer facts.

**Cursed only if** `Export` is “all names mentioned in the body.” Mention ≠ hold on all paths.

**Precision you give up vs [06](./06-recommendation.md):** no anonymous `|` in `ensure`; no use-site Join type for `PhonePrefix` / `ValidMessage`. Runtime is as strong. Static is a brand plus a conjunctive name playlist.

**Recommendation relative to 06:** ship **this** as the typechecker’s **use-site** model (do not unfold Joins into caller types). Ship **fail-closed `if` and `|` in guard bodies** for runtime. Keep **literal unions as types** for enums. Treat 06’s assertion-position `|` at call sites and unfolded `φ_G` as a **later precision** upgrade, not a prerequisite for disjunctive checks actually running.

That matches the end-consumer sentence: “If it is an or, give it a name. `ensure` lists names. The guard is what runs the or.”
