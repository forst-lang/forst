# Function requirements (normative RFC)

**Audience:** Language designers, compiler implementers, and RFC authors. **Status:** **Normative target** for Forst **function requirements** (capabilities): explicit dependency declaration, compile-time wiring, and idiomatic Go lowering.

**Depends on:** [00 — Prior art](./00-prior-art.md), [errors normative](../errors/02-first-class-errors-normative.md), [guard / `ensure`](../guard/guard.md), [sidecar decisions](../sidecar/10-decisions.md), [effect overview](../effect/00-overview.md).

**Consolidates:** [Agent A — `require` + `provide`](./designs/agent-a-explicit-require.md) (primary surface), [Agent B — `using` context](./designs/agent-b-using-context.md) (inference + handler-context patterns, rejected keyword), [Agent C — operators](./designs/agent-c-operator-syntax.md) (type-indexed traceability, rejected operators).

---

## 1. Design rationale

### 1.1 Problem

Backend Forst code needs **testable dependencies** (loggers, clocks, repositories) without:

- reflection-heavy DI containers,
- hidden globals,
- conflating **service handles** with **request data**,
- or a mandatory Effect-style runtime monad on the Go emit path.

Prior art ([00 §4](./00-prior-art.md#4-cross-cutting-themes)) clusters into three families: environment monads (`R`), algebraic effect handlers, and explicit interfaces. Forst’s **Go transpilation** constraint and **TypeScript-familiar** audience favor **explicit interfaces** with a **thin, English-keyword sugar layer** that erases to struct fields — the Rust/Go baseline elevated with compile-time requirement tracking.

### 1.2 Chosen primitive

**One declaration form, two keywords:**

| Construct | Kind | Role |
| --- | --- | --- |
| **`requirement`** | Declaration | Named capability contract (method set). |
| **`require`** | Statement | Obtain a requirement binding inside a function body. |
| **`provide`** | Call suffix / block | Satisfy requirements at a call site or lexical scope. |

**Keyword budget:** `requirement` (decl) + **`require`** + **`provide`** = two new statement/call keywords. No third keyword for forwarding (`provide forward` is a modifier on `provide`, not a reserved word).

**Why not reuse `ensure`?** Today **`ensure`** means guard, narrowing, and **`Result`** failure propagation ([errors 02](../errors/02-first-class-errors-normative.md), [guard](../guard/guard.md)). Overloading it for “needs Logger” would collide with established semantics. Requirements are **compile-time wired**, not runtime validation.

**Why Agent A over Agent B (`using`)?** Agent B minimizes call-site syntax but **hides** the satisfaction boundary — harder for code review, public APIs, and LLM agents that generate call sites from signatures alone. Agent A’s **`provide { … }`** makes the contract visible where behavior is wired (Effect **`provide`** familiarity) while Agent B’s **handler-context reuse** is preserved as **`provide ctx { … }`**.

**Why not Agent C operators (`?T`, `@require`)?** Type-indexed slots are grep-friendly for tooling, but **`?`** risks collision with future optional syntax, **`@require`** is less readable than English keywords for Forst’s target audience, and **`needs ?UserRepo`** separates requirements from the **`require`/`provide`** vocabulary used at wiring sites. Traceability goals are met via **`requirement` names**, optional exported **`requires`** clauses, and discovery metadata (§8).

**Why not full Effect `R` monad?** Effect-TS’s third type parameter is valuable for TS clients ([00 §2.1](./00-prior-art.md#21-typescript-effect-effecteffecta-e-r)), but mandating a runtime effect interpreter on the Go path conflicts with Forst’s performance and interop goals. Requirements **track capabilities in Forst types** and **lower to plain Go**; optional TS metadata may expose requirement lists without shipping Effect.

### 1.3 Tradeoffs accepted

| Accepted cost | Mitigation |
| --- | --- |
| Call-site verbosity on deep trees | **`provide ctx { … }`** at handler scope; **`provide forward`** inside trusted modules |
| Per-function synthesized needs structs in Go | Optional future merge of identical sets within a package |
| No optional/partial requirements in v1 | Explicit nullable fields on context structs, or split functions |

### 1.4 Rejected alternatives (summary)

See §10 for the full comparison table.

---

## 2. Goals and non-goals

### Goals

| Goal | Normative intent |
| --- | --- |
| **Parameters = data** | Function parameters carry **inputs** only (`Order`, `String`, request bodies). |
| **Requirements = runtime logic** | Loggers, clocks, DB handles, HTTP clients are **requirements**, not ordinary params. |
| **Explicit + traceable** | Every **`require`** maps to a struct field read; every **`provide`** maps to a struct literal or context field extraction in Go. |
| **Inferrable** | A function’s requirement set is inferred from **`require`** statements and transitive callees; no duplicate annotation required. |
| **Testable** | Tests **`provide`** fake structs — no mock registry, no monkey-patching. |
| **Go-compatible lowering** | Interfaces + value needs struct as first parameter; no reflection, no service locator. |
| **Handler fit** | **`provide ctx { … }`** reuses existing context struct fields ([sidecar](../sidecar/10-decisions.md)). |
| **LLM-friendly** | Named **`requirement`** blocks list methods; tests copy the block into a fake struct. |

### Non-goals (v1)

| Non-goal | Rule |
| --- | --- |
| **Data via requirements** | If a value is **per-request input**, it is a **parameter** or a **field on a data struct** — not a requirement. |
| **DI frameworks** | No runtime container, no `map[reflect.Type]interface{}`, no global registry. |
| **Effect rows / handlers** | No Koka/Unison-style effect rows or handler blocks in user syntax. |
| **Optional requirements** | No “maybe has Logger”; use explicit `Option` fields or separate functions. |
| **Cross-cutting `using` implicit context** | Rejected Agent B keyword; context threading is **explicit at `provide` boundaries**. |
| **Wire-format capabilities** | Sidecar JSON APIs send **data** only; host constructs **`AppContext`** and **`provide`**s server-side. |
| **Nominal `implements`** | Structural satisfaction only (same rules as shape guards). |

### The Go rule (normative)

> **If a struct field solves it, it is not a requirement mechanism — it is data layout.**

Requirements **lower to struct fields** on a synthesized needs struct or on an existing handler context type. They do **not** introduce a parallel DI container. When `AppContext` already has `logger: StdLogger`, **`provide ctx { … }`** forwards `ctx.logger` — one context object, not two.

---

## 3. Core constructs

### 3.1 `requirement` — capability contract

A **requirement** is a package-scoped declaration naming a **method set** the runtime must supply. It is **not** a value type.

```forst
requirement Logger {
    info(msg String)
    error(msg String)
}

requirement Clock {
    now(): Int
}

requirement UserRepo {
    find(id String): Result(User, Error)
    save(user User): Result(User, Error)
}
```

**Rules:**

1. Methods use the same syntax as shape fields / function signatures.
2. Requirements may reference named types and **`Result`**.
3. Implementations are ordinary Forst types that **structurally satisfy** the method set (no `implements` keyword).
4. Distinct requirement names denote distinct capabilities even if method sets overlap (`AuditLogger` vs `AppLogger`).

**Lowering:** each `requirement R { … }` → Go `interface R { … }` with exported method names (`info` → `Info`).

### 3.2 `require` — obtain in body

Inside a function body, **`require`** binds a local name to an implementation supplied by the caller’s **`provide`**:

```forst
require <name>: <RequirementIdent>
require <RequirementIdent>          // shorthand: name = lowerCamel(RequirementIdent)
```

**Rules:**

1. Bindings are **lexically scoped** to the rest of the function (including nested blocks unless shadowed).
2. **`require` does not fetch globals.** Missing supply → **compile error**, never runtime panic.
3. Duplicate **`require`** for the same requirement in one scope → error: *"`Logger` already required"*.
4. Parameter namespace and requirement namespace are distinct: parameter `logger` + **`require logger: Logger`** is allowed.
5. **`require`** is invalid inside **`requirement`** method bodies (requirements declare contracts, not business logic).

**Lowering:** `require logger: Logger` → `logger := needs.Logger` where `needs` is the synthesized first parameter (§6).

### 3.3 `provide` — satisfy at boundary

**`provide`** satisfies requirements at a **call site** or **scope entry**.

#### 3.3.1 Postfix on calls (primary)

```forst
<callee>(<args>) provide { <RequirementIdent>: <expr>, ... }
<callee>(<args>) provide forward
```

- Map keys are **requirement identifiers** (`Logger`, not `logger`) for grep and LLM predictability.
- Duplicate keys in one map → error.
- **`provide forward`** (modifier, not a keyword): inherit matching requirements from the **enclosing `provide` scope** (function-wide; see §5.4). Public/library APIs should prefer explicit maps.

#### 3.3.2 Block forms

```forst
provide <expr> { <body> }           // satisfy inner calls from struct fields
provide { <RequirementIdent>: <expr>, ... } { <body> }
```

**`provide ctx { … }`** (from Agent B): when `ctx` is a struct whose fields structurally implement callee requirements, the compiler extracts fields by convention (`Logger` → `ctx.logger`, `UserRepo` → `ctx.userRepo`) unless overridden in an inner explicit map.

**Lowering:** block body → lexical scope emitting struct literals or field extractions for each inner call (§6.4).

### 3.4 Optional: exported `requires` clause (documentation)

For **exported** functions, an optional clause documents the inferred set (Agent C traceability + Agent A public API honesty):

```forst
func expireToken(token Token) requires Logger, Clock
    : Result(Token, Error) {
    require logger: Logger
    require clock: Clock
    ...
}
```

**Rules:**

1. If present, must **exactly match** the inferred set (compiler-checked).
2. Omitted on internal helpers — inference still applies.
3. Does not change Go lowering; feeds docs, LSP, and discovery JSON (§8).

---

## 4. Normative syntax (grammar sketch)

```ebnf
RequirementDecl   ::= "requirement" IDENT "{" MethodDecl* "}"
RequireStmt       ::= "require" ( IDENT ":" )? RequirementIdent
ProvideSuffix     ::= "provide" ( "forward" | "{" ProvideEntry ( "," ProvideEntry )* "}" )
ProvideBlock      ::= "provide" ( StructExpr | "{" ProvideEntry ( "," ProvideEntry )* "}" ) Block
ProvideEntry      ::= RequirementIdent ":" Expr
RequiresClause    ::= "requires" RequirementIdent ( "," RequirementIdent )*
FunctionDecl      ::= "func" IDENT Params RequiresClause? ReturnType? Block
CallExpr          ::= Callee Args ProvideSuffix?
```

**Precedence:** **`provide`** postfix binds tighter than statement separators — `f(x) provide { … }` is one call expression.

---

## 5. Inference rules

### 5.1 Intra-function collection

For function `F`:

1. Collect all **`require`** statements in `F`’s body (all branches — conditional **`require`** still counts; wiring must be total at entrypoints).
2. Distinct requirement identifiers → **`Req(F)`**.
3. Unused **`require`** → warning (same severity as unused variable).

### 5.2 Transitive requirements (call graph)

When **`F`** calls **`G`** and **`G`** needs **`Req(G)`**:

- The call must include **`provide { … }`**, **`provide forward`**, or sit inside a **`provide`** block / scope that supplies **`Req(G)`**.
- **`Needs(F) ⊇ Req(G)`** after fixed-point propagation over the package call graph.
- Missing supply → error at the **call site**: *"`expireToken` needs `Clock`; add `Clock` to provide map"*.

No requirement annotation on **`G`**’s signature is **required** — the set is inferred from **`require`** + callees.

### 5.3 Structural satisfaction for `provide ctx`

**`provide ctx { … }`** is valid when, for every requirement **`R`** needed by a callee in the block:

- **`ctx`** has field **`f`** such that **`typeof(f)`** structurally implements **`R`**, and
- default field resolution: **`Logger` → `logger`**, **`UserRepo` → `userRepo`** (lower camel case of requirement ident), overridable via explicit **`provide { Logger: ctx.auditLog }`**.

Same structural rules as shape guards.

### 5.4 `provide forward` scope

**`provide forward`** inherits from the **innermost enclosing** active **`provide`** scope (block or function entry **`provide ctx { … }`**). Does not cross function boundaries without an outer **`provide`**. Forwarding is **opt-in** per call — default is explicit map.

### 5.5 Entry points

Functions designated as entry points (**`main`**, sidecar exports, **`test`** blocks) must have all requirements **bound** before any **`require`** is evaluated:

- via **`provide { … } { … }`** at the top, or
- via **`provide ctx { … }`** where **`ctx`** satisfies the transitive set.

Unbound requirements → **compile error**.

### 5.6 Cross-package calls

When package **`api`** calls package **`domain`**’s **`getUser`** needing **`Logger`**:

- **`api`**’s callers must **`provide Logger`** (or forward from a scope that includes it).
- Each package emits its own requirement interfaces; wiring is the **host’s** responsibility at the process edge (same as Go).

### 5.7 Interaction with `ensure` / `Result`

- Missing requirements: **compile-time** error.
- Requirement **methods** may return **`Result`**; **wiring** itself is not fallible.
- Orthogonal to **`ensure`** — requirements supply services; **`ensure`** validates **data**.

---

## 6. Go lowering strategy

### 6.1 Requirement → interface

```forst
requirement Logger {
    info(msg String)
}
```

```go
type Logger interface {
    Info(msg string)
}
```

### 6.2 Function → needs struct + first parameter

For `expireToken` with **`Req = { Logger, Clock }`**:

```forst
func expireToken(token Token): Result(Token, Error) {
    require logger: Logger
    require clock: Clock
    if token.expiresAt < clock.now() {
        logger.info("token expired")
        return Err(Expired { tokenId: token.id })
    }
    return Ok(token)
}
```

```go
type expireTokenNeeds struct {
    Logger Logger
    Clock  Clock
}

func expireToken(needs expireTokenNeeds, token Token) (Result[Token, error], error) {
    logger := needs.Logger
    clock := needs.Clock
    if token.ExpiresAt < clock.Now() {
        logger.Info("token expired")
        return Err(Expired{TokenId: token.Id}), nil
    }
    return Ok(token), nil
}
```

Functions with **empty** **`Req(F)`** emit **no** synthetic struct — zero overhead for pure helpers.

### 6.3 Call-site `provide` → struct literal

```forst
expireToken(token) provide {
    Logger: ctx.logger,
    Clock:  ctx.clock,
}
```

```go
expireToken(expireTokenNeeds{
    Logger: ctx.Logger,
    Clock:  ctx.Clock,
}, token)
```

### 6.4 `provide ctx { … }` block

```forst
provide ctx {
    saveOrder(req)
    notify(req.id)
}
```

```go
{
    saveOrder(saveOrderNeeds{Logger: ctx.Logger, Database: ctx.Database}, req)
    notify(notifyNeeds{Logger: ctx.Logger}, req.Id)
}
```

Compiler may reuse temporaries when the same field subset repeats (optimization detail).

### 6.5 Handler boundary (sidecar)

Exported handlers keep **public** data parameters only:

```forst
func CreateUser(input CreateUserRequest, ctx AppContext): Result(CreateUserResponse, Error) {
    provide ctx {
        return createUserInternal(input)
    }
}
```

External JSON-RPC / HTTP clients send **data**; the sidecar host builds **`AppContext`** once per request. Internal helpers may take synthesized needs structs invisible on the wire.

### 6.6 Performance guarantees

- **No reflection**, no service locator, no global registry.
- Hot path cost = struct literal of interface words (same as explicit Go injection).
- **`provide forward`** → field copies from outer needs struct; no allocation when reusing stack temps.
- Inlining-friendly: needs struct is a plain value parameter.

### 6.7 End-to-end minimal example

**Forst**

```forst
package demo

requirement Logger {
    info(msg String)
}

func greet(name String) {
    require logger: Logger
    logger.info("hello " + name)
}

func main() {
    provide { Logger: StdLogger {} } {
        greet("world")
    }
}
```

**Generated Go**

```go
type Logger interface { Info(msg string) }

type greetNeeds struct { Logger Logger }

func greet(needs greetNeeds, name string) {
    needs.Logger.Info("hello " + name)
}

func main() {
    scope := greetNeeds{Logger: StdLogger{}}
    greet(scope, "world")
}
```

---

## 7. Tests and LLM/agent ergonomics

### 7.1 Test pattern (normative)

Tests **`provide`** fake implementations as **plain struct literals** — isomorphic to Effect’s **`Layer.succeed`**, but visible in source:

```forst
test "expireToken rejects expired token" {
    token := Token { id: "t1", expiresAt: 1000 }

    result := expireToken(token) provide {
        Logger: NopLogger {},
        Clock:  FakeClock { fixedMs: 2000 },
    }

    ensure result is Err(Expired {})
}
```

### 7.2 Fake implementation pattern

An agent (or human) reads the **`requirement`** block and emits a struct + methods:

```forst
type FakeClock = { fixedMs: Int }
func (c FakeClock) now(): Int { return c.fixedMs }

type NopLogger = {}
func (l NopLogger) info(msg String) {}
func (l NopLogger) error(msg String) {}
```

No framework registration API to learn.

### 7.3 Agent checklist

| Step | Source |
| --- | --- |
| 1. List methods to implement | `requirement R { … }` |
| 2. Build fake struct | Copy method names/signatures |
| 3. Wire test | `provide { R: FakeR { … } }` |
| 4. Assert behavior | Existing **`ensure` / `is`** on **`Result`** |

### 7.4 Discovery / LSP (recommended)

Extend discovery JSON per exported function:

```json
{
  "name": "expireToken",
  "requirements": ["Logger", "Clock"]
}
```

Derived from inference or **`requires`** clause. Enables “generate provide map” code actions when callees gain new **`require`** lines.

---

## 8. Compiler phases (implementation checklist)

1. **Parse** — `RequirementDecl`, `RequireStmt`, `ProvideSuffix`, `ProvideBlock`, optional `RequiresClause`.
2. **Collect** — per-function **`Req(F)`** from **`require`**; fixed-point over call graph for **`Needs(F)`**.
3. **Check** — satisfiable **`provide`** at every call; structural field matching for **`provide ctx`**; entry-point completeness; **`requires`** clause consistency.
4. **Transform** — emit interfaces, **`fnNeeds`** structs, struct literals at provide sites; **`require`** → field access.
5. **Metadata** — discovery **`requirements`** array; optional **`forst doctor`** wiring checklist for entrypoints.

Diagnostics anchor on **`provide`** for missing keys and on **`require`** for unused bindings.

---

## 9. Relation to existing Forst features

| Feature | Interaction |
| --- | --- |
| **`ensure` / `is`** | Orthogonal — requirements wired at compile time; **`ensure`** validates **data**. |
| **Shape guards** | Same structural typing for “implements requirement”. |
| **`Result` / errors** | Requirement methods may return **`Result`**; wiring is not fallible. |
| **Sidecar handlers** | Host builds **`AppContext`**; handler uses **`provide ctx { … }`**. |
| **Effect RFC** | Requirements replace ad-hoc service locators on Go; TS Effect layers remain **optional** at the boundary. |

---

## 10. Comparison vs rejected alternatives

| Criterion | **Normative** (`require` + `provide`) | Agent B (`using`) | Agent C (`?T` / `@require`) | Full Effect `R` monad |
| --- | --- | --- | --- | --- |
| **Call-site clarity** | Requirements visible in **`provide { … }`** | Hidden — plain call | Hidden — bundle at edge only | Runtime **`pipe(provide)`** |
| **English readability** | **`require` / `provide`** like **`ensure`** | **`using`** + implicit rules | **`?` / `@`** operators | **`yield*`**, Layers |
| **Go lowering** | Struct + interfaces | Same | **`ReqSet` pointer** | New runtime library |
| **Handler fit** | **`provide ctx { … }`** | Natural **`ctx`** fields | Host **`ReqSet`** | App-root Layer |
| **Inference** | From **`require`** + callees | From **`using`** + callees | From **`@require`** + callees | Type-level **`R`** |
| **Public API honesty** | Explicit **`provide`** + optional **`requires`** | Easy to over-hide deps | **`needs ?T`** clause | **`R`** in TS types |
| **LLM test generation** | Literal **`provide { … }` maps** | **`using … with`** | **`ReqSet{ … }` + `with`** | Layer boilerplate |
| **Keyword / sigil budget** | 2 keywords + 1 decl | 1 keyword + 1 decl | 2–3 keywords + operators | External library |
| **Refactor signal** | Callers’ **`provide`** maps break at compile time | Propagation error on **`using`** | Slot metadata change | **`R`** channel change |
| **Forst fit** | **Chosen** | Rejected: implicit boundary | Rejected: operator readability | Rejected: Go runtime cost |

---

## 11. Edge cases

| Case | Behavior |
| --- | --- |
| Circular call graphs | SCC analysis; error if cycle cannot be satisfied with consistent **`provide`** scope |
| Two loggers in one function | Distinct requirement names (`AuditLogger`, `AppLogger`) |
| Inner **`provide { … } { … }`** | Shadows outer scope for that block only — lexical, not dynamic |
| Go callers of exported Forst fns | Pass needs struct as first Go parameter explicitly |
| Concurrency | Needs structs hold interfaces; thread-safety is implementation’s contract (same as Go) |
| Unused requirement in **`requires`** clause | Error — must match inference exactly |

---

## 12. Open questions

1. **Merged needs structs** — deduplicate identical **`{ Logger, Clock }`** types within a package for smaller Go API surface?
2. **Built-in requirements** — prelude **`Env`**, **`Random`** as predeclared requirements?
3. **`provide forward` in public APIs** — lint discourage on exported surfaces?
4. **TS emit** — expose requirement lists as JSDoc / branded types for Effect users, or Go-style interfaces only?

---

## Document status

**Normative target.** Implementation tracks compiler checklist §8. Competing agent designs remain in [`designs/`](./designs/) for history.
