# Merged Go statement plan

**Goal:** Reach **≥79.5%** merged statement totals on:

```bash
cd forst
go test -coverprofile=profile.cov ./cmd/forst/... ./internal/... -count=1
bash scripts/check_coverage_threshold.sh profile.cov 79.5
```

Or: `task check:coverage MIN_COVERAGE=79.5` (from `forst/`, after `profile.cov` exists or let the task generate it).

Shortcut: `task check:coverage:adoption` (repo root) defaults **MIN_COVERAGE=79.5**; use `MIN_COVERAGE=80` to override.

**Last measured baseline (repo snapshot):** **~76.8%** total (`go tool cover -func=profile.cov` last line). Re-run after substantive test changes.

**Progress toward ≥79.5%:** Additional tests (builtin dispatch error paths, `LookupInferredType`, `transform_pipeline_batch` snippets, `typedef_expr` pointer case) did **not** yet close the gap—merged % moves slowly when new tests duplicate already-exercised statements. The next gains need **cold** branches (use `go tool cover -func` on a package profile and target functions still &lt;70% or `go tool cover -html` on `profile.cov`).

**Why merged:** One number across `cmd/forst/...` and `internal/...` matches local `check:coverage` / threshold script behavior; per-package % differs.

---

## Completed (reference)

- `cmd/forst/lsp/document_uri_test.go` — `legacyLocalPathFromFileURI` success path, non-`file://` rejection, `localPathFromFileURI` non-file scheme → legacy.
- `internal/transformer/go/transform_pipeline_batch_test.go` — `transformInitPostStmt`: post as assignment (`i = i + 1`) and as bare call (`tick()`).
- `cmd/forst/lsp/navigation_locals_test.go` — `handleDefinition` short decl in `else` branch and `if x := …; cond` init (`TestHandleDefinition_ifInitShortDecl`); else-if / for-body locals still return nil (not asserted).
- `internal/typechecker/narrowing_test.go` — `TestRefinedTypesForIsNarrowing_literalLHSFailsGetLeftmostVariable` (renamed from misleading assertion-subject name).
- `internal/typechecker/lookup_field_value_constraint.go` — `inferValueConstraintType` heavily covered via `lookup_field_value_constraint_test.go` (literals, refs, nil, dot paths, `ReferenceNode` value form, errors).
- `internal/typechecker/go_builtins_dispatch_edge_test.go` — `TestCheckBuiltinFunctionCall_dispatchArityAndOperandErrors` (len/append/copy/delete/clear/close/min/complex/real/panic error branches).
- `internal/typechecker/lookup_inferred_test.go` — `LookupInferredType` requireInferred / empty / missing paths.
- `cmd/forst/lsp` — `find_innermost_scope_test.go` for else-if / final else / ensure block scopes; `navigation_locals_test.go` — `TestHandleDefinition_shortDeclInsideIfBlock`, `TestDefiningTokenForLocalBinding_ifBody_matchesHandleDefinition` (if-body local go-to-def).
- `internal/goload/load_test.go` — `TestLoadByPkgPath_os`.
- `internal/transformer/go/transform_pipeline_batch_test.go` — extra minimal programs (break/continue, range, map+delete, type alias field, …) + `log.SetOutput(io.Discard)` fix.
- `internal/transformer/go/typedef_expr_branch_test.go` — `TestTransformTypeDefExpr_pointerAssertionExpr_deref` for `*TypeDefAssertionExpr`.
- `internal/transformer/go/pipeline_emit_ensure_test.go` — `TestPipeline_nominalErrorUnion_typedef_emitsMembersAndUnion` (nominal errors + `type U = A | B` end-to-end emit).
- `internal/typechecker/narrow_if_test.go` — `TestAssertionNodeFromIsRHS_branches` (`assertionNodeFromIsRHS` switch cases).

---

## Remaining work (priority order)

Merged % moves slowly unless **new branches** run in **large** packages. Priority:

1. **`internal/transformer/go`** (often ~70% package coverage, many statements)  
   - Add **`compileForstPipeline`** (or batch) tests for cold paths in `statement.go`, `expression.go`, `shape.go`, `ensure*.go`, `typeguard.go`.  
   - Use `go test -coverprofile=t.cov ./internal/transformer/go` then `go tool cover -func=t.cov | sort` to find low % functions.

2. **`cmd/forst/lsp`** (large surface, completion / server)  
   - Integration tests hitting **`getCompletionsForPosition`**, **`completion.go`** helpers still under-covered (`deepestScopeInFunction`, handler paths in `server.go`).

3. **`internal/typechecker`** (after big wins elsewhere)  
   - **`go_builtins_dispatch.go`** error branches and rare builtins.  
   - **`narrow_if.go`** / **`infer_assertion.go`** gaps where `go tool cover` still shows holes.

4. **Examples pipeline**  
   - `TestTransformForstFileToGo_examplesInTree` skips on parse/typecheck/transform failure; fixing or narrowing failing `examples/in/**/*.ft` files increases real pipeline coverage (optional).

---

## Gate policy

- **`Taskfile.yml`** `check:coverage` default **`MIN_COVERAGE`** stays **72** until the team explicitly raises it—bumping before the merge actually clears the threshold breaks `task check:coverage` for everyone.
- When merged **≥79.5%** is stable: verify with `check_coverage_threshold.sh`, then consider default **`MIN_COVERAGE=79.5`** (or **80** if policy requires a round number) and CI alignment.

---

## Checklist (copy into issues / todos)

- [ ] Merged ≥79.5% verified with `check_coverage_threshold.sh profile.cov 79.5`
- [ ] Transformer/go: targeted `compileForstPipeline` or batch tests for top uncovered emit paths
- [ ] LSP: tests for completion/server branches still red in `go tool cover -func` (merged or `cmd/forst/lsp` profile)
- [ ] Typechecker: builtins dispatch and/or narrowing tests if still below target after 1–2
- [ ] (Optional) Reduce skips in `examples_batch_transform_test.go`
- [ ] Document or bump `MIN_COVERAGE` only after gate passes

