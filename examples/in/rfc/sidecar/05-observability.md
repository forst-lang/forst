# Developer Operations and Observability

### [00-sidecar.md](00-sidecar.md) - Back to Sidecar RFC

## Overview

The sidecar integration introduces unique challenges for developer operations, tracing, and observability. This document outlines the specific issues teams will face and proposes solutions for maintaining visibility across the TypeScript-Forst boundary.

## Key Challenges

### 1. Distributed Tracing Complexity

**Problem**: Traditional tracing tools assume single-language applications. The sidecar creates a distributed system where:

- **Trace context propagation** across TypeScript → Go boundary
- **Correlation IDs** must flow through multiple transport layers
- **Span relationships** between TypeScript and Forst functions
- **Performance attribution** becomes unclear (which layer is slow?)

**Example Scenario**:

```typescript
// TypeScript application
app.post("/process-data", async (req, res) => {
  // How do we trace this call into the Forst sidecar?
  const result = await processData(req.body);
  res.json(result);
});
```

```go
// Forst sidecar
func processData(input ProcessDataInput) {
  // How do we correlate this with the TypeScript span?
  // Which spans belong to which request?
  for record in input.records {
    go processRecord(record) // Concurrent processing complicates tracing
  }
}
```

### 2. Multi-Transport Observability

**Problem**: Different transport mechanisms (HTTP, IPC, Direct) require different observability approaches:

- **HTTP**: Standard HTTP metrics, logs, tracing
- **IPC**: Unix domain socket monitoring, process-level metrics
- **Direct**: Shared memory access, FFI call tracking

**Transport-Specific Challenges**:

| Transport | Tracing | Metrics  | Logging  | Debugging |
| --------- | ------- | -------- | -------- | --------- |
| HTTP      | Easy    | Standard | Standard | Easy      |
| IPC       | Hard    | Custom   | Custom   | Hard      |
| Direct    | Hardest | Custom   | Custom   | Hardest   |

### 3. Performance Attribution

**Problem**: When performance issues occur, it's unclear which layer is responsible:

```typescript
// Is the bottleneck in TypeScript, transport, or Forst?
const start = performance.now();
const result = await processData(req.body); // Where is the time spent?
const duration = performance.now() - start;
```

**Potential Bottlenecks**:

- **TypeScript serialization**: JSON.stringify overhead
- **Transport latency**: HTTP vs IPC vs Direct
- **Forst processing**: Go runtime, memory allocation
- **Network I/O**: If using HTTP transport

## Proposed Solutions

### 1. Unified Tracing with OpenTelemetry

```typescript
// @forst/sidecar/lib/tracing.ts
export class ForstTracing {
  private tracer: Tracer;
  private context: Context;

  constructor() {
    this.tracer = trace.getTracer("forst-sidecar");
  }

  async traceFunctionCall<T>(
    functionName: string,
    args: any[],
    fn: () => Promise<T>
  ): Promise<T> {
    const span = this.tracer.startSpan(`forst.${functionName}`, {
      attributes: {
        "forst.function": functionName,
        "forst.transport": this.getTransportType(),
        "forst.args.size": JSON.stringify(args).length,
      },
    });

    try {
      const result = await fn();
      span.setStatus({ code: SpanStatusCode.OK });
      return result;
    } catch (error) {
      span.setStatus({
        code: SpanStatusCode.ERROR,
        message: error.message,
      });
      throw error;
    } finally {
      span.end();
    }
  }

  private getTransportType(): string {
    // Detect current transport (HTTP, IPC, Direct)
    return this.config.transport;
  }
}
```

```go
// Generated tracing code in Forst
package main

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
    "context"
)

func processDataWithTracing(ctx context.Context, input ProcessDataInput) {
    tracer := otel.Tracer("forst-sidecar")

    ctx, span := tracer.Start(ctx, "forst.processData", trace.WithAttributes(
        attribute.String("forst.function", "processData"),
        attribute.Int("forst.input.records", len(input.Records)),
    ))
    defer span.End()

    // Process with child spans for each record
    for i, record := range input.Records {
        go func(idx int, r Record) {
            ctx, childSpan := tracer.Start(ctx, "forst.processRecord", trace.WithAttributes(
                attribute.Int("forst.record.index", idx),
                attribute.String("forst.record.id", r.ID),
            ))
            defer childSpan.End()

            processRecord(r)
        }(i, record)
    }
}
```

### 2. Transport-Aware Metrics

