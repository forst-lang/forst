---
Feature Name: type-guards
Start Date: 2025-05-28
---

# Type Guards

## Summary

Introduce first-class type guards as a language feature to enable type narrowing without requiring type assertions or branded types. Type guards will allow developers to define predicates that narrow types at compile time, making type-safe code more ergonomic and maintainable.

This RFC defines the semantics, constraints, and compiler behavior for type guards in Forst. It ensures that type guards are:

- Predictable and composable
- Efficient to analyze statically
- Safe for use in type narrowing, Go code generation, and `.d.ts` emission

## Motivation

Currently, Forst provides built-in `ensure` statements with an `is` operator for runtime validation and error handling.

### Status Quo

The `ensure` keyword combined with the `is` operator is used to perform type checks that are run both at compile time and at runtime. The basic syntax is:

```go
func MaybeReadInt(str: String) {
    x, err := strconv.Atoi(str)
    ensure x is Int() or err
}
```

This ensures that `x` is of type `Int` before proceeding, or returns the error if the type check fails. The compiler can then guarantee type safety for `x` in the rest of the function. The `ensure` statement automatically makes the function return a tuple of `(Int, error)` when the type check fails.

Such assertions can also be used to enforce more complex and parametrized validation rules and throw custom errors. When transpiled to Go, these assertions shall eventually become well-structured error types that implement the `error` interface, with proper error wrapping using `fmt.Errorf` and `errors.Is`/`errors.As` for error handling.

```go
func mustNotExceedSpeedLimit(speed: Int) {
    ensure speed is LessThan(100)
        or TooFast("Speed must not exceed 100 km/h")
}
```

### Desired State

However, Forst currently lacks a way for users to define their own assertions like `LessThan`. The `is` operator only works with built-in assertions, requiring developers to write custom validation functions that don't integrate with the type system.

As a direct comparison, both TypeScript and Go require complex workarounds for type narrowing, each with their own drawbacks:

TypeScript's approaches include:

1. Type assertions (e.g. `value as Type`) that bypass type checking and can be unsafe
2. Branded types that require artificial type discriminators
3. Complex intersection types that don't match runtime structures
4. User-defined type guards with `is` keyword that are verbose to implement
5. Inheritance hierarchies that can be inflexible and lead to deep class trees

Go's approaches include:

1. Type assertions (e.g. `value.(Type)`) that can panic at runtime
2. Type switches that are verbose and require duplicated logic
3. Interface satisfaction checks that don't provide compile-time type narrowing
4. Custom validation functions that can't affect the type system
5. Embedding types which provides inheritance-like behavior but lacks true subtyping

While inheritance can help model some type relationships, it often leads to rigid hierarchies that don't adapt well to changing requirements. These workarounds in both languages make code harder to read and maintain, while potentially introducing runtime overhead or safety issues. By making type guards a first-class feature, we can provide a more elegant and type-safe solution that works naturally with the type system.

## Definitions

**Type Guard**: A pure predicate function with signature `is (x T) Guard(...) -> Bool`, used with the `ensure` keyword to narrow the type of `x` at runtime.

**Base Type**: In the above scenario, the base type of `Guard` is `T`. We can write `T.Guard(...)` to refer to a subtype of `T` where the type guard predicate holds.

**Shape**: A structural object defined via `Shape({ ... })`, used for validating and refining types.

**Refinement**: A narrowing of a type, e.g., `x is Shape({ ... })`, which adds constraints or fields to the original shape.

**Ensure**: A statement that validates a condition at runtime and narrows types in the success branch, e.g. `ensure x is Guard() or err`.

## Guide-level explanation

Type guards are pure (side-effect free) predicates that narrow types at compile time. They are defined using a top-level `is` keyword and can be used to refine types in conditional blocks. Here's a simple example:

```go
type Password = string

is (password: Password) Strong {
    return len(password) >= 12
}

func validatePassword(password: Password) {
    if password is Strong() {
        // password is now narrowed to Password.Strong()
        // Can safely use it in contexts requiring strong passwords
    }
    // More verbose syntax including the base type:
    if password is Password.Strong() {
        // ...
    }
}
```

### Type Guard Composition

> **Warning**
> This section is a work in progress and subject to change. The examples and details provided here are for illustration purposes and may not reflect the final implementation.

