import { spawn, ChildProcess } from "node:child_process";
import { existsSync, watch } from "node:fs";
import { resolve } from "node:path";
import { ForstConfig, ServerInfo, FunctionInfo } from "./types";

export class ForstServer {
  private process: ChildProcess | null = null;
  private config: ForstConfig;
  private forstPath: string;
  private status: ServerInfo["status"] = "stopped";
  private port: number;
  private host: string;
  private functions: FunctionInfo[] = [];
  private fileWatchers: Array<() => void> = [];

  constructor(config: ForstConfig, forstPath: string) {
    this.config = config;
    this.forstPath = forstPath;
    this.port = config.port || 8080;
    this.host = config.host || "localhost";
  }

  /**
   * Start the Forst development server
   */
  async start(): Promise<ServerInfo> {
    if (this.status === "running") {
      return this.getServerInfo();
    }

    this.status = "starting";

    try {
      // Start the server process using the resolved forstPath
      await this.startServerProcess();

      // Set up file watching
      await this.setupFileWatching();

      this.status = "running";
      console.log(
        `🚀 Forst development server started on http://${this.host}:${this.port}`
      );

      return this.getServerInfo();
    } catch (error) {
      this.status = "error";
      console.error("Failed to start Forst server:", error);
      throw error;
    }
  }

  /**
   * Stop the Forst development server
   */
  async stop(): Promise<void> {
    if (this.status === "stopped") {
      return;
    }

    this.status = "stopped";

    // Stop file watchers
    this.fileWatchers.forEach((unwatch) => unwatch());
    this.fileWatchers = [];

    // Kill the server process
    if (this.process) {
      this.process.kill("SIGTERM");
      this.process = null;
    }

    console.log("🛑 Forst development server stopped");
  }

  /**
   * Restart the server
   */
  async restart(): Promise<ServerInfo> {
    console.log("🔄 Restarting Forst development server...");
    await this.stop();
    return this.start();
  }

  /**
   * Get server information
   */
  getServerInfo(): ServerInfo {
    return {
      pid: this.process?.pid || 0,
      port: this.port,
      host: this.host,
      status: this.status,
      functions: this.functions,
    };
  }

  /**
   * Start the server process
   */
  private async startServerProcess(): Promise<void> {
    const args = [
      "dev",
      "-port",
      (this.config.port || 8080).toString(),
      "-root",
      this.config.rootDir || ".",
    ];

    console.log(
      `Starting Forst server with: ${this.forstPath} ${args.join(" ")}`
    );

    this.process = spawn(this.forstPath, args, {
      stdio: ["pipe", "pipe", "pipe"],
      cwd: this.config.rootDir || process.cwd(),
    });

    // Handle process events
    this.process.on("error", (error) => {
      console.error("Forst server process error:", error);
      this.status = "error";
    });

    this.process.on("exit", (code, signal) => {
      console.log(
        `Forst server process exited with code ${code}, signal ${signal}`
      );
      this.status = "stopped";
    });

    // Handle stdout/stderr
    this.process.stdout?.on("data", (data) => {
      const output = data.toString();
      console.log(`[Forst] ${output.trim()}`);

      // Parse function discovery from output
      this.parseFunctionDiscovery(output);

      // Check if server is ready (HTTP server listening)
      if (output.includes("HTTP server listening")) {
        console.log("[DEBUG] Server ready detected from stdout");
        this.status = "running";
      }
    });

    this.process.stderr?.on("data", (data) => {
      const error = data.toString();
      console.error(`[Forst Error] ${error.trim()}`);

      // Check if server is ready (HTTP server listening)
      if (error.includes("HTTP server listening")) {
        console.log("[DEBUG] Server ready detected from stderr");
        this.status = "running";
      }
    });

    console.log("[DEBUG] Waiting for server to be ready...");
    // Wait for server to be ready
    await this.waitForServerReady();
    console.log("[DEBUG] Server ready check completed");
  }

  /**
   * Wait for the server to be ready
   */
  private async waitForServerReady(): Promise<void> {
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        reject(new Error("Server startup timeout"));
      }, 10000);

      // Simple approach: wait a bit for server to start, then check if it's responding
      setTimeout(async () => {
        try {
          // Check if server is responding
          const response = await fetch(
            `http://${this.host}:${this.port}/health`
          );
          if (response.ok) {
            this.status = "running";
            clearTimeout(timeout);
            resolve();
          } else {
            clearTimeout(timeout);
            reject(new Error("Server health check failed"));
          }
        } catch (error) {
          // If health check fails, still set as running if process is alive
          if (this.process && !this.process.killed) {
            this.status = "running";
            clearTimeout(timeout);
            resolve();
          } else {
            clearTimeout(timeout);
            reject(new Error("Server process not responding"));
          }
        }
      }, 2000); // Wait 2 seconds for server to start
    });
  }

  /**
   * Set up file watching for hot reloading
   */
  private async setupFileWatching(): Promise<void> {
    const forstDir = resolve(this.config.forstDir || "./forst");

    if (!existsSync(forstDir)) {
      return;
    }

    // Watch for .ft file changes
    const watcher = watch(
      forstDir,
      { recursive: true },
      (eventType, filename) => {
        if (filename && filename.endsWith(".ft")) {
          console.log(
            `📝 Detected change in ${filename}, triggering reload...`
          );
          this.handleFileChange();
        }
      }
    );

    this.fileWatchers.push(() => watcher.close());
  }

  /**
   * Handle file changes
   */
  private handleFileChange(): void {
    // Debounce file changes
    if (this.fileChangeTimeout) {
      clearTimeout(this.fileChangeTimeout);
    }
    this.fileChangeTimeout = setTimeout(() => {
      this.restart().catch((error) => {
        console.error("Failed to restart server after file change:", error);
      });
    }, 1000);
  }

  private fileChangeTimeout: NodeJS.Timeout | null = null;

  /**
   * Parse function discovery from server output
   */
  private parseFunctionDiscovery(output: string): void {
    // Look for function discovery patterns in the output
    const functionMatch = output.match(
      /Discovered public function: (\w+)\.(\w+)/
    );
    if (functionMatch) {
      const [, packageName, functionName] = functionMatch;
      const functionInfo: FunctionInfo = {
        package: packageName,
        name: functionName,
        supportsStreaming: false, // TODO: Parse from output
        inputType: "any",
        outputType: "any",
        parameters: [],
        returnType: "any",
        filePath: "",
      };

      // Check if function already exists
      const existingIndex = this.functions.findIndex(
        (f) => f.package === packageName && f.name === functionName
      );

      if (existingIndex === -1) {
        this.functions.push(functionInfo);
        console.log(`✨ Discovered function: ${packageName}.${functionName}`);
      }
    }
  }

  /**
   * Get the server URL
   */
  getServerUrl(): string {
    return `http://${this.host}:${this.port}`;
  }

  /**
   * Check if server is running
   */
  isRunning(): boolean {
    return this.status === "running" && this.process !== null;
  }
}
