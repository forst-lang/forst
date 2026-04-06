# Comprehensive Error Handling: Forst → Effect Integration

## Overview

This document provides a comprehensive approach to error handling in Forst that seamlessly integrates with Effect's tagged error system, modern microservices practices, and observability tools like OpenTelemetry. The focus is on creating a robust error API that enforces good practices and enables excellent developer experience.

**Related:** [19-forst-effect-npm-observability.md](./19-forst-effect-npm-observability.md) packages the Effect-facing npm story, trace/log correlation across the boundary, and open design items for `@forst/effect`.

## Core Principles

### 1. **Structured Error Types**

- Every error has a clear type and context
- Errors are serializable and traceable
- Error hierarchy enables proper handling

### 2. **Observability First**

- Built-in tracing and logging
- OpenTelemetry integration
- Distributed tracing support

### 3. **Type Safety**

- Compile-time error type checking
- Runtime error validation
- Seamless TypeScript integration

### 4. **Developer Experience**

- Clear error messages
- Easy error handling patterns
- Excellent debugging support

## Forst Error System Design

### 1. Base Error Types

```forst
// Base error types in Forst
error Error {
    message: String
    code: String
    timestamp: Int64
    traceId: String?
    spanId: String?
    context: Map[String, String]?
}

error ValidationError extends Error {
    field: String
    value: String?
    constraint: String
}

error DatabaseError extends Error {
    operation: String
    table: String?
    query: String?
    constraint: String?
}

error NetworkError extends Error {
    url: String?
    statusCode: Int?
    method: String?
    timeout: Bool
}

error BusinessError extends Error {
    domain: String
    operation: String
    resourceId: String?
}

error SystemError extends Error {
    component: String
    severity: Severity
    retryable: Bool
}

enum Severity {
    Low
    Medium
    High
    Critical
}
```

### 2. Error Context and Tracing

```forst
// Error context and tracing support
struct ErrorContext {
    traceId: String
    spanId: String
    userId: String?
    requestId: String?
    service: String
    version: String
    environment: String
    metadata: Map[String, String]
}

func createErrorContext() ErrorContext {
    return ErrorContext{
        traceId: generateTraceId(),
        spanId: generateSpanId(),
        service: "forst-service",
        version: getVersion(),
        environment: getEnvironment(),
        metadata: Map[String, String]{}
    }
}

func enrichError(error: Error, context: ErrorContext) Error {
    return Error{
        message: error.message,
        code: error.code,
        timestamp: getCurrentTimestamp(),
        traceId: context.traceId,
        spanId: context.spanId,
        context: mergeMaps(error.context, context.metadata)
    }
}
```

### 3. Error Factory Functions

```forst
// Error factory functions
func newValidationError(field: String, value: String?, constraint: String) ValidationError {
    context := createErrorContext()

    return ValidationError{
        message: "Validation failed for field: " + field,
        code: "VALIDATION_ERROR",
        timestamp: getCurrentTimestamp(),
        traceId: context.traceId,
        spanId: context.spanId,
        field: field,
        value: value,
        constraint: constraint,
        context: context.metadata
    }
}

func newDatabaseError(operation: String, table: String?, query: String?, err: Error) DatabaseError {
    context := createErrorContext()

    return DatabaseError{
        message: "Database operation failed: " + err.message,
        code: "DATABASE_ERROR",
        timestamp: getCurrentTimestamp(),
        traceId: context.traceId,
        spanId: context.spanId,
        operation: operation,
        table: table,
        query: query,
        context: context.metadata
    }
}

func newNetworkError(url: String?, statusCode: Int?, method: String?, timeout: Bool) NetworkError {
    context := createErrorContext()

    return NetworkError{
        message: "Network request failed",
        code: "NETWORK_ERROR",
        timestamp: getCurrentTimestamp(),
        traceId: context.traceId,
        spanId: context.spanId,
        url: url,
        statusCode: statusCode,
        method: method,
        timeout: timeout,
        context: context.metadata
    }
}

func newBusinessError(domain: String, operation: String, resourceId: String?, message: String) BusinessError {
    context := createErrorContext()

    return BusinessError{
        message: message,
        code: "BUSINESS_ERROR",
        timestamp: getCurrentTimestamp(),
        traceId: context.traceId,
        spanId: context.spanId,
        domain: domain,
        operation: operation,
        resourceId: resourceId,
        context: context.metadata
    }
}

func newSystemError(component: String, severity: Severity, retryable: Bool, message: String) SystemError {
    context := createErrorContext()

    return SystemError{
        message: message,
        code: "SYSTEM_ERROR",
        timestamp: getCurrentTimestamp(),
        traceId: context.traceId,
        spanId: context.spanId,
        component: component,
        severity: severity,
        retryable: retryable,
        context: context.metadata
    }
}
```

