# Single-return Forst, Go multi-returns, unions, and Result ergonomics

This document is a **companion analysis** to [00-crystal-inspired-optionals.md](./00-crystal-inspired-optionals.md). It goes deeper on four topics:

1. **Ergonomics** if Forst **source** abandons **multiple return values** in favor of a **single** typed result.
2. **Mapping** Go’s **`(T, U, …, error)`** (and plain multi-value returns) into that model so **imported** APIs stay **coherent** for Forst authors.
3. **Unifying** that model with **binary type expressions** (`T | U`, `T & U`) as one **language primitive** (TypeScript-inspired unions/intersections), rather than parallel ad hoc types.
4. **Rust-inspired** features worth **selectively** borrowing for **`Result`**-style types (without committing to Rust’s full trait system).

**`Result` spelling and errors:** Forst uses **`Result(Success, ErrorType)`** with an **error-kinded** second parameter, **not** Rust’s default **`Result<T, E>`**—see [02-result-and-error-types.md](./02-result-and-error-types.md).

**Related:** [07](./07-background-motivation-and-tradeoffs.md) (motivation / tradeoffs), [08](./08-go-interop-lowering.md) (**lowering** **`T?`** to Go—this doc’s §2 is **import** mapping), [09](./09-ensure-is-narrowing-and-binary-types.md) (**`ensure`** / **`is`**).

**Status:** exploratory design note. It does **not** specify syntax or schedule.

---

## Crystal reference: how types are written (official type grammar)

