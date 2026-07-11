---
Feature Name: providers
Design: Agent A — explicit `require` + `provide`
Start Date: 2026-06-27
Status: Design proposal (competition entry)
---

# Function Requirements — Agent A: Explicit `require` + `provide`

## Summary

Agent A makes **runtime capabilities** first-class in Forst with two keywords — **`require`** and **`provide`** — and one primitive declaration form, **`requirement`**. Functions declare what they need with **`require`** (in the body or an optional signature clause); callers satisfy those needs with **`provide`** at **boundaries** (entry points, tests, HTTP wiring). Internal call chains inherit an already-provided bundle — no re-provision at every hop.

The design reads like **`ensure`**: a short, English verb at the point of use. Wiring mirrors **Effect-TS `Effect.provide` / `provideService`**: implementations are chosen explicitly at the edge, then flow inward. Everything lowers to **Go interfaces + a struct field** — no DI framework, no reflection, no scope globals.

**New surface area:** one primitive (`requirement`) and two keywords (`require`, `provide`).

**Depends on:** [prior art](../00-prior-art.md), [guard / `ensure`](../../guard/guard.md), [errors normative](../../errors/02-first-class-errors-normative.md), [simple effect CLI](../../effect/13-simple-effect-cli.md).

**Competes with:** [Agent B — implicit `using`](agent-b-using-context.md), [Agent C — `?` / `@require` operators](agent-c-operator-syntax.md).

---

## Design philosophy

| Principle | Agent A choice |
| --- | --- |
| **Parameters = data** | Function parameters carry command inputs (`Order`, `Int`, `String`). |
| **Requirements = runtime logic** | Loggers, clocks, DB handles, HTTP clients are capabilities — not struct-field DI. |
| **Explicit at the edge** | **`provide`** chooses implementations in `main`, tests, and sidecar bootstrap — never implicit globals. |
| **Readable like `ensure`** | **`require Logger`** is a vertical, scan-friendly statement — not operator soup or hidden context. |
| **Effect.provide-shaped** | Satisfying a requirement **shrinks** the unsatisfied set; tests swap fakes with one `provide` block. |
| **Go rule** | If a struct field solves it, use a struct field — synthesized **`Deps`** bundles, passed as first param. |
| **LLM-friendly tests** | Read `requirement` block → implement methods → `provide R: FakeR { … }` — no container API. |

---

## 1. Core primitive and keywords

### 1.1 Primitive: `requirement`

A **requirement** is a named capability contract — methods only, no instance data in the declaration. Same role as a Go interface or Effect `Context.Tag` service shape.

```ft
requirement Logger {
    info(msg String)
    error(msg String)
}

requirement Clock {
    now(): Int
}

requirement Database {
    query(sql String): Result([]Row, DbError)
}
```

Requirements are **package-scoped** (like `type`, `error`). Implementations are ordinary Forst types that **structurally satisfy** the method set (same rules as shape guards).

**Lowering:** each `requirement R` → Go `interface R` with exported method names (`info` → `Info`).

### 1.2 Keyword: `require`

**`require`** declares or resolves a capability dependency.

**Forms:**

| Form | Role |
| --- | --- |
| `require R` | Bind requirement **`R`** to local name **`r`** (lower-camel of `R`: `Logger` → `logger`). |
| `require name is R` | Bind **`R`** to explicit local name **`name`**. |
| `require R, Clock` | Multiple requirements in one statement (comma-separated). |
| `func f(...) require R, Clock` | Optional **signature clause** — documents and optionally constrains inferred needs. |

**In the body**, `require` is a **statement** (like `ensure`), not an expression:

```ft
func expireToken(token Token): Result(Token, Error) {
    require logger is Logger
    require clock is Clock

    if token.expiresAt < clock.now() {
        logger.info("token expired")
        return Err(Expired { tokenId: token.id })
    }
    return Ok(token)
}
```

Shorthand when the local name matches convention:

```ft
require Logger   // ≡ require logger is Logger
require Clock
```

**Semantics:** each `require` adds **`R`** to the function’s **need set** `Needs(f)` and introduces a local binding backed by the caller’s **provided bundle** (see §3).

