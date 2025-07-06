# Development Workflow and Debugging

### [00-sidecar.md](00-sidecar.md) - Back to Sidecar RFC

## Overview

The sidecar integration introduces unique challenges for development workflows. This document covers hot reloading, debugging strategies, and development tooling specific to the TypeScript-Forst boundary.

## Development Challenges

### 1. Hot Reloading Complexity

**Problem**: Traditional hot reloading doesn't work across language boundaries:

```typescript
// TypeScript changes are hot-reloaded
app.post("/process-data", async (req, res) => {
  // This change is immediately available
  const result = await processData(req.body);
  res.json(result);
});
```

```go
// Forst changes require compilation and restart
func processData(input ProcessDataInput) {
  // This change requires recompilation and sidecar restart
  return { processed: len(input.records), status: "success" }
}
```

**Challenges**:

- **Compilation Time**: Forst files need to be compiled to Go
- **Process Restart**: Sidecar process needs to be restarted
- **State Loss**: In-memory state is lost on restart
- **Connection Reset**: Active connections are dropped

### 2. Debugging Across Boundaries

**Problem**: Debugging becomes complex when issues span TypeScript and Forst:

```typescript
// Where is the bug? TypeScript, transport, or Forst?
const result = await processData(req.body);
console.log(result); // Wrong data - which layer is wrong?
```

**Debugging Challenges**:

- **Error Attribution**: Which layer caused the error?
- **State Inspection**: How to inspect Forst state from TypeScript?
- **Step-through Debugging**: Can't step through both languages simultaneously
- **Log Correlation**: Matching logs across processes

### 3. Development Environment Setup

**Problem**: Setting up a development environment is more complex:

```bash
# Need to manage multiple processes
npm run dev          # TypeScript development server
forst compile        # Compile Forst files
./dist/forst/sidecar # Run sidecar process
```

**Setup Challenges**:

- **Process Coordination**: Starting/stopping multiple processes
- **Port Management**: Avoiding port conflicts
- **File Watching**: Watching both TypeScript and Forst files
- **Environment Variables**: Managing config across processes

## Development Workflow Solutions

### 1. Integrated Development Server

```typescript
// @forst/sidecar/lib/dev-server.ts
export class ForstDevServer {
  private typescriptServer: any;
  private forstCompiler: ForstCompiler;
  private sidecarProcess: SidecarProcessManager;
  private fileWatcher: chokidar.FSWatcher;
  private config: DevConfig;

  constructor(config: DevConfig) {
    this.config = config;
    this.forstCompiler = new ForstCompiler();
    this.sidecarProcess = new SidecarProcessManager(config.sidecar);
    this.setupFileWatching();
  }

  async start(): Promise<void> {
    console.log("Starting Forst development server...");

    // Start TypeScript development server
    await this.startTypeScriptServer();

    // Compile and start sidecar
    await this.compileAndStartSidecar();

    // Start file watching
    this.startFileWatching();

    console.log("Development server ready!");
    console.log(
      `- TypeScript: http://localhost:${this.config.typescript.port}`
    );
    console.log(`- Sidecar: http://localhost:${this.config.sidecar.http.port}`);
  }

  private async startTypeScriptServer(): Promise<void> {
    // Start your existing TypeScript dev server
    // This could be Next.js, Express, etc.
    console.log("Starting TypeScript development server...");
  }

  private async compileAndStartSidecar(): Promise<void> {
    console.log("Compiling Forst files...");
    await this.forstCompiler.compileAll();

    console.log("Starting sidecar process...");
    await this.sidecarProcess.start();
  }

  private setupFileWatching(): void {
    this.fileWatcher = chokidar.watch([
      "./src/**/*.ts",
      "./src/**/*.tsx",
      "./forst/**/*.ft",
    ]);

    this.fileWatcher.on("change", (path) => {
      this.handleFileChange(path);
    });
  }

  private async handleFileChange(path: string): Promise<void> {
    if (path.endsWith(".ft")) {
      await this.handleForstChange(path);
    } else {
      await this.handleTypeScriptChange(path);
    }
  }

  private async handleForstChange(path: string): Promise<void> {
    console.log(`Forst file changed: ${path}`);

    try {
      // Recompile the specific file
      await this.forstCompiler.compile(path);

      // Restart sidecar with hot reload
      await this.sidecarProcess.hotReload();

      console.log("Forst changes applied successfully");
    } catch (error) {
      console.error("Failed to apply Forst changes:", error);
    }
  }

  private async handleTypeScriptChange(path: string): Promise<void> {
    console.log(`TypeScript file changed: ${path}`);
    // TypeScript changes are handled by the existing dev server
  }

  private startFileWatching(): void {
    console.log("File watching started");
  }
}
```

### 2. Hot Reload Implementation

```typescript
// @forst/sidecar/lib/hot-reload.ts
export class SidecarHotReload {
  private process: ChildProcess | null = null;
  private config: SidecarConfig;

