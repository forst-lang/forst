# Optionals, `Nil`, and the error / nil pattern (hub)

This **hub** replaces a single long RFC: **each numbered document** below focuses on **one** topic. Read **[07](./07-background-motivation-and-tradeoffs.md)** for motivation and tradeoffs, or jump by topic from the index.

**Status:** exploratory—no implementation commitment.

---

## Stance (read first)

- **Nilable values** in Forst mean **`T | Nil`** (Crystal-style **`T?`** sugar)—[Type grammar — Nilable](https://crystal-lang.org/reference/latest/syntax_and_semantics/type_grammar.html#nilable). This is **not** TypeScript’s **`?` / `undefined`** juggling ([03](./03-typescript-emission-optionals-and-result.md)).
- **Crystal naming:** In the **official type grammar**, **optional / nilable** values are written **`T?`** or **`T | ::Nil`**, **not** **`Maybe(T)`** (that name is **Haskell** / common in **FP**; **Rust** uses **`Option<T>`**). You *could* define `alias Maybe(T) = T | Nil` in Crystal, but it is **not** part of the standard spelling—see **Crystal reference** in [01](./01-single-return-unions-and-go-interop.md). This RFC uses **`T?` / `T | Nil`** when talking about **Crystal** so we stay aligned with the reference.
- **Failures** use **`error`** / **`Result(Success, ErrorSubtype)`** ([02](./02-result-and-error-types.md)), not optional **`?`** as a **stand-in** for errors.

---

## Topic index

| # | Document | Focus |
|---|----------|--------|
| **00** | *This file* | Hub, stance, index |
| [01](./01-single-return-unions-and-go-interop.md) | Single-return, **Crystal** type grammar, **Go import** mapping, **unions** vs **binary types**, Rust-style **`Result`** ergonomics |
| [02](./02-result-and-error-types.md) | **`Result(Success, Error)`**, **error** hierarchy, **subtyping**, Go / TS |
| [03](./03-typescript-emission-optionals-and-result.md) | **`.d.ts`** / client mapping; **`T \| Nil`** vs TS emit |
| [04](./04-tooling-migration-lsp-and-testing.md) | Compiler surface, **LSP**, migration, **testing** |
| [05](./05-cross-language-comparison.md) | **Go / Forst / TS / Crystal** matrix |
| [06](./06-nil-safety-pointers-and-optionals.md) | Pointers, **nil**, what the **checker** proves today |
| [07](./07-background-motivation-and-tradeoffs.md) | Motivation, baseline, **design space** A–D, **pros/cons**, **open questions**, roadmap implications |
| [08](./08-go-interop-lowering.md) | **Lowering** **`T?`** to Go (`*T`, `(T,bool)`, …) |
| [09](./09-ensure-is-narrowing-and-binary-types.md) | **`ensure`**, **`is`**, **narrowing**, **coherence** with unions / **`Result`** |
| [10](./10-type-guards-shape-guards-and-optionals.md) | **Type guards**, **shape guards**, **unlocks** / **risks** vs **`T \| Nil`** / **`Result`** |

---

## References

- [PHILOSOPHY.md](../../../../PHILOSOPHY.md), [ROADMAP.md](../../../../ROADMAP.md)
- [generics](../generics/README.md), [guard](../guard/guard.md), [errors](../errors/README.md), [typescript-client](../typescript-client/README.md)
- **Crystal:** [Nilable](https://crystal-lang.org/reference/latest/syntax_and_semantics/type_grammar.html#nilable), [Union](https://crystal-lang.org/reference/latest/syntax_and_semantics/type_grammar.html#union)

---

## Document status

**Hub only**—detail lives in **01–10**. Revisit after **generics** and **narrowing** milestones.