### 4. Error Handling Patterns

```forst
// Error handling patterns
func handleError(error: Error) Error {
    // Log error with context
    logError(error)

    // Record metrics
    recordErrorMetrics(error)

    // Check if error should be retried
    if isRetryableError(error) {
        scheduleRetry(error)
    }

    // Check if error should trigger alert
    if shouldAlert(error) {
        sendAlert(error)
    }

    return error
}

func isRetryableError(error: Error) Bool {
    switch error {
    case ValidationError:
        return false
    case DatabaseError:
        return true
    case NetworkError:
        return !error.timeout
    case BusinessError:
        return false
    case SystemError:
        return error.retryable
    default:
        return false
    }
}

func shouldAlert(error: Error) Bool {
    switch error {
    case SystemError:
        return error.severity == Critical || error.severity == High
    case DatabaseError:
        return true
    case NetworkError:
        return error.statusCode >= 500
    default:
        return false
    }
}
```

## Go Implementation

### 1. Error Types in Go

```go
// Generated Go error types
package errors

import (
    "encoding/json"
    "fmt"
    "time"
)

// Base error interface
type ForstError interface {
    Error() string
    Code() string
    Timestamp() int64
    TraceID() string
    SpanID() string
    Context() map[string]string
    ToJSON() string
}

// Base error implementation
type BaseError struct {
    Message   string            `json:"message"`
    Code      string            `json:"code"`
    Timestamp int64             `json:"timestamp"`
    TraceID   string            `json:"traceId"`
    SpanID    string            `json:"spanId"`
    Context   map[string]string `json:"context,omitempty"`
}

func (e *BaseError) Error() string {
    return e.Message
}

func (e *BaseError) Code() string {
    return e.Code
}

func (e *BaseError) Timestamp() int64 {
    return e.Timestamp
}

func (e *BaseError) TraceID() string {
    return e.TraceID
}

func (e *BaseError) SpanID() string {
    return e.SpanID
}

func (e *BaseError) Context() map[string]string {
    return e.Context
}

func (e *BaseError) ToJSON() string {
    data, _ := json.Marshal(e)
    return string(data)
}

// Validation error
type ValidationError struct {
    BaseError
    Field      string `json:"field"`
    Value      string `json:"value,omitempty"`
    Constraint string `json:"constraint"`
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation failed for field %s: %s", e.Field, e.Message)
}

// Database error
type DatabaseError struct {
    BaseError
    Operation  string `json:"operation"`
    Table      string `json:"table,omitempty"`
    Query      string `json:"query,omitempty"`
    Constraint string `json:"constraint,omitempty"`
}

func (e *DatabaseError) Error() string {
    return fmt.Sprintf("database error in %s: %s", e.Operation, e.Message)
}

// Network error
type NetworkError struct {
    BaseError
    URL        string `json:"url,omitempty"`
    StatusCode int    `json:"statusCode,omitempty"`
    Method     string `json:"method,omitempty"`
    Timeout    bool   `json:"timeout"`
}

func (e *NetworkError) Error() string {
    return fmt.Sprintf("network error: %s", e.Message)
}

// Business error
type BusinessError struct {
    BaseError
    Domain     string `json:"domain"`
    Operation  string `json:"operation"`
    ResourceID string `json:"resourceId,omitempty"`
}

func (e *BusinessError) Error() string {
    return fmt.Sprintf("business error in %s.%s: %s", e.Domain, e.Operation, e.Message)
}

// System error
type SystemError struct {
    BaseError
    Component string `json:"component"`
    Severity  string `json:"severity"`
    Retryable bool   `json:"retryable"`
}

func (e *SystemError) Error() string {
    return fmt.Sprintf("system error in %s: %s", e.Component, e.Message)
}
```

