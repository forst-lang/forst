# Effect Runtime via Forst/Go

## Approach

Create a pure Go implementation of Effect primitives that can be consumed by Forst code, providing a runtime that strictly adheres to Effect-TS semantics while being optimized for Go's execution model. This approach treats Effect as a first-class runtime concept rather than a library.

## Core Effect Runtime

### Effect Type System

```go
// Core Effect type in Go runtime
type Effect[A, E any] interface {
    // Core operations
    Map[B any](f func(A) B) Effect[B, E]
    FlatMap[B, F any](f func(A) Effect[B, F]) Effect[B, E | F]
    Recover(f func(E) A) Effect[A, Never]
    MapError[F any](f func(E) F) Effect[A, F]

    // Execution
    Run() (A, E, bool) // value, error, success
    RunSync() A        // panics on error
    RunAsync() <-chan Result[A, E]

    // Resource management
    Bracket[R any](release func(A) Effect[Unit, Never], use func(A) Effect[R, E]) Effect[R, E]

    // Concurrency
    Fork() Effect[Fiber[A, E], Never]
    Race[B, F any](other Effect[B, F]) Effect[A | B, E | F]
    Par[B, F any](other Effect[B, F]) Effect[(A, B), E | F]
}

// Result type for async operations
type Result[A, E any] struct {
    Value A
    Error E
    Success bool
}

// Fiber represents a running effect
type Fiber[A, E any] interface {
    Await() Effect[A, E]
    Interrupt() Effect[Unit, Never]
    IsRunning() bool
}
```

### Effect Constructors

```go
// Success effect
func Success[A any](value A) Effect[A, Never] {
    return &successEffect[A]{value: value}
}

// Failure effect
func Failure[E any](err E) Effect[Never, E] {
    return &failureEffect[E]{err: err}
}

// Async effect
func Async[A, E any](f func() (A, E, bool)) Effect[A, E] {
    return &asyncEffect[A, E]{fn: f}
}

// Sync effect (immediate execution)
func Sync[A, E any](f func() (A, E, bool)) Effect[A, E] {
    return &syncEffect[A, E]{fn: f}
}

// FromGoError converts Go error to Effect
func FromGoError[A any](value A, err error) Effect[A, GoError] {
    if err != nil {
        return Failure[GoError](GoError{Err: err})
    }
    return Success(value)
}
```

### Core Effect Implementations

```go
// Success effect implementation
type successEffect[A any] struct {
    value A
}

func (e *successEffect[A]) Map[B any](f func(A) B) Effect[B, Never] {
    return Success(f(e.value))
}

func (e *successEffect[A]) FlatMap[B, F any](f func(A) Effect[B, F]) Effect[B, F] {
    return f(e.value)
}

func (e *successEffect[A]) Recover(f func(Never) A) Effect[A, Never] {
    return e // Never can't be recovered from
}

func (e *successEffect[A]) MapError[F any](f func(Never) F) Effect[A, F] {
    return Success(e.value) // Never maps to any error type
}

func (e *successEffect[A]) Run() (A, Never, bool) {
    return e.value, Never{}, true
}

func (e *successEffect[A]) RunSync() A {
    return e.value
}

func (e *successEffect[A]) RunAsync() <-chan Result[A, Never] {
    ch := make(chan Result[A, Never], 1)
    ch <- Result[A, Never]{Value: e.value, Success: true}
    close(ch)
    return ch
}

// Failure effect implementation
type failureEffect[E any] struct {
    err E
}

func (e *failureEffect[E]) Map[B any](f func(A) B) Effect[B, E] {
    return Failure(e.err)
}

func (e *failureEffect[E]) FlatMap[B, F any](f func(A) Effect[B, F]) Effect[B, E | F] {
    return Failure(e.err)
}

func (e *failureEffect[E]) Recover(f func(E) A) Effect[A, Never] {
    return Success(f(e.err))
}

func (e *failureEffect[E]) MapError[F any](f func(E) F) Effect[A, F] {
    return Failure(f(e.err))
}

func (e *failureEffect[E]) Run() (A, E, bool) {
    var zero A
    return zero, e.err, false
}

func (e *failureEffect[E]) RunSync() A {
    panic(fmt.Sprintf("Effect failed: %v", e.err))
}

func (e *failureEffect[E]) RunAsync() <-chan Result[A, E] {
    ch := make(chan Result[A, E], 1)
    ch <- Result[A, E]{Error: e.err, Success: false}
    close(ch)
    return ch
}
```

## Error System

### Tagged Error Types

