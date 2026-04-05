# Composable Error Types

## Approach

Enhance Forst's existing error handling with composable error types and structured error handling patterns, without implementing a full Effect system. This approach builds on Forst's existing `ensure` statements and error handling.

## Enhanced Error Types

### Tagged Error System

```go
// Built-in tagged error support
type DatabaseError struct {
    _tag: "DatabaseError"
    message: String
    code: Int
    details: Map[String, String]
}

type ValidationError struct {
    _tag: "ValidationError"
    field: String
    value: String
    constraint: String
}

type NetworkError struct {
    _tag: "NetworkError"
    url: String
    statusCode: Int
    retryable: Bool
}

// Union error types
type AppError = DatabaseError | ValidationError | NetworkError
type UserError = ValidationError | DatabaseError
```

### Error Constructors

```go
// Error constructor functions
func NewDatabaseError(message String, code Int) DatabaseError {
    return DatabaseError{
        _tag: "DatabaseError",
        message: message,
        code: code,
        details: Map[String, String]{},
    }
}

func NewValidationError(field String, value String, constraint String) ValidationError {
    return ValidationError{
        _tag: "ValidationError",
        field: field,
        value: value,
        constraint: constraint,
    }
}

func NewNetworkError(url String, statusCode Int, retryable Bool) NetworkError {
    return NetworkError{
        _tag: "NetworkError",
        url: url,
        statusCode: statusCode,
        retryable: retryable,
    }
}
```

## Enhanced Ensure Statements

### Pattern Matching in Ensure

```go
// Pattern matching for error handling
func validateUser(user: User) {
    ensure user.name is Min(3) or NewValidationError("name", user.name, "min_length")
    ensure user.email is Email or NewValidationError("email", user.email, "email_format")
    ensure user.age is Range(18, 120) or NewValidationError("age", user.age.ToString(), "age_range")
}

// Error recovery with ensure
func safeGetUser(id: Int) User {
    user, err := getUserById(id)
    ensure err is Nil() or {
        // Recover from specific errors
        match err {
            case DatabaseError{code: 404} => return User{id: -1, name: "Unknown"}
            case DatabaseError{code: 500} => panic("Database unavailable")
            case _ => return err
        }
    }
    return user
}
```

### Error Transformation

```go
// Map errors to different types
func mapDatabaseError(err: DatabaseError) ValidationError {
    return NewValidationError("database", err.message, "database_error")
}

// Error composition
func validateAndSave(user: User) {
    // Validate first
    ensure user.name is Min(3) or NewValidationError("name", user.name, "min_length")
    
    // Save to database
    err := saveUser(user)
    ensure err is Nil() or {
        // Transform database errors to validation errors
        match err {
            case DatabaseError{code: 409} => NewValidationError("email", user.email, "already_exists")
            case _ => err
        }
    }
}
```

## Result Types

### Optional Result Type

```go
// Result type for operations that may fail
type Result[A, E] struct {
    value: A
    error: E
    success: Bool
}

// Result constructors
func Ok[A any](value A) Result[A, Never] {
    return Result[A, Never]{
        value: value,
        success: true,
    }
}

func Err[E any](error E) Result[Never, E] {
    return Result[Never, E]{
        error: error,
        success: false,
    }
}

// Result operations
func (r Result[A, E]) IsOk() Bool {
    return r.success
}

func (r Result[A, E]) IsErr() Bool {
    return !r.success
}

func (r Result[A, E]) Unwrap() A {
    ensure r.success or panic("Called Unwrap on error result")
    return r.value
}

func (r Result[A, E]) UnwrapOr(defaultValue A) A {
    if r.success {
        return r.value
    }
    return defaultValue
}

func (r Result[A, E]) Map[B any](f func(A) B) Result[B, E] {
    if r.success {
        return Ok(f(r.value))
    }
    return Err(r.error)
}

func (r Result[A, E]) FlatMap[B any, F any](f func(A) Result[B, F]) Result[B, E | F] {
    if r.success {
        return f(r.value)
    }
    return Err(r.error)
}
```

## Error Handling Patterns

### Try-Catch Style with Ensure