```typescript
// @forst/sidecar/lib/metrics.ts
export class ForstMetrics {
  private metrics: {
    requestDuration: Histogram;
    requestCount: Counter;
    errorCount: Counter;
    transportLatency: Histogram;
  };

  constructor() {
    this.metrics = this.initializeMetrics();
  }

  recordFunctionCall(
    functionName: string,
    duration: number,
    transport: string,
    success: boolean
  ) {
    this.metrics.requestDuration.record(duration, {
      function: functionName,
      transport: transport,
    });

    this.metrics.requestCount.add(1, {
      function: functionName,
      transport: transport,
      success: success.toString(),
    });

    if (!success) {
      this.metrics.errorCount.add(1, {
        function: functionName,
        transport: transport,
      });
    }
  }

  recordTransportLatency(transport: string, latency: number) {
    this.metrics.transportLatency.record(latency, {
      transport: transport,
    });
  }
}
```

### 3. Structured Logging with Correlation

```typescript
// @forst/sidecar/lib/logging.ts
export class ForstLogger {
  private logger: Logger;

  constructor() {
    this.logger = winston.createLogger({
      format: winston.format.combine(
        winston.format.timestamp(),
        winston.format.json()
      ),
      transports: [
        new winston.transports.Console(),
        new winston.transports.File({ filename: "forst-sidecar.log" }),
      ],
    });
  }

  logFunctionCall(
    functionName: string,
    args: any[],
    result: any,
    duration: number,
    traceId: string
  ) {
    this.logger.info("Forst function call", {
      function: functionName,
      args: this.sanitizeArgs(args),
      result: this.sanitizeResult(result),
      duration: duration,
      traceId: traceId,
      transport: this.getTransportType(),
      timestamp: new Date().toISOString(),
    });
  }

  logError(functionName: string, error: Error, traceId: string) {
    this.logger.error("Forst function error", {
      function: functionName,
      error: error.message,
      stack: error.stack,
      traceId: traceId,
      transport: this.getTransportType(),
      timestamp: new Date().toISOString(),
    });
  }
}
```

### 4. Performance Profiling Tools

```typescript
// @forst/sidecar/lib/profiling.ts
export class ForstProfiler {
  private profiles: Map<string, ProfileData> = new Map();

  startProfile(functionName: string): string {
    const profileId = uuid();
    const startTime = performance.now();

    this.profiles.set(profileId, {
      functionName,
      startTime,
      transport: this.getTransportType(),
      traceId: this.getCurrentTraceId(),
    });

    return profileId;
  }

  endProfile(profileId: string, result?: any) {
    const profile = this.profiles.get(profileId);
    if (!profile) return;

    const endTime = performance.now();
    const duration = endTime - profile.startTime;

    // Record detailed timing breakdown
    this.recordTimingBreakdown(profileId, {
      totalDuration: duration,
      transportLatency: this.measureTransportLatency(),
      processingTime: duration - this.measureTransportLatency(),
      resultSize: JSON.stringify(result).length,
    });
  }

  generateReport(): ProfilingReport {
    return {
      summary: this.generateSummary(),
      breakdown: this.generateBreakdown(),
      recommendations: this.generateRecommendations(),
    };
  }
}
```

## Implementation Strategy

### Phase 1: Basic Observability

1. **HTTP Transport Tracing**

   - Implement OpenTelemetry for HTTP calls
   - Add correlation headers
   - Basic metrics collection

2. **Logging Infrastructure**
   - Structured logging with correlation IDs
   - Log aggregation setup
   - Error tracking

### Phase 2: Advanced Observability

1. **IPC Transport Tracing**

   - Custom span propagation for Unix sockets
   - Process-level metrics
   - Socket monitoring

2. **Performance Profiling**
   - Detailed timing breakdowns
   - Bottleneck identification
   - Optimization recommendations

### Phase 3: Production Observability

1. **Distributed Tracing**

   - Full OpenTelemetry integration
   - Span correlation across boundaries
   - Performance attribution

2. **Advanced Metrics**
   - Custom Prometheus metrics
   - Alerting rules
   - Dashboard creation

## Monitoring Dashboards

### 1. Transport Performance Dashboard

```yaml
# grafana/dashboards/transport-performance.yaml
dashboard:
  title: "Forst Sidecar Transport Performance"
  panels:
    - title: "Request Latency by Transport"
      type: "graph"
      metrics:
        - "forst_request_duration_seconds"
        - "forst_transport_latency_seconds"

    - title: "Throughput by Transport"
      type: "graph"
      metrics:
        - "forst_requests_total"

    - title: "Error Rate by Transport"
      type: "graph"
      metrics:
        - "forst_errors_total"
```

### 2. Function Performance Dashboard