The [Crystal language reference — Type grammar](https://crystal-lang.org/reference/latest/syntax_and_semantics/type_grammar.html) defines **concrete** spellings for restrictions, return types, aliases, and generics. The points below matter when comparing **single-return Forst** to a **Crystal-inspired** surface (see [00](./00-crystal-inspired-optionals.md)).

### Unions and nilable types

- **Union:** `Int32 | String` — the pipe **`|`** builds a **union** type (“Int32 or String”).
- **Nilable shorthand:** `Int32?` is **exactly** the same type as **`Int32 | ::Nil`** (the docs use **`::Nil`** for the root **`Nil`** type). There is **no** separate `Maybe(Int32)` in the **standard type grammar**; “maybe a value” is expressed as **`T?`** or **`T | Nil`**, not a **`Maybe(T)`** type constructor in the reference.

### Generic type constructors use parentheses

Crystal writes **`Array(String)`**, **`Tuple(Int32, String)`**, **`Pointer(Int32)`**, **`NamedTuple(x: Int32, y: String)`**, **`Proc(Int32, String)`**—the **type name** followed by **`(...)`** for type arguments. That is the closest analogue to Forst’s preferred **`Result(Value, Error)`** spelling (parentheses + comma), even though **`Result`** in Forst is **not** copied from Crystal’s stdlib.

### Tuples as a single return type

In the same grammar, a **tuple type** is written **`Tuple(Int32, String)`**; the shorthand **`{Int32, String}`** in **type** positions aliases to that **`Tuple(...)`**. A function that returns “two things” without using multiple return values can therefore return **one** value whose type is **`Tuple(...)`** or **`NamedTuple(...)`**—still **one** return type in the signature.

### What Crystal does *not* prescribe for errors

Crystal’s docs emphasize **unions** and **`Nil`** for **absence**. **Recoverable errors** in application code are often modeled with **exceptions** or **nilable** returns, not a built-in **`Result(T, E)`** in the **type grammar** the way Rust does. Forst’s **`Result(Value, Error)`** is therefore **Go-oriented** (mapping to **`(T, error)`**) while **borrowing** Crystal’s **union** / **`?`** ergonomics for **optionals**, not a lift of a Crystal **`Result`** keyword.

### Summary table (Crystal vs illustrative Forst)

| Idea | Crystal (reference) | Forst (illustrative / target) |
| ---- | ------------------- | ------------------------------- |
| Optional value | **`String?`** ≡ **`String \| ::Nil`** | **`T?`** or **`T \| Nil`**; same **union** story |
| Two values, one return | **`Tuple(Int32, String)`** or **`{Int32, String}`** (type position) | **`Tuple(Int32, String)`** or named **struct** / **tuple** |
| Success / failure | Not one fixed **`Result`** in type grammar | **`Result(Value, ErrorSubtype)`** → Go **`(Value, error)`** ([02](./02-result-and-error-types.md)) |
| Union syntax | **`A \| B`** | **`A \| B`** ([ROADMAP](../../../../ROADMAP.md) **binary types**) |

**References:** [Type grammar — Union](https://crystal-lang.org/reference/latest/syntax_and_semantics/type_grammar.html#union), [Type grammar — Nilable](https://crystal-lang.org/reference/latest/syntax_and_semantics/type_grammar.html#nilable), [Type grammar — Tuple](https://crystal-lang.org/reference/latest/syntax_and_semantics/type_grammar.html#tuple), [Type grammar — NamedTuple](https://crystal-lang.org/reference/latest/syntax_and_semantics/type_grammar.html#namedtuple).

---

## 1. Ergonomics: no multiple returns in Forst *source*

### 1.1 What “get rid of multiple returns” could mean

| Strictness | Forst author writes | Emitted Go might still be |
| ---------- | ------------------- | --------------------------- |
| **Soft** | Prefer **one** return type; multi-return **discouraged** by style / lints | `(T, error)` where the compiler **splits** a nominal result |
| **Hard** | **Only** one return expression type per function | `(T, error)` only at **FFI** boundary or **inside** generated helpers |

The **hard** rule is a **language design** choice: the **surface** grammar would not have **comma-separated return types** like Go. Instead, every function returns **exactly one** type—often a **product** (**`Tuple(...)`** / **struct** / **NamedTuple**-like) or a **sum** (**`Result(...)`** or **`T \| Nil`**), analogous to how **Crystal** composes types in its [type grammar](https://crystal-lang.org/reference/latest/syntax_and_semantics/type_grammar.html).

### 1.2 Ergonomic benefits

- **One place to look:** Callers pattern-match or destructure **once** instead of juggling **`v, err :=`** ordering and **ignored** returns (`_, err`).
- **Composable APIs:** Chaining **`map`**, **`and_then`**, **`?`-style** propagation (if added) applies to **one** value; no special cases for “second return is always `error`”.
- **Alignment with TypeScript:** TS functions return **one** value; “multiple things” are **objects** or **tuples** as **one** value—closer to **mental model** of frontend developers.
- **Alignment with Crystal-style types:** A **tuple** return **`Tuple(Int, String)`** matches the **single return type** + **product** pattern from Crystal’s grammar (see table above).
- **Documentation:** Signature **`→ Result(Int, ParseError)`** states **success vs failure** in the **type**, not in positional convention.

### 1.3 Ergonomic costs and risks

- **Verbosity without sugar:** Rust’s **`?`** and **method chains** make **`Result`** bearable; without them, **`if let`** / **`match`**-heavy code can feel **worse** than **`if err != nil`**.
- **Loss of Go isomorphism at source:** Authors who **think** in **`(T, error)`** must learn **Forst’s** nominal **`Result`** (or tuple) instead of **mirroring** Go tutorials line-for-line.
- **Product types for “innocent” multi-values:** A Go function returning **`(Int, String)`** without `error` maps naturally to a **single** Forst type **`Tuple(Int, String)`** (Crystal **Tuple**-shaped) or a **small struct**—**clear**, but needs **tuple literals** and **destructuring** in Forst.
- **Migration:** Existing Forst and **copy-pasted** Go patterns using **two returns** need a **story** (see §2).

### 1.4 Relation to “optional” (`T?`) vs `Result`

- **`T?`** (or **`T \| Nil`**, Crystal-style) models **absence**—often **not** an error.
- **`Result(Success, ErrorSubtype)`** models **success** or **failure**; the **failure** side is **always** in the **`Error`** hierarchy ([02](./02-result-and-error-types.md)).

Single-return **does not** replace **`error`** by **`?`**; it **bundles** value + error into **one** return **type**. **`T?`** remains orthogonal (e.g. **`Result(Int?, NotFoundError)`** is usually wrong—prefer **`Result(Int, NotFoundError)`**).

---

## 2. Mapping Go multi-returns into Forst’s single-return model

### 2.1 What Go actually provides (`go/types`)

Go does **not** have a built-in **`Result`** type. The compiler exposes function returns as a **`types.Signature`** whose **`Results()`** is a **`types.Tuple`** of **`types.Var`** values in **source order** ([Go spec: Return values](https://go.dev/ref/spec#Return_values)). Anything else is **idiom**, not a distinct type kind:

| Go surface | Under `go/types` |
| ---------- | ----------------- |
| **`func() (T, error)`** | Tuple length **2**; types **`T`** and **`error`** (predeclared **`error`** is a **named interface type**) |
| **`func() (T, U)`** (no `error`) | Tuple length **2**; arbitrary **`T`**, **`U`** |
| **`func() (T, U, error)`** | Tuple length **3** |
| **`func() (error)`** | Tuple length **1**; single **`error`** |
| **Named results** | Same tuple; **names** are **not** part of **type** identity for mapping |

**Implication:** A **predictable** Forst mapping must be a **pure function** of **(ordered tuple of `go/types.Type)`** (plus possibly **which** slots **`goTypeToForstType`** can map). There is **no** flag in **`Signature`** that says “this is idiomatic error handling.”

---

### 2.2 What Forst does today (baseline)

Qualified Go calls are checked in **`checkGoSignature`** ([`go_interop.go`](../../../../forst/internal/typechecker/go_interop.go)): each **result** slot **`i`** is translated with **`goTypeToForstType(res.At(i).Type())`** independently. If **any** slot is **unsupported**, the call is rejected with **`unsupported Go return type`**.

- **Supported** today (see [`goTypeToForstType`](../../../../forst/internal/typechecker/go_interop.go)): **`error`** (string match → **`Error`**), **basic** kinds → **`Bool`/`Int`/`Float`/`String`**, **slices** → **`[]T`**, **pointers** → **`*T`**. **Other** shapes (most **named** interfaces, **structs**, **channels**, **maps** beyond what’s wired) → **unsupported**.
- **Hover** uses **`goSignatureReturnsToForst`** ([`go_hover.go`](../../../../forst/internal/typechecker/go_hover.go)) with the **same** per-slot mapping—**no** nominal **`Result`**; the UI shows **comma-separated** Forst types (e.g. **`String`, `Error`**).

So **today**: **`(T, error)`** is **two** return types in the **typechecker**, **not** one **`Result(T, Error)`** type.

---

### 2.3 A precise **`Result`** rule for **imported** types

To get a **stable, predictable** Forst type **`Result(Success, Failure)`** ([02](./02-result-and-error-types.md)) from **Go** **without** guessing:

**Rule R2 (recommended strawman)**

1. Let **`n = sig.Results().Len()`**.
2. Map each **`go/types`** result with **`goTypeToForstType`** to **`F₀ … F_{n-1}`**. If **any** fails → **no** `Result` shortcut (keep **error** / **tuple** story as today, or **diagnostic**).
3. **If `n == 2`** and **`F₁`** is exactly the Forst **`Error`** builtin (i.e. Go’s last parameter type is the **predeclared `error`** and maps to **`Error`**), then the **imported** return type is **`Result(F₀, Error)`** (or **`Result(F₀, ErrorSubtype)`** only if we later extend mapping for **named** error types—see §2.5).
4. **Otherwise** do **not** use **`Result`**: represent as **`Tuple(F₀, …)`** / **multiple** return types in **Forst** until **tuple** syntax is unified, or as **symbol-tied** structural types.

**Round-trip:** Emitting Go from **`Result(F₀, Error)`** **must** produce **`(T_go, error)`** where **`T_go`** is the inverse of **`F₀`** for supported **`F₀`**.

**Why last-slot `error`:** Matches **overwhelming** stdlib convention (`io`, `os`, `strconv`, …) and is **detectable** from **`go/types`** without heuristics on **parameter names**.

---

### 2.4 Other arities (predictable fallbacks)

| **`go/types` tuple** | Mapped **`Fᵢ`** all OK | Suggested Forst surface (strawman) |
| ---------------------- | ------------------------ | ----------------------------------- |
| **`n == 0`** | — | **`Void`** (already in [`goSignatureReturnsToForst`](../../../../forst/internal/typechecker/go_hover.go)) |
| **`n == 1`** | **`F₀`** | **`F₀`** (no `Result`) |
| **`n == 2`**, last **not** **`error`** | **`F₀`, `F₁`** | **`Tuple(F₀, F₁)`** (or **anonymous** 2-tuple)—**not** `Result` |
| **`n == 2`**, last is **`error`** | **`F₀`, `Error`** | **`Result(F₀, Error)`** |
| **`n == 3`**, last is **`error`** | **`F₀`, `F₁`, `Error`** | **`Result(Tuple(F₀, F₁), Error)`** *or* stay **3** returns until **product** types settle—**pick one** and document |
| **`n ≥ 2`**, **no** trailing **`error`** | **`F₀…`** | **`Tuple(F₀, …)`** |
| **Single `error` only** | **`Error`** | **`Result(Void, Error)`** *or* **`Error`** only—**design** choice ( **`Result`** keeps **one** composable pattern) |

**Unpredictable / unstable** until **`goTypeToForstType`** grows: returns using **named** error types **`MyErr`** (not identical to **`error`** in the **mapping**), **structs**, **channels**, etc.—those slots may stay **unsupported** or map to a **wide** **`Error`** with **lossy** typing.

---

### 2.5 Precision limits (named `error`, generics, methods)

- **Predeclared `error`:** Today’s mapper treats **`t.String() == "error"`** specially ([`go_interop.go`](../../../../forst/internal/typechecker/go_interop.go)). **Named** types whose **underlying** is **`error`** may **not** map to **`Error`** until **`go/types`** **`Implements`** checks are wired—**Rule R2** should **only** fire when the **second** result is **provably** the **`error`** interface we model as **`Error`**, or we **explicitly** widen to **`Result(F₀, Error)`** and **lose** static **subtype** information.
- **Generics (Go 1.18+):** **`func() (T, error)`** on a **generic** function** instantiates** to **concrete** tuples in **`go/types`**; **`Result`** mapping applies to **instantiated** signatures.
- **Methods / qualification:** **`Result`** identity should depend on **mapped** **`Fᵢ`**, not on **whether** the symbol is **`pkg.F`** vs **`(*T).M`**—same **tuple** → same **`Result`** type (structural **nominal** equality).

---

### 2.6 Strategies (how to implement R2)

| Strategy | Idea | Predictability |
| -------- | ---- | -------------- |
| **A — Structural tuple only** | **`(T, error)`** → **`Tuple(T, Error)`** | **High** (purely structural); **no** `Result` **keyword** in **Forst** |
| **B — `Result` alias of tuple** | **`Result(T, Error)` ≡** same **representation** as **`Tuple(T, Error)`** with a **standard** name | **High**; **call** / **emit** **round-trip** rules stay **simple** |
| **C — Adapter wrapper** | Thin **Forst** wrapper around **`import`** | **Stable** API, **extra** stack frame unless **inlined** |

**Coherence (exports):** Forst→Go **splits** **`Result(T, Error)`** → **`(T, error)`** for **public** Go **API** ([08](./08-go-interop-lowering.md), [02](./02-result-and-error-types.md)).

---

### 2.7 Summary

| Question | Answer |
| -------- | ------ |
| **What does Go “support”?** | **Ordered** **`go/types.Tuple`** of result types—**no** **`Result`** marker. |
| **What is predictable?** | **`Result(F₀, Error)`** **iff** **`n==2`** and slot **1** maps to **`Error`** under the **same** rules as today. |
| **What does today’s compiler do?** | **Per-slot** types—**no** **`Result`**—see **`checkGoSignature`**. |
| **Next step for `Result` imports?** | Implement **Rule R2** + **tuple** fallback for other arities; extend **`goTypeToForstType`** for **named** errors **before** promising **`Result(T, ParseError)`** from **`strconv.Atoi`**. |

---

## 3. Same primitive as binary type expressions (TypeScript- and Crystal-style unions)

[ROADMAP](../../../../ROADMAP.md) already tracks **binary type expressions** (`T | U`, `T & U`) in the parser/AST, with **meet/join** semantics and **codegen** still **incomplete**. **Optional/nilable** types and **`Result`**-like types **interact** with that work.

In **Crystal**, **`|`** in **type** positions is the **native** way to form unions ([Type grammar — Union](https://crystal-lang.org/reference/latest/syntax_and_semantics/type_grammar.html#union)); **`T?`** is sugar for **`T | ::Nil`**. Forst’s **binary types** + **`T?`** aim at the **same** ergonomics: **one** narrowing story for **optional** values and (with **`Result`**) **discriminated** success/failure.

### 3.1 What “one primitive” could mean

| Concept | As binary / algebra | Notes |
| ------- | --------------------- | ----- |
| **`T?`** (optional value) | **`T \| Nil`** (Crystal: **`T \| ::Nil`**) | Same **sentinel** **absence** story as Crystal’s **nilable** shorthand |
| **`Result(Success, ErrorSubtype)`** | **`Ok(Success) \| Err(ErrorSubtype)`** (needs **generics** or built-in **`Result`** for **constructors**) | **Discriminated** union; **narrowing** on **tag**; **Err** payload is **error-kinded** ([02](./02-result-and-error-types.md)) |
| **Intersection** | **`T & U`** | **Property** merge in TS; in Go may lower to **embedded** structs or **approximation** |

**One internal representation** (per roadmap): **binary** `|` / `&` on **`TypeNode`** (and **assertions**), **not** a **separate** parallel hierarchy only for **`?`**. Shorthand **`T?`** is then **syntax sugar** for **`T | Nil`** (Crystal-style), and **`Result`** is **`Ok | Err`** with **parameters**—**same** **union** machinery.

### 3.2 Why unify

- **Narrowing** one implementation: **`if`** / **`match`** / **`ensure`** refine **`T | U`** uniformly ([SEMANTICS_NARROWING.md](../../../../forst/internal/typechecker/SEMANTICS_NARROWING.md) already anticipates **join** at **if** merge points when unions exist).
- **Fewer** “special” types in **`go/types`** mapping: **`Result`** becomes **patterns** on **unions**, not a **hard-coded** third kind beside **optional**.
- **TypeScript emission:** **`T | U`** maps to **union types** naturally; **`Result`** maps to **discriminated unions** if **`Ok`/`Err`** carry **kind** fields (see [typescript-client](../typescript-client/README.md)).

### 3.3 Tensions and simplification limits

- **Go codegen:** Arbitrary **unions** do **not** lower to **idiomatic** Go without **wrappers** or **interface{}**-style **erasure**. The **language** may **restrict** **surface** unions to **lowerable** subsets (**Result**, **optional**, **enum-like** tags) while **sharing** the **same** **internal** **algebra** for **checking**.
- **Generics / built-in **`Result`:** **`Result(Success, Failure)`** as **`Ok|Err`** requires **type constructors** or a **builtin**; binary **`|`** alone is **not** enough without **[user generics](../generics/README.md)**. The **second** parameter remains **error-kinded** ([02](./02-result-and-error-types.md)).

**Conclusion:** **Yes**, **`T?`**, **discriminated results**, and **general** **`T | U`** **can** share **one** **type algebra** and **narrowing** pipeline; **emit** may still **special-case** **lowerable** forms to **`(T, error)`** and **pointers**.

**`ensure` and `is`:** The same **union** / **optional** machinery should power **narrowing** after **`if x is …`** and **`ensure x is …`** (see [guard RFC](../guard/guard.md)). If **single-return `Result`** replaces **raw** **`(T, error)`** in Forst **source**, **`ensure`** on **`Result`**-shaped values needs **designed** patterns so **guard**-style checks and **`Result`** propagation do **not** split into **two** idioms—see [09-ensure-is-narrowing-and-binary-types.md](./09-ensure-is-narrowing-and-binary-types.md).

---

## 4. Rust-inspired features for `Result` types (selective borrow)

Rust’s **`Result<T, E>`** is **well-studied**. Features worth **considering** for Forst (with **Go interop** constraints):

| Rust feature | Purpose | Forst adaptation notes |
| ------------ | ------- | ---------------------- |
| **`?` operator** | Propagate **`Err`** early | **Sugar** lowering to **`if err != nil { return … }`** in Go; must respect **PHILOSOPHY** (**explicit**—could require **explicit** **`try`**/`?` only in **`Result`**-returning functions) |
| **`map`, `map_err`, `and_then`** | Combinators without **unwrap** | **Methods** or **package** functions on **`Result`**; **generic** methods need **generics** |
| **`match` / `if let`** | Exhaustive or **pattern** binding | Align with **future** **`match`** or **`switch`** + **narrowing** |
| **`#[must_use]`** | Force **handling** of **`Result`** | **Compiler** lint on **ignored** **`Result`**—fits **“errors must be handled”** |
| **`From` / `Into` conversions** | **`?`** with **automatic** **coercion** of **error** types | **Careful**: **interop** with **`error`** **interface** in Go |
| **`ok_or` / `transpose`** | **`Option` ↔ `Result`** | Links **`T?`** and **`Result`** when **both** exist (Crystal uses **`T?`**, not **`Maybe`**, for the optional side) |
| **Specialization of `panic!` / `unwrap`** | **Escape** hatch | **Discouraged** per PHILOSOPHY; **debug** only or **lint** |

**What *not* to copy wholesale**

- **Rust’s** **`trait`** **system** for **overloading** **`?`**—Go has **no** traits; keep **lowering** **predictable**.

Forst does **not** treat the **failure** side as an **unconstrained** type parameter **`E`**; **`Result(Success, ErrorSubtype)`** maps to Go **`error`** per [02](./02-result-and-error-types.md) and the [errors RFC](../errors/README.md).

---

## 5. Cross-cutting summary

| Question | Short answer |
| -------- | ------------ |
| **Ergonomics of single return?** | **Clearer** **boundaries** and **one** **return** **type**; **Crystal**-style **`Tuple`** / **`T \| Nil`** for products and **nilables** (not TS **optional** / **`undefined`** **juggling**—[00](./00-crystal-inspired-optionals.md)). |
| **Go multi-return mapping?** | **`go/types`** tuple + **`Rule R2`** (§2.3): **`Result(F₀, Error)`** **iff** **`n==2`** and last maps to **`Error`**; today **`checkGoSignature`** is **per-slot** only (§2.2). |
| **Same primitive as `T \| U`?** | **Yes** for **typechecker** and **narrowing** (Crystal uses **`|`** + **`::Nil`** for nilables); **emit** may **specialize** to **idiomatic** Go. |
| **Rust features?** | **`?`**, **combinators**, **`must_use`**, **pattern** **matching**, **`transpose`**—**behind** **generics** and **explicit** **design** for **error** **conversion**. |

---

## References

- [00-crystal-inspired-optionals.md](./00-crystal-inspired-optionals.md) — hub.
- [02-result-and-error-types.md](./02-result-and-error-types.md) — **`Result(Success, Error)`** and **subtypes**.
- [07](./07-background-motivation-and-tradeoffs.md), [08](./08-go-interop-lowering.md), [09](./09-ensure-is-narrowing-and-binary-types.md)
- [03-typescript-emission-optionals-and-result.md](./03-typescript-emission-optionals-and-result.md), [04-tooling-migration-lsp-and-testing.md](./04-tooling-migration-lsp-and-testing.md), [05-cross-language-comparison.md](./05-cross-language-comparison.md), [06](./06-nil-safety-pointers-and-optionals.md)
- [ROADMAP.md](../../../../ROADMAP.md) — binary type expressions, narrowing, Go interop.
- [00-user-generics-and-type-parameters.md](../generics/00-user-generics-and-type-parameters.md) — **`Result`** needs **generics**.
- [SEMANTICS_NARROWING.md](../../../../forst/internal/typechecker/SEMANTICS_NARROWING.md) — control-flow **join**.
- [PHILOSOPHY.md](../../../../PHILOSOPHY.md) — explicit **errors**, **predictability**.
- **Crystal:** [Type grammar](https://crystal-lang.org/reference/latest/syntax_and_semantics/type_grammar.html) (Union, Nilable, Tuple, NamedTuple).
- **Rust:** [The Rust Book — Error Handling](https://doc.rust-lang.org/book/ch09-00-error-handling.html), [`Result`](https://doc.rust-lang.org/std/result/enum.Result.html).

---

## Document status

**Exploratory analysis.** Revise when **binary types** inference/emit and **generics** **roadmap** **milestone** **land**.