```go
// Try-catch style error handling
func processUser(user: User) Result[User, AppError] {
    // Try validation
    validationResult := tryValidate(user)
    ensure validationResult.IsOk() or validationResult.error
    
    // Try database save
    saveResult := trySaveUser(user)
    ensure saveResult.IsOk() or saveResult.error
    
    // Try send email
    emailResult := trySendWelcomeEmail(user)
    ensure emailResult.IsOk() or emailResult.error
    
    return Ok(user)
}

func tryValidate(user: User) Result[User, ValidationError] {
    if len(user.name) < 3 {
        return Err(NewValidationError("name", user.name, "min_length"))
    }
    if !user.email.Contains("@") {
        return Err(NewValidationError("email", user.email, "email_format"))
    }
    return Ok(user)
}

func trySaveUser(user: User) Result[User, DatabaseError] {
    err := db.Save(user)
    if err != nil {
        return Err(NewDatabaseError(err.Error(), 500))
    }
    return Ok(user)
}

func trySendWelcomeEmail(user: User) Result[Unit, NetworkError] {
    err := emailService.Send(user.email, "Welcome!", "Welcome to our service!")
    if err != nil {
        return Err(NewNetworkError("email_service", 500, true))
    }
    return Ok(Unit{})
}
```

### Error Recovery

```go
// Error recovery patterns
func resilientGetUser(id: Int) User {
    user, err := getUserById(id)
    ensure err is Nil() or {
        // Recover from specific errors
        match err {
            case DatabaseError{code: 404} => return User{id: -1, name: "Unknown"}
            case DatabaseError{code: 500, retryable: true} => {
                // Retry logic
                user, retryErr := retryGetUser(id, 3)
                ensure retryErr is Nil() or retryErr
                return user
            }
            case _ => return err
        }
    }
    return user
}

func retryGetUser(id: Int, attempts: Int) (User, DatabaseError) {
    for i := 0; i < attempts; i++ {
        user, err := getUserById(id)
        if err == nil {
            return user, nil
        }
        if !err.retryable {
            return User{}, err
        }
        // Wait before retry
        time.Sleep(time.Duration(i+1) * time.Second)
    }
    return User{}, NewDatabaseError("Max retries exceeded", 500)
}
```

## Go Code Generation

### Generated Go Code

```go
// Result type generates to:
type Result[A, E any] struct {
    value A
    err E
    success bool
}

func Ok[A any](value A) *Result[A, Never] {
    return &Result[A, Never]{
        value: value,
        success: true,
    }
}

func Err[E any](err E) *Result[Never, E] {
    return &Result[Never, E]{
        err: err,
        success: false,
    }
}

// Error types generate to:
type DatabaseError struct {
    Tag string `json:"_tag"`
    Message string `json:"message"`
    Code int `json:"code"`
    Details map[string]string `json:"details"`
}

func (e *DatabaseError) Error() string {
    return e.Message
}

// Pattern matching generates to:
func matchError(err error) interface{} {
    switch e := err.(type) {
    case *DatabaseError:
        if e.Code == 404 {
            return User{ID: -1, Name: "Unknown"}
        }
        if e.Code == 500 && e.Retryable {
            return retryGetUser(id, 3)
        }
        return err
    default:
        return err
    }
}
```

## TypeScript Generation

### Generated TypeScript

```typescript
// Error types generate to tagged unions
type DatabaseError = {
  readonly _tag: "DatabaseError"
  readonly message: string
  readonly code: number
  readonly details: Record<string, string>
}

type ValidationError = {
  readonly _tag: "ValidationError"
  readonly field: string
  readonly value: string
  readonly constraint: string
}

type AppError = DatabaseError | ValidationError | NetworkError

// Result type generates to:
type Result<A, E> = 
  | { readonly _tag: "Ok"; readonly value: A }
  | { readonly _tag: "Err"; readonly error: E }

// API functions
function processUser(user: User): Result<User, AppError>
function tryValidate(user: User): Result<User, ValidationError>
function trySaveUser(user: User): Result<User, DatabaseError>
```

## Advantages

- **Builds on existing patterns**: Extends Forst's current error handling
- **Familiar syntax**: Uses existing `ensure` statements
- **Go compatibility**: Generates clean Go code
- **Type safety**: Compile-time error handling guarantees
- **Incremental adoption**: Can be adopted gradually

## Challenges

- **Limited composability**: Not as powerful as full Effect system
- **Error propagation**: Manual error handling required
- **No automatic resource management**: Manual cleanup required
- **Limited concurrency support**: No structured concurrency

## Implementation Strategy

1. **Phase 1**: Enhanced error types and tagged errors
2. **Phase 2**: Pattern matching in ensure statements
3. **Phase 3**: Result types and operations
4. **Phase 4**: Error recovery patterns
5. **Phase 5**: TypeScript generation
