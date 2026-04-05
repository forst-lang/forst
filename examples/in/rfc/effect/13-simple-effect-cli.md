# Forst CLI: Go-Powered Effects Made Simple

## Overview

Forst is a CLI tool that makes creating Effects that run in Go as easy as possible. Write Effect code in TypeScript, get Go performance automatically.

## Core Concept

**Write Effect code in TypeScript, run it in Go with zero configuration.**

Forst automatically generates Go implementations of your Effect functions and provides a seamless sidecar integration.

## Getting Started

### 1. Install Forst CLI

```bash
npm install -g @forst/cli
```

### 2. Create Effect Functions

```typescript
// effects/user.ts
import { Effect, Data } from "effect";

// Define your Effect functions
export const getUserById = (id: number): Effect.Effect<User, UserError> =>
  Effect.gen(function* () {
    // This will run in Go
    const user = yield* Effect.tryPromise({
      try: () => fetch(`/api/users/${id}`).then((r) => r.json()),
      catch: () => new DatabaseError("User not found"),
    });
    return user;
  });

export const createUser = (
  input: CreateUserInput
): Effect.Effect<User, UserError> =>
  Effect.gen(function* () {
    // This will run in Go
    const user = yield* Effect.tryPromise({
      try: () =>
        fetch("/api/users", {
          method: "POST",
          body: JSON.stringify(input),
        }).then((r) => r.json()),
      catch: () => new DatabaseError("Failed to create user"),
    });
    return user;
  });

// Error types
export class DatabaseError extends Data.TaggedError("DatabaseError")<{
  message: string;
}> {}

export class ValidationError extends Data.TaggedError("ValidationError")<{
  field: string;
  value: string;
}> {}

export type UserError = DatabaseError | ValidationError;
```

### 3. Generate Go Code

```bash
# Generate Go implementations
forst generate effects/

# This creates:
# - effects/user.go (Go implementation)
# - effects/user_sidecar.go (Sidecar server)
# - effects/user_client.ts (TypeScript client)
```

### 4. Use in Your App

```typescript
// app.ts
import { getUserById, createUser } from "./effects/user_client";

const program = Effect.gen(function* () {
  // These now run in Go automatically
  const user = yield* getUserById(123);
  const newUser = yield* createUser({
    name: "John",
    email: "john@example.com",
  });

  return { user, newUser };
});

// Run with Go performance
const result = await Effect.runPromise(program);
```

## How It Works

### 1. Effect Detection

Forst scans your TypeScript files for Effect functions and automatically:

- Detects Effect function signatures
- Extracts error types and data types
- Generates corresponding Go implementations
- Creates sidecar integration code

### 2. Go Code Generation

```go
// Generated: effects/user.go
package main

import (
    "context"
    "encoding/json"
    "net/http"
)

// Generated from getUserById
func getUserById(ctx context.Context, id int) (*User, error) {
    // Your business logic here
    user, err := fetchUserFromDB(ctx, id)
    if err != nil {
        return nil, &DatabaseError{
            Tag: "DatabaseError",
            Message: "User not found",
        }
    }
    return user, nil
}

// Generated from createUser
func createUser(ctx context.Context, input CreateUserInput) (*User, error) {
    // Your business logic here
    user, err := saveUserToDB(ctx, input)
    if err != nil {
        return nil, &DatabaseError{
            Tag: "DatabaseError",
            Message: "Failed to create user",
        }
    }
    return user, nil
}
```

### 3. Sidecar Integration

```go
// Generated: effects/user_sidecar.go
package main

import (
    "encoding/json"
    "net/http"
)

func main() {
    http.HandleFunc("/call", func(w http.ResponseWriter, r *http.Request) {
        var req struct {
            Function string      `json:"function"`
            Args     interface{} `json:"args"`
        }

        json.NewDecoder(r.Body).Decode(&req)

        switch req.Function {
        case "getUserById":
            // Call Go function and return result
            result, err := getUserById(context.Background(), req.Args.(map[string]interface{})["id"].(int))
            if err != nil {
                http.Error(w, err.Error(), 500)
                return
            }
            json.NewEncoder(w).Encode(result)
        }
    })

    http.ListenAndServe(":8080", nil)
}
```

### 4. TypeScript Client

```typescript
// Generated: effects/user_client.ts
import { Effect } from "effect";
import { ForstClient } from "@forst/client";

const client = new ForstClient({ port: 8080 });

export const getUserById = (id: number): Effect.Effect<User, UserError> =>
  Effect.tryPromise({
    try: () => client.call("getUserById", { id }),
    catch: (error) => new DatabaseError({ message: error.message }),
  });

export const createUser = (
  input: CreateUserInput
): Effect.Effect<User, UserError> =>
  Effect.tryPromise({
    try: () => client.call("createUser", input),
    catch: (error) => new DatabaseError({ message: error.message }),
  });
```

## CLI Commands

### Generate Go Code

