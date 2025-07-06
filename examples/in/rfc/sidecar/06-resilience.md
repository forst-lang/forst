# Resilience and Error Handling

### [00-sidecar.md](00-sidecar.md) - Back to Sidecar RFC

## Overview

The sidecar integration introduces unique failure modes that don't exist in single-process applications. This document addresses how to handle failures gracefully across the TypeScript-Forst boundary.

## Key Failure Modes

### 1. Transport Failures

**Problem**: The communication channel between TypeScript and Forst can fail in different ways:

```typescript
// What happens when the sidecar is unreachable?
const result = await processData(req.body); // This can fail in multiple ways
```

**Failure Scenarios**:

- **IPC Socket Broken**: Unix domain socket becomes unavailable
- **HTTP Server Down**: Sidecar HTTP server crashes
- **Network Partition**: TypeScript can't reach sidecar
- **Transport Timeout**: Sidecar takes too long to respond

### 2. Process Failures

**Problem**: The Forst sidecar process can crash or become unresponsive:

```typescript
// Sidecar process dies during execution
const result = await processData(req.body); // Hangs indefinitely
```

**Failure Scenarios**:

- **Sidecar Crash**: Go binary panics or exits
- **Memory Exhaustion**: Sidecar runs out of memory
- **CPU Starvation**: Sidecar becomes unresponsive
- **Resource Limits**: Container kills sidecar process

### 3. Data Serialization Failures

**Problem**: Data can't be properly serialized between TypeScript and Forst:

```typescript
// Complex objects that can't be serialized
const complexData = {
  circular: {},
  functions: () => {},
  undefined: undefined,
  // These cause serialization errors
};
```

## Resilience Patterns

### 1. Circuit Breaker Pattern

```typescript
// @forst/sidecar/lib/circuit-breaker.ts
export class SidecarCircuitBreaker {
  private state: "CLOSED" | "OPEN" | "HALF_OPEN" = "CLOSED";
  private failureCount = 0;
  private lastFailureTime = 0;
  private readonly threshold = 5;
  private readonly timeout = 60000; // 1 minute

  async execute<T>(
    operation: () => Promise<T>,
    fallback?: () => Promise<T>
  ): Promise<T> {
    if (this.state === "OPEN") {
      if (Date.now() - this.lastFailureTime > this.timeout) {
        this.state = "HALF_OPEN";
      } else {
        return this.handleFallback(fallback);
      }
    }

    try {
      const result = await operation();
      this.onSuccess();
      return result;
    } catch (error) {
      this.onFailure();
      return this.handleFallback(fallback);
    }
  }

  private onSuccess(): void {
    this.failureCount = 0;
    this.state = "CLOSED";
  }

  private onFailure(): void {
    this.failureCount++;
    this.lastFailureTime = Date.now();

    if (this.failureCount >= this.threshold) {
      this.state = "OPEN";
    }
  }

  private async handleFallback<T>(fallback?: () => Promise<T>): Promise<T> {
    if (fallback) {
      return fallback();
    }
    throw new Error("Sidecar unavailable and no fallback provided");
  }
}
```

### 2. Retry with Exponential Backoff

```typescript
// @forst/sidecar/lib/retry.ts
export class SidecarRetry {
  private readonly maxRetries = 3;
  private readonly baseDelay = 1000; // 1 second

  async execute<T>(
    operation: () => Promise<T>,
    options: RetryOptions = {}
  ): Promise<T> {
    let lastError: Error;

    for (let attempt = 0; attempt <= this.maxRetries; attempt++) {
      try {
        return await operation();
      } catch (error) {
        lastError = error;

        if (attempt === this.maxRetries) {
          break;
        }

        if (this.isRetryableError(error)) {
          await this.delay(this.calculateDelay(attempt));
          continue;
        }

        // Non-retryable error, fail immediately
        throw error;
      }
    }

    throw lastError!;
  }

  private isRetryableError(error: Error): boolean {
    // Retry on network errors, timeouts, temporary failures
    const retryableErrors = [
      "ECONNREFUSED",
      "ETIMEDOUT",
      "ENOTFOUND",
      "ECONNRESET",
    ];

    return retryableErrors.some((code) => error.message.includes(code));
  }

  private calculateDelay(attempt: number): number {
    return this.baseDelay * Math.pow(2, attempt);
  }

  private delay(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }
}
```

