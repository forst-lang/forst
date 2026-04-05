# Effect-Forst Integration: CLI Tool for Go-Powered Effects

## Overview

Forst is a CLI tool that makes creating Effects that run in Go as easy as possible. Write Effect code in TypeScript, get Go performance automatically.

## Core Concept

**Write Effect code in TypeScript, run it in Go with zero configuration.**

Forst automatically generates Go implementations of your Effect functions and provides a seamless sidecar integration.

## Architecture

### Simplified Sidecar Pattern

```typescript
// TypeScript side - Effect services generated from Forst
import { Effect, Data } from "effect";
import { ForstEffect } from "@forst/effect";

// Auto-generated Effect service from Forst function
export class UserService extends Data.TaggedClass("UserService")<{}> {
  // Generated from Forst function: getUserById(id: Int) User
  getUserById(id: number): Effect.Effect<User, UserError> {
    return ForstEffect.call("getUserById", { id });
  }

  // Generated from Forst function: createUser(user: User) User
  createUser(user: CreateUserInput): Effect.Effect<User, UserError> {
    return ForstEffect.call("createUser", user);
  }

  // Generated from Forst function: validateUser(user: User) Bool
  validateUser(user: User): Effect.Effect<boolean, ValidationError> {
    return ForstEffect.call("validateUser", user);
  }
}
```

### Forst Backend - Native Go Implementation

```go
// Forst code - leverages Go's native strengths
package main

import (
    "context"
    "database/sql"
    "fmt"
)

// Native Go implementation with proper error handling
func getUserById(ctx context.Context, id int) (*User, error) {
    conn, err := db.Acquire(ctx)
    if err != nil {
        return nil, &DatabaseError{
            Tag: "DatabaseError",
            Message: "Failed to acquire connection",
            Code: 500,
        }
    }
    defer conn.Release()

    var user User
    err = conn.QueryRowContext(ctx, "SELECT id, name, email FROM users WHERE id = $1", id).
        Scan(&user.ID, &user.Name, &user.Email)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, &NotFoundError{
                Tag: "NotFoundError",
                Message: "User not found",
                ID: id,
            }
        }
        return nil, &DatabaseError{
            Tag: "DatabaseError",
            Message: err.Error(),
            Code: 500,
        }
    }

    return &user, nil
}

// Type definitions with tagged errors
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

type DatabaseError struct {
    Tag     string `json:"_tag"`
    Message string `json:"message"`
    Code    int    `json:"code"`
}

func (e *DatabaseError) Error() string {
    return e.Message
}

type NotFoundError struct {
    Tag     string `json:"_tag"`
    Message string `json:"message"`
    ID      int    `json:"id"`
}

func (e *NotFoundError) Error() string {
    return e.Message
}
```

## Deep Integration Features

### 1. Tagged Error Integration

All Forst errors are automatically converted to Effect tagged errors:

```typescript
// Auto-generated from Forst error types
export class DatabaseError extends Data.TaggedError("DatabaseError")<{
  message: string;
  code: number;
}> {}

export class NotFoundError extends Data.TaggedError("NotFoundError")<{
  message: string;
  id: number;
}> {}

export class ValidationError extends Data.TaggedError("ValidationError")<{
  field: string;
  value: string;
  constraint: string;
}> {}

// Union type for all possible errors
export type UserError = DatabaseError | NotFoundError | ValidationError;
```

### 2. Type Guards and Shape Guards

Forst type guards and shape guards become Effect services:

```go
// Forst code with type guards
func validateUser(user: User) Bool {
    ensure user.name is Min(3) or ValidationError{
        field: "name",
        value: user.name,
        constraint: "min_length"
    }
    ensure user.email is Email or ValidationError{
        field: "email",
        value: user.email,
        constraint: "email_format"
    }
    return true
}

func processUserData(data: UserData) ProcessedUser {
    ensure data is ValidUserData or ValidationError{
        field: "data",
        value: data.ToString(),
        constraint: "valid_user_data"
    }
    // Process data...
}
```

```typescript
// Generated Effect services with type guards
export class ValidationService extends Data.TaggedClass(
  "ValidationService"
)<{}> {
  validateUser(user: User): Effect.Effect<boolean, ValidationError> {
    return ForstEffect.call("validateUser", user);
  }

  processUserData(
    data: UserData
  ): Effect.Effect<ProcessedUser, ValidationError> {
    return ForstEffect.call("processUserData", data);
  }
}

// Type guards become Effect predicates
export const isUserValid = (
  user: User
): Effect.Effect<boolean, ValidationError> =>
  Effect.gen(function* () {
    const isValid = yield* ValidationService.validateUser(user);
    return isValid;
  });
```

### 3. Custom Data Types as Effect Services

Forst custom types become Effect services with proper type safety:

```go
// Forst custom types
type ProcessedUser struct {
    User User
    Metadata ProcessedMetadata
    Timestamp int64
}

type ProcessedMetadata struct {
    ProcessingTime int64
    Version string
    Checksum string
}

func createProcessedUser(user: User) ProcessedUser {
    return ProcessedUser{
        User: user,
        Metadata: ProcessedMetadata{
            ProcessingTime: time.Now().Unix(),
            Version: "1.0",
            Checksum: generateChecksum(user),
        },
        Timestamp: time.Now().Unix(),
    }
}
```

```typescript
// Generated TypeScript types and services
export interface ProcessedUser {
  readonly user: User;
  readonly metadata: ProcessedMetadata;
  readonly timestamp: number;
}

export interface ProcessedMetadata {
  readonly processingTime: number;
  readonly version: string;
  readonly checksum: string;
}

export class ProcessedUserService extends Data.TaggedClass(
  "ProcessedUserService"
)<{}> {
  createProcessedUser(user: User): Effect.Effect<ProcessedUser, UserError> {
    return ForstEffect.call("createProcessedUser", user);
  }
}
```

## Effect Integration Patterns

### 1. Service Layer Pattern

```typescript
// Effect services that wrap Forst functions
export class UserEffectService extends Data.TaggedClass("UserEffectService")<{
  userService: UserService;
  validationService: ValidationService;
}> {
  // Composed Effect that uses multiple Forst functions
  createValidatedUser(input: CreateUserInput): Effect.Effect<User, UserError> {
    return Effect.gen(function* () {
      // Validate input using Forst validation
      const isValid = yield* this.validationService.validateUser(input);
      if (!isValid) {
        return yield* Effect.fail(
          new ValidationError({
            field: "input",
            value: JSON.stringify(input),
            constraint: "invalid_input",
          })
        );
      }

      // Create user using Forst function
      const user = yield* this.userService.createUser(input);
      return user;
    });
  }

  // Error recovery with Forst functions
  getUserWithFallback(id: number): Effect.Effect<User, never> {
    return Effect.gen(function* () {
      const user = yield* this.userService.getUserById(id);
      return user;
    }).pipe(
      Effect.catchAll((error) => {
        // Handle different Forst error types
        switch (error._tag) {
          case "NotFoundError":
            return Effect.succeed({
              id: -1,
              name: "Unknown User",
              email: "unknown@example.com",
            });
          case "DatabaseError":
            return Effect.succeed({
              id: -1,
              name: "Database Unavailable",
              email: "error@example.com",
            });
          default:
            return Effect.succeed({
              id: -1,
              name: "Error",
              email: "error@example.com",
            });
        }
      })
    );
  }
}
```

### 2. Resource Management with Forst

```typescript
// Effect resource management using Forst functions
export class DatabaseEffectService extends Data.TaggedClass(
  "DatabaseEffectService"
)<{}> {
  withTransaction<R>(
    operation: (tx: Transaction) => Effect.Effect<R, DatabaseError>
  ): Effect.Effect<R, DatabaseError> {
    return Effect.gen(function* () {
      // Begin transaction using Forst
      const tx = yield* ForstEffect.call("beginTransaction", {});

      try {
        // Execute operation
        const result = yield* operation(tx);

        // Commit transaction using Forst
        yield* ForstEffect.call("commitTransaction", { tx });

        return result;
      } catch (error) {
        // Rollback transaction using Forst
        yield* ForstEffect.call("rollbackTransaction", { tx });
        throw error;
      }
    });
  }
}
```

### 3. Concurrency with Forst Functions

```typescript
// Effect concurrency using Forst functions
export class ConcurrencyEffectService extends Data.TaggedClass(
  "ConcurrencyEffectService"
)<{
  userService: UserService;
}> {
  // Parallel execution of Forst functions
  processUsersInParallel(userIds: number[]): Effect.Effect<User[], UserError> {
    return Effect.gen(function* () {
      // Create effects for each user
      const userEffects = userIds.map((id) => this.userService.getUserById(id));

      // Execute in parallel
      const users = yield* Effect.all(userEffects, {
        concurrency: "unbounded",
      });

      return users;
    });
  }

  // Race between Forst functions
  getUserWithTimeout(
    id: number,
    timeoutMs: number
  ): Effect.Effect<User, UserError> {
    return Effect.gen(function* () {
      const userEffect = this.userService.getUserById(id);
      const timeoutEffect = Effect.fail(
        new TimeoutError({
          message: "Operation timed out",
          timeoutMs,
        })
      ).pipe(Effect.delay(timeoutMs));

      const result = yield* Effect.race(userEffect, timeoutEffect);
      return result;
    });
  }
}
```

## Implementation Details

### 1. ForstEffect Client

```typescript
// Core ForstEffect client
export class ForstEffect {
  private static client: ForstClient;

  static initialize(config: ForstConfig) {
    this.client = new ForstClient(config);
  }

  static call<T, E extends Data.TaggedError.Any>(
    functionName: string,
    args: any
  ): Effect.Effect<T, E> {
    return Effect.tryPromise({
      try: () => this.client.callFunction<T>(functionName, args),
      catch: (error) => this.convertError(error) as E,
    });
  }

  private static convertError(error: any): Data.TaggedError.Any {
    // Convert Forst errors to Effect tagged errors
    if (error._tag) {
      return error;
    }

    // Convert generic errors
    return new UnknownError({
      message: error.message || "Unknown error",
      originalError: error,
    });
  }
}
```