```go
// TaggedError represents a tagged error
type TaggedError struct {
    Tag     string
    Message string
    Data    map[string]interface{}
}

func (e *TaggedError) Error() string {
    return e.Message
}

// Error constructors
func NewTaggedError(tag, message string) *TaggedError {
    return &TaggedError{
        Tag:     tag,
        Message: message,
        Data:    make(map[string]interface{}),
    }
}

// Specific error types
type DatabaseError = TaggedError
type ValidationError = TaggedError
type NetworkError = TaggedError
type GoError struct {
    Err error
}

func (e GoError) Error() string {
    return e.Err.Error()
}

// Error constructors
func NewDatabaseError(message string, code int) *DatabaseError {
    return &DatabaseError{
        Tag:     "DatabaseError",
        Message: message,
        Code:    code,
        Data:    map[string]interface{}{"code": code},
    }
}

func NewValidationError(field, value, constraint string) *ValidationError {
    return &ValidationError{
        Tag:     "ValidationError",
        Message: fmt.Sprintf("Validation failed for field %s", field),
        Data:    map[string]interface{}{"field": field, "value": value, "constraint": constraint},
    }
}

func NewNetworkError(message string, statusCode int, retryable bool) *NetworkError {
    return &NetworkError{
        Tag:     "NetworkError",
        Message: message,
        Data:    map[string]interface{}{"statusCode": statusCode, "retryable": retryable},
    }
}
```

## Resource Management

### Bracket Pattern

```go
// Bracket provides automatic resource cleanup
func Bracket[A, E, R any](
    acquire Effect[A, E],
    release func(A) Effect[Unit, Never],
    use func(A) Effect[R, E],
) Effect[R, E] {
    return &bracketEffect[A, E, R]{
        acquire: acquire,
        release: release,
        use:     use,
    }
}

type bracketEffect[A, E, R any] struct {
    acquire Effect[A, E]
    release func(A) Effect[Unit, Never]
    use     func(A) Effect[R, E]
}

func (e *bracketEffect[A, E, R]) Map[B any](f func(R) B) Effect[B, E] {
    return &bracketEffect[A, E, B]{
        acquire: e.acquire,
        release: e.release,
        use: func(a A) Effect[B, E] {
            return e.use(a).Map(f)
        },
    }
}

func (e *bracketEffect[A, E, R]) FlatMap[B, F any](f func(R) Effect[B, F]) Effect[B, E | F] {
    return &bracketEffect[A, E | F, B]{
        acquire: e.acquire.MapError(func(err E) E | F { return err }),
        release: func(a A) Effect[Unit, Never] {
            return e.release(a)
        },
        use: func(a A) Effect[B, E | F] {
            return e.use(a).FlatMap(f)
        },
    }
}

func (e *bracketEffect[A, E, R]) Run() (R, E, bool) {
    // Acquire resource
    resource, err, success := e.acquire.Run()
    if !success {
        var zero R
        return zero, err, false
    }

    // Ensure cleanup
    defer func() {
        e.release(resource).Run()
    }()

    // Use resource
    return e.use(resource).Run()
}
```

## Concurrency Primitives

### Fiber System

```go
// Fiber represents a running effect
type fiber[A, E any] struct {
    result chan Result[A, E]
    done   bool
    mu     sync.RWMutex
}

func (f *fiber[A, E]) Await() Effect[A, E] {
    return Async(func() (A, E, bool) {
        result := <-f.result
        if result.Success {
            return result.Value, result.Error, true
        }
        var zero A
        return zero, result.Error, false
    })
}

func (f *fiber[A, E]) Interrupt() Effect[Unit, Never] {
    return Sync(func() (Unit, Never, bool) {
        f.mu.Lock()
        defer f.mu.Unlock()
        if !f.done {
            close(f.result)
            f.done = true
        }
        return Unit{}, Never{}, true
    })
}

func (f *fiber[A, E]) IsRunning() bool {
    f.mu.RLock()
    defer f.mu.RUnlock()
    return !f.done
}

// Fork creates a new fiber
func (e *asyncEffect[A, E]) Fork() Effect[Fiber[A, E], Never] {
    return Sync(func() (Fiber[A, E], Never, bool) {
        f := &fiber[A, E]{
            result: make(chan Result[A, E], 1),
            done:   false,
        }

        go func() {
            defer func() {
                f.mu.Lock()
                f.done = true
                f.mu.Unlock()
            }()

            value, err, success := e.fn()
            f.result <- Result[A, E]{
                Value:   value,
                Error:   err,
                Success: success,
            }
        }()

        return f, Never{}, true
    })
}
```

### Concurrency Operations

