# 1. Design Philosophy

## Core Mission

- Replace TypeScript as the go-to language for type-safe backend development
- Take inspiration from TypeScript's best features while maintaining our own identity
- Focus on practical features that solve real development problems

## Adoption

We want maximum chances for adoption, given that our main goal is to replace TypeScript on the backend.

So we want to rely on a large existing ecosystem of libraries, and we also want the language to compile quickly to allow for fast iteration. This essentially only leaves Go as a compilation target at the moment.

So our language will transpile both to Go (for the backend code) and to TypeScript (for client types).

## Language Design

- Embrace structural typing - where types are defined by their shape and structure
- Enable robust type checking and validation at both compile and runtime
- Provide clear, actionable development feedback through type information
- Focus on predictable, deterministic behavior
- Maintain strong type safety without excessive verbosity
- Keep the language spec concise and unambiguous

## Interoperability

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

## Language Features

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

## Anti-Features

### Anti-Features (Things We Won't Do)

#### No unpredictable control flow

While we don't forbid features like panics or goto statements, we prioritize language constructs that encourage predictable execution paths. Our goal is to have the compiler and type system guide developers toward explicit error handling (using Go's standard error return values) and structured control flow that is easy to reason about, debug, and maintain.

#### No implicit type coercion

We reject implicit type coercion. It is a dangerous pattern that can lead to data corruption and hard-to-find bugs. Backend systems need strong type safety to prevent unintended type conversions that could compromise data integrity.

#### No inheritance-based OOP

We choose not to support inheritance-based OOP. Experience has shown that deep inheritance hierarchies often become difficult to maintain as codebases grow. Composition provides the flexibility we need while avoiding the common pitfalls of inheritance.

#### No side effects in type system

Side effects in the type system add unnecessary complexity that we want to avoid. They can make type checking behavior unpredictable and debugging more difficult. Our type system focuses on clear, straightforward validation of data shapes without hidden interactions between types.
