# Process Management and Lifecycle

### [00-sidecar.md](00-sidecar.md) - Back to Sidecar RFC

## Overview

The sidecar integration requires careful management of the Forst process lifecycle. This document covers how to start, monitor, and maintain the sidecar process alongside the TypeScript application.

## Sidecar Lifecycle

### 1. Process Startup

**Problem**: The sidecar needs to be started before TypeScript can use it:

```typescript
// TypeScript needs to wait for sidecar to be ready
const sidecar = new ForstSidecar();
await sidecar.start(); // How do we know when it's ready?

// Now we can use it
const result = await processData(req.body);
```

**Startup Challenges**:

- **Compilation Time**: Forst files need to be compiled to Go
- **Port Binding**: HTTP server needs to bind to available port
- **Socket Creation**: IPC socket needs proper permissions
- **Health Check**: Process needs to respond to health checks

### 2. Process Monitoring

**Problem**: The sidecar process can fail silently:

```typescript
// Sidecar process dies but TypeScript doesn't know
const result = await processData(req.body); // Hangs indefinitely
```

**Monitoring Challenges**:

- **Process Death**: Go binary crashes or exits
- **Resource Exhaustion**: Memory/CPU limits exceeded
- **Network Issues**: Port conflicts, firewall blocks
- **Permission Issues**: Socket file permissions, file system access

## Process Management Implementation

### 1. Sidecar Process Manager

```typescript
// @forst/sidecar/lib/process-manager.ts
export class SidecarProcessManager {
  private process: ChildProcess | null = null;
  private config: SidecarConfig;
  private healthCheck: SidecarHealthCheck;

  constructor(config: SidecarConfig) {
    this.config = config;
    this.healthCheck = new SidecarHealthCheck(config);
  }

  async start(): Promise<void> {
    // Compile Forst files first
    await this.compileForstFiles();

    // Start the sidecar process
    await this.spawnProcess();

    // Wait for it to be healthy
    await this.waitForHealthy();
  }

  async stop(): Promise<void> {
    if (this.process) {
      this.process.kill("SIGTERM");
      await this.waitForProcessExit();
    }
  }

  async restart(): Promise<void> {
    await this.stop();
    await this.start();
  }

  private async compileForstFiles(): Promise<void> {
    const { exec } = require("child_process");
    const util = require("util");
    const execAsync = util.promisify(exec);

    try {
      console.log("Compiling Forst files...");
      await execAsync("forst compile ./forst -o ./dist/forst");
      console.log("Forst compilation complete");
    } catch (error) {
      throw new Error(`Forst compilation failed: ${error.message}`);
    }
  }

  private async spawnProcess(): Promise<void> {
    const { spawn } = require("child_process");

    const args = this.buildProcessArgs();
    this.process = spawn("./dist/forst/sidecar", args, {
      stdio: ["pipe", "pipe", "pipe"],
      env: this.buildEnvironment(),
    });

    // Handle process events
    this.process.on("error", (error) => {
      console.error("Sidecar process error:", error);
    });

    this.process.on("exit", (code, signal) => {
      console.log(`Sidecar process exited with code ${code}, signal ${signal}`);
      this.process = null;
    });

    // Handle stdout/stderr
    this.process.stdout?.on("data", (data) => {
      console.log("Sidecar stdout:", data.toString());
    });

    this.process.stderr?.on("data", (data) => {
      console.error("Sidecar stderr:", data.toString());
    });
  }

  private buildProcessArgs(): string[] {
    const args = [];

    if (this.config.transport === "http") {
      args.push("--http-port", this.config.http.port.toString());
    }

    if (this.config.transport === "ipc") {
      args.push("--ipc-socket", this.config.ipc.socketPath);
    }

    if (this.config.debug) {
      args.push("--debug");
    }

    return args;
  }

  private buildEnvironment(): NodeJS.ProcessEnv {
    return {
      ...process.env,
      FORST_ENV: this.config.environment,
      FORST_LOG_LEVEL: this.config.logLevel,
    };
  }

  private async waitForHealthy(): Promise<void> {
    const maxAttempts = 30; // 30 seconds
    let attempts = 0;

    while (attempts < maxAttempts) {
      if (await this.healthCheck.checkHealth()) {
        console.log("Sidecar is healthy and ready");
        return;
      }

      attempts++;
      await this.delay(1000);
    }

    throw new Error("Sidecar failed to become healthy within timeout");
  }

  private async waitForProcessExit(): Promise<void> {
    return new Promise((resolve) => {
      if (this.process) {
        this.process.on("exit", () => resolve());
      } else {
        resolve();
      }
    });
  }

  private delay(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }
}
```

### 2. Health Check Implementation