```go
// Race two effects
func (e *asyncEffect[A, E]) Race[B, F any](other Effect[B, F]) Effect[A | B, E | F] {
    return Async(func() (A | B, E | F, bool) {
        ch := make(chan Result[A | B, E | F], 2)

        // Run first effect
        go func() {
            value, err, success := e.Run()
            if success {
                ch <- Result[A | B, E | F]{
                    Value:   value,
                    Success: true,
                }
            } else {
                ch <- Result[A | B, E | F]{
                    Error:   err,
                    Success: false,
                }
            }
        }()

        // Run second effect
        go func() {
            value, err, success := other.Run()
            if success {
                ch <- Result[A | B, E | F]{
                    Value:   value,
                    Success: true,
                }
            } else {
                ch <- Result[A | B, E | F]{
                    Error:   err,
                    Success: false,
                }
            }
        }()

        // Return first result
        result := <-ch
        if result.Success {
            return result.Value, result.Error, true
        }
        var zero A | B
        return zero, result.Error, false
    })
}

// Run two effects in parallel
func (e *asyncEffect[A, E]) Par[B, F any](other Effect[B, F]) Effect[(A, B), E | F] {
    return Async(func() ((A, B), E | F, bool) {
        ch1 := make(chan Result[A, E], 1)
        ch2 := make(chan Result[B, F], 1)

        // Run first effect
        go func() {
            value, err, success := e.Run()
            ch1 <- Result[A, E]{Value: value, Error: err, Success: success}
        }()

        // Run second effect
        go func() {
            value, err, success := other.Run()
            ch2 <- Result[B, F]{Value: value, Error: err, Success: success}
        }()

        // Wait for both results
        result1 := <-ch1
        result2 := <-ch2

        if result1.Success && result2.Success {
            return (result1.Value, result2.Value), nil, true
        }

        // Return first error
        if !result1.Success {
            var zero (A, B)
            return zero, result1.Error, false
        }
        var zero (A, B)
        return zero, result2.Error, false
    })
}
```

## Forst Integration

### Forst Effect Types

```go
// Forst code that uses the Go runtime
import effect "github.com/forst-lang/effect-runtime"

// Effect type alias
type Effect[A, E] = effect.Effect[A, E]

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

func Async[A, E any](f func() (A, E, Bool)) Effect[A, E] {
    return effect.Async(func() (A, E, bool) {
        value, err, success := f()
        return value, err, bool(success)
    })
}

// Error constructors
func NewDatabaseError(message String, code Int) DatabaseError {
    return effect.NewDatabaseError(string(message), int(code))
}

func NewValidationError(field String, value String, constraint String) ValidationError {
    return effect.NewValidationError(string(field), string(value), string(constraint))
}
```

### Usage Examples

```go
// Database operations
func getUserById(id: Int) Effect[User, DatabaseError] {
    return Async(func() (User, DatabaseError, Bool) {
        user, err := db.GetUser(int(id))
        if err != nil {
            return User{}, NewDatabaseError(err.Error(), 500), false
        }
        return user, nil, true
    })
}

func createUser(user: User) Effect[User, DatabaseError] {
    return Async(func() (User, DatabaseError, Bool) {
        err := db.CreateUser(user)
        if err != nil {
            return User{}, NewDatabaseError(err.Error(), 500), false
        }
        return user, nil, true
    })
}

// Email operations
func sendEmail(to String, subject String, body String) Effect[Unit, NetworkError] {
    return Async(func() (Unit, NetworkError, Bool) {
        err := emailService.Send(string(to), string(subject), string(body))
        if err != nil {
            return Unit{}, NewNetworkError(err.Error(), 500, true), false
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
        return Failure(NewValidationError("name", name, "min_length"))
    }
    if !email.Contains("@") {
        return Failure(NewValidationError("email", email, "email_format"))
    }
    return Success(User{Name: name, Email: email})
}

// Resource management
func withDatabaseConnection[R any](
    use: func(Connection) Effect[R, DatabaseError]
) Effect[R, DatabaseError] {
    return effect.Bracket(
        acquire: Async(func() (Connection, DatabaseError, Bool) {
            conn, err := db.Open()
            if err != nil {
                return Connection{}, NewDatabaseError(err.Error(), 500), false
            }
            return conn, nil, true
        }),
        release: func(conn Connection) Effect[Unit, Never] {
            return Async(func() (Unit, Never, Bool) {
                conn.Close()
                return Unit{}, nil, true
            })
        },
        use: use,
    )
}

// Concurrency
func processUsers(users: List[User]) Effect[List[ProcessedUser], DatabaseError] {
    return Async(func() (List[ProcessedUser], DatabaseError, Bool) {
        // Process users in parallel
        effects := users.Map(func(user User) Effect[ProcessedUser, DatabaseError] {
            return processUser(user)
        })

        // Collect results
        results := List[ProcessedUser]{}
        for _, effect := range effects {
            user, err, success := effect.Run()
            if !success {
                return List[ProcessedUser]{}, err, false
            }
            results = results.Append(user)
        }

        return results, nil, true
    })
}
```

