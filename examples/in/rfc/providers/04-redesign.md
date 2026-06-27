# Function requirements — redesign (actionable forward path)

**Audience:** Language designers, compiler implementers, RFC coordinators.

**Status:** **Recommended direction** — supersedes [03 — Design critique](./03-design-critique-and-alternatives.md) as the actionable path; revises [01 — Normative spec](./01-normative-spec.md) before implementation.

**Depends on:** [00 — Prior art](./00-prior-art.md), [01 — Normative spec](./01-normative-spec.md), [02 — Examples](./02-examples.ft), [03 — Critique](./03-design-critique-and-alternatives.md), [effect CLI sketch](../effect/13-simple-effect-cli.md).

---

## Executive summary

**Verdict:** Ship **compile-time capability inference + Go lowering** (interfaces, synthesized needs structs, entry-point completeness). **Do not** ship a parallel `requirement` declaration primitive.

**Recommended surface:**

| Piece | Choice |
| --- | --- |
| Capability contracts | `type Logger = capability { … }` (or tagged shape) — **not** `requirement` |
| Body binding | **`use`** — optional when inference from `ctx` suffices |
| Wiring boundary | **`supply`** (or keep `provide` if Effect alignment wins) — **`supply ctx { … }` handler-first |
| Export docs | Optional **`uses Logger, Clock`** clause (compiler-checked against inference) |
| Go emit | Unchanged in spirit: interface + `fnNeeds` struct + struct literals |

**Honest bottom line:** This design **does not** eliminate fake struct authoring or `main`-level composition. It **does** reduce **dependency drift** in deep call graphs and makes test/production wiring **uniform and grep-able**. Without the **checker**, the keywords are ~85% redundant with idiomatic Go. **Inference-first** is the rent; syntax is the lease.

