# First use case analysis for Forst (first principles)

This document proposes what Forst should build as a **first useful library/tool** that showcases real value while matching current language and interoperability maturity.

It is intentionally opinionated and constrained by current roadmap reality.

---

## 1) Problem framing from first principles

A first use case should maximize four things at once:

1. **Signal of core value**  
   It should demonstrate what Forst is uniquely good at today: structural typing, runtime validation from type constraints, explicit error handling, predictable control flow, and TS type generation.

2. **Adoption probability**  
   Teams should be able to use it next to existing Go code with low migration risk.

3. **Scope control**  
   It must avoid roadmap areas still planned/partial (for example: user generics, channels/select, broad Go type mapping).

4. **Operational usefulness**  
   It should solve a real backend or infrastructure pain, not a toy problem.

If one candidate is impressive but hard to adopt, it fails.  
If one candidate is easy but does not showcase Forst strengths, it also fails.

---

## 2) Constraints from philosophy and roadmap

### Philosophy-aligned requirements

- Prefer explicit, deterministic control flow.
- Favor typed validation and traceable, explicit error paths.
- Avoid hidden behavior and implicit coercion.
- Keep compiler/tooling reasoning straightforward.

### Practical constraints from current status

- Go interop works for common cases but has real boundary limits (maps/struct-heavy APIs/interfaces can hit unsupported paths).
- TS interop (`forst generate`, dev server JSON contract) is usable and good for demos, but still evolving.
- Experimental language areas should be used carefully as showcase core (Result/nominal errors are promising, but avoid depending on unstable edges).

Implication: first use case should be **boundary-heavy and validation-heavy**, but with a **narrow and controlled Go/TS interop surface**.

---

## 3) Evaluation dimensions

Each option is graded on:

- **Philosophy fit**: how strongly it demonstrates Forst principles.
- **Current feasibility**: how much can be built now without waiting on planned features.
- **Adoption friction**: integration complexity for existing teams.
- **Showcase strength**: demo value for Forst narrative.
- **Scope safety**: likelihood of staying small and shippable.
- **Backend utility**: day-to-day usefulness in real systems.

Grades: **High / Medium / Low**

---

## 4) Candidate options

### Option A: Webhook Validation + Idempotency Toolkit

Library that verifies signatures, validates payload shapes, classifies typed errors, and enforces deduplication.

**Why this exists:** webhooks are hostile, duplicated, and noisy; bad handling causes real incidents.

### Option B: Database Seeding + Fixture Validation Tool

Tool that validates fixture schemas, applies ordered seed plans, and emits explicit failure reports.

**Why this exists:** broken local/dev/staging data wastes large amounts of backend time.

### Option C: DSAR Export Assembler (GDPR-adjacent slice)

Tool that gathers per-user data from internal sources, validates structures, and emits auditable export bundles.

**Why this exists:** teams need repeatable, auditable user data export workflows.

### Option D: Outbox Relay Worker Toolkit

Library/tool for validating outbox payloads, retriable delivery, and explicit poison/dead-letter outcomes.

**Why this exists:** reliable async delivery is core infra in service architectures.

### Option E: Config Validation + Safe Loader Library

Library that loads env/config files into typed structures, validates constraints, and produces actionable startup errors.

**Why this exists:** configuration drift and silent misconfiguration cause frequent production failures.

---

## 5) Comparative matrix

| Option | Philosophy fit | Current feasibility | Adoption friction | Showcase strength | Scope safety | Backend utility |
| --- | --- | --- | --- | --- | --- | --- |
| A) Webhook toolkit | High | High | Medium | High | High | High |
| B) DB seeding tool | High | High | Medium | Medium | Medium | High |
| C) DSAR assembler | High | Medium | Medium | Medium | Medium | Medium |
| D) Outbox relay toolkit | High | Medium | Medium | High | Medium | High |
| E) Config loader | High | High | Low | Medium | High | High |

---

## 6) Reasoned assessment

### A) Webhook toolkit

**Strengths**
- Hits Forst core directly: input validation + typed errors + deterministic flow.
- Bounded interface: HTTP headers/body in, validated event out.
- Easy incremental adoption: wrap existing Go handlers.
- Strong narrative: prevents duplicate processing, signature bypasses, and malformed payload bugs.

**Risks**
- Provider-specific differences can expand scope.
- Idempotency storage abstraction can become too generic too early.

**Control**
- Start with one provider shape and one dedupe backend.

### B) DB seeding tool

**Strengths**
- Very practical internal tool with immediate productivity gain.
- Strong validation story.

**Risks**
- Can drift into ORM/migration framework territory.
- Integration expectations vary widely across teams.

**Control**
- Single dialect, strict fixture format, explicit non-goals.

### C) DSAR assembler

**Strengths**
- Real compliance-adjacent utility with strong error/audit semantics.

**Risks**
- Legal/process expectations can exceed technical scope.
- Harder to generalize as first public showcase.

**Control**
- Position as data export pipeline, not "GDPR compliance automation."

### D) Outbox relay toolkit

**Strengths**
- Serious infra value; robust error handling showcase.

**Risks**
- Operational complexity (backoff, jitter, poison policies, metrics) can expand quickly.

**Control**
- Narrow to one transport and one retry policy for v1.

### E) Config loader

**Strengths**
- Easy adoption and high day-one utility.
- Simple, deterministic showcase of validation principles.

**Risks**
- Less differentiated marketing story than webhook/outbox.

**Control**
- Keep strict diagnostics and typed error taxonomy as core differentiator.

---

## 7) Recommendation

### Primary first use case: **Option A — Webhook Validation + Idempotency Toolkit**

This is the best first target because it balances:

- **High practical value** (common backend pain),
- **Strong Forst signal** (typed guards + explicit errors),
- **Contained scope** (single ingress flow),
- **Feasible implementation now** (does not require immature language/runtime surfaces).

### Secondary fallback (if maximum simplicity is required): **Option E — Config loader**

If the team needs the shortest path to shipping something useful with low integration risk, config loader is the safest second choice.

---

## 8) Suggested v1 scope (for Option A)

1. Signature verification for one provider format.
2. Event envelope and payload shape validation.
3. Typed error model (at least 10 concrete error kinds).
4. Idempotency check with one storage backend.
5. Dispatch contract for known event types.
6. Structured result/reporting object for observability.

Non-goals for v1:

- Multi-provider plugin ecosystem.
- Queue/broker abstraction layer.
- Full framework integration matrix.

---

## 9) Success criteria

The first use case is successful when:

- Integration into an existing Go service takes less than 30 minutes.
- Duplicate webhook delivery is deterministically handled.
- Invalid signatures and malformed payloads produce stable typed errors.
- Demo includes generated TS types for event contracts where relevant.
- Test suite includes high-signal fixtures for malformed and replay scenarios.

---

## 10) Why this is the correct sequencing

Forst should win early by being **useful and predictable**, not by claiming broad platform coverage too soon.

A focused webhook toolkit proves the core language thesis with real backend impact and manageable risk.  
After that, the same validation-and-errors backbone can be reused for outbox, config, and DSAR-style tools.
