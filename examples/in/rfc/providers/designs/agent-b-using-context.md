---
Feature Name: providers
Design: Agent B — implicit context with `using`
Start Date: 2026-06-27
---

# Function Requirements — Agent B: `using` Context

## Summary

Agent B treats **requirements** as named, structurally-typed capabilities (interfaces) that functions **pull from an implicit context** via the `using` keyword. Call sites stay clean: parameters carry **data**, context carries **runtime behavior**. The compiler infers each function’s requirement set from `using` bindings, threads a single Go struct through the call graph, and keeps everything statically traceable.

**New surface area:** one primitive (`requirement`) and one keyword (`using`). No imports, no service locators, no effect rows in signatures.

## Design Philosophy

| Principle | Agent B choice |
|-----------|----------------|
| Parameters = data | Function parameters remain plain values (`Order`, `String`, …). |
| Requirements = behavior | Clocks, loggers, DB handles, HTTP clients live in context. |
| Go rule | If a struct field solves it, use a struct field — not DI frameworks. |
| Readability | `using logger := Logger` reads like “this block uses Logger”. |
| Testability | Tests override context bindings with `using … with …` at the top of a block. |
| Traceability | Every `using` is emitted as a named struct field; no globals. |

Inspired by Unison abilities and Roc’s context scopes, but lowered to idiomatic Go: **one requirements struct per function**, passed as the first parameter (or embedded in an existing handler struct when present).

---

## 1. Core Primitive(s)

### Primitive: `requirement`

A **requirement** is a named capability — a shape of methods the runtime must supply. It is *not* a value type; it is a contract.

```forst
requirement Logger {
    info(msg String)
    error(msg String)
}

requirement Clock {
    now(): Int
}

requirement Database {
    query(sql String): Result([]Row, Error)
}
```

Requirements are package-scoped (like `type` and `error`). They may reference other named types and `Result`. Methods use the same function syntax as shape fields.

### Keyword: `using`

Inside a function body, **`using`** binds a requirement from the caller’s context into a local name:

```forst
using <name> := <RequirementIdent>
```

- `<name>` is optional when it equals the lowercased requirement name (`using Logger` ≡ `using logger := Logger`).
- Bindings are **lexically scoped** to the rest of the function (and nested blocks/functions that inherit context).
- A function’s **requirement set** is the union of all requirements referenced by `using` in its body (directly or via transitive calls — see inference).

No second keyword. Test/production overrides reuse `using` with a **`with`** clause (not a keyword — parsed as part of the `using` statement):

```forst
using clock := Clock with FakeClock { fixedMs: 1_700_000_000_000 }
```

---

## 2. Syntax Examples

### 2.1 Basic usage

```forst
package orders

requirement Logger {
    info(msg String)
}

requirement Clock {
    now(): Int
}

func expireToken(token Token): Result(Token, Error) {
    using logger := Logger
    using clock := Clock

    if token.expiresAt < clock.now() {
        logger.info("token expired")
        return Err(Expired { tokenId: token.id })
    }
    return Ok(token)
}
```

Call site — **no requirement arguments**:

```forst
func handleRefresh(body RefreshBody, ctx RequestContext): Result(Session, Error) {
    token, err := parseToken(body.token)
    ensure err is Nil() or err

    return expireToken(token)   // ctx supplies Logger + Clock implicitly
}
```

### 2.2 Context struct with requirements (Go-friendly)

When a handler already has a context struct, requirements are **fields**, not parallel DI:

```forst
type RequestContext = {
    requestId: String,
    logger: Logger,      // satisfies requirement Logger
    clock: Clock,
    db: Database,
}

func handleOrder(req OrderRequest, ctx RequestContext): Result(Receipt, Error) {
    using logger := Logger from ctx
    using db := Database from ctx

    logger.info("order " + req.id)
    return saveOrder(req, ctx)
}

func saveOrder(req OrderRequest, ctx RequestContext): Result(Receipt, Error) {
    using db := Database from ctx   // same field, no duplicate parameter
    ...
}
```

`from ctx` is optional sugar when the context parameter is named `ctx` and the field name matches the requirement (`ctx.logger` for `Logger`).

