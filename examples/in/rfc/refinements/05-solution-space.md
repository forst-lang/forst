# 05 — Solution space

**Status:** Comparison. **Accepted mix is [12](./12-accepted-decision.md).** This table is historical. Assertion Join is **`or`**; typed failure is **`else`**; **`|`** is types. Historical rows that show `|` inside `is` or `or` as failure are not the accepted spelling.

Scoring is for the **end consumer** (handler author, TS client, Go emit) **and** for staying in the fragment of [00](./00-the-trap.md).

Legend: **Yes** / **Partial** / **No**. **Trap** = this is the slope in [00 §5](./00-the-trap.md).

---

## 1. Options

### A — Status quo

Sequential `ensure` only (conjunction). `if` in guards exists but unmatched succeeds. No assertion `|`. Enums are comments (`String` + hope).

- Expressiveness: **No** (the complaint that started this RFC).
- Analyzable: **Partial** (conjunction is fine; `if` is unsound).
- Teaching: **Partial** (simple until you need `or`).

### B — Opaque `return bool` (early guard RFC examples)

```ft
is (p Password) Strong { return len(p) >= 12 }
```

- Expressiveness: **Yes** (any predicate).
- Analyzable: **No**. `φ` is opaque. Narrowing is a **brand**. Field presence after `LoggedIn` cannot be extracted unless you lie.
- Soundness: **No** unless you trust the author (TypeScript `x is T`).
- Trap: **Yes** — step 1 of the slope.

### C — Boolean `ensure` / `||` / `==`

```ft
ensure x == "a" || x == "b" or err
```

- Collides with `or err` and with TG-2.
- Analyzable: **No** (arbitrary expressions).
- Teaching: **No** (two kinds of `ensure`).
- Trap: **Yes**.

### D — Reuse `or` for logical or

```ft
ensure x is Foo() or Bar()   // Bar is currently an error
```

- Breaks [errors 02](../errors/02-first-class-errors-normative.md) and every existing file.
- Teaching: **No**.
- Trap: syntax-level.

### E — Assertion and typedef `|` (Join in the existing algebra)

```ft
ensure x is HasPrefix("+") | HasPrefix("0") or BadPhone()
type Phone = String.HasPrefix("+") | String.HasPrefix("0")
```

- Expressiveness: **Yes** for kind 2 ([02](./02-three-kinds-of-union.md)).
- Analyzable: **Yes** (Join of atoms).
- Teaching: **Yes** if `or` stays failure ([03](./03-or-vs-pipe.md)).
- Go emit: **Yes** for one-carrier refinements.
- Fits Meet/Join already on typedefs: **Yes**.

### F — Literal / singleton unions as the enum

```ft
type GameStatus = "playing" | "x_won" | "o_won" | "draw"
```

- Expressiveness: **Yes** for kind 1.
- Analyzable: **Yes** (finite set inclusion).
- TS emit: **Yes** (this is TS’s good feature).
- Teaching: **Yes** (the tictactoe comment becomes code).
- Does not require type guards: **Yes**.

### G — Builtin `OneOf` / `Equals` only

```ft
ensure op is OneOf(["sum", "average"]) or InvalidInput()
```

- Expressiveness: **Partial** (enums only, not PhoneNumber, not tagged shapes).
- Analyzable: **Yes**.
- Teaching: **Partial** (third vocabulary next to `Value` and `|`).
- Verdict: **subsumed by F + E**. Sidecar sketches can migrate to `Value` unions.

### H — Restricted `if` in guards as Join of paths, unmatched fails

[04](./04-guard-bodies.md).

- Expressiveness: **Yes** for kind 3 (discriminated shapes).
- Analyzable: **Yes** iff TG-2 holds and DNF is capped.
- Teaching: **Partial** — must document “cases fail closed,” not “if like Go.”
- Required anyway: **Yes**, to fix implicit `return true`.

### I — Unrestricted `if` / functions in guards, still no loops

`if x.role == "admin"`, helper calls, local `:=`.

- Expressiveness: **Yes**.
- Analyzable: **No**.
- Trap: **Yes** (step 3 without SMT — you silently stop extracting).

### J — Loops in guards / quantified `φ`

- Trap: **Yes**. Quantifiers. SMT or unsound.

### K — Liquid / SMT refinements

User writes `{x: Int | x > 0 ∧ gcd(x,3)==1}` or the checker proves `Min(12) ⇒ Min(10)` in general arithmetic + UDFs.

- Expressiveness: **Yes**.
- Compile-time: **No** (unpredictable).
- Teaching: **No** (“could not prove”).
- PHILOSOPHY / SEMANTICS_NARROWING: **forbidden**.
- Trap: **Yes** (the destination of the slope).