## Go Code Generation

### Generated Go Code

```go
// Forst Effect types generate to Go runtime calls
func getUserById(id int) effect.Effect[User, *effect.DatabaseError] {
    return effect.Async(func() (User, *effect.DatabaseError, bool) {
        user, err := db.GetUser(id)
        if err != nil {
            return User{}, effect.NewDatabaseError(err.Error(), 500), false
        }
        return user, nil, true
    })
}

// Effect operations generate to runtime calls
func (e effect.Effect[A, E]) Map[B any](f func(A) B) effect.Effect[B, E] {
    return e.Map(f)
}

func (e effect.Effect[A, E]) FlatMap[B, F any](f func(A) effect.Effect[B, F]) effect.Effect[B, E | F] {
    return e.FlatMap(f)
}

// Resource management generates to bracket calls
func withDatabaseConnection[R any](
    use func(Connection) effect.Effect[R, *effect.DatabaseError],
) effect.Effect[R, *effect.DatabaseError] {
    return effect.Bracket(
        effect.Async(func() (Connection, *effect.DatabaseError, bool) {
            conn, err := db.Open()
            if err != nil {
                return Connection{}, effect.NewDatabaseError(err.Error(), 500), false
            }
            return conn, nil, true
        }),
        func(conn Connection) effect.Effect[effect.Unit, effect.Never] {
            return effect.Async(func() (effect.Unit, effect.Never, bool) {
                conn.Close()
                return effect.Unit{}, effect.Never{}, true
            })
        },
        use,
    )
}
```

## TypeScript Generation

### Generated TypeScript

```typescript
// Effect types generate to TypeScript interfaces
interface Effect<A, E> {
  map<B>(f: (a: A) => B): Effect<B, E>;
  flatMap<B, F>(f: (a: A) => Effect<B, F>): Effect<B, E | F>;
  recover(f: (e: E) => A): Effect<A, never>;
  mapError<F>(f: (e: E) => F): Effect<A, F>;
  run(): { value: A; error: E; success: boolean };
  runSync(): A;
  runAsync(): Promise<{ value: A; error: E; success: boolean }>;
  fork(): Effect<Fiber<A, E>, never>;
  race<B, F>(other: Effect<B, F>): Effect<A | B, E | F>;
  par<B, F>(other: Effect<B, F>): Effect<[A, B], E | F>;
}

interface Fiber<A, E> {
  await(): Effect<A, E>;
  interrupt(): Effect<void, never>;
  isRunning(): boolean;
}

// Error types generate to tagged unions
type DatabaseError = {
  readonly _tag: "DatabaseError";
  readonly message: string;
  readonly code: number;
  readonly data: Record<string, unknown>;
};

type ValidationError = {
  readonly _tag: "ValidationError";
  readonly message: string;
  readonly data: { field: string; value: string; constraint: string };
};

type AppError = DatabaseError | ValidationError | NetworkError;

// API functions
function getUserById(id: number): Effect<User, DatabaseError>;
function createUser(user: User): Effect<User, DatabaseError>;
function sendEmail(
  to: string,
  subject: string,
  body: string
): Effect<void, NetworkError>;
function withDatabaseConnection<R>(
  use: (conn: Connection) => Effect<R, DatabaseError>
): Effect<R, DatabaseError>;
```

## Advantages

- **Pure Go implementation**: No JavaScript dependencies
- **Strict Effect semantics**: Faithful implementation of Effect primitives
- **Performance**: Optimized for Go's execution model
- **Type safety**: Full compile-time type checking
- **Concurrency**: Built-in support for structured concurrency
- **Resource management**: Automatic cleanup with bracket pattern

## Challenges

- **Complexity**: Full Effect runtime is complex to implement
- **Go limitations**: Go's type system limitations for error unions
- **Memory management**: Need to handle Effect chains efficiently
- **Debugging**: Effect chains can be hard to debug
- **Learning curve**: Developers need to understand Effect patterns

## Implementation Strategy

1. **Phase 1**: Core Effect types and operations
2. **Phase 2**: Error system and tagged errors
3. **Phase 3**: Resource management and bracket pattern
4. **Phase 4**: Concurrency primitives and fibers
5. **Phase 5**: Advanced patterns and optimizations
6. **Phase 6**: TypeScript generation and tooling
