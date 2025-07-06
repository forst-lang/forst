# Streaming Support

### [00-sidecar.md](00-sidecar.md) - Back to Sidecar RFC

The sidecar architecture supports streaming between Node.js HTTP requests and the Forst server, enabling real-time data processing and large dataset handling.

## Streaming Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   TypeScript    │    │   Forst Sidecar │    │   Go Binary     │
│   Application   │◄──►│   (HTTP Server) │◄──►│   (Business     │
│   (Streaming)   │    │   (Streaming)   │    │    Logic)       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │                       │                       │
    HTTP/Streaming          HTTP/Streaming          Compiled
    Protocol              Protocol              Go Code
```

## 1. HTTP Streaming Implementation

```typescript
// @forst/sidecar/lib/streaming-client.ts
export class ForstStreamingClient {
  private baseUrl: string;

  constructor(config: { baseUrl: string }) {
    this.baseUrl = config.baseUrl;
  }

  // Stream large datasets to Forst for processing
  async streamToForst<T>(
    endpoint: string,
    dataStream: ReadableStream<T>,
    options: StreamingOptions = {}
  ): Promise<ReadableStream<any>> {
    const response = await fetch(`${this.baseUrl}/${endpoint}`, {
      method: "POST",
      headers: {
        "Content-Type": "application/octet-stream",
        "Transfer-Encoding": "chunked",
        ...options.headers,
      },
      body: this.createStreamBody(dataStream),
      duplex: "half",
    });

    if (!response.body) {
      throw new Error("No response body available for streaming");
    }

    return response.body;
  }

  // Process streaming data from Forst
  async processStreamingData<T>(
    endpoint: string,
    input: any,
    options: StreamingOptions = {}
  ): Promise<ReadableStream<T>> {
    const response = await fetch(`${this.baseUrl}/${endpoint}`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Accept: "application/octet-stream",
        ...options.headers,
      },
      body: JSON.stringify(input),
      duplex: "half",
    });

    if (!response.body) {
      throw new Error("No response body available for streaming");
    }

    return response.body;
  }

  private createStreamBody<T>(
    stream: ReadableStream<T>
  ): ReadableStream<Uint8Array> {
    return new ReadableStream({
      async start(controller) {
        const reader = stream.getReader();

        try {
          while (true) {
            const { done, value } = await reader.read();
            if (done) break;

            // Convert data to bytes for transmission
            const chunk = JSON.stringify(value) + "\n";
            controller.enqueue(new TextEncoder().encode(chunk));
          }
        } finally {
          controller.close();
        }
      },
    });
  }
}

interface StreamingOptions {
  headers?: Record<string, string>;
  chunkSize?: number;
  timeout?: number;
}
```

## 2. Forst Streaming Server

```go
// Generated from forst/routes/streaming_processor.ft
package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "sync"
)

type StreamingRecord struct {
    ID   string `json:"id"`
    Data string `json:"data"`
}

type ProcessedRecord struct {
    ID     string `json:"id"`
    Result string `json:"result"`
    Status string `json:"status"`
}

// Stream processing endpoint
func handleStreamingProcess(w http.ResponseWriter, r *http.Request) {
    // Set headers for streaming
    w.Header().Set("Content-Type", "application/octet-stream")
    w.Header().Set("Transfer-Encoding", "chunked")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    // Create a channel for processed results
    results := make(chan ProcessedRecord, 100)

    // Start processing in background
    go func() {
        defer close(results)

        reader := bufio.NewReader(r.Body)
        for {
            line, err := reader.ReadString('\n')
            if err == io.EOF {
                break
            }
            if err != nil {
                http.Error(w, err.Error(), http.StatusBadRequest)
                return
            }

            // Parse incoming record
            var record StreamingRecord
            if err := json.Unmarshal([]byte(line), &record); err != nil {
                continue // Skip invalid records
            }

            // Process record and send result immediately
            processed := processRecord(record)
            results <- processed
        }
    }()

    // Stream results back to client
    encoder := json.NewEncoder(w)
    for result := range results {
        if err := encoder.Encode(result); err != nil {
            break
        }
        w.(http.Flusher).Flush() // Ensure data is sent immediately
    }
}