### 1.3 Keyword: `provide`

**`provide`** supplies concrete implementations for one or more requirements.

**Forms:**

| Form | Role |
| --- | --- |
| **`provide R: impl, S: impl { … }`** | Block: install bindings for the dynamic extent of `{ … }`. |
| **`provide { R: impl, S: impl }`** | Block with brace map (Effect-style service record). |
| **`f(args, provide { R: impl })`** | Call-site suffix when the caller is not already inside a `provide` block. |

```ft
provide Logger: NopLogger {}, Clock: FakeClock { fixedMs: 1_700_000_000_000 } {
    result := expireToken(token)
    ensure result is Err(Expired {})
}
```

Call-site suffix (explicit wiring at a single call):

```ft
receipt := saveOrder(order, provide { Database: txDb, Logger: reqLogger })
```

**Semantics:** `provide` **satisfies** requirements for the block or call. Unsatisfied requirements after `provide` → compile error at the enclosing entry point.

---

## 2. Syntax examples

### 2.1 Function definitions — inferred needs

No signature clause required; the compiler infers `Needs(f)` from `require` statements:

```ft
package orders

requirement Logger {
    info(msg String)
}

requirement Clock {
    now(): Int
}

func expireToken(token Token): Result(Token, Error) {
    require Logger
    require Clock

    if token.expiresAt < clock.now() {
        logger.info("token expired")
        return Err(Expired { tokenId: token.id })
    }
    return Ok(token)
}
```

**Inferred metadata (conceptual):** `expireToken(token Token) Result(Token, Error) needs Logger, Clock`.

### 2.2 Optional signature clause — exported contract

For public APIs and sidecar exports, authors may write needs explicitly:

```ft
func expireToken(token Token): Result(Token, Error) require Logger, Clock {
    require Logger
    require Clock
    ...
}
```

**Rules:**

- Body `require` set **must equal** signature clause set (order irrelevant).
- Mismatch → compile error: *"body requires `Clock` but signature omits it"* / *"`require Logger` unused in body"*.

Signature-only documentation without body `require` is **not** allowed — `require` must appear where the capability is **used**, keeping bodies grep-able.

### 2.3 In-body obtain — chained internal calls

Once an ancestor `provide` block or handler bundle satisfies `Logger` and `Clock`, **internal calls do not repeat `provide`**:

```ft
func handleRefresh(body RefreshBody): Result(Session, Error) require Logger, Clock {
    require Logger

    token, err := parseToken(body.token)
    ensure err is Nil() or err

    return expireToken(token)   // inherits Logger + Clock from scope bundle
}

func parseToken(raw String): Result(Token, Error) {
    require Logger
    logger.info("parsing token")
    ...
}
```

`parseToken` needs only `Logger`; `handleRefresh` needs `Logger, Clock` (transitive union). The checker verifies **`Needs(caller) ⊇ Needs(callee)`** at each call.

### 2.4 Handler edge — explicit bundle struct (Go-friendly)

HTTP handlers may take a **named deps struct** instead of relying only on `provide` blocks. Fields satisfy requirements; **`require … from deps`** connects them (sugar, not a third keyword — parsed as part of `require`):

```ft
type RequestDeps = {
    logger: Logger,
    clock: Clock,
    db: Database,
}

func handleOrder(req OrderRequest, deps RequestDeps): Result(Receipt, Error) {
    require Logger from deps
    require Database from deps

    logger.info("order " + req.id)
    return saveOrder(req, deps)
}

func saveOrder(req OrderRequest, deps RequestDeps): Result(Receipt, Error) {
    require Database from deps
    ...
}
```

When the deps parameter is named `deps` and the field follows lower-camel convention (`Logger` → `deps.logger`), **`require Logger from deps`** is optional sugar — the compiler resolves `deps.logger` automatically after `require Logger`.

This matches the Go pattern in [13-simple-effect-cli.md](../../effect/13-simple-effect-cli.md): **`context.Context` for cancellation**, **`RequestDeps` for business capabilities** — never overloading `ensure` or `Result` for DI.

