# `ensure`, `is`, and narrowing for optionals and `Result`

**Topic:** How **optionals** (**`T | Nil`**) and **`Result`** compose with **`ensure`**, **`if` … `is`**, and **binary type expressions**—one **narrowing** pipeline ([guard RFC](../guard/guard.md), [SEMANTICS_NARROWING.md](../../../../forst/internal/typechecker/SEMANTICS_NARROWING.md)).

---

## Today’s behavior (summary)

- **`ensure subject is Assertion(...) or err`** — runtime check, **narrows** subject on **success**, **returns** **`error`** on failure (often **multi-return** in **generated Go**).
- **`if`** + **`is`** — **then** branch sees **narrowed** type.

Details: [guard RFC](../guard/guard.md).

### Compiler enforcement (`Result` + `if` … `is Err`)

The typechecker rejects **`return Err(...)`** inside the **then** branch of **`if subject is Err(...)`** when **`subject`** is a built-in **`Result(S, F)`**—use **`ensure subject is Ok()`** (and optional **`or err`**) to propagate failure instead ([errors RFC 01](../errors/01-ensure-only-failure-returns.md)). **`return Ok(...)`** in that branch for **recovery** remains valid. **`else`** after **`is Ok`** (failure region without negated narrowing) is **not** covered yet.

---

## How optionals and unions fit

- **`T | Nil`** is a **sum** type. **`is`** can discriminate **present** vs **absent** like **refinements** or **tagged** variants—**one** rule: after a successful **`is`**, the binding has a **smaller** type.
- **`Result(Success, ErrorSubtype)`** uses the **same** narrowing idea: **success** vs **failure** branches refine payloads ([02](./02-result-and-error-types.md)). **`Ok`** and **`Err`** are **built-in guards** on **`Result`** for **`is`/`ensure`** ([12 §0](./12-result-primitives-without-ok-err.md#0-scope-split-guards-vs-constructors)); **only** **value construction** at **`return`** is **open** in [12](./12-result-primitives-without-ok-err.md).

---

## Shared implementation

[ROADMAP](../../../../ROADMAP.md): **binary types** (`T | U`) and **narrowing** should share **one** **type algebra**. **`ensure`** successor narrowing is part of that pipeline; **compound `ensure` subjects** (field paths) remain **deferred** per [SEMANTICS_NARROWING.md](../../../../forst/internal/typechecker/SEMANTICS_NARROWING.md).

---

## Coherence: does this story hold?

**Yes, if:**

1. **`is`** = “check succeeds → **narrow**; else **failure** path.”
2. **Failures** = **`error`** or **`Err`** in **`Result`**—not **`?`** **smuggling** errors.
3. **`ensure`** stays **explicit** ([PHILOSOPHY](../../../../PHILOSOPHY.md)).

**Risks:** Mixing **`ensure … or err`** with **`Result`** + **`?`** without **one** **style** guide; **`ensure`** on **`Result`**-shaped values needs **designed** patterns ([01](./01-single-return-unions-and-go-interop.md)); **user-defined guards** must apply to **`T?`** and **unions**, not only **base** types.

**Bottom line:** **Discriminate → narrow → continue** with **explicit** failures—**quality** depends on **one** narrowing implementation and **clear** docs ([00](./00-crystal-inspired-optionals.md) hub).

**Deeper (guards):** [10-type-guards-shape-guards-and-optionals.md](./10-type-guards-shape-guards-and-optionals.md) — **type** / **shape** guards, **unlocks**, **risks**.

---

## References

- [00-crystal-inspired-optionals.md](./00-crystal-inspired-optionals.md)
- [01-single-return-unions-and-go-interop.md](./01-single-return-unions-and-go-interop.md) §3 (unions)
- [04-tooling-migration-lsp-and-testing.md](./04-tooling-migration-lsp-and-testing.md)

---

## Document status

**Exploratory.** Revise when **union** inference and **`ensure`** on **unions** land.