// Large dataset streaming endpoint
func handleLargeDatasetStream(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/octet-stream")
    w.Header().Set("Transfer-Encoding", "chunked")

    // Process large dataset in chunks
    var wg sync.WaitGroup
    results := make(chan ProcessedRecord, 1000)

    // Start multiple workers
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()

            reader := bufio.NewReader(r.Body)
            for {
                line, err := reader.ReadString('\n')
                if err == io.EOF {
                    break
                }
                if err != nil {
                    continue
                }

                var record StreamingRecord
                if err := json.Unmarshal([]byte(line), &record); err != nil {
                    continue
                }

                processed := processRecord(record)
                results <- processed
            }
        }()
    }

    // Stream results as they're processed
    go func() {
        wg.Wait()
        close(results)
    }()

    encoder := json.NewEncoder(w)
    for result := range results {
        if err := encoder.Encode(result); err != nil {
            break
        }
        w.(http.Flusher).Flush()
    }
}

func processRecord(record StreamingRecord) ProcessedRecord {
    // High-performance processing logic
    return ProcessedRecord{
        ID:     record.ID,
        Result: fmt.Sprintf("processed_%s", record.Data),
        Status: "success",
    }
}
```

## 3. TypeScript Streaming Integration

```typescript
// @forst/sidecar/lib/streaming-integration.ts
export class StreamingIntegration {
  private streamingClient: ForstStreamingClient;

  constructor(config: { baseUrl: string }) {
    this.streamingClient = new ForstStreamingClient(config);
  }

  // Stream large dataset processing
  async processLargeDataset(
    records: AsyncIterable<DatasetRecord>
  ): Promise<AsyncIterable<ProcessedRecord>> {
    const inputStream = this.createReadableStream(records);
    const outputStream = await this.streamingClient.streamToForst(
      "streaming-process",
      inputStream
    );

    return this.createAsyncIterable(outputStream);
  }

  // Real-time data processing
  async processRealTimeData(
    input: any
  ): Promise<AsyncIterable<ProcessedRecord>> {
    const outputStream = await this.streamingClient.processStreamingData(
      "real-time-process",
      input
    );

    return this.createAsyncIterable(outputStream);
  }

