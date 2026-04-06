# Type guards, shape guards, and optionals / `Result`

**Topic:** How **Crystal-style** **`T | Nil`**, **`Result(Success, Error)`**, and **unions** relate to **type guards** and **shape guards** ([guard RFC](../guard/guard.md)), what **unlocks** if they share one **narrowing** story, and what **risks** appear.

**Status:** exploratory analysis. It does not override the [guard RFC](../guard/guard.md); it connects that work to the [optionals hub](./00-crystal-inspired-optionals.md).

---

## 1. Terms (from the guard RFC)

| Term | Meaning (informal) |
| ---- | ------------------ |
| **Type guard** | A **pure** predicate **`is (x: T) Name { … }`** used with **`is`** / **`ensure`** to **refine** **`T`** into **`T.Name`** (or similar) on the **success** branch. |
| **Shape guard** | **`x is Shape({ … })`** (or **Match** constraints)—**structural** refinement: **fields** and **constraints** on **shapes**. |
| **Refinement** | A **narrower** static type after a **successful** check. |
| **`ensure`** | Statement: validate at **runtime**, **narrow** in the **success** continuation, **fail** with **`error`** (often **multi-return** in **generated Go** today). |

**Today:** **`is`** is wired for **built-in** assertions and **shape**-style checks; **user-defined** **`is (x) Guard`** is **desired state** in the RFC, not fully shipped everywhere ([guard RFC § Desired State](../guard/guard.md)).

---

## 2. How optionals and `Result` fit the same picture

### 2.1 **`T | Nil` as a sum type**

- **`T | Nil`** has two **inhabitants** in principle: a **value of `T`** or **`Nil`**. That is the same **discriminate → narrow** pattern as **`x is Int()`** or **`x is Strong()`**: the **`is`** side picks **which** case holds.
- A **natural** guard family (illustrative): **`Present`** / **`Nil`** (names arbitrary)—**`if x is Present()`** narrows **`T | Nil`** → **`T`**.

### 2.2 **`Result(Success, Error)` as a sum type**

