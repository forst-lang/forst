# `forst test` — command specification

**Status:** Draft — target behavior for `cmd/forst test`.

**Depends on:** [requirements SPEC § Testing](../requirements/SPEC.md#testing-go-native), [ADR-018](../requirements/ADR.md#adr-018-go-native-test-entrypoints).

---

## Purpose

Run Forst tests with the same mental model as **`go test`**: discover `Test*` functions in `*_test.ft`, compile to Go, execute via the Go toolchain.

---

## Invocation

```bash
forst test [packages/paths...] [go test flags...]
```

| Argument | Meaning |
| --- | --- |
| **packages/paths** | Optional. Directories or `.ft` paths under a module root. Default: `.` (config root). |
| **go test flags** | Passed through after `--` or when recognized as `go test` flags (e.g. `-v`, `-run`, `-count`, `-parallel`). |

Examples:

```bash
forst test
forst test ./examples/in/rfc/requirements
forst test -v -run TestExpireToken
forst test ./pkg -- -count=1
```

---

## Pipeline

1. **Resolve config** — `ftconfig.json` / `ForstConfig` (same as `forst run`, `forst generate`).
2. **Discover** — `*.ft` + `*_test.ft` per [02 — Discovery](./02-discovery-and-layout.md).
3. **Merge packages** — per-package AST merge (same as multi-file compile).
4. **Typecheck** — full checker pass including requirements ([ADR-009](../requirements/ADR.md#adr-009-transitive-inference-and-mandatory-completeness)).
5. **Emit** — production `.go` + `*_test.go` per [03 — Emit bridge](./03-emit-and-go-test-bridge.md).
6. **Run** — `go test` in module root with forwarded flags.

---

## Exit codes

| Code | Meaning |
| --- | --- |
| **0** | All tests passed; compile succeeded |
| **1** | Test failure or compile/typecheck error |
| **2** | Usage / config error |

Match `go test` conventions where practical so CI scripts can swap `go test` → `forst test` with minimal changes.

---

## Output

- **Compile / typecheck errors** — Forst diagnostics on stderr (same format as `forst run`).
- **Test run** — pass through **`go test`** stdout/stderr verbatim (`-v` shows subtests from `t.Run`).
- **Requirements completeness errors** — reported at typecheck (before `go test`) with obligation chains per [SPEC](../requirements/SPEC.md).

---

## Non-goals (v1)

| Item | Notes |
| --- | --- |
| **Replace `go test` runtime** | Always delegate execution to Go toolchain |
| **TS / sidecar tests** | Requirements and tests are Go-side only ([ADR-011](../requirements/ADR.md#adr-011-typescript-never-sees-requirements)) |
| **Coverage UI** | Use `go test -cover` passthrough when needed |
| **Watch mode** | Future; use external file watcher + `forst test` in v1 |
| **Benchmark runner** | `BenchmarkXxx` reserved; not v1 |

---

## Environment

| Variable | Effect |
| --- | --- |
| **`FORST_CONFIG`** | Explicit config path (if supported by shared loader) |
| **`FORST_SKIP_TS_E2E`** | N/A for test command |

---

## Related commands

| Command | Role |
| --- | --- |
| **`forst run`** | Run `main` / single entry — not test discovery |
| **`go test`** | May run directly on emitted `*_test.go` after manual compile (debug) |
