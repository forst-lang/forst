# Forst

[![CI](https://github.com/forst-lang/forst/actions/workflows/lint-test-coverage.yml/badge.svg)](https://github.com/forst-lang/forst/actions) [![Go Report Card](https://goreportcard.com/badge/github.com/forst-lang/forst)](https://goreportcard.com/report/github.com/forst-lang/forst)
[![Coverage Status](https://coveralls.io/repos/github/forst-lang/forst/badge.svg?branch=main)](https://coveralls.io/github/forst-lang/forst?branch=main) [![License](https://img.shields.io/github/license/forst-lang/forst)](https://github.com/forst-lang/forst/blob/main/LICENSE)

Forst is a programming language for backend development.

Its primary goal is to make it easy to stop using TypeScript on the backend:

- Support inferring TypeScript types from backend endpoints.
- Value types for primitive data to avoid memory heavy heap allocations.
- Statically type-checked, but inferring and narrowing types as much as possible.
- Allow nested type definitions to validate and narrow down input data.

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

func createUser(op: trpc.Mutation.Input({
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

## Inspirations

Our primary inspiration is TypeScript's structural type system and its success in making JavaScript development more ergonomical and robust. We aim to bring similar benefits to Go development.

We also draw inspiration from:

- Go's simplicity and performance
- Rust's approach to error handling and type safety
- tRPC's type-safe API design
- Zod's runtime type validation and schema composition
