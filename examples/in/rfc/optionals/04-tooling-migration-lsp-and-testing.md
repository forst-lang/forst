# Tooling, migration, LSP, and testing

Companion analysis for the [optionals / Result](./README.md) thread. It covers **what must change in the compiler and editor** if **`T?`**, **unions**, and **`Result(Success, Error)`** ship, how **migrations** could work, and how **testing** should prove progress—topics only **summarized** in [00](./00-crystal-inspired-optionals.md) (e.g. “implementation cost,” open questions).

**Status:** exploratory checklist, not a committed roadmap.

---

## 1. Compiler pipeline surface area

Roughly the pipeline in [debugging rules](../../../../.cursor/rules/debugging.mdc):

| Layer | Work for optionals / unions / `Result` |
| ----- | ---------------------------------------- |
| **Lexer / parser** | **`?`**, **`|`** in **type** positions, **`Result(...)`** type constructor ([generics](../generics/README.md) may share **generic** syntax). |
| **AST** | Optional / union / **`Result`** **nodes** or **normalized** binary **`|`** on [`TypeNode`](../../../../forst/internal/ast/type.go). |
| **Typechecker** | **Unification**, **subtyping** (including **`Error`** hierarchy — [02](./02-result-and-error-types.md)), **narrowing** ([SEMANTICS_NARROWING.md](../../../../forst/internal/typechecker/SEMANTICS_NARROWING.md)), interaction with **`ensure` / `is`** ([09](./09-ensure-is-narrowing-and-binary-types.md)). |
| **Transformer (Go)** | Lowering **`T?`** → **pointer** / **`(T, bool)`** / **struct** ([08](./08-go-interop-lowering.md)); split **`Result`** → **`(T, error)`**. |
| **Transformer (TS)** | [03-typescript-emission-optionals-and-result.md](./03-typescript-emission-optionals-and-result.md). |
| **LSP** | **Hover** / **completion** on **narrowed** types; **diagnostics** when **`?`** / **`Result`** misused; optional **code actions** for **unwrap** patterns (future). |

**Order of delivery** (suggested): **AST + typechecker** **core** for **`T | Nil`** and **narrowing** before **full** **`Result`** **combinators**—mirrors [generics RFC](../generics/00-user-generics-and-type-parameters.md) **staging**.

---

## 2. LSP and developer experience

- **Diagnostics:** Unchecked **`Result`**, impossible **narrowing**, **`Result`** vs **`T?`** confusion—**lint**-level messages aligned with [PHILOSOPHY](../../../../PHILOSOPHY.md) (**explicit** handling).
- **Hover:** Show **narrowed** type after **`if`** / **`ensure`** on **`T?`** and **`Result`** branches (depends on **single** narrowing implementation).
- **Go to def / rename:** **`Result`** and **error** **subtypes** must participate in the **same** **symbol** tables as today’s **types**.

---

## 3. Migration and adoption

| Strategy | Description |
| -------- | ----------- |
| **Style guide** | Document **`?` vs `*T` vs `Result`** ([00](./00-crystal-inspired-optionals.md)) so **mixed** codebases stay **readable**. |
| **Per-package opt-in** | If needed, **feature** flags or **`forst:`** **pragma** (only if complexity warrants—**avoid** permanent **forks**). |
| **Codemod (future)** | **Pointer-or-nil** → **`T?`** where **sound**; **`v, err :=`** → **`Result`** only with **human** review—**full** automation is **hard** across **Go** **FFI**. |
| **Examples** | New **`examples/in/*.ft`** + **golden** **`out/`** when syntax exists ([testing rules](../../../../.cursor/rules/testing.mdc)). |

Open questions from [07](./07-background-motivation-and-tradeoffs.md) (**syntax**, **primitive lowering**, **generics vs `T?` first**, **migration**) should be **closed** in **ADR** or **ROADMAP** before **large** **migrations** are **recommended**.

---

## 4. Testing strategy (project priorities)

Per [.cursor/rules/priorities.mdc](../../../../.cursor/rules/priorities.mdc):

1. **Unit tests** — **Narrowing** rules for **`T?`** and **unions**; **subtyping** for **`Result(Int, ParseError)`** vs **`Result(Int, Error)`**; **parser** fixtures for **`?`** and **`|`** in **typedefs**.
2. **Integration tests** — **`task example:*`** or **new** **`example:optional`** once **examples** exist; **full** pipeline **lexer → emit**.
3. **Regression** — **Golden** Go and **TS** output for **small** **fixtures** when **emit** rules **freeze**.

**Log output:** When implementing, **trace**/**debug** logs at **narrowing** and **emit** boundaries help **debug** **wrong** lowering ([debugging](../../../../.cursor/rules/debugging.mdc)).

---

## 5. Relationship to other analyses

- [00-crystal-inspired-optionals.md](./00-crystal-inspired-optionals.md) — **hub**.
- [01-single-return-unions-and-go-interop.md](./01-single-return-unions-and-go-interop.md) — **single return** and **Go** **mapping**.
- [02-result-and-error-types.md](./02-result-and-error-types.md) — **`Result`** and **errors**.
- [03-typescript-emission-optionals-and-result.md](./03-typescript-emission-optionals-and-result.md) — **TS** **emitter**.
- [07](./07-background-motivation-and-tradeoffs.md) — **open questions** (this doc §3 **migration**).
- [08](./08-go-interop-lowering.md), [09](./09-ensure-is-narrowing-and-binary-types.md)

---

## References

- [ROADMAP.md](../../../../ROADMAP.md)
- [PHILOSOPHY.md](../../../../PHILOSOPHY.md)
- [SEMANTICS_NARROWING.md](../../../../forst/internal/typechecker/SEMANTICS_NARROWING.md)

---

## Document status

**Exploratory.** Update when **implementation** **milestones** are **scheduled** on [ROADMAP](../../../../ROADMAP.md).
