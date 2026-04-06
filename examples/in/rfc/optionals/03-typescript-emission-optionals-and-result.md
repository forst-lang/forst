# TypeScript emission: optionals, unions, and `Result`

Companion to the [optionals hub](./00-crystal-inspired-optionals.md). **TypeScript-only** analysis: how **`T | Nil`**, **unions**, and **`Result`** should surface in **generated** `.d.ts`—what exists in **`GetTypeScriptType`** and what is **future** work.

**Status:** exploratory. Aligns with [typescript-client](../typescript-client/README.md) and [00-implementation-plan.md](../typescript-client/00-implementation-plan.md).

---

## 1. Goals (from PHILOSOPHY) — source vs emit

- **Accurate** types for **Node** and **browser** consumers ([PHILOSOPHY.md](../../../../PHILOSOPHY.md)).
- **Minimize `any`** on hot paths; unions should **reflect** Forst **intent**.

**Critical distinction ([00](./00-crystal-inspired-optionals.md)):** In **Forst**, **`T?`** means **`T | Nil`** (Crystal-style)—**one** **absence** case in the **type** system. We **reject** adopting TypeScript’s **meaning** for **`?`**: **optional properties**, pervasive **`T \| undefined`**, and the **juggling** between **omitted** vs **`undefined`** vs **`null`**. **Emission** to `.d.ts` is a **mapping** problem: it must **not** redefine Forst as “possibly **undefined**” in the **TS** sense.

- **Consistency** with the [error system](../errors/README.md): tagged / structured errors in TS where the Forst model exposes them.

---

## 2. Today’s emitter (baseline)

`GetTypeScriptType` in [`forst/internal/transformer/ts/type_mapping.go`](../../../../forst/internal/transformer/ts/type_mapping.go) maps:

- Slices, maps, pointers (e.g. **`*T` → `(T) | null`**).
- **`error` → `unknown`** (conservative).

Unparameterized **`[]`** / **`map`** still fall back to **`any[]`** / **`Record<any, any>`** per [implementation plan](../typescript-client/00-implementation-plan.md). **No** first-class **`T?`** or **`Result`** in the typechecker yet means **no** dedicated emission path for those **concepts**—this document describes the **target** once the **language** supports them.

---

## 3. Nilable values (`T | Nil`, incl. `T?`)

**Forst semantics:** **`String | Nil`** (or **`String?`**) is a **union** with a **dedicated** **`Nil`** type—not “maybe **undefined**” as in TS **optional** **fields**.

**Emission principles (avoid TS `?` juggling)**

| Approach | Illustrative `.d.ts` | Fit |
| -------- | -------------------- | --- |
| **Union with `null`** | **`string \| null`** | **Single** absent **sentinel**; works with **JSON** often using **`null`**. **Prefer** over **`string \| undefined`** as the **default** story for **nilable** scalars so consumers **don’t** mix **three** states (**missing** / **`undefined`** / **`null`**). |
| **Branded / literal `Nil`** | **`string \| Nil`** where **`Nil`** is a **declared** **const** or **brand** | Preserves **Forst** **`Nil`** **literally**; may need **helper** **predicates** in **generated** **client** code. |
| **Avoid as default** | **`string \| undefined`**, **`name?: string`** | **Do not** treat as the **primary** mapping for Forst **`T \| Nil`**—that mirrors TS **optional** **properties** and **`undefined`** **juggling**, which this design **explicitly** **avoids** in **language** **semantics**. |

**Shape fields:** A field **`x: T | Nil`** is **`T \| null`** (or **`T \| Nil`**) in the **object** type—**not** **`x?: T`** **unless** we **document** that **omission** is **not** the same as **Nil** (generally **avoid** **conflating** the two).

**Narrowing in TS:** Prefer **`x !== null`** (or **`isNil(x)`**) consistent with the **chosen** **sentinel**—**not** **`x !== undefined`** as the **only** **story**.

---

## 4. Unions (`A | B`)

Binary type expressions ([ROADMAP](../../../../ROADMAP.md)) map naturally to **TypeScript union types**: **`A | B`**.

**Caveats**

- **Parenthesization:** Same as today for **array of union** (`(A | B)[]` vs `A | B[]`)—[`type_mapping.go`](../../../../forst/internal/transformer/ts/type_mapping.go) already parenthesizes unions inside **arrays** where needed.
- **Exhaustiveness:** TS **does not** enforce **exhaustive** **`switch`** at compile time for all unions; Forst may be **stricter** in analysis—**emitted** types alone do **not** copy that guarantee into TS.

---

## 5. `Result(Success, ErrorSubtype)`

Forst **`Result`** is **`error-kinded`** on the **failure** side ([02](./02-result-and-error-types.md)). In TypeScript, natural shapes include:

| Approach | Shape (illustrative) | Pros / cons |
| -------- | -------------------- | ----------- |
| **Discriminated union** | `{ ok: true; value: T } \| { ok: false; error: E }` | **Narrowing** on **`ok`**; aligns with **Effect**-style **tagged** errors if **`E`** is branded. |
| **Tuple-like** | **`[T, null] \| [null, E]`** | Rare in TS **API** style; **less** idiomatic than **object** tags. |
| **Separate** **`Result` type alias** | **`type Result<T, E> = …`** | Matches **frontend** expectations if **one** canonical **`Result`** is **exported** from **`types.d.ts`**. |

**Integration with errors RFC:** Structured **`Error`** subtypes should emit as **interfaces** or **branded** types with a **discriminant** (`_tag`, **`kind`**, etc.) so **clients** can **`switch`** and integrate with **Effect** patterns described in [errors](../errors/README.md).

**Interop with today’s `error` → `unknown`:** Until **`Result`** exists end-to-end, **`(T, error)`**-shaped APIs may continue to emit **tuple** or **overload** forms; migrating to **`Result`** in Forst should **replace** those with **discriminated** shapes **without** widening success/failure to **`unknown`** unnecessarily.

---

## 6. Sidecar / HTTP JSON

Runtime **invocation** uses JSON **envelopes** ([sidecar](../sidecar/00-sidecar.md), [ROADMAP](../../../../ROADMAP.md)). **JSON** lacks **`undefined`** on the wire; **`null`** is the natural **runtime** partner for **`Nil`** in many APIs. **OpenAPI** / **schema** generation (planned) should align **wire** **null** with **static** **`T \| null`** / **`Nil`** **semantics**—**without** implying **omitted** **fields** are **`Nil`** unless **explicitly** **specified**.

---

## 7. Work items (staged)

1. **Extend `GetTypeScriptType`** for **`TypeOptional`** / **union** / **`Result`** when those **AST** nodes exist and are **inferred**.
2. **Tests:** Golden **`types.d.ts`** snippets for **`T?`**, **`|`** , **`Result`** (mirror **`type_mapping_test.go`** style).
3. **Docs:** Update [typescript-client README](../typescript-client/README.md) when **`forst generate`** **stabilizes** **new** type forms.

---

## References

- [00-crystal-inspired-optionals.md](./00-crystal-inspired-optionals.md) — § TypeScript emission.
- [02-result-and-error-types.md](./02-result-and-error-types.md) — **`Result`** and **errors**.
- [typescript-client/00-implementation-plan.md](../typescript-client/00-implementation-plan.md)
- [PHILOSOPHY.md](../../../../PHILOSOPHY.md)

---

## Document status

**Exploratory.** Revise when **`internal/transformer/ts`** gains **branches** for **optional** / **union** / **`Result`** types.
