# Prior art: `Result`-style success/failure construction in other languages (analysis)

**Audience:** Language designers comparing **how values** enter a **success vs failure** channel—**with** or **without** explicit **`Ok`/`Err`-style constructors—and how that pairs with **errors defined “implicitly”** (factories, literals, validation).

**Status:** **Reference / analysis** — not normative for Forst syntax. Informs [12 — Result primitives without `Ok`/`Err`](./12-result-primitives-without-ok-err.md).

**Relations:** [02-result-and-error-types.md](./02-result-and-error-types.md), [errors architecture](../errors/00-error-system-architecture.md), [09](./09-ensure-is-narrowing-and-binary-types.md) (`ensure`), [PHILOSOPHY.md](../../../../PHILOSOPHY.md).

---

## 1. Summary table

| Language / ecosystem | How success/failure is introduced | Explicit `Ok`/`Err`-like wrappers? | Notes on “implicit” errors |
| --- | --- | --- | --- |
| **Rust** | `Ok(x)` / `Err(e)` for `Result<T,E>`; `?` propagates | **Yes** — enum variants | Errors are **typed** `E`; no implicit error type without defining `E` |
| **Swift** | `Result` enum with `.success` / `.failure` | **Yes** — case constructors | `throw` is separate; `Result` is **pure** sum for non-throwing APIs |
| **Kotlin** | `Result.success` / `Result.failure`, or stdlib helpers | **Yes** (static factories) | Exceptions can be wrapped; **implicit** exception→`Result` is **not** default |
| **Haskell** | `Right` / `Left` for `Either e a` | **Yes** | `Left` is **not** error-specific—**semantic** convention |
| **OCaml / F#** | `Ok` / `Error` (`Result`) or `Some`/`None` (`option`) | **Yes** | `result` type is stdlib-shaped like Rust |
| **Go** | `return v, nil` / `return zero, err` | **No** — **positional** convention | **Maximum implicitness** on the **success** side; **failure** is **any** `error` value |
| **Zig** | `error.Name` and `!T` error unions; `try` | **No** `Ok`/`Err` — **errors are values** in a set | **Error sets** are **declared**; very explicit **error names**, less **wrapper** noise |
| **TypeScript** (idiomatic) | `Promise` rejection; or `{ ok: true, value } \| { ok: false, error }` | **Often yes** in hand-rolled unions | **Discriminated unions** repeat `ok` flags; libraries (**neverthrow**, **fp-ts**) use **constructors** |
| **Effect (`effect` / TS)** | `Effect.succeed` / `Effect.fail`; **`Effect.gen`** + **`yield*`** | **Yes** for primitives; **generators** hide **some** repetition | **Three** type params **`A,E,R`**; **lazy** runtime; see **§5** |

---

## 2. Patterns in more detail

### 2.1 Explicit sum constructors (Rust, OCaml, Swift `Result`)

**Pattern:** A **nominal** or **algebraic** type with **two** data constructors—**success** and **failure**—spelled **`Ok`/`Err`** or **`.success`/`.failure`**.

**Pros:** Unambiguous in **source** and **types**; pattern matching / `match` is regular.  
**Cons:** **Every** return repeats the **wrapper**; authors feel **clutter** (the motivation for [12](./12-result-primitives-without-ok-err.md)).

### 2.2 Positional multi-return (Go)

**Pattern:** **`(T, error)`**; **success** = **non-nil T + nil error** (usually); **failure** = **non-nil error**.

**Pros:** No **`Ok`/`Err`** tokens; **failure** is **any** value satisfying **`error`**.  
**Cons:** **Positional** discipline; **easy to misuse** (`nil` both, wrong order); **less** static **failure typing** unless conventions or linters help.

### 2.3 Error unions without `Ok`/`Err` (Zig)

**Pattern:** Values of type **`!T`** combine **`T`** with a **set of named errors**; **`error.Name`** introduces failures; **success** is often **plain** `return x` when the function returns **`!T`**.

**Pros:** **Low wrapper noise** on success; **errors** are **first-class names** in the type system.  
**Cons:** Different mental model from **`Result(S,F)`** with arbitrary **`F`**; not a drop-in for Go’s **`error`** interface story.

### 2.4 Exceptions + optional `Result` (Swift, Kotlin, Java)

**Pattern:** **`throw`** for control flow; **`Result`** used when **not** throwing or when **bridging**.

**Pros:** **Short** happy paths.  
**Cons:** **Implicit** jumps—**explicit errors** are a **Forst non-goal** at language level ([PHILOSOPHY.md](../../../../PHILOSOPHY.md)); Forst stays closer to **Go + typed `Result`**.

### 2.5 Library-level constructors (TypeScript, FP)

