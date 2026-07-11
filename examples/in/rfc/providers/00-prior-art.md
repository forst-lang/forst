# Prior art: function requirements, effects, and testable dependency injection

**Audience:** Language designers and compiler contributors researching a Forst **function requirements** (or **capabilities**) system for testability and explicit dependency tracking.

**Depends on:** [effect hub](../effect/00-overview.md), [errors hub](../errors/README.md), [optionals cross-language comparison](../optionals/05-cross-language-comparison.md), [guard / `ensure`](../guard/guard.md).

**Status:** Research reference — not normative language design.

---

## 1. Problem framing

Forst targets **TypeScript-familiar** backend authors who want **Go-grade performance** and **explicit, traceable behavior**. A recurring design question is how functions declare **what they need** (database, clock, HTTP client, logger) separately from **what they receive as input** (user id, request body), and how tests **swap implementations** without reflection-heavy DI frameworks.

This document surveys prior art under nine evaluation axes:

| Axis | Question |
| --- | --- |
| **Syntax clarity** | Can a non-expert read a signature and understand dependencies? |
| **Explicitness / traceability** | Is the **source** of each requirement visible at the call site and in types? |
| **Inference vs explicit import** | Does the compiler infer requirements, or must authors import/provide them manually? |
| **Test / mock ergonomics** | How easy is it to substitute fakes at test time? |
| **Boilerplate** | Ceremony for declaring, wiring, and composing dependencies |
| **Go transpilation feasibility** | Can requirements lower to idiomatic Go without runtime magic? |
| **Data vs runtime-logic separation** | Are **inputs** (data) cleanly separated from **capabilities** (services)? |
| **LLM / agent test generation** | Can an agent infer mocks from types/signatures alone? |
| **Gap vs Forst goals** | Where the approach conflicts with Forst's Go interop, `Result`/`ensure`, or sidecar story |

**Legend for matrix cells:** ★★★ excellent · ★★ workable · ★ weak · — not applicable

---

## 2. Systems surveyed (short profiles)

### 2.1 TypeScript Effect (`Effect.Effect<A, E, R>`)

Effect-TS tracks **requirements** in the third type parameter `R` of `Effect<A, E, R>`. Services are identified by **`Context.Tag`** values; programs **yield** tags inside `Effect.gen`, and callers **provide** implementations via `Effect.provide`, `Effect.provideService`, or **`Layer`** composition.

```typescript
class Logger extends Context.Tag("Logger")<
  Logger,
  { readonly log: (s: string) => Effect.Effect<void> }
>() {}

const program = Effect.gen(function* () {
  const logger = yield* Logger
  yield* logger.log("hello")
})
// Effect<void, never, Logger>

const runnable = program.pipe(
  Effect.provideService(Logger, { log: (s) => Effect.sync(() => console.log(s)) })
)
```

**Mechanism:** Reader monad + typed environment map; `R` shrinks as services are provided (`Exclude<R, Tag>`). **Layers** encode dependency graphs and startup effects.

**Strengths:** Requirements are **compile-time tracked**; test doubles are `provideService` at the edge; aligns with Forst's existing Effect RFC thread and TS sidecar clients.

**Weaknesses:** Heavy type-level machinery; `Effect.gen` / `yield*` is unfamiliar to many TS authors; `R` union semantics (`R1 | R2` means **both** required) is easy to misread; full Layer graphs add ceremony.