### 2. Error Factory Functions

```go
// Error factory functions
package errors

import (
    "crypto/rand"
    "encoding/hex"
    "time"
)

func NewValidationError(field, value, constraint string) *ValidationError {
    return &ValidationError{
        BaseError: BaseError{
            Message:   fmt.Sprintf("Validation failed for field: %s", field),
            Code:      "VALIDATION_ERROR",
            Timestamp: time.Now().Unix(),
            TraceID:   generateTraceID(),
            SpanID:    generateSpanID(),
            Context:   make(map[string]string),
        },
        Field:      field,
        Value:      value,
        Constraint: constraint,
    }
}

func NewDatabaseError(operation, table, query string, err error) *DatabaseError {
    return &DatabaseError{
        BaseError: BaseError{
            Message:   fmt.Sprintf("Database operation failed: %s", err.Error()),
            Code:      "DATABASE_ERROR",
            Timestamp: time.Now().Unix(),
            TraceID:   generateTraceID(),
            SpanID:    generateSpanID(),
            Context:   make(map[string]string),
        },
        Operation: operation,
        Table:     table,
        Query:     query,
    }
}

func NewNetworkError(url string, statusCode int, method string, timeout bool) *NetworkError {
    return &NetworkError{
        BaseError: BaseError{
            Message:   "Network request failed",
            Code:      "NETWORK_ERROR",
            Timestamp: time.Now().Unix(),
            TraceID:   generateTraceID(),
            SpanID:    generateSpanID(),
            Context:   make(map[string]string),
        },
        URL:        url,
        StatusCode: statusCode,
        Method:     method,
        Timeout:    timeout,
    }
}

func NewBusinessError(domain, operation, resourceID, message string) *BusinessError {
    return &BusinessError{
        BaseError: BaseError{
            Message:   message,
            Code:      "BUSINESS_ERROR",
            Timestamp: time.Now().Unix(),
            TraceID:   generateTraceID(),
            SpanID:    generateSpanID(),
            Context:   make(map[string]string),
        },
        Domain:     domain,
        Operation:  operation,
        ResourceID: resourceID,
    }
}

func NewSystemError(component, severity string, retryable bool, message string) *SystemError {
    return &SystemError{
        BaseError: BaseError{
            Message:   message,
            Timestamp: time.Now().Unix(),
            TraceID:   generateTraceID(),
            SpanID:    generateSpanID(),
            Context:   make(map[string]string),
        },
        Component: component,
        Severity:  severity,
        Retryable: retryable,
    }
}

func generateTraceID() string {
    bytes := make([]byte, 16)
    rand.Read(bytes)
    return hex.EncodeToString(bytes)
}

func generateSpanID() string {
    bytes := make([]byte, 8)
    rand.Read(bytes)
    return hex.EncodeToString(bytes)
}
```

### 3. OpenTelemetry Integration