```typescript
// @forst/sidecar/lib/health-check.ts
export class SidecarHealthCheck {
  private config: SidecarConfig;
  private lastCheck = 0;
  private healthStatus: "healthy" | "unhealthy" | "unknown" = "unknown";

  constructor(config: SidecarConfig) {
    this.config = config;
  }

  async checkHealth(): Promise<boolean> {
    // Cache health status for 5 seconds
    if (Date.now() - this.lastCheck < 5000) {
      return this.healthStatus === "healthy";
    }

    try {
      if (this.config.transport === "http") {
        return await this.checkHTTPHealth();
      } else if (this.config.transport === "ipc") {
        return await this.checkIPCHealth();
      } else {
        return await this.checkDirectHealth();
      }
    } catch (error) {
      this.healthStatus = "unhealthy";
      this.lastCheck = Date.now();
      return false;
    }
  }

  private async checkHTTPHealth(): Promise<boolean> {
    const response = await fetch(
      `http://localhost:${this.config.http.port}/health`,
      {
        method: "GET",
        timeout: 5000,
      }
    );

    this.healthStatus = response.ok ? "healthy" : "unhealthy";
    this.lastCheck = Date.now();
    return this.healthStatus === "healthy";
  }

  private async checkIPCHealth(): Promise<boolean> {
    const fs = require("fs");
    const path = require("path");

    // Check if socket file exists and is accessible
    if (!fs.existsSync(this.config.ipc.socketPath)) {
      this.healthStatus = "unhealthy";
      this.lastCheck = Date.now();
      return false;
    }

    // Try to connect to socket
    try {
      const net = require("net");
      const socket = new net.Socket();

      await new Promise((resolve, reject) => {
        socket.connect(this.config.ipc.socketPath, () => {
          socket.end();
          resolve(undefined);
        });

        socket.on("error", reject);
        socket.setTimeout(5000, () => {
          socket.destroy();
          reject(new Error("IPC connection timeout"));
        });
      });

      this.healthStatus = "healthy";
      this.lastCheck = Date.now();
      return true;
    } catch (error) {
      this.healthStatus = "unhealthy";
      this.lastCheck = Date.now();
      return false;
    }
  }

  private async checkDirectHealth(): Promise<boolean> {
    // For direct integration, check if the shared library is loaded
    try {
      // This would check if the FFI library is accessible
      // Implementation depends on the specific FFI library used
      this.healthStatus = "healthy";
      this.lastCheck = Date.now();
      return true;
    } catch (error) {
      this.healthStatus = "unhealthy";
      this.lastCheck = Date.now();
      return false;
    }
  }
}
```

### 3. Resource Monitoring

```typescript
// @forst/sidecar/lib/resource-monitor.ts
export class SidecarResourceMonitor {
  private process: ChildProcess | null = null;
  private config: SidecarConfig;

  constructor(process: ChildProcess, config: SidecarConfig) {
    this.process = process;
    this.config = config;
  }

  startMonitoring(): void {
    if (!this.process) return;

    // Monitor memory usage
    setInterval(() => {
      this.checkMemoryUsage();
    }, 30000); // Every 30 seconds

    // Monitor CPU usage
    setInterval(() => {
      this.checkCPUUsage();
    }, 60000); // Every minute

    // Monitor process status
    setInterval(() => {
      this.checkProcessStatus();
    }, 10000); // Every 10 seconds
  }

  private async checkMemoryUsage(): Promise<void> {
    if (!this.process?.pid) return;

    try {
      const { exec } = require("child_process");
      const util = require("util");
      const execAsync = util.promisify(exec);

      const { stdout } = await execAsync(`ps -o rss= -p ${this.process.pid}`);
      const memoryUsageMB = parseInt(stdout.trim()) / 1024;

      if (memoryUsageMB > this.config.resources.maxMemoryMB) {
        console.warn(`Sidecar memory usage high: ${memoryUsageMB}MB`);
        this.handleResourceExhaustion("memory", memoryUsageMB);
      }
    } catch (error) {
      console.error("Failed to check memory usage:", error);
    }
  }

  private async checkCPUUsage(): Promise<void> {
    if (!this.process?.pid) return;

    try {
      const { exec } = require("child_process");
      const util = require("util");
      const execAsync = util.promisify(exec);

      const { stdout } = await execAsync(`ps -o %cpu= -p ${this.process.pid}`);
      const cpuUsage = parseFloat(stdout.trim());

      if (cpuUsage > this.config.resources.maxCPUPercent) {
        console.warn(`Sidecar CPU usage high: ${cpuUsage}%`);
        this.handleResourceExhaustion("cpu", cpuUsage);
      }
    } catch (error) {
      console.error("Failed to check CPU usage:", error);
    }
  }

  private checkProcessStatus(): void {
    if (!this.process) return;

    if (this.process.killed) {
      console.error("Sidecar process was killed");
      this.handleProcessDeath();
    }
  }