### 2.5 Production wiring — `main`

```ft
func main() {
    provide {
        Logger:   StdLogger { level: "info" },
        Clock:    SystemClock {},
        Database: PostgresDatabase { dsn: env("DATABASE_URL") },
    } {
        runServer()
    }
}
```

Sidecar / Go host equivalent:

```go
deps := orders.Deps{
    Logger:   stdLogger,
    Clock:    systemClock,
    Database: postgresDb,
}
http.HandleFunc("/refresh", orders.WrapHandleRefresh(deps))
```

### 2.6 Tests — mock instantiation (LLM-friendly)

Tests use the **same `provide` syntax** as production with different implementations:

```ft
test "expireToken rejects expired token" {
    provide {
        Clock:  FakeClock { fixedMs: 2000 },
        Logger: NopLogger {},
    } {
        result := expireToken(Token { id: "t1", expiresAt: 1000 })
        ensure result is Err(Expired {})
    }
}

test "expireToken accepts valid token" {
    provide Clock: FakeClock { fixedMs: 500 } {
        result := expireToken(Token { id: "t1", expiresAt: 1000 })
        ensure result is Ok(_)
    }
}
```

**Agent / LLM recipe:**

1. Read `requirement Clock { now(): Int }`.
2. Emit `FakeClock` with `now()` method.
3. Wrap test body in `provide Clock: FakeClock { … }`.
4. No mock registry, no code generation required for v1.

Minimal fakes:

```ft
type FakeClock = { fixedMs: Int }
func (c FakeClock) now(): Int { return c.fixedMs }

type NopLogger = {}
func (l NopLogger) info(msg String) {}
func (l NopLogger) error(msg String) {}
```

### 2.7 Call-site `provide` — localized override

When only one call needs a different implementation (e.g. audit logger on a single code path):

```ft
func auditAction(action String, deps RequestDeps) {
    require Logger from deps

    doAction(action, provide { Logger: AuditLogger { sink: deps.logger } })
}
```

The inner `provide` **shadows** `Logger` for the callee’s bundle only; outer scope unchanged after the call returns.

### 2.8 Orthogonality with `ensure` / `Result`

```ft
func loadUser(id Int): Result(User, Error) require Database {
    require Database

    row := db.query("SELECT …")
    ensure row is Found() or NotFound { id: id }
    return Ok(decodeUser(row))
}
```

| Construct | Handles |
| --- | --- |
| **`require` / `provide`** | **Compile-time** capability wiring (missing → build failure). |
| **`ensure` / `is`** | **Runtime** validation, narrowing, `Result` propagation on **data**. |

Requirements do **not** appear in `Result`’s type parameters (contrast Effect’s `Effect<A, E, R>`). Forst keeps binary `Result(S, F)` per [errors normative](../../errors/02-first-class-errors-normative.md).

---

## 3. Go transpilation strategy

### 3.1 Requirement → Go interface

```ft
requirement Logger {
    info(msg String)
    error(msg String)
}
```

```go
type Logger interface {
    Info(msg string)
    Error(msg string)
}
```

### 3.2 Need set → synthesized `Deps` struct

For each function with non-empty `Needs(f)`, the compiler synthesizes a **deps struct** (package-local, stable name **`Deps`** when shared, or **`expireTokenDeps`** when sets differ — merge pass deduplicates identical sets):

```go
type expireTokenDeps struct {
    Logger Logger
    Clock  Clock
}
```

**Lowering rule:** deps struct is the **first parameter**:

```go
func expireToken(deps expireTokenDeps, token Token) (Result[Token, error], error) {
    if token.ExpiresAt < deps.Clock.Now() {
        deps.Logger.Info("token expired")
        return Err(Expired{TokenId: token.Id}), nil
    }
    return Ok(token), nil
}
```

Functions with **empty** `Needs` emit ordinary Go signatures — zero overhead for pure helpers.

### 3.3 `require` lowering

| Forst | Go |
| --- | --- |
| `require Logger` | `logger := deps.Logger` (local binding) |
| `require db is Database` | `db := deps.Database` |
| `require Logger from deps` | `logger := deps.Logger` (when `deps` is a named param) |

