# Background, motivation, design space, and tradeoffs

**Topic:** Why explore **`T | Nil`** / **`T?`**, how it relates to **`error`** and **`Result`**, baseline context, and **pros / cons / open questions**. For **single-return** and **unions** depth, see [01](./01-single-return-unions-and-go-interop.md).

---

## Motivation and baseline

- **Go** uses **`(T, error)`** and **`nil`** for absence in many places; **Forst** already has **`error`**, **pointers**, **`ensure`**, **guards** ([guard RFC](../guard/guard.md)); [ROADMAP](../../../../ROADMAP.md) tracks **narrowing**.
- **PHILOSOPHY** requires **explicit** error handling ([PHILOSOPHY.md](../../../../PHILOSOPHY.md)).

**Pain points (conceptual):**

- **Duplication** of `if err != nil` when **absence** and **failure** are related but **typed** separately.
- **Ambiguity** between “pointer may be nil” and “value may be missing” without a **first-class optional** in **source**.
- **Friction** with hand-written **TypeScript** overusing **`undefined`**—emit must not copy that model; hub stance in [00](./00-crystal-inspired-optionals.md).

---

## Design space (not prescriptive)

| Approach | Idea | Relation to `error` / `nil` |
| -------- | ---- | --------------------------- |
| **A. Nilable only (`T?`)** | `T?` for “value or absent”; failures still **`(T, error)`** | **Smallest** change; optionals replace many **pointer-or-nil** uses for **values** |
| **B. Result** | **`Result(Value, ErrorSubtype)`** ([02](./02-result-and-error-types.md)); needs [generics](../generics/README.md) or builtin | **Central** failure handling; **interop** → **`(T, error)`** |
| **C. Optional error (`Error?`)** | Optional **`error`** field / return | “Possibly an error” (English, not **`Maybe(T)`**); **discipline** vs **`nil`** |
| **D. Hybrid** | **`T?`** for absence; **`error`** for failure; **narrowing** | Close to **Go** with clearer **source** types |

**Stance:** Crystal-style **optionals** (**`T | Nil`**) **complement** **`error`**; they do **not** automatically **replace** **`(T, error)`** unless **`B`** lands with **generics**.

---

## Language ergonomics (summary)

**Wins:** Fewer **sentinels**; **narrowing** after guards ([ROADMAP](../../../../ROADMAP.md)); **unions** align with TS **`A | B`** for **non-Nil** sums; **`T | Nil`** stays **Crystal-style**, not TS **optional-field** style.

**Costs:** **`?` vs `error` vs `Result`** teaching; **compiler** surface; **style** drift vs raw Go; **TS readers** may misread **`?`**—see [00](./00-crystal-inspired-optionals.md) **stance**, [03](./03-typescript-emission-optionals-and-result.md).

---

## Pros and cons

### Pros

| Pro | Detail |
| --- | ------ |
| **Clearer intent** | Absence is **typed**, not **zero** / **nil** by convention. |
| **Union-friendly TS emit** | **`A \| B`** maps to TS; **`T \| Nil`** mapped **deliberately** ([03](./03-typescript-emission-optionals-and-result.md)). |
| **Safer refactors** | **`T`** → **`T?`** is **visible** at compile time. |
| **Crystal / Ruby familiarity** | Lower **learning curve** for some authors. |
| **Narrowing** | **`if`** / **`ensure`** refine **`T?` → `T`** when implemented ([09](./09-ensure-is-narrowing-and-binary-types.md)). |

### Cons

| Con | Detail |
| --- | ------ |
| **Go mismatch** | Lowering **`T?`** is a **policy** choice ([08](./08-go-interop-lowering.md)). |
| **Concept overlap** | **`T?`**, **`*T`**, **`(T, error)`**, **`Result`**—needs a **style guide**. |
| **Implementation cost** | See [04](./04-tooling-migration-lsp-and-testing.md). |
| **`?` for errors** | **`Error?`** fields OK; using **`?`** to **hide** failures violates **PHILOSOPHY** unless **`Result`** replaces it. |
| **Interop** | Boundaries may need **wrappers** until **one** model wins. |

---

## Implications for the language’s future

- **Nil deref:** Today’s checker does **not** prove **non-nil** pointers ([06](./06-nil-safety-pointers-and-optionals.md)); **`T \| Nil`** is mainly **clarity** / **narrowing**, not full **memory safety** without extra rules.
- **Narrowing** becomes **central** if **`T?`** is core ([SEMANTICS_NARROWING.md](../../../../forst/internal/typechecker/SEMANTICS_NARROWING.md)).
- **Generics** + **optionals** ordering: see [generics RFC](../generics/README.md).
- **Binary types** **`T | U`** should **share** one **algebra** with **nilable** ([01](./01-single-return-unions-and-go-interop.md) §3).
- Realistic slogan: **explicit optionals for absence; keep `error` for failure**—unless **`Result`** **everywhere** is adopted.

---

## Open questions

1. **Syntax:** `T?` vs `T | Nil` vs **keyword** `optional`.
2. **Primitives:** How **`Int?`** lowers to Go without **unboxed** pointers everywhere ([08](./08-go-interop-lowering.md)).
3. **`Result` vs `T?` first:** [generics](../generics/README.md) vs ship **optionals** first.
4. **`ensure` / guards:** Shared **narrowing** for **`T?`**? ([09](./09-ensure-is-narrowing-and-binary-types.md)).
5. **Migration / codemod:** [04](./04-tooling-migration-lsp-and-testing.md) §3.

---

## References

- [00-crystal-inspired-optionals.md](./00-crystal-inspired-optionals.md)
- [ROADMAP.md](../../../../ROADMAP.md), [PHILOSOPHY.md](../../../../PHILOSOPHY.md)

---

## Document status

**Exploratory.** Fold findings into **ROADMAP** / ADRs when syntax is chosen.
