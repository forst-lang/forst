# Type Guard Interoperability with Go

## Overview

This document analyzes how the proposed type guards interact with Go's type system, syntax, and semantics. It focuses on transpilation strategies, compatibility considerations, and potential feature conflicts.

## Type System Integration

Forst's type system is richer than Go's, particularly in its support for type narrowing and predicates. This section demonstrates how we bridge this gap through transpilation.

### Basic Type Guards

Forst's type guards provide compile-time type narrowing with runtime validation. Here's a simple example:

```go
type Password = string

is (password Password) Strong {
    return len(password) >= 12
}

func validatePassword(password Password) {
    if password is Strong() {
        // password is now narrowed to Password.Strong()
    }
}
```

When transpiled to Go, we need to maintain type safety while working within Go's type system limitations. The transpiled code uses type aliases and explicit validation:

```go
type T_Password = string

// Generated sub-type
type T_StrongPassword struct {
    value T_Password
}

// Generated validation function
func IsStrong(password T_Password) bool {
    return len(password) >= 12
}

// Generated error type
type E_PasswordError struct {
    msg string
}

func (e E_PasswordError) Error() string {
    return e.msg
}

// Transpiled function
func validatePassword(password T_Password) error {
    if !IsStrong(password) {
        return E_PasswordError{msg: "password does not meet strength requirements"}
    }
    // Type narrowing is handled by the compiler
    return nil
}
```

The key challenge here is preserving Forst's type narrowing capabilities in Go's type system. We achieve this through:

- Type aliases to maintain explicit typing
- Generated validation functions for runtime checks
- Compile-time type narrowing
- Custom error types for validation failures

### Shape Guards and Structural Typing

Forst's shape guards extend type safety to structural types. This is particularly important for API validation and data processing:

```go
is (m MutationArg) Input(input Shape) {
    return m is Shape({ input })
}
```

The transpiled Go code needs to handle structural validation while maintaining type safety:

```go
// Generated type
type T_MutationInput struct {
    Input interface{}
}

// Generated validation function
func HasInput(m T_MutationArg) bool {
    if m.Input == nil {
        return false
    }
    // Additional shape validation logic
    return true
}

// Generated error type
type E_MutationError struct {
    msg string
}

func (e E_MutationError) Error() string {
    return e.msg
}
```

## Error Handling Integration

Forst's error handling combines type guards with error generation, providing a more ergonomic approach than Go's explicit error returns. Here's how it works:

```go
func mustNotExceedSpeedLimit(speed Int) {
    ensure speed is LessThan(100)
        or TooFast("Speed must not exceed 100 km/h")
}
```

The transpiled Go code maintains Go's error handling patterns while preserving Forst's type safety:

```go
type E_SpeedError struct {
    msg string
}

func (e E_SpeedError) Error() string {
    return e.msg
}

func mustNotExceedSpeedLimit(speed int) error {
    if speed >= 100 {
        return E_SpeedError{msg: "Speed must not exceed 100 km/h"}
    }
    return nil
}
```

## Type Narrowing and Method Receivers

Forst's type narrowing works particularly well with method receivers, allowing for clean, type-safe APIs:

```go
type Role = string

is (role Role) Admin {
    return role == "admin"
}

type User = Shape({
    role: Role
})

func (user: User) GrantAccess() {
    // ...
}

func processUser(user: User) {
    if user.role is Admin() {
        // user.role is narrowed to Role.Admin
        user.GrantAccess()
    }
}
```

The transpiled Go code uses method receivers to maintain a similar API style:

```go
// Role
type T_Role string

// Generated validation function
func (role T_Role) IsAdmin bool {
    return role == "admin"
}

type T_User struct {
    role T_Role
}

func processUser(user T_User) error {
    if !user.role.IsAdmin() {
        return fmt.Errorf("user is not an admin")
    }
    // Type narrowing is handled by the compiler
    return user.GrantAccess()
}
```

## Generic Type Support

The initial implementation focuses on concrete types, deferring generic support for later. This decision is based on several factors:

1. Go's generic type system is relatively new and still evolving
2. Type guards with generics add significant complexity
3. Most use cases can be handled with concrete types
4. Generic support can be added incrementally

### Type Guard Compatibility with Go Generics

Type guards can work harmoniously with Go's generic system. Here's an example of how they might interact:

```go
// Forst
type Result<T> = Shape({
    value: T,
    error: Error
})

is (result Result<T>) Success {
    return result.error == nil
}

func processResult<T>(result Result<T>) {
    if result is Success() {
        // result.value is narrowed to T
        useValue(result.value)
    }
}
```

Transpiles to Go:

```go
type Result[T any] struct {
    Value T
    Error error
}

func (r Result[T]) IsSuccess() bool {
    return r.Error == nil
}

func processResult[T any](result Result[T]) error {
    if !result.IsSuccess() {
        return result.Error
    }
    // Type narrowing is handled by the compiler
    return useValue(result.Value)
}
```

### Potential Conflicts and Resolutions

While type guards and Go generics can work together, there are some areas that require careful consideration:

1. Type Parameter Constraints

   - Go's type constraints are interface-based
   - Type guards can be used as part of constraint definitions
   - Example:

     ```go
     // Forst
     type Validatable<T> = Shape({
         value: T
     })

     is (v Validatable<T>) Valid {
         return v.value != nil
     }

     // Go
     type Validatable interface {
         IsValid() bool
     }
     ```

2. Method Sets and Type Guards

   - Go requires explicit method sets for interfaces
   - Type guards can be transpiled to satisfy interface requirements
   - No fundamental conflict, just different syntax