Multiple `require` in one statement share the same `deps` parameter.

### 3.4 Internal call lowering

Caller already has `deps handleRefreshDeps` with `{ Logger, Clock, … }`. Callee needs subset `{ Logger, Clock }`:

```go
func handleRefresh(deps handleRefreshDeps, body RefreshBody) (...) {
    ...
    return expireToken(expireTokenDeps{
        Logger: deps.Logger,
        Clock:  deps.Clock,
    }, token)
}
```

**Same pointer-free struct literal** as explicit Go — no runtime set algebra. Inclusion **`Needs(callee) ⊆ Needs(caller)`** checked statically.

When caller uses **`RequestDeps`** with matching fields, the transformer **reuses fields directly** — no nested DI container at the handler boundary:

```go
func handleOrder(deps RequestDeps, req OrderRequest) (...) {
    deps.Logger.Info("order " + req.Id)
    return saveOrder(deps, req)
}
```

### 3.5 `provide` block lowering

```ft
provide {
    Logger: NopLogger {},
    Clock:  FakeClock { fixedMs: 2000 },
} {
    result := expireToken(token)
}
```

```go
func Test_expireToken_rejects() {
    deps := expireTokenDeps{
        Logger: NopLogger{},
        Clock:  FakeClock{FixedMs: 2000},
    }
    result := expireToken(deps, Token{Id: "t1", ExpiresAt: 1000})
    ...
}
```

For multi-function test blocks, the compiler merges **`Needs(union of block)`** into one literal.

### 3.6 Call-site `provide` suffix

```ft
saveOrder(order, provide { Database: txDb })
```

```go
saveOrder(saveOrderDeps{Database: txDb, Logger: outerDeps.Logger, ...}, order)
```

When the caller has scope deps, **`provide { Database: txDb }`** means **override `Database` only**; other fields copied from scope (compile error if scope lacks a required field).

### 3.7 Traceability guarantees

1. **Source grep:** `require Database` finds all resolution sites.
2. **Source grep:** `provide {` / `provide Database:` finds all wiring sites.
3. **Go grep:** `deps.Database` matches 1:1 (modulo synthesized struct name).
4. **Diagnostics:** missing requirement errors anchor on the **`require`** line; missing wiring on the **entry point** or **`provide`** block.

No `map[reflect.Type]interface{}`, no service locator, no init()-time globals.

### 3.8 Performance

- One struct pass-through per frame (interface words inside struct — same as explicit Go params).
- `provide` blocks are **compile-time** struct literals — no allocation pool.
- Inlining-friendly; stack traces show ordinary Go functions.

### 3.9 Discovery / sidecar metadata

Extend [`discovery.FunctionInfo`](../../../../forst/internal/discovery/discovery.go) with **`requirements: ["Logger", "Clock"]`** per export, derived from signature clause or inference. TypeScript `forst generate` can emit JSDoc *"@requires Logger, Clock"* for sidecar clients; the **host** constructs `Deps` once per process or request.

---

## 4. Inference rules

### 4.1 Collect `require` sites

Within function **`F`**, collect **`R(F) = { T | require mentions T in F’s body }`**.

### 4.2 Signature clause

If **`require T1, T2, …`** appears on the signature:

- **`R(F)` must equal** `{ T1, T2, … }`.
- Mismatch → error (see §2.2).

### 4.3 No signature clause

**Exported metadata** `Needs(F) = R(F)` (sorted lexicographically) attached to function type for docs, LSP, and discovery JSON.

### 4.4 Transitive propagation

When **`F`** calls **`G`**:

- **`Needs(F) ⊇ Needs(G)`** after fixed-point inference over the call graph.
- Violation at call site: *"`expireToken` needs `Clock`; add `require Clock` or satisfy via `provide` / deps field"*.

No annotation on **`G`**’s signature required if inference suffices; exported functions may add clause for documentation.

### 4.5 `provide` block scope

Inside **`provide { … } { body }`**:

- Let **`P`** = requirement types keyed in the map.
- For every function called transitively in **`body`**, **`Needs(called) ⊆ P`** must hold.
- **`P`** may be a **superset** (extra bindings ignored unless shadowing).