- **`Result`** is **logically** **`Ok(Success) | Err(Error)`** ([01](./01-single-return-unions-and-go-interop.md) §3, [02](./02-result-and-error-types.md)). **Type guards** in the guard RFC already sketch **`is (result) Success`** for a **`Result`-like** union ([guard RFC — Generic Type Guards](../guard/guard.md#generic-type-guards) example).
- **Unlock:** One **`is Err(...)`** / **`is Ok(...)`** story aligns **`ensure`** on **`Result`** with **`if`** narrowing—**same** **machinery** as **shape** and **value** guards, provided **`InferAssertionType`** / **unions** share one **representation** ([SEMANTICS_NARROWING.md](../../../../forst/internal/typechecker/SEMANTICS_NARROWING.md)).

### 2.3 **Shape guards and optional **fields****

- **`Shape({ name: String? })`** (if the **language** allows **optional** **fields** as **`T | Nil`**) ties **shape** **syntax** to **nilable** **types**: **shape** **validation** can **refine** “field **present** vs **absent**” where the **static** type already carries **`| Nil`**.
- **Unlock:** **Fewer** ad hoc **sentinels** in **guards**; **optional** **presence** is **typed** before **`ensure`**.

---

## 3. Possible unlocks

| Unlock | Detail |
| ------ | ------ |
| **One narrowing pipeline** | **`if` / `is` / `ensure`** + **unions** + **refinements** use **one** **join** / **narrow** implementation ([09](./09-ensure-is-narrowing-and-binary-types.md)). **Optionals** become **another** **sum**, not a **second** analyzer. |
| **Composable** guards on ** unions** | **User-defined** **`is (x: T \| U) CaseA`** (future)—**same** **composition** rules as **password** **Strong** / **VeryStrong** ([guard RFC](../guard/guard.md)). |
| **`Result`** **ergonomics** | **`if r is Ok()`** / **`ensure r is Ok() or err`** with **narrowed** **payload** types—**matches** the **generics** RFC’s **`Result[Int, AppError]`**-style **example** once **`Result`** and **generics** exist ([generics RFC §10](../generics/00-user-generics-and-type-parameters.md#result-types-generics-and-narrowing-ok-and-err)). |
| **LSP / UX** | **Hover** shows **narrowed** **union** after **`is`**; **optional**/`Result` **discriminants** are **first-class** in **IDE** **messages** ([04](./04-tooling-migration-lsp-and-testing.md)). |
| **TS / Go emit** | **Guards** already aim to affect **`.d.ts`** and **Go** ([guard RFC Summary](../guard/guard.md)); **nilable** unions give **emit** a **clear** **target** ([03](./03-typescript-emission-optionals-and-result.md)). |
| **Documentation** | **One** **vocabulary** for **“narrowing”** — **beginners** learn **`is`** **once**, apply to **values**, **shapes**, **optionals**, and **`Result`**. |

---

## 4. Potential risks

| Risk | Detail |
| ---- | ------ |
| **Implementation surface** | **Unions** + **refinements** + **user guards** + **`ensure`** **compound** subjects ([SEMANTICS_NARROWING.md](../../../../forst/internal/typechecker/SEMANTICS_NARROWING.md) **non-goals** for **path** **subjects**)—**schedule** risk if **everything** lands at once. |
| **Soundness** | **Narrowing** **`T | Nil`** requires **exact** **runtime** **agreement** with **static** **semantics** (what is **`Nil`** at **runtime** in **Go**?). **Wrong** **lowering** = **unsound** **guards** ([08](./08-go-interop-lowering.md), [06](./06-nil-safety-pointers-and-optionals.md)). |
| **Duplicate concepts** | **Shape** **constraints** vs **assertion** **refinements** vs **union** **tags**—without **one** **internal** **algebra**, the **compiler** maintains **parallel** **systems** ([ROADMAP](../../../../ROADMAP.md), [generics RFC](../generics/00-user-generics-and-type-parameters.md)). |
| **RFC example tension** | The guard RFC’s **illustrative** **`type Result<T> = T | Error`** differs from Forst’s **`Result(Success, ErrorSubtype)`** as **`Ok|Err`** ([02](./02-result-and-error-types.md)). **Docs** must **disambiguate** so **guard** **authors** don’t **mix** **models**. |
| **Performance** | **Static** **analysis** of **composed** **guards** on **large** **unions** can **blow** up; **“efficient to analyze statically”** is a **guard** **goal**—**binary** **types** **meet/join** **must** stay **bounded** in practice. |
| **User confusion** | **`ensure` on optional** vs **`ensure` on Result** vs **shape** **failure**—**three** **failure** **modes**; **style** **guide** **required** ([07](./07-background-motivation-and-tradeoffs.md)). |
| **Go** **interop** | **Guards** **emit** to **Go**; **`Result`** **imports** from **Go** use **Rule R2** ([01](./01-single-return-unions-and-go-interop.md) §2.3). **Mismatched** **expectations** if **Guards** **assume** **multi-return** but **call** **site** uses **`Result`**. |

---

## 5. Suggested direction

1. **Unify** **union** / **optional** / **refinement** **AST** and **narrowing** **before** **promising** **user-defined** **guards** on **every** **combinator**.
2. **Document** the **relationship** between **guard** **RFC** **examples** and **`Result(Success, Error)`** ([02](./02-result-and-error-types.md)) so **generics** **examples** **don’t** **fork** **semantics**.
3. **Stage:** **built-in** **`Nil`** **discrimination** + **`Result`** **tags** **before** **arbitrary** `**is (x: A|B) …**` **user** **guards** if **complexity** **requires** it ([04](./04-tooling-migration-lsp-and-testing.md)).

---

## References

- [guard RFC](../guard/guard.md), [anonymous_objects.md](../guard/anonymous_objects.md), [interop.md](../guard/interop.md) (if present)
- [00](./00-crystal-inspired-optionals.md), [09](./09-ensure-is-narrowing-and-binary-types.md)
- [SEMANTICS_NARROWING.md](../../../../forst/internal/typechecker/SEMANTICS_NARROWING.md)

---

## Document status

**Exploratory.** Revise when **user-defined** **guards** and **binary** **types** **land** in the **checker**.
