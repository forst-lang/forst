# Narrowing and control-flow join — semantic snapshot

Single source of truth for **soundness claims** and **join policy** referenced by the control-flow narrowing plan. Update when behavior changes.

## Soundness (summary)

- **Level A (Forst-only):** Static types and refinements follow `refinedTypesForIsNarrowing` / `InferAssertionType` and branch scope rules.
- **Level B (Go FFI):** No guarantee unless a future invalidation / trust policy is implemented; lattice ops stay free of memory effects.

## Merge after `if` (§3.2)

- At the join point after a completed `if` / `else-if` / `else` chain, the type of a binding for **uses in the continuation** is the **enclosing (pre-if) type** until union types exist.
- Implementation: `JoinAfterIfMerge` in `typeops.go`; `endIfChainApplyJoin` in `narrow_if.go` (trace-only alignment with `LookupVariable`).

## Layers

1. **`typeops.go`** — `Meet`/`Join`/`JoinAfterIfMerge` on types (pure).
2. **`flow_fact.go`** — `FlowTypeFact` for provenance; optional `MergeFlowFactsAtIfJoin`.
3. **FFI invalidation** — future; not in `Meet`/`Join`.

## Non-goals (v1)

- Whole-program pointer analysis; full alias tracking; SMT; CFG-based join for arbitrary loops (spike only).
- **Compound `ensure` subjects** (e.g. field paths): deferred — needs occurrence/path keys; see `applyEnsureSuccessorNarrowing` skip logic.

## Periodic review (anti-ossification)

Before major assertion or generics work: IR vs AST for lattice ops, variadic join, `FlowTypeFact` vs raw `TypeNode`, generics dispatch in `Meet`/`Join`.

## References

- `infer_if.go` — TS/Flow/SSA/Kotlin mapping (doc comment).
- `narrow_if.go` — branch narrowing + merge hook.