  private handleResourceExhaustion(
    type: "memory" | "cpu",
    value: number
  ): void {
    // Log the issue
    console.error(`Sidecar ${type} exhaustion: ${value}`);

    // Emit event for external handling
    this.emit("resource-exhaustion", { type, value });

    // Optionally restart the process
    if (this.config.autoRestart) {
      console.log("Auto-restarting sidecar due to resource exhaustion");
      this.restart();
    }
  }

  private handleProcessDeath(): void {
    console.error("Sidecar process died unexpectedly");

    // Emit event for external handling
    this.emit("process-death");

    // Optionally restart the process
    if (this.config.autoRestart) {
      console.log("Auto-restarting sidecar due to process death");
      this.restart();
    }
  }

  private async restart(): Promise<void> {
    // Implementation would depend on the process manager
    // This is just a placeholder
  }
}
```

## Configuration

### 1. Sidecar Configuration

```typescript
// @forst/sidecar/lib/config.ts
export interface SidecarConfig {
  // Transport configuration
  transport: "http" | "ipc" | "direct";

  // HTTP configuration
  http?: {
    port: number;
    host?: string;
    cors?: boolean;
  };

  // IPC configuration
  ipc?: {
    socketPath: string;
    permissions?: number;
  };

  // Process configuration
  process?: {
    autoRestart: boolean;
    restartDelay: number;
    maxRestarts: number;
  };

  // Resource limits
  resources?: {
    maxMemoryMB: number;
    maxCPUPercent: number;
    maxFileDescriptors: number;
  };

  // Health check configuration
  healthCheck?: {
    interval: number;
    timeout: number;
    failureThreshold: number;
  };

  // Development configuration
  debug?: boolean;
  environment?: string;
  logLevel?: string;
}

export const defaultConfig: SidecarConfig = {
  transport: "http",
  http: {
    port: 8080,
    host: "localhost",
    cors: true,
  },
  ipc: {
    socketPath: "/tmp/forst.sock",
    permissions: 0o600,
  },
  process: {
    autoRestart: true,
    restartDelay: 5000,
    maxRestarts: 5,
  },
  resources: {
    maxMemoryMB: 512,
    maxCPUPercent: 80,
    maxFileDescriptors: 1024,
  },
  healthCheck: {
    interval: 30000,
    timeout: 5000,
    failureThreshold: 3,
  },
  debug: false,
  environment: "development",
  logLevel: "info",
};
```

### 2. Process Manager Usage

```typescript
// @forst/sidecar/lib/sidecar.ts
export class ForstSidecar {
  private processManager: SidecarProcessManager;
  private config: SidecarConfig;
  private isStarted = false;

  constructor(config: Partial<SidecarConfig> = {}) {
    this.config = { ...defaultConfig, ...config };
    this.processManager = new SidecarProcessManager(this.config);
  }

  async start(): Promise<void> {
    if (this.isStarted) {
      console.warn("Sidecar already started");
      return;
    }

    try {
      console.log("Starting Forst sidecar...");
      await this.processManager.start();
      this.isStarted = true;
      console.log("Forst sidecar started successfully");
    } catch (error) {
      console.error("Failed to start sidecar:", error);
      throw error;
    }
  }

  async stop(): Promise<void> {
    if (!this.isStarted) {
      console.warn("Sidecar not started");
      return;
    }

    try {
      console.log("Stopping Forst sidecar...");
      await this.processManager.stop();
      this.isStarted = false;
      console.log("Forst sidecar stopped successfully");
    } catch (error) {
      console.error("Failed to stop sidecar:", error);
      throw error;
    }
  }

  async restart(): Promise<void> {
    await this.stop();
    await this.start();
  }

  isHealthy(): Promise<boolean> {
    return this.processManager.healthCheck.checkHealth();
  }
}
```

## Best Practices

### 1. Process Management Best Practices

- **Always check health** before using sidecar functions
- **Implement graceful shutdown** with proper signal handling
- **Monitor resource usage** to prevent exhaustion
- **Auto-restart on failures** with exponential backoff
- **Log process events** for debugging

### 2. Health Check Best Practices

- **Check multiple endpoints** (HTTP, IPC, process status)
- **Cache health status** to avoid excessive checks
- **Use appropriate timeouts** for different transports
- **Handle transport-specific failures** differently
- **Provide detailed health information** for debugging

### 3. Resource Management Best Practices

- **Set appropriate resource limits** based on expected load
- **Monitor memory and CPU usage** continuously
- **Implement circuit breakers** for resource exhaustion
- **Provide resource usage metrics** for monitoring
- **Plan for resource scaling** as usage grows

The key to successful sidecar process management is treating it like a separate service that needs proper lifecycle management, health monitoring, and resource management. This ensures the sidecar remains reliable and responsive throughout the application's lifecycle.
