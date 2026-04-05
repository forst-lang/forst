# Forst Native Effects: Simple Primitives for Powerful Effects

## Overview

Write Effects directly in Forst using simple but powerful primitives. Generate TypeScript clients that feel native without explicit `.call()` methods.

## Core Concept

**Write Effects in Forst, get TypeScript integration automatically.**

Use Forst's native primitives to create Effect functionality, then generate seamless TypeScript clients.

## Forst Effect Primitives

### 1. Error Definition

```go
// Explicit error definition in Forst
error DatabaseError {
    message: String
    code: Int
}

error ValidationError {
    field: String
    value: String
    constraint: String
}

// Union error type
error UserError = DatabaseError | ValidationError
```

### 2. Effect Type

```go
// Built-in Effect type
type Effect[A, E] struct {
    // Internal representation
}

// Effect constructors
func Success[A any](value A) Effect[A, Never] {
    // Implementation
}

func Failure[E any](err E) Effect[Never, E] {
    // Implementation
}

// Effect operations
func (e Effect[A, E]) Map[B any](f func(A) B) Effect[B, E] {
    // Implementation
}

func (e Effect[A, E]) FlatMap[B any, F any](f func(A) Effect[B, F]) Effect[B, E | F] {
    // Implementation
}
```

### 3. Effect Functions

```go
// Write Effect functions in Forst
func getUserById(id: Int) Effect[User, UserError] {
    return Effect.Async(func() (User, UserError, Bool) {
        user, err := db.GetUser(id)
        if err != nil {
            return User{}, DatabaseError{
                message: err.Error(),
                code: 500,
            }, false
        }
        return user, nil, true
    })
}

func createUser(input: CreateUserInput) Effect[User, UserError] {
    return Effect.Async(func() (User, UserError, Bool) {
        // Validate input
        if len(input.Name) < 3 {
            return User{}, ValidationError{
                field: "name",
                value: input.Name,
                constraint: "min_length",
            }, false
        }

        // Create user
        user, err := db.CreateUser(input)
        if err != nil {
            return User{}, DatabaseError{
                message: err.Error(),
                code: 500,
            }, false
        }

        return user, nil, true
    })
}
```

## Effect Services

### Purpose of Effect Services

Effect services in TypeScript serve to:

1. **Group related functions** under a common interface
2. **Provide dependency injection** for testing and modularity
3. **Enable composition** of related operations
4. **Abstract implementation details** from consumers

### Forst Service Implementation

```go
// Define service in Forst
service UserService {
    getUserById(id: Int) Effect[User, UserError]
    createUser(input: CreateUserInput) Effect[User, UserError]
    updateUser(id: Int, input: UpdateUserInput) Effect[User, UserError]
    deleteUser(id: Int) Effect[Unit, UserError]
}

// Service implementation
func (s UserService) getUserById(id: Int) Effect[User, UserError] {
    return getUserById(id)
}

func (s UserService) createUser(input: CreateUserInput) Effect[User, UserError] {
    return createUser(input)
}

func (s UserService) updateUser(id: Int, input: UpdateUserInput) Effect[User, UserError] {
    return Effect.Async(func() (User, UserError, Bool) {
        // Implementation
    })
}

func (s UserService) deleteUser(id: Int) Effect[Unit, UserError] {
    return Effect.Async(func() (Unit, UserError, Bool) {
        // Implementation
    })
}
```

### Service Generation Decision

Generate a service when:

1. **Multiple related functions** exist
2. **Shared state or dependencies** are needed
3. **Testing isolation** is required
4. **API boundaries** need to be defined

## Sidecar Integration

### 1. Function Registration

```go
// Auto-register Effect functions for sidecar
func init() {
    // Register individual functions
    sidecar.Register("getUserById", getUserById)
    sidecar.Register("createUser", createUser)

    // Register service
    sidecar.RegisterService("UserService", UserService{})
}
```

### 2. TypeScript Client Generation

