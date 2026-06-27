# Function requirements (05) — language and product implications

**Audience:** Language designers, product leads, compiler implementers, teams evaluating Forst for backend migration.

**Status:** Strategic analysis — evaluates what adopting [SPEC — Function requirements](./SPEC.md) *commits* Forst to, independent of implementation phasing ([06 — Feasibility](./06-feasibility-analysis.md)).

---

## 1. Executive summary

The minimal `use` / `with` design is strategically valuable **if and only if** transitive inference and wiring-root completeness ship as a non-optional milestone: without them, the surface collapses into syntactic sugar over hand-written Go needs structs—the same break-even [06](./06-feasibility-analysis.md) documents for MVP-only delivery. When inference lands, Forst gains a **differentiated compile-time capability graph** that aligns with Go constructor injection, Effect’s *provide* mental model without an Effect runtime, and the sidecar’s **data-only wire boundary**—while staying hostile to reflection DI and global service locators. The bet is narrow but defensible: **testability and explicit service boundaries as a language primitive**, not a framework. Success makes Forst backends easier to test, migrate from TypeScript incrementally, and reason about for humans and agents; failure mode is “Go with two extra keywords and no checker payoff,” which accelerates neither TS replacement nor LLM-assisted development.

---

## 2. What this design commits Forst to

### Compile-time capability graph vs runtime DI

Forst commits to **static need sets** (`Needs(f)`) computed from bodies and call graphs, satisfied by **`with` at wiring roots** (`main`, handlers, test helpers). There is no runtime service locator, no container, no reflection-based injection. Production wiring and test wiring use the **same mechanism**—only the concrete types in `AppContext` or inner `with { … }` maps differ.

Implication: every serious backend eventually owns an **`AppContext`-shaped composition root** (or equivalent explicit maps at call sites). Teams cannot “opt out” into framework magic without leaving idiomatic Forst.

### Go as ground truth for lowering

Every requirement feature must answer: *what would careful Go emit?* Contracts lower to **Go interfaces** (or imported interface aliases); `use` lowers to reads from a synthesized **`fnNeeds` struct** prepended to the function signature; `with` lowers to **struct literals and field copies**. Wrapper types (`ClientDoer` over `*http.Client`, `PostgresStore` over `*store.Postgres`) are first-class—not an escape hatch.

Implication: Forst’s requirements story **inherits Go’s tradeoffs**: interface dispatch, explicit wiring at boundaries, no lifecycle graph in the language. Effect-style `Layer` composition is intentionally **not** replicated; graph reasoning stays in author code (`main`, `ciAppContext()`).

### Inference as language feature (not optional sugar)

The design’s contract with authors is: **add `use metrics: Metrics` in a leaf function; the checker errors at the handler or `main` until wiring supplies `Metrics`**. No parallel `uses Logger, Clock` export clause—discovery JSON and LSP **`needs`** metadata are **derived**, not duplicated in source.

Implication: marketing “compile-time completeness” before v1 inference is **misleading**. Partial MVP (direct `use` sites only, manual call-site `with`) is a compiler experiment, not a product promise.

### Two-keyword surface (`use`, `with`)

| Keyword | Commitment |
| --- | --- |
| **`use`** | Body-local binding to a contract type; aggregates into `Needs(f)`. |
| **`with`** | Scope or call-site supply; merges/shadows scope bindings; supports `with ctx`, explicit maps, postfix `… with { … }`, and `with forward`. |

No third keyword for contracts, harnesses, or overrides. **Policy lives in plain functions** (`ciAppContext()`, `BlockedHttp`, `RecordingEmail`) rather than compiler-enforced test presets.

### Types-as-contracts (no parallel capability system)

Contracts are **ordinary Forst types**: shapes with methods, nominal typedefs, or **imported Go types** (`io.Writer`, `store.UserRepository`, `*sql.DB` when authors accept concrete coupling). Structural satisfaction for implementations; **distinct nominal names = distinct requirement keys** (`AuditLogger` ≠ `AppLogger` even with overlapping methods).

Implication: capability design **is** API design. There is no separate “requirements DSL” to learn—or for the compiler to maintain in parallel with the type system.

