# Providers — primitive design

**Audience:** Language designers, compiler implementers, application authors.

**Status:** Normative spec — complete feature (`use`, `with`, transitive inference, mandatory completeness).

**Decisions:** [ADR — Design decisions](./ADR.md) (`with` takes Provider shape literals only, scope forwarding, derived `Providers(f)`, etc.).

---

## Vision

Forst backend code needs **swappable runtime services** (loggers, clocks, databases, HTTP clients) without reflection DI, globals, or mock frameworks. Tests must substitute **plain structs and imported interfaces** that structurally satisfy method sets — the same pattern Go and Rust already use.

The language should treat **testability as a primitive**, not a framework:

- Business functions declare what they **use** in the body.
- Entry points (handlers, `main`, test helpers) **wire** implementations with **`with <wiring> { … }`** — a **shape literal**, variable, or call result that **satisfies a Provider** ([ADR-042](./ADR.md#adr-042-provider-and-providers-vocabulary)).
- **ProviderScope forwarding is default** — inside a `with` scope, callees inherit requirements automatically; **nested** `with innerMap { … }` overlays keys (no postfix `with` on calls).
- **`Providers(f)` is derived sugar** for tooling (LSP, docs, discovery JSON) — not an author-written export clause.
- Contracts are **ordinary types** — Forst shapes with methods, nominal Forst types, or **imported Go types** — not a parallel declaration family.

**Parameters carry per-invocation data** (`Order`, `userId`, request bodies). **Providers carry host-provided services** (logger, clock, DB pool). Sidecar wire formats stay data-only; the host builds Providers server-side.

This design deliberately avoids test DSL keywords. Integration tests are **plain functions** composed with `with`, shared setup helpers, and standard Go `_test.go` entry points where needed.

---

## Core primitives

Two keywords. Everything else is types and inference.

| Primitive | Role |
| --- | --- |
| **`use`** | Bind a Provider (contract type) inside a function body. |
| **`with`** | Wire **Providers**: **`with <wiring> { … }`** only. Wiring must satisfy a **Providers** shape ([ADR-042](./ADR.md#adr-042-provider-and-providers-vocabulary)); field names = root contract idents; values assignable; **nil forbidden**. |

**Why only two:**

- **`use`** is the author-facing read: "this function depends on X." The compiler aggregates `use` sites into **`Providers(f)`** per function.
- **`with`** takes a **Providers wiring expression** and a body. No struct forwarding, no postfix form on calls ([ADR-030](./ADR.md#adr-030-with-takes-provider-shape-literals-only)).
- Providers from an outer `with` **always forward** to inner calls. **Nested** `with smaller { … }` merges wiring (inner keys shadow). Missing supply is a **hard compile error**.
- No third keyword for contracts, exports, harnesses, or overrides. **Types are the contract.** Wiring completeness is a **checker property**, not a declaration.

---

## Providers identification

Providers are **inferred**, not declared on function signatures.

### Vocabulary

| Term | Meaning |
| --- | --- |
| **Provider** | One contract slot — root ident + contract type (e.g. `Logger: Logger`). |
| **`Providers(f)`** | Inferred set for function `f` — dynamically `Providers(A \| B \| C)` from `use` and transitive calls. |
| **Providers bundle** | Author-named shape typedef (e.g. `CIProviders`) or inline wiring literal passed to `with`. |
| **Providers struct** | Deduped Go emit type (e.g. `Providers_a1b2c3`) — first parameter at the Go boundary. |

### Sources of `Providers(f)`

| Source | Rule |
| --- | --- |
| **`use x: T`** | Provider `T` (root ident of `T`) ∈ **`Providers(f)`**. |
| **`use T`** | Shorthand: bind name `lowerCamel(T)` to type `T`. |
| **Transitive calls** | If `f` calls `g`, then **`Providers(f) ⊇ Providers(g)`** minus Providers satisfied locally inside `f` via inner `with`. |
| **Context parameters** | **Never** — method calls on context struct fields do **not** contribute ([ADR-005](./ADR.md#adr-005-context-parameters-never-contribute-to-providersf), [ADR-011](./ADR.md#adr-011-parameters-are-data-use-is-runtime-logic)). |

There is **no** `uses Logger, Clock` export clause in source. **`Providers(f)`** is **derived sugar** for tooling — **`use` sites and transitive calls** are the single source of truth. LSP hover, generated docs, and discovery JSON **project** that set; authors never write it twice.

### How the compiler knows what's needed

1. **Type-check the body.** Each `use` binds a local whose type is the contract.
2. **Resolve contract types.** A contract may be:
   - a Forst shape or nominal type with methods;
   - an imported Go interface (`io.Writer`, a repo interface from your module);
   - an imported concrete or pointer type (`*sql.DB`, `*http.Client`).
3. **Walk calls.** Propagate **`Providers(g)`** from callees upward unless an inner `with` satisfies Providers locally.
4. **Propagate upward.** Callees' Providers become callers' obligations unless an inner `with` satisfies them locally.
5. **Check entry points.** Wiring roots (`main`, exported HTTP handlers, or any function whose Providers are not satisfied by an enclosing `with`) must have all **`Providers(f)`** provided by an outer `with` or explicit wiring at the call site.

### Nominal vs structural distinction

- **Distinct type names = distinct Providers** unless linked by **type alias** (`type AuditLogger = Logger` → same slot as **`Logger`**) ([ADR-042](./ADR.md#adr-042-provider-and-providers-vocabulary)).
- **Structural satisfaction** for implementations: a Forst struct with matching methods, or a Go type assignable to an imported interface, satisfies the contract without inheritance or registration.
- **`with` wiring keys:** **root contract ident only** — `with { Logger: … }` yes; `with { AuditLogger: … }` is a **compile error** when `type AuditLogger = Logger` ([ADR-041](./ADR.md#adr-041-root-contract-ident-only-in-with-keys)).

---

## Types are the contract

No `capability` keyword. No `requirement` block. Authors write **types** the compiler already understands.

### Forst shapes and nominal types

```forst
type Logger = {
    info(msg String)
    error(msg String)
}

type StdLogger = { level: String }

func (l StdLogger) info(msg String) {
    fmt.Println(l.level + ": " + msg)
}

func (l StdLogger) error(msg String) {
    fmt.Println("ERR " + l.level + ": " + msg)
}
```

`Logger` is the contract. `StdLogger` is a production implementation. `NopLogger` in tests is another struct with the same methods.

### Imported Go interfaces

```forst
import "io"

func writeReport(title String) {
    use w: io.Writer
    w.Write([]Byte(title))
}
```

The contract **is** `io.Writer`. Tests pass `bytes.Buffer` or a small fake struct with `Write([]Byte) (Int, Error)`.

### Imported concrete and pointer types

```forst
import (
    "database/sql"
    "net/http"
)

func queryUser(id String): Result(User, Error) {
    use db: *sql.DB
    // db.QueryRow, db.Query — methods resolved via Go interop
}
```

When the contract is `*sql.DB`, tests supply `sql.Open` on SQLite in-memory or a thin wrapper struct that forwards `Query`/`Exec` — no Forst-specific mock type.

### Provider wiring (the only `with` form)

There is **no** `with ctx` struct sugar and **no** postfix `f() with { … }` (avoids conflict with `ensure` and keeps one wiring shape).

```forst
// Shape literal at wiring root
with {
    Logger:   &StdLogger { level: "info" },
    UserRepo: &PostgresRepo { dsn: env("DATABASE_URL") },
} {
    handleGetUser("42")
}

// Wiring variable (typical for tests)
wiring := ciUserApiServices()
with wiring {
    handleCreateUser(req)
}

// Nested overlay — inner keys shadow; outer keys forward
with ciUserApiServices() {
    with { EmailSender: &RecordingEmail {} } {
        handleCreateUser(req)
    }
}
```

**Wiring rules ([ADR-030](./ADR.md#adr-030-with-takes-provider-shape-literals-only), [ADR-019](./ADR.md#adr-019-superset-wiring-allowed)):**

- Keys are **root contract idents** (`Logger`, `UserRepo`, …). **Unknown keys → hard error.** Known-but-unused keys → **warning**.
- **Values** must be **assignable** to the contract type (pointers allowed, not required). **Nil forbidden.**
- ProviderScope wiring may be a **superset** of any callee’s **`Providers(g)`**; lowering copies only required fields.
- Handlers are **not special** — same `use` / `with` as any function. Services may still be passed as **ordinary parameters** outside `use`/`with`; that path is unconstrained by this feature.

Fat CI fixtures are **Providers bundles** — returned from a function or written inline; no conversion helpers.

```forst
func ciUserApiServices(): CIProviders {
    return {
        Logger:      &NopLogger {},
        UserRepo:    &InMemoryUserRepo { users: map[String]User{} },
        HttpClient:  &BlockedHttp {},
        EmailSender: &NoopEmail {},
    }
}
```

```forst
func handleGetUser(id String): Result(User, Error) {
    return getUser(id)   // caller must enclose in `with wiring { … }`
}
```

---

## Wrapping foreign Go

Forst already imports Go packages and type-checks qualified calls via `go/types`. Requirements extend that model: **wrap foreign symbols in Forst terms**, then `use` the wrapper type.

### Pattern: wrap a stdlib client method

Foreign code:

```go
// net/http — (*Client).Do(*Request) (*Response, error)
```

Forst wrapper:

```forst
import "net/http"

// Contract: anything that can perform HTTP round-trips the way we need.
type HttpDoer = {
    do(req *http.Request): Result(*http.Response, Error)
}

// Adapter over *http.Client — production default.
type ClientDoer = { client: *http.Client }

func (c ClientDoer) do(req *http.Request): Result(*http.Response, Error) {
    return c.client.Do(req)
}

func fetchHealth(url String) {
    use http: HttpDoer
    req, err := http.NewRequest("GET", url, nil)?
    resp := http.do(req)?
    defer resp.Body.Close()
    // ...
}
```

**How `use` resolves:** `HttpDoer` is a Forst shape. `ClientDoer` satisfies it structurally. At call sites, `with { HttpDoer: ClientDoer { client: &http.Client{} } }` supplies the adapter.

### Pattern: wrap a Go repository type

```forst
import "example.com/internal/store"

// Re-export the Go interface as the contract (preferred when the Go module already defines it).
type UserStore = store.UserRepository

func saveUser(user User) {
    use repo: UserStore
    repo.Save(user)?
}
```

If the Go package exports only a concrete `*store.Postgres`:

```forst
type UserStore = {
    save(u User): Result(Void, Error)
    find(id String): Result(User, Error)
}

type PostgresStore = { inner: *store.Postgres }

func (p PostgresStore) save(u User): Result(Void, Error) {
    return p.inner.Save(toGoUser(u))
}

func (p PostgresStore) find(id String): Result(User, Error) {
    return fromGoUser(p.inner.Find(id))
}
```

Business code `use`s `UserStore`, not `*store.Postgres`. Tests use `InMemoryStore` with the same methods.

### Go lowering for wrappers

| Forst | Emitted Go |
| --- | --- |
| `type HttpDoer = { do(...) }` | `type HttpDoer interface { Do(...) (...) }` (exported method names per Go rules) |
| `type ClientDoer = { client: *http.Client }` | `type ClientDoer struct { Client *http.Client }` |
| `use http: HttpDoer` in `fetchHealth` | `http := providers.HttpDoer` inside synthesized Providers struct |
| `with wiring { fetchHealth() }` | `fetchHealth(Providers_…{ HttpDoer: wiring.HttpDoer, … })` — merge scope Providers → struct literal at compile time |

For **imported Go interfaces**, lowering may **alias** the interface in generated Go (`type UserStore = store.UserRepository`) or emit a local interface with identical method set when name mangling requires it.

**Interface shim:** When a Forst shape wraps methods whose signatures must match an existing Go interface exactly, the transformer emits adapter methods on a struct that forwards to the wrapped value — same pattern as hand-written Go adapter types.

---

## `with` semantics

**One form only:** `with <wiring> { <body> }` ([ADR-030](./ADR.md#adr-030-with-takes-provider-shape-literals-only)).

- `<wiring>` — **shape literal** `{ Logger: NopLogger {}, … }`, variable (`wiring`, `ciUserApiServices()`), or any expression assignable to a **Providers** shape.
- **No** `with ctx` on named struct params. **No** postfix `f(x) with { … }` on calls (nested `with` instead; avoids `ensure` confusion).

### Scope entry and scope forwarding

A `with` block establishes an **scope Provider** for all calls inside. **Nested** `with inner { … }` **merges** outer bindings; inner fields **shadow** outer fields for the inner scope only ([ADR-007](./ADR.md#adr-007-always-forward-scope-no-with-forward), [ADR-008](./ADR.md#adr-008-nested-with-overlays-outer-wiring)).

**Default: always forward.** Every call inside a wired scope receives callee requirements from the merged scope map. There is **no** opt-out — unsatisfied need is a **hard compile error**.

Inside a function that already receives a synthesized **Providers** struct parameter at the Go boundary, the function body treats that struct as scope for its callees (same merge rules).

### Merge and shadow rules

Nested `with` is **associative up to shadowing**: merged scope at inner scope equals outer keys minus shadowed keys, union inner keys. Shadowing the same key twice with assignable types is idempotent.

| Situation | Behavior |
| --- | --- |
| Call inside `with wiring { … }` | **`Providers(g)`** for callee `g` **fully supplied from scope** (auto-forward). |
| Inner `with { Logger: &fake } { … }` inside outer `with wiring` | Inner wiring overlays outer; `Logger` shadows for inner scope; other keys forward from outer. |
| Conflicting types for same key | Compile error. |
| Wiring at root (no outer scope) | Must be **complete** for all **`Providers(f)`** reachable in the body. |
| Attempt to omit a required Provider | **No syntax** — unsatisfied Provider is always a compile error. |
| Unknown wiring key | **Hard error** ([ADR-025](./ADR.md#adr-025-unknown-wiring-keys-hard-error)). |
| Known-but-unused wiring key | **Warning** with callee context. |

### Entry-point wiring

```forst
func main() {
    with {
        Logger:   &StdLogger { level: "info" },
        UserRepo: &PostgresRepo { dsn: env("DATABASE_URL") },
    } {
        runServer()
    }
}
```

Production and tests use the **same** `with` mechanism. The difference is which concrete values appear in the wiring.

---

## Inference

### Rules (normative sketch)

| ID | Rule |
| --- | --- |
| **N1** | `use x: T` ⇒ root ident of `T` ∈ **`Providers(f)`**. |
| **N2** | Method call on `use`-bound name ⇒ methods must exist on `T` (standard type-check). |
| **N3** | `f` calls `g` ⇒ **`Providers(f) ⊇ Providers(g)`** minus Providers satisfied locally inside `f` via inner `with`. |
| **N4** | Wiring root: all **`Providers(f)`** must be satisfied by enclosing `with`, synthesized **Providers** struct, or inner `with` in the same body. **Mandatory completeness** ([ADR-015](./ADR.md#adr-015-unsatisfied-providers-are-hard-compile-errors)). |
| **N5** | Nested `with inner { … }` **merges** with outer scope (inner keys shadow). Merged scope = `(outer \ shadowedKeys) ∪ inner`. |
| **N6** | Wiring may be a **superset** of **`Providers(callee)`**; lowering copies only required keys. Known-but-unused keys → LSP **warning** ([ADR-024](./ADR.md#adr-024-known-but-unused-wiring-keys-warning)). |
| **N7** | Wiring values **assignable** to contract; **nil forbidden** ([ADR-037](./ADR.md#adr-037-nil-forbidden-in-wiring)). No optional-`use` syntax ([ADR-039](./ADR.md#adr-039-no-optional-use-syntax)) — optional behavior uses **parameters** or noop impls. |
| **N8** | Distinct nominal types are distinct Providers; **`type X = Y` shares slot** with `Y` ([ADR-042](./ADR.md#adr-042-provider-and-providers-vocabulary)). |
| **N9** | **Unknown wiring keys → hard error** — field name is not a known contract root ident ([ADR-025](./ADR.md#adr-025-unknown-wiring-keys-hard-error)). |
| **N10** | **Satisfaction:** scope **`U`** satisfies call to **`f`** iff **`Providers(f) ⊆ keys(U)`** with assignable field types ([ADR-043](./ADR.md#adr-043-satisfaction-relation-for-wiring)). |

### Transitive propagation

```
main / test helper
    └── with ciUserApiServices() { handleCreateUser(...) }
            └── createUserInternal(...)
                    └── use logger: Logger
                    └── use repo: UserRepo
                    └── use email: EmailSender
```

If `createUserInternal` gains `use metrics: Metrics`, the checker errors at **`handleCreateUser` or `main`** until the scope wiring supplies `Metrics`. No manual audit of intermediate helpers.

### Completeness errors

Example diagnostic:

```
createUserInternal requires EmailSender; not supplied at handleCreateUser (line 42)
  required by: createUserInternal → sendWelcome
  supply EmailSender in with-block: with { EmailSender: &… } { … } or add to ciUserApiServices()
```

---

## Cross-package inference

Real backends split across packages. **`Providers(f)`** must propagate through imports: when `repos` gains `use metrics: Metrics`, `services` and `handlers` inherit that obligation until a wiring root satisfies it ([ADR-016](./ADR.md#adr-016-transitive-providers-inference)).

### Fixed-point algorithm (normative sketch)

1. **Build module graph** from imports (same graph as typechecker package loading).
2. **Per function:** compute **`Providers(f)`** from N1–N10 within the package.
3. **Cross-package calls:** when `f` in package `P` calls exported `g` in package `Q`, **`Providers(f) ⊇ Providers(g)`** (minus local inner-`with` satisfaction in `f`).
4. **Iterate to fixed point** over the module graph until no **`Providers(f)`** grows.
5. **Cycle policy:** mutual recursion does not remove Providers — if `f ↔ g` and either requires `Logger`, both inherit the obligation unless locally satisfied.
6. **Error attachment:** completeness errors attach at the **nearest wiring root** in the importers’ chain (typically `main`, test helper, or sidecar entry), with obligation chain in the diagnostic.

**GA requirement:** multi-package integration tests must pass before feature GA ([ADR-035](./ADR.md#adr-035-cross-package-graph-normative-before-ga)).

---

## Discovery JSON

Go-side artifact for host authors, agents, and cross-package callers. **Not** emitted to TypeScript ([ADR-020](./ADR.md#adr-020-typescript-emit-excludes-providers-concept)).

### Schema (v1)

```json
{
  "version": 1,
  "packages": {
    "example.com/users": {
      "functions": {
        "handleCreateUser": {
          "providers": ["Logger", "UserRepo", "EmailSender"],
          "runnable": false
        },
        "healthCheck": {
          "providers": [],
          "runnable": true
        }
      }
    }
  }
}
```

| Field | Meaning |
| --- | --- |
| `providers` | Ordered list of root contract idents in **`Providers(f)`** |
| `runnable` | `true` iff **`Providers(f) = ∅`** — eligible for TS/sidecar export |

**Invalidation:** any edit to a `use` site or callee **`Providers(g)`** invalidates discovery JSON and LSP caches for the function and all transitive importers ([ADR-036](./ADR.md#adr-036-single-graph-source-for-checker-json-and-lsp)).

---

## LSP and tooling

Always-forward scope hides Providers at inner call sites ([ADR-007](./ADR.md#adr-007-always-forward-scope-no-with-forward)). **LSP ships with feature GA** — not fast-follow ([ADR-034](./ADR.md#adr-034-lsp-ships-at-feature-ga)).

| Feature | Requirement |
| --- | --- |
| **Derived `Providers(f)` hover** | On function hover — distinct from fixture typedef (`CIProviders`) constraints |
| **Obligation chains** | Completeness errors show `required by: f → g → h` |
| **Effective scope** | At cursor inside nested `with`, show merged Providers keys in scope (shadow markers) |
| **Unknown wiring key** | **Error** diagnostic + quick-fix when possible |
| **Known-but-unused key** | **Warning** naming callee context |
| **Sidecar export gate** | **Error** when exporting function with **`Providers(f) ≠ ∅`** |

Single source: typechecker fixed-point, discovery JSON, and LSP derive from the same Providers propagation state.

---

## Go lowering

Every feature must answer: **what Go would a careful author write?** The transformer synthesizes that.

### Providers struct (deduped)

For **`Providers(f) = { Logger, Clock }`** — fields use **contract types** by value ([ADR-026](./ADR.md#adr-026-providers-struct-fields-by-value)); **do not** emit `*interface` fields. Identical inferred sets share one emitted type ([ADR-013](./ADR.md#adr-013-deduped-providers-struct-naming), e.g. `Providers_a1b2c3`):

```go
type Providers_a1b2c3 struct {
    Logger Logger
    Clock  Clock
}
```

The struct is passed **by value** as the first argument (`providers Providers_a1b2c3`). Empty **`Providers(f)`** ⇒ no synthetic struct; function keeps data parameters only.

**Superset wiring:** When lowering from scope wiring, copy **only** fields in **`Providers(callee)`**; extra Providers bundle fields are skipped ([ADR-019](./ADR.md#adr-019-superset-wiring-allowed)). Known-but-unused fields in a wiring shape literal emit an LSP **warning** ([ADR-024](./ADR.md#adr-024-known-but-unused-wiring-keys-warning)).

### Function signature

```forst
func expireToken(token Token): Result(Token, Error) {
    use logger: Logger
    use clock: Clock
    if token.expiresAt < clock.now() {
        logger.info("token expired: " + token.id)
        return Err(Expired { tokenId: token.id })
    }
    return Ok(token)
}
```

Lowers to:

```go
type Logger interface {
    Info(msg string)
    Error(msg string)
}

type Clock interface {
    Now() int
}

type Providers_a1b2c3 struct {
    Logger Logger
    Clock  Clock
}

func expireToken(providers Providers_a1b2c3, token Token) (Token, error) {
    logger := providers.Logger
    clock := providers.Clock
    if token.ExpiresAt < clock.Now() {
        logger.Info("token expired: " + token.ID)
        return Token{}, &Expired{TokenID: token.ID}
    }
    return token, nil
}
```

(`Result(Token, Error)` in Forst lowers to `(Token, error)` in Go. `use` binds locals from Providers struct fields ([ADR-027](./ADR.md#adr-027-first-parameter-named-providers-use-binds-from-fields)). **Nil forbidden** at wiring sites ([ADR-037](./ADR.md#adr-037-nil-forbidden-in-wiring)).)

### Shared mock reuse

Fakes are reused **by convention** ([ADR-029](./ADR.md#adr-029-mock-reuse-by-convention-not-compiler-enforced)) — not compiler-enforced:

```go
var testClock = &FakeClock{FixedMs: 2000}

func TestExpireToken(t *testing.T) {
    expireToken(Providers_a1b2c3{
        Logger: &NopLogger{},
        Clock:  testClock, // same pointer across tests
    }, token)
}
```

In Forst, `ciUserApiServices()` + `with wiring { … }` shares one scope Providers bundle; lowered Go copies **field values** into each call’s Providers struct literal. Pointer wiring values (e.g. `&NopLogger{}`) assign to interface-typed fields as in ordinary Go.

### `with` block lowering

```forst
with wiring {
    expireToken(token)
}
```

Assume `wiring` is a variable or merged scope with fields `Logger` and `Clock` (Providers-shaped). Lowers to:

```go
expireToken(Providers_a1b2c3{
    Logger: wiring.Logger,
    Clock:  wiring.Clock,
}, token)
```

(Fields not in **`Providers(expireToken)`** are omitted. The transformer expands the merged scope Providers to a struct literal at compile time — **direct field copies**, not map lookup or reflection.)

### Shape literal lowering

```forst
with {
    Logger: &NopLogger{},
    Clock:  &FakeClock { fixedMs: 2000 },
} {
    expireToken(token)
}
```

```go
expireToken(Providers_a1b2c3{
    Logger: &NopLogger{},
    Clock:  &FakeClock{FixedMs: 2000},
}, token)
```

### Imported interface as contract

When `T` is `io.Writer`, emitted Go uses `io.Writer` directly in the Providers struct — no duplicate interface definition.

---

## Testing (Go-native)

There are **no** `test`, `harness`, or `override` keywords. Tests are **Go tests written in Forst** — discovered and run like `go test` ([ADR-044](./ADR.md#adr-044-test-entrypoints-use-test-and-testingt)). The **`forst test`** command is specified separately in the [forst-test RFC](../forst-test/README.md).

### Test entrypoints

Import `testing` like any Go package. Test functions use Go’s naming and signature rules:

```forst
import "testing"

func TestExpireTokenRejectsExpired(t *testing.T) {
    token := Token { id: "t1", expiresAt: 1000 }
    with {
        Logger: &NopLogger {},
        Clock:  &FakeClock { fixedMs: 2000 },
    } {
        result := expireToken(token)
        ensure result is Err(Expired {})
    }
}
```

| Rule | Detail |
| --- | --- |
| **Name** | `Test[A-Z]…` — same as `go test` |
| **Signature** | First and only parameter: `t *testing.T` |
| **File** | `*_test.ft` (e.g. `auth_test.ft`) |
| **Package** | Default: same package as code under test; optional `package foo_test` for black-box tests ([ADR-044](./ADR.md#adr-044-test-entrypoints-use-test-and-testingt)) |
| **`Providers(f)`** | `*testing.T` is **excluded** — never in Providers struct |

**`ensure` in tests:** refinement checks only (`result is Ok()`, `result is Err(T)`, type constraints) — not `ensure expr == …`. Failures lower to **`t.Helper()` + `t.Fatalf(...)`** (or `t.Errorf` when an `ensure` block is present).

### Table tests with `t.Run`

Use **`t.Run` + nested `with`** as the **permanent** table-test override path ([ADR-044](./ADR.md#adr-044-test-entrypoints-use-test-and-testingt)):

```forst
func TestExpireToken_table(t *testing.T) {
    token := Token { id: "t1", expiresAt: 1000 }
    cases := [] {
        { name: "expired", fixedMs: 3000 },
        { name: "valid",   fixedMs: 500 },
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            with ciUserApiServices() {
                with { Clock: &FakeClock { fixedMs: tc.fixedMs } } {
                    result := expireToken(token)
                    if tc.name == "expired" {
                        ensure result is Err(Expired {})
                    } else {
                        ensure result is Ok()
                    }
                }
            }
        })
    }
}
```

Lowers to Go subtests — standard `go test -run` / verbose output.

### Requirements + tests

Production **`use` / `with`** work unchanged inside test bodies:

- **Wiring root** — outer `with { … }` must satisfy all **`Providers(f)`** reachable in the test.
- **Fat fixture** — `with ciUserApiServices() { … }` for integration breadth.
- **Nested overlay** — inner `with { Clock: fake } { … }` shadows one capability ([ADR-008](./ADR.md#adr-008-nested-with-overlays-outer-wiring)).

```forst
func TestExpireTokenWithSharedFixture(t *testing.T) {
    token := Token { id: "t1", expiresAt: 1000 }
    with ciUserApiServices() {
        with { Clock: &FakeClock { fixedMs: 2000 } } {
            result := expireToken(token)
            ensure result is Err(Expired {})
        }
    }
}
```

### Fixtures (helpers, not test entrypoints)

Define **plain functions** returning Provider-shaped literals. Helpers **do not** take `t` unless the author wants it for logging.

```forst
// auth_test.ft or testfixtures/context.ft

func ciUserApiServices(): CIProviders {
    return {
        Logger:      &NopLogger {},
        Clock:       &FakeClock { fixedMs: 1_700_000_000_000 },
        UserRepo:    &InMemoryUserRepo { users: map[String]User{} },
        HttpClient:  &BlockedHttp {},
        EmailSender: &NoopEmail {},
        Metrics:     &NopMetrics {},
    }
}

type BlockedHttp = {}

func (BlockedHttp) get(url String): Result(Bytes, Error) {
    return Err(NetworkDisabled { url: url })
}
```

Handler-level test:

```forst
func TestCreateUserPersists(t *testing.T) {
    req := CreateUserRequest { name: "Ada", email: "a@b.c" }
    with ciUserApiServices() {
        resp := handleCreateUser(req)
        ensure resp is Ok()
    }
}
```

Selective override — nested `with` (no postfix override syntax):

```forst
func TestCreateUserRecordsEmail(t *testing.T) {
    recording := &RecordingEmail { messages: [] }
    with ciUserApiServices() {
        with { EmailSender: recording } {
            resp := handleCreateUser(CreateUserRequest { name: "Ada", email: "a@b.c" })
            ensure resp is Ok()
        }
    }
}
```

Optional one-off merge via user helper or nested `with` only — no postfix override syntax.

### Supported test paths

| Path | Role |
| --- | --- |
| **`Test*` in `*_test.ft` with `t *testing.T`** | **Primary** — normative |
| **Go `_test.go` calling exported generated helpers** | Escape hatch / brownfield |
| **Bare `func testFoo()` without `*testing.T`** | **Discouraged** — not discovered by `forst test` |

### Blocking network in CI

**Policy lives in helper types**, not keywords:

- `ciUserApiServices()` wires `BlockedHttp` — accidental real HTTP fails at runtime and is obvious in review.
- Teams may add **`localUserApiServices()`** for dev machines with `RealHttpClient`.
- CI runs tests that call `ciUserApiServices()`; linters can grep for `RealHttp` in `Test*` functions.

### Go emit for tests

Test functions lower to **`func TestXxx(t *testing.T)`** in **`*_test.go`**. Code under test lowers as elsewhere: **Providers struct** + wiring literals at call sites ([ADR-012](./ADR.md#adr-012-providers-struct-as-first-parameter), [ADR-044](./ADR.md#adr-044-test-entrypoints-use-test-and-testingt)). Run via **`forst test`** or **`go test`** on emitted output — see [forst-test RFC](../forst-test/README.md).

---

## TypeScript and sidecar boundary

Providers are **Go-side only** ([ADR-020](./ADR.md#adr-020-typescript-emit-excludes-providers-concept)).

| Layer | Providers |
| --- | --- |
| **Forst / Go** | `use`, `with` (Providers shape literal), inferred **`Providers(f)`**, `ciUserApiServices()`-style fixtures |
| **Sidecar wire** | **Data only** — JSON-serializable parameters and returns |
| **TypeScript client** | **Runnable exports only** — functions where **`Providers(f) = ∅`** after host wiring |

Functions with non-empty **`Providers(f)`** are **never** emitted to TypeScript. Declaring a sidecar / TS export on such a function is a **hard compile error** ([ADR-022](./ADR.md#adr-022-hard-error-on-sidecar-export-with-non-empty-providersf)). TS does not model Providers, `@needs`, or an Effect `R` channel. The sidecar host wires services before exposing a handler; TS callers invoke fully-provided endpoints with payload data only.

Go-side discovery JSON and LSP expose inferred **`Providers(f)`** for host authors and agents building `main` / service bundles — not for the TS client artifact ([ADR-023](./ADR.md#adr-023-go-side-discovery-json-for-providersf)).

### Brownfield (v1)

No bridge syntax for legacy Go `Deps` structs ([ADR-040](./ADR.md#adr-040-no-brownfield-bridge-syntax-v1)). Brownfield teams:

1. Rewrite handlers to `use` / `with`, or
2. Use Go `_test.go` escape hatch for hand-written struct literals, or
3. Follow a **conversion guide** mapping `type ServerDeps struct { … }` fields → `CIProviders` typedef + fixture helper.

---

## LLM ergonomics

Minimal surface helps agents and humans alike.

| Property | Why it helps |
| --- | --- |
| **Two keywords** | `use` = read dependency; `with` = write wiring. No parallel vocab (`require`, `supply`, `harness`, `capability`). |
| **Types as contracts** | Agent copies method sets from existing `type` definitions or imported Go interfaces — same task as writing a Go fake. |
| **No export clause** | Cannot drift between `uses Logger` annotation and body; inference is single source of truth. |
| **Shared helpers** | `ciUserApiServices()` is grep-able, composable, and valid Forst — agents generate one helper, then delta maps in inner `with`. |
| **Go lowering transparency** | Generated code resembles hand-written Go; agents can validate against `_test.go` patterns they already know. |

Recommended agent workflow:

1. Read inferred **`Providers(f)`** from discovery JSON / LSP (`providers: ["Logger", "UserRepo"]`).
2. Implement fakes as structs + methods matching those types.
3. Wire at boundary with `with wiring { … }` (shape literal or `ciUserApiServices()`).
4. For integration breadth, use `ciUserApiServices()`-style helpers; use **`t.Run`** for table cases and nested `with { Key: &fake }` for per-case wiring deltas ([ADR-044](./ADR.md#adr-044-test-entrypoints-use-test-and-testingt)).

---

## Examples

### End-to-end: token expiry

```forst
package auth

import (
    "fmt"
    "testing"
)

type Logger = {
    info(msg String)
}

type Clock = {
    now(): Int
}

type Token = {
    id:        String,
    expiresAt: Int,
}

type StdLogger = { prefix: String }

func (l StdLogger) info(msg String) {
    fmt.Println(l.prefix + msg)
}

type SystemClock = {}

func (SystemClock) now(): Int {
    // lowered to time.Now().UnixMilli() or host hook
    return 0
}

func expireToken(token Token): Result(Token, Error) {
    use logger: Logger
    use clock: Clock
    if token.expiresAt < clock.now() {
        logger.info("token expired: " + token.id)
        return Err(Expired { tokenId: token.id })
    }
    return Ok(token)
}

func handleRefresh(token Token): Result(Token, Error) {
    return expireToken(token)   // scope from outer `with`
}

func main() {
    with {
        Logger: &StdLogger { prefix: "[auth] " },
        Clock:  &SystemClock {},
    } {
        // serve handleRefresh inside wired scope
    }
}

// --- fakes ---

type NopLogger = {}

func (NopLogger) info(msg String) {}

type FakeClock = { fixedMs: Int }

func (c FakeClock) now(): Int { return c.fixedMs }

func TestExpiredToken(t *testing.T) {
    token := Token { id: "t1", expiresAt: 1000 }
    with {
        Logger: &NopLogger {},
        Clock:  &FakeClock { fixedMs: 2000 },
    } {
        result := expireToken(token)
        ensure result is Err(Expired {})
    }
}

error Expired { tokenId: String }
```

### HTTP + email handler (five requirements)

```forst
package users

import (
    "net/http"
    "testing"
)

type Logger = { info(msg String) }
type UserRepo = {
    find(id String): Result(User, Error)
    save(user User): Result(User, Error)
}
type EmailSender = {
    send(to String, subject String, body String): Result(Void, Error)
}
type HttpDoer = {
    do(req *http.Request): Result(*http.Response, Error)
}

func handleCreateUser(input CreateUserRequest): Result(CreateUserResponse, Error) {
    return createUserInternal(input)
}

func createUserInternal(input CreateUserRequest): Result(CreateUserResponse, Error) {
    use logger: Logger
    use repo: UserRepo
    use email: EmailSender
    user := User { id: newId(), name: input.name, email: input.email }
    saved := repo.save(user)?
    email.send(saved.email, "welcome", "hello " + saved.name)?
    logger.info("created user " + saved.id)
    return Ok(CreateUserResponse { id: saved.id })
}

func ciUserApiServices(): CIProviders {
    return {
        Logger:      &NopLogger {},
        UserRepo:    &InMemoryUserRepo { users: map[String]User{} },
        EmailSender: &NoopEmail {},
        HttpClient:  &BlockedHttp {},
    }
}

func TestCreateUserNoNetwork(t *testing.T) {
    with ciUserApiServices() {
        resp := handleCreateUser(CreateUserRequest { name: "Ada", email: "a@b.c" })
        ensure resp is Ok()
    }
}
```

---

## Open questions

| # | Question | Status |
| --- | --- | --- |
| 1 | **Providers struct dedup naming** | **Decided** — [ADR-013](./ADR.md#adr-013-deduped-providers-struct-naming) |
| 2 | **Provider typing** | **Decided** — [ADR-042](./ADR.md#adr-042-provider-and-providers-vocabulary) |
| 3 | **Superset wiring** | **Decided** — [ADR-019](./ADR.md#adr-019-superset-wiring-allowed) |
| 4 | **Emit / Providers struct fields** | **Decided** — [ADR-026](./ADR.md#adr-026-providers-struct-fields-by-value), [ADR-028](./ADR.md#adr-028-wiring-value-pointers-optional) |
| 5 | **Mock reuse** | **Convention** — [ADR-029](./ADR.md#adr-029-mock-reuse-by-convention-not-compiler-enforced) |
| 6 | **Method name casing** | Follow Go export rules on emit; Forst source stays idiomatic |
| 7 | **Cross-package test helpers** | Yes — plain functions, no special module kind |
| 8 | **Sidecar boundary** | **Decided** — [ADR-022](./ADR.md#adr-022-hard-error-on-sidecar-export-with-non-empty-providersf) |
| 9 | **Discovery JSON schema** | **Decided** — [§ Discovery JSON](#discovery-json); [ADR-023](./ADR.md#adr-023-go-side-discovery-json-for-providersf), [ADR-035](./ADR.md#adr-035-cross-package-graph-normative-before-ga) |
| 10 | **Parallel tests + mutable fakes** | Document immutability; prefer fresh `ciUserApiServices()` per test |
| 11 | **TS client emit** | **None** — [ADR-020](./ADR.md#adr-020-typescript-emit-excludes-providers-concept), [ADR-021](./ADR.md#adr-021-runnable-exports-only-when-providersf-is-empty) |
| 12 | **Brownfield bridge** | **Decided** — [ADR-040](./ADR.md#adr-040-no-brownfield-bridge-syntax-v1) |
| 13 | **Alias keys in `with`** | **Decided** — [ADR-041](./ADR.md#adr-041-root-contract-ident-only-in-with-keys) |
| 14 | **Context-field inference** | **Decided** — [ADR-005](./ADR.md#adr-005-context-parameters-never-contribute-to-providersf) |

---

## Summary

| Piece | Choice |
| --- | --- |
| Body binding | **`use`** |
| Boundary wiring | **`with`** |
| Contracts | **Ordinary types** — shapes, nominal types, imported Go types |
| Identification | **Inference** from `use` sites and transitive calls only; **`Providers(f)` is tooling sugar** |
| ProviderScope forwarding | **Always** inside `with` scopes; **nested `with`** overlays outer keys |
| Tests | **`Test*` + `*testing.T`** in `*_test.ft`; **`t.Run` + nested `with`** tables ([ADR-033](./ADR.md#adr-033-trun--nested-with-permanent-table-test-path), [ADR-044](./ADR.md#adr-044-test-entrypoints-use-test-and-testingt)) |
| **`with` form** | Providers shape literal, variable, or helper return ([ADR-030](./ADR.md#adr-030-with-takes-provider-shape-literals-only)) |
| Foreign Go | **Wrap** with Forst types; `use` the wrapper contract |
| Go emit | Deduped **`Providers_*`** struct (by value) + wiring literals ([ADR-012](./ADR.md#adr-012-providers-struct-as-first-parameter), [ADR-013](./ADR.md#adr-013-deduped-providers-struct-naming), [ADR-026](./ADR.md#adr-026-providers-struct-fields-by-value)) |
| GA gates | [ADR-034](./ADR.md#adr-034-lsp-ships-at-feature-ga), [ADR-035](./ADR.md#adr-035-cross-package-graph-normative-before-ga), [ADR-036](./ADR.md#adr-036-single-graph-source-for-checker-json-and-lsp) |

Two keywords. Types you already import. Wiring that lowers to Go a senior engineer would write by hand — with compile-time completeness the hand-written version lacks.