Type guards can be composed to create more complex predicates:

```go
is (password: Password) HasUppercase {
    return strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
}

is (password: Password) HasNumber {
    return strings.ContainsAny(password, "0123456789")
}

is (password: Password) VeryStrong {
    return password is Strong() &&
           password is HasUppercase() &&
           password is HasNumber()
}
```

### Generic Type Guards

> **Warning**
> This section is a work in progress and subject to change. The examples and details provided here are for illustration purposes and may not reflect the final implementation.

Type guards work with generic types:

```go
type Result<T> = T | Error

is <T>(result: Result<T>) Success {
    return result is T
}

func <T> unwrap(result: Result<T>): T {
    if result is Success() {
        return result // Type is narrowed to T
    }
    panic("Expected success")
}
```

### Module Boundaries

> **Warning**
> This section is a work in progress and subject to change. The examples and details provided here are for illustration purposes and may not reflect the final implementation.

Type guards can be exported and imported like other declarations:

```go
// internal/auth/guards.ft
is (password: Password) Strong {
    return len(password) >= 12
}

// main.ft
import "internal/auth/guards"

func validateUser(password: Password) {
    if password is guards.Strong() {
        // ...
    }
}
```

## Type Guard Rules

> **Warning**
> This section is a work in progress and subject to change. The examples and rules reflect the current Forst specification and may evolve as the language matures.

### TG-1: Only `if` / `else if` / `else` Blocks Are Allowed

> Type guards do not return values. Instead, they must use `if`, `else if`, and `else` branches only.

- Type guards must not contain `return` statements.
- All branching must be explicit using `if`, `else if`, and `else`.
- Each condition must be an `is` assertion.
- `ensure` may be used within a branch to refine the shape.

### TG-2: Conditions Must Be `is` Assertions

> `if` and `else if` conditions inside type guards must use the `is` operator with valid type assertions.

- Valid: `if x.role is String.Equals("admin")`
- Invalid: `if isAdmin(x)`
- Invalid: `if x.role == "admin"`

### TG-3: Guards May Only Use `ensure` for Refinement

> Inside a type guard, the only mechanism for refinement or validation is `ensure`.
> The `ensure` must refine or preserve the type of the receiver.

- `ensure` must only be used for type assertions.
- `or` clauses are not allowed inside `ensure` in type guards.
- Refinement must never weaken or remove any existing fields from the shape.

### TG-4: Recursive Refinement Applies to Sub-Shapes

> Any refinement performed within a shape guard must apply recursively to all sub-shapes.

- Refinements on nested shapes must also preserve or narrow existing fields.
- You may not destructure and redefine sub-shapes.
- Every nested assertion must follow the same rules as top-level shape assertions.

### TG-5: All Branches Must Be Statically Analyzable

> Type guards must be analyzable by the compiler in polynomial time.

- Branching must be finite and predictable.
- Guards that generate combinatorial or deeply recursive shape paths will be rejected.
- Each path through the guard must be semantically clear and fast to evaluate.

### TG-6: Shape Syntax Must Use Bare `{}`

> Shape expressions must use bare braces. The `Shape` keyword is restricted.

- Use `{ field: Type }` for inline shape refinements.
- The `Shape` keyword is only allowed:
  - In guard parameter declarations: `is (g GuardedType) Guarded(s Shape) { <block> }`
  - In type definitions: `type User Shape`
- Do not use `Shape({...})` syntax anywhere else.

### TG-7: No External Identifiers

> Type guards must not reference any identifiers outside the receiver or guard parameters.

- You may only use:
  - Fields of the guard receiver
  - Parameters explicitly passed to the guard
- Global variables, module imports, or outer-scope bindings are disallowed

## Reference-level explanation

Type guards are essentially boolean functions with a special `is` syntax that enables type narrowing. While they behave like regular functions that:

1. Take a value of a base type
2. Return a boolean indicating if the value satisfies some condition

The key difference is that when used in conditionals, they also:

- Narrow the type to a more specific type representing values that satisfy the guard

For example, `is Strong(password)` compiles to a regular boolean function call, but the type system understands that a true result means the password meets the strength criteria.

### Implementation Details

The implementation will:

1. Parse type guard declarations into regular function AST nodes
2. Track the type narrowing implications in the type checker
3. Generate appropriate boolean function calls
4. Ensure type safety is maintained when the narrowed types cross function boundaries

### Runtime Behavior

Type guards are pure functions that:

1. Cannot modify their input parameters
2. Must be deterministic
3. Should be fast and side-effect free
4. Can be inlined by the compiler

Example of invalid type guard:

```go
// Invalid: Modifies input
is (password: Password) Strong {
    password = password + "!" // Error: Cannot modify input
    return len(password) >= 12
}

// Invalid: Non-deterministic
is (password: Password) Strong {
    return random() > 0.5 // Error: Must be deterministic
}
```

## Drawbacks

1. Additional complexity in the type system

   - New type narrowing rules to implement
   - More complex type inference
   - Additional compiler passes

2. Potential performance impact from runtime type checks

   - Guard function calls add overhead
   - Type information must be preserved
   - Runtime type information needed

3. Learning curve for developers new to the concept

   - New syntax to learn
   - Different from Go/TypeScript approaches
   - Requires understanding of type narrowing

4. May encourage over-use of type guards where simpler solutions exist
   - Could lead to complex type hierarchies
   - Might be used where basic types suffice
   - Potential for over-engineering

## Rationale and prior art

The main alternatives and prior art come from Go and TypeScript, which are the most relevant languages to Forst's design:

### Go's Approach

Go provides two main mechanisms for type checking and narrowing:

1. Type assertions with `x.(T)` syntax

   - Runtime type checking
   - Panics on failed assertions
   - No compile-time safety
   - Example:

   ```go
   if str, ok := val.(string); ok {
       // str is now a string
   } else {
       // handle failure
   }
   ```

2. Type switches with `switch x.(type)`

   - Runtime type checking
   - Verbose syntax
   - Requires duplicated logic
   - Example:

   ```go
   switch v := val.(type) {
   case string:
       // v is a string
   case int:
       // v is an int
   default:
       // handle other cases
   }
   ```

Drawbacks of Go's approach:

- Runtime panics on failed assertions
- No compile-time type narrowing
- Verbose error handling
- Scattered validation logic
- No way to define custom type predicates

### TypeScript's Approach

TypeScript offers several mechanisms for type narrowing:

1. Type assertions with `as` syntax

   - Bypasses type checking
   - Can be unsafe
   - Example:

   ```ts
   const str = val as string;
   ```

2. Type predicates with `is` keyword

   - More verbose implementation
   - Less integrated with type system
   - Example:

   ```ts
   function isString(val: unknown): val is string {
     return typeof val === "string";
   }
   ```

3. Branded types

   - Requires artificial type discriminators
   - Complex to implement
   - Example:

   ```ts
   type Branded<T, B> = T & { __brand: B };
   type StrongPassword = Branded<string, "StrongPassword">;
   ```

Drawbacks of TypeScript's approach:

- Type assertions can be unsafe
- Branded types are artificial and complex
- Type predicates are verbose
- No first-class type guard support

## Use Cases

We aim to provide a type-safe solution that makes typical development tasks more ergonomic to accomplish. Type guards directly address common pain points in modern development:

- **API Development**: Instead of writing repetitive validation code and error handling, developers can define validation rules once and reuse them across endpoints
- **Data Processing**: Type guards make it easy to validate and transform data at system boundaries, reducing the need for defensive programming
- **Error Handling**: By moving validation into the type system, error cases are caught earlier and handled more consistently
- **Refactoring**: Type guards make it safer to refactor code by ensuring validation rules are maintained across changes
- **Documentation**: The type system itself documents validation requirements, reducing the need for separate documentation

For example, defining a tRPC server context with proper validation becomes much simpler.

### Example: Input Validation with Type Guards

Type guards are particularly powerful for implementing input validation in API systems like tRPC. They allow for declarative validation rules that are enforced at compile time while still providing runtime validation. Here's an example:

```go
// trpc package
is (m Mutation) Input(input Shape) {
  return m is Shape({ input })
}

is (m Mutation) Context(ctx Shape) {
  return m is Shape({ ctx })
}

// app.ft
type AppContext = Shape({
  sessionId: String.Nullable()
})

type AppMutation = trpc.Mutation.Context(AppContext)

is (ctx AppContext) LoggedIn() {
  return ctx.userId != nil
}

// validate.ft

// Define validation rules using type guards
type PhoneNumber =
  String.Min(3).Max(10) & (
    String.HasPrefix("+")
    | String.HasPrefix("0")
  )

// Use in tRPC-like mutation
func createUser({ ctx, input }: AppMutation.Input({
  id: UUID.V4(),
  name: String.Min(3).Max(10),
  phoneNumber: PhoneNumber,
  bankAccount: {
    iban: String.Min(10).Max(34),
  },
})) {
  // The type system ensures all validation rules have been checked
  // before this function is called
  ensure ctx is trpc.Context.LoggedIn()
  return createUserDatabase(
    input.id,
    input.name,
    input.phoneNumber,
    input.bankAccount.iban
  )
}
```

This approach:

- Makes authentication requirements explicit in the type system
- Automatically validates context before handler execution
- Provides clear error messages when requirements aren't met
- Reduces boilerplate in handler implementations
- Makes it impossible to forget authentication checks

#### Clear Intent

Type guards make the intent of code immediately clear through explicit validation rules. For example:

```go
if password is Strong() {
    // password is now narrowed to Password.Strong()
}
```

#### Compile-time Safety

Type guards provide strong compile-time guarantees without runtime panics. The type system ensures all validation rules are checked before code execution, with clear error messages when validation fails.

#### Minimal Runtime Overhead

Type guards avoid the overhead of interface{} boxing and runtime type information. The compiler can inline guard functions and optimize validation checks, maintaining Go's performance characteristics.

#### Natural Tagged Unions

Type guards enable natural tagged unions through first-class support in the type system. This provides clear type narrowing and pattern matching capabilities without complex inheritance hierarchies.

#### Centralized Validation

Type guards centralize validation logic into reusable, composable rules. This makes validation requirements clear and maintainable, while keeping the implementation details encapsulated.

Forst's design philosophy prioritizes explicit, compile-time type safety over runtime convenience. Unlike Go (simplicity over safety) or TypeScript (flexibility over safety), Forst accepts more upfront work and some runtime overhead to maintain Go compatibility while providing stronger type guarantees.

## Implementation Plan

### Phase 1: Basic Type Guards

1. Implement type guard syntax and parsing
2. Add basic type narrowing
3. Support simple type guards
4. Add ensure statement integration

### Phase 2: Advanced Features

1. Add type guard composition
2. Implement generic type guards
3. Add pattern matching support
4. Improve type inference

### Phase 3: Optimization

1. Add guard function inlining
2. Optimize runtime checks
3. Improve error messages
4. Add IDE support

## Migration Strategy

1. Provide migration tools for:

   - Converting type assertions to type guards
   - Updating ensure statements
   - Refactoring validation functions

2. Documentation and examples:

   - Migration guides
   - Best practices
   - Common patterns

3. Backward compatibility:
   - Support old syntax during transition
   - Gradual deprecation of alternatives
   - Clear upgrade path

## Future possibilities

1. Type guard composition

   - Allow combining multiple guards
   - Support guard inheritance
   - Enable guard libraries

2. Type guard inference

   - Infer guard conditions
   - Suggest guard improvements
   - Auto-generate guards

3. Type guard macros

   - Generate guards from patterns
   - Create guard templates
   - Support guard metaprogramming

4. Type guard libraries
   - Standard guard library
   - Domain-specific guards
   - Guard testing tools

## Unresolved questions

1. How to handle type guards across module boundaries?

   - Export/import semantics
   - Versioning strategy
   - Dependency management

2. Should type guards be allowed to modify the value they're checking?

   - Current answer: No, for safety and clarity
   - Future consideration: Allow with explicit opt-in
   - Impact on type system

3. How to handle type guards in generic contexts?

   - Generic type parameters
   - Type constraints
   - Type inference

## Performance considerations

1. Runtime overhead

   - Guard function calls
   - Type information storage
   - Pattern matching cost

2. Compile-time impact

   - Type checking complexity
   - Build time increase
   - Memory usage

3. Optimization opportunities

   - Guard inlining
   - Type information caching
   - Pattern matching optimization
