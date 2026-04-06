# Cross-language comparison: optionals, errors, and `Result`

One-page **reference** aligning **Go**, **target Forst**, **TypeScript**, and **Crystal** on **absence**, **failure**, and **product** returns. Details live in [00](./00-crystal-inspired-optionals.md), [01](./01-single-return-unions-and-go-interop.md) (Crystal **type grammar**), and [02](./02-result-and-error-types.md).

---

## Absence of a value (“maybe no value”)

| | Go | Forst (target) | TypeScript (ecosystem) | Crystal |
| --- | --- | --- | --- | --- |
| **Meaning of “optional”** | **`nil`** (typed locations) | **`T \| Nil`** (incl. **`T?`** sugar)—**not** TS **optional** / **`undefined`** semantics ([00](./00-crystal-inspired-optionals.md)) | Often **`T \| undefined`**, **`?`** **properties**, **`null`**—**many** **states** | **`T?`** ≡ **`T \| ::Nil`** ([type grammar](https://crystal-lang.org/reference/latest/syntax_and_semantics/type_grammar.html#nilable)) |
| **Primitives** | No **`int?`**; use **pointer** or **sentinel** | **Policy** (see [08](./08-go-interop-lowering.md)) | **`number \| undefined`**, etc. | Union with **`Nil`** |

**Emit** to TS is **not** the **same** column: [03](./03-typescript-emission-optionals-and-result.md) maps **`T \| Nil`** without defaulting to TS’s **`undefined`** **juggling**.

---

## Recoverable failure (“operation failed”)

| | Go | Forst (target) | TypeScript | Crystal |
| --- | --- | --- | --- | --- |
| **Primary encoding** | **`(T, error)`**; **`error`** **interface**, **`nil`** = success | **`Result(T, ErrorSubtype)`** → emit **`(T, error)`** ([02](./02-result-and-error-types.md)) | **Exceptions**, **`Promise` rejection**, or **typed** **`Result`** **patterns** in **user** code | Often **exceptions**; **not** the same as Forst **`Result`** **keyword** |
| **Typed error domain** | **`errors.Is` / `As`**, **custom** **`error`** **types** | **`Error`** **hierarchy** + **`Result`** **failure** **parameter** | **Branded** / **tagged** **errors** (e.g. **Effect**) | **Exception** **types** |

---

## Multiple values from one function

| | Go | Forst (target) | TypeScript | Crystal |
| --- | --- | --- | --- | --- |
| **Surface** | **Multiple** **returns** **`(T, U, …)`** | **Prefer** **single** **type**: **`Tuple(T, U)`** or **`Result`** ([01](./01-single-return-unions-and-go-interop.md)) | **Single** **return** (**object** / **tuple** **value**) | **`Tuple(T, U)`** / **`NamedTuple`** ([type grammar — Tuple](https://crystal-lang.org/reference/latest/syntax_and_semantics/type_grammar.html#tuple)) |

---

## Unions in the type system

| | Go | Forst (target) | TypeScript | Crystal |
| --- | --- | --- | --- | --- |
| **Sum types** | **Limited** (**interface{}**, **type** **switch**) | **`T \| U`**, **`T?`** ([ROADMAP](../../../../ROADMAP.md)) | **`A \| B`** **unions** | **`A \| B`** **pipe** in **types** |

---

## Document status

**Reference only.** Update when **Forst** **syntax** **freezes**.