**Harness (integration tests):** Named **`harness`** presets layer CI-safe defaults over production wiring; tests declare **deltas only** via `override`. See [§ Integration test harness](#integration-test-harness).

**If v1 cannot ship inference:** defer keywords; emit interfaces from capability `type`s and let authors write deps structs manually ([03 § break-even](./03-design-critique-and-alternatives.md#break-even-heuristic)).

---

## Problem statement

### Testability

Backend Forst needs **swappable runtime services** (loggers, clocks, repositories) without reflection DI, globals, or monkey-patching. Tests must substitute **plain structs** that structurally satisfy method sets — the pattern Rust/Go already use, elevated with **compile-time wiring completeness** at `test` / `main` entry points.

### LLM agents

Agents generate tests from **named contracts with listed methods**. A parallel `requirement` keyword does not improve this over `type Logger = capability { … }` plus receiver methods (already demonstrated in [02-examples.ft](./02-examples.ft)). What agents need is **discovery metadata** (`uses` / `requirements` in JSON) and **localized wiring blocks** (`supply { … }`) at test entry — not a third type-declaration family.

### Runtime logic vs data

**Parameters = per-invocation input** (`Order`, `userId`, request bodies). **Capabilities = host-provided services** (logger, clock, DB pool). Forst must enforce this distinction culturally and in the checker:

- Sidecar wire format stays **data-only**; host builds `AppContext` server-side ([sidecar decisions](../sidecar/10-decisions.md)).
- Do **not** overload `context.Context`, `ensure`, or `Result` for DI ([00 §5.1](./00-prior-art.md#51-ensure--validation-and-control-flow-not-requirements)).
- The Go rule holds: if a struct field solves it, it is **data layout**, not a new mechanism ([01 §2](./01-normative-spec.md#the-go-rule-normative)).

### Relation to Effect / simple CLI

[13-simple-effect-cli.md](../effect/13-simple-effect-cli.md) sketches `getUserById(ctx context.Context, id int)` — explicit params, no `R` monad. Requirements are the **Forst-side** answer for **internal** capability tracking that lowers to the same Go bytes. TS Effect `provide` / Layers remain **optional** at the client boundary; Forst does not mandate a runtime effect interpreter on the Go path.

---

## Prior art synthesis

Full survey: [00 — Prior art](./00-prior-art.md).

| Family | Examples | Forst fit |
| --- | --- | --- |
| Environment monad `R` | Effect-TS, ZIO | Valuable **mental model** for TS audience; **rejected** as mandatory Go runtime |
| Algebraic handlers | Unison, Koka | Excellent tests; **hard** Go transpilation |
| Explicit interfaces | Go, Rust | **Primary lowering target** |
| Context parameters | Kotlin | Inspires `supply ctx { … }` sugar |

**Test patterns that work for agents** ([00 §4.2](./00-prior-art.md#42-test-instantiation-patterns-what-works-for-agents)): named interfaces, methods in types, wiring localized in `main` / `test` / `supply` block — not registries.

**Forst codebase today** ([00 §5](./00-prior-art.md#5-forst-codebase-audit-current-state)): `ensure` = guards/narrowing; no `require`/`provide` grammar; shapes + methods on concrete types already exist; transformer lacks **interface emission** from capability types and **deps struct synthesis**.

**Examples repo:** No committed Forst `test` blocks outside this RFC folder. Test patterns appear only in [02-examples.ft](./02-examples.ft) and RFC markdown (e.g. `agent-c-operator-syntax.md` Go `_test.go` sketch). Integration examples use `task example:*` goldens, not capability DI.

---

## Critique summary

From [03](./03-design-critique-and-alternatives.md) — condensed.

### Overlap with Go

`requirement` lowers to a Go `interface`; `require` lowers to a field read; `provide` lowers to a struct literal. **~85% surface redundancy** with careful hand-written Go. **Non-redundant value:** transitive need propagation, entry-point completeness, optional discovery metadata.

### Real value vs ceremony

| Pays rent | Does not |
| --- | --- |
| Leaf gains `use Database` → compile errors at callers / entry | Automatic mock codegen |
| `supply ctx { … }` at handler edge | Application-wide binding registry |
| Uniform test `supply { … }` blocks | Escape from fake method implementation |
| `uses` clause + discovery JSON | Layer composition graphs |

### Naming

`requirement` / `require` / `requires` collide in English and with “function requires param”. **`use` / `supply`** (or `provide`) pair better with **`ensure`** (verb-first body statements).

### Shape-based types

`02-examples.ft` already uses `type StdLogger = { … }` + `func (l StdLogger) info(…)`. Capability contracts should be **`type` + capability kind**, not a third decl primitive. Structural satisfaction reuses shape-guard rules.

---

## Testing boilerplate analysis

**Question:** Does the design actually reduce testing boilerplate?

**Method:** Same scenario across three styles — `expireToken` needs `Logger` + `Clock`; one test asserts expired token returns `Err(Expired)`. Count **lines that exist only for testing/DI** (interfaces/capability decls, fakes, wiring, body `require`/`use`). Exclude domain types, business logic, and pure assertions.

**Repo note:** Counts below are from representative sketches aligned with [02-examples.ft](./02-examples.ft). No other `examples/in/` files exercise this pattern yet.

### A. Plain Go / Forst today (manual)

Hand-written interfaces, per-function deps struct, struct literal at test entry. This is what Forst **already lowers to**.

```go
// --- capability interfaces (shared prod + test) ---
type Logger interface {
    Info(msg string)
    Error(msg string)
}

type Clock interface {
    Now() int
}

// --- business (prod) ---
type expireTokenNeeds struct {
    Logger Logger
    Clock  Clock
}

func expireToken(needs expireTokenNeeds, token Token) (Result[Token], error) {
    if token.ExpiresAt < needs.Clock.Now() {
        needs.Logger.Info("token expired: " + token.Id)
        return Err(Expired{TokenId: token.Id}), nil
    }
    return Ok(token), nil
}

// --- fakes (test-only) ---
type NopLogger struct{}

func (NopLogger) Info(msg string) {}
func (NopLogger) Error(msg string) {}

type FakeClock struct{ FixedMs int }

func (c FakeClock) Now() int { return c.FixedMs }

// --- test ---
func TestExpireTokenRejectsExpired(t *testing.T) {
    token := Token{Id: "t1", ExpiresAt: 1000}
    result, _ := expireToken(expireTokenNeeds{
        Logger: NopLogger{},
        Clock:  FakeClock{FixedMs: 2000},
    }, token)
    // assert Err(Expired) — 1–2 lines
}
```

| Category | Lines |
| --- | ---: |
| Interface decls (`Logger`, `Clock`) | 9 |
| `expireTokenNeeds` + field reads in body | 4 |
| Fake types + methods | 8 |
| Test wiring (literal + call) | 6 |
| **Total DI/test boilerplate** | **27** |
| **Per additional test** (reuse fakes) | **~6** |

### B. Normative v0 (`requirement` + `require` + `provide`)

```forst
requirement Logger {
    info(msg String)
    error(msg String)
}

requirement Clock {
    now(): Int
}

func expireToken(token Token) requires Logger, Clock
    : Result(Token, Error) {
    require logger: Logger
    require clock: Clock
    if token.expiresAt < clock.now() {
        logger.info("token expired: " + token.id)
        return Err(Expired { tokenId: token.id })
    }
    return Ok(token)
}

type NopLogger = {}
func (l NopLogger) info(msg String) {}
func (l NopLogger) error(msg String) {}

type FakeClock = { fixedMs: Int }
func (c FakeClock) now(): Int { return c.fixedMs }

test "expireToken rejects expired token" {
    token := Token { id: "t1", expiresAt: 1000 }
    result := expireToken(token) provide {
        Logger: NopLogger {},
        Clock:  FakeClock { fixedMs: 2000 },
    }
    ensure result is Err(Expired {})
}
```

| Category | Lines |
| --- | ---: |
| `requirement` decls | 10 |
| `requires` + `require` in body | 5 |
| Fake types + methods | 8 |
| Test `provide` map + call | 7 |
| **Total** | **30** |
| **Per additional test** | **~7** |

**vs plain Go:** **+3 lines** on first test — `requirement` duplicates interface syntax; `require` duplicates deps field access. Emitted Go is **identical** to A.

### C. Proposed redesign (capability `type` + inference-first + `supply`)

Handler-first profile: leaf uses explicit `use` only where no `ctx`; handler tests build one `TestContext`.

```forst
type Logger = capability {
    info(msg String)
    error(msg String)
}

type Clock = capability {
    now(): Int
}

// Leaf: explicit use (no ctx param)
func expireToken(token Token): Result(Token, Error) {
    use logger: Logger
    use clock: Clock
    if token.expiresAt < clock.now() {
        logger.info("token expired: " + token.id)
        return Err(Expired { tokenId: token.id })
    }
    return Ok(token)
}

// Handler: inference I2 from ctx (no use lines)
type AppContext = {
    logger: StdLogger,
    clock:  SystemClock,
}

func handleRefresh(token Token, ctx AppContext): Result(Token, Error) {
    supply ctx {
        return expireToken(token)
    }
}

// --- shared test fixtures ---
type NopLogger = {}
func (l NopLogger) info(msg String) {}
func (l NopLogger) error(msg String) {}

type FakeClock = { fixedMs: Int }
func (c FakeClock) now(): Int { return c.fixedMs }

// Test at leaf (direct)
test "expireToken rejects expired token" {
    token := Token { id: "t1", expiresAt: 1000 }
    result := expireToken(token) supply {
        Logger: NopLogger {},
        Clock:  FakeClock { fixedMs: 2000 },
    }
    ensure result is Err(Expired {})
}

// Test at handler (integration-style) — one context, many tests
test "handleRefresh via test context" {
    ctx := AppContext {
        logger: NopLogger {},
        clock:  FakeClock { fixedMs: 2000 },
    }
    result := handleRefresh(token, ctx)  // no supply block
    ensure result is Err(Expired {})
}
```

| Category | Lines (leaf test) | Lines (handler test) |
| --- | ---: | ---: |
| Capability `type` decls | 10 | 10 (shared) |
| `use` in body | 2 | 0 |
| `supply ctx` in handler | — | 3 |
| Fake types + methods | 8 | 8 (shared) |
| Test wiring | 7 | 4 (ctx literal + call) |
| **Total first test** | **27** | **25** (amortized) |
| **Per additional leaf test** | **~7** | **~2** (reuse `ctx`) |

**vs normative v0:** **−3 lines** at leaf (drop `requirement` + `requires`); **−5 lines** per handler-level test when `TestContext` is reused.

### Side-by-side wiring (test entry only)

| Style | Test entry snippet | Lines |
| --- | --- | ---: |
| Plain Go | `expireToken(expireTokenNeeds{Logger: NopLogger{}, Clock: FakeClock{…}}, token)` | 3 |
| Normative | `expireToken(token) provide { Logger: NopLogger {}, Clock: FakeClock {…} }` | 4 |
| Redesign leaf | `expireToken(token) supply { Logger: NopLogger {}, Clock: FakeClock {…} }` | 4 |
| Redesign handler | `ctx := TestAppContext {…}; handleRefresh(token, ctx)` | 2–4 once, then 1/call |

### When boilerplate goes down

| Situation | Reduction |
| --- | --- |
| Many tests through same handler / `AppContext` | **Handler test + shared `TestContext`** — biggest win |
| Deep trees when leaf gains a capability | **Checker** forces entry update — saves review time, not lines |
| `supply ctx` / `supply forward` in prod | Avoids repeated per-call maps |

### When boilerplate stays or moves

| Situation | What happens |
| --- | --- |
| **Fake struct authoring** | **Unchanged** — every design needs methods on `NopLogger`, `FakeClock`, `FakeUserRepo` ([02-examples.ft](./02-examples.ft) ~26 lines for three fakes) |
| **`use` / `require` in leaves** | Moves cost from explicit deps **parameter** to body statements — similar total |
| **Capability / interface decls** | Moves from Go `interface` to Forst `type = capability` — **same line count** |
| **Per-function `fnNeeds` in Go API** | Still emitted unless merge pass lands — **export surface churn**, not test lines |
| **Cross-package tests** | Still wire at process edge — same as Go `main` |

### Effect / Layer comparison (honesty check)

Effect-TS test:

```typescript
const runnable = program.pipe(
  Effect.provideService(Logger, { log: (s) => Effect.sync(() => {}) }),
  Effect.provideService(Clock, { now: () => 2000 }),
)
```

Comparable line count to `supply { … }`; Layers add **more** boilerplate for graphs Forst explicitly rejects. Forst wins on **Go-native fakes**, not on eliminating wiring.

### Testing boilerplate verdict

| Claim in RFC | Honest assessment |
| --- | --- |
| “Tests provide fake structs — no mock framework” | **True** — same as Go |
| “LLM reads requirement block → fake” | **True** for `type` + methods too |
| “Reduces test boilerplate” | **Mostly false** for single leaf tests; **true** for handler-level suites with shared context; **true** for **drift prevention**, not line count |
| “Readable wiring” | **True** — `supply` map keys grep requirement names |

**Design implication:** Optimize for **handler-first testing** and **shared test fixtures**, not leaf `supply` maps on every test. Document `TestAppContext` pattern prominently.

---

## App-level override flows

### Production

**Primary path — `supply ctx` at handler:**

```forst
func CreateUser(input CreateUserRequest, ctx AppContext): Result(CreateUserResponse, Error) {
    supply ctx {
        return createUserInternal(input)
    }
}
```

1. Sidecar host / `main` builds `AppContext` once per request or boot.
2. Compiler maps `Logger` → `ctx.logger`, `UserRepo` → `ctx.db` (structural + naming convention).
3. Emitted Go copies fields into callee needs literals inside the block.

**`main` entry:**

```forst
func main() {
    ctx := AppContext {
        logger: StdLogger { level: "info" },
        clock:  SystemClock {},
        db:     PostgresDatabase { dsn: env("DATABASE_URL") },
    }
    supply ctx { runServer() }
}
```

**Still manual:** picking concrete impls, pools, config, sidecar bootstrap in Go, cross-package wiring at process edge.

### Test

**Leaf test** — `supply { … }` suffix or block (full transitive set for calls inside).

**Suite test** — shared `TestAppContext` / `supply ctx` at describe scope (Forst `test` block nesting TBD).

```forst
test "user flows" {
  ctx := TestAppContext { logger: NopLogger {}, clock: FakeClock { fixedMs: 0 }, db: FakeUserRepo {…} }
  supply ctx {
    test "getUser" { … }
    test "saveUser" { … }
  }
}
```

### Partial override

```forst
func auditAction(action String, ctx AppContext) {
    use logger: Logger
    doAction(action) supply { Logger: AuditLogger { sink: ctx.logger } }
}
```

Inner `supply` shadows `Logger` for the callee; other capabilities merge from scope scope (`supply forward`).

**Still manual:** author knows callee needs or trusts scope merge + compiler errors.

### Flow diagram

```
Production                          Test
──────────                          ────
main / sidecar host                 test block / shared fixture
    │                                   │
    ▼                                   ▼
build AppContext                   build TestAppContext or supply map
    │                                   │
    ▼                                   ▼
supply ctx { handlers }            supply ctx { … }  OR  supply { … } { leaf }
    │                                   │
    ▼                                   ▼
internal calls + forward           same inference rules
    │
    ▼
optional supply { Logger: override }  ← shadow only
```

---

## Integration test harness

**Audience:** Same as this doc — language designers and implementers evaluating whether Forst needs a **first-class test harness** on top of `type = capability`, `use`, and `supply`.

**Status:** **Recommended addition** to the redesign surface — not a substitute for inference/checker work, but the piece that makes **handler-level integration tests** scale when a call graph pulls **8+ capabilities**.

**Problem addressed:** Unit tests swap one or two fakes at a leaf `supply { … }` block. Integration tests through a handler reuse `AppContext` but still force authors to **enumerate every capability** the transitive closure needs — and to remember which impls must be **hermetic in CI** (no network, no email, no real DB). That is combinatorial override explosion, not a fake-authoring problem.

---

### 1. Problem — unit fakes vs integration harness

| Concern | Unit test (leaf) | Integration test (handler) |
| --- | --- | --- |
| **Scope** | One function, 1–3 capabilities | Handler + transitive callees, often **8–15** capabilities |
| **Wiring site** | `supply { Logger: …, Clock: … }` suffix | Shared `TestAppContext` or repeated full maps |
| **CI safety** | Author picks fakes consciously | Easy to accidentally wire `RealHttpClient` or `SMTPEmailSender` |
| **Per-test variance** | Whole map is small | Only **one** dep differs (e.g. recording HTTP); rest should inherit |
| **Agent ergonomics** | Copy capability block → fake | Must discover **full** need set + CI policy |

**Unit fakes remain necessary.** Harness does not codegen mocks. It **centralizes defaults** and **merge semantics** so integration tests list **deltas**, not the full capability matrix every time.

**Combinatorial explosion (concrete):** Suppose `handleCreateUser` transitively needs `{ Logger, Clock, UserRepo, HttpClient, EmailSender, Metrics, IdGenerator, Config }`. Eight capabilities × three environments (CI hermetic, local integration, prod) × occasional per-test overrides (recording HTTP, failing email) yields **24+ wiring variants** if each test builds a literal. A harness preset collapses the **8-capability baseline** to **one declaration**; tests override **0–2** fields.

**What harness is not:** A DI container, a mock framework, or testcontainers. It is **syntactic sugar + compile-time merge** over the same struct literals `supply` already lowers to.

---

### 2. Harness architecture — layered wiring

Four layers, outermost wins on conflict (test override > harness preset > production default). Production defaults are **not** auto-inherited in tests — tests opt into a harness explicitly — but harness authors **mirror production shape** with safe impls.

```
┌─────────────────────────────────────────────────────────────┐
│  Layer 4 — Per-test override (0–N capability keys)          │
├─────────────────────────────────────────────────────────────┤
│  Layer 3 — Harness preset (CI, LocalIntegration, …)         │
├─────────────────────────────────────────────────────────────┤
│  Layer 2 — Suite / nested test scope override (optional)    │
├─────────────────────────────────────────────────────────────┤
│  Layer 1 — Production wiring (main / AppContext bootstrap)  │
│            Reference only in tests; not executed in CI      │
└─────────────────────────────────────────────────────────────┘
```

| Layer | Role | Example |
| --- | --- | --- |
| **Production** | Real impls, env config, pools | `PostgresDatabase`, `SendGridEmail`, `StdLogger` |
| **Harness preset** | CI-safe profile: in-memory, fixed clock, blocked network | `harness CI { … }` |
| **Suite scope** | Shared override for a `test` group | `test suite "API" uses CI { override … }` |
| **Per-test** | Delta only | `override HttpClient with RecordingHttp { … }` |

#### Layer merging semantics

Harness merge reuses **`supply` shadow rules** from [App-level override flows](#app-level-override-flows):

1. **`supply ctx`** (or `supply harnessCtx`) builds a **concrete context struct** — same as production `AppContext`.
2. **Harness preset** is a **capability-keyed map** (`Logger: NopLogger {}`, …) aligned with `Needs(entry)` for the package or declared **`uses`** set on the harness.
3. **Merge** = struct literal fill: start from harness defaults; apply suite overrides; apply per-test overrides. **Rightmost wins** per capability key.
4. **Unmapped capabilities** in the transitive closure → **compile error** (same as bare `supply`). Harness must be **complete for its declared scope** or explicitly **`partial`** (see open questions).
5. **`supply ctx { … }` inside test** uses merged context — inference I2–I4 unchanged.

**Key invariant:** Harness does not introduce a runtime registry. Lowered Go is **`AppContext{…}` literal + optional `mergeHarness(base, overrides)` helper** when overrides are dynamic.

---

### 3. Concrete Forst syntax

#### Keyword choice: `harness` (not `test harness`)

| Candidate | Verdict |
| --- | --- |
| **`harness CI { … }`** | **Preferred** — noun decl, grep-friendly, matches industry term |
| `test harness CI` | Parses awkwardly next to `test "…"` blocks |
| `profile CI` | Collides with build profiles / sidecar config |
| `fixture App` | Implies data setup, not capability wiring |

**`harness`** is package-scoped (like `type`). **`uses`** on a harness lists capabilities the preset must satisfy — compiler checks completeness against implementations listed in the body.

#### Declaring a harness preset

```forst
harness CI uses Logger, Clock, UserRepo, HttpClient, EmailSender, Metrics {
    Logger:      NopLogger {},
    Clock:       FakeClock { fixedMs: 1_700_000_000_000 },
    UserRepo:    InMemoryUserRepo { users: map[String]User{} },
    HttpClient:  BlockedHttp {},           // fail closed — no network
    EmailSender: NoopEmail {},
    Metrics:     NopMetrics {},
}
```

Optional **local integration** profile — same shape, selective real deps:

```forst
harness LocalIntegration uses Logger, Clock, UserRepo, HttpClient, EmailSender, Metrics {
    Logger:      StdLogger { level: "debug" },
    Clock:       SystemClock {},
    UserRepo:    InMemoryUserRepo { users: map[String]User{} },
    HttpClient:  RealHttpClient { baseUrl: env("API_URL") },  // real network OK
    EmailSender: NoopEmail {},                                 // still no email in dev CI
    Metrics:     NopMetrics {},
}
```

**Environment selection** (optional sugar — can stay manual in v1):

```forst
// Resolved at test compile or via build tag / env("FORST_TEST_HARNESS")
default harness CI

// Or explicit in test file:
when env("CI") == "true" { default harness CI } else { default harness LocalIntegration }
```

Prefer **`default harness CI`** plus **`test "…" uses LocalIntegration`** for tests that need real HTTP locally — avoids magic env coupling in the checker core.

#### Binding tests to a harness

```forst
test "createUser returns 201" uses CI {
    ctx := harnessContext(CI)   // lowered: merged AppContext literal
    resp := handleCreateUser(req, ctx)
    ensure resp.status == 201
}
```

**Sugar** — implicit context when the test body calls handlers with `supply harness`:

```forst
test "createUser returns 201" uses CI {
    supply CI {
        resp := handleCreateUser(req)
        ensure resp.status == 201
    }
}
```

`supply CI { … }` desugars to: build context from harness preset + empty override map; run body with scope merge.

#### Selective override (deltas only)

```forst
test "createUser sends welcome email" uses CI {
    override {
        EmailSender: RecordingEmail { sent: [] },
    } {
        supply CI {
            handleCreateUser(req)
            ensure RecordingEmail.last().subject == "welcome"
        }
    }
}
```

Single-capability override sugar:

```forst
test "retries on 503" uses CI {
    override HttpClient with FlakyHttp { failures: 2, then: CI.HttpClient } {
        supply CI { handleCreateUser(req) }
    }
}
```

**`CI.HttpClient`** refers to the preset’s default for that capability — useful when overriding one field but delegating to the harness default after a stub phase (lowered as nested struct or wrapper).

#### Suite-level harness scope

```forst
test suite "User API" uses CI {
    override HttpClient with RecordingHttp { calls: [] }

    test "GET /users/:id" { … }      // inherits CI + RecordingHttp
    test "POST /users" { … }
}
```

Nested suites merge outer overrides inward (same shadow semantics).

---

### 4. Go lowering

Harness lowers to **plain Go test helpers** — no reflection, no service locator.

| Forst | Emitted Go (sketch) |
| --- | --- |
| `harness CI uses … { Logger: NopLogger{}, … }` | `func harnessCI() AppContext { return AppContext{ Logger: NopLogger{}, … } }` |
| `supply CI { body }` | `ctx := harnessCI(); supplyCtx(ctx, func() { body })` — inline or helper |
| `override { EmailSender: X } { … }` | `ctx := mergeHarness(harnessCI(), HarnessOverrides{ EmailSender: X }); …` |
| `harnessContext(CI)` | alias for `harnessCI()` |
| Discovery JSON | `"harnesses": [{ "name": "CI", "capabilities": ["Logger",…], "blocked": ["HttpClient"] }]` |

**`mergeHarness` helper** (generated once per package when overrides use non-literal expressions):

```go
func mergeHarness(base AppContext, o HarnessOverrides) AppContext {
    if o.EmailSender != nil { base.EmailSender = o.EmailSender }
    if o.HttpClient != nil { base.HttpClient = o.HttpClient }
    // … only overridden fields
    return base
}
```

**Test main entry:** No separate process required. Harness is **test-scoped** unless an integration test uses `test main` (future) to boot HTTP — then `supply CI { runTestServer() }` mirrors production `main` with the same merge path.

**Relation to `supply ctx`:** Production `handleCreateUser(input, ctx AppContext)` unchanged. Tests pass `harnessCI()` as `ctx`. Inner calls use `supply ctx { … }` inside the handler — **same inference path as production**.

---

### 5. Comparison — harness vs alternatives

| Approach | Strengths | Weaknesses vs Forst harness |
| --- | --- | --- |
| **Plain Go test helpers** (`newTestApp(t)`) | Zero new syntax; full control | No compile-time completeness; helpers drift from `Needs`; agents grep poorly |
| **Effect Layers** | Composable graphs, `Layer.provide` | Heavy TS/runtime story; rejected for Go path; more ceremony than struct literals |
| **testify/mock** | Expectations on interfaces | Mock framework coupling; not structural fakes; brittle for integration breadth |
| **testcontainers** | Real Postgres/Redis in CI | Slow, flaky, not hermetic; complements harness (`harness IntegrationDB` optional profile) |
| **Wire / fx DI** | Graph validation | Reflection; conflicts with Forst’s explicit struct lowering |

**Honest positioning:** Harness is **~80% a naming and merge convention** over hand-written `newTestApp`. The **20% that matters** is **compile-time completeness** (harness `uses` vs transitive `Needs`) and **discovery JSON** for agents. Without checker integration, a Go helper function is equivalent.

**vs Effect Layers specifically:** Effect tests stack `Layer.provide` calls — similar line count to `override { … }` chains. Forst harness is **flatter**: one preset map + deltas, lowers to **one struct literal**, not a layered monad interpreter.

---

### 6. LLM / agent ergonomics

Agents fail integration tests when they:

1. Omit a capability the handler transitively needs.
2. Wire a **real** network client in CI.
3. Copy-paste **8-line** `supply` maps from a sibling test with one field different.

**Harness contract for agents:**

```json
{
  "harnesses": [
    {
      "name": "CI",
      "capabilities": {
        "Logger": "NopLogger",
        "HttpClient": "BlockedHttp",
        "EmailSender": "NoopEmail"
      },
      "policy": { "network": "blocked", "email": "noop", "db": "in-memory" }
    }
  ],
  "handlers": [
    { "name": "handleCreateUser", "uses": ["Logger", "Clock", "UserRepo", "HttpClient", "EmailSender"] }
  ]
}
```

**Agent workflow (normative recommendation for tooling docs):**

1. Pick **`CI`** unless the user asks for local integration or testcontainers.
2. List **only overrides** vs `CI` (typically 0–2 capabilities).
3. Implement override fakes by copying **capability method sets** from discovery — same as today.
4. Never wire `RealHttpClient` / `PostgresDatabase` when `policy.network == "blocked"`.

**Lint (future):** flag `Real*` impl types inside `uses CI` tests unless explicitly annotated `allowNetwork`.

---

### 7. Worked example — HTTP handler, five capabilities

Handler pulls **Logger**, **Clock**, **UserRepo**, **HttpClient**, **EmailSender** (plus **Metrics** in harness for realism).

```forst
package users

// --- capabilities (unchanged redesign) ---
type Logger = capability { info(msg String); error(msg String) }
type Clock = capability { now(): Int }
type UserRepo = capability {
    find(id String): Result(User, Error)
    save(user User): Result(User, Error)
}
type HttpClient = capability {
    get(url String): Result(Bytes, Error)
    post(url String, body Bytes): Result(Bytes, Error)
}
type EmailSender = capability { send(to String, subject String, body String): Result(Void, Error) }
type Metrics = capability { increment(name String) }

type AppContext = {
    logger:      StdLogger,
    clock:       SystemClock,
    db:          PostgresDatabase,
    http:        RealHttpClient,
    email:       SendGridEmail,
    metrics:     PrometheusMetrics,
}

// --- production fakes / stubs for harness ---
type NopLogger = {}
func (NopLogger) info(msg String) {}
func (NopLogger) error(msg String) {}

type FakeClock = { fixedMs: Int }
func (c FakeClock) now(): Int { return c.fixedMs }

type InMemoryUserRepo = { users: map[String]User }
func (r InMemoryUserRepo) find(id String): Result(User, Error) { … }
func (r InMemoryUserRepo) save(user User): Result(User, Error) { … }

type BlockedHttp = {}
func (BlockedHttp) get(url String): Result(Bytes, Error) {
    return Err(NetworkDisabled { url: url })
}
func (BlockedHttp) post(url String, body Bytes): Result(Bytes, Error) {
    return Err(NetworkDisabled { url: url })
}

type NoopEmail = {}
func (NoopEmail) send(to String, subject String, body String): Result(Void, Error) {
    return Ok(Void {})
}

type NopMetrics = {}
func (NopMetrics) increment(name String) {}

// --- harness presets ---
harness CI uses Logger, Clock, UserRepo, HttpClient, EmailSender, Metrics {
    Logger:      NopLogger {},
    Clock:       FakeClock { fixedMs: 1_700_000_000_000 },
    UserRepo:    InMemoryUserRepo { users: map[String]User{} },
    HttpClient:  BlockedHttp {},
    EmailSender: NoopEmail {},
    Metrics:     NopMetrics {},
}

// --- handler (inference from ctx) ---
func handleCreateUser(input CreateUserRequest, ctx AppContext): Result(CreateUserResponse, Error) {
    supply ctx {
        return createUserInternal(input)
    }
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

// --- tests: deltas only ---
test "createUser persists user" uses CI {
    supply CI {
        repo := InMemoryUserRepo { users: map[String]User{} }
        override { UserRepo: repo } {
            resp := handleCreateUser(CreateUserRequest { name: "Ada", email: "a@b.c" }, harnessContext(CI))
            ensure resp is Ok(_)
            ensure repo.users[resp.id].name == "Ada"
        }
    }
}

test "createUser records welcome email" uses CI {
    override EmailSender with RecordingEmail { messages: [] } {
        supply CI {
            handleCreateUser(CreateUserRequest { name: "Ada", email: "a@b.c" }, harnessContext(CI))
            ensure RecordingEmail.last().subject == "welcome"
        }
    }
}

test "createUser does not hit network" uses CI {
    supply CI {
        // BlockedHttp would fail test if createUserInternal regressed to HTTP
        result := handleCreateUser(CreateUserRequest { name: "X", email: "x@y.z" }, harnessContext(CI))
        ensure result is Ok(_)
    }
}
```

**Note:** `harnessContext(CI)` + inner `override` nesting is intentionally verbose in this sketch to show merge layers — sugar forms in §3 collapse common cases.

---

### 8. Line-count / boilerplate — does harness win?

**Scenario:** Same handler, **10 integration tests**, **8 capabilities**, **1 test** needs `RecordingEmail`, **1 test** needs custom `InMemoryUserRepo` seed.

| Style | First test | Per additional test | Tests needing override |
| --- | ---: | ---: | ---: |
| **Full `supply` map every test** | ~12 lines wiring | ~12 lines | ~14 lines (map + fake) |
| **Shared `TestAppContext` variable** | ~10 lines setup + ~2/call | ~2 | ~6 (mutate or rebuild ctx) |
| **`harness CI` + `supply CI`** | ~8 lines harness decl (amortized) | **~3–4** | **~5–7** (`override` block) |

**Harness declaration cost:** ~8–12 lines once per profile (CI, LocalIntegration). Amortizes after **2–3** integration tests.

| Claim | Assessment |
| --- | --- |
| Harness eliminates fake structs | **False** — same `NopLogger`, `BlockedHttp`, etc. |
| Harness reduces **per-test** wiring | **True** — ~12 lines → ~3–4 for the common case |
| Harness prevents CI footguns | **True** when preset uses `BlockedHttp` / `NoopEmail` and checker lint enforces `uses CI` |
| Harness helps agents | **True** — preset + delta is a strict prompt contract |
| Harness beats shared Go helper without syntax | **Marginal** unless **`uses` completeness** and discovery JSON ship |

**Verdict:** Harness is **not** worth a new keyword if v1 only has leaf tests. It **is** worth it when **handler integration suites** are the default testing style — matching the conclusion in [Testing boilerplate analysis](#testing-boilerplate-analysis).

---

### 9. Harness open questions

| # | Question | Lean |
| --- | --- | --- |
| H1 | **Lifecycle** — per-test fresh harness vs shared mutable repo | **Fresh default** (`harnessContext` returns new struct); document shared-state tests as explicit |
| H2 | **Parallel tests** | Harness ctx must be **per-goroutine** or immutable; mutable fakes (`RecordingEmail`) require **t.Parallel** discipline or subtest isolation |
| H3 | **Hermetic vs selective integration** | **`CI`** hermetic by default; **`LocalIntegration`** opt-in; optional **`testcontainers` profile** as third harness, not built-in |
| H4 | **`partial harness`** | Allow incomplete preset + force explicit override for missing caps, or require full `uses` set? **Require full set** for v1 |
| H5 | **Override syntax** | `override { … }` block vs `supply CI override { … }` postfix — pick one; block form mirrors `supply` |
| H6 | **Production inheritance** | Should `harness CI extends Production` copy field names only? **Defer** — too magic; explicit maps safer |
| H7 | **Cross-package harness** | Re-export `harness CI` from `testfixtures` package? **Yes** — lowered as exported Go helper |
| H8 | **Sidecar e2e** | Host builds real `AppContext`; Forst harness is **Go-side unit/integration** only — sidecar tests stay in TS/Go process tests |

---

### 10. Harness next steps (checker-adjacent)

| Phase | Work |
| --- | --- |
| **H-a** | Parse `harness` decl + `uses CI` on `test` / `test suite` |
| **H-b** | Checker: harness `uses` ⊆ implementations; test entry `Needs` ⊆ harness + overrides |
| **H-c** | Lower `supply CI` → preset helper; `override` → `mergeHarness` |
| **H-d** | Discovery JSON: harness catalog + policies |
| **H-e** | Example in [02-examples.ft](./02-examples.ft) + golden `_test.go` with `task example:providers` |

**Dependency:** Phases **1–5** in [Next steps](#next-steps) (`use`, `supply`, inference) must land first; harness is **phase 6–7** sugar.

---

## Recommended redesign

Normative syntax using **existing `type` + shapes + methods**. Inference-first; optional exported `uses` for docs.

### 1. Capability contracts — drop `requirement`

```forst
type Logger = capability {
    info(msg String)
    error(msg String)
}

type Clock = capability {
    now(): Int
}

type UserRepo = capability {
    find(id String): Result(User, Error)
    save(user User): Result(User, Error)
}
```

**Rules:**

- `capability { … }` on `type` → emit Go `interface`, not struct.
- Implementations: concrete `type` + receiver methods (as `StdLogger` in [02-examples.ft](./02-examples.ft)).
- Distinct type names = distinct capabilities (`AuditLogger` vs `AppLogger`) even if method sets overlap.
- Alternative sugar: `type Logger = { info(msg String) }` with capability kind inferred when body contains only method signatures — **prefer explicit `capability` tag** for clarity.

**Lowering:** identical to [01 §6.1](./01-normative-spec.md#61-requirement--interface) but source is `TypeDefNode` with capability kind.

### 2. Body binding — `use` (optional with inference)

```forst
use <name>: <CapabilityType>
use <CapabilityType>          // name = lowerCamel(type ident)
```

**When required:**

- Leaf functions **without** a `ctx` / `deps` parameter.
- When author wants explicit readability despite inference.

**When optional (inference I2):**

```forst
func handleGetUser(id String, ctx AppContext): Result(User, Error) {
    ctx.logger.info("getUser")
    return ctx.db.find(id)
}
// Inferred Needs = { Logger, UserRepo } from selector types
```

**Inference rules:**

| Rule | Description |
| --- | --- |
| **I1** | `use x: T` ⇒ `T ∈ Needs(f)` |
| **I2** | Method call on `ctx.field` where `typeof(field)` satisfies capability `T` ⇒ `T ∈ Needs(f)` |
| **I3** | `f` calls `g` ⇒ `Needs(f) ⊇ Needs(g)` (transitive) |
| **I4** | `main` / `test` / exported handlers: all needs supplied |
| **I5** | Nominal capability types distinct by name |
| **I6** | Per-request payload fields excluded — only capability kinds |

**Default:** I2 **on** for functions with `ctx`/`deps` param; I1 **required** for leaves without context.

### 3. Wiring — `supply` (recommended) or `provide`

**Keyword choice:** **`supply`** avoids “provide data” ambiguity next to data params; **`provide`** keeps Effect-TS vocabulary. Pick one; this doc uses **`supply`**. Aliases acceptable in parser if both documented.

```forst
// Postfix on call
getUser(id) supply { UserRepo: ctx.db }

// Block — handler-first
supply ctx {
    saveOrder(req)
    notify(req.id)
}

// Explicit map block
supply { UserRepo: FakeUserRepo { users: … } } {
    result := getUser("42")
}

// Forwarding (modifier, not keyword)
expireToken(token) supply forward
```

**`supply ctx`:** field resolution `Logger` → `ctx.logger`, `UserRepo` → `ctx.userRepo`; override via explicit map.

**Lowering:** unchanged from [01 §6](./01-normative-spec.md#6-go-lowering-strategy) — `fnNeeds` struct literal, field copies.

### 4. Optional export clause — `uses`

```forst
func expireToken(token Token) uses Logger, Clock : Result(Token, Error) {
    use logger: Logger
    use clock: Clock
    …
}
```

Must match inferred set exactly. Omitted on internal helpers. Feeds LSP / discovery JSON.

### 5. End-to-end example

```forst
package demo

type Logger = capability { info(msg String) }

type StdLogger = { level: String }
func (l StdLogger) info(msg String) {}

type AppContext = { logger: StdLogger }

func greet(name String) {
    use logger: Logger
    logger.info("hello " + name)
}

func handleGreet(name String, ctx AppContext) {
    supply ctx { greet(name) }
}

func main() {
    ctx := AppContext { logger: StdLogger { level: "info" } }
    supply ctx { handleGreet("world") }
}

type NopLogger = {}
func (l NopLogger) info(msg String) {}

test "greet logs" {
    supply { Logger: NopLogger {} } { greet("Ada") }
}
```

### 6. Go lowering (unchanged in spirit)

| Forst | Go |
| --- | --- |
| `type Logger = capability { … }` | `type Logger interface { … }` |
| `use logger: Logger` | `logger := needs.Logger` |
| `Needs(f) = {Logger, Clock}` | `type fNeeds struct { Logger Logger; Clock Clock }` |
| `supply ctx { g(x) }` | `g(gNeeds{Logger: ctx.Logger, …}, x)` |
| Empty needs | No synthetic struct |

**Transformer work:** capability kind detection; interface emission; `fnNeeds` synthesis in `transformFunction`; reuse shape guard for satisfaction.

---

## Comparison table

| Criterion | Plain Go | v0 normative (`requirement`/`require`/`provide`) | **Redesign** (`type` capability + `use`/`supply`) |
| --- | --- | --- | --- |
| Capability decl | `interface` | `requirement` | `type = capability` |
| Body bind | deps param / field read | `require` | `use` (optional w/ I2) |
| Wiring | struct literal | `provide` | `supply` / `provide` |
| Transitive check | Manual | Compiler | Compiler |
| Entry completeness | Manual | Compiler | Compiler |
| Handler pattern | manual field copy | `provide ctx` | `supply ctx` (primary) |
| Fake structs in tests | Manual | Manual | Manual |
| Leaf test wiring lines | ~6 | ~7 | ~7 (leaf) / ~2 (handler suite) |
| Parallel type system | No | Yes (`requirement`) | No |
| Effect TS vocabulary | No | `provide` | Optional `provide` alias |
| Go emit | Native | Synthesized | Synthesized |
| LLM contract source | interface | `requirement` block | `type` + methods |
| Discovery metadata | Comments | `requires` + JSON | `uses` + JSON |

---

## Migration from `01-normative-spec.md`

| v0 (`01`) | Redesign (`04`) |
| --- | --- |
| `requirement Logger { … }` | `type Logger = capability { … }` |
| `require logger: Logger` | `use logger: Logger` (or infer from `ctx`) |
| `provide { … }` | `supply { … }` (or keep `provide`) |
| `provide forward` | `supply forward` |
| `requires Logger, Clock` | `uses Logger, Clock` |
| `RequirementDecl` AST | `TypeDefNode` + capability kind |
| Diagnostics | Rename messages; anchor on `supply` / `use` |

**Semantic preservation:** inference graph, structural satisfaction, entry-point rules, sidecar data-only wire — **unchanged**.

**Doc status:** [01](./01-normative-spec.md) remains historical normative target until implementation RFC lands; **implementers should follow 04**.

---

## Open questions

1. **Needs struct merge** — one `PackageDeps` per distinct capability set vs per-function `fnNeeds` ([01 §12](./01-normative-spec.md#12-open-questions)).
2. **`supply` vs `provide`** — single keyword vote (Effect alignment vs Forst branding).
3. **I2 strictness** — infer from `ctx.logger.info` only when `ctx` param name matches convention (`ctx`, `deps`, `app`)?
4. **Nested `test` blocks** — shared `supply ctx` scope for subtests?
5. **Partial `supply` suffix** — scope merge rules when map is strict subset ([03](./03-design-critique-and-alternatives.md) medium priority).
6. **Built-in capabilities** — prelude `Env`, `Random` as predeclared `type`s?
7. **TS emit** — JSDoc `@uses` vs Effect `R` branded types.
8. **Lint** — discourage `supply forward` on exported surfaces?
9. **Capability-only shapes** — `type Logger = { info(msg String) }` without `capability` keyword if method-only?
10. **Harness `partial` vs complete `uses`** — see [harness open questions](#9-harness-open-questions).
11. **Harness override syntax** — block `override { … }` vs postfix on `supply CI`.
12. **Parallel / mutable fakes** — `RecordingEmail` + `t.Parallel` policy (document vs lint).

---

## Next steps

**Order:** checker before syntax sugar.

| Phase | Work |
| --- | --- |
| **1. Inference checker** | Collect `use` + I2 from selectors; transitive `Needs(f)`; entry-point completeness errors |
| **2. Capability kinds + Go interfaces** | `type = capability` → interface emission; structural satisfaction |
| **3. Needs struct synthesis** | `fnNeeds` prepend param; empty-needs skip |
| **4. Parser: `use`, `supply`** | Postfix + block forms; `supply forward` |
| **5. `supply ctx` lowering** | Field extraction + naming convention |
| **6. Optional `uses` clause** | Export docs + discovery JSON |
| **7. Examples + tasks** | Update [02-examples.ft](./02-examples.ft); add `task example:providers` when compiles |
| **8. Needs merge pass** | Package-level dedup (ABI stability) |
| **9. Harness presets** | `harness CI`, `uses CI` on tests, `override` merge lowering — [§ Integration test harness](#integration-test-harness) |
| **10. Harness discovery** | JSON catalog + CI policy hints for agents |

**Do not implement first:** `requirement` decl, operator slots (`?T`), scope `using` (Agent B).

**Validation:** Unit tests per inference rule; integration example mirroring [02-examples.ft](./02-examples.ft) with golden Go in `examples/out/`.

---

## Document status

**Recommended forward path.** Supersedes [03](./03-design-critique-and-alternatives.md) for implementation decisions. Revises [01](./01-normative-spec.md) surface syntax; [00](./00-prior-art.md) remains research reference.
