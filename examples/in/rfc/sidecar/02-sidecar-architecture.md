# Architecture and Communication Patterns

## Architecture Overview

The sidecar integration supports multiple communication patterns optimized for different use cases:

### Pattern 1: Local IPC (Recommended for Development)

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   TypeScript    │    │   Forst Sidecar │    │   Go Binary     │
│   Application   │◄──►│   (IPC Server)  │◄──►│   (Business     │
│                 │    │                 │    │    Logic)       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │                       │                       │
    Unix Domain              Unix Domain              Compiled
    Sockets/                 Sockets/                 Go Code
    Named Pipes              Named Pipes
```

### Pattern 2: HTTP (Recommended for Production)

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

### Pattern 3: Direct Integration (Experimental)

```
┌─────────────────┐    ┌─────────────────┐
│   TypeScript    │    │   Go Binary     │
│   Application   │◄──►│   (Shared       │
│                 │    │    Memory)      │
└─────────────────┘    └─────────────────┘
         │                       │
         │                       │
    FFI/Shared              Compiled
    Memory                  Go Code
```

## Communication Pattern Analysis

### HTTP vs IPC Trade-offs

**HTTP Advantages:**

- **Familiar protocol**: Standard REST/JSON patterns
- **Production ready**: Works with existing load balancers, proxies, monitoring
- **Cross-platform**: Works on any OS without special permissions
- **Debugging**: Easy to inspect with curl, Postman, browser dev tools
- **Observability**: Standard HTTP metrics, logs, tracing
- **Scaling**: Can be deployed as separate services

**HTTP Disadvantages:**

- **Overhead**: TCP handshakes, HTTP headers, JSON serialization
- **Latency**: ~1-5ms per call (significant for high-frequency operations)
- **Memory**: JSON parsing/stringification overhead
- **Complexity**: Network stack, connection pooling, retry logic

**IPC Advantages:**

- **Performance**: ~0.1-0.5ms latency (10x faster than HTTP)
- **Efficiency**: Direct memory sharing, minimal serialization
- **Simplicity**: No network stack, connection management
- **Security**: Process isolation without network exposure
- **Development**: Faster iteration, no port conflicts

**IPC Disadvantages:**

- **Platform specific**: Unix domain sockets vs named pipes
- **Debugging**: Harder to inspect, requires special tools
- **Deployment**: More complex in containerized environments
- **Scaling**: Limited to single machine

### Recommended Approach: Hybrid Architecture

```typescript
// @forst/sidecar/lib/client.ts
export class ForstClient {
  private mode: "ipc" | "http" | "direct";
  private ipcClient: IPCClient;
  private httpClient: HTTPClient;
  private directClient: DirectClient;

  constructor(config: ForstConfig) {
    this.mode = this.detectOptimalMode(config);
    this.initializeClient();
  }

  private detectOptimalMode(config: ForstConfig): "ipc" | "http" | "direct" {
    if (config.mode === "development") return "ipc";
    if (config.mode === "production") return "http";
    if (config.experimental?.direct) return "direct";

    // Auto-detect based on environment
    if (process.env.NODE_ENV === "development") return "ipc";
    return "http";
  }

  async callFunction<T>(name: string, args: any[]): Promise<T> {
    switch (this.mode) {
      case "ipc":
        return this.ipcClient.call(name, args);
      case "http":
        return this.httpClient.call(name, args);
      case "direct":
        return this.directClient.call(name, args);
    }
  }
}
```

## Core Components

### 1. Automatic Route Detection

The sidecar automatically detects Forst files and generates corresponding TypeScript functions:

```typescript
// Auto-generated from forst/routes/process_data.ft
export async function processData(
  input: ProcessDataInput
): Promise<ProcessDataResult> {
  // Uses optimal transport based on environment
  return forstClient.call("processData", [input]);
}

// Transport-specific implementations:
// HTTP: fetch("http://localhost:8080/process-data", {...})
// IPC: ipcClient.send("processData", input)
// Direct: directClient.invoke("processData", input)
```

```typescript
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

export interface DatasetRecord {
  id: string;
  data: string;
  metadata: {
    category: string;
    priority: number;
  };
}

export interface ProcessedRecord {
  id: string;
  result: string;
  processingTime: number;
}

export interface SearchQuery {
  query: string;
  filters: {
    category?: string;
    price?: number;
  };
  limit: number;
}

export interface SearchResult {
  query: string;
  results: Array<{
    id: string;
    title: string;
    score: number;
  }>;
  totalCount: number;
}

export interface CalculationInput {
  numbers: number[];
  operation: "sum" | "average" | "variance";
}

export interface CalculationResult {
  result: number;
  computationTime: number;
}
```

### 2. High-Performance Go Implementation

Forst files are compiled to optimized Go binaries:

```go
// Generated from forst/routes/process_data.ft
package main

import (
    "encoding/json"
    "net/http"
    "sync"
    "fmt"
)

// Generated from Forst shapes - no JSON annotations needed
type Record struct {
    ID   string
    Data string
}

type ProcessDataInput struct {
    Records []Record
}

type ProcessedRecord struct {
    ID        string
    Processed bool
    Result    string
}

