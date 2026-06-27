# Design solutions — responses to [08 — Design analysis](./08-design-analysis.md)

**Audience:** Language designers deciding follow-up ADRs — plus anyone who read the critique and wondered “so what do we actually do?”

**Status:** Disposition ledger — see [ADR-016](./ADR.md#adr-016-post-critique-decisions-09--locked) for locked items; [10 — Needs map typing options](./10-needs-map-typing-options.md) for open typing work.

**Related:** [SPEC](./SPEC.md), [ADR](./ADR.md), [08](./08-design-analysis.md), [10](./10-needs-map-typing-options.md).

---

## Disposition (locked vs open)

| # | Topic | Status |
| ---: | --- | --- |
| 1 | Mandatory completeness | **Locked** — [ADR-016](./ADR.md#adr-016-post-critique-decisions-09--locked) |
| 2 | Comma-merge / override sugar | **Deferred** — nested `with` is fine |
| 3 | Fixture / map typing, typos | **Open** — [10](./10-needs-map-typing-options.md) |
| 4 | ProviderScope traceability | **Open** — covered in [10 § Q4](./10-needs-map-typing-options.md#cross-cutting-scope-traceability-q4) |
| 5 | Emit field types, pointers | **Partial** — no required pointers; emit in [10 § Q3](./10-needs-map-typing-options.md#cross-cutting-emit-lowering-q3) |
| 6 | Header `needs` syntax | **Locked** — inference + LSP coloring only; inner `with` satisfies locally |
| 7 | Least privilege / `pick` | **By design** — no subtract |
| 8 | Dedup `{fn}Needs` structs | **Locked** — implementation detail |
| 9 | Brownfield Go struct bridge | **Deferred** — no `forst:wire`; see [10 § Q6](./10-needs-map-typing-options.md#cross-cutting-brownfield-q6) |
| 10 | Alias shares slot | **Locked** — root key in `with` TBD in [10 § Q5](./10-needs-map-typing-options.md#cross-cutting-alias-keys-in-with-q5) |
| 11 | Nil / optional requirements | **Locked** — no nil; optional = parameter; no `?` |
| 12 | TS host manifest | **Out of scope** — user-space |
| 13 | TS / Effect `R` | **Locked** — [ADR-011](./ADR.md#adr-011-typescript-never-sees-requirements) |

---

## Before you read the list — what is this about?

Forst lets functions say **`use logger: Logger`** (“I need a logger”) and lets tests say **`with { Logger: &FakeLogger {} } { … }`** (“here’s a fake logger for everything inside this block”).

That’s useful for testing: swap real database / HTTP / email with fakes without rewriting every function.

The [design review (08)](./08-design-analysis.md) asked two harsh critics: *what’s still wrong with this?* This document answers: *here’s a simple fix for each complaint.*

**Vocabulary cheat sheet:**

| Term | Plain meaning |
| --- | --- |
| **Requirement / capability** | A service the code needs to run — logger, database, clock, HTTP client. Not the user’s request data (name, email); the *infrastructure* behind the scenes. |
| **`use`** | Inside a function: “I need X.” |
| **`with`** | Around a block: “For everything in here, supply these implementations.” |
| **Map / wiring map** | A list like `{ Logger: &fake, UserRepo: &fakeDb }` — names on the left, implementations on the right. |
| **ProviderScope / forwarding** | If you’re already inside a `with` block, inner function calls automatically get those services — you don’t repeat the map on every line. |
| **Inference / completeness check** | The compiler follows the call chain: if deep function D starts needing Email, every path that reaches D must supply Email — or build fails. |
| **Fixture** | A shared test setup, e.g. `ciUserApiServices()` returning a map full of safe fakes for CI. |

---

## Index

| # | Problem (short) | Solution (short) |
| ---: | --- | --- |
| [1](#1-inference-shipped-late-or-never) | Language ships before the smart checker | **Turn on strict checking per package when ready** |
| [2](#2-one-key-test-overrides-need-nested-with) | Changing one fake in a test is awkward | **Comma-merge `with`** |
| [3](#3-open-fat-maps-typo-keys) | Typos in big test setup maps | **Shape (C) or typedef union (F/D)** — see [unions baseline](#existing-forst-unions-and-intersections-reuse-baseline) |
| [4](#4-scope-wiring-invisible-at-call-sites) | Hard to see what services a call got | **Dev-only trace comments in generated Go** |
| [5](#5-pointer-to-interface-in-emitted-go) | Generated Go looks weird to Go devs | **Normal interfaces in structs; pointers only in maps** |
| [6](#6-requirements-absent-from-signatures) | Function header doesn’t list dependencies | **Auto-generated “needs list” for docs/tools** |
| [7](#7-no-least-privilege-no-subtract) | Tests can’t easily hide one service | **`pick` — take only the keys you want** |
| [8](#8-per-function-needs-struct-sprawl) | Compiler emits too many similar struct types | **Reuse one struct when the need list matches** |
| [9](#9-brownfield-go-appservices-struct) | Existing Go “services struct” doesn’t plug in | **Tags + one conversion function** |
| [10](#10-nominal-duplicate-slots-auditlogger--applogger) | Two type names → wire the same thing twice | **Aliases share one slot** |
| [11](#11-optional-requirements-and-nil) | “Maybe I have a logger, maybe not” | **No-op fakes first; optional `?` later** |
| [12](#12-ts-strangler-no-wiring-visibility) | TypeScript team can’t see blocked routes | **Host manifest: “not runnable yet” + why** |
| [13](#13-no-composable-r-for-effect-authors) | Not Effect-TS; don’t pretend it is | **Say so clearly in docs** |

---

## 1. Inference shipped late or never

### In plain English — the problem

Imagine buying a spell-checker that only highlights words you manually marked as “check this.” You still have to find mistakes yourself. That’s `use`/`with` **without** the compiler following the whole call tree: you write extra syntax, but you don’t get the main benefit — **“you forgot to plug in Email in tests” becomes a build error instead of a production bug.”**

If the language ships before that follow-the-chain checker works, teams get **more typing, less safety** than plain Go.

### In plain English — the solution

**Don’t force everyone on day one.** Add a switch per package: “this package must have full requirement checking.” Internal teams flip it on when the compiler is ready; app code in CI runs with it on. Until then, packages stay in “training wheels” mode.

Like `strict` TypeScript or `-race` in Go tests — opt in when you’re ready, required when it matters.

### Technical detail

**Problem ([08 § Round 1](./08-design-analysis.md#what-both-critics-attack-hardest)):** `use`/`with` without transitive checking is strictly worse than hand-written Go.

**Solution: Package completeness gate**

- Package directive: **`complete requirements`** (or `forst.toml` setting).
- When set: any function needing services must have them supplied by a `with` block (or equivalent) — else **compile error**.
- Default **off** during compiler bring-up; **on** for application packages in CI.

```forst
package users complete requirements

func createUserInternal(...) {
    use repo: UserRepo
    // ...
}
```

**Cost:** One package-level modifier, not a new keyword.

---

## 2. One-key test overrides need nested `with`

### In plain English — the problem

You have a standard CI test setup (logger, fake DB, blocked HTTP, …). One test needs a **different clock** — pretend “now” is 2000 so a token looks expired.

Today that means **blocks inside blocks**:

```forst
with bigSetup {
    with { only Clock changed } {
        run test
    }
}
```

In normal Go tests you’d write one struct literal per table row. Nested blocks feel heavy for “change one field.”

### In plain English — the solution

Allow **several maps in one `with`**, merged left to right — like spreading overrides on top of defaults:

```forst
with bigSetup, { Clock: &FakeClock { fixedMs: 2000 } } {
    run test
}
```

Same idea as “start from defaults, override clock” — **one line, one block**, no nesting.

### Technical detail

**Solution: Comma-merge `with`**

```forst
with ciUserApiServices(), { Clock: &FakeClock { fixedMs: 2000 } } {
    result := expireToken(token)
    ensure result is Err(Expired {})
}
```

Later maps override earlier keys (ADR-005). No postfix `f() with { }`.

---

## 3. Open fat maps, typo keys

### In plain English — the problem

Your CI helper returns a big map: Logger, UserRepo, HttpClient, Metrics, …

You typo **`Metricks`** instead of **`Metrics`**. Nothing breaks until **months later** some deep function starts using Metrics — then tests fail mysteriously. The typo lived in the shared fixture the whole time.

Open maps are flexible but **bad at catching spelling mistakes** on keys you’re not using yet.

### In plain English — the solution

**Two speeds:**

1. **Big shared fixtures** — use a **fixed checklist** type: “this helper must only contain Logger, UserRepo, Metrics, …” Typos fail **when you write the helper**, not months later.

2. **Small one-off overrides** (comma-merge from §2) — still checked: unknown key name = error immediately.

Think: Excel template with named columns vs. a freeform JSON blob.

### Technical detail

**Solution: Closed fixture type + open overlay**

```forst
type CI = Needs<Logger, UserRepo, HttpClient, EmailSender, Metrics>

func ciUserApiServices(): CI {
    return CI {
        Logger:  &NopLogger {},
        Metrics: &NopMetrics {},  // Metricks → compile error here
    }
}
```

Overlay literals in comma-merge: unknown key → **error**; extra valid but unused key → **warning**.

---

## 4. ProviderScope wiring invisible at call sites

### In plain English — the problem

Inside a `with` block, inner calls **automatically** get logger, DB, email, etc. You don’t see that on the line that calls `sendWelcome(user)`.

When something goes wrong at 2am (“wrong email implementation was used”), you can’t easily **grep the code** and see which setup was active. The compiler knows; the source file doesn’t show it.

### In plain English — the solution

When building for **development/debug**, the compiler adds **comments in the generated Go** above each call:

```go
// forst: services from main.go line 38 — Logger, UserRepo, EmailSender
sendWelcome(needs, user)
```

Production builds skip the comments. No runtime cost — just a paper trail for humans debugging.

### Technical detail

**Solution: Dev-only emit trace comments**

Flag: `forst build -trace-needs` or `debug.needsTrace: true`.

---

## 5. Pointer-to-interface in emitted Go

### In plain English — the problem

Forst compiles to Go. Go developers expect dependency structs to look like:

```go
type Needs struct {
    Logger Logger  // interface
}
```

The current design sometimes emits **`Logger *Logger`** — a pointer *to* an interface. Go style guides say that’s odd and reviewers push back. Same behavior, worse first impression.

### In plain English — the solution

**Maps** (where you wire fakes) still use pointers: `Logger: &FakeLogger{}` — so tests can share one fake instance.

**Generated function parameters** use normal interface fields — what Go devs already write by hand.

Split: pointers where you **configure**, interfaces where you **call**.

### Technical detail

**Solution: Interfaces in struct, pointers only in maps**

```go
type expireTokenNeeds struct {
    Logger Logger  // interface, not *Logger
    Clock  Clock
}
```

Map values remain `&NopLogger{}`; assign into interface fields at call site.

---

## 6. Requirements absent from signatures

### In plain English — the problem

You read:

```forst
func handleCreateUser(input CreateUserRequest): Result(...)
```

The header shows **request data**, not **“needs logger, database, email.”** Those live inside the body (`use logger`, etc.). New callers only discover missing wiring when the compiler errors — not by reading the function signature.

### In plain English — the solution

The compiler **already knows** the list. Expose it without making authors maintain two lists:

- IDE hover: “Needs: Logger, UserRepo, EmailSender”
- Optional auto-generated comment or `forst show needs` output
- Same list in API discovery JSON for tooling

You still write `use` in the body once; docs stay in sync automatically.

### Technical detail

**Solution: Read-only `needs` stub (derived, not authored)**

```forst
// derived — do not edit
needs handleCreateUser = { Logger, UserRepo, EmailSender }
```

LSP / JSON only is enough; on-disk stub optional.

---

## 7. No least privilege / no subtract

### In plain English — the problem

Your CI map includes **everything** — including HttpClient. You run a test that should **never touch the network**, but the full map is still “in the air” for all inner calls. If someone adds `use http: HttpClient` deep inside, it silently works — including real HTTP if the map wasn’t careful.

You can’t say “inside **this** block, only Logger and UserRepo exist — nothing else.”

### In plain English — the solution

A small library helper **`pick`**: take the big map, keep **only** the keys you name:

```forst
with needs.pick(ciUserApiServices(), Logger, UserRepo) {
    auditOnly(...)  // HttpClient not in the smaller map
}
```

Not “remove HttpClient from the universe” — **start a smaller scope** with a subset. Explicit and readable.

### Technical detail

**Solution: `pick` projection at scope entry**

```forst
import "forst/needs"

with needs.pick(ciUserApiServices(), Logger, UserRepo) {
    auditAndNotify(...)
}
```

Compile-time: picked keys must exist in source map.

---

## 8. Per-function `{fn}Needs` struct sprawl

### In plain English — the problem

The compiler generates a Go struct for each function’s dependencies. Forty handlers that all need `{Logger, UserRepo, Clock}` might get **forty struct types** with the same three fields — different names, same shape. Refactoring and reading generated code gets noisy.

### In plain English — the solution

**Deduplicate:** if two functions need the exact same set, emit **one shared struct type** (maybe with a hash name or a friendly alias). Like using one `AppDeps` struct instead of `Handler1Deps`, `Handler2Deps`, … when they’re identical.

### Technical detail

**Solution: Canonicalize needs structs by set hash**

Sort `Needs(f)` → hash → shared `Needs_a1b2c3` for all functions with identical sets.

---

## 9. Brownfield Go `AppServices` struct

### In plain English — the problem

Your existing Go app already has:

```go
type AppServices struct {
    Log *slog.Logger
    DB  *sql.DB
}
```

passed from `main`. Forst wants **maps** with keys like `Logger`, `UserRepo`. You don’t want to rewrite the whole app — you want a **bridge**.

### In plain English — the solution

1. Mark fields once: “this field wires to requirement `Logger`” (a struct tag).
2. Write **one function** (or generate it): `needsFromApp(app)` → Forst map.
3. Use `with needsFromApp(app) { … }` at the boundary.

Legacy struct stays; conversion happens in one place.

### Technical detail

**Solution: `forst:wire` struct tags + one adapter**

```go
type AppServices struct {
    Log *slog.Logger `forst:"Logger"`
    DB  *sql.DB       `forst:"UserRepo"`
}
```

```forst
with needsFromApp(app) {
    handleCreateUser(req)
}
```

---

## 10. Nominal duplicate slots (`AuditLogger` + `AppLogger`)

### In plain English — the problem

You define two types, `AuditLogger` and `AppLogger`, that both mean “something with `.info()`”. They’re really the same logger in production. But the type system treats them as **two different slots** — so you wire the same `&StdLogger{}` **twice** in the map under two keys. Redundant and confusing.

### In plain English — the solution

- **`type AuditLogger = Logger`** (alias) → **one slot**, one map entry. Alias is just another name for the same requirement.
- **Two genuinely different type definitions** → still two slots (if you really want separation); docs say: prefer one logger type unless you truly need two.

Also: using `use audit: Logger` and `use app: Logger` in one function = **one** slot, not two.

### Technical detail

**Solution: `type X = Y` shares slot; distinct nominal defs stay distinct**

Rule amendment to N8; clarify in 05.

---

## 11. Optional requirements and nil

### In plain English — the problem

Sometimes logging is optional — “if we have a logger, log; if not, skip.” The strict design says: **every** `use` must be wired somewhere, always. That forces you to always pass *something*, and nil pointers can still crash if someone messes up.

### In plain English — the solution

**Keep it simple first:** wire a **no-op logger** (`&NopLogger{}`) that does nothing — satisfies “must be wired,” safe in tests.

**Later (optional):** `use logger?: Logger` meaning “might be missing; check before use.” Prefer no-op fakes over nil for v1.

### Technical detail

**Preferred v1:** require key always; optional behavior = `&NopLogger{}`.

**Possible v2:** `use logger?: Logger` with nil-safe lowering.

---

## 12. TS strangler, no wiring visibility

### In plain English — the problem

You’re migrating: TypeScript frontend calls Go/Forst handlers through a sidecar. TypeScript only sees routes that are **fully wired and ready**. A handler still missing Email in the Go host **doesn’t appear** in the TS client — but the TS team may not know *why* it’s missing or that someone is still working on wiring.

### In plain English — the solution

A **Go-side manifest** (not in TypeScript types) lists endpoints and status:

```json
"handleCreateUser": {
  "runnable": false,
  "reason": "host must wire EmailSender first",
  "needs": ["Logger", "UserRepo", "EmailSender"]
}
```

Platform/dashboard shows “blocked routes.” TS client unchanged — still no fake dependency types on the wire.

### Technical detail

**Solution: Host manifest `runnable: false` + reason**

Discovery JSON extension; ADR-011 unchanged for TS emit.

---

## 13. No composable `R` for Effect authors

### In plain English — the problem

If you come from **Effect-TS**, dependencies show up in the **function type** — you can see and compose `R` (requirements). Forst deliberately **does not** put that in signatures or TypeScript. Effect fans may feel something important is missing.

### In plain English — the solution

**Be explicit:** Forst requirements are a **Go backend testing / wiring aid**, not a full effect system. TypeScript stays data-only; the Go host owns services.

Host Go code *can* export a named `Needs<…>` type for its own use — but that’s not exposed to TS clients. Effect stays on the TS shell if you want it; Forst doesn’t try to be half-Effect.

### Technical detail

**Solution: Explicit non-goal + host-only `Needs` type export**

Document in 07/ADR; no `R` in Forst function types or TS stubs.

---

## Prioritization (superseded by ADR-016)

See disposition table at top. Remaining work: **[10 — Needs map typing options](./10-needs-map-typing-options.md)**.

---

## Summary

Follow-ups from [08](./08-design-analysis.md) are **locked** in [ADR-016](./ADR.md#adr-016-post-critique-decisions-09--locked) or **deferred** to [10](./10-needs-map-typing-options.md). Still **two keywords** (`use`, `with`) and **maps** — no header `needs`, no `?`, no subtract, no TS exposure.
