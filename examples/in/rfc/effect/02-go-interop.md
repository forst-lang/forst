# Go Interop Wrapper Approach

## Approach

Create a Go library that provides Effect-like functionality, then provide Forst bindings that generate clean calls to this library. This approach leverages Go's existing ecosystem while providing Effect patterns through a well-designed API.

## Go Effect Library

### Core Library Structure

```go
// go.mod
module github.com/forst-lang/effect

go 1.21

// effect.go
package effect

// Effect represents a computation that may fail
type Effect[A, E any] struct {
    fn func() (A, E, bool) // value, error, completed
}

// Success creates a successful effect
func Success[A any](value A) *Effect[A, Never] {
    return &Effect[A, Never]{
        fn: func() (A, Never, bool) {
            return value, Never{}, true
        },
    }
}

// Failure creates a failed effect
func Failure[E any](err E) *Effect[Never, E] {
    return &Effect[Never, E]{
        fn: func() (Never, E, bool) {
            var zero Never
            return zero, err, true
        },
    }
}

// Map transforms the success value
func (e *Effect[A, E]) Map[B any](f func(A) B) *Effect[B, E] {
    return &Effect[B, E]{
        fn: func() (B, E, bool) {
            val, err, completed := e.fn()
            if completed && err == nil {
                return f(val), err, true
            }
            var zero B
            return zero, err, completed
        },
    }
}

// FlatMap chains effects
func (e *Effect[A, E]) FlatMap[B, F any](f func(A) *Effect[B, F]) *Effect[B, E | F] {
    return &Effect[B, E | F]{
        fn: func() (B, E | F, bool) {
            val, err, completed := e.fn()
            if completed && err == nil {
                return f(val).fn()
            }
            var zero B
            return zero, err, completed
        },
    }
}

// Run executes the effect and returns the result
func (e *Effect[A, E]) Run() (A, E, bool) {
    return e.fn()
}
```

### Error Types

```go
// errors.go
package effect

// TaggedError represents a tagged error
type TaggedError struct {
    Tag     string
    Message string
    Data    map[string]interface{}
}

func (e *TaggedError) Error() string {
    return e.Message
}

// NewTaggedError creates a new tagged error
func NewTaggedError(tag, message string) *TaggedError {
    return &TaggedError{
        Tag:     tag,
        Message: message,
        Data:    make(map[string]interface{}),
    }
}

// Error types
type DatabaseError = TaggedError
type ValidationError = TaggedError
type NetworkError = TaggedError

// Error constructors
func NewDatabaseError(message string) *DatabaseError {
    return &DatabaseError{
        Tag:     "DatabaseError",
        Message: message,
        Data:    make(map[string]interface{}),
    }
}

func NewValidationError(field, message string) *ValidationError {
    return &ValidationError{
        Tag:     "ValidationError",
        Message: message,
        Data:    map[string]interface{}{"field": field},
    }
}
```

### Resource Management

```go
// resource.go
package effect

// Bracket provides automatic resource cleanup
func Bracket[A, E, R any](
    acquire func() *Effect[A, E],
    release func(A) *Effect[Unit, Never],
    use func(A) *Effect[R, E],
) *Effect[R, E] {
    return &Effect[R, E]{
        fn: func() (R, E, bool) {
            // Acquire resource
            resource, err, completed := acquire().fn()
            if !completed || err != nil {
                var zero R
                return zero, err, completed
            }
            
            // Use resource
            result, err, completed := use(resource).fn()
            
            // Always attempt to release
            release(resource).fn()
            
            return result, err, completed
        },
    }
}

// Unit represents the unit type
type Unit struct{}

var unit = Unit{}
```

## Forst Bindings

### Forst Effect Types

```go
// Forst code that uses the Go library
import effect "github.com/forst-lang/effect"

// Effect type alias
type Effect[A, E] = *effect.Effect[A, E]

// Error type definitions
type DatabaseError = *effect.DatabaseError
type ValidationError = *effect.ValidationError
type NetworkError = *effect.NetworkError

// Effect constructors
func Success[A any](value A) Effect[A, Never] {
    return effect.Success(value)
}

func Failure[E any](err E) Effect[Never, E] {
    return effect.Failure(err)
}

// Effect operations
func (e Effect[A, E]) Map[B any](f func(A) B) Effect[B, E] {
    return e.Map(f)
}

func (e Effect[A, E]) FlatMap[B any, F any](f func(A) Effect[B, F]) Effect[B, E | F] {
    return e.FlatMap(f)
}

// Error constructors
func NewDatabaseError(message String) DatabaseError {
    return effect.NewDatabaseError(string(message))
}

func NewValidationError(field String, message String) ValidationError {
    return effect.NewValidationError(string(field), string(message))
}
```

### Usage Examples

