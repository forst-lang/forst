# Native Microservice Sidecar: Forst → Go → Native Code

## Overview

This document explores compiling Forst code to Go, then to native code, and running it as a local microservice sidecar that communicates via HTTP while executing in its own process separate from Node.js's single-threaded runtime.

## Core Concept

**Forst → Go → Native Code → HTTP Microservice Sidecar**

Instead of Node.js addons, Forst compiles to native Go binaries that run as independent microservices, communicating with Node.js applications via HTTP while leveraging Go's full concurrency capabilities.

## Architecture

### 1. Compilation Pipeline

```
Forst Source (.ft) → Go Code → Native Binary → HTTP Microservice → Node.js Client
```

### 2. Process Separation

```
┌─────────────────┐    HTTP     ┌──────────────────┐
│   Node.js App   │ ←────────→ │  Forst Service   │
│  (Single-thread)│            │  (Multi-thread)  │
│                 │            │  (Native Go)     │
└─────────────────┘            └──────────────────┘
```

## Existing Code Support Analysis

### 1. Forst Compiler Support

#### ✅ **Well Supported**

- **Go compilation** - Forst already compiles to Go
- **Type system** - Full type safety maintained
- **Error handling** - `ensure` statements work perfectly
- **Function definitions** - All Forst functions compile to Go

#### ⚠️ **Needs Enhancement**

- **HTTP server generation** - Need to generate HTTP endpoints
- **Request/response handling** - Need JSON serialization
- **Service discovery** - Need automatic endpoint registration
- **Health checks** - Need health monitoring

### 2. Go Runtime Support

#### ✅ **Excellent Support**

- **HTTP server** - `net/http` package
- **JSON handling** - `encoding/json` package
- **Concurrency** - Goroutines and channels
- **Error handling** - Go's error system
- **Logging** - Structured logging

#### ✅ **Performance Benefits**

- **True parallelism** - Multiple goroutines
- **No event loop** - No single-threaded bottleneck
- **Memory efficiency** - Go's garbage collector
- **Native performance** - Compiled binary

## Implementation Strategy

### 1. Forst to Go Compilation

```go
// Generated from Forst source
// example.ft -> example.go

package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
)

// User represents a user in the system
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// Request represents an HTTP request
type Request struct {
    Method string                 `json:"method"`
    Params map[string]interface{} `json:"params"`
}

// Response represents an HTTP response
type Response struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}

// processUser - Generated from Forst
func processUser(id int) (*User, error) {
    // Database query
    user, err := getUserById(id)
    if err != nil {
        return nil, fmt.Errorf("database error: %w", err)
    }

    // Validation
    if user.Name == "" {
        return nil, fmt.Errorf("validation error: name is required")
    }

    if user.Email == "" {
        return nil, fmt.Errorf("validation error: email is required")
    }

    return user, nil
}

func getUserById(id int) (*User, error) {
    // Mock database query
    if id <= 0 {
        return nil, fmt.Errorf("invalid user ID")
    }

    return &User{
        ID:    id,
        Name:  "John Doe",
        Email: "john@example.com",
    }, nil
}
```

### 2. HTTP Server Generation

```go
// Generated HTTP server
func main() {
    // Create HTTP server
    mux := http.NewServeMux()

    // Register Forst functions as HTTP endpoints
    mux.HandleFunc("/processUser", handleProcessUser)
    mux.HandleFunc("/validateUser", handleValidateUser)
    mux.HandleFunc("/createUser", handleCreateUser)
    mux.HandleFunc("/health", handleHealth)

    // Start server
    server := &http.Server{
        Addr:    ":8080",
        Handler: mux,
    }

    // Graceful shutdown
    go func() {
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Server failed to start: %v", err)
        }
    }()

    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Println("Shutting down server...")
    server.Close()
}

func handleProcessUser(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req Request
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    // Extract parameters
    id, ok := req.Params["id"].(float64)
    if !ok {
        http.Error(w, "Invalid id parameter", http.StatusBadRequest)
        return
    }

    // Call Forst function
    result, err := processUser(int(id))
    if err != nil {
        response := Response{
            Success: false,
            Error:   err.Error(),
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
        return
    }

    response := Response{
        Success: true,
        Data:    result,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

### 3. Node.js Client Generation

```javascript
// Generated Node.js client
import fetch from "node-fetch";

class ForstClient {
  constructor(baseUrl = "http://localhost:8080") {
    this.baseUrl = baseUrl;
  }

  async callFunction(name, params) {
    try {
      const response = await fetch(`${this.baseUrl}/${name}`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          method: name,
          params: params,
        }),
      });

      if (!response.ok) {
        throw new Error(`HTTP error: ${response.status}`);
      }

      const result = await response.json();
      return result;
    } catch (error) {
      return {
        success: false,
        error: error.message,
      };
    }
  }

  async processUser(id) {
    return this.callFunction("processUser", { id });
  }

  async validateUser(user) {
    return this.callFunction("validateUser", { user });
  }

  async createUser(user) {
    return this.callFunction("createUser", { user });
  }

  async health() {
    return this.callFunction("health", {});
  }
}

