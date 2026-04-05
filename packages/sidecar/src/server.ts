import { spawn, ChildProcess } from "node:child_process";
import { existsSync, watch } from "node:fs";
import { resolve } from "node:path";
import { ForstConfig, ServerInfo } from "./types";
import {
  DevServerChildProcessNotResponding,
  DevServerChildShutdownTimeout,
  DevServerHealthCheckHttpFailure,
  DevServerStartupTimeout,
} from "./errors";
import { serverLogger, forstLogger } from "./logger";

/**
 * Spawns and supervises `forst dev`, exposes the listen URL via `getServerUrl()`, and watches `.ft` files for reload.
 */
export class ForstServer {
  private process: ChildProcess | null = null;
  private config: ForstConfig;
  private forstPath: string;
  private status: ServerInfo["status"] = "stopped";
  private port: number;
  private host: string;
  private fileWatchers: Array<() => void> = [];
  private shutdownHandler: () => void;

  constructor(config: ForstConfig, forstPath: string) {
    this.config = config;
    this.forstPath = forstPath;
    this.port = config.port || 8080;
    this.host = config.host || "localhost";

    // Set up interrupt handlers
    this.shutdownHandler = async () => {
      serverLogger.info(
        "Received interrupt signal, shutting down gracefully..."
      );
      try {
        await this.stop();
        process.exit(0);
      } catch (error) {
        serverLogger.error("Error during shutdown:", error);
        process.exit(1);
      }
    };
  }

  /**
   * Start the Forst development server
   */
  async start(): Promise<ServerInfo> {
    if (this.status === "running") {
      return this.getServerInfo();
    }

    this.status = "starting";

    process.on("SIGINT", this.shutdownHandler);
    process.on("SIGTERM", this.shutdownHandler);

    try {
      // Start the server process using the resolved forstPath
      await this.startServerProcess();

      // Set up file watching
      await this.setupFileWatching();

      this.status = "running";
      forstLogger.info(
        `🚀 Forst development server started on http://${this.host}:${this.port}`
      );

      return this.getServerInfo();
    } catch (error) {
      this.status = "error";
      serverLogger.error("Failed to start Forst server:", error);
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

    // Remove interrupt handlers
    process.off("SIGINT", this.shutdownHandler);
    process.off("SIGTERM", this.shutdownHandler);

    // Stop file watchers
    this.fileWatchers.forEach((unwatch) => unwatch());
    this.fileWatchers = [];

    // Kill the server process with robust cleanup
    if (this.process) {
      try {
        // First try graceful shutdown with SIGTERM
        this.process.kill("SIGTERM");

        // Wait for graceful shutdown with timeout
        await new Promise<void>((resolve, reject) => {
          const timeout = setTimeout(() => {
            reject(new DevServerChildShutdownTimeout());
          }, 5000); // 5 second timeout

          this.process!.once("exit", (code, signal) => {
            clearTimeout(timeout);
            forstLogger.debug(
              `Forst server process exited gracefully with code ${code}, signal ${signal}`
            );
            resolve();
          });
        });
      } catch (error) {
        serverLogger.warn("Graceful shutdown failed, forcing kill:", error);

        // Force kill with SIGKILL if graceful shutdown fails
        try {
          this.process.kill("SIGKILL");

          // Wait a bit for the process to be killed
          await new Promise<void>((resolve) => {
            setTimeout(() => {
              forstLogger.info("Forst server process force killed");
              resolve();
            }, 1000);
          });
        } catch (killError) {
          serverLogger.error("Failed to force kill process:", killError);
        }
      } finally {
        this.process = null;
      }
    }

    forstLogger.info("🛑 Forst development server stopped");
  }

  /**
   * Restart the server
   */
  async restart(): Promise<ServerInfo> {
    serverLogger.info("🔄 Restarting Forst development server...");
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
    };
  }

  /**
   * Project root passed to `forst dev -root` and used as the child process cwd.
   * Uses `rootDir` if set, else `forstDir`, else `./forst` so discovery matches the `.ft` tree by default.
   */
  private effectiveProjectRoot(): string {
    return resolve(this.config.rootDir ?? this.config.forstDir ?? "./forst");
  }

  /**
   * Directory watched for `.ft` changes. Prefer `forstDir`, then `rootDir`, then {@link effectiveProjectRoot}.
   */
  private effectiveWatchDir(): string {
    return resolve(
      this.config.forstDir ?? this.config.rootDir ?? this.effectiveProjectRoot()
    );
  }

