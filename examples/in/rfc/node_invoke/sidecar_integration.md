---
Feature Name: sidecar-integration
Start Date: 2025-07-06
---

# Sidecar Integration for TypeScript Backend Performance

## Summary

Introduce a sidecar integration approach that enables gradual adoption of Forst for high-performance backend operations within TypeScript applications. This feature addresses the fundamental performance and memory issues that plague TypeScript backends by allowing developers to replace individual routes with Forst implementations without disrupting existing workflows.

The sidecar approach provides a seamless bridge that allows TypeScript applications to invoke Forst-compiled Go binaries for CPU-intensive, memory-heavy, or concurrent operations while maintaining the familiar TypeScript development experience. This enables teams to solve performance bottlenecks incrementally without requiring a complete rewrite.

## Motivation

TypeScript backends face significant challenges that limit their scalability:

1. **Memory management issues**: Difficult-to-handle memory spikes and memory leaks
2. **Poor concurrent performance**: Single-threaded event loop limits throughput
3. **CPU-intensive bottlenecks**: Complex business logic becomes performance bottlenecks
4. **Gradual migration barriers**: Rewriting entire applications is risky and time-consuming
5. **Development workflow disruption**: Teams can't afford to stop feature development

### Status Quo

Current TypeScript backends struggle with:

```typescript
// Memory-intensive data processing
app.post("/process-data", async (req, res) => {
  const data = req.body;

  // This can cause memory spikes and poor performance
  const processed = await heavyDataProcessing(data);
  const enriched = await enrichWithExternalData(processed);
  const validated = await validateComplexRules(enriched);

  res.json(validated);
});

// Concurrent request handling becomes bottleneck
app.get("/search", async (req, res) => {
  // Single-threaded processing limits throughput
  const results = await performComplexSearch(req.query);
  res.json(results);
});
```

These operations become performance bottlenecks and memory sinks, especially under load.

### Desired State

With sidecar integration, developers can gradually replace problematic routes:

```typescript
// Original TypeScript route (kept for non-critical paths)
app.post("/simple-endpoint", async (req, res) => {
  res.json({ status: "ok" });
});

// Replaced with Forst implementation for performance-critical operations
import { processData } from "@forst/sidecar";

app.post("/process-data", async (req, res) => {
  // This now runs in a high-performance Go binary
  const result = await processData(req.body);
  res.json(result);
});
```

The Forst implementation provides:

- **Memory efficiency**: Go's garbage collector handles memory spikes better
- **Concurrent performance**: Go's goroutines handle concurrent requests efficiently
- **CPU optimization**: Compiled Go code outperforms interpreted TypeScript
- **Gradual adoption**: Replace routes one-by-one without disruption

## Definitions

**Sidecar**: A lightweight Go binary that exposes Forst-compiled business logic as HTTP services.

**Route Replacement**: The process of replacing individual TypeScript route handlers with Forst implementations.

**Gradual Adoption**: The ability to migrate performance-critical routes incrementally without rewriting entire applications.

**Zero-Effort Integration**: Drop-in NPM package that requires minimal configuration.

**Dev Server Integration**: Automatic hot-reloading of Forst files during development.

## Guide-level explanation

The sidecar integration enables developers to solve TypeScript backend performance issues by gradually replacing problematic routes with high-performance Forst implementations.

### Basic Usage

```bash
# Install the sidecar package
npm install @forst/sidecar

# That's it! No configuration needed
```

```typescript
// Your existing TypeScript app
import express from "express";
import { processData, searchUsers } from "@forst/sidecar";

const app = express();

// Keep existing routes unchanged
app.get("/health", (req, res) => {
  res.json({ status: "ok" });
});

// Replace performance-critical routes with Forst implementations
app.post("/process-data", async (req, res) => {
  try {
    const result = await processData(req.body);
    res.json(result);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

app.get("/search", async (req, res) => {
  const results = await searchUsers(req.query);
  res.json(results);
});
```

### Forst Implementation