---

## 3. Future constraints

How 05 shapes—or limits—evolution of the rest of Forst.

### Effect / error channel integration

The [Effect RFC thread](../effect/00-overview.md) targets **`Effect<A, E>` error channels**, resource cleanup, and TS interop—not an **`R`** requirement parameter in Forst source. Requirements supply **host services**; **`Result` + `ensure`** continue to govern **data validation and failure propagation** on inputs ([00 — Prior art §4.3](./00-prior-art.md#43-data-vs-runtime-logic-separation)).

Constraint: **do not overload `ensure` or `Result` for DI.** Any future native Effect type must compose orthogonally: services via `use`/`with`, errors via `Result`/nominal `error` types. **TypeScript never sees requirements** — [ADR-011](./ADR.md#adr-011-typescript-never-sees-requirements).

### Generics, async, concurrency

- **User generics** ([ROADMAP — Generics](../../ROADMAP.md#generics-aliases-and-nominal-features)): requirement keys stay **nominal typedef idents**; generic contracts (`Repo[T]`) need explicit policy for whether `Needs(f)` keys the instantiated type or the definition—unresolved in 05, likely deferred.
- **Go wrap of generic APIs**: [06](./06-feasibility-analysis.md) flags uneven generic/variadic interop; wrapper adapters may remain hand-written Go at the edge.
- **`go` / goroutines**: requirements are **not goroutine-local**; scope `with` scopes follow lexical structure, not `context.Context` cancellation trees. Async code that spawns work must pass needs explicitly or re-enter a `with` scope—same as explicit Go.

### Package boundaries and cross-package inference

Today’s checker is **per merged package** with no exported-needs summary across packages ([06 §5.1](./06-feasibility-analysis.md#1-inference-fixed-point-across-packages--critical)). v1 fixed-point is single-package; v2 adds discovery **`needs`** for cross-package wiring.

Constraint: until v2, **incomplete wiring can slip at package edges**—`main` in package `cmd` may not see transitive needs from `internal/users` without discipline (re-export helpers, shared `testfixtures` package, or explicit maps at every cross-package call). This pushes toward **fat `main` / fat test fixture modules**, not invisible cross-module magic.

### Nominal vs structural typing trajectory

05 requires **structural satisfaction for implementations** but **nominal identity for requirement keys** (rule N6). That tension interacts with today’s compiler: structural shape comparison for guards, nominal-error leaks in edge cases ([06 §5](./06-feasibility-analysis.md#5-hard-problems-ranked-by-severity)). Adopting 05 **forces a clearer nominal layer for service contracts** while keeping data shapes structural—a permanent fork in the type system’s “when does name matter?” story.

### Sidecar / TS client generation

[Sidecar decisions](../sidecar/10-decisions.md): **wire format stays data-only**; the **host builds `AppContext` server-side**. `use`/`with` are **Go-side only**—not serialized over HTTP invoke.

Constraint: TypeScript **`forst generate`** emits **only functions with `Needs(f) = ∅`** — wire parameters and return types for JSON over HTTP. Capabilities are **host concerns** wired in Go/Forst before export. TypeScript has **no** `needs`, `@needs`, or Effect `R` channel ([ADR-011](./ADR.md#adr-011-typescript-never-sees-requirements)).

### Module system, visibility

Requirements do not introduce a **`test`-only module kind**. `ciAppContext()` is a **plain exported function**—typically in `testfixtures` or test sections compiled with the package. Visibility rules follow normal Forst/Go export semantics; “test-only wiring” is convention + linter grep, not compiler enforcement.

### Cannot easily add runtime service locators or global registries

The design is **deliberately incompatible** with:

- global `GetDB()` singletons as first-class patterns;
- runtime plugin registries without explicit `with` at the root;
- TS-side `provideService` at invoke time (sidecar contract forbids it).

Teams wanting those patterns will fight the checker or bypass it in hand-written Go—an intentional friction.

### Dependent types, refinement types ([ROADMAP anti-features](../../ROADMAP.md#anti-features))

ROADMAP lists **dependent types and arbitrary type-level computation** as **not planned**. Requirements inference is a **fixed-point over call graphs and scope scopes**, not refinement of values (`Logger` indexed by log level, etc.). Shape guards and `ensure` refine **data**; `use`/`with` refine **who supplies services**—orthogonal planes that should not merge without breaking decidability and tooling speed goals.

---

## 4. How it helps backend developers

### Test isolation without mock frameworks

Leaf tests wire fakes with postfix **`with { Logger: NopLogger {}, Clock: FakeClock { … } }`**—structs that **structurally satisfy** contracts, including imported Go interfaces via `bytes.Buffer` or thin adapters. No testify/mockery code generation ceremony.

### Handler / context patterns

Handlers take **`(input, ctx AppContext)`** and **`with ctx { … }`** to satisfy transitive needs—mirroring how teams already bundle deps, but with **compile-time propagation** when inference ships. Field forwarding (`logger` → `Logger`) matches senior Go “context struct” habits without overloading `context.Context`.

### CI-safe integration tests

**`ciAppContext()`** centralizes safe defaults: **`InMemoryUserRepo`**, **`NoopEmail`**, **`BlockedHttp`** that returns **`Err(NetworkDisabled { … })`** on any accidental network call. Policy is **visible in code review**, not a magic `CI` harness keyword. Selective overrides nest **`with { EmailSender: recording }`** inside **`with ctx`**.

### Onboarding: mental model vs Go / Java DI

| Familiar from | Maps to |
| --- | --- |
| Go manual constructor injection | `fnNeeds` struct + literals at call sites |
| Java/Spring `@Autowired` | **Not replicated**—explicit `with` instead |
| Effect `provide` / `Layer` | **`with` at boundary**; no layer graph |
| Rust `impl Trait` params | Similar explicitness; inference reduces signature noise |

Backend developers who already write **`func f(deps Deps, arg T)`** in Go will recognize emitted code immediately; the win is **checker-backed completeness**, not a new runtime model.

### When it does NOT help

- **Data mocking**: seeded DB rows, fixture JSON, golden files—still ordinary test data; `InMemoryUserRepo { users: map[…] }` is manual.
- **Simple CRUD**: one repo, no cross-cutting deps—`use`/`with` adds ceremony vs a single extra parameter.
- **Observability, auth middleware, validation**: orthogonal (`ensure`, guards, HTTP middleware in Go)—requirements do not replace them.
- **Network-level isolation without discipline**: `ciAppContext()` only helps if tests **actually call it**; nothing stops a test from constructing `RealHttpClient` unless lint/convention catches it.

---

## 5. How it helps replace TypeScript

Forst’s stated goal ([ROADMAP](../../ROADMAP.md)): **TypeScript-grade ergonomics**, **Go performance**, **incremental TS client interop**.

### Parity with Effect.provide / Layer mental model—without Effect runtime

Teams migrating from **Effect-TS** backends get a familiar split: **data in parameters**, **services wired at the edge**. `with ctx { handleCreateUser(input) }` parallels **`Effect.provide`** / running against a composed environment— but lowers to **plain Go**, not a reader monad interpreter. What they **lose**: typed **`R`** in every signature, **`Layer`** lifecycle, **`Effect.gen`**. What they **gain**: zero runtime Effect dependency, native Go deploy artifacts, debugger-friendly stacks.

### Generated TS types

**Rule ([ADR-011](./ADR.md#adr-011-typescript-never-sees-requirements)):** Only **runnable** exports — functions with no outstanding needs — appear in `@forst/client` / `forst generate`. TypeScript describes **wire data only**; it does not model capabilities or host wiring.

Strategic line: **TS types describe the wire; Go/Forst describes the host.** Migration teams keep Express/TS routes that call **`forst.user.CreateUser(body)`** while **`main`/`forst dev` startup** builds production `AppContext` and exports fully-wired handlers — matching [@forst/sidecar incremental adoption](../../../packages/sidecar/README.md).

### Sidecar boundary: Forst ↔ TS vs Go-internal

| Crosses boundary | Stays Go-internal |
| --- | --- |
| JSON request/response **data** shapes | `Logger`, `UserRepo`, `HttpDoer` wiring |
| Generated `.d.ts` for invoke args/results | `with ctx`, `ciAppContext()`, `BlockedHttp` |
| Discovery: function name, param types | Inferred **`needs`** for host implementers (tooling) |

This **accelerates strangler-fig migration**: move logic to `.ft`, keep TS shell, never expose DI through HTTP.

### Developer velocity vs TS + Jest + mocks

- **Less**: mock module hoisting, `jest.mock` fragility, duplicate type definitions for test doubles.
- **More**: upfront contract typedefs, **`ciAppContext()` maintenance**, wrapper structs for foreign Go (until receiver methods + wrap patterns are smooth).
- **Net win** when services are **many and transitive**; **net loss** for trivial handlers until inference saves manual audit.

### Migration path from TS backends

1. Extract pure business functions to Forst with **`use`** for existing Go interfaces.
2. Wire **`AppContext`** in Go `main` / sidecar host process.
3. TS calls **`forst generate`** + sidecar invoke for routes still in Node.
4. Replace TS route bodies as `.ft` coverage grows; **`with` never moves to TS**.

---

## 6. How it helps LLM agents

### Minimal keyword surface (`use`, `with`)

Two verbs with stable semantics beat frameworks with ten concepts (`provide`, `Layer`, `Tag`, `Context`, `override`, `harness`). Prompts can anchor: **“bind with `use`, wire with `with`.”**

### Inference reduces signature hallucination

Agents need not invent **`uses Logger, Db`** clauses that drift from bodies. Target workflow ([SPEC § LLM ergonomics](./SPEC.md#llm-ergonomics)):

1. Read compiler **`needs: ["Logger", "UserRepo"]`** from discovery/LSP.
2. Generate fakes matching those typedefs.
3. Wire **`with { … }`** or call **`ciAppContext()`**.

Without inference JSON, agents revert to **guessing** from `use` lines only—incomplete for transitive deps.

### Grep-able fixture functions vs framework magic

**`ciAppContext()`**, **`BlockedHttp`**, **`NopLogger`** are **valid Forst**, searchable symbols—not macro-expanded harness names. Agents can copy a team’s fixture file and delta inner **`with { EmailSender: recording }`**.

### Test generation workflow

```
read function → infer needs (tooling) → implement struct fakes → postfix with or with ctx block → ensure assertions
```

Matches how agents already write Go table tests—**lowered Go resembles hand-written injection**, so validation against `_test.go` patterns works.

### Where agents still fail

- **Partial maps** at call sites when scope scope is implicit—must read enclosing **`with ctx`**.
- **`with forward`** merge semantics—easy to omit callee gaps.
- **Wrap syntax** for foreign Go (`ClientDoer`, `PostgresStore`)—more boilerplate than a single `use repo: UserStore` line suggests.
- **Field → key convention** (`logger` → `Logger`)—heuristic, undocumented in source until spec is tool-visible.
- **Cross-package needs** (pre-v2)—agents may wire locally complete tests that fail at `main`.

### Discovery JSON / LSP opportunities

Extend **`FunctionInfo`** with **`needs: []string`** and transitive paths for diagnostics text. Enables:

- agent preflight before codegen;
- IDE “missing Metrics at handler” same as human;
- Go-side discovery `needs` JSON for host authors and agents (not in TS emit).

**Ship inference metadata with v1**, not as optional polish—agents amplify the checker’s value.

---

## 7. Where it falls short

Honest gaps relative to product ambitions and prior art ([00](./00-prior-art.md)).

| Gap | Consequence |
| --- | --- |
| **No lifecycle / dependency graph (Effect Layers)** | Startup ordering, scoped resources, `acquire/release` stay manual in `main`. |
| **Fake authoring unchanged** | Still write struct + methods; no codegen mocks; LLM task is “implement interface,” not “import mock.” |
| **Leaf test boilerplate** | Postfix **`with { … }`** on every leaf test until helpers exist; 04’s harness idea was rejected for surface minimalism. |
| **Cross-package wiring at `main`** | Composition roots accumulate; v2 discovery mitigates, does not eliminate. |
| **No network-level test isolation without discipline** | `BlockedHttp` is opt-in convention, not compiler guarantee. |
| **Generics on wrapped Go** | Thin adapters break; hand-written Go shims persist at edges. |
| **Parallel tests + mutable fakes** | Shared **`RecordingEmail`** slices race; docs say prefer fresh **`ciAppContext()`** per test—agent/human footgun. |
| **Doesn’t solve validation, auth, observability** | **`ensure`**, middleware, OpenTelemetry remain separate investments. |
| **Risk: Go-with-extra-steps** | MVP without inference ≈ manual needs structs ([06 verdict](./06-feasibility-analysis.md#12-verdict)); keywords add parse weight without differentiation. |
| **Receiver methods prerequisite** | Entire 05 example set **does not parse today**; strategic story blocked on method semantics ([06 §6](./06-feasibility-analysis.md#6-what-is-net-new)). |

---

## 8. Strategic recommendations

Prioritize work that **maximizes positive implications** and **contains negative ones**.

1. **Treat v1 inference as the product gate**, not a follow-up: transitive **`Needs`**, scope **`with`**, wiring-root diagnostics with call chains—as sketched in 05’s completeness errors. Do not promote **`use`/`with`** publicly until then.
2. **Ship receiver methods + contract interface emission first** (MVP phase 0): without them, “types as contracts” is aspirational.
3. **Invest in Go-side discovery/`needs` JSON in the same release as v1 inference**—humans and agents share one source of truth; **never** emit requirements to TypeScript ([ADR-011](./ADR.md#adr-011-typescript-never-sees-requirements)).
4. **Document and test `ciAppContext()` / `BlockedHttp` as org conventions**, optional linter rules (`RealHttp` in `test*` functions)—compensate for no harness keyword without reintroducing DSL.
5. **Keep sidecar wire data-only**— resist pressure to pass service tokens over HTTP; document host **`AppContext`** construction in sidecar host examples.
6. **Align error/effect RFCs explicitly**: requirements ≠ **`Result`** failure types; avoid unified “effects row” that revives Koka complexity ([00 §6.1](./00-prior-art.md#61-non-goals-from-prior-art-failures)).
7. **Plan cross-package needs (v2) before large multi-package repos adopt**—otherwise **`main` wiring debt** becomes the complaint that DI frameworks were meant to solve.
8. **If inference slips > one quarter, pause keyword marketing** and ship **documented Go patterns** only—credibility cost of “compile-time completeness” without checker is high for a developer-tools product ([priorities](../../../.cursor/rules/priorities.mdc)).

---

## 9. Comparison snapshot

Small matrix—not full [00](./00-prior-art.md) survey.

| Axis | Plain Go (manual DI) | TS + Effect | TS + Jest mocks | Forst 05 (`use`/`with` + inference) |
| --- | --- | --- | --- | --- |
| **Wiring traceability** | Explicit at every call | **`R` in types** + `provide` | Hidden in mock setup | Inferred needs + **`with` at roots** |
| **Test doubles** | Hand fakes | `provideService` | `jest.mock` / manual | Struct fakes + **`with` maps** |
| **Compile-time completeness** | None | Strong (typed `R`) | None | **Strong at wiring roots** (v1+) |
| **Runtime overhead** | Minimal | Effect interpreter / fiber | Mock overhead | **Go interfaces only** |
| **TS backend story** | N/A (Go only) | Native | Native | **Logic in Forst/Go; TS via sidecar** |
| **Lifecycle / graphs** | Manual `main` | **Layers** | N/A | **Manual `main` / helpers** |
| **LLM-friendly surface** | Verbose signatures | Heavy type algebra | Framework-specific | **Two keywords + typedefs** |
| **Boilerplate** | High at scale | Medium–high | Medium (mock files) | **Medium** (fixtures + wrap) |

---

## References

- [SPEC — Function requirements](./SPEC.md)
- [06 — Feasibility analysis](./06-feasibility-analysis.md)
- [00 — Prior art](./00-prior-art.md)
- [Effect hub](../effect/00-overview.md)
- [Sidecar decisions](../sidecar/10-decisions.md)
- [ROADMAP.md](../../../ROADMAP.md)
- [@forst/sidecar](../../../packages/sidecar/README.md)
