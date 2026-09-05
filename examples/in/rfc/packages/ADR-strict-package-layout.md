# ADR: Strict package layout for Go target

**Status:** Accepted.

**Context:** Forst transpiles to Go. Go requires one package per directory. Adopters colocate `.ft` with TypeScript and need to split one logical package across multiple files without guessing compiler behavior.

---

## Decision

**Strict layout** is the Go-target contract:

| Rule | Behavior |
| --- | --- |
| Same directory, same `package` | Supported. Merged compilation unit. |
| Same `package` in sibling directories | Compile error. |
| Multiple package names in one directory | Compile error (matches Go). Exception: `package foo` + `package foo_test` in the same directory. |
| `files.include` globs | Select which `.ft` files are discovered. Do not override directory rules. |

Recommended multi-file layout:

```text
auth/
  auth.ft
  bcrypt.ft
```

Both files declare `package auth`.

---

## Rationale

- Go emit requires one directory per package at build time.
- Name-only merge across directories produced non-deterministic import paths (first sorted file’s directory).
- LSP already merges same-package files in one directory only.

---

## Alternatives considered

| Option | Outcome |
| --- | --- |
| **Flexible:** merge by package name regardless of path | Rejected. Hides layout bugs until Go build fails. |
| **Explicit:** `ftconfig` package → glob manifest | Deferred. Escape hatch for future monorepos that cannot use one dir per package. |

---

## Implementation

- `forstpkg.ValidateGoPackageLayout` runs in `modulecheck.ScanModule` and `collectSamePackageFtPaths` (`-root`).
- `--allow-stem-package-mismatch` remains generate-only for file naming. It does not bypass layout errors.

---

## Consequences

- Splitting helpers: add files under the package directory, not sibling folders.
- Colocation with TypeScript: use `feature/auth/*.ft`, not `feature/authentication/auth.ft` and `feature/authentication/crypto/bcrypt.ft` with the same package name.