```go
// OpenTelemetry integration
package tracing

import (
    "context"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/trace"
)

type ErrorTracer struct {
    tracer trace.Tracer
}

func NewErrorTracer() *ErrorTracer {
    return &ErrorTracer{
        tracer: otel.Tracer("forst-errors"),
    }
}

func (et *ErrorTracer) RecordError(ctx context.Context, err ForstError) {
    span := trace.SpanFromContext(ctx)
    if span.IsRecording() {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())

        // Add error attributes
        span.SetAttributes(
            attribute.String("error.code", err.Code()),
            attribute.String("error.trace_id", err.TraceID()),
            attribute.String("error.span_id", err.SpanID()),
        )

        // Add context attributes
        for key, value := range err.Context() {
            span.SetAttributes(attribute.String("error.context."+key, value))
        }
    }
}

func (et *ErrorTracer) RecordErrorWithSpan(ctx context.Context, err ForstError, spanName string) {
    ctx, span := et.tracer.Start(ctx, spanName)
    defer span.End()

    et.RecordError(ctx, err)
}
```

## TypeScript/Effect Integration

### 1. Effect Tagged Errors

```typescript
// Effect tagged errors
import { Data } from "effect";

// Base error class
export class ForstError extends Data.TaggedError("ForstError")<{
  message: string;
  code: string;
  timestamp: number;
  traceId: string;
  spanId: string;
  context?: Record<string, string>;
}> {}

// Specific error types
export class ValidationError extends Data.TaggedError("ValidationError")<{
  message: string;
  code: string;
  timestamp: number;
  traceId: string;
  spanId: string;
  context?: Record<string, string>;
  field: string;
  value?: string;
  constraint: string;
}> {}

export class DatabaseError extends Data.TaggedError("DatabaseError")<{
  message: string;
  code: string;
  timestamp: number;
  traceId: string;
  spanId: string;
  context?: Record<string, string>;
  operation: string;
  table?: string;
  query?: string;
  constraint?: string;
}> {}

export class NetworkError extends Data.TaggedError("NetworkError")<{
  message: string;
  code: string;
  timestamp: number;
  traceId: string;
  spanId: string;
  context?: Record<string, string>;
  url?: string;
  statusCode?: number;
  method?: string;
  timeout: boolean;
}> {}

export class BusinessError extends Data.TaggedError("BusinessError")<{
  message: string;
  code: string;
  timestamp: number;
  traceId: string;
  spanId: string;
  context?: Record<string, string>;
  domain: string;
  operation: string;
  resourceId?: string;
}> {}

export class SystemError extends Data.TaggedError("SystemError")<{
  message: string;
  code: string;
  timestamp: number;
  traceId: string;
  spanId: string;
  context?: Record<string, string>;
  component: string;
  severity: "Low" | "Medium" | "High" | "Critical";
  retryable: boolean;
}> {}
```

### 2. Error Conversion Functions

```typescript
// Error conversion functions
export function convertForstError(error: any): ForstError {
  if (error.code === "VALIDATION_ERROR") {
    return new ValidationError({
      message: error.message,
      code: error.code,
      timestamp: error.timestamp,
      traceId: error.traceId,
      spanId: error.spanId,
      context: error.context,
      field: error.field,
      value: error.value,
      constraint: error.constraint,
    });
  }

  if (error.code === "DATABASE_ERROR") {
    return new DatabaseError({
      message: error.message,
      code: error.code,
      timestamp: error.timestamp,
      traceId: error.traceId,
      spanId: error.spanId,
      context: error.context,
      operation: error.operation,
      table: error.table,
      query: error.query,
      constraint: error.constraint,
    });
  }

  if (error.code === "NETWORK_ERROR") {
    return new NetworkError({
      message: error.message,
      code: error.code,
      timestamp: error.timestamp,
      traceId: error.traceId,
      spanId: error.spanId,
      context: error.context,
      url: error.url,
      statusCode: error.statusCode,
      method: error.method,
      timeout: error.timeout,
    });
  }

  if (error.code === "BUSINESS_ERROR") {
    return new BusinessError({
      message: error.message,
      code: error.code,
      timestamp: error.timestamp,
      traceId: error.traceId,
      spanId: error.spanId,
      context: error.context,
      domain: error.domain,
      operation: error.operation,
      resourceId: error.resourceId,
    });
  }

  if (error.code === "SYSTEM_ERROR") {
    return new SystemError({
      message: error.message,
      code: error.code,
      timestamp: error.timestamp,
      traceId: error.traceId,
      spanId: error.spanId,
      context: error.context,
      component: error.component,
      severity: error.severity,
      retryable: error.retryable,
    });
  }

  // Fallback to base error
  return new ForstError({
    message: error.message || "Unknown error",
    code: error.code || "UNKNOWN_ERROR",
    timestamp: error.timestamp || Date.now(),
    traceId: error.traceId || "",
    spanId: error.spanId || "",
    context: error.context,
  });
}
```

