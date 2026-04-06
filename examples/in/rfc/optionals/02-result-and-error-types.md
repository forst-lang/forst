# `Result` types, `Error`, and error subtypes

This document is a **companion** to the [optionals hub](./00-crystal-inspired-optionals.md) and [01-single-return-unions-and-go-interop.md](./01-single-return-unions-and-go-interop.md). **Motivation / tradeoffs:** [07](./07-background-motivation-and-tradeoffs.md). It fixes a **Forst-facing convention** for **`Result`** and explains how that choice ties into the **error hierarchy**, **subtyping**, **Go `error`**, and **narrowing**—separate from the **optional** (`T?`) story.

**Status:** exploratory specification sketch. Final syntax (parentheses vs generics, keyword spelling) is **not** locked.

---

## 1. Convention: `Result(Success, Failure)` with a failure type in the `Error` family

Forst documentation should **not** default to Rust’s **`Result<T, E>`** spelling, where **`E`** can be **any** type (including non-errors).

**Preferred Forst shape (illustrative):**

```text
Result(SuccessType, ErrorType)
```

- **`SuccessType`** — the value when the operation **succeeds** (e.g. `Int`, `User`, `String`).
- **`ErrorType`** — the type of the **failure** case. It is **always** an **error type** in the Forst sense: it **extends** or **refines** the **`Error`** concept defined in the [error system architecture](../errors/00-error-system-architecture.md), **not** an arbitrary `E`.

**Examples (illustrative):**

| Signature (illustrative) | Meaning |
| ------------------------ | ------- |
| **`Result(Int, ParseError)`** | Success is **`Int`**; failure is **`ParseError`** (a **subtype** of **`Error`**). |
| **`Result(User, NotFoundError)`** | Narrow failure domain for **this** API. |
| **`Result(String, Error)`** | Success is **`String`**; failure may be **any** structured **`Error`** (widest failure type). |

The **`Result(X, Error)`** form (with the **base** **`Error`** name in the second position) is the **maximally general** failure side while staying **in the error system**. More **specific** APIs should prefer **`Result(X, ConcreteError)`** so callers can **`is`**-narrow or **`match`** on the **failure** branch without guessing.

**Why spell it this way**

- **Readability:** Matches **function-call** intuition (`Result` as a **type constructor** with **two** operands), distinct from Rust’s **angle-bracket** generics in docs.
- **Alignment with the errors RFC:** Failures are **first-class** **domain types** with **factories**, **wrapping**, and **observability**—not opaque **`E`**.
- **Go interop:** The **failure** side still lowers to **`error`** (or **`error`** + **type assertions** / **`errors.As`** for subtypes); see §4.

**Contrast with Rust**

| | Rust `Result<T, E>` | Forst `Result(Success, ErrorType)` |
| --- | --- | --- |
| **Second parameter** | **Any** `E` | **Error** (or **subtype** / **refinement** of **`Error`**) |
| **Typical use** | `E` is often **`String`**, **`()`**, or custom non-error | **`ErrorType`** participates in **hierarchy**, **tagged** TS emit, **`ensure`** story |

Rust remains a **useful** reference for **combinators** and **`?`** (see [01](./01-single-return-unions-and-go-interop.md)); the **type** story is **deliberately** **tighter** in Forst.

---

## 2. Error hierarchy and subtypes

The [errors RFC](../errors/README.md) describes a **hierarchical**, **structured** error model (base **`Error`**, categories like **`ValidationError`**, **`DatabaseError`**, etc.). **`Result(Success, Failure)`** **inherits** that model:

- **`Failure`** **must** be **compatible** with **`Error`** (subtyping, nominal extension, or a **declared** refinement—exact mechanism follows the same rules as the rest of the type system).
- **API authors** choose **how narrow** the failure type is:
  - **Narrow:** **`Result(Token, ExpiredError)`** — callers know **only** certain failures are **possible** at the **type** level (when the compiler can prove it).
  - **Wide:** **`Result(Token, Error)`** — any **`Error`**; use at **boundaries**, **logging**, or **generic** handlers.

**Subtyping and assignability**

- A **`Result(Int, ParseError)`** is naturally **usable** where **`Result(Int, Error)`** is expected if **`ParseError`** is a **subtype** of **`Error`** (standard **width** subtyping on the **failure** parameter).
- The **inverse** (treating **`Result(Int, Error)`** as **`Result(Int, ParseError)`**) is **unsound** without a **runtime** check—same as narrowing **`error`** to **`ParseError`** in Go with **`errors.As`**.

