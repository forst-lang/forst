# Forst &emsp; [![CI]][actions] [![Release]][release] [![Go Report Card]][goreport] [![Coverage Status]][coveralls] [![License]][license]

[CI]: https://img.shields.io/github/actions/workflow/status/forst-lang/forst/lint-test-coverage.yml
[actions]: https://github.com/forst-lang/forst/actions
[release]: https://img.shields.io/github/v/release/forst-lang/forst?filter=v*
[Go Report Card]: https://goreportcard.com/badge/github.com/forst-lang/forst
[goreport]: https://goreportcard.com/report/github.com/forst-lang/forst
[Coverage Status]: https://coveralls.io/repos/github/forst-lang/forst/badge.svg?branch=main
[coveralls]: https://coveralls.io/github/forst-lang/forst?branch=main
[License]: https://img.shields.io/github/license/forst-lang/forst

**Forst is a programming language that brings TypeScript's type safety and developer experience to Go.**

Its primary goal is to help you move away from TypeScript on the backend:

- Generate instantly usable TypeScript types from backend endpoints – enabling full-stack development without build steps.
- Strong static typing with aggressive inference and smart narrowing – so you move fast while staying safe.
- Data schemas acting as guards, automatically validating deeply nested input data through type definitions – to keep untrusted user input out of your application logic.

See also [ROADMAP.md](./ROADMAP.md) for planned work and **feature parity**.

## Why?

We love TypeScript's efficiency in structuring data.

We love Go's efficiency at compile and runtime.

We want the best of both worlds.

We want to be to Go what TypeScript is to JavaScript.

## Examples

Place an order: validate input, attach domain failures, assert invariants with `ensure`, and narrow `Result` values at call sites. The snippets follow one **catalog order** story (`StockKeepingUnit` + `Quantity`, stock check, order id). For more samples, see [`examples/in/`](examples/in/).

### Hello World

*Before (Go)* and *After (Forst)* — Forst is a superset of Go for ordinary programs: the same `package main` source can be built with the standard Go toolchain or compiled with Forst, so a minimal executable looks identical on both sides. Use a **`.ft`** file for the Forst CLI (`forst run`, `forst generate`, …); `go build` expects the usual **`.go`** name if you compile the same text with Go.

```golang
package main

import "fmt"

func main() {
	fmt.Println("Hello World!")
}
```

### Input shape

*Before (Go)* — You declare a request struct with plain `string` and `int` fields, then enforce stock-keeping unit length and quantity range with hand-written `if` checks and generic errors such as `errors.New`. The types themselves do not carry those constraints; validation lives entirely in imperative code.

```golang
type PlaceOrderInput struct {
	StockKeepingUnit string
	Quantity         int
}

func placeOrder(in PlaceOrderInput) (string, error) {
	if len(in.StockKeepingUnit) < 1 || len(in.StockKeepingUnit) > 64 {
		return "", errors.New("invalid stock keeping unit")
	}
	if in.Quantity < 1 || in.Quantity > 99 {
		return "", errors.New("invalid quantity")
	}
	// function body continues...
}
```

*After (Forst)* — With Forst, you put constraints directly on the fields themselves. No need for manual validation as invalid inputs are rejected automatically.

```golang
type PlaceOrderInput = {
	stockKeepingUnit: String.Min(1).Max(64),
	quantity: Int.Min(1).Max(99),
}

func placeOrder(in: PlaceOrderInput) {
	// in is already validated; invariants enforced by typechecker
}
```

### Nominal errors

*Before (Go)* — Each failure kind is a distinct struct type implementing `error`, with `Error()` returning a human-readable string. Callers who need structure use `errors.As`, type switches, or sentinel comparisons; there is no single dedicated syntax for “named” domain errors beyond conventions.