Entry into a nested function that needs **`Database`** when the block only provides **`Logger`** → compile error on the call inside the block.

### 4.6 ProviderScope deps from parameter

If **`F(deps RequestDeps, …)`** and **`RequestDeps`** fields structurally implement **`Needs(F)`**, the checker treats **`deps`** as satisfying those requirements without an enclosing `provide` block.

Field resolution convention: requirement **`Logger`** → field **`logger`** (lower camel); override with **`require Logger from deps.auditLog`**.

### 4.7 Entry points

Designated entry points (**`main`**, **`test`** blocks, exported sidecar handlers) must have **all** `Needs` satisfied:

- via enclosing **`provide`**, or
- via **`deps`** parameter with complete fields, or
- via call-site **`provide`** on the outermost invoke.

Unsatisfied needs at entry → compile error with wiring checklist: *"`handleRefresh` still needs `[Clock]` — add to `provide` or `RequestDeps`"*.

### 4.8 Cross-package calls

Package **`api`** calls package **`domain`**’s **`getUser`** needing **`Database`**. **`api`’s** inferred **`handleGetUser`** needs **`Database`** too. Each package emits its own **`Deps`** struct; **`main`** wires **`domain.Deps.Database`** once and passes into wrappers. No cross-package implicit context.

### 4.9 Subtyping

Concrete **`PostgresDatabase`** satisfies **`Database`**; deps field type is **`Database`** (interface), not the concrete type — standard Go.

### 4.10 Interaction with `ensure` narrowing

`require` bindings are **not** narrowed by `ensure`. Capabilities are stable for the function’s duration unless shadowed by an inner `provide`.

---

## 5. Why explicit `provide` beats implicit context for Forst

Comparison baseline: **implicit context threading** ([Agent B](agent-b-using-context.md) `using`, scope `ReqSet` without call-site visibility).

| Criterion | Agent A (`require` + `provide`) | Implicit context |
| --- | --- | --- |
| **Wiring visibility** | **`provide` blocks and suffixes** are grep-able test/production diffs | Bindings live in type inference / hidden first param |
| **Effect / TS alignment** | Same mental model as **`Effect.provideService`** for sidecar authors | Closer to Unison/Kotlin implicit context |
| **Library transparency** | **`require Database`** in body + optional signature clause documents deps without reading inference | Call sites look clean; deps discovered by IDE lens or errors |
| **Test locality** | **`provide FakeDb { … }`** wraps only the test — mocks colocated | **`using … with`** (Agent B) is similar; **`@require`** (Agent C) needs `with req` |
| **Go transpilation** | Struct literal at **`provide`** → identical to hand-written Go | Also lowers to struct — but **where** implementations enter is implicit |
| **Sidecar host** | Host **must** construct `Deps` — matches production checklist | Host may forget a field until runtime if inference-only |
| **Refactoring safety** | Adding **`require AuditLog`** forces **`provide`** / deps updates at **known boundaries** | Propagation errors at internal call sites — harder to grep "who wires AuditLog?" |
| **LLM test generation** | Pattern: read requirements → fake types → **`provide`** map | Implicit designs require "know the context threading model" |
| **Over-use risk** | **`provide`** ceremony discourages hiding deps | Easy to add **`using Database`** without updating handler deps docs |

**When Agent A wins (Forst-specific):**

1. **Go is the source of truth** — Forst transpiles to explicit struct wiring; **`provide`** makes that wiring **visible in source**, not only in emitted Go.
2. **Sidecar / Effect interop** — TS clients already think in **`provide`**; Forst backend code mirrors the same boundary.
3. **`ensure` culture** — Forst authors already read vertical keywords; **`require`** / **`provide`** extend that style without new sigils (`?`, `@`).
4. **Production debugging** — "Which implementation of `Logger` ran?" → search **`provide Logger:`** and **`main`** / handler setup, not every intermediate frame.
5. **No DI for mockable data** — test inputs stay **parameters**; only **behavior** goes in **`provide`**. Avoids the Go anti-pattern of stuffing mock **data** into service structs.

