---
Feature Name: type-guards
Start Date: 2025-05-28
---

# Type Guards

## Summary

Introduce first-class type guards as a language feature to enable type narrowing without requiring type assertions or branded types. Type guards will allow developers to define predicates that narrow types at compile time, making type-safe code more ergonomic and maintainable.

## Motivation

Currently, Forst provides built-in `ensure` statements with an `is` operator for runtime validation and error handling:

```go
func mustNotExceedSpeedLimit(speed: Int) {
    ensure speed is LessThan(100)
        or TooFast("Speed must not exceed 100 km/h")
}
```

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

## Guide-level explanation

Type guards are predicates that narrow types at compile time. They are defined using the `is` keyword and can be used to refine types in conditional blocks. Here's a simple example:

```go
type Password = string

is (password: Password) Strong {
    return len(password) >= 12
}

fun validatePassword(password: Password) {
    if password is Strong() {
        // pwd is now narrowed to Password.Strong()
        // Can safely use it in contexts requiring strong passwords
    }
    // More verbose syntax:
    if password is Password.Strong() {
        // ...
    }
}
```

### Input Validation with Type Guards

Type guards are particularly powerful for implementing input validation in API systems like tRPC. They allow for declarative validation rules that are enforced at compile time while still providing runtime validation. Here's an example:

```go
// Define validation rules using type guards
type PhoneNumber =
  String.Min(3).Max(10) & (
    String.HasPrefix("+")
    | String.HasPrefix("0")
  )

// Use in tRPC-like mutation
func createUser(op: trpc.Mutation.Input({
  id: UUID.V4(),
  name: String.Min(3).Max(10),
  phoneNumber: PhoneNumber,
  bankAccount: {
    iban: String.Min(10).Max(34),
  },
})) {
  // The type system ensures all validation rules have been checked
  // before this function is called
  fmt.Println("Creating user with id: %s", op.input.id)
  return 300.3f
}
```

In this example:

1. The `trpc.Mutation.Input` type guard ensures that:

   - All required fields are present
   - Each field satisfies its validation rules
   - Nested objects are properly validated
   - The validation happens before the function is called

2. The type system guarantees that:

   - Invalid inputs cannot reach the function body
   - Validation rules are enforced at compile time
   - Runtime validation is automatically generated
   - Type information is preserved throughout the call chain

3. Benefits:
   - Declarative validation rules
   - Type-safe API boundaries
   - Automatic runtime validation
   - Clear error messages
   - Reusable validation rules

This pattern can be extended to support:

- Custom validation rules
- Complex nested structures
- Optional fields
- Default values
- Cross-field validation

### Type Guard Composition

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

Type guards can be exported and imported like other declarations:

```go
// auth.guards.forst
export is (password: Password) Strong {
    return len(password) >= 12
}

// main.forst
import auth.guards

fun validateUser(password: Password) {
    if password is auth.guards.Strong() {
        // ...
    }
}
```

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

### Usage in ensure statements

Here's an example using type guards with ensure:

```go
is (value: Int) LessThan(other: Int) {
    return value < other
}

func mustNotExceedSpeedLimit(speed: Int) {
    ensure speed is LessThan(100)
        or TooFast("Speed must not exceed 100 km/h")
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

### Forst's Solution

We aim to provide a type-safe solution that makes typical development tasks more ergonomic to accomplish. Type guards directly address common pain points in modern development:

- **API Development**: Instead of writing repetitive validation code and error handling, developers can define validation rules once and reuse them across endpoints
- **Data Processing**: Type guards make it easy to validate and transform data at system boundaries, reducing the need for defensive programming
- **Error Handling**: By moving validation into the type system, error cases are caught earlier and handled more consistently
- **Refactoring**: Type guards make it safer to refactor code by ensuring validation rules are maintained across changes
- **Documentation**: The type system itself documents validation requirements, reducing the need for separate documentation

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
