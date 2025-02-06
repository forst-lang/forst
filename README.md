# Crysma

Crysma is a programming language for backend development.

Its primary goal is to make it easy to stop using TypeScript on the backend:

- Support inferring TypeScript types from backend endpoints.
- Value types for primitive data to avoid memory heavy heap allocations.
- Statically type-checked, but inferring and narrowing types as much as possible.
- Allow nested type definitions to validate and narrow down input data.

## Why?

We love Ruby's efficiency in expressing logic.

We love TypeScript's efficiency in structuring data.

We love Go's efficiency in compiler and runtime performance.

We want the best of all three worlds.

We want to be for TypeScript what Crystal is to Ruby.

## Features

- Static typing
- Strong type inference
- Seamless TypeScript type generation inspired by tRPC – making types of API endpoints available should be the default
- Structural typing for function parameters and return values
- Type-based assertions that allow complex type narrowing
- First-class scoped errors including stack traces, avoiding exceptions
- No class or module reopening

## Our Design Philosophy

### Interoperability

All APIs are meant for consumption.

- So we should make it easy to provide types that help users interact with them safely.

We acknowledge that TypeScript is the de facto standard for frontend development.

- So we should focus on interoperability with TypeScript clients.

### OOP vs. Functional Programming

We like the idea of simple value types that have functions. This is where OOP shines.

- So we want to offer the ability to connect types with methods.

But it is simply not feasible for all procedures to have a single receiver. In practice, they often involve multiple objects, collections, and other data structures working together in complex ways.

- So we want to offer parameters that allow passing arbitrary and nested data structures to a procedure.

### Metaprogramming

We want to be compiler-friendly. If it is hard for the compiler to infer types, then it is also hard for the programmer to reason about them. It is a doubly dangerous game to play.

- So we want to avoid naive metaprogramming like re-opening classes that requires dynamic method resolution.

We do not like implicit control flows that require going on wild goose chases to find the right execution path.

- So we do not support macros, and when control flow is changed, we require this to be done using English keywords.

### Error Handling

We want to be able to trace where a runtime error originates. But we don't want to be surprised by an exception, or even to handle all possible errors at every level of the call stack.

- So errors should need to either be handled or explicitly ignored.

We love type inference and type narrowing, but we accept that we need to give up some syntax sugar in return.

- We don't mind giving hints to the compiler to help it work, including the use of extra keywords to signal intent.

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

fn handle_http_request(req: HTTP::Request)
  Routers.trpc.handle(req)
end
```

`select` above will fail when given wrong types. Its argument type is `Array(User.Column)`. The precise typing of SQL queries is still to be defined, depending on how we want the query builder to work.

```cr
type PhoneNumber = String.Min(10).Max(15)

class Routers::UserRouter
  fn createUser!(input: {
    id: UUID,
    name: String.Min(3).Max(10),
    phoneNumbers: Array(PhoneNumber),
  })
    log("Creating user with id: #{input.id}")
  end
end
```

Metaprogramming such as macros is impossible, and re-opening classes is not allowed:

```cr
class A; end
class A; end # Syntax error: Class A has been closed
```

This will fail to compile, which should make it easier for the compiler to infer types.

Traits from Rust instead of classes? Allow any objects that satisfy the trait to be used in place of the trait.