3. Type Inference

   - Go's type inference is more limited than Forst's
   - Type guards can help provide additional type information
   - Example:

     ```go
     // Forst
     type ValidResult<T> = Shape({
         value: T
     })

     is (v ValidResult<T>) Valid {
         return v.value != nil
     }

     func mapValues<T, U>(values []T, fn (T) U) []ValidResult<U>

     // Go
     func mapValues[T any, U Validatable](values []T, fn func(T) U) []U
     ```

4. Generic Type Bounds

   - Go's type bounds are explicit
   - Type guards can be used to define implicit bounds
   - Example:

     ```go
     // Forst
     type Number = Int | Float

     is (n Number) Numeric {
         return n is Int() || n is Float()
     }

     // Go
     type Number interface {
         ~int | ~float64
     }
     ```

### Type Composition Challenges

Type composition presents unique challenges when combining type guards with generics. Here's an analysis of the key issues and their solutions:

1. Nested Generic Types

   - Challenge: Type guards on nested generic types can be complex
   - Solution: Flatten type hierarchies where possible
   - Example:

     ```go
     // Forst
     type Container<T> = Shape({
         items: []T
     })

     type ValidContainer<T> = Container<T> where T is Valid()

     is (container ValidContainer<T>) NonEmpty {
         return len(container.items) > 0
     }

     // Go
     type Container[T any] struct {
         Items []T
     }

     type ValidContainer[T Validatable] struct {
         Container[T]
     }

     func (c ValidContainer[T]) IsNonEmpty() bool {
         return len(c.Items) > 0
     }
     ```

2. Type Guard Composition

   - Challenge: Combining multiple type guards on the same type
   - Solution: Generate composite validation functions
   - Example:

     ```go
     // Forst
     type User = Shape({
         name: String,
         age: Int
     })

     is (user User) ValidName {
         return len(user.name) > 0
     }

     is (user User) ValidAge {
         return user.age >= 0
     }

     is (user User) Valid {
         return user is ValidName() and user is ValidAge()
     }

     // Go
     type User struct {
         Name string
         Age  int
     }

     func (u User) IsValidName() bool {
         return len(u.Name) > 0
     }

     func (u User) IsValidAge() bool {
         return u.Age >= 0
     }

     func (u User) IsValid() bool {
         return u.IsValidName() && u.IsValidAge()
     }
     ```

3. Generic Type Constraints with Composition

   - Challenge: Expressing complex type constraints with composed type guards
   - Solution: Use interface composition
   - Example:

     ```go
     // Forst
     type Validated<T> = T where T is Valid()
     type Serializable<T> = T where T is JSON()

     type Processable<T> = Validated<T> & Serializable<T>

     // Go
     type Validatable interface {
         IsValid() bool
     }

     type Serializable interface {
         ToJSON() ([]byte, error)
     }

     type Processable interface {
         Validatable
         Serializable
     }
     ```

4. Type Guard Inheritance

   - Challenge: Type guards on inherited generic types
   - Solution: Use embedding and method forwarding
   - Example:

     ```go
     // Forst
     type Base<T> = Shape({
         id: T
     })

     type Derived<T> = Shape({
         base: Base<T>,
         extra: String
     })

     is (base Base<T>) ValidID {
         return base.id != nil
     }

     is (derived Derived<T>) Valid {
         return derived.base is ValidID()
     }

     // Go
     type Base[T any] struct {
         ID T
     }

     type Derived[T any] struct {
         Base[T]
         Extra string
     }

     func (b Base[T]) IsValidID() bool {
         return b.ID != nil
     }

     func (d Derived[T]) IsValid() bool {
         return d.IsValidID()
     }
     ```

5. Type Guard Specialization

   - Challenge: Specializing type guards for specific generic instantiations
   - Solution: Use type-specific validation functions
   - Example:

     ```go
     // Forst
     type Number = Int | Float

     is (n Number) Positive {
         return n > 0
     }

     is (n Int) Even {
         return n % 2 == 0
     }

     // Go
     type Number interface {
         ~int | ~float64
     }

     func IsPositive[T Number](n T) bool {
         return n > 0
     }

     func IsEven(n int) bool {
         return n%2 == 0
     }
     ```

These composition challenges highlight the need for careful design in the type guard system. The solutions focus on:

- Maintaining type safety across composition boundaries
- Preserving the expressiveness of type guards
- Ensuring efficient runtime performance
- Providing clear error messages for composition errors

## Performance Considerations

The transpilation process needs to balance type safety with performance. Key areas of focus include:

1. Runtime Overhead

   - Type guard function calls
   - Runtime type validation
   - Error type generation
   - Type information storage

2. Optimization Strategies
   - Inlining type guard functions
   - Optimizing runtime checks
   - Minimizing type information overhead
   - Using compile-time optimizations

## Future Directions

The type guard system is designed to be extensible. Future improvements will focus on:

1. Generic Support

   - Adding generic type support
   - Designing the type guard system
   - Considering Go's generics
   - Planning for constraints

2. Tooling Integration

   - IDE support for type guards
   - Debugging tools
   - Error message formatting
   - Type information display

3. Performance Optimization
   - Function inlining
   - Check elimination
   - Type information optimization
   - Runtime overhead reduction

## Conclusion

Type guards can be effectively transpiled to Go while maintaining type safety and error handling patterns. The initial implementation focuses on concrete types and clear error handling, with generic support added later based on real-world usage patterns. The key is to maintain type safety and error handling patterns while working within Go's type system limitations.