export default ForstClient;
```

### 4. TypeScript Declarations

```typescript
// Generated TypeScript declarations
export interface User {
  id: number;
  name: string;
  email: string;
}

export interface ForstResponse<T = any> {
  success: boolean;
  data?: T;
  error?: string;
}

export class ForstClient {
  constructor(baseUrl?: string);

  callFunction(
    name: string,
    params: Record<string, any>
  ): Promise<ForstResponse>;
  processUser(id: number): Promise<ForstResponse<User>>;
  validateUser(user: User): Promise<ForstResponse<User>>;
  createUser(user: Omit<User, "id">): Promise<ForstResponse<User>>;
  health(): Promise<ForstResponse>;
}
```

## Effect Integration Opportunities

### 1. Effect as HTTP Client

```typescript
// Effect-based HTTP client
import { Effect } from "effect";
import { ForstClient } from "@forst/client";

class EffectForstClient {
  constructor(private client: ForstClient) {}

  processUser(id: number) {
    return Effect.tryPromise({
      try: () => this.client.processUser(id),
      catch: (error) => new ForstError({ message: error.message }),
    });
  }

  validateUser(user: User) {
    return Effect.tryPromise({
      try: () => this.client.validateUser(user),
      catch: (error) => new ForstError({ message: error.message }),
    });
  }

  // Effect.gen for composition
  processUserWithValidation(id: number) {
    return Effect.gen(function* () {
      const user = yield* this.processUser(id);
      const validated = yield* this.validateUser(user.data);
      return validated;
    });
  }
}
```

### 2. Effect Services

```typescript
// Effect services for Forst functions
export class ForstService {
  static processUser = (id: number) =>
    Effect.tryPromise({
      try: () => forstClient.processUser(id),
      catch: (error) => new ForstError({ message: error.message }),
    });

  static validateUser = (user: User) =>
    Effect.tryPromise({
      try: () => forstClient.validateUser(user),
      catch: (error) => new ForstError({ message: error.message }),
    });

  static createUser = (user: Omit<User, "id">) =>
    Effect.tryPromise({
      try: () => forstClient.createUser(user),
      catch: (error) => new ForstError({ message: error.message }),
    });
}
```

### 3. Effect Error Handling

```typescript
// Effect error types for Forst
export class ForstError extends Data.TaggedError("ForstError")<{
  message: string;
  code?: number;
  details?: Record<string, any>;
}> {}

export class DatabaseError extends Data.TaggedError("DatabaseError")<{
  message: string;
  query?: string;
}> {}

export class ValidationError extends Data.TaggedError("ValidationError")<{
  field: string;
  message: string;
}> {}

// Error conversion
function convertForstError(error: any): Effect.Error {
  if (error.message.includes("database")) {
    return new DatabaseError({ message: error.message });
  }
  if (error.message.includes("validation")) {
    return new ValidationError({
      field: "unknown",
      message: error.message,
    });
  }
  return new ForstError({ message: error.message });
}
```

### 4. Effect Resource Management

```typescript
// Effect resource management for Forst client
export const ForstClientService = Effect.scoped(
  Effect.gen(function* () {
    const client = yield* Effect.sync(() => new ForstClient());

    yield* Effect.addFinalizer(() => Effect.sync(() => client.close?.()));

    return client;
  })
);

// Usage with Effect.gen
const processUserWithEffect = (id: number) =>
  Effect.gen(function* () {
    const client = yield* ForstClientService;
    const result = yield* Effect.tryPromise({
      try: () => client.processUser(id),
      catch: convertForstError,
    });
    return result;
  });
```

## Performance Benefits

### 1. True Parallelism

```go
// Go service can handle multiple requests concurrently
func handleProcessUser(w http.ResponseWriter, r *http.Request) {
    // Each request runs in its own goroutine
    go func() {
        // Process user in parallel
        result, err := processUser(id)
        // Send response
    }()
}
```

### 2. No Node.js Bottlenecks

- **No event loop** - Each request gets its own goroutine
- **No single-threaded execution** - True parallelism
- **No memory sharing** - Isolated process memory
- **No V8 limitations** - Native Go performance

### 3. Scalability

```go
// Horizontal scaling
func main() {
    // Can run multiple instances
    instances := os.Getenv("FORST_INSTANCES")
    if instances == "" {
        instances = "4" // Default to 4 instances
    }

    // Load balancing
    for i := 0; i < instances; i++ {
        go startServer(fmt.Sprintf(":%d", 8080+i))
    }
}
```

## Development Workflow

### 1. Forst Development

```bash
# Write Forst code
# example.ft
func processUser(id: Int) User {
    user := getUserById(id)
    ensure user != nil, "User not found"

    validated := validateUser(user)
    ensure validated, "User validation failed"

    return validated
}

# Compile to Go
forst compile --target=go --output=src/

# Generate HTTP server
forst generate --target=http-server --output=src/

# Build native binary
go build -o forst-service src/main.go

