# Resource Management Patterns

## Approach

Implement automatic resource management patterns in Forst that provide safe resource acquisition and cleanup, similar to Effect's bracket pattern, while generating idiomatic Go code with proper error handling.

## Resource Management Types

### Resource Type Definition

```go
// Resource represents a managed resource
type Resource[A, E] struct {
    acquire: func() (A, E, Bool)  // value, error, success
    release: func(A) (Unit, Never, Bool)  // cleanup function
}

// Resource constructor
func NewResource[A any, E any](
    acquire: func() (A, E, Bool),
    release: func(A) (Unit, Never, Bool),
) Resource[A, E] {
    return Resource[A, E]{
        acquire: acquire,
        release: release,
    }
}
```

### Bracket Pattern

```go
// Bracket provides automatic resource cleanup
func Bracket[A any, E any, R any](
    resource: Resource[A, E],
    use: func(A) (R, E, Bool),
) (R, E, Bool) {
    // Acquire resource
    value, err, success := resource.acquire()
    if !success || err != nil {
        var zero R
        return zero, err, success
    }
    
    // Ensure cleanup happens
    defer func() {
        resource.release(value)
    }()
    
    // Use resource
    return use(value)
}

// Bracket with error transformation
func BracketMap[A any, E any, R any, F any](
    resource: Resource[A, E],
    use: func(A) (R, F, Bool),
    mapError: func(E) F,
) (R, F, Bool) {
    value, err, success := resource.acquire()
    if !success || err != nil {
        var zero R
        return zero, mapError(err), success
    }
    
    defer func() {
        resource.release(value)
    }()
    
    return use(value)
}
```

## Common Resource Patterns

### Database Connections

```go
// Database connection resource
func NewDatabaseConnection() Resource[Connection, DatabaseError] {
    return NewResource(
        acquire: func() (Connection, DatabaseError, Bool) {
            conn, err := db.Open()
            if err != nil {
                return Connection{}, NewDatabaseError(err.Error(), 500), false
            }
            return conn, nil, true
        },
        release: func(conn Connection) (Unit, Never, Bool) {
            err := conn.Close()
            if err != nil {
                // Log error but don't propagate
                log.Printf("Failed to close connection: %v", err)
            }
            return Unit{}, nil, true
        },
    )
}

// Usage
func withDatabase[R any](
    use: func(Connection) (R, DatabaseError, Bool)
) (R, DatabaseError, Bool) {
    return Bracket(NewDatabaseConnection(), use)
}

// Example usage
func getUserById(id: Int) (User, DatabaseError, Bool) {
    return withDatabase(func(conn Connection) (User, DatabaseError, Bool) {
        user, err := conn.GetUser(id)
        if err != nil {
            return User{}, NewDatabaseError(err.Error(), 500), false
        }
        return user, nil, true
    })
}
```

### File Operations

```go
// File resource
func NewFileResource(path String) Resource[File, FileError] {
    return NewResource(
        acquire: func() (File, FileError, Bool) {
            file, err := os.Open(string(path))
            if err != nil {
                return File{}, NewFileError(err.Error(), "open_failed"), false
            }
            return File{file: file}, nil, true
        },
        release: func(file File) (Unit, Never, Bool) {
            err := file.file.Close()
            if err != nil {
                log.Printf("Failed to close file: %v", err)
            }
            return Unit{}, nil, true
        },
    )
}

// File operations
func readFile(path String) (String, FileError, Bool) {
    return Bracket(
        NewFileResource(path),
        func(file File) (String, FileError, Bool) {
            content, err := file.ReadAll()
            if err != nil {
                return "", NewFileError(err.Error(), "read_failed"), false
            }
            return string(content), nil, true
        },
    )
}

func writeFile(path String, content String) (Unit, FileError, Bool) {
    return Bracket(
        NewFileResource(path),
        func(file File) (Unit, FileError, Bool) {
            err := file.WriteString(string(content))
            if err != nil {
                return Unit{}, NewFileError(err.Error(), "write_failed"), false
            }
            return Unit{}, nil, true
        },
    )
}
```

### HTTP Clients

```go
// HTTP client resource
func NewHTTPClient() Resource[HTTPClient, NetworkError] {
    return NewResource(
        acquire: func() (HTTPClient, NetworkError, Bool) {
            client := &http.Client{
                Timeout: 30 * time.Second,
            }
            return HTTPClient{client: client}, nil, true
        },
        release: func(client HTTPClient) (Unit, Never, Bool) {
            // HTTP client cleanup (close idle connections)
            client.client.CloseIdleConnections()
            return Unit{}, nil, true
        },
    )
}

// HTTP operations
func makeRequest(url String, method String, body String) (String, NetworkError, Bool) {
    return Bracket(
        NewHTTPClient(),
        func(client HTTPClient) (String, NetworkError, Bool) {
            req, err := http.NewRequest(string(method), string(url), strings.NewReader(string(body)))
            if err != nil {
                return "", NewNetworkError(err.Error(), 0, false), false
            }
            
            resp, err := client.client.Do(req)
            if err != nil {
                return "", NewNetworkError(err.Error(), 0, true), false
            }
            defer resp.Body.Close()
            
            responseBody, err := io.ReadAll(resp.Body)
            if err != nil {
                return "", NewNetworkError(err.Error(), resp.StatusCode, false), false
            }
            
            return string(responseBody), nil, true
        },
    )
}
```

## Nested Resource Management

### Multiple Resources

