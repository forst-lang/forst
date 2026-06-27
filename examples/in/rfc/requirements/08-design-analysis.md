# Design analysis — competing critiques of `use` / `with`

**Subject:** [SPEC — Function requirements](./SPEC.md) as locked by [ADR-001–017](./ADR.md) (Usable shape literals, inferred `Needs(f)`, mandatory completeness, nested `with`, always-forward ambient).

**Method:** Three adversarial critics read **only** [ADR.md](./ADR.md) and [SPEC.md](./SPEC.md). This document synthesizes overlap, divergence, and adoption implications. No prior analysis docs (06, 07, 09, 10) were inputs.

| Critic | Persona | Title | Scores |
| --- | --- | --- | --- |
| [Go pragmatist](68a4634e-9c67-482b-ab15-a00458acf4b7) | Senior Go backend engineer | *Honest Go, Awkward Tests* | **6/10** vs hand-written Go deps + table tests |
| [PL theorist](6a6fad23-fc16-490a-a140-adb755d4a824) | Effects / row types researcher | *Ambient DI Without an Environment Algebra* | **6/10** elegance, **5/10** semantic coherence |
| [Compiler/DX engineer](dbeeda4c-a7b1-444a-802d-4cc62831403c) | LSP / typechecker tooling | *Derived Needs, Hidden Wiring* | **6/10** implementability + DX (complete feature per ADR-009) |

---

## Executive summary

All three critics land near **6/10** on their primary axis. Convergence is unusually strong: the design is **honest about Go lowering** and **small on the surface** (two keywords, ordinary contract types, no DI runtime), but the **product is transitive completeness checking** (ADR-009), not syntax. Without day-one inference, LSP, and obligation-chain diagnostics (ADR-003), the feature becomes **annotated Go with worse override ergonomics** than hand-written `{fn}Needs` structs.

Shared top risks:

1. ~~**SPEC vs ADR emit inconsistency**~~ — **Resolved:** SPEC § Go lowering now matches ADR-013 (contract-typed fields by value; direct field copies, not map indexing).
2. **Nested `with` as the only override** (ADR-005, ADR-016 §2) — table-driven unit tests lose to one-shot struct literals; integration suites win.
3. **Always-forward ambient** (ADR-004) — call sites hide callee requirements; mitigation is 100% tooling-dependent with no signature anchor (ADR-016 §3).
4. **Fat Usable typo asymmetry** (ADR-012, ADR-017) — missing fields hard-error; unknown/extra fields in CI fixtures may warn-only when no reachable callee needs them.
5. **No brownfield bridge** (ADR-016 §6) — existing Go `Deps` structs have no incremental adoption path.

**Verdict:** Adopt for **greenfield Forst backends** where Go owns wiring and integration-style tests dominate. Defer for **table-heavy unit test cultures** and **brownfield monoliths** until tooling checklist ships and override ergonomics are pilot-tested.

---

## What all three critics attack hardest

### 1. Inference + completeness *is* the feature — keywords alone are tax

ADR-009 specifies a **single complete feature**: transitive propagation, cross-package fixed-point, mandatory completeness, no opt-out. All three rank failure to ship the checker + LSP pass as the primary failure mode.

- **Go:** Leaf `use` changes fail at distant wiring roots — good for safety, bad for locality.
- **PL:** No compositional law tying `Needs(f)`, `Usable`, and `{fn}Needs` — three names, one phenomenon.
- **DX:** ADR-003’s “no header `needs`” swaps duplicate author lists for **invisible compiler state** unless hover, inlay hints, and obligation chains ship day-one.

### 2. Nested `with` vs one-shot struct literals

ADR-015 bans postfix `f(x) with { … }`; ADR-005 makes nested blocks the **only** override form.