```typescript
// Generated TypeScript client
import { Effect, Data } from "effect";

// Error types
export class DatabaseError extends Data.TaggedError("DatabaseError")<{
  message: string;
  code: number;
}> {}

export class ValidationError extends Data.TaggedError("ValidationError")<{
  field: string;
  value: string;
  constraint: string;
}> {}

export type UserError = DatabaseError | ValidationError;

// Direct function calls (no .call() needed)
export const getUserById = (id: number): Effect.Effect<User, UserError> => {
  return Effect.tryPromise({
    try: () => sidecarClient.invoke("getUserById", { id }),
    catch: (error) => convertError(error),
  });
};

export const createUser = (
  input: CreateUserInput
): Effect.Effect<User, UserError> => {
  return Effect.tryPromise({
    try: () => sidecarClient.invoke("createUser", input),
    catch: (error) => convertError(error),
  });
};

// Service interface
export class UserService extends Data.TaggedClass("UserService")<{}> {
  getUserById(id: number): Effect.Effect<User, UserError> {
    return getUserById(id);
  }

  createUser(input: CreateUserInput): Effect.Effect<User, UserError> {
    return createUser(input);
  }

  updateUser(
    id: number,
    input: UpdateUserInput
  ): Effect.Effect<User, UserError> {
    return Effect.tryPromise({
      try: () => sidecarClient.invoke("UserService.updateUser", { id, input }),
      catch: (error) => convertError(error),
    });
  }

  deleteUser(id: number): Effect.Effect<void, UserError> {
    return Effect.tryPromise({
      try: () => sidecarClient.invoke("UserService.deleteUser", { id }),
      catch: (error) => convertError(error),
    });
  }
}
```

### 3. Usage in TypeScript

```typescript
// Direct function usage
const program = Effect.gen(function* () {
  const user = yield* getUserById(123);
  const newUser = yield* createUser({
    name: "John",
    email: "john@example.com",
  });
  return { user, newUser };
});

// Service usage
const programWithService = Effect.gen(function* () {
  const userService = new UserService({});

  const user = yield* userService.getUserById(123);
  const updatedUser = yield* userService.updateUser(123, { name: "Jane" });

  return { user, updatedUser };
});

// Run the program
const result = await Effect.runPromise(program);
```

## Core Primitives

### 1. Error Handling

```go
// Explicit error definition
error DatabaseError {
    message: String
    code: Int
}

// Error creation
func NewDatabaseError(message: String, code: Int) DatabaseError {
    return DatabaseError{
        message: message,
        code: code,
    }
}

// Error handling in functions
func getUserById(id: Int) Effect[User, UserError] {
    return Effect.Async(func() (User, UserError, Bool) {
        user, err := db.GetUser(id)
        if err != nil {
            return User{}, NewDatabaseError(err.Error(), 500), false
        }
        return user, nil, true
    })
}
```

### 2. Effect Composition

```go
// Map operation
func (e Effect[A, E]) Map[B any](f func(A) B) Effect[B, E] {
    return Effect.Async(func() (B, E, Bool) {
        value, err, success := e.Run()
        if !success {
            var zero B
            return zero, err, false
        }
        return f(value), nil, true
    })
}

// FlatMap operation
func (e Effect[A, E]) FlatMap[B any, F any](f func(A) Effect[B, F]) Effect[B, E | F] {
    return Effect.Async(func() (B, E | F, Bool) {
        value, err, success := e.Run()
        if !success {
            var zero B
            return zero, err, false
        }
        return f(value).Run()
    })
}
```

### 3. Resource Management

```go
// Bracket pattern for resource management
func Bracket[A any, E any, R any](
    acquire: Effect[A, E],
    release: func(A) Effect[Unit, Never],
    use: func(A) Effect[R, E],
) Effect[R, E] {
    return Effect.Async(func() (R, E, Bool) {
        resource, err, success := acquire.Run()
        if !success {
            var zero R
            return zero, err, false
        }

        defer func() {
            release(resource).Run()
        }()

        return use(resource).Run()
    })
}

// Usage
func withDatabase[R any](use: func(Connection) Effect[R, DatabaseError]) Effect[R, DatabaseError] {
    return Bracket(
        acquire: Effect.Async(func() (Connection, DatabaseError, Bool) {
            conn, err := db.Open()
            if err != nil {
                return Connection{}, NewDatabaseError(err.Error(), 500), false
            }
            return conn, nil, true
        }),
        release: func(conn Connection) Effect[Unit, Never] {
            return Effect.Async(func() (Unit, Never, Bool) {
                conn.Close()
                return Unit{}, nil, true
            })
        },
        use: use,
    )
}
```

### 4. Concurrency

```go
// Parallel execution
func Par[A any, B any, E any, F any](
    left: Effect[A, E],
    right: Effect[B, F],
) Effect[(A, B), E | F] {
    return Effect.Async(func() ((A, B), E | F, Bool) {
        ch := make(chan Result[A, E], 1)
        ch2 := make(chan Result[B, F], 1)

        go func() {
            value, err, success := left.Run()
            ch <- Result[A, E]{Value: value, Error: err, Success: success}
        }()

        go func() {
            value, err, success := right.Run()
            ch2 <- Result[B, F]{Value: value, Error: err, Success: success}
        }()

        result1 := <-ch
        result2 := <-ch2

        if result1.Success && result2.Success {
            return (result1.Value, result2.Value), nil, true
        }

        if !result1.Success {
            var zero (A, B)
            return zero, result1.Error, false
        }
        var zero (A, B)
        return zero, result2.Error, false
    })
}

// Usage
func getUserAndProfile(id: Int) Effect[(User, Profile), UserError] {
    return Par(
        left: getUserById(id),
        right: getProfileById(id),
    )
}
```

## TypeScript Integration

### 1. Automatic Type Generation

```typescript
// Generated from Forst types
export interface User {
  readonly id: number;
  readonly name: string;
  readonly email: string;
}

export interface CreateUserInput {
  readonly name: string;
  readonly email: string;
}

// Generated from Forst errors
export class DatabaseError extends Data.TaggedError("DatabaseError")<{
  message: string;
  code: number;
}> {}

export class ValidationError extends Data.TaggedError("ValidationError")<{
  field: string;
  value: string;
  constraint: string;
}> {}

export type UserError = DatabaseError | ValidationError;
```

### 2. Seamless Function Calls

```typescript
// No .call() needed - direct function calls
const program = Effect.gen(function* () {
  const user = yield* getUserById(123);
  const newUser = yield* createUser({
    name: "John",
    email: "john@example.com",
  });
  return { user, newUser };
});
```

### 3. Service Usage

```typescript
// Service usage with dependency injection
const programWithService = Effect.gen(function* () {
  const userService = new UserService({});

  const user = yield* userService.getUserById(123);
  const updatedUser = yield* userService.updateUser(123, { name: "Jane" });

  return { user, updatedUser };
});
```

## Benefits

### 1. Native Go Performance

- **10x performance improvement** over TypeScript
- **Native concurrency** with goroutines
- **Memory efficiency** with Go's garbage collector
- **CPU optimization** with compiled Go code

### 2. Simple Primitives

- **Explicit error definition** with clear syntax
- **Effect composition** with Map and FlatMap
- **Resource management** with Bracket pattern
- **Concurrency** with Par and Race

### 3. Seamless Integration

- **Direct function calls** in TypeScript
- **Automatic type generation** from Forst
- **No explicit .call() methods** needed
- **Familiar Effect patterns** for TypeScript developers

### 4. Service Architecture

- **Group related functions** under services
- **Dependency injection** for testing
- **Clear API boundaries** between modules
- **Composable operations** for complex workflows

## Conclusion

Forst provides simple but powerful primitives for creating Effect functionality:

1. **Write Effects in Forst** using native Go primitives
2. **Define errors explicitly** with clear syntax
3. **Create services** for related functions
4. **Generate TypeScript clients** that feel native
5. **Get Go performance** with familiar Effect patterns

**The result**: TypeScript developers can use familiar Effect patterns while getting Go's performance benefits through seamless sidecar integration.