### 2.3 Block-scoped `using`

For localized capability use (e.g. a transaction):

```forst
func transfer(from String, to String, amount Int) {
    using db := Database
    using tx := Database.Transaction from db {
        debit(from, amount)
        credit(to, amount)
        tx.commit()
    }
}
```

The block inherits outer `using` bindings; inner `using` adds to the scope only inside `{ … }`.

### 2.4 Production wiring (entry point)

```forst
func main() {
    ctx := AppContext {
        logger: StdLogger { level: "info" },
        clock:  SystemClock {},
        db:     PostgresDatabase { dsn: env("DATABASE_URL") },
    }
    runServer(ctx)
}
```

`AppContext` fields are **concrete implementations** whose shapes satisfy the corresponding `requirement` declarations (structural subtyping, same as shape guards).

### 2.5 Tests — override with `with`

```forst
test "expireToken rejects expired token" {
    using clock := Clock with FakeClock { fixedMs: 2000 }
    using logger := Logger with NopLogger {}

    result := expireToken(Token { id: "t1", expiresAt: 1000 })
    ensure result is Err(Expired {})
}

test "expireToken accepts valid token" {
    using clock := Clock with FakeClock { fixedMs: 500 }

    result := expireToken(Token { id: "t1", expiresAt: 1000 })
    ensure result is Ok(_)
}
```

Tests do not need a framework-specific mock registry: they **rebind context** at the block top, same syntax as production, different implementations.

### 2.6 Fake implementations (minimal boilerplate)

```forst
type FakeClock = {
    fixedMs: Int,
}

func (c FakeClock) now(): Int {
    return c.fixedMs
}

type NopLogger = {}

func (l NopLogger) info(msg String) {}
func (l NopLogger) error(msg String) {}
```

No code generation, no registration macro — satisfy the requirement structurally.

---

## 3. Go Transpilation Strategy

### 3.1 Requirement → Go interface

Each `requirement R { … }` becomes a Go interface in the same package:

```go
type Logger interface {
    Info(msg string)
    Error(msg string)
}

type Clock interface {
    Now() int64
}
```

Method names are exported; Forst `info` → Go `Info` (existing naming rules).

### 3.2 Function requirement set → struct

For function `expireToken` with requirements `{ Logger, Clock }`, the compiler synthesizes:

```go
type expireTokenReq struct {
    Logger Logger
    Clock  Clock
}
```

**Lowering rule:** the requirements struct is passed as the **first parameter**:

```go
func expireToken(req expireTokenReq, token Token) (Result[Token, error], error) {
    if token.ExpiresAt < req.Clock.Now() {
        req.Logger.Info("token expired")
        return Err(Expired{TokenId: token.Id}), nil
    }
    return Ok(token), nil
}
```

This is strictly Go-compatible: plain structs and interfaces, no reflection, no `context.Context` magic unless the user chooses to put values there.

### 3.3 Call-site lowering

When `handleRefresh` calls `expireToken(token)` and its own requirement set is a superset `{ Logger, Clock, … }`:

1. Compiler verifies `handleRefresh`’s context includes `Logger` and `Clock`.
2. Emits a slice or sub-struct extraction:

```go
func handleRefresh(ctx handleRefreshReq, body RefreshBody) (...) {
    ...
    return expireToken(expireTokenReq{
        Logger: ctx.Logger,
        Clock:  ctx.Clock,
    }, token)
}
```

When the caller already carries a named `RequestContext` struct, **reuse fields** — do not nest DI containers. If `RequestContext` embeds or fields-match requirements, the transformer passes `ctx` fields directly (no duplicate wrapper type at the handler boundary).

### 3.4 Context parameter merging

If a function has `(req Data, ctx RequestContext)` and `RequestContext` already contains requirement fields, the compiler **does not** add a second synthetic req struct at the public API. Internal helpers may still use synthesized `fooReq` structs for functions without a shared context type.

Priority:

1. Explicit context struct field (`ctx.logger: Logger`) — preferred at HTTP/handler edges.
2. Synthesized `fnReq` struct — for leaf/internal functions.
3. Test `using … with` — builds a literal `fnReq` at block entry.