**Honest cost:** more characters at entry points and localized overrides than Agent B’s **`expireToken(token)`**. Agent A treats that as a **feature**: wiring is **documentation you can diff**.

**Hybrid compatibility:** Agent A accepts **`RequestDeps`** at handler edges (struct fields) — same as Agent B — while keeping **`provide`** as the canonical test and **`main`** pattern. Internal leaf functions use synthesized **`fnDeps`** structs.

---

## 6. Edge cases

### 6.1 Duplicate `require` in one scope

```ft
require Logger
require Logger
```

Error: *"`Logger` already required in this function"*.

### 6.2 Shadowing with inner `provide`

```ft
provide Logger: AppLogger {} {
    ...
    doAudit(provide { Logger: AuditLogger {} })
}
```

Inner call sees **`AuditLogger`**; outer block unchanged after call returns. Go lowering: distinct struct literals per call.

### 6.3 Partial `provide` at call site

```ft
f(x, provide { Database: txDb })
```

**Allowed** when caller has scope **`deps`** satisfying the rest of **`Needs(f)`**. Compiler emits merged literal. **Error** if neither scope nor suffix supplies a required type.

### 6.4 Optional requirements

**Not in v1.** Use explicit **`Option`**-typed fields on **`RequestDeps`**, or split functions. Partial wiring invites nil checks Agent A avoids at entry points.

### 6.5 Conditional `require`

```ft
if debug {
    require AuditLog
    auditLog.record(...)
}
```

**`AuditLog ∈ Needs(F)`** regardless of branch (static analysis, Go-style). Rationale: entry points must wire totally; conditional **use** must not hide deps from checklist.

**v2 escape hatch:** `require? AuditLog` or signature `require AuditLog optional` — omitted from entry wiring when nil allowed.

### 6.6 Same capability, two implementations in one function

Use **distinct requirement names**: **`AuditLogger`** vs **`AppLogger`**. v1 does not support two slots for one requirement type.

### 6.7 Name collisions

**`require db is Database`** when parameter **`db`** exists → error. Requirement locals are bindings, not parameters.

### 6.8 Unused `require`

Warning: *"`Logger` required but never used"* — same severity as unused variable.

### 6.9 Empty need set

Functions with no **`require`** compile to ordinary Go — no synthetic deps param.

### 6.10 Go interop — exported Forst functions

Go callers pass synthesized **`fnDeps`** as first argument. Document in generated Go headers:

```go
// Needs: Logger, Clock
func ExpireToken(deps ExpireTokenDeps, token Token) ...
```

### 6.11 Circular call graphs

**`A → B → A`** with incompatible need sets → SCC analysis; error if no single **`Deps`** struct satisfies the cycle.

### 6.12 Requirement methods must not `require`

Inside **`requirement`** declarations, methods list signatures only — no **`require`** (capabilities cannot pull scope deps at interface definition time).

### 6.13 Concurrency

Deps structs passed by value (interfaces inside). Implementations must be goroutine-safe — same contract as Go. **`provide Database: tx`** scoped to a block must not let **`tx`** escape — compiler may warn on escape (future).

### 6.14 Conflicts with reserved words

**`require`** is a new keyword — not overloaded from **`ensure`**. Prior art recommends **not** using **`ensure`** for DI ([§5.1 prior art](../00-prior-art.md)).

### 6.15 Failure modes

| Situation | Behavior |
| --- | --- |
| Missing **`provide`** at entry | Compile error + wiring checklist |
| Call-site **`provide`** missing field | Compile error at call |
| Signature / body `require` mismatch | Compile error on signature |
| **`Needs(caller) ⊉ Needs(callee)`** | Compile error at call |
| Nil implementation at runtime (host bug) | Panic or structured error (build-tag policy; v1 recommend panic in tests) |

---

## 7. Compiler phases (sketch)