### 3. Health Check and Recovery

```typescript
// @forst/sidecar/lib/health-check.ts
export class SidecarHealthCheck {
  private healthStatus: "healthy" | "unhealthy" | "unknown" = "unknown";
  private lastCheck = 0;
  private readonly checkInterval = 30000; // 30 seconds

  async checkHealth(): Promise<boolean> {
    if (Date.now() - this.lastCheck < this.checkInterval) {
      return this.healthStatus === "healthy";
    }

    try {
      const response = await fetch("http://localhost:8080/health", {
        method: "GET",
        timeout: 5000,
      });

      this.healthStatus = response.ok ? "healthy" : "unhealthy";
      this.lastCheck = Date.now();

      return this.healthStatus === "healthy";
    } catch (error) {
      this.healthStatus = "unhealthy";
      this.lastCheck = Date.now();
      return false;
    }
  }

  async waitForHealthy(timeout = 30000): Promise<boolean> {
    const startTime = Date.now();

    while (Date.now() - startTime < timeout) {
      if (await this.checkHealth()) {
        return true;
      }
      await this.delay(1000);
    }

    return false;
  }

  private delay(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }
}
```

### 4. Graceful Degradation

```typescript
// @forst/sidecar/lib/graceful-degradation.ts
export class GracefulDegradation {
  private fallbackImplementations = new Map<string, Function>();

  registerFallback(functionName: string, fallback: Function): void {
    this.fallbackImplementations.set(functionName, fallback);
  }

  async executeWithFallback<T>(
    functionName: string,
    args: any[],
    sidecarCall: () => Promise<T>
  ): Promise<T> {
    try {
      return await sidecarCall();
    } catch (error) {
      const fallback = this.fallbackImplementations.get(functionName);

      if (fallback) {
        console.warn(`Sidecar failed for ${functionName}, using fallback`);
        return fallback(...args);
      }

      throw error;
    }
  }
}

// Usage example
const gracefulDegradation = new GracefulDegradation();

// Register fallback implementations
gracefulDegradation.registerFallback("processData", async (data) => {
  // Slower but functional TypeScript implementation
  return processDataInTypeScript(data);
});

// Use with automatic fallback
const result = await gracefulDegradation.executeWithFallback(
  "processData",
  [req.body],
  () => processData(req.body)
);
```

## Error Handling Strategies

### 1. Transport-Specific Error Handling

```typescript
// @forst/sidecar/lib/error-handling.ts
export class SidecarErrorHandler {
  handleHTTPError(error: Error): SidecarError {
    if (error.message.includes("ECONNREFUSED")) {
      return new SidecarError(
        "SIDECAR_UNREACHABLE",
        "Sidecar server is not running"
      );
    }

    if (error.message.includes("ETIMEDOUT")) {
      return new SidecarError("SIDECAR_TIMEOUT", "Sidecar request timed out");
    }

    if (error.message.includes("ENOTFOUND")) {
      return new SidecarError(
        "SIDECAR_NOT_FOUND",
        "Sidecar endpoint not found"
      );
    }

    return new SidecarError("SIDECAR_UNKNOWN", error.message);
  }

  handleIPCError(error: Error): SidecarError {
    if (error.message.includes("ENOENT")) {
      return new SidecarError(
        "IPC_SOCKET_MISSING",
        "IPC socket file not found"
      );
    }

    if (error.message.includes("EPERM")) {
      return new SidecarError(
        "IPC_PERMISSION_DENIED",
        "IPC socket permission denied"
      );
    }

    return new SidecarError("IPC_UNKNOWN", error.message);
  }

  handleSerializationError(error: Error): SidecarError {
    return new SidecarError(
      "SERIALIZATION_ERROR",
      `Failed to serialize data: ${error.message}`
    );
  }
}
```