### 3.5 Performance

- **Zero allocation** on hot paths when the same context struct is threaded pointer-free.
- Sub-struct extraction at call sites is a struct literal copy of interface words (same cost as explicit parameter passing).
- No global mutable state; inlining-friendly.
- Optional future optimization: merge identical `fnReq` types across functions in a package to reduce duplication (ABI-stable within package).

### 3.6 Traceability

Every requirement use maps to:

- a struct field name in generated Go, and
- a source `using` span in diagnostics.

Stack traces and profiles show ordinary Go calls — no hidden runtime.

---

## 4. Inference Rules

### 4.1 Intra-function

1. Parse all `using` statements in the function body (including nested blocks).
2. Collect distinct requirement identifiers → `fnRequirements`.
3. Each `using` binding must resolve to a requirement declared in the package or a built-in requirement (future: `Env`, `Random`).

### 4.2 Transitive requirements (call graph)

If `A` calls `B`, and `B` needs `{ Logger, Clock }`:

- `A` must expose `{ Logger, Clock }` in its context (direct `using` or via context parameter fields).
- The typechecker propagates **required ⊆ available** at each call site.
- If `A` lacks `Clock`, error at the call to `B`: *"`expireToken` needs `Clock`; add `using clock := Clock` or a `ctx.clock` field"*.

No annotation on `B`’s signature in source — requirements are **inferred**, not written twice.

### 4.3 Context struct compatibility

`RequestContext` satisfies a requirement set `{ Logger, Clock, Database }` when:

- for each requirement `R`, there is a field `f` such that `typeof(f)` structurally implements `R` (same rules as shape guards / method sets), and
- optional `using X from ctx` resolves field by convention: `Logger` → `logger`, `Database` → `db` (lower camel case), overridable by explicit `from ctx.db`.

### 4.4 Entry points

Functions designated as entry points (`main`, exported sidecar handlers, test blocks) must have requirements **fully bound**:

- production: all fields populated on `AppContext` or synthesized req at startup;
- tests: all requirements bound via `using … with` or explicit context literal.

Unbound requirements at an entry point → compile error.

### 4.5 No import inference

Requirements are discovered from **package declarations** and **use sites**, not from imports. Same package merge rules as other top-level decls.

### 4.6 Interaction with `ensure` / `Result`

Requirements do not participate in `ensure` failure typing. A missing requirement is a **compile-time** error, not a runtime `Result` — mirroring Go’s “missing dependency” as build failure, not panics.

---

## 5. Tradeoffs vs Explicit Provide

Comparison baseline: **explicit provide** (Agent A-style) — requirements listed in the function signature or a `provide` clause at call sites.

| Topic | Agent B (`using`) | Explicit provide |
|-------|-------------------|------------------|
| Call-site noise | Low — `expireToken(token)` | Higher — `expireToken(token, provide { … })` |
| Discoverability | Jump to body / IDE “requirements” lens | Visible in signature |
| Go mapping | Hidden first `req` param | Often identical Go lowering |
| Refactoring | Adding `using Database` updates inference chain | Same, but signature churn if explicit |
| Learning curve | Must learn context threading | Must learn provide syntax |
| Handler APIs | Natural fit: `ctx` struct fields | Risk of duplicating provide + ctx |
| LLM test gen | `using … with Fake…` at test top is uniform | Explicit literals also easy |
| Over-use risk | Easy to hide heavy deps | Slightly more visible |

**When Agent B wins:** application code with a stable handler context; deep call trees; alignment with “struct field, not DI”.

**When explicit provide wins:** library functions meant to be dependency-transparent; public APIs where requirements are part of the contract surface.

**Hybrid (recommended norm):** infer + `using` internally; allow optional **`requires` clause** on exported functions for documentation only (does not change lowering). Not part of Agent B’s minimal keyword budget but compatible later.

---

## 6. Edge Cases

### 6.1 Requirement shadowing

Inner block `using logger := Logger with NopLogger {}` shadows outer `logger` only inside the block. Go lowering uses distinct local variables or block-scoped struct literals — no dynamic scope.

