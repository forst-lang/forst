# Forst `Result`, `Tuple`, and single-return functions (user-facing RFC)

**Audience:** Forst language users and API authors planning ahead.  
**Status:** **Normative direction** — reflects **locked design decisions** for the compiler and tooling. Exact grammar, keyword spelling, and release timing follow implementation; this document is the **contract** for *what* the language is moving toward and *why* it matters to you.

**Relations:** Builds on [02-result-and-error-types.md](./02-result-and-error-types.md) (philosophy and error family) and [01-single-return-unions-and-go-interop.md](./01-single-return-unions-and-go-interop.md) (Go mapping). Where older docs say “exploratory” or “not locked,” **this** doc states the **intended** user-visible shape unless a section explicitly marks “TBD at implementation.”

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

### 1.4 Constructors: `Ok` and `Err`

Values of type `Result(T, E)` are built with dedicated **language** forms **`Ok(...)`** and **`Err(...)`** (not user-definable functions with the same names). That keeps codegen, diagnostics, and learning materials unambiguous.

**Implication:** You write `return Ok(value)` / `return Err(err)` instead of `return value, nil` / `return zero, err` in Forst source. The compiler still emits **idiomatic Go** `(T, error)` at the boundary (see §3).

### 1.5 Consuming `Result`: `is` and narrowing (not multi-assign)

The primary way to **unpack** a `Result` in v1 is **flow-sensitive narrowing** with **`is`**, symmetrically on success and failure:

- **`if r is Ok(v)`** — in the then-branch, **`v`** has the success type (and **`r`** is known to be the success case where the type system tracks that).  
- **`if r is Err(e)`** — in the then-branch, **`e`** has the failure type, narrowed according to **`Result(T, F)`** (e.g. `F` a specific error subtype).

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
- **Minor open details:** Exact JSON shape for `returnDescriptors`, fine points of **`is`** syntax for `Ok`/`Err`, and success-side subtyping rules for `Result`—specified at implementation time and reflected back into this hub if needed.

---

## 7. References

| Doc | Role |
|-----|------|
| [02](./02-result-and-error-types.md) | Error-kinded **`Failure`**, hierarchy, Rust contrast |
| [01](./01-single-return-unions-and-go-interop.md) | Single-return ergonomics, Go `go/types` mapping |
| [09](./09-ensure-is-narrowing-and-binary-types.md) | **`ensure`**, **`is`**, narrowing |
| [ROADMAP.md](../../../../ROADMAP.md) | Feature status |
| [PHILOSOPHY.md](../../../../PHILOSOPHY.md) | Design boundaries |

---

## Document history

| Change | Notes |
|--------|--------|
| Initial | User-facing consolidation of **Result**, **Tuple**, **single return**, **Ok/Err**, **symmetric `is` narrowing**, **Go interop**, **parenthesized type display**, **versioned discovery**, **flag-day migration**, **test-first** implementation strategy. |