### 3. Effect Error Handling

```typescript
// Effect error handling
import { Effect, pipe } from "effect";

export class ForstErrorHandler {
  static handleError = (
    error: ForstError
  ): Effect.Effect<never, ForstError, never> =>
    Effect.gen(function* () {
      // Log error
      yield* Effect.logError("Forst error occurred", error);

      // Record metrics
      yield* Effect.sync(() => {
        // Record error metrics
        console.log(`Error recorded: ${error.code} - ${error.message}`);
      });

      // Check if error should be retried
      if (ForstErrorHandler.isRetryableError(error)) {
        yield* Effect.log("Error is retryable, scheduling retry");
      }

      // Check if error should trigger alert
      if (ForstErrorHandler.shouldAlert(error)) {
        yield* Effect.log("Error should trigger alert");
      }

      return error;
    });

  static isRetryableError = (error: ForstError): boolean => {
    if (error instanceof ValidationError) return false;
    if (error instanceof DatabaseError) return true;
    if (error instanceof NetworkError) return !error.timeout;
    if (error instanceof BusinessError) return false;
    if (error instanceof SystemError) return error.retryable;
    return false;
  };

  static shouldAlert = (error: ForstError): boolean => {
    if (error instanceof SystemError) {
      return error.severity === "Critical" || error.severity === "High";
    }
    if (error instanceof DatabaseError) return true;
    if (error instanceof NetworkError) {
      return (error.statusCode ?? 0) >= 500;
    }
    return false;
  };
}
```

### 4. Effect Services with Error Handling

```typescript
// Effect services with error handling
export class ForstService {
  static processUser = (id: number) =>
    Effect.gen(function* () {
      // Call Forst function
      const result = yield* Effect.tryPromise({
        try: () => forstClient.processUser(id),
        catch: (error) => convertForstError(error),
      });

      // Handle different error types
      if (result.success) {
        return result.data;
      } else {
        const forstError = convertForstError(result.error);
        yield* ForstErrorHandler.handleError(forstError);
        return yield* Effect.fail(forstError);
      }
    });

  static processUserWithRetry = (id: number) =>
    Effect.retry(ForstService.processUser(id), {
      times: 3,
      delay: "exponential",
    });

  static processUserWithTimeout = (id: number) =>
    Effect.timeout(ForstService.processUser(id), "5 seconds");

  static processUserWithCircuitBreaker = (id: number) =>
    Effect.gen(function* () {
      // Circuit breaker logic
      const result = yield* Effect.tryPromise({
        try: () => circuitBreaker.call(() => forstClient.processUser(id)),
        catch: (error) => convertForstError(error),
      });

      if (result.success) {
        return result.data;
      } else {
        const forstError = convertForstError(result.error);
        yield* ForstErrorHandler.handleError(forstError);
        return yield* Effect.fail(forstError);
      }
    });
}
```

## API Design

### 1. Error Response Format