This gives **one** story: **typed** failures in **signatures**, **wider** **`Error`** at **composition** points, **narrowing** with **`is`** / **`errors.As`** where needed.

---

## 3. Unions of error types (future binary types)

If Forst gains **`|`** on types ([ROADMAP](../../../../ROADMAP.md)), the **second** parameter might be a **small** **union** of **error** types, e.g. **`ParseError | IoError`**, when an operation can fail in **a few** known ways but not **every** **`Error`**. That remains **within** the **error** family—still **not** an arbitrary non-error **`E`**.

**Interaction with `Result(X, Error)`:** **`Error`** is the **top** of the failure side for **generic** handling; **unions** of **specific** errors are **middle** ground between **one** concrete type and **`Error`**.

---

## 4. Go interoperability

- **Emitted** Go for a **`Result`-returning** Forst function should remain **idiomatic** **`(SuccessType, error)`** (or a **thin** wrapper that **implements** **`error`** for structured types), consistent with [PHILOSOPHY](../../../../PHILOSOPHY.md).
- **Mapping `Error` subtypes** to Go uses the same **wrapping**, **`fmt.Errorf`**, **`errors.Is`**, **`errors.As`** strategy as the [errors architecture](../errors/00-error-system-architecture.md).
- **Imported** Go **`(T, error)`** can be **surfaced** as **`Result(T, Error)`** (or **`Result(T, UnknownGoError)`** if the **static** shape is **opaque**)—see [01 §2](./01-single-return-unions-and-go-interop.md).

**Principle:** **`Result`’s** second parameter is **always** **error-flavored** in **Forst** types; **Go** still sees **`error`** at the **interface** boundary.

---

## 5. Narrowing, `ensure`, and `is` on the failure branch

- After **`if r is Ok(...)`**, the **success** payload narrows like any **union** (see [09](./09-ensure-is-narrowing-and-binary-types.md)).
- After **`if r is Err(...)`** (or **`ensure r is Ok(...) or err`** patterns), the **failure** value should narrow to **`ParseError`** (etc.), not **`Error`**, when the **static** type is **`Result(Int, ParseError)`**.

**`ensure`** composes with **`Result`** when **failure** is **explicit** and **typed**—same **narrowing** pipeline as **optionals** and **unions**.

---

## 6. TypeScript emission

Failure types that are **`Error`** subtypes should map to **tagged** / **discriminated** shapes in generated **TypeScript** (aligned with the [error system](../errors/README.md) and [typescript-client](../typescript-client/README.md) plans). **`Result(Success, ConcreteError)`** should **not** degrade to **`any`** on the **error** side when the **concrete** error shape is known.

---

## 7. Summary

| Topic | Direction |
| ----- | --------- |
| **Spelling** | Prefer **`Result(SuccessType, ErrorType)`** in **Forst** docs and **future** syntax; **`ErrorType`** is **`Error`** or a **subtype** / **union** of **errors**. |
| **Not** | Rust-default **`Result<T, E>`** with **unconstrained** **`E`** as the **primary** story. |
| **Hierarchy** | **`Result`** **inherits** the **errors RFC** model; **narrow** failure types in **APIs**, **widen** to **`Error`** at **edges**. |
| **Interop** | **`(T, error)`** in Go; **structured** subtypes via **`errors.As`** / **wrapping**. |

---

## References

- [00-crystal-inspired-optionals.md](./00-crystal-inspired-optionals.md) (hub)
- [01-single-return-unions-and-go-interop.md](./01-single-return-unions-and-go-interop.md), [07](./07-background-motivation-and-tradeoffs.md)
- [03-typescript-emission-optionals-and-result.md](./03-typescript-emission-optionals-and-result.md) — **`Result`** in **`.d.ts`**
- [errors/README.md](../errors/README.md), [00-error-system-architecture.md](../errors/00-error-system-architecture.md)
- [PHILOSOPHY.md](../../../../PHILOSOPHY.md)
- [ROADMAP.md](../../../../ROADMAP.md) — binary types, narrowing

---

## Document status

**Exploratory.** Update when **syntax** for **`Result`** and **error** **definitions** is **fixed** in the **compiler** and when **generics** land enough to **encode** **`Result`** **parametrically** while keeping the **second** parameter **error-kinded**.