1. **Parse** — `RequirementDecl`, `RequireStmt`, `ProvideStmt`, optional signature **`require`** clause, call-site **`provide`** suffix.
2. **Collect** — **`R(F)`** per function; fixed-point **`Needs(F)`** over call graph.
3. **Check** — signature/body equality; **`Needs(caller) ⊇ Needs(callee)`**; entry-point satisfaction; **`RequestDeps`** field compatibility.
4. **Transform** — emit interfaces, **`Deps`** / **`fnDeps`** structs; **`require`** → local from **`deps`**; **`provide`** → struct literals; merge at call sites.
5. **Metadata** — discovery JSON, hover docs listing **`Needs(F)`**.

Diagnostics anchor on **`require`** for missing capabilities and on **`provide`** / entry points for missing wiring.

---

## 8. Relation to existing Forst features

| Feature | Interaction |
| --- | --- |
| **`ensure` / `is`** | Orthogonal — data validation vs capability wiring. |
| **Shape guards** | Same structural typing for "implements **`requirement`**". |
| **`Result` / errors** | Requirement methods may return **`Result`**; wiring itself is not fallible. |
| **Sidecar** | Host builds **`Deps`**; JSON API carries **data** params only. |
| **Effect RFC** | Requirements are **capabilities**, not an **`Effect`** monad; optional TS **`R`** in `.d.ts` for interop only. |

---

## 9. Comparison snapshot

| Criterion | Agent A | Agent B (`using`) | Agent C (`?` / `@require`) |
| --- | --- | --- | --- |
| Keywords | **2** (`require`, `provide`) | 1 (`using`) | 2–3 + operators |
| Wiring visibility | **High** (`provide`) | Low (implicit) | Medium (`with req`) |
| Readability | **`ensure`-like** | English `using` | Type-indexed |
| Go lowering | Struct + interfaces | Same | Same |
| Test pattern | **`provide { … }`** | **`using … with`** | **`with req`** |
| Call-site noise | Low internally; **explicit at edge** | Lowest | Lowest |
| Effect alignment | **Strong** | Weak | Medium |

---

## 10. Minimal example (end-to-end)

**Forst**

```ft
package demo

requirement Logger {
    info(msg String)
}

func greet(name String) {
    require Logger
    logger.info("hello " + name)
}

func main() {
    provide Logger: StdLogger {} {
        greet("world")
    }
}
```

**Generated Go (conceptual)**

```go
type Logger interface { Info(msg string) }

type greetDeps struct { Logger Logger }

func greet(deps greetDeps, name string) {
    deps.Logger.Info("hello " + name)
}

func main() {
    deps := greetDeps{Logger: StdLogger{}}
    greet(deps, "world")
}
```

**Test**

```ft
test "greet logs" {
    provide Logger: CapturingLogger { lines: [] } {
        greet("Ada")
        ensure CapturingLogger.last() == "hello Ada"
    }
}
```

---

## 11. Open questions

1. **Merge identical `Needs` sets** — one package-level **`Deps`** vs per-function structs (ABI / diff noise)?
2. **Call-site `provide` vs block-only** — allow both in v1, or block-only for simplicity?
3. **`require Logger from deps` sugar** — always require explicit `from` or default when param named `deps`?
4. **Runtime nil policy** — panic vs **`MissingRequirement` error** for host misconfiguration?
5. **TS `forst generate`** — emit Effect-style **`R`** type param on exports, or JSDoc **`@requires`** only?
6. **Built-in requirements** — prelude **`Env`**, **`Random`**, **`Stdout`** as predeclared requirements?

---

## 12. Recommended next step

Add **`examples/in/rfc/providers/explicit_basic.ft`** with golden Go output and **`TestProviders_explicitBasic`** integration test — same pattern as `nominal_error.ft`. Validates parse → infer → **`provide`** lowering before committing lexer tokens.

---

## Verdict

Agent A delivers **requirements as first-class, inferrable capabilities** with **two familiar keywords**, **Effect-shaped explicit wiring**, and **idiomatic Go lowering**. The cost is **`provide` ceremony at boundaries** — intentional traceability for Forst’s Go-first, sidecar-aware, **`ensure`-fluent** audience. Parameters stay **data**; **`require` / `provide`** own **runtime logic**; tests and LLMs get a **single, readable mock pattern** without DI frameworks.
