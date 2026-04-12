# Crystal-inspired optionals and the error / nil pattern (RFC)

Exploration of **`T | Nil`** (Crystal-style **`T?`**) vs Go’s **`(value, error)`** and **pointer-or-nil** patterns—**not** TypeScript optional / **`undefined`** juggling. The **hub** is [00-crystal-inspired-optionals.md](./00-crystal-inspired-optionals.md); **topic-focused** docs are **01–10**.

## Topic index

| Doc | Topic |
|-----|--------|
| [00](./00-crystal-inspired-optionals.md) | **Hub** — stance, index |
| [01](./01-single-return-unions-and-go-interop.md) | Single-return, Crystal **grammar**, Go **import** mapping, unions, Rust **`Result`** ideas |
| [02](./02-result-and-error-types.md) | **`Result(Success, Error)`**, error hierarchy |
| [03](./03-typescript-emission-optionals-and-result.md) | TypeScript **emission** |
| [04](./04-tooling-migration-lsp-and-testing.md) | Tooling, LSP, migration, testing |
| [05](./05-cross-language-comparison.md) | Go / Forst / TS / Crystal matrix |
| [06](./06-nil-safety-pointers-and-optionals.md) | Nil deref vs **`T \| Nil`** value |
| [07](./07-background-motivation-and-tradeoffs.md) | Motivation, design space, **pros/cons**, **open questions** |
| [08](./08-go-interop-lowering.md) | **Lowering** optionals to Go |
| [09](./09-ensure-is-narrowing-and-binary-types.md) | **`ensure`**, **`is`**, narrowing |
| [10](./10-type-guards-shape-guards-and-optionals.md) | **Type / shape guards** vs optionals **`Result`** |
| [11](./11-forst-result-tuple-single-return.md) | **User-facing** — **`Result`**, **`Tuple`**, **single return**, **`is`** narrowing, Go interop, tooling, migration (**normative**, **except** `Result` **construction** — see **12**) |
| [12](./12-result-primitives-without-ok-err.md) | **Exploratory** — **§5.7** **stated preference**: **`return x`** **success**, **nominal** **error** **constructors** **failure**, **no** **`Ok`/`Err`** **keywords** at **returns**; **`is Ok`/`Err`** **guards** kept; **§5.6** brainstorm; **§7** **[errors RFC](../errors/README.md)**; **withdraws** mandatory **`Ok`/`Err`** **constructors** ([11](./11-forst-result-tuple-single-return.md) **§1.4**) |
| [13](./13-result-constructors-prior-art.md) | **Analysis** — **`Ok`/`Err`-style** vs **avoiding** wrappers; **Go/Zig**; **Effect**: **`Effect<A,E,R>`**, **`gen`/`yield*`**, **`mapError`/`catchTag`**, vs Forst **`ensure`** for **validation** + **refinement** |

**Errors RFC hub:** [02 — first-class errors (normative)](../errors/02-first-class-errors-normative.md) — **`ensure`-only** failure for **`Result`**, **`Err(Nominal)`**, **`or`** **LUB**, inference; [01 — ensure-only propagation](../errors/01-ensure-only-failure-returns.md) — **`ensure … is Ok() or …`** vs **`if`**; ties **[12 §5.7](./12-result-primitives-without-ok-err.md)** ( **02** refines failure **`return`** vs **`ensure`** ).

## See also

- [PHILOSOPHY.md](../../../../PHILOSOPHY.md) — explicit errors, TypeScript + Go interop.
- [ROADMAP.md](../../../../ROADMAP.md) — narrowing, `error`.
- [Error system RFC](../errors/README.md), [generics RFC](../generics/README.md).
