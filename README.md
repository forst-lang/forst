# Forst &emsp; [![CI]][actions] [![Release]][release] [![Go Report Card]][goreport] [![Coverage Status]][coveralls] [![License]][license]

[CI]: https://img.shields.io/github/actions/workflow/status/forst-lang/forst/lint-test-coverage.yml
[actions]: https://github.com/forst-lang/forst/actions
[release]: https://img.shields.io/github/v/release/forst-lang/forst
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

See also [ROADMAP.md](./ROADMAP.md) for planned work and **feature parity** (tables by theme).

## Why?

We love TypeScript's efficiency in structuring data.

We love Go's efficiency at compile and runtime.

We want the best of both worlds.

We want to be to Go what TypeScript is to JavaScript.

## Features

- Static typing
- Strong type inference
- Backwards compatibility with Go
- Seamless TypeScript type generation inspired by tRPC – publishing types of API endpoints should be easy
- Structural typing for function parameters and return values
- Type-based assertions that allow complex type narrowing in function parameters
- First-class scoped errors including stack traces, avoiding exceptions
- No class or module reopening

## Design Philosophy

See also [PHILOSOPHY.md](./PHILOSOPHY.md) for what guides and motivates us.

## Examples

### Hello World

Forst aims to be backwards-compatible with Go:

```go
package main
import fmt "fmt"

func main() {
  fmt.Println("Hello World!")
}
```

### Basic Function

```go
func diff(a: Int, b: Int) Bool {
  return 3 - 2
}
```

Specifying the return type is optional. It will be inferred automatically.

### Input Validation

Define types and validate inputs both at compile time and at runtime.

```go
type PhoneNumber =
  String.Min(3).Max(10) & (
    String.HasPrefix("+")
    | String.HasPrefix("0")
  )

func createUser(op: Mutation.Input({
  id: UUID.V4(),
  name: String.Min(3).Max(10),
  phoneNumber: PhoneNumber,
  bankAccount: {
    iban: String.Min(10).Max(34),
  },
})) {
  fmt.Println("Creating user with id: %s", op.input.id)
  return 300.3f
}
```

## Development

Install Task using the [official instructions](https://taskfile.dev/installation/).

Run tests:

```bash
task test                  # Run all tests
task test:unit             # Run compiler unit tests
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

The workspace includes an optional extension in [`packages/vscode-forst`](./packages/vscode-forst): it registers `.ft` and talks to the compiler’s HTTP LSP (`forst lsp`) for diagnostics. After `bun install` at the repo root, run `task build:vscode` to compile it (or rely on the F5 **preLaunchTask** in [`.vscode/launch.json`](./.vscode/launch.json)). CI runs the same compile as the first step of `task ci:test`. See [`packages/vscode-forst/README.md`](./packages/vscode-forst/README.md) for F5 and troubleshooting.

## TypeScript client output

You can generate **TypeScript types and a small client** from your Forst code so front ends or Node callers get the same shapes your server uses, without copying types by hand.

Run `forst generate` with a `.ft` file or a folder of `.ft` files; it writes a `generated/` tree (declarations plus helpers) and a `client/` stub you can wire to your app. The dev server can also expose types over HTTP while you iterate.

## Inspirations

Our primary inspiration is TypeScript's structural type system and its enormous success in making JavaScript development more ergonomic, robust and gradually typeable. We aim to bring similar benefits to Go development, insofar as they are not already present.

We also draw inspiration from:

- **Zod** — constraints and shape guards as composable runtime checks on nested data.
- **tRPC** — one source of truth for API shapes, with **TypeScript types and a small client** generated from Forst (`forst generate`, `examples/client-integration/`).
- **Go** and **Rust** — **errors as values** and explicit control flow (`ensure` … `or …`), not exceptions.