func processData(input ProcessDataInput) (map[string]interface{}, error) {
    // Validate input using Forst-generated validation
    if err := validateProcessDataInput(input); err != nil {
        return nil, err
    }

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

// Forst-generated validation functions from shape guards
func validateRecord(record Record) error {
    if len(record.ID) < 1 {
        return fmt.Errorf("Record ID cannot be empty")
    }
    if len(record.Data) < 1 {
        return fmt.Errorf("Record data cannot be empty")
    }
    return nil
}

func validateProcessDataInput(input ProcessDataInput) error {
    if len(input.Records) < 1 {
        return fmt.Errorf("At least one record is required")
    }

    for _, record := range input.Records {
        if err := validateRecord(record); err != nil {
            return fmt.Errorf("Invalid record in input: %v", err)
        }
    }
    return nil
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

### 3. Multi-Transport Server Implementation

The sidecar supports multiple transport protocols:

```typescript
// @forst/sidecar/lib/server.ts
export class ForstServer {
  private httpServer: HTTPServer;
  private ipcServer: IPCServer;
  private compiler: ForstCompiler;

  constructor(config: ForstConfig) {
    this.compiler = new ForstCompiler();
    this.initializeServers(config);
  }

  private initializeServers(config: ForstConfig) {
    if (config.transports.includes("http")) {
      this.httpServer = new HTTPServer({
        port: config.http?.port || 8080,
        cors: config.http?.cors || true,
      });
    }

    if (config.transports.includes("ipc")) {
      this.ipcServer = new IPCServer({
        socketPath: config.ipc?.socketPath || "/tmp/forst.sock",
        permissions: config.ipc?.permissions || 0o600,
      });
    }
  }

  async start() {
    await this.compiler.compileAll();

    if (this.httpServer) {
      await this.httpServer.start();
      console.log(`HTTP server running on port ${this.httpServer.port}`);
    }

    if (this.ipcServer) {
      await this.ipcServer.start();
      console.log(`IPC server running on ${this.ipcServer.socketPath}`);
    }
  }
}

// IPC Server Implementation
class IPCServer {
  private server: net.Server;
  private socketPath: string;

  constructor(config: { socketPath: string; permissions: number }) {
    this.socketPath = config.socketPath;
    this.server = net.createServer(this.handleConnection.bind(this));
  }

  private handleConnection(socket: net.Socket) {
    socket.on("data", async (data) => {
      const request = JSON.parse(data.toString());
      const result = await this.executeFunction(request.function, request.args);
      socket.write(JSON.stringify({ success: true, result }));
    });
  }

  async start() {
    return new Promise((resolve) => {
      this.server.listen(this.socketPath, () => {
        // Set socket permissions for security
        fs.chmodSync(this.socketPath, 0o600);
        resolve(undefined);
      });
    });
  }
}
```

### 4. Development Server Integration

The sidecar provides seamless development experience with hot reloading:

```typescript
// @forst/sidecar/lib/dev-server.ts
export class ForstDevServer {
  private watcher: chokidar.FSWatcher;
  private compiler: ForstCompiler;
  private server: ForstServer;

  constructor(config: ForstConfig) {
    this.watcher = chokidar.watch("./forst/**/*.ft");
    this.compiler = new ForstCompiler();
    this.server = new ForstServer(config);

    this.watcher.on("change", (path) => {
      this.recompile(path);
    });
  }

  private async recompile(path: string) {
    console.log(`Recompiling ${path}...`);
    await this.compiler.compile(path);
    await this.server.restart();
    console.log("Recompilation complete");
  }

  async start() {
    await this.server.start();
    console.log("Development server started with hot reloading");
  }
}
```

## Performance Benefits

### 1. Memory Efficiency

Go's garbage collector handles memory spikes better than Node.js:

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

  // Efficient memory usage for large datasets
  results := make(Array(ProcessedRecord), 0, len(data))

  for record in data {
    processed := processRecord(record)
    results = append(results, processed)
  }

  return results
}
```

### 2. Concurrent Performance

Go's goroutines handle concurrent requests efficiently:

```go
// forst/routes/concurrent_requests.ft
type Request = {
  id: String,
  data: String,
  priority: Int
}

type Response = {
  id: String,
  result: String,
  status: String
}

is (request Request) Valid() {
  ensure request.id is Min(1) or InvalidRequest("Request ID cannot be empty")
  ensure request.data is Min(1) or InvalidRequest("Request data cannot be empty")
  ensure request.priority is Between(1, 5) or InvalidRequest("Priority must be between 1 and 5")
}

func handleConcurrentRequests(requests Array(Request)) Array(Response) {
  ensure requests is Min(1) or InvalidInput("At least one request is required")

  // Validate all requests
  for request in requests {
    ensure request is Valid() or InvalidInput("Invalid request in batch")
  }

  // Concurrent request handling
  var wg sync.WaitGroup
  responses := make(chan Response, len(requests))

  for request in requests {
    wg.Add(1)
    go func(r Request) {
      defer wg.Done()
      response := processRequest(r)
      responses <- response
    }(request)
  }

  wg.Wait()
  close(responses)

  var results Array(Response)
  for resp in responses {
    results = append(results, resp)
  }

  return results
}
```

### 3. CPU Optimization

Compiled Go code outperforms interpreted TypeScript:

```go
// forst/routes/complex_calculation.ft
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

func performComplexCalculation(input CalculationInput) CalculationResult {
  ensure input is Valid() or InvalidInput("Invalid calculation input")

  // CPU-intensive operations run efficiently
  // Compiled code runs much faster than interpreted TypeScript
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