**References:** [Effect services docs](https://effect.website/docs/requirements-management/services/), [Effect.ts source](https://github.com/Effect-TS/effect/blob/main/packages/effect/src/Effect.ts).

---

### 2.2 ZIO (Scala)

ZIO predates Effect-TS and uses **`ZIO[R, E, A]`** (environment **first**, success **last**). **`ZLayer[RIn, E, ROut]`** builds services; **`provide` / `provideSome`** satisfies `R`.

```scala
val effect: ZIO[Database & Logger, Throwable, User] = for {
  db  <- ZIO.service[Database]
  log <- ZIO.service[Logger]
  _   <- log.info("fetching")
  u   <- db.findUser(123)
} yield u

val app = effect.provide(databaseLayer, loggerLayer)
```

**Mechanism:** Same conceptual model as Effect-TS (environment monad + layers), with mature macro-assisted wiring and `ZIOAppDefault` entry points.

**Strengths:** Battle-tested DI story; excellent compile-time missing-dependency errors; interpreter swapping for tests is idiomatic.

**Weaknesses:** Scala-only; layer algebra (`++`, `>>>`) has a learning curve; transpilation to Go would discard most of the abstraction unless reimplemented.

**References:** [ZIO contextual types](https://zio.dev/reference/contextual/), [ZIO DI](https://zio.dev/reference/di/), [ZLayer](https://zio.dev/reference/datasources/zlayer/).

---

### 2.3 Roc — platforms, tasks, and “abilities” (disambiguation)

Roc uses **two unrelated words** that often collide with “requirements” research:

#### A. **Platforms + `Task` (managed effects / I/O requirements)**

Roc's stdlib has **no I/O**. Every app selects **exactly one platform** (`pf`) that supplies primitives (`Stdout`, HTTP, etc.). Effectful functions return **`Task a`** descriptions; **`main!`** runs them via the platform host.

```roc
main! = Task.ok {}
# pf.Stdout.line!("Hello")?  # I/O from platform, not stdlib
```

**Mechanism:** Capabilities come from the **platform boundary**, not per-function type parameters. Testing swaps **platform implementations** (or test platforms).

**Strengths:** Extremely simple mental model for apps; pure functions by default; host can mock I/O wholesale.

**Weaknesses:** **Coarse-grained** — hard to express “this pure helper needs only `Clock` but not `Database`” inside the language today; not per-function requirement tracking.

#### B. **Abilities (typeclass constraints, not effects)**

Roc **abilities** (`Eq`, `Hash`, `Inspect`, …) constrain **types**, not runtime services:

```roc
StatsDB := Dict Str { score : Dec } implements [ Eq, Hash ]
```

These are **compile-time typeclass dictionaries**, closer to Rust/Haskell type classes than Effect `R`.

**References:** [Roc abilities](https://www.roc-lang.org/abilities), [Roc platforms](https://roc-lang.org/platforms), [Roc functional overview](https://www.roc-lang.org/functional).

---

### 2.4 Unison abilities (algebraic effects)

Unison **abilities** are **algebraic effects**: functions declare `{IO, Exception, MyKV}` in the type; **`handle`** blocks supply handlers that interpret effect requests.

```unison
findUser : Text ->{IO, Exception} User
findUser id =
  match Http.get ("https://api/users/" ++ id) with
    Left err  -> Exception.raise err
    Right raw -> decodeUser raw

test = handle findUser "42" with
  cases { IO.send -> _ -> "mock-json" }
```

**Mechanism:** Effect handlers (Frank lineage); ability polymorphism; direct-style code with typed scope capabilities.

**Strengths:** **Excellent test story** — swap handlers for pure mocks; requirements visible in signature `{…}`; no monadic `flatMap` noise.

**Weaknesses:** Unison-specific runtime (UCM); `{Ability}` syntax and handler blocks are unfamiliar; **transpiling handlers to Go** requires a codegen strategy (interface + explicit handler parameter, or effect monad).

**References:** [Unison abilities docs](https://www.unison-lang.org/docs/fundamentals/abilities/), [pragdave on abilities](https://pragdave.me/discover/unison/2023-03-11-abilities/), [LWN Unison overview](https://lwn.net/Articles/978955/).

---

### 2.5 Haskell — MTL, type classes, and mock libraries

**MTL** encodes effects as **type class constraints** on the monad:

```haskell
sendAndFetch :: (MonadDB m, MonadHTTP m, DBRecord a)
             => HTTPRequest -> m (Either AppError a)
```

Production uses `IO` or a concrete stack; tests use **`ReaderT` over a record of stubs**, **`MockT`**, **`test-fixture`**, or **`HMock`** Template Haskell.

**Mechanism:** Dictionary passing (type classes); constraint polymorphism; interpreter at `main`.

**Strengths:** Mature mocking ecosystem; constraints list **exact** capabilities; GHC catches missing instances.

**Weaknesses:** **Constraint noise** in signatures; `Monad*` naming proliferation; instance orphan/overlap issues; **not Go-transpilable** without generating Go interfaces + struct fields per constraint.

**References:** [test-fixture](https://hackage.haskell.org/package/test-fixture), [monad-mock](https://hackage.haskell.org/package/monad-mock), [HMock](https://hackage.haskell.org/package/HMock), [mock effects with data kinds](https://chrispenner.ca/posts/mock-effects-with-data-kinds).

---

### 2.6 Polysemy / extensible effects (Haskell)

**Polysemy** uses **`Member (Effect r)`** rows and **`Sem r a`** instead of MTL classes. Interpreters are **pure functions** composed at the edge:

```haskell
reservationServer
  :: Member (KVS Reservation) r
  => Sem r ()
```

Tests **`interpret`** `KVS` into `State` maps; production into SQLite.

**Mechanism:** Freer/extensible effects; open effect rows; no functional dependencies (unlike MTL).

**Strengths:** Clean architecture friendly; **swap interpreters** without changing business logic; less instance boilerplate than MTL.

**Weaknesses:** Requires many GHC extensions; effect row inference needs **`polysemy-plugin`**; still far from Go surface syntax.

**References:** [polysemy](https://github.com/polysemy-research/polysemy), [Clean Architecture with Polysemy](https://thma.github.io/posts/2020-05-29-polysemy-clean-architecture.html).

---

### 2.7 Rust — traits, generics, and `async-trait`

Rust DI is **explicit interfaces**:

```rust
#[async_trait]
trait UserRepository: Send + Sync {
    async fn find_by_id(&self, id: u64) -> Result<Option<User>, DbError>;
}

struct UserService<R: UserRepository> { repo: R }

// Production: PostgresUserRepository
// Tests: MockUserRepository or mockall-generated MockUserRepository
```

**Mechanism:** Static dispatch (generics) or dynamic (`Arc<dyn Trait>`); **`#[async_trait]`** for async trait objects; **`mockall`** for expectations.

**Strengths:** **Maps 1:1 to Go** (interface + struct field or constructor injection); extremely traceable; LLMs handle trait mocks well.

**Weaknesses:** **No unified requirement accumulator** — each dependency is a separate generic param or struct field; `async-trait`/`dyn` overhead and object-safety limits; composition = manual wiring.

**References:** [async-trait](https://docs.rs/async-trait/latest/async_trait/), [Rust service layer testing patterns](https://leapcell.io/blog/building-flexible-and-testable-service-layers-with-rust-traits).

---

### 2.8 Kotlin — context parameters / receivers (Arrow)

**Context parameters** (stable Kotlin 2.4+) declare **implicit compile-time dependencies**:

```kotlin
context(logger: Logger, db: Database)
fun saveUser(user: User) {
    logger.log("Saving")
    db.save(user)
}

context(testLogger, testDb) { saveUser(user) }
```

**Arrow** historically proposed **context receivers** for effect interfaces (`Database`, `Logger`) as an alternative to `flatMap` stacks.

**Mechanism:** Compiler desugars to **extra parameters** (zero runtime cost); coeffect / execution-context model.

**Strengths:** **English-readable** at call site (`context(...)` block); separates **constructor-injected stable deps** from **call-site execution context**; good for coroutine-scoped wiring.

**Weaknesses:** Not a full effect system alone; easy to misuse for **data** that should be ordinary parameters; JVM-specific (though KMP exists); Forst would need a **source-level** equivalent — Go has no context parameters.

**References:** [Kotlin KEEP context receivers](https://github.com/Kotlin/KEEP/blob/context-parameters/proposals/context-receivers.md), [Arrow effects and contexts](https://arrow-kt.io/learn/design/effects-contexts/), [context parameters ≠ DI (DEV)](https://dev.to/camilyed/kotlin-context-parameters-not-di-but-execution-context-3g0i).

---

### 2.9 Koka — row-polymorphic effects + handlers

Koka annotates **every function** with an effect row:

```koka
fun divide(x : int, y : int) : div <exn> int
  if y == 0 then throw("divide by zero") else x / y
```

User-defined effects use **`effect`** declarations and **`handle`** with **`with`**:

```koka
effect yield<a> { fun yield(i : a) : () }

fun traverse(xs : list<int>) : yield<int> ()
  match xs
    Cons(x, xs) -> { yield(x); traverse(xs) }
    Nil         -> ()
```

**Mechanism:** Row polymorphism (`<exn, div | r>`); scoped labels; effect handlers compiled via type-directed translation (often to CPS/JS).

**Strengths:** **Fine-grained effect tracking** in signatures; handlers compose; direct-style `with` blocks.

**Weaknesses:** Effect rows are **esoteric** for TS/Go audiences; inference relies on row variables; Go lowering would need explicit effect parameters or a monadic translation.

**References:** [Koka documentation](https://koka-lang.github.io/koka/doc/index.html), [Koka row-polymorphic effects (MSR)](https://arxiv.org/pdf/1406.2061), [algebraic effects compilation (POPL'17)](https://www.microsoft.com/en-us/research/wp-content/uploads/2016/12/algeff.pdf).

---

### 2.10 Frank — scope abilities and multihandlers

Frank treats **functions as operators** that may **handle** commands. Effect polymorphism propagates an **scope ability inward** (dual to Koka's outward rows). **Multihandlers** interpret multiple command sources at once.

**Mechanism:** Bidirectional effect typing; adapters for effect encapsulation (“effect pollution”).

**Strengths:** Theoretical clarity; direct-style; influenced Unison.

**Weaknesses:** Niche language; scope ability rules are hard to teach; no production Go/TS toolchain.

**References:** [Do be do be do (POPL 2017)](https://homepages.inf.ed.ac.uk/slindley/papers/frankly.pdf), [Frank JFP paper](https://homepages.inf.ed.ac.uk/slindley/papers/frankly-jfp.pdf).

---

### 2.11 Go ecosystem baseline (implicit prior art)

Go's idiomatic pattern — already used in Forst RFC sketches — is **explicit parameters**:

```go
func getUserById(ctx context.Context, id int, repo UserRepository) (*User, error)
```

Or **struct + interface fields** (service struct with `repo UserRepository`). Tests use **fake implementations** of interfaces; **`context.Context`** carries cancellation/deadlines, not business DI.

**Mechanism:** Nominal interfaces; constructor/wiring in `main`; no language-level requirement aggregation.

**Strengths:** Trivial transpilation (Forst's primary backend); LLMs generate fakes easily; stack traces and grep-friendly.

**Weaknesses:** Requirement lists **don't compose** in types; large parameter lists or “god structs”; easy to hide dependencies in globals.

**References:** [effect RFC simple CLI sketch](../effect/13-simple-effect-cli.md) (`context.Context` + explicit repos).

---

## 3. Comparison matrix

| System | Syntax clarity | Explicitness / traceability | Inference vs explicit | Test / mock ergonomics | Boilerplate | Go transpilation | Data vs logic separation | LLM test-gen friendliness | Gap vs Forst goals |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| **Effect-TS `R`** | ★★ (`yield*` noise) | ★★★ types + tags | ★★ inferred `R`, explicit `provide` | ★★★ `provideService` | ★★ Layers for graphs | ★★ monad/runtime | ★★★ tags ≠ args | ★★ needs Effect literacy | TS-first; heavy runtime; Go RFCs focus on `Result` not `R` |
| **ZIO** | ★★ | ★★★ | ★★ | ★★★ | ★★ | ★ JVM-only origin | ★★★ | ★★ | Same as Effect; not TS-native |
| **Roc platform / Task** | ★★★ | ★★ (platform boundary) | ★★★ inferred tasks | ★★ platform swap | ★★★ app level | ★★ host-specific | ★★ coarse | ★★ | Too coarse for per-function reqs; abilities ≠ services |
| **Roc abilities** | ★★★ | ★★★ | ★★★ derived | ★ (not DI) | ★★★ | ★★★ dict passing | — type-level | ★★ | Wrong abstraction (Eq/Hash, not Logger) |
| **Unison abilities** | ★★ `{IO,…}` | ★★★ | ★★ scope | ★★★ handlers | ★★★ | ★ handlers hard in Go | ★★★ | ★★ handler syntax | Runtime/handler model ≠ Go sidecar |
| **Haskell MTL** | ★ `(MonadX m)` | ★★★ | ★★ constraints | ★★★ fixtures/mocks | ★★ | ★ | ★★ | ★★ | Not TS-familiar; constraint soup |
| **Polysemy** | ★ | ★★★ | ★★ rows | ★★★ interpreters | ★★ | ★ | ★★★ | ★ | Academic Haskell tooling |
| **Rust traits** | ★★★ | ★★★ | ★ explicit | ★★★ mockall | ★★ wiring | ★★★ interfaces | ★★★ | ★★★ | No aggregated `R`; manual composition |
| **Kotlin context** | ★★★ | ★★★ signatures | ★★ explicit `context` | ★★★ | ★★★ | ★★ desugar to params | ★★★ | ★★★ | Go has no native equivalent |
| **Koka effects** | ★ rows | ★★★ | ★★★ inference | ★★★ handlers | ★★ | ★ | ★★★ | ★ | Row syntax alien to TS/Go users |
| **Frank** | ★ | ★★ scope | ★★★ | ★★ | ★★★ | ★ | ★★ | ★ | Research language |
| **Go interfaces (baseline)** | ★★★ | ★★★ | ★ manual | ★★★ | ★★★ simple | ★★★ native | ★★ (param creep) | ★★★ | No type-level requirement sum |

---

## 4. Cross-cutting themes

### 4.1 Three families

1. **Environment monad (`R`)** — Effect-TS, ZIO: accumulate requirements in a type parameter; provide at the edge.
2. **Algebraic effects / handlers** — Unison, Koka, Frank, Polysemy: operations as typed commands; interpreters/handlers supply meaning.
3. **Explicit interfaces / context** — Rust, Go, Kotlin context parameters: dependencies are interfaces or implicit parameters, wired manually or via `context { }`.

Forst's **Go transpilation** constraint strongly favors **(3)** or a **thin sugar layer** that lowers to **(3)**, optionally with **Effect-like tracking in Forst types only** (erased or struct-bundled at Go emit time).

### 4.2 Test instantiation patterns (what works for agents)

| Pattern | Example | Agent-friendly because… |
| --- | --- | --- |
| **Interface + fake struct** | Rust `MockUserRepository` | Signature lists fields/methods to implement |
| **`provideService(Tag, impl)`** | Effect-TS | Tag name + interface type hint mock shape |
| **Handler block** | Unison `handle … with` | Mock responses colocated with test |
| **MTL `Fixture` record** | Haskell test-fixture | Field-per-method stub map |
| **Platform swap** | Roc test platform | Only works for coarse I/O |
| **Extra function parameters** | Go `repo UserRepository` | Obvious, grep-able, no magic |

**LLM-friendly designs** share: **named interfaces**, **methods listed in types**, **wiring localized** in test `main` or `context`/`provide` block — not implicit globals.

### 4.3 Data vs runtime-logic separation

Best-in-class systems keep **operation inputs** as ordinary parameters and **capabilities** as requirements:

- Kotlin: **constructor** = stable deps; **context** = execution environment; **parameters** = command data ([DEV article](https://dev.to/camilyed/kotlin-context-parameters-not-di-but-execution-context-3g0i)).
- Effect: **function args** = data; **`R`** = services.
- Go baseline often **blurs** this by passing `context.Context` alongside business args — Forst should avoid overloading `ensure` or `Result` to mean “DI”.

---

## 5. Forst codebase audit (current state)

### 5.1 `ensure` — validation and control flow, not requirements

Today **`ensure`** is a **guard / narrowing / failure-propagation** construct, not a capability system:

```ft
ensure name is Min(1)
    or TooShort("Name must be at least 1 character long")

ensure result is Ok() {
    fmt.Printf("Conditions not met: %s", result.Error())
}
```

Compiler support: [`EnsureNode`](../../../../forst/internal/ast/ensure.go), parser in [`ensure.go`](../../../../forst/internal/parser/ensure.go), narrowing in [`narrowing_test.go`](../../../../forst/internal/typechecker/narrowing_test.go). RFCs: [guard](../guard/guard.md), [ensure-only failure](../errors/01-ensure-only-failure-returns.md), [09 — narrowing](../optionals/09-ensure-is-narrowing-and-binary-types.md).

**Implication:** Reusing the **`ensure` keyword** for “requires Logger” would **collide** with established guard semantics unless carefully overloaded (not recommended).

### 5.2 No first-class requirement / effect / DI types

Search of `forst/` and `examples/in/rfc/` shows:

- **Effect RFCs** ([`examples/in/rfc/effect/`](../effect/README.md)) focus on **`Effect[A, E]` error channels**, resource management, and TS interop — **not** an `R`-style requirement parameter in Forst source.
- [`15-forst-redesign-for-effects.md`](../effect/15-forst-redesign-for-effects.md) discusses verbosity of explicit param types and `ensure` for errors, but not service requirements.
- [`13-simple-effect-cli.md`](../effect/13-simple-effect-cli.md) sketches **Go `context.Context` + repository interfaces** — closest existing Forst pattern for DI.

There is **no** `requires`, `context`, `ability`, or `provide` syntax in the Forst grammar today.

### 5.3 Function parameters today

Functions use **explicit typed parameters** only:

- AST: [`SimpleParamNode`](../../../../forst/internal/ast/param.go), [`DestructuredParamNode`](../../../../forst/internal/ast/param.go) (destructuring **TODO** in typechecker/register).
- Parser: [`parseFunctionSignature`](../../../../forst/internal/parser/function.go) — comma-separated params, optional single return type (`Result` / named types).
- Typechecker: [`registerFunction`](../../../../forst/internal/typechecker/register.go) stores `ParameterSignature` per ident.
- **No** implicit parameters, type-class constraints, or effect rows.

### 5.4 Go transformer patterns

[`transformFunctionParams`](../../../../forst/internal/transformer/go/function.go) maps each Forst param 1:1 to a Go field:

- Assertion types → inferred via `InferAssertionType` or aliased user type names.
- Emits `go/ast` field list entries — **no** automatic interface bundling or context injection unless added by convention.

Other transformer conventions relevant to future requirements:

- **`Result`** lowers to struct `{ V T; Err error }` or tuple patterns (see transformer tests).
- Shapes → Go structs; nominal errors → Go types / `error` interface.
- [`typedef_expr.go`](../../../../forst/internal/transformer/go/typedef_expr.go) can emit `interface{}` for open shapes — pattern for **capability interfaces** could mirror nominal error emission.

**Implication:** A Forst requirements system that **lowers to “extra interface-typed parameters + wiring struct”** fits the existing pipeline better than a runtime effect interpreter.

### 5.5 Sidecar / discovery

[`discovery.FunctionInfo`](../../../../forst/internal/discovery/discovery.go) exports parameter names/types for HTTP sidecar routing. Any requirement system should **serialize capability needs** into discovery metadata if sidecar handlers must inject deps.

---

## 6. Implications for a Forst “function requirements” design

### 6.1 Non-goals (from prior art failures)

- **Do not** copy Roc **abilities** naming — confuses typeclass constraints with services.
- **Do not** overload **`ensure`** for DI — it already means guard/narrow/`Result` propagation.
- **Avoid** full Koka/Frank row polymorphism in user-facing syntax — poor TS/Go familiarity and LLM confusion.
- **Avoid** mandatory Effect **`Layer`** ceremony in Forst source if Go emit target is plain structs.

### 6.2 Promising hybrids for Forst

| Direction | Surface sketch | Lowers to Go | TS / sidecar |
| --- | --- | --- | --- |
| **A. Explicit capability params (Rust-like)** | `func f(repo UserRepo, id Int)` | `func f(repo UserRepo, id int)` | Same interfaces in generated `.d.ts` |
| **B. `requires` clause + struct bundling** | `func f(id Int) requires (Logger, Db)` | `func f(deps Deps, id int)` or appended params | Generate `Deps` interface bundle |
| **C. Context-parameters sugar** | `context(log Logger) func f(...)` | Extra params + call-site wrapper | TS: optional `@forst/context` metadata |
| **D. Effect-style `R` (types only)** | `func f(id Int) Result(User, Err) needs Logger` | Erase to params; track in Forst types for tests | Align with Effect client `provide` story |

**Recommendation (research, not decision):** Start from **A + B** — **English-named interfaces** as capability types, optional **`requires`** clause for ergonomics, **explicit wiring** in `main` / test helpers. Add **D** only if TS Effect interop requires tracked `R` in generated types.

### 6.3 Alignment with existing Forst pillars

- **`Result` + `ensure`:** Requirements supply **services**; **`ensure`** continues to handle **validation and failure propagation** on **data** ([errors normative](../errors/02-first-class-errors-normative.md)).
- **Shape guards:** Capability interfaces can be **structural** (`type Logger = { log: fn(String) }`) matching Forst's structural typing.
- **Go interop:** Prefer **interface + constructor injection** over effect handlers — matches [`13-simple-effect-cli.md`](../effect/13-simple-effect-cli.md).
- **LLM tests:** Require **named capability types** with **documented method shapes** in discovery JSON and hover docs ([`hoverdoc/builtins.go`](../../../../forst/internal/hoverdoc/builtins.go) pattern).

---

## 7. Open research questions

1. Should Forst track requirements **per function** or **per module/package** (Roc platform style)?
2. Can **`requires`** be inferred from **free identifiers** (`log.info(...)`) without full effect inference?
3. How do requirements interact with **`Result`**-returning functions and sidecar **JSON** APIs (capabilities provided by host vs request body)?
4. Should generated TypeScript expose **`R`** for Effect users, or only Go-style interfaces?
5. What is the **test harness** convention — `provide` helper, struct literal, or codegen mocks?

---

## 8. References (external)

| Topic | Link |
| --- | --- |
| Effect-TS services / `provide` | https://effect.website/docs/requirements-management/services/ |
| Effect `Effect<A,E,R>` | https://github.com/Effect-TS/effect/blob/main/packages/effect/src/Effect.ts |
| ZIO environment & DI | https://zio.dev/reference/contextual/ , https://zio.dev/reference/di/ |
| Roc abilities vs platforms | https://www.roc-lang.org/abilities , https://roc-lang.org/platforms |
| Unison abilities | https://www.unison-lang.org/docs/fundamentals/abilities/ |
| Koka effects | https://koka-lang.github.io/koka/doc/index.html |
| Frank (POPL'17) | https://homepages.inf.ed.ac.uk/slindley/papers/frankly.pdf |
| Kotlin context parameters KEEP | https://github.com/Kotlin/KEEP/blob/context-parameters/proposals/context-receivers.md |
| Arrow effects & contexts | https://arrow-kt.io/learn/design/effects-contexts/ |
| Rust async trait / testing | https://docs.rs/async-trait/ , https://leapcell.io/blog/building-flexible-and-testable-service-layers-with-rust-traits |
| Haskell test-fixture / HMock | https://hackage.haskell.org/package/test-fixture , https://hackage.haskell.org/package/HMock |
| Polysemy | https://github.com/polysemy-research/polysemy |
| Effect systems survey (dissertation) | https://www.dantb.dev/files/dissertation.pdf |

---

## Document status

**Research reference.** Update when Forst **`requires`** / capabilities syntax is proposed in a follow-on RFC (`01-requirements-syntax.md`, etc.).
