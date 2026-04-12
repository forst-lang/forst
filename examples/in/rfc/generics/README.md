# User generics and type parameters (RFC)

This folder specifies a **phased design** for user-defined generic types and functions in Forst: declared type parameters, instantiation, constraints, Go codegen, and incremental Go interop. It is a **design and roadmap** document only—implementation follows separately.

## Documents

- **[00-user-generics-and-type-parameters.md](./00-user-generics-and-type-parameters.md)** — Full RFC: motivation, benefits, current baseline, staged capabilities, design decisions, implementation phases, risks and mitigations (with allowed / not-supported examples), and testing strategy.

## See also

- [ROADMAP.md](../../../../ROADMAP.md) — language feature status (Generic types are currently **planned**).
- [PHILOSOPHY.md](../../../../PHILOSOPHY.md) — inference and explicitness principles that constrain generic inference.
- [examples/in/generics.ft](../../../../examples/in/generics.ft) — today’s **built-in** parametric types (`[]T`, `*T`, `map[K]V`), not user polymorphism.
- [optionals / `Result`](../optionals/02-result-and-error-types.md), [optionals 12](../optionals/12-result-primitives-without-ok-err.md) (**open** **construction** only; **`Ok`/`Err`** **guards** on **`Result`** **for now**), and [generics RFC §10](./00-user-generics-and-type-parameters.md#result-types-generics-and-narrowing-ok-and-err) — **`Result(Success, Failure)`** as a **parameterized** type and **`if` / `ensure`** narrowing.
