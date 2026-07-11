# Forst Providers (`use` / `with`)

Specifications for **Providers**: how Forst functions declare **runtime behavior** they need (loggers, clocks, databases) separately from **data parameters**, and how tests **swap implementations** at scope boundaries without DI frameworks.

## Current recommended direction

**Start here:** **[04 — Redesign](./04-redesign.md)** — actionable forward path: `type` capability + `use` / `supply`, inference-first, no `requirement` primitive. Supersedes the critique doc for implementation.

**Alternative minimal design:** **[SPEC — Function requirements](./SPEC.md)** — `use` / `with` only, imported types as contracts, no test DSL.

**RFC (consolidated):** **[RFC — Providers](./RFC.md)** — reader-facing RFC synthesizing [SPEC](./SPEC.md) + [ADR](./ADR.md): motivation, RFC 2119 requirements, decision registry.

**Design decisions:** **[ADR](./ADR.md)** — 46 atomic accepted decisions (always-forward scope, derived `Providers(f)`, etc.).

**Feasibility:** **[06 — Feasibility analysis](./06-feasibility-analysis.md)** — compiler audit of 05: partially feasible, phased MVP → v1 → v2.

**Strategic implications:** **[07 — Language implications](./07-language-implications.md)** — product and language evolution commitments if 05 is adopted.

**Design analysis:** **[08 — Design analysis](./08-design-analysis.md)** — three adversarial critiques (Go, PL, compiler/DX) from [SPEC](./SPEC.md) + [ADR](./ADR.md) only; synthesis and adoption risk register.

**Design solutions:** **[09 — Design solutions](./09-design-solutions.md)** — disposition ledger for critique follow-ups (locked in [ADR-003](./ADR.md#adr-003-no-author-written-providers-clause-on-signatures)–[ADR-046](./ADR.md#adr-046-ensure-in-tests-lowers-to-tfatal--terror)).

**Needs map typing (open):** **[10 — Needs map typing options](./10-needs-map-typing-options.md)** — shapes, **typedef unions (`A | B`)**, intersections; builds on implemented binary types.

**Test runner (CLI):** **[forst-test RFC](../forst-test/README.md)** — `forst test` command, discovery, emit bridge (test **syntax** is in [SPEC § Testing](./SPEC.md#testing-go-native) + [ADR-044](./ADR.md#adr-044-test-entrypoints-use-test-and-testingt)).

| Doc | Role |
| --- | --- |
| **[RFC — Function requirements](./RFC.md)** | **Consolidated RFC** — motivation, RFC 2119 requirements (R-LEX…R-TEST), decision registry, conformance checklist. |
| **[SPEC — Function requirements](./SPEC.md)** | **Normative spec** — algorithms, schemas, lowering examples, full test patterns. |
| **[ADR — Design decisions](./ADR.md)** | **46 atomic ADRs** — one decision per record; traceability for SPEC/RFC. |
| **[06 — Feasibility analysis](./06-feasibility-analysis.md)** | Compiler audit of 05 — phased MVP → v1 → v2; conditional go. |
| **[07 — Language implications](./07-language-implications.md)** | Strategic analysis — commitments, constraints, TS/agent/product impact of 05. |
| **[04 — Redesign](./04-redesign.md)** | **Recommended direction** — executive summary, testing boilerplate analysis, **[integration test harness](./04-redesign.md#integration-test-harness)**, syntax, migration from 01, next steps. |
| **[03 — Critique & alternatives](./03-design-critique-and-alternatives.md)** | Critical review of 01 — overlap with Go, naming, shape-based types. Historical; 04 consolidates actionable conclusions. |
| **[01 — Normative spec](./01-normative-spec.md)** | Consolidated v0 design (`requirement` + `require` + `provide`) — **historical target** until implementation follows 04. |
| **[00 — Prior art](./00-prior-art.md)** | Research reference — Effect/ZIO, Unison, Rust, Go, Kotlin context, codebase audit. **Not normative.** |
| **[02 — Examples](./providers.ft)** | Minimal compiling Providers showcase (`use` / `with` / nested transitive `with`). |

### Competing designs (archived proposals)

These informed **01**; kept for traceability, not as competing targets:

| Design | File |
| --- | --- |
| Agent A — explicit `require` + `provide` | [designs/agent-a-explicit-require.md](./designs/agent-a-explicit-require.md) |
| Agent B — implicit `using` context | [designs/agent-b-using-context.md](./designs/agent-b-using-context.md) |
| Agent C — `?T` / `@require` operators | [designs/agent-c-operator-syntax.md](./designs/agent-c-operator-syntax.md) |

## Redesign surface (summary)

From **[04](./04-redesign.md)**:

| Token | Role |
| --- | --- |
| **`type T = capability { … }`** | Capability contract (emits Go interface). |
| **`use`** | Bind a capability in the body (optional when inference from `ctx` suffices). |
| **`supply`** | Satisfy capabilities at a call site or scope entry (`provide` acceptable alias). |
| **`harness`** | Named test preset (e.g. `CI`) with default fakes; tests `uses CI` and `override` deltas only. |

**Lowers to:** Go interfaces + per-function needs struct + struct literals at boundaries — same as hand-written Go constructor injection.

## Related RFCs

- [Errors hub](../errors/README.md) — `ensure` is validation/narrowing, **not** requirements.
- [Effect hub](../effect/00-overview.md) — error channels and TS interop; requirements are **Go-side capabilities**, not a full `Effect<A,E,R>` monad.
- [Sidecar decisions](../sidecar/10-decisions.md) — host builds handler service bundles; wire format stays **data-only**; TS sees **runnable exports only** ([ADR-021](./ADR.md#adr-021-runnable-exports-only-when-providersf-is-empty)).
- [Guard / `ensure`](../guard/guard.md) — orthogonal to capability wiring.
- [Forst test runner](../forst-test/README.md) — `forst test` CLI (discovery, emit, `go test` bridge).

## Quick start

See **[04 — Redesign](./04-redesign.md)** for full rules. Sketch:

```ft
type Logger = capability {
    info(msg String)
}

func greet(name String) {
    use logger: Logger
    logger.info("hello " + name)
}

func main() {
    supply { Logger: StdLogger {} } {
        greet("world")
    }
}
```

Tests supply fakes the same way — handler suites can share a test context:

```ft
test "greet logs" {
    supply { Logger: NopLogger {} } {
        greet("Ada")
    }
}
```

**Integration suites:** use a named **`harness`** preset (e.g. `CI`) for CI-safe defaults, then **`override`** only what differs — see **[04 § Integration test harness](./04-redesign.md#integration-test-harness)**.