# Run service
./forst-service
```

### 2. Node.js Integration

```typescript
// Install Forst client
npm install @forst/client

// Use in Node.js
import { ForstClient } from '@forst/client';

const client = new ForstClient('http://localhost:8080');

const result = await client.processUser(123);
if (result.success) {
  console.log('User:', result.data);
} else {
  console.error('Error:', result.error);
}
```

### 3. Effect Integration

```typescript
// Use with Effect
import { Effect } from "effect";
import { ForstService } from "@forst/effect";

const program = Effect.gen(function* () {
  const user = yield* ForstService.processUser(123);
  const validated = yield* ForstService.validateUser(user.data);
  return validated;
});

const result = await Effect.runPromise(program);
```

## Deployment Strategies

### 1. Local Development

```bash
# Start Forst service
./forst-service

# Start Node.js app
npm start
```

### 2. Docker Deployment

```dockerfile
# Dockerfile.forst
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o forst-service .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/forst-service .
EXPOSE 8080
CMD ["./forst-service"]
```

```yaml
# docker-compose.yml
version: "3.8"
services:
  forst-service:
    build:
      context: .
      dockerfile: Dockerfile.forst
    ports:
      - "8080:8080"
    environment:
      - FORST_INSTANCES=4

  node-app:
    build: .
    ports:
      - "3000:3000"
    depends_on:
      - forst-service
    environment:
      - FORST_SERVICE_URL=http://forst-service:8080
```

### 3. Kubernetes Deployment

```yaml
# forst-service.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: forst-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: forst-service
  template:
    metadata:
      labels:
        app: forst-service
    spec:
      containers:
        - name: forst-service
          image: forst-service:latest
          ports:
            - containerPort: 8080
          resources:
            requests:
              memory: "128Mi"
              cpu: "100m"
            limits:
              memory: "512Mi"
              cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: forst-service
spec:
  selector:
    app: forst-service
  ports:
    - port: 8080
      targetPort: 8080
```

## Monitoring and Observability

### 1. Health Checks

```go
// Health check endpoint
func handleHealth(w http.ResponseWriter, r *http.Request) {
    health := map[string]interface{}{
        "status":    "healthy",
        "timestamp": time.Now().Unix(),
        "version":   "1.0.0",
        "uptime":    time.Since(startTime).Seconds(),
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(health)
}
```

### 2. Metrics

```go
// Prometheus metrics
import "github.com/prometheus/client_golang/prometheus"

var (
    requestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "forst_requests_total",
            Help: "Total number of requests",
        },
        []string{"method", "status"},
    )

    requestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "forst_request_duration_seconds",
            Help: "Request duration in seconds",
        },
        []string{"method"},
    )
)
```

### 3. Logging

```go
// Structured logging
import "github.com/sirupsen/logrus"

var log = logrus.New()

func handleProcessUser(w http.ResponseWriter, r *http.Request) {
    start := time.Now()

    log.WithFields(logrus.Fields{
        "method": "processUser",
        "id":     id,
    }).Info("Processing user request")

    // ... process request

    log.WithFields(logrus.Fields{
        "method":   "processUser",
        "duration": time.Since(start),
        "success":  err == nil,
    }).Info("User request completed")
}
```

## Comparison with Other Approaches

### vs Node.js Addons

| Aspect          | Native Microservice | Node.js Addon |
| --------------- | ------------------- | ------------- |
| **Performance** | Very High           | High          |
| **Concurrency** | True Parallelism    | Limited       |
| **Memory**      | Isolated            | Shared        |
| **Scalability** | Horizontal          | Vertical      |
| **Deployment**  | Complex             | Simple        |
| **Debugging**   | Separate Process    | Same Process  |

### vs Sidecar Pattern

| Aspect             | Native Microservice | Sidecar   |
| ------------------ | ------------------- | --------- |
| **Communication**  | HTTP                | IPC/HTTP  |
| **Performance**    | High                | Very High |
| **Complexity**     | Medium              | High      |
| **Scalability**    | Excellent           | Good      |
| **Resource Usage** | Higher              | Lower     |

## Conclusion

The **native microservice sidecar approach** provides significant benefits:

### **Advantages**

1. **True Parallelism** - No Node.js single-threaded bottleneck
2. **Go Performance** - Full native Go performance
3. **Scalability** - Horizontal scaling capabilities
4. **Isolation** - Separate process memory and execution
5. **Effect Integration** - Perfect for Effect-based applications

### **Trade-offs**

1. **Complexity** - More complex deployment and management
2. **Network Overhead** - HTTP communication vs direct calls
3. **Resource Usage** - Separate process memory usage
4. **Debugging** - More complex debugging across processes

### **Effect Integration Benefits**

1. **Perfect Error Handling** - Effect's error system works well with HTTP
2. **Resource Management** - Effect's resource management for HTTP clients
3. **Composition** - Effect.gen for composing Forst functions
4. **Services** - Effect services for Forst function organization

This approach positions Forst as a powerful microservice platform that can handle high-performance workloads while maintaining excellent integration with Effect-based TypeScript applications.