```yaml
# grafana/dashboards/function-performance.yaml
dashboard:
  title: "Forst Function Performance"
  panels:
    - title: "Function Call Duration"
      type: "heatmap"
      metrics:
        - "forst_function_duration_seconds"

    - title: "Function Error Rate"
      type: "graph"
      metrics:
        - "forst_function_errors_total"

    - title: "Memory Usage by Function"
      type: "graph"
      metrics:
        - "forst_memory_bytes"
```

## Alerting Rules

```yaml
# prometheus/alerts/forst-sidecar.yaml
groups:
  - name: forst-sidecar
    rules:
      - alert: ForstSidecarHighLatency
        expr: histogram_quantile(0.95, forst_request_duration_seconds) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Forst sidecar high latency"
          description: "95th percentile latency is {{ $value }}s"

      - alert: ForstSidecarHighErrorRate
        expr: rate(forst_errors_total[5m]) / rate(forst_requests_total[5m]) > 0.05
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Forst sidecar high error rate"
          description: "Error rate is {{ $value }}%"

      - alert: ForstSidecarTransportFailure
        expr: up{job="forst-sidecar"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Forst sidecar transport failure"
          description: "Sidecar is not responding"
```

## Debugging Tools

### 1. Transport Inspector

```typescript
// @forst/sidecar/lib/debug.ts
export class TransportInspector {
  inspectHTTPCall(url: string, options: RequestInit) {
    console.log("HTTP Call:", {
      url,
      method: options.method,
      headers: options.headers,
      body: options.body,
    });
  }

  inspectIPCCall(functionName: string, args: any[]) {
    console.log("IPC Call:", {
      function: functionName,
      args: args,
      socketPath: this.config.ipc.socketPath,
    });
  }

  inspectDirectCall(functionName: string, args: any[]) {
    console.log("Direct Call:", {
      function: functionName,
      args: args,
      memoryAddress: this.getFunctionAddress(functionName),
    });
  }
}
```

### 2. Performance Analyzer

```typescript
// @forst/sidecar/lib/analyzer.ts
export class PerformanceAnalyzer {
  analyzeCall(functionName: string, metrics: CallMetrics): Analysis {
    const analysis = {
      bottlenecks: [],
      recommendations: [],
      optimizations: [],
    };

    // Identify bottlenecks
    if (metrics.transportLatency > metrics.totalDuration * 0.5) {
      analysis.bottlenecks.push("Transport latency is high");
      analysis.recommendations.push("Consider switching to IPC transport");
    }

    if (metrics.serializationTime > metrics.totalDuration * 0.3) {
      analysis.bottlenecks.push("Serialization overhead is high");
      analysis.recommendations.push("Consider using binary serialization");
    }

    if (metrics.processingTime < metrics.totalDuration * 0.2) {
      analysis.bottlenecks.push("Transport overhead dominates");
      analysis.recommendations.push("Consider direct integration");
    }

    return analysis;
  }
}
```

## Best Practices

### 1. Tracing Best Practices

- **Always propagate trace context** across TypeScript-Forst boundary
- **Use consistent span naming** conventions
- **Add relevant attributes** to spans (function name, transport type, etc.)
- **Correlate logs with traces** using trace IDs

### 2. Metrics Best Practices

- **Measure at transport boundaries** (TypeScript → Forst, Forst → TypeScript)
- **Track both success and error rates**
- **Monitor resource usage** (memory, CPU, network)
- **Set appropriate alerting thresholds**

### 3. Logging Best Practices

- **Use structured logging** with consistent fields
- **Include correlation IDs** in all log entries
- **Sanitize sensitive data** before logging
- **Use appropriate log levels** (debug, info, warn, error)

### 4. Debugging Best Practices

- **Start with HTTP transport** for easier debugging
- **Use transport-specific debugging tools**
- **Profile performance** before and after optimizations
- **Monitor resource usage** during development

## Future Considerations

### 1. Advanced Tracing Features

- **Automatic instrumentation** for common patterns
- **Custom span processors** for business logic
- **Trace sampling** based on performance characteristics
- **Distributed context propagation** across multiple sidecars

### 2. Enhanced Metrics

- **Custom business metrics** (records processed, data size, etc.)
- **Resource utilization** tracking (memory, CPU, network)
- **Transport-specific metrics** (socket connections, buffer usage)
- **Performance regression** detection

### 3. Improved Debugging

- **Interactive debugging** tools for sidecar functions
- **Memory profiling** and leak detection
- **Concurrent execution** visualization
- **Performance bottleneck** identification

The observability challenges in the sidecar integration are significant but solvable with the right tools and practices. The key is to implement observability as a first-class concern, not an afterthought.
