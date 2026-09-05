# Narrowing and control-flow join — semantic snapshot

Single source of truth for **soundness claims** and **join policy** referenced by the control-flow narrowing plan. Update when behavior changes.

## Soundness (summary)

- **Level A (Forst-only):** Static types and refinements follow `refinedTypesForIsNarrowing` / `InferAssertionType` and branch scope rules.
- **Level B (Go FFI):** Untrusted. A Go call may mutate storage reachable from its arguments and may return aliases into that storage. Active refinement facts whose dependencies may-alias that storage are dropped. Lattice ops stay free of memory effects. Normative: [refinements RFC 13](../../../examples/in/rfc/refinements/13-refinement-stability.md).

## Merge after `if` (§3.2)

- At the join point after a completed `if` / `else-if` / `else` chain, the type of a binding for **uses in the continuation** is the **enclosing (pre-if) type** until union types exist.
- Implementation: `JoinAfterIfMerge` in `typeops.go`; `endIfChainApplyJoin` in `narrow_if.go` (trace-only alignment with `LookupVariable`).

## Layers

1. **`typeops.go`** — `Meet`/`Join`/`JoinAfterIfMerge` on types (pure).
2. **`flow_fact.go`** — `FlowTypeFact` for provenance; optional `MergeFlowFactsAtIfJoin`.
3. **FFI invalidation** — [RFC 13](../../../examples/in/rfc/refinements/13-refinement-stability.md): fact-layer drop after untrusted Go / unknown calls. Not in `Meet`/`Join`. Implemented in analyzable-refinements phase 4e.

## Non-goals (v1)

- Whole-program pointer analysis; full alias tracking; SMT; CFG-based join for arbitrary loops (spike only). **Why no SMT / no WP over guard bodies:** [refinements RFC 00](../../../examples/in/rfc/refinements/00-the-trap.md); **accepted algebra:** [12](../../../examples/in/rfc/refinements/12-accepted-decision.md) (`All`/`Any`/`Atom`). **Mutation / alias / Go:** [13](../../../examples/in/rfc/refinements/13-refinement-stability.md) (invalidate facts, no borrow checker; v1 alias analysis is conservative may-alias, not a general points-to solver).
- **Compound `ensure` subjects** (e.g. field paths): deferred — needs occurrence/path keys; see `applyEnsureSuccessorNarrowing` skip logic.

## Periodic review (anti-ossification)

Before major assertion or generics work: IR vs AST for lattice ops, variadic join, `FlowTypeFact` vs raw `TypeNode`, generics dispatch in `Meet`/`Join`.

## References

- `infer_if.go` — TS/Flow/SSA/Kotlin mapping (doc comment).
- `narrow_if.go` — branch narrowing + merge hook.
- [refinements RFC](../../../examples/in/rfc/refinements/README.md) — closed atom algebra; no SMT / WP.
