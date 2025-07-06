---
Feature Name: sidecar
Start Date: 2025-07-06
---

# Sidecar Integration RFC

This RFC proposes a sidecar integration approach that enables gradual adoption of Forst for high-performance backend operations within TypeScript applications.

## Introduction

The sidecar integration addresses fundamental performance and memory issues that plague TypeScript backends by allowing developers to replace individual routes with Forst implementations without disrupting existing workflows.

## RFC Structure

### [01-overview.md](01-overview.md) - Overview and Motivation

- **Summary**: High-level introduction to the sidecar approach
- **Motivation**: Why TypeScript backends need performance improvements
- **Definitions**: Key terms and concepts
- **Guide-level explanation**: Basic usage and workflow
- **Forst Implementation**: Examples using proper Forst shapes and guards

### [02-architecture.md](02-architecture.md) - Architecture and Communication Patterns

- **Architecture Overview**: Multi-transport patterns (IPC, HTTP, Direct)
- **Communication Analysis**: HTTP vs IPC trade-offs
- **Core Components**: Automatic route detection, Go implementation, server architecture
- **Performance Benefits**: Memory efficiency, concurrent performance, CPU optimization

### [03-configuration.md](03-configuration.md) - Configuration and Setup

- **Transport Selection**: Environment-specific configuration
- **Implementation Plan**: Phased development approach
- **Examples**: Real-world use cases with TypeScript and Forst implementations
- **Performance Comparison**: Metrics for different transport options

### [04-trade-offs.md](04-trade-offs.md) - Trade-offs and Recommendations

- **Drawbacks**: Limitations of each approach
- **Recommendations**: When to use different transport patterns
- **Migration Strategy**: Step-by-step adoption guide
- **Future Considerations**: Advanced features and optimizations

## Key Features

### 🚀 **Performance Benefits**

- **10x faster latency** with IPC transport (~0.2ms vs ~2ms)
- **Memory efficiency** through Go's garbage collector
- **Concurrent processing** with goroutines
- **CPU optimization** with compiled Go code

### 🔧 **Developer Experience**

- **Zero-config setup** with automatic detection
- **Hot reloading** during development
- **Type safety** with automatic TypeScript interface generation
- **Gradual adoption** - replace routes one-by-one

### 🏗️ **Architecture Flexibility**

- **Hybrid transport**: IPC for development, HTTP for production
- **Multi-pattern support**: HTTP, IPC, and experimental direct integration
- **Environment-aware**: Automatic transport selection
- **Production-ready**: Standard observability and monitoring

## Quick Start

```bash
# Install the sidecar package
npm install @forst/sidecar

# Start development server (uses IPC automatically)
npm run dev
```

```typescript
// Your existing TypeScript app
import { processData } from "@forst/sidecar";

app.post("/process-data", async (req, res) => {
  // This now runs in a high-performance Go binary
  const result = await processData(req.body);
  res.json(result);
});
```

```go
// forst/routes/process_data.ft
type ProcessDataInput = {
  records: Array({ id: String, data: String })
}

is (input ProcessDataInput) Valid() {
  ensure input.records is Min(1) or InvalidInput("At least one record is required")
}

func processData(input ProcessDataInput) {
  ensure input is Valid() or InvalidInput("Invalid input data")

  // High-performance processing with goroutines
  for record in input.records {
    go processRecord(record)
  }

  return { processed: len(input.records), status: "success" }
}
```

## Transport Comparison

| Transport | Latency | Throughput | Debugging | Production Ready |
| --------- | ------- | ---------- | --------- | ---------------- |
| IPC       | ~0.2ms  | 100k req/s | Hard      | No               |
| HTTP      | ~2ms    | 10k req/s  | Easy      | Yes              |
| Direct    | ~0.1ms  | 500k req/s | Hard      | No               |

## Migration Strategy

1. **Start with HTTP**: Easy debugging and familiar patterns
2. **Profile performance**: Identify bottlenecks and high-frequency calls
3. **Migrate to IPC**: Move performance-critical functions to IPC
4. **Monitor and optimize**: Use metrics to guide further optimization

## Contributing

This RFC is structured to be easily digestible and actionable. Each section focuses on specific aspects of the sidecar integration:

- **01-overview.md**: Start here for the big picture
- **02-architecture.md**: Deep dive into technical implementation
- **03-configuration.md**: Practical setup and examples
- **04-trade-offs.md**: Decision-making framework and future considerations

The modular structure allows teams to focus on the aspects most relevant to their needs while providing comprehensive coverage of the entire system.