```go
// Multiple resource management
func withDatabaseAndFile[R any](
    dbPath String,
    filePath String,
    use: func(Connection, File) (R, AppError, Bool)
) (R, AppError, Bool) {
    // Acquire database connection
    dbConn, dbErr, dbSuccess := NewDatabaseConnection().acquire()
    if !dbSuccess || dbErr != nil {
        var zero R
        return zero, dbErr, dbSuccess
    }
    
    // Ensure database cleanup
    defer func() {
        NewDatabaseConnection().release(dbConn)
    }()
    
    // Acquire file
    file, fileErr, fileSuccess := NewFileResource(filePath).acquire()
    if !fileSuccess || fileErr != nil {
        var zero R
        return zero, fileErr, fileSuccess
    }
    
    // Ensure file cleanup
    defer func() {
        NewFileResource(filePath).release(file)
    }()
    
    // Use both resources
    return use(dbConn, file)
}

// Usage example
func processDataFile(dbPath String, filePath String) (ProcessedData, AppError, Bool) {
    return withDatabaseAndFile(
        dbPath,
        filePath,
        func(conn Connection, file File) (ProcessedData, AppError, Bool) {
            // Read data from file
            content, fileErr, fileSuccess := file.ReadAll()
            if !fileSuccess || fileErr != nil {
                return ProcessedData{}, fileErr, fileSuccess
            }
            
            // Process data
            data := parseData(string(content))
            
            // Save to database
            err := conn.SaveData(data)
            if err != nil {
                return ProcessedData{}, NewDatabaseError(err.Error(), 500), false
            }
            
            return data, nil, true
        },
    )
}
```

## Error Types for Resources

### Resource-Specific Errors

```go
// Database errors
type DatabaseError struct {
    _tag: "DatabaseError"
    message: String
    code: Int
    operation: String
}

func NewDatabaseError(message String, code Int, operation String) DatabaseError {
    return DatabaseError{
        _tag: "DatabaseError",
        message: message,
        code: code,
        operation: operation,
    }
}

// File errors
type FileError struct {
    _tag: "FileError"
    message: String
    operation: String
    path: String
}

func NewFileError(message String, operation String, path String) FileError {
    return FileError{
        _tag: "FileError",
        message: message,
        operation: operation,
        path: path,
    }
}

// Network errors
type NetworkError struct {
    _tag: "NetworkError"
    message: String
    statusCode: Int
    retryable: Bool
}

func NewNetworkError(message String, statusCode Int, retryable Bool) NetworkError {
    return NetworkError{
        _tag: "NetworkError",
        message: message,
        statusCode: statusCode,
        retryable: retryable,
    }
}

// Union error type
type AppError = DatabaseError | FileError | NetworkError
```

## Go Code Generation

### Generated Go Code

```go
// Resource type generates to:
type Resource[A, E any] struct {
    acquire func() (A, E, bool)
    release func(A) (Unit, Never, bool)
}

func NewResource[A, E any](
    acquire func() (A, E, bool),
    release func(A) (Unit, Never, bool),
) *Resource[A, E] {
    return &Resource[A, E]{
        acquire: acquire,
        release: release,
    }
}

// Bracket generates to:
func Bracket[A, E, R any](
    resource *Resource[A, E],
    use func(A) (R, E, bool),
) (R, E, bool) {
    value, err, success := resource.acquire()
    if !success || err != nil {
        var zero R
        return zero, err, success
    }
    
    defer func() {
        resource.release(value)
    }()
    
    return use(value)
}

// Database connection generates to:
func NewDatabaseConnection() *Resource[Connection, *DatabaseError] {
    return NewResource(
        func() (Connection, *DatabaseError, bool) {
            conn, err := db.Open()
            if err != nil {
                return Connection{}, &DatabaseError{
                    Message: err.Error(),
                    Code: 500,
                }, false
            }
            return conn, nil, true
        },
        func(conn Connection) (Unit, Never, bool) {
            err := conn.Close()
            if err != nil {
                log.Printf("Failed to close connection: %v", err)
            }
            return Unit{}, Never{}, true
        },
    )
}
```

## TypeScript Generation

### Generated TypeScript

```typescript
// Resource types generate to:
type Resource<A, E> = {
  acquire(): { value: A; error: E; success: boolean }
  release(value: A): { value: void; error: never; success: boolean }
}

// Error types generate to tagged unions
type DatabaseError = {
  readonly _tag: "DatabaseError"
  readonly message: string
  readonly code: number
  readonly operation: string
}

type FileError = {
  readonly _tag: "FileError"
  readonly message: string
  readonly operation: string
  readonly path: string
}

type AppError = DatabaseError | FileError | NetworkError

// API functions
function withDatabase<R>(
  use: (conn: Connection) => { value: R; error: DatabaseError; success: boolean }
): { value: R; error: DatabaseError; success: boolean }

function readFile(path: string): { value: string; error: FileError; success: boolean }
function writeFile(path: string, content: string): { value: void; error: FileError; success: boolean }
```

## Advantages

- **Automatic cleanup**: Resources are automatically cleaned up
- **Error safety**: Proper error handling for resource operations
- **Go compatibility**: Generates idiomatic Go code
- **Type safety**: Compile-time guarantees for resource management
- **Familiar patterns**: Similar to Go's defer statements

## Challenges

- **Complexity**: Resource management can be complex
- **Error propagation**: Need to handle errors from both acquire and use
- **Nested resources**: Managing multiple resources can be tricky
- **Performance**: Some overhead for resource management

## Implementation Strategy

1. **Phase 1**: Basic Resource type and Bracket pattern
2. **Phase 2**: Common resource patterns (database, file, HTTP)
3. **Phase 3**: Nested resource management
4. **Phase 4**: Error handling and recovery
5. **Phase 5**: TypeScript generation