```go
// forst/routes/process_data.ft
package routes

import (
  "encoding/json"
  "net/http"
)

type ProcessDataInput struct {
  Records []Record `json:"records"`
}

type Record struct {
  ID   string `json:"id"`
  Data string `json:"data"`
}

func processData(input ProcessDataInput) {
  // High-performance data processing
  // Memory-efficient operations
  // Concurrent processing with goroutines

  for i := range input.Records {
    // Process each record efficiently
    go processRecord(input.Records[i])
  }

  return map[string]interface{}{
    "processed": len(input.Records),
    "status": "success",
  }
}

// forst/routes/search.ft
func searchUsers(query map[string]interface{}) {
  // Complex search logic with high performance
  // Efficient memory usage
  // Concurrent database queries

  return searchResults
}
```

### Development Workflow

1. **Identify bottlenecks**: Find routes with performance issues
2. **Create Forst file**: Write the route logic in Forst
3. **Import and use**: Import the generated function in TypeScript
4. **Test and deploy**: The sidecar handles compilation and deployment

### Directory Structure

```
my-app/
├── src/                    # TypeScript application
│   ├── routes/
│   ├── middleware/
│   └── app.ts
├── forst/                  # Forst implementations
│   ├── routes/
│   │   ├── process_data.ft
│   │   └── search.ft
│   └── shared/
│       └── types.ft
├── package.json
└── forst.config.js        # Optional configuration
```

## Reference-level explanation

The sidecar integration is implemented as a multi-layered system that provides seamless integration between TypeScript applications and high-performance Forst implementations.

### Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   TypeScript    │    │   Forst Sidecar │    │   Go Binary     │
│   Application   │◄──►│   (HTTP Server) │◄──►│   (Business     │
│                 │    │                 │    │    Logic)       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │                       │                       │
    HTTP/JSON              HTTP/JSON              Compiled
    Protocol              Protocol              Go Code
```

### Core Components

#### 1. Automatic Route Detection

The sidecar automatically detects Forst files and generates corresponding TypeScript functions:

```typescript
// Auto-generated from forst/routes/process_data.ft
export async function processData(
  input: ProcessDataInput
): Promise<ProcessDataResult> {
  const response = await fetch("http://localhost:8080/process-data", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });

  if (!response.ok) {
    throw new Error(`Forst execution failed: ${response.statusText}`);
  }

  return response.json();
}

export interface ProcessDataInput {
  records: Array<{
    id: string;
    data: string;
  }>;
}

export interface ProcessDataResult {
  processed: number;
  status: string;
}
```

#### 2. High-Performance Go Implementation

Forst files are compiled to optimized Go binaries:

```go
// Generated from forst/routes/process_data.ft
package main

import (
    "encoding/json"
    "net/http"
    "sync"
)

type ProcessDataInput struct {
    Records []Record `json:"records"`
}

type Record struct {
    ID   string `json:"id"`
    Data string `json:"data"`
}

func processData(input ProcessDataInput) (map[string]interface{}, error) {
    // High-performance concurrent processing
    var wg sync.WaitGroup
    results := make(chan ProcessedRecord, len(input.Records))

    for _, record := range input.Records {
        wg.Add(1)
        go func(r Record) {
            defer wg.Done()
            // Process record with Go's efficient memory management
            processed := processRecord(r)
            results <- processed
        }(record)
    }

    wg.Wait()
    close(results)

    // Collect results efficiently
    processedCount := 0
    for range results {
        processedCount++
    }

    return map[string]interface{}{
        "processed": processedCount,
        "status":    "success",
    }, nil
}

func main() {
    http.HandleFunc("/process-data", func(w http.ResponseWriter, r *http.Request) {
        var input ProcessDataInput
        if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }

        result, err := processData(input)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        json.NewEncoder(w).Encode(result)
    })

    http.ListenAndServe(":8080", nil)
}
```

#### 3. Development Server Integration

The sidecar provides seamless development experience:

```typescript
// @forst/sidecar/lib/dev-server.ts
export class ForstDevServer {
  private watcher: chokidar.FSWatcher;
  private compiler: ForstCompiler;

  constructor() {
    this.watcher = chokidar.watch("./forst/**/*.ft");
    this.compiler = new ForstCompiler();

    this.watcher.on("change", (path) => {
      this.recompile(path);
    });
  }