  constructor(process: ChildProcess, config: SidecarConfig) {
    this.process = process;
    this.config = config;
  }

  async hotReload(): Promise<void> {
    if (!this.process) {
      throw new Error("No sidecar process to reload");
    }

    try {
      // Send reload signal to sidecar
      await this.sendReloadSignal();

      // Wait for sidecar to be ready
      await this.waitForReady();

      console.log("Sidecar hot reload completed");
    } catch (error) {
      console.error("Hot reload failed, restarting sidecar...");
      await this.restartSidecar();
    }
  }

  private async sendReloadSignal(): Promise<void> {
    if (this.config.transport === "http") {
      await this.sendHTTPReloadSignal();
    } else if (this.config.transport === "ipc") {
      await this.sendIPCReloadSignal();
    }
  }

  private async sendHTTPReloadSignal(): Promise<void> {
    const response = await fetch(
      `http://localhost:${this.config.http.port}/reload`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
      }
    );

    if (!response.ok) {
      throw new Error(`HTTP reload signal failed: ${response.statusText}`);
    }
  }

  private async sendIPCReloadSignal(): Promise<void> {
    const net = require("net");
    const socket = new net.Socket();

    await new Promise((resolve, reject) => {
      socket.connect(this.config.ipc.socketPath, () => {
        socket.write(JSON.stringify({ action: "reload" }));
        socket.end();
        resolve(undefined);
      });

      socket.on("error", reject);
      socket.setTimeout(5000, () => {
        socket.destroy();
        reject(new Error("IPC reload signal timeout"));
      });
    });
  }

  private async waitForReady(): Promise<void> {
    const healthCheck = new SidecarHealthCheck(this.config);
    const maxAttempts = 10;
    let attempts = 0;

    while (attempts < maxAttempts) {
      if (await healthCheck.checkHealth()) {
        return;
      }

      attempts++;
      await this.delay(500);
    }

    throw new Error("Sidecar not ready after hot reload");
  }

  private async restartSidecar(): Promise<void> {
    // Implementation would restart the entire sidecar process
    // This is a fallback when hot reload fails
  }

  private delay(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }
}
```

### 3. Debugging Tools

```typescript
// @forst/sidecar/lib/debug-tools.ts
export class SidecarDebugTools {
  private config: SidecarConfig;

  constructor(config: SidecarConfig) {
    this.config = config;
  }

  // Debug transport communication
  async debugTransport(functionName: string, args: any[]): Promise<void> {
    console.log("=== Transport Debug ===");
    console.log(`Function: ${functionName}`);
    console.log(`Args:`, JSON.stringify(args, null, 2));
    console.log(`Transport: ${this.config.transport}`);

    if (this.config.transport === "http") {
      await this.debugHTTPTransport(functionName, args);
    } else if (this.config.transport === "ipc") {
      await this.debugIPCTransport(functionName, args);
    }
  }

  private async debugHTTPTransport(
    functionName: string,
    args: any[]
  ): Promise<void> {
    const url = `http://localhost:${this.config.http.port}/${functionName}`;
    const body = JSON.stringify(args);

    console.log(`HTTP Request: POST ${url}`);
    console.log(`Request Body:`, body);

    try {
      const response = await fetch(url, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body,
      });

      const result = await response.text();
      console.log(`Response Status: ${response.status}`);
      console.log(`Response Body:`, result);
    } catch (error) {
      console.error(`HTTP Error:`, error);
    }
  }

  private async debugIPCTransport(
    functionName: string,
    args: any[]
  ): Promise<void> {
    const net = require("net");
    const socket = new net.Socket();

    console.log(`IPC Socket: ${this.config.ipc.socketPath}`);
    console.log(`IPC Request:`, { function: functionName, args });

    try {
      await new Promise((resolve, reject) => {
        socket.connect(this.config.ipc.socketPath, () => {
          const request = JSON.stringify({ function: functionName, args });
          socket.write(request);
          resolve(undefined);
        });

        socket.on("data", (data) => {
          console.log(`IPC Response:`, data.toString());
          socket.end();
        });

        socket.on("error", reject);
        socket.setTimeout(5000, () => {
          socket.destroy();
          reject(new Error("IPC timeout"));
        });
      });
    } catch (error) {
      console.error(`IPC Error:`, error);
    }
  }

  // Debug sidecar state
  async debugSidecarState(): Promise<void> {
    console.log("=== Sidecar State Debug ===");

    try {
      const response = await fetch(
        `http://localhost:${this.config.http.port}/debug/state`
      );
      const state = await response.json();

      console.log("Sidecar State:", JSON.stringify(state, null, 2));
    } catch (error) {
      console.error("Failed to get sidecar state:", error);
    }
  }

  // Debug performance
  async debugPerformance(functionName: string, args: any[]): Promise<void> {
    console.log("=== Performance Debug ===");

    const start = performance.now();
    const result = await this.callFunction(functionName, args);
    const duration = performance.now() - start;

    console.log(`Function: ${functionName}`);
    console.log(`Duration: ${duration.toFixed(2)}ms`);
    console.log(`Result Size: ${JSON.stringify(result).length} bytes`);
  }

  private async callFunction(functionName: string, args: any[]): Promise<any> {
    // Implementation depends on the transport
    // This is a simplified version
    return {};
  }
}
```

### 4. Development Configuration

```typescript
// @forst/sidecar/lib/dev-config.ts
export interface DevConfig {
  // TypeScript development server
  typescript: {
    port: number;
    host: string;
  };

