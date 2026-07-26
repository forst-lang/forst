---
Feature Name: switch-and-match
Start Date: 2026-07-20
---

# `switch` vs `match`

## Summary

`switch` / `case` / `default` / `fallthrough` are lexed but have no AST, typechecker, or Go-emit support (`ROADMAP.md` marks the row 📋 **planned**, "design open: may track Go's `switch` closely, or lean toward a `match`-style construct"). This RFC resolves that question by comparing three options — **(A)** Go-faithful `switch`, **(B)** a new `match` expression/statement layered on Forst's existing `is`/narrowing substrate, **(C)** both — against prior art in Go, Rust, Swift, Kotlin, TypeScript, and Scala, and against Forst's actual narrowing implementation today.

**Recommendation:** ship **(A)** Go-faithful `switch` first (see [Decision](#decision)), track `match` as a distinct, separately-scoped future RFC rather than blocking `switch` on it.

## Motivation

Two different jobs get conflated under "we need `switch`":

1. **Backwards-compat job:** teams porting hand-written Go to Forst have existing `switch` statements (tag switches on strings/ints, `switch { case cond1: ...}` no-tag form, `switch v := x.(type)`) that should port mechanically, the same way `for`/`if`/`defer` do today. This is a **translation fidelity** problem — the migration story falls apart if `switch` is the one common statement that needs manual rewriting.
2. **Pattern-matching job:** Forst already has `is`, shape guards, `Result(S, F)` + `Ok()`/`Err()` narrowing, and closed nominal-error unions (`type ErrKind = ParseError | IoError`) that behave like a discriminated sum. Chained `if x is Foo() { } else if x is Bar() { } else { }` is the same shape as a `match` over that sum, just without exhaustiveness checking or a dedicated syntax. This is a **language-native ergonomics** problem, orthogonal to Go source compatibility.

Conflating them either forces Go's `switch` to grow narrowing semantics it was never designed for, or forces a new `match` construct to also cover Go idioms (tag switch on strings, fallthrough) that don't map cleanly onto pattern matching. Splitting them lets each stay faithful to its job.

## Prior art

| Language | Construct | Exhaustiveness | Fallthrough | Guard clauses | Narrows bound var | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| **Go** | `switch` (tag or no-tag), `switch v := x.(type)` | No (not enforced) | Explicit `fallthrough`, opt-in per case | `case x > 5:` in no-tag form | Only in `.(type)` form (`v` per case) | Cases don't auto-break in other C-family langs, but Go cases do; only `fallthrough` cascades. |
| **Rust** | `match` (expression) | **Yes**, compiler-enforced (or explicit `_`) | N/A (no fallthrough concept) | `if` guards on arms (`Some(x) if x > 5`) | Yes — binds and narrows per arm via destructuring | Arms are values; `match` is an expression, not a statement. |
| **Swift** | `switch` (statement/expr-ish) | **Yes**, compiler-enforced | No implicit fallthrough; explicit `fallthrough` keyword available | `case let x where x > 5:` | Yes via `case let`/pattern binding | Closest hybrid: Go-shaped keyword, Rust-shaped exhaustiveness + binding. |
| **Kotlin** | `when` (expression or statement) | Enforced only when used as an expression over a sealed type | N/A | `x in 1..5 ->` arbitrary conditions per branch | Yes via smart-cast on sealed-class branches | `when` unifies tag switch and type-based dispatch under one keyword. |
| **TypeScript** | `switch` (statement, Go/C-shaped) + discriminated-union narrowing via control flow analysis (no dedicated match keyword) | No language enforcement; `never` exhaustiveness is a userland pattern (`assertNever`) | Yes, C-style fallthrough (implicit unless `break`) | No | Yes, via control-flow narrowing on `case` literal comparisons against a tag field | TS deliberately kept `switch` C-shaped and put exhaustiveness in the type checker as a pattern, not a keyword. |
| **Scala** | `match` (expression) | Warned (not enforced) unless `sealed` hierarchy | N/A | `case x if x > 5 =>` | Yes via full destructuring/patterns | Pattern matching on case classes; heaviest pattern-matching investment. |

Two clusters emerge: **C-family tag switches** (Go, TS, pre-Swift-3 feel) prioritize *statement-shaped, mechanical translation*; **pattern-matching constructs** (Rust, Scala, Kotlin `when`, Swift `switch`) prioritize *exhaustiveness and binding on sums*. Swift is the only language that tried to make one keyword do both, and it needed extra syntax (`case let … where`) to get there — it didn't get pattern matching "for free" out of a C-style switch.

## How each option fits Forst's existing infrastructure

Forst already has a narrowing pipeline: `is` assertions, `ensure … is … or …`, `if x is Foo() { }` branch-scoped narrowing (`forst/internal/typechecker/infer_if.go`, `narrow_if.go`), a join policy at branch exit (`SEMANTICS_NARROWING.md`), and closed nominal-error unions that already narrow via `if x is Err(ParseError)` (see [`union_error_narrowing.ft`](../../../union_error_narrowing.ft)). Any `match` design does **not** get to invent a second narrowing engine — it has to sit on top of `infer_if.go`'s branch-scope model, or that model needs generalizing first (join-across-N-arms, not just if/else-if/else, is the same problem either way).

| Option | Fits today's narrowing engine? | Fits Go-emit story? | New typechecker surface | New AST/parser surface |
| --- | --- | --- | --- | --- |
| **(A) Go-faithful `switch`** | No interaction — it's a plain multi-way branch on values, same job as Go's; doesn't touch `is`/narrowing at all except an optional `switch v := x.(type)` form, which *does* overlap with narrowing (see below) | Direct 1:1 emit to `go/ast.SwitchStmt`; no new Go idioms invented | Small: per-case expression typecheck, duplicate-case-literal detection (`go vet` parity), tag-type vs case-type compatibility | Small: `SwitchStmt`/`CaseClause` nodes, reuse existing expression/block parsing |
| **(B) `match` (pattern-matching)** | Directly reuses/extends `infer_if.go`'s narrowing — each arm is structurally an `is` check with its own branch scope; needs generalizing the if/else-if/else join to N arms plus **exhaustiveness** analysis over closed unions (`ErrKind = ParseError \| IoError`), `Result(S,F)`, and shape guards | No Go equivalent; emits as a **generated** `if`/`else-if` chain (or Go `switch v := x.(type)` where the sum happens to be interfaces) — i.e. `match` is sugar, not a new Go primitive | Larger: exhaustiveness checker over the union/shape-guard type algebra (shared with future binary-type-expression work per `ROADMAP.md`'s "narrowing and future optionals should share one internal type algebra") | Larger: new `MatchExpr`/`MatchArm` nodes with pattern syntax (literal, shape destructure, guard clause) |
| **(C) Both** | Each stays scoped to its own job | Each stays scoped to its own job | Sum of A + B, done independently | Sum of A + B, done independently |

The key finding: **`match` is not an alternative spelling of `switch` in Forst** the way Swift tried to make it one keyword. Forst's `is`/narrowing pipeline already *is* most of a `match` — a real `match` construct here is closer to "give the existing `if x is A() / else if x is B()` chain a dedicated, exhaustiveness-checked syntax" than "port a C-style switch." Building `match` teaches nothing that helps with Go's tag-switch semantics (duplicate-case checks, fallthrough, no-tag boolean-case form), and building Go's `switch` teaches nothing that helps with exhaustiveness over sums. They are genuinely separable projects with separable ROI.

## `switch v := x.(type)` — the one overlap

Go's type switch is the one place the two options collide: it is syntactically a `switch`, but semantically a narrowing construct (each `case T:` arm binds `v: T`). Forst's anti-feature stance already routes around Go's `x.(T)` (ROADMAP: "Type assertions `x.(T)`, type switch — 📋 planned... Distinct from Forst's `is`/`ensure`/narrowing"). Recommendation: **do not implement `switch v := x.(type)` as Forst surface syntax.** Go-interop callers needing a real Go type switch keep it in hand-written `.go` (same escape hatch as `unsafe`); Forst-native code expresses the same intent with `if x is Foo() / else if x is Bar()` today, and with `match` if/when built.

## Pros/cons

### (A) Go-faithful `switch` (recommended, now)

**Pros:**

- Mechanical Go→Forst translation for the single most common Go control-flow form not yet supported — directly serves "migrations should be seamless."
- No new semantics to design: copy Go's spec (tag switch, no-tag switch, `fallthrough`, multiple values per `case`, `default`).
- Small, boundable surface: parser (`SwitchStmt`/`CaseClause`), typechecker (tag/case compatibility + duplicate-literal diagnostics), 1:1 Go emit. No interaction with narrowing, generics, or the error-union work in flight.
- Unblocks real-world Go files with `switch` today (very common in Go, e.g. any dispatch-on-error-kind, dispatch-on-enum-like-const code) without waiting on a bigger design.

**Cons:**

- Inherits Go's own weaknesses: no exhaustiveness checking, `fallthrough` is an easy footgun, no binding/destructuring — this RFC's authors won't fix Go's `switch` design, only port it.
- Slight tension with "no unpredictable behavior": `fallthrough`'s implicit-unless-declared-otherwise semantics is a real deviation from Forst's general preference for explicit control flow — mitigated by requiring `fallthrough` as an explicit keyword (same as Go; nothing implicit at the token level) and documenting it as a Go-interop concession, not a Forst-native idiom to encourage.

### (B) `match` (pattern matching, future/separate)

**Pros:**

- Turns an existing but verbose pattern (`if x is A() {} else if x is B() {} else {}`) into a first-class, exhaustiveness-checked construct — meaningfully better ergonomics for the union/`Result`/nominal-error work already in flight.
- Naturally extends `is`-based narrowing rather than inventing new semantics from scratch.
- Prior art (Rust, Swift, Kotlin) shows this is a well-understood, well-loved feature when unions are closed/sealed — which Forst's nominal-error unions already are.

**Cons:**

- Large surface: needs the union/shape-guard exhaustiveness algorithm, N-arm join generalization of `infer_if.go`, new pattern grammar (literal patterns, shape-destructure patterns, guard clauses `if`), and a decision on expression-vs-statement form.
- Directly depends on/should be sequenced after the in-flight "binary types + narrowing share one type algebra" work (`ROADMAP.md`), and ties into user generics for `Result[S, F]` narrowing (see [generics RFC §10](../generics/00-user-generics-and-type-parameters.md#result-types-generics-and-narrowing-ok-and-err)) — building it now means building on moving ground.
- Zero contribution to the "port existing Go code" goal this planning session is specifically about — it's a language-native feature, not a compat feature.

### (C) Both, same milestone

**Pros:** Complete story, no follow-up decision needed later.

**Cons:** No shared implementation cost is saved by doing them together (see fit table above — they don't share AST/typechecker/emit machinery beyond the lexer's existing tokens); bundling them only delays shipping `switch` behind `match`'s much larger design/exhaustiveness work, which directly contradicts the "seamless migration first" goal driving this RFC.

## Decision

Ship **(A) Go-faithful `switch` / `case` / `default` / `fallthrough`** as part of the Go-backwards-compatibility Tier 1 work (this session's plan). Do **not** implement `switch v := x.(type)` — keep Go type switches as a hand-written-`.go` escape hatch, matching the existing `x.(T)` stance.

Track **(B) `match`** as an explicitly separate, future RFC scoped to the narrowing/union work already referenced in `ROADMAP.md` ("narrowing and future optionals should share one internal type algebra"); do not schedule it opportunistically inside `switch` work, since the two share almost no implementation surface (see fit table).

## Unresolved questions (for the future `match` RFC, not blocking `switch`)

1. Expression or statement? Rust/Scala make `match` an expression (used for its value); Kotlin's `when` supports both. Forst's function-centric, explicit-return style (no implicit last-expression returns) leans toward **statement**-only `match`, consistent with `if`/`for` today — but this needs its own RFC discussion, not a decision bundled into `switch`.
2. Pattern grammar: literal patterns (`match x { 1 => ... }`) are easy; shape-destructure patterns (`match x { { name, age } => ... }`) need to reuse shape-guard machinery; how far to go before it becomes its own mini type-level language (would violate "no type-level computation" if pushed too far).
3. Exhaustiveness enforcement: hard-error like Rust, or warning like Scala? Given Forst's "predictable, deterministic behavior" principle, hard-error over closed unions (nominal-error unions, `Result(S,F)`) is the likely answer, but open/unsealed cases (arbitrary shapes, `interface{}`) need a documented non-exhaustive fallback (`_` / `default` arm required).

## References

- [ROADMAP.md](../../../../ROADMAP.md#control-flow--statements) — `switch` row.
- [guard.md](../guard/guard.md) — `is` type guards this RFC's `match` option would build on.
- [SEMANTICS_NARROWING.md](../../../../forst/internal/typechecker/SEMANTICS_NARROWING.md) — join/soundness rules any N-arm `match` must respect.
- [union_error_types.ft](../../../union_error_types.ft), [union_error_narrowing.ft](../../../union_error_narrowing.ft) — today's closed nominal-error unions and their `if x is Err(...)` narrowing.
- [generics RFC §10](../generics/00-user-generics-and-type-parameters.md#result-types-generics-and-narrowing-ok-and-err) — `Result[S,F]` narrowing dependency for a future `match`.
- Go spec — [`SwitchStmt`](https://go.dev/ref/spec#Switch_statements); Rust Reference — [`match` expressions](https://doc.rust-lang.org/reference/expressions/match-expr.html); Swift Language Guide — [`switch`](https://docs.swift.org/swift-book/documentation/the-swift-programming-language/controlflow/#Switch); Kotlin docs — [`when`](https://kotlinlang.org/docs/control-flow.html#when-expression); TypeScript Handbook — [narrowing](https://www.typescriptlang.org/docs/handbook/2/narrowing.html).

## Document status

**Decision recorded.** `switch` is in scope for the Go-backwards-compatibility implementation plan; `match` is deliberately deferred to its own future RFC.