### 2. Error Recovery Strategies

```typescript
// @forst/sidecar/lib/recovery.ts
export class SidecarRecovery {
  private readonly circuitBreaker = new SidecarCircuitBreaker();
  private readonly retry = new SidecarRetry();
  private readonly healthCheck = new SidecarHealthCheck();
  private readonly gracefulDegradation = new GracefulDegradation();

  async executeWithRecovery<T>(
    functionName: string,
    args: any[],
    operation: () => Promise<T>
  ): Promise<T> {
    // Check health first
    if (!(await this.healthCheck.checkHealth())) {
      console.warn("Sidecar unhealthy, attempting recovery...");
      await this.attemptRecovery();
    }

    // Execute with circuit breaker and retry
    return this.circuitBreaker.execute(
      () => this.retry.execute(operation),
      () =>
        this.gracefulDegradation.executeWithFallback(
          functionName,
          args,
          operation
        )
    );
  }

  private async attemptRecovery(): Promise<void> {
    // Try to restart sidecar process
    try {
      await this.restartSidecar();
      await this.healthCheck.waitForHealthy();
    } catch (error) {
      console.error("Failed to recover sidecar:", error);
    }
  }

  private async restartSidecar(): Promise<void> {
    // Implementation depends on deployment strategy
    // Could be process restart, container restart, etc.
  }
}
```

## Monitoring and Alerting

### 1. Error Rate Monitoring

```typescript
// @forst/sidecar/lib/monitoring.ts
export class SidecarMonitoring {
  private errorCount = 0;
  private totalRequests = 0;

  recordRequest(success: boolean): void {
    this.totalRequests++;
    if (!success) {
      this.errorCount++;
    }
  }

  getErrorRate(): number {
    return this.totalRequests > 0 ? this.errorCount / this.totalRequests : 0;
  }

  shouldAlert(): boolean {
    return this.getErrorRate() > 0.1; // 10% error rate
  }
}
```

### 2. Alerting Rules

```yaml
# prometheus/alerts/sidecar-resilience.yaml
groups:
  - name: sidecar-resilience
    rules:
      - alert: SidecarHighErrorRate
        expr: rate(sidecar_errors_total[5m]) / rate(sidecar_requests_total[5m]) > 0.1
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "Sidecar high error rate"
          description: "Error rate is {{ $value }}%"

      - alert: SidecarCircuitBreakerOpen
        expr: sidecar_circuit_breaker_state == 1
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Sidecar circuit breaker open"
          description: "Sidecar is failing and circuit breaker is open"

      - alert: SidecarUnreachable
        expr: up{job="forst-sidecar"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Sidecar unreachable"
          description: "Sidecar process is not responding"
```

## Best Practices

### 1. Error Handling Best Practices

- **Always provide fallbacks** for critical functions
- **Use circuit breakers** to prevent cascading failures
- **Implement retries** with exponential backoff
- **Monitor error rates** and set appropriate alerts
- **Log errors** with sufficient context for debugging

### 2. Recovery Best Practices

- **Check health** before making requests
- **Attempt recovery** when sidecar is unhealthy
- **Graceful degradation** to TypeScript implementations
- **Automatic restart** of sidecar processes
- **Manual intervention** for persistent failures

### 3. Monitoring Best Practices

- **Track error rates** by function and transport
- **Monitor circuit breaker state** changes
- **Alert on high error rates** and unreachable sidecars
- **Log recovery attempts** and their success/failure
- **Track performance** of fallback implementations

The key to resilient sidecar integration is expecting failures and having multiple layers of protection: circuit breakers, retries, fallbacks, and monitoring. This ensures that when the sidecar fails, the application continues to function, even if with degraded performance.
