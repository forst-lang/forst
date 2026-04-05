import { logger } from "./logger";

import type { ForstConfig, FunctionInfo, InvokeResponse, ServerInfo, StreamingResult } from "./types";
import type { Request, RequestHandler } from "express";
import { CompilerNotFound, SidecarNotStarted } from "./errors";
import { ForstUtils } from "./utils";
import { ForstSidecarClient } from "./client";
import { ForstServer } from "./server";

async function ensureForstBinary(): Promise<string> {
  return await ForstUtils.ensureCompiler();
}

/**
 * Main Forst sidecar class that provides the complete integration
 */
export class ForstSidecar {
  private server?: ForstServer;
  private client: ForstSidecarClient | null = null;
  private forstPath: string | null = null;
  private config: ForstConfig;
  private _customCompilerPath: string | null = null; // Intentionally awkward - don't use this normally

  constructor(config?: Partial<ForstConfig>) {
    this.config = {
      mode: "development",
      port: 8080,
      host: "localhost",
      logLevel: "info",
      ...config,
    };
  }

  /**
   * Set a custom compiler path (INTENTIONALLY AWKWARD - don't use this normally)
   * This bypasses the normal binary resolution and should only be used for testing
   * @param path - The path to the custom Forst compiler binary
   */
  _setCustomCompilerPath(path: string): void {
    logger.warn(
      "⚠️  Using custom compiler path - this is intentionally awkward and not recommended for normal use"
    );
    this._customCompilerPath = path;
  }

  /**
   * Start the sidecar development server
   */
  async start(): Promise<void> {
    logger.info("🚀 Starting Forst sidecar...");

    if (this._customCompilerPath) {
      this.forstPath = this._customCompilerPath;
      if (!this.forstPath) {
        throw new CompilerNotFound("Custom compiler path was empty.");
      }
      logger.info(`🔧 Using custom compiler path: ${this.forstPath}`);
    } else {
      try {
        this.forstPath = await ensureForstBinary();
      } catch (e) {
        throw new CompilerNotFound(
          "Failed to download or resolve the Forst compiler.",
          { cause: e }
        );
      }
    }

    // Initialize server with the resolved forst path
    this.server = new ForstServer(this.config, this.forstPath);

    // Start the development server
    await this.server.start();

    // Initialize the client
    this.client = new ForstSidecarClient({
      baseUrl: this.server.getServerUrl(),
      timeout: 30000,
      retries: 1,
    });

    logger.info("✅ Forst sidecar started successfully");
  }

  /**
   * Stop the sidecar development server
   */
  async stop(): Promise<void> {
    if (!this.server) {
      logger.debug("ForstSidecar.stop(): server was never started; skipping.");
      return;
    }
    logger.info("🛑 Stopping Forst sidecar...");
    await this.server.stop();
    this.client = null;
    logger.info("✅ Forst sidecar stopped");
  }

  /**
   * Get the client for making function calls
   */
  getClient(): ForstSidecarClient {
    if (!this.client) {
      throw new SidecarNotStarted();
    }
    return this.client;
  }

  /**
   * Get server information. Requires {@link start} to have completed successfully.
   */
  getServerInfo(): ServerInfo {
    if (!this.server) {
      throw new SidecarNotStarted();
    }
    return this.server.getServerInfo();
  }

  /**
   * Whether the underlying dev server process is running. Returns false if {@link start} has not run.
   */
  isRunning(): boolean {
    if (!this.server) {
      return false;
    }
    return this.server.isRunning();
  }

  /**
   * Discover available functions
   */
  async discoverFunctions(): Promise<FunctionInfo[]> {
    if (!this.client) {
      throw new SidecarNotStarted();
    }
    return this.client.discoverFunctions();
  }

  /**
   * Invoke a Forst function with **positional** arguments. The dev server passes `args` as JSON
   * to the executor (same shape as `POST /invoke`: typically a JSON array matching Forst parameter order).
   */
  async invoke(
    packageName: string,
    functionName: string,
    args: unknown[] = []
  ): Promise<InvokeResponse<unknown>> {
    if (!this.client) {
      throw new SidecarNotStarted();
    }
    return this.client.invokeFunction(packageName, functionName, args as any[]);
  }

  /**
   * Invoke a Forst function with streaming. Uses the same **`args`** shape as {@link invoke}
   * (positional JSON array for the executor); the HTTP layer still sends it as the `args` field on `POST /invoke`.
   */
  async invokeStreaming(
    packageName: string,
    functionName: string,
    args: unknown[] = [],
    onResult?: (result: StreamingResult) => void
  ): Promise<void> {
    if (!this.client) {
      throw new SidecarNotStarted();
    }
    return this.client.invokeStreaming(
      packageName,
      functionName,
      args,
      onResult
    );
  }

  /**
   * Health check
   */
  async healthCheck(): Promise<boolean> {
    if (!this.client) {
      return false;
    }
    return this.client.healthCheck();
  }
}

/**
 * Express.js middleware for easy integration
 */
export function createExpressMiddleware(sidecar: ForstSidecar): RequestHandler {
  return async (req, res, next) => {
    (req as Request & { forst: ForstSidecar }).forst = sidecar;
    next();
  };
}

/**
 * Auto-start function for zero-config usage
 */
export async function autoStart(
  config?: Partial<ForstConfig>
): Promise<ForstSidecar> {
  const sidecar = new ForstSidecar(config);
  await sidecar.start();

  process.on("SIGINT", async () => {
    logger.info("\n🛑 Received SIGINT, shutting down gracefully...");
    await sidecar.stop();
    process.exit(0);
  });

  process.on("SIGTERM", async () => {
    logger.info("\n🛑 Received SIGTERM, shutting down gracefully...");
    await sidecar.stop();
    process.exit(0);
  });

  return sidecar;
}

export default ForstSidecar;
