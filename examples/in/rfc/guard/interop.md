# Type Guard Interoperability with Go

## Overview

This document analyzes how Forst's type guards interact with Go's type system, syntax, and semantics. It focuses on transpilation strategies and compatibility considerations.

## Core Interoperability

### Basic Type Guards

Forst:

```go
type Password = string

is (password Password) Strong {
    return len(password) >= 12
}

func validatePassword(password Password) {
    ensure password is Strong() {
        return fmt.Error("password does not meet strength requirements")
    }
}
```

Transpiles to Go:

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

### Shape Guards

Forst:

```go
is (m MutationArg) Input(input Shape) {
    return m is Shape({ input })
}
```

Transpiles to Go:

```go
// Generated type
type MutationInput struct {
    Input interface{}
}

// Generated validation function
func HasInput(m MutationArg) bool {
    if m.Input == nil {
        return false
    }
    // Additional shape validation logic
    return true
}

// Generated error type
type MutationError struct {
    msg string
}

func (e MutationError) Error() string {
    return e.msg
}
```

## Error Handling Integration

Forst's `ensure` statements integrate with Go's error handling patterns:

Forst:

```go
func mustNotExceedSpeedLimit(speed Int) {
    ensure speed is LessThan(100)
        or TooFast("Speed must not exceed 100 km/h")
}
```

Transpiles to Go:

```go
type SpeedError struct {
    msg string
}

func (e SpeedError) Error() string {
    return e.msg
}

func mustNotExceedSpeedLimit(speed int) error {
    if speed >= 100 {
        return SpeedError{msg: "Speed must not exceed 100 km/h"}
    }
    return nil
}
```

## Type Narrowing

Forst's type narrowing is handled at compile time and transpiled to Go's type system:

Forst:

```go
type Role = string

is (role Role) Admin {
    return role == "admin"
}

type User = Shape({
    role: Role
})

func processUser(user: User) {
    if user.role is Admin() {
        // user.role is narrowed to Role.Admin
        user.GrantAccess()
    }
}
```

Transpiles to Go:

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

## Limitations and Considerations

1. **Type Information Loss**

   - Go's runtime type system is less rich than Forst's
   - Some type information is lost during transpilation
   - Type narrowing must be handled at compile time

2. **Error Handling**

   - Go's error handling is more verbose
   - Error types must be generated for each guard
   - Error wrapping must be handled explicitly

3. **Performance Considerations**

   - Runtime type checks add overhead
   - Generated code must be optimized
   - Type information must be preserved where possible

4. **Debugging**
   - Stack traces must be preserved
   - Error messages must be clear
   - Type information should be available in debug builds

## Future Considerations

1. **Generic Support**

   - Initial implementation will focus on concrete types
   - Generic support can be added later
   - Type guards should be designed to be extensible

2. **Tooling Integration**

   - IDE support for type guards
   - Debugging support
   - Error message formatting

3. **Performance Optimization**
   - Inlining of guard functions
   - Elimination of redundant checks
   - Type information optimization

## Conclusion

Type guards can be effectively transpiled to Go while maintaining type safety and error handling patterns. The initial implementation should focus on concrete types and clear error handling, with generic support added later based on real-world usage patterns.