```go
// Database operations
func getUserById(id: Int) Effect[User, DatabaseError] {
    return Effect.Async(func() (User, DatabaseError, Bool) {
        user, err := db.GetUser(id)
        if err != nil {
            return User{}, NewDatabaseError(err.Error()), true
        }
        return user, nil, true
    })
}

func createUser(user: User) Effect[User, DatabaseError] {
    return Effect.Async(func() (User, DatabaseError, Bool) {
        err := db.CreateUser(user)
        if err != nil {
            return User{}, NewDatabaseError(err.Error()), true
        }
        return user, nil, true
    })
}

// Email operations
func sendEmail(to String, subject String, body String) Effect[Unit, NetworkError] {
    return Effect.Async(func() (Unit, NetworkError, Bool) {
        err := emailService.Send(to, subject, body)
        if err != nil {
            return Unit{}, NewNetworkError(err.Error()), true
        }
        return Unit{}, nil, true
    })
}

// Composed operations
func registerUser(name String, email String) Effect[User, DatabaseError | ValidationError] {
    return validateUser(name, email).FlatMap(func(validated User) Effect[User, DatabaseError] {
        return createUser(validated)
    })
}

func validateUser(name String, email String) Effect[User, ValidationError] {
    if len(name) < 3 {
        return Failure(NewValidationError("name", "Name must be at least 3 characters"))
    }
    if !email.Contains("@") {
        return Failure(NewValidationError("email", "Invalid email format"))
    }
    return Success(User{Name: name, Email: email})
}
```

### Resource Management

```go
// Database connection management
func withDatabaseConnection[R any](
    use: func(Connection) Effect[R, DatabaseError]
) Effect[R, DatabaseError] {
    return effect.Bracket(
        acquire: func() Effect[Connection, DatabaseError] {
            return Effect.Async(func() (Connection, DatabaseError, Bool) {
                conn, err := db.Open()
                if err != nil {
                    return Connection{}, NewDatabaseError(err.Error()), true
                }
                return conn, nil, true
            })
        },
        release: func(conn Connection) Effect[Unit, Never] {
            return Effect.Async(func() (Unit, Never, Bool) {
                conn.Close()
                return Unit{}, nil, true
            })
        },
        use: use,
    )
}

// Usage
func getUserWithConnection(id: Int) Effect[User, DatabaseError] {
    return withDatabaseConnection(func(conn Connection) Effect[User, DatabaseError] {
        return Effect.Async(func() (User, DatabaseError, Bool) {
            user, err := conn.GetUser(id)
            if err != nil {
                return User{}, NewDatabaseError(err.Error()), true
            }
            return user, nil, true
        })
    })
}
```

## Generated Go Code

### Forst to Go Translation

```go
// Forst code:
func getUserById(id: Int) Effect[User, DatabaseError] {
    return Effect.Async(func() (User, DatabaseError, Bool) {
        user, err := db.GetUser(id)
        if err != nil {
            return User{}, NewDatabaseError(err.Error()), true
        }
        return user, nil, true
    })
}

// Generated Go code:
func getUserById(id int) *effect.Effect[User, *effect.DatabaseError] {
    return effect.NewEffect(func() (User, *effect.DatabaseError, bool) {
        user, err := db.GetUser(id)
        if err != nil {
            return User{}, effect.NewDatabaseError(err.Error()), true
        }
        return user, nil, true
    })
}
```

## TypeScript Generation

### Effect Types in TypeScript

```typescript
// Generated TypeScript types
type Effect<A, E> = {
  map<B>(f: (a: A) => B): Effect<B, E>
  flatMap<B, F>(f: (a: A) => Effect<B, F>): Effect<B, E | F>
  run(): { value: A; error: E; completed: boolean }
}

type DatabaseError = {
  readonly _tag: "DatabaseError"
  readonly message: string
  readonly data: Record<string, unknown>
}

type ValidationError = {
  readonly _tag: "ValidationError"
  readonly message: string
  readonly data: { field: string }
}

// API functions
function getUserById(id: number): Effect<User, DatabaseError>
function createUser(user: User): Effect<User, DatabaseError>
function sendEmail(to: string, subject: string, body: string): Effect<void, NetworkError>
```

## Advantages

- **Go ecosystem**: Leverages existing Go libraries and patterns
- **Performance**: Minimal overhead, uses Go's native error handling
- **Familiarity**: Go developers can understand the generated code
- **Incremental adoption**: Can be adopted gradually
- **Tooling**: Can use existing Go tooling and debugging

## Challenges

- **Type safety**: Go's type system limitations for error unions
- **Runtime overhead**: Some runtime type checking required
- **API design**: Need to design clean Forst API over Go library
- **Error handling**: Go's error handling is less structured than Effect

## Implementation Strategy

1. **Phase 1**: Create Go Effect library with core operations
2. **Phase 2**: Add Forst bindings and type definitions
3. **Phase 3**: Implement resource management patterns
4. **Phase 4**: Add TypeScript generation
5. **Phase 5**: Advanced patterns (batching, concurrency)