  // Express.js streaming integration
  async handleStreamingRequest(req: Request, res: Response): Promise<void> {
    // Set streaming headers
    res.setHeader("Content-Type", "application/octet-stream");
    res.setHeader("Transfer-Encoding", "chunked");
    res.setHeader("Cache-Control", "no-cache");
    res.setHeader("Connection", "keep-alive");

    try {
      // Create streaming request to Forst
      const forstResponse = await fetch(
        "http://localhost:8080/streaming-process",
        {
          method: "POST",
          headers: {
            "Content-Type": "application/octet-stream",
            "Transfer-Encoding": "chunked",
          },
          body: req.body,
          duplex: "half",
        }
      );

      if (!forstResponse.body) {
        throw new Error("No response body from Forst");
      }

      // Pipe Forst response to client
      const reader = forstResponse.body.getReader();
      const encoder = new TextEncoder();

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        // Send chunk to client
        res.write(value);
        res.flush?.(); // Flush if available
      }

      res.end();
    } catch (error) {
      console.error("Streaming error:", error);
      res.status(500).json({ error: "Streaming failed" });
    }
  }

  private createReadableStream<T>(
    iterable: AsyncIterable<T>
  ): ReadableStream<T> {
    return new ReadableStream({
      async start(controller) {
        try {
          for await (const item of iterable) {
            controller.enqueue(item);
          }
        } finally {
          controller.close();
        }
      },
    });
  }

  private createAsyncIterable<T>(
    stream: ReadableStream<Uint8Array>
  ): AsyncIterable<T> {
    return {
      async *[Symbol.asyncIterator]() {
        const reader = stream.getReader();
        const decoder = new TextDecoder();

        try {
          while (true) {
            const { done, value } = await reader.read();
            if (done) break;

            const chunk = decoder.decode(value, { stream: true });
            const lines = chunk.split("\n").filter((line) => line.trim());

            for (const line of lines) {
              try {
                const item = JSON.parse(line);
                yield item;
              } catch (error) {
                console.warn("Failed to parse streaming chunk:", error);
              }
            }
          }
        } finally {
          reader.releaseLock();
        }
      },
    };
  }
}
```

## 4. Streaming Use Cases

### Large Dataset Processing

```typescript
// Process millions of records without loading into memory
async function processLargeDataset() {
  const streamingIntegration = new StreamingIntegration({
    baseUrl: "http://localhost:8080",
  });

  // Create async iterable for large dataset
  const records = createLargeDatasetIterator();

  // Stream processing with immediate results
  for await (const processed of streamingIntegration.processLargeDataset(
    records
  )) {
    console.log("Processed:", processed);
    // Handle each result as it arrives
  }
}
```

### Real-time Data Processing

```typescript
// Real-time data processing with streaming
async function handleRealTimeData() {
  const streamingIntegration = new StreamingIntegration({
    baseUrl: "http://localhost:8080",
  });

  // Process real-time data stream
  const results = streamingIntegration.processRealTimeData({
    source: "sensor-data",
    filters: { temperature: "high" },
  });

  for await (const result of results) {
    // Handle real-time processed data
    updateDashboard(result);
  }
}
```

### Express.js Streaming Endpoint

```typescript
// Express.js endpoint with streaming
app.post("/streaming-process", async (req, res) => {
  const streamingIntegration = new StreamingIntegration({
    baseUrl: "http://localhost:8080",
  });

  await streamingIntegration.handleStreamingRequest(req, res);
});
```

## 5. Streaming Performance Benefits

- **Memory Efficiency**: Process large datasets without loading into memory
- **Real-time Processing**: Immediate results as data is processed
- **Backpressure Handling**: Automatic flow control between Node.js and Forst
- **Scalability**: Handle millions of records efficiently
- **Low Latency**: Minimal buffering for real-time applications

## 6. Streaming Configuration

```typescript
// @forst/sidecar/lib/streaming-config.ts
export interface StreamingConfig {
  // Chunk size for streaming
  chunkSize: number;

  // Timeout for streaming operations
  timeout: number;

  // Buffer size for processing
  bufferSize: number;

  // Number of concurrent workers
  workers: number;

  // Enable compression
  compression: boolean;

  // Retry configuration
  retry: {
    attempts: number;
    delay: number;
  };
}

export const defaultStreamingConfig: StreamingConfig = {
  chunkSize: 1024 * 1024, // 1MB chunks
  timeout: 30000, // 30 seconds
  bufferSize: 1000, // 1000 items
  workers: 10, // 10 concurrent workers
  compression: true,
  retry: {
    attempts: 3,
    delay: 1000,
  },
};
```

## 7. Error Handling and Resilience

```typescript
// @forst/sidecar/lib/streaming-error-handler.ts
export class StreamingErrorHandler {
  private config: StreamingConfig;

  constructor(config: StreamingConfig) {
    this.config = config;
  }

  // Handle streaming errors with retry logic
  async handleStreamingError<T>(
    operation: () => Promise<T>,
    context: string
  ): Promise<T> {
    let lastError: Error;

    for (let attempt = 0; attempt < this.config.retry.attempts; attempt++) {
      try {
        return await operation();
      } catch (error) {
        lastError = error;

        if (this.isRetryableError(error)) {
          await this.delay(this.config.retry.delay * Math.pow(2, attempt));
          continue;
        }

        throw error;
      }
    }

    throw lastError!;
  }

  private isRetryableError(error: Error): boolean {
    const retryableErrors = [
      "ECONNRESET",
      "ETIMEDOUT",
      "ENOTFOUND",
      "ECONNREFUSED",
    ];

    return retryableErrors.some((code) => error.message.includes(code));
  }

  private delay(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }
}
```

The streaming architecture enables efficient processing of large datasets and real-time data flows while maintaining the performance benefits of the Forst sidecar approach.