### L — Full dependent types

`Vec(n)`, types contain terms.

- PHILOSOPHY: **forbidden**.
- Product: Forst is not Agda.

### M — TypeScript-style user `x is T` plus CFA only

Put all unions in the type grammar (F + discriminated typedefs), keep user guards **opaque**.

- Analyzable user guards: **No**.
- Product: **Partial** — we would throw away LoggedIn field extraction, which is Forst’s point vs TS.
- Verdict: do **F** like TS; do **not** do opaque guards like TS.

### N — Separate runtime schema language (Zod / `OneOf` DSL)

Types and validators diverge.

- Teaching: **No** (two worlds). Forst’s pitch is one `is`/`ensure` pipeline.

### O — Go `iota` / const enums only

- JSON string status: **No**.
- TS string unions: **No**.
- Keep `iota` for Go ports; it is not this RFC.

### P — Wait for `match`

- `match` needs this algebra first ([switch-match](../switch-match/00-switch-and-match.md)).
- Blocking enums on `match` **No**.

### Q — Nominal brands only (`T.G` is a name, body uninterpreted)

- Implementation cheap.
- LoggedIn present-fields: **No**.
- Worse than A for the documented examples.

### R — `&` / `|` only at typedef, never in `ensure`

Authors must name every Join. Phone prefix inline becomes a typedef forever.

- Teaching: **Partial**. Extra names are fine for GameStatus, noisy for a one-off.
- Verdict: typedef remains the **reuse** form; assertion `|` is the **local** form.

### S — “Opaque” opt-in guard (`is … G opaque { return … }`)

Escape hatch for checksums.

- v1: **No**. Users will mark everything opaque. Revisit only after E/F/H are taught.
- If ever: different keyword so `φ` is clearly uninterpreted; still no narrowing of fields.

---

## 2. Matrix (consumer + theory)

| Option | Enums | P∨Q on one carrier | Tagged shapes | `ensure` disjunction | Stays decidable | Fits `or` = error | End-user clarity |
| --- | --- | --- | --- | --- | --- | --- | --- |
| A status quo | No | No | Broken if | No | Partial | Yes | Low once they need or |
| B opaque return | Hack | Hack | Hack | Hack | No | Yes | False (looks easy) |
| C boolean ensure | Partial | Partial | No | Mis-spelled | No | No | Low |
| D `or` as logic | — | — | — | Ambiguous | — | **No** | **No** |
| **E assertion `\|`** | Via F | **Yes** | No | **Yes** | **Yes** | **Yes** | **High** |
| **F literal unions** | **Yes** | Literals only | No | `ensure is Status` | **Yes** | **Yes** | **High** |
| G `OneOf` builtin | Yes | No | No | Yes | Yes | Yes | Medium (extra name) |
| **H restricted if** | Wrong tool | Worse than E | **Yes** | Via E inside arms | **Yes** if capped | **Yes** | Medium (fail closed) |
| I loose if | Hack | Hack | Partial | — | No | Yes | Low |
| J loops | — | — | All-elements | — | No | Yes | Low |
| K SMT | Yes | Yes | Yes | Yes | No in practice | Yes | **No** |
| L dependent | — | — | — | — | No | — | No |
| M TS opaque guards | If F | If F | If unions | `if` CFA | Guards no | Yes | Medium |
| N Zod split | Yes | Yes | Yes | Schema | Runtime only | — | No |
| O iota | No (strings) | No | No | No | Yes | Yes | Wrong domain |
| P wait for match | Delay | Delay | Delay | Delay | — | — | Delay |
| Q brands only | Name | Name | Name | Name | Yes | Yes | Low (lies about fields) |
| R typedef-only `\|` | Yes | Named only | Named only | Only via type name | Yes | Yes | Medium |
| S opaque opt-in | Escape | Escape | Escape | Escape | By refusing | Yes | v1 footgun |

---

## 3. Combinations that actually work

The only combination that hits all three unions **and** the trap **and** existing `or` is:

**F + E + H**, with **A’s conjunctive `ensure` kept**, **B/C/D/I/J/K/L/N/S out**, **G subsumed**, **O kept as Go interop unrelated**, **P later**, **R as the named form of E**.

That is the recommendation in [06](./06-recommendation.md).

## 4. What we refuse even if users ask twice

1. `ensure` of a boolean expression.
2. `or` meaning logical or.
3. `return` in type guards.
4. Guard conditions that are not `is`.
5. Loops / quantifiers in guards.
6. SMT or “just use Z3.”
7. Encoding enums as type guards.
8. Making `if` in guards succeed when no arm matches.