  private async recompile(path: string) {
    console.log(`Recompiling ${path}...`);
    await this.compiler.compile(path);
    console.log("Recompilation complete");
  }
}
```

### Performance Benefits

#### 1. Memory Efficiency

Go's garbage collector handles memory spikes better than Node.js:

```go
// Efficient memory usage for large datasets
func processLargeDataset(data []Record) []ProcessedRecord {
    // Go's memory management prevents spikes
    results := make([]ProcessedRecord, 0, len(data))

    for _, record := range data {
        processed := processRecord(record)
        results = append(results, processed)
    }

    return results
}
```

#### 2. Concurrent Performance

Go's goroutines handle concurrent requests efficiently:

```go
// Concurrent request handling
func handleConcurrentRequests(requests []Request) []Response {
    var wg sync.WaitGroup
    responses := make(chan Response, len(requests))

    for _, req := range requests {
        wg.Add(1)
        go func(r Request) {
            defer wg.Done()
            response := processRequest(r)
            responses <- response
        }(req)
    }

    wg.Wait()
    close(responses)

    var results []Response
    for resp := range responses {
        results = append(results, resp)
    }

    return results
}
```

#### 3. CPU Optimization

Compiled Go code outperforms interpreted TypeScript:

```go
// CPU-intensive operations run efficiently
func performComplexCalculation(data []float64) float64 {
    // Compiled code runs much faster than interpreted TypeScript
    result := 0.0
    for _, value := range data {
        result += math.Pow(value, 2)
    }
    return math.Sqrt(result)
}
```

## Implementation Plan

### Phase 1: Core Sidecar Infrastructure

1. **Automatic Detection**

   - Scan `forst/` directory for `.ft` files
   - Generate TypeScript interfaces automatically
   - Create HTTP endpoints for each Forst function

2. **Development Server**

   - File watching and hot reloading
   - Automatic recompilation
   - Development HTTP server

3. **Zero-Config Setup**
   - NPM package with minimal setup
   - Automatic binary compilation
   - Default configuration

### Phase 2: Production Integration

1. **Deployment Support**

   - Docker integration
   - Kubernetes sidecar support
   - Health checks and monitoring

2. **Performance Monitoring**

   - Memory usage tracking
   - Response time metrics
   - Error rate monitoring

3. **TypeScript Type Generation**
   - Automatic interface generation
   - Frontend type sharing
   - API documentation

### Phase 3: Advanced Features

1. **Shared State**

   - Database connections
   - Caching layers
   - Session management

2. **Middleware Support**

   - Authentication
   - Rate limiting
   - Logging

3. **Testing Integration**
   - Unit test generation
   - Integration test support
   - Performance benchmarks

## Configuration

### Minimal Configuration (Zero Config)

```bash
# Install the package
npm install @forst/sidecar

# Start development server
npm run dev
```

The sidecar automatically:

- Detects `forst/` directory
- Compiles `.ft` files to Go binaries
- Generates TypeScript interfaces
- Starts development server

### Optional Configuration

```javascript
// forst.config.js (optional)
module.exports = {
  // Custom directory structure
  forstDir: "./forst",
  outputDir: "./dist/forst",

  // Development server
  devServer: {
    port: 8080,
    watch: true,
  },

  // Compilation options
  compilation: {
    optimization: "speed",
    debug: process.env.NODE_ENV === "development",
  },
};
```

## Examples

### Memory-Intensive Data Processing

```typescript
// Original TypeScript (memory issues)
app.post("/process-large-dataset", async (req, res) => {
  const data = req.body;

  // This causes memory spikes
  const processed = data.map((record) => {
    return heavyProcessing(record);
  });

  res.json(processed);
});

// Replaced with Forst implementation
import { processLargeDataset } from "@forst/sidecar";

app.post("/process-large-dataset", async (req, res) => {
  const result = await processLargeDataset(req.body);
  res.json(result);
});
```

```go
// forst/routes/process_large_dataset.ft
func processLargeDataset(data []Record) []ProcessedRecord {
  // Efficient memory management
  results := make([]ProcessedRecord, 0, len(data))

  // Concurrent processing
  var wg sync.WaitGroup
  processed := make(chan ProcessedRecord, len(data))

  for _, record := range data {
    wg.Add(1)
    go func(r Record) {
      defer wg.Done()
      result := heavyProcessing(r)
      processed <- result
    }(record)
  }

  wg.Wait()
  close(processed)

  for result := range processed {
    results = append(results, result)
  }

  return results
}
```

### Concurrent Request Handling

```typescript
// Original TypeScript (bottleneck)
app.get("/search", async (req, res) => {
  const queries = req.query.queries.split(",");

  // Single-threaded processing
  const results = [];
  for (const query of queries) {
    const result = await performSearch(query);
    results.push(result);
  }

  res.json(results);
});