```typescript
// Standardized error response format
export interface ForstErrorResponse {
  success: false;
  error: {
    type: string;
    message: string;
    code: string;
    timestamp: number;
    traceId: string;
    spanId: string;
    context?: Record<string, string>;
    // Type-specific fields
    field?: string;
    value?: string;
    constraint?: string;
    operation?: string;
    table?: string;
    query?: string;
    url?: string;
    statusCode?: number;
    method?: string;
    timeout?: boolean;
    domain?: string;
    resourceId?: string;
    component?: string;
    severity?: string;
    retryable?: boolean;
  };
}
```

### 2. Error Handling Middleware

```typescript
// Error handling middleware
export const errorHandlingMiddleware = (
  req: Request,
  res: Response,
  next: NextFunction
) => {
  try {
    // Process request
    next();
  } catch (error) {
    const forstError = convertForstError(error);

    // Log error
    console.error("Request error:", forstError);

    // Record metrics
    recordErrorMetrics(forstError);

    // Send error response
    res.status(getHttpStatusFromError(forstError)).json({
      success: false,
      error: {
        type: forstError._tag,
        message: forstError.message,
        code: forstError.code,
        timestamp: forstError.timestamp,
        traceId: forstError.traceId,
        spanId: forstError.spanId,
        context: forstError.context,
        ...getTypeSpecificFields(forstError),
      },
    });
  }
};

function getHttpStatusFromError(error: ForstError): number {
  if (error instanceof ValidationError) return 400;
  if (error instanceof DatabaseError) return 500;
  if (error instanceof NetworkError) return 502;
  if (error instanceof BusinessError) return 422;
  if (error instanceof SystemError) return 500;
  return 500;
}

function getTypeSpecificFields(error: ForstError): Record<string, any> {
  if (error instanceof ValidationError) {
    return {
      field: error.field,
      value: error.value,
      constraint: error.constraint,
    };
  }
  if (error instanceof DatabaseError) {
    return {
      operation: error.operation,
      table: error.table,
      query: error.query,
      constraint: error.constraint,
    };
  }
  if (error instanceof NetworkError) {
    return {
      url: error.url,
      statusCode: error.statusCode,
      method: error.method,
      timeout: error.timeout,
    };
  }
  if (error instanceof BusinessError) {
    return {
      domain: error.domain,
      operation: error.operation,
      resourceId: error.resourceId,
    };
  }
  if (error instanceof SystemError) {
    return {
      component: error.component,
      severity: error.severity,
      retryable: error.retryable,
    };
  }
  return {};
}
```

## Observability Integration

### 1. OpenTelemetry Integration

```typescript
// OpenTelemetry integration
import { trace, context } from "@opentelemetry/api";

export class OpenTelemetryErrorHandler {
  static recordError = (error: ForstError): Effect.Effect<void, never, never> =>
    Effect.sync(() => {
      const span = trace.getActiveSpan();
      if (span) {
        span.recordException(error);
        span.setStatus({ code: 1, message: error.message });

        // Add error attributes
        span.setAttributes({
          "error.code": error.code,
          "error.trace_id": error.traceId,
          "error.span_id": error.spanId,
          "error.type": error._tag,
        });

        // Add context attributes
        if (error.context) {
          Object.entries(error.context).forEach(([key, value]) => {
            span.setAttributes({ [`error.context.${key}`]: value });
          });
        }
      }
    });

  static recordErrorWithSpan = (error: ForstError, spanName: string) =>
    Effect.gen(function* () {
      const tracer = trace.getTracer("forst-errors");
      const span = tracer.startSpan(spanName);

      yield* Effect.sync(() => {
        span.recordException(error);
        span.setStatus({ code: 1, message: error.message });
        span.end();
      });
    });
}
```

### 2. Metrics Integration

```typescript
// Metrics integration
export class ErrorMetrics {
  static recordError = (error: ForstError): Effect.Effect<void, never, never> =>
    Effect.sync(() => {
      // Record error counter
      console.log(`Error counter: ${error.code}`);

      // Record error rate
      console.log(`Error rate: ${error.code}`);

      // Record error duration
      console.log(`Error duration: ${error.code}`);
    });

  static recordErrorRate = (
    error: ForstError
  ): Effect.Effect<void, never, never> =>
    Effect.sync(() => {
      // Record error rate metrics
      console.log(`Error rate recorded: ${error.code}`);
    });
}
```