```bash
# Generate from specific directory
forst generate src/effects/

# Generate with custom output directory
forst generate src/effects/ --output dist/

# Watch for changes and regenerate
forst generate src/effects/ --watch
```

### Start Sidecar

```bash
# Start the Go sidecar server
forst start

# Start with custom port
forst start --port 8081

# Start in development mode with hot reload
forst start --dev
```

### Build for Production

```bash
# Build optimized Go binary
forst build --output dist/sidecar

# Build with custom configuration
forst build --config forst.config.js
```

## Configuration

### forst.config.js

```javascript
module.exports = {
  // Input directory for Effect functions
  input: "src/effects/",

  // Output directory for generated code
  output: "dist/",

  // Sidecar configuration
  sidecar: {
    port: 8080,
    host: "localhost",
  },

  // Go build configuration
  go: {
    module: "github.com/yourorg/effects",
    version: "1.21",
  },

  // TypeScript configuration
  typescript: {
    target: "ES2020",
    module: "ESNext",
  },
};
```

## Advanced Features

### 1. Custom Go Implementations

```typescript
// effects/user.ts
export const getUserById = (id: number): Effect.Effect<User, UserError> =>
  Effect.gen(function* () {
    // This will run in Go
    const user = yield* Effect.tryPromise({
      try: () => fetch(`/api/users/${id}`).then((r) => r.json()),
      catch: () => new DatabaseError("User not found"),
    });
    return user;
  });
```

```go
// effects/user.go - Custom implementation
func getUserById(ctx context.Context, id int) (*User, error) {
    // Your custom Go implementation
    // Use goroutines, channels, interfaces, etc.

    // Example: Use Go's native concurrency
    var user User
    var err error

    done := make(chan struct{})
    go func() {
        defer close(done)
        user, err = fetchUserFromDB(ctx, id)
    }()

    select {
    case <-done:
        if err != nil {
            return nil, &DatabaseError{
                Tag: "DatabaseError",
                Message: err.Error(),
            }
        }
        return &user, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}
```

### 2. Error Handling

```typescript
// All errors become tagged errors automatically
export class DatabaseError extends Data.TaggedError("DatabaseError")<{
  message: string;
  code?: number;
}> {}

export class ValidationError extends Data.TaggedError("ValidationError")<{
  field: string;
  value: string;
  constraint: string;
}> {}

export class NetworkError extends Data.TaggedError("NetworkError")<{
  url: string;
  statusCode: number;
}> {}

export type AppError = DatabaseError | ValidationError | NetworkError;
```

### 3. Type Safety

```typescript
// TypeScript types are automatically generated
export interface User {
  readonly id: number;
  readonly name: string;
  readonly email: string;
}

export interface CreateUserInput {
  readonly name: string;
  readonly email: string;
}

// Go structs are generated with proper JSON tags
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}
```

## Benefits

### 1. Zero Configuration

- Install CLI tool
- Write Effect functions
- Run `forst generate`
- Get Go performance automatically

### 2. Familiar Developer Experience

- Keep writing Effect code in TypeScript
- No need to learn Go syntax
- Same error handling patterns
- Same type safety

### 3. Go Performance

- 10x performance improvement
- Native concurrency with goroutines
- Memory efficiency
- CPU optimization

### 4. Gradual Migration

- Migrate functions one at a time
- Keep existing TypeScript code
- A/B test performance improvements
- Easy rollback if needed

## Use Cases

### 1. Database Operations

```typescript
// Heavy database operations in Go
export const processLargeDataset = (
  data: Dataset
): Effect.Effect<ProcessedData, ProcessingError> =>
  Effect.gen(function* () {
    // This runs in Go with native performance
    const processed = yield* Effect.tryPromise({
      try: () => processData(data),
      catch: () => new ProcessingError("Failed to process data"),
    });
    return processed;
  });
```

### 2. File Processing

```typescript
// File I/O operations in Go
export const processFiles = (
  files: File[]
): Effect.Effect<ProcessedFile[], FileError> =>
  Effect.gen(function* () {
    // This runs in Go with native file handling
    const processed = yield* Effect.tryPromise({
      try: () => processFiles(files),
      catch: () => new FileError("Failed to process files"),
    });
    return processed;
  });
```

### 3. API Integrations

```typescript
// External API calls in Go
export const fetchExternalData = (
  url: string
): Effect.Effect<ExternalData, NetworkError> =>
  Effect.gen(function* () {
    // This runs in Go with native HTTP client
    const data = yield* Effect.tryPromise({
      try: () => fetchData(url),
      catch: () => new NetworkError("Failed to fetch data"),
    });
    return data;
  });
```

## Conclusion

Forst CLI makes it incredibly easy to get Go performance for your Effect functions:

1. **Write Effect code** in TypeScript (familiar)
2. **Run `forst generate`** (automatic)
3. **Get Go performance** (10x improvement)
4. **Zero configuration** (just works)

**The result**: TypeScript developers can keep their familiar Effect patterns while getting Go's performance benefits through a simple CLI tool.
