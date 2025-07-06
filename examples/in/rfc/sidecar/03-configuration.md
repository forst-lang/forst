# Configuration and Setup

## Transport Selection

The sidecar automatically selects the optimal transport based on your environment:

```javascript
// forst.config.js
module.exports = {
  // Transport configuration
  transports: {
    // Development: Use IPC for maximum performance
    development: {
      mode: "ipc",
      ipc: {
        socketPath: "/tmp/forst-dev.sock",
        permissions: 0o600,
      },
    },

    // Production: Use HTTP for observability and scaling
    production: {
      mode: "http",
      http: {
        port: 8080,
        cors: true,
        healthCheck: "/health",
      },
    },

    // Testing: Use HTTP for easy debugging
    testing: {
      mode: "http",
      http: {
        port: 0, // Random port
        cors: false,
      },
    },
  },

  // Compilation options
  compilation: {
    optimization: process.env.NODE_ENV === "production" ? "speed" : "debug",
    debug: process.env.NODE_ENV === "development",
  },

  // Directory structure
  forstDir: "./forst",
  outputDir: "./dist/forst",
};
```

## Minimal Configuration (Zero Config)

```bash
# Install the package
npm install @forst/sidecar

# Start development server (uses IPC automatically)
npm run dev
```

The sidecar automatically:

- Detects `forst/` directory
- Compiles `.ft` files to Go binaries
- Generates TypeScript interfaces
- Starts development server with optimal transport

## Environment-Specific Configuration

```typescript
// package.json scripts
{
  "scripts": {
    "dev": "forst-sidecar --mode=development",     // Uses IPC
    "start": "forst-sidecar --mode=production",    // Uses HTTP
    "test": "forst-sidecar --mode=testing"         // Uses HTTP
  }
}
```

## Performance Comparison

| Transport | Latency | Throughput | Debugging | Production Ready |
| --------- | ------- | ---------- | --------- | ---------------- |
| IPC       | ~0.2ms  | 100k req/s | Hard      | No               |
| HTTP      | ~2ms    | 10k req/s  | Easy      | Yes              |
| Direct    | ~0.1ms  | 500k req/s | Hard      | No               |

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
type DatasetRecord = {
  id: String,
  data: String,
  metadata: { category: String, priority: Int }
}

type ProcessedRecord = {
  id: String,
  result: String,
  processingTime: Int
}

is (record DatasetRecord) Valid() {
  ensure record.id is Min(1) or InvalidRecord("Record ID cannot be empty")
  ensure record.data is Min(1) or InvalidRecord("Record data cannot be empty")
  ensure record.metadata.priority is Between(1, 10) or InvalidRecord("Priority must be between 1 and 10")
}

func processLargeDataset(data Array(DatasetRecord)) Array(ProcessedRecord) {
  ensure data is Min(1) or InvalidInput("At least one record is required")

  // Validate all records
  for record in data {
    ensure record is Valid() or InvalidInput("Invalid record in dataset")
  }

  // Efficient memory management
  results := make(Array(ProcessedRecord), 0, len(data))

  // Concurrent processing
  var wg sync.WaitGroup
  processed := make(chan ProcessedRecord, len(data))

  for record in data {
    wg.Add(1)
    go func(r DatasetRecord) {
      defer wg.Done()
      result := heavyProcessing(r)
      processed <- result
    }(record)
  }

  wg.Wait()
  close(processed)

  for result in processed {
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
type SearchQuery = {
  query: String,
  filters: { category: *String, price: *Int },
  limit: Int
}

type SearchResult = {
  query: String,
  results: Array({ id: String, title: String, score: Float }),
  totalCount: Int
}

is (query SearchQuery) Valid() {
  ensure query.query is Min(1) or InvalidQuery("Search query cannot be empty")
  ensure query.limit is Between(1, 100) or InvalidQuery("Limit must be between 1 and 100")
}

func performConcurrentSearch(queries Array(SearchQuery)) Array(SearchResult) {
  ensure queries is Min(1) or InvalidInput("At least one query is required")

  // Validate all queries
  for query in queries {
    ensure query is Valid() or InvalidInput("Invalid search query")
  }

  var wg sync.WaitGroup
  results := make(chan SearchResult, len(queries))

  for query in queries {
    wg.Add(1)
    go func(q SearchQuery) {
      defer wg.Done()
      result := performSearch(q)
      results <- result
    }(query)
  }

  wg.Wait()
  close(results)

  var searchResults Array(SearchResult)
  for result in results {
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
type CalculationInput = {
  numbers: Array(Float),
  operation: String
}

type CalculationResult = {
  result: Float,
  computationTime: Int
}

is (input CalculationInput) Valid() {
  ensure input.numbers is Min(1) or InvalidInput("At least one number is required")
  ensure input.operation is OneOf(["sum", "average", "variance"]) or InvalidInput("Invalid operation")

  for number in input.numbers {
    ensure number is Finite() or InvalidInput("Numbers must be finite")
  }
}

func calculate(input CalculationInput) CalculationResult {
  ensure input is Valid() or InvalidInput("Invalid calculation input")

  // Compiled Go code runs much faster
  result := 0.0

  if input.operation == "sum" {
    for number in input.numbers {
      result += number
    }
  } else if input.operation == "average" {
    sum := 0.0
    for number in input.numbers {
      sum += number
    }
    result = sum / len(input.numbers)
  } else if input.operation == "variance" {
    // Complex statistical calculation
    mean := 0.0
    for number in input.numbers {
      mean += number
    }
    mean = mean / len(input.numbers)

    variance := 0.0
    for number in input.numbers {
      variance += math.Pow(number - mean, 2)
    }
    result = variance / len(input.numbers)
  }

  return {
    result: result,
    computationTime: getComputationTime()
  }
}
```