## Testing Strategy

### 1. Error Testing

```typescript
// Error testing
import { describe, it, expect } from "vitest";
import { Effect } from "effect";

describe("Forst Error Handling", () => {
  it("should convert validation error correctly", () => {
    const error = {
      code: "VALIDATION_ERROR",
      message: "Validation failed",
      field: "email",
      constraint: "email_format",
    };

    const forstError = convertForstError(error);

    expect(forstError).toBeInstanceOf(ValidationError);
    expect(forstError.field).toBe("email");
    expect(forstError.constraint).toBe("email_format");
  });

  it("should handle retryable errors", () => {
    const error = new DatabaseError({
      message: "Connection failed",
      code: "DATABASE_ERROR",
      timestamp: Date.now(),
      traceId: "trace-123",
      spanId: "span-456",
      operation: "SELECT",
    });

    expect(ForstErrorHandler.isRetryableError(error)).toBe(true);
  });

  it("should handle non-retryable errors", () => {
    const error = new ValidationError({
      message: "Invalid email",
      code: "VALIDATION_ERROR",
      timestamp: Date.now(),
      traceId: "trace-123",
      spanId: "span-456",
      field: "email",
      constraint: "email_format",
    });

    expect(ForstErrorHandler.isRetryableError(error)).toBe(false);
  });
});
```

### 2. Integration Testing

```typescript
// Integration testing
describe("Forst Service Integration", () => {
  it("should handle Forst errors correctly", async () => {
    const program = Effect.gen(function* () {
      const result = yield* ForstService.processUser(123);
      return result;
    });

    const result = await Effect.runPromise(program);

    if (result._tag === "Left") {
      expect(result.left).toBeInstanceOf(ForstError);
    } else {
      expect(result.right).toBeDefined();
    }
  });
});
```

## Best Practices

### 1. Error Design Principles

1. **Always include context** - Every error should have trace ID, span ID, and relevant context
2. **Use specific error types** - Don't use generic errors, create specific types for different scenarios
3. **Include retry information** - Clearly indicate if an error is retryable
4. **Provide actionable messages** - Error messages should help developers understand what went wrong
5. **Maintain error hierarchy** - Use inheritance to create clear error relationships

### 2. Error Handling Patterns

1. **Fail fast** - Don't continue processing after encountering an error
2. **Log errors** - Always log errors with full context
3. **Record metrics** - Track error rates and types
4. **Use circuit breakers** - Prevent cascading failures
5. **Implement retries** - Handle transient errors gracefully

### 3. Observability Best Practices

1. **Use structured logging** - Include trace IDs and context in all logs
2. **Record error metrics** - Track error rates, types, and patterns
3. **Implement distributed tracing** - Follow requests across service boundaries
4. **Set up alerting** - Alert on critical errors and error rate spikes
5. **Monitor error trends** - Track error patterns over time

## Conclusion

This comprehensive error handling system provides:

### **Key Benefits**

1. **Type Safety** - Compile-time and runtime error type checking
2. **Observability** - Built-in tracing, logging, and metrics
3. **Developer Experience** - Clear error messages and easy handling
4. **Production Ready** - Circuit breakers, retries, and alerting
5. **Effect Integration** - Seamless integration with Effect's error system

### **Modern Practices**

1. **Structured Errors** - Clear error types with context
2. **Distributed Tracing** - OpenTelemetry integration
3. **Error Classification** - Retryable vs non-retryable errors
4. **Circuit Breakers** - Fault tolerance patterns
5. **Metrics and Alerting** - Production monitoring

This approach positions Forst as a production-ready system with excellent error handling that integrates seamlessly with modern observability tools and Effect's error system.