**Pattern:** **`ok()`** / **`err()`** from **neverthrow**, **`Either.right` / `left`**, etc.

**Pros:** **Composable**, **typed**.  
**Cons:** **Imports** and **wrapper** ceremony; same **clutter** objection at scale.

---

## 3. “Implicit” error values: what other systems do

Several languages **avoid repeating** a **failure constructor** by making **errors** cheap to **introduce** in other ways:

| Mechanism | Example idea | Tradeoff |
| --- | --- | --- |
| **String / fmt errors** | `errors.New("…")`, `fmt.Errorf` | Fast to write; **weak** typing |
| **Sentinel errors** | Package-level `var ErrFoo = …` | Shared identity; still **manual** defs |
| **Type-specific factories** | `ParseError { field, … }` | **Structured**; need **one** place per type |
| **Validation hooks** | Schema / `ensure`-like checks that **produce** domain errors | **Implicit** in the sense “**you don’t name the `Result` branch**”—you **name the rule** |

Forst’s direction combines:

- **`ensure`** — **runtime** check + **narrowing**; failure path can **return** or **bind** an **`error`** without writing a full **`Result`** combinator story at every site ([09](./09-ensure-is-narrowing-and-binary-types.md)).  
- **Error types and factories** — **domain** errors as **named shapes** ([errors architecture](../errors/00-error-system-architecture.md)); call sites can write **`SomeError(...)`** or similar **constructors** that **stand in** for “**make a failure value**” **without** necessarily wrapping every return in **`Err(...)`** if the **language** later allows **`return thatError`** or **`return expr`** with **inferred** branch.

That **separation**—**guards** (`is Ok` / `ensure … is …`) **vs** **how the failure value is built** (`ParseError(…)`, **`fmt`**, **`ensure` failure**)—is **aligned** with **“implicit is good”** in the **sense of not repeating `Err()`**, **not** in the sense of **hidden** control flow.

---

## 4. Implications for Forst (non-binding)