```golang
type UnknownStockKeepingUnit struct{}

func (e UnknownStockKeepingUnit) Error() string { return "unknown stock keeping unit" }

type InsufficientStock struct {
	StockKeepingUnit string
	Requested        int
	Available        int
}

func (e InsufficientStock) Error() string {
	return fmt.Sprintf("insufficient stock for %s: need %d, have %d",
		e.StockKeepingUnit, e.Requested, e.Available)
}
```

*After (Forst)* — `error Name { … }` introduces a nominal failure with an explicit payload shape. The language treats it as a first-class error variant, so you get a dedicated declaration instead of bolting domain meaning onto generic structs and `Error()` strings alone.

```golang
error UnknownStockKeepingUnit {}

error InsufficientStock {
	stockKeepingUnit: String,
	requested:        Int,
	available:        Int,
}
```

### Catalog lookup and stock

*Before (Go)* — Map lookup returns a value plus a boolean: you use `value, ok := catalog[key]` to tell “missing key” from “present but zero”, then branch with `if` statements and construct `error` returns yourself.

```golang
avail, ok := catalog[in.StockKeepingUnit]
if !ok {
	return "", UnknownStockKeepingUnit{}
}
if in.Quantity > avail {
	return "", InsufficientStock{
		StockKeepingUnit: in.StockKeepingUnit,
		Requested:        in.Quantity,
		Available:        avail,
	}
}
```

*After (Forst)* — A **map read** `catalog[key]` has type **`Result(V, Error)`**: a missing key becomes the failure side of that `Result` (generated code uses comma-ok under the hood). You **must** handle failure before using the success value—the usual pattern is **`ensure x is Ok()`**, which narrows `x` to `V`. **Comma-ok assignment (`v, ok := m[k]`) is not supported** as Forst syntax. See [`examples/in/map_catalog.ft`](examples/in/map_catalog.ft) for `ensure avail is Ok()` after a lookup.

```golang
avail := catalog[in.stockKeepingUnit]
ensure avail is Ok()
ensure in.quantity is Max(avail) or InsufficientStock({
	stockKeepingUnit: in.stockKeepingUnit,
	requested:        in.quantity,
	available:        avail,
})
```

`ensure … or …` is not allowed in `func main` (parser rule); put guards that use `or` in a non-`main` function if needed.

### Caller — success path and narrowing

*Before (Go)* — The API returns `(string, error)`. The caller must test `err != nil` before using the order id, then decide how to log, wrap, or branch on concrete error types—standard Go error discipline, with no automatic narrowing of the success value.

```golang
id, err := PlaceOrder(PlaceOrderInput{
	StockKeepingUnit: "ITEM-1",
	Quantity:         2,
})
if err != nil {
	// switch on type, wrap, log, etc.
	return
}
_ = id
```

*After (Forst)* — `placeOrder` yields a `Result`-style value. `ensure x is Ok()` refines `x` to the success payload (here the order id), so the following code can treat it as that value without a separate `if err != nil` block. The call site passes a single anonymous record `{ stockKeepingUnit, quantity }`, usually wrapped by `Mutation.Input`, instead of naming a separate struct type for this invocation.

```golang
x := placeOrder({
	stockKeepingUnit: "ITEM-1",
	quantity:         2,
})
ensure x is Ok() or x
// use success value from x (order id / narrowed payload per compiler)
```

If `placeOrder` accepts a bare record, use `placeOrder({ stockKeepingUnit: "ITEM-1", quantity: 2 })` instead.

### Types across the wire

*Before (TypeScript)* — *You have to define TypeScript interfaces manually to describe the structured data being passed around. These interfaces use broad types like `string` or `number`, which means the input is only type-checked in a very basic way—e.g., every string is accepted, not just valid SKUs or positive quantities. To provide runtime validation (for example, checking `quantity > 0` or `stockKeepingUnit` matches a pattern), you’d need extra schema validators like Zod or Effect.Schema.*

```typescript
interface PlaceOrderInput {
  stockKeepingUnit: string; // any string: not validated further by TypeScript
  quantity: number;         // any number: not automatically checked for positivity, etc.
}
```