### 2. Automatic Service Generation

```typescript
// Auto-generated from Forst functions
export class AutoGeneratedUserService extends Data.TaggedClass(
  "AutoGeneratedUserService"
)<{}> {
  // Generated from: func getUserById(id: Int) User
  getUserById(id: number): Effect.Effect<User, UserError> {
    return ForstEffect.call("getUserById", { id });
  }

  // Generated from: func createUser(user: User) User
  createUser(user: CreateUserInput): Effect.Effect<User, UserError> {
    return ForstEffect.call("createUser", user);
  }

  // Generated from: func validateUser(user: User) Bool
  validateUser(user: User): Effect.Effect<boolean, ValidationError> {
    return ForstEffect.call("validateUser", user);
  }
}
```

### 3. Type Generation

```typescript
// Auto-generated TypeScript types from Forst
export interface User {
  readonly id: number;
  readonly name: string;
  readonly email: string;
}

export interface CreateUserInput {
  readonly name: string;
  readonly email: string;
}

// Error types with proper tagging
export class DatabaseError extends Data.TaggedError("DatabaseError")<{
  readonly message: string;
  readonly code: number;
}> {}

export class NotFoundError extends Data.TaggedError("NotFoundError")<{
  readonly message: string;
  readonly id: number;
}> {}

export class ValidationError extends Data.TaggedError("ValidationError")<{
  readonly field: string;
  readonly value: string;
  readonly constraint: string;
}> {}

// Union types for function error handling
export type UserError = DatabaseError | NotFoundError | ValidationError;
```

## Usage Examples

### 1. Basic Usage

```typescript
// Simple Effect usage with Forst functions
const program = Effect.gen(function* () {
  const userService = new UserService({});

  // Get user using Forst function
  const user = yield* userService.getUserById(123);

  // Validate user using Forst validation
  const isValid = yield* userService.validateUser(user);

  if (!isValid) {
    return yield* Effect.fail(
      new ValidationError({
        field: "user",
        value: JSON.stringify(user),
        constraint: "invalid_user",
      })
    );
  }

  return user;
});

// Run the program
const result = await Effect.runPromise(program);
```

### 2. Error Handling

```typescript
// Comprehensive error handling
const programWithErrorHandling = Effect.gen(function* () {
  const userService = new UserService({});

  try {
    const user = yield* userService.getUserById(123);
    return user;
  } catch (error) {
    // Handle specific Forst errors
    switch (error._tag) {
      case "NotFoundError":
        console.log(`User ${error.id} not found`);
        return yield* Effect.succeed(null);
      case "DatabaseError":
        console.log(`Database error: ${error.message}`);
        return yield* Effect.fail(error);
      case "ValidationError":
        console.log(`Validation error: ${error.field} - ${error.constraint}`);
        return yield* Effect.fail(error);
      default:
        return yield* Effect.fail(error);
    }
  }
});
```

### 3. Resource Management

```typescript
// Resource management with Forst functions
const programWithResources = Effect.gen(function* () {
  const dbService = new DatabaseEffectService({});

  return yield* dbService.withTransaction((tx) => {
    return Effect.gen(function* () {
      // Use transaction for multiple operations
      const user = yield* ForstEffect.call("createUser", {
        name: "John",
        email: "john@example.com",
      });
      const profile = yield* ForstEffect.call("createProfile", {
        userId: user.id,
        bio: "Hello world",
      });

      return { user, profile };
    });
  });
});
```

## Benefits

### 1. Native Go Performance

- **10x performance improvement** over TypeScript
- **Native concurrency** with goroutines
- **Memory efficiency** with Go's garbage collector
- **CPU optimization** with compiled Go code

### 2. Effect Developer Experience

- **Familiar Effect patterns** for TypeScript developers
- **Type safety** across the boundary
- **Tagged errors** for proper error handling
- **Composable services** for complex operations

### 3. Seamless Integration

- **Zero configuration** setup
- **Automatic type generation** from Forst code
- **Hot reloading** during development
- **Production ready** deployment

### 4. Best of Both Worlds

- **TypeScript DX** with Effect patterns
- **Go performance** for backend operations
- **Gradual migration** path
- **Clear separation** of concerns

## Conclusion

This approach positions Forst as the crucial intermediary that enables TypeScript developers to access Go's native capabilities through Effect services. Instead of translating Effect patterns to Go, we expose Go-native Forst code as Effect services that feel natural to TypeScript developers.

**Key advantages:**

- Leverages Go's native strengths (goroutines, channels, interfaces)
- Preserves Effect developer experience
- Enables gradual migration
- Provides clear separation of concerns
- Maintains type safety across boundaries

**The result:** TypeScript developers can use familiar Effect patterns while getting Go's performance benefits through a seamless sidecar integration.