| Plain Go | Forst |
| --- | --- |
| `expireToken(expireTokenNeeds{Clock: tt.clock, Logger: nop}, tok)` per table row | `t.Run(name, func(t *testing.T) { with ciUserApiServices() { with { Clock: fake } { … } } })` per row ([ADR-018](./ADR.md#adr-018-go-native-test-entrypoints), [SPEC § Testing](./SPEC.md#testing-go-native)) |

Go critic: daily ergonomics loss vs one-shot struct literals **partially addressed** by **`t.Run`** table tests in Forst. PL critic: ceremony a row-polymorphic provide would collapse. DX critic: shadow stacks opaque without “effective ambient Usable” IDE command.

**Design response (if any):** **`t.Run`** subtests are the normative table-test path ([ADR-018](./ADR.md#adr-018-go-native-test-entrypoints)). Postfix `with` desugar remains optional future sugar. Go `_test.go` bypass still valid for hand-written struct literals ([ADR-007](./ADR.md#adr-007-go-lowering--needs-struct-as-first-parameter)).

### 3. Always-forward ambient hides wiring

ADR-004 + ADR-016 §4: no subtract, pick, or least-privilege. Inner calls inherit full ambient; source at `return expireToken(token)` shows **zero** requirement evidence.

- **Go:** Harder incident debugging vs explicit deps struct at call site.
- **PL:** Implicit capability passing without environment algebra — opposite of grep-able Effect `provide`.
- **DX:** “Find wiring roots” and effective-ambient views are **mandatory** LSP features, not nice-to-haves.

### 4. Usable vs `Needs(f)` — typo safety and namespace split

ADR-017: **Usable** = fixture-shaped wiring (fields = contract idents). ADR-003: **`Needs(f)`** = inferred per function. ADR-012: superset allowed; extras → warning.

All three cite **`Metricks`** in `ciUserApiServices()`: caught at typedef definition, but may survive as yellow squiggle in fat fixtures until a distant leaf `use`s `Metrics`. Unknown idents should hard-error (ADR-012 already leans this way).

Alias keys (`AuditLogger` vs `Logger`) remain **TBD** (ADR-016 §7) — all three flag this as coherence hole.

### 5. ~~SPEC lowering contradicts ADR-013~~ (resolved)

ADR-013: needs struct fields use **contract types by value**; pointers optional in wiring only. SPEC § Go lowering now matches: `Logger Logger` not `*Logger`; ambient → call site via **direct field copies** (`wiring.Logger`), not map indexing.

---

## Where the critics diverge

| Topic | Go pragmatist | PL theorist | Compiler/DX |
| --- | --- | --- | --- |
| **Primary failure mode** | Migration + test verbosity | Missing environment algebra | Tooling not shipped with language |
| **Effect comparison** | TS/Go split is fine (ADR-011) | Lost `{R}` without substitute | N/A |
| **Usable shapes** | Adds fixture taxonomy; merge helpers unfinished | Two namespaces for one phenomenon | Parallel type rules on shapes |
| **No optional `use`** | Forces noop proliferation in fat fixtures | Forbids typed optional effects | Nil-at-wiring checker is crisp once diagnostics exist |
| **Score yardstick** | vs hand-written Go DI | vs row-polymorphic elegance | vs ADR-009 all-or-nothing ship bar |
| **Best audience** | Greenfield Forst, integration tests | Authors who wanted signatures to carry requirements | Teams with LSP investment |

Same design, different yardsticks — all near **6/10**, none near **8/10** until emit lock + tooling + pilot data.

---

## Convergent fair wins (none dispute)

1. **Two keywords, types as contracts** (ADR-001, ADR-002) — no parallel capability DSL; fakes are ordinary structs; imported Go interfaces work.
2. **No runtime DI container** (ADR-008) — wiring visible at `main`, tests, and fixtures; boring stack traces.
3. **Transitive mandatory completeness** (ADR-009) — the feature hand-written Go lacks; leaf change → error at wiring root with callee chain.
4. **Body as source of truth** (ADR-003) — no author-maintained `uses` clauses that drift from bodies.
5. **Superset fat fixtures** (ADR-012, ADR-014) — one `ciUserApiServices()` serves heterogeneous callees; matches real CI culture.
6. **TS boundary discipline** (ADR-011) — runnable exports only; host wires before expose; no fake capability channel in generated TS.
7. **Usable = shape literals** (ADR-015, ADR-017) — same `{ field: value }` syntax as data shapes; not Go maps; not `with ctx` struct forwarding.

---

## Critic summaries

### Go pragmatist — [Honest Go, Awkward Tests](68a4634e-9c67-482b-ab15-a00458acf4b7)

**Adopt:** Greenfield Forst backend, Go-owned wiring, integration-test mental model.

**Wait:** Table-driven unit test teams; brownfield monoliths until bridge exists.

**Conditions to reach 7+/10:** Align SPEC emit with ADR-013; table-test-friendly wiring path; documented brownfield `Deps` → Usable migration; pointer vs value locked in SPEC not deferred docs.

### PL theorist — [Ambient DI Without an Environment Algebra](6a6fad23-fc16-490a-a140-adb755d4a824)

**Net vs Effect:** Gained Go-native emit, cross-package inference, mandatory completeness, honest TS split. Lost scoped `{R}`, algebraic provide/subtract, Layer lifecycle, optional requirements as types.

**Minimal fixes:** Lock alias normalization; unknown keys → error; read-only derived `// needs:` annotation; postfix desugar; one normative merge law (`merged = outer ⊕ inner`, keys(ambient) ⊇ Needs(callee)).

### Compiler/DX — [Derived Needs, Hidden Wiring](dbeeda4c-a7b1-444a-802d-4cc62831403c)

**Day-one tooling checklist (abbreviated):**

- Derived `Needs(f)` hover distinct from Usable typedef constraints
- Completeness diagnostics with `required by: f → g → h` chains
- Cross-package needs graph = discovery JSON = checker fixed-point
- Effective ambient Usable per `with` scope (shadow markers)
- Unknown wiring field → **error**; known-but-unused → **warning** with callee context
- Root ident enforcement + alias quick-fixes
- Sidecar export error when `Needs(f) ≠ ∅`

**Bottom line:** ADR-009 correctly treats inference as the feature, but ADR-003 + ADR-012 transfer complexity from source text to tooling. Ship the checklist first.

---

## Synthesis — adoption risk register

| Risk | Severity | Critics | Mitigation |
| --- | --- | --- | --- |
| Cross-package fixed-point incomplete or stale | Critical | All | Single graph source; multi-package integration tests; LSP invalidation on leaf `use` change |
| SPEC emit contradicts ADR-013 | ~~High~~ **Resolved** | All | Fixed in SPEC § Go lowering |
| Nested `with` tax on table tests | High | Go, PL | **`t.Run`** + nested `with` per subtest ([ADR-018](./ADR.md#adr-018-go-native-test-entrypoints)); Go `_test.go` bypass |
| Fat Usable typo → warning-only | High | All, DX | Hard error on unknown contract idents |
| No brownfield bridge | High | Go | Document hand-written conversion; revisit after Usable pilot |
| Always-forward hides deps at call site | High | All | Day-one inlay hints + effective-ambient IDE |
| Alias key spelling TBD | Medium | PL, DX | Lock root-ident-only; alias quick-fix |
| Usable vs `Needs(f)` conflated in tooling | Medium | PL, DX | Separate LSP fields / hover sections |
| Noop proliferation as graph grows | Medium | Go | Accept or allow parameter-level optional data (ADR-006) |
| LLM omits nested `with` on single-service patch | Medium | DX | Diagnostic snippets + discovery JSON |

---

## Conclusion

The design is **coherent as a complete feature** (ADR-009) and **honest about Go** (ADR-007, ADR-008) — but all three critics agree it is **not yet honest in its own normative spec** (emit examples) and **not yet shippable without a tooling pass** that ADR-003 implicitly demands.

Treat [SPEC](./SPEC.md) as target semantics; treat this document as the **adoption risk register**. Revisit scores after: (1) package-graph completeness pilot, (2) table-test override ergonomics measurement, (3) day-one LSP checklist shipped.

---

## References

- [SPEC — Function requirements](./SPEC.md)
- [ADR — Design decisions](./ADR.md)
