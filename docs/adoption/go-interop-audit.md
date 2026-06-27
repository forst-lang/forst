# Forst↔Go interoperability audit (codepath-based)

This document enumerates where real Go modules first hit **unsupported** or **partial** behavior, based on the compiler and loader (not exhaustive of all call sites).

## Loader (`internal/goload`)

- **`LoadByPkgPath`** uses `golang.org/x/tools/go/packages` with `NeedTypes | NeedDeps | NeedImports`.
- Failures surface as **no typed packages**; `initGoImportPackages` logs at debug and **skips Forst↔Go boundary checks** when load fails (`go_interop.go`).

## Type mapping (`internal/typechecker/go_interop.go` — `goTypeToForstType`)

Supported today (roughly):

- Go **`error`** → Forst `error`.
- **Basic**: bool, numeric kinds, string (including untyped).
- **Slice** → Forst array type with element mapping.
- **Pointer** → Forst pointer with inner mapping.

**Unsupported (`ok == false`):**

- **`unsafe.Pointer`** and other basics not listed (see `types.Basic` switch default).
- **Maps**, **structs**, **function types**, **channels**, **named types** that do not reduce to the above through `Underlying()` in this switch (the `default` branch returns false).
- **Generic** instantiations as distinct `go/types` shapes often fail unless they underlying to slice/pointer/basic.

**Symptoms at call sites:**

- `checkGoQualifiedCall` → `checkGoSignature` → **`unsupported Go return type %s`** on results.
- `checkOneGoParam` → **Forst type … not assignable to Go parameter** when `goTypeToForstType` fails for the parameter type.

## Builtins (`fmt.Print` / `fmt.Println` / `fmt.Printf`)

- Typechecking treats **`fmt.Print`** / **`fmt.Println`** arguments like Go **`...any`** (**`Object`**). **`fmt.Printf`** keeps a **`String`** format as the first argument; remaining arguments are **`Object`**.

## Qualified calls (`checkGoQualifiedCall` → `checkGoSignature`)

- When **`go/packages`** loaded a package, **`go/types`** drives return shapes (built-in overrides are not applied in that case).
- **Folding** (expression / single `:=`): Go **`(S1, …, Sn, error)`** with **`n ≥ 1`** and **`error`** last maps to **`Result(S, Error)`** if **`n == 1`**, else **`Result(Tuple(S1,…,Sn), Error)`**.
- **Unfolding** (multi-assignment): **`a, b, err := pkg.F()`** uses **`checkGoQualifiedCall(..., fold=false)`** so each name gets the per-slot type when **`len(LHS)`** matches **`len(returns)`**.
- Non-`*types.Func` objects (e.g. some **builtins** represented differently) → **`is not a function`** (see roadmap: `unsafe.Sizeof`-style issues).

## Priority for “real module” pain

1. **Return and parameter types** that are **structs**, **interfaces with methods**, **maps**, or **stdlib generics** — blocked by `goTypeToForstType` / assignability.
2. **`unsafe` package** — special-cased out or not a normal `types.Func`; roadmap documents gaps.
3. **Loader / `go/packages` failures** — silent loss of boundary checking when `GoWorkspaceDir` is wrong or `packages.Load` fails.

See [ROADMAP.md](../../ROADMAP.md) **Go interoperability** and **Builtins / unsafe** rows for product-level status.
