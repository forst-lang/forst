# Sidecar Integration for TypeScript Backend Performance

### [00-sidecar.md](00-sidecar.md) - Back to Sidecar RFC

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

type Record = {
  id: String,
  data: String
}

type ProcessDataInput = {
  records: Array(Record)
}

is (record Record) Valid() {
  ensure record.id is Min(1) or InvalidRecord("Record ID cannot be empty")
  ensure record.data is Min(1) or InvalidRecord("Record data cannot be empty")
}

is (input ProcessDataInput) Valid() {
  ensure input.records is Min(1) or InvalidInput("At least one record is required")

  for record in input.records {
    ensure record is Valid() or InvalidInput("Invalid record in input")
  }
}

func processData(input ProcessDataInput) {
  ensure input is Valid() or InvalidInput("Invalid input data")

  // High-performance data processing
  // Memory-efficient operations
  // Concurrent processing with goroutines

  for record in input.records {
    // Process each record efficiently
    go processRecord(record)
  }

  return {
    processed: len(input.records),
    status: "success"
  }
}

// forst/routes/search.ft
type SearchQuery = {
  query: String,
  filters: { category: *String, price: *Int },
  limit: Int
}

is (query SearchQuery) Valid() {
  ensure query.query is Min(1) or InvalidQuery("Search query cannot be empty")
  ensure query.limit is Between(1, 100) or InvalidQuery("Limit must be between 1 and 100")
}

func searchUsers(query SearchQuery) {
  ensure query is Valid() or InvalidQuery("Invalid search query")

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
