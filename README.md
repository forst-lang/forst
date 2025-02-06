# crysma

Crysma is a programming language for backend development and streamlined client-server communication. It has primary goal of being the reason people choose not to use TypeScript on the backend:

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
- Seamless TypeScript type generation inspired by tRPC
- Structural typing for function parameters and return values
- First-class assertions that allow complex type narrowing
- First-class scoped errors including stack traces
- No class or module reopening

## Our Design Philosophy

We do not want to use TypeScript provide a tRPC like seamless full-stack development experience. This language was heavily inspired by Crystal.
We draw extraordinary inspiration from Rust.

## Ideas

- Compile down to Go
- Types have functions that define constraints and narrow them down
- All types are capital case or anonymous struct types
- All nominal types are aliases of constraints on other types
  - e.g. `Array(String)` is an anonymous, non-nominal type
  - but `type StringArray = Array(String)` is a nominal type
- Traits from Rust instead of classes? Allow any objects that satisfy the trait to be used in place of the trait.
  - Disallow methods in parameter object types
- Pattern matching from Rust / match statements
- Don't fall through in switch statements
- Make function signatures GraphQL compatible as well

## Hello World

```cr
type Input = {
  name: String.Min(3).Max(10),
}

class Routers::UserRouter
  fn users
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

  fn self.trpc
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