  // Sidecar configuration
  sidecar: SidecarConfig;

  // Development features
  features: {
    hotReload: boolean;
    debugMode: boolean;
    fileWatching: boolean;
    autoRestart: boolean;
  };

  // Debugging options
  debugging: {
    transportDebug: boolean;
    performanceDebug: boolean;
    stateDebug: boolean;
    logLevel: "debug" | "info" | "warn" | "error";
  };
}

export const defaultDevConfig: DevConfig = {
  typescript: {
    port: 3000,
    host: "localhost",
  },
  sidecar: {
    transport: "http",
    http: {
      port: 8080,
      host: "localhost",
      cors: true,
    },
    debug: true,
    environment: "development",
    logLevel: "debug",
  },
  features: {
    hotReload: true,
    debugMode: true,
    fileWatching: true,
    autoRestart: true,
  },
  debugging: {
    transportDebug: false,
    performanceDebug: false,
    stateDebug: false,
    logLevel: "info",
  },
};
```

## Development Workflow

### 1. Setup Development Environment

```bash
# Install dependencies
npm install @forst/sidecar

# Start development server (handles everything)
npm run dev
```

```typescript
// package.json scripts
{
  "scripts": {
    "dev": "forst-sidecar dev",
    "dev:debug": "forst-sidecar dev --debug",
    "dev:no-hot-reload": "forst-sidecar dev --no-hot-reload",
    "build": "forst-sidecar build",
    "start": "forst-sidecar start"
  }
}
```

### 2. Development Workflow

```typescript
// 1. Start development server
const devServer = new ForstDevServer(config);
await devServer.start();

// 2. Make changes to TypeScript files
// - Hot reloaded automatically
// - No sidecar restart needed

// 3. Make changes to Forst files
// - Automatically compiled
// - Sidecar hot reloaded
// - No manual restart needed

// 4. Debug issues
const debugTools = new SidecarDebugTools(config);
await debugTools.debugTransport("processData", [req.body]);
await debugTools.debugSidecarState();
await debugTools.debugPerformance("processData", [req.body]);
```

### 3. Debugging Workflow

```typescript
// When something goes wrong:

// 1. Check if sidecar is running
const isHealthy = await sidecar.isHealthy();
console.log("Sidecar healthy:", isHealthy);

// 2. Debug transport communication
await debugTools.debugTransport("processData", [req.body]);

// 3. Check sidecar state
await debugTools.debugSidecarState();

// 4. Check performance
await debugTools.debugPerformance("processData", [req.body]);

// 5. Check logs
console.log("TypeScript logs:", typescriptLogs);
console.log("Sidecar logs:", sidecarLogs);
```

## Best Practices

### 1. Development Best Practices

- **Use hot reload** for faster iteration
- **Enable debug mode** during development
- **Watch file changes** automatically
- **Log transport communication** when debugging
- **Monitor performance** during development

### 2. Debugging Best Practices

- **Start with transport debugging** to isolate issues
- **Check sidecar health** before debugging
- **Use performance debugging** to identify bottlenecks
- **Correlate logs** across TypeScript and Forst
- **Use state debugging** to inspect sidecar state

### 3. Workflow Best Practices

- **Automate setup** with development server
- **Provide clear error messages** for common issues
- **Support multiple debugging levels** (transport, performance, state)
- **Enable easy restart** when hot reload fails
- **Maintain development/production parity**

The key to successful development workflow is making the sidecar integration feel as seamless as possible, with automatic compilation, hot reloading, and comprehensive debugging tools. This allows developers to focus on their business logic rather than managing the integration complexity.
