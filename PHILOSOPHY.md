# Design Philosophy

## Core Mission

- Replace TypeScript as the go-to language for type-safe backend development
- Take inspiration from TypeScript's best features while maintaining our own identity
- Focus on practical features that solve real development problems

## Guiding Principles

- Embrace structural typing - where types are defined by their shape and structure
- Enable robust type checking and validation at both compile and runtime
- Provide clear, actionable development feedback through type information
- Focus on predictable, deterministic behavior
- Maintain strong type safety without excessive verbosity
- Keep the language spec concise and unambiguous
- We don't mind giving the compiler hints to make our code more reliable

## Goals

### Adoption

We want maximum chances for adoption, given that our main goal is to replace TypeScript on the backend.

So we want to rely on a large existing ecosystem of libraries, and we also want the language to compile quickly to allow for fast iteration. This essentially only leaves Go as a compilation target at the moment.

So our language will transpile both to Go (for the backend code) and to TypeScript (for client types).

### Interoperability

All APIs are meant for consumption.

- So we should make it easy to provide types that help users interact with them safely.

We acknowledge that TypeScript is the de facto standard for frontend development.

- So we should focus on interoperability with TypeScript clients.

### Go Integration

- Generate clean, idiomatic Go code that's easy to understand and maintain
- Support direct use of Go packages and types with minimal overhead
- Work seamlessly with existing Go tools, testing and deployment

### Frontend Integration

- Generate accurate TypeScript definitions from backend types
- Ensure type safety across API boundaries
- Minimize friction in full-stack development workflows

## Anti-Features

These are patterns that we deliberately choose not to support in the language because they would work against our core principles and goals.

## No unpredictable behavior

In order to be reliable, a system must be predictable.

While we don't forbid features like panics or goto statements, we prioritize language constructs that encourage predictable execution paths. Our goal is to have the compiler and type system guide developers toward explicit error handling (using Go's standard error return values) and structured control flow that is easy to reason about, debug, and maintain.

We do not like implicit control flows that require going on wild goose chases to find the right execution path.

- So we do not support macros, and when control flow is changed, we require this to be done using English keywords.

### No surprising errors

We want to be able to trace where a runtime error originates. But we don't want to be surprised by an exception, or even to handle all possible errors at every level of the call stack.

- So errors should need to either be handled or explicitly ignored.

We love type inference and type narrowing, but we accept that we need to give up some syntax sugar in return.

- So we don't mind giving hints to the compiler to help it work, including the use of extra syntax to signal intent.

## No metaprogramming

### No complex type inference

We want to be compiler-friendly. If it is hard for the compiler to infer types, then it is also hard for the programmer to reason about them. It is a doubly dangerous game to play.

We allow type inference in specific, well-defined cases:

- Function return types can be inferred when the implementation has a clear, unambiguous return type
- Variable types can be inferred from their initialization expression when there is a single, obvious type
- Generic type parameters can be inferred at call sites when argument types uniquely determine the parameters

We require explicit type annotations in all other cases, particularly:

- Function parameters must always declare their types
- Struct fields must always declare their types
- Interface method signatures must be fully typed
- Type aliases and custom types must declare their underlying type
- Generic type constraints must be explicitly specified
- Variables initialized as nil must specify their type

When in doubt, we require being explicit with types.

### No needless object orientation

We like the idea of simple value objects that have functions naturally associated with them. We also think that hiding data structure specific implementation details makes sense. This is where OOP shines.

- So we want to allow attaching methods to nominal types, as well as making properties and methods private.

But it is simply not feasible for all procedures to have a single receiver. In practice, they often involve multiple objects, collections, and other data structures working together in complex ways.

- So we want to offer parameters that allow passing arbitrary and nested data structures to a procedure.

### No side effects in type system

Side effects in the type system add unnecessary complexity that we want to avoid. They can make type checking behavior unpredictable and debugging more difficult. Our type system focuses on clear, straightforward validation of data shapes without hidden interactions between types.

#### No implicit type conversions

While other languages may silently convert numbers between int and float, we require explicit conversion to make type changes visible.

We reject implicit type coercion. It is a dangerous pattern that can lead to data corruption and hard-to-find bugs. Backend systems need strong type safety to prevent unintended type conversions that could compromise data integrity.

#### No dependent Types

Some languages allow types to depend on runtime values, but we keep type checking separate from runtime behavior.

#### No type-level computation

No template metaprogramming like in C++, no type families or type-level functions. While we support type assertions and constraints for type checking and inference, we do not allow arbitrary functions or computations at the type level. Types are validated during compilation but not "executed" like programs.

#### No runtime type modifications

No monkey patching or extending types at runtime - type definitions are fixed at compile time.

This keeps our type system predictable and easier to reason about, both for developers and for tooling.
