# `switch` vs `match` (RFC)

> **Implemented (2026):** Go-faithful `switch` / `case` / `default` / `fallthrough` shipped. This RFC remains as design history; **`match`** is still a separate future RFC.

This folder resolves the open design question flagged in [ROADMAP.md](../../../../ROADMAP.md#control-flow--statements) for `switch` / `case` / `default` / `fallthrough`: implement Go's `switch` as-is, design a `match`-style construct instead, or do both.

## Documents

- **[00-switch-and-match.md](./00-switch-and-match.md)** — Full comparison: prior art (Go, Rust, Swift, Kotlin, TypeScript, Scala), how each option fits today's `is` / `ensure` / narrowing / `Result` / nominal-error-union infrastructure, pros/cons, and a recommendation.

## See also

- [ROADMAP.md](../../../../ROADMAP.md#control-flow--statements) — `switch` row, ✅ done; `match` deferred to a future RFC.
- [guard.md](../guard/guard.md) — type guards, `is`, `ensure`; the narrowing substrate `match` would reuse.
- [SEMANTICS_NARROWING.md](../../../../forst/internal/typechecker/SEMANTICS_NARROWING.md) — soundness/join rules for narrowing that any `match` design must respect.
- [union_error_narrowing.ft](../../../union_error_narrowing.ft) — today's `if x is Err(ParseError)` narrowing over a closed nominal-error union; the running example used throughout the RFC.
