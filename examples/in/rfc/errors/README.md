# Forst Error System

## Overview

This directory contains comprehensive documentation and specifications for Forst's error handling system, designed to integrate seamlessly with Effect's tagged error system and modern observability tools.

## Documents

### Core Architecture

- **[00-error-system-architecture.md](./00-error-system-architecture.md)** - Complete error system architecture and implementation guide

### Related language design

- **[optionals hub](../optionals/00-crystal-inspired-optionals.md)** — Index of **`T | Nil`**, **`Result`**, and related topic docs (**01–09**).
- **[optionals / Result and error types](../optionals/02-result-and-error-types.md)** — How **`Result(Value, ErrorSubtype)`** ties **`Result`** to the **`Error`** hierarchy.

## Key Features

### 1. Structured Error Types

- **Hierarchical error system** with base and specific error types
- **Type-safe error creation** with factory functions
- **Rich error context** with tracing and metadata

### 2. Observability Integration

- **OpenTelemetry support** for distributed tracing
- **Structured logging** with trace IDs and context
- **Metrics collection** for error monitoring
- **Alerting integration** for critical errors

### 3. Effect Integration

- **Tagged error conversion** from Forst to Effect
- **Type-safe error handling** in TypeScript
- **Effect.gen patterns** for error composition
- **Resource management** for error handling

### 4. Modern Practices

- **Circuit breakers** for fault tolerance
- **Retry strategies** with exponential backoff
- **Error classification** (retryable vs non-retryable)
- **Graceful degradation** patterns

## Error Type Hierarchy

```
Error (Base)
├── ValidationError
├── DatabaseError
├── NetworkError
├── BusinessError
└── SystemError
```

## Implementation Phases

### Phase 1: Core Error System

- Base error types in Forst
- Error factory functions
- Context and tracing support

### Phase 2: Go Implementation

- Go error types and interfaces
- OpenTelemetry integration
- Error serialization

### Phase 3: TypeScript Integration

- Effect tagged errors
- Error conversion functions
- Effect error handling

### Phase 4: API and Observability

- HTTP error responses
- Status code mapping
- Observability tools integration

### Phase 5: Testing and Documentation

- Unit and integration tests
- Documentation and examples
- Tutorials and guides

### Phase 6: Production Readiness

- Performance optimization
- Security review
- Production monitoring

## Quick Start

### 1. Define Error Types in Forst

```forst
error ValidationError extends Error {
    field: String
    value: String?
    constraint: String
}
```

### 2. Use Error Factory Functions

```forst
func validateEmail(email: String) (String, Error) {
    if !isValidEmailFormat(email) {
        return "", newValidationError("email", email, "email_format")
    }
    return email, nil
}
```

### 3. Handle Errors in TypeScript

```typescript
import { Effect } from "effect";
import { ForstService } from "@forst/effect";

const program = Effect.gen(function* () {
  const result = yield* ForstService.processUser(123);
  return result;
});

const result = await Effect.runPromise(program);
```

## Benefits

### For Developers

- **Type safety** with compile-time error checking
- **Clear error messages** with actionable information
- **Easy error handling** with Effect patterns
- **Rich debugging context** with trace IDs

### For Operations

- **Distributed tracing** across services
- **Error monitoring** and alerting
- **Performance tracking** and metrics
- **Fault tolerance** with circuit breakers

### For Production

- **Reliable error handling** with retry strategies
- **Observability** with OpenTelemetry
- **Monitoring** with structured logs and metrics
- **Alerting** for critical errors

## Best Practices

1. **Always include context** - Every error should have trace ID and relevant context
2. **Use specific error types** - Create specific types for different scenarios
3. **Include retry information** - Clearly indicate if an error is retryable
4. **Provide actionable messages** - Error messages should help developers understand what went wrong
5. **Maintain error hierarchy** - Use inheritance to create clear error relationships

## Integration

### With Effect

- Seamless tagged error conversion
- Effect.gen error handling patterns
- Resource management for error handling
- Composition of error handling logic

### With OpenTelemetry

- Automatic error recording in spans
- Error attributes and context
- Distributed tracing across services
- Metrics collection for error rates

### With Modern Tools

- Structured logging with ELK stack
- Metrics with Prometheus and Grafana
- Alerting with PagerDuty or similar
- Monitoring with DataDog or similar

This error system positions Forst as a production-ready platform with excellent error handling that integrates seamlessly with modern observability tools and Effect's error system.