### 6.2 Duplicate requirements

Two `using` lines for the same requirement in one scope → error: *"`Logger` already bound"*.

### 6.3 Optional requirements

Not in v1. Use explicit `Option`-typed fields on context or split functions. Partial capabilities invite runtime nil checks Agent B avoids.

### 6.4 Circular call graphs

`A → B → A` with incompatible requirement sets → standard SCC analysis; error if a cycle cannot be satisfied with a single context struct.

### 6.5 Same requirement, different implementations in one function

Rare (e.g. two loggers). v1: require distinct requirement names (`AuditLogger`, `AppLogger`). Method merging / tagging is out of scope.

### 6.6 Go interop callbacks

Go code calling exported Forst functions must pass the synthesized `fnReq` struct as the first argument. Sidecar/TS clients continue to pass JSON **data** only; the sidecar host builds `AppContext` once per request (existing pattern in handler RFCs).

### 6.7 Concurrency

Requirements structs are passed by value (interfaces inside). Safe if implementations are goroutine-safe — same contract as Go today. Block-scoped `Database.Transaction` must not escape the block; compiler may emit escape analysis hints via scoped struct literal (future).

### 6.8 Unused `using`

Warning: *bound `Logger` never used* — same severity as unused variable.

### 6.9 Conflicts with data parameters

Requirement names must not collide with parameter names in the same function. `using db := Database` when parameter `db` exists → error.

### 6.10 Empty requirement set

Functions with no `using` compile to ordinary Go functions — no synthetic struct. Requirements are opt-in per function.

---

## 7. Compiler Phases (sketch)

1. **Parse** — `RequirementDecl`, `UsingStmt` AST nodes.
2. **Collect** — per-function requirement sets from `using` (+ transitive calls, fixed point).
3. **Check** — structural implementability of context fields; entry-point binding completeness.
4. **Transform** — emit interfaces, `fnReq` structs, call-site struct literals; map `using` identifiers to `req.Field` accesses.

Diagnostics anchor on the `using` line for missing bindings and on call sites for propagation failures.

---

## 8. Relation to Existing Forst Features

| Feature | Interaction |
|---------|-------------|
| `ensure` / `is` | Orthogonal — requirements are compile-time wired, not validated at runtime. |
| Shape guards | Same structural typing for “implements requirement”. |
| `Result` / errors | Requirement methods may return `Result`; requirement wiring itself is not fallible. |
| Sidecar handlers | `RequestContext` at boundary; sidecar executor constructs context per invoke. |
| Effect RFC | Requirements are **capabilities**, not algebraic effects; no `Effect` monad required. |

---

## 9. Open Questions

1. **`requires` export clause** — documentation-only vs enforced visibility?
2. **Built-in requirements** — ship `Env`, `Random`, `Stdout` as predeclared in `forst` prelude?
3. **Name mangling** — merge identical requirement sets to one Go struct type per package?
4. **LSP** — “Show requirements for function” code lens on `func` keyword?

---

## 10. Minimal Example (end-to-end)

**Forst**

```forst
package demo

requirement Logger {
    info(msg String)
}

func greet(name String) {
    using logger := Logger
    logger.info("hello " + name)
}

func main() {
    ctx := { logger: StdLogger {} }
    greet("world")   // inferred: ctx.logger satisfies Logger
}
```

**Generated Go (conceptual)**

```go
type Logger interface { Info(msg string) }

type greetReq struct { Logger Logger }

func greet(req greetReq, name string) {
    req.Logger.Info("hello " + name)
}

type mainContext struct { Logger Logger }

func main() {
    ctx := mainContext{Logger: StdLogger{}}
    greet(greetReq{Logger: ctx.Logger}, "world")
}
```

---

## Verdict

Agent B delivers **one primitive, one keyword**, keeps call sites free of dependency boilerplate, maps cleanly to **Go struct fields and interfaces**, and gives tests a **single, readable pattern** (`using … with`). The cost is implicit context threading — mitigated by strict compile-time inference, field-traceable lowering, and optional exported `requires` documentation in a later phase.