*After (TypeScript)* — *With Forst, you define your types in one place. Running `forst generate` outputs unified **generated/types.d.ts** (see [`forst/cmd/forst/generate.go`](./forst/cmd/forst/generate.go)). You just import these. There is no need for extra types and they are available on the consuming code with zero config.*

```typescript
import type { PlaceOrderInput } from "./generated/types";

async function submitOrder(input: PlaceOrderInput) {
  // input is structurally checked, but fields are still unconstrained primitives unless validated
  // For runtime checks, use e.g. Zod: placeOrderInputSchema.parse(input)
  // JSON.stringify(input) — same shapes the server checked in .ft
}
```

Adjust the import path to your output layout. See [TypeScript client output](#typescript-client-output) for `forst generate` and the `generated/` tree.

## Features

- Static typing
- Strong type inference
- Backwards compatibility with Go
- Seamless TypeScript type generation inspired by tRPC – publishing types of API endpoints should be easy
- Structural typing for function parameters and return values
- Type-based assertions that allow complex type narrowing in function parameters
- First-class nominal errors and explicit `ensure` / `Result` control flow—no exceptions; depth and parity are tracked in [ROADMAP.md](./ROADMAP.md)

## Design Philosophy

See also [PHILOSOPHY.md](./PHILOSOPHY.md) for what guides and motivates us.

## Development

Install Task using the [official instructions](https://taskfile.dev/installation/).

Run tests:

```bash
task test                  # Go tests: forst/internal/... and forst/cmd/forst/...
task ci:test               # Full CI suite (Go tests + VS Code extension build + @forst/cli & @forst/sidecar + examples)
task test:unit             # Run compiler unit tests (internal only)
task test:unit:parser      # Run parser tests
task test:unit:typechecker # Run typechecker tests
```

Run examples:

```bash
task test:integration                    # Run compilation examples / integration tests
task example -- ../examples/in/basic.ft  # Run specific example
task example:function                    # Run function example
```

## VS Code

The workspace includes an optional extension in [`packages/vscode-forst`](./packages/vscode-forst): it registers `.ft` and talks to the compiler’s HTTP LSP (`forst lsp`) for diagnostics. Its **release cadence is separate** from compiler `v*` tags (see `vscode-forst-v*` in [`.github/workflows/publish-vscode-extension.yml`](./.github/workflows/publish-vscode-extension.yml)). After `bun install` at the repo root, run `task build:vscode` to compile it (or rely on the F5 **preLaunchTask** in [`.vscode/launch.json`](./.vscode/launch.json)). CI runs the same compile as the first step of `task ci:test`. See [`packages/vscode-forst/README.md`](./packages/vscode-forst/README.md) for F5 and troubleshooting.

## npm

**[`@forst/cli`](./packages/cli/README.md)** installs the Forst compiler in JS/TS projects: `npm i -D @forst/cli`, then `npx forst` / `node_modules/.bin/forst` (it pulls the matching native binary from GitHub Releases). For the dev-server + HTTP client, use **[`@forst/sidecar`](./packages/sidecar/README.md)** instead.

## TypeScript client output

You can generate **TypeScript types and a small client** from your Forst code so front ends or Node callers get the same shapes your server uses, without copying types by hand.

Run `forst generate` with a `.ft` file or a folder of `.ft` files; it writes a `generated/` tree (declarations plus helpers) and a `client/` stub you can wire to your app. The dev server can also expose types over HTTP while you iterate.

## Inspirations

Our primary inspiration is TypeScript's structural type system and its enormous success in making JavaScript development more ergonomic, robust and gradually typeable. We aim to bring similar benefits to Go development, insofar as they are not already present.

We also draw inspiration from:

- **Zod** — constraints and shape guards as composable runtime checks on nested data.
- **tRPC** — one source of truth for API shapes, with **TypeScript types and a small client** generated from Forst (`forst generate`, `examples/client-integration/`).
- **Go** and **Rust** — **errors as values** and explicit control flow (`ensure` … `or …`), not exceptions.