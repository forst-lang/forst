# Forst

Forst is a programming language for backend development.

[![Go Report Card](https://goreportcard.com/badge/github.com/forst-lang/forst)](https://goreportcard.com/report/github.com/forst-lang/forst)
[![License](https://img.shields.io/github/license/forst-lang/forst)](https://github.com/forst-lang/forst/blob/main/LICENSE)
[![GitHub stars](https://img.shields.io/github/stars/forst-lang/forst)](https://github.com/forst-lang/forst/stargazers)
[![GitHub issues](https://img.shields.io/github/issues/forst-lang/forst)](https://github.com/forst-lang/forst/issues)
[![CI](https://github.com/forst-lang/forst/actions/workflows/lint-test-coverage.yml/badge.svg)](https://github.com/forst-lang/forst/actions)

Its primary goal is to make it easy to stop using TypeScript on the backend:

- Support inferring TypeScript types from backend endpoints.
- Value types for primitive data to avoid memory heavy heap allocations.
- Statically type-checked, but inferring and narrowing types as much as possible.
- Allow nested type definitions to validate and narrow down input data.

See also [ROADMAP.md](./ROADMAP.md) for details on what we are planning.

## Why?

We love Ruby's efficiency in expressing logic.

We love TypeScript's efficiency in structuring data.

We love Go's efficiency at compile and runtime.

We want the best of all three worlds.

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

## Our Design Philosophy

### Interoperability

All APIs are meant for consumption.

- So we should make it easy to provide types that help users interact with them safely.

We acknowledge that TypeScript is the de facto standard for frontend development.

- So we should focus on interoperability with TypeScript clients.

### Adoption

We want maximum chances for adoption, given that our main goal is to replace TypeScript on the backend.

So we want to rely on a large existing ecosystem of libraries, and we also want the language to compile quickly to allow for fast iteration. This essentially only leaves Go as a compilation target at the moment.

So our language will transpile both to Go (for the backend code) and to TypeScript (for client types).

### OOP vs. Functional Programming

We like the idea of simple value objects that have functions naturally associated with them. We also think that hiding data structure specific implementation details makes sense. This is where OOP shines.

- So we want to allow attaching methods to nominal types, as well as making properties and methods private.

But it is simply not feasible for all procedures to have a single receiver. In practice, they often involve multiple objects, collections, and other data structures working together in complex ways.

- So we want to offer parameters that allow passing arbitrary and nested data structures to a procedure.

It is also not necessary for types to inherit code from other types.

- So we do not support inheritance, and we encourage composition of functions instead.

### Metaprogramming

We want to be compiler-friendly. If it is hard for the compiler to infer types, then it is also hard for the programmer to reason about them. It is a doubly dangerous game to play.

- So we want to avoid naive metaprogramming like re-opening classes that requires dynamic method resolution.

We do not like implicit control flows that require going on wild goose chases to find the right execution path.

- So we do not support macros, and when control flow is changed, we require this to be done using English keywords.

### Error Handling

We want to be able to trace where a runtime error originates. But we don't want to be surprised by an exception, or even to handle all possible errors at every level of the call stack.

- So errors should need to either be handled or explicitly ignored.

We love type inference and type narrowing, but we accept that we need to give up some syntax sugar in return.

- So don't mind giving hints to the compiler to help it work, including the use of extra syntax to signal intent.

### Inspirations

This language was heavily inspired by Crystal. We also draw some inspiration from Rust.

## Hello World

Our "Hello World" program works exactly like in Ruby and in Crystal:

```cr
def main
  puts "Hello, World!"
end
```

## In Practice

```cr
type Input = {
  name: String.Min(3).Max(10),
}

class Routers::UserRouter
  def users
    DB::Client.pool.query do |q|
      q.selectFrom(User)
        .select([
          User.id,
          User.name,
        ])
        .where(User.deletedAt.null())
        .execute()
    end
  end
end

module Routers
  @@trpc = TRPC::Router.from_routes({
    user: UserRouter.new,
  })

  def self.trpc
    @@trpc
  end
end

def handle_http_request(req: HTTP::Request)
  Routers.trpc.handle(req)
end
```

`select` above will fail when given wrong types. Its argument type is `Array(User.Column)`. The precise typing of SQL queries is still to be defined, depending on how we want the query builder to work.

```cr
type PhoneNumber = String.Min(10).Max(15)

class Routers::UserRouter
  def createUser!(input: {
    id: UUID,
    name: String.Min(3).Max(10),
    phoneNumbers: Array(PhoneNumber),
  })
    puts "Creating user with id: #{input.id}"
  end
end
```

Metaprogramming such as macros is impossible, and re-opening classes is not allowed:

```cr
class A; end
class A; end # Syntax error: Class A is already defined
```

This will fail to compile, which should make it easier for the compiler to infer types.
