# Forst `Result`, `Tuple`, and single-return functions (user-facing RFC)

**Audience:** Forst language users and API authors planning ahead.  
**Status:** **Normative direction** for **`Result`**, **`Tuple`**, **single-return** signatures, **Go lowering**, **tooling**, and **migration** — unless a section points to a more specific RFC. **Exception:** how **`Result`** values are **constructed** at **return** sites is **not locked**; see [12 — Result primitives without `Ok`/`Err`](./12-result-primitives-without-ok-err.md) (**exploratory**), which **withdraws** only the plan that **`Ok(...)`** / **`Err(...)`** are the **mandatory constructors**. **`Ok`** and **`Err`** remain **built-in type guards** on **`Result`** for **`is` / `ensure` narrowing** ([12 §0](./12-result-primitives-without-ok-err.md#0-scope-split-guards-vs-constructors)) — **intended for now**.

**Relations:** Builds on [02-result-and-error-types.md](./02-result-and-error-types.md) (philosophy and error family) and [01-single-return-unions-and-go-interop.md](./01-single-return-unions-and-go-interop.md) (Go mapping). Where older docs say “exploratory” or “not locked,” **this** doc states the **intended** user-visible shape **except** where **§1.4** defers **construction** to [12](./12-result-primitives-without-ok-err.md).

---

## 1. What you will see in the language

### 1.1 One return type per function

Forst functions will expose **exactly one** return type in source—aligned with TypeScript-style “one value out,” not Go’s raw multi-value return list as a **language** feature.

- **Today (transitional):** Some Forst code uses Go-like `(T, Error)` return lists and `return x, err`.  
- **Target:** A single type such as **`Result(T, E)`**, **`Tuple(...)`**, **`Void`**, or a concrete type—**not** comma-separated return types in the signature.

**Implication:** You think in **named outcomes** (`Result`, `Tuple`, …) instead of positional “second slot is always `error`.”

### 1.2 `Result(Success, Failure)` — success vs failure in the type

`Result` is a **type constructor** with two type arguments:

```text
Result(SuccessType, FailureType)
```

- **`SuccessType`** — value when the operation succeeds.  
- **`FailureType`** — type of the failure case. In Forst it is **error-kinded**: it must be compatible with the language’s **`Error`** concept (including subtypes such as a domain-specific error), **not** an arbitrary non-error `E` as in Rust’s default `Result<T, E>` story.

See [02-result-and-error-types.md](./02-result-and-error-types.md) for the rationale and hierarchy.

**Implication:** Narrowing and tooling can treat **failure** as part of the **same** error system as `ensure` and structured errors, not as an opaque extra return slot.

### 1.3 `Tuple(T1, …, Tn)` — products without an error channel

Go APIs sometimes return **multiple values that are not `(T, error)`**—for example several numbers or a quotient and remainder. Forst maps those to a **single** type:

```text
Tuple(T1, T2)        -- typical pair
Tuple(T1, …, Tn)     -- arity matches the Go signature
```

**Implication:** “Multiple things” from pure Go tuples become **one** Forst value whose type is `Tuple(...)`. This is **not** the same as `Result`; there is no required failure side.

### 1.4 Producing `Result` values (construction open — see RFC 12)

The language still needs a **clear, low-clutter** way to **build** **success** and **failure** values for `Result(T, E)` at **return** sites. A previous revision of this document **locked** **`Ok(...)`** / **`Err(...)`** as the **constructors** for those values; **that decision alone** is withdrawn ([12](./12-result-primitives-without-ok-err.md)).

**Stable for now (separate from construction):** **`Ok`** and **`Err`** as **built-in type guards** on **`Result`** for **`if r is Ok(...)`** / **`if r is Err(...)`** and **`ensure`** ([12 §0](./12-result-primitives-without-ok-err.md#0-scope-split-guards-vs-constructors)).

**Active design:** [12 — Result primitives without `Ok`/`Err`](./12-result-primitives-without-ok-err.md) records **goals**, **tradeoffs**, and **candidate** **construction** directions. **Leading candidate ([12 §5.7](./12-result-primitives-without-ok-err.md)):** **ordinary** **`return x`** for **success**, **explicit** **nominal** **error** **constructors** for **failure**—**no** **`Ok`/`Err`** **keywords** at **return** sites (**`is Ok`/`is Err`** **guards** unchanged). Until that work stabilizes and is folded back here, **user-facing** **constructor** syntax is **not** part of the stable contract.

**Implication:** Examples that show **`return Ok(x)`** / **`return Err(e)`** show one **transitional** construction style, not a **locked** one. **`is`** / **`ensure`** examples using **`Ok`/`Err` guards** match the **intended narrowing** story **for now**. **Lowering** to idiomatic Go **`(T, error)`** (§2) remains the **direction**.

### 1.5 Consuming `Result`: `is` and narrowing (not multi-assign)

The primary way to **unpack** a `Result` is **flow-sensitive narrowing** with **`is`**, using the **built-in `Ok`/`Err` guards** on **`Result`** ([12 §0](./12-result-primitives-without-ok-err.md#0-scope-split-guards-vs-constructors)) — **for now** the **normative** intent:

- **`if r is Ok(v)`** — in the then-branch, **`v`** / **`r`** carry the **success** type **`S`** for `Result(S, F)` where the typechecker models it.  
- **`if r is Err(e)`** — in the then-branch, **`e`** / **`r`** carry the **failure** type **`F`** (e.g. a specific **error** subtype).

There is **no** reliance on method-only combinators as the **sole** story for v1, and **no** long-term dependency on `a, b := f()` for `Result`-typed calls.

**Implication:** Control flow stays **explicit** and readable ([PHILOSOPHY.md](../../../../PHILOSOPHY.md)); you get **typed** branches similar in spirit to TypeScript narrowing, without exceptions.

---

## 2. Go interoperability (mental model)

| Go surface (conceptual) | Forst type (conceptual) |
| ------------------------ | ------------------------- |
| **`(T, error)`** | **`Result(T, Error)`** or a **more specific** `Result(T, ConcreteError)` when the static shape allows |
| **`(T1, …, Tn)`** with **no** `error`** | **`Tuple(T1, …, Tn)`** with matching arity |

**Lowering:** When **you** write Forst that returns `Result(T, E)`, generated Go remains **idiomatic** **`(T, error)`** for the success and failure channels. `Tuple` lowers to a Go **multi-value** return with the same arity.

**Implication:** Reading generated Go still feels familiar to Go reviewers; reading Forst source emphasizes **named** result types.

---

## 3. Types in documentation and the IDE

Parameterized types are shown with **parentheses** in human-facing messages (diagnostics, LSP hover), e.g. **`Result(Int, Error)`**, **`Array(String)`**, **`Map(String, Int)`**—one consistent style for “type with parameters,” not angle-bracket-only spellings.

**Implication:** Hover and errors should match RFC examples and tutorials; small visual churn if you relied on older internal `<...>` display in tooling output.

---

## 4. Tooling and ecosystem

### 4.1 Discovery / sidecars / TypeScript

Downstream tools (discovery JSON, executor, TypeScript generation) will move to a **versioned** schema that describes returns with **structured descriptors** (e.g. scalar vs `Result` vs `Tuple`, and how they relate to **lowered Go**), instead of overloading a single boolean like “multiple returns.”

**Implication:** Integrations should key off the **schema version** and new fields; the logical Forst return type and the **number of Go result slots** may differ (e.g. one `Result` → two Go slots).

### 4.2 Migration

The in-repo language migration is intended as a **flag day** with **no** long compiler deprecation window for old multi-return **syntax**: when the release ships, **new rules only**, and sources are updated to **`Result` / `Tuple` / single return** together with that release.

**Implication:** Projects outside this repo should pin compiler versions until they are ready to migrate `.ft` in one step; release notes will call out the break.

---

## 5. What stays familiar

- **Philosophy:** Explicit control flow, traceable errors, no exceptions as the default story ([PHILOSOPHY.md](../../../../PHILOSOPHY.md)).  
- **Go output:** **`error`** at the FFI boundary for failure; structured errors remain the long-term companion ([errors RFC](../errors/README.md)).  
- **`ensure` and refinement:** Still part of the same narrowing / validation story; **`ensure`** rules evolve to align with **`Result`**-shaped returns rather than positional “last slot is `error`” only (details in compiler work).

---

## 6. Implementation notes (for contributors, not language semantics)

These items are **tracked in the compiler** and do not change the user contract above, but explain why work is bundled:

- **Tests first:** End-to-end and pipeline tests (compile Forst → generated Go → **`go test` / build**) land early so refactors stay safe.  
- **Tuple arity:** **n-ary** `Tuple` in the first milestone so rare Go signatures with more than two results are not stranded.  
- **Minor open details:** Exact JSON shape for `returnDescriptors`, fine points of **`is`** / **discriminator** syntax for **`Result`** (see [12](./12-result-primitives-without-ok-err.md)), and success-side subtyping rules for `Result`—specified at implementation time and reflected back into this hub if needed.

---

## 7. References

| Doc | Role |
|-----|------|
| [02](./02-result-and-error-types.md) | Error-kinded **`Failure`**, hierarchy, Rust contrast |
| [01](./01-single-return-unions-and-go-interop.md) | Single-return ergonomics, Go `go/types` mapping |
| [09](./09-ensure-is-narrowing-and-binary-types.md) | **`ensure`**, **`is`**, narrowing |
| [12](./12-result-primitives-without-ok-err.md) | **Exploratory** — alternatives to **`Ok`/`Err`**; **open** construction + discriminator story |
| [ROADMAP.md](../../../../ROADMAP.md) | Feature status |
| [PHILOSOPHY.md](../../../../PHILOSOPHY.md) | Design boundaries |

---

## Document history

| Change | Notes |
|--------|--------|
| Initial | User-facing consolidation of **Result**, **Tuple**, **single return**, **Ok/Err**, **symmetric `is` narrowing**, **Go interop**, **parenthesized type display**, **versioned discovery**, **flag-day migration**, **test-first** implementation strategy. |
| RFC 12 | **Withdraws** locked **`Ok`/`Err`** **constructor** decision only; **§1.4** construction open; **§1.5** **`Ok`/`Err` as guards** intended **for now**; points to [12](./12-result-primitives-without-ok-err.md). |
| RFC 12 §5.7 | **§1.4** — **leading candidate**: **`return x`** **success**, **nominal** **error** **constructors** **failure** ([12 §5.7](./12-result-primitives-without-ok-err.md)). |