  /**
   * Start the server process
   */
  private async startServerProcess(): Promise<void> {
    const root = this.effectiveProjectRoot();
    const args = [
      "dev",
      "-port",
      (this.config.port || 8080).toString(),
      "-root",
      root,
      "-log-level",
      this.config.logLevel || "info",
    ];

    serverLogger.info(
      `Starting Forst server with: ${this.forstPath} ${args.join(" ")}`
    );

    this.process = spawn(this.forstPath, args, {
      stdio: ["pipe", "pipe", "pipe"],
      cwd: root,
    });

    // Handle process events
    this.process.on("error", (error) => {
      serverLogger.error("Forst server process error:", error);
      this.status = "error";
    });

    this.process.on("exit", (code, signal) => {
      serverLogger.info(
        `Forst server process exited with code ${code}, signal ${signal}`
      );
      this.status = "stopped";
    });

    // Handle stdout/stderr
    this.process.stdout?.on("data", (data) => {
      const output = data.toString();
      const trimmedOutput = output.trim();

      // Only log non-empty output
      if (trimmedOutput) {
        forstLogger.info(`[Forst] ${trimmedOutput}`);
      }

      // Check if server is ready (HTTP server listening)
      if (output.includes("HTTP server listening")) {
        serverLogger.debug("Server ready detected from stdout");
        this.status = "running";
      }
    });

    this.process.stderr?.on("data", (data) => {
      const error = data.toString();
      const trimmedError = error.trim();

      // Only log non-empty error output
      if (trimmedError) {
        // Forst compiler output goes to stderr but isn't necessarily an error
        // Check if it looks like an actual error vs debug/info output
        const logMethods = {
          "level=debug": forstLogger.debug,
          "level=info": forstLogger.info,
          "level=warn": forstLogger.warn,
          "level=error": forstLogger.error,
        } as const;

        // Find the appropriate log level based on the error message, defaulting to info
        const [, logMethod] = Object.entries(logMethods).find(([level]) =>
          trimmedError.includes(level)
        ) || [null, forstLogger.info];

        logMethod(trimmedError);
      }

      // Check if server is ready (HTTP server listening)
      if (error.includes("HTTP server listening")) {
        serverLogger.debug("Server ready detected from stderr");
        this.status = "running";
      }
    });

    serverLogger.debug("Waiting for server to be ready...");
    // Wait for server to be ready
    await this.waitForServerReady();
    serverLogger.debug("Server ready check completed");
  }

  /**
   * Wait for the server to be ready
   */
  private async waitForServerReady(): Promise<void> {
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        serverLogger.error("⏰ Server startup timeout after 10 seconds");
        reject(new DevServerStartupTimeout());
      }, 10000);

      // Simple approach: wait a bit for server to start, then check if it's responding
      setTimeout(async () => {
        try {
          const healthUrl = `http://${this.host}:${this.port}/health`;
          serverLogger.debug(`🏥 Checking server health at: ${healthUrl}`);

          // Check if server is responding
          const response = await fetch(healthUrl);
          serverLogger.debug(
            `🏥 Health check response status: ${response.status}`
          );

          if (response.ok) {
            const healthData = await response.text();
            serverLogger.debug(`🏥 Health check response body: ${healthData}`);
            this.status = "running";
            clearTimeout(timeout);
            serverLogger.info("✅ Server is ready and responding");
            resolve();
          } else {
            const errorText = await response.text();
            serverLogger.error(
              `❌ Server health check failed with status ${response.status}: ${errorText}`
            );
            clearTimeout(timeout);
            reject(
              new DevServerHealthCheckHttpFailure(response.status, errorText)
            );
          }
        } catch (error) {
          serverLogger.error(`❌ Health check request failed:`, error);
          // If health check fails, still set as running if process is alive
          if (this.process && !this.process.killed) {
            serverLogger.warn(
              "⚠️  Health check failed but process is alive, marking as running"
            );
            this.status = "running";
            clearTimeout(timeout);
            resolve();
          } else {
            serverLogger.error(
              "❌ Health check failed and process is not alive"
            );
            clearTimeout(timeout);
            reject(new DevServerChildProcessNotResponding());
          }
        }
      }, 2000); // Wait 2 seconds for server to start
    });
  }

  /**
   * Set up file watching for hot reloading
   */
  private async setupFileWatching(): Promise<void> {
    const forstDir = this.effectiveWatchDir();

    if (!existsSync(forstDir)) {
      return;
    }

    // Watch for .ft file changes
    const watcher = watch(
      forstDir,
      { recursive: true },
      (_eventType, filename) => {
        if (filename && filename.endsWith(".ft")) {
          forstLogger.info(
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
        serverLogger.error(
          "Failed to restart server after file change:",
          error
        );
      });
    }, 1000);
  }

  private fileChangeTimeout: NodeJS.Timeout | null = null;

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
