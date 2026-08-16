# Forst &emsp; [![CI]][actions] [![Release]][release] [![Coverage Status]][coveralls] [![License]][license]

[CI]: https://img.shields.io/github/actions/workflow/status/forst-lang/forst/lint-test-coverage.yml
[actions]: https://github.com/forst-lang/forst/actions
[release]: https://img.shields.io/github/v/release/forst-lang/forst?filter=v*
[Coverage Status]: https://coveralls.io/repos/github/forst-lang/forst/badge.svg?branch=main
[coveralls]: https://coveralls.io/github/forst-lang/forst?branch=main
[License]: https://img.shields.io/github/license/forst-lang/forst

**A programming language that helps you move TypeScript backends to Golang.**

To accomplish this, Forst supports you in four key ways:

| Benefit | How it works |
| --- | --- |
| Go ecosystem | Compiles directly to Go. You get full access to the Go package ecosystem, standard library and build tools. |
| TypeScript integration | Generates native TS definitions and a client directly from your backend source. No need for intermediate languages like GraphQL or Protobuf just for type safety. |
| Node.js interop | Call existing Node.js code directly from Forst to keep your existing tools and libraries during migration. |
| Incremental migration | Allows you to move small parts of your code at a time while your existing codebase keeps running, so you can keep shipping during migration. |

## Why?

Building backend APIs often forces a trade off between runtime performance and developer speed. Maintaining separate schema contracts leads to lots of glue code. Full backend rewrites into more performant languages are often impossible to do safely.

Forst exists to remove that friction. You can migrate incrementally or write completely new backends with native performance and an ergonomic DX by design.

## Examples

Forst code builds on Go foundations. Here is how a handler evolves from a basic executable to full interop and type generation.

### Hello World

Forst files use the `.ft` extension and support standard Go source structure.

```golang
package main

import "fmt"

func main() {
	fmt.Println("Hello World!")
}
```

### Validation and nominal errors

Constraints on types validate boundary data automatically. You declare structured domain errors with `error` and check invariants with `ensure`.

```golang
type PlaceOrderInput = {
	stockKeepingUnit: String.Min(1).Max(64),
	quantity:         Int.Min(1).Max(99),
}

error InsufficientStock {
	requested: Int,
	available: Int,
}

func PlaceOrder(in: PlaceOrderInput, available: Int) Result(String, Error) {
	ensure in.quantity is Max(available) or InsufficientStock({
		requested: in.quantity,
		available: available,
	})
	return Ok("ORDER-123")
}
```

### Go interop

Call standard Go packages or existing Go functions directly in Forst source.

```golang
import "fmt"

func PlaceOrder(in: PlaceOrderInput, available: Int) Result(String, Error) {
	ensure in.quantity is Max(available) or InsufficientStock({
		requested: in.quantity,
		available: available,
	})
	orderID := fmt.Sprintf("ORDER-%d", 123)
	return Ok(orderID)
}
```

### Node.js interop

Import existing JavaScript or TypeScript modules into Forst with `import node` during migration.

```golang
import node "./legacy/payment"

func PlaceOrder(in: PlaceOrderInput, available: Int) Result(String, Error) {
	ensure in.quantity is Max(available) or InsufficientStock({
		requested: in.quantity,
		available: available,
	})
	receipt := payment.Charge({ amount: in.quantity * 10 })
	return Ok(receipt.id)
}
```

### TypeScript client generation

Run `forst generate` to create typed client packages for your frontend. Callers receive end to end type safety without writing schema glue code.

```typescript
import { $orders } from "@forst/gen/orders";

const order = await $orders.PlaceOrder({
  stockKeepingUnit: "SKU-99",
  quantity: 2,
});
```

### Effect TS interop

Enable Effect mode in `ftconfig.json` to emit native Effect services and layers.

```typescript
import { Effect } from "effect";
import { $orders } from "@forst/gen/orders";

const program = Effect.gen(function* () {
  const order = yield* $orders.PlaceOrder({
    stockKeepingUnit: "SKU-99",
    quantity: 2,
  });
  return order;
}).pipe(Effect.provide($orders.Default));

await Effect.runPromise(program);
```

## Features

Highlights from the [docs feature comparison](docs/why.mdx).

### Language

| Capability | Forst |
| --- | --- |
| Structural typing | Built-in records, signatures, and `is` narrowing |
| Validation on types | Built-in field constraints; boundary runtime checks |
| Error handling | `ensure`, nominal `error` types, and `Result`; no exceptions |
| Mocking and DI | `use` / `with` providers; no external DI framework |
| Type narrowing | `is` / `ensure` and type guards |
| Goroutines | Native `go` and `defer` via Go output |

### Interop and adoption

| Capability | Forst |
| --- | --- |
| Go module ecosystem | Import Go packages natively |
| JS / npm ecosystem | Call legacy JS/TS via `import node` |
| Shared server ↔ client types | `forst generate` from the same `.ft` source |
| Call backend from Node | Generated client and HTTP invoke |
| Incremental migration | Mix `.ft`, `.go`, and legacy code in one codebase |

See [ROADMAP.md](./ROADMAP.md) for experimental features and planned work.

## Design Philosophy

See also [PHILOSOPHY.md](./PHILOSOPHY.md) for what guides and motivates us.

## Install and tooling

### npm packages

| Package | Purpose |
| --- | --- |
| [`@forst/cli`](./packages/cli/README.md) | Forst compiler in JS/TS projects |
| [`@forst/sidecar`](./packages/sidecar/README.md) | Dev server and HTTP client during migration |

Install the compiler in a Node project:

```bash
npm i -D @forst/cli
npx forst version
```

`@forst/cli` pulls the matching native binary from GitHub Releases.

### VS Code extension

Optional extension in [`packages/vscode-forst`](./packages/vscode-forst).

- Registers `.ft` files in the editor
- Diagnostics via the compiler HTTP LSP (`forst lsp`)

See [`packages/vscode-forst/README.md`](./packages/vscode-forst/README.md) for installation and troubleshooting.

### Linux (.deb)

On Debian or Ubuntu, install the compiler from [GitHub Releases](https://github.com/forst-lang/forst/releases) (pick `amd64` or `arm64`):

```bash
wget https://github.com/forst-lang/forst/releases/download/vX.Y.Z/forst_X.Y.Z-1_amd64.deb
sudo apt install ./forst_X.Y.Z-1_amd64.deb
forst version
```

See [docs/installation.mdx](docs/installation.mdx) for other install paths (npm, native binary, Docker).

## TypeScript client output

You can generate **TypeScript types and a small client** from your Forst code so front ends or Node callers get the same shapes your server uses, without copying types by hand.

Run `forst generate` with a `.ft` file or a folder of `.ft` files; it writes a `generated/` tree (declarations plus helpers) and a `client/` stub you can wire to your app. The dev server can also expose types over HTTP while you iterate.

## Inspirations

Our primary inspiration is TypeScript's structural type system and its enormous success in making JavaScript development more ergonomic, robust and gradually typeable. We aim to bring similar benefits to Go development, insofar as they are not already present.

We also draw inspiration from:

- **Zod** — constraints and shape guards as composable runtime checks on nested data.
- **tRPC** — one source of truth for API shapes, with **TypeScript types and a small client** generated from Forst (`forst generate`, `examples/client-integration/`).
- **Go** and **Rust** — **errors as values** and explicit control flow (`ensure` … `or …`) instead of exceptions.