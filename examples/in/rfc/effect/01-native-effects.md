# Native Effect Types in Forst

## Approach

Integrate Effect types directly into Forst's type system, providing first-class support for functional programming patterns while generating idiomatic Go code.

## Core Effect Types

### Effect Type Definition

```go
// Built-in Effect type in Forst
type Effect[A, E] struct {
    // Internal representation for Go code generation
    _internal struct {
        value A
        error E
        completed bool
    }
}

// Success constructor
func Success[A any](value A) Effect[A, Never] {
    return Effect[A, Never]{
        _internal: struct {
            value A
            error Never
            completed bool
        }{
            value: value,
            completed: true,
        },
    }
}

// Failure constructor
func Failure[E any](error E) Effect[Never, E] {
    return Effect[Never, E]{
        _internal: struct {
            value Never
            error E
            completed bool
        }{
            error: error,
            completed: true,
        },
    }
}
```

### Error Types

```go
// Tagged error pattern
type DatabaseError struct {
    _tag: "DatabaseError"
    message: String
    code: Int
}

type ValidationError struct {
    _tag: "ValidationError"
    field: String
    value: String
}

// Union error type
type AppError = DatabaseError | ValidationError | NetworkError
```

## Effect Operations

### Map and FlatMap

```go
// Map operation - transform success value
func (e Effect[A, E]) Map[B any](f func(A) B) Effect[B, E] {
    // Implementation handles error propagation
}

// FlatMap operation - chain effects
func (e Effect[A, E]) FlatMap[B any, F any](f func(A) Effect[B, F]) Effect[B, E | F] {
    // Implementation handles error union types
}

// Usage example
func getUserById(id: Int) Effect[User, DatabaseError] {
    // Database operation
}

func sendEmail(user: User) Effect[Unit, NetworkError] {
    // Email operation
}

func notifyUser(id: Int) Effect[Unit, DatabaseError | NetworkError] {
    return getUserById(id).FlatMap(sendEmail)
}
```

### Error Handling

```go
// Recover from specific errors
func (e Effect[A, E]) Recover(f func(E) A) Effect[A, Never] {
    // Convert error to success value
}

// Map errors to different error types
func (e Effect[A, E]) MapError[F any](f func(E) F) Effect[A, F] {
    // Transform error type
}

// Usage
func safeGetUser(id: Int) Effect[User, Never] {
    return getUserById(id).Recover(func(err DatabaseError) User {
        return User{id: -1, name: "Unknown"}
    })
}
```

## Resource Management

### Bracket Pattern

```go
// Automatic resource cleanup
func Bracket[A any, E any, R any](
    acquire: Effect[A, E],
    release: func(A) Effect[Unit, Never],
    use: func(A) Effect[R, E],
) Effect[R, E] {
    // Implementation ensures release is called
}

// Usage example
func withDatabaseConnection[R any](
    use: func(Connection) Effect[R, DatabaseError]
) Effect[R, DatabaseError] {
    return Bracket(
        acquire: openConnection(),
        release: func(conn Connection) Effect[Unit, Never] {
            return conn.Close().MapError(func(err error) Never {
                // Log error but don't propagate
                return Never{}
            })
        },
        use: use,
    )
}
```

## Go Code Generation

### Generated Go Code

```go
// Effect[A, E] generates to:
type Effect[A, E any] struct {
    value A
    err E
    completed bool
}

// Map operation generates to:
func (e *Effect[A, E]) Map[B any](f func(A) B) *Effect[B, E] {
    if e.completed && e.err == nil {
        return &Effect[B, E]{
            value: f(e.value),
            completed: true,
        }
    }
    return &Effect[B, E]{
        err: e.err,
        completed: e.completed,
    }
}

// FlatMap operation generates to:
func (e *Effect[A, E]) FlatMap[B, F any](f func(A) *Effect[B, F]) *Effect[B, E | F] {
    if e.completed && e.err == nil {
        result := f(e.value)
        if result.completed && result.err == nil {
            return &Effect[B, E | F]{
                value: result.value,
                completed: true,
            }
        }
        return &Effect[B, E | F]{
            err: result.err,
            completed: result.completed,
        }
    }
    return &Effect[B, E | F]{
        err: e.err,
        completed: e.completed,
    }
}
```

## TypeScript Generation

### Generated TypeScript

```typescript
// Effect types generate to TypeScript unions
type Effect<A, E> =
  | { readonly _tag: "Success"; readonly value: A }
  | { readonly _tag: "Failure"; readonly error: E };

// Error types generate to tagged unions
type DatabaseError = {
  readonly _tag: "DatabaseError";
  readonly message: string;
  readonly code: number;
};

type AppError = DatabaseError | ValidationError | NetworkError;

// Effect operations generate to functions
function map<A, B, E>(effect: Effect<A, E>, f: (a: A) => B): Effect<B, E>;

function flatMap<A, B, E, F>(
  effect: Effect<A, E>,
  f: (a: A) => Effect<B, F>
): Effect<B, E | F>;
```

## Advantages

- **First-class support**: Effect types are part of the language
- **Type safety**: Compile-time guarantees for error handling
- **Go compatibility**: Generates idiomatic Go code
- **Performance**: Minimal runtime overhead
- **Developer experience**: Clear, explicit error handling

## Challenges

- **Complexity**: Adds significant complexity to the type system
- **Go interop**: May require runtime type checking for error unions
- **Learning curve**: Developers need to understand Effect patterns
- **Tooling**: Requires sophisticated type inference and code generation

## Implementation Strategy

1. **Phase 1**: Basic Effect types and operations
2. **Phase 2**: Error union types and tagged errors
3. **Phase 3**: Resource management and bracket pattern
4. **Phase 4**: Advanced patterns (batching, concurrency)
5. **Phase 5**: TypeScript generation and tooling
