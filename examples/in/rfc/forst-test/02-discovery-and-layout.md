# Discovery and layout

**Status:** Draft.

**Depends on:** [01 — Command spec](./01-command-spec.md), [ADR-018](../requirements/ADR.md#adr-018-go-native-test-entrypoints).

---

## File discovery

| Pattern | Role |
| --- | --- |
| **`*.ft`** | Production / library source |
| **`*_test.ft`** | Test source — discovered only by `forst test` (and included in package merge for typecheck) |

Respect **`ForstConfig`** include/exclude globs (same as `FindForstFiles` in `cmd/forst`).

---

## Test function discovery

A function is a **test entrypoint** when **all** of:

1. Defined in a `*_test.ft` file (or merged package containing test files).
2. Name matches **`Test[A-Z][A-Za-z0-9]*`** (Go rule).
3. Signature: exactly **one** parameter of type **`*testing.T`** (via `import "testing"`).

| Name pattern | First parameter | Result |
| --- | --- | --- |
| `TestFoo` | `t *testing.T` | Test entrypoint |
| `TestFoo` | (none) | **Compile error** — ADR-018 |
| `testFoo` | any | Helper — **not** run as test |
| `ciUserApiServices` | any | Fixture helper — **not** run as test |

**Reserved (future):** `BenchmarkXxx(b *testing.B)`, `ExampleXxx` — not v1.

---

## Package layout

### In-package tests (default)

```text
auth/
  auth.ft          package auth
  auth_test.ft     package auth
```

- Tests may call **unexported** Forst symbols in the same package.
- Simplest for `use` / `with` + internal helpers.

### External test package

```text
auth/
  auth.ft          package auth
  auth_test.ft     package auth_test
                   import "module/auth"
```

- Tests see only **exported** API — same as Go.
- Requirements wiring (`with ciUserApiServices()`) still works inside `Test*` bodies.

---

## Merge rules

- All `.ft` files in a package directory merge into **one** typecheck unit (same as `forst run` on merged packages).
- Test files contribute `Test*` functions; production files contribute code under test.
- **`Needs(f)`** computed across full merged package — test-only `use` sites in helpers propagate like production code.

---

## Config roots

| Input | Root resolution |
| --- | --- |
| `forst test` | Config discovery from cwd upward |
| `forst test ./path` | Package root containing `path` |
| Monorepo | One module root per `go.mod`; Forst config may live at repo or package level |

---

## Discovery JSON (optional)

`forst test` may emit a machine-readable list of discovered tests (name, file, package) for CI and IDE — schema TBD; not required for v1.

---

## Anti-patterns (not discovered)

| Pattern | Status |
| --- | --- |
| `func testFoo()` without `*testing.T` | **Discouraged** — not a test entrypoint |
| Go `_test.go` only | Valid via `go test` directly; `forst test` focuses on `*_test.ft` |
| Tests in non-`_test.ft` files | **Error** or warning — tests must live in `*_test.ft` |