// Replaced with Forst implementation
import { performConcurrentSearch } from "@forst/sidecar";

app.get("/search", async (req, res) => {
  const queries = req.query.queries.split(",");
  const results = await performConcurrentSearch(queries);
  res.json(results);
});
```

```go
// forst/routes/search.ft
func performConcurrentSearch(queries []string) []SearchResult {
  var wg sync.WaitGroup
  results := make(chan SearchResult, len(queries))g

  for _, query := range queries {
    wg.Add(1)
    go func(q string) {
      defer wg.Done()
      result := performSearch(q)
      results <- result
    }(query)
  }

  wg.Wait()
  close(results)

  var searchResults []SearchResult
  for result := range results {
    searchResults = append(searchResults, result)
  }

  return searchResults
}
```

### CPU-Intensive Calculations

```typescript
// Original TypeScript (slow)
app.post("/calculate", async (req, res) => {
  const { numbers } = req.body;

  // CPU-intensive calculation
  const result = numbers.reduce((acc, num) => {
    return acc + Math.pow(num, 2);
  }, 0);

  res.json({ result: Math.sqrt(result) });
});

// Replaced with Forst implementation
import { calculate } from "@forst/sidecar";

app.post("/calculate", async (req, res) => {
  const result = await calculate(req.body);
  res.json(result);
});
```

```go
// forst/routes/calculate.ft
import "math"

func calculate(input map[string]interface{}) map[string]interface{} {
  numbers := input["numbers"].([]float64)

  // Compiled Go code runs much faster
  result := 0.0
  for _, num := range numbers {
    result += math.Pow(num, 2)
  }

  return map[string]interface{}{
    "result": math.Sqrt(result),
  }
}
```

## Drawbacks

1. **Additional complexity**: Introduces HTTP layer between TypeScript and business logic
2. **Deployment overhead**: Requires Go binaries in production
3. **Network latency**: HTTP calls add latency compared to direct function calls
4. **Debugging complexity**: Distributed debugging across TypeScript and Go
5. **Learning curve**: Teams need to learn Forst syntax for new implementations

## Rationale and alternatives

### Alternative Approaches

1. **Complete rewrite**: Rewrite entire applications in Go

   - Drawback: High risk and time investment
   - Drawback: Disrupts development workflow

2. **Microservices**: Split into separate Go services

   - Drawback: Complex deployment and orchestration
   - Drawback: Network overhead between services

3. **WebAssembly**: Compile Go to WASM
   - Drawback: Limited Go standard library support
   - Drawback: Larger binary sizes

### Why Sidecar Integration

The sidecar approach provides the best balance of:

1. **Gradual adoption**: Replace routes incrementally
2. **Performance benefits**: Leverages Go's efficiency
3. **Development experience**: Maintains TypeScript workflow
4. **Risk mitigation**: Low-risk migration path
5. **Immediate value**: Solve performance issues quickly

### Prior Art

This approach is inspired by:

1. **Kubernetes sidecars**: Independent containers that enhance main applications
2. **Service mesh patterns**: Lightweight proxies for service communication
3. **Gradual migration strategies**: Incremental system improvements
4. **Performance optimization**: Targeted improvements for bottlenecks

## Conclusion

The sidecar integration approach provides a practical solution for solving TypeScript backend performance issues through gradual adoption of Forst implementations. By enabling teams to replace problematic routes incrementally, this approach allows immediate performance improvements without disrupting existing development workflows.

The zero-config setup and automatic type generation make it easy for teams to start solving performance bottlenecks immediately, while the HTTP-based architecture ensures compatibility with existing deployment and monitoring infrastructure.

This approach aligns with Forst's goal of bringing Go's performance to TypeScript applications while providing a practical migration path that teams can adopt without risk.
