# 10 — Named predicates do not make type unions moot

**Status:** Answer to “if we only have nominal type guards, can we drop `A | B` from the language?” Scope is **predicates only** (no effects, no monads). Related: [02](./02-three-kinds-of-union.md), [09](./09-nominal-proxies.md).

---

## 1. The instinct that is right

Reused **or**-of-checks should have **one name**. Do not scatter

```ft
ensure p is HasPrefix("+") or HasPrefix("0") else BadPhone()
```

through handlers if the same Join is the domain rule. Write `PhonePrefix` once, `ensure p is PhonePrefix() else BadPhone()` everywhere. That is ordinary functional taste applied to **predicates**: small named tests, composed with Meet (sequential `ensure`) and with Join **hidden behind a name** ([09](./09-nominal-proxies.md)).

For **that** fragment — refinements of **one carrier**, reused — anonymous assertion `or` of constraints is optional sugar. The language can push authors toward a typedef or a type guard and still be coherent. Call-site `or` is **allowed** ([12 §5](./12-accepted-decision.md)); `|` is types. Unions are **not** what makes PhonePrefix work. The **name** is.

That does **not** scale to “the entire language has no type unions.”

## 2. A predicate is not a type in a signature

A type guard `is (x T) G` is a **test on an existing `T`**. It produces a brand `T.G` and a `G_*` function. It answers: “may I treat this `T` as having passed `G`?”

A union `A | B` is a **type constructor**. It answers: “what values may appear **here**?” — field, parameter, `Result` failure, return.

You can always fake the second with the first if you pick a **top** carrier (`Shape`, `Error`, `any`) and a guard `Valid`. Then every signature is `Shape` / `Error` and every check is `ensure x is Valid()`. That is **boolean blindness** one level up: **brand blindness**. The runtime still branches; the type at the boundary no longer says *which* alternatives exist. Callers cannot `if x is ParseError` with exhaustiveness. TS emit becomes `unknown` or a brand. Go emit becomes `any` / `error` without a sealed set.

The FP move you want is **name the predicate**. The FP move that wrecks backends is **encode every sum as a predicate on a blob**.

## 3. What `|` is for when it is not a predicate Join

From [02](./02-three-kinds-of-union.md):

| Kind | Predicate on one `T`? | If we only have named guards |
| --- | --- | --- |
| `HasPrefix("+") ∨ HasPrefix("0")` | **Yes** | `is (p String) PhonePrefix { … }`. Union **moot** at call sites. |
| `"playing" \| "x_won" \| …` | Membership test, but the interesting object is the **set** | A guard `GameStatus` is a brand on `String`. Field type `status: String` plus `ensure is GameStatus` is the tictactoe hack. Field type `status: GameStatus` needs **`GameStatus` to be a type**, not only a test. Literal union **or** a typedef alias. Not “only a guard.” |
| `{kind: login, …} \| {kind: token, …}` | No. Different fields. | Guard `ValidMessage` on `Shape` checks at runtime. After `ensure m is ValidMessage()`, `m.token` is not known to exist unless you **split** again. The split is occurrence typing on a **union of shapes**, or another `if m.kind is …`. The union is the type of the value; the guard is the validator. |
| `ParseError \| IoError` | Predicate on `Error` possible | `Result(Int, LoadError)` needs `LoadError` as a **type** so callers and TS/Go know the closed set. A guard `is (e Error) LoadError` does not fill a type parameter. |
| `Result(S, F)` as Ok \| Err | Built-in sum | `is Ok()` / `is Err()` are **guards on that sum**. Delete the sum and there is nothing to guard. |
| `T \| Nil` | Present / Nil are guards | The type is the sum. Guards discriminate it ([optionals](../optionals/README.md)). |
| Join after `if` / `else` | Not a user predicate | Continuation type is `T_then ∨ T_else` ([SEMANTICS_NARROWING](../../../../forst/internal/typechecker/SEMANTICS_NARROWING.md)). If the language has no Join, merge stays the pre-if type forever. |

So: **predicate Join can be named away. Sums, optional, Result, error sets, control-flow merge, and field types cannot.** Those uses of `|` are not “conditions in user code.” They are the **shape of data**.

## 4. “As few conditions as possible”

Agreed, for **checks**:

- Boundary: one `ensure x is NamedRule() else Err`.
- NamedRule’s body: the or, written once.
- Callers: no `|`, no duplicated prefix tests.

That is not the same as **as few types as possible**. The status **field** should still be `GameStatus`, not `String` plus a ritual `ensure` at every read. Putting the set in the **type** removes conditions from code. Putting it only in a guard **adds** conditions at every boundary where the type was `String`.

Literal unions are how you get **fewer** `ensure`s, not more. Named guards are how you get fewer **duplicated** `ensure`s for messy predicates. Use both.

## 5. Subtyping: names vs sets

Structural union: `"a" | "b"` <: `"a" | "b" | "c"` (set inclusion). Cheap, matches JSON, matches TS.

Nominal guards: `String.GameStatus` and `String.GameStatusOrPaused` are **incomparable** unless you conjunctively export or write a third name that Meets/Joins them by hand ([09](./09-nominal-proxies.md) export rules). That is fine for `LoggedIn`. It is painful for enums. Deleting `|` and using only brands **throws away** the one structural theory that is decidable and user-obvious (finite sets).

## 6. Would “no unions, only guards” make 09 simpler?

Slightly, for the PhonePrefix story. You would still need:

- `T.G` as a type (the brand) — that is already a type, derived from a predicate.
- Some way to write **fields and parameters** of that type (`status: GameStatus`). If `GameStatus` is only `is (s String) GameStatus`, Forst already allows `String.GameStatus` as a refinement type. That is “the guard induces a type,” not “unions are gone.”
- For two **different** named types (`ParseError | IoError`), inducing a guard on `Error` does not give you a sealed failure type for `Result`.

You would **not** get to delete binary types from typedefs, Result, or narrowing join. You would reimplement them badly as `is (x Top) SomeName`.

## 7. Verdict

| Proposal | Verdict |
| --- | --- |
| Reused predicate Joins are **named** (guard or typedef), call-site `ensure` is one atom | **Yes.** [09](./09-nominal-proxies.md). That is not anti-union; it is anti-**anonymous** predicate or. |
| Users should write as few **conditions** as possible by naming predicates once | **Yes.** |
| Therefore **type unions in general** are moot | **No.** Predicates ≠ sums ≠ literal sets in field position ≠ `Result` ≠ error sets ≠ if-join. |
| Delete `A \| B` from the language and use only type guards | **No.** Brand-blind `Shape`/`Error` APIs, worse TS/Go, no exhaustiveness, weaker enums. |
| Literal unions stay types; messy P∨Q stays named guards; tagged JSON may be a guard for **validation** and a union type for **the value** | **Yes.** |

The functional instinct applies to **tests**: name them, reuse them, keep the algebra to Meet of names at use sites. It does not apply to **what a function returns**. `load(): Result(Int, ParseError | IoError)` is not a predicate you forgot to name. It is the contract.
