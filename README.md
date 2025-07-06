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

See also [ROADMAP.md](./ROADMAP.md) for details on what we are planning.

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

## Inspirations

Our primary inspiration is TypeScript's structural type system and its enormous success in making JavaScript development more ergonomic, robust and gradually typeable. We aim to bring similar benefits to Go development, insofar as they are not already present.

We also draw inspiration from:

- Rust's approach to error handling and type safety
- tRPC's type-safe API design
- Zod's runtime type validation and schema composition