1. **Keeping `Ok`/`Err` as guards** ([12 §0](./12-result-primitives-without-ok-err.md#0-scope-split-guards-vs-constructors)) matches **Rust/Swift/OCaml**-style **discriminants** for **narrowing**, independent of whether **returns** use **`Ok(x)`**.  
2. **Avoiding constructors** at **return** sites is **closest in spirit** to **Go** (no wrapper) and **Zig** (plain success, **named** failures)—but Forst still wants **typed** **`Result(S,F)`** and **error-kinded** **`F`** ([02](./02-result-and-error-types.md)).  
3. **`ensure` + typed error factories** (`SomeError(…)`) is a **strong** story: authors **define errors** **locally** to the domain **once**, then **return** or **propagate** them with **less** **wrapper** noise if the **compiler** accepts **`return e`** where **`e: F`** and **`F`** is the **failure** type of **`Result(S,F)`**.

---

## 5. Deep dive: Effect (TypeScript, `effect` package)

The **[Effect](https://effect.website/)** ecosystem (successor to the **`fp-ts`** + **`@effect/io`** line) is the richest **TypeScript** reference for **typed success/failure** without **exceptions** as the default. It is **not** the same object as Forst’s **`Result(S,F)`** (compile-time **Go** lowering, **no** runtime interpreter by default), but it shows how **constructors**, **generators**, and **type-level** projection interact.

### 5.1 What `Effect` is (the type)

`Effect` values are **descriptions** of workflows: **lazy** computations that may **succeed**, **fail**, or require **services** before they can run. The core shape is **three** type parameters (see [Effect interface](https://effect-ts.github.io/effect/effect/Effect.ts.html) / project docs):

```text
Effect<A, E, R>
  A  — success value (“what you get if it works”)
  E  — failure / error type
  R  — required context (dependencies, layers, services)
```

- **`A`** is the **analogue** of Forst’s **success** side of **`Result(S,F)`**.  
- **`E`** is the **failure** side; unlike Rust’s unconstrained **`E`**, Effect code often uses **typed errors** (classes, tagged unions, **`Data.TaggedError`**, etc.), aligned with Forst’s **error family** direction ([02](./02-result-and-error-types.md)).  
- **`R`** has **no** direct analogue in **`Result(S,F)`** alone: it is **dependency injection** / **capabilities** (“this program needs a `Database` service”). That is **orthogonal** to the **`Ok`/`Err`** discussion but explains why **`Effect`** types look busier than a **binary** `Result`.

**`Effect.Effect`** is the **namespace + interface** for these values; documentation and **IDE** types usually show **`Effect.Effect<A, E, R>`** or the shorthand **`Effect<A, E, R>`** depending on import style.

### 5.2 Type-level `Success` (and projecting `A` / `E` / `R`)

Because **`Effect`** is generic in **`A`**, **`E`**, and **`R`**, the library exposes **type-level helpers** to **extract** those parameters from a **concrete** `Effect` type—for example **`Effect.Effect.Success<T>`** (name may vary slightly by **version**; check the **installed** `effect` package’s typings) to obtain **`A`** from **`T extends Effect.Effect<A, …>`**.

**Use case:** API signatures, **return types** of **`Effect.gen`**, and **layer** wiring without repeating **`A`** manually.

**Contrast with Forst:** Forst’s **`Result(S,F)`** is **two** parameters; **projecting** `S` and `F` is **structural** on the **type constructor**. If Forst later adds **type aliases** for “**success type of**” a **`Result`**, the **Effect** pattern is the **reference** for **ergonomic** type extraction.

### 5.3 Constructors: `Effect.succeed` and `Effect.fail` (not `Ok`/`Err`)

At the **value** level, the **default** success/failure constructors are:

- **`Effect.succeed(a)`** — **`Effect<A, never, never>`**-style (no failure, no requirements).  
- **`Effect.fail(e)`** — failure with **`e: E`**.

So Effect uses **`succeed` / `fail`**, not **`Ok` / `Err`**, but the **role** is the same as **Rust’s** variants: **explicit** tagging of **which** channel is used. Many **combinators** (`Effect.promise`, `Effect.tryPromise`, …) **desugar** to these **primitives** internally.

### 5.4 Generators: `Effect.gen` and `yield*`

**`Effect.gen`** takes a **generator function** and builds **one** `Effect` that **sequences** the inner steps:

- **`yield* someEffect`** **runs** `someEffect` and **binds** its **success** value to the **local** continuation.  
- If **`someEffect`** **fails**, the **whole** generator **short-circuits** with that **failure** (similar to **`?`** in Rust or **`try`** in Zig).  
- The **final** `return expr` in the generator gives the **overall** **success** value **`A`** of the **combined** effect.

**Type inference:** The **resulting** `Effect` typically has:

- **`A`** = type of the **final** `return`.  
- **`E`** = **union** of all **failure** types that can be produced by **any** `yield*`’d effect (and sometimes **widened**).  
- **`R`** = **combined** **requirements** of all **steps** (how exactly **`R`** merges depends on the **version**; think “**needs** all **services** any **step** needs”).

So **generators** **reduce** **visible** **`succeed`/`fail`** **noise** in the **middle** of a workflow: you write **imperative** `yield*` steps, not **`return Ok(yield* …)`**. You still **construct** **leaf** effects with **`succeed`/`fail`** (or other **primitives**) **at the edges**.

**Comparison to Forst `ensure`:** **`Effect.gen`** is **sequencing** + **automatic** **error propagation** inside a **block**. Forst **`ensure`** is **validation** + **narrowing** in the **typechecker**; **different** concern, but both **avoid** repeating **failure wrappers** at **every** line inside a **happy-path** region.

### 5.5 Takeaways for Forst

| Topic | Effect practice | Possible Forst resonance |
| --- | --- | --- |
| **Naming** | **`succeed`/`fail`** not **`Ok`/`Err`** | Forst may prefer **typed** **`return`** or **factories** over **`Ok`/`Err`** **constructors** ([12](./12-result-primitives-without-ok-err.md)); **guards** can stay **`Ok`/`Err`** on **`is`**. |
| **Generators** | **`Effect.gen`** + **`yield*`** | A **future** language feature could **sequence** **`Result`**-like steps with **implicit** **branch** selection; **not** the same as **`Effect`**’s **runtime** (`R` **layers**). |
| **Type** **helpers** | **`Effect.Effect.Success<…>`** (etc.) | If **`Result`** is **everywhere**, **projecting** **`S`** and **`F`** from **expression** types helps **IDE** and **API** authors. |
| **Three** **params** | **`A,E,R`** | Forst’s **`Result`** is **two**-sorted; **capabilities** might be a **separate** feature (e.g. **implicit context**), not **overloading** **`Result`**. |

### 5.6 Error channel mapping (transform, refine, recover)

Effect treats **errors as values** in the **`E`** slot. **Mapping** and **handling** are usually **combinators** on **`Effect`**, not language syntax:

| Mechanism | Role |
| --- | --- |
| **`Effect.mapError`** | **`(e: E) => E2`** — transforms the **failure** without changing **success**; **widens** or **refines** **`E`** depending on the function. Still **fails**; does not “fix” the run. |
| **`Effect.mapBoth`** | Maps **both** **success** and **failure** in one step. |
| **`Effect.catchAll`** | **`(e: E) => Effect<A2, E2, R2>`** — **recovers** from **any** failure by **substituting** a new effect (often **success** or a **different** failure). |
| **`Effect.catchTag`** / **`catchTags`** | When errors are **`Data.TaggedError`** / **`Schema.TaggedError`** with a **`_tag`** discriminator, **handlers** are **selected** by **tag** — **type-safe** “**match** on error variant” at the **value** level. |
| **`Effect.either`** | **`Effect<A, E, R>` → `Effect<Either<E, A>, never, R>`** (conceptually) — **materializes** the sum for **manual** branching. |

**Typical style:** **`pipe(effect, Effect.mapError(…), Effect.catchTag("HttpError", …))`** — **explicit**, **composable**, but **verbose** when every boundary needs **mapping** + **tag** plumbing.

**Docs:** [Error channel operations](https://effect.website/docs/error-management/error-channel-operations/), [Expected errors](https://effect.website/docs/error-management/expected-errors/).

### 5.7 Cleaner mapping with Forst **`ensure`** (contrast)

Forst’s built-in **`ensure`** ([09](./09-ensure-is-narrowing-and-binary-types.md)) addresses a **different** slice of the problem than **`mapError`**:

| Concern | Effect | Forst **`ensure`** |
| --- | --- | --- |
| **Primary job** | **Run** an effect; **transform** or **catch** on the **error channel** at **composition** time. | **Check** a **predicate** on a **value**; on **success**, **narrow** the **type** for **following** code; on **failure**, **exit** via **`error`** (or **`ensure` block**). |
| **Types** | **`E`** is **tracked** through **unions** and **tags**; **recovery** returns new **`Effect`s**. | **Assertions** + **`is`** tie **runtime** checks to **refinement** (**success**/`Error` **subtypes** per [02](./02-result-and-error-types.md)). |
| **Noise** | Many **small** steps need **`.pipe`**, **`mapError`**, **`catchTag`** unless **abstracted** into helpers. | One **`ensure x is …`** can **replace** a **chain** of “**validate** then **mapError** to **domain** error” **if** the **constraint** is expressible as an **assertion** / **type guard**. |

**How Forst could be “cleaner”** (design directions, not implemented promises):

1. **Validation as assertions, not manual `mapError`** — **`ensure amount is Positive()`** (illustrative) **narrows** **`amount`** and **fails** with a **standard** failure path, instead of **`Effect.fail(ValidationError(…))`** + **`mapError`** at every call site.  
2. **Tagged errors** — Effect’s **`catchTag`** relies on **`_tag`**. Forst’s **error hierarchy** ([errors architecture](../errors/00-error-system-architecture.md)) can expose **tags** for **TS** / **discovery**; **`if err is SomeTag`**-style **narrowing** on **`Result`** **failure** **payloads** would mirror **`catchTag`** **without** a **runtime** **effect** **interpreter**.  
3. **No `R` channel** — Forst does **not** need **`Effect.mapError`** for **service** wiring; **`ensure`** stays **about** **values** and **types**, not **dependency** **injection**.

**Summary:** Effect’s **error mapping** is **general** and **explicit**; Forst **`ensure`** is **narrow** and **declarative** for **validation** + **refinement**. **Combining** them in spirit means: **prefer** **`ensure`** + **`is`** **when** the failure is “**this** **value** **did not** **satisfy** **the** **spec**”; **reserve** **Effect-style** **pipelines** (or **hand-written** **mapping**) for **cross-cutting** **remapping** at **API** **boundaries**.

**Further reading** (Effect): [Using generators](https://effect.website/docs/getting-started/using-generators/), [Creating effects](https://effect.website/docs/getting-started/creating-effects/), [Error channel operations](https://effect.website/docs/error-management/error-channel-operations/).

---

## 6. References

| Topic | Link |
| --- | --- |
| Forst `Result` / `Error` family | [02](./02-result-and-error-types.md) |
| Construction exploration | [12](./12-result-primitives-without-ok-err.md) |
| `ensure` / narrowing | [09](./09-ensure-is-narrowing-and-binary-types.md) |
| Error system (factories, hierarchy) | [errors architecture](../errors/00-error-system-architecture.md) |
| Effect — `Effect<A,E,R>`, generators | [effect.website](https://effect.website/) |
| Effect — error channel | [Error channel operations](https://effect.website/docs/error-management/error-channel-operations/) |

---

## Document history

| Change | Notes |
| --- | --- |
| Initial | Prior art + **implicit** error patterns; ties **`ensure`** and **error factories** to constructor-avoidance goals. |
| §5 Effect | **`Effect<A,E,R>`**, **`Effect.succeed`/`fail`**, **`Effect.gen`** + **`yield*`**, **`Effect.Effect.Success`**-style helpers; contrast with Forst. |
| §5.6–5.7 | **Effect** **`mapError`/`catchTag`/…** vs Forst **`ensure`** — cleaner **validation** + **refinement**. |